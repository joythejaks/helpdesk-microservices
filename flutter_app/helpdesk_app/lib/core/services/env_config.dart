class EnvConfig {
  static const String apiBaseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://10.0.2.2:8080', // Default untuk Android Emulator
  );

  static const String wsUrl = String.fromEnvironment(
    'WS_URL',
    defaultValue: 'ws://10.0.2.2:8083/ws',
  );

  static const bool isProduction = bool.fromEnvironment(
    'IS_PRODUCTION',
    defaultValue: false,
  );

  // Empty by default — sentry_flutter no-ops (logs a warning, sends
  // nothing) when initialized with no DSN, so builds without one
  // (including CI) stay safe. Set via --dart-define=SENTRY_DSN=... once
  // a real Sentry project exists.
  static const String sentryDsn = String.fromEnvironment('SENTRY_DSN');
}
