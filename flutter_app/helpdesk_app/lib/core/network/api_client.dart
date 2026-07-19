import 'dart:convert';

import 'package:http/http.dart' as http;

import '../constants/api.dart';

class ApiException implements Exception {
  const ApiException(this.message, [this.code]);

  final String message;
  final String? code;

  @override
  String toString() => message;
}

class ApiClient {
  ApiClient({http.Client? httpClient})
    : _httpClient = httpClient ?? http.Client();

  final http.Client _httpClient;

  /// Invoked once when a request comes back 401. Should attempt a token
  /// refresh and return the new access token, or null if the session
  /// could not be recovered (the original 401 is then surfaced as-is).
  Future<String?> Function()? onUnauthorized;

  Future<Map<String, dynamic>> get(
    String path, {
    String? token,
    Map<String, String>? query,
  }) {
    final uri = Uri.parse(
      '${Api.baseUrl}$path',
    ).replace(queryParameters: query);
    return _sendWithRefresh(
      token: token,
      request: (t) => _httpClient.get(uri, headers: _headers(t)),
    );
  }

  Future<Map<String, dynamic>> post(
    String path, {
    Map<String, dynamic>? body,
    String? token,
  }) {
    final uri = Uri.parse('${Api.baseUrl}$path');
    return _sendWithRefresh(
      token: token,
      request: (t) => _httpClient.post(
        uri,
        headers: _headers(t),
        body: jsonEncode(body ?? {}),
      ),
    );
  }

  Future<Map<String, dynamic>> patch(
    String path, {
    Map<String, dynamic>? body,
    String? token,
  }) {
    final uri = Uri.parse('${Api.baseUrl}$path');
    return _sendWithRefresh(
      token: token,
      request: (t) => _httpClient.patch(
        uri,
        headers: _headers(t),
        body: jsonEncode(body ?? {}),
      ),
    );
  }

  Future<Map<String, dynamic>> _sendWithRefresh({
    required String? token,
    required Future<http.Response> Function(String? token) request,
  }) async {
    final response = await request(token);
    if (response.statusCode == 401 && token != null && onUnauthorized != null) {
      final refreshed = await onUnauthorized!();
      if (refreshed != null) {
        return _decode(await request(refreshed));
      }
    }
    return _decode(response);
  }

  Map<String, String> _headers(String? token) {
    return {
      'Content-Type': 'application/json',
      if (token != null && token.isNotEmpty) 'Authorization': 'Bearer $token',
    };
  }

  Map<String, dynamic> _decode(http.Response response) {
    final decoded = jsonDecode(response.body) as Map<String, dynamic>;
    final success = decoded['success'] == true;

    if (!success || response.statusCode >= 400) {
      throw ApiException(
        (decoded['message'] as String?) ?? 'Request gagal',
        decoded['error'] as String?,
      );
    }

    return decoded;
  }
}
