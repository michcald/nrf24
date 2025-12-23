//go:build tinygo

package nrf24

import (
	"machine"
)

func init() {
	globalLogger = &serialLogger{}
}

// serialLogger is a default logger for TinyGo that uses machine.Serial directly
// to avoid the memory overhead of the fmt package.
type serialLogger struct{}

func (l *serialLogger) log(level, msg string) {
	machine.Serial.Write([]byte(level))
	machine.Serial.Write([]byte(msg))
	machine.Serial.Write([]byte("\r\n"))
}

func (l *serialLogger) Debug(msg string) { l.log("[DEBUG] ", msg) }
func (l *serialLogger) Info(msg string)  { l.log("[INFO]  ", msg) }
func (l *serialLogger) Warn(msg string)  { l.log("[WARN]  ", msg) }
func (l *serialLogger) Error(msg string) { l.log("[ERROR] ", msg) }