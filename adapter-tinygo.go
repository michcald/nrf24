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

// NewTinyGo creates a new NRF24L01 driver for TinyGo systems.
func NewTinyGo(c Config, spi *machine.SPI, csPin, cePin, irqPin machine.Pin) (*Device, error) {
	// Configure CS pin as output and set high (inactive)
	csPin.Configure(machine.PinConfig{Mode: machine.PinOutput})
	csPin.High()

	ceWrapper := &tinygoPin{pin: cePin}
	
	var irqWrapper Pin
	if irqPin != machine.NoPin {
		irqWrapper = &tinygoPin{pin: irqPin}
	}

	spiWrapper := &tinygoSPI{spi: spi, cs: csPin}

	return NewWithHardware(c, spiWrapper, ceWrapper, irqWrapper)
}
