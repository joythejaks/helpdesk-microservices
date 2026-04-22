import 'package:flutter/material.dart';

class HelpdeskTheme {
  // Colors from Stitch Design System
  static const background = Color(0xFFF8F9FF);
  static const error = Color(0xFFBA1A1A);
  static const errorContainer = Color(0xFFFFDAD6);
  static const inverseOnSurface = Color(0xFFEFF1F6);
  static const inversePrimary = Color(0xFF98CBFF);
  static const inverseSurface = Color(0xFF2E3135);
  static const onBackground = Color(0xFF191C20);
  static const onError = Color(0xFFFFFFFF);
  static const onErrorContainer = Color(0xFF93000A);
  static const onPrimary = Color(0xFFFFFFFF);
  static const onPrimaryContainer = Color(0xFFBADBFF);
  static const onPrimaryFixed = Color(0xFF001D33);
  static const onPrimaryFixedVariant = Color(0xFF004A77);
  static const onSecondary = Color(0xFFFFFFFF);
  static const onSecondaryContainer = Color(0xFF58676D);
  static const onSecondaryFixed = Color(0xFF101D23);
  static const onSecondaryFixedVariant = Color(0xFF3B494F);
  static const onSurface = Color(0xFF191C20);
  static const onSurfaceVariant = Color(0xFF414750);
  static const onTertiary = Color(0xFFFFFFFF);
  static const onTertiaryContainer = Color(0xFF95E5E5);
  static const onTertiaryFixed = Color(0xFF002020);
  static const onTertiaryFixedVariant = Color(0xFF004F4F);
  static const outline = Color(0xFF717881);
  static const outlineVariant = Color(0xFFC0C7D1);
  static const primary = Color(0xFF004976);
  static const primaryContainer = Color(0xFF00629B);
  static const primaryFixed = Color(0xFFCEE5FF);
  static const primaryFixedDim = Color(0xFF98CBFF);
  static const secondary = Color(0xFF536167);
  static const secondaryContainer = Color(0xFFD6E5EC);
  static const secondaryFixed = Color(0xFFD6E5EC);
  static const secondaryFixedDim = Color(0xFFBAC9D0);
  static const surface = Color(0xFFF8F9FF);
  static const surfaceBright = Color(0xFFF8F9FF);
  static const surfaceContainer = Color(0xFFECEEF3);
  static const surfaceContainerHigh = Color(0xFFE6E8ED);
  static const surfaceContainerHighest = Color(0xFFE0E2E8);
  static const surfaceContainerLow = Color(0xFFF2F3F9);
  static const surfaceContainerLowest = Color(0xFFFFFFFF);
  static const surfaceDim = Color(0xFFD8DAE0);
  static const surfaceTint = Color(0xFF02639C);
  static const surfaceVariant = Color(0xFFDDE3EA);
  static const tertiary = Color(0xFF00696B);
  static const tertiaryContainer = Color(0xFF006969);
  static const tertiaryFixed = Color(0xFFA0F0F0);
  static const tertiaryFixedDim = Color(0xFF81D3D3);

  // Legacy aliases for compatibility
  static const surfaceLow = surfaceContainerLow;
  static const surfaceLowest = surfaceContainerLowest;
  static const surfaceHigh = surfaceContainerHigh;
  static const onVariant = onSurfaceVariant;

  static ThemeData light() {
    final scheme = ColorScheme.fromSeed(
      seedColor: primaryContainer,
      brightness: Brightness.light,
      surface: surface,
    );

    return ThemeData(
      useMaterial3: true,
      colorScheme: scheme.copyWith(
        primary: primary,
        primaryContainer: primaryContainer,
        surface: surface,
        onSurface: onSurface,
      ),
      scaffoldBackgroundColor: surface,
      fontFamily: 'Inter',
      textTheme: const TextTheme(
        displayMedium: TextStyle(
          fontFamily: 'Manrope',
          fontSize: 42,
          height: 1,
          fontWeight: FontWeight.w800,
          color: onSurface,
        ),
        headlineMedium: TextStyle(
          fontFamily: 'Manrope',
          fontSize: 28,
          height: 1.12,
          fontWeight: FontWeight.w800,
          color: onSurface,
        ),
        titleLarge: TextStyle(
          fontFamily: 'Manrope',
          fontSize: 21,
          fontWeight: FontWeight.w800,
          color: onSurface,
        ),
        titleMedium: TextStyle(
          fontSize: 16,
          fontWeight: FontWeight.w700,
          color: onSurface,
        ),
        bodyMedium: TextStyle(fontSize: 14, height: 1.45, color: onSurface),
        bodySmall: TextStyle(fontSize: 12, height: 1.35, color: onVariant),
        labelLarge: TextStyle(fontSize: 14, fontWeight: FontWeight.w700),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: surfaceHigh,
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: BorderSide.none,
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(8),
          borderSide: const BorderSide(color: Color(0x33004976)),
        ),
        contentPadding: const EdgeInsets.symmetric(
          horizontal: 16,
          vertical: 15,
        ),
      ),
    );
  }
}
