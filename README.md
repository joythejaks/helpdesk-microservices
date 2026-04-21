# Helpdesk Microservices

Helpdesk Microservices is a portfolio-grade ticketing system with Go microservices, an API gateway, RabbitMQ notifications, PostgreSQL persistence, and a Flutter client built with BLoC.

## Architecture

- `api-gateway`: public entrypoint, JWT validation, and reverse proxy to internal services.
- `auth-service`: user registration, login, refresh token rotation, logout, and JWT issuance.
- `ticket-service`: ticket creation and ticket listing, with RabbitMQ event publishing.
- `notification-service`: RabbitMQ consumer and WebSocket notification delivery.
- `flutter_app/helpdesk_app`: Flutter mobile app using BLoC, repositories, token storage, and the gateway API.

## Tech Stack

- Go, Gin, GORM
- PostgreSQL
- RabbitMQ
- Docker Compose
- Flutter, BLoC, Shared Preferences

## Run Backend

```bash
cd backend
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

## Tests

Run Go tests per service:

```bash
cd backend/auth-service && go test ./...
cd ../ticket-service && go test ./...
cd ../notification-service && go test ./...
cd ../api-gateway && go test ./...
```

Run Flutter checks:

```bash
cd flutter_app/helpdesk_app
flutter analyze
flutter test
```
