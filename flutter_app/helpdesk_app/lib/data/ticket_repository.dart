import 'package:helpdesk_app/core/network/api_client.dart';
import 'package:helpdesk_app/core/storage/token_storage.dart';
import 'package:helpdesk_app/models/ticket.dart';

class TicketRepository {
  TicketRepository({
    required ApiClient apiClient,
    required TokenStorage tokenStorage,
  }) : _apiClient = apiClient,
       _tokenStorage = tokenStorage;

  final ApiClient _apiClient;
  final TokenStorage _tokenStorage;

  Future<List<Ticket>> getTickets({int page = 1, int limit = 20}) async {
    final token = await _requireToken();
    final response = await _apiClient.get(
      '/tickets',
      token: token,
      query: {'page': '$page', 'limit': '$limit'},
    );

    final data = response['data'];
    if (data is! List) return [];

    return data
        .whereType<Map<String, dynamic>>()
        .map(Ticket.fromJson)
        .toList(growable: false);
  }

  Future<void> createTicket({
    required String title,
    required String description,
  }) async {
    final token = await _requireToken();
    await _apiClient.post(
      '/tickets',
      token: token,
      body: {'title': title, 'description': description},
    );
  }

  Future<String> _requireToken() async {
    final token = await _tokenStorage.readAccessToken();
    if (token == null || token.isEmpty) {
      throw const ApiException('Sesi login tidak ditemukan', 'UNAUTHORIZED');
    }
    return token;
  }
}
