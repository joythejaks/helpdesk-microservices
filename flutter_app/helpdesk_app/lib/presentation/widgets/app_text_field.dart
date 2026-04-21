import 'package:flutter/material.dart';

import '../../core/theme/helpdesk_theme.dart';

class AppTextField extends StatelessWidget {
  const AppTextField({
    super.key,
    required this.label,
    required this.icon,
    this.controller,
    this.keyboardType,
    this.obscureText = false,
    this.maxLines = 1,
  });

  final String label;
  final IconData icon;
  final TextEditingController? controller;
  final TextInputType? keyboardType;
  final bool obscureText;
  final int maxLines;

  @override
  Widget build(BuildContext context) {
    return TextField(
      controller: controller,
      keyboardType: keyboardType,
      obscureText: obscureText,
      maxLines: obscureText ? 1 : maxLines,
      decoration: InputDecoration(
        labelText: label,
        prefixIcon: Icon(icon, color: HelpdeskTheme.primary),
      ),
    );
  }
}
