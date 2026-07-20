package domain

import "time"

// PeriodCount is one bucket of the admin summary report — ticket counts for
// a single period/status combination.
type PeriodCount struct {
	Period string `json:"period"`
	Status string `json:"status"`
	Count  int    `json:"count"`
}

// AgentStat is one agent's throughput/performance row in the agent
// performance report.
type AgentStat struct {
	AgentID               uint    `json:"agent_id"`
	TotalAssigned         int     `json:"total_assigned"`
	TotalResolved         int     `json:"total_resolved"`
	AvgResolutionSeconds  float64 `json:"avg_resolution_seconds"`
}

// CriticalTrend summarizes recent High-priority ticket volume — Count is
// the full total within the window, Tickets a capped preview list.
type CriticalTrend struct {
	Count   int                 `json:"count"`
	Tickets []CriticalTicketRow `json:"tickets"`
}

type CriticalTicketRow struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type ReportRepository interface {
	// SummaryByPeriod buckets ticket counts by created_at truncated to
	// groupBy ("day", "week", "month", or "year") and status, within
	// [from, to].
	SummaryByPeriod(from, to time.Time, groupBy string) ([]PeriodCount, error)

	// AgentPerformance aggregates per-agent throughput and average
	// resolution time for tickets assigned within [from, to].
	AgentPerformance(from, to time.Time) ([]AgentStat, error)

	// HighPriorityTrend counts High-priority tickets created since the
	// given time, plus a capped preview list for display.
	HighPriorityTrend(since time.Time) (CriticalTrend, error)

	// QueueSize counts tickets with no assigned agent — the same
	// predicate the agent-facing unassigned queue already uses.
	QueueSize() (int64, error)
}
