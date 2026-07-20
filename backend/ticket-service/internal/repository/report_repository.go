package repository

import (
	"fmt"
	"time"

	"ticket-service/internal/domain"

	"gorm.io/gorm"
)

var validGroupBy = map[string]bool{
	"day": true, "week": true, "month": true, "year": true,
}

type reportRepository struct {
	db *gorm.DB
}

func NewReportRepository(db *gorm.DB) domain.ReportRepository {
	return &reportRepository{db}
}

func (r *reportRepository) SummaryByPeriod(from, to time.Time, groupBy string) ([]domain.PeriodCount, error) {
	if !validGroupBy[groupBy] {
		groupBy = "day"
	}

	var rows []domain.PeriodCount
	err := r.db.Raw(fmt.Sprintf(`
		SELECT to_char(date_trunc('%s', created_at), 'YYYY-MM-DD') AS period,
		       status,
		       COUNT(*) AS count
		FROM tickets
		WHERE created_at BETWEEN ? AND ?
		GROUP BY period, status
		ORDER BY period
	`, groupBy), from, to).Scan(&rows).Error

	return rows, err
}

func (r *reportRepository) AgentPerformance(from, to time.Time) ([]domain.AgentStat, error) {
	var rows []domain.AgentStat
	err := r.db.Raw(`
		SELECT assigned_agent_id AS agent_id,
		       COUNT(*) AS total_assigned,
		       COUNT(*) FILTER (WHERE status IN ('resolved', 'closed')) AS total_resolved,
		       COALESCE(AVG(EXTRACT(EPOCH FROM (resolved_at - assigned_at)))
		                FILTER (WHERE resolved_at IS NOT NULL AND assigned_at IS NOT NULL), 0) AS avg_resolution_seconds
		FROM tickets
		WHERE assigned_agent_id IS NOT NULL
		  AND assigned_at BETWEEN ? AND ?
		GROUP BY assigned_agent_id
		ORDER BY total_assigned DESC
	`, from, to).Scan(&rows).Error

	return rows, err
}

func (r *reportRepository) HighPriorityTrend(since time.Time) (domain.CriticalTrend, error) {
	var count int64
	if err := r.db.Model(&domain.Ticket{}).
		Where("priority = ? AND created_at >= ?", "High", since).
		Count(&count).Error; err != nil {
		return domain.CriticalTrend{}, err
	}

	var tickets []domain.Ticket
	if err := r.db.
		Where("priority = ? AND created_at >= ?", "High", since).
		Order("created_at desc").
		Limit(10).
		Find(&tickets).Error; err != nil {
		return domain.CriticalTrend{}, err
	}

	rows := make([]domain.CriticalTicketRow, len(tickets))
	for i, t := range tickets {
		rows[i] = domain.CriticalTicketRow{ID: t.ID, Title: t.Title, CreatedAt: t.CreatedAt}
	}

	return domain.CriticalTrend{Count: int(count), Tickets: rows}, nil
}

func (r *reportRepository) QueueSize() (int64, error) {
	var count int64
	err := r.db.Model(&domain.Ticket{}).
		Where("assigned_agent_id IS NULL").
		Count(&count).Error
	return count, err
}
