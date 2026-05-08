from contextlib import asynccontextmanager
import logging
from typing import AsyncIterator

from fastapi import FastAPI, Request, status

from app.config import load_settings
from app.loki import LokiClient
from app.rabbit import RabbitAuditConsumer
from app.schemas import AuditEvent, HealthResponse

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s %(message)s")


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncIterator[None]:
    settings = load_settings()
    loki = LokiClient(settings.loki_url)
    consumer = RabbitAuditConsumer(settings.amqp_url, settings.bindings, loki)

    app.state.settings = settings
    app.state.loki = loki
    app.state.consumer = consumer

    consumer.start()
    try:
        yield
    finally:
        consumer.stop()
        loki.close()


app = FastAPI(title="todoe audit", version="1.0.0", lifespan=lifespan)


@app.get("/health", response_model=HealthResponse)
def health(request: Request) -> HealthResponse:
    settings = request.app.state.settings
    consumer = request.app.state.consumer
    return HealthResponse(
        service="audit",
        status="ok",
        rabbit_connected=consumer.connected,
        loki_url=settings.loki_url,
        queues=[binding.queue for binding in settings.bindings],
    )


@app.post("/audit/events", status_code=status.HTTP_202_ACCEPTED)
def record_event(event: AuditEvent, request: Request) -> dict[str, str]:
    request.app.state.loki.push(event.type, event.payload)
    return {"status": "accepted"}
