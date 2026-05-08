from __future__ import annotations

from collections import deque
from dataclasses import dataclass
from datetime import UTC, datetime
from threading import Lock
from typing import Any


@dataclass(frozen=True)
class AuditRecord:
    event_type: str
    payload: Any
    source: str
    created_at: datetime


class AuditCache:
    def __init__(self, max_size: int) -> None:
        self._records: deque[AuditRecord] = deque(maxlen=max_size)
        self._lock = Lock()

    def append(self, event_type: str, payload: Any, source: str) -> AuditRecord:
        record = AuditRecord(
            event_type=event_type,
            payload=payload,
            source=source,
            created_at=datetime.now(UTC),
        )
        with self._lock:
            self._records.appendleft(record)
        return record

    def list(self, limit: int = 100) -> list[AuditRecord]:
        with self._lock:
            return list(self._records)[:limit]

    def count(self) -> int:
        with self._lock:
            return len(self._records)
