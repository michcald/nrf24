package nrf24

// Logger defines the logging interface for simple string messages.
// Using simple strings instead of formatted strings helps reduce binary size
// and memory allocations on microcontrollers (TinyGo).
type Logger interface {
	Debug(msg string)
	Info(msg string)
	Warn(msg string)
	Error(msg string)
}

var globalLogger Logger = &nopLogger{}

// SetLogger sets the global logger instance.
func SetLogger(l Logger) {
	if l == nil {
		globalLogger = &nopLogger{}
		return
	}
	globalLogger = l
}

// nopLogger is a logger that does nothing.
type nopLogger struct{}

func (l *nopLogger) Debug(msg string) {}
func (l *nopLogger) Info(msg string)  {}
func (l *nopLogger) Warn(msg string)  {}
func (l *nopLogger) Error(msg string) {}
