import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'core/network/api_client.dart';
import 'core/network/notification_api_client.dart';
import 'core/services/env_config.dart';
import 'core/services/websocket_service.dart';
import 'core/storage/token_storage.dart';
import 'core/theme/helpdesk_theme.dart';
import 'data/admin_repository.dart';
import 'data/auth_repository.dart';
import 'data/notification_repository.dart';
import 'data/ticket_repository.dart';
import 'models/app_user.dart';
import 'models/ticket.dart';
import 'presentation/bloc/auth/auth_bloc.dart';
import 'presentation/bloc/notification/notification_bloc.dart';
import 'presentation/bloc/theme/theme_cubit.dart';
import 'presentation/bloc/ticket/ticket_bloc.dart';
import 'presentation/navigation/role_router.dart';
import 'presentation/screens/agent/ticket_detail_screen.dart';
import 'presentation/screens/attachment/attachments_screen.dart';
import 'presentation/screens/auth/login_screen.dart';
import 'presentation/screens/auth/register_screen.dart';
import 'presentation/screens/auth/splash_screen.dart';
import 'presentation/screens/notification/notifications_screen.dart';

void main() {
  runApp(const HelpdeskApp());
}

class HelpdeskApp extends StatelessWidget {
  const HelpdeskApp({super.key});

  static final GlobalKey<NavigatorState> navigatorKey =
      GlobalKey<NavigatorState>();

  @override
  Widget build(BuildContext context) {
    return MultiRepositoryProvider(
      providers: [
        RepositoryProvider(create: (_) => ApiClient()),
        RepositoryProvider(create: (_) => NotificationApiClient()),
        RepositoryProvider(create: (_) => TokenStorage()),
        RepositoryProvider(
          create: (_) => WebSocketService(url: EnvConfig.wsUrl),
        ),
        RepositoryProvider(
          create: (context) => AuthRepository(
            apiClient: context.read<ApiClient>(),
            tokenStorage: context.read<TokenStorage>(),
          ),
        ),
        RepositoryProvider(
          create: (context) => NotificationRepository(
            apiClient: context.read<NotificationApiClient>(),
            tokenStorage: context.read<TokenStorage>(),
            onUnauthorized: context.read<AuthRepository>().refreshTokens,
          ),
        ),
        RepositoryProvider(
          create: (context) => TicketRepository(
            apiClient: context.read<ApiClient>(),
            tokenStorage: context.read<TokenStorage>(),
          ),
        ),
        RepositoryProvider(
          create: (context) => AdminRepository(
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
          BlocProvider(
            create: (context) => NotificationBloc(
              webSocketService: context.read<WebSocketService>(),
              ticketBloc: context.read<TicketBloc>(),
            ),
          ),
          BlocProvider(create: (_) => ThemeCubit()),
        ],
        child: MultiBlocListener(
          listeners: [
            BlocListener<AuthBloc, AuthState>(
              listener: (context, state) async {
                final notifications = context.read<NotificationBloc>();
                if (state is Authenticated) {
                  final token = await context
                      .read<TokenStorage>()
                      .readAccessToken();
                  if (token != null && token.isNotEmpty) {
                    notifications.add(NotificationConnectRequested(token));
                  }
                  navigatorKey.currentState?.pushNamedAndRemoveUntil(
                    '/home',
                    (route) => false,
                    arguments: state.user,
                  );
                } else if (state is Unauthenticated) {
                  notifications.add(const NotificationDisconnectRequested());
                  navigatorKey.currentState?.pushNamedAndRemoveUntil(
                    '/login',
                    (route) => false,
                  );
                }
              },
            ),
          ],
          child: BlocBuilder<ThemeCubit, ThemeMode>(
            builder: (context, themeMode) => MaterialApp(
              navigatorKey: navigatorKey,
              title: 'Helpdesk Ticketing System',
              debugShowCheckedModeBanner: false,
              theme: HelpdeskTheme.light(),
              darkTheme: HelpdeskTheme.dark(),
              themeMode: themeMode,
              initialRoute: '/splash',
              routes: {
                '/splash': (context) => const SplashScreen(),
                '/login': (context) => const LoginScreen(),
                '/register': (context) => const RegisterScreen(),
                '/notifications': (context) => const NotificationsScreen(),
              },
              onGenerateRoute: (settings) {
                switch (settings.name) {
                  case '/ticket-detail':
                    final ticket = settings.arguments as Ticket;
                    return MaterialPageRoute(
                      builder: (_) => TicketDetailScreen(ticket: ticket),
                    );
                  case '/home':
                    final user = settings.arguments as AppUser;
                    return MaterialPageRoute(builder: (_) => homeForRole(user));
                  case '/ticket-attachments':
                    final ticketId = settings.arguments as String;
                    return MaterialPageRoute(
                      builder: (_) => AttachmentsScreen(ticketId: ticketId),
                    );
                }
                return null;
              },
            ),
          ),
        ),
      ),
    );
  }
}
