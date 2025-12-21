package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/michcald/nrf24"
)

func main() {
	fmt.Println("Starting NRF24L01+ Receiver...")

	// Setup clean shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

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
		log.Fatalf("Failed to initialize radio: %v", err)
	}
	defer radio.Close()

	fmt.Printf("Radio initialized: %s\n", radio)
	fmt.Println("Waiting for packets...")

	for {
		// Wait for packet with a 1-second timeout loop to check for context cancel
		// Or pass the main ctx directly if we don't need periodic checks.
		packet, err := radio.ReceiveBlocking(ctx)
		if err != nil {
			if ctx.Err() != nil {
				// Cancelled by user
				break
			}
			// Other error (e.g. timeout if we used WithTimeout)
			continue 
		}

		fmt.Printf("Received: %s\n", string(packet))
	}
}
