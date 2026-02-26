package meshcore

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/kellegous/poop"
)

type Notification interface {
	NotificationCode() NotificationCode
}

type NotificationCode byte

const (
	// Response notifications, arrive in response to a command.
	NotificationTypeOk               NotificationCode = 0
	NotificationTypeErr              NotificationCode = 1
	NotificationTypeContactsStart    NotificationCode = 2
	NotificationTypeContact          NotificationCode = 3
	NotificationTypeEndOfContacts    NotificationCode = 4
	NotificationTypeSelfInfo         NotificationCode = 5
	NotificationTypeSent             NotificationCode = 6
	NotificationTypeContactMsgRecv   NotificationCode = 7
	NotificationTypeChannelMsgRecv   NotificationCode = 8
	NotificationTypeCurrTime         NotificationCode = 9
	NotificationTypeNoMoreMessages   NotificationCode = 10
	NotificationTypeExportContact    NotificationCode = 11
	NotificationTypeBatteryVoltage   NotificationCode = 12
	NotificationTypeDeviceInfo       NotificationCode = 13
	NotificationTypePrivateKey       NotificationCode = 14
	NotificationTypeDisabled         NotificationCode = 15
	NotificationTypeChannelMsgRecvV3 NotificationCode = 16
	NotificationTypeContactMsgRecvV3 NotificationCode = 17
	NotificationTypeChannelInfo      NotificationCode = 18
	NotificationTypeSignStart        NotificationCode = 19
	NotificationTypeSignature        NotificationCode = 20
	// Push notifications, can arrive without a corresponding command.
	NotificationTypeAdvert         NotificationCode = 0x80 // when companion is set to auto add contacts
	NotificationTypePathUpdated    NotificationCode = 0x81
	NotificationTypeSendConfirmed  NotificationCode = 0x82
	NotificationTypeMsgWaiting     NotificationCode = 0x83
	NotificationTypeRawData        NotificationCode = 0x84
	NotificationTypeLoginSuccess   NotificationCode = 0x85
	NotificationTypeLoginFail      NotificationCode = 0x86 // not usable yet
	NotificationTypeStatus         NotificationCode = 0x87
	NotificationTypeLogRxData      NotificationCode = 0x88
	NotificationTypeTraceData      NotificationCode = 0x89
	NotificationTypeNewAdvert      NotificationCode = 0x8A // when companion is set to manually add contacts
	NotificationTypeTelemetry      NotificationCode = 0x8B
	NotificationTypeBinaryResponse NotificationCode = 0x8C
)

var notificationCodeText = map[NotificationCode]string{
	NotificationTypeOk:               "Ok",
	NotificationTypeErr:              "Err",
	NotificationTypeContactsStart:    "ContactsStart",
	NotificationTypeContact:          "Contact",
	NotificationTypeEndOfContacts:    "EndOfContacts",
	NotificationTypeSelfInfo:         "SelfInfo",
	NotificationTypeSent:             "Sent",
	NotificationTypeContactMsgRecv:   "ContactMsgRecv",
	NotificationTypeChannelMsgRecv:   "ChannelMsgRecv",
	NotificationTypeCurrTime:         "CurrTime",
	NotificationTypeNoMoreMessages:   "NoMoreMessages",
	NotificationTypeExportContact:    "ExportContact",
	NotificationTypeBatteryVoltage:   "BatteryVoltage",
	NotificationTypeDeviceInfo:       "DeviceInfo",
	NotificationTypePrivateKey:       "PrivateKey",
	NotificationTypeDisabled:         "Disabled",
	NotificationTypeContactMsgRecvV3: "ContactMsgRecvV3",
	NotificationTypeChannelMsgRecvV3: "ChannelMsgRecvV3",
	NotificationTypeChannelInfo:      "ChannelInfo",
	NotificationTypeSignStart:        "SignStart",
	NotificationTypeSignature:        "Signature",
	NotificationTypeAdvert:           "PushAdvert",
	NotificationTypePathUpdated:      "PushPathUpdated",
	NotificationTypeSendConfirmed:    "PushSendConfirmed",
	NotificationTypeMsgWaiting:       "PushMsgWaiting",
	NotificationTypeRawData:          "PushRawData",
	NotificationTypeLoginSuccess:     "PushLoginSuccess",
	NotificationTypeLoginFail:        "PushLoginFail",
	NotificationTypeStatus:           "PushStatus",
	NotificationTypeLogRxData:        "PushLogRxData",
	NotificationTypeTraceData:        "PushTraceData",
	NotificationTypeNewAdvert:        "PushNewAdvert",
	NotificationTypeTelemetry:        "PushTelemetryResponse",
	NotificationTypeBinaryResponse:   "PushBinaryResponse",
}

