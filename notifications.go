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

// TODO(kellegous): Has to be exported?
type Notification interface {
	NotificationCode() ResponseCode
}

// TODO(kellegous): Has to be exported?
type ResponseCode byte

const (
	// Response notifications, arrive in response to a command.
	ResponseOk             ResponseCode = 0
	ResponseErr            ResponseCode = 1
	ResponseContactsStart  ResponseCode = 2
	ResponseContact        ResponseCode = 3
	ResponseEndOfContacts  ResponseCode = 4
	ResponseSelfInfo       ResponseCode = 5
	ResponseSent           ResponseCode = 6
	ResponseContactMsgRecv ResponseCode = 7
	ResponseChannelMsgRecv ResponseCode = 8
	ResponseCurrTime       ResponseCode = 9
	ResponseNoMoreMessages ResponseCode = 10
	ResponseExportContact  ResponseCode = 11
	ResponseBatteryVoltage ResponseCode = 12
	ResponseDeviceInfo     ResponseCode = 13
	ResponsePrivateKey     ResponseCode = 14
	ResponseDisabled       ResponseCode = 15
	ResponseChannelInfo    ResponseCode = 18
	ResponseSignStart      ResponseCode = 19
	ResponseSignature      ResponseCode = 20
	// Push notifications, can arrive without a corresponding command.
	ResponsePushAdvert            ResponseCode = 0x80 // when companion is set to auto add contacts
	ResponsePushPathUpdated       ResponseCode = 0x81
	ResponsePushSendConfirmed     ResponseCode = 0x82
	ResponsePushMsgWaiting        ResponseCode = 0x83
	ResponsePushRawData           ResponseCode = 0x84
	ResponsePushLoginSuccess      ResponseCode = 0x85
	ResponsePushLoginFail         ResponseCode = 0x86 // not usable yet
	ResponsePushStatusResponse    ResponseCode = 0x87
	ResponsePushLogRxData         ResponseCode = 0x88
	ResponsePushTraceData         ResponseCode = 0x89
	ResponsePushNewAdvert         ResponseCode = 0x8A // when companion is set to manually add contacts
	ResponsePushTelemetryResponse ResponseCode = 0x8B
	ResponsePushBinaryResponse    ResponseCode = 0x8C
)

type NotificationType byte

func (t NotificationType) responseCode() ResponseCode {
	return ResponseCode(t)
}

const (
	NotificationTypeAdvert            NotificationType = NotificationType(ResponsePushAdvert)
	NotificationTypePathUpdated       NotificationType = NotificationType(ResponsePushPathUpdated)
	NotificationTypeSendConfirmed     NotificationType = NotificationType(ResponsePushSendConfirmed)
	NotificationTypeMsgWaiting        NotificationType = NotificationType(ResponsePushMsgWaiting)
	NotificationTypeRawData           NotificationType = NotificationType(ResponsePushRawData)
	NotificationTypeLoginSuccess      NotificationType = NotificationType(ResponsePushLoginSuccess)
	NotificationTypeLoginFail         NotificationType = NotificationType(ResponsePushLoginFail)
	NotificationTypeStatusResponse    NotificationType = NotificationType(ResponsePushStatusResponse)
	NotificationTypeLogRxData         NotificationType = NotificationType(ResponsePushLogRxData)
	NotificationTypeTraceData         NotificationType = NotificationType(ResponsePushTraceData)
	NotificationTypeNewAdvert         NotificationType = NotificationType(ResponsePushNewAdvert)
	NotificationTypeTelemetryResponse NotificationType = NotificationType(ResponsePushTelemetryResponse)
	NotificationTypeBinaryResponse    NotificationType = NotificationType(ResponsePushBinaryResponse)
)

