import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/core/services/file_saver/file_saver.dart';
import 'package:helpdesk_app/data/ticket_repository.dart';
import 'package:helpdesk_app/models/ticket_attachment.dart';
import 'package:helpdesk_app/presentation/bloc/attachment/attachment_bloc.dart';
import 'package:helpdesk_app/presentation/widgets/app_text_field.dart';
import 'package:helpdesk_app/presentation/widgets/header_bar.dart';

enum _TypeFilter { all, images, documents }

enum _SortOrder { newest, oldest, nameAsc, largest }

class AttachmentsScreen extends StatelessWidget {
  const AttachmentsScreen({super.key, required this.ticketId});

  final String ticketId;

  @override
  Widget build(BuildContext context) {
    return BlocProvider(
      create: (context) =>
          AttachmentBloc(ticketRepository: context.read<TicketRepository>())
            ..add(AttachmentsRequested(ticketId)),
      child: _AttachmentsView(ticketId: ticketId),
    );
  }
}

class _AttachmentsView extends StatefulWidget {
  const _AttachmentsView({required this.ticketId});

  final String ticketId;

  @override
  State<_AttachmentsView> createState() => _AttachmentsViewState();
}

class _AttachmentsViewState extends State<_AttachmentsView> {
  final _searchController = TextEditingController();
  String _query = '';
  _TypeFilter _typeFilter = _TypeFilter.all;
  _SortOrder _sortOrder = _SortOrder.newest;

  @override
  void initState() {
    super.initState();
    _searchController.addListener(() {
      setState(() => _query = _searchController.text.trim().toLowerCase());
    });
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  List<TicketAttachment> _applyFilters(List<TicketAttachment> attachments) {
    var result = attachments.where((a) {
      final matchesQuery =
          _query.isEmpty || a.filename.toLowerCase().contains(_query);
      final matchesType = switch (_typeFilter) {
        _TypeFilter.all => true,
        _TypeFilter.images => a.contentType.startsWith('image/'),
        _TypeFilter.documents => !a.contentType.startsWith('image/'),
      };
      return matchesQuery && matchesType;
    }).toList();

    result.sort((a, b) {
      return switch (_sortOrder) {
        _SortOrder.newest => b.createdAt.compareTo(a.createdAt),
        _SortOrder.oldest => a.createdAt.compareTo(b.createdAt),
        _SortOrder.nameAsc => a.filename.toLowerCase().compareTo(
          b.filename.toLowerCase(),
        ),
        _SortOrder.largest => b.size.compareTo(a.size),
      };
    });
    return result;
  }

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return Scaffold(
      backgroundColor: colors.surface,
      floatingActionButton: FloatingActionButton(
        onPressed: () => _pickAndUpload(context),
        backgroundColor: colors.primary,
        foregroundColor: colors.onPrimary,
        child: const Icon(Icons.add),
      ),
      body: SafeArea(
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(24, 20, 24, 10),
              child: HeaderBar(
                title: 'Ticket Attachments',
                subtitle: 'Semua berkas pada tiket ini',
                leading: Icons.arrow_back,
                onLeadingTap: () => Navigator.pop(context),
              ),
            ),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 24),
              child: AppTextField(
                controller: _searchController,
                label: 'Cari berkas',
                icon: Icons.search,
              ),
            ),
            const SizedBox(height: 12),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 24),
              child: Row(
                children: [
                  Expanded(
                    child: Wrap(
                      spacing: 8,
                      children: [
                        ChoiceChip(
                          label: const Text('All'),
                          selected: _typeFilter == _TypeFilter.all,
                          onSelected: (_) =>
                              setState(() => _typeFilter = _TypeFilter.all),
                        ),
                        ChoiceChip(
                          label: const Text('Images'),
                          selected: _typeFilter == _TypeFilter.images,
                          onSelected: (_) =>
                              setState(() => _typeFilter = _TypeFilter.images),
                        ),
                        ChoiceChip(
                          label: const Text('Documents'),
                          selected: _typeFilter == _TypeFilter.documents,
                          onSelected: (_) => setState(
                            () => _typeFilter = _TypeFilter.documents,
                          ),
                        ),
                      ],
                    ),
                  ),
                  PopupMenuButton<_SortOrder>(
                    icon: const Icon(Icons.sort),
                    onSelected: (value) => setState(() => _sortOrder = value),
                    itemBuilder: (context) => const [
                      PopupMenuItem(
                        value: _SortOrder.newest,
                        child: Text('Terbaru'),
                      ),
                      PopupMenuItem(
                        value: _SortOrder.oldest,
                        child: Text('Terlama'),
                      ),
                      PopupMenuItem(
                        value: _SortOrder.nameAsc,
                        child: Text('Nama A-Z'),
                      ),
                      PopupMenuItem(
                        value: _SortOrder.largest,
                        child: Text('Ukuran terbesar'),
                      ),
                    ],
                  ),
                ],
              ),
            ),
            const SizedBox(height: 8),
            Expanded(
              child: BlocBuilder<AttachmentBloc, AttachmentState>(
                builder: (context, state) {
                  if (state is AttachmentLoading ||
                      state is AttachmentInitial) {
                    return const Center(child: CircularProgressIndicator());
                  }
                  final all = switch (state) {
                    AttachmentLoaded(:final attachments) => attachments,
                    AttachmentUploading(:final attachments) => attachments,
                    AttachmentFailure(:final attachments) => attachments,
                    _ => <TicketAttachment>[],
                  };
                  if (state is AttachmentFailure && all.isEmpty) {
                    return Center(child: Text(state.message));
                  }
                  final filtered = _applyFilters(all);

                  return Column(
                    children: [
                      Padding(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 24,
                          vertical: 4,
                        ),
                        child: Row(
                          children: [
                            Text(
                              '${filtered.length} ATTACHMENTS',
                              style: Theme.of(context).textTheme.bodySmall,
                            ),
                          ],
                        ),
                      ),
                      if (state is AttachmentUploading)
                        const Padding(
                          padding: EdgeInsets.symmetric(
                            horizontal: 24,
                            vertical: 4,
                          ),
                          child: LinearProgressIndicator(),
                        ),
                      Expanded(
                        child: filtered.isEmpty
                            ? const Center(
                                child: Text('Tidak ada berkas yang cocok.'),
                              )
                            : ListView.separated(
                                padding: const EdgeInsets.symmetric(
                                  horizontal: 24,
                                  vertical: 8,
                                ),
                                itemCount: filtered.length,
                                separatorBuilder: (_, _) =>
                                    const Divider(height: 1),
                                itemBuilder: (context, index) =>
                                    _AttachmentTile(
                                      attachment: filtered[index],
                                      onTap: () =>
                                          _download(context, filtered[index]),
                                    ),
                              ),
                      ),
                    ],
                  );
                },
              ),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _pickAndUpload(BuildContext context) async {
    final result = await FilePicker.platform.pickFiles(withData: true);
    final file = result?.files.single;
    if (file == null || file.bytes == null) return;
    if (!context.mounted) return;
    context.read<AttachmentBloc>().add(
      AttachmentUploadRequested(widget.ticketId, file.name, file.bytes!),
    );
  }

  Future<void> _download(
    BuildContext context,
    TicketAttachment attachment,
  ) async {
    try {
      final result = await context.read<TicketRepository>().downloadAttachment(
        ticketId: widget.ticketId,
        attachmentId: '${attachment.id}',
      );
      final saved = await saveBytes(attachment.filename, result.bytes);
      if (!context.mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Tersimpan: $saved')));
    } catch (error) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Gagal mengunduh: $error')));
    }
  }
}

