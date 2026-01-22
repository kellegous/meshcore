package meshcore

import (
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