func (c NotificationCode) String() string {
	return notificationCodeText[c]
}

type ErrorCode byte

const (
	ErrorCodeUnknown            ErrorCode = 0
	ErrorCodeUnsupportedCommand ErrorCode = 1
	ErrorCodeNotFound           ErrorCode = 2
	ErrorCodeTableFull          ErrorCode = 3
	ErrorCodeBadState           ErrorCode = 4
	ErrorCodeFileIOError        ErrorCode = 5
	ErrorCodeIllegalArgument    ErrorCode = 6
)

var errorText = map[ErrorCode]string{
	ErrorCodeUnknown:            "unknown error",
	ErrorCodeUnsupportedCommand: "unsupported command",
	ErrorCodeNotFound:           "not found",
	ErrorCodeTableFull:          "table full",
	ErrorCodeBadState:           "bad state",
	ErrorCodeFileIOError:        "file io error",
	ErrorCodeIllegalArgument:    "illegal argument",
}

type CommandError struct {
	Code ErrorCode
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("error: %d (%s)", e.Code, errorText[e.Code])
}

func readNotification(code NotificationCode, data []byte) (Notification, error) {
	switch code {
	case NotificationTypeOk:
		return readOkNotification(data)
	case NotificationTypeErr:
		return readErrNotification(data)
	case NotificationTypeContactsStart:
		return readContactStartNotification(data)
	case NotificationTypeContact:
		return readContactNotification(data)
	case NotificationTypeEndOfContacts:
		return readEndOfContactsNotification(data)
	case NotificationTypeSelfInfo:
		return readSelfInfoNotification(data)
	case NotificationTypeSent:
		return readSentNotification(data)
	case NotificationTypeContactMsgRecv:
		return readContactMsgRecvNotification(data)
	case NotificationTypeChannelMsgRecv:
		return readChannelMsgRecvNotification(data)
	case NotificationTypeCurrTime:
		return readCurrTimeNotification(data)
	case NotificationTypeNoMoreMessages:
		return readNoMoreMessagesNotification(data)
	case NotificationTypeExportContact:
		return readExportContactNotification(data)
	case NotificationTypeBatteryVoltage:
		return readBatteryVoltageNotification(data)
	case NotificationTypeDeviceInfo:
		return readDeviceInfoNotification(data)
	case NotificationTypePrivateKey:
		return readPrivateKeyNotification(data)
	case NotificationTypeDisabled:
		return readDisabledNotification(data)
	case NotificationTypeContactMsgRecvV3:
		return readContactMsgRecvV3Notification(data)
	case NotificationTypeChannelMsgRecvV3:
		return readChannelMsgRecvV3Notification(data)
	case NotificationTypeChannelInfo:
		return readChannelInfoNotification(data)
	case NotificationTypeSignStart:
		return readSignStartNotification(data)
	case NotificationTypeSignature:
		return readSignatureNotification(data)
	case NotificationTypeAdvert:
		return readAdvertNotification(data)
	case NotificationTypePathUpdated:
		return readPathUpdatedNotification(data)
	case NotificationTypeSendConfirmed:
		return readSendConfirmedNotification(data)
	case NotificationTypeMsgWaiting:
		return &MsgWaitingNotification{}, nil
	case NotificationTypeRawData:
		return readRawDataNotification(data)
	case NotificationTypeLoginSuccess:
		return readLoginSuccessNotification(data)
	case NotificationTypeLoginFail:
		return readLoginFailNotification(data)
	case NotificationTypeStatus:
		return readStatusNotification(data)
	case NotificationTypeLogRxData:
		return readLogRxDataNotification(data)
	case NotificationTypeBinaryResponse:
		return readBinaryResponseNotification(data)
	case NotificationTypeTraceData:
		return readTraceDataNotification(data)
	case NotificationTypeNewAdvert:
		return readNewAdvertNotification(data)
	case NotificationTypeTelemetry:
		return readTelemetryNotification(data)
	}
	return nil, poop.New("unknown notification code")
}

type OkNotification struct{}

