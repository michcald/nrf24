//go:build !tinygo

package main

import (
	"fmt"

	"github.com/michcald/nrf24"
)

func Setup() (*nrf24.Device, error) {
	fmt.Println("Starting NRF24L01+ Receiver...")

	config := nrf24.Config{
		ChannelNumber:        76,
		CePin:                25,
		IrqPin:               24, // Using IRQ pin for interrupt-driven receive
		DataRate:             nrf24.DataRate1mbps,
		EnableAutoAck:        true,
		EnableDynamicPayload: true,
		RxAddr:               nrf24.Address{0xE7, 0xE7, 0xE7, 0xE7, 0xE7},
	}

	radio, err := nrf24.New(config)
	if err != nil {
		return nil, err
	}
	
	fmt.Printf("Radio initialized: %s\n", radio)
	return radio, nil
}

func Log(msg string) {
	fmt.Println(msg)
}
