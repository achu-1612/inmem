package log

import (
	"os"

	log "github.com/sirupsen/logrus"
)

// Logger is an interface that defines the logging methods used in the in-memory database package.
// It provides a consistent way to log messages at different levels (Info, Debug, Warn, Error) with optional formatting.
type Logger interface {
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
}

// make sure stdLogger implements the Logger interface.
var _ Logger = (*stdLogger)(nil)

type stdLogger struct {
	std  *log.Logger
	name string
}

func (l *stdLogger) Info(args ...interface{}) {
	l.std.WithField("inmem ", l.name).Info(args...)
}

func (l *stdLogger) Infof(format string, args ...interface{}) {
	l.std.WithField("inmem", l.name).Infof(format, args...)
}

func (l *stdLogger) Debug(args ...interface{}) {
	l.std.WithField("inmem", l.name).Debug(args...)
}

func (l *stdLogger) Debugf(format string, args ...interface{}) {
	l.std.WithField("inmem  ", l.name).Debugf(format, args...)
}

func (l *stdLogger) Warn(args ...interface{}) {
	l.std.WithField("inmem", l.name).Warn(args...)
}

func (l *stdLogger) Warnf(format string, args ...interface{}) {
	l.std.WithField("inmem", l.name).Warnf(format, args...)
}

func (l *stdLogger) Error(args ...interface{}) {
	l.std.WithField("inmem", l.name).Error(args...)
}

func (l *stdLogger) Errorf(format string, args ...interface{}) {
	l.std.WithField("inmem", l.name).Errorf(format, args...)
}

// make sure supressedLoger implements the Logger interface.
var _ Logger = (*supressedLoger)(nil)

type supressedLoger struct{}

func (l *supressedLoger) Info(args ...interface{})                  {}
func (l *supressedLoger) Infof(format string, args ...interface{})  {}
func (l *supressedLoger) Debug(args ...interface{})                 {}
func (l *supressedLoger) Debugf(format string, args ...interface{}) {}
func (l *supressedLoger) Warn(args ...interface{})                  {}
func (l *supressedLoger) Warnf(format string, args ...interface{})  {}
func (l *supressedLoger) Error(args ...interface{})                 {}
func (l *supressedLoger) Errorf(format string, args ...interface{}) {}

// New creates a new logger instance with the specified name and logging level.
func New(name string, suprresed, debugLogs bool) Logger {
	if suprresed {
		return &supressedLoger{}
	}

	l := log.New()
	l.SetOutput(os.Stdout)
	l.SetLevel(log.InfoLevel)

	if debugLogs {
		l.SetLevel(log.DebugLevel)
	}

	l.SetFormatter(&log.TextFormatter{
		DisableColors:   false,
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		PadLevelText:    true,
	})

	return &stdLogger{
		std:  l,
		name: name,
	}

}
