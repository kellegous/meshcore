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

	go func() {
		var buf [1024]byte
		for {
			n, err := port.Read(buf[:])
			if err != nil {
				return
			}

			fmt.Println(buf[:n])
		}
	}()
	// TODO(kellegous): Need to start notifying routine
	return meshcore.NewConnection(&tx{
		port:     port,
		notifier: notifier,
	}), nil
}