func (e *OkNotification) NotificationCode() NotificationCode {
	return NotificationTypeOk
}

func readOkNotification(_ []byte) (*OkNotification, error) {
	return &OkNotification{}, nil
}

type ErrNotification struct {
	Code ErrorCode
}

func (e *ErrNotification) NotificationCode() NotificationCode {
	return NotificationTypeErr
}

func (e *ErrNotification) Error() error {
	return &CommandError{Code: e.Code}
}

func readErrNotification(data []byte) (*ErrNotification, error) {
	if len(data) == 0 {
		return &ErrNotification{Code: ErrorCodeUnknown}, nil
	}
	return &ErrNotification{Code: ErrorCode(data[0])}, nil
}

func hasErrorCode(err error, code ErrorCode) bool {
	var resErr *CommandError
	if errors.As(err, &resErr) {
		return resErr.Code == code
	}
	return false
}

type ContactStartNotification struct{}

func (e *ContactStartNotification) NotificationCode() NotificationCode {
	return NotificationTypeContactsStart
}

func readContactStartNotification(_ []byte) (*ContactStartNotification, error) {
	return &ContactStartNotification{}, nil
}

type ContactNotification struct {
	Contact Contact
}

func (e *ContactNotification) NotificationCode() NotificationCode {
	return NotificationTypeContact
}

func readContactNotification(data []byte) (*ContactNotification, error) {
	var notif ContactNotification
	if err := notif.Contact.readFrom(bytes.NewReader(data)); err != nil {
		return nil, poop.Chain(err)
	}
	return &notif, nil
}

type EndOfContactsNotification struct{}

func (e *EndOfContactsNotification) NotificationCode() NotificationCode {
	return NotificationTypeEndOfContacts
}

func readEndOfContactsNotification(_ []byte) (*EndOfContactsNotification, error) {
	return &EndOfContactsNotification{}, nil
}

type SelfInfoNotification struct {
	SelfInfo SelfInfo
}

func (e *SelfInfoNotification) NotificationCode() NotificationCode {
	return NotificationTypeSelfInfo
}

func readSelfInfoNotification(data []byte) (*SelfInfoNotification, error) {
	var n SelfInfoNotification
	r := bytes.NewReader(data)
	if err := n.SelfInfo.readFrom(r); err != nil {
		return nil, poop.Chain(err)
	}
	return &n, nil
}

type SentNotification struct {
	Result         int8
	ExpectedAckCRC uint32
	EstTimeout     uint32
}

func (e *SentNotification) NotificationCode() NotificationCode {
	return NotificationTypeSent
}

func readSentNotification(data []byte) (*SentNotification, error) {
	var n SentNotification
	r := bytes.NewReader(data)
	if err := binary.Read(r, binary.LittleEndian, &n.Result); err != nil {
		return nil, poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &n.ExpectedAckCRC); err != nil {
		return nil, poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &n.EstTimeout); err != nil {
		return nil, poop.Chain(err)
	}
	return &n, nil
}

type ContactMsgRecvNotification struct {
	ContactMessage ContactMessage
}

func (e *ContactMsgRecvNotification) NotificationCode() NotificationCode {
	return NotificationTypeContactMsgRecv
}

func readContactMsgRecvNotification(data []byte) (*ContactMsgRecvNotification, error) {
	var n ContactMsgRecvNotification
	if err := n.ContactMessage.readFrom(bytes.NewReader(data)); err != nil {
		return nil, poop.Chain(err)
	}
	return &n, nil
}

type ChannelMsgRecvNotification struct {
	ChannelMessage ChannelMessage
}

func (e *ChannelMsgRecvNotification) NotificationCode() NotificationCode {
	return NotificationTypeChannelMsgRecv
}

func readChannelMsgRecvNotification(data []byte) (*ChannelMsgRecvNotification, error) {
	var n ChannelMsgRecvNotification
	if err := n.ChannelMessage.readFrom(bytes.NewReader(data)); err != nil {
		return nil, poop.Chain(err)
	}
	return &n, nil
}

type CurrTimeNotification struct {
	Time time.Time
}

func (e *CurrTimeNotification) NotificationCode() NotificationCode {
	return NotificationTypeCurrTime
}

func readCurrTimeNotification(data []byte) (*CurrTimeNotification, error) {
	var n CurrTimeNotification
	var err error
	n.Time, err = readTime(bytes.NewReader(data))
	if err != nil {
		return nil, poop.Chain(err)
	}
	return &n, nil
}

