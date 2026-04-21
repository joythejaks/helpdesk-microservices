class Ticket {
  const Ticket({
    required this.id,
    required this.title,
    this.requester = 'Requester',
    this.department = 'Helpdesk',
    required this.status,
    this.priority = 'Medium',
    required this.time,
    required this.description,
    this.progress = .25,
    this.userId,
    this.createdAt,
  });

  final String id;
  final String title;
  final String requester;
  final String department;
  final String status;
  final String priority;
  final String time;
  final String description;
  final double progress;
  final int? userId;
  final DateTime? createdAt;

  factory Ticket.fromJson(Map<String, dynamic> json) {
    final rawStatus = (json['status'] as String?) ?? 'open';
    final createdAt = DateTime.tryParse((json['created_at'] as String?) ?? '');

    return Ticket(
      id: '${json['id'] ?? ''}',
      title: (json['title'] as String?) ?? '-',
      description: (json['description'] as String?) ?? '',
      status: _formatStatus(rawStatus),
      time: _relativeTime(createdAt),
      userId: json['user_id'] is int ? json['user_id'] as int : null,
      createdAt: createdAt,
      progress: _progressForStatus(rawStatus),
    );
  }

  static String _formatStatus(String value) {
    return switch (value.toLowerCase()) {
      'open' => 'Open',
      'in_progress' || 'in progress' => 'In Progress',
      'resolved' || 'closed' => 'Resolved',
      _ => value,
    };
  }

  static double _progressForStatus(String value) {
    return switch (value.toLowerCase()) {
      'resolved' || 'closed' => 1,
      'in_progress' || 'in progress' => .62,
      _ => .25,
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
