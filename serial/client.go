package serial

import (
	"context"
	"encoding/binary"
	"io"

	"github.com/kellegous/meshcore"
	"github.com/kellegous/poop"
	"go.bug.st/serial"
)

const (
	incomingFrameType = 0x3e // ">"
	outgoingFrameType = 0x3c // "<"
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

	transport := &tx{
		port:     port,
		notifier: notifier,
	}

	// TODO(kellegous): This needs to become a part of the
	// transport interface.
	onRecvError := func(err error) {
		// suppress errors if the transport is disconnected
		if transport.isDisconnected.Load() {
			return
		}
		panic(err)
	}

	go func() {
		defer port.Close()

		for {
			// the js library does a lot of nonsense that seems
			// unsound. For instance, if the type is wrong, it
			// reads the next byte. That seems destined to fail.
			var hdr header
			if err := hdr.readFrom(port); err != nil {
				onRecvError(err)
				return
			}

			if hdr.Length == 0 {
				onRecvError(poop.New("frame length is 0"))
				return
			}

			data := make([]byte, hdr.Length)
			if _, err := io.ReadFull(port, data); err != nil {
				onRecvError(err)
				return
			}

			code := meshcore.NotificationCode(data[0])
			notifier.Notify(code, data[1:])
		}
	}()

	return meshcore.NewConnection(transport), nil
}

type header struct {
	Type   byte
	Length uint16
}

func (h *header) readFrom(r io.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, &h.Type); err != nil {
		return poop.Chain(err)
	}

	// TODO(kellegous): why would we receive an outgoing frame?
	if h.Type != incomingFrameType && h.Type != outgoingFrameType {
		return poop.Newf("invalid frame type: %d", h.Type)
	}

	if err := binary.Read(r, binary.LittleEndian, &h.Length); err != nil {
		return poop.Chain(err)
	}

	return nil
}
