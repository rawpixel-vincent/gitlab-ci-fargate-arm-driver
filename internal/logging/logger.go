package logging

import (
	"io"
)

const (
	FormatText       = "text"
	FormatTextSimple = "text-simple"
	FormatJSON       = "json"
)

type Logger interface {
	WithField(key string, value interface{}) Logger
	WithFields(fields Fields) Logger
	WithError(err error) Logger

	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Printf(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Warningf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Panicf(format string, args ...interface{})

	Debug(args ...interface{})
	Info(args ...interface{})
	Print(args ...interface{})
	Warn(args ...interface{})
	Warning(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})
	Panic(args ...interface{})

	SetLevel(level string) error
	SetFormat(logFormat string) error
	SetOutput(w io.Writer)
}

func New() Logger {
	return newLogrus()
}
