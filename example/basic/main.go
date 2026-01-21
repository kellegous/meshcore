package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/kellegous/meshcore"
	meshcore_bluetooth "github.com/kellegous/meshcore/bluetooth"
	"github.com/kellegous/poop"
	"tinygo.org/x/bluetooth"
)

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

	client, err := meshcore_bluetooth.NewClient(bluetooth.DefaultAdapter)
	if err != nil {
		return poop.Chain(err)
	}

	device, err := client.LookupDevice(ctx, flag.Arg(0))
	if err != nil {
		return poop.Chain(err)
	}

	conn, err := client.Connect(ctx, device.Address)
	if err != nil {
		return poop.Chain(err)
	}
	defer conn.Disconnect()

	contacts, err := conn.GetContacts(ctx, nil)
	if err != nil {
		return poop.Chain(err)
	}

	for _, contact := range contacts {
		fmt.Printf("contact: %+v\n", contact)
	}

	t, err := conn.GetDeviceTime(ctx)
	if err != nil {
		return poop.Chain(err)
	}
	fmt.Printf("device time: %s\n", t)

	if len(contacts) > 0 {
		contact := contacts[0]

		sr, err := conn.SendTextMessage(ctx, &contact.PublicKey, "Hello, world!", meshcore.TextTypePlain)
		if err != nil {
			return poop.Chain(err)
		}
		fmt.Printf("sent message: %+v\n", sr)
	}

	return nil
}
