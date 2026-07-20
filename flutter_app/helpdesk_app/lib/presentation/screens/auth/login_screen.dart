import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/core/theme/helpdesk_theme.dart';
import 'package:helpdesk_app/presentation/bloc/auth/auth_bloc.dart';
import 'package:helpdesk_app/presentation/widgets/app_frame.dart';
import 'package:helpdesk_app/presentation/widgets/app_mark.dart';
import 'package:helpdesk_app/presentation/widgets/app_text_field.dart';
import 'package:helpdesk_app/presentation/widgets/gradient_button.dart';

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  final _formKey = GlobalKey<FormState>();

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
          Navigator.of(context)
              .pushReplacementNamed('/home', arguments: state.user);
        }

        if (state is AuthFailure) {
          ScaffoldMessenger.of(
            context,
          ).showSnackBar(SnackBar(content: Text(state.message)));
        }
      },
      child: AppFrame(
        child: Form(
          key: _formKey,
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
                validator: (value) {
                  if (value == null || value.isEmpty) return 'Email tidak boleh kosong';
                  if (!value.contains('@')) return 'Format email tidak valid';
                  return null;
                },
              ),
              const SizedBox(height: 14),
              AppTextField(
                controller: _passwordController,
                label: 'Password',
                icon: Icons.lock_outline,
                obscureText: true,
                validator: (value) {
                  if (value == null || value.isEmpty) return 'Password tidak boleh kosong';
                  if (value.length < 8) return 'Password minimal 8 karakter';
                  if (value.length > 72) return 'Password maksimal 72 karakter';
                  return null;
                },
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
                  onPressed: () => Navigator.of(context).pushNamed('/register'),
                  child: const Text('Buat akun baru'),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  void _submitLogin() {
    if (!_formKey.currentState!.validate()) {
      return;
    }

    context.read<AuthBloc>().add(
      AuthLoginSubmitted(
        email: _emailController.text.trim(),
        password: _passwordController.text,
      ),
    );
  }
}
