import 'package:flutter/material.dart';

import 'package:helpdesk_app/models/app_user.dart';
import 'package:helpdesk_app/presentation/screens/agent/agent_shell.dart';
import 'package:helpdesk_app/presentation/screens/user/dashboard_shell.dart';

/// Single source of truth for "which shell does this role land on after
/// login" — used by both SplashScreen (returning session) and LoginScreen
/// (fresh login) so they can't drift apart again.
Widget homeForRole(AppUser user) {
  switch (user.role.toLowerCase()) {
    case 'agent':
    case 'admin':
      // No dedicated admin dashboard yet (separate backlog item) — an
      // admin can see everything an agent can, so the agent shell is a
      // reasonable stand-in until that's built.
      return const AgentShell();
    default:
      return const DashboardShell();
  }
}
