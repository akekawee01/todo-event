# todoe

Modular monolith with Ports & Adapters architecture, append-only persistence, and event-driven side effects. Domains run as independent binaries communicating over RabbitMQ.

## Services

| Binary | Port | Role |
|---|---|---|
| `cmd/api` | `3000` | Auth (`/auth/*`), tasks (`/tasks/*`), health (`/health`). Consumes `user.activated` to create credentials. MongoDB. |
| `cmd/onboarding` | `3002` | User registration & onboarding flow (`/users/*`). Publishes `user.events`; consumes `credit.results`. **MySQL** (per-service database). |
| `cmd/captcha` | `3010` | Captcha challenge issue + verify (`/captcha/*`). |
| `cmd/welcome` | — | Logs the four onboarding milestones from `user.events`. |
| `cmd/credit` | — | Scores users on `user.email_verified` and publishes to `credit.results`. |
| `cmd/audit` | — | Forwards `task.events` to Loki. |

## Architecture

### Communication Layers

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Frontend Layer                                │
│  ┌──────────────────┐              ┌──────────────────┐                │
│  │   Task UI        │              │  Onboarding UI   │                │
│  │   web/vue        │              │  web/onboarding  │                │
│  │   (Browser)      │              │  (Browser)       │                │
│  └────────┬─────────┘              └────────┬─────────┘                │
│           │                                 │                          │
│           │         HTTP (REST/JSON)        │                          │
│           └───────────────┬─────────────────┘                          │
│                           ▼                                            │
└───────────────────────────┼────────────────────────────────────────────┘
                            │
