# todoe

Modular monolith with Ports & Adapters architecture, append-only persistence, and event-driven side effects. Domains run as independent binaries communicating over RabbitMQ.

## Services

| Service | Port | Role |
|---|---|---|
| `cmd/api` | `3000` | Auth (`/auth/*`), tasks (`/tasks/*`), health (`/health`). Consumes `user.activated` to create credentials. MongoDB. |
| `cmd/onboarding` | `3003` | User registration & onboarding flow (`/users/*`), captcha (`/captcha/*`). Credit scoring and welcome logging run in-process. **MySQL + MongoDB**. |
| `services/audit_fastapi` | `3004` | FastAPI audit service. Consumes `task.events` and `onboarding.events`, forwards audit entries to Loki, exposes `/health`. |
| `web/vue` | `5173` | Task UI. Proxies `/api/*` to `cmd/api`. |
| `web/onboarding` | `5174` | Onboarding UI. Proxies user/captcha requests to `cmd/onboarding`. |

## Architecture

### Service topology

HTTP from the browsers, fanout pub/sub over RabbitMQ between services, **per-service storage** (MongoDB for api/onboarding, MySQL for onboarding), audit stream to Loki/Grafana. Credit scoring and welcome logging run in-process inside `cmd/onboarding`.

```
   ┌──────────────────┐                ┌──────────────────────────────────┐
   │   Task UI        │                │         Onboarding UI            │
   │   web/vue        │                │         web/onboarding           │
   └────────┬─────────┘                └─┬──────────────────────┬─────────┘
            │                            │                      │
            │ /auth/*     /api/users/*   │              /captcha/*
            │ /tasks/*    /auth/login    │                      │
            ▼                            ▼                      ▼
   ┌──────────────────┐      ┌───────────────────────────────────────────┐
   │  cmd/api :3000   │      │           cmd/onboarding :3003            │
   │  auth · tasks    │      │  users · captcha · credit(in-proc)        │
   │  health          │      │  welcome-logging(in-proc)                 │
   │  [MongoDB]       │      │  [MySQL + MongoDB]                        │
   └────────┬─────────┘      └───────────────┬───────────────────────────┘
            │ AMQP                           │ AMQP
            ▼                               ▼
   ╔═════════════════════════════════════════════════════════════╗
   ║                         RabbitMQ                            ║
   ║                                                             ║
   ║  task.events       ─► audit.task.events    ─► audit_fastapi ║
   ║  onboarding.events ─► audit.user.events    ─► audit_fastapi ║
   ║  user.events       ─► authen.user.events   ─► cmd/api       ║
   ╚═════════════════════════════════════════════════════════════╝
                                                      │
                                             ┌────────▼─────────┐
                                             │  audit FastAPI   │
                                             │    → Loki :3100  │
                                             └────────┬─────────┘
                                                      ▼
                                             ┌──────────────────┐
                                             │  Grafana :3001   │
                                             └──────────────────┘

   Storage tier:
     MongoDB :27017  db: todoe              ◄── cmd/api · cmd/onboarding
       collections: auth_events, auth_credentials, auth_sessions,
                    tasks_events, tasks_view, captcha_events, captcha_challenges
     MySQL   :3306   db: todoe_onboarding   ◄── cmd/onboarding
       tables:      users_events, users_view
```

### Onboarding choreography

The four-step user flow. Credit scoring and welcome logging are in-process within `cmd/onboarding`; only `user.activated` crosses the process boundary to `cmd/api`.

```
  Browser       Onboarding (in-process)               RabbitMQ        API
     │         HTTP │  EventBus │ Credit │ Welcome        │              │
     │              │           │        │                │              │
     │ POST /users/register     │        │                │              │
     ├─────────────►│           │        │                │              │
     │              │─EventRegistered──►│ step 1/4       │              │
     │              │           │        │                │              │
     │ POST /users/:id/verify-email      │                │              │
     ├─────────────►│           │        │                │              │
     │              │─EventEmailVerified►│ step 2/4       │              │
     │              │           │───────►│ scores(email)  │              │
     │              │◄RecordCreditScore──┘                │              │
     │              │─EventCreditScored──────────────────►│ step 3/4    │              │
     │              │           │        │                │              │
     │ POST /users/:id/complete-profile  │                │              │
     ├─────────────►│           │        │                │              │
     │              │─EventProfileCompleted──────────────►│ step 4/4    │
     │              │─user.activated (AMQP)───────────────────────────►│
     │              │           │        │                │    ActivateUser
     │              │           │        │                │              │
     │ POST /auth/login                                   │              │
     ├───────────────────────────────────────────────────────────────►│
     │◄───────────────────────────────────── session token ────────────┤
```

## Prerequisites

Install these first:

- Go `1.25+`
- Docker + Docker Compose
- Bun `1.x` (for frontend apps)
- `curl` (for quick API checks)

## Repo Setup

```bash
# from repo root
go mod download
go mod verify
```

Frontend dependencies:

```bash
cd web/vue && bun install
cd ../onboarding && bun install
cd ../..
```

## Environment Variables

The backend binaries use environment variables, but all of them have local defaults.

Create a `.env` (optional but recommended) in repo root:

```env
# Shared
MONGO_URI=mongodb://root:root@localhost:27017
AMQP_URL=amqp://guest:guest@localhost:5672/

# Onboarding (MySQL) — multiStatements=true is required by golang-migrate
MYSQL_DSN=todoe:todoe@tcp(localhost:3306)/todoe_onboarding?parseTime=true&multiStatements=true

# Audit service
LOKI_URL=http://localhost:3100
```

