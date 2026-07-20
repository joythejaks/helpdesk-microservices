import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/presentation/bloc/auth/auth_bloc.dart';
import 'package:helpdesk_app/presentation/widgets/app_frame.dart';
import 'package:helpdesk_app/presentation/widgets/app_text_field.dart';
import 'package:helpdesk_app/presentation/widgets/gradient_button.dart';
import 'package:helpdesk_app/presentation/widgets/header_bar.dart';

class RegisterScreen extends StatefulWidget {
  const RegisterScreen({super.key});

  @override
  State<RegisterScreen> createState() => _RegisterScreenState();
}

class _RegisterScreenState extends State<RegisterScreen> {
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
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('Registrasi berhasil')),
          );
          Navigator.of(context).pop();
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
            padding: const EdgeInsets.fromLTRB(24, 42, 20, 28),
            children: [
              HeaderBar(
                title: 'Register',
                subtitle: 'Siapkan akses helpdesk internal',
                leading: Icons.arrow_back,
                onLeadingTap: () => Navigator.of(context).pop(),
              ),
              const SizedBox(height: 36),
              AppTextField(
                controller: _emailController,
                label: 'Email kantor',
                icon: Icons.mail_outline,
                keyboardType: TextInputType.emailAddress,
                validator: (value) {
                  if (value == null || value.isEmpty) return 'Email tidak boleh kosong';
                  final emailRegex = RegExp(r'^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$');
                  if (!emailRegex.hasMatch(value)) return 'Format email tidak valid';
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
      ),
    );
  }
  void _submitRegister() {
    if (_formKey.currentState!.validate()) {
      context.read<AuthBloc>().add(
        AuthRegisterSubmitted(
          email: _emailController.text.trim(),
          password: _passwordController.text,
        ),
      );
    }
  }
}