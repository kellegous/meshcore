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

// singleCallUnsubFn is an optmization that makes defensive unsubscribing
// more efficient. A caller can call the func with defer and also call it
// inline with no concerns about performance or safety.
func singleCallUnsubFn(n *Notifier, code NotificationCode, l *listener) func() {
	var hasBeenCalled bool
	return func() {
		if hasBeenCalled {
			return
		}
		n.unsubscribe(code, l)
		hasBeenCalled = true
	}
}

func (n *Notifier) Subscribe(code NotificationCode, fn func(data []byte)) func() {
	n.lock.Lock()
	defer n.lock.Unlock()
	lr := &listener{fn: fn}
	n.listeners[code] = append(n.listeners[code], lr)
	return singleCallUnsubFn(n, code, lr)
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
