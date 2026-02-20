package meshcore

import (
	"errors"
	"fmt"
)

type Notification interface {
	NotificationCode() NotificationCode
}

type NotificationCode byte

const (
	// Response notifications, arrive in response to a command.
	NotificationTypeOk             NotificationCode = 0
	NotificationTypeErr            NotificationCode = 1
	NotificationTypeContactsStart  NotificationCode = 2
	NotificationTypeContact        NotificationCode = 3
	NotificationTypeEndOfContacts  NotificationCode = 4
	NotificationTypeSelfInfo       NotificationCode = 5
	NotificationTypeSent           NotificationCode = 6
	NotificationTypeContactMsgRecv NotificationCode = 7
	NotificationTypeChannelMsgRecv NotificationCode = 8
	NotificationTypeCurrTime       NotificationCode = 9
	NotificationTypeNoMoreMessages NotificationCode = 10
	NotificationTypeExportContact  NotificationCode = 11
	NotificationTypeBatteryVoltage NotificationCode = 12
	NotificationTypeDeviceInfo     NotificationCode = 13
	NotificationTypePrivateKey     NotificationCode = 14
	NotificationTypeDisabled       NotificationCode = 15
	NotificationTypeChannelInfo    NotificationCode = 18
	NotificationTypeSignStart      NotificationCode = 19
	NotificationTypeSignature      NotificationCode = 20
	// Push notifications, can arrive without a corresponding command.
	NotificationTypeAdvert            NotificationCode = 0x80 // when companion is set to auto add contacts
	NotificationTypePathUpdated       NotificationCode = 0x81
	NotificationTypeSendConfirmed     NotificationCode = 0x82
	NotificationTypeMsgWaiting        NotificationCode = 0x83
	NotificationTypeRawData           NotificationCode = 0x84
	NotificationTypeLoginSuccess      NotificationCode = 0x85
	NotificationTypeLoginFail         NotificationCode = 0x86 // not usable yet
	NotificationTypeStatusResponse    NotificationCode = 0x87
	NotificationTypeLogRxData         NotificationCode = 0x88
	NotificationTypeTraceData         NotificationCode = 0x89
	NotificationTypeNewAdvert         NotificationCode = 0x8A // when companion is set to manually add contacts
	NotificationTypeTelemetryResponse NotificationCode = 0x8B
	NotificationTypeBinaryResponse    NotificationCode = 0x8C
)

var notificationCodeText = map[NotificationCode]string{
	NotificationTypeOk:                "Ok",
	NotificationTypeErr:               "Err",
	NotificationTypeContactsStart:     "ContactsStart",
	NotificationTypeContact:           "Contact",
	NotificationTypeEndOfContacts:     "EndOfContacts",
	NotificationTypeSelfInfo:          "SelfInfo",
	NotificationTypeSent:              "Sent",
	NotificationTypeContactMsgRecv:    "ContactMsgRecv",
	NotificationTypeChannelMsgRecv:    "ChannelMsgRecv",
	NotificationTypeCurrTime:          "CurrTime",
	NotificationTypeNoMoreMessages:    "NoMoreMessages",
	NotificationTypeExportContact:     "ExportContact",
	NotificationTypeBatteryVoltage:    "BatteryVoltage",
	NotificationTypeDeviceInfo:        "DeviceInfo",
	NotificationTypePrivateKey:        "PrivateKey",
	NotificationTypeDisabled:          "Disabled",
	NotificationTypeChannelInfo:       "ChannelInfo",
	NotificationTypeSignStart:         "SignStart",
	NotificationTypeSignature:         "Signature",
	NotificationTypeAdvert:            "PushAdvert",
	NotificationTypePathUpdated:       "PushPathUpdated",
	NotificationTypeSendConfirmed:     "PushSendConfirmed",
	NotificationTypeMsgWaiting:        "PushMsgWaiting",
	NotificationTypeRawData:           "PushRawData",
	NotificationTypeLoginSuccess:      "PushLoginSuccess",
	NotificationTypeLoginFail:         "PushLoginFail",
	NotificationTypeStatusResponse:    "PushStatusResponse",
	NotificationTypeLogRxData:         "PushLogRxData",
	NotificationTypeTraceData:         "PushTraceData",
	NotificationTypeNewAdvert:         "PushNewAdvert",
	NotificationTypeTelemetryResponse: "PushTelemetryResponse",
	NotificationTypeBinaryResponse:    "PushBinaryResponse",
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

type ErrNotification struct {
	Code ErrorCode
}

func (e *ErrNotification) Error() error {
	return &CommandError{Code: e.Code}
}

func hasErrorCode(err error, code ErrorCode) bool {
	var resErr *CommandError
	if errors.As(err, &resErr) {
		return resErr.Code == code
	}
	return false
}
