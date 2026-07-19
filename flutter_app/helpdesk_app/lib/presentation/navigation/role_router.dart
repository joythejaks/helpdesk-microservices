import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/data/admin_repository.dart';
import 'package:helpdesk_app/models/app_user.dart';
import 'package:helpdesk_app/presentation/bloc/admin/admin_reports_bloc.dart';
import 'package:helpdesk_app/presentation/screens/admin/admin_shell.dart';
import 'package:helpdesk_app/presentation/screens/agent/agent_shell.dart';
import 'package:helpdesk_app/presentation/screens/user/dashboard_shell.dart';

/// Single source of truth for "which shell does this role land on after
/// login" — used by both SplashScreen (returning session) and LoginScreen
/// (fresh login) so they can't drift apart again.
Widget homeForRole(AppUser user) {
  switch (user.role.toLowerCase()) {
    case 'admin':
      // AdminReportsBloc is shell-scoped (created only for admin sessions)
      // rather than in main.dart's app-wide MultiBlocProvider, since
      // non-admin sessions never need it.
      return BlocProvider(
        create: (context) =>
            AdminReportsBloc(adminRepository: context.read<AdminRepository>()),
        child: const AdminShell(),
      );
    case 'agent':
      return const AgentShell();
    default:
      return const DashboardShell();
  }
}
