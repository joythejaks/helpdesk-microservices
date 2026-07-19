import 'dart:async';

import 'package:flutter_bloc/flutter_bloc.dart';

import '../../../data/auth_repository.dart';
import '../../../models/app_user.dart';

sealed class AuthEvent {
  const AuthEvent();
}

class AuthStarted extends AuthEvent {
  const AuthStarted();
}

class AuthLoginSubmitted extends AuthEvent {
  const AuthLoginSubmitted({required this.email, required this.password});

  final String email;
  final String password;
}

class AuthRegisterSubmitted extends AuthEvent {
  const AuthRegisterSubmitted({required this.email, required this.password});

  final String email;
  final String password;
}

class AuthLogoutRequested extends AuthEvent {
  const AuthLogoutRequested();
}

sealed class AuthState {
  const AuthState();
}

class AuthInitial extends AuthState {
  const AuthInitial();
}

class AuthLoading extends AuthState {
  const AuthLoading();
}

class Authenticated extends AuthState {
  const Authenticated(this.user);

  final AppUser user;
}

class Unauthenticated extends AuthState {
  const Unauthenticated();
}

class AuthFailure extends AuthState {
  const AuthFailure(this.message);

  final String message;
}

class AuthBloc extends Bloc<AuthEvent, AuthState> {
  AuthBloc({required AuthRepository authRepository})
    : _authRepository = authRepository,
      super(const AuthInitial()) {
    on<AuthStarted>(_onStarted);
    on<AuthLoginSubmitted>(_onLoginSubmitted);
    on<AuthRegisterSubmitted>(_onRegisterSubmitted);
    on<AuthLogoutRequested>(_onLogoutRequested);
    _sessionExpiredSubscription = _authRepository.onSessionExpired.listen(
      (_) => add(const AuthLogoutRequested()),
    );
  }

  final AuthRepository _authRepository;
  late final StreamSubscription<void> _sessionExpiredSubscription;

  @override
  Future<void> close() {
    _sessionExpiredSubscription.cancel();
    return super.close();
  }

  Future<void> _onStarted(AuthStarted event, Emitter<AuthState> emit) async {
    emit(const AuthLoading());

    final hasSession = await _authRepository.hasSession();
    if (!hasSession) {
      emit(const Unauthenticated());
      return;
    }

    try {
      final user = await _authRepository.getCurrentUser();
      emit(Authenticated(user));
    } catch (_) {
      // Stored token is invalid/expired — clear it and send them to login.
      await _authRepository.logout();
      emit(const Unauthenticated());
    }
  }

  Future<void> _onLoginSubmitted(
    AuthLoginSubmitted event,
    Emitter<AuthState> emit,
  ) async {
    emit(const AuthLoading());
    try {
      await _authRepository.login(email: event.email, password: event.password);
      final user = await _authRepository.getCurrentUser();
      emit(Authenticated(user));
    } catch (error) {
      emit(AuthFailure(error.toString()));
      emit(const Unauthenticated());
    }
  }

  Future<void> _onRegisterSubmitted(
    AuthRegisterSubmitted event,
    Emitter<AuthState> emit,
  ) async {
    emit(const AuthLoading());
    try {
      await _authRepository.register(
        email: event.email,
        password: event.password,
      );
      await _authRepository.login(email: event.email, password: event.password);
      final user = await _authRepository.getCurrentUser();
      emit(Authenticated(user));
    } catch (error) {
      emit(AuthFailure(error.toString()));
      emit(const Unauthenticated());
    }
  }

  Future<void> _onLogoutRequested(
    AuthLogoutRequested event,
    Emitter<AuthState> emit,
  ) async {
    emit(const AuthLoading());
    try {
      await _authRepository.logout();
    } finally {
      emit(const Unauthenticated());
    }
  }
}
