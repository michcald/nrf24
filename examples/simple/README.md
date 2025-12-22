# Simple NRF24L01+ Example

This example demonstrates a basic one-way communication setup between two NRF24L01+ modules, supporting both Linux (Raspberry Pi, etc.) and microcontrollers via TinyGo (Raspberry Pi Pico 2).

## Structure

*   **sender/**: A program that sends a "Hello World [counter]" message every second.
*   **receiver/**: A program that listens for messages and prints them to the console.

Each directory contains a shared `main.go` and platform-specific setup files:
- `setup-periph.go`: Linux configuration using `periph.io`.
- `setup-tinygo-pico2.go`: TinyGo configuration for Raspberry Pi Pico 2.

## Prerequisites

### For Linux
- A Raspberry Pi or other board with SPI/GPIO support.
- Go installed on the device.

### For TinyGo
- A Raspberry Pi Pico 2.
- TinyGo installed on your machine.

## Pin Connections

| nRF24L01 Pin | Pico 2 Pin | RPi 3B Pin | Note |
| :--- | :--- | :--- | :--- |
| **VCC** | **VBUS (40)** | **5V (2/4)** | **Use 5V if using an adapter**, 3.3V for raw module |
| **GND** | GND | GND | Any Ground pin |
| **CE** | GP20 | GPIO 25 | Chip Enable |
| **CSN** | GP17 | SPI0 CS0 (8) | SPI Chip Select |
| **SCK** | GP18 | SPI0 SCLK (11) | SPI Clock |
| **MOSI** | GP19 | SPI0 MOSI (10) | SPI Data Input |
| **MISO** | GP16 | SPI0 MISO (9) | SPI Data Output |
| **IRQ** | GP21 | (Optional) | Interrupt Request |

## Troubleshooting

- **Power Issues**: The most common cause of `max retransmissions reached` is insufficient power.
  - **Using an Adapter**: If you are using the common nRF24L01 socket adapter (with a built-in regulator), you **must** connect it to **5V** (VBUS on Pico). Connecting it to 3.3V will provide insufficient voltage to the radio.
  - **Raw Module**: If connecting directly, ensure the 3.3V supply is stable. Adding a 10µF capacitor across VCC/GND is highly recommended.
- **Address Mismatch**: Ensure the Sender's `targetAddr` matches the Receiver's `RxAddr`.
- **Wiring**: Double-check SPI pins. Note that TinyGo on Pico 2 requires explicit pin assignment in the code.

## Usage

### Linux
1.  Navigate to the example directory (e.g., `receiver`).
2.  Build and run:
    ```bash
    make build
    ./receiver # or ./sender
    ```

### Pico 2 (TinyGo)
1.  Navigate to the example directory.
2.  Flash the board:
    ```bash
    make flash-pico2
    ```
3.  Monitor the output via serial (115200 baud).