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
	Type       ContactType
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

	if outPathLen > 0 {
		c.OutPath = outPath[:outPathLen]
	}

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

type ContactType byte

const (
	ContactTypeNone     ContactType = 0
	ContactTypeChat     ContactType = 1
	ContactTypeRepeater ContactType = 2
	ContactTypeRoom     ContactType = 3
)

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

type SignStartResponse struct {
	MaxSignDataLen uint32
}

func (s *SignStartResponse) readFrom(r io.Reader) error {
	var reserved byte
	if err := binary.Read(r, binary.LittleEndian, &reserved); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &s.MaxSignDataLen); err != nil {
		return poop.Chain(err)
	}
	return nil
}

type SignatureResponse struct {
	Signature [64]byte
}

func (s *SignatureResponse) readFrom(r io.Reader) error {
	if _, err := io.ReadFull(r, s.Signature[:]); err != nil {
		return poop.Chain(err)
	}
	return nil
}

type BinaryResponse struct {
	Tag          uint32
	ResponseData []byte
}

func (b *BinaryResponse) readFrom(r io.Reader) error {
	var reserved byte
	if err := binary.Read(r, binary.LittleEndian, &reserved); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &b.Tag); err != nil {
		return poop.Chain(err)
	}
	var err error
	b.ResponseData, err = io.ReadAll(r)
	if err != nil {
		return poop.Chain(err)
	}
	return nil
}

type Neighbour struct {
	PublicKeyPrefix []byte
	HeardSecondsAgo uint32
	Snr             float64
}

func (n *Neighbour) readFrom(r io.Reader, pubKeyPrefixLength byte) error {
	n.PublicKeyPrefix = make([]byte, pubKeyPrefixLength)
	if _, err := io.ReadFull(r, n.PublicKeyPrefix); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &n.HeardSecondsAgo); err != nil {
		return poop.Chain(err)
	}
	var snr int8
	if err := binary.Read(r, binary.LittleEndian, &snr); err != nil {
		return poop.Chain(err)
	}
	n.Snr = float64(snr) / 4
	return nil
}

type TraceData struct {
	PathLen    uint8
	Flags      uint8
	Tag        uint32
	AuthCode   uint32
	PathHashes []byte
	PathSnrs   []byte
	LastSnr    float64
}

func (t *TraceData) readFrom(r io.Reader) error {
	var reserved byte
	if err := binary.Read(r, binary.LittleEndian, &reserved); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &t.PathLen); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &t.Flags); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &t.Tag); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &t.AuthCode); err != nil {
		return poop.Chain(err)
	}
	t.PathHashes = make([]byte, t.PathLen)
	t.PathSnrs = make([]byte, t.PathLen)
	if _, err := io.ReadFull(r, t.PathHashes); err != nil {
		return poop.Chain(err)
	}
	if _, err := io.ReadFull(r, t.PathSnrs); err != nil {
		return poop.Chain(err)
	}
	var lastSnr int8
	if err := binary.Read(r, binary.LittleEndian, &lastSnr); err != nil {
		return poop.Chain(err)
	}
	t.LastSnr = float64(lastSnr) / 4
	return nil
}

type AdvertEvent struct {
	PublicKey PublicKey
}

func (a *AdvertEvent) readFrom(r io.Reader) error {
	if _, err := io.ReadFull(r, a.PublicKey.key[:]); err != nil {
		return poop.Chain(err)
	}
	return nil
}

type NewAdvertEvent struct {
	PublicKey  PublicKey
	Type       ContactType
	Flags      byte
	OutPath    []byte
	AdvName    string
	LastAdvert time.Time
	AdvLat     float64
	AdvLon     float64
}

func (n *NewAdvertEvent) readFrom(r io.Reader) error {
	if _, err := io.ReadFull(r, n.PublicKey.key[:]); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &n.Type); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &n.Flags); err != nil {
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
	if outPathLen > 0 {
		n.OutPath = outPath[:outPathLen]
	}
	var err error
	n.AdvName, err = readCString(r, 32)
	if err != nil {
		return poop.Chain(err)
	}
	n.LastAdvert, err = readTime(r)
	if err != nil {
		return poop.Chain(err)
	}
	n.AdvLat, n.AdvLon, err = readLatLon(r)
	if err != nil {
		return poop.Chain(err)
	}
	return nil
}

type PathUpdatedEvent struct {
	PublicKey PublicKey
}

func (p *PathUpdatedEvent) readFrom(r io.Reader) error {
	if _, err := io.ReadFull(r, p.PublicKey.key[:]); err != nil {
		return poop.Chain(err)
	}
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
		return &CommandError{Code: ErrorCodeUnknown}
	}
	return &CommandError{
		Code: ErrorCode(data[0]),
	}
}

func writeCommandCode(w io.Writer, code CommandCode) error {
	return binary.Write(w, binary.LittleEndian, byte(code))
}
