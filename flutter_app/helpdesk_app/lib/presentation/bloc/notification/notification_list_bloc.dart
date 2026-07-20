import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/data/notification_repository.dart';
import 'package:helpdesk_app/models/notification.dart';

sealed class NotificationListEvent {
  const NotificationListEvent();
}

class NotificationListRequested extends NotificationListEvent {
  const NotificationListRequested({this.unreadOnly = false});

  final bool unreadOnly;
}

class NotificationMarkReadRequested extends NotificationListEvent {
  const NotificationMarkReadRequested(this.id);

  final int id;
}

class NotificationMarkAllReadRequested extends NotificationListEvent {
  const NotificationMarkAllReadRequested();
}

sealed class NotificationListState {
  const NotificationListState();
}

class NotificationListInitial extends NotificationListState {
  const NotificationListInitial();
}

class NotificationListLoading extends NotificationListState {
  const NotificationListLoading();
}

class NotificationListLoaded extends NotificationListState {
  const NotificationListLoaded(this.items, {this.unreadOnly = false});

  final List<AppNotification> items;
  final bool unreadOnly;
}

class NotificationListFailure extends NotificationListState {
  const NotificationListFailure(this.message, [this.items = const []]);

  final String message;
  final List<AppNotification> items;
}

class NotificationListBloc
    extends Bloc<NotificationListEvent, NotificationListState> {
  NotificationListBloc({required NotificationRepository notificationRepository})
    : _notificationRepository = notificationRepository,
      super(const NotificationListInitial()) {
    on<NotificationListRequested>(_onRequested);
    on<NotificationMarkReadRequested>(_onMarkReadRequested);
    on<NotificationMarkAllReadRequested>(_onMarkAllReadRequested);
  }

  final NotificationRepository _notificationRepository;

  Future<void> _onRequested(
    NotificationListRequested event,
    Emitter<NotificationListState> emit,
  ) async {
    emit(const NotificationListLoading());
    try {
      final items = await _notificationRepository.getNotifications(
        unreadOnly: event.unreadOnly,
      );
      emit(NotificationListLoaded(items, unreadOnly: event.unreadOnly));
    } catch (error) {
      emit(NotificationListFailure(error.toString()));
    }
  }

  Future<void> _onMarkReadRequested(
    NotificationMarkReadRequested event,
    Emitter<NotificationListState> emit,
  ) async {
    final current = state;
    if (current is! NotificationListLoaded) return;

    try {
      await _notificationRepository.markRead(event.id);
      final updated = current.items
          .map((n) => n.id == event.id ? n.copyWith(read: true) : n)
          .toList(growable: false);
      emit(NotificationListLoaded(updated, unreadOnly: current.unreadOnly));
    } catch (error) {
      emit(NotificationListFailure(error.toString(), current.items));
    }
  }

  Future<void> _onMarkAllReadRequested(
    NotificationMarkAllReadRequested event,
    Emitter<NotificationListState> emit,
  ) async {
    final current = state;
    if (current is! NotificationListLoaded) return;

    try {
      await _notificationRepository.markAllRead();
      final updated = current.items
          .map((n) => n.copyWith(read: true))
          .toList(growable: false);
      emit(NotificationListLoaded(updated, unreadOnly: current.unreadOnly));
    } catch (error) {
      emit(NotificationListFailure(error.toString(), current.items));
    }
  }
}
