import 'package:flutter/material.dart';

class HeaderBar extends StatelessWidget {
  const HeaderBar({
    super.key,
    required this.title,
    required this.subtitle,
    this.leading,
    this.trailing,
    this.onLeadingTap,
  });

  final String title;
  final String subtitle;
  final IconData? leading;
  final IconData? trailing;
  final VoidCallback? onLeadingTap;

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
          IconButton.filledTonal(onPressed: () {}, icon: Icon(trailing)),
      ],
    );
  }
}
