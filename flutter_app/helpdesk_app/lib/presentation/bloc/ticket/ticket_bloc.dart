import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/data/ticket_repository.dart';
import 'package:helpdesk_app/models/ticket.dart';

sealed class TicketEvent {
  const TicketEvent();
}

class TicketsRequested extends TicketEvent {
  const TicketsRequested();
}

class TicketCreateSubmitted extends TicketEvent {
  const TicketCreateSubmitted({required this.title, required this.description});

  final String title;
  final String description;
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
  }

  final TicketRepository _ticketRepository;

  Future<void> _onTicketsRequested(
    TicketsRequested event,
    Emitter<TicketState> emit,
  ) async {
    emit(const TicketLoading());
    try {
      final tickets = await _ticketRepository.getTickets();
      emit(TicketLoaded(tickets));
    } catch (error) {
      emit(TicketFailure(error.toString()));
    }
  }

  Future<void> _onTicketCreateSubmitted(
    TicketCreateSubmitted event,
    Emitter<TicketState> emit,
  ) async {
    final previousTickets = switch (state) {
      TicketLoaded(:final tickets) => tickets,
      TicketCreating(:final tickets) => tickets,
      TicketFailure(:final tickets) => tickets,
      _ => <Ticket>[],
    };

    emit(TicketCreating(previousTickets));
    try {
      await _ticketRepository.createTicket(
        title: event.title,
        description: event.description,
      );
      final tickets = await _ticketRepository.getTickets();
      emit(TicketLoaded(tickets));
    } catch (error) {
      emit(TicketFailure(error.toString(), previousTickets));
    }
  }
}
