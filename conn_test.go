package meshcore

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"iter"
	"reflect"
	"testing"
	"time"

	"github.com/kellegous/poop"
)

type fakeTransport struct {
	ch   chan []byte
	done chan struct{}
	*Notifier
	notificationCenter *NotificationCenter
}

var _ Transport = (*fakeTransport)(nil)

func (t *fakeTransport) Write(p []byte) (n int, err error) {
	t.ch <- p
	return len(p), nil
}

func (t *fakeTransport) Disconnect() error {
	return nil
}

func (t *fakeTransport) Subscribe2(ctx context.Context, codes ...NotificationCode) iter.Seq2[Notification, error] {
	return t.notificationCenter.Subscribe(ctx, codes...)
}

func DoCommand(
	op func(conn *Conn),
) *Controller {
	tx := &fakeTransport{
		ch:                 make(chan []byte, 1),
		done:               make(chan struct{}),
		Notifier:           NewNotifier(),
		notificationCenter: NewNotificationCenter(),
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
	c.tx.Notify(code, data)
	c.tx.notificationCenter.Publish(code, data)
}

func (c *Controller) Recv() []byte {
	return <-c.tx.ch
}

func (c *Controller) Wait() {
	<-c.tx.done
}

func fakePublicKey(id byte) PublicKey {
	key := [32]byte{}
	key[0] = id
	return PublicKey{key: key}
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
		PublicKey:  fakePublicKey(1),
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
		PublicKey:  fakePublicKey(2),
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

		controller.Notify(NotificationTypeContactsStart, nil)
		for _, contact := range expected {
			var buf bytes.Buffer
			contact.writeTo(&buf)
			controller.Notify(NotificationTypeContact, buf.Bytes())
		}

		controller.Notify(NotificationTypeEndOfContacts, nil)

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

		controller.Notify(NotificationTypeContactsStart, nil)
		controller.Notify(NotificationTypeEndOfContacts, nil)

		controller.Wait()
	})
}

func TestAddOrUpdateContact(t *testing.T) {
	contact := &Contact{
		PublicKey:  fakePublicKey(42),
		Type:       ContactTypeChat,
		Flags:      0,
		OutPath:    []byte{1, 2, 3},
		AdvName:    "test",
		LastAdvert: time.Unix(100, 0),
		AdvLat:     37.7,
		AdvLon:     -122.4,
		LastMod:    time.Unix(101, 0),
	}

	contactBytes := func() []byte {
		var buf bytes.Buffer
		contact.writeTo(&buf)
		return buf.Bytes()
	}()

	t.Run("success", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.AddOrUpdateContact(t.Context(), contact); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(contact, contact) {
				t.Fatalf("expected %s, got %s",
					describe(contact),
					describe(contact),
				)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandAddUpdateContact),
			Bytes(contactBytes...),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeOk, nil)

		controller.Wait()
	})

	t.Run("error", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.AddOrUpdateContact(t.Context(), contact); err == nil || err.Error() != "error: 5 (file io error)" {
				t.Fatalf("expected error: error: 5 (file io error), got %v", err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandAddUpdateContact),
			Bytes(contactBytes...),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeErr, BytesFrom(Byte(byte(ErrorCodeFileIOError))))

		controller.Wait()
	})
}

func TestRemoveContact(t *testing.T) {
	key := fakePublicKey(42)

	t.Run("success", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			// todo: key should be a value?
			if err := conn.RemoveContact(t.Context(), &key); err != nil {
				t.Fatal(err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandRemoveContact),
			Bytes(key.Bytes()...),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeOk, nil)

		controller.Wait()
	})

	t.Run("error", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.RemoveContact(t.Context(), &key); err == nil || err.Error() != "error: 5 (file io error)" {
				t.Fatalf("expected error: error: 5 (file io error), got %v", err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandRemoveContact),
			Bytes(key.Bytes()...),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeErr, BytesFrom(Byte(byte(ErrorCodeFileIOError))))

		controller.Wait()
	})
}

func TestGetDeviceTime(t *testing.T) {
	expected := time.Unix(100, 0)

	t.Run("success", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			time, err := conn.GetDeviceTime(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(time, expected) {
				t.Fatalf("expected %s, got %s",
					describe(expected),
					describe(time),
				)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandGetDeviceTime),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeCurrTime, BytesFrom(Time(expected, binary.LittleEndian)))

		controller.Wait()
	})

	t.Run("error", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if _, err := conn.GetDeviceTime(t.Context()); err == nil || err.Error() != "error: 5 (file io error)" {
				t.Fatalf("expected error: error: 5 (file io error), got %v", err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandGetDeviceTime),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeErr, BytesFrom(Byte(byte(ErrorCodeFileIOError))))

		controller.Wait()
	})
}

