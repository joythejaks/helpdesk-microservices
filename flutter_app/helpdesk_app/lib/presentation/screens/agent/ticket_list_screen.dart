import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/models/ticket.dart';
import 'package:helpdesk_app/presentation/bloc/ticket/ticket_bloc.dart';
import 'package:helpdesk_app/presentation/widgets/app_text_field.dart';
import 'package:helpdesk_app/presentation/widgets/header_bar.dart';
import 'package:helpdesk_app/presentation/widgets/status_filter.dart';
import 'package:helpdesk_app/presentation/widgets/ticket_card.dart';

class TicketListScreen extends StatelessWidget {
  const TicketListScreen({super.key, required this.onOpenTicket});

  final ValueChanged<Ticket> onOpenTicket;

  @override
  Widget build(BuildContext context) {
    return BlocBuilder<TicketBloc, TicketState>(
      builder: (context, state) {
        final tickets = switch (state) {
          TicketLoaded(:final tickets) => tickets,
          TicketCreating(:final tickets) => tickets,
          TicketFailure(:final tickets) => tickets,
          _ => <Ticket>[],
        };

        return RefreshIndicator(
          onRefresh: () async {
            context.read<TicketBloc>().add(const TicketsRequested());
          },
          child: ListView(
            padding: const EdgeInsets.fromLTRB(24, 42, 20, 112),
            children: [
              const HeaderBar(
                title: 'Ticket List',
                subtitle: 'Semua permintaan dukungan',
                trailing: Icons.tune,
              ),
              const SizedBox(height: 22),
              const AppTextField(label: 'Cari tiket', icon: Icons.search),
              const SizedBox(height: 16),
              const StatusFilter(),
              const SizedBox(height: 20),
              if (state is TicketLoading)
                const Center(child: CircularProgressIndicator())
              else if (state is TicketFailure && tickets.isEmpty)
                Center(child: Text(state.message))
              else if (tickets.isEmpty)
                const Center(child: Text('Belum ada ticket.'))
              else
                ...tickets.map(
                  (ticket) => TicketCard(
                    ticket: ticket,
                    onTap: () => onOpenTicket(ticket),
                  ),
                ),
            ],
          ),
        );
      },
    );
  }
}
