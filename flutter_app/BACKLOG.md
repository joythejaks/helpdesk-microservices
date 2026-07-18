# Flutter App Backlog

Findings from reviewing `helpdesk_app/` against the backend's admin/agent/
user ticket workflow (see `backend/BACKLOG.md` and the plan at
`C:\Users\joyth\.claude\plans\cuddly-marinating-rivest.md`). Confirmed with
`flutter analyze` — not yet scheduled, not yet executed.

## Critical — app doesn't build / navigation is broken

- [ ] **Fix the compile error.** `lib/presentation/screens/auth/login_screen.dart:42`
      reads `state.user.role`, but `Authenticated` in
      `lib/presentation/bloc/auth/auth_bloc.dart` has no `user` field at
      all. `flutter analyze` confirms: `error - The getter 'user' isn't
      defined for the type 'Authenticated'`. The app cannot currently run.
- [ ] **`AuthBloc` needs to carry the logged-in user's identity (id, email,
      role).** This is the root cause of the bug above and blocks every
      other role-based decision in the app. Two ways to get the role:
      decode it out of the JWT client-side (it's already in the access
      token's claims, no backend change needed), or wait for the backend's
      planned `GET /me` (in `backend/BACKLOG.md`, not built yet). Worth
      deciding which before touching `login_screen.dart`.
- [ ] **`SplashScreen` ignores session state.** It unconditionally
      navigates to `/login` after a 1.5s delay
      (`lib/presentation/screens/auth/splash_screen.dart`) — it never
      listens to `AuthBloc`'s `AuthStarted` result (`Authenticated` vs
      `Unauthenticated`), even though that's already dispatched in
      `main.dart`. A user with a valid stored token is forced to log in
      again every time the app opens.
- [ ] **Two conflicting navigation paths exist.** `main.dart`'s `/dashboard`
      route goes to `DashboardShell`, which mixes `HomeScreen`,
      `TicketListScreen`, `CreateTicketScreen`, and `AgentDashboardScreen`
      into one bottom-nav-bar with no role gating at all. Separately,
      `login_screen.dart` manually pushes `TicketListScreen` (agent) or
      `HomeScreen` (everyone else) directly, bypassing `DashboardShell`
      entirely. These two need to be reconciled into one coherent,
      role-aware navigation flow before adding admin screens on top.

## Important — doesn't match the 3-role backend built this session

- [ ] **No admin role/UI exists at all.** Login only branches "agent" vs
      everything else; there's no admin dashboard, no reports screen. The
      backend has `GET /reports/summary` and `GET /reports/agents`
      (admin-only) with nothing consuming them.
- [ ] **No staff-provisioning screen** for `POST /auth/admin/staff` — an
      admin has no in-app way to create an agent account.
- [ ] **No ticket assignment UI** — `PATCH /tickets/:id/assign` (agent
      self-claim from the unassigned queue, or admin assigning to a
      specific agent) isn't called from anywhere.
- [ ] **No status transition UI** — `PATCH /tickets/:id/status` isn't
      called from anywhere either. The "Percakapan & Aktivitas" section in
      `lib/presentation/screens/agent/ticket_detail_screen.dart` is
      hardcoded fake data (`_buildCommentTile('System', 'Tiket telah
      berhasil dibuat...')`), and the send button just clears the text
      field without calling any repository/bloc method.
- [ ] **`Ticket.fromJson` silently drops real data.** `priority`,
      `requester`, and `department` exist as constructor fields
      (`lib/models/ticket.dart`) but `fromJson` never reads them from the
      API response — every ticket displays the hardcoded default
      (`priority: 'Medium'`) regardless of its actual value. This is a data
      bug, not just a missing feature.
- [ ] **Status label mapping is stale.** `Ticket._formatStatus` only knows
      about `open`/`in_progress`/`resolved`/`closed` — doesn't handle the
      `assigned` or `pending` statuses added to the backend's state machine
      this session.
- [ ] **`TicketRepository.getTickets()` sends no scope/filter params.** The
      backend now needs `scope=mine|queue` for agents and
      `status`/`priority`/`department`/`from`/`to` filters for admin — the
      client only ever sends `page`/`limit`.
- [ ] **`WebSocketService` is fully built but never instantiated anywhere**
      (confirmed via grep — only referenced inside its own file). All of
      this session's backend work on targeted WebSocket notifications
      (`SendToUser`/`SendToRoles`, structured JSON events) has zero
      consumer on the Flutter side — no live ticket updates, no
      notification bell/badge.
- [ ] **No token refresh flow.** `ApiClient` doesn't handle a 401 by
      calling `/auth/refresh` and retrying; `AuthRepository` never calls
      the refresh endpoint at all despite storing the refresh token. Once
      the 2-hour access token expires mid-session, requests just start
      failing with no recovery.

## Nice-to-have — polish

- [ ] Client-side password validation says "minimal 6 karakter"
      (`login_screen.dart`, `register_screen.dart`) but the backend now
      requires `min=8` — bumping this avoids a round-trip just to get
      rejected with a mismatched message.
- [ ] Lint cleanup from `flutter analyze`: 3x `avoid_print` in
      `websocket_service.dart` (swap for a real logger), 2 unused imports
      (`gradient_button.dart` in `ticket_detail_screen.dart`,
      `agent_dashboard_screen.dart` in `login_screen.dart`), missing curly
      braces in `home_screen.dart:135`.
- [ ] `AgentDashboardScreen`'s "SLA Response" numbers are hardcoded
      (`ProgressLine(label: 'Network', value: .86)`, etc.) — already has a
      `// TODO: fetch SLA data from analytics endpoint` comment. Wire to
      `GET /reports/agents` once the assignment/status UI above exists.
- [ ] Navigation is currently a mix of named routes (`main.dart`) and ad-hoc
      `Navigator.push(MaterialPageRoute(...))` calls scattered across
      screens. Worth a decision once admin screens are added: keep growing
      the named-route table, or bring in a router package
      (e.g. `go_router`) for role-based route guards.
- [ ] "Attach file" UI in `create_ticket_screen.dart` is decorative only
      (icon + text, no picker) — consistent with attachments not existing
      on the backend yet either (`backend/BACKLOG.md`), not a bug, just
      unimplemented.
