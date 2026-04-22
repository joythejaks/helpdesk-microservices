import 'package:flutter/material.dart';

import 'package:helpdesk_app/models/ticket.dart';
import 'status_chip.dart';
import 'surface_card.dart';

class TicketCard extends StatelessWidget {
  const TicketCard({super.key, required this.ticket, required this.onTap});

  final Ticket ticket;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: SurfaceCard(
        onTap: onTap,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                StatusChip(text: ticket.status),
                const SizedBox(width: 8),
                StatusChip(text: ticket.priority),
                const Spacer(),
                Text(ticket.time, style: Theme.of(context).textTheme.bodySmall),
              ],
            ),
            const SizedBox(height: 16),
            Text(ticket.title, style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 6),
            Text(
              '${ticket.requester} - ${ticket.department}',
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ],
        ),
      ),
    );
  }
}
