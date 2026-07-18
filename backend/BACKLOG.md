# Backend Backlog

Gaps identified after building the admin/agent/user ticket workflow
(status lifecycle, assignment, reporting, staff provisioning, targeted
notifications — see git history / plan at
`C:\Users\joyth\.claude\plans\cuddly-marinating-rivest.md` for that work).
Not yet scheduled — pick items off here as needed.

## Critical — blocks frontend work

- [x] **`GET /tickets/:id/history`** (ticket-service) — added, same
      ownership rules as `GET /tickets/:id` (admin: any, agent: if
      assigned, user: if owner). Returns the `TicketStatusHistory` audit
      trail oldest-first.
- [x] **`GET /me`** (auth-service, proxied at `/auth/me`) — returns the
      caller's own `{id, email, role}`, resolved from `X-User-ID`. No
      password hash in the response.
- [x] **List-agents endpoint** (`GET /admin/agents` in auth-service, proxied
      at `/auth/admin/agents`, admin-only) — returns every account with
      `role=agent` so an admin can pick an `agent_id` to assign.
- [x] **`Requester` removed from ticket creation.** It duplicated
      `UserID` (the actual trusted identity) and could be set to anything
      by the client. `CreateTicketRequest` no longer accepts it — a
      requester's display name should be resolved from `UserID` via
      `GET /me` / `GET /admin/agents` on the frontend once needed.
      `Department` was kept as-is: it's a ticket category/routing tag
      (e.g. "IT", "HR"), not the requester's identity, so it doesn't have
      the same trust problem.

## Important — real functional gaps

- [ ] **Ticket comments/notes thread.** The `pending` status
      ("waiting on user for more info") implies back-and-forth
      communication that doesn't exist yet — tickets only have a single
      `Description` set at creation. Needs: new table (ticket_id, author_id,
      body, created_at), `POST/GET /tickets/:id/comments`, and probably a
      new notification event type (`ticket_commented`).
- [ ] **Notification persistence.** Delivery is WebSocket-only right now
      (`notification-service/internal/delivery/ws`) — if a user isn't
      connected when an event fires, it's gone. Needs a notifications table
      + `GET /notifications` (+ read/unread state) so the frontend can show
      a bell icon with a backlog, not just live pushes.
- [ ] **Full end-to-end verification with Docker running.** Everything this
      session was validated via `go build`/`go vet`/unit tests and isolated
      httptest smoke tests — the Docker daemon wasn't available, so the full
      stack (`docker compose up`) has never actually been run together.
      Worth doing a real pass — register → login → create ticket → assign →
      transition status → confirm notification arrives — once Docker is
      available.

## Production infrastructure (not new services)

Service count (4) is right for this project's scope — splitting further
(e.g. a separate `user-service`) would add operational overhead without a
real benefit. What's actually missing for "production-style" is
cross-cutting infra, not more domain services:

- [ ] **Observability** — metrics/centralized logs/tracing across the 4
      services (currently structured logs to stdout only, per-service, not
      aggregated). RabbitMQ decouples ticket-service from
      notification-service, which makes failures there hard to see without
      tracing tying a request across the hop.
- [ ] **TLS / reverse proxy in front of the gateway** — currently plain
      HTTP. Needs Nginx/Traefik/Caddy or TLS termination at the load
      balancer before this is internet-facing.
- [ ] **`notification-service` can't horizontally scale as-is.** WebSocket
      connections live in an in-memory map local to the process
      (`internal/delivery/ws/ws.go`, the `clients` var). RabbitMQ's default
      competing-consumers behavior means each published event goes to only
      *one* replica — if the target user is connected to a different
      replica, they never get it. Only matters once this service needs
      more than one instance; fix is a fanout exchange (every replica gets
      every message, filters locally) or a shared broadcast layer (e.g.
      Redis pub/sub) between replicas.

## Nice-to-have — later

- [ ] File attachments on tickets (screenshots/logs).
- [ ] SLA/due-date tracking per priority.
- [ ] Full-text search on ticket title/description.
- [ ] Replace `db.AutoMigrate` with a real migration tool
      (e.g. `golang-migrate`) once schema changes start needing renames/
      backfills instead of purely additive columns.