func TestGetBatteryVoltage(t *testing.T) {
	expected := uint16(100)

	t.Run("success", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			voltage, err := conn.GetBatteryVoltage(t.Context())
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(voltage, expected) {
				t.Fatalf("expected %d, got %d", expected, voltage)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandGetBatteryVoltage),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeBatteryVoltage, BytesFrom(Uint16(expected, binary.LittleEndian)))

		controller.Wait()
	})

	t.Run("error", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if _, err := conn.GetBatteryVoltage(t.Context()); err == nil || err.Error() != "error: 5 (file io error)" {
				t.Fatalf("expected error: error: 5 (file io error), got %v", err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandGetBatteryVoltage),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeErr, BytesFrom(Byte(byte(ErrorCodeFileIOError))))

		controller.Wait()
	})
}

func TestSendTextMessage(t *testing.T) {
	recipient := fakePublicKey(42)
	message := "hello"
	textType := TextTypePlain
	expected := &SentResponse{
		Result:         0,
		ExpectedAckCRC: 1234567890,
		EstTimeout:     1000,
	}

	t.Run("success", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			sr, err := conn.SendTextMessage(t.Context(), &recipient, message, textType)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(sr, expected) {
				t.Fatalf("expected %s, got %s",
					describe(expected),
					describe(sr),
				)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSendTxtMsg),
			Byte(byte(textType)),
			Byte(0),
			AnyBytes(4), /// time = now
			Bytes(recipient.Prefix(6)...),
			String(message),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeSent, BytesFrom(
			Byte(0),
			Uint32(expected.ExpectedAckCRC, binary.LittleEndian),
			Uint32(expected.EstTimeout, binary.LittleEndian),
		))

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
		telemetry, err := conn.GetTelemetry(t.Context(), &key)
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

	controller.Notify(NotificationTypeTelemetryResponse, BytesFrom(
		Byte(0),
		Bytes(key.Prefix(6)...),
		Bytes(1, 2, 3),
	))

	controller.Wait()
}

func TestGetChannel(t *testing.T) {
	idx := uint8(3)
	expected := &ChannelInfo{
		Index: 3,
		Name:  "chan",
		Secret: fakeBytes(16, func(i int) byte {
			return byte(i + 1)
		}),
	}

	t.Run("success", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			channel, err := conn.GetChannel(t.Context(), idx)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(channel, expected) {
				t.Fatalf("expected %s, got %s",
					describe(expected),
					describe(channel),
				)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandGetChannel),
			Byte(idx),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeChannelInfo, BytesFrom(
			Byte(expected.Index),
			CString(expected.Name, 32),
			Bytes(expected.Secret...),
		))

		controller.Wait()
	})

	t.Run("error", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if _, err := conn.GetChannel(t.Context(), idx); err == nil || err.Error() != "error: 5 (file io error)" {
				t.Fatalf("expected error: error: 5 (file io error), got %v", err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandGetChannel),
			Byte(idx),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeErr, BytesFrom(Byte(byte(ErrorCodeFileIOError))))

		controller.Wait()
	})
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

	controller.Notify(NotificationTypeOk, nil)

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

	controller.Notify(NotificationTypeDeviceInfo, BytesFrom(
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
			NotificationTypeContactMsgRecv,
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
			NotificationTypeChannelMsgRecv,
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

		controller.Notify(NotificationTypeNoMoreMessages, nil)

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

		controller.Notify(NotificationTypeOk, nil)

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

		controller.Notify(NotificationTypeOk, nil)

		controller.Wait()
	})

	// TODO(kellegous): test error cases
}

