package main

import (
	"fmt"
	"time"

	"github.com/michcald/nrf24"
)

func main() {
	radio, err := Setup()
	if err != nil {
		Log("Setup failed: " + err.Error())
		return
	}
	defer radio.Close()

	targetAddr := nrf24.Address{0xE7, 0xE7, 0xE7, 0xE7, 0xE7}
	counter := 0

	Log("Sending messages...\r\n")

	for {
		counter++
		msg := fmt.Sprintf("Hello World %d", counter)
		Log("Sending: " + msg + "... ")

		err := radio.Transmit(targetAddr, []byte(msg))
		if err != nil {
			Log("Failed: " + err.Error() + "\r\n")
		} else {
			Log("Success!\r\n")
		}

		time.Sleep(1 * time.Second)
	}
}
