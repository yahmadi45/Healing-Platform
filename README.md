# SentinelOps

**PFE topic:** Projet 10 — DevOps, SSO et supervision

**Current scope:** engineering MVP with a Go modular-monolith API, PostgreSQL persistence, React/TypeScript/Vite operations console, Docker Compose, and Prometheus metrics.

> **Demo safety boundary:** failure and recovery actions change SentinelOps demo records only. The application does not connect to the Docker socket, execute host commands, stop/restart real containers, deploy releases, or perform real infrastructure recovery. Keycloak/OIDC, RBAC, NATS JetStream, a host agent, security scanning, and backup/restore are future milestones, not implemented capabilities.

## Contents

- [Architecture](#architecture)
- [Repository layout](#repository-layout)
- [Requirements](#requirements)
- [Run the complete stack](#run-the-complete-stack)
- [Use the dashboard](#use-the-dashboard)
- [HTTP API](#http-api)
- [Prometheus metrics](#prometheus-metrics)
- [Configuration and secrets](#configuration-and-secrets)
- [Frontend development](#frontend-development)
- [Backend development and tests](#backend-development-and-tests)
- [Docker build and runtime](#docker-build-and-runtime)
- [Persistence and migrations](#persistence-and-migrations)
- [Security boundaries](#security-boundaries)
- [Troubleshooting](#troubleshooting)
- [Roadmap](#roadmap)
- [Two-PC development workflow](#two-pc-development-workflow)

## Architecture

```text
Browser
  │ HTTP :8080
  ▼
Go API / modular monolith
  ├── internal/config       validated environment configuration
  ├── internal/domain       API and domain value types
  ├── internal/httpapi      REST handlers, health, metrics
  ├── internal/store        repository contract and in-memory adapter
  └── internal/postgres     PostgreSQL adapter and embedded migrations
            │
            └── PostgreSQL 17 (Compose named volume)

React + TypeScript + Vite dashboard
  └── production assets are built in a Node stage and embedded in the Go binary

Prometheus :9090 ── scrapes sentinelops:8080/metrics every 15 seconds
```

- **Backend:** Go HTTP server with request timeouts, structured JSON logging, graceful SIGINT/SIGTERM shutdown, and repository interfaces.
- **Storage:** PostgreSQL is selected when connection configuration is present. Without it, the API uses a volatile in-memory demonstration store.
- **Database schema:** the current migration covers demo services, incidents, and operation events. It is not the full proposed PFE ER diagram.
- **Frontend:** React, TypeScript, Vite, responsive dashboard components, typed API client, and unit/UI tests.
- **Production image:** Node compiles the frontend, Go compiles the server with the generated `dist` assets embedded, and the final Alpine runtime runs non-root without Node/npm.
- **API paths:** both `/api/...` and `/api/v1/...` are currently supported for the implemented services, incidents, events, stats, and simulated actions.
- **No message broker or real infrastructure executor is included yet.**

## Repository layout

```text
.github/workflows/ci.yml        GitHub Actions checks
apps/web/                       React + TypeScript + Vite app and tests
internal/config/                environment loading and validation
internal/domain/                API/domain types
internal/httpapi/               API handlers and embedded UI serving
internal/postgres/              pgx repository and SQL migrations
internal/store/                 repository interface and in-memory store
internal/httpapi/web/           production UI embed input; Docker build replaces it with Vite dist
web/                            legacy/static root files retained during the transition
Dockerfile                      frontend, Go build, and non-root runtime stages
docker-compose.yml               SentinelOps, PostgreSQL, and Prometheus
prometheus.yml                   scrape configuration
```

## Requirements

For the containerized stack:

- Docker Engine with the Compose plugin, or Docker Desktop.
- Ports `8080` and `9090` available on the host.

For local development outside Docker:

- Go toolchain matching `go.mod`.
- Node.js and npm compatible with the Vite project.
- PostgreSQL only if testing the persistent adapter locally; otherwise the in-memory adapter can be used.

## Run the complete stack

Run from the repository root, where `docker-compose.yml` is located:

```sh
docker compose up --build --pull always
```

Compose starts PostgreSQL, waits for its health check, starts SentinelOps and applies embedded SQL migrations, then starts Prometheus after the API is healthy.

Open:

- Dashboard: <http://localhost:8080>
- API health: <http://localhost:8080/healthz>
- Readiness (includes database connectivity): <http://localhost:8080/readyz>
- Metrics: <http://localhost:8080/metrics>
- Prometheus: <http://localhost:9090>
- Prometheus scrape targets: <http://localhost:9090/targets>

Inspect containers and logs:

```sh
docker compose ps
docker compose logs -f sentinelops postgres prometheus
```

Stop the services while retaining database data:

```sh
docker compose down
```

A full local reset removes the Compose database volume and its contents:

```sh
docker compose down -v
```

Use `down -v` only when intentionally deleting the local PostgreSQL data.

## Use the dashboard

The overview shows service counts, healthy services, open incidents, recovery totals, service states, last checks, and the incident timeline. Dashboard data refreshes periodically and after actions.

- **Simulate failure** changes a healthy demo service to unhealthy and records an incident/event.
- **Simulate recovery** returns an unhealthy demo service to healthy, resolves its open incident, and increments the recovery count.
- A per-service recovery limit is enforced by the demo store.
- The UI explicitly marks these as simulated actions. They do not control Docker or the host.

When PostgreSQL is enabled through Compose, demo state is persisted in the database. In-memory mode loses its state when the process exits.

## HTTP API

JSON responses are used except for `/metrics`.

| Method | Endpoint | Purpose |
|---|---|---|
| `GET` | `/api/services` and `/api/v1/services` | List tracked demo services |
| `GET` | `/api/incidents` and `/api/v1/incidents` | List incidents, latest first |
| `GET` | `/api/events` and `/api/v1/events` | List operation events |
| `GET` | `/api/stats` and `/api/v1/stats` | Return service, health, incident, and recovery totals |
| `POST` | `/api/services/{name}/fail` and `/api/v1/services/{name}/fail` | Simulate a service failure |
| `POST` | `/api/services/{name}/recover` and `/api/v1/services/{name}/recover` | Simulate a service recovery |
| `GET` | `/healthz` | Process health check |
| `GET` | `/readyz` | Repository/database readiness check |
| `GET` | `/metrics` | Prometheus exposition format |

Example requests:

```sh
curl http://localhost:8080/api/v1/services
curl http://localhost:8080/api/v1/stats
curl -X POST http://localhost:8080/api/v1/services/api/fail
curl http://localhost:8080/api/v1/incidents
curl -X POST http://localhost:8080/api/v1/services/api/recover
```

Unknown API paths return `404`; API paths are never routed to the frontend fallback. Missing static assets also return `404`. Unknown extensionless browser routes serve the React `index.html` to support client-side routing.

Action errors include `404` for unknown services/actions, `409` for conflicting state, `403` when demo actions are disabled, and `429` when the recovery policy limit is reached.

## Prometheus metrics

Prometheus scrapes `sentinelops:8080/metrics` every 15 seconds. Use the Prometheus expression browser to query:

- `sentinelops_services_total`
- `sentinelops_services_unhealthy`
- `sentinelops_incidents_open`
- `sentinelops_recoveries_total`

Check <http://localhost:9090/targets> and ensure the `sentinelops` target is `UP`. These are application/demo-record metrics, not measurements of all Docker containers or host CPU, RAM, or disk.

## Configuration and secrets

For local overrides, copy `.env.example` to `.env`, replace the sample database password with a unique local value, and run Compose from the project root. Supported settings:

| Variable | Default | Purpose |
|---|---|---|
| `LISTEN_ADDR` | `:8080` | API listen address inside the container |
| `ALLOW_DEMO_ACTIONS` | `true` | Enables simulated failure/recovery actions |
| `SENTINELOPS_PORT` | `8080` | Host-published API/dashboard port |
| `PROMETHEUS_PORT` | `9090` | Host-published Prometheus port |
| `POSTGRES_DB` | `sentinelops` | Compose database name |
| `POSTGRES_USER` | `sentinelops` | Compose database user |
| `POSTGRES_PASSWORD` | local sample value | Compose database password; replace locally |
| `DATABASE_URL` | unset | Optional Go PostgreSQL connection URL |
| `POSTGRES_HOST` | unset outside Compose | PostgreSQL host for the Go API |
| `POSTGRES_PORT` | `5432` | PostgreSQL port |
| `POSTGRES_SSLMODE` | `disable` | PostgreSQL TLS mode; local-only default in Compose |
| `SENTINELOPS_API_TARGET` | `http://localhost:8080` | Vite proxy target for local frontend development |

Compose provides both supported environment mechanisms: service `environment:` values and automatic `.env` interpolation. For other invocations, environment variables can be set with `docker run -e NAME=value`, `docker run --env-file .env`, or Compose `env_file:`. Never bake credentials into the image, commit a real `.env`, or reuse the sample local password in a shared or production system.

## Frontend development

From the repository root, use two terminals.

Terminal 1, start the Go API. Without database variables, it uses the in-memory demo store:

```sh
go run .
```

Terminal 2, start Vite:

```sh
cd apps/web
npm ci
npm run dev
```

Open <http://localhost:5173>. Vite proxies `/api`, `/healthz`, `/readyz`, and `/metrics` to `SENTINELOPS_API_TARGET` (default `http://localhost:8080`).

Available frontend scripts:

```sh
npm run lint
npm test
npm run build
```

The production `dist` is generated by the Docker build; it is not required to be committed.

## Backend development and tests

From the repository root:

```sh
gofmt -w main.go main_test.go internal
go test -race ./...
go vet ./...
```

The test suite covers repository behavior, API/action flows, configuration validation, health/metrics routes, and static-route boundaries. PostgreSQL migrations are applied at service startup when a database is configured.

## Docker build and runtime

Build the production image:

```sh
docker build -t sentinelops:local .
```

Validate Compose without starting services:

```sh
docker compose config --quiet
```

The multi-stage build runs `npm ci` from the frontend lockfile, builds Vite, places generated files in `internal/httpapi/web`, compiles the Go executable, then copies only the binary into the Alpine runtime. The final image runs as a non-root `app` user and does not contain Node.js or npm.

GitHub Actions checks frontend install/lint/tests/build, Go formatting/race tests/vet, Compose configuration, and Docker image compilation for pull requests and pushes to `main`.

## Persistence and migrations

`DATABASE_URL` can be used directly by the API. Compose instead supplies PostgreSQL connection fields, which are assembled into a URL by `internal/config`.

At startup the server:

1. Opens the PostgreSQL driver when database settings are present.
2. Applies embedded, versioned SQL migrations transactionally.
3. Pings the database before constructing the PostgreSQL repository.
4. Serves `/readyz` based on the selected repository's health.

The current schema is deliberately small: `services`, `incidents`, `operation_events`, and `schema_migrations`. PostgreSQL data lives in the `postgres_data` named volume. Do not remove that volume unless you intend to delete persisted local demo records.

## Security boundaries

This is a local learning MVP, not a production-ready control plane.

- No login, Keycloak/OIDC, RBAC, CSRF defense, or user audit identity is implemented. Keep it on a trusted machine or isolated lab; do not expose it publicly.
- Simulated actions modify database/demo records only; no host process, Docker container, or release is controlled.
- Do not expose the Docker daemon TCP port or mount the Docker socket into this application.
- Use unique local passwords and a proper secrets mechanism for shared environments. Do not commit tokens, passwords, or private keys.
- A future infrastructure agent must have a narrow allowlist, authenticated identity, least privilege, bounded timeouts, idempotency, audit, and post-action verification; never accept arbitrary shell commands over HTTP.
- Metrics and logs currently describe this application only. They do not constitute host monitoring or a durable security audit system.

## Troubleshooting

### Compose says no configuration file was found

Run commands from the directory containing `docker-compose.yml` (the repository root), or pass `-f` with its full path:

```sh
docker compose -f path/to/docker-compose.yml up --build --pull always
```

### A port is already in use

Set a different published host port in `.env`, for example `SENTINELOPS_PORT=8081`, then open `http://localhost:8081`. Set `PROMETHEUS_PORT` similarly if port `9090` is occupied.

### Check service state and logs

```sh
docker compose ps
docker compose logs --tail=100 sentinelops postgres prometheus
```

### API is not ready

Check `docker compose ps`, inspect SentinelOps and PostgreSQL logs, and verify `/readyz`. The API waits for PostgreSQL health in Compose and applies migrations on startup.

### Prometheus target is DOWN

Open <http://localhost:9090/targets>, confirm SentinelOps is healthy, and inspect its logs. From within the Compose network the configured target is `sentinelops:8080`; use the Compose service DNS name rather than host `localhost`.

### Dashboard loads without styling or JavaScript

Inspect the browser network panel and confirm the hashed `/assets/...` files return `200`. Missing assets intentionally return `404`; rebuild the image from the repository root with `docker compose up --build --pull always`.

### Clean local rebuild

```sh
docker compose build --no-cache sentinelops
docker compose up --pull always
```

This does not delete database data. `docker compose down -v` does.

## Roadmap

1. **Identity and access:** Keycloak/OIDC, server-side role checks, and audit actor identity.
2. **Reliable operations:** request validation, pagination, correlation IDs, idempotency keys, and durable audit records.
3. **Deployment workflow:** release state machine and security gates in an isolated lab; no production control by default.
4. **Observability:** richer Prometheus/Grafana dashboards, Loki log correlation, OpenTelemetry traces, and alert ingestion.
5. **Backup and disaster recovery:** protected backups, isolated restore verification, and measured RPO/RTO.
6. **Multi-host extension:** authenticated Go agent and least-privilege command protocol; consider NATS JetStream only after durable asynchronous work is justified.

Treat roadmap items as unimplemented until code, tests, and documented evidence exist.

## Two-PC development workflow

Use the storage-rich PC as the development/Docker host for the repository, images, PostgreSQL volume, and Compose services. Use the smaller PC as an editor/browser client, preferably through VS Code Remote SSH. Keep each contributor or coding agent on its own branch/worktree, open a pull request, run CI, and review before merging. Do not share a mutable working tree between agents.
