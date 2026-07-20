class AppNotification {
  const AppNotification({
    required this.id,
    required this.eventType,
    this.ticketId,
    this.title,
    this.status,
    required this.read,
    required this.createdAt,
  });

  final int id;
  final String eventType;
  final int? ticketId;
  final String? title;
  final String? status;
  final bool read;
  final DateTime createdAt;

  factory AppNotification.fromJson(Map<String, dynamic> json) {
    final event = (json['event'] as Map<String, dynamic>?) ?? const {};
    return AppNotification(
      id: json['id'] as int,
      eventType: (event['type'] as String?) ?? 'unknown',
      ticketId: event['ticket_id'] as int?,
      title: event['title'] as String?,
      status: event['status'] as String?,
      read: (json['read'] as bool?) ?? false,
      createdAt:
          DateTime.tryParse((json['created_at'] as String?) ?? '') ??
          DateTime.now(),
    );
  }

  AppNotification copyWith({bool? read}) {
    return AppNotification(
      id: id,
      eventType: eventType,
      ticketId: ticketId,
      title: title,
      status: status,
      read: read ?? this.read,
      createdAt: createdAt,
    );
  }
}
