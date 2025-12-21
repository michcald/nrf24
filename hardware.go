package nrf24

import (
	"periph.io/x/conn/v3/gpio"
)

// realPin wraps a gpio.PinIO to satisfy the pin interface.
type realPin struct {
	gpio.PinIO
	stopWatch chan struct{}
}

func (p *realPin) Out(l gpio.Level) error {
	return p.PinIO.Out(l)
}

func (p *realPin) In(pull gpio.Pull, edge gpio.Edge) error {
	return p.PinIO.In(pull, edge)
}

func (p *realPin) Read() gpio.Level {
	return p.PinIO.Read()
}

func (p *realPin) Watch(edge gpio.Edge, handler func()) error {
	// Ensure we are in input mode with the correct edge detection
	if err := p.PinIO.In(gpio.PullUp, edge); err != nil {
		return err
	}

	p.stopWatch = make(chan struct{})

	go func() {
		for {
			// Wait for edge with -1 (infinite timeout)
			if p.PinIO.WaitForEdge(-1) {
				select {
				case <-p.stopWatch:
					return
				default:
					handler()
				}
			} else {
				// WaitForEdge returned false (timeout or error), check stop
				select {
				case <-p.stopWatch:
					return
				default:
					// Just continue or verify pin state
					// periph.io WaitForEdge might return false on timeout, 
					// but with -1 it should block.
				}
			}
		}
	}()
	return nil
}

func (p *realPin) Unwatch() error {
	if p.stopWatch != nil {
		close(p.stopWatch)
		p.stopWatch = nil
	}
	// Disable edge detection
	return p.PinIO.In(gpio.PullUp, gpio.NoEdge)
}
