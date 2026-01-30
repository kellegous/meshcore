package meshcore

import (
	"context"
	"slices"
	"sync"
)

type listener struct {
	fn func(data []byte)
}

type Notifier struct {
	lock      sync.RWMutex
	listeners map[NotificationCode][]*listener
}

func NewNotifier() *Notifier {
	return &Notifier{
		listeners: make(map[NotificationCode][]*listener),
	}
}

func (n *Notifier) waitFor(
	ctx context.Context,
	fn func(NotificationCode, []byte),
	codes ...NotificationCode,
) error {
	unsubs := make([]func(), len(codes))
	ch := make(chan struct{})
	for _, code := range codes {
		unsubs = append(unsubs, n.Subscribe(code, func(data []byte) {
			fn(code, data)
			close(ch)
		}))
	}

	defer func() {
		for _, unsub := range unsubs {
			unsub()
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ch:
		return nil
	}
}

func (n *Notifier) Subscribe(code NotificationCode, fn func(data []byte)) func() {
	n.lock.Lock()
	defer n.lock.Unlock()
	lr := &listener{fn: fn}
	n.listeners[code] = append(n.listeners[code], lr)
	return func() {
		n.unsubscribe(code, lr)
	}
}

func (n *Notifier) unsubscribe(code NotificationCode, lr *listener) {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.listeners[code] = slices.DeleteFunc(n.listeners[code], func(l *listener) bool {
		return l == lr
	})
}

func (n *Notifier) Notify(code NotificationCode, data []byte) {
	n.lock.RLock()
	defer n.lock.RUnlock()
	for _, l := range n.listeners[code] {
		l.fn(data)
	}
}
