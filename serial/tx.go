package serial

import (
	"bytes"
	"encoding/binary"
	"sync/atomic"

	"github.com/kellegous/meshcore"
	"github.com/kellegous/poop"
	"go.bug.st/serial"
)

type tx struct {
	port           serial.Port
	isDisconnected atomic.Bool
	*meshcore.NotificationCenter
	opts *ConnectOptions
}

var _ meshcore.Transport = (*tx)(nil)

func (t *tx) Write(p []byte) (int, error) {
	var buf bytes.Buffer
	buf.WriteByte(outgoingFrameType)
	binary.Write(&buf, binary.LittleEndian, uint16(len(p)))
	buf.Write(p)

	if nf := t.opts.onSend; nf != nil && len(p) > 0 {
		nf(meshcore.CommandCode(p[0]), p[1:])
	}

	n, err := t.port.Write(buf.Bytes())
	if err != nil {
		return 0, poop.Chain(err)
	}
	return n - 3, nil
}

func (t *tx) Disconnect() error {
	t.isDisconnected.Store(true)
	return t.port.Close()
}
