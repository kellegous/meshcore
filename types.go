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

func ReadContact(r io.Reader) (*Contact, error) {
	var c Contact

	if err := c.PublicKey.readFrom(r); err != nil {
		return nil, poop.Chain(err)
	}

	if err := binary.Read(r, binary.LittleEndian, &c.Type); err != nil {
		return nil, err
	}

	if err := binary.Read(r, binary.LittleEndian, &c.Flags); err != nil {
		return nil, poop.Chain(err)
	}

	if err := binary.Read(r, binary.LittleEndian, &c.OutPathLen); err != nil {
		return nil, poop.Chain(err)
	}

	var outPath [64]byte
	if _, err := io.ReadFull(r, outPath[:]); err != nil {
		return nil, poop.Chain(err)
	}
	c.OutPath = outPath[:c.OutPathLen]

	var err error
	c.AdvName, err = readCString(r, 32)
	if err != nil {
		return nil, poop.Chain(err)
	}

	c.LastAdvert, err = readTime(r)
	if err != nil {
		return nil, poop.Chain(err)
	}

	c.AdvLat, err = readLocation(r)
	if err != nil {
		return nil, poop.Chain(err)
	}

	c.AdvLon, err = readLocation(r)
	if err != nil {
		return nil, poop.Chain(err)
	}

	c.LastMod, err = readTime(r)
	if err != nil {
		return nil, poop.Chain(err)
	}
	return &c, nil
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

func readLocation(r io.Reader) (float64, error) {
	var lat uint32
	if err := binary.Read(r, binary.LittleEndian, &lat); err != nil {
		return 0, poop.Chain(err)
	}
	return float64(lat) / 1e6, nil
}

func readError(data []byte) error {
	if len(data) == 0 {
		return &ResponseError{Code: ErrorCodeUnknown}
	}
	return &ResponseError{
		Code: ErrorCode(data[0]),
	}
}
