import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:helpdesk_app/presentation/bloc/auth/auth_bloc.dart';
import 'package:helpdesk_app/presentation/screens/auth/login_screen.dart';

// Fake AuthBloc yang tidak butuh network/storage
class _FakeAuthBloc extends Fake implements AuthBloc {
  @override
  AuthState get state => const Unauthenticated();

  @override
  Stream<AuthState> get stream => const Stream.empty();

  @override
  void add(AuthEvent event) {}

  @override
  Future<void> close() async {}
}

void main() {
  testWidgets('LoginScreen menampilkan form email dan password', (tester) async {
    await tester.pumpWidget(
      BlocProvider<AuthBloc>(
        create: (_) => _FakeAuthBloc(),
        child: const MaterialApp(home: LoginScreen()),
      ),
    );

    expect(find.text('Helpdesk\nTicketing'), findsOneWidget);
    expect(find.text('Masuk'), findsOneWidget);
    expect(find.text('Buat akun baru'), findsOneWidget);
  });
}