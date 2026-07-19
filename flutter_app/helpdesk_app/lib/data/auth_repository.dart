import '../core/network/api_client.dart';
import '../core/storage/token_storage.dart';
import '../models/app_user.dart';

class AuthRepository {
  AuthRepository({
    required ApiClient apiClient,
    required TokenStorage tokenStorage,
  }) : _apiClient = apiClient,
       _tokenStorage = tokenStorage;

  final ApiClient _apiClient;
  final TokenStorage _tokenStorage;

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
    required String email,
    required String password,
    String role = 'user',
  }) async {
    await _apiClient.post(
      '/auth/register',
      body: {'email': email, 'password': password, 'role': role},
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
}