func TestExportContact(t *testing.T) {
	expected := []byte{1, 2, 3, 4, 5, 6}

	t.Run("self contact", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			advertPacket, err := conn.ExportContact(t.Context(), nil)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(advertPacket, expected) {
				t.Fatalf("expected %v, got %v", expected, advertPacket)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandExportContact),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(
			NotificationTypeExportContact,
			BytesFrom(Bytes(expected...)))

		controller.Wait()
	})

	t.Run("non-self contact", func(t *testing.T) {
		key := fakePublicKey(42)
		controller := DoCommand(func(conn *Conn) {
			advertPacket, err := conn.ExportContact(t.Context(), &key)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(advertPacket, expected) {
				t.Fatalf("expected %v, got %v", expected, advertPacket)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandExportContact),
			Bytes(key.Bytes()...),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(
			NotificationTypeExportContact,
			BytesFrom(Bytes(expected...)))

		controller.Wait()
	})
}

func TestShareContact(t *testing.T) {
	key := fakePublicKey(42)

	t.Run("success", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.ShareContact(t.Context(), key); err != nil {
				t.Fatal(err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandShareContact),
			Bytes(key.Bytes()...),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeOk, nil)

		controller.Wait()
	})

	t.Run("error", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.ShareContact(t.Context(), key); err == nil || err.Error() != "error: 5 (file io error)" {
				t.Fatalf("expected error: error: 5 (file io error), got %v", err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandShareContact),
			Bytes(key.Bytes()...),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeErr,
			BytesFrom(Byte(byte(ErrorCodeFileIOError))))

		controller.Wait()
	})
}

func TestExportPrivateKey(t *testing.T) {
	expected := fakeBytes(64, func(i int) byte {
		return byte(i + 1)
	})

	t.Run("enabled", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			privateKey, err := conn.ExportPrivateKey(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(privateKey, expected) {
				t.Fatalf("expected %v, got %v", expected, privateKey)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandExportPrivateKey),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(
			NotificationTypePrivateKey,
			BytesFrom(Bytes(expected...)))

		controller.Wait()
	})

	t.Run("disabled", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			_, err := conn.ExportPrivateKey(t.Context())
			if err == nil || err.Error() != "private key is disabled" {
				t.Fatalf("expected error: private key is disabled, got %v", err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandExportPrivateKey),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeDisabled, nil)

		controller.Wait()
	})
}

func TestImportPrivateKey(t *testing.T) {
	expected := fakeBytes(64, func(i int) byte {
		return byte(i + 1)
	})

	t.Run("success", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.ImportPrivateKey(t.Context(), expected); err != nil {
				t.Fatal(err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandImportPrivateKey),
			Bytes(expected...),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeOk, nil)

		controller.Wait()
	})

	t.Run("disabled", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.ImportPrivateKey(t.Context(), expected); err == nil || err.Error() != "private key is disabled" {
				t.Fatalf("expected error: private key is disabled, got %v", err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandImportPrivateKey),
			Bytes(expected...),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeDisabled, nil)

		controller.Wait()
	})
}

func TestGetStatus(t *testing.T) {
	key := fakePublicKey(42)
	expected := &StatusResponse{
		PubKeyPrefix: [6]byte{42, 0, 0, 0, 0, 0},
		StatusData:   []byte{1, 2, 3},
	}

	t.Run("success", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			status, err := conn.GetStatus(t.Context(), key)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(status, expected) {
				t.Fatalf("expected %s, got %s", describe(expected), describe(status))
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSendStatusReq),
			Bytes(key.Bytes()...),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeStatusResponse, BytesFrom(
			Byte(0),
			Bytes(key.Prefix(6)...),
			Bytes(1, 2, 3),
		))

		controller.Wait()
	})
}

func TestSendChannelTextMessage(t *testing.T) {
	channelIndex := byte(3)
	message := "hello"
	textType := TextTypePlain

	t.Run("success", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.SendChannelTextMessage(
				t.Context(),
				channelIndex,
				message,
				textType,
			); err != nil {
				t.Fatal(err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSendChannelTxtMsg),
			Byte(byte(textType)),
			Byte(channelIndex),
			AnyBytes(4),
			String(message),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeOk, nil)

		controller.Wait()
	})

	t.Run("error", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.SendChannelTextMessage(
				t.Context(),
				channelIndex,
				message,
				textType,
			); err == nil || err.Error() != "error: 5 (file io error)" {
				t.Fatalf("expected error: error: 5 (file io error), got %v", err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSendChannelTxtMsg),
			Byte(byte(textType)),
			Byte(channelIndex),
			AnyBytes(4),
			String(message),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeErr,
			BytesFrom(Byte(byte(ErrorCodeFileIOError))))

		controller.Wait()
	})
}

