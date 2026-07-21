import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/data/notification_repository.dart';
import 'package:helpdesk_app/models/notification.dart';
import 'package:helpdesk_app/presentation/bloc/notification/notification_list_bloc.dart';
import 'package:helpdesk_app/presentation/widgets/header_bar.dart';

class NotificationsScreen extends StatelessWidget {
  const NotificationsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return BlocProvider(
      create: (context) => NotificationListBloc(
        notificationRepository: context.read<NotificationRepository>(),
      )..add(const NotificationListRequested()),
      child: const _NotificationsView(),
    );
  }
}

class _NotificationsView extends StatefulWidget {
  const _NotificationsView();

  @override
  State<_NotificationsView> createState() => _NotificationsViewState();
}

class _NotificationsViewState extends State<_NotificationsView> {
  bool _unreadOnly = false;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Theme.of(context).colorScheme.surface,
      body: SafeArea(
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(24, 20, 24, 10),
              child: HeaderBar(
                title: 'Notifications',
                subtitle: 'Riwayat aktivitas tiketmu',
                leading: Icons.arrow_back,
                onLeadingTap: () => Navigator.pop(context),
                trailing: Icons.done_all,
                onTrailingTap: () => context.read<NotificationListBloc>().add(
                  const NotificationMarkAllReadRequested(),
                ),
              ),
            ),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 24),
              child: Row(
                children: [
                  ChoiceChip(
                    label: const Text('All'),
                    selected: !_unreadOnly,
                    onSelected: (_) => _setFilter(context, unreadOnly: false),
                  ),
                  const SizedBox(width: 8),
                  ChoiceChip(
                    label: const Text('Unread'),
                    selected: _unreadOnly,
                    onSelected: (_) => _setFilter(context, unreadOnly: true),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 12),
            Expanded(
              child: BlocBuilder<NotificationListBloc, NotificationListState>(
                builder: (context, state) {
                  if (state is NotificationListLoading ||
                      state is NotificationListInitial) {
                    return const Center(child: CircularProgressIndicator());
                  }
                  final items = switch (state) {
                    NotificationListLoaded(:final items) => items,
                    NotificationListFailure(:final items) => items,
                    _ => <AppNotification>[],
                  };
                  if (state is NotificationListFailure && items.isEmpty) {
                    return Center(child: Text(state.message));
                  }
                  if (items.isEmpty) {
                    return const Center(child: Text('Belum ada notifikasi.'));
                  }
                  return ListView.separated(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 24,
                      vertical: 8,
                    ),
                    itemCount: items.length,
                    separatorBuilder: (_, _) => const Divider(height: 1),
                    itemBuilder: (context, index) => _NotificationTile(
                      notification: items[index],
                      onTap: () => context.read<NotificationListBloc>().add(
                        NotificationMarkReadRequested(items[index].id),
                      ),
                    ),
                  );
                },
              ),
            ),
          ],
        ),
      ),
    );
  }

  void _setFilter(BuildContext context, {required bool unreadOnly}) {
    setState(() => _unreadOnly = unreadOnly);
    context.read<NotificationListBloc>().add(
      NotificationListRequested(unreadOnly: unreadOnly),
    );
  }
}

class _NotificationTile extends StatelessWidget {
  const _NotificationTile({required this.notification, required this.onTap});

  final AppNotification notification;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return ListTile(
      onTap: onTap,
      contentPadding: EdgeInsets.zero,
      leading: CircleAvatar(
        backgroundColor: notification.read
            ? colors.surfaceContainerHigh
            : colors.primaryContainer,
        child: Icon(_iconFor(notification.eventType), size: 18),
      ),
      title: Text(
        _titleFor(notification),
        style: TextStyle(
          fontWeight: notification.read ? FontWeight.normal : FontWeight.bold,
        ),
      ),
      subtitle: Text(_relativeTime(notification.createdAt)),
      trailing: notification.read
          ? null
          : Container(
              width: 8,
              height: 8,
              decoration: BoxDecoration(
                color: colors.primary,
                shape: BoxShape.circle,
              ),
            ),
    );
  }

  static IconData _iconFor(String eventType) {
    return switch (eventType) {
      'ticket_created' => Icons.confirmation_number_outlined,
      'ticket_assigned' => Icons.assignment_ind_outlined,
      'ticket_status_changed' => Icons.sync_alt,
      'ticket_commented' => Icons.chat_bubble_outline,
      'ticket_attachment_added' => Icons.attach_file,
      _ => Icons.notifications_outlined,
    };
  }

  static String _titleFor(AppNotification n) {
    if (n.title != null && n.title!.isNotEmpty) return n.title!;
    final ticketPart = n.ticketId != null ? 'Tiket #${n.ticketId}' : 'Tiket';
    return switch (n.eventType) {
      'ticket_created' => '$ticketPart dibuat',
      'ticket_assigned' => '$ticketPart ditugaskan',
      'ticket_status_changed' =>
        '$ticketPart status berubah${n.status != null ? ' ke ${n.status}' : ''}',
      'ticket_commented' => '$ticketPart mendapat komentar baru',
      'ticket_attachment_added' => '$ticketPart mendapat lampiran baru',
      _ => ticketPart,
    };
  }

  static String _relativeTime(DateTime value) {
    final diff = DateTime.now().difference(value.toLocal());
    if (diff.inMinutes < 1) return 'baru saja';
    if (diff.inMinutes < 60) return '${diff.inMinutes} menit lalu';
    if (diff.inHours < 24) return '${diff.inHours} jam lalu';
    return '${diff.inDays} hari lalu';
  }
}
