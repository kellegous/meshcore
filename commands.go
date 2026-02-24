package meshcore

import (
	"bytes"
	"encoding/binary"
	"io"
	"time"

	"github.com/kellegous/poop"
)

type CommandCode byte

const (
	CommandAppStart          CommandCode = 1
	CommandSendTxtMsg        CommandCode = 2
	CommandSendChannelTxtMsg CommandCode = 3
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

var commandCodeText = map[CommandCode]string{
	CommandAppStart:          "AppStart",
	CommandSendTxtMsg:        "SendTxtMsg",
	CommandSendChannelTxtMsg: "SendChannelTxtMsg",
	CommandGetContacts:       "GetContacts",
	CommandGetDeviceTime:     "GetDeviceTime",
	CommandSetDeviceTime:     "SetDeviceTime",
	CommandSendSelfAdvert:    "SendSelfAdvert",
	CommandSetAdvertName:     "SetAdvertName",
	CommandAddUpdateContact:  "AddUpdateContact",
	CommandSyncNextMessage:   "SyncNextMessage",
	CommandSetRadioParams:    "SetRadioParams",
	CommandSetTxPower:        "SetTxPower",
	CommandResetPath:         "ResetPath",
	CommandSetAdvertLatLon:   "SetAdvertLatLon",
	CommandRemoveContact:     "RemoveContact",
	CommandShareContact:      "ShareContact",
	CommandExportContact:     "ExportContact",
	CommandImportContact:     "ImportContact",
	CommandReboot:            "Reboot",
	CommandGetBatteryVoltage: "GetBatteryVoltage",
	CommandSetTuningParams:   "SetTuningParams",
	CommandDeviceQuery:       "DeviceQuery",
	CommandExportPrivateKey:  "ExportPrivateKey",
	CommandImportPrivateKey:  "ImportPrivateKey",
	CommandSendRawData:       "SendRawData",
	CommandSendLogin:         "SendLogin",
	CommandSendStatusReq:     "SendStatusReq",
	CommandGetChannel:        "GetChannel",
	CommandSetChannel:        "SetChannel",
	CommandSignStart:         "SignStart",
	CommandSignData:          "SignData",
	CommandSignFinish:        "SignFinish",
	CommandSendTracePath:     "SendTracePath",
	CommandSetOtherParams:    "SetOtherParams",
	CommandSendTelemetryReq:  "SendTelemetryReq",
	CommandSendBinaryReq:     "SendBinaryReq",
}

func (c CommandCode) String() string {
	return commandCodeText[c]
}

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

type BinaryRequestType byte

const (
	BinaryRequestTypeGetTelemetryData BinaryRequestType = 0x03
	BinaryRequestTypeGetAvgMinMax     BinaryRequestType = 0x04
	BinaryRequestTypeGetAccessList    BinaryRequestType = 0x05
	BinaryRequestTypeGetNeighbours    BinaryRequestType = 0x06
)

func writeAddOrUpdateContactCommand(w io.Writer, contact *Contact) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandAddUpdateContact); err != nil {
		return poop.Chain(err)
	}
	if err := contact.writeTo(&buf); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeGetContactsCommand(w io.Writer, since time.Time) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandGetContacts); err != nil {
		return poop.Chain(err)
	}
	if !since.IsZero() {
		if err := writeTime(&buf, since); err != nil {
			return poop.Chain(err)
		}
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeSendTextMessageCommand(
	w io.Writer,
	recipient *PublicKey,
	message string,
	textType TextType,
	attempt byte,
	sendTime time.Time,
) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandSendTxtMsg); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, byte(textType)); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, attempt); err != nil {
		return poop.Chain(err)
	}
	if err := writeTime(&buf, sendTime); err != nil {
		return poop.Chain(err)
	}
	if err := recipient.writePrefixTo(&buf, 6); err != nil {
		return poop.Chain(err)
	}
	if _, err := buf.Write([]byte(message)); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeSendChannelTextMessageCommand(
	w io.Writer,
	channelIndex byte,
	message string,
	textType TextType,
	sendTime time.Time,
) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandSendChannelTxtMsg); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, byte(textType)); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, channelIndex); err != nil {
		return poop.Chain(err)
	}
	if err := writeTime(&buf, sendTime); err != nil {
		return poop.Chain(err)
	}
	if err := writeString(&buf, message); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeRemoveContactCommand(w io.Writer, key *PublicKey) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandRemoveContact); err != nil {
		return poop.Chain(err)
	}
	if err := key.writeTo(&buf); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeGetTelemetryCommand(w io.Writer, key *PublicKey) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandSendTelemetryReq); err != nil {
		return poop.Chain(err)
	}
	// reserved bytes
	if _, err := buf.Write([]byte{0, 0, 0}); err != nil {
		return poop.Chain(err)
	}
	if err := key.writeTo(&buf); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeGetChannelCommand(w io.Writer, idx uint8) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandGetChannel); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, idx); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeSetChannelCommand(w io.Writer, channel *ChannelInfo) error {
	if len(channel.Secret) != 16 {
		return poop.Newf("secret length must be 16")
	}

	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandSetChannel); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, channel.Index); err != nil {
		return poop.Chain(err)
	}
	if err := writeCString(&buf, channel.Name, 32); err != nil {
		return poop.Chain(err)
	}
	if _, err := buf.Write(channel.Secret); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeDeviceQueryCommand(w io.Writer, appTargetVer byte) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandDeviceQuery); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, appTargetVer); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeRebootCommand(w io.Writer) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandReboot); err != nil {
		return poop.Chain(err)
	}
	if err := writeString(&buf, "reboot"); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeSendAdvertCommand(w io.Writer, advertType SelfAdvertType) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandSendSelfAdvert); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, byte(advertType)); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeExportContactCommand(w io.Writer, key *PublicKey) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandExportContact); err != nil {
		return poop.Chain(err)
	}
	if key != nil {
		if err := key.writeTo(&buf); err != nil {
			return poop.Chain(err)
		}
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeImportContactCommand(w io.Writer, advertPacket []byte) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandImportContact); err != nil {
		return poop.Chain(err)
	}
	if _, err := buf.Write(advertPacket); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeShareContactCommand(w io.Writer, key *PublicKey) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandShareContact); err != nil {
		return poop.Chain(err)
	}
	if err := key.writeTo(&buf); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeGetStatusCommand(w io.Writer, key *PublicKey) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandSendStatusReq); err != nil {
		return poop.Chain(err)
	}
	if err := key.writeTo(&buf); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeImportPrivateKeyCommand(w io.Writer, privateKey []byte) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandImportPrivateKey); err != nil {
		return poop.Chain(err)
	}
	if _, err := buf.Write(privateKey); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeSetAdvertLatLonCommand(w io.Writer, lat, lon float64) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandSetAdvertLatLon); err != nil {
		return poop.Chain(err)
	}
	if err := writeLatLon(&buf, lat, lon); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeSetAdvertNameCommand(w io.Writer, name string) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandSetAdvertName); err != nil {
		return poop.Chain(err)
	}
	if err := writeString(&buf, name); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeSetDeviceTimeCommand(w io.Writer, time time.Time) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandSetDeviceTime); err != nil {
		return poop.Chain(err)
	}
	if err := writeTime(&buf, time); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeResetPathCommand(w io.Writer, key *PublicKey) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandResetPath); err != nil {
		return poop.Chain(err)
	}
	if err := key.writeTo(&buf); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeSignDataCommand(w io.Writer, data []byte) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandSignData); err != nil {
		return poop.Chain(err)
	}
	if _, err := buf.Write(data); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeCommandAppStartCommand(w io.Writer) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandAppStart); err != nil {
		return poop.Chain(err)
	}
	var appVer byte = 1
	if err := binary.Write(&buf, binary.LittleEndian, appVer); err != nil {
		return poop.Chain(err)
	}
	var reserved [6]byte
	if _, err := buf.Write(reserved[:]); err != nil {
		return poop.Chain(err)
	}
	appName := "test"
	if err := writeString(&buf, appName); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeSetRadioParamsCommand(w io.Writer, radioFreq uint32, radioBw uint32, radioSf byte, radioCr byte) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandSetRadioParams); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, radioFreq); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, radioBw); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, radioSf); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, radioCr); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeSendBinaryRequestCommand(w io.Writer, recipient PublicKey, payload []byte) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandSendBinaryReq); err != nil {
		return poop.Chain(err)
	}
	if err := recipient.writeTo(&buf); err != nil {
		return poop.Chain(err)
	}
	if _, err := buf.Write(payload); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeSetTXPowerCommand(w io.Writer, power byte) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandSetTxPower); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, power); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func boolToByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

func writeSetOtherParamsCommand(w io.Writer, manualAddContacts bool) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandSetOtherParams); err != nil {
		return poop.Chain(err)
	}

	if err := binary.Write(&buf, binary.LittleEndian, boolToByte(manualAddContacts)); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeSendTracePathCommand(w io.Writer, tag uint32, auth uint32, path []byte) error {
	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandSendTracePath); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, tag); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, auth); err != nil {
		return poop.Chain(err)
	}
	// flags
	if err := binary.Write(&buf, binary.LittleEndian, byte(0)); err != nil {
		return poop.Chain(err)
	}
	if _, err := buf.Write(path); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func writeLoginCommand(w io.Writer, key PublicKey, password string) error {
	if len(password) > 15 {
		return poop.New("password is too long (max 15 characters)")
	}

	var buf bytes.Buffer
	if err := writeCommandCode(&buf, CommandSendLogin); err != nil {
		return poop.Chain(err)
	}
	if err := key.writeTo(&buf); err != nil {
		return poop.Chain(err)
	}
	if err := writeString(&buf, password); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(buf.Bytes()); err != nil {
		return poop.Chain(err)
	}
	return nil
}
