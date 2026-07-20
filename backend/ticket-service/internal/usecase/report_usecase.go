package usecase

import (
	"time"

	"ticket-service/internal/domain"
)

type ReportUsecase struct {
	repo domain.ReportRepository
}

func NewReportUsecase(r domain.ReportRepository) *ReportUsecase {
	return &ReportUsecase{repo: r}
}

func (u *ReportUsecase) Summary(from, to time.Time, groupBy string) ([]domain.PeriodCount, error) {
	return u.repo.SummaryByPeriod(from, to, groupBy)
}

func (u *ReportUsecase) AgentPerformance(from, to time.Time) ([]domain.AgentStat, error) {
	return u.repo.AgentPerformance(from, to)
}

// CriticalTrend reports High-priority tickets created in the last
// windowHours.
func (u *ReportUsecase) CriticalTrend(windowHours int) (domain.CriticalTrend, error) {
	since := time.Now().Add(-time.Duration(windowHours) * time.Hour)
	return u.repo.HighPriorityTrend(since)
}

func (u *ReportUsecase) QueueSize() (int64, error) {
	return u.repo.QueueSize()
}
