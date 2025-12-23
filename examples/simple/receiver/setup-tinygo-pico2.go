//go:build tinygo

package main

import (
	"machine"
	"time"

	"github.com/michcald/nrf24"
)

func Setup() (*nrf24.Device, error) {
	machine.Serial.Configure(machine.UARTConfig{BaudRate: 115200})
	time.Sleep(2 * time.Second) // Give time to open serial monitor
	machine.Serial.Write([]byte("Starting NRF24L01+ Receiver on Pico 2...\r\n"))

	// SPI configuration for Pico 2
	// SCK: GP18, TX(MOSI): GP19, RX(MISO): GP16
	err := machine.SPI0.Configure(machine.SPIConfig{
		Frequency: 1000000,
		Mode:      0,
		SCK:       machine.GP18,
		SDO:       machine.GP19,
		SDI:       machine.GP16,
	})
	if err != nil {
		machine.Serial.Write([]byte("Failed to configure SPI\r\n"))
		return nil, err
	}

	config := nrf24.Config{
		RadioConfig: nrf24.RadioConfig{
			ChannelNumber:        76,
			DataRate:             nrf24.DataRate1mbps,
			EnableAutoAck:        true,
			EnableDynamicPayload: true,
			RxAddr:               nrf24.Address{0xE7, 0xE7, 0xE7, 0xE7, 0xE7},
		},
		SPI:    machine.SPI0,
		CSPin:  machine.GP17,
		CEPin:  machine.GP20,
		IRQPin: machine.NoPin, // Use polling for debugging to rule out IRQ issues
	}

	time.Sleep(1 * time.Second)

	machine.Serial.Write([]byte("Wiring: SCK=GP18, MOSI=GP19, MISO=GP16, CS=GP17, CE=GP20, IRQ=Not Used (Polling)\r\n"))

	radio, err := nrf24.New(config)
	if err != nil {
		machine.Serial.Write([]byte("Failed to initialize radio\r\n"))
		return nil, err
	}

	machine.Serial.Write([]byte("Radio initialized: "))
	machine.Serial.Write([]byte(radio.String()))
	machine.Serial.Write([]byte("\r\n"))
	return radio, nil
}

func Log(msg string) {
	machine.Serial.Write([]byte(msg))
	machine.Serial.Write([]byte("\r\n"))
}
