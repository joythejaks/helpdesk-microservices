class TicketAttachment {
  const TicketAttachment({
    required this.id,
    required this.ticketId,
    required this.uploaderId,
    required this.filename,
    required this.contentType,
    required this.size,
    required this.createdAt,
  });

  final int id;
  final int ticketId;
  final int uploaderId;
  final String filename;
  final String contentType;
  final int size;
  final DateTime createdAt;

  factory TicketAttachment.fromJson(Map<String, dynamic> json) {
    return TicketAttachment(
      id: json['id'] as int,
      ticketId: json['ticket_id'] as int,
      uploaderId: json['uploader_id'] as int,
      filename: (json['filename'] as String?) ?? '',
      contentType: (json['content_type'] as String?) ?? 'application/octet-stream',
      size: (json['size'] as num?)?.toInt() ?? 0,
      createdAt:
          DateTime.tryParse((json['created_at'] as String?) ?? '') ??
          DateTime.now(),
    );
  }
}
