package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Log = logrus.New()

// Fields adalah alias untuk logrus.Fields agar bisa digunakan di handler
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

// WithTraceId menambahkan trace_id ke dalam log untuk tracking antar servis
func WithTraceId(traceId string) *logrus.Entry {
	return Log.WithField("trace_id", traceId)
}
