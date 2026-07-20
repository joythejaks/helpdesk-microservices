import 'dart:async';

import '../core/network/api_client.dart';
import '../core/storage/token_storage.dart';
import '../models/app_user.dart';

class AuthRepository {
  AuthRepository({
    required ApiClient apiClient,
    required TokenStorage tokenStorage,
  }) : _apiClient = apiClient,
       _tokenStorage = tokenStorage {
    _apiClient.onUnauthorized = refreshTokens;
  }

  final ApiClient _apiClient;
  final TokenStorage _tokenStorage;
  final _sessionExpiredController = StreamController<void>.broadcast();

  /// Fires when a 401 could not be recovered by a token refresh (no
  /// refresh token stored, or the refresh call itself failed) — local
  /// session storage has already been cleared by the time this fires.
  Stream<void> get onSessionExpired => _sessionExpiredController.stream;

  /// Called by [ApiClient] when a request comes back 401. Rotates the
  /// stored token pair via `POST /auth/refresh` and returns the new access
  /// token, or null (and clears the session) if refresh isn't possible.
  Future<String?> refreshTokens() async {
    final refreshToken = await _tokenStorage.readRefreshToken();
    if (refreshToken == null || refreshToken.isEmpty) {
      await _handleSessionExpired();
      return null;
    }
    try {
      final response = await _apiClient.post(
        '/auth/refresh',
        body: {'refresh_token': refreshToken},
      );
      final data = response['data'] as Map<String, dynamic>;
      final newAccessToken = data['access_token'] as String;
      await _tokenStorage.saveTokens(
        accessToken: newAccessToken,
        refreshToken: data['refresh_token'] as String,
      );
      return newAccessToken;
    } catch (_) {
      await _handleSessionExpired();
      return null;
    }
  }

  Future<void> _handleSessionExpired() async {
    await _tokenStorage.clear();
    _sessionExpiredController.add(null);
  }

  Future<bool> hasSession() async {
    final token = await _tokenStorage.readAccessToken();
    return token != null && token.isNotEmpty;
  }

  /// Fetches the currently authenticated caller's own account — lets the
  /// app know who's logged in and what role they have without decoding
  /// the JWT itself.
  Future<AppUser> getCurrentUser() async {
    final token = await _tokenStorage.readAccessToken();
    if (token == null || token.isEmpty) {
      throw const ApiException('Sesi login tidak ditemukan', 'UNAUTHORIZED');
    }
    final response = await _apiClient.get('/auth/me', token: token);
    return AppUser.fromJson(response['data'] as Map<String, dynamic>);
  }

  Future<void> login({required String email, required String password}) async {
    final response = await _apiClient.post(
      '/auth/login',
      body: {'email': email, 'password': password},
    );

    final data = response['data'] as Map<String, dynamic>;
    await _tokenStorage.saveTokens(
      accessToken: data['access_token'] as String,
      refreshToken: data['refresh_token'] as String,
    );
  }

  Future<void> register({
    required String name,
    required String email,
    required String password,
    required String department,
  }) async {
    await _apiClient.post(
      '/auth/register',
      body: {
        'name': name,
        'email': email,
        'password': password,
        'department': department,
      },
    );
  }

  Future<void> logout() async {
    final token = await _tokenStorage.readAccessToken();
    if (token != null) {
      try {
        await _apiClient.post('/auth/logout', token: token);
      } catch (_) {
        // Best-effort — always clear the local session regardless of
        // whether the server call succeeded (e.g. token already expired).
      }
    }
    await _tokenStorage.clear();
  }

  Future<void> changePassword({
    required String oldPassword,
    required String newPassword,
  }) async {
    final token = await _requireToken();
    await _apiClient.post(
      '/auth/change-password',
      token: token,
      body: {'old_password': oldPassword, 'new_password': newPassword},
    );
  }

  Future<AppUser> updateProfile({
    required String name,
    required String department,
  }) async {
    final token = await _requireToken();
    final response = await _apiClient.patch(
      '/auth/me',
      token: token,
      body: {'name': name, 'department': department},
    );
    return AppUser.fromJson(response['data'] as Map<String, dynamic>);
  }

  Future<AppUser> updateAvailability(String availability) async {
    final token = await _requireToken();
    final response = await _apiClient.patch(
      '/auth/me/availability',
      token: token,
      body: {'availability': availability},
    );
    return AppUser.fromJson(response['data'] as Map<String, dynamic>);
  }

  Future<String> _requireToken() async {
    final token = await _tokenStorage.readAccessToken();
    if (token == null || token.isEmpty) {
      throw const ApiException('Sesi login tidak ditemukan', 'UNAUTHORIZED');
    }
    return token;
  }
}
