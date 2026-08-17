import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// Access/refresh tokens via Keychain on iOS/macOS and Keystore-backed
/// EncryptedSharedPreferences on Android, instead of plaintext prefs —
/// closes a real gap (tokens were readable as plaintext XML/plist on a
/// rooted/jailbroken device or via any other file-read bug).
class TokenStorage {
  static const _accessTokenKey = 'access_token';
  static const _refreshTokenKey = 'refresh_token';

  static const _storage = FlutterSecureStorage(
    aOptions: AndroidOptions(encryptedSharedPreferences: true),
  );

  Future<String?> readAccessToken() => _storage.read(key: _accessTokenKey);

  Future<String?> readRefreshToken() => _storage.read(key: _refreshTokenKey);

  Future<void> saveTokens({
    required String accessToken,
    required String refreshToken,
  }) async {
    await _storage.write(key: _accessTokenKey, value: accessToken);
    await _storage.write(key: _refreshTokenKey, value: refreshToken);
  }

  Future<void> clear() async {
    await _storage.delete(key: _accessTokenKey);
    await _storage.delete(key: _refreshTokenKey);
  }
}
