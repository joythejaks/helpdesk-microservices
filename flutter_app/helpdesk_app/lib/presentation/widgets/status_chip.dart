import 'package:flutter/material.dart';

import '../../core/theme/helpdesk_theme.dart';

class StatusChip extends StatelessWidget {
  const StatusChip({super.key, required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    final colors = switch (text) {
      'High' => (HelpdeskTheme.errorContainer, const Color(0xFF93000A)),
      'Closed' => (HelpdeskTheme.surfaceHigh, HelpdeskTheme.onVariant),
      'Resolved' => (HelpdeskTheme.tertiaryFixed, const Color(0xFF002020)),
      'In Progress' => (HelpdeskTheme.primaryFixed, const Color(0xFF001D33)),
      'Pending' => (const Color(0xFFFFE0B2), const Color(0xFF7A4A00)),
      'Assigned' => (
        HelpdeskTheme.secondaryContainer,
        HelpdeskTheme.onSecondaryContainer,
      ),
      _ => (HelpdeskTheme.surfaceHigh, HelpdeskTheme.onVariant),
    };

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: colors.$1,
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        text,
        style: TextStyle(
          color: colors.$2,
          fontSize: 11,
          fontWeight: FontWeight.w800,
        ),
      ),
    );
  }
}
