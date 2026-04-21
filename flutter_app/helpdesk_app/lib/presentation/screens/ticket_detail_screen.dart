import 'package:flutter/material.dart';

import '../../core/theme/helpdesk_theme.dart';
import '../../models/ticket.dart';
import '../widgets/app_frame.dart';
import '../widgets/detail_row.dart';
import '../widgets/gradient_button.dart';
import '../widgets/header_bar.dart';
import '../widgets/status_chip.dart';
import '../widgets/surface_card.dart';
import '../widgets/timeline_item.dart';

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
              subtitle: ticket.id,
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
                    ticket.description,
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
                  DetailRow(label: 'Requester', value: ticket.requester),
                  DetailRow(label: 'Department', value: ticket.department),
                  DetailRow(label: 'Created', value: ticket.time),
                ],
              ),
            ),
            const SizedBox(height: 16),
            const SurfaceCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  TimelineItem(title: 'Ticket dibuat', time: '09:10'),
                  TimelineItem(title: 'Ditugaskan ke agent', time: '09:18'),
                  TimelineItem(title: 'Diagnosa awal berjalan', time: '09:42'),
                ],
              ),
            ),
            const SizedBox(height: 22),
            GradientButton(
              label: 'Update Status',
              icon: Icons.sync,
              onPressed: () {},
            ),
          ],
        ),
      ),
    );
  }
}
