//go:build !tinygo

package nrf24

import (
	"fmt"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
	"periph.io/x/conn/v3/spi/spireg"
	"periph.io/x/host/v3"
)

// realPin wraps a gpio.PinIO to satisfy the Pin interface.
type realPin struct {
	gpio.PinIO
	stopWatch chan struct{}
}

func (p *realPin) Out(l Level) error {
	if l == High {
		return p.PinIO.Out(gpio.High)
	}
	return p.PinIO.Out(gpio.Low)
}

func (p *realPin) In(pull Pull) error {
	var pPull gpio.Pull
	switch pull {
	case PullFloat:
		pPull = gpio.Float
	case PullDown:
		pPull = gpio.PullDown
	case PullUp:
		pPull = gpio.PullUp
	default:
		pPull = gpio.PullNoChange
	}
	return p.PinIO.In(pPull, gpio.NoEdge)
}

func (p *realPin) Read() Level {
	if p.PinIO.Read() == gpio.High {
		return High
	}
	return Low
}

func (p *realPin) Watch(edge Edge, handler func()) error {
	var pEdge gpio.Edge
	switch edge {
	case RisingEdge:
		pEdge = gpio.RisingEdge
	case FallingEdge:
		pEdge = gpio.FallingEdge
	case BothEdges:
		pEdge = gpio.BothEdges
	default:
		pEdge = gpio.NoEdge
	}

	// Ensure we are in input mode with the correct edge detection
	if err := p.PinIO.In(gpio.PullUp, pEdge); err != nil {
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

// Config holds the configuration for the Linux/periph.io driver.
type Config struct {
	RadioConfig
	// CEPin is the GPIO pin number (BCM numbering) for the Chip Enable (CE) pin.
	// Defaults to 25 if not provided.
	CEPin int
	// IRQPin is the GPIO pin number (BCM numbering) for the Interrupt Request (IRQ) pin.
	// Optional. If not provided, polling is used.
	IRQPin int
	// SpiBusPath is the path to the SPI bus (e.g., "/dev/spidev0.0").
	// Defaults to "/dev/spidev0.0" if not provided.
	SpiBusPath string
	// SpiClockHz is the SPI clock frequency in Hz.
	// Defaults to 1000000 (1MHz) if not provided.
	SpiClockHz int
}

// New creates and initializes a new NRF24L01 driver for Linux systems.
// It applies configuration defaults, initializes the GPIO and SPI interfaces using periph.io,
// and configures the radio module.
// It returns the initialized driver or an error if hardware initialization fails.
func New(c Config) (*Device, error) {
	// 1. Initialize periph.io host (Required for both SPI and GPIO)
	if _, err := host.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize periph.io host: %w", err)
	}

	// 2. Default SPI Path
	if c.SpiBusPath == "" {
		c.SpiBusPath = "/dev/spidev0.0"
	}

	// 3. Open the SPI Port
	p, err := spireg.Open(c.SpiBusPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SPI port: %w", err)
	}

	// 4. Default Clock
	if c.SpiClockHz == 0 {
		c.SpiClockHz = 1000000
	}

	// 5. Create the SPI Connection (Mode 0, 8 bits)
	conn, err := p.Connect(physic.Frequency(c.SpiClockHz)*physic.Hertz, spi.Mode0, 8)
	if err != nil {
		p.Close()
		return nil, fmt.Errorf("failed to create SPI connection: %w", err)
	}

	// 6. Setup CE Pin
	if c.CEPin == 0 {
		c.CEPin = 25
	}
	ceName := fmt.Sprintf("GPIO%d", c.CEPin)
	realCe := gpioreg.ByName(ceName)
	if realCe == nil {
		p.Close()
		return nil, fmt.Errorf("failed to open CE pin %s", ceName)
	}
	ceWrapper := &realPin{PinIO: realCe}

	// 7. Setup IRQ Pin
	var irqWrapper Pin
	if c.IRQPin != 0 {
		irqName := fmt.Sprintf("GPIO%d", c.IRQPin)
		realIrq := gpioreg.ByName(irqName)
		if realIrq == nil {
			p.Close()
			return nil, fmt.Errorf("failed to open IRQ pin %s", irqName)
		}
		irqWrapper = &realPin{PinIO: realIrq}
	}

	// 8. Call internal constructor
	hwConfig := HardwareConfig{
		RadioConfig: c.RadioConfig,
		CE:          ceWrapper,
		IRQ:         irqWrapper,
	}
	dev, err := NewWithHardware(hwConfig, conn)
	if err != nil {
		p.Close()
		return nil, err
	}

	// Store the port closer so we can close it later
	dev.nrfPort = p
	return dev, nil
}
