package usecase_test

import (
	"testing"
	"time"

	"ticket-service/internal/domain"
	"ticket-service/internal/usecase"
)

type fakeReportRepository struct {
	highPriorityTrendSince time.Time
	criticalTrend          domain.CriticalTrend
	queueSize              int64
}

func (f *fakeReportRepository) SummaryByPeriod(from, to time.Time, groupBy string) ([]domain.PeriodCount, error) {
	return nil, nil
}

func (f *fakeReportRepository) AgentPerformance(from, to time.Time) ([]domain.AgentStat, error) {
	return nil, nil
}

func (f *fakeReportRepository) HighPriorityTrend(since time.Time) (domain.CriticalTrend, error) {
	f.highPriorityTrendSince = since
	return f.criticalTrend, nil
}

func (f *fakeReportRepository) QueueSize() (int64, error) {
	return f.queueSize, nil
}

func TestCriticalTrend_ComputesSinceFromWindowHours(t *testing.T) {
	repo := &fakeReportRepository{}
	uc := usecase.NewReportUsecase(repo)

	before := time.Now().Add(-24 * time.Hour)
	if _, err := uc.CriticalTrend(24); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	after := time.Now().Add(-24 * time.Hour)

	if repo.highPriorityTrendSince.Before(before.Add(-time.Second)) ||
		repo.highPriorityTrendSince.After(after.Add(time.Second)) {
		t.Errorf("expected since ~24h ago, got %v (window %v..%v)", repo.highPriorityTrendSince, before, after)
	}
}

func TestCriticalTrend_ReturnsRepoData(t *testing.T) {
	want := domain.CriticalTrend{
		Count: 3,
		Tickets: []domain.CriticalTicketRow{
			{ID: 1, Title: "Server down", CreatedAt: time.Now()},
		},
	}
	repo := &fakeReportRepository{criticalTrend: want}
	uc := usecase.NewReportUsecase(repo)

	got, err := uc.CriticalTrend(24)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Count != want.Count || len(got.Tickets) != len(want.Tickets) {
		t.Errorf("expected %+v, got %+v", want, got)
	}
}

func TestQueueSize_ReturnsRepoValue(t *testing.T) {
	repo := &fakeReportRepository{queueSize: 12}
	uc := usecase.NewReportUsecase(repo)

	size, err := uc.QueueSize()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if size != 12 {
		t.Errorf("expected 12, got %d", size)
	}
}
