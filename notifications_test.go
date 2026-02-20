package meshcore

import (
	"encoding/binary"
	"reflect"
	"testing"
	"time"
)

func validateError(t *testing.T, got error, expected error) {
	if expected == nil {
		if got != nil {
			t.Fatalf("expected no error, got %v", got)
		}
	} else {
		if got == nil {
			t.Fatalf("expected error %v, got none", expected)
		}
		if got.Error() != expected.Error() {
			t.Fatalf("expected error %v, got %v", expected, got)
		}
	}
}

func TestReadNotification(t *testing.T) {
	type expected struct {
		Notification Notification
		Error        error
	}

	fakePublicKey := fakePublicKey(42)

	var fakePublicKeyPrefix [6]byte
	copy(fakePublicKeyPrefix[:], fakePublicKey.Prefix(6))

	var fakePrivateKey [64]byte
	copy(fakePrivateKey[:], fakeBytes(64, func(i int) byte { return byte(i + 1) }))

	tests := []struct {
		Name     string
		Code     NotificationCode
		Data     []byte
		Expected expected
	}{
		{
			Name: "Ok",
			Code: NotificationTypeOk,
			Data: nil,
			Expected: expected{
				Notification: &OkNotification{},
			},
		},
		{
			Name: "Err (w/ code)",
			Code: NotificationTypeErr,
			Data: []byte{0x01},
			Expected: expected{
				Notification: &ErrNotification{Code: ErrorCodeUnsupportedCommand},
			},
		},
		{
			Name: "Err (w/o code)",
			Code: NotificationTypeErr,
			Data: []byte{},
			Expected: expected{
				Notification: &ErrNotification{Code: ErrorCodeUnknown},
			},
		},
		{
			Name: "ContactsStart",
			Code: NotificationTypeContactsStart,
			Data: nil,
			Expected: expected{
				Notification: &ContactStartNotification{},
			},
		},
		{
			Name: "Contact",
			Code: NotificationTypeContact,
			Data: BytesFrom(
				Bytes(fakePublicKey.Bytes()...),
				Byte(byte(ContactTypeChat)),
				Byte(0),
				Byte(2),
				Bytes(fakeBytes(64, func(i int) byte { return byte(i + 1) })...),
				CString("test", 32),
				Time(time.Unix(100, 0), binary.LittleEndian),
				LatLon(37.774929, -122.419416, binary.LittleEndian),
				Time(time.Unix(101, 0), binary.LittleEndian),
			),
			Expected: expected{
				Notification: &ContactNotification{Contact: Contact{
					PublicKey:  fakePublicKey,
					Type:       ContactTypeChat,
					Flags:      0,
					OutPath:    []byte{1, 2},
					AdvName:    "test",
					LastAdvert: time.Unix(100, 0),
					AdvLat:     37.774929,
					AdvLon:     -122.419416,
					LastMod:    time.Unix(101, 0),
				}},
				Error: nil,
			},
		},
		{
			Name: "EndOfContacts",
			Code: NotificationTypeEndOfContacts,
			Data: nil,
			Expected: expected{
				Notification: &EndOfContactsNotification{},
			},
		},
		{
			Name: "SelfInfo",
			Code: NotificationTypeSelfInfo,
			Data: BytesFrom(
				Byte(byte(1)),
				Byte(byte(20)), // TxPower
				Byte(byte(30)), // MaxTxPower
				Bytes(fakePublicKey.Bytes()...),
				LatLon(37.774929, -122.419416, binary.LittleEndian),
				Bytes(0, 0, 0),
				Byte(byte(0)),                       // ManualAddContacts
				Uint32(910525, binary.LittleEndian), // RadioFreq
				Uint32(62500, binary.LittleEndian),  // RadioBw
				Byte(byte(6)),                       // RadioSf
				Byte(byte(8)),                       // RadioCr
				String("testname"),
			),
			Expected: expected{
				Notification: &SelfInfoNotification{
					Type:              1,
					TxPower:           20,
					MaxTxPower:        30,
					PublicKey:         fakePublicKey,
					AdvLat:            37.774929,
					AdvLon:            -122.419416,
					ManualAddContacts: 0,
					RadioFreq:         910.525,
					RadioBw:           62.5,
					RadioSf:           6,
					RadioCr:           8,
					Name:              "testname",
				},
			},
		},
		{
			Name: "Sent",
			Code: NotificationTypeSent,
			Data: BytesFrom(
				Byte(byte(1)),
				Uint32(1234567890, binary.LittleEndian),
				Uint32(1000, binary.LittleEndian),
			),
			Expected: expected{
				Notification: &SentNotification{
					Result:         1,
					ExpectedAckCRC: 1234567890,
					EstTimeout:     1000,
				},
			},
		},
		{
			Name: "ContactMsgRecv",
			Code: NotificationTypeContactMsgRecv,
			Data: BytesFrom(
				Bytes(fakePublicKeyPrefix[:]...),
				Byte(byte(2)),
				Byte(byte(TextTypeSignedPlain)),
				Time(time.Unix(100, 0), binary.LittleEndian),
				String("test"),
			),
			Expected: expected{
				Notification: &ContactMsgRecvNotification{
					ContactMessage: ContactMessage{
						PubKeyPrefix: fakePublicKeyPrefix,
						PathLen:      2,
						TextType:     TextTypeSignedPlain,
						SenderTime:   time.Unix(100, 0),
						Text:         "test",
					},
				},
			},
		},
		{
			Name: "ChannelMsgRecv",
			Code: NotificationTypeChannelMsgRecv,
			Data: BytesFrom(
				Byte(byte(1)),
				Byte(byte(2)),
				Byte(byte(TextTypeSignedPlain)),
				Time(time.Unix(100, 0), binary.LittleEndian),
				String("test"),
			),
			Expected: expected{
				Notification: &ChannelMsgRecvNotification{
					ChannelMessage: ChannelMessage{
						ChannelIndex: 1,
						PathLen:      2,
						TextType:     TextTypeSignedPlain,
						SenderTime:   time.Unix(100, 0),
						Text:         "test",
					},
				},
			},
		},
		{
			Name: "CurrTime",
			Code: NotificationTypeCurrTime,
			Data: BytesFrom(
				Time(time.Unix(100, 0), binary.LittleEndian),
			),
			Expected: expected{
				Notification: &CurrTimeNotification{
					Time: time.Unix(100, 0),
				},
			},
		},
		{
			Name: "NoMoreMessages",
			Code: NotificationTypeNoMoreMessages,
			Data: nil,
			Expected: expected{
				Notification: &NoMoreMessagesNotification{},
			},
		},
		{
			Name: "ExportContact",
			Code: NotificationTypeExportContact,
			Data: BytesFrom(
				Bytes(1, 2, 3, 4, 5, 6),
			),
			Expected: expected{
				Notification: &ExportContactNotification{
					AdvertPacket: []byte{1, 2, 3, 4, 5, 6},
				},
			},
		},
		{
			Name: "BatteryVoltage",
			Code: NotificationTypeBatteryVoltage,
			Data: BytesFrom(
				Uint16(12345, binary.LittleEndian),
			),
			Expected: expected{
				Notification: &BatteryVoltageNotification{
					Voltage: 12345,
				},
			},
		},
		{
			Name: "DeviceInfo",
			Code: NotificationTypeDeviceInfo,
			Data: BytesFrom(
				Byte(byte(1)),
				Bytes(0, 0, 0, 0, 0, 0), // reserved 6 bytes
				CString("test_x", 12),
				String("test_y"),
			),
			Expected: expected{
				Notification: &DeviceInfoNotification{
					DeviceInfo: DeviceInfo{
						FirmwareVersion:   1,
						FirmwareBuildDate: "test_x",
						ManufacturerModel: "test_y",
					},
				},
			},
		},
		{
			Name: "PrivateKey",
			Code: NotificationTypePrivateKey,
			Data: BytesFrom(
				Bytes(fakePrivateKey[:]...),
			),
			Expected: expected{
				Notification: &PrivateKeyNotification{
					PrivateKey: fakePrivateKey,
				},
			},
		},
		{
			Name: "Disabled",
			Code: NotificationTypeDisabled,
			Data: nil,
			Expected: expected{
				Notification: &DisabledNotification{},
			},
		},
		{
			Name: "ChannelInfo",
			Code: NotificationTypeChannelInfo,
			Data: BytesFrom(
				Byte(byte(1)),
				CString("test", 32),
				Bytes(fakeBytes(16, func(i int) byte { return byte(i + 1) })...),
			),
			Expected: expected{
				Notification: &ChannelInfoNotification{
					ChannelInfo: ChannelInfo{
						Index:  1,
						Name:   "test",
						Secret: fakeBytes(16, func(i int) byte { return byte(i + 1) }),
					},
				},
			},
		},
		{
			Name: "SignStart",
			Code: NotificationTypeSignStart,
			Data: BytesFrom(
				Byte(byte(0)),
				Uint32(1024, binary.LittleEndian),
			),
			Expected: expected{
				Notification: &SignStartNotification{
					MaxSignDataLen: 1024,
				},
			},
		},
		{
			Name: "Signature",
			Code: NotificationTypeSignature,
			Data: BytesFrom(
				Bytes(fakePrivateKey[:]...),
			),
			Expected: expected{
				Notification: &SignatureNotification{
					Signature: fakePrivateKey,
				},
			},
		},
		{
			Name: "Advert",
			Code: NotificationTypeAdvert,
			Data: BytesFrom(
				Bytes(fakePublicKey.Bytes()...),
			),
			Expected: expected{
				Notification: &AdvertNotification{
					PublicKey: fakePublicKey,
				},
			},
		},
		{
			Name: "PathUpdated",
			Code: NotificationTypePathUpdated,
			Data: BytesFrom(
				Bytes(fakePublicKey.Bytes()...),
			),
			Expected: expected{
				Notification: &PathUpdatedNotification{
					PublicKey: fakePublicKey,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			notification, err := readNotification(test.Code, test.Data)
			validateError(t, err, test.Expected.Error)
			if !reflect.DeepEqual(notification, test.Expected.Notification) {
				t.Fatalf("expected notification %s, got %s",
					describe(test.Expected.Notification),
					describe(notification),
				)
			}

			if test.Code != test.Expected.Notification.NotificationCode() {
				t.Fatalf("expected notification code %s, got %s",
					test.Expected.Notification.NotificationCode(),
					test.Code,
				)
			}
		})
	}
}
