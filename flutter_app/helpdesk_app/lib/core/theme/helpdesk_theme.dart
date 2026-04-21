import 'package:flutter/material.dart';

class HelpdeskTheme {
  static const surface = Color(0xFFF8F9FF);
  static const surfaceLow = Color(0xFFF2F3F9);
  static const surfaceLowest = Color(0xFFFFFFFF);
  static const surfaceHigh = Color(0xFFE6E8ED);
  static const primary = Color(0xFF004976);
  static const primaryContainer = Color(0xFF00629B);
  static const primaryFixed = Color(0xFFCEE5FF);
  static const tertiaryFixed = Color(0xFFA0F0F0);
  static const errorContainer = Color(0xFFFFDAD6);
  static const onSurface = Color(0xFF191C20);
  static const onVariant = Color(0xFF414750);

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
