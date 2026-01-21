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
	OutPathLen int8
	OutPath    []byte
	AdvName    string
	LastAdvert time.Time
	AdvLat     float64
	AdvLon     float64
	LastMod    time.Time
}

func (c *Contact) writeTo(w io.Writer) error {
	if int(c.OutPathLen) > len(c.OutPath) {
		return poop.Newf("outPathLen is greater than outPath length")
	}

	if c.OutPathLen < 0 {
		return poop.Newf("outPathLen is less than 0")
	}

	if c.OutPathLen > 64 {
		return poop.Newf("outPathLen is greater than 64")
	}

	outPath := make([]byte, 64)
	copy(outPath[:c.OutPathLen], c.OutPath)

	if err := c.PublicKey.writeTo(w); err != nil {
		return poop.Chain(err)
	}

	if err := binary.Write(w, binary.LittleEndian, c.Type); err != nil {
		return poop.Chain(err)
	}

	if err := binary.Write(w, binary.LittleEndian, c.Flags); err != nil {
		return poop.Chain(err)
	}

	if err := binary.Write(w, binary.LittleEndian, c.OutPathLen); err != nil {
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

	if err := binary.Read(r, binary.LittleEndian, &c.OutPathLen); err != nil {
		return poop.Chain(err)
	}

	var outPath [64]byte
	if _, err := io.ReadFull(r, outPath[:]); err != nil {
		return poop.Chain(err)
	}
	c.OutPath = outPath[:c.OutPathLen]

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

func ReadSentResponse(r io.Reader) (*SentResponse, error) {
	var sr SentResponse
	if err := binary.Read(r, binary.LittleEndian, &sr.Result); err != nil {
		return nil, poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &sr.ExpectedAckCRC); err != nil {
		return nil, poop.Chain(err)
	}
	if err := binary.Read(r, binary.LittleEndian, &sr.EstTimeout); err != nil {
		return nil, poop.Chain(err)
	}
	return &sr, nil
}

func ReadTime(r io.Reader) (time.Time, error) {
	var ts uint32
	if err := binary.Read(r, binary.LittleEndian, &ts); err != nil {
		return time.Time{}, poop.Chain(err)
	}
	return time.Unix(int64(ts), 0), nil
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

func readTime(r io.Reader) (time.Time, error) {
	var ts uint32
	if err := binary.Read(r, binary.LittleEndian, &ts); err != nil {
		return time.Time{}, poop.Chain(err)
	}
	return time.Unix(int64(ts), 0), nil
}

func writeTime(w io.Writer, t time.Time) error {
	if err := binary.Write(w, binary.LittleEndian, uint32(t.Unix())); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func readLatLon(r io.Reader) (float64, float64, error) {
	var lat, lon uint32
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
