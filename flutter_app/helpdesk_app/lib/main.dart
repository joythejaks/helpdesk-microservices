import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'core/network/api_client.dart';
import 'core/storage/token_storage.dart';
import 'core/theme/helpdesk_theme.dart';
import 'data/auth_repository.dart';
import 'data/ticket_repository.dart';
import 'presentation/bloc/auth/auth_bloc.dart';
import 'presentation/bloc/ticket/ticket_bloc.dart';
import 'presentation/screens/user/dashboard_shell.dart';
import 'presentation/screens/auth/login_screen.dart';
import 'presentation/screens/auth/splash_screen.dart';

void main() {
  runApp(const HelpdeskApp());
}

class HelpdeskApp extends StatelessWidget {
  const HelpdeskApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MultiRepositoryProvider(
      providers: [
        RepositoryProvider(create: (_) => ApiClient()),
        RepositoryProvider(create: (_) => TokenStorage()),
        RepositoryProvider(
          create: (context) => AuthRepository(
            apiClient: context.read<ApiClient>(),
            tokenStorage: context.read<TokenStorage>(),
          ),
        ),
        RepositoryProvider(
          create: (context) => TicketRepository(
            apiClient: context.read<ApiClient>(),
            tokenStorage: context.read<TokenStorage>(),
          ),
        ),
      ],
      child: MultiBlocProvider(
        providers: [
          BlocProvider(
            create: (context) =>
                AuthBloc(authRepository: context.read<AuthRepository>())
                  ..add(const AuthStarted()),
          ),
          BlocProvider(
            create: (context) =>
                TicketBloc(ticketRepository: context.read<TicketRepository>()),
          ),
        ],
        child: MaterialApp(
          title: 'Helpdesk Ticketing System',
          debugShowCheckedModeBanner: false,
          theme: HelpdeskTheme.light(),
          initialRoute: '/splash',
          routes: {
            '/splash': (context) => const SplashScreen(),
            '/login': (context) => BlocBuilder<AuthBloc, AuthState>(
              builder: (context, state) {
                if (state is Authenticated) {
                  return const DashboardShell();
                }
                return const LoginScreen();
              },
            ),
            '/dashboard': (context) => const DashboardShell(),
          },
        ),
      ),
    );
  }
}