class _AttachmentTile extends StatelessWidget {
  const _AttachmentTile({required this.attachment, required this.onTap});

  final TicketAttachment attachment;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return ListTile(
      onTap: onTap,
      contentPadding: EdgeInsets.zero,
      leading: CircleAvatar(
        backgroundColor: colors.primaryContainer,
        child: Icon(
          _iconForContentType(attachment.contentType),
          color: colors.primary,
        ),
      ),
      title: Text(attachment.filename, overflow: TextOverflow.ellipsis),
      subtitle: Text(
        '${_formatFileSize(attachment.size)} • Diunggah ${_formatDate(attachment.createdAt)}',
      ),
      trailing: const Icon(Icons.download_outlined),
    );
  }

  static IconData _iconForContentType(String contentType) {
    if (contentType.startsWith('image/')) return Icons.image_outlined;
    if (contentType == 'application/pdf') return Icons.picture_as_pdf_outlined;
    return Icons.insert_drive_file_outlined;
  }

  static String _formatFileSize(int bytes) {
    if (bytes < 1024) return '$bytes B';
    if (bytes < 1024 * 1024) return '${(bytes / 1024).toStringAsFixed(1)} KB';
    return '${(bytes / (1024 * 1024)).toStringAsFixed(1)} MB';
  }

  static String _formatDate(DateTime value) {
    final local = value.toLocal();
    final d = local.day.toString().padLeft(2, '0');
    final m = local.month.toString().padLeft(2, '0');
    return '$d/$m/${local.year}';
  }
}
