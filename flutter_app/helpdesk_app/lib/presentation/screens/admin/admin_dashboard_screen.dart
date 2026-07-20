import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/presentation/bloc/admin/admin_reports_bloc.dart';
import 'package:helpdesk_app/presentation/bloc/notification/notification_bloc.dart';
import 'package:helpdesk_app/presentation/widgets/detail_row.dart';
import 'package:helpdesk_app/presentation/widgets/header_bar.dart';
import 'package:helpdesk_app/presentation/widgets/metric_card.dart';
import 'package:helpdesk_app/presentation/widgets/surface_card.dart';

class AdminDashboardScreen extends StatelessWidget {
  const AdminDashboardScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return BlocBuilder<AdminReportsBloc, AdminReportsState>(
      builder: (context, state) {
        return RefreshIndicator(
          onRefresh: () async {
            context.read<AdminReportsBloc>().add(const AdminReportsRequested());
          },
          child: ListView(
            padding: const EdgeInsets.fromLTRB(24, 42, 20, 112),
            children: [
              BlocBuilder<NotificationBloc, NotificationState>(
                builder: (context, notifState) => HeaderBar(
                  title: 'Admin Reports',
                  subtitle: 'Ringkasan tiket & performa agent',
                  trailing: Icons.notifications_outlined,
                  trailingBadgeCount: notifState.unreadCount,
                  onTrailingTap: () {
                    context.read<NotificationBloc>().add(const NotificationCleared());
                    context.read<AdminReportsBloc>().add(const AdminReportsRequested());
                    Navigator.of(context).pushNamed('/notifications');
                  },
                ),
              ),
              const SizedBox(height: 22),
              if (state is AdminReportsLoading)
                const Center(child: CircularProgressIndicator())
              else if (state is AdminReportsFailure)
                Center(child: Text(state.message))
              else if (state is AdminReportsLoaded)
                ..._buildLoaded(context, state)
              else
                const SizedBox.shrink(),
            ],
          ),
        );
      },
    );
  }

  List<Widget> _buildLoaded(BuildContext context, AdminReportsLoaded state) {
    final totals = <String, int>{};
    for (final row in state.summary) {
      totals[row.status] = (totals[row.status] ?? 0) + row.count;
    }

    return [
      if (totals.isEmpty)
        const Text('Belum ada data tiket pada periode ini.')
      else
        GridView.count(
          crossAxisCount: 2,
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          crossAxisSpacing: 12,
          mainAxisSpacing: 12,
          childAspectRatio: 1.5,
          children: totals.entries
              .map(
                (e) => MetricCard(
                  value: '${e.value}',
                  label: e.key,
                  icon: Icons.confirmation_number_outlined,
                ),
              )
              .toList(),
        ),
      const SizedBox(height: 24),
      Text('Performa Agent', style: Theme.of(context).textTheme.titleMedium),
      const SizedBox(height: 12),
      if (state.agentReports.isEmpty)
        const Text('Belum ada agent dengan tiket yang ditugaskan.')
      else
        SurfaceCard(
          child: Column(
            children: state.agentReports
                .map(
                  (a) => Padding(
                    padding: const EdgeInsets.only(bottom: 8),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          'Agent #${a.agentId}',
                          style: const TextStyle(fontWeight: FontWeight.w800),
                        ),
                        DetailRow(label: 'Ditugaskan', value: '${a.totalAssigned}'),
                        DetailRow(label: 'Selesai', value: '${a.totalResolved}'),
                        DetailRow(
                          label: 'Rata-rata penyelesaian',
                          value: _formatDuration(a.avgResolutionSeconds),
                        ),
                        const Divider(),
                      ],
                    ),
                  ),
                )
                .toList(),
          ),
        ),
    ];
  }

  String _formatDuration(double seconds) {
    if (seconds <= 0) return '-';
    final hours = seconds ~/ 3600;
    final minutes = (seconds % 3600) ~/ 60;
    return '${hours}j ${minutes}m';
  }
}
