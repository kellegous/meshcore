package main

import (
	"context"
	"fmt"

	"github.com/kellegous/poop"
	"go.bug.st/serial"
)

func main() {
	if err := run(context.Background()); err != nil {
		poop.HitFan(err)
	}
}

func run(ctx context.Context) error {
	port, err := serial.Open("/dev/cu.usbserial-0001", &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		StopBits: serial.OneStopBit,
		Parity:   serial.NoParity,
	})
	if err != nil {
		return poop.Chain(err)
	}
	defer port.Close()

	fmt.Println(port)

	return nil
}