type NoMoreMessagesNotification struct{}

func (e *NoMoreMessagesNotification) NotificationCode() NotificationCode {
	return NotificationTypeNoMoreMessages
}

func readNoMoreMessagesNotification(_ []byte) (*NoMoreMessagesNotification, error) {
	return &NoMoreMessagesNotification{}, nil
}

type ExportContactNotification struct {
	AdvertPacket []byte
}

func (e *ExportContactNotification) NotificationCode() NotificationCode {
	return NotificationTypeExportContact
}

func readExportContactNotification(data []byte) (*ExportContactNotification, error) {
	var n ExportContactNotification
	n.AdvertPacket = make([]byte, len(data))
	copy(n.AdvertPacket, data)
	return &n, nil
}

type BatteryVoltageNotification struct {
	Voltage uint16
}

func (e *BatteryVoltageNotification) NotificationCode() NotificationCode {
	return NotificationTypeBatteryVoltage
}

func readBatteryVoltageNotification(data []byte) (*BatteryVoltageNotification, error) {
	var n BatteryVoltageNotification
	r := bytes.NewReader(data)
	if err := binary.Read(r, binary.LittleEndian, &n.Voltage); err != nil {
		return nil, poop.Chain(err)
	}
	return &n, nil
}

type DeviceInfoNotification struct {
	DeviceInfo DeviceInfo
}

func (e *DeviceInfoNotification) NotificationCode() NotificationCode {
	return NotificationTypeDeviceInfo
}

func readDeviceInfoNotification(data []byte) (*DeviceInfoNotification, error) {
	var n DeviceInfoNotification
	if err := n.DeviceInfo.readFrom(bytes.NewReader(data)); err != nil {
		return nil, poop.Chain(err)
	}
	return &n, nil
}

type PrivateKeyNotification struct {
	PrivateKey [64]byte
}

func (e *PrivateKeyNotification) NotificationCode() NotificationCode {
	return NotificationTypePrivateKey
}

func readPrivateKeyNotification(data []byte) (*PrivateKeyNotification, error) {
	var n PrivateKeyNotification
	r := bytes.NewReader(data)
	if _, err := io.ReadFull(r, n.PrivateKey[:]); err != nil {
		return nil, poop.Chain(err)
	}
	return &n, nil
}

type DisabledNotification struct{}

func (e *DisabledNotification) NotificationCode() NotificationCode {
	return NotificationTypeDisabled
}

func readDisabledNotification(_ []byte) (*DisabledNotification, error) {
	return &DisabledNotification{}, nil
}

type ChannelInfoNotification struct {
	ChannelInfo ChannelInfo
}

func (e *ChannelInfoNotification) NotificationCode() NotificationCode {
	return NotificationTypeChannelInfo
}

func readChannelInfoNotification(data []byte) (*ChannelInfoNotification, error) {
	var n ChannelInfoNotification
	if err := n.ChannelInfo.readFrom(bytes.NewReader(data)); err != nil {
		return nil, poop.Chain(err)
	}
	return &n, nil
}

type SignStartNotification struct {
	MaxSignDataLen uint32
}

func (e *SignStartNotification) NotificationCode() NotificationCode {
	return NotificationTypeSignStart
}

func readSignStartNotification(data []byte) (*SignStartNotification, error) {
	var n SignStartNotification
	r := bytes.NewReader(data)
	var reserved byte
	if err := binary.Read(r, binary.LittleEndian, &reserved); err != nil {
		return nil, poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &n.MaxSignDataLen); err != nil {
		return nil, poop.Chain(err)
	}
	return &n, nil
}

type SignatureNotification struct {
	Signature [64]byte
}

func (e *SignatureNotification) NotificationCode() NotificationCode {
	return NotificationTypeSignature
}

func readSignatureNotification(data []byte) (*SignatureNotification, error) {
	var n SignatureNotification
	r := bytes.NewReader(data)
	if _, err := io.ReadFull(r, n.Signature[:]); err != nil {
		return nil, poop.Chain(err)
	}
	return &n, nil
}

type AdvertNotification struct {
	PublicKey PublicKey
}

func (e *AdvertNotification) NotificationCode() NotificationCode {
	return NotificationTypeAdvert
}