┌───────────────────────────┼────────────────────────────────────────────┐
│  Backend API Layer        ▼                                            │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                      HTTP Handlers                               │   │
│  │  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐ │   │
│  │  │  cmd/api :3000   │  │ cmd/onboarding   │  │  cmd/captcha     │ │ │   │
│  │  │  /auth/*         │  │  :3002 /users/*  │  │  :3010 /captcha/*│ │ │   │
│  │  └────────┬─────────┘  └────────┬─────────┘  └────────┬─────────┘ │ │   │
│  └───────────┼────────────────────┼─────────────────────┼───────────┘ │   │
└──────────────┼────────────────────┼─────────────────────┼──────────────┘
               │                     │                     │
               │                     │                     │
               ▼                     ▼                     ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                     Message Bus Layer (RabbitMQ)                        │
│  ════════════════════════════════════════════════════════════════════════│
│  ║  task.events    ─► audit.task.events      ─► cmd/audit              ║
│  ║  user.events    ─► welcome.user.events    ─► cmd/welcome            ║
│  ║                 ─► credit.user.events     ─► cmd/credit             ║
│  ║                 ─► authen.user.events     ─► cmd/api                ║
│  ║  credit.results ─► onboarding.credit.results ─► cmd/onboarding       ║
│  ════════════════════════════════════════════════════════════════════════│
│                                                                          │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐      │
│  │  cmd/welcome     │  │  cmd/credit      │  │  cmd/audit       │      │
│  │  (subscriber)    │  │  (subscriber)    │  │  (subscriber)    │      │
│  └──────────────────┘  └──────────────────┘  └──────────┬─────────┘      │
│                                                            │             │
└────────────────────────────────────────────────────────────┼─────────────┘
                                                             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                     Observability Layer                                 │
│  ┌──────────────────┐        ┌──────────────────┐                        │
│  │  Loki :3100      │◄───────│  cmd/audit       │                        │
│  │  (Log Aggregator)│        │  (publisher)     │                        │
│  └────────┬─────────┘        └──────────────────┘                        │
│           ▼                                                                │
│  ┌──────────────────┐                                                      │
│  │  Grafana :3001   │                                                      │
│  │  (Dashboard)     │                                                      │
│  └──────────────────┘                                                      │
└───────────────────────────────────────────────────────────────────────────┘
```

**Key Points:**
- **Frontend → Backend**: HTTP only (REST API)
- **Backend → Services**: AMQP via RabbitMQ (async pub/sub)
- **Frontend never connects to RabbitMQ directly** — by design for security and simplicity

### Service topology

HTTP from the browsers, fanout pub/sub over RabbitMQ between services, **per-service storage** (MongoDB for api/captcha, MySQL for onboarding), audit stream to Loki/Grafana.

```
   ┌──────────────────┐                          ┌──────────────────┐
   │   Task UI        │                          │  Onboarding UI   │
   │   web/vue        │                          │  web/onboarding  │
   └────────┬─────────┘                          └─┬──────┬─────┬───┘
            │                                      │      │     │
            │ /auth/*                /api/users/*  │      │     │ /captcha/*
            │ /tasks/*              /auth/login    │      │     │
            ▼                                      ▼      │     ▼
   ┌──────────────────┐         ┌──────────────────┐      │   ┌──────────────────┐
   │  cmd/api :3000   │         │ cmd/onboarding   │      │   │  cmd/captcha     │
   │  auth · tasks    │         │  :3002 users     │      │   │  :3010           │
   │  health          │         │                  │      │   │                  │
   │  [MongoDB]       │         │  [MySQL]         │      │   │  [MongoDB]       │
   └────────┬─────────┘         └────────┬─────────┘      │   └────────┬─────────┘
            │ AMQP                       │ AMQP           │            │ AMQP
            ▼                            ▼                ▼            ▼
   ╔═════════════════════════════════════════════════════════════════════════════╗
   ║                              RabbitMQ                                        ║
   ║                                                                              ║
   ║  task.events    ─► audit.task.events           ─► cmd/audit                  ║
   ║  user.events    ─► welcome.user.events         ─► cmd/welcome                ║
   ║                 ─► credit.user.events          ─► cmd/credit                 ║
   ║                 ─► authen.user.events          ─► cmd/api                    ║
   ║  credit.results ─► onboarding.credit.results   ─► cmd/onboarding             ║
   ╚═════════════════════════════════════════════════════════════════════════════╝
            ▲                  ▲
            │ pub user.events  │ pub credit.results
            │                  │
   ┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐
   │  cmd/welcome     │    │  cmd/credit      │    │  cmd/audit       │
   │  logs 4 steps    │    │  scores users    │    │  → Loki :3100    │
   │  [stateless]     │    │  [stateless]     │    │  [Loki]          │
   └──────────────────┘    └──────────────────┘    └────────┬─────────┘
                                                            ▼
                                                   ┌──────────────────┐
                                                   │  Grafana :3001   │
                                                   └──────────────────┘

   Storage tier:
     MongoDB :27017  db: todoe              ◄── cmd/api · cmd/captcha
       collections: auth_events, auth_credentials, auth_sessions,
                    tasks_events, tasks_view, captcha_events, captcha_challenges
     MySQL   :3306   db: todoe_onboarding   ◄── cmd/onboarding
       tables:      users_events, users_view
```

### Onboarding choreography

The four-step user flow as it crosses processes. Every cross-service link goes through RabbitMQ, durable and replayable.

```
  Browser     Onboarding     RabbitMQ        Welcome     Credit       API
     │            │             │               │           │           │
     │ POST /users/register     │               │           │           │
     ├───────────►│             │               │           │           │
     │            │  user.registered            │           │           │
     │            ├────────────►│  ─────────────►│ step 1/4 │           │
     │            │             │               │           │           │
     │ POST /users/:id/verify-email              │           │           │
     ├───────────►│             │               │           │           │
     │            │  user.email_verified        │           │           │
     │            ├────────────►│  ─────────────►│ step 2/4 │           │
     │            │             │  ─────────────────────────►│           │
     │            │             │               │  scores   │           │
     │            │             │  ◄─────credit.results──────┤           │
     │            │◄────────────┤               │           │           │
     │            │  RecordCreditScore          │           │           │
     │            │  user.credit_scored         │           │           │
     │            ├────────────►│  ─────────────►│ step 3/4 │           │
     │            │             │               │           │           │
     │ POST /users/:id/complete-profile          │           │           │
     ├───────────►│             │               │           │           │
     │            │  user.profile_completed     │           │           │
     │            ├────────────►│  ─────────────►│ step 4/4 │           │
     │            │  user.activated             │           │           │
     │            ├────────────►│  ──────────────────────────────────────►│
     │            │             │               │           │ ActivateUser
     │            │             │               │           │           │
     │ POST /auth/login                         │           │           │
     ├──────────────────────────────────────────────────────────────────►│
     │◄───────────────────────── session token ─────────────────────────┤
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

## Start Infrastructure

Run MongoDB, MySQL, RabbitMQ, Loki, Grafana:

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

## Run Services (6 terminals)

```bash
go run ./cmd/api          # :3000  auth, tasks, health
go run ./cmd/onboarding   # :3002  user registration & onboarding
go run ./cmd/captcha      # :3010  captcha challenges
go run ./cmd/welcome      #        logs onboarding milestones
go run ./cmd/credit       #        scores users on email-verified
go run ./cmd/audit        #        forwards task events to Loki
```

Each binary connects to MongoDB and RabbitMQ on startup; HTTP services additionally listen on the port shown above.

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

## Walk the onboarding flow

With all six services and the onboarding UI running, open [http://localhost:5173](http://localhost:5173) and step through:

1. **Register** — fill in name + email.
2. **Captcha** — solve the math challenge.
3. **Verify email** — `cmd/welcome` prints the verification token in its log ("step 1/4 …"). Paste it into the UI.
4. **Wait for credit** — `cmd/credit` scores the user; `cmd/welcome` logs "step 3/4".
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
| `task.events` | `audit.task.events` | `cmd/audit` |
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
- Audit events are pushed by `cmd/audit` into Loki.

## Stop Everything

Ctrl-C the six Go processes (and any frontends), then:

```bash
docker compose down            # keeps volumes
docker compose down -v         # also wipes Mongo + MySQL + RabbitMQ data
```

## FAQ

### Why doesn't the frontend connect to RabbitMQ directly?

Browser-based frontends **should not** connect to message queues like RabbitMQ directly. Here's why:

| Concern | Explanation |
|---|---|
| **Protocol** | AMQP is a TCP protocol — browsers don't support it natively. You'd need a WebSocket bridge or WebAssembly library. |
| **Security** | Opening AMQP ports to the public internet is a severe security risk. Anyone could publish/consume messages. |
| **Complexity** | Frontend code shouldn't be concerned with messaging infrastructure, connection management, or message serialization. |
| **State** | HTTP gives immediate request/response. RabbitMQ is async — you'd need WebSocket or polling to consume events. |

**Best Practice:**
```
Frontend (Browser)
    ↓ HTTP
Backend API (Go)
    ↓ AMQP
RabbitMQ → Other Services
```

The backend acts as a bridge: HTTP in, AMQP out. Frontends stay simple and stateless.

### Is this microservices or modular monolith?

**Microservices.** Each `cmd/*` is an independently deployable binary:
- Separate HTTP ports (3000, 3002, 3010)
- Separate database access patterns (MongoDB vs MySQL)
- Async communication via RabbitMQ (no direct service-to-service calls)

The term "modular" refers to the internal package organization (`internal/*` as shared libraries), but the runtime topology is microservices.

### Can I run services without Docker?

Yes, but you'll need to run MongoDB, MySQL, and RabbitMQ locally. Docker Compose is recommended for development as it handles all infrastructure dependencies.

```bash
# Option 1: Docker Compose (recommended)
docker compose up -d

# Option 2: Run services only (requires manual infrastructure setup)
go run ./cmd/api
go run ./cmd/onboarding
# ... etc
```


