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
	listeners map[byte][]*listener
}

func NewNotifier() *Notifier {
	return &Notifier{
		listeners: make(map[byte][]*listener),
	}
}

func (n *Notifier) Subscribe(code EventCode, fn func(data []byte)) func() {
	n.lock.Lock()
	defer n.lock.Unlock()
	lr := &listener{fn: fn}
	n.listeners[code.event()] = append(n.listeners[code.event()], lr)
	return func() {
		n.unsubscribe(code, lr)
	}
}

func (n *Notifier) unsubscribe(code EventCode, lr *listener) {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.listeners[code.event()] = slices.DeleteFunc(n.listeners[code.event()], func(l *listener) bool {
		return l == lr
	})
}

func (n *Notifier) Notify(code EventCode, data []byte) {
	n.lock.RLock()
	defer n.lock.RUnlock()
	for _, l := range n.listeners[code.event()] {
		l.fn(data)
	}
}
