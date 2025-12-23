//go:build !tinygo

package nrf24

import (
	"log"
)

func init() {
	globalLogger = &stdLogger{}
}

// stdLogger is a default logger that uses the standard library log package.
type stdLogger struct{}

func (l *stdLogger) Debug(msg string) {
	log.Print("[DEBUG] " + msg)
}

func (l *stdLogger) Info(msg string) {
	log.Print("[INFO]  " + msg)
}

func (l *stdLogger) Warn(msg string) {
	log.Print("[WARN]  " + msg)
}

func (l *stdLogger) Error(msg string) {
	log.Print("[ERROR] " + msg)
}