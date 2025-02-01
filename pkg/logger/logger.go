package logger

import (
	"os"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
)

// Globaler Logger
var Log = logrus.New()

type CustomFormatter struct {
	logrus.TextFormatter
}

func WithPackage(p string) *logrus.Entry {
	return Log.WithFields(logrus.Fields{"package": p})
}

func init() {
	Log.SetLevel(logrus.DebugLevel)
	Log.SetFormatter(&nested.Formatter{
		HideKeys:        true,
		TrimMessages:    true,
		CallerFirst:     true,
		TimestampFormat: "15:04:05.000",
	})
	Log.SetOutput(os.Stdout)
}
