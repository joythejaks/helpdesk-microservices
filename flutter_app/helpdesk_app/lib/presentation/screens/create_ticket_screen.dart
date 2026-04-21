import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import '../../core/theme/helpdesk_theme.dart';
import '../bloc/ticket/ticket_bloc.dart';
import '../widgets/app_text_field.dart';
import '../widgets/gradient_button.dart';
import '../widgets/header_bar.dart';
import '../widgets/surface_card.dart';

class CreateTicketScreen extends StatefulWidget {
  const CreateTicketScreen({super.key});

  @override
  State<CreateTicketScreen> createState() => _CreateTicketScreenState();
}

class _CreateTicketScreenState extends State<CreateTicketScreen> {
  final _titleController = TextEditingController();
  final _descriptionController = TextEditingController();

  @override
  void dispose() {
    _titleController.dispose();
    _descriptionController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return BlocListener<TicketBloc, TicketState>(
      listener: (context, state) {
        if (state is TicketLoaded) {
          _titleController.clear();
          _descriptionController.clear();
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('Ticket berhasil dibuat')),
          );
        }

        if (state is TicketFailure) {
          ScaffoldMessenger.of(
            context,
          ).showSnackBar(SnackBar(content: Text(state.message)));
        }
      },
      child: ListView(
        padding: const EdgeInsets.fromLTRB(24, 42, 20, 112),
        children: [
          const HeaderBar(
            title: 'Create Ticket',
            subtitle: 'Laporkan kendala dengan konteks lengkap',
            trailing: Icons.close,
          ),
          const SizedBox(height: 28),
          AppTextField(
            controller: _titleController,
            label: 'Judul kendala',
            icon: Icons.subject,
          ),
          const SizedBox(height: 14),
          const AppTextField(label: 'Kategori', icon: Icons.category_outlined),
          const SizedBox(height: 14),
          const AppTextField(label: 'Prioritas', icon: Icons.flag_outlined),
          const SizedBox(height: 14),
          AppTextField(
            controller: _descriptionController,
            label: 'Deskripsi',
            icon: Icons.notes,
            maxLines: 5,
          ),
          const SizedBox(height: 18),
          const SurfaceCard(
            child: Row(
              children: [
                Icon(Icons.attach_file, color: HelpdeskTheme.primary),
                SizedBox(width: 12),
                Expanded(
                  child: Text(
                    'Lampirkan screenshot atau dokumen pendukung',
                    style: TextStyle(fontWeight: FontWeight.w700),
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 22),
          BlocBuilder<TicketBloc, TicketState>(
            builder: (context, state) {
              final isCreating = state is TicketCreating;
              return GradientButton(
                label: isCreating ? 'Mengirim...' : 'Kirim Ticket',
                icon: Icons.send_outlined,
                onPressed: isCreating ? () {} : _submitTicket,
              );
            },
          ),
        ],
      ),
    );
  }

  void _submitTicket() {
    context.read<TicketBloc>().add(
      TicketCreateSubmitted(
        title: _titleController.text.trim(),
        description: _descriptionController.text.trim(),
      ),
    );
  }
}
