# nrf24

A lightweight, idiomatic Go library for interfacing with the nRF24L01+ 2.4GHz wireless transceiver. This library is designed for Linux-based systems (Raspberry Pi, BeagleBone, etc.) using the standard spidev interface.

## Features

- Simple API: High-level abstractions for Transmitting (TX) and Receiving (RX).

- Configurable: Full control over power levels, data rates, and channel frequencies.

- Automatic Retries: Built-in support for Enhanced ShockBurst™ (Auto-ACK).

- Dynamic Payloads: Support for variable packet lengths.
