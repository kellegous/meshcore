package meshcore

import "fmt"

type ResponseCode byte

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
