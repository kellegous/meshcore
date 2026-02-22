package bluetooth

import "github.com/kellegous/meshcore"

type ConnectOptions struct {
	onNotification func(code meshcore.ResponseCode, data []byte)
}

type ConnectOption func(*ConnectOptions)

// WithNotificationCallback sets the callback for notifications that is mostly used
// for debugging purposes.
func WithNotificationCallback(fn func(code meshcore.ResponseCode, data []byte)) ConnectOption {
	return func(opts *ConnectOptions) {
		opts.onNotification = fn
	}
}
