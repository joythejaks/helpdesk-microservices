import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/core/services/file_saver/file_saver.dart';
import 'package:helpdesk_app/core/theme/helpdesk_theme.dart';
import 'package:helpdesk_app/data/admin_repository.dart';
import 'package:helpdesk_app/data/ticket_repository.dart';
import 'package:helpdesk_app/models/app_user.dart';
import 'package:helpdesk_app/models/ticket.dart';
import 'package:helpdesk_app/models/ticket_attachment.dart';
import 'package:helpdesk_app/models/ticket_comment.dart';
import 'package:helpdesk_app/presentation/bloc/attachment/attachment_bloc.dart';
import 'package:helpdesk_app/presentation/bloc/auth/auth_bloc.dart';
import 'package:helpdesk_app/presentation/bloc/comment/comment_bloc.dart';
import 'package:helpdesk_app/presentation/bloc/ticket/ticket_bloc.dart';
import 'package:helpdesk_app/presentation/widgets/app_text_field.dart';
import 'package:helpdesk_app/presentation/widgets/detail_row.dart';
import 'package:helpdesk_app/presentation/widgets/gradient_button.dart';
import 'package:helpdesk_app/presentation/widgets/header_bar.dart';
import 'package:helpdesk_app/presentation/widgets/surface_card.dart';
import 'package:helpdesk_app/presentation/widgets/timeline_item.dart';

class TicketDetailScreen extends StatefulWidget {
  final Ticket ticket;
  const TicketDetailScreen({super.key, required this.ticket});

  @override
  State<TicketDetailScreen> createState() => _TicketDetailScreenState();
}

class _TicketDetailScreenState extends State<TicketDetailScreen> {
  final _commentController = TextEditingController();
  List<AppUser> _agents = const [];
  int? _selectedAgentId;

  @override
  void initState() {
    super.initState();
    final authState = context.read<AuthBloc>().state;
    if (authState is Authenticated && authState.user.role.toLowerCase() == 'admin') {
      _loadAgents();
    }
  }

  Future<void> _loadAgents() async {
    try {
      final agents = await context.read<AdminRepository>().listAgents();
      if (!mounted) return;
      setState(() => _agents = agents);
    } catch (_) {
      // Best-effort — assign UI just shows an empty picker on failure.
    }
  }

  @override
  void dispose() {
    _commentController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return MultiBlocProvider(
      providers: [
        BlocProvider(
          create: (context) => CommentBloc(
            ticketRepository: context.read<TicketRepository>(),
          )..add(CommentsRequested(widget.ticket.id)),
        ),
        BlocProvider(
          create: (context) => AttachmentBloc(
            ticketRepository: context.read<TicketRepository>(),
          )..add(AttachmentsRequested(widget.ticket.id)),
        ),
      ],
      child: Scaffold(
        backgroundColor: HelpdeskTheme.surface,
        body: SafeArea(
          child: Builder(
            builder: (context) {
              final ticket = _liveTicket(context);
              final authState = context.watch<AuthBloc>().state;
              final currentUser = authState is Authenticated ? authState.user : null;

              return Column(
                children: [
                  Padding(
                    padding: const EdgeInsets.fromLTRB(24, 20, 24, 10),
                    child: HeaderBar(
                      title: 'Ticket Detail',
                      subtitle: 'ID: #${ticket.id}',
                      leading: Icons.arrow_back,
                      onLeadingTap: () => Navigator.pop(context),
                    ),
                  ),
                  Expanded(
                    child: ListView(
                      padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
                      children: [
                        _buildStatusHeader(ticket),
                        const SizedBox(height: 24),
                        Text('Judul Kendala', style: Theme.of(context).textTheme.titleSmall),
                        const SizedBox(height: 8),
                        Text(ticket.title, style: Theme.of(context).textTheme.headlineSmall),
                        const SizedBox(height: 20),
                        Text('Deskripsi', style: Theme.of(context).textTheme.titleSmall),
                        const SizedBox(height: 8),
                        SurfaceCard(
                          child: Text(
                            ticket.description,
                            style: const TextStyle(height: 1.6, fontSize: 15),
                          ),
                        ),
                        const SizedBox(height: 24),
                        Text('Detail', style: Theme.of(context).textTheme.titleSmall),
                        const SizedBox(height: 8),
                        SurfaceCard(child: _buildDetails(ticket)),
                        const SizedBox(height: 24),
                        Row(
                          children: [
                            Text('Lampiran', style: Theme.of(context).textTheme.titleSmall),
                            const Spacer(),
                            TextButton.icon(
                              onPressed: () => _pickAndUpload(context, ticket.id),
                              icon: const Icon(Icons.attach_file, size: 18),
                              label: const Text('Tambah'),
                            ),
                          ],
                        ),
                        ..._buildAttachments(context, ticket),
                        const SizedBox(height: 8),
                        if (currentUser != null) ..._buildActions(context, ticket, currentUser),
                        const SizedBox(height: 8),
                        ..._buildTimeline(ticket),
                        const SizedBox(height: 8),
                        const Divider(),
                        const SizedBox(height: 24),
                        Text('Percakapan & Aktivitas', style: Theme.of(context).textTheme.titleMedium),
                        const SizedBox(height: 16),
                        ..._buildComments(context),
                      ],
                    ),
                  ),
                  _buildCommentInputArea(context),
                ],
              );
            },
          ),
        ),
      ),
    );
  }

