from __future__ import annotations

import json
import logging
import threading
import time
from typing import Any

import pika
from pika.adapters.blocking_connection import BlockingChannel
from pika.spec import Basic, BasicProperties

from app.cache import AuditCache
from app.config import Binding
from app.loki import LokiClient

logger = logging.getLogger("audit.rabbit")


class RabbitAuditConsumer:
    def __init__(
        self,
        amqp_url: str,
        bindings: tuple[Binding, ...],
        loki: LokiClient,
        cache: AuditCache,
    ) -> None:
        self._amqp_url = amqp_url
        self._bindings = bindings
        self._loki = loki
        self._cache = cache
        self._stop = threading.Event()
        self._connected = False
        self._connection: pika.BlockingConnection | None = None
        self._channel: BlockingChannel | None = None
        self._thread: threading.Thread | None = None
        self._lock = threading.Lock()

    @property
    def connected(self) -> bool:
        with self._lock:
            return self._connected

    def start(self) -> None:
        self._thread = threading.Thread(target=self._run, name="rabbit-audit", daemon=True)
        self._thread.start()

    def stop(self) -> None:
        self._stop.set()
        channel = self._channel
        connection = self._connection
        if channel and connection and connection.is_open:
            connection.add_callback_threadsafe(channel.stop_consuming)
        if self._thread:
            self._thread.join(timeout=10)

    def _set_connected(self, connected: bool) -> None:
        with self._lock:
            self._connected = connected

    def _run(self) -> None:
        while not self._stop.is_set():
            try:
                self._consume()
            except Exception:
                logger.exception("rabbit consumer stopped unexpectedly")
            finally:
                self._set_connected(False)
                self._close_connection()

            if not self._stop.is_set():
                time.sleep(5)

    def _consume(self) -> None:
        params = pika.URLParameters(self._amqp_url)
        self._connection = pika.BlockingConnection(params)
        self._channel = self._connection.channel()
        self._channel.basic_qos(prefetch_count=10)

        for binding in self._bindings:
            self._channel.exchange_declare(
                exchange=binding.exchange,
                exchange_type="fanout",
                durable=True,
            )
            self._channel.queue_declare(queue=binding.queue, durable=True)
            self._channel.queue_bind(
                queue=binding.queue,
                exchange=binding.exchange,
                routing_key="",
            )
            self._channel.basic_consume(
                queue=binding.queue,
                on_message_callback=self._handle_delivery,
                auto_ack=False,
            )

        self._set_connected(True)
        logger.info("audit consumer listening on %s", [b.queue for b in self._bindings])
        self._channel.start_consuming()

    def _handle_delivery(
        self,
        channel: BlockingChannel,
        method: Basic.Deliver,
        _properties: BasicProperties,
        body: bytes,
    ) -> None:
        try:
            message = json.loads(body)
            event_type = str(message["type"])
            payload: Any = message.get("payload")
        except Exception:
            logger.exception("failed to decode audit message")
            channel.basic_nack(delivery_tag=method.delivery_tag, requeue=False)
            return

        try:
            logger.info("audit received event type=%s", event_type)
            self._cache.append(event_type, payload, method.exchange)
            self._loki.push(event_type, payload)
        except Exception:
            logger.exception("failed to push audit event to loki")
        finally:
            channel.basic_ack(delivery_tag=method.delivery_tag)

    def _close_connection(self) -> None:
        try:
            if self._channel and self._channel.is_open:
                self._channel.close()
        except Exception:
            logger.debug("error closing rabbit channel", exc_info=True)

        try:
            if self._connection and self._connection.is_open:
                self._connection.close()
        except Exception:
            logger.debug("error closing rabbit connection", exc_info=True)
