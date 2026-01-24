package meshcore

type CommandCode byte

const (
	CommandAppStart          CommandCode = 1
	CommandSendTxtMsg        CommandCode = 2
	SendChannelTxtMsg        CommandCode = 3
	CommandGetContacts       CommandCode = 4
	CommandGetDeviceTime     CommandCode = 5
	CommandSetDeviceTime     CommandCode = 6
	CommandSendSelfAdvert    CommandCode = 7
	CommandSetAdvertName     CommandCode = 8
	CommandAddUpdateContact  CommandCode = 9
	CommandSyncNextMessage   CommandCode = 10
	CommandSetRadioParams    CommandCode = 11
	CommandSetTxPower        CommandCode = 12
	CommandResetPath         CommandCode = 13
	CommandSetAdvertLatLon   CommandCode = 14
	CommandRemoveContact     CommandCode = 15
	CommandShareContact      CommandCode = 16
	CommandExportContact     CommandCode = 17
	CommandImportContact     CommandCode = 18
	CommandReboot            CommandCode = 19
	CommandGetBatteryVoltage CommandCode = 20
	CommandSetTuningParams   CommandCode = 21 // todo
	CommandDeviceQuery       CommandCode = 22
	CommandExportPrivateKey  CommandCode = 23
	CommandImportPrivateKey  CommandCode = 24
	CommandSendRawData       CommandCode = 25
	CommandSendLogin         CommandCode = 26 // todo
	CommandSendStatusReq     CommandCode = 27 // todo
	CommandGetChannel        CommandCode = 31
	CommandSetChannel        CommandCode = 32
	CommandSignStart         CommandCode = 33
	CommandSignData          CommandCode = 34
	CommandSignFinish        CommandCode = 35
	CommandSendTracePath     CommandCode = 36
	CommandSetOtherParams    CommandCode = 38
	CommandSendTelemetryReq  CommandCode = 39
	CommandSendBinaryReq     CommandCode = 50
)

type TextType byte

const (
	TextTypePlain       TextType = 0
	TextTypeCliData     TextType = 1
	TextTypeSignedPlain TextType = 2
)

type SelfAdvertType byte

const (
	SelfAdvertTypeZeroHop SelfAdvertType = 0
	SelfAdvertTypeFlood   SelfAdvertType = 1
)
