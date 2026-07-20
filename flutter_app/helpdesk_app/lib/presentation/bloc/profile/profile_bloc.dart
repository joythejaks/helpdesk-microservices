import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/data/auth_repository.dart';
import 'package:helpdesk_app/models/app_user.dart';

sealed class ProfileEvent {
  const ProfileEvent();
}

class ProfileUpdateSubmitted extends ProfileEvent {
  const ProfileUpdateSubmitted({required this.name, required this.department});

  final String name;
  final String department;
}

class PasswordChangeSubmitted extends ProfileEvent {
  const PasswordChangeSubmitted({
    required this.oldPassword,
    required this.newPassword,
  });

  final String oldPassword;
  final String newPassword;
}

class AvailabilityChangeSubmitted extends ProfileEvent {
  const AvailabilityChangeSubmitted(this.availability);

  final String availability;
}

sealed class ProfileState {
  const ProfileState();
}

class ProfileInitial extends ProfileState {
  const ProfileInitial();
}

class ProfileSubmitting extends ProfileState {
  const ProfileSubmitting();
}

class ProfileUpdated extends ProfileState {
  const ProfileUpdated(this.user);

  final AppUser user;
}

class PasswordChanged extends ProfileState {
  const PasswordChanged();
}

class ProfileFailure extends ProfileState {
  const ProfileFailure(this.message);

  final String message;
}

class ProfileBloc extends Bloc<ProfileEvent, ProfileState> {
  ProfileBloc({required AuthRepository authRepository})
    : _authRepository = authRepository,
      super(const ProfileInitial()) {
    on<ProfileUpdateSubmitted>(_onProfileUpdateSubmitted);
    on<PasswordChangeSubmitted>(_onPasswordChangeSubmitted);
    on<AvailabilityChangeSubmitted>(_onAvailabilityChangeSubmitted);
  }

  final AuthRepository _authRepository;

  Future<void> _onProfileUpdateSubmitted(
    ProfileUpdateSubmitted event,
    Emitter<ProfileState> emit,
  ) async {
    emit(const ProfileSubmitting());
    try {
      final user = await _authRepository.updateProfile(
        name: event.name,
        department: event.department,
      );
      emit(ProfileUpdated(user));
    } catch (error) {
      emit(ProfileFailure(error.toString()));
    }
  }

  Future<void> _onPasswordChangeSubmitted(
    PasswordChangeSubmitted event,
    Emitter<ProfileState> emit,
  ) async {
    emit(const ProfileSubmitting());
    try {
      await _authRepository.changePassword(
        oldPassword: event.oldPassword,
        newPassword: event.newPassword,
      );
      emit(const PasswordChanged());
    } catch (error) {
      emit(ProfileFailure(error.toString()));
    }
  }

  Future<void> _onAvailabilityChangeSubmitted(
    AvailabilityChangeSubmitted event,
    Emitter<ProfileState> emit,
  ) async {
    emit(const ProfileSubmitting());
    try {
      final user = await _authRepository.updateAvailability(event.availability);
      emit(ProfileUpdated(user));
    } catch (error) {
      emit(ProfileFailure(error.toString()));
    }
  }
}
