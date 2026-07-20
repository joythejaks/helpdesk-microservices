class Api {
  static const baseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://10.0.2.2:8080',
  );

  /// notification-service's own base URL — never proxied through the
  /// gateway (see main.dart's comment on the /ws connection for why),
  /// so it needs a separate host:port from [baseUrl].
  static const notificationBaseUrl = String.fromEnvironment(
    'NOTIFICATION_API_BASE_URL',
    defaultValue: 'http://10.0.2.2:8083',
  );
}
