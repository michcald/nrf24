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
		ChannelNumber:        76,
		DataRate:             nrf24.DataRate1mbps,
		EnableAutoAck:        true,
		EnableDynamicPayload: true,
		RxAddr:               nrf24.Address{0xD7, 0xD7, 0xD7, 0xD7, 0xD7},
	}

	// Pins for Pico 2
	csPin := machine.GP17
	cePin := machine.GP20
	irqPin := machine.GP21

	radio, err := nrf24.NewTinyGo(config, machine.SPI0, csPin, cePin, irqPin)
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
