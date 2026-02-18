package serial

import (
	"context"
	"fmt"

	"github.com/kellegous/meshcore"
	"github.com/kellegous/poop"
	"go.bug.st/serial"
)

func Connect(ctx context.Context, address string) (*meshcore.Conn, error) {
	port, err := serial.Open(address, &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		StopBits: serial.OneStopBit,
		Parity:   serial.NoParity,
	})
	if err != nil {
		return nil, poop.Chain(err)
	}

	notifier := meshcore.NewNotifier()

	// TODO(kellegous): This needs to become a part of the
	// transport interface.
	onRecvError := func(err error) {
		panic(err)
	}

	go func() {
		defer port.Close()

		var buf [1024]byte
		for {
			n, err := port.Read(buf[:])
			if err != nil {
				onRecvError(err)
				return
			} else if n == 0 {
				onRecvError(poop.New("read 0 bytes"))
				return
			}

			fmt.Println(buf[:n])

			code := meshcore.NotificationCode(buf[0])
			notifier.Notify(code, buf[1:n])
		}
	}()

	return meshcore.NewConnection(&tx{
		port:     port,
		notifier: notifier,
	}), nil
}
