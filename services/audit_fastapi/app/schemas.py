from typing import Any

from pydantic import BaseModel


class AuditEvent(BaseModel):
    type: str
    payload: Any = None


class HealthResponse(BaseModel):
    service: str
    status: str
    rabbit_connected: bool
    loki_url: str
    queues: list[str]
