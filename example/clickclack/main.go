package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"iter"
	"os"
	"strings"
	"time"

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

	if err := func() error {
		narrator.Printf("%s\n", center("Deleting Channels", width))
		g, ctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			return resetChannels(ctx, click)
		})
		g.Go(func() error {
			return resetChannels(ctx, clack)
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

	if err := func() error {
		narrator.Printf("%s\n", center("Getting & Setting Time", width))

		if err := getAndSetTime(ctx, click); err != nil {
			return poop.Chain(err)
		}

		if err := getAndSetTime(ctx, clack); err != nil {
			return poop.Chain(err)
		}

		return nil
	}(); err != nil {
		return poop.Chain(err)
	}

	if err := func() error {
		narrator.Printf("%s\n", center("Sending message to public channel", width))

		if err := exchangeChannelMessage(ctx, click, clack, 0); err != nil {
			return poop.Chain(err)
		}

		if err := exchangeChannelMessage(ctx, clack, click, 0); err != nil {
			return poop.Chain(err)
		}

		return nil
	}(); err != nil {
		return poop.Chain(err)
	}

	if err := func() error {
		narrator.Printf("%s\n", center("Sending message to contact", width))

		if err := exchangeContactMessage(ctx, click, clack); err != nil {
			return poop.Chain(err)
		}

		if err := exchangeContactMessage(ctx, clack, click); err != nil {
			return poop.Chain(err)
		}

		return nil
	}(); err != nil {
		return poop.Chain(err)
	}

	if err := func() error {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		narrator.Printf("%s\n", center("Sending message to secret channel", width))

		var key [16]byte
		if _, err := rand.Read(key[:]); err != nil {
			return poop.Chain(err)
		}

		channel := &meshcore.ChannelInfo{
			Index:  1,
			Name:   "Super Secret",
			Secret: key[:],
		}

		click.Printf("Adding secret channel")
		if err := click.SetChannel(ctx, channel); err != nil {
			return poop.Chain(err)
		}

		clack.Printf("Adding secret channel")
		if err := clack.SetChannel(ctx, channel); err != nil {
			return poop.Chain(err)
		}

		if err := exchangeChannelMessage(ctx, click, clack, channel.Index); err != nil {
			return poop.Chain(err)
		}

		time.Sleep(1 * time.Second)

		if err := exchangeChannelMessage(ctx, clack, click, channel.Index); err != nil {
			return poop.Chain(err)
		}

		return nil
	}(); err != nil {
		return poop.Chain(err)
	}

	// set advert name

	// send a contact message

	// sign a message

	// create a new channel & send message to it

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

func exchangeContactMessage(
	ctx context.Context,
	sender *Actor,
	receiver *Actor,
) error {
	g, bgCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		for notif, err := range receiver.Notifications(bgCtx, meshcore.NotificationTypeMsgWaiting) {
			if err != nil {
				return poop.Chain(err)
			}

			msgWaiting, ok := notif.(*meshcore.MsgWaitingNotification)
			if !ok {
				return poop.Newf("unexpected sent notification: %T", msgWaiting)
			}

			receiver.Printf("message waiting")

			msg, err := receiver.SyncNextMessage(ctx)
			if err != nil {
				return poop.Chain(err)
			}

			contactMsg := msg.FromContact()
			if contactMsg == nil {
				return poop.New("no contact message received")
			}

			receiver.Printf(
				"received message from %s with the text: %s",
				hex.EncodeToString(contactMsg.PubKeyPrefix[:]),
				contactMsg.Text,
			)

			break
		}
		return nil
	})

	sender.Printf("sending message to %s", hex.EncodeToString(receiver.Info.PublicKey.Prefix(6)))
	if _, err := sender.SendTextMessage(ctx, &receiver.Info.PublicKey, "Hello", meshcore.TextTypePlain); err != nil {
		return poop.Chain(err)
	}

	if err := g.Wait(); err != nil {
		return poop.Chain(err)
	}

	return nil
}

func exchangeChannelMessage(
	ctx context.Context,
	sender *Actor,
	receiver *Actor,
	channelIndex byte,
) error {
	next, done := iter.Pull2(
		receiver.Notifications(ctx, meshcore.NotificationTypeMsgWaiting),
	)
	defer done()

	g, bgCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		notif, err, _ := next()
		if err != nil {
			return poop.Chain(err)
		}

		if _, ok := notif.(*meshcore.MsgWaitingNotification); !ok {
			return poop.Newf("unexpected notification: %T", notif)
		}

		msg, err := receiver.SyncNextMessage(bgCtx)
		if err != nil {
			return poop.Chain(err)
		}

		if msg == nil {
			return poop.New("no message received")
		}

		channelMsg := msg.FromChannel()
		if channelMsg == nil {
			return poop.New("no channel message received")
		}

		receiver.Printf(
			"received message from channel #%d with the text: %s",
			channelMsg.ChannelIndex,
			channelMsg.Text,
		)

		return nil
	})

	sender.Printf("sending message to channnel #%d", channelIndex)
	if err := sender.SendChannelTextMessage(ctx, channelIndex, "Hello", meshcore.TextTypePlain); err != nil {
		return poop.Chain(err)
	}

	if err := g.Wait(); err != nil {
		return poop.Chain(err)
	}

	return nil
}

func getAndSetTime(
	ctx context.Context,
	actor *Actor,
) error {
	t, err := actor.GetDeviceTime(ctx)
	if err != nil {
		return poop.Chain(err)
	}
	actor.Printf("device time: %s", t.Format(time.RFC3339))

	now := time.Now()
	if err := actor.SetDeviceTime(ctx, now); err != nil {
		return poop.Chain(err)
	}
	actor.Printf("set device time to %s", now.Format(time.RFC3339))

	return nil
}

func resetContacts(
	ctx context.Context,
	actor *Actor,
) error {
	contacts, err := actor.GetContacts(ctx, nil)
	if err != nil {
		return poop.Chain(err)
	}

	for _, contact := range contacts {
		if err := actor.RemoveContact(ctx, &contact.PublicKey); err != nil {
			return poop.Chain(err)
		}
	}

	actor.Printf("removed %d contacts", len(contacts))

	return nil
}

func resetChannels(
	ctx context.Context,
	actor *Actor,
) error {
	channels, err := actor.GetChannels(ctx)
	if err != nil {
		return poop.Chain(err)
	}

	var count int
	for _, channel := range channels {
		if channel.Index == 0 || channel.Name == "" {
			continue
		}
		if err := actor.DeleteChannel(ctx, channel.Index); err != nil {
			return poop.Chain(err)
		}
		count++
	}

	actor.Printf("removed %d channels", count)

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
