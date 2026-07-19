import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/models/ticket.dart';
import 'package:helpdesk_app/presentation/bloc/ticket/ticket_bloc.dart';
import 'package:helpdesk_app/presentation/widgets/app_frame.dart';
import 'package:helpdesk_app/presentation/widgets/glass_nav_bar.dart';
import 'package:helpdesk_app/presentation/screens/agent/agent_dashboard_screen.dart';
import 'package:helpdesk_app/presentation/screens/agent/ticket_detail_screen.dart';
import 'package:helpdesk_app/presentation/screens/agent/ticket_list_screen.dart';

const _navItems = [
  (Icons.dashboard_outlined, 'Dashboard'),
  (Icons.confirmation_number_outlined, 'Tickets'),
];

/// Shell for agent (and, until a dedicated admin UI exists, admin) logins —
/// keeps the agent dashboard reachable alongside the ticket list, which
/// wasn't possible before (agents were pushed straight into a bare
/// TicketListScreen with no nav bar at all).
class AgentShell extends StatefulWidget {
  const AgentShell({super.key});

  @override
  State<AgentShell> createState() => _AgentShellState();
}

class _AgentShellState extends State<AgentShell> {
  int index = 0;

  @override
  void initState() {
    super.initState();
    context.read<TicketBloc>().add(const TicketsRequested());
  }

  @override
  Widget build(BuildContext context) {
    final pages = [
      const AgentDashboardScreen(),
      TicketListScreen(onOpenTicket: _openTicket),
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
