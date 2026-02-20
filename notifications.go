package meshcore

import (
	"context"
	"iter"
	"slices"
	"sync"

	"github.com/kellegous/poop"
)

type Notification interface {
	NotificationCode() NotificationCode
}

type notificationData struct {
	Notification Notification
	Error        error
}

type subscription struct {
	ch chan *notificationData
}

func (s *subscription) cancel() {
	close(s.ch)
}

func (s *subscription) publish(notification Notification, error error) {
	s.ch <- &notificationData{Notification: notification, Error: error}
}

type Notifications struct {
	lck     sync.RWMutex
	streams map[NotificationCode][]*subscription
}

func NewEventStream() *Notifications {
	return &Notifications{
		streams: make(map[NotificationCode][]*subscription),
	}
}

func (e *Notifications) register(codes []NotificationCode, s *subscription) func() {
	e.lck.Lock()
	defer e.lck.Unlock()

	for _, code := range codes {
		e.streams[code] = append(e.streams[code], s)
	}

	return func() {
		e.lck.Lock()
		defer e.lck.Unlock()

		defer s.cancel()

		for _, code := range codes {
			e.streams[code] = slices.DeleteFunc(e.streams[code], func(ss *subscription) bool {
				return s == ss
			})
		}
	}
}

func (e *Notifications) Subscribe(
	ctx context.Context,
	codes ...NotificationCode,
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

func (e *Notifications) Publish(code NotificationCode, data []byte) {
	e.lck.RLock()
	defer e.lck.RUnlock()

	streams := e.streams[code]
	if len(streams) == 0 {
		return
	}

	notification, err := decodeNotification(code, data)
	for _, s := range streams {
		s.publish(notification, err)
	}
}

func decodeNotification(code NotificationCode, data []byte) (Notification, error) {
	return nil, poop.Newf("not implemented")
}
