package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

// Logger is a structured logger interface
type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})

	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})

	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
	WithError(err error) Logger
}

// logrusLogger wraps logrus.Logger to implement our Logger interface
type logrusLogger struct {
	logger *logrus.Logger
	entry  *logrus.Entry
}

// New creates a new logger instance
func New(level string, environment string) Logger {
	log := logrus.New()

	// Set log level
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	log.SetLevel(logLevel)

	// Set formatter based on environment
	if environment == "production" {
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			ForceColors:     true,
		})
	}

	log.SetOutput(os.Stdout)

	return &logrusLogger{
		logger: log,
		entry:  log.WithFields(logrus.Fields{}),
	}
}

// Debug logs a debug message
func (l *logrusLogger) Debug(args ...interface{}) {
	l.entry.Debug(args...)
}

// Info logs an info message
func (l *logrusLogger) Info(args ...interface{}) {
	l.entry.Info(args...)
}

// Warn logs a warning message
func (l *logrusLogger) Warn(args ...interface{}) {
	l.entry.Warn(args...)
}

// Error logs an error message
func (l *logrusLogger) Error(args ...interface{}) {
	l.entry.Error(args...)
}

// Fatal logs a fatal message and exits
func (l *logrusLogger) Fatal(args ...interface{}) {
	l.entry.Fatal(args...)
}

// Debugf logs a formatted debug message
func (l *logrusLogger) Debugf(format string, args ...interface{}) {
	l.entry.Debugf(format, args...)
}

// Infof logs a formatted info message
func (l *logrusLogger) Infof(format string, args ...interface{}) {
	l.entry.Infof(format, args...)
}

// Warnf logs a formatted warning message
func (l *logrusLogger) Warnf(format string, args ...interface{}) {
	l.entry.Warnf(format, args...)
}

// Errorf logs a formatted error message
func (l *logrusLogger) Errorf(format string, args ...interface{}) {
	l.entry.Errorf(format, args...)
}

// Fatalf logs a formatted fatal message and exits
func (l *logrusLogger) Fatalf(format string, args ...interface{}) {
	l.entry.Fatalf(format, args...)
}

// WithField adds a single field to the logger
func (l *logrusLogger) WithField(key string, value interface{}) Logger {
	return &logrusLogger{
		logger: l.logger,
		entry:  l.entry.WithField(key, value),
	}
}

// WithFields adds multiple fields to the logger
func (l *logrusLogger) WithFields(fields map[string]interface{}) Logger {
	logrusFields := make(logrus.Fields)
	for k, v := range fields {
		logrusFields[k] = v
	}

	return &logrusLogger{
		logger: l.logger,
		entry:  l.entry.WithFields(logrusFields),
	}
}

// WithError adds an error field to the logger
func (l *logrusLogger) WithError(err error) Logger {
	return &logrusLogger{
		logger: l.logger,
		entry:  l.entry.WithError(err),
	}
}
