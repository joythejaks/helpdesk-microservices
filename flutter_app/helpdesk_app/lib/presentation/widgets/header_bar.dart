import 'package:flutter/material.dart';

class HeaderBar extends StatelessWidget {
  const HeaderBar({
    super.key,
    required this.title,
    required this.subtitle,
    this.leading,
    this.trailing,
    this.onLeadingTap,
    this.onTrailingTap,
    this.trailingBadgeCount,
  });

  final String title;
  final String subtitle;
  final IconData? leading;
  final IconData? trailing;
  final VoidCallback? onLeadingTap;
  final VoidCallback? onTrailingTap;
  final int? trailingBadgeCount;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        if (leading != null)
          IconButton.filledTonal(onPressed: onLeadingTap, icon: Icon(leading)),
        if (leading != null) const SizedBox(width: 8),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(title, style: Theme.of(context).textTheme.titleLarge),
              const SizedBox(height: 4),
              Text(subtitle, style: Theme.of(context).textTheme.bodySmall),
            ],
          ),
        ),
        if (trailing != null)
          Stack(
            clipBehavior: Clip.none,
            children: [
              IconButton.filledTonal(
                onPressed: onTrailingTap,
                icon: Icon(trailing),
              ),
              if ((trailingBadgeCount ?? 0) > 0)
                Positioned(
                  right: -2,
                  top: -2,
                  child: Container(
                    padding: const EdgeInsets.all(3),
                    constraints: const BoxConstraints(
                      minWidth: 16,
                      minHeight: 16,
                    ),
                    decoration: const BoxDecoration(
                      color: Color(0xFFBA1A1A),
                      shape: BoxShape.circle,
                    ),
                    child: Text(
                      '$trailingBadgeCount',
                      textAlign: TextAlign.center,
                      style: const TextStyle(
                        color: Colors.white,
                        fontSize: 9,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                  ),
                ),
            ],
          ),
      ],
    );
  }
}
