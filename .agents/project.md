# Project Map

`todoe` is an event-sourced demo app using ports and adapters. Go services publish domain events through RabbitMQ; the FastAPI audit service consumes audit queues, caches recent audit entries, and pushes them to Loki. Frontends are Vue apps served through nginx in Docker Compose.

## Services

| Service | Path | Port | Purpose |
|---|---|---:|---|
| API | `cmd/api` | 3000 | Auth, sessions, tasks, health. MongoDB. |
| Onboarding | `cmd/onboarding` | 3003 | Registration, captcha, email verification, credit scoring, profile completion. MySQL + MongoDB. |
| Audit | `services/audit_fastapi` | 3004 | Consumes domain audit queues, caches recent events, pushes to Loki. |
| Task UI | `web/vue` | 5173 | Task frontend, proxies `/api/*` to API. |
| Onboarding UI | `web/onboarding` | 5174 | Onboarding frontend, proxies user/captcha routes. |
| Grafana | Docker image | 3001 | Loki UI. |
| Loki | Docker image | 3100 | Audit log store. |
| RabbitMQ | Docker image | 5672, 15672 | Fanout event exchanges and management UI. |
| MongoDB | Docker image | 27017 | API/onboarding document stores. |
| MySQL | Docker image | 3306 | Onboarding event store/projection. |

## Domain Event Flow

- `task.events` -> `audit.task.events` -> audit cache + Loki.
- `user.events` -> `audit.user.domain.events` -> audit cache + Loki.
- `user.events` -> `authen.user.events` -> API credential creation on `user.activated`.
- `auth.events` -> `audit.auth.events` -> audit cache + Loki.
- `captcha.events` -> `audit.captcha.events` -> audit cache + Loki.

## Important Routes

API:
- `GET /health`
- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/logout`
- `POST /tasks`
- `GET /tasks`
- `GET /tasks/:id`
- `PATCH /tasks/:id/status`

Onboarding:
- `POST /users/register`
- `GET /users/activated`
- `GET /users/:id`
- `GET /users/:id/history`
- `PATCH /users/:id`
- `POST /users/:id/verify-email`
- `POST /users/:id/complete-profile`
- `POST /captcha`
- `POST /captcha/:id/verify`

Audit:
- `GET /health`
- `GET /audit/events`
- `POST /audit/events`

## Verification Commands

```bash
go test ./...
python3 -m compileall -q services/audit_fastapi
docker compose config --quiet
bunx --bun vite build
docker compose build api onboarding audit task-ui onboarding-ui
```

Run frontend build commands from the relevant frontend directory.
