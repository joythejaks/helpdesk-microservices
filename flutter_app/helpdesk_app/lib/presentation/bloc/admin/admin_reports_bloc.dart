import 'package:flutter_bloc/flutter_bloc.dart';

import 'package:helpdesk_app/data/admin_repository.dart';
import 'package:helpdesk_app/models/report_summary.dart';

sealed class AdminReportsEvent {
  const AdminReportsEvent();
}

class AdminReportsRequested extends AdminReportsEvent {
  const AdminReportsRequested({this.from, this.to, this.groupBy = 'day'});

  final DateTime? from;
  final DateTime? to;
  final String groupBy;
}

sealed class AdminReportsState {
  const AdminReportsState();
}

class AdminReportsInitial extends AdminReportsState {
  const AdminReportsInitial();
}

class AdminReportsLoading extends AdminReportsState {
  const AdminReportsLoading();
}

class AdminReportsLoaded extends AdminReportsState {
  const AdminReportsLoaded({
    required this.summary,
    required this.agentReports,
    required this.criticalTrend,
    required this.queueSize,
  });

  final List<ReportSummaryRow> summary;
  final List<AgentReportRow> agentReports;
  final CriticalTrend criticalTrend;
  final int queueSize;
}

class AdminReportsFailure extends AdminReportsState {
  const AdminReportsFailure(this.message);

  final String message;
}

class AdminReportsBloc extends Bloc<AdminReportsEvent, AdminReportsState> {
  AdminReportsBloc({required AdminRepository adminRepository})
    : _adminRepository = adminRepository,
      super(const AdminReportsInitial()) {
    on<AdminReportsRequested>(_onRequested);
  }

  final AdminRepository _adminRepository;

  Future<void> _onRequested(
    AdminReportsRequested event,
    Emitter<AdminReportsState> emit,
  ) async {
    emit(const AdminReportsLoading());
    try {
      final results = await Future.wait([
        _adminRepository.getSummary(
          from: event.from,
          to: event.to,
          groupBy: event.groupBy,
        ),
        _adminRepository.getAgentReports(from: event.from, to: event.to),
        _adminRepository.getCriticalTrend(),
        _adminRepository.getQueueSize(),
      ]);
      emit(
        AdminReportsLoaded(
          summary: results[0] as List<ReportSummaryRow>,
          agentReports: results[1] as List<AgentReportRow>,
          criticalTrend: results[2] as CriticalTrend,
          queueSize: results[3] as int,
        ),
      );
    } catch (error) {
      emit(AdminReportsFailure(error.toString()));
    }
  }
}
