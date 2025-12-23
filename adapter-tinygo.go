//go:build tinygo

package nrf24

import (
	"machine"
)

// tinygoPin wraps a machine.Pin to satisfy the Pin interface.
type tinygoPin struct {
	pin machine.Pin
}

func (p *tinygoPin) Out(l Level) error {
	p.pin.Configure(machine.PinConfig{Mode: machine.PinOutput})
	p.pin.Set(bool(l))
	return nil
}

func (p *tinygoPin) In(pull Pull) error {
	var mPull machine.PinMode
	switch pull {
	case PullUp:
		mPull = machine.PinInputPullup
	case PullDown:
		mPull = machine.PinInputPulldown
	default:
		mPull = machine.PinInput
	}
	p.pin.Configure(machine.PinConfig{Mode: mPull})
	return nil
}

func (p *tinygoPin) Read() Level {
	return Level(p.pin.Get())
}

func (p *tinygoPin) Watch(edge Edge, handler func()) error {
	var mEdge machine.PinChange
	switch edge {
	case RisingEdge:
		mEdge = machine.PinRising
	case FallingEdge:
		mEdge = machine.PinFalling
	case BothEdges:
		mEdge = machine.PinToggle
	default:
		return nil
	}

	return p.pin.SetInterrupt(mEdge, func(machine.Pin) {
		handler()
	})
}

func (p *tinygoPin) Unwatch() error {
	// TinyGo doesn't always have a clear "Unwatch", but we can set to NoEdge equivalent
	// or just disable interrupt by setting a nil handler if supported.
	// For now, we'll just reconfigure.
	p.pin.Configure(machine.PinConfig{Mode: machine.PinInput})
	return nil
}

// tinygoSPI wraps a machine.SPI to satisfy the SPI interface.
type tinygoSPI struct {
	spi *machine.SPI
	cs  machine.Pin
}

func (s *tinygoSPI) Tx(w, r []byte) error {
	s.cs.Low()
	err := s.spi.Tx(w, r)
	s.cs.High()
	return err
}

// Config holds the configuration for the TinyGo driver.
type Config struct {
	RadioConfig
	// SPI is the SPI bus to use.
	SPI *machine.SPI
	// CSPin is the Chip Select (CS) pin.
	CSPin machine.Pin
	// CEPin is the Chip Enable (CE) pin.
	CEPin machine.Pin
	// IRQPin is the Interrupt Request (IRQ) pin.
	// Use machine.NoPin if not using interrupts.
	IRQPin machine.Pin
}

// New creates a new NRF24L01 driver for TinyGo systems.
func New(c Config) (*Device, error) {
	// Configure CS pin as output and set high (inactive)
	c.CSPin.Configure(machine.PinConfig{Mode: machine.PinOutput})
	c.CSPin.High()

	ceWrapper := &tinygoPin{pin: c.CEPin}
	
	var irqWrapper Pin
	if c.IRQPin != machine.NoPin {
		irqWrapper = &tinygoPin{pin: c.IRQPin}
	}

	spiWrapper := &tinygoSPI{spi: c.SPI, cs: c.CSPin}

	hwConfig := HardwareConfig{
		RadioConfig: c.RadioConfig,
		CE:          ceWrapper,
		IRQ:         irqWrapper,
	}

	return NewWithHardware(hwConfig, spiWrapper)
}
