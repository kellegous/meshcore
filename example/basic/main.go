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
	var sendMessage, getTelemetry, getChannels bool
	flag.BoolVar(
		&sendMessage,
		"send-message",
		false,
		"send a message to the device",
	)
	flag.BoolVar(
		&getTelemetry,
		"get-telemetry",
		false,
		"get the telemetry from the device",
	)
	flag.BoolVar(
		&getChannels,
		"get-channels",
		false,
		"get the channels from the device",
	)
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

	if sendMessage && len(contacts) > 0 {
		contact := contacts[0]

		sr, err := conn.SendTextMessage(ctx, &contact.PublicKey, "Hello, world!", meshcore.TextTypePlain)
		if err != nil {
			return poop.Chain(err)
		}
		fmt.Printf("sent message: %+v\n", sr)
	}
	if getTelemetry && len(contacts) > 0 {
		contact := contacts[0]
		telemetry, err := conn.GetTelemetry(ctx, &contact.PublicKey)
		if err != nil {
			return poop.Chain(err)
		}
		fmt.Printf("telemetry: %+v\n", telemetry)
	}
	if getChannels {
		channels, err := conn.GetChannels(ctx)
		if err != nil {
			return poop.Chain(err)
		}
		for _, channel := range channels {
			fmt.Printf("channel: %+v\n", channel)
		}
	}

	return nil
}
