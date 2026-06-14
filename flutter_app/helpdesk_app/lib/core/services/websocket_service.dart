import 'dart:convert';
import 'package:web_socket_channel/web_socket_channel.dart';
import 'package:helpdesk_app/pkg/logger.dart'; // Asumsi logger tersedia

class WebSocketService {
  WebSocketChannel? _channel;
  final String url;
  int _retryCount = 0;
  bool _isManualDisconnect = false;

  WebSocketService({required this.url});

  void connect(String token, Function(Map<String, dynamic>) onMessageReceived) {
    try {
      _isManualDisconnect = false;
      // Menambahkan token ke URL sebagai query param untuk autentikasi di WS
      final wsUri = Uri.parse('$url?token=$token');
      _channel = WebSocketChannel.connect(wsUri);

      _channel!.stream.listen(
        (message) {
          _retryCount = 0; // Reset retry saat sukses
          final data = jsonDecode(message);
          onMessageReceived(data);
        },
        onError: (error) {
          print('WS Error: $error');
          _retryConnection(token, onMessageReceived);
        },
        onDone: () {
          if (!_isManualDisconnect) {
            print('WS Connection Lost. Retrying...');
            _retryConnection(token, onMessageReceived);
          }
        },
      );
    } catch (e) {
      print('Could not connect to WS: $e');
    }
  }

  void _retryConnection(String token, Function(Map<String, dynamic>) onMessageReceived) {
    if (_retryCount > 5) return; // Maksimal retry
    _retryCount++;
    
    // Exponential backoff: 2s, 4s, 8s, 16s...
    final delay = Duration(seconds: _retryCount * 2);
    Future.delayed(delay, () {
      connect(token, onMessageReceived);
    });
  }

  void sendMessage(Map<String, dynamic> message) {
    if (_channel != null) {
      _channel!.sink.add(jsonEncode(message));
    }
  }

  void disconnect() {
    _isManualDisconnect = true;
    _channel?.sink.close();
  }
}