  Ticket _liveTicket(BuildContext context) {
    final tickets = switch (context.watch<TicketBloc>().state) {
      TicketLoaded(:final tickets) => tickets,
      TicketMutating(:final tickets) => tickets,
      TicketCreating(:final tickets) => tickets,
      TicketFailure(:final tickets) => tickets,
      _ => <Ticket>[],
    };
    for (final t in tickets) {
      if (t.id == widget.ticket.id) return t;
    }
    return widget.ticket;
  }

  Widget _buildStatusHeader(Ticket ticket) {
    return Row(
      children: [
        _buildBadge(ticket.status, HelpdeskTheme.primary),
        const SizedBox(width: 8),
        _buildBadge(ticket.priority, Colors.orange),
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

  Widget _buildDetails(Ticket ticket) {
    return Column(
      children: [
        DetailRow(
          label: 'Departemen',
          value: ticket.department.isNotEmpty ? ticket.department : '-',
        ),
        DetailRow(label: 'Prioritas', value: ticket.priority),
        DetailRow(
          label: 'Pemohon',
          value: ticket.userId != null ? 'User #${ticket.userId}' : '-',
        ),
        DetailRow(
          label: 'Agent',
          value: ticket.assignedAgentId != null
              ? 'Agent #${ticket.assignedAgentId}'
              : 'Belum ditugaskan',
        ),
        DetailRow(
          label: 'Batas Waktu',
          value: ticket.dueAt != null ? _formatDateTime(ticket.dueAt!) : '-',
        ),
      ],
    );
  }

  List<Widget> _buildActions(BuildContext context, Ticket ticket, AppUser currentUser) {
    final role = currentUser.role.toLowerCase();
    final widgets = <Widget>[];

    if (role == 'agent' && ticket.assignedAgentId == null) {
      widgets.add(
        GradientButton(
          label: 'Klaim Tiket',
          icon: Icons.assignment_ind_outlined,
          onPressed: () => context.read<TicketBloc>().add(
            TicketAssignRequested(ticketId: ticket.id),
          ),
        ),
      );
    } else if (role == 'admin') {
      widgets.add(
        Row(
          children: [
            Expanded(
              child: DropdownButtonFormField<int>(
                initialValue: _selectedAgentId,
                decoration: const InputDecoration(labelText: 'Pilih agent'),
                items: _agents
                    .map((a) => DropdownMenuItem(value: a.id, child: Text(a.email)))
                    .toList(),
                onChanged: (value) => setState(() => _selectedAgentId = value),
              ),
            ),
            const SizedBox(width: 12),
            FilledButton(
              onPressed: _selectedAgentId == null
                  ? null
                  : () => context.read<TicketBloc>().add(
                      TicketAssignRequested(
                        ticketId: ticket.id,
                        agentId: _selectedAgentId,
                      ),
                    ),
              child: const Text('Assign'),
            ),
          ],
        ),
      );
    }

    final canTransition = role == 'admin' ||
        (role == 'agent' && ticket.assignedAgentId == currentUser.id);
    if (canTransition && ticket.legalNextStatuses.isNotEmpty) {
      if (widgets.isNotEmpty) widgets.add(const SizedBox(height: 16));
      widgets.add(
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: ticket.legalNextStatuses.map((next) {
            return OutlinedButton(
              onPressed: () => context.read<TicketBloc>().add(
                TicketStatusChangeRequested(ticketId: ticket.id, status: next),
              ),
              child: Text('Ubah ke ${Ticket.formatStatusLabel(next)}'),
            );
          }).toList(),
        ),
      );
    }

    if (widgets.isEmpty) return const [];
    return [...widgets, const SizedBox(height: 8)];
  }

  List<Widget> _buildTimeline(Ticket ticket) {
    final entries = <Widget>[];
    if (ticket.createdAt != null) {
      entries.add(TimelineItem(title: 'Tiket dibuat', time: _formatDateTime(ticket.createdAt!)));
    }
    if (ticket.assignedAt != null) {
      entries.add(TimelineItem(title: 'Ditugaskan ke agent', time: _formatDateTime(ticket.assignedAt!)));
    }
    if (ticket.resolvedAt != null) {
      entries.add(TimelineItem(title: 'Ditandai selesai', time: _formatDateTime(ticket.resolvedAt!)));
    }
    if (ticket.closedAt != null) {
      entries.add(TimelineItem(title: 'Tiket ditutup', time: _formatDateTime(ticket.closedAt!)));
    }
    return entries;
  }

  List<Widget> _buildAttachments(BuildContext context, Ticket ticket) {
    return [
      BlocBuilder<AttachmentBloc, AttachmentState>(
        builder: (context, state) {
          final attachments = switch (state) {
            AttachmentLoaded(:final attachments) => attachments,
            AttachmentUploading(:final attachments) => attachments,
            AttachmentFailure(:final attachments) => attachments,
            _ => <TicketAttachment>[],
          };

          if (state is AttachmentLoading) {
            return const Padding(
              padding: EdgeInsets.symmetric(vertical: 12),
              child: Center(child: CircularProgressIndicator()),
            );
          }
          if (state is AttachmentFailure && attachments.isEmpty) {
            return Padding(
              padding: const EdgeInsets.symmetric(vertical: 8),
              child: Text(state.message),
            );
          }
          if (attachments.isEmpty) {
            return const Padding(
              padding: EdgeInsets.symmetric(vertical: 8),
              child: Text('Belum ada lampiran.'),
            );
          }
          return Column(
            children: [
              ...attachments.map(
                (a) => _buildAttachmentTile(context, ticket.id, a),
              ),
              if (state is AttachmentUploading)
                const Padding(
                  padding: EdgeInsets.symmetric(vertical: 8),
                  child: LinearProgressIndicator(),
                ),
            ],
          );
        },
      ),
    ];
  }

  Widget _buildAttachmentTile(
    BuildContext context,
    String ticketId,
    TicketAttachment attachment,
  ) {
    return ListTile(
      contentPadding: EdgeInsets.zero,
      leading: Icon(_iconForContentType(attachment.contentType), color: HelpdeskTheme.primary),
      title: Text(attachment.filename, overflow: TextOverflow.ellipsis),
      subtitle: Text(_formatFileSize(attachment.size)),
      trailing: const Icon(Icons.download_outlined),
      onTap: () => _downloadAttachment(context, ticketId, attachment),
    );
  }

  IconData _iconForContentType(String contentType) {
    if (contentType.startsWith('image/')) return Icons.image_outlined;
    if (contentType == 'application/pdf') return Icons.picture_as_pdf_outlined;
    return Icons.insert_drive_file_outlined;
  }

  String _formatFileSize(int bytes) {
    if (bytes < 1024) return '$bytes B';
    if (bytes < 1024 * 1024) return '${(bytes / 1024).toStringAsFixed(1)} KB';
    return '${(bytes / (1024 * 1024)).toStringAsFixed(1)} MB';
  }

  Future<void> _pickAndUpload(BuildContext context, String ticketId) async {
    final result = await FilePicker.platform.pickFiles(withData: true);
    final file = result?.files.single;
    if (file == null || file.bytes == null) return;
    if (!context.mounted) return;
    context.read<AttachmentBloc>().add(
      AttachmentUploadRequested(ticketId, file.name, file.bytes!),
    );
  }

  Future<void> _downloadAttachment(
    BuildContext context,
    String ticketId,
    TicketAttachment attachment,
  ) async {
    try {
      final result = await context.read<TicketRepository>().downloadAttachment(
        ticketId: ticketId,
        attachmentId: '${attachment.id}',
      );
      final saved = await saveBytes(attachment.filename, result.bytes);
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Tersimpan: $saved')),
      );
    } catch (error) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Gagal mengunduh: $error')),
      );
    }
  }

  List<Widget> _buildComments(BuildContext context) {
    return [
      BlocBuilder<CommentBloc, CommentState>(
        builder: (context, state) {
          final comments = switch (state) {
            CommentLoaded(:final comments) => comments,
            CommentSubmitting(:final comments) => comments,
            CommentFailure(:final comments) => comments,
            _ => <TicketComment>[],
          };

          if (state is CommentLoading) {
            return const Padding(
              padding: EdgeInsets.symmetric(vertical: 16),
              child: Center(child: CircularProgressIndicator()),
            );
          }
          if (state is CommentFailure && comments.isEmpty) {
            return Padding(
              padding: const EdgeInsets.symmetric(vertical: 16),
              child: Text(state.message),
            );
          }
          if (comments.isEmpty) {
            return const Padding(
              padding: EdgeInsets.symmetric(vertical: 16),
              child: Text('Belum ada percakapan.'),
            );
          }
          return Column(
            children: comments
                .map(
                  (c) => _buildCommentTile(
                    '${c.authorRole[0].toUpperCase()}${c.authorRole.substring(1)} #${c.authorId}',
                    c.body,
                    _formatDateTime(c.createdAt),
                  ),
                )
                .toList(),
          );
        },
      ),
    ];
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

  Widget _buildCommentInputArea(BuildContext context) {
    return Container(
      padding: const EdgeInsets.fromLTRB(24, 12, 24, 32),
      decoration: BoxDecoration(
        color: HelpdeskTheme.surface,
        boxShadow: [BoxShadow(color: Colors.black.withValues(alpha: 13), blurRadius: 10, offset: const Offset(0, -5))],
      ),
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
          BlocBuilder<CommentBloc, CommentState>(
            builder: (context, state) {
              final submitting = state is CommentSubmitting;
              return IconButton.filled(
                onPressed: submitting ? null : () => _submitComment(context),
                icon: const Icon(Icons.send),
                style: IconButton.styleFrom(backgroundColor: HelpdeskTheme.primary),
              );
            },
          ),
        ],
      ),
    );
  }

  void _submitComment(BuildContext context) {
    final text = _commentController.text.trim();
    if (text.isEmpty) return;
    context.read<CommentBloc>().add(CommentSubmitted(widget.ticket.id, text));
    _commentController.clear();
  }

  static String _formatDateTime(DateTime value) {
    final local = value.toLocal();
    final d = local.day.toString().padLeft(2, '0');
    final m = local.month.toString().padLeft(2, '0');
    final h = local.hour.toString().padLeft(2, '0');
    final min = local.minute.toString().padLeft(2, '0');
    return '$d/$m/${local.year} $h:$min';
  }
}
