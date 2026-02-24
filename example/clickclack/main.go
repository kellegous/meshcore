package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/kellegous/meshcore"
	meshcore_bluetooth "github.com/kellegous/meshcore/bluetooth"
	meshcore_serial "github.com/kellegous/meshcore/serial"
	"github.com/kellegous/poop"
	"golang.org/x/sync/errgroup"
	"tinygo.org/x/bluetooth"
)

type Flags struct {
	Verbose bool
	Color   bool
}

type Actor struct {
	Name string
	*meshcore.Conn
	Printf func(format string, a ...any) (int, error)
}

func main() {
	if err := run(context.Background()); err != nil {
		poop.HitFan(err)
	}
}

func getPrintf(
	useColor bool,
	c *color.Color,
) func(format string, a ...any) (int, error) {
	if !useColor {
		return fmt.Printf
	}
	return c.Printf
}

func run(ctx context.Context) error {
	var flags Flags
	flag.BoolVar(&flags.Verbose, "verbose", false, "verbose output")
	flag.BoolVar(&flags.Color, "color", true, "color output")

	flag.Parse()
	if flag.NArg() != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <address> <address>\n", os.Args[0])
		os.Exit(1)
	}

	narrator := getPrintf(flags.Color, color.New(color.FgGreen))

	click, err := newActor(
		ctx,
		"click",
		flag.Arg(0),
		getPrintf(flags.Color, color.New(color.FgCyan)),
		onSend("click", flags.Verbose),
		onRecv("click", flags.Verbose),
	)
	if err != nil {
		return poop.Chain(err)
	}
	defer click.Disconnect()

	click.Printf("click: %s\n", flag.Arg(0))

	clack, err := newActor(
		ctx,
		"clack",
		flag.Arg(1),
		getPrintf(flags.Color, color.New(color.FgYellow)),
		onSend("clack", flags.Verbose),
		onRecv("clack", flags.Verbose),
	)
	if err != nil {
		return poop.Chain(err)
	}
	defer clack.Disconnect()

	clack.Printf("clack: %s\n", flag.Arg(1))

	clickInfo, err := click.GetSelfInfo(ctx)
	if err != nil {
		return poop.Chain(err)
	}
	click.Printf(
		"click: self info PublicKey=%s, AdvertName=%s, AdvertLat=%0.3f, AdvertLon=%0.3f\n",
		clickInfo.PublicKey.String(),
		clickInfo.Name,
		clickInfo.AdvLat,
		clickInfo.AdvLon,
	)

	clackInfo, err := clack.GetSelfInfo(ctx)
	if err != nil {
		return poop.Chain(err)
	}
	clack.Printf(
		"clack: self info PublicKey=%s, AdvertName=%s, AdvertLat=%0.3f, AdvertLon=%0.3f\n",
		clackInfo.PublicKey.String(),
		clackInfo.Name,
		clackInfo.AdvLat,
		clackInfo.AdvLon,
	)

	// reset contacts
	if err := func() error {
		narrator("resetting contacts\n")
		defer narrator("contacts reset\n")

		g, ctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			return resetContacts(ctx, click.Conn)
		})
		g.Go(func() error {
			return resetContacts(ctx, clack.Conn)
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

		if err := discover(ctx, click.Conn, clack.Conn); err != nil {
			return poop.Chain(err)
		}

		if err := discover(ctx, clack.Conn, click.Conn); err != nil {
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

func newActor(
	ctx context.Context,
	name string,
	addr string,
	printf func(format string, a ...any) (int, error),
	onSend func(code meshcore.CommandCode, data []byte),
	onRecv func(code meshcore.NotificationCode, data []byte),
) (*Actor, error) {
	conn, err := connect(ctx, addr, onSend, onRecv)
	if err != nil {
		return nil, poop.Chain(err)
	}
	return &Actor{
		Name:   name,
		Conn:   conn,
		Printf: printf,
	}, nil
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
