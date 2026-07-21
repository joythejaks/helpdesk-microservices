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
    final colors = Theme.of(context).colorScheme;
    return Padding(
      padding: const EdgeInsets.fromLTRB(18, 0, 18, 18),
      child: ClipRRect(
        borderRadius: BorderRadius.circular(22),
        child: BackdropFilter(
          filter: ImageFilter.blur(sigmaX: 20, sigmaY: 20),
          child: Container(
            color: colors.surfaceContainerLowest.withValues(alpha: .86),
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 8),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.spaceAround,
              children: [
                for (var i = 0; i < items.length; i++)
                  AnimatedContainer(
                    duration: const Duration(milliseconds: 150),
                    padding: const EdgeInsets.symmetric(horizontal: 4),
                    decoration: BoxDecoration(
                      // The active pill intentionally uses the brand's
                      // Fixed pair (not the adaptive colorScheme) so it
                      // reads identically as a light accent in both themes.
                      color: i == index
                          ? HelpdeskTheme.primaryFixed
                          : Colors.transparent,
                      borderRadius: BorderRadius.circular(18),
                    ),
                    child: IconButton(
                      tooltip: items[i].$2,
                      isSelected: i == index,
                      selectedIcon: Icon(
                        items[i].$1,
                        color: HelpdeskTheme.onPrimaryFixed,
                      ),
                      onPressed: () => onChanged(i),
                      icon: Icon(items[i].$1, color: colors.onSurfaceVariant),
                    ),
                  ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
