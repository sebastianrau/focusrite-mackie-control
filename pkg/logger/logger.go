package logger

import (
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// Register packages here, default debug Level should not be used
var packageLogLevels = map[string]logrus.Level{
	"main":   logrus.WarnLevel,
	"config": logrus.WarnLevel,

	"focusriteclient": logrus.WarnLevel,
	"fc-audio":        logrus.WarnLevel,

	"focusrite-xml":    logrus.WarnLevel,
	"focusrite-config": logrus.WarnLevel,

	"gui-main": logrus.WarnLevel,

	"mcu":           logrus.WarnLevel,
	"mcu-connector": logrus.WarnLevel,

	"monitor-controller": logrus.WarnLevel,
	"controller-config":  logrus.WarnLevel,

	"default": logrus.WarnLevel,
}

type CustomLogger struct {
	pkg    string
	level  logrus.Level
	logger *logrus.Entry
}

func init() {
	log.SetLevel(logrus.DebugLevel)

	log.SetFormatter(&nested.Formatter{
		HideKeys:        true,
		TrimMessages:    true,
		CallerFirst:     true,
		TimestampFormat: "15:04:05.000",
	})
}

// NewLogger gibt einen neuen CustomLogger für ein Package zurück
func WithPackage(pkg string) *CustomLogger {
	l := log.WithField("package", pkg)

	logLevel := packageLogLevels["default"]
	if level, exists := packageLogLevels[pkg]; exists {
		logLevel = level
	}

	return &CustomLogger{
		pkg:    pkg,
		level:  logLevel,
		logger: l,
	}
}

// getLogLevel prüft das konfigurierte Log-Level für das aktuelle Package
func (l *CustomLogger) getLogLevel() logrus.Level {
	return l.level
}

// ------------------------- print methoden ----------------------

// Logging-Methoden
func (l *CustomLogger) Debugf(format string, args ...interface{}) {
	if logrus.DebugLevel <= l.getLogLevel() {
		l.logger.Debugf(format, args...)
	}
}

func (l *CustomLogger) Debugln(args ...interface{}) {
	if logrus.DebugLevel <= l.getLogLevel() {
		l.logger.Debugln(args...)
	}
}

func (l *CustomLogger) Debug(args ...interface{}) {
	if logrus.DebugLevel <= l.getLogLevel() {
		l.logger.Debug(args...)
	}
}

func (l *CustomLogger) Infof(format string, args ...interface{}) {
	if logrus.InfoLevel <= l.getLogLevel() {
		l.logger.Infof(format, args...)
	}
}

func (l *CustomLogger) Info(args ...interface{}) {
	if logrus.InfoLevel <= l.getLogLevel() {
		l.logger.Info(args...)
	}
}

func (l *CustomLogger) Warningf(format string, args ...interface{}) {
	if logrus.WarnLevel <= l.getLogLevel() {
		l.logger.Warningf(format, args...)
	}
}

func (l *CustomLogger) Warn(args ...interface{}) {
	if logrus.WarnLevel <= l.getLogLevel() {
		l.logger.Warn(args...)
	}
}

func (l *CustomLogger) Warnf(format string, args ...interface{}) {
	if logrus.WarnLevel <= l.getLogLevel() {
		l.logger.Warnf(format, args...)
	}
}

func (l *CustomLogger) Errorf(format string, args ...interface{}) {
	if logrus.ErrorLevel <= l.getLogLevel() {
		l.logger.Errorf(format, args...)
	}
}

func (l *CustomLogger) Error(args ...interface{}) {
	if logrus.ErrorLevel <= l.getLogLevel() {
		l.logger.Error(args...)
	}
}

func (l *CustomLogger) Errorln(args ...interface{}) {
	if logrus.ErrorLevel <= l.getLogLevel() {
		l.logger.Errorln(args...)
	}
}

func (l *CustomLogger) Fatalf(format string, args ...interface{}) {
	if logrus.FatalLevel <= l.getLogLevel() {
		l.logger.Fatalf(format, args...)
	}
}

func (l *CustomLogger) Fatal(args ...interface{}) {
	if logrus.FatalLevel <= l.getLogLevel() {
		l.logger.Fatal(args...)
	}
}

func (l *CustomLogger) Panicf(format string, args ...interface{}) {
	if logrus.PanicLevel <= l.getLogLevel() {
		l.logger.Panicf(format, args...)
	}
}

func (l *CustomLogger) Panic(args ...interface{}) {
	if logrus.PanicLevel <= l.getLogLevel() {
		l.logger.Panic(args...)
	}
}
