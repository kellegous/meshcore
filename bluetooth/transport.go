package bluetooth

import (
	"tinygo.org/x/bluetooth"

	"github.com/kellegous/meshcore"
)

type Transport struct {
	device   bluetooth.Device
	toDevice bluetooth.DeviceCharacteristic
	notifier *meshcore.Notifier
}

var _ meshcore.Transport = (*Transport)(nil)

func (t *Transport) Write(p []byte) (n int, err error) {
	return t.toDevice.Write(p)
}

func (t *Transport) Disconnect() error {
	return t.device.Disconnect()
}

func (t *Transport) Notifier() *meshcore.Notifier {
	return t.notifier
}