func TestSetAdvertLatLon(t *testing.T) {
	lat := 37.7
	lon := -122.4

	t.Run("success", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.SetAdvertLatLon(t.Context(), lat, lon); err != nil {
				t.Fatal(err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSetAdvertLatLon),
			LatLon(lat, lon, binary.LittleEndian),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeOk, nil)

		controller.Wait()
	})

	t.Run("error", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.SetAdvertLatLon(t.Context(), lat, lon); err == nil || err.Error() != "error: 5 (file io error)" {
				t.Fatalf("expected error: error: 5 (file io error), got %v", err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSetAdvertLatLon),
			LatLon(lat, lon, binary.LittleEndian),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeErr,
			BytesFrom(Byte(byte(ErrorCodeFileIOError))))

		controller.Wait()
	})
}

func TestSetAdvertName(t *testing.T) {
	name := "testname"

	t.Run("success", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.SetAdvertName(t.Context(), name); err != nil {
				t.Fatal(err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSetAdvertName),
			String(name),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeOk, nil)

		controller.Wait()
	})

	t.Run("error", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.SetAdvertName(t.Context(), name); err == nil || err.Error() != "error: 5 (file io error)" {
				t.Fatalf("expected error: error: 5 (file io error), got %v", err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSetAdvertName),
			String(name),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeErr,
			BytesFrom(Byte(byte(ErrorCodeFileIOError))))

		controller.Wait()
	})
}

func TestSetDeviceTime(t *testing.T) {
	time := time.Unix(100, 0)

	t.Run("success", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.SetDeviceTime(t.Context(), time); err != nil {
				t.Fatal(err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSetDeviceTime),
			Time(time, binary.LittleEndian),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeOk, nil)

		controller.Wait()
	})

	t.Run("error", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.SetDeviceTime(t.Context(), time); err == nil || err.Error() != "error: 5 (file io error)" {
				t.Fatalf("expected error: error: 5 (file io error), got %v", err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSetDeviceTime),
			Time(time, binary.LittleEndian),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeErr,
			BytesFrom(Byte(byte(ErrorCodeFileIOError))))

		controller.Wait()
	})
}

func TestResetPath(t *testing.T) {
	key := fakePublicKey(42)

	t.Run("success", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.ResetPath(t.Context(), key); err != nil {
				t.Fatal(err)
			}
		})
		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandResetPath),
			Bytes(key.Bytes()...),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeOk, nil)

		controller.Wait()
	})

	t.Run("error", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.ResetPath(t.Context(), key); err == nil || err.Error() != "error: 5 (file io error)" {
				t.Fatalf("expected error: error: 5 (file io error), got %v", err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandResetPath),
			Bytes(key.Bytes()...),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeErr,
			BytesFrom(Byte(byte(ErrorCodeFileIOError))))

		controller.Wait()
	})
}

func TestSign(t *testing.T) {
	shortMessage := []byte("Hello, world!")
	longMessage := fakeBytes(129, func(i int) byte {
		return byte(i + 1)
	})

	expected := fakeBytes(64, func(i int) byte {
		return byte(i + 1)
	})

	t.Run("success short", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			signature, err := conn.Sign(t.Context(), shortMessage)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(signature, expected) {
				t.Fatalf("expected %v, got %v", expected, signature)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSignStart),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeSignStart, BytesFrom(
			Byte(0),
			Uint32(1024, binary.LittleEndian),
		))

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSignData),
			Bytes(shortMessage...),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeSignature, BytesFrom(Bytes(expected...)))

		controller.Wait()
	})

	t.Run("success long", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			signature, err := conn.Sign(t.Context(), longMessage)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(signature, expected) {
				t.Fatalf("expected %v, got %v", expected, signature)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSignStart),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeSignStart, BytesFrom(
			Byte(0),
			Uint32(1024, binary.LittleEndian),
		))

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSignData),
			Bytes(longMessage[:128]...),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeOk, nil)

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSignData),
			Bytes(longMessage[128:]...),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeSignature, BytesFrom(Bytes(expected...)))

		controller.Wait()
	})
}

func TestImportContact(t *testing.T) {
	advertPacket := fakeBytes(100, func(i int) byte {
		return byte(i + 1)
	})

	t.Run("success", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.ImportContact(t.Context(), advertPacket); err != nil {
				t.Fatal(err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandImportContact),
			Bytes(advertPacket...),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeOk, nil)
		controller.Wait()
	})

	t.Run("error", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.ImportContact(t.Context(), advertPacket); err == nil || err.Error() != "error: 5 (file io error)" {
				t.Fatalf("expected error: error: 5 (file io error), got %v", err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandImportContact),
			Bytes(advertPacket...),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeErr,
			BytesFrom(Byte(byte(ErrorCodeFileIOError))))

		controller.Wait()
	})
}

