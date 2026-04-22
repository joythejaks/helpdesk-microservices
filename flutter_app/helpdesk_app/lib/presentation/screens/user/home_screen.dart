import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/models/ticket.dart';
import 'package:helpdesk_app/presentation/bloc/ticket/ticket_bloc.dart';
import 'package:helpdesk_app/presentation/widgets/gradient_button.dart';
import 'package:helpdesk_app/presentation/widgets/header_bar.dart';
import 'package:helpdesk_app/presentation/widgets/metric_card.dart';
import 'package:helpdesk_app/presentation/widgets/ticket_card.dart';

class HomeScreen extends StatelessWidget {
  const HomeScreen({
    super.key,
    required this.onOpenTicket,
    required this.onCreate,
  });

  final ValueChanged<Ticket> onOpenTicket;
  final VoidCallback onCreate;

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
        final openCount = tickets
            .where((ticket) => ticket.status != 'Resolved')
            .length;
        final resolvedCount = tickets
            .where((ticket) => ticket.status == 'Resolved')
            .length;

        return RefreshIndicator(
          onRefresh: () async {
            context.read<TicketBloc>().add(const TicketsRequested());
          },
          child: ListView(
            padding: const EdgeInsets.fromLTRB(24, 42, 20, 112),
            children: [
              HeaderBar(
                title: 'Selamat pagi, Agent',
                subtitle: '$openCount tiket aktif perlu dipantau hari ini',
                trailing: Icons.notifications_none,
              ),
              const SizedBox(height: 26),
              Row(
                children: [
                  Expanded(
                    child: MetricCard(
                      value: '$openCount',
                      label: 'Aktif',
                      icon: Icons.confirmation_number_outlined,
                    ),
                  ),
                  const SizedBox(width: 12),
                  const Expanded(
                    child: MetricCard(
                      value: '0',
                      label: 'Urgent',
                      icon: Icons.priority_high,
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 12),
              Row(
                children: [
                  const Expanded(
                    child: MetricCard(
                      value: '91%',
                      label: 'SLA',
                      icon: Icons.speed_outlined,
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: MetricCard(
                      value: '$resolvedCount',
                      label: 'Selesai',
                      icon: Icons.task_alt,
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 26),
              GradientButton(
                label: 'Buat Ticket Baru',
                icon: Icons.add,
                onPressed: onCreate,
              ),
              const SizedBox(height: 28),
              Text(
                'Tiket Prioritas',
                style: Theme.of(context).textTheme.titleLarge,
              ),
              const SizedBox(height: 12),
              if (state is TicketLoading)
                const Center(child: CircularProgressIndicator())
              else if (tickets.isEmpty)
                const _EmptyTicketMessage()
              else
                ...tickets
                    .take(3)
                    .map(
                      (ticket) => TicketCard(
                        ticket: ticket,
                        onTap: () => onOpenTicket(ticket),
                      ),
                    ),
            ],
          ),
        );
      },
    );
  }
}

class _EmptyTicketMessage extends StatelessWidget {
  const _EmptyTicketMessage();

  @override
  Widget build(BuildContext context) {
    return const Padding(
      padding: EdgeInsets.symmetric(vertical: 24),
      child: Center(child: Text('Belum ada ticket dari backend.')),
    );
  }
}
