package meshcore

import (
	"errors"
	"fmt"
)

type NotificationCode byte

const (
	ResponseOk             NotificationCode = 0
	ResponseErr            NotificationCode = 1
	ResponseContactsStart  NotificationCode = 2
	ResponseContact        NotificationCode = 3
	ResponseEndOfContacts  NotificationCode = 4
	ResponseSelfInfo       NotificationCode = 5
	ResponseSent           NotificationCode = 6
	ResponseContactMsgRecv NotificationCode = 7
	ResponseChannelMsgRecv NotificationCode = 8
	ResponseCurrTime       NotificationCode = 9
	ResponseNoMoreMessages NotificationCode = 10
	ResponseExportContact  NotificationCode = 11
	ResponseBatteryVoltage NotificationCode = 12
	ResponseDeviceInfo     NotificationCode = 13
	ResponsePrivateKey     NotificationCode = 14
	ResponseDisabled       NotificationCode = 15
	ResponseChannelInfo    NotificationCode = 18
	ResponseSignStart      NotificationCode = 19
	ResponseSignature      NotificationCode = 20
	PushAdvert             NotificationCode = 0x80 // when companion is set to auto add contacts
	PushPathUpdated        NotificationCode = 0x81
	PushSendConfirmed      NotificationCode = 0x82
	PushMsgWaiting         NotificationCode = 0x83
	PushRawData            NotificationCode = 0x84
	PushLoginSuccess       NotificationCode = 0x85
	PushLoginFail          NotificationCode = 0x86 // not usable yet
	PushStatusResponse     NotificationCode = 0x87
	PushLogRxData          NotificationCode = 0x88
	PushTraceData          NotificationCode = 0x89
	PushNewAdvert          NotificationCode = 0x8A // when companion is set to manually add contacts
	PushTelemetryResponse  NotificationCode = 0x8B
	PushBinaryResponse     NotificationCode = 0x8C
)

var notificationCodeText = map[NotificationCode]string{
	ResponseOk:             "Ok",
	ResponseErr:            "Err",
	ResponseContactsStart:  "ContactsStart",
	ResponseContact:        "Contact",
	ResponseEndOfContacts:  "EndOfContacts",
	ResponseSelfInfo:       "SelfInfo",
	ResponseSent:           "Sent",
	ResponseContactMsgRecv: "ContactMsgRecv",
	ResponseChannelMsgRecv: "ChannelMsgRecv",
	ResponseCurrTime:       "CurrTime",
	ResponseNoMoreMessages: "NoMoreMessages",
	ResponseExportContact:  "ExportContact",
	ResponseBatteryVoltage: "BatteryVoltage",
	ResponseDeviceInfo:     "DeviceInfo",
	ResponsePrivateKey:     "PrivateKey",
	ResponseDisabled:       "Disabled",
	ResponseChannelInfo:    "ChannelInfo",
	ResponseSignStart:      "SignStart",
	ResponseSignature:      "Signature",
	PushAdvert:             "PushAdvert",
	PushPathUpdated:        "PushPathUpdated",
	PushSendConfirmed:      "PushSendConfirmed",
	PushMsgWaiting:         "PushMsgWaiting",
	PushRawData:            "PushRawData",
	PushLoginSuccess:       "PushLoginSuccess",
	PushLoginFail:          "PushLoginFail",
	PushStatusResponse:     "PushStatusResponse",
	PushLogRxData:          "PushLogRxData",
	PushTraceData:          "PushTraceData",
	PushNewAdvert:          "PushNewAdvert",
	PushTelemetryResponse:  "PushTelemetryResponse",
	PushBinaryResponse:     "PushBinaryResponse",
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

type ResponseError struct {
	Code ErrorCode
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("response error: %d (%s)", e.Code, errorText[e.Code])
}

func hasErrorCode(err error, code ErrorCode) bool {
	var resErr *ResponseError
	if errors.As(err, &resErr) {
		return resErr.Code == code
	}
	return false
}
