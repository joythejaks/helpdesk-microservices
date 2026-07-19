import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/data/admin_repository.dart';

sealed class StaffEvent {
  const StaffEvent();
}

class StaffCreateSubmitted extends StaffEvent {
  const StaffCreateSubmitted({
    required this.email,
    required this.password,
    required this.role,
  });

  final String email;
  final String password;
  final String role; // 'agent' | 'admin'
}

sealed class StaffState {
  const StaffState();
}

class StaffInitial extends StaffState {
  const StaffInitial();
}

class StaffSubmitting extends StaffState {
  const StaffSubmitting();
}

class StaffCreated extends StaffState {
  const StaffCreated();
}

class StaffFailure extends StaffState {
  const StaffFailure(this.message);

  final String message;
}

class StaffBloc extends Bloc<StaffEvent, StaffState> {
  StaffBloc({required AdminRepository adminRepository})
    : _adminRepository = adminRepository,
      super(const StaffInitial()) {
    on<StaffCreateSubmitted>(_onSubmitted);
  }

  final AdminRepository _adminRepository;

  Future<void> _onSubmitted(
    StaffCreateSubmitted event,
    Emitter<StaffState> emit,
  ) async {
    emit(const StaffSubmitting());
    try {
      await _adminRepository.createStaff(
        email: event.email,
        password: event.password,
        role: event.role,
      );
      emit(const StaffCreated());
    } catch (error) {
      emit(StaffFailure(error.toString()));
    }
  }
}
