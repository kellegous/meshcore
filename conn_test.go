package meshcore

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

type fakeTransport struct {
	ch       chan []byte
	done     chan struct{}
	notifier *Notifier
}

var _ Transport = (*fakeTransport)(nil)

func (t *fakeTransport) Write(p []byte) (n int, err error) {
	t.ch <- p
	return len(p), nil
}

func (t *fakeTransport) Disconnect() error {
	return nil
}

func (t *fakeTransport) Notifier() *Notifier {
	return t.notifier
}

func DoCommand(
	op func(conn *Conn),
) *Controller {
	tx := &fakeTransport{
		ch:       make(chan []byte),
		done:     make(chan struct{}),
		notifier: NewNotifier(),
	}
	go func() {
		defer close(tx.done)
		op(NewConnection(tx))
	}()
	return &Controller{
		tx: tx,
	}
}

type Controller struct {
	tx *fakeTransport
}

func (c *Controller) Notify(code EventCode, data []byte) {
	c.tx.notifier.Notify(code, data)
}

func (c *Controller) Recv() []byte {
	return <-c.tx.ch
}

func (c *Controller) Wait() {
	<-c.tx.done
}

func fakePublicKey(id byte) *PublicKey {
	key := [32]byte{}
	key[0] = id
	return &PublicKey{key: key}
}

func fakeBytes(n int, fn func(i int) byte) []byte {
	bs := make([]byte, n)
	for i := 0; i < n; i++ {
		bs[i] = fn(i)
	}
	return bs
}

func describe(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(b)
}

func TestGetContacts(t *testing.T) {
	contactA := &Contact{
		PublicKey:  *fakePublicKey(1),
		Type:       1,
		Flags:      2,
		OutPath:    []byte{1, 2, 3},
		AdvName:    "A",
		LastAdvert: time.Unix(100, 0),
		AdvLat:     37.7,
		AdvLon:     -122.4,
		LastMod:    time.Unix(101, 0),
	}
	contactB := &Contact{
		PublicKey:  *fakePublicKey(2),
		Type:       1,
		Flags:      2,
		OutPath:    []byte{1, 2, 3},
		AdvName:    "B",
		LastAdvert: time.Unix(200, 0),
		AdvLat:     37.7,
		AdvLon:     -122.4,
		LastMod:    time.Unix(201, 0),
	}

	t.Run("default options", func(t *testing.T) {
		expected := []*Contact{contactA, contactB}

		controller := DoCommand(func(conn *Conn) {
			contacts, err := conn.GetContacts(t.Context(), nil)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(contacts, expected) {
				t.Fatalf("expected %s, got %s",
					describe(expected),
					describe(contacts),
				)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandGetContacts),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(ResponseContactsStart, nil)
		for _, contact := range expected {
			var buf bytes.Buffer
			contact.writeTo(&buf)
			controller.Notify(ResponseContact, buf.Bytes())
		}

		controller.Notify(ResponseEndOfContacts, nil)

		controller.Wait()
	})
	t.Run("using since", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			contacts, err := conn.GetContacts(t.Context(), &GetContactsOptions{
				Since: time.Unix(100, 0),
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(contacts) != 0 {
				t.Fatalf("expected 0 contacts, got %d", len(contacts))
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandGetContacts),
			Time(time.Unix(100, 0), binary.LittleEndian),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(ResponseContactsStart, nil)
		controller.Notify(ResponseEndOfContacts, nil)

		controller.Wait()
	})
}

func TestGetTelemetry(t *testing.T) {
	key := fakePublicKey(42)
	expected := &TelemetryResponse{
		pubKeyPrefix:  [6]byte{42, 0, 0, 0, 0, 0},
		LPPSensorData: []byte{1, 2, 3},
	}

	controller := DoCommand(func(conn *Conn) {
		telemetry, err := conn.GetTelemetry(t.Context(), key)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(telemetry, expected) {
			t.Fatalf("expected %s, got %s",
				describe(expected),
				describe(telemetry),
			)
		}
	})

	if err := ValidateBytes(
		controller.Recv(),
		Command(CommandSendTelemetryReq),
		Bytes(0, 0, 0),
		Bytes(key.Bytes()...),
	); err != nil {
		t.Fatal(err)
	}

	controller.Notify(PushTelemetryResponse, BytesFrom(
		Byte(0),
		Bytes(key.Prefix(6)...),
		Bytes(1, 2, 3),
	))

	controller.Wait()
}

func TestSetChannel(t *testing.T) {
	channel := &ChannelInfo{
		Index: 3,
		Name:  "chan",
		Secret: fakeBytes(16, func(i int) byte {
			return byte(i + 1)
		}),
	}

	controller := DoCommand(func(conn *Conn) {
		if err := conn.SetChannel(t.Context(), channel); err != nil {
			t.Fatal(err)
		}
	})

	if err := ValidateBytes(
		controller.Recv(),
		Command(CommandSetChannel),
		Byte(channel.Index),
		CString(channel.Name, 32),
		Bytes(channel.Secret...),
	); err != nil {
		t.Fatal(err)
	}

	controller.Notify(ResponseOk, nil)

	controller.Wait()
}
