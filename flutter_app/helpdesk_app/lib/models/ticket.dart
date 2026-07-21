class Ticket {
  const Ticket({
    required this.id,
    required this.title,
    this.department = '',
    required this.status,
    required this.rawStatus,
    this.priority = 'Medium',
    required this.time,
    required this.description,
    this.progress = .1,
    this.userId,
    this.assignedAgentId,
    this.createdAt,
    this.dueAt,
    this.assignedAt,
    this.resolvedAt,
    this.closedAt,
  });

  final String id;
  final String title;
  final String department;
  final String status; // display label, e.g. "In Progress"
  final String
  rawStatus; // backend value, e.g. "in_progress" — PATCH payloads use this
  final String priority;
  final String time;
  final String description;
  final double progress;
  final int? userId;
  final int? assignedAgentId;
  final DateTime? createdAt;
  final DateTime? dueAt;
  final DateTime? assignedAt;
  final DateTime? resolvedAt;
  final DateTime? closedAt;

  static const _transitions = <String, List<String>>{
    'assigned': ['in_progress'],
    'in_progress': ['pending', 'resolved'],
    'pending': ['in_progress', 'resolved'],
    'resolved': ['closed', 'in_progress'],
  };

  /// Legal next raw statuses per the backend's state machine — 'open' and
  /// 'closed' resolve to an empty list ('open' only advances via the
  /// /assign endpoint, 'closed' is terminal).
  List<String> get legalNextStatuses => _transitions[rawStatus] ?? const [];

  /// Public wrapper so UI can render a display label for a raw status value
  /// it doesn't have a full [Ticket] for yet (e.g. a legal-next-status button).
  static String formatStatusLabel(String rawStatus) => _formatStatus(rawStatus);

  factory Ticket.fromJson(Map<String, dynamic> json) {
    final rawStatus = ((json['status'] as String?) ?? 'open').toLowerCase();
    final createdAt = DateTime.tryParse((json['created_at'] as String?) ?? '');
    final priority = json['priority'] as String?;

    return Ticket(
      id: '${json['id'] ?? ''}',
      title: (json['title'] as String?) ?? '-',
      description: (json['description'] as String?) ?? '',
      status: _formatStatus(rawStatus),
      rawStatus: rawStatus,
      priority: (priority != null && priority.isNotEmpty) ? priority : 'Medium',
      department: (json['department'] as String?) ?? '',
      time: _relativeTime(createdAt),
      userId: json['user_id'] is int ? json['user_id'] as int : null,
      assignedAgentId: json['assigned_agent_id'] is int
          ? json['assigned_agent_id'] as int
          : null,
      createdAt: createdAt,
      dueAt: DateTime.tryParse((json['due_at'] as String?) ?? ''),
      assignedAt: DateTime.tryParse((json['assigned_at'] as String?) ?? ''),
      resolvedAt: DateTime.tryParse((json['resolved_at'] as String?) ?? ''),
      closedAt: DateTime.tryParse((json['closed_at'] as String?) ?? ''),
      progress: _progressForStatus(rawStatus),
    );
  }

  static String _formatStatus(String value) {
    return switch (value) {
      'open' => 'Open',
      'assigned' => 'Assigned',
      'in_progress' => 'In Progress',
      'pending' => 'Pending',
      'resolved' => 'Resolved',
      'closed' => 'Closed',
      _ => value,
    };
  }

  static double _progressForStatus(String value) {
    return switch (value) {
      'closed' => 1,
      'resolved' => .9,
      'pending' => .5,
      'in_progress' => .62,
      'assigned' => .35,
      _ => .1, // open / unknown
    };
  }

  static String _relativeTime(DateTime? value) {
    if (value == null) return 'baru saja';

    final diff = DateTime.now().difference(value.toLocal());
    if (diff.inMinutes < 1) return 'baru saja';
    if (diff.inMinutes < 60) return '${diff.inMinutes} menit lalu';
    if (diff.inHours < 24) return '${diff.inHours} jam lalu';
    return '${diff.inDays} hari lalu';
  }
}
