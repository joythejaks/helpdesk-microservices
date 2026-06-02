import 'package:flutter/material.dart';

import 'package:helpdesk_app/core/theme/helpdesk_theme.dart';
import 'package:helpdesk_app/models/ticket.dart';
import 'package:helpdesk_app/presentation/widgets/app_frame.dart';
import 'package:helpdesk_app/presentation/widgets/detail_row.dart';
import 'package:helpdesk_app/presentation/widgets/header_bar.dart';
import 'package:helpdesk_app/presentation/widgets/status_chip.dart';
import 'package:helpdesk_app/presentation/widgets/surface_card.dart';
import 'package:helpdesk_app/presentation/widgets/timeline_item.dart';

class TicketDetailScreen extends StatelessWidget {
  const TicketDetailScreen({super.key, required this.ticket});

  final Ticket ticket;

  @override
  Widget build(BuildContext context) {
    return AppFrame(
      child: Scaffold(
        backgroundColor: Colors.transparent,
        body: ListView(
          padding: const EdgeInsets.fromLTRB(24, 42, 20, 32),
          children: [
            HeaderBar(
              title: 'Ticket Detail',
              subtitle: '#${ticket.id}',
              leading: Icons.arrow_back,
              onLeadingTap: () => Navigator.of(context).pop(),
            ),
            const SizedBox(height: 28),
            SurfaceCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      StatusChip(text: ticket.status),
                      const SizedBox(width: 8),
                      StatusChip(text: ticket.priority),
                    ],
                  ),
                  const SizedBox(height: 18),
                  Text(
                    ticket.title,
                    style: Theme.of(context).textTheme.headlineMedium,
                  ),
                  const SizedBox(height: 12),
                  Text(
                    ticket.description.isNotEmpty
                        ? ticket.description
                        : 'Tidak ada deskripsi.',
                    style: const TextStyle(
                      color: HelpdeskTheme.onVariant,
                      height: 1.5,
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 16),
            SurfaceCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  DetailRow(label: 'ID Tiket', value: '#${ticket.id}'),
                  DetailRow(label: 'Status', value: ticket.status),
                  DetailRow(label: 'Prioritas', value: ticket.priority),
                  DetailRow(label: 'Dibuat', value: ticket.time),
                ],
              ),
            ),
            const SizedBox(height: 16),
            SurfaceCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    'Timeline',
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                  const SizedBox(height: 16),
                  TimelineItem(title: 'Ticket dibuat', time: ticket.time),
                  if (ticket.status == 'In Progress' ||
                      ticket.status == 'Resolved')
                    const TimelineItem(
                      title: 'Ditugaskan ke agent',
                      time: 'Dalam proses',
                    ),
                  if (ticket.status == 'Resolved')
                    const TimelineItem(
                      title: 'Ticket diselesaikan',
                      time: 'Selesai',
                    ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}