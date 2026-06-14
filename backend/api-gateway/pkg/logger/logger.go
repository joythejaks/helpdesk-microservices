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

	Log.WithField("service", service).Info("logger initialized")
}
