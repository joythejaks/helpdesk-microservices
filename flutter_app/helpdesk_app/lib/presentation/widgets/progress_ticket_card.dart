import 'package:flutter/material.dart';

import '../../models/ticket.dart';
import 'progress_line.dart';
import 'surface_card.dart';

class ProgressTicketCard extends StatelessWidget {
  const ProgressTicketCard({super.key, required this.ticket});

  final Ticket ticket;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: SurfaceCard(
        child: ProgressLine(
          label: '${ticket.id}  ${ticket.title}',
          value: ticket.progress,
        ),
      ),
    );
  }
}
