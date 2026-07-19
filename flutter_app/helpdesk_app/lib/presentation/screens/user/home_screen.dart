import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/models/ticket.dart';
import 'package:helpdesk_app/presentation/bloc/notification/notification_bloc.dart';
import 'package:helpdesk_app/presentation/bloc/ticket/ticket_bloc.dart';
import 'package:helpdesk_app/presentation/widgets/gradient_button.dart';
import 'package:helpdesk_app/presentation/widgets/header_bar.dart';
import 'package:helpdesk_app/presentation/widgets/metric_card.dart';
import 'package:helpdesk_app/presentation/widgets/ticket_card.dart';

String _greeting() {
  final now = DateTime.now();
  final hour = now.hour;
  if (hour < 11) return 'Selamat Pagi';
  if (hour < 15) return 'Selamat Siang';
  if (hour < 19) return 'Selamat Sore';
  return 'Selamat Malam';
}

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
          TicketMutating(:final tickets) => tickets,
          TicketFailure(:final tickets) => tickets,
          _ => <Ticket>[],
        };

        // Pindahkan kalkulasi ini ke dalam helper atau getter agar build lebih bersih
        final metrics = _calculateMetrics(tickets);

        return RefreshIndicator(
          onRefresh: () async {
            context.read<TicketBloc>().add(const TicketsRequested());
          },
          child: ListView(
            padding: const EdgeInsets.fromLTRB(24, 42, 20, 112),
            children: [
              BlocBuilder<NotificationBloc, NotificationState>(
                builder: (context, notifState) => HeaderBar(
                  title: '${_greeting()}, User',
                  subtitle: '${metrics.open} tiket aktif perlu dipantau hari ini',
                  trailing: Icons.notifications_none,
                  trailingBadgeCount: notifState.unreadCount,
                  onTrailingTap: () =>
                      context.read<NotificationBloc>().add(const NotificationCleared()),
                ),
              ),
              const SizedBox(height: 26),
              Row(
                children: [
                  Expanded(
                    child: MetricCard(
                      value: '${metrics.open}',
                      label: 'Aktif',
                      icon: Icons.confirmation_number_outlined,
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: MetricCard(
                      value: '${metrics.urgent}',
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
                      // Placeholder SLA bisa diganti dengan rata-rata jam dari backend
                      value: '2.4h',
                      label: 'SLA',
                      icon: Icons.speed_outlined,
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: MetricCard(
                      value: '${metrics.resolved}',
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

  ({int open, int resolved, int urgent}) _calculateMetrics(List<Ticket> tickets) {
    int open = 0;
    int resolved = 0;
    int urgent = 0;

    for (var t in tickets) {
      if (t.status == 'Resolved') resolved++; else open++;
      if (t.priority == 'High') urgent++;
    }

    return (open: open, resolved: resolved, urgent: urgent);
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