func readAdvertNotification(data []byte) (*AdvertNotification, error) {
	var n AdvertNotification
	if err := n.PublicKey.readFrom(bytes.NewReader(data)); err != nil {
		return nil, poop.Chain(err)
	}
	return &n, nil
}

type PathUpdatedNotification struct {
	PublicKey PublicKey
}

func (e *PathUpdatedNotification) NotificationCode() NotificationCode {
	return NotificationTypePathUpdated
}

func readPathUpdatedNotification(data []byte) (*PathUpdatedNotification, error) {
	var n PathUpdatedNotification
	if err := n.PublicKey.readFrom(bytes.NewReader(data)); err != nil {
		return nil, poop.Chain(err)
	}
	return &n, nil
}

type LoginSuccessNotification struct {
	PubKeyPrefix [6]byte
}

func (e *LoginSuccessNotification) NotificationCode() NotificationCode {
	return NotificationTypeLoginSuccess
}

//	PUSH_CODE_LOGIN_SUCCESS {
//		code: byte,    // constant 0x85
//		permissions: byte,     // is_admin if lowest bit is 1
//		pub_key_prefix: bytes(6)     // public key prefix (first 6 bytes)
//		tag: int32,
//		new_permissions: byte     // V7+
//	}
func readLoginSuccessNotification(data []byte) (*LoginSuccessNotification, error) {
	// TODO(kellegous): This is not complete. We only marshal the pub key
	// but the permissions, tag and new_permissions need to be read.
	var n LoginSuccessNotification
	r := bytes.NewReader(data)
	var reserved byte
	if err := binary.Read(r, binary.LittleEndian, &reserved); err != nil {
		return nil, poop.Chain(err)
	}
	if _, err := io.ReadFull(r, n.PubKeyPrefix[:]); err != nil {
		return nil, poop.Chain(err)
	}
	return &n, nil
}

type LoginFailNotification struct{}

func (e *LoginFailNotification) NotificationCode() NotificationCode {
	return NotificationTypeLoginFail
}

func readLoginFailNotification(_ []byte) (*LoginFailNotification, error) {
	// TODO(kellegous): The specs aren't clear if this has a payload. it suggests
	// that failures are usually due to a timeout, but it doesn't say if there is
	// a code embedded in the notification data.
	return &LoginFailNotification{}, nil
}

type StatusNotification struct {
	Status Status
}

func (e *StatusNotification) NotificationCode() NotificationCode {
	return NotificationTypeStatus
}

func readStatusNotification(data []byte) (*StatusNotification, error) {
	var n StatusNotification
	if err := n.Status.readFrom(bytes.NewReader(data)); err != nil {
		return nil, poop.Chain(err)
	}
	return &n, nil
}

type BinaryResponseNotification struct {
	BinaryResponse BinaryResponse
}

func (e *BinaryResponseNotification) NotificationCode() NotificationCode {
	return NotificationTypeBinaryResponse
}

func readBinaryResponseNotification(data []byte) (*BinaryResponseNotification, error) {
	var n BinaryResponseNotification
	if err := n.BinaryResponse.readFrom(bytes.NewReader(data)); err != nil {
		return nil, poop.Chain(err)
	}
	return &n, nil
}

type TraceDataNotification struct {
	TraceData TraceData
}

func (e *TraceDataNotification) NotificationCode() NotificationCode {
	return NotificationTypeTraceData
}

func readTraceDataNotification(data []byte) (*TraceDataNotification, error) {
	var n TraceDataNotification
	r := bytes.NewReader(data)
	if err := n.TraceData.readFrom(r); err != nil {
		return nil, poop.Chain(err)
	}
	return &n, nil
}

type NewAdvertNotification struct {
	PublicKey  PublicKey
	Type       ContactType
	Flags      byte
	OutPath    []byte
	AdvName    string
	LastAdvert time.Time
	AdvLat     float64
	AdvLon     float64
}

func (e *NewAdvertNotification) NotificationCode() NotificationCode {
	return NotificationTypeNewAdvert
}

