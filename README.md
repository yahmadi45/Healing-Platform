# SentinelOps — Engineering MVP

SentinelOps is a learning platform for **Projet 10 — DevOps, SSO et supervision**. The current milestone is a modular Go control-plane API, PostgreSQL persistence, Docker Compose, Prometheus metrics, and a small demo console.

> **Scope and safety:** failure and recovery actions update SentinelOps demo records only. The platform does not connect to the Docker socket, execute host commands, deploy application releases, or perform real recovery. OIDC/Keycloak, RBAC, NATS, a host agent, security scanning, and backup/restore are future milestones, not implemented features.

## Current architecture

```text
Browser ── HTTP :8080 ──> Go API (modular monolith)
                              ├── config: validated environment settings
                              ├── domain: API/domain data types
                              ├── httpapi: routes, JSON, metrics, health
                              ├── store: repository interface + in-memory adapter
                              └── postgres: pgx adapter + embedded SQL migrations
                                      │
                                      └── PostgreSQL 17 (Compose named volume)

Prometheus :9090 ── scrapes sentinelops:8080/metrics every 15 seconds
```

- Go API: `main.go`, with internal modules under `internal/`.
- Persistence: PostgreSQL is used when configured; startup applies embedded migrations. Without database configuration, the server deliberately falls back to the in-memory demo adapter.
- Frontend: the current UI is static HTML/CSS/JavaScript embedded in the API. It is not yet the planned React/TypeScript/Vite frontend.
- Health: `/healthz` checks the process; `/readyz` checks the selected repository/database.
- Graceful shutdown: SIGINT/SIGTERM stops HTTP serving with a timeout.
- SQL schema: `internal/postgres/migrations/0001_foundation.sql` creates services, incidents, operation events, constraints, and indexes.

## Repository layout

```text
.github/workflows/ci.yml       Go tests, vet, Compose validation, image build
internal/config/               Environment parsing and validation
internal/domain/               Domain/API types
internal/httpapi/              HTTP handlers, metrics, static UI
internal/postgres/              PostgreSQL repository and SQL migrations
internal/store/                 Repository contract and in-memory adapter
web/                             Current static demo UI
Dockerfile                       Multi-stage Go build; non-root runtime
docker-compose.yml               SentinelOps, PostgreSQL, Prometheus
prometheus.yml                   Prometheus scrape target
```

## Run locally

From the repository root:

```sh
docker compose up --build --pull always
```

Open:

- Dashboard: <http://localhost:8080>
- Health: <http://localhost:8080/healthz>
- Readiness: <http://localhost:8080/readyz>
- Prometheus metrics: <http://localhost:8080/metrics>
- Prometheus UI: <http://localhost:9090>
- Prometheus targets: <http://localhost:9090/targets>

Check services and logs:

```sh
docker compose ps
docker compose logs -f sentinelops postgres prometheus
```

Stop containers but keep PostgreSQL data:

```sh
docker compose down
```

`docker compose down -v` deletes the PostgreSQL named volume and its data; use it only when you intentionally want a full local reset.

## Configuration and secrets

Copy `.env.example` to `.env` and set a unique local PostgreSQL password. Compose also supports `SENTINELOPS_PORT`, `PROMETHEUS_PORT`, `ALLOW_DEMO_ACTIONS`, `POSTGRES_DB`, `POSTGRES_USER`, and `POSTGRES_PASSWORD`.

The example credentials are for isolated local development only. Do not commit a real `.env`, put secrets in the Docker image, or reuse these credentials outside a disposable local environment. A production deployment needs an external secret-management mechanism, TLS, authentication, authorization, and network restrictions.

The API accepts either `DATABASE_URL`, or `POSTGRES_HOST`, `POSTGRES_PORT`, `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, and `POSTGRES_SSLMODE`. PostgreSQL URI values are escaped by Go's URL builder. `LISTEN_ADDR` defaults to `:8080`.

## API and demo actions

| Method | Endpoint | Purpose |
|---|---|---|
| GET | `/api/v1/services` | List demo services |
| GET | `/api/v1/incidents` | List incidents |
| GET | `/api/v1/events` | List recent operation events |
| GET | `/api/v1/stats` | Service and incident summary |
| POST | `/api/v1/services/{name}/fail` | Simulate service failure |
| POST | `/api/v1/services/{name}/recover` | Simulate recovery, limited to three per service |
| GET | `/healthz` | Process health |
| GET | `/readyz` | Repository/database readiness |
| GET | `/metrics` | Prometheus exposition format |

Example:

```sh
curl http://localhost:8080/api/v1/services
curl -X POST http://localhost:8080/api/v1/services/api/fail
curl http://localhost:8080/api/v1/incidents
curl -X POST http://localhost:8080/api/v1/services/api/recover
```

These are simulator state transitions. No Docker containers or Linux host services are stopped or restarted. The API currently has no login or RBAC, so keep it on a trusted local machine/lab and do not expose it publicly.

## Prometheus metrics

- `sentinelops_services_total`
- `sentinelops_services_unhealthy`
- `sentinelops_incidents_open`
- `sentinelops_recoveries_total`

Prometheus scrapes every 15 seconds. Use `/targets` to confirm `sentinelops` is `UP`; query a metric by name in the expression browser. These metrics reflect application records, not host/container telemetry.

## Tests and CI

Run Go checks when Go is installed:

```sh
gofmt -w $(find . -name '*.go')
go test -race -cover ./...
go vet ./...
docker compose config --quiet
docker build -t sentinelops:local .
```

GitHub Actions runs formatting, race-enabled Go tests, `go vet`, Compose validation, and a Docker image build for pushes and pull requests targeting `main`.

## M1 boundaries and design decisions

- Modular monolith rather than microservices; repository interfaces isolate storage from HTTP handlers.
- PostgreSQL migrations are embedded, versioned, sorted, and each migration runs inside a transaction.
- Database mutations lock the service row and persist the service state, incident, and event in one transaction.
- The current schema is intentionally limited to demo services/incidents/events; it is not the full proposed PFE ERD.
- No NATS/outbox yet: add them when durable asynchronous work is implemented, not as unused infrastructure.
- No arbitrary shell or Docker API access. Any future agent execution must use a strict allowlist, authenticated identity, least privilege, bounded timeouts, idempotency, and audit.

## Next milestones

1. Add CI-backed React/TypeScript/Vite UI that consumes `/api/v1` and retains the current demo dashboard behavior.
2. Add OIDC login through Keycloak and server-side role authorization before privileged endpoints.
3. Add request validation, pagination, correlation IDs, and durable audit records.
4. Define deployment state machine and security-gate workflow; initially execute only in an isolated lab.
5. Add alert ingestion/correlation and integrate Prometheus/Grafana/Loki/OpenTelemetry.
6. Add backup/restore with documented RPO/RTO and a repeatable isolated restore test.
7. Add a Go agent and NATS JetStream only after enrollment, identity, command contracts, and idempotency are defined.

## Two-PC workflow

Keep the repository, Docker images, database volume, and Compose services on the storage-rich PC. Use the smaller PC as a VS Code/SSH client. Work on feature branches or isolated Git worktrees, keep Copilot's UI work on its own branch, and merge only after CI passes. Do not share one mutable worktree between multiple agents.
