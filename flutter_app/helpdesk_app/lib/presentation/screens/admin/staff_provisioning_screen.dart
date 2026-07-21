import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/data/admin_repository.dart';
import 'package:helpdesk_app/presentation/bloc/admin/staff_bloc.dart';
import 'package:helpdesk_app/presentation/widgets/app_text_field.dart';
import 'package:helpdesk_app/presentation/widgets/gradient_button.dart';
import 'package:helpdesk_app/presentation/widgets/header_bar.dart';

class StaffProvisioningScreen extends StatefulWidget {
  const StaffProvisioningScreen({super.key});

  @override
  State<StaffProvisioningScreen> createState() =>
      _StaffProvisioningScreenState();
}

class _StaffProvisioningScreenState extends State<StaffProvisioningScreen> {
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  final _formKey = GlobalKey<FormState>();
  String _role = 'agent';

  @override
  void dispose() {
    _emailController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return BlocProvider(
      create: (context) =>
          StaffBloc(adminRepository: context.read<AdminRepository>()),
      child: BlocListener<StaffBloc, StaffState>(
        listener: (context, state) {
          if (state is StaffCreated) {
            _emailController.clear();
            _passwordController.clear();
            ScaffoldMessenger.of(context).showSnackBar(
              const SnackBar(content: Text('Akun staff berhasil dibuat')),
            );
          }
          if (state is StaffFailure) {
            ScaffoldMessenger.of(
              context,
            ).showSnackBar(SnackBar(content: Text(state.message)));
          }
        },
        child: Form(
          key: _formKey,
          child: ListView(
            padding: const EdgeInsets.fromLTRB(24, 42, 20, 112),
            children: [
              const HeaderBar(
                title: 'Staff Provisioning',
                subtitle: 'Buat akun agent atau admin baru',
              ),
              const SizedBox(height: 28),
              AppTextField(
                controller: _emailController,
                label: 'Email',
                icon: Icons.mail_outline,
                keyboardType: TextInputType.emailAddress,
                validator: (value) {
                  if (value == null || value.isEmpty)
                    return 'Email tidak boleh kosong';
                  final emailRegex = RegExp(
                    r'^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$',
                  );
                  if (!emailRegex.hasMatch(value))
                    return 'Format email tidak valid';
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
                  if (value == null || value.isEmpty)
                    return 'Password tidak boleh kosong';
                  if (value.length < 8) return 'Password minimal 8 karakter';
                  if (value.length > 72) return 'Password maksimal 72 karakter';
                  return null;
                },
              ),
              const SizedBox(height: 14),
              Row(
                children: [
                  ChoiceChip(
                    label: const Text('Agent'),
                    selected: _role == 'agent',
                    onSelected: (_) => setState(() => _role = 'agent'),
                  ),
                  const SizedBox(width: 8),
                  ChoiceChip(
                    label: const Text('Admin'),
                    selected: _role == 'admin',
                    onSelected: (_) => setState(() => _role = 'admin'),
                  ),
                ],
              ),
              const SizedBox(height: 22),
              BlocBuilder<StaffBloc, StaffState>(
                builder: (context, state) {
                  final isSubmitting = state is StaffSubmitting;
                  return GradientButton(
                    label: isSubmitting ? 'Membuat akun...' : 'Buat Akun Staff',
                    icon: Icons.person_add_alt_1,
                    onPressed: isSubmitting ? () {} : () => _submit(context),
                  );
                },
              ),
            ],
          ),
        ),
      ),
    );
  }

  void _submit(BuildContext context) {
    if (!_formKey.currentState!.validate()) return;
    context.read<StaffBloc>().add(
      StaffCreateSubmitted(
        email: _emailController.text.trim(),
        password: _passwordController.text,
        role: _role,
      ),
    );
  }
}
