import 'package:flutter/material.dart';

import '../../core/theme/helpdesk_theme.dart';

class AppMark extends StatelessWidget {
  const AppMark({super.key});

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Container(
          width: 42,
          height: 42,
          decoration: BoxDecoration(
            gradient: const LinearGradient(
              colors: [HelpdeskTheme.primary, HelpdeskTheme.primaryContainer],
              begin: Alignment.topLeft,
              end: Alignment.bottomRight,
            ),
            borderRadius: BorderRadius.circular(12),
          ),
          child: const Icon(Icons.support_agent, color: Colors.white),
        ),
        const SizedBox(width: 12),
        const Text(
          'Helpdesk',
          style: TextStyle(fontWeight: FontWeight.w800, fontSize: 18),
        ),
      ],
    );
  }
}
