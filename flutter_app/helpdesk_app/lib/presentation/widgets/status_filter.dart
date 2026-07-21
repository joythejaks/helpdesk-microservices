import 'package:flutter/material.dart';

class StatusFilter extends StatelessWidget {
  const StatusFilter({
    super.key,
    required this.selected,
    required this.onSelected,
  });

  final String selected;
  final ValueChanged<String> onSelected;

  static const _filters = [
    'All',
    'Open',
    'Assigned',
    'In Progress',
    'Pending',
    'Resolved',
    'Closed',
  ];

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: _filters.map((filter) {
        final isSelected = filter == selected;
        return GestureDetector(
          onTap: () => onSelected(filter),
          child: AnimatedContainer(
            duration: const Duration(milliseconds: 150),
            padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 7),
            decoration: BoxDecoration(
              color: isSelected ? colors.primary : colors.surfaceContainerHigh,
              borderRadius: BorderRadius.circular(999),
            ),
            child: Text(
              filter,
              style: TextStyle(
                fontSize: 12,
                fontWeight: FontWeight.w700,
                color: isSelected ? colors.onPrimary : colors.onSurfaceVariant,
              ),
            ),
          ),
        );
      }).toList(),
    );
  }
}
