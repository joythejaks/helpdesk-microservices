import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/models/ticket.dart';
import 'package:helpdesk_app/presentation/bloc/notification/notification_bloc.dart';
import 'package:helpdesk_app/presentation/bloc/ticket/ticket_bloc.dart';
import 'package:helpdesk_app/presentation/widgets/header_bar.dart';
import 'package:helpdesk_app/presentation/widgets/progress_line.dart';
import 'package:helpdesk_app/presentation/widgets/progress_ticket_card.dart';
import 'package:helpdesk_app/presentation/widgets/surface_card.dart';

class AgentDashboardScreen extends StatelessWidget {
  const AgentDashboardScreen({super.key});

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

        return ListView(
          padding: const EdgeInsets.fromLTRB(24, 42, 20, 112),
          children: [
            BlocBuilder<NotificationBloc, NotificationState>(
              builder: (context, notifState) => HeaderBar(
                title: 'Agent Dashboard',
                subtitle: 'Ringkasan performa dan antrean kerja',
                trailing: Icons.notifications_outlined,
                trailingBadgeCount: notifState.unreadCount,
                onTrailingTap: () {
                  context.read<NotificationBloc>().add(
                    const NotificationCleared(),
                  );
                  Navigator.of(context).pushNamed('/notifications');
                },
              ),
            ),
            const SizedBox(height: 26),
            Builder(
              builder: (context) {
                final sla = _computeSla(tickets);
                return SurfaceCard(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'SLA Response',
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                      const SizedBox(height: 18),
                      ProgressLine(
                        label: 'Selesai Tepat Waktu',
                        value: sla.onTimeRate,
                      ),
                      ProgressLine(
                        label: 'Tiket Aktif Overdue',
                        value: sla.overdueActiveRate,
                      ),
                    ],
                  ),
                );
              },
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

const _resolvedStatuses = {'resolved', 'closed'};
const _activeStatuses = {'assigned', 'in_progress', 'pending'};

({double onTimeRate, double overdueActiveRate}) _computeSla(
  List<Ticket> tickets,
) {
  var resolvedWithDue = 0;
  var resolvedOnTime = 0;
  var activeWithDue = 0;
  var activeOverdue = 0;
  final now = DateTime.now();

  for (final t in tickets) {
    if (_resolvedStatuses.contains(t.rawStatus) && t.dueAt != null) {
      resolvedWithDue++;
      final completedAt = t.resolvedAt ?? t.closedAt;
      if (completedAt != null && !completedAt.isAfter(t.dueAt!)) {
        resolvedOnTime++;
      }
    } else if (_activeStatuses.contains(t.rawStatus) && t.dueAt != null) {
      activeWithDue++;
      if (t.dueAt!.isBefore(now)) activeOverdue++;
    }
  }

  return (
    onTimeRate: resolvedWithDue == 0 ? 0 : resolvedOnTime / resolvedWithDue,
    overdueActiveRate: activeWithDue == 0 ? 0 : activeOverdue / activeWithDue,
  );
}
