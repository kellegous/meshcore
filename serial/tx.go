package serial

import (
	"github.com/kellegous/meshcore"
	"go.bug.st/serial"
)

type tx struct {
	port     serial.Port
	notifier *meshcore.Notifier
}

var _ meshcore.Transport = (*tx)(nil)

func (t *tx) Write(p []byte) (n int, err error) {
	return t.port.Write(p)
}

func (t *tx) Disconnect() error {
	return t.port.Close()
}

func (t *tx) Notifier() *meshcore.Notifier {
	return t.notifier
}
