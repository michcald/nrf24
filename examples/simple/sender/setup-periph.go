//go:build !tinygo

package main

import (
	"fmt"

	"github.com/michcald/nrf24"
)

func Setup() (*nrf24.Device, error) {
	fmt.Println("Starting NRF24L01+ Sender...")

	config := nrf24.Config{
		RadioConfig: nrf24.RadioConfig{
			ChannelNumber:        76,
			DataRate:             nrf24.DataRate1mbps,
			EnableAutoAck:        true,
			EnableDynamicPayload: true,
			RxAddr:               nrf24.Address{0xD7, 0xD7, 0xD7, 0xD7, 0xD7},
			AutoRetransmitDelay:  500, // 500us
			AutoRetransmitCount:  15,  // Max retries
		},
		CEPin: 25,
	}

	radio, err := nrf24.New(config)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Radio initialized: %s\n", radio)
	return radio, nil
}

func Log(msg string) {
	fmt.Print(msg)
}
