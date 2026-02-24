package serial

import "github.com/kellegous/meshcore"

type ConnectOptions struct {
	onRecv func(code meshcore.NotificationCode, data []byte)
	onSend func(code meshcore.CommandCode, data []byte)
}

type ConnectOption func(*ConnectOptions)

func OnRecv(fn func(code meshcore.NotificationCode, data []byte)) ConnectOption {
	return func(opts *ConnectOptions) {
		opts.onRecv = fn
	}
}

func OnSend(fn func(code meshcore.CommandCode, data []byte)) ConnectOption {
	return func(opts *ConnectOptions) {
		opts.onSend = fn
	}
}
