import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/core/theme/helpdesk_theme.dart';
import 'package:helpdesk_app/data/auth_repository.dart';
import 'package:helpdesk_app/models/app_user.dart';
import 'package:helpdesk_app/presentation/bloc/auth/auth_bloc.dart';
import 'package:helpdesk_app/presentation/bloc/profile/profile_bloc.dart';
import 'package:helpdesk_app/presentation/widgets/app_text_field.dart';
import 'package:helpdesk_app/presentation/widgets/header_bar.dart';
import 'package:helpdesk_app/presentation/widgets/surface_card.dart';

const _availabilityOptions = [
  ('available', 'Available'),
  ('busy', 'Busy'),
  ('offline', 'Offline'),
];

class ProfileScreen extends StatelessWidget {
  const ProfileScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return BlocProvider(
      create: (context) => ProfileBloc(authRepository: context.read<AuthRepository>()),
      child: const _ProfileView(),
    );
  }
}

class _ProfileView extends StatefulWidget {
  const _ProfileView();

  @override
  State<_ProfileView> createState() => _ProfileViewState();
}

class _ProfileViewState extends State<_ProfileView> {
  AppUser? _user;

  @override
  void initState() {
    super.initState();
    final authState = context.read<AuthBloc>().state;
    if (authState is Authenticated) _user = authState.user;
  }

