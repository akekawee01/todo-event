from dataclasses import dataclass
import os


@dataclass(frozen=True)
class Binding:
    exchange: str
    queue: str


@dataclass(frozen=True)
class Settings:
    amqp_url: str
    loki_url: str
    cache_size: int
    bindings: tuple[Binding, ...]


def load_settings() -> Settings:
    return Settings(
        amqp_url=os.getenv("AMQP_URL", "amqp://guest:guest@localhost:5672/"),
        loki_url=os.getenv("LOKI_URL", "http://localhost:3100"),
        cache_size=int(os.getenv("AUDIT_CACHE_SIZE", "500")),
        bindings=(
            Binding(exchange="task.events", queue="audit.task.events"),
            Binding(exchange="user.events", queue="audit.user.domain.events"),
            Binding(exchange="auth.events", queue="audit.auth.events"),
            Binding(exchange="captcha.events", queue="audit.captcha.events"),
        ),
    )
