package bluetooth

import (
	"tinygo.org/x/bluetooth"

	"github.com/kellegous/meshcore"
)

type tx struct {
	device   bluetooth.Device
	toDevice bluetooth.DeviceCharacteristic
	*meshcore.Notifier
}

var _ meshcore.Transport = (*tx)(nil)

func (t *tx) Write(p []byte) (n int, err error) {
	return t.toDevice.Write(p)
}

func (t *tx) Disconnect() error {
	return t.device.Disconnect()
}
