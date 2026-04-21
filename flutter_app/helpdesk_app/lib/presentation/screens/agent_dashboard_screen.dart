import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../../models/ticket.dart';
import '../bloc/ticket/ticket_bloc.dart';
import '../widgets/header_bar.dart';
import '../widgets/progress_line.dart';
import '../widgets/progress_ticket_card.dart';
import '../widgets/surface_card.dart';

class AgentDashboardScreen extends StatelessWidget {
  const AgentDashboardScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return BlocBuilder<TicketBloc, TicketState>(
      builder: (context, state) {
        final tickets = switch (state) {
          TicketLoaded(:final tickets) => tickets,
          TicketCreating(:final tickets) => tickets,
          TicketFailure(:final tickets) => tickets,
          _ => <Ticket>[],
        };

        return ListView(
          padding: const EdgeInsets.fromLTRB(24, 42, 20, 112),
          children: [
            const HeaderBar(
              title: 'Agent Dashboard',
              subtitle: 'Ringkasan performa dan antrean kerja',
              trailing: Icons.account_circle_outlined,
            ),
            const SizedBox(height: 26),
            SurfaceCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    'SLA Response',
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                  const SizedBox(height: 18),
                  const ProgressLine(label: 'Network', value: .86),
                  const ProgressLine(label: 'Account', value: .72),
                  const ProgressLine(label: 'Hardware', value: .54),
                ],
              ),
            ),
            const SizedBox(height: 16),
            Text(
              'My Tickets Progress',
              style: Theme.of(context).textTheme.titleLarge,
            ),
            const SizedBox(height: 12),
            if (state is TicketLoading)
              const Center(child: CircularProgressIndicator())
            else if (tickets.isEmpty)
              const Center(child: Text('Belum ada progress ticket.'))
            else
              ...tickets.map((ticket) => ProgressTicketCard(ticket: ticket)),
          ],
        );
      },
    );
  }
}
