package main

import (
	"context"

	meshcore_serial "github.com/kellegous/meshcore/serial"
	"github.com/kellegous/poop"
)

func main() {
	if err := run(context.Background()); err != nil {
		poop.HitFan(err)
	}
}

func run(ctx context.Context) error {
	c, err := meshcore_serial.Connect(ctx, "/dev/cu.usbserial-0001")
	if err != nil {
		return poop.Chain(err)
	}
	defer c.Disconnect()

	ch := make(chan struct{})

	<-ch

	return nil
}
