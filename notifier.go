package meshcore

import (
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
