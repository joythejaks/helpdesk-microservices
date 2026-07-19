import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/core/services/websocket_service.dart';
import 'package:helpdesk_app/presentation/bloc/ticket/ticket_bloc.dart';

sealed class NotificationEvent {
  const NotificationEvent();
}

class NotificationConnectRequested extends NotificationEvent {
  const NotificationConnectRequested(this.token);

  final String token;
}

class NotificationDisconnectRequested extends NotificationEvent {
  const NotificationDisconnectRequested();
}

class NotificationCleared extends NotificationEvent {
  const NotificationCleared();
}

class _NotificationMessageReceived extends NotificationEvent {
  const _NotificationMessageReceived(this.payload);

  final Map<String, dynamic> payload;
}

class NotificationState {
  const NotificationState({this.unreadCount = 0});

  final int unreadCount;
}

const _ticketEventTypes = {
  'ticket_created',
  'ticket_assigned',
  'ticket_status_changed',
  'ticket_commented',
  'ticket_attachment_added',
};

/// Owns the WebSocketService connection lifecycle — instantiating it here
/// is what actually wires up the service (previously built but never used
/// anywhere in the app). Incoming ticket events trigger a TicketBloc
/// refresh (preserving whatever filter is active) and bump an unread badge
/// count surfaced on each dashboard's HeaderBar.
class NotificationBloc extends Bloc<NotificationEvent, NotificationState> {
  NotificationBloc({
    required WebSocketService webSocketService,
    required TicketBloc ticketBloc,
  }) : _webSocketService = webSocketService,
       _ticketBloc = ticketBloc,
       super(const NotificationState()) {
    on<NotificationConnectRequested>(_onConnect);
    on<NotificationDisconnectRequested>(_onDisconnect);
    on<NotificationCleared>((event, emit) => emit(const NotificationState()));
    on<_NotificationMessageReceived>(_onMessageReceived);
  }

  final WebSocketService _webSocketService;
  final TicketBloc _ticketBloc;

  void _onConnect(
    NotificationConnectRequested event,
    Emitter<NotificationState> emit,
  ) {
    _webSocketService.connect(
      event.token,
      (payload) => add(_NotificationMessageReceived(payload)),
    );
  }

  void _onDisconnect(
    NotificationDisconnectRequested event,
    Emitter<NotificationState> emit,
  ) {
    _webSocketService.disconnect();
    emit(const NotificationState());
  }

  void _onMessageReceived(
    _NotificationMessageReceived event,
    Emitter<NotificationState> emit,
  ) {
    final type = event.payload['type'] as String?;
    if (type != null && _ticketEventTypes.contains(type)) {
      _ticketBloc.refresh();
    }
    emit(NotificationState(unreadCount: state.unreadCount + 1));
  }

  @override
  Future<void> close() {
    _webSocketService.disconnect();
    return super.close();
  }
}