func readNewAdvertNotification(data []byte) (*NewAdvertNotification, error) {
	var n NewAdvertNotification
	r := bytes.NewReader(data)
	if err := n.PublicKey.readFrom(r); err != nil {
		return nil, poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &n.Type); err != nil {
		return nil, poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &n.Flags); err != nil {
		return nil, poop.Chain(err)
	}
	var outPathLen int8
	if err := binary.Read(r, binary.LittleEndian, &outPathLen); err != nil {
		return nil, poop.Chain(err)
	}
	var outPath [64]byte
	if _, err := io.ReadFull(r, outPath[:]); err != nil {
		return nil, poop.Chain(err)
	}
	if outPathLen > 0 {
		n.OutPath = outPath[:outPathLen]
	}
	var err error
	n.AdvName, err = readCString(r, 32)
	if err != nil {
		return nil, poop.Chain(err)
	}
	n.LastAdvert, err = readTime(r)
	if err != nil {
		return nil, poop.Chain(err)
	}
	n.AdvLat, n.AdvLon, err = readLatLon(r)
	if err != nil {
		return nil, poop.Chain(err)
	}
	return &n, nil
}

type TelemetryNotification struct {
	Telemetry Telemetry
}

func (e *TelemetryNotification) NotificationCode() NotificationCode {
	return NotificationTypeTelemetry
}

func readTelemetryNotification(data []byte) (*TelemetryNotification, error) {
	var n TelemetryNotification
	r := bytes.NewReader(data)
	if err := n.Telemetry.readFrom(r); err != nil {
		return nil, poop.Chain(err)
	}
	return &n, nil
}

type SendConfirmedNotification struct {
	ACKCode   uint32
	RoundTrip time.Duration
}

func (e *SendConfirmedNotification) NotificationCode() NotificationCode {
	return NotificationTypeSendConfirmed
}

func readSendConfirmedNotification(data []byte) (*SendConfirmedNotification, error) {
	var n SendConfirmedNotification
	r := bytes.NewReader(data)
	if err := binary.Read(r, binary.LittleEndian, &n.ACKCode); err != nil {
		return nil, poop.Chain(err)
	}
	var roundTrip uint32
	if err := binary.Read(r, binary.LittleEndian, &roundTrip); err != nil {
		return nil, poop.Chain(err)
	}
	n.RoundTrip = time.Duration(roundTrip) * time.Millisecond
	return &n, nil
}

type RawDataNotification struct {
	LastSNR  float64
	LastRSSI int8
	Payload  []byte
}

func (e *RawDataNotification) NotificationCode() NotificationCode {
	return NotificationTypeRawData
}

//	PUSH_CODE_RAW_DATA {
//	  code: byte,    // constant 0x84
//	  SNR_mult_4: signed-byte,     // SNR * 4
//	  RSSI: signed-byte,
//	  reserved: byte,     // constant 0xFF
//	  payload: bytes     // remainder of frame
//	}
func readRawDataNotification(data []byte) (*RawDataNotification, error) {
	var n RawDataNotification
	r := bytes.NewReader(data)

	var lastSnr int8
	if err := binary.Read(r, binary.LittleEndian, &lastSnr); err != nil {
		return nil, poop.Chain(err)
	}
	n.LastSNR = float64(lastSnr) / 4

	if err := binary.Read(r, binary.LittleEndian, &n.LastRSSI); err != nil {
		return nil, poop.Chain(err)
	}
	var reserved byte
	if err := binary.Read(r, binary.LittleEndian, &reserved); err != nil {
		return nil, poop.Chain(err)
	}

	if reserved != 0xff {
		return nil, poop.New("reserved byte is not 0xff")
	}

	var err error
	n.Payload, err = io.ReadAll(r)
	if err != nil {
		return nil, poop.Chain(err)
	}
	return &n, nil
}

type MsgWaitingNotification struct{}

func (e *MsgWaitingNotification) NotificationCode() NotificationCode {
	return NotificationTypeMsgWaiting
}

type LogRxDataNotification struct {
	LastSNR  float64
	LastRSSI int8
	Payload  []byte
}

func (e *LogRxDataNotification) NotificationCode() NotificationCode {
	return NotificationTypeLogRxData
}

// Not documented in the specs.
func readLogRxDataNotification(data []byte) (*LogRxDataNotification, error) {
	var n LogRxDataNotification
	r := bytes.NewReader(data)
	var lastSnr int8
	if err := binary.Read(r, binary.LittleEndian, &lastSnr); err != nil {
		return nil, poop.Chain(err)
	}
	n.LastSNR = float64(lastSnr) / 4

	if err := binary.Read(r, binary.LittleEndian, &n.LastRSSI); err != nil {
		return nil, poop.Chain(err)
	}

	var err error
	n.Payload, err = io.ReadAll(r)
	if err != nil {
		return nil, poop.Chain(err)
	}

	return &n, nil
}

