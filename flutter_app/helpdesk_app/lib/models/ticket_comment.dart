class TicketComment {
  const TicketComment({
    required this.id,
    required this.ticketId,
    required this.authorId,
    required this.authorRole,
    required this.body,
    this.isInternal = false,
    required this.createdAt,
  });

  final int id;
  final int ticketId;
  final int authorId;
  final String authorRole;
  final String body;
  final bool isInternal;
  final DateTime createdAt;

  factory TicketComment.fromJson(Map<String, dynamic> json) {
    return TicketComment(
      id: json['id'] as int,
      ticketId: json['ticket_id'] as int,
      authorId: json['author_id'] as int,
      authorRole: (json['author_role'] as String?) ?? '',
      body: (json['body'] as String?) ?? '',
      isInternal: (json['is_internal'] as bool?) ?? false,
      createdAt:
          DateTime.tryParse((json['created_at'] as String?) ?? '') ??
          DateTime.now(),
    );
  }
}
