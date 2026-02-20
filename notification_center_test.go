package meshcore

import (
	"errors"
	"iter"
	"testing"
)

func TestShutdown(t *testing.T) {
	t.Run("no subscriptions", func(t *testing.T) {
		nc := NewNotificationCenter()
		nc.Shutdown()
	})

	t.Run("shdown cancels subscriptions", func(t *testing.T) {
		nc := NewNotificationCenter()

		nextA, doneA := iter.Pull2(nc.Subscribe(t.Context(), NotificationTypeOk))
		defer doneA()

		nextB, doneB := iter.Pull2(nc.Subscribe(t.Context(), NotificationTypeOk))
		defer doneB()

		nc.Shutdown()

		if _, err, _ := nextA(); !errors.Is(err, ErrShutdown) {
			t.Fatalf("expected %v, got %v", ErrShutdown, err)
		}

		if _, err, _ := nextB(); !errors.Is(err, ErrShutdown) {
			t.Fatalf("expected %v, got %v", ErrShutdown, err)
		}
	})
}
