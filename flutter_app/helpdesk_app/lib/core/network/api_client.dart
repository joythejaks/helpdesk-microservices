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

  Future<Map<String, dynamic>> get(
    String path, {
    String? token,
    Map<String, String>? query,
  }) async {
    final uri = Uri.parse(
      '${Api.baseUrl}$path',
    ).replace(queryParameters: query);
    final response = await _httpClient.get(uri, headers: _headers(token));
    return _decode(response);
  }

  Future<Map<String, dynamic>> post(
    String path, {
    Map<String, dynamic>? body,
    String? token,
  }) async {
    final response = await _httpClient.post(
      Uri.parse('${Api.baseUrl}$path'),
      headers: _headers(token),
      body: jsonEncode(body ?? {}),
    );
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
