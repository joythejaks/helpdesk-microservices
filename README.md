# Helpdesk Microservices

A ticketing (helpdesk) system built from Go microservices behind an API gateway, with a Flutter client. Tickets, comments, attachments and reports, JWT auth with per-device sessions, and real-time notifications over WebSocket.

## Tech Stack

| Layer | Technology |
| --- | --- |
| Backend language | Go 1.25 (each service is its own module) |
| HTTP | Gin, `net/http/httputil` reverse proxy in the gateway |
| Auth | JWT (`golang-jwt`), bcrypt, rotating per-session refresh tokens |
| Database | PostgreSQL, GORM (pgx driver), `golang-migrate` with embedded SQL migrations |
| Messaging | RabbitMQ (`amqp091-go`): fanout exchange, durable queue, dead-letter queue, publisher confirms |
| Real-time | WebSocket (`gorilla/websocket`) |
| Cache / rate limiting | Redis (`go-redis`), Lua token bucket shared across replicas |
| Resilience | Circuit breakers (`gobreaker`), health/liveness endpoints, graceful shutdown |
| API docs | Swagger / OpenAPI (`swaggo`) on auth-service and ticket-service |
| Edge / TLS | Caddy |
| Observability | Prometheus metrics, Grafana dashboards, structured JSON logs |
| Containers | Docker, Docker Compose |
| Orchestration | Kubernetes with kustomize, CloudNativePG (Postgres HA), RabbitMQ Cluster Operator, cert-manager, metrics-server, HPA |
| Load testing | k6 |
| CI | GitHub Actions (Go build/vet/test, Flutter analyze/test) |
| Client | Flutter, BLoC (`flutter_bloc`), `flutter_secure_storage`, `web_socket_channel`, `fl_chart`, Sentry |

## Architecture

| Service | Port | Role |
| --- | --- | --- |
| `api-gateway` | 8080 | Public entrypoint: JWT validation, rate limiting, circuit breakers, reverse proxy |
| `auth-service` | 8081 | Users, login, JWT and refresh-token sessions, bootstrap admin |
| `ticket-service` | 8082 | Tickets, comments, attachments, reports; publishes events to RabbitMQ |
| `notification-service` | 8083 | Consumes events and pushes them to clients over WebSocket |

Each service has its own PostgreSQL database. Redis holds shared rate-limit state.

```
client ──► Caddy (443) ──► api-gateway ──► auth-service ──► auth-db
                │                     └─► ticket-service ─► ticket-db
                │                              │ publish (confirms)
                │                              ▼
                │                    RabbitMQ  ticket_events (fanout)
                │                              │ ticket_created (durable, DLQ)
                └─ /ws, /notifications ─► notification-service ─► notification-db
```

## Features

- **Auth**: register, login, roles (including admin-created staff), profile and availability, change password. Every login is its own refresh-token session (up to `MAX_SESSIONS_PER_USER`, default 5, oldest evicted), so several devices can stay signed in. Refresh rotates only the presented session and a replayed token is rejected. Logout can end one session or all. Changing the password revokes every session.
- **Tickets**: create, list, detail, history, assign, status changes, comments, attachments, and reports (summary, agent performance, critical trends, queue size).
- **Notifications**: ticket events flow through RabbitMQ to WebSocket clients; failed messages go to a dead-letter queue. The fanout exchange lets notification-service scale to several replicas.
- **Protection**: Redis-backed rate limits (gateway, login, ticket creation, WebSocket connects), per-upstream circuit breakers, trusted-proxy handling for client IPs.
- **Operations**: `/health` (dependency-aware), `/healthz` (liveness), `/metrics`, versioned migrations, database backup sidecars, HA Postgres and RabbitMQ on Kubernetes.

## Run with Docker Compose

```bash
cd backend
docker compose up --build
```

