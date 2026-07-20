import 'package:fl_chart/fl_chart.dart';
import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/core/theme/helpdesk_theme.dart';
import 'package:helpdesk_app/models/report_summary.dart';
import 'package:helpdesk_app/presentation/bloc/admin/admin_reports_bloc.dart';
import 'package:helpdesk_app/presentation/bloc/notification/notification_bloc.dart';
import 'package:helpdesk_app/presentation/widgets/detail_row.dart';
import 'package:helpdesk_app/presentation/widgets/header_bar.dart';
import 'package:helpdesk_app/presentation/widgets/metric_card.dart';
import 'package:helpdesk_app/presentation/widgets/surface_card.dart';

const _dayLabels = ['Min', 'Sen', 'Sel', 'Rab', 'Kam', 'Jum', 'Sab'];

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
          children: [
            ...totals.entries.map(
              (e) => MetricCard(
                value: '${e.value}',
                label: e.key,
                icon: Icons.confirmation_number_outlined,
              ),
            ),
            MetricCard(
              value: '${state.queueSize}',
              label: 'Queue Size',
              icon: Icons.inbox_outlined,
            ),
          ],
        ),
      const SizedBox(height: 24),
      Text('Team Performance', style: Theme.of(context).textTheme.titleMedium),
      const SizedBox(height: 4),
      Text('7 hari terakhir', style: Theme.of(context).textTheme.bodySmall),
      const SizedBox(height: 12),
      _buildTeamPerformanceChart(state.summary),
      const SizedBox(height: 24),
      _buildCriticalTrends(context, state.criticalTrend),
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

  Widget _buildTeamPerformanceChart(List<ReportSummaryRow> summary) {
    final today = DateTime.now();
    final days = List.generate(
      7,
      (i) => DateTime(today.year, today.month, today.day)
          .subtract(Duration(days: 6 - i)),
    );
    final totals = {for (final d in days) d: 0};
    for (final row in summary) {
      final date = DateTime.tryParse(row.period);
      if (date == null) continue;
      final key = DateTime(date.year, date.month, date.day);
      if (totals.containsKey(key)) {
        totals[key] = totals[key]! + row.count;
      }
    }

    final maxCount = totals.values.fold(0, (a, b) => a > b ? a : b);
    final maxY = maxCount == 0 ? 5.0 : (maxCount * 1.2);

    return SurfaceCard(
      child: SizedBox(
        height: 160,
        child: BarChart(
          BarChartData(
            alignment: BarChartAlignment.spaceAround,
            maxY: maxY,
            gridData: const FlGridData(show: false),
            borderData: FlBorderData(show: false),
            titlesData: FlTitlesData(
              leftTitles: const AxisTitles(
                sideTitles: SideTitles(showTitles: false),
              ),
              topTitles: const AxisTitles(
                sideTitles: SideTitles(showTitles: false),
              ),
              rightTitles: const AxisTitles(
                sideTitles: SideTitles(showTitles: false),
              ),
              bottomTitles: AxisTitles(
                sideTitles: SideTitles(
                  showTitles: true,
                  getTitlesWidget: (value, meta) {
                    final index = value.toInt();
                    if (index < 0 || index >= days.length) {
                      return const SizedBox.shrink();
                    }
                    return Padding(
                      padding: const EdgeInsets.only(top: 6),
                      child: Text(
                        _dayLabels[days[index].weekday % 7],
                        style: const TextStyle(fontSize: 11),
                      ),
                    );
                  },
                ),
              ),
            ),
            barGroups: [
              for (var i = 0; i < days.length; i++)
                BarChartGroupData(
                  x: i,
                  barRods: [
                    BarChartRodData(
                      toY: totals[days[i]]!.toDouble(),
                      color: HelpdeskTheme.primary,
                      width: 16,
                      borderRadius: BorderRadius.circular(4),
                    ),
                  ],
                ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildCriticalTrends(BuildContext context, CriticalTrend trend) {
    return SurfaceCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Icon(Icons.warning_amber_rounded, color: Colors.deepOrange),
              const SizedBox(width: 8),
              Text('Critical Ticket Trends', style: Theme.of(context).textTheme.titleMedium),
            ],
          ),
          const SizedBox(height: 8),
          Text(
            '${trend.count} tiket High-priority dalam 24 jam terakhir',
            style: const TextStyle(fontWeight: FontWeight.w700),
          ),
          const SizedBox(height: 12),
          if (trend.tickets.isEmpty)
            const Text('Tidak ada tiket High-priority baru.')
          else
            ...trend.tickets.take(5).map(
              (t) => Padding(
                padding: const EdgeInsets.only(bottom: 8),
                child: Row(
                  children: [
                    Expanded(
                      child: Text(t.title, overflow: TextOverflow.ellipsis),
                    ),
                    const SizedBox(width: 8),
                    Text(
                      _relativeTime(t.createdAt),
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                  ],
                ),
              ),
            ),
        ],
      ),
    );
  }

  String _relativeTime(DateTime value) {
    final diff = DateTime.now().difference(value.toLocal());
    if (diff.inMinutes < 1) return 'baru saja';
    if (diff.inMinutes < 60) return '${diff.inMinutes} menit lalu';
    if (diff.inHours < 24) return '${diff.inHours} jam lalu';
    return '${diff.inDays} hari lalu';
  }
}
