package meshcore

import (
	"context"
	"errors"
	"iter"
	"slices"
	"sync"
	"sync/atomic"
)

var ErrShutdown = errors.New("shutdown")

type notificationData struct {
	Notification Notification
	Error        error
}

type subscription struct {
	isClosed atomic.Bool
	ch       chan *notificationData
}

func (s *subscription) cancel() {
	if s.isClosed.CompareAndSwap(false, true) {
		close(s.ch)
	}
}

func (s *subscription) publish(notification Notification, error error) {
	s.ch <- &notificationData{Notification: notification, Error: error}
}

type NotificationCenter struct {
	lck           sync.RWMutex
	subscriptions map[ResponseCode][]*subscription
}

func NewNotificationCenter() *NotificationCenter {
	return &NotificationCenter{
		subscriptions: make(map[ResponseCode][]*subscription),
	}
}

func (e *NotificationCenter) register(codes []ResponseCode, s *subscription) func() {
	e.lck.Lock()
	defer e.lck.Unlock()

	for _, code := range codes {
		e.subscriptions[code] = append(e.subscriptions[code], s)
	}

	return func() {
		e.lck.Lock()
		defer e.lck.Unlock()

		defer s.cancel()

		for _, code := range codes {
			e.subscriptions[code] = slices.DeleteFunc(e.subscriptions[code], func(ss *subscription) bool {
				return s == ss
			})
		}
	}
}

func (e *NotificationCenter) Subscribe(
	ctx context.Context,
	codes ...ResponseCode,
) iter.Seq2[Notification, error] {
	s := &subscription{
		ch: make(chan *notificationData),
	}

	release := e.register(codes, s)

	return func(yield func(Notification, error) bool) {
		defer release()

		for {
			select {
			case data, ok := <-s.ch:
				if !ok {
					yield(nil, ErrShutdown)
					return
				}
				if !yield(data.Notification, data.Error) {
					return
				}
			case <-ctx.Done():
				yield(nil, ctx.Err())
				return
			}
		}
	}
}

func (e *NotificationCenter) Publish(code ResponseCode, data []byte) {
	e.lck.RLock()
	defer e.lck.RUnlock()

	streams := e.subscriptions[code]
	if len(streams) == 0 {
		return
	}

	notification, err := readNotification(code, data)
	for _, s := range streams {
		s.publish(notification, err)
	}
}

func (e *NotificationCenter) Shutdown() {
	e.lck.Lock()
	defer e.lck.Unlock()

	for _, subs := range e.subscriptions {
		for _, sub := range subs {
			sub.cancel()
		}
	}

	e.subscriptions = make(map[ResponseCode][]*subscription)
}