| What | Address |
| --- | --- |
| API gateway | `http://localhost:8080` |
| Caddy (TLS) | `https://localhost` |
| Auth / Ticket / Notification | `localhost:8081` / `8082` / `8083` |
| RabbitMQ management | `http://localhost:15672` |
| Prometheus / Grafana | `http://localhost:9090` / `http://localhost:3000` |
| Postgres (auth / ticket / notification) | `localhost:5433` / `5434` / `5435` |

Notes:

- Caddy serves `SITE_ADDRESS` (default `localhost`, using its own local CA). Set a real domain for a public certificate.
- Each database has a backup sidecar writing to `backend/backups/<db>/`.
- Set `BOOTSTRAP_ADMIN_EMAIL` and `BOOTSTRAP_ADMIN_PASSWORD` on auth-service to create the first admin at startup.
- `JWT_SECRET` and `INTERNAL_SHARED_SECRET` must be your own values; do not ship the defaults. RabbitMQ credentials can be overridden with `RABBITMQ_DEFAULT_USER`, `RABBITMQ_DEFAULT_PASS`, `RABBITMQ_ERLANG_COOKIE`.

## Run on Kubernetes

The manifests in `backend/k8s/` are a kustomize tree targeting a single-node cluster (Docker Desktop). There is no image registry: images are built locally as `helpdesk/<service>:local`.

```bash
backend/k8s/scripts/apply.sh
```

`apply.sh` builds the four images and imports them into the node, installs the operators, generates the secrets, and applies the tree. The steps are also available separately in `backend/k8s/scripts/` (`install-operators.sh`, `install-metrics-server.sh`, `gen-secrets.sh`).

- `data/postgres.yaml`: CloudNativePG clusters, 2 instances each, asynchronous replication.
- `data/rabbitmq.yaml`: RabbitMQ cluster; `RABBITMQ_QUEUE_TYPE=quorum` switches the queues to quorum queues.
- `hpa.yaml`: autoscaling (needs metrics-server). `observability.yaml`: Prometheus and Grafana.
- `loadtest/golden-path.js`: k6 test (login, create ticket, list).

A one-node cluster exercises the manifests and pod-level failover, not node failure.

## Configuration

| Variable | Service | Purpose |
| --- | --- | --- |
| `APP_PORT` | all | Listen port |
| `DB_HOST` `DB_PORT` `DB_USER` `DB_PASSWORD` `DB_NAME` | auth, ticket, notification | Postgres connection |
| `DB_MAX_OPEN_CONNS` `DB_MAX_IDLE_CONNS` `DB_CONN_MAX_LIFETIME` | auth, ticket, notification | Connection pool bounds |
| `JWT_SECRET` | auth, gateway | Signs and validates tokens |
| `INTERNAL_SHARED_SECRET` | gateway, ticket | Gateway-to-service trust |
| `MAX_SESSIONS_PER_USER` | auth | Concurrent sessions per user (default 5) |
| `BOOTSTRAP_ADMIN_EMAIL` `BOOTSTRAP_ADMIN_PASSWORD` | auth | Creates the first admin |
| `AUTH_SERVICE_URL` `TICKET_SERVICE_URL` | gateway | Upstreams |
| `ALLOWED_ORIGINS` | gateway | CORS allow-list |
| `REDIS_URL` | gateway, auth, ticket, notification | Shared rate-limit state; the limiter fails open if Redis is unreachable |
| `RATE_LIMIT_RPS` `RATE_LIMIT_BURST` | gateway | Global limit |
| `AUTH_RATE_LIMIT_RPS` `AUTH_RATE_LIMIT_BURST` | auth | Login/refresh limit |
| `TICKET_RATE_LIMIT_RPS` `TICKET_RATE_LIMIT_BURST` | ticket | Ticket creation limit |
| `WS_RATE_LIMIT_RPS` `WS_RATE_LIMIT_BURST` `MAX_WS_CONNECTIONS` | notification | WebSocket connect limits |
| `TRUSTED_PROXIES` | gateway, notification | CIDRs whose `X-Forwarded-For` is trusted |
| `RABBITMQ_URL` | ticket, notification | Broker connection |
| `RABBITMQ_QUEUE_TYPE` | ticket, notification | `classic` (default) or `quorum` |
| `ENABLE_SWAGGER` | auth, ticket | Serves Swagger UI at `/swagger/index.html` (default off) |

