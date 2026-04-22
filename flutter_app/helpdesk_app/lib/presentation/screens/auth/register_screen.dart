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
  final _namaController = TextEditingController();
  final _departemenController = TextEditingController();

  @override
  void dispose() {
    _emailController.dispose();
    _passwordController.dispose();
    _namaController.dispose();
    _departemenController.dispose();
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
          Navigator.of(context).pop(); // Kembali ke halaman login
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
            AppTextField(
              controller: _namaController,
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
            AppTextField(
              controller: _departemenController,
              label: 'Departemen',
              icon: Icons.apartment,
            ),
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
    final email = _emailController.text.trim();
    final password = _passwordController.text;
    final nama = _namaController.text.trim();
    final departemen = _departemenController.text.trim();

    if (email.isEmpty || password.isEmpty || nama.isEmpty || departemen.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Semua field harus diisi')),
      );
      return;
    }

    context.read<AuthBloc>().add(
      AuthRegisterSubmitted(
        email: email,
        password: password,
        // Jika perlu, tambahkan nama dan departemen ke event
      ),
    );
  }
}
