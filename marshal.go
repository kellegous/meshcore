package meshcore

import (
	"bytes"
	"encoding/binary"
	"io"
	"time"

	"github.com/kellegous/poop"
)

type Contact struct {
	PublicKey  PublicKey
	Type       byte
	Flags      byte
	OutPath    []byte
	AdvName    string
	LastAdvert time.Time
	AdvLat     float64
	AdvLon     float64
	LastMod    time.Time
}

func (c *Contact) writeTo(w io.Writer) error {
	if len(c.OutPath) > 64 {
		return poop.Newf("outPath length is greater than 64")
	}

	outPath := make([]byte, 64)
	copy(outPath, c.OutPath)

	if err := c.PublicKey.writeTo(w); err != nil {
		return poop.Chain(err)
	}

	if err := binary.Write(w, binary.LittleEndian, c.Type); err != nil {
		return poop.Chain(err)
	}

	if err := binary.Write(w, binary.LittleEndian, c.Flags); err != nil {
		return poop.Chain(err)
	}

	if err := binary.Write(w, binary.LittleEndian, int8(len(c.OutPath))); err != nil {
		return poop.Chain(err)
	}

	if _, err := w.Write(outPath); err != nil {
		return poop.Chain(err)
	}

	if err := writeCString(w, c.AdvName, 32); err != nil {
		return poop.Chain(err)
	}

	if err := writeTime(w, c.LastAdvert); err != nil {
		return poop.Chain(err)
	}

	if err := writeLatLon(w, c.AdvLat, c.AdvLon); err != nil {
		return poop.Chain(err)
	}

	if err := writeTime(w, c.LastMod); err != nil {
		return poop.Chain(err)
	}

	return nil
}

func (c *Contact) readFrom(r io.Reader) error {
	if err := c.PublicKey.readFrom(r); err != nil {
		return poop.Chain(err)
	}

	if err := binary.Read(r, binary.LittleEndian, &c.Type); err != nil {
		return poop.Chain(err)
	}

	if err := binary.Read(r, binary.LittleEndian, &c.Flags); err != nil {
		return poop.Chain(err)
	}

	var outPathLen int8
	if err := binary.Read(r, binary.LittleEndian, &outPathLen); err != nil {
		return poop.Chain(err)
	}

	var outPath [64]byte
	if _, err := io.ReadFull(r, outPath[:]); err != nil {
		return poop.Chain(err)
	}
	c.OutPath = outPath[:outPathLen]

	var err error
	c.AdvName, err = readCString(r, 32)
	if err != nil {
		return poop.Chain(err)
	}

	c.LastAdvert, err = readTime(r)
	if err != nil {
		return poop.Chain(err)
	}

	c.AdvLat, c.AdvLon, err = readLatLon(r)
	if err != nil {
		return poop.Chain(err)
	}

	c.LastMod, err = readTime(r)
	if err != nil {
		return poop.Chain(err)
	}

	return nil
}

type SentResponse struct {
	Result         int8
	ExpectedAckCRC uint32
	EstTimeout     uint32
}

func (s *SentResponse) readFrom(r io.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, &s.Result); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &s.ExpectedAckCRC); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &s.EstTimeout); err != nil {
		return poop.Chain(err)
	}
	return nil
}

type TelemetryResponse struct {
	// Reserved byte
	pubKeyPrefix  [6]byte
	LPPSensorData []byte
}

func (t *TelemetryResponse) readFrom(r io.Reader) error {
	var reserved byte
	if err := binary.Read(r, binary.LittleEndian, &reserved); err != nil {
		return poop.Chain(err)
	}

	if _, err := io.ReadFull(r, t.pubKeyPrefix[:]); err != nil {
		return poop.Chain(err)
	}

	var err error
	t.LPPSensorData, err = io.ReadAll(r)
	if err != nil {
		return poop.Chain(err)
	}

	return nil
}

type ChannelInfo struct {
	Index  uint8
	Name   string
	Secret []byte
}

func (c *ChannelInfo) readFrom(r io.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, &c.Index); err != nil {
		return poop.Chain(err)
	}

	var err error
	c.Name, err = readCString(r, 32)
	if err != nil {
		return poop.Chain(err)
	}

	c.Secret, err = io.ReadAll(r)
	if err != nil {
		return poop.Chain(err)
	} else if len(c.Secret) != 16 {
		return poop.Newf("secret length is not 16")
	}

	return nil
}

type DeviceInfo struct {
	FirmwareVersion   int8
	FirmwareBuildDate string
	ManufacturerModel string
}

func (d *DeviceInfo) readFrom(r io.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, &d.FirmwareVersion); err != nil {
		return poop.Chain(err)
	}
	var reserved [6]byte
	if _, err := io.ReadFull(r, reserved[:]); err != nil {
		return poop.Chain(err)
	}
	var err error
	d.FirmwareBuildDate, err = readCString(r, 12)
	if err != nil {
		return poop.Chain(err)
	}
	d.ManufacturerModel, err = readString(r)
	if err != nil {
		return poop.Chain(err)
	}
	return nil
}

