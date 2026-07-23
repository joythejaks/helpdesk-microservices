// Package metrics wires up basic Prometheus request metrics for this
// service. Deliberately minimal — request rate/latency/error-rate per
// route, not full distributed tracing (see backend/BACKLOG.md's
// "Observability" item for why this scope was chosen).
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	requestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests processed, labeled by method/route/status.",
		},
		[]string{"method", "route", "status"},
	)

	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds, labeled by method/route.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Wrap records request count/duration for a handler under a caller-given
// route label (the stdlib mux doesn't expose the matched pattern back on
// the request, so callers pass the same string used to register the
// route — e.g. "GET /notifications") instead of the raw URL, so
// /notifications/123/read and /notifications/456/read share one label
// instead of exploding cardinality per ID.
func Wrap(route string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next(rec, r)

		status := strconv.Itoa(rec.status)
		requestsTotal.WithLabelValues(r.Method, route, status).Inc()
		requestDuration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
	}
}
