import 'package:helpdesk_app/core/network/api_client.dart';
import 'package:helpdesk_app/core/storage/token_storage.dart';
import 'package:helpdesk_app/models/app_user.dart';
import 'package:helpdesk_app/models/report_summary.dart';

class AdminRepository {
  AdminRepository({
    required ApiClient apiClient,
    required TokenStorage tokenStorage,
  }) : _apiClient = apiClient,
       _tokenStorage = tokenStorage;

  final ApiClient _apiClient;
  final TokenStorage _tokenStorage;

  Future<List<ReportSummaryRow>> getSummary({
    DateTime? from,
    DateTime? to,
    String groupBy = 'day',
  }) async {
    final token = await _requireToken();
    final query = <String, String>{'group_by': groupBy};
    if (from != null) query['from'] = _formatDate(from);
    if (to != null) query['to'] = _formatDate(to);

    final response = await _apiClient.get(
      '/reports/summary',
      token: token,
      query: query,
    );
    final data = response['data'];
    if (data is! List) return [];
    return data
        .whereType<Map<String, dynamic>>()
        .map(ReportSummaryRow.fromJson)
        .toList(growable: false);
  }

  Future<List<AgentReportRow>> getAgentReports({
    DateTime? from,
    DateTime? to,
  }) async {
    final token = await _requireToken();
    final query = <String, String>{};
    if (from != null) query['from'] = _formatDate(from);
    if (to != null) query['to'] = _formatDate(to);

    final response = await _apiClient.get(
      '/reports/agents',
      token: token,
      query: query,
    );
    final data = response['data'];
    if (data is! List) return [];
    return data
        .whereType<Map<String, dynamic>>()
        .map(AgentReportRow.fromJson)
        .toList(growable: false);
  }

  Future<CriticalTrend> getCriticalTrend({int hours = 24}) async {
    final token = await _requireToken();
    final response = await _apiClient.get(
      '/reports/critical-trends',
      token: token,
      query: {'hours': '$hours'},
    );
    return CriticalTrend.fromJson(response['data'] as Map<String, dynamic>);
  }

  Future<int> getQueueSize() async {
    final token = await _requireToken();
    final response = await _apiClient.get('/reports/queue-size', token: token);
    final data = response['data'] as Map<String, dynamic>;
    return (data['queue_size'] as num?)?.toInt() ?? 0;
  }

  Future<List<AppUser>> listAgents() async {
    final token = await _requireToken();
    final response = await _apiClient.get(
      '/auth/admin/agents',
      token: token,
    );
    final data = response['data'];
    if (data is! List) return [];
    return data
        .whereType<Map<String, dynamic>>()
        .map(AppUser.fromJson)
        .toList(growable: false);
  }

  Future<void> createStaff({
    required String email,
    required String password,
    required String role,
  }) async {
    final token = await _requireToken();
    await _apiClient.post(
      '/auth/admin/staff',
      token: token,
      body: {'email': email, 'password': password, 'role': role},
    );
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