func TestGetSelfInfo(t *testing.T) {
	expected := &SelfInfoResponse{
		Type:              1,
		TxPower:           2,
		MaxTxPower:        3,
		PublicKey:         fakePublicKey(42),
		AdvLat:            1.0,
		AdvLon:            2.0,
		ManualAddContacts: 4,
		RadioFreq:         910.525,
		RadioBw:           62.5,
		RadioSf:           7,
		RadioCr:           8,
		Name:              "testname",
	}

	t.Run("success", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			selfInfo, err := conn.GetSelfInfo(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(selfInfo, expected) {
				t.Fatalf("expected %s, got %s", describe(expected), describe(selfInfo))
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandAppStart),
			Byte(1),
			Bytes(0, 0, 0, 0, 0, 0),
			String("test"),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeSelfInfo, BytesFrom(
			Byte(byte(expected.Type)),
			Byte(expected.TxPower),
			Byte(expected.MaxTxPower),
			Bytes(expected.PublicKey.Bytes()...),
			LatLon(expected.AdvLat, expected.AdvLon, binary.LittleEndian),
			Bytes(0, 0, 0),
			Byte(expected.ManualAddContacts),
			Uint32(uint32(expected.RadioFreq*1000), binary.LittleEndian),
			Uint32(uint32(expected.RadioBw*1000), binary.LittleEndian),
			Byte(expected.RadioSf),
			Byte(expected.RadioCr),
			String(expected.Name),
		))

		controller.Wait()
	})

	t.Run("error", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if _, err := conn.GetSelfInfo(t.Context()); err == nil || err.Error() != "error: 5 (file io error)" {
				t.Fatalf("expected error: error: 5 (file io error), got %v", err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandAppStart),
			Byte(1),
			Bytes(0, 0, 0, 0, 0, 0),
			String("test"),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeErr,
			BytesFrom(Byte(byte(ErrorCodeFileIOError))))

		controller.Wait()
	})
}

func TestSetRadioParams(t *testing.T) {
	radioFreq := 910.525 // Is this in MHz?
	radioBw := 125.0     // This is in kHz
	radioSf := byte(7)
	radioCr := byte(5)

	t.Run("success", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.SetRadioParams(t.Context(), radioFreq, radioBw, radioSf, radioCr); err != nil {
				t.Fatal(err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSetRadioParams),
			Uint32(uint32(radioFreq*1000), binary.LittleEndian),
			Uint32(uint32(radioBw*1000), binary.LittleEndian),
			Byte(radioSf),
			Byte(radioCr),
		); err != nil {
			t.Fatal(err)
		}
		controller.Notify(NotificationTypeOk, nil)
		controller.Wait()
	})

	t.Run("error", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.SetRadioParams(t.Context(), radioFreq, radioBw, radioSf, radioCr); err == nil || err.Error() != "error: 5 (file io error)" {
				t.Fatalf("expected error: error: 5 (file io error), got %v", err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSetRadioParams),
			Uint32(uint32(radioFreq*1000), binary.LittleEndian),
			Uint32(uint32(radioBw*1000), binary.LittleEndian),
			Byte(radioSf),
			Byte(radioCr),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeErr, BytesFrom(Byte(byte(ErrorCodeFileIOError))))
		controller.Wait()
	})
}

