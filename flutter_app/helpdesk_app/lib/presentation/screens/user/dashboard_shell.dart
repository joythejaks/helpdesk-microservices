import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/models/ticket.dart';
import 'package:helpdesk_app/presentation/bloc/ticket/ticket_bloc.dart';
import 'package:helpdesk_app/presentation/widgets/app_frame.dart';
import 'package:helpdesk_app/presentation/widgets/glass_nav_bar.dart';
import 'package:helpdesk_app/presentation/screens/user/create_ticket_screen.dart';
import 'package:helpdesk_app/presentation/screens/user/home_screen.dart';
import 'package:helpdesk_app/presentation/screens/agent/ticket_list_screen.dart';
import 'package:helpdesk_app/presentation/screens/profile/profile_screen.dart';

const _navItems = [
  (Icons.home_outlined, 'Home'),
  (Icons.confirmation_number_outlined, 'Tickets'),
  (Icons.add_circle_outline, 'Create'),
  (Icons.person_outline, 'Profile'),
];

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
      const ProfileScreen(),
    ];

    final colors = Theme.of(context).colorScheme;
    return AppFrame(
      child: Scaffold(
        backgroundColor: Colors.transparent,
        body: pages[index],
        floatingActionButton: (index == 0 || index == 1)
            ? FloatingActionButton(
                onPressed: () => setState(() => index = 2),
                backgroundColor: colors.primary,
                foregroundColor: colors.onPrimary,
                child: const Icon(Icons.add),
              )
            : null,
        bottomNavigationBar: GlassNavBar(
          index: index,
          onChanged: (value) => setState(() => index = value),
          items: _navItems,
        ),
      ),
    );
  }

  void _openTicket(Ticket ticket) {
    Navigator.of(context).pushNamed('/ticket-detail', arguments: ticket);
  }
}
