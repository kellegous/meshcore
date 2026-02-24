package meshcore

import (
	"encoding/hex"
	"encoding/json"
	"io"

	"github.com/kellegous/poop"
)

type PublicKey struct {
	key [32]byte
}

func (k *PublicKey) String() string {
	return hex.EncodeToString(k.key[:])
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

func (k *PublicKey) writePrefixTo(w io.Writer, n int) error {
	if _, err := w.Write(k.key[:n]); err != nil {
		return poop.Chain(err)
	}
	return nil
}

func (k *PublicKey) Bytes() []byte {
	return k.key[:]
}

func (k *PublicKey) Prefix(n int) []byte {
	return k.key[:n]
}

func (k *PublicKey) MarshalJSON() ([]byte, error) {
	s := hex.EncodeToString(k.key[:])
	return json.Marshal(s)
}
