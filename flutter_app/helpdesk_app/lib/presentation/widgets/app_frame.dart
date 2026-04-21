import 'package:flutter/material.dart';

import '../../core/theme/helpdesk_theme.dart';

class AppFrame extends StatelessWidget {
  const AppFrame({super.key, required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    return ColoredBox(
      color: HelpdeskTheme.surface,
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 430),
          child: Material(color: Colors.transparent, child: child),
        ),
      ),
    );
  }
}