var responseCodeText = map[ResponseCode]string{
	ResponseOk:                    "Ok",
	ResponseErr:                   "Err",
	ResponseContactsStart:         "ContactsStart",
	ResponseContact:               "Contact",
	ResponseEndOfContacts:         "EndOfContacts",
	ResponseSelfInfo:              "SelfInfo",
	ResponseSent:                  "Sent",
	ResponseContactMsgRecv:        "ContactMsgRecv",
	ResponseChannelMsgRecv:        "ChannelMsgRecv",
	ResponseCurrTime:              "CurrTime",
	ResponseNoMoreMessages:        "NoMoreMessages",
	ResponseExportContact:         "ExportContact",
	ResponseBatteryVoltage:        "BatteryVoltage",
	ResponseDeviceInfo:            "DeviceInfo",
	ResponsePrivateKey:            "PrivateKey",
	ResponseDisabled:              "Disabled",
	ResponseChannelInfo:           "ChannelInfo",
	ResponseSignStart:             "SignStart",
	ResponseSignature:             "Signature",
	ResponsePushAdvert:            "PushAdvert",
	ResponsePushPathUpdated:       "PushPathUpdated",
	ResponsePushSendConfirmed:     "PushSendConfirmed",
	ResponsePushMsgWaiting:        "PushMsgWaiting",
	ResponsePushRawData:           "PushRawData",
	ResponsePushLoginSuccess:      "PushLoginSuccess",
	ResponsePushLoginFail:         "PushLoginFail",
	ResponsePushStatusResponse:    "PushStatusResponse",
	ResponsePushLogRxData:         "PushLogRxData",
	ResponsePushTraceData:         "PushTraceData",
	ResponsePushNewAdvert:         "PushNewAdvert",
	ResponsePushTelemetryResponse: "PushTelemetryResponse",
	ResponsePushBinaryResponse:    "PushBinaryResponse",
}

