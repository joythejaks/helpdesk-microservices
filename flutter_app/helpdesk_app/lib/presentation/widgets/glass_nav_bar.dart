import 'dart:ui';

import 'package:flutter/material.dart';

import '../../core/theme/helpdesk_theme.dart';

class GlassNavBar extends StatelessWidget {
  const GlassNavBar({
    super.key,
    required this.index,
    required this.onChanged,
    required this.items,
  });

  final int index;
  final ValueChanged<int> onChanged;
  final List<(IconData, String)> items;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(18, 0, 18, 18),
      child: ClipRRect(
        borderRadius: BorderRadius.circular(22),
        child: BackdropFilter(
          filter: ImageFilter.blur(sigmaX: 20, sigmaY: 20),
          child: Container(
            color: HelpdeskTheme.surfaceLowest.withValues(alpha: .86),
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 8),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.spaceAround,
              children: [
                for (var i = 0; i < items.length; i++)
                  IconButton(
                    tooltip: items[i].$2,
                    isSelected: i == index,
                    selectedIcon: Icon(
                      items[i].$1,
                      color: HelpdeskTheme.primary,
                    ),
                    onPressed: () => onChanged(i),
                    icon: Icon(items[i].$1, color: HelpdeskTheme.onVariant),
                  ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
