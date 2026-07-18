package http

import (
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
