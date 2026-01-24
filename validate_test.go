package meshcore

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/kellegous/poop"
)

type Pattern struct {
	Data []byte
	Desc string
}

func Byte(b byte) Pattern {
	return Pattern{
		Data: []byte{b},
		Desc: fmt.Sprintf("byte(%d)", b),
	}
}

func Command(c CommandCode) Pattern {
	return Pattern{
		Data: []byte{byte(c)},
		Desc: fmt.Sprintf("command(%d)", c),
	}
}

func Bytes(bs ...byte) Pattern {
	return Pattern{
		Data: bs,
		Desc: fmt.Sprintf("bytes(%v)", bs),
	}
}

func String(s string) Pattern {
	return Pattern{
		Data: []byte(s),
		Desc: fmt.Sprintf("string(%q)", s),
	}
}

func Int32(i int32, e binary.ByteOrder) Pattern {
	buf := make([]byte, 4)
	e.PutUint32(buf, uint32(i))
	return Pattern{
		Data: buf,
		Desc: fmt.Sprintf("int32(%d, %s)", i, e.String()),
	}
}

func Uint32(i uint32, e binary.ByteOrder) Pattern {
	buf := make([]byte, 4)
	e.PutUint32(buf, i)
	return Pattern{
		Data: buf,
		Desc: fmt.Sprintf("uint32(%d, %s)", i, e.String()),
	}
}

func Time(t time.Time, e binary.ByteOrder) Pattern {
	buf := make([]byte, 4)
	e.PutUint32(buf, uint32(t.Unix()))
	return Pattern{
		Data: buf,
		Desc: fmt.Sprintf("time(%s, %s)", t.Format(time.RFC3339), e.String()),
	}
}

func LatLon(lat float64, lon float64, e binary.ByteOrder) Pattern {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint32(buf, uint32(lat*1e6))
	binary.BigEndian.PutUint32(buf[4:], uint32(lon*1e6))
	return Pattern{
		Data: buf,
		Desc: fmt.Sprintf("latlon(%f, %f, %s)", lat, lon, e.String()),
	}
}

func CString(s string, maxLen int) Pattern {
	if len(s) > maxLen-1 {
		panic("s is longer than maxLen")
	}

	buf := make([]byte, maxLen)
	copy(buf, s)
	buf[len(s)] = 0
	return Pattern{
		Data: buf,
		Desc: fmt.Sprintf("cstring(%q, %d)", s, maxLen),
	}
}

func BytesFrom(patterns ...Pattern) []byte {
	var buf bytes.Buffer
	for _, pattern := range patterns {
		buf.Write(pattern.Data)
	}
	return buf.Bytes()
}

func ValidateBytes(
	got []byte,
	patterns ...Pattern,
) error {
	var log []string
	for i, pattern := range patterns {
		n := len(pattern.Data)
		if len(got) < n {
			log = append(
				log,
				fmt.Sprintf("%d:%s not enough bytes", i, pattern.Desc),
			)
			return poop.New(strings.Join(log, ", "))
		}

		if !bytes.HasPrefix(got, pattern.Data) {
			log = append(
				log,
				fmt.Sprintf("%d:%s 👎", i, pattern.Desc),
			)
			return poop.New(strings.Join(log, ", "))
		}

		log = append(
			log,
			fmt.Sprintf("%d:%s 👍", i, pattern.Desc),
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
