package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Log = logrus.New()

// Fields is an alias for logrus.Fields so handlers can use it directly
type Fields = logrus.Fields

func Init(service string) {
	Log.SetFormatter(&logrus.JSONFormatter{})
	Log.SetOutput(os.Stdout)
	Log.SetLevel(logrus.InfoLevel)

	Log.WithFields(logrus.Fields{
		"service": service,
		"version": "1.0.0",
	}).Info("logger initialized")
}

// WithTraceId adds trace_id to a log entry for tracking across services
func WithTraceId(traceId string) *logrus.Entry {
	return Log.WithField("trace_id", traceId)
}
