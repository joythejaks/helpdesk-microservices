import 'package:flutter/material.dart';

import 'package:helpdesk_app/core/theme/helpdesk_theme.dart';

class SplashScreen extends StatelessWidget {
  const SplashScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Container(
        decoration: const BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
            colors: [HelpdeskTheme.primary, Color(0xFF00243A)],
          ),
        ),
        child: SafeArea(
          child: Center(
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                // Glowing circular badge — matches the design's soft
                // halo behind the app icon on the splash screen.
                Container(
                  width: 120,
                  height: 120,
                  decoration: BoxDecoration(
                    color: Colors.white,
                    shape: BoxShape.circle,
                    boxShadow: [
                      BoxShadow(
                        color: Colors.white.withAlpha(60),
                        blurRadius: 48,
                        spreadRadius: 12,
                      ),
                    ],
                  ),
                  child: const Icon(
                    Icons.confirmation_number_outlined,
                    size: 56,
                    color: HelpdeskTheme.primary,
                  ),
                ),
                const SizedBox(height: 32),
                Text(
                  'Helpdesk\nTicketing',
                  textAlign: TextAlign.center,
                  style: Theme.of(context).textTheme.displayMedium?.copyWith(
                    fontFamily: 'Manrope',
                    color: Colors.white,
                    fontWeight: FontWeight.w800,
                    height: 1.0,
                  ),
                ),
                const SizedBox(height: 16),
                Text(
                  'PRECISION SUPPORT SOLUTIONS',
                  style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                    fontFamily: 'Inter',
                    color: Colors.white.withAlpha(178),
                    fontWeight: FontWeight.w700,
                    letterSpacing: 2,
                  ),
                ),
                const SizedBox(height: 48),
                const _SplashDots(),
                const SizedBox(height: 32),
                const CircularProgressIndicator(
                  valueColor: AlwaysStoppedAnimation<Color>(Colors.white),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

/// Purely decorative — there's no real onboarding carousel behind the
/// splash screen, this just mirrors the uploaded design's page-indicator
/// dots.
class _SplashDots extends StatelessWidget {
  const _SplashDots();

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: List.generate(3, (i) {
        final active = i == 1;
        return Container(
          margin: const EdgeInsets.symmetric(horizontal: 4),
          width: active ? 20 : 8,
          height: 8,
          decoration: BoxDecoration(
            color: Colors.white.withAlpha(active ? 255 : 102),
            borderRadius: BorderRadius.circular(4),
          ),
        );
      }),
    );
  }
}
