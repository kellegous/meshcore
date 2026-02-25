package meshcore

import (
	"bytes"
	"encoding/binary"
	"io"
	"time"

	"github.com/kellegous/poop"
)

type SelfInfo struct {
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

func (s *SelfInfo) readFrom(r io.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, &s.Type); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &s.TxPower); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &s.MaxTxPower); err != nil {
		return poop.Chain(err)
	}
	if err := s.PublicKey.readFrom(r); err != nil {
		return poop.Chain(err)
	}
	var err error
	s.AdvLat, s.AdvLon, err = readLatLon(r)
	if err != nil {
		return poop.Chain(err)
	}
	var reserved [3]byte
	if _, err := io.ReadFull(r, reserved[:]); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &s.ManualAddContacts); err != nil {
		return poop.Chain(err)
	}
	var freq, bw uint32
	if err := binary.Read(r, binary.LittleEndian, &freq); err != nil {
		return poop.Chain(err)
	}
	s.RadioFreq = float64(freq) / 1000
	if err := binary.Read(r, binary.LittleEndian, &bw); err != nil {
		return poop.Chain(err)
	}
	s.RadioBw = float64(bw) / 1000
	if err := binary.Read(r, binary.LittleEndian, &s.RadioSf); err != nil {
		return poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &s.RadioCr); err != nil {
		return poop.Chain(err)
	}
	s.Name, err = readString(r)
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
	// TODO(kellegous): PathLen should not be a field, it should just
	// adjust the path-based slices accordingly.
	PathLen    uint8
	Flags      uint8
	Tag        uint32
	AuthCode   uint32
	PathHashes []byte
	// TODO(kellegous): These should be float64 and be divided by 4.
	PathSNRs []byte
	LastSNR  float64
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
	t.PathSNRs = make([]byte, t.PathLen)
	if _, err := io.ReadFull(r, t.PathHashes); err != nil {
		return poop.Chain(err)
	}
	if _, err := io.ReadFull(r, t.PathSNRs); err != nil {
		return poop.Chain(err)
	}
	var lastSnr int8
	if err := binary.Read(r, binary.LittleEndian, &lastSnr); err != nil {
		return poop.Chain(err)
	}
	t.LastSNR = float64(lastSnr) / 4
	return nil
}

type Status struct {
	PubKeyPrefix [6]byte
	StatusData   []byte
}

func (s *Status) readFrom(r io.Reader) error {
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
		return poop.Chain(poop.Chain(err))
	}
	return nil
}

type Telemetry struct {
	PubKeyPrefix  [6]byte
	LPPSensorData []byte
}

func (t *Telemetry) readFrom(r io.Reader) error {
	var reserved byte
	if err := binary.Read(r, binary.LittleEndian, &reserved); err != nil {
		return poop.Chain(err)
	}
	if _, err := io.ReadFull(r, t.PubKeyPrefix[:]); err != nil {
		return poop.Chain(err)
	}
	var err error
	t.LPPSensorData, err = io.ReadAll(r)
	if err != nil {
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

func writeCommandCode(w io.Writer, code CommandCode) error {
	return binary.Write(w, binary.LittleEndian, byte(code))
}
