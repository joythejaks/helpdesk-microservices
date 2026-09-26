package http

import (
	"strconv"
	"time"

	"ticket-service/internal/usecase"
	"ticket-service/pkg/response"

	"github.com/gin-gonic/gin"
)

type ReportHandler struct {
	usecase *usecase.ReportUsecase
}

func NewReportHandler(u *usecase.ReportUsecase) *ReportHandler {
	return &ReportHandler{usecase: u}
}

// parseReportRange reads ?from=&to= (YYYY-MM-DD), defaulting to the last 30
// days when omitted/unparseable.
func parseReportRange(c *gin.Context) (from, to time.Time) {
	to = time.Now()
	from = to.AddDate(0, 0, -30)

	if v := c.Query("from"); v != "" {
		if parsed, err := time.Parse("2006-01-02", v); err == nil {
			from = parsed
		}
	}
	if v := c.Query("to"); v != "" {
		if parsed, err := time.Parse("2006-01-02", v); err == nil {
			// include the whole "to" day
			to = parsed.Add(24*time.Hour - time.Second)
		}
	}
	return from, to
}

// @Summary Ticket summary per period
// @Description Admin only. Without `from`/`to` the last 30 days are used.
// @Tags Reports
// @Produce json
// @Security BearerAuth
// @Param from query string false "Start date (YYYY-MM-DD)"
// @Param to query string false "End date (YYYY-MM-DD)"
// @Param group_by query string false "Grouping" Enums(day, week, month, year) default(day)
// @Success 200 {object} response.Response{data=[]domain.PeriodCount}
// @Failure 403 {object} response.Response "forbidden"
// @Router /reports/summary [get]
func (h *ReportHandler) Summary(c *gin.Context) {
	if c.GetHeader("X-User-ROLE") != "admin" {
		response.Error(c, 403, "forbidden", "FORBIDDEN")
		return
	}

	from, to := parseReportRange(c)
	groupBy := c.DefaultQuery("group_by", "day")

	rows, err := h.usecase.Summary(from, to, groupBy)
	if err != nil {
		response.Error(c, 500, "failed to build report", "INTERNAL_ERROR")
		return
	}

	response.Success(c, rows)
}

// @Summary Agent performance
// @Description Admin only. Without `from`/`to` the last 30 days are used.
// @Tags Reports
// @Produce json
// @Security BearerAuth
// @Param from query string false "Start date (YYYY-MM-DD)"
// @Param to query string false "End date (YYYY-MM-DD)"
// @Success 200 {object} response.Response{data=[]domain.AgentStat}
// @Failure 403 {object} response.Response "forbidden"
// @Router /reports/agents [get]
func (h *ReportHandler) AgentPerformance(c *gin.Context) {
	if c.GetHeader("X-User-ROLE") != "admin" {
		response.Error(c, 403, "forbidden", "FORBIDDEN")
		return
	}

	from, to := parseReportRange(c)

	rows, err := h.usecase.AgentPerformance(from, to)
	if err != nil {
		response.Error(c, 500, "failed to build report", "INTERNAL_ERROR")
		return
	}

	response.Success(c, rows)
}

// @Summary High-priority ticket trend
// @Description Admin only.
// @Tags Reports
// @Produce json
// @Security BearerAuth
// @Param hours query int false "Time window in hours" default(24)
// @Success 200 {object} response.Response{data=domain.CriticalTrend}
// @Failure 403 {object} response.Response "forbidden"
// @Router /reports/critical-trends [get]
func (h *ReportHandler) CriticalTrend(c *gin.Context) {
	if c.GetHeader("X-User-ROLE") != "admin" {
		response.Error(c, 403, "forbidden", "FORBIDDEN")
		return
	}

	hours := 24
	if v := c.Query("hours"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			hours = parsed
		}
	}

	trend, err := h.usecase.CriticalTrend(hours)
	if err != nil {
		response.Error(c, 500, "failed to build report", "INTERNAL_ERROR")
		return
	}

	response.Success(c, trend)
}

// @Summary Number of unassigned tickets
// @Description Admin only. The data field holds `queue_size`.
// @Tags Reports
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Failure 403 {object} response.Response "forbidden"
// @Router /reports/queue-size [get]
func (h *ReportHandler) QueueSize(c *gin.Context) {
	if c.GetHeader("X-User-ROLE") != "admin" {
		response.Error(c, 403, "forbidden", "FORBIDDEN")
		return
	}

	size, err := h.usecase.QueueSize()
	if err != nil {
		response.Error(c, 500, "failed to build report", "INTERNAL_ERROR")
		return
	}

	response.Success(c, gin.H{"queue_size": size})
}
