import 'package:flutter/material.dart';

import 'package:helpdesk_app/core/theme/helpdesk_theme.dart';

class SurfaceCard extends StatelessWidget {
  const SurfaceCard({super.key, required this.child, this.onTap});

  final Widget child;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: HelpdeskTheme.surfaceLowest,
      borderRadius: BorderRadius.circular(12),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(12),
        child: Padding(padding: const EdgeInsets.all(18), child: child),
      ),
    );
  }
}
