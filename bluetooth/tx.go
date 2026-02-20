package bluetooth

import (
	"context"
	"iter"

	"tinygo.org/x/bluetooth"

	"github.com/kellegous/meshcore"
)

type tx struct {
	device   bluetooth.Device
	toDevice bluetooth.DeviceCharacteristic
	*meshcore.Notifier
	notificationCenter *meshcore.NotificationCenter
}

var _ meshcore.Transport = (*tx)(nil)

func (t *tx) Write(p []byte) (n int, err error) {
	return t.toDevice.Write(p)
}

func (t *tx) Disconnect() error {
	return t.device.Disconnect()
}

func (t *tx) Subscribe2(ctx context.Context, codes ...meshcore.NotificationCode) iter.Seq2[meshcore.Notification, error] {
	return t.notificationCenter.Subscribe(ctx, codes...)
}