func (c ResponseCode) String() string {
	return responseCodeText[c]
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

func readNotification(code ResponseCode, data []byte) (Notification, error) {
	switch code {
	case ResponseOk:
		return readOkNotification(data)
	case ResponseErr:
		return readErrNotification(data)
	case ResponseContactsStart:
		return readContactStartNotification(data)
	case ResponseContact:
		return readContactNotification(data)
	case ResponseEndOfContacts:
		return readEndOfContactsNotification(data)
	case ResponseSelfInfo:
		return readSelfInfoNotification(data)
	case ResponseSent:
		return readSentNotification(data)
	case ResponseContactMsgRecv:
		return readContactMsgRecvNotification(data)
	case ResponseChannelMsgRecv:
		return readChannelMsgRecvNotification(data)
	case ResponseCurrTime:
		return readCurrTimeNotification(data)
	case ResponseNoMoreMessages:
		return readNoMoreMessagesNotification(data)
	case ResponseExportContact:
		return readExportContactNotification(data)
	case ResponseBatteryVoltage:
		return readBatteryVoltageNotification(data)
	case ResponseDeviceInfo:
		return readDeviceInfoNotification(data)
	case ResponsePrivateKey:
		return readPrivateKeyNotification(data)
	case ResponseDisabled:
		return readDisabledNotification(data)
	case ResponseChannelInfo:
		return readChannelInfoNotification(data)
	case ResponseSignStart:
		return readSignStartNotification(data)
	case ResponseSignature:
		return readSignatureNotification(data)
	case ResponsePushAdvert:
		return readAdvertNotification(data)
	case ResponsePushPathUpdated:
		return readPathUpdatedNotification(data)
	case ResponsePushLoginSuccess:
		return readLoginSuccessNotification(data)
	case ResponsePushStatusResponse:
		return readStatusResponseNotification(data)
	case ResponsePushBinaryResponse:
		return readBinaryResponseNotification(data)
	case ResponsePushTraceData:
		return readTraceDataNotification(data)
	case ResponsePushNewAdvert:
		return readNewAdvertNotification(data)
	case ResponsePushTelemetryResponse:
		return readTelemetryResponseNotification(data)
	}
	return nil, poop.New("unknown notification code")
}

type OkNotification struct{}

func (e *OkNotification) NotificationCode() ResponseCode {
	return ResponseOk
}

func readOkNotification(_ []byte) (*OkNotification, error) {
	return &OkNotification{}, nil
}

type ErrNotification struct {
	Code ErrorCode
}

func (e *ErrNotification) NotificationCode() ResponseCode {
	return ResponseErr
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

func (e *ContactStartNotification) NotificationCode() ResponseCode {
	return ResponseContactsStart
}

func readContactStartNotification(_ []byte) (*ContactStartNotification, error) {
	return &ContactStartNotification{}, nil
}

type ContactNotification struct {
	Contact Contact
}

func (e *ContactNotification) NotificationCode() ResponseCode {
	return ResponseContact
}

func readContactNotification(data []byte) (*ContactNotification, error) {
	var notif ContactNotification
	if err := notif.Contact.readFrom(bytes.NewReader(data)); err != nil {
		return nil, poop.Chain(err)
	}
	return &notif, nil
}

type EndOfContactsNotification struct{}

func (e *EndOfContactsNotification) NotificationCode() ResponseCode {
	return ResponseEndOfContacts
}

func readEndOfContactsNotification(_ []byte) (*EndOfContactsNotification, error) {
	return &EndOfContactsNotification{}, nil
}

type SelfInfoNotification struct {
	Type              byte
	TxPower           byte
	MaxTxPower        byte
	PublicKey         PublicKey
	AdvLat            float64
	AdvLon            float64
	ManualAddContacts byte
	RadioFreq         float64
	RadioBw           float64
	RadioSf           byte
	RadioCr           byte
	Name              string
}

func (e *SelfInfoNotification) NotificationCode() ResponseCode {
	return ResponseSelfInfo
}

func readSelfInfoNotification(data []byte) (*SelfInfoNotification, error) {
	var n SelfInfoNotification
	r := bytes.NewReader(data)
	if err := binary.Read(r, binary.LittleEndian, &n.Type); err != nil {
		return nil, poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &n.TxPower); err != nil {
		return nil, poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &n.MaxTxPower); err != nil {
		return nil, poop.Chain(err)
	}
	if err := n.PublicKey.readFrom(r); err != nil {
		return nil, poop.Chain(err)
	}
	var err error
	n.AdvLat, n.AdvLon, err = readLatLon(r)
	if err != nil {
		return nil, poop.Chain(err)
	}
	var reserved [3]byte
	if _, err := io.ReadFull(r, reserved[:]); err != nil {
		return nil, poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &n.ManualAddContacts); err != nil {
		return nil, poop.Chain(err)
	}
	var freq, bw uint32
	if err := binary.Read(r, binary.LittleEndian, &freq); err != nil {
		return nil, poop.Chain(err)
	}
	n.RadioFreq = float64(freq) / 1000
	if err := binary.Read(r, binary.LittleEndian, &bw); err != nil {
		return nil, poop.Chain(err)
	}
	n.RadioBw = float64(bw) / 1000
	if err := binary.Read(r, binary.LittleEndian, &n.RadioSf); err != nil {
		return nil, poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &n.RadioCr); err != nil {
		return nil, poop.Chain(err)
	}
	n.Name, err = readString(r)
	if err != nil {
		return nil, poop.Chain(err)
	}
	return &n, nil
}

type SentNotification struct {
	Result         int8
	ExpectedAckCRC uint32
	EstTimeout     uint32
}

func (e *SentNotification) NotificationCode() ResponseCode {
	return ResponseSent
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

func (e *ContactMsgRecvNotification) NotificationCode() ResponseCode {
	return ResponseContactMsgRecv
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

func (e *ChannelMsgRecvNotification) NotificationCode() ResponseCode {
	return ResponseChannelMsgRecv
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

func (e *CurrTimeNotification) NotificationCode() ResponseCode {
	return ResponseCurrTime
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

func (e *NoMoreMessagesNotification) NotificationCode() ResponseCode {
	return ResponseNoMoreMessages
}

func readNoMoreMessagesNotification(_ []byte) (*NoMoreMessagesNotification, error) {
	return &NoMoreMessagesNotification{}, nil
}

type ExportContactNotification struct {
	AdvertPacket []byte
}

func (e *ExportContactNotification) NotificationCode() ResponseCode {
	return ResponseExportContact
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

func (e *BatteryVoltageNotification) NotificationCode() ResponseCode {
	return ResponseBatteryVoltage
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

func (e *DeviceInfoNotification) NotificationCode() ResponseCode {
	return ResponseDeviceInfo
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

func (e *PrivateKeyNotification) NotificationCode() ResponseCode {
	return ResponsePrivateKey
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

func (e *DisabledNotification) NotificationCode() ResponseCode {
	return ResponseDisabled
}

func readDisabledNotification(_ []byte) (*DisabledNotification, error) {
	return &DisabledNotification{}, nil
}

type ChannelInfoNotification struct {
	ChannelInfo ChannelInfo
}

func (e *ChannelInfoNotification) NotificationCode() ResponseCode {
	return ResponseChannelInfo
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

func (e *SignStartNotification) NotificationCode() ResponseCode {
	return ResponseSignStart
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

func (e *SignatureNotification) NotificationCode() ResponseCode {
	return ResponseSignature
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

func (e *AdvertNotification) NotificationCode() ResponseCode {
	return ResponsePushAdvert
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

func (e *PathUpdatedNotification) NotificationCode() ResponseCode {
	return ResponsePushPathUpdated
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

func (e *LoginSuccessNotification) NotificationCode() ResponseCode {
	return ResponsePushLoginSuccess
}

func readLoginSuccessNotification(data []byte) (*LoginSuccessNotification, error) {
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

type StatusResponseNotification struct {
	PubKeyPrefix [6]byte
	StatusData   []byte
}

func (e *StatusResponseNotification) NotificationCode() ResponseCode {
	return ResponsePushStatusResponse
}

func readStatusResponseNotification(data []byte) (*StatusResponseNotification, error) {
	var n StatusResponseNotification
	r := bytes.NewReader(data)
	var reserved byte
	if err := binary.Read(r, binary.LittleEndian, &reserved); err != nil {
		return nil, poop.Chain(err)
	}
	if _, err := io.ReadFull(r, n.PubKeyPrefix[:]); err != nil {
		return nil, poop.Chain(err)
	}

	var err error
	n.StatusData, err = io.ReadAll(r)
	if err != nil {
		return nil, poop.Chain(err)
	}
	return &n, nil
}

type BinaryResponseNotification struct {
	Tag          uint32
	ResponseData []byte
}

func (e *BinaryResponseNotification) NotificationCode() ResponseCode {
	return ResponsePushBinaryResponse
}

func readBinaryResponseNotification(data []byte) (*BinaryResponseNotification, error) {
	var n BinaryResponseNotification
	r := bytes.NewReader(data)
	var reserved byte
	if err := binary.Read(r, binary.LittleEndian, &reserved); err != nil {
		return nil, poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &n.Tag); err != nil {
		return nil, poop.Chain(err)
	}
	var err error
	n.ResponseData, err = io.ReadAll(r)
	if err != nil {
		return nil, poop.Chain(err)
	}
	return &n, nil
}

type TraceDataNotification struct {
	TraceData TraceData
}

func (e *TraceDataNotification) NotificationCode() ResponseCode {
	return ResponsePushTraceData
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

func (e *NewAdvertNotification) NotificationCode() ResponseCode {
	return ResponsePushNewAdvert
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

type TelemetryResponseNotification struct {
	PubKeyPrefix  [6]byte
	LPPSensorData []byte
}

func (e *TelemetryResponseNotification) NotificationCode() ResponseCode {
	return ResponsePushTelemetryResponse
}

func readTelemetryResponseNotification(data []byte) (*TelemetryResponseNotification, error) {
	var n TelemetryResponseNotification
	r := bytes.NewReader(data)

	var reserved byte
	if err := binary.Read(r, binary.LittleEndian, &reserved); err != nil {
		return nil, poop.Chain(err)
	}
	if _, err := io.ReadFull(r, n.PubKeyPrefix[:]); err != nil {
		return nil, poop.Chain(err)
	}
	var err error
	n.LPPSensorData, err = io.ReadAll(r)
	if err != nil {
		return nil, poop.Chain(err)
	}
	return &n, nil
}
