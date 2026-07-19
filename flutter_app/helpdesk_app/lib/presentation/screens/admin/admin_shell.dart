import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/models/ticket.dart';
import 'package:helpdesk_app/presentation/bloc/admin/admin_reports_bloc.dart';
import 'package:helpdesk_app/presentation/bloc/ticket/ticket_bloc.dart';
import 'package:helpdesk_app/presentation/screens/admin/admin_dashboard_screen.dart';
import 'package:helpdesk_app/presentation/screens/admin/staff_provisioning_screen.dart';
import 'package:helpdesk_app/presentation/screens/agent/ticket_detail_screen.dart';
import 'package:helpdesk_app/presentation/screens/agent/ticket_list_screen.dart';
import 'package:helpdesk_app/presentation/widgets/app_frame.dart';
import 'package:helpdesk_app/presentation/widgets/glass_nav_bar.dart';

const _navItems = [
  (Icons.dashboard_outlined, 'Reports'),
  (Icons.confirmation_number_outlined, 'Tickets'),
  (Icons.person_add_alt_outlined, 'Staff'),
];

/// Shell for admin logins — reports/tickets/staff-provisioning tabs. Reuses
/// the existing TicketListScreen/TicketDetailScreen for the Tickets tab
/// (admin sees every ticket purely via server-side role scoping — no
/// client-side special-casing needed).
class AdminShell extends StatefulWidget {
  const AdminShell({super.key});

  @override
  State<AdminShell> createState() => _AdminShellState();
}

class _AdminShellState extends State<AdminShell> {
  int index = 0;

  @override
  void initState() {
    super.initState();
    context.read<TicketBloc>().add(const TicketsRequested());
    context.read<AdminReportsBloc>().add(const AdminReportsRequested());
  }

  @override
  Widget build(BuildContext context) {
    final pages = [
      const AdminDashboardScreen(),
      TicketListScreen(onOpenTicket: _openTicket),
      const StaffProvisioningScreen(),
    ];

    return AppFrame(
      child: Scaffold(
        backgroundColor: Colors.transparent,
        body: pages[index],
        bottomNavigationBar: GlassNavBar(
          index: index,
          onChanged: (value) => setState(() => index = value),
          items: _navItems,
        ),
      ),
    );
  }

  void _openTicket(Ticket ticket) {
    Navigator.of(context).push(
      MaterialPageRoute(builder: (_) => TicketDetailScreen(ticket: ticket)),
    );
  }
}