func TestSendBinaryRequest(t *testing.T) {
	recipient := fakePublicKey(42)
	payload := fakeBytes(100, func(i int) byte {
		return byte(i + 1)
	})
	tag := uint32(1234567890)
	expected := &BinaryResponse{
		Tag:          tag,
		ResponseData: payload,
	}

	t.Run("success", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			res, err := conn.SendBinaryRequest(t.Context(), fakePublicKey(42), payload)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(res, expected) {
				t.Fatalf("expected %s, got %s", describe(expected), describe(res))
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSendBinaryReq),
			Bytes(recipient.Bytes()...),
			Bytes(payload...),
		); err != nil {
			t.Fatal(err)
		}

		// SentResponse
		controller.Notify(NotificationTypeSent, BytesFrom(
			Byte(0),
			Uint32(tag, binary.LittleEndian),
			Uint32(1000, binary.LittleEndian),
		))

		// BinaryResponse
		controller.Notify(NotificationTypeBinaryResponse, BytesFrom(
			Byte(0),
			Uint32(tag, binary.LittleEndian),
			Bytes(payload...),
		))

		controller.Wait()
	})

	t.Run("error", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if _, err := conn.SendBinaryRequest(t.Context(), fakePublicKey(42), payload); err == nil || err.Error() != "error: 5 (file io error)" {
				t.Fatalf("expected error: error: 5 (file io error), got %v", err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSendBinaryReq),
			Bytes(recipient.Bytes()...),
			Bytes(payload...),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeErr, BytesFrom(Byte(byte(ErrorCodeFileIOError))))

		controller.Wait()
	})

	t.Run("errant tag", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			res, err := conn.SendBinaryRequest(t.Context(), fakePublicKey(42), payload)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(res, expected) {
				t.Fatalf("expected %s, got %s", describe(expected), describe(res))
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSendBinaryReq),
			Bytes(recipient.Bytes()...),
			Bytes(payload...),
		); err != nil {
			t.Fatal(err)
		}

		// SentResponse
		controller.Notify(NotificationTypeSent, BytesFrom(
			Byte(0),
			Uint32(tag, binary.LittleEndian),
			Uint32(1000, binary.LittleEndian),
		))

		controller.Notify(NotificationTypeBinaryResponse, BytesFrom(
			Byte(0),
			Uint32(tag+1, binary.LittleEndian), // errant tag
			Bytes(payload...),
		))

		// BinaryResponse
		controller.Notify(NotificationTypeBinaryResponse, BytesFrom(
			Byte(0),
			Uint32(tag, binary.LittleEndian),
			Bytes(payload...),
		))

		controller.Wait()
	})
}

func TestSetTXPower(t *testing.T) {
	power := byte(10)

	t.Run("success", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.SetTXPower(t.Context(), power); err != nil {
				t.Fatal(err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSetTxPower),
			Byte(power),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeOk, nil)

		controller.Wait()
	})

	t.Run("error", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.SetTXPower(t.Context(), power); err == nil || err.Error() != "error: 5 (file io error)" {
				t.Fatalf("expected error: error: 5 (file io error), got %v", err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSetTxPower),
			Byte(power),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeErr, BytesFrom(Byte(byte(ErrorCodeFileIOError))))

		controller.Wait()
	})
}

func TestSetOtherParams(t *testing.T) {
	manualAddContacts := true

	t.Run("success", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.SetOtherParams(t.Context(), manualAddContacts); err != nil {
				t.Fatal(err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSetOtherParams),
			Bool(manualAddContacts),
		); err != nil {
			t.Fatal(err)
		}
		controller.Notify(NotificationTypeOk, nil)
		controller.Wait()
	})

	t.Run("error", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.SetOtherParams(t.Context(), manualAddContacts); err == nil || err.Error() != "error: 5 (file io error)" {
				t.Fatalf("expected error: error: 5 (file io error), got %v", err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSetOtherParams),
			Bool(manualAddContacts),
		); err != nil {
			t.Fatal(err)
		}
		controller.Notify(NotificationTypeErr, BytesFrom(Byte(byte(ErrorCodeFileIOError))))
		controller.Wait()
	})
}

func TestGetNeighbours(t *testing.T) {
	recipient := fakePublicKey(42)
	orderBy := NeighborsOrderNewestToOldest
	pubKeyPrefixLength := byte(6)
	tag := uint32(1234567890)
	expected := []*Neighbour{
		{
			PublicKeyPrefix: fakeBytes(6, func(i int) byte {
				return byte(i + 1)
			}),
			HeardSecondsAgo: uint32(100),
			Snr:             10.0,
		},
		{
			PublicKeyPrefix: fakeBytes(6, func(i int) byte {
				return byte(i * 2)
			}),
			HeardSecondsAgo: uint32(200),
			Snr:             20.0,
		},
	}

	t.Run("success", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			neighbours, err := conn.GetNeighbours(
				t.Context(),
				recipient,
				10, // count
				0,  // offset
				orderBy,
				pubKeyPrefixLength)
			if err != nil {
				t.Fatal(poop.Flatten(err))
			}
			if !reflect.DeepEqual(neighbours, expected) {
				t.Fatalf("expected %s, got %s", describe(expected), describe(neighbours))
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			BinaryRequest(recipient,
				Byte(byte(BinaryRequestTypeGetNeighbours)),
				Byte(0),
				Byte(10),
				Uint16(0, binary.LittleEndian), // offset
				Byte(byte(orderBy)),
				Byte(pubKeyPrefixLength),
				AnyBytes(4),
			)...,
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(NotificationTypeSent, BytesFrom(
			Byte(0),
			Uint32(tag, binary.LittleEndian),
			Uint32(1000, binary.LittleEndian),
		))

		controller.Notify(NotificationTypeBinaryResponse, BinaryResponseFrom(
			tag,
			Uint16(2, binary.LittleEndian),
			Uint16(2, binary.LittleEndian),
			Bytes(expected[0].PublicKeyPrefix...),
			Uint32(expected[0].HeardSecondsAgo, binary.LittleEndian),
			Byte(byte(expected[0].Snr*4)),
			Bytes(expected[1].PublicKeyPrefix...),
			Uint32(expected[1].HeardSecondsAgo, binary.LittleEndian),
			Byte(byte(expected[1].Snr*4)),
		))

		controller.Wait()
	})
}

