package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/kellegous/meshcore"
	meshcore_bluetooth "github.com/kellegous/meshcore/bluetooth"
	meshcore_serial "github.com/kellegous/meshcore/serial"
	"github.com/kellegous/poop"
	"tinygo.org/x/bluetooth"
)

func connect(ctx context.Context, name string) (*meshcore.Conn, error) {
	tx, addr, ok := strings.Cut(name, ":")
	if !ok {
		return nil, poop.Newf("invalid name: %s", name)
	}

	switch tx {
	case "ble", "bluetooth":
		client, err := meshcore_bluetooth.NewClient(bluetooth.DefaultAdapter)
		if err != nil {
			return nil, poop.Chain(err)
		}
		device, err := client.LookupDevice(ctx, addr)
		if err != nil {
			return nil, poop.Chain(err)
		}
		return client.Connect(ctx, device.Address)
	case "usb", "serial":
		return meshcore_serial.Connect(ctx, addr)
	}

	return nil, poop.Newf("invalid transport: %s", tx)
}

func main() {
	if err := run(context.Background()); err != nil {
		poop.HitFan(err)
	}
}

func run(ctx context.Context) error {
	flag.Parse()

	if flag.NArg() != 1 {
		return poop.Newf("expected 1 argument, got %d", flag.NArg())
	}

	conn, err := connect(ctx, flag.Arg(0))
	if err != nil {
		return poop.Chain(err)
	}
	defer conn.Disconnect()

	ctx, done := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer done()

	for advert, err := range conn.Notifications(ctx, meshcore.NotificationTypeAdvert) {
		if errors.Is(err, context.Canceled) {
			break
		} else if err != nil {
			return poop.Chain(err)
		}
		fmt.Printf("advert: %+v\n", advert)
	}

	return nil
}
