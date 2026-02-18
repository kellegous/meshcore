package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"flag"
	"fmt"
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
	var sendMessage,
		getTelemetry,
		getChannels,
		deviceQuery,
		reboot,
		syncNextMessage,
		sendAdvert,
		exportContact,
		getStatus,
		exportPrivateKey,
		sign,
		getSelfInfo bool

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
	flag.BoolVar(
		&deviceQuery,
		"device-query",
		false,
		"query the device information",
	)
	flag.BoolVar(
		&reboot,
		"reboot",
		false,
		"reboot the device",
	)
	flag.BoolVar(
		&syncNextMessage,
		"sync-next-message",
		false,
		"synchronize the next message from the device",
	)
	flag.BoolVar(
		&sendAdvert,
		"send-advert",
		false,
		"send an advert to the device",
	)
	flag.BoolVar(
		&exportContact,
		"export-contact",
		false,
		"export a contact from the device",
	)
	flag.BoolVar(
		&getStatus,
		"get-status",
		false,
		"get the status from the device",
	)
	flag.BoolVar(
		&exportPrivateKey,
		"export-private-key",
		false,
		"export the private key from the device",
	)
	flag.BoolVar(
		&sign,
		"sign",
		false,
		"sign a message with the device",
	)
	flag.BoolVar(
		&getSelfInfo,
		"get-self-info",
		false,
		"get the self information from the device",
	)
	flag.Parse()

	if flag.NArg() != 1 {
		return poop.Newf("expected 1 argument, got %d", flag.NArg())
	}

	conn, err := connect(ctx, flag.Arg(0))
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
	if deviceQuery {
		deviceInfo, err := conn.DeviceQuery(ctx, 1)
		if err != nil {
			return poop.Chain(err)
		}
		fmt.Printf("device info: %+v\n", deviceInfo)
	}
	if reboot {
		if err := conn.Reboot(ctx); err != nil {
			return poop.Chain(err)
		}
		fmt.Printf("rebooted device\n")
	}
	if syncNextMessage {
		message, err := conn.SyncNextMessage(ctx)
		if err != nil {
			return poop.Chain(err)
		}
		fmt.Printf("next message: %+v\n", message)
	}
	if sendAdvert {
		if err := conn.SendAdvert(ctx, meshcore.SelfAdvertTypeZeroHop); err != nil {
			return poop.Chain(err)
		}
		fmt.Printf("sent advert\n")
	}
	if exportContact {
		advertPacket, err := conn.ExportContact(ctx, nil)
		if err != nil {
			return poop.Chain(err)
		}
		fmt.Printf("advert packet: %+v (%d)\n", advertPacket, len(advertPacket))
	}
	if getStatus {
		if len(contacts) == 0 {
			return poop.New("no contacts found")
		}
		contact := contacts[0]
		status, err := conn.GetStatus(ctx, contact.PublicKey)
		if err != nil {
			return poop.Chain(err)
		}
		fmt.Printf("status: %+v\n", status)
	}
	if exportPrivateKey {
		key, err := conn.ExportPrivateKey(ctx)
		if err != nil {
			return poop.Chain(err)
		}
		fmt.Printf("private key: %s\n", hex.EncodeToString(key))
	}
	if sign {
		message := []byte("Hello, world!")
		signature, err := conn.Sign(ctx, message)
		if err != nil {
			return poop.Chain(err)
		}
		fmt.Printf("signature: %s\n", hex.EncodeToString(signature))

		selfInfo, err := conn.GetSelfInfo(ctx)
		if err != nil {
			return poop.Chain(err)
		}
		fmt.Printf("public key: %s\n", hex.EncodeToString(selfInfo.PublicKey.Bytes()))

		if ed25519.Verify(
			selfInfo.PublicKey.Bytes(),
			message,
			signature,
		) {
			fmt.Printf("signature is valid\n")
		} else {
			fmt.Printf("signature is invalid\n")
		}
	}
	if getSelfInfo {
		selfInfo, err := conn.GetSelfInfo(ctx)
		if err != nil {
			return poop.Chain(err)
		}
		fmt.Printf("self info: %+v\n", selfInfo)
	}
	return nil
}
