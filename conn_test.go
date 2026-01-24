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

func (c *Controller) Notify(code NotificationCode, data []byte) {
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

func TestDeviceQuery(t *testing.T) {
	expected := &DeviceInfo{
		FirmwareVersion:   3,
		FirmwareBuildDate: "2024-01-15",
		ManufacturerModel: "lilygo-t-echo",
	}

	appTargetVer := byte(42)

	controller := DoCommand(func(conn *Conn) {
		deviceInfo, err := conn.DeviceQuery(t.Context(), appTargetVer)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(deviceInfo, expected) {
			t.Fatalf("expected %s, got %s",
				describe(expected),
				describe(deviceInfo),
			)
		}
	})

	if err := ValidateBytes(
		controller.Recv(),
		Command(CommandDeviceQuery),
		Byte(appTargetVer),
	); err != nil {
		t.Fatal(err)
	}

	controller.Notify(ResponseDeviceInfo, BytesFrom(
		Byte(byte(expected.FirmwareVersion)),
		Bytes(0, 0, 0, 0, 0, 0), // reserved 6 bytes
		CString(expected.FirmwareBuildDate, 12),
		String(expected.ManufacturerModel),
	))

	controller.Wait()
}

func TestSyncNextMessage(t *testing.T) {
	fromContact := &ContactMessage{
		PubKeyPrefix: [6]byte{1, 2, 3, 4, 5, 6},
		PathLen:      1,
		TextType:     TextTypePlain,
		SenderTime:   time.Unix(100, 0),
		Text:         "hello",
	}

	fromChannel := &ChannelMessage{
		ChannelIndex: 1,
		PathLen:      1,
		TextType:     TextTypePlain,
		SenderTime:   time.Unix(100, 0),
		Text:         "hello",
	}

	t.Run("from contact", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			message, err := conn.SyncNextMessage(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(message, fromContact) {
				t.Fatalf("expected %s, got %s",
					describe(fromContact),
					describe(message),
				)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSyncNextMessage),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(
			ResponseContactMsgRecv,
			BytesFrom(
				Bytes(fromContact.PubKeyPrefix[:]...),
				Byte(fromContact.PathLen),
				Byte(byte(fromContact.TextType)),
				Time(fromContact.SenderTime, binary.LittleEndian),
				String(fromContact.Text),
			))

		controller.Wait()
	})

	t.Run("from channel", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			message, err := conn.SyncNextMessage(t.Context())
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(message, fromChannel) {
				t.Fatalf("expected %s, got %s",
					describe(fromChannel),
					describe(message),
				)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSyncNextMessage),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(
			ResponseChannelMsgRecv,
			BytesFrom(
				Byte(fromChannel.ChannelIndex),
				Byte(fromChannel.PathLen),
				Byte(byte(fromChannel.TextType)),
				Time(fromChannel.SenderTime, binary.LittleEndian),
				String(fromChannel.Text),
			))

		controller.Wait()
	})

	t.Run("no more messages", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			message, err := conn.SyncNextMessage(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if message != nil {
				t.Fatalf("expected nil message, got %s",
					describe(message),
				)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSyncNextMessage),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(ResponseNoMoreMessages, nil)

		controller.Wait()
	})

	// TODO(kellegous): test error cases
}

func TestSendAdvert(t *testing.T) {
	t.Run("zero hop", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.SendAdvert(t.Context(), SelfAdvertTypeZeroHop); err != nil {
				t.Fatal(err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSendSelfAdvert),
			Byte(byte(SelfAdvertTypeZeroHop)),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(ResponseOk, nil)

		controller.Wait()
	})

	t.Run("flood", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.SendAdvert(t.Context(), SelfAdvertTypeFlood); err != nil {
				t.Fatal(err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSendSelfAdvert),
			Byte(byte(SelfAdvertTypeFlood)),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(ResponseOk, nil)

		controller.Wait()
	})

	// TODO(kellegous): test error cases
}