If omitted:
- `MONGO_URI` defaults to `mongodb://root:root@localhost:27017`
- `AMQP_URL` defaults to `amqp://guest:guest@localhost:5672/`
- `MYSQL_DSN` defaults to `todoe:todoe@tcp(localhost:3306)/todoe_onboarding?parseTime=true&multiStatements=true`
- `LOKI_URL` defaults to `http://localhost:3100`

## Start Everything

Run the full stack:

```bash
docker compose up -d
docker compose ps        # wait until mysql + rabbitmq show "healthy" (~10s)
```

Exposed ports:
- MongoDB: `27017`
- MySQL: `3306` (user `todoe` / password `todoe`, database `todoe_onboarding`)
- RabbitMQ AMQP: `5672`
- RabbitMQ management UI: `15672` (guest/guest)
- Loki: `3100`
- Grafana: `3001` (container `3000`)
- Task UI: `5173`
- Onboarding UI: `5174`
- API: `3000`
- Onboarding API: `3003`
- Audit FastAPI: `3004`

## Run Services Locally

```bash
go run ./cmd/api          # :3000  auth, tasks, health
go run ./cmd/onboarding   # :3003  users, captcha, credit scoring, welcome logging
```

Run the FastAPI audit service from another terminal:

```bash
cd services/audit_fastapi
python -m venv .venv
. .venv/bin/activate
pip install -r requirements.txt
uvicorn app.main:app --host 0.0.0.0 --port 3004
```

`cmd/api`, `cmd/onboarding`, and the audit service connect to RabbitMQ on startup. `cmd/api` and `cmd/onboarding` also use MongoDB, and `cmd/onboarding` additionally uses MySQL.

## Run Frontends (optional)

### Task UI

```bash
cd web/vue
bun run dev
```

### Onboarding UI

```bash
cd web/onboarding
bun run dev
```

Open [http://localhost:5173](http://localhost:5173).
When both frontends run in Docker Compose, open the onboarding UI at [http://localhost:5174](http://localhost:5174).

## Walk the onboarding flow

With the stack running, open [http://localhost:5174](http://localhost:5174) and step through:

1. **Register** — fill in name + email.
2. **Captcha** — solve the math challenge.
3. **Verify email** — `cmd/onboarding` prints the verification token in its log ("step 1/4 …"). Paste it into the UI.
4. **Wait for credit** — credit scoring runs immediately in-process; the log shows "step 3/4" right after step 2.
5. **Complete profile** — submit a bio. `cmd/api` consumes `user.activated` and prints a temp password to its log.
6. **Login** — auth screen accepts the email + that temp password; you land in the task UI.

### Verify storage end-to-end

```bash
# MySQL — onboarding's append-only events + projection view
mysql -h 127.0.0.1 -P 3306 -u todoe -ptodoe todoe_onboarding \
  -e "SELECT id, status, credit_approved FROM users_view; \
      SELECT type FROM users_events ORDER BY created_at;"

# MongoDB — credential created from the user.activated message
docker exec -it todoe-mongo mongosh -u root -p root \
  --authenticationDatabase admin todoe \
  --eval 'db.auth_credentials.findOne()'

# RabbitMQ — queues, bindings, ready/unacked counts
open http://localhost:15672      # guest / guest
```

### Health check

```bash
curl http://localhost:3000/health
curl http://localhost:3004/health
```

### Manual audit event

```bash
curl -X POST http://localhost:3004/audit/events \
  -H 'Content-Type: application/json' \
  -d '{"type":"audit.manual","payload":{"source":"curl"}}'
```

## Build & Test

```bash
go build ./...
go test ./...
```

## Messaging

Cross-domain events flow through RabbitMQ fanout exchanges with durable queues and persistent delivery. Each publishing service declares the queues its downstream consumers expect at startup, so messages buffer on disk while consumers are offline:

| Exchange | Queue | Consumer |
|---|---|---|
| `task.events` | `audit.task.events` | `services/audit_fastapi` |
| `onboarding.events` | `audit.user.events` | `services/audit_fastapi` |
| `user.events` | `welcome.user.events` | `cmd/welcome` |
| `user.events` | `credit.user.events` | `cmd/credit` |
| `user.events` | `authen.user.events` | `cmd/api` (creates credential on `user.activated`) |
| `credit.results` | `onboarding.credit.results` | `cmd/onboarding` |

Inspect queues, bindings, and ready/unacked counts at [http://localhost:15672](http://localhost:15672) (guest/guest).

## Schema migrations (MySQL)

`cmd/onboarding` runs MySQL migrations at startup via `golang-migrate`. Migration files live next to the adapter in `domain/user/adapter/migrations/` and are embedded into the binary with `//go:embed`. State is tracked in the `schema_migrations` table.

To add a migration, drop a new pair into `domain/user/adapter/migrations/`:

```
0002_<short_description>.up.sql
0002_<short_description>.down.sql
```

Number them sequentially, three or four digits, snake_case suffix. The next time `cmd/onboarding` boots it applies any new versions automatically; existing versions are skipped.

To roll back the most recent migration during development:

```bash
go run -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1 \
  -path domain/user/adapter/migrations \
  -database "mysql://todoe:todoe@tcp(localhost:3306)/todoe_onboarding?multiStatements=true" \
  down 1
```

## Observability

- Grafana: [http://localhost:3001](http://localhost:3001)
- Loki datasource is provisioned from `provisioning/`.
- Audit events are pushed by `services/audit_fastapi` into Loki.

## Stop Everything

Ctrl-C any local backend/frontend processes, then:

```bash
docker compose down            # keeps volumes
docker compose down -v         # also wipes Mongo + MySQL + RabbitMQ data
```
