import 'package:flutter/material.dart';

import '../../core/theme/helpdesk_theme.dart';

class TimelineItem extends StatelessWidget {
  const TimelineItem({super.key, required this.title, required this.time});

  final String title;
  final String time;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 16),
      child: Row(
        children: [
          const Icon(Icons.check_circle, color: HelpdeskTheme.primaryContainer),
          const SizedBox(width: 12),
          Expanded(
            child: Text(
              title,
              style: const TextStyle(fontWeight: FontWeight.w700),
            ),
          ),
          Text(time, style: Theme.of(context).textTheme.bodySmall),
        ],
      ),
    );
  }
}
