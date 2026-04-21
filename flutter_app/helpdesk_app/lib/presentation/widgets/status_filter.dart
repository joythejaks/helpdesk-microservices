import 'package:flutter/material.dart';

import 'status_chip.dart';

class StatusFilter extends StatelessWidget {
  const StatusFilter({super.key});

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: const [
        StatusChip(text: 'All'),
        StatusChip(text: 'Open'),
        StatusChip(text: 'In Progress'),
        StatusChip(text: 'Resolved'),
      ],
    );
  }
}
