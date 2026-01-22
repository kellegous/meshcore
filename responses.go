package meshcore

import "fmt"

type EventCode interface {
	event() byte
}

type ResponseCode byte

func (c ResponseCode) event() byte {
	return byte(c)
}

const (
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
)

type PushCode byte

func (c PushCode) event() byte {
	return byte(c)
}

const (
	PushAdvert            PushCode = 0x80 // when companion is set to auto add contacts
	PushPathUpdated       PushCode = 0x81
	PushSendConfirmed     PushCode = 0x82
	PushMsgWaiting        PushCode = 0x83
	PushRawData           PushCode = 0x84
	PushLoginSuccess      PushCode = 0x85
	PushLoginFail         PushCode = 0x86 // not usable yet
	PushStatusResponse    PushCode = 0x87
	PushLogRxData         PushCode = 0x88
	PushTraceData         PushCode = 0x89
	PushNewAdvert         PushCode = 0x8A // when companion is set to manually add contacts
	PushTelemetryResponse PushCode = 0x8B
	PushBinaryResponse    PushCode = 0x8C
)

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

type ResponseError struct {
	Code ErrorCode
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("response error: %d", e.Code)
}
