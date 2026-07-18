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
