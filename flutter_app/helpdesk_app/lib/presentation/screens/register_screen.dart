import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../bloc/auth/auth_bloc.dart';
import '../widgets/app_frame.dart';
import '../widgets/app_text_field.dart';
import '../widgets/gradient_button.dart';
import '../widgets/header_bar.dart';
import 'dashboard_shell.dart';

class RegisterScreen extends StatefulWidget {
  const RegisterScreen({super.key});

  @override
  State<RegisterScreen> createState() => _RegisterScreenState();
}

class _RegisterScreenState extends State<RegisterScreen> {
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
        if (state is Authenticated) {
          Navigator.of(context).pushAndRemoveUntil(
            MaterialPageRoute(builder: (_) => const DashboardShell()),
            (route) => false,
          );
        }

        if (state is AuthFailure) {
          ScaffoldMessenger.of(
            context,
          ).showSnackBar(SnackBar(content: Text(state.message)));
        }
      },
      child: AppFrame(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(24, 42, 20, 28),
          children: [
            HeaderBar(
              title: 'Register',
              subtitle: 'Siapkan akses helpdesk internal',
              leading: Icons.arrow_back,
              onLeadingTap: () => Navigator.of(context).pop(),
            ),
            const SizedBox(height: 36),
            const AppTextField(
              label: 'Nama lengkap',
              icon: Icons.person_outline,
            ),
            const SizedBox(height: 14),
            AppTextField(
              controller: _emailController,
              label: 'Email kantor',
              icon: Icons.mail_outline,
              keyboardType: TextInputType.emailAddress,
            ),
            const SizedBox(height: 14),
            const AppTextField(label: 'Departemen', icon: Icons.apartment),
            const SizedBox(height: 14),
            AppTextField(
              controller: _passwordController,
              label: 'Password',
              icon: Icons.lock_outline,
              obscureText: true,
            ),
            const SizedBox(height: 22),
            BlocBuilder<AuthBloc, AuthState>(
              builder: (context, state) {
                final isLoading = state is AuthLoading;
                return GradientButton(
                  label: isLoading ? 'Memproses...' : 'Daftar',
                  icon: Icons.check,
                  onPressed: isLoading ? () {} : _submitRegister,
                );
              },
            ),
          ],
        ),
      ),
    );
  }

  void _submitRegister() {
    context.read<AuthBloc>().add(
      AuthRegisterSubmitted(
        email: _emailController.text.trim(),
        password: _passwordController.text,
      ),
    );
  }
}
