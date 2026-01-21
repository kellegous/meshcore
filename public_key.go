package meshcore

import (
	"io"

	"github.com/kellegous/poop"
)

type PublicKey struct {
	key [32]byte
}

func (k *PublicKey) readFrom(r io.Reader) error {
	if _, err := io.ReadFull(r, k.key[:]); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func (k *PublicKey) writeTo(w io.Writer) error {
	if _, err := w.Write(k.key[:]); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func (k *PublicKey) Prefix(n int) []byte {
	return k.key[:n]
}
