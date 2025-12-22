# nrf24

[![CI](https://github.com/michcald/nrf24/actions/workflows/ci.yml/badge.svg)](https://github.com/michcald/nrf24/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/michcald/nrf24.svg)](https://pkg.go.dev/github.com/michcald/nrf24)

A robust, idiomatic, and concurrent-safe Go library for interfacing with the nRF24L01+ 2.4GHz wireless transceiver.

**nrf24** is designed from the ground up to be truly cross-platform, supporting:
- **Linux** (Raspberry Pi, BeagleBone, etc.) via `periph.io`.
- **Microcontrollers** (Pico 2, ESP32, Arduino Nano RP2040, etc.) via **TinyGo**.
- **Custom Hardware**: Easily extendable to any platform by implementing simple interfaces.

This library has been verified to work for communication between a **Raspberry Pi 3B** (Linux) and a **Raspberry Pi Pico 2** (TinyGo).

## Origin

This package was born out of the necessity for a personal IoT project that required reliable, high-performance communication between Linux-based gateway devices and microcontroller-based sensors.

## Features

- **Board Agnostic:** Core logic is decoupled from hardware. Use the provided Linux or TinyGo adapters, or write your own.
- **Robust Concurrency:** Thread-safe API (`Transmit`, `Receive`, `Ping`) allowing safe use from multiple goroutines.
- **Interrupt Driven:** Supports `WaitForInterrupt` and `ReceiveBlocking` using hardware IRQ pins for high efficiency.
- **Full Multiceiver Support:** Configure and listen on all 6 data pipes simultaneously.
- **Advanced Hardware Features:**
  - **Dynamic Payloads:** Variable packet lengths up to 32 bytes.
  - **Auto-Ack & Retries:** Reliable delivery with configurable hardware retransmission.
  - **ACK Payloads:** Piggyback response data on automatic acknowledgements.
  - **No-Ack Transmit:** Efficient broadcast support.
- **Typed Errors:** Programmatic detection of `ErrMaxRetries` vs `ErrTimeout`.
- **Unit Tested:** Interface-based design allows full logic verification without physical hardware.

## Quick Start

### Linux (Raspberry Pi, etc.)

```go
package main

import (
	"context"
	"fmt"
	"github.com/michcald/nrf24"
)

func main() {
	config := nrf24.Config{
		ChannelNumber: 76,
		CePin:         25,
		RxAddr:        nrf24.Address{0xE7, 0xE7, 0xE7, 0xE7, 0xE7},
	}

	radio, _ := nrf24.New(config)
	defer radio.Close()

	fmt.Println("Listening...")
	pkt, _ := radio.ReceiveBlocking(context.Background())
	fmt.Printf("Received: %s\n", string(pkt))
}
```

### TinyGo (Pico 2, etc.)

```go
package main

import (
    "context"
    "machine"
    "github.com/michcald/nrf24"
)

func main() {
    config := nrf24.Config{
        ChannelNumber: 76,
    }
    
    // Initialize hardware via machine package
    machine.SPI0.Configure(machine.SPIConfig{})
    cePin := machine.GP25
    irqPin := machine.GP24
    
    radio, _ := nrf24.NewTinyGo(config, machine.SPI0, cePin, irqPin)
    
    radio.ReceiveBlocking(context.Background())
}
```

## Hardware Setup

The nRF24L01+ is sensitive to power quality. Follow these guidelines for reliable communication:

- **Power Supply**:
  - **Using a Socket Adapter**: If using the common 8-pin socket adapter (with built-in voltage regulator), you **must** connect it to **5V** (VBUS on Pico). The adapter regulates this down to a stable 3.3V for the radio.
  - **Direct Connection**: If connecting the module directly to your board, use the **3.3V** rail. Adding a 10µF capacitor across the module's VCC and GND pins is highly recommended to filter noise.
- **Wiring**: See the [Simple Example README](examples/simple/README.md) for detailed pinout tables for Raspberry Pi and Pico 2.

## Troubleshooting

Common issues and their solutions:

- **"max retransmissions reached"**:
  - **Power**: This is the #1 cause. The radio draws current spikes during transmission. If the voltage drops, the packet fails. Ensure you are using a capacitor or the 5V rail for adapters.
  - **Connection**: The receiver might be off, or on a different channel/address.
  - **Interference**: Try a different channel (e.g., > 100) to avoid WiFi interference.

- **"failed to verify NRF24L01 connection"**:
  - The driver reads back the channel register during initialization to confirm the SPI connection.
  - If this fails, check your **SPI wiring** (MISO, MOSI, SCK) and ensure the correct pins are used in your code.

- **Receiver Freezes/Stops**:
  - **Power**: Insufficient power can cause the radio to lock up.
  - **Code**: On microcontrollers (TinyGo), avoid creating many temporary objects in your main loop to prevent Garbage Collection pauses.

## Examples

Check the [examples/](examples/) directory for more detailed implementations:
*   [Simple Sender/Receiver](examples/simple/): Basic one-way communication setup.

## Contributors

- **[michcald](https://github.com/michcald)**: A Go expert who did the architectural thinking and requirements, despite limited experience with IoT and radio devices.
- **Gemini (AI)**: Did the heavy lifting of the implementation, hardware logic mapping, and testing suite.

## License

MIT - See [LICENSE](LICENSE) for details.
