import 'package:helpdesk_app/core/network/api_client.dart';
import 'package:helpdesk_app/core/storage/token_storage.dart';
import 'package:helpdesk_app/models/ticket.dart';
import 'package:helpdesk_app/models/ticket_comment.dart';

class TicketRepository {
  TicketRepository({
    required ApiClient apiClient,
    required TokenStorage tokenStorage,
  }) : _apiClient = apiClient,
       _tokenStorage = tokenStorage;

  final ApiClient _apiClient;
  final TokenStorage _tokenStorage;

  Future<List<Ticket>> getTickets({
    int page = 1,
    int limit = 20,
    String? scope,
    String? status,
    String? priority,
    String? department,
    String? search,
    bool? overdue,
    DateTime? from,
    DateTime? to,
  }) async {
    final token = await _requireToken();
    final query = <String, String>{'page': '$page', 'limit': '$limit'};
    if (scope != null && scope.isNotEmpty) query['scope'] = scope;
    if (status != null && status.isNotEmpty) query['status'] = status;
    if (priority != null && priority.isNotEmpty) query['priority'] = priority;
    if (department != null && department.isNotEmpty) {
      query['department'] = department;
    }
    if (search != null && search.isNotEmpty) query['search'] = search;
    if (overdue == true) query['overdue'] = 'true';
    if (from != null) query['from'] = _formatDate(from);
    if (to != null) query['to'] = _formatDate(to);

    final response = await _apiClient.get(
      '/tickets',
      token: token,
      query: query,
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

  Future<void> assignTicket({required String ticketId, int? agentId}) async {
    final token = await _requireToken();
    await _apiClient.patch(
      '/tickets/$ticketId/assign',
      token: token,
      body: {if (agentId != null && agentId > 0) 'agent_id': agentId},
    );
  }

  Future<void> updateTicketStatus({
    required String ticketId,
    required String status,
  }) async {
    final token = await _requireToken();
    await _apiClient.patch(
      '/tickets/$ticketId/status',
      token: token,
      body: {'status': status},
    );
  }

  Future<List<TicketComment>> getComments(String ticketId) async {
    final token = await _requireToken();
    final response = await _apiClient.get(
      '/tickets/$ticketId/comments',
      token: token,
    );
    final data = response['data'];
    if (data is! List) return [];
    return data
        .whereType<Map<String, dynamic>>()
        .map(TicketComment.fromJson)
        .toList(growable: false);
  }

  Future<TicketComment> addComment({
    required String ticketId,
    required String body,
  }) async {
    final token = await _requireToken();
    final response = await _apiClient.post(
      '/tickets/$ticketId/comments',
      token: token,
      body: {'body': body},
    );
    return TicketComment.fromJson(response['data'] as Map<String, dynamic>);
  }

  Future<String> _requireToken() async {
    final token = await _tokenStorage.readAccessToken();
    if (token == null || token.isEmpty) {
      throw const ApiException('Sesi login tidak ditemukan', 'UNAUTHORIZED');
    }
    return token;
  }

  static String _formatDate(DateTime d) {
    final y = d.year.toString().padLeft(4, '0');
    final m = d.month.toString().padLeft(2, '0');
    final day = d.day.toString().padLeft(2, '0');
    return '$y-$m-$day';
  }
}