func TestTracePath(t *testing.T) { // TODO: fix this test
	path := fakeBytes(10, func(i int) byte {
		return byte(i + 1)
	})
	expected := &TraceData{
		PathLen:    10,
		Flags:      0,
		AuthCode:   0,
		PathHashes: path,
		PathSnrs:   path,
		LastSnr:    10.0,
	}

	var tag []byte
	t.Run("success", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			traceData, err := conn.TracePath(t.Context(), path)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(traceData, expected) {
				t.Fatalf("expected %s, got %s", describe(expected), describe(traceData))
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSendTracePath),
			AnyBytesCapture(4, &tag),
			Uint32(expected.AuthCode, binary.LittleEndian),
			Byte(byte(expected.Flags)),
			Bytes(path...),
		); err != nil {
			t.Fatal(err)
		}

		expected.Tag = binary.LittleEndian.Uint32(tag)

		controller.Notify(NotificationTypeTraceData, BytesFrom(
			Byte(0),
			Byte(expected.PathLen),
			Byte(expected.Flags),
			Uint32(expected.Tag, binary.LittleEndian),
			Uint32(expected.AuthCode, binary.LittleEndian),
			Bytes(expected.PathHashes...),
			Bytes(expected.PathSnrs...),
			Byte(byte(expected.LastSnr*4)),
		))

		controller.Wait()
	})

	t.Run("error", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			_, err := conn.TracePath(t.Context(), path)
			if err == nil || err.Error() != "error: 5 (file io error)" {
				t.Fatalf("expected error: error: 5 (file io error), got %v", err)
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSendTracePath),
			AnyBytesCapture(4, &tag),
			Uint32(expected.AuthCode, binary.LittleEndian),
			Byte(byte(expected.Flags)),
			Bytes(path...),
		); err != nil {
			t.Fatal(err)
		}

		expected.Tag = binary.LittleEndian.Uint32(tag)

		controller.Notify(NotificationTypeErr, BytesFrom(Byte(byte(ErrorCodeFileIOError))))

		controller.Wait()
	})

	t.Run("errant tag", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			traceData, err := conn.TracePath(t.Context(), path)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(traceData, expected) {
				t.Fatalf("expected %s, got %s", describe(expected), describe(traceData))
			}
		})

		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSendTracePath),
			AnyBytesCapture(4, &tag),
			Uint32(expected.AuthCode, binary.LittleEndian),
			Byte(byte(expected.Flags)),
			Bytes(path...),
		); err != nil {
			t.Fatal(err)
		}

		expected.Tag = binary.LittleEndian.Uint32(tag)

		// tag does not match.
		controller.Notify(NotificationTypeTraceData, BytesFrom(
			Byte(0),
			Byte(expected.PathLen),
			Byte(expected.Flags),
			Uint32(expected.Tag+1, binary.LittleEndian), // errant tag
			Uint32(expected.AuthCode, binary.LittleEndian),
			Bytes(expected.PathHashes...),
			Bytes(expected.PathSnrs...),
			Byte(byte(expected.LastSnr*4)),
		))

		controller.Notify(NotificationTypeTraceData, BytesFrom(
			Byte(0),
			Byte(expected.PathLen),
			Byte(expected.Flags),
			Uint32(expected.Tag, binary.LittleEndian),
			Uint32(expected.AuthCode, binary.LittleEndian),
			Bytes(expected.PathHashes...),
			Bytes(expected.PathSnrs...),
			Byte(byte(expected.LastSnr*4)),
		))

		controller.Wait()
	})
}

