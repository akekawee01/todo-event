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
    bindings: tuple[Binding, ...]


def load_settings() -> Settings:
    return Settings(
        amqp_url=os.getenv("AMQP_URL", "amqp://guest:guest@localhost:5672/"),
        loki_url=os.getenv("LOKI_URL", "http://localhost:3100"),
        bindings=(
            Binding(exchange="task.events", queue="audit.task.events"),
            Binding(exchange="onboarding.events", queue="audit.user.events"),
        ),
    )
