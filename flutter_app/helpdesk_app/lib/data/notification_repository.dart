import 'package:helpdesk_app/core/network/api_client.dart';
import 'package:helpdesk_app/core/network/notification_api_client.dart';
import 'package:helpdesk_app/core/storage/token_storage.dart';
import 'package:helpdesk_app/models/notification.dart';

class NotificationRepository {
  NotificationRepository({
    required NotificationApiClient apiClient,
    required TokenStorage tokenStorage,
    required Future<String?> Function() onUnauthorized,
  }) : _apiClient = apiClient,
       _tokenStorage = tokenStorage {
    _apiClient.onUnauthorized = onUnauthorized;
  }

  final NotificationApiClient _apiClient;
  final TokenStorage _tokenStorage;

  Future<List<AppNotification>> getNotifications({
    bool unreadOnly = false,
    int page = 1,
    int limit = 20,
  }) async {
    final token = await _requireToken();
    final query = <String, String>{'page': '$page', 'limit': '$limit'};
    if (unreadOnly) query['unread_only'] = 'true';

    final response = await _apiClient.get(
      '/notifications',
      token: token,
      query: query,
    );
    final data = response['data'];
    if (data is! List) return [];
    return data
        .whereType<Map<String, dynamic>>()
        .map(AppNotification.fromJson)
        .toList(growable: false);
  }

  Future<void> markRead(int id) async {
    final token = await _requireToken();
    await _apiClient.patch('/notifications/$id/read', token: token);
  }

  Future<void> markAllRead() async {
    final token = await _requireToken();
    await _apiClient.patch('/notifications/read-all', token: token);
  }

  Future<String> _requireToken() async {
    final token = await _tokenStorage.readAccessToken();
    if (token == null || token.isEmpty) {
      throw const ApiException('Sesi login tidak ditemukan', 'UNAUTHORIZED');
    }
    return token;
  }
}