type Message interface {
	FromContact() *ContactMessage
	FromChannel() *ChannelMessage
}

type ContactMessage struct {
	PubKeyPrefix [6]byte
	PathLen      byte
	TextType     TextType
	SenderTime   time.Time
	Text         string
}

func (c *ContactMessage) FromContact() *ContactMessage {
	return c
}

func (c *ContactMessage) FromChannel() *ChannelMessage {
	return nil
}

func (c *ContactMessage) readFrom(r io.Reader) error {
	if _, err := io.ReadFull(r, c.PubKeyPrefix[:]); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &c.PathLen); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &c.TextType); err != nil {
		return poop.Chain(err)
	}
	var err error
	c.SenderTime, err = readTime(r)
	if err != nil {
		return poop.Chain(err)
	}
	c.Text, err = readString(r)
	if err != nil {
		return poop.Chain(err)
	}
	return nil
}

type ChannelMessage struct {
	ChannelIndex byte
	PathLen      byte
	TextType     TextType
	SenderTime   time.Time
	Text         string
}

func (c *ChannelMessage) FromContact() *ContactMessage {
	return nil
}

func (c *ChannelMessage) FromChannel() *ChannelMessage {
	return c
}

func (c *ChannelMessage) readFrom(r io.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, &c.ChannelIndex); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &c.PathLen); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &c.TextType); err != nil {
		return poop.Chain(err)
	}
	var err error
	c.SenderTime, err = readTime(r)
	if err != nil {
		return poop.Chain(err)
	}
	c.Text, err = readString(r)
	if err != nil {
		return poop.Chain(err)
	}
	return nil
}

type StatusResponse struct {
	PubKeyPrefix [6]byte
	StatusData   []byte
}

func (s *StatusResponse) readFrom(r io.Reader) error {
	var reserved byte
	if err := binary.Read(r, binary.LittleEndian, &reserved); err != nil {
		return poop.Chain(err)
	}

	if _, err := io.ReadFull(r, s.PubKeyPrefix[:]); err != nil {
		return poop.Chain(err)
	}

	var err error
	s.StatusData, err = io.ReadAll(r)
	if err != nil {
		return poop.Chain(err)
	}

	return nil
}

type SendResponse struct {
	Type           int8   // 1 = flood, 0 = direct
	ExpectedAckCRC uint32 // can also serve as a tag
	EstTimeout     time.Duration
}

func (s *SendResponse) readFrom(r io.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, &s.Type); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &s.ExpectedAckCRC); err != nil {
		return poop.Chain(err)
	}
	var timeout uint32
	if err := binary.Read(r, binary.LittleEndian, &timeout); err != nil {
		return poop.Chain(err)
	}
	s.EstTimeout = time.Duration(timeout) * time.Millisecond
	return nil
}

func readCString(r io.Reader, maxLen int) (string, error) {
	buf := make([]byte, maxLen)
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return "", poop.Chain(err)
	}

	ix := bytes.Index(buf[:], []byte{0})
	if ix == -1 {
		return "", poop.New("cstring is not null-terminated")
	}

	return string(buf[:ix]), nil
}

func writeCString(w io.Writer, s string, maxLen int) error {
	if len(s) > maxLen-1 {
		return poop.Newf("string is longer than max length")
	}

	buf := make([]byte, maxLen)
	copy(buf[:len(s)], s)
	if _, err := w.Write(buf); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func readString(r io.Reader) (string, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return "", poop.Chain(err)
	}
	return string(b), nil
}

func writeString(w io.Writer, s string) error {
	if _, err := w.Write([]byte(s)); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func readTime(r io.Reader) (time.Time, error) {
	var ts uint32
	if err := binary.Read(r, binary.LittleEndian, &ts); err != nil {
		return time.Time{}, poop.Chain(err)
	}
	return time.Unix(int64(ts), 0), nil
}

func writeTime(w io.Writer, t time.Time) error {
	if t.IsZero() {
		t = time.Unix(0, 0)
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(t.Unix())); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func readLatLon(r io.Reader) (float64, float64, error) {
	var lat, lon int32
	if err := binary.Read(r, binary.LittleEndian, &lat); err != nil {
		return 0, 0, poop.Chain(err)
	}

	if err := binary.Read(r, binary.LittleEndian, &lon); err != nil {
		return 0, 0, poop.Chain(err)
	}

	return float64(lat) / 1e6, float64(lon) / 1e6, nil
}

func writeLatLon(w io.Writer, lat, lon float64) error {
	if err := binary.Write(w, binary.LittleEndian, uint32(lat*1e6)); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(lon*1e6)); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func readError(data []byte) error {
	if len(data) == 0 {
		return &ResponseError{Code: ErrorCodeUnknown}
	}
	return &ResponseError{
		Code: ErrorCode(data[0]),
	}
}

func writeCommandCode(w io.Writer, code CommandCode) error {
	return binary.Write(w, binary.LittleEndian, byte(code))
}

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
