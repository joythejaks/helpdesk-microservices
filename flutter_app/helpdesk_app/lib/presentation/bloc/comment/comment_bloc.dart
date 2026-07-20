import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/data/ticket_repository.dart';
import 'package:helpdesk_app/models/ticket_comment.dart';

sealed class CommentEvent {
  const CommentEvent();
}

class CommentsRequested extends CommentEvent {
  const CommentsRequested(this.ticketId);

  final String ticketId;
}

class CommentSubmitted extends CommentEvent {
  const CommentSubmitted(this.ticketId, this.body, {this.isInternal = false});

  final String ticketId;
  final String body;
  final bool isInternal;
}

sealed class CommentState {
  const CommentState();
}

class CommentInitial extends CommentState {
  const CommentInitial();
}

class CommentLoading extends CommentState {
  const CommentLoading();
}

class CommentLoaded extends CommentState {
  const CommentLoaded(this.comments);

  final List<TicketComment> comments;
}

class CommentSubmitting extends CommentState {
  const CommentSubmitting(this.comments);

  final List<TicketComment> comments;
}

class CommentFailure extends CommentState {
  const CommentFailure(this.message, [this.comments = const []]);

  final String message;
  final List<TicketComment> comments;
}

class CommentBloc extends Bloc<CommentEvent, CommentState> {
  CommentBloc({required TicketRepository ticketRepository})
    : _ticketRepository = ticketRepository,
      super(const CommentInitial()) {
    on<CommentsRequested>(_onRequested);
    on<CommentSubmitted>(_onSubmitted);
  }

  final TicketRepository _ticketRepository;

  Future<void> _onRequested(
    CommentsRequested event,
    Emitter<CommentState> emit,
  ) async {
    emit(const CommentLoading());
    try {
      emit(CommentLoaded(await _ticketRepository.getComments(event.ticketId)));
    } catch (error) {
      emit(CommentFailure(error.toString()));
    }
  }

  Future<void> _onSubmitted(
    CommentSubmitted event,
    Emitter<CommentState> emit,
  ) async {
    final previous = switch (state) {
      CommentLoaded(:final comments) => comments,
      CommentSubmitting(:final comments) => comments,
      CommentFailure(:final comments) => comments,
      _ => <TicketComment>[],
    };
    emit(CommentSubmitting(previous));
    try {
      final comment = await _ticketRepository.addComment(
        ticketId: event.ticketId,
        body: event.body,
        isInternal: event.isInternal,
      );
      emit(CommentLoaded([...previous, comment]));
    } catch (error) {
      emit(CommentFailure(error.toString(), previous));
    }
  }
}
