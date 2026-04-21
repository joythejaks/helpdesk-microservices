import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../../core/theme/helpdesk_theme.dart';
import '../../models/ticket.dart';
import '../bloc/ticket/ticket_bloc.dart';
import '../widgets/app_frame.dart';
import '../widgets/glass_nav_bar.dart';
import 'agent_dashboard_screen.dart';
import 'create_ticket_screen.dart';
import 'home_screen.dart';
import 'ticket_detail_screen.dart';
import 'ticket_list_screen.dart';

class DashboardShell extends StatefulWidget {
  const DashboardShell({super.key});

  @override
  State<DashboardShell> createState() => _DashboardShellState();
}

class _DashboardShellState extends State<DashboardShell> {
  int index = 0;

  @override
  void initState() {
    super.initState();
    context.read<TicketBloc>().add(const TicketsRequested());
  }

  @override
  Widget build(BuildContext context) {
    final pages = [
      HomeScreen(
        onOpenTicket: _openTicket,
        onCreate: () => setState(() => index = 2),
      ),
      TicketListScreen(onOpenTicket: _openTicket),
      const CreateTicketScreen(),
      const AgentDashboardScreen(),
    ];

    return AppFrame(
      child: Scaffold(
        backgroundColor: Colors.transparent,
        body: pages[index],
        floatingActionButton: index == 2
            ? null
            : FloatingActionButton(
                onPressed: () => setState(() => index = 2),
                backgroundColor: HelpdeskTheme.primary,
                foregroundColor: Colors.white,
                child: const Icon(Icons.add),
              ),
        bottomNavigationBar: GlassNavBar(
          index: index,
          onChanged: (value) => setState(() => index = value),
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
