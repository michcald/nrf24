package main

import (
	"context"
	"time"
)

func main() {
	radio, err := Setup()
	if err != nil {
		Log("Setup failed: " + err.Error())
		return
	}
	defer radio.Close()

	Log("Waiting for packets...")

	ctx := context.Background()
	for {
		packet, err := radio.ReceiveBlocking(ctx)
		if err != nil {
			Log("Receive error: " + err.Error())
			time.Sleep(100 * time.Millisecond)
			continue
		}

		Log("Received: " + string(packet))
	}
}
