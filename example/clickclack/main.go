package main

import (
	"context"
	"encoding/hex"
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
	"golang.org/x/term"
	"tinygo.org/x/bluetooth"
)

type Printer func(format string, a ...any) (int, error)

func (p Printer) Printf(format string, a ...any) (int, error) {
	return p(format, a...)
}

type Flags struct {
	Verbose bool
	Color   bool
}

type Actor struct {
	Info *meshcore.SelfInfo
	*meshcore.Conn
	Printer
}

func main() {
	if err := run(context.Background()); err != nil {
		poop.HitFan(err)
	}
}

func getPrinter(
	useColor bool,
	c *color.Color,
) Printer {
	if !useColor {
		return fmt.Printf
	}
	return c.Printf
}

func center(text string, n int) string {
	spaces := n - len(text)
	if spaces < 0 {
		spaces = 0
	}
	l := spaces / 2
	r := spaces - l
	return strings.Repeat(" ", l) + text + strings.Repeat(" ", r)
}

func getWidth() (int, error) {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 0, err
	}
	return width, nil
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

	width, err := getWidth()
	if err != nil {
		return poop.Chain(err)
	}

	narrator := getPrinter(flags.Color, color.New(color.BgGreen, color.FgBlack))

	click, err := newActor(
		ctx,
		flag.Arg(0),
		getPrinter(flags.Color, color.New(color.FgCyan)),
		flags.Verbose,
	)
	if err != nil {
		return poop.Chain(err)
	}
	defer click.Disconnect()

	click.Printf("connected to %s", flag.Arg(0))

	clack, err := newActor(
		ctx,
		flag.Arg(1),
		getPrinter(flags.Color, color.New(color.FgYellow)),
		flags.Verbose,
	)
	if err != nil {
		return poop.Chain(err)
	}
	defer clack.Disconnect()

	clack.Printf("connected to %s", flag.Arg(1))

	// reset contacts
	if err := func() error {
		narrator.Printf("%s\n", center("Resetting Contacts", width))

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
		narrator.Printf("%s\n", center("Exchanging Contacts", width))

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

func onSend(verbose bool, p Printer) func(code meshcore.CommandCode, data []byte) {
	if !verbose {
		return func(code meshcore.CommandCode, data []byte) {}
	}
	return func(code meshcore.CommandCode, data []byte) {
		p.Printf("<send: %v>", code)
	}
}

func onRecv(verbose bool, p Printer) func(code meshcore.NotificationCode, data []byte) {
	if !verbose {
		return func(code meshcore.NotificationCode, data []byte) {}
	}
	return func(code meshcore.NotificationCode, data []byte) {
		p.Printf("<recv: %v>", code)
	}
}

func newActor(
	ctx context.Context,
	addr string,
	printer Printer,
	verbose bool,
) (*Actor, error) {
	var info *meshcore.SelfInfo
	p := func(format string, a ...any) (int, error) {
		// TODO(kellegous): This is kind of dumb.
		if info == nil {
			return 0, nil
		}
		msg := fmt.Sprintf(format, a...)
		key := hex.EncodeToString(info.PublicKey.Prefix(6))
		return printer.Printf("%s: %s\n", key, msg)
	}

	conn, err := connect(ctx, addr, onSend(verbose, p), onRecv(verbose, p))
	if err != nil {
		return nil, poop.Chain(err)
	}

	info, err = conn.GetSelfInfo(ctx)
	if err != nil {
		return nil, poop.Chain(err)
	}

	return &Actor{Info: info, Conn: conn, Printer: p}, nil
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
	advertiser *Actor,
	listener *Actor,
) error {
	g, bgCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		for notif, err := range listener.Notifications(bgCtx, meshcore.NotificationTypeAdvert) {
			if err != nil {
				return poop.Chain(err)
			}

			advert, ok := notif.(*meshcore.AdvertNotification)
			if !ok {
				return poop.Newf("unexpected advert notification: %T", advert)
			}

			listener.Printf(
				"received advert for %s",
				hex.EncodeToString(advert.PublicKey.Prefix(6)),
			)
			break
		}
		return nil
	})

	advertiser.Printf("sending advert")
	if err := advertiser.SendAdvert(ctx, meshcore.SelfAdvertTypeFlood); err != nil {
		return poop.Chain(err)
	}

	if err := g.Wait(); err != nil {
		return poop.Chain(err)
	}

	return nil
}
