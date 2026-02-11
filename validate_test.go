package meshcore

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/kellegous/poop"
)

type Pattern interface {
	Len() int
	Desc() string
	Compare(got []byte) bool
}

type LiteralPattern interface {
	Pattern
	Bytes() []byte
}

type fromBytes struct {
	data []byte
	desc string
}

var _ LiteralPattern = (*fromBytes)(nil)

func (f *fromBytes) Len() int {
	return len(f.data)
}

func (f *fromBytes) Desc() string {
	return f.desc
}

func (f *fromBytes) Compare(got []byte) bool {
	return bytes.Equal(got, f.data)
}

func (f *fromBytes) Bytes() []byte {
	return f.data
}

type anyBytes struct {
	len  int
	desc string
}

var _ Pattern = (*anyBytes)(nil)

func (a *anyBytes) Len() int {
	return a.len
}

func (a *anyBytes) Desc() string {
	return a.desc
}

func (a *anyBytes) Compare(got []byte) bool {
	return len(got) == a.len
}

func Byte(b byte) LiteralPattern {
	return &fromBytes{
		data: []byte{b},
		desc: fmt.Sprintf("byte(%d)", b),
	}
}

func Bool(b bool) LiteralPattern {
	return &fromBytes{
		data: []byte{boolToByte(b)},
		desc: fmt.Sprintf("bool(%t)", b),
	}
}

func Command(c CommandCode) LiteralPattern {
	return &fromBytes{
		data: []byte{byte(c)},
		desc: fmt.Sprintf("command(%d)", c),
	}
}

func Bytes(bs ...byte) LiteralPattern {
	return &fromBytes{
		data: bs,
		desc: fmt.Sprintf("bytes(%v)", bs),
	}
}

func String(s string) LiteralPattern {
	return &fromBytes{
		data: []byte(s),
		desc: fmt.Sprintf("string(%q)", s),
	}
}

func Int32(i int32, e binary.ByteOrder) LiteralPattern {
	buf := make([]byte, 4)
	e.PutUint32(buf, uint32(i))
	return &fromBytes{
		data: buf,
		desc: fmt.Sprintf("int32(%d, %s)", i, e.String()),
	}
}

func Uint32(i uint32, e binary.ByteOrder) LiteralPattern {
	buf := make([]byte, 4)
	e.PutUint32(buf, i)
	return &fromBytes{
		data: buf,
		desc: fmt.Sprintf("uint32(%d, %s)", i, e.String()),
	}
}

func Uint16(i uint16, e binary.ByteOrder) LiteralPattern {
	buf := make([]byte, 2)
	e.PutUint16(buf, i)
	return &fromBytes{
		data: buf,
		desc: fmt.Sprintf("uint16(%d, %s)", i, e.String()),
	}
}

func Int16(i int16, e binary.ByteOrder) LiteralPattern {
	return Uint16(uint16(i), e)
}

func Time(t time.Time, e binary.ByteOrder) LiteralPattern {
	buf := make([]byte, 4)
	e.PutUint32(buf, uint32(t.Unix()))
	return &fromBytes{
		data: buf,
		desc: fmt.Sprintf("time(%s, %s)", t.Format(time.RFC3339), e.String()),
	}
}

func LatLon(lat float64, lon float64, e binary.ByteOrder) LiteralPattern {
	buf := make([]byte, 8)
	e.PutUint32(buf, uint32(lat*1e6))
	e.PutUint32(buf[4:], uint32(lon*1e6))
	return &fromBytes{
		data: buf,
		desc: fmt.Sprintf("latlon(%f, %f, %s)", lat, lon, e.String()),
	}
}

func AnyBytes(n int) Pattern {
	return &anyBytes{
		len:  n,
		desc: fmt.Sprintf("any bytes(%d)", n),
	}
}

func CString(s string, maxLen int) LiteralPattern {
	if len(s) > maxLen-1 {
		panic("s is longer than maxLen")
	}

	buf := make([]byte, maxLen)
	copy(buf, s)
	buf[len(s)] = 0
	return &fromBytes{
		data: buf,
		desc: fmt.Sprintf("cstring(%q, %d)", s, maxLen),
	}
}

func BytesFrom(patterns ...LiteralPattern) []byte {
	var buf bytes.Buffer
	for _, pattern := range patterns {
		buf.Write(pattern.Bytes())
	}
	return buf.Bytes()
}

func ValidateBytes(
	got []byte,
	patterns ...Pattern,
) error {
	var log []string
	for i, pattern := range patterns {
		n := pattern.Len()
		if len(got) < n {
			log = append(
				log,
				fmt.Sprintf("%d:%s not enough bytes", i, pattern.Desc()),
			)
			return poop.New(strings.Join(log, ", "))
		}

		if !pattern.Compare(got[:n]) {
			log = append(
				log,
				fmt.Sprintf("%d:%s 👎", i, pattern.Desc()),
			)
			return poop.New(strings.Join(log, ", "))
		}

		log = append(
			log,
			fmt.Sprintf("%d:%s 👍", i, pattern.Desc()),
		)

		got = got[n:]
	}

	if len(got) > 0 {
		log = append(
			log,
			fmt.Sprintf("extra bytes: %v", got),
		)
		return poop.New(strings.Join(log, ", "))
	}

	return nil
}
