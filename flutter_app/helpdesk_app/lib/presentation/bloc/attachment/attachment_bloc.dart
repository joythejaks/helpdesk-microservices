import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/data/ticket_repository.dart';
import 'package:helpdesk_app/models/ticket_attachment.dart';

sealed class AttachmentEvent {
  const AttachmentEvent();
}

class AttachmentsRequested extends AttachmentEvent {
  const AttachmentsRequested(this.ticketId);

  final String ticketId;
}

class AttachmentUploadRequested extends AttachmentEvent {
  const AttachmentUploadRequested(this.ticketId, this.filename, this.bytes);

  final String ticketId;
  final String filename;
  final List<int> bytes;
}

sealed class AttachmentState {
  const AttachmentState();
}

class AttachmentInitial extends AttachmentState {
  const AttachmentInitial();
}

class AttachmentLoading extends AttachmentState {
  const AttachmentLoading();
}

class AttachmentLoaded extends AttachmentState {
  const AttachmentLoaded(this.attachments);

  final List<TicketAttachment> attachments;
}

class AttachmentUploading extends AttachmentState {
  const AttachmentUploading(this.attachments);

  final List<TicketAttachment> attachments;
}

class AttachmentFailure extends AttachmentState {
  const AttachmentFailure(this.message, [this.attachments = const []]);

  final String message;
  final List<TicketAttachment> attachments;
}

class AttachmentBloc extends Bloc<AttachmentEvent, AttachmentState> {
  AttachmentBloc({required TicketRepository ticketRepository})
    : _ticketRepository = ticketRepository,
      super(const AttachmentInitial()) {
    on<AttachmentsRequested>(_onRequested);
    on<AttachmentUploadRequested>(_onUploadRequested);
  }

  final TicketRepository _ticketRepository;

  Future<void> _onRequested(
    AttachmentsRequested event,
    Emitter<AttachmentState> emit,
  ) async {
    emit(const AttachmentLoading());
    try {
      emit(AttachmentLoaded(await _ticketRepository.getAttachments(event.ticketId)));
    } catch (error) {
      emit(AttachmentFailure(error.toString()));
    }
  }

  Future<void> _onUploadRequested(
    AttachmentUploadRequested event,
    Emitter<AttachmentState> emit,
  ) async {
    final previous = switch (state) {
      AttachmentLoaded(:final attachments) => attachments,
      AttachmentUploading(:final attachments) => attachments,
      AttachmentFailure(:final attachments) => attachments,
      _ => <TicketAttachment>[],
    };
    emit(AttachmentUploading(previous));
    try {
      final attachment = await _ticketRepository.uploadAttachment(
        ticketId: event.ticketId,
        filename: event.filename,
        bytes: event.bytes,
      );
      emit(AttachmentLoaded([...previous, attachment]));
    } catch (error) {
      emit(AttachmentFailure(error.toString(), previous));
    }
  }
}
