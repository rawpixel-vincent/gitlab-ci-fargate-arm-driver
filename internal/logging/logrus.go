package logging

import (
	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

var formatters = map[string]func() Formatter{
	FormatText:       newTextFormatter,
	FormatTextSimple: newTextSimpleFormatter,
	FormatJSON:       newJSONFormatter,
}

type Fields = logrus.Fields
type Formatter = logrus.Formatter

func newLogrus() Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(newTextFormatter())
	logger.SetOutput(os.Stderr)

	return logrusWithLoggerAndEntry(logger, logger.WithField("PID", os.Getpid()))
}

type logrusLogger struct {
	*logrus.Entry

	logger *logrus.Logger
}

func logrusWithLoggerAndEntry(logger *logrus.Logger, entry *logrus.Entry) Logger {
	log := new(logrusLogger)
	log.logger = logger
	log.Entry = entry

	return log
}

func (l *logrusLogger) WithField(key string, value interface{}) Logger {
	return logrusWithLoggerAndEntry(l.logger, l.Entry.WithField(key, value))
}

func (l *logrusLogger) WithFields(fields Fields) Logger {
	return logrusWithLoggerAndEntry(l.logger, l.Entry.WithFields(fields))
}

func (l *logrusLogger) WithError(err error) Logger {
	return logrusWithLoggerAndEntry(l.logger, l.Entry.WithError(err))
}

func (l *logrusLogger) SetLevel(level string) error {
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("couldn't parse log level: %w", err)
	}

	l.logger.SetLevel(logLevel)

	return nil
}

func (l *logrusLogger) SetFormat(logFormat string) error {
	formatterFactory, ok := formatters[logFormat]
	if !ok {
		return fmt.Errorf("unsupported logging format %q", logFormat)
	}

	l.logger.SetFormatter(formatterFactory())

	return nil
}

func (l *logrusLogger) SetOutput(w io.Writer) {
	l.logger.SetOutput(w)
}

func newJSONFormatter() Formatter {
	return new(logrus.JSONFormatter)
}

func newTextFormatter() Formatter {
	formatter := new(logrus.TextFormatter)
	formatter.FullTimestamp = true
	formatter.ForceColors = true

	return formatter
}

func newTextSimpleFormatter() Formatter {
	return new(logrus.TextFormatter)
}
