import 'package:flutter/material.dart';
import 'package:helpdesk_app/core/theme/helpdesk_theme.dart';
import 'package:helpdesk_app/models/ticket.dart';
import 'package:helpdesk_app/presentation/widgets/header_bar.dart';
import 'package:helpdesk_app/presentation/widgets/surface_card.dart';
import 'package:helpdesk_app/presentation/widgets/app_text_field.dart';
import 'package:helpdesk_app/presentation/widgets/gradient_button.dart';

class TicketDetailScreen extends StatefulWidget {
  final Ticket ticket;
  const TicketDetailScreen({super.key, required this.ticket});

  @override
  State<TicketDetailScreen> createState() => _TicketDetailScreenState();
}

class _TicketDetailScreenState extends State<TicketDetailScreen> {
  final _commentController = TextEditingController();
  final _formKey = GlobalKey<FormState>();

  @override
  void dispose() {
    _commentController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: HelpdeskTheme.surface,
      body: SafeArea(
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(24, 20, 24, 10),
              child: HeaderBar(
                title: 'Ticket Detail',
                subtitle: 'ID: ${widget.ticket.id.toUpperCase().substring(0, 8)}',
                leading: Icons.arrow_back,
                onLeadingTap: () => Navigator.pop(context),
              ),
            ),
            Expanded(
              child: ListView(
                padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
                children: [
                  _buildStatusHeader(),
                  const SizedBox(height: 24),
                  Text('Judul Kendala', style: Theme.of(context).textTheme.titleSmall),
                  const SizedBox(height: 8),
                  Text(widget.ticket.title, style: Theme.of(context).textTheme.headlineSmall),
                  const SizedBox(height: 20),
                  Text('Deskripsi', style: Theme.of(context).textTheme.titleSmall),
                  const SizedBox(height: 8),
                  SurfaceCard(
                    child: Text(
                      widget.ticket.description,
                      style: const TextStyle(height: 1.6, fontSize: 15),
                    ),
                  ),
                  const SizedBox(height: 32),
                  const Divider(),
                  const SizedBox(height: 24),
                  Text('Percakapan & Aktivitas', style: Theme.of(context).textTheme.titleMedium),
                  const SizedBox(height: 16),
                  _buildCommentTile('System', 'Tiket telah berhasil dibuat dan masuk antrean.', 'Tadi', isSystem: true),
                  // Placeholder untuk komentar selanjutnya
                ],
              ),
            ),
            _buildCommentInputArea(),
          ],
        ),
      ),
    );
  }

  Widget _buildStatusHeader() {
    return Row(
      children: [
        _buildBadge(widget.ticket.status, HelpdeskTheme.primary),
        const SizedBox(width: 8),
        _buildBadge(widget.ticket.priority, Colors.orange),
      ],
    );
  }

  Widget _buildBadge(String text, Color color) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 25),
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: color.withValues(alpha: 127)),
      ),
      child: Text(
        text,
        style: TextStyle(color: color, fontWeight: FontWeight.bold, fontSize: 12),
      ),
    );
  }

  Widget _buildCommentTile(String user, String msg, String time, {bool isSystem = false}) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 16),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          CircleAvatar(
            backgroundColor: isSystem ? Colors.grey[200] : HelpdeskTheme.primaryContainer,
            child: Icon(isSystem ? Icons.settings : Icons.person, size: 18),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Text(user, style: const TextStyle(fontWeight: FontWeight.bold)),
                    const Spacer(),
                    Text(time, style: TextStyle(color: Colors.grey[500], fontSize: 11)),
                  ],
                ),
                const SizedBox(height: 4),
                Text(msg, style: const TextStyle(height: 1.4)),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildCommentInputArea() {
    return Container(
      padding: const EdgeInsets.fromLTRB(24, 12, 24, 32),
      decoration: BoxDecoration(
        color: HelpdeskTheme.surface,
        boxShadow: [BoxShadow(color: Colors.black.withValues(alpha: 13), blurRadius: 10, offset: const Offset(0, -5))],
      ),
      child: Form(
        key: _formKey,
        child: Row(
          children: [
            Expanded(
              child: AppTextField(
                controller: _commentController,
                label: 'Tulis balasan...',
                icon: Icons.chat_bubble_outline,
              ),
            ),
            const SizedBox(width: 12),
            IconButton.filled(
              onPressed: _submitComment,
              icon: const Icon(Icons.send),
              style: IconButton.styleFrom(backgroundColor: HelpdeskTheme.primary),
            ),
          ],
        ),
      ),
    );
  }

  void _submitComment() {
    if (_commentController.text.trim().isEmpty) return;
    // Logic kirim komen via BLoC akan ditambahkan di sini
    _commentController.clear();
  }
}