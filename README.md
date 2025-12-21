# nrf24

[![CI](https://github.com/michcald/nrf24/actions/workflows/ci.yml/badge.svg)](https://github.com/michcald/nrf24/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/michcald/nrf24.svg)](https://pkg.go.dev/github.com/michcald/nrf24)

A robust, idiomatic, and concurrent-safe Go library for interfacing with the nRF24L01+ 2.4GHz wireless transceiver. 

**Target Platform:** This library is designed for Linux-based systems (Raspberry Pi, BeagleBone, Jetson Nano, etc.) using the standard `spidev` and `sysfs/cdev` GPIO interfaces via `periph.io`. It is **not** intended for bare-metal microcontrollers.

## Origin

This package was born out of the necessity for a personal IoT project that required reliable, high-performance communication between Linux-based gateway devices.

## Features

- **Robust Concurrency:** Thread-safe API (`Transmit`, `Receive`, `Ping`) allowing safe use from multiple goroutines.
- **Interrupt Driven:** Supports `WaitForInterrupt` and `ReceiveBlocking` using hardware IRQ pins for high efficiency.
- **Full Multiceiver Support:** Configure and listen on all 6 data pipes simultaneously.
- **Advanced Hardware Features:**
  - **Dynamic Payloads:** Variable packet lengths up to 32 bytes.
  - **Auto-Ack & Retries:** Reliable delivery with configurable hardware retransmission.
  - **ACK Payloads:** Piggyback response data on automatic acknowledgements for instant bi-directional communication.
  - **No-Ack Transmit:** Efficient broadcast support.
- **Diagnostics:** On-the-fly carrier detection, packet loss counters, and manual FIFO management.
- **Typed Errors:** Programmatic detection of `ErrMaxRetries` vs `ErrTimeout`.
- **Unit Tested:** Interface-based design allows full logic verification without physical hardware.

## Installation

```bash
go get github.com/michcald/nrf24
```

## Quick Start

### Receiver

```go
package main

import (
	"context"
	"fmt"
	"log"
	"github.com/michcald/nrf24"
)

func main() {
	config := nrf24.Config{
		ChannelNumber: 76,
		CePin:         25,
		IrqPin:        24, 
		RxAddr:        nrf24.Address{0xE7, 0xE7, 0xE7, 0xE7, 0xE7},
	}

	radio, _ := nrf24.New(config)
	defer radio.Close()

	fmt.Println("Listening...")
	// Efficiently block until a packet arrives
	pkt, _ := radio.ReceiveBlocking(context.Background())
	fmt.Printf("Received: %s\n", string(pkt))
}
```

### Transmitter

```go
package main

import (
	"fmt"
	"log"
	"github.com/michcald/nrf24"
)

func main() {
	config := nrf24.Config{
		ChannelNumber: 76,
		CePin:         25,
		// IRQ not strictly needed for basic TX, but good for error handling
	}

	radio, _ := nrf24.New(config)
	defer radio.Close()

	target := nrf24.Address{0xE7, 0xE7, 0xE7, 0xE7, 0xE7}
	err := radio.Transmit(target, []byte("Hello Go!"))
	if err != nil {
		fmt.Printf("Transmit failed: %v\n", err)
	} else {
		fmt.Println("Message sent!")
	}
}
```

## Examples

Check the [examples/](examples/) directory for more detailed implementations:
*   [Simple Sender/Receiver](examples/simple/): Basic one-way communication setup.

## Contributors

- **[michcald](https://github.com/michcald)**: A Go expert who did the architectural thinking and requirements, despite limited experience with IoT and radio devices.
- **Gemini (AI)**: Did the heavy lifting of the implementation, hardware logic mapping, and testing suite.

## License

MIT - See [LICENSE](LICENSE) for details.