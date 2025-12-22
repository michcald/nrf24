package nrf24

// Level represents the logical level of a pin (Low or High).
type Level bool

const (
	Low  Level = false
	High Level = true
)

// Pull represents the internal pull-up/down resistor state.
type Pull uint8

const (
	PullNoChange Pull = iota
	PullFloat
	PullDown
	PullUp
)

// Edge represents the signal edge to trigger an interrupt.
type Edge uint8

const (
	NoEdge Edge = iota
	RisingEdge
	FallingEdge
	BothEdges
)

// SPI represents a generic SPI connection.
type SPI interface {
	// Tx sends w and reads into r.
	// len(r) must be >= len(w).
	Tx(w, r []byte) error
}

// Pin represents a generic GPIO pin.
type Pin interface {
	// Out sets the pin as output with the given level.
	Out(l Level) error
	// In sets the pin as input with the given pull mode.
	In(pull Pull) error
	// Read returns the current level of the pin.
	Read() Level
	// Watch configures an interrupt/callback on the specified edge.
	// The handler should be called when the edge is detected.
	Watch(edge Edge, handler func()) error
	// Unwatch removes the interrupt/callback.
	Unwatch() error
}