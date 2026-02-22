package meshcore

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/kellegous/poop"
)

func TestContactReadWrite(t *testing.T) {
	pk := PublicKey{key: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}}
	tests := []struct {
		Name          string
		Contact       *Contact
		ExpectedError error
	}{
		{
			Name: "success",
			Contact: &Contact{
				PublicKey:  pk,
				Type:       1,
				Flags:      2,
				OutPath:    []byte{4, 5, 6},
				AdvName:    "test",
				LastAdvert: time.Unix(420, 0),
				AdvLat:     37.774929,
				AdvLon:     -122.419416,
				LastMod:    time.Unix(666, 0),
			},
		},
		{
			Name: "outPath length is greater than 64",
			Contact: &Contact{
				PublicKey:  pk,
				Type:       1,
				Flags:      2,
				OutPath:    make([]byte, 65),
				AdvName:    "test",
				LastAdvert: time.Unix(420, 0),
				AdvLat:     37.774929,
				AdvLon:     -122.419416,
				LastMod:    time.Unix(666, 0),
			},
			ExpectedError: poop.New("outPath length is greater than 64"),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var buf bytes.Buffer

			if err := test.Contact.writeTo(&buf); !checkError(t, err, test.ExpectedError) {
				return
			}

			var b Contact
			if err := b.readFrom(bytes.NewReader(buf.Bytes())); err != nil {
				t.Fatalf("failed to read contact: %v", poop.Flatten(err))
			}

			if !reflect.DeepEqual(test.Contact, &b) {
				t.Fatalf("contact mismatch: expected %+v, got %+v", test.Contact, &b)
			}
		})
	}
}
