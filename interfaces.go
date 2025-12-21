package nrf24

import (
	"periph.io/x/conn/v3/gpio"
)

// pin is an interface wrapper for gpio.PinIO to allow for mocking in tests.
// It is unexported to avoid cluttering the public API.
type pin interface {
	Out(l gpio.Level) error
	In(pull gpio.Pull, edge gpio.Edge) error
	Read() gpio.Level
	// Watch starts a background routine to detect edges and calls handler.
	// This emulates the callback behavior using periph.io's blocking WaitForEdge.
	Watch(edge gpio.Edge, handler func()) error
	Unwatch() error
}