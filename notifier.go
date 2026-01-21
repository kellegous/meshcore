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
	listeners map[ResponseCode][]*listener
}

func NewNotifier() *Notifier {
	return &Notifier{
		listeners: make(map[ResponseCode][]*listener),
	}
}

func (n *Notifier) Subscribe(code ResponseCode, fn func(data []byte)) func() {
	n.lock.Lock()
	defer n.lock.Unlock()
	lr := &listener{fn: fn}
	n.listeners[code] = append(n.listeners[code], lr)
	return func() {
		n.unsubscribe(code, lr)
	}
}

func (n *Notifier) unsubscribe(code ResponseCode, lr *listener) {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.listeners[code] = slices.DeleteFunc(n.listeners[code], func(l *listener) bool {
		return l == lr
	})
}

func (n *Notifier) Notify(code ResponseCode, data []byte) {
	n.lock.RLock()
	defer n.lock.RUnlock()
	for _, l := range n.listeners[code] {
		l.fn(data)
	}
}
