import 'package:flutter/material.dart';

import 'package:helpdesk_app/core/theme/helpdesk_theme.dart';

class SplashScreen extends StatefulWidget {
  const SplashScreen({super.key});

  @override
  State<SplashScreen> createState() => _SplashScreenState();
}

class _SplashScreenState extends State<SplashScreen> {
  @override
  void initState() {
    super.initState();
    _navigateToLogin();
  }

  void _navigateToLogin() {
    Future.delayed(const Duration(seconds: 3), () {
      if (mounted) {
        Navigator.of(context).pushReplacementNamed('/login');
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: HelpdeskTheme.surface,
      body: Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            // Logo placeholder - bisa diganti dengan gambar logo
            Container(
              width: 120,
              height: 120,
              decoration: BoxDecoration(
                color: HelpdeskTheme.primaryContainer,
                borderRadius: BorderRadius.circular(24),
                boxShadow: [
                  BoxShadow(
                    color: HelpdeskTheme.primary.withValues(alpha: 51),
                    blurRadius: 24,
                    offset: const Offset(0, 8),
                  ),
                ],
              ),
              child: Icon(
                Icons.support_agent,
                size: 60,
                color: HelpdeskTheme.onPrimaryContainer,
              ),
            ),
            const SizedBox(height: 32),
            Text(
              'Helpdesk\nTicketing',
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.displayMedium?.copyWith(
                    fontFamily: 'Manrope',
                    color: HelpdeskTheme.onSurface,
                    fontWeight: FontWeight.w800,
                    height: 1.0,
                  ),
            ),
            const SizedBox(height: 16),
            Text(
              'Precision Support Solutions',
              style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                    fontFamily: 'Inter',
                    color: HelpdeskTheme.onSurfaceVariant,
                    fontWeight: FontWeight.w500,
                  ),
            ),
            const SizedBox(height: 48),
            CircularProgressIndicator(
              valueColor: AlwaysStoppedAnimation<Color>(HelpdeskTheme.primary),
            ),
          ],
        ),
      ),
    );
  }
}