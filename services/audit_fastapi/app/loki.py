from __future__ import annotations

import json
import time
from typing import Any

import httpx


class LokiClient:
    def __init__(self, base_url: str) -> None:
        self._base_url = base_url.rstrip("/")
        self._client = httpx.Client(timeout=5.0)

    @property
    def base_url(self) -> str:
        return self._base_url

    def push(self, event_type: str, payload: Any) -> None:
        line = json.dumps(
            {"event_type": event_type, "payload": payload},
            separators=(",", ":"),
            default=str,
        )
        body = {
            "streams": [
                {
                    "stream": {"service": "audit", "event_type": event_type},
                    "values": [[str(time.time_ns()), line]],
                }
            ]
        }
        response = self._client.post(
            f"{self._base_url}/loki/api/v1/push",
            json=body,
            headers={"Content-Type": "application/json"},
        )
        response.raise_for_status()

    def close(self) -> None:
        self._client.close()
