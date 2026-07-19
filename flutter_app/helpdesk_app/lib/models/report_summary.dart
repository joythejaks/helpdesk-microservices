class ReportSummaryRow {
  const ReportSummaryRow({
    required this.period,
    required this.status,
    required this.count,
  });

  final String period;
  final String status;
  final int count;

  factory ReportSummaryRow.fromJson(Map<String, dynamic> json) {
    return ReportSummaryRow(
      period: json['period'] as String,
      status: json['status'] as String,
      count: json['count'] as int,
    );
  }
}

class AgentReportRow {
  const AgentReportRow({
    required this.agentId,
    required this.totalAssigned,
    required this.totalResolved,
    required this.avgResolutionSeconds,
  });

  final int agentId;
  final int totalAssigned;
  final int totalResolved;
  final double avgResolutionSeconds;

  factory AgentReportRow.fromJson(Map<String, dynamic> json) {
    return AgentReportRow(
      agentId: json['agent_id'] as int,
      totalAssigned: json['total_assigned'] as int,
      totalResolved: json['total_resolved'] as int,
      avgResolutionSeconds: (json['avg_resolution_seconds'] as num).toDouble(),
    );
  }
}
