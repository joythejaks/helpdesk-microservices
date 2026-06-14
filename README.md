# Helpdesk Microservices

Helpdesk Microservices is a portfolio-grade ticketing system with Go microservices, an API gateway, RabbitMQ notifications, PostgreSQL persistence, and a Flutter client built with BLoC.

## 🏗️ Architecture Overview

The system is designed using a microservices pattern, communicating via REST and asynchronous events:

- **API Gateway**: Public entrypoint, JWT validation, and reverse proxy.
- **Auth Service**: Identity management, JWT issuance, and token rotation.
- **Ticket Service**: Ticket lifecycle management with RabbitMQ event publishing.
- **Notification Service**: Real-time updates via WebSockets and RabbitMQ consumption.
- **Flutter App**: Cross-platform mobile client with Clean Architecture and BLoC pattern.

## ✨ Key Features

- **Secure Auth**: JWT-based authentication with Refresh Token rotation (Security best practice).
- **Real-time Notifications**: Instant updates when ticket status changes.
- **Event-Driven**: Ticket creation triggers background notifications via RabbitMQ.
- **Observability**: Structured JSON logging and `/health` endpoints for all services.
- **Resilience**: WebSocket with exponential backoff and graceful shutdown support.
- **Clean UI**: Modern Flutter interface with dark/light theme support.
- **Scalable**: Services are containerized and ready for horizontal scaling.

## 🛠️ Tech Stack

- Go, Gin, GORM
- PostgreSQL
- RabbitMQ
- Docker Compose
- Flutter, BLoC, Flutter Secure Storage

## 🚀 Deployment & Operations

### Run Full Stack (Docker)

Ensure you have Docker and Docker Compose installed, then run:

```bash
docker compose up --build
```

Default ports:

- API Gateway: `http://localhost:8080`
- Auth Service: `http://localhost:8081`
- Ticket Service: `http://localhost:8082`
- Notification Service: `http://localhost:8083`
- RabbitMQ Management: `http://localhost:15672`

RabbitMQ credentials can be overridden from the shell or a compose `.env` file:

```bash
RABBITMQ_DEFAULT_USER=guest
RABBITMQ_DEFAULT_PASS=guest
RABBITMQ_ERLANG_COOKIE=secretcookie
```

## Run Flutter App

Android emulator uses the default `10.0.2.2` gateway address:

```bash
cd flutter_app/helpdesk_app
flutter pub get
flutter run
```

For desktop, web, or a physical device, override the gateway URL:

```bash
flutter run --dart-define=API_BASE_URL=http://localhost:8080
```

## API Overview

Public auth endpoints:

- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/refresh`

Protected endpoints:

- `POST /auth/logout`
- `GET /tickets`
- `POST /tickets`
- `GET /tickets/:id`

Protected routes require:

```http
Authorization: Bearer <access_token>
```

## 🛠️ Common Troubleshooting

**1. Database Connection Timeout**
If the services start before PostgreSQL is ready, they will retry 10 times with a 2-second delay. Ensure the `POSTGRES_USER` matches the `DB_USER` in your `.env`.

**2. WebSocket Connection Failure**
If running on a physical Android device, ensure the `WS_URL` points to your machine's local IP (e.g., `ws://192.168.1.5:8083/ws`) instead of `10.0.2.2`.

**3. Authorization 403 Forbidden**
Ensure the API Gateway is correctly forwarding the `X-User-Role` header. Check the Gateway logs to verify JWT claim extraction.

## 🛡️ Production Hardening

- [ ] **SSL/TLS**: Always wrap the API Gateway with Nginx or a Load Balancer (like Cloudflare) to enable HTTPS.
- [ ] **Secrets**: Use a Secret Manager (Vault/AWS Secrets) instead of plain `.env` files in actual production.
- [ ] **Rate Limiting**: Enable rate limiting at the Gateway level to prevent DDoS.
- [ ] **Logs**: Forward JSON logs to a centralized collector (ELK or Grafana Loki).

## 🧪 Testing

Run Go tests:

```bash
cd backend/auth-service && go test ./...
cd ../ticket-service && go test ./...
```

Run Flutter checks:

```bash
cd flutter_app/helpdesk_app
flutter analyze
flutter test
```
