//go:build tinygo

package main

import (
	"machine"

	"github.com/michcald/nrf24"
)

func Setup() (*nrf24.Device, error) {
	machine.Serial.Configure(machine.UARTConfig{BaudRate: 115200})
	machine.Serial.Write([]byte("Starting NRF24L01+ Sender on Pico 2...\r\n"))

	// SPI configuration for Pico 2
	err := machine.SPI0.Configure(machine.SPIConfig{
		Frequency: 1000000,
		Mode:      0,
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
			RxAddr:               nrf24.Address{0xD7, 0xD7, 0xD7, 0xD7, 0xD7},
		},
		SPI:    machine.SPI0,
		CSPin:  machine.GP17,
		CEPin:  machine.GP20,
		IRQPin: machine.GP21,
	}

	radio, err := nrf24.New(config)
	if err != nil {
		machine.Serial.Write([]byte("Failed to initialize radio\r\n"))
		return nil, err
	}

	machine.Serial.Write([]byte("Radio initialized.\r\n"))
	return radio, nil
}

func Log(msg string) {
	machine.Serial.Write([]byte(msg))
}
