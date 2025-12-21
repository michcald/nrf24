package main

import (
	"fmt"
	"log"
	"time"

	"github.com/michcald/nrf24"
)

func main() {
	fmt.Println("Starting NRF24L01+ Sender...")

	config := nrf24.Config{
		ChannelNumber:        76,
		CePin:                25,
		DataRate:             nrf24.DataRate1mbps,
		EnableAutoAck:        true,
		EnableDynamicPayload: true,
		RxAddr:               nrf24.Address{0xE7, 0xE7, 0xE7, 0xE7, 0xE7},
	}

	radio, err := nrf24.New(config)
	if err != nil {
		log.Fatalf("Failed to initialize radio: %v", err)
	}
	defer radio.Close()

	fmt.Printf("Radio initialized: %s\n", radio)

	targetAddr := nrf24.Address{0xE7, 0xE7, 0xE7, 0xE7, 0xE7}
	counter := 0

	for {
		counter++
		msg := fmt.Sprintf("Hello World %d", counter)
		fmt.Printf("Sending: %s... ", msg)

		err := radio.Transmit(targetAddr, []byte(msg))
		if err != nil {
			fmt.Printf("Failed: %v\n", err)
		} else {
			fmt.Printf("Success!\n")
		}

		time.Sleep(1 * time.Second)
	}
}
