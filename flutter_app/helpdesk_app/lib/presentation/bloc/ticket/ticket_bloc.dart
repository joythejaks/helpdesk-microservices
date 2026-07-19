import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/data/ticket_repository.dart';
import 'package:helpdesk_app/models/ticket.dart';

sealed class TicketEvent {
  const TicketEvent();
}

class TicketsRequested extends TicketEvent {
  const TicketsRequested({
    this.scope,
    this.status,
    this.priority,
    this.department,
    this.search,
    this.overdue,
    this.from,
    this.to,
    this.page = 1,
    this.limit = 20,
  });

  final String? scope;
  final String? status;
  final String? priority;
  final String? department;
  final String? search;
  final bool? overdue;
  final DateTime? from;
  final DateTime? to;
  final int page;
  final int limit;
}

class TicketCreateSubmitted extends TicketEvent {
  const TicketCreateSubmitted({required this.title, required this.description});

  final String title;
  final String description;
}

class TicketAssignRequested extends TicketEvent {
  const TicketAssignRequested({required this.ticketId, this.agentId});

  final String ticketId;
  final int? agentId; // null => self-claim (agent role)
}

class TicketStatusChangeRequested extends TicketEvent {
  const TicketStatusChangeRequested({
    required this.ticketId,
    required this.status,
  });

  final String ticketId;
  final String status; // raw backend value, e.g. 'in_progress'
}

sealed class TicketState {
  const TicketState();
}

class TicketInitial extends TicketState {
  const TicketInitial();
}

class TicketLoading extends TicketState {
  const TicketLoading();
}

class TicketLoaded extends TicketState {
  const TicketLoaded(this.tickets);

  final List<Ticket> tickets;
}

class TicketCreating extends TicketState {
  const TicketCreating(this.tickets);

  final List<Ticket> tickets;
}

/// Ticket [ticketId] inside [tickets] is currently being assigned or
/// transitioned — lets list/detail UI show a per-row spinner instead of
/// blanking the whole list like TicketCreating does.
class TicketMutating extends TicketState {
  const TicketMutating(this.tickets, this.ticketId);

  final List<Ticket> tickets;
  final String ticketId;
}

class TicketFailure extends TicketState {
  const TicketFailure(this.message, [this.tickets = const []]);

  final String message;
  final List<Ticket> tickets;
}

class TicketBloc extends Bloc<TicketEvent, TicketState> {
  TicketBloc({required TicketRepository ticketRepository})
    : _ticketRepository = ticketRepository,
      super(const TicketInitial()) {
    on<TicketsRequested>(_onTicketsRequested);
    on<TicketCreateSubmitted>(_onTicketCreateSubmitted);
    on<TicketAssignRequested>(_onTicketAssignRequested);
    on<TicketStatusChangeRequested>(_onTicketStatusChangeRequested);
  }

  final TicketRepository _ticketRepository;
  TicketsRequested _lastRequest = const TicketsRequested();

  /// Re-fetches tickets using the last filter the user asked for — used by
  /// mutation handlers below and by external callers (e.g. a WebSocket
  /// push) that want an up-to-date list without resetting active filters.
  void refresh() => add(_lastRequest);

  Future<void> _onTicketsRequested(
    TicketsRequested event,
    Emitter<TicketState> emit,
  ) async {
    _lastRequest = event;
    emit(const TicketLoading());
    try {
      emit(TicketLoaded(await _fetch(event)));
    } catch (error) {
      emit(TicketFailure(error.toString()));
    }
  }

  Future<void> _onTicketCreateSubmitted(
    TicketCreateSubmitted event,
    Emitter<TicketState> emit,
  ) async {
    final previousTickets = _currentTickets();

    emit(TicketCreating(previousTickets));
    try {
      await _ticketRepository.createTicket(
        title: event.title,
        description: event.description,
      );
      emit(TicketLoaded(await _fetch(_lastRequest)));
    } catch (error) {
      emit(TicketFailure(error.toString(), previousTickets));
    }
  }

  Future<void> _onTicketAssignRequested(
    TicketAssignRequested event,
    Emitter<TicketState> emit,
  ) async {
    final previousTickets = _currentTickets();
    emit(TicketMutating(previousTickets, event.ticketId));
    try {
      await _ticketRepository.assignTicket(
        ticketId: event.ticketId,
        agentId: event.agentId,
      );
      emit(TicketLoaded(await _fetch(_lastRequest)));
    } catch (error) {
      emit(TicketFailure(error.toString(), previousTickets));
    }
  }

  Future<void> _onTicketStatusChangeRequested(
    TicketStatusChangeRequested event,
    Emitter<TicketState> emit,
  ) async {
    final previousTickets = _currentTickets();
    emit(TicketMutating(previousTickets, event.ticketId));
    try {
      await _ticketRepository.updateTicketStatus(
        ticketId: event.ticketId,
        status: event.status,
      );
      emit(TicketLoaded(await _fetch(_lastRequest)));
    } catch (error) {
      emit(TicketFailure(error.toString(), previousTickets));
    }
  }

  List<Ticket> _currentTickets() => switch (state) {
    TicketLoaded(:final tickets) => tickets,
    TicketCreating(:final tickets) => tickets,
    TicketMutating(:final tickets) => tickets,
    TicketFailure(:final tickets) => tickets,
    _ => <Ticket>[],
  };

  Future<List<Ticket>> _fetch(TicketsRequested r) => _ticketRepository.getTickets(
    page: r.page,
    limit: r.limit,
    scope: r.scope,
    status: r.status,
    priority: r.priority,
    department: r.department,
    search: r.search,
    overdue: r.overdue,
    from: r.from,
    to: r.to,
  );
}
