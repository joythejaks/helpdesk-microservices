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

- [x] **Ticket comments/notes thread.** `TicketComment` (ticket_id,
      author_id, author_role, body, created_at), `POST/GET
      /tickets/:id/comments` in ticket-service, authorization reuses
      `TicketUsecase.GetTicketByID` (owner/assigned agent/admin only).
      Posting publishes a `ticket_commented` event: staff commenting
      notifies the owner, the owner commenting notifies the assigned agent
      (or the `admin` role if the ticket isn't assigned yet).
- [x] **Notification persistence.** notification-service now has its own
      Postgres DB (`notification-db` in `docker-compose.yml`) and a
      `Notification` table. `GET /notifications` (+ `unread_only`
      filter), `PATCH /notifications/:id/read`, `PATCH
      /notifications/read-all` — self-authenticated via `Authorization:
      Bearer` (this service was never behind the gateway to begin with,
      since the gateway's reverse proxy doesn't handle WS upgrades; REST
      now follows the same trust model `/ws` already used). Scope
      decision: only events with a concrete `TargetUserID` are persisted
      (ticket_assigned, ticket_status_changed, ticket_commented-to-one-
      person) — role broadcasts (`TargetRoles`, e.g. new-ticket pings to
      admin+agent) stay WebSocket-only, since persisting those would need
      this service to know the full admin/agent roster, which lives in
      auth-service's DB, not this one.
- [x] **Full end-to-end verification with Docker running.** Docker is
      reachable now. `docker compose up --build` initially failed on
      **`api-gateway`** — its `Dockerfile` built `./cmd/main.go` (a single
      file) instead of the whole `./cmd` package, so `middleware.go`
      (rate limiter, request-ID/internal-secret middleware, self-
      healthcheck) silently wasn't compiled in — undefined-symbol build
      errors. The other three services only ever had one file in `cmd/`,
      so the same pattern happened to work for them. Fixed to
      `go build -o main ./cmd`. With that fix, the full stack builds and
      ran the real flow via `curl` against `localhost:8080`: register →
      (promoted to admin directly in `auth-db` — there's still no
      bootstrap path for the very first admin, see below) → `POST
      /auth/admin/staff` → login as user/agent/admin → create ticket →
      agent self-claim via `scope=queue` → status transition → comments
      both directions → `GET /reports/summary` / `GET /reports/agents` /
      `GET /auth/admin/agents` as admin → `POST /auth/refresh`. Every
      response matched its documented shape exactly. Did not re-verify
      notification persistence/mark-read in this pass (that's the
      notification-service's own REST API, unrelated to this round of
      changes).
- [ ] **No bootstrap path for the first admin account.** `POST
      /auth/register` always forces `role="user"` (correct — role must
      never come from the client) and `POST /auth/admin/staff` requires
      an existing admin to call it, so there's no way to create the
      *first* admin without reaching directly into `auth-db`. Fine for
      dev; worth a one-time seed/CLI/env-var-gated bootstrap before this
      goes anywhere real.

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

- [x] **File attachments on tickets.** Stored inline in Postgres (`bytea`
      column on `TicketAttachment`, 5MB cap enforced in
      `AttachmentUsecase`) rather than adding object storage — kept in the
      "no new infra" spirit already established for this project's scale.
      `POST/GET /tickets/:id/attachments` (metadata-only list, blob never
      loaded for listing), `GET /tickets/:id/attachments/:attachmentId`
      (download). Same ownership rules as comments, and uploading
      publishes `ticket_attachment_added` through the same targeting logic
      (now shared between comments and attachments as
      `ticketActivityTarget` in `internal/delivery/http/activity_target.go`).
- [x] **SLA/due-date tracking per priority.** `Ticket.DueAt` is computed at
      creation from a `slaDurations` map in `ticket_usecase.go` (High: 4h,
      Medium: 24h, Low: 72h — easy to tune later). `GET /tickets?overdue=true`
      filters to tickets past `DueAt` that aren't resolved/closed yet.
- [x] **Full-text search on ticket title/description.** `GET
      /tickets?search=` does an `ILIKE` match against both columns —
      deliberately not real Postgres full-text search (`tsvector`/`GIN`
      index); `ILIKE` is plenty for this project's ticket volume and adds
      no schema complexity. Worth upgrading if ticket volume ever makes
      `ILIKE` scans slow.
- [ ] Replace `db.AutoMigrate` with a real migration tool
      (e.g. `golang-migrate`) once schema changes start needing renames/
      backfills instead of purely additive columns. Deliberately not done
      this session — none of the changes above needed a rename/backfill,
      and adopting migration tooling blind (Docker unavailable to verify
      against a real Postgres) risked breaking every service's startup for
      no concrete near-term need.
