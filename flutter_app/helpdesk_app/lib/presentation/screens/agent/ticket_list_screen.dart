import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/models/ticket.dart';
import 'package:helpdesk_app/presentation/bloc/ticket/ticket_bloc.dart';
import 'package:helpdesk_app/presentation/widgets/app_text_field.dart';
import 'package:helpdesk_app/presentation/widgets/header_bar.dart';
import 'package:helpdesk_app/presentation/widgets/status_filter.dart';
import 'package:helpdesk_app/presentation/widgets/ticket_card.dart';

class TicketListScreen extends StatefulWidget {
  const TicketListScreen({super.key, required this.onOpenTicket});

  final ValueChanged<Ticket> onOpenTicket;

  @override
  State<TicketListScreen> createState() => _TicketListScreenState();
}

class _TicketListScreenState extends State<TicketListScreen> {
  final _searchController = TextEditingController();
  String _selectedFilter = 'All';
  String _searchQuery = '';

  @override
  void initState() {
    super.initState();
    _searchController.addListener(() {
      setState(() => _searchQuery = _searchController.text.toLowerCase());
    });
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  List<Ticket> _applyFilters(List<Ticket> tickets) {
    final query = _searchQuery.toLowerCase();
    return tickets.where((t) {
      final matchesFilter =
          _selectedFilter == 'All' || t.status == _selectedFilter;
      final matchesSearch =
          query.isEmpty ||
          t.title.toLowerCase().contains(query) ||
          t.id.toLowerCase().contains(query);
      return matchesFilter && matchesSearch;
    }).toList();
  }

  @override
  Widget build(BuildContext context) {
    return BlocBuilder<TicketBloc, TicketState>(
      builder: (context, state) {
        final allTickets = switch (state) {
          TicketLoaded(:final tickets) => tickets,
          TicketCreating(:final tickets) => tickets,
          TicketMutating(:final tickets) => tickets,
          TicketFailure(:final tickets) => tickets,
          _ => <Ticket>[],
        };

        final filtered = _applyFilters(allTickets);

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
              AppTextField(
                controller: _searchController,
                label: 'Cari tiket',
                icon: Icons.search,
              ),
              const SizedBox(height: 16),
              StatusFilter(
                selected: _selectedFilter,
                onSelected: (value) => setState(() => _selectedFilter = value),
              ),
              const SizedBox(height: 20),
              if (state is TicketLoading)
                const Center(child: CircularProgressIndicator())
              else if (state is TicketFailure && allTickets.isEmpty)
                Center(child: Text(state.message))
              else if (filtered.isEmpty)
                const Center(child: Text('Tidak ada tiket yang cocok.'))
              else
                ...filtered.map(
                  (ticket) => TicketCard(
                    ticket: ticket,
                    onTap: () => widget.onOpenTicket(ticket),
                  ),
                ),
            ],
          ),
        );
      },
    );
  }
}