## API

Through the gateway (`http://localhost:8080`):

- Public: `POST /auth/register`, `POST /auth/login`, `POST /auth/refresh`
- Authenticated: `POST /auth/logout`, `GET|PATCH /auth/me`, `PATCH /auth/me/availability`, `POST /auth/change-password`
- Admin: `POST /auth/admin/staff`, `GET /auth/admin/agents`
- Tickets: `GET|POST /tickets`, `GET /tickets/:id`, `GET /tickets/:id/history`, `PATCH /tickets/:id/assign`, `PATCH /tickets/:id/status`, plus comments and attachments under `/tickets/:id/...`
- Reports: `GET /reports/summary|agents|critical-trends|queue-size`
- Real-time: WebSocket at `/ws` (notification-service, through Caddy)

Protected routes need `Authorization: Bearer <access_token>`. `POST /auth/logout` accepts an optional `{"refresh_token": "..."}` to end only that session; without a body it ends all of the user's sessions. Access tokens last 2 hours and are not revocable, so they stay valid until they expire even after logout or a password change.

### Interactive docs (Swagger)

auth-service and ticket-service serve Swagger UI when `ENABLE_SWAGGER=true` (off by default, and off in Kubernetes):

- auth: `http://localhost:8081/swagger/index.html`
- ticket: `http://localhost:8082/swagger/index.html`

Both describe the API as clients see it, through the gateway (`localhost:8080`, auth paths under `/auth`). "Try it out" therefore calls the gateway: add the docs page origin (for example `http://localhost:8081`) to the gateway's `ALLOWED_ORIGINS`, log in through `POST /auth/login`, and enter `Bearer <access_token>` (with the word `Bearer`) in the Authorize dialog. notification-service (WebSocket) and the gateway itself are not covered.

The generated files (`docs/` in each service) are committed. After changing a handler annotation, regenerate them from the service directory:

```bash
go install github.com/swaggo/swag/cmd/swag@v1.16.6
swag init -g cmd/main.go
```

## Run the Flutter app

The Android emulator reaches the gateway at `10.0.2.2` by default:

```bash
cd flutter_app/helpdesk_app
flutter pub get
flutter run
```

For desktop, web or a physical device, override the gateway URL:

```bash
flutter run --dart-define=API_BASE_URL=http://localhost:8080
```

## Testing

```bash
cd backend/auth-service && go test ./...
cd backend/ticket-service && go test ./...
cd backend/notification-service && go test ./...
cd backend/api-gateway && go test ./...

cd flutter_app/helpdesk_app && flutter analyze && flutter test
```

`-race` needs cgo; on Windows run it in a Linux container. Opt-in integration tests are skipped unless pointed at real services: `POSTGRES_TEST_DSN` (a superuser DSN; auth-service repository tests create and drop a scratch database) and `RABBITMQ_TEST_URL` (ticket-service publisher and notification-service consumer). CI runs the unit tests and Flutter checks, not the integration tests.

## Troubleshooting

- **Database connection timeout**: services retry 10 times, 2 seconds apart. Make sure `POSTGRES_USER` matches `DB_USER`.
- **WebSocket fails on a physical Android device**: point `WS_URL` at your machine's LAN IP (for example `ws://192.168.1.5:8083/ws`), not `10.0.2.2`.
- **403 Forbidden**: check the gateway logs to confirm the JWT claims are extracted and forwarded.

## Before deploying

Set `ALLOWED_ORIGINS`, rotate `JWT_SECRET` and the other secrets, and give Caddy a real domain. Still open: centralized logs and tracing, alerting, NetworkPolicies, and TLS to the database (`sslmode=disable` today).