//	RESP_CODE_CONTACT_MSG_RECV_V3 {
//		code: byte,   // constant 16
//		snr: byte,     // SNR*4
//		reserved: bytes(2),   // zeroes
//		pubkey_prefix: bytes(6),     // just first 6 bytes of sender's public key
//		path_len: byte,     // 0xFF if was sent direct, otherwise hop count for flood-mode
//		txt_type: byte,     // one of TXT_TYPE_*  (0 = plain)
//		sender_timestamp: uint32,
//		text: varchar    // remainder of frame
//	  }
type ContactMsgRecvV3Notification struct {
	SNR             float64
	PublicKeyPrefix [6]byte
	PathLen         byte
	TextType        TextType
	SenderTime      time.Time
	Text            string
}

func (e *ContactMsgRecvV3Notification) NotificationCode() NotificationCode {
	return NotificationTypeContactMsgRecvV3
}

func readContactMsgRecvV3Notification(data []byte) (*ContactMsgRecvV3Notification, error) {
	var n ContactMsgRecvV3Notification
	r := bytes.NewReader(data)

	var snr byte
	if err := binary.Read(r, binary.LittleEndian, &snr); err != nil {
		return nil, poop.Chain(err)
	}
	n.SNR = float64(snr) / 4

	var reserved [2]byte
	if _, err := io.ReadFull(r, reserved[:]); err != nil {
		return nil, poop.Chain(err)
	}

	if _, err := io.ReadFull(r, n.PublicKeyPrefix[:]); err != nil {
		return nil, poop.Chain(err)
	}

	if err := binary.Read(r, binary.LittleEndian, &n.PathLen); err != nil {
		return nil, poop.Chain(err)
	}

	if err := binary.Read(r, binary.LittleEndian, &n.TextType); err != nil {
		return nil, poop.Chain(err)
	}

	var err error
	n.SenderTime, err = readTime(r)
	if err != nil {
		return nil, poop.Chain(err)
	}

	n.Text, err = readString(r)
	if err != nil {
		return nil, poop.Chain(err)
	}

	n.Text, err = readString(r)
	if err != nil {
		return nil, poop.Chain(err)
	}

	return &n, nil
}

//	RESP_CODE_CHANNEL_MSG_RECV_V3 {
//		code: byte,   // constant 17
//		snr: byte,     // SNR*4
//		reserved: bytes(2),   // zeroes
//		channel_idx: byte,   // reserved (0 for now, ie. 'public')
//		path_len: byte,     // 0xFF if was sent direct, otherwise hop count for flood-mode
//		txt_type: byte,     // one of TXT_TYPE_*  (0 = plain)
//		sender_timestamp: uint32,
//		text: varchar    // remainder of frame
//	  }
type ChannelMsgRecvV3Notification struct {
	SNR          float64
	ChannelIndex byte
	PathLen      byte
	TextType     TextType
	SenderTime   time.Time
	Text         string
}

func (e *ChannelMsgRecvV3Notification) NotificationCode() NotificationCode {
	return NotificationTypeChannelMsgRecvV3
}

func readChannelMsgRecvV3Notification(data []byte) (*ChannelMsgRecvV3Notification, error) {
	var n ChannelMsgRecvV3Notification
	r := bytes.NewReader(data)

	var snr byte
	if err := binary.Read(r, binary.LittleEndian, &snr); err != nil {
		return nil, poop.Chain(err)
	}
	n.SNR = float64(snr) / 4

	var reserved [2]byte
	if _, err := io.ReadFull(r, reserved[:]); err != nil {
		return nil, poop.Chain(err)
	}

	if err := binary.Read(r, binary.LittleEndian, &n.ChannelIndex); err != nil {
		return nil, poop.Chain(err)
	}

	if err := binary.Read(r, binary.LittleEndian, &n.PathLen); err != nil {
		return nil, poop.Chain(err)
	}

	if err := binary.Read(r, binary.LittleEndian, &n.TextType); err != nil {
		return nil, poop.Chain(err)
	}

	var err error
	n.SenderTime, err = readTime(r)
	if err != nil {
		return nil, poop.Chain(err)
	}

	n.Text, err = readString(r)
	if err != nil {
		return nil, poop.Chain(err)
	}

	return &n, nil
}