  @override
  Widget build(BuildContext context) {
    final user = _user;
    return BlocListener<ProfileBloc, ProfileState>(
      listener: (context, state) {
        if (state is ProfileUpdated) {
          setState(() => _user = state.user);
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('Profil diperbarui')),
          );
        }
        if (state is PasswordChanged) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('Password berhasil diganti')),
          );
        }
        if (state is ProfileFailure) {
          ScaffoldMessenger.of(
            context,
          ).showSnackBar(SnackBar(content: Text(state.message)));
        }
      },
      child: ListView(
        padding: const EdgeInsets.fromLTRB(24, 42, 20, 112),
        children: [
          const HeaderBar(title: 'Profile', subtitle: 'Kelola akun & preferensi kamu'),
          const SizedBox(height: 22),
          if (user != null) _buildAccountCard(context, user),
          const SizedBox(height: 22),
          Text('Availability', style: Theme.of(context).textTheme.titleSmall),
          const SizedBox(height: 8),
          _buildAvailabilitySelector(context, user),
          const SizedBox(height: 24),
          Text('Account Settings', style: Theme.of(context).textTheme.titleSmall),
          const SizedBox(height: 8),
          SurfaceCard(
            child: Column(
              children: [
                _buildSettingsTile(
                  icon: Icons.person_outline,
                  label: 'Edit Profile',
                  onTap: user == null ? null : () => _showEditProfileDialog(context, user),
                ),
                const Divider(height: 1),
                _buildSettingsTile(
                  icon: Icons.lock_outline,
                  label: 'Change Password',
                  onTap: () => _showChangePasswordDialog(context),
                ),
                const Divider(height: 1),
                _buildSettingsTile(
                  icon: Icons.notifications_outlined,
                  label: 'Notification Preferences',
                  subtitle: 'Segera hadir',
                  onTap: null,
                ),
              ],
            ),
          ),
          const SizedBox(height: 24),
          OutlinedButton.icon(
            style: OutlinedButton.styleFrom(
              minimumSize: const Size.fromHeight(52),
              foregroundColor: Colors.red,
              side: const BorderSide(color: Colors.red),
            ),
            onPressed: () => context.read<AuthBloc>().add(const AuthLogoutRequested()),
            icon: const Icon(Icons.logout),
            label: const Text('Log Out'),
          ),
        ],
      ),
    );
  }

  Widget _buildAccountCard(BuildContext context, AppUser user) {
    return SurfaceCard(
      child: Column(
        children: [
          const CircleAvatar(
            radius: 36,
            backgroundColor: HelpdeskTheme.primaryContainer,
            child: Icon(Icons.person, size: 36, color: HelpdeskTheme.primary),
          ),
          const SizedBox(height: 14),
          Text(
            user.name.isNotEmpty ? user.name : user.email,
            style: Theme.of(context).textTheme.titleMedium,
          ),
          const SizedBox(height: 4),
          Text(user.email, style: Theme.of(context).textTheme.bodySmall),
          const SizedBox(height: 10),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
            decoration: BoxDecoration(
              color: HelpdeskTheme.primary.withValues(alpha: 25),
              borderRadius: BorderRadius.circular(20),
            ),
            child: Text(
              user.role[0].toUpperCase() + user.role.substring(1),
              style: const TextStyle(
                color: HelpdeskTheme.primary,
                fontWeight: FontWeight.bold,
                fontSize: 12,
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildAvailabilitySelector(BuildContext context, AppUser? user) {
    return BlocBuilder<ProfileBloc, ProfileState>(
      builder: (context, state) {
        final current = user?.availability ?? 'offline';
        return Wrap(
          spacing: 8,
          children: _availabilityOptions.map((opt) {
            final (value, label) = opt;
            return ChoiceChip(
              label: Text(label),
              selected: current == value,
              onSelected: (_) =>
                  context.read<ProfileBloc>().add(AvailabilityChangeSubmitted(value)),
            );
          }).toList(),
        );
      },
    );
  }

  Widget _buildSettingsTile({
    required IconData icon,
    required String label,
    String? subtitle,
    VoidCallback? onTap,
  }) {
    return ListTile(
      leading: Icon(icon, color: HelpdeskTheme.primary),
      title: Text(label),
      subtitle: subtitle != null ? Text(subtitle) : null,
      trailing: onTap != null ? const Icon(Icons.chevron_right) : null,
      enabled: onTap != null,
      onTap: onTap,
    );
  }

  void _showEditProfileDialog(BuildContext context, AppUser user) {
    final nameController = TextEditingController(text: user.name);
    final departmentController = TextEditingController(text: user.department);
    final formKey = GlobalKey<FormState>();
    final profileBloc = context.read<ProfileBloc>();

    showDialog<void>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: const Text('Edit Profile'),
        content: Form(
          key: formKey,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              AppTextField(
                controller: nameController,
                label: 'Nama',
                icon: Icons.person_outline,
                validator: (v) => (v == null || v.isEmpty) ? 'Nama tidak boleh kosong' : null,
              ),
              const SizedBox(height: 12),
              AppTextField(
                controller: departmentController,
                label: 'Departemen',
                icon: Icons.apartment_outlined,
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(),
            child: const Text('Batal'),
          ),
          FilledButton(
            onPressed: () {
              if (!formKey.currentState!.validate()) return;
              profileBloc.add(
                ProfileUpdateSubmitted(
                  name: nameController.text.trim(),
                  department: departmentController.text.trim(),
                ),
              );
              Navigator.of(dialogContext).pop();
            },
            child: const Text('Simpan'),
          ),
        ],
      ),
    );
  }

  void _showChangePasswordDialog(BuildContext context) {
    final oldController = TextEditingController();
    final newController = TextEditingController();
    final formKey = GlobalKey<FormState>();
    final profileBloc = context.read<ProfileBloc>();

    showDialog<void>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: const Text('Change Password'),
        content: Form(
          key: formKey,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              AppTextField(
                controller: oldController,
                label: 'Password lama',
                icon: Icons.lock_outline,
                obscureText: true,
                validator: (v) => (v == null || v.isEmpty) ? 'Wajib diisi' : null,
              ),
              const SizedBox(height: 12),
              AppTextField(
                controller: newController,
                label: 'Password baru',
                icon: Icons.lock_reset,
                obscureText: true,
                validator: (v) {
                  if (v == null || v.isEmpty) return 'Wajib diisi';
                  if (v.length < 8) return 'Minimal 8 karakter';
                  if (v.length > 72) return 'Maksimal 72 karakter';
                  return null;
                },
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(),
            child: const Text('Batal'),
          ),
          FilledButton(
            onPressed: () {
              if (!formKey.currentState!.validate()) return;
              profileBloc.add(
                PasswordChangeSubmitted(
                  oldPassword: oldController.text,
                  newPassword: newController.text,
                ),
              );
              Navigator.of(dialogContext).pop();
            },
            child: const Text('Simpan'),
          ),
        ],
      ),
    );
  }
}
