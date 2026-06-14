import 'dart:convert';
import 'package:web_socket_channel/web_socket_channel.dart';
import 'package:helpdesk_app/pkg/logger.dart'; // Asumsi logger tersedia

class WebSocketService {
  WebSocketChannel? _channel;
  final String url;

  WebSocketService({required this.url});

  void connect(String token, Function(Map<String, dynamic>) onMessageReceived) {
    try {
      // Menambahkan token ke URL sebagai query param untuk autentikasi di WS
      final wsUri = Uri.parse('$url?token=$token');
      _channel = WebSocketChannel.connect(wsUri);

      _channel!.stream.listen(
        (message) {
          final data = jsonDecode(message);
          onMessageReceived(data);
        },
        onError: (error) {
          print('WS Error: $error');
          _retryConnection(token, onMessageReceived);
        },
        onDone: () {
          print('WS Connection Closed');
        },
      );
    } catch (e) {
      print('Could not connect to WS: $e');
    }
  }

  void _retryConnection(String token, Function(Map<String, dynamic>) onMessageReceived) {
    Future.delayed(const Duration(seconds: 5), () {
      connect(token, onMessageReceived);
    });
  }

  void sendMessage(Map<String, dynamic> message) {
    if (_channel != null) {
      _channel!.sink.add(jsonEncode(message));
    }
  }

  void disconnect() {
    _channel?.sink.close();
  }
}