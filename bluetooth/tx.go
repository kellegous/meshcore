package bluetooth

import (
	"tinygo.org/x/bluetooth"

	"github.com/kellegous/meshcore"
)

type tx struct {
	device   bluetooth.Device
	toDevice bluetooth.DeviceCharacteristic
	*meshcore.NotificationCenter
	opts *ConnectOptions
}

var _ meshcore.Transport = (*tx)(nil)

func (t *tx) Write(p []byte) (n int, err error) {
	if nf := t.opts.onSend; nf != nil && len(p) > 0 {
		nf(meshcore.CommandCode(p[0]), p[1:])
	}

	return t.toDevice.Write(p)
}

func (t *tx) Disconnect() error {
	return t.device.Disconnect()
}