func TestLogin(t *testing.T) {
	key := fakePublicKey(42)
	password := "password"

	t.Run("success", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.Login(t.Context(), key, password); err != nil {
				t.Fatal(err)
			}
		})
		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSendLogin),
			Bytes(key.Bytes()...),
			String(password),
		); err != nil {
			t.Fatal(err)
		}
		controller.Notify(NotificationTypeLoginSuccess, BytesFrom(
			Byte(0),
			Bytes(key.Prefix(6)...),
		))
		controller.Wait()
	})

	t.Run("error", func(t *testing.T) {
		controller := DoCommand(func(conn *Conn) {
			if err := conn.Login(t.Context(), key, password); err == nil || err.Error() != "error: 5 (file io error)" {
				t.Fatalf("expected error: error: 5 (file io error), got %v", err)
			}
		})
		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSendLogin),
			Bytes(key.Bytes()...),
			String(password),
		); err != nil {
			t.Fatal(err)
		}

		controller.Notify(
			NotificationTypeErr,
			BytesFrom(
				Byte(byte(ErrorCodeFileIOError)),
			))

		controller.Wait()
	})

	t.Run("errant key", func(t *testing.T) {
		otherKey := fakePublicKey(43)
		controller := DoCommand(func(conn *Conn) {
			if err := conn.Login(t.Context(), key, password); err != nil {
				t.Fatal(err)
			}
		})
		if err := ValidateBytes(
			controller.Recv(),
			Command(CommandSendLogin),
			Bytes(key.Bytes()...),
			String(password),
		); err != nil {
			t.Fatal(err)
		}
		controller.Notify(
			NotificationTypeLoginSuccess,
			BytesFrom(
				Byte(0),
				Bytes(otherKey.Prefix(6)...),
			))
		controller.Notify(
			NotificationTypeLoginSuccess,
			BytesFrom(
				Byte(0),
				Bytes(key.Prefix(6)...),
			))
		controller.Wait()
	})
}

func TestOnAdvert(t *testing.T) {
	key := fakePublicKey(42)

	subReady := make(chan struct{})

	controller := DoCommand(func(conn *Conn) {
		subTriggered := make(chan struct{})
		unsub := conn.OnAdvert(func(advertEvent *AdvertEvent) {
			if !reflect.DeepEqual(advertEvent.PublicKey, key) {
				t.Fatalf("expected %s, got %s", describe(key), describe(advertEvent.PublicKey))
			}

			close(subTriggered)
		})
		defer unsub()

		close(subReady)
		<-subTriggered
	})

	<-subReady

	controller.Notify(NotificationTypeAdvert, BytesFrom(
		Bytes(key.Bytes()...),
	))

	controller.Wait()
}

func TestOnNewAdvert(t *testing.T) {
	outPath := fakeBytes(64, func(i int) byte {
		return byte(i + 1)
	})

	expected := &NewAdvertEvent{
		PublicKey:  fakePublicKey(42),
		Type:       ContactTypeChat,
		Flags:      0,
		OutPath:    outPath[:6],
		AdvName:    "test",
		LastAdvert: time.Unix(420, 0),
		AdvLat:     37.774929,
		AdvLon:     -122.419416,
	}

	subReady := make(chan struct{})

	controller := DoCommand(func(conn *Conn) {
		subTriggered := make(chan struct{})
		unsub := conn.OnNewAdvert(func(newAdvertEvent *NewAdvertEvent) {
			if !reflect.DeepEqual(newAdvertEvent, expected) {
				t.Fatalf("expected %s, got %s", describe(expected), describe(newAdvertEvent))
			}

			close(subTriggered)
		})
		defer unsub()

		close(subReady)
		<-subTriggered
	})

	<-subReady

	controller.Notify(NotificationTypeNewAdvert, BytesFrom(
		Bytes(expected.PublicKey.Bytes()...),
		Byte(byte(ContactTypeChat)),
		Byte(0),
		Byte(byte(len(expected.OutPath))),
		Bytes(outPath...),
		CString("test", 32),
		Time(time.Unix(420, 0), binary.LittleEndian),
		LatLon(37.774929, -122.419416, binary.LittleEndian),
	))

	controller.Wait()
}
