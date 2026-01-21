package meshcore

import (
	"bytes"
	"encoding/binary"
	"io"
	"time"

	"github.com/kellegous/poop"
)

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
