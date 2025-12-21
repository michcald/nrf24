# Simple NRF24L01+ Example

This example demonstrates a basic one-way communication setup between two NRF24L01+ modules.

## Structure

*   **sender/**: A program that sends a "Hello World [counter]" message every second.
*   **receiver/**: A program that listens for messages and prints them to the console.

## Prerequisites

*   Two Raspberry Pis (or other Linux boards with SPI/GPIO).
*   Two NRF24L01+ modules wired correctly.
*   Go installed on both devices.

## Wiring (Default)

*   **VCC**: 3.3V
*   **GND**: GND
*   **CSN**: SPI CS0 (GPIO 8 / Pin 24) -> Configured as `SpiBusPath: "/dev/spidev0.0"`
*   **CE**: GPIO 25 (Pin 22) -> Configurable in code/flags.
*   **MOSI**: SPI MOSI (GPIO 10 / Pin 19)
*   **MISO**: SPI MISO (GPIO 9 / Pin 21)
*   **SCK**: SPI SCLK (GPIO 11 / Pin 23)
*   **IRQ**: GPIO 24 (Pin 18) -> Optional, receiver uses it for efficient waiting.

## Usage

1.  **Build Receiver:**
    ```bash
    cd receiver
    make build
    ./receiver
    ```

2.  **Build Sender:**
    ```bash
    cd sender
    make build
    ./sender
    ```

You should see messages appearing on the receiver's console.
