package nrf24

import (
	"log"
)

// Logger defines the logging interface with levels.
type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// stdLogger is a default logger that uses the standard library log package.
type stdLogger struct{}

func (l *stdLogger) Debugf(format string, args ...interface{}) {
	log.Printf("[DEBUG] "+format, args...)
}

func (l *stdLogger) Infof(format string, args ...interface{}) {
	log.Printf("[INFO]  "+format, args...)
}

func (l *stdLogger) Warnf(format string, args ...interface{}) {
	log.Printf("[WARN]  "+format, args...)
}

func (l *stdLogger) Errorf(format string, args ...interface{}) {
	log.Printf("[ERROR] "+format, args...)
}

// nopLogger is a logger that does nothing.
type nopLogger struct{}

func (l *nopLogger) Debugf(format string, args ...interface{}) {}
func (l *nopLogger) Infof(format string, args ...interface{})  {}
func (l *nopLogger) Warnf(format string, args ...interface{})  {}
func (l *nopLogger) Errorf(format string, args ...interface{}) {}
