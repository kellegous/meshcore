package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/kellegous/meshcore"
	meshcore_bluetooth "github.com/kellegous/meshcore/bluetooth"
	meshcore_serial "github.com/kellegous/meshcore/serial"
	"github.com/kellegous/poop"
	"golang.org/x/sync/errgroup"
	"tinygo.org/x/bluetooth"
)

type Flags struct {
	Verbose bool
}

func main() {
	if err := run(context.Background()); err != nil {
		poop.HitFan(err)
	}
}

func run(ctx context.Context) error {
	var flags Flags
	flag.BoolVar(&flags.Verbose, "verbose", false, "verbose output")

	flag.Parse()
	if flag.NArg() != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <address> <address>\n", os.Args[0])
		os.Exit(1)
	}

	click, err := connect(
		ctx, flag.Arg(0),
		onSend("click", flags.Verbose),
		onRecv("click", flags.Verbose),
	)
	if err != nil {
		return poop.Chain(err)
	}
	defer click.Disconnect()

	fmt.Printf("click: %s\n", flag.Arg(0))

	clack, err := connect(
		ctx, flag.Arg(1),
		onSend("clack", flags.Verbose),
		onRecv("clack", flags.Verbose),
	)
	if err != nil {
		return poop.Chain(err)
	}
	defer clack.Disconnect()

	fmt.Printf("clack: %s\n", flag.Arg(1))

	clickInfo, err := click.GetSelfInfo(ctx)
	if err != nil {
		return poop.Chain(err)
	}
	fmt.Printf("click info: %+v\n", clickInfo)

	clackInfo, err := clack.GetSelfInfo(ctx)
	if err != nil {
		return poop.Chain(err)
	}
	fmt.Printf("clack info: %+v\n", clackInfo)

	// reset contacts
	if err := func() error {
		fmt.Printf("resetting contacts\n")
		defer fmt.Printf("contacts reset\n")

		g, ctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			return resetContacts(ctx, click)
		})
		g.Go(func() error {
			return resetContacts(ctx, clack)
		})
		if err := g.Wait(); err != nil {
			return poop.Chain(err)
		}
		return nil
	}(); err != nil {
		return poop.Chain(err)
	}

	// exchange contacts
	if err := func() error {
		fmt.Printf("exchanging contacts\n")
		defer fmt.Printf("contacts exchanged\n")

		if err := discover(ctx, click, clack); err != nil {
			return poop.Chain(err)
		}

		if err := discover(ctx, clack, click); err != nil {
			return poop.Chain(err)
		}

		return nil
	}(); err != nil {
		return poop.Chain(err)
	}

	return nil
}

func onSend(name string, verbose bool) func(code meshcore.CommandCode, data []byte) {
	if !verbose {
		return func(code meshcore.CommandCode, data []byte) {}
	}
	return func(code meshcore.CommandCode, data []byte) {
		fmt.Printf("%s: <send: %v>\n", name, code)
	}
}

func onRecv(name string, verbose bool) func(code meshcore.NotificationCode, data []byte) {
	if !verbose {
		return func(code meshcore.NotificationCode, data []byte) {}
	}
	return func(code meshcore.NotificationCode, data []byte) {
		fmt.Printf("%s: <recv: %v>\n", name, code)
	}
}

func connect(
	ctx context.Context,
	name string,
	onSend func(code meshcore.CommandCode, data []byte),
	onRecv func(code meshcore.NotificationCode, data []byte),
) (*meshcore.Conn, error) {
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
		return client.Connect(
			ctx,
			device.Address,
			meshcore_bluetooth.OnRecv(onRecv),
			meshcore_bluetooth.OnSend(onSend),
		)
	case "usb", "serial":
		return meshcore_serial.Connect(
			ctx,
			addr,
			meshcore_serial.OnRecv(onRecv),
			meshcore_serial.OnSend(onSend),
		)
	}

	return nil, poop.Newf("invalid transport: %s", tx)
}

func resetContacts(
	ctx context.Context,
	conn *meshcore.Conn,
) error {
	contacts, err := conn.GetContacts(ctx, nil)
	if err != nil {
		return poop.Chain(err)
	}

	for _, contact := range contacts {
		if err := conn.RemoveContact(ctx, &contact.PublicKey); err != nil {
			return poop.Chain(err)
		}
	}

	return nil
}

func discover(
	ctx context.Context,
	advertiser *meshcore.Conn,
	listener *meshcore.Conn,
) error {
	g, bgCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		for advert, err := range listener.Notifications(bgCtx, meshcore.NotificationTypeAdvert) {
			if err != nil {
				return poop.Chain(err)
			}
			fmt.Printf("advert: %v\n", advert)
			break
		}
		return nil
	})

	if err := advertiser.SendAdvert(ctx, meshcore.SelfAdvertTypeFlood); err != nil {
		return poop.Chain(err)
	}

	if err := g.Wait(); err != nil {
		return poop.Chain(err)
	}

	return nil
}
