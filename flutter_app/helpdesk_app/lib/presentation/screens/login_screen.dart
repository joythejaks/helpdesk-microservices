import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../../core/theme/helpdesk_theme.dart';
import '../bloc/auth/auth_bloc.dart';
import '../widgets/app_frame.dart';
import '../widgets/app_mark.dart';
import '../widgets/app_text_field.dart';
import '../widgets/gradient_button.dart';
import 'register_screen.dart';

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();

  @override
  void dispose() {
    _emailController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return BlocListener<AuthBloc, AuthState>(
      listener: (context, state) {
        if (state is AuthFailure) {
          ScaffoldMessenger.of(
            context,
          ).showSnackBar(SnackBar(content: Text(state.message)));
        }
      },
      child: AppFrame(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(24, 42, 20, 24),
          children: [
            const AppMark(),
            const SizedBox(height: 88),
            Text(
              'Helpdesk\nTicketing',
              style: Theme.of(context).textTheme.displayMedium,
            ),
            const SizedBox(height: 14),
            const Text(
              'Masuk untuk memantau tiket, membuat laporan baru, dan menjaga alur dukungan tetap rapi.',
              style: TextStyle(color: HelpdeskTheme.onVariant, height: 1.5),
            ),
            const SizedBox(height: 32),
            AppTextField(
              controller: _emailController,
              label: 'Email',
              icon: Icons.mail_outline,
              keyboardType: TextInputType.emailAddress,
            ),
            const SizedBox(height: 14),
            AppTextField(
              controller: _passwordController,
              label: 'Password',
              icon: Icons.lock_outline,
              obscureText: true,
            ),
            const SizedBox(height: 20),
            BlocBuilder<AuthBloc, AuthState>(
              builder: (context, state) {
                final isLoading = state is AuthLoading;
                return GradientButton(
                  label: isLoading ? 'Memproses...' : 'Masuk',
                  icon: Icons.arrow_forward,
                  onPressed: isLoading ? () {} : _submitLogin,
                );
              },
            ),
            const SizedBox(height: 14),
            Center(
              child: TextButton(
                onPressed: () => Navigator.of(context).push(
                  MaterialPageRoute(builder: (_) => const RegisterScreen()),
                ),
                child: const Text('Buat akun baru'),
              ),
            ),
          ],
        ),
      ),
    );
  }

  void _submitLogin() {
    context.read<AuthBloc>().add(
      AuthLoginSubmitted(
        email: _emailController.text.trim(),
        password: _passwordController.text,
      ),
    );
  }
}
