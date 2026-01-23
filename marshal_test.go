package meshcore

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/kellegous/poop"
)

func checkError(t *testing.T, got error, expected error) bool {
	if got != nil && expected == nil {
		t.Fatalf("unexpected error: %v", got)
	} else if got == nil && expected != nil {
		t.Fatalf("expected error: %v", expected)
	} else if got != nil && expected != nil {
		if got.Error() != expected.Error() {
			t.Fatalf("expected error: %v, got %v", expected, got)
		}
		return false
	}
	return true
}

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

func TestWriteCString(t *testing.T) {
	type expected struct {
		Value []byte
		Error error
	}
	tests := []struct {
		Name     string
		Input    string
		MaxLen   int
		Expected expected
	}{
		{
			Name:     "success",
			Input:    "ok",
			MaxLen:   4,
			Expected: expected{Value: []byte{'o', 'k', 0, 0}},
		},
		{
			Name:     "string is longer than max length",
			Input:    "ok",
			MaxLen:   2,
			Expected: expected{Error: poop.New("string is longer than max length")},
		},
		{
			Name:     "empty string",
			Input:    "",
			MaxLen:   4,
			Expected: expected{Value: []byte{0, 0, 0, 0}},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := writeCString(&buf, test.Input, test.MaxLen); !checkError(t, err, test.Expected.Error) {
				return
			}

			if !bytes.Equal(buf.Bytes(), test.Expected.Value) {
				t.Fatalf("expected value: %v, got %v", test.Expected.Value, buf.Bytes())
			}
		})
	}
}

func TestReadCString(t *testing.T) {
	type expected struct {
		Value string
		Error error
	}
	tests := []struct {
		Name     string
		Input    []byte
		MaxLen   int
		Expected expected
	}{
		{
			Name:     "success",
			Input:    []byte{'o', 'k', 0, 0},
			MaxLen:   4,
			Expected: expected{Value: "ok"},
		},
		{
			Name:     "string is not null-terminated",
			Input:    []byte{'o', 'k', 'x', 'x'},
			MaxLen:   4,
			Expected: expected{Error: poop.New("cstring is not null-terminated")},
		},
		{
			Name:     "empty string",
			Input:    []byte{0, 0, 0, 0},
			MaxLen:   4,
			Expected: expected{Value: ""},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			value, err := readCString(bytes.NewReader(test.Input), test.MaxLen)
			if !checkError(t, err, test.Expected.Error) {
				return
			}
			if value != test.Expected.Value {
				t.Fatalf("expected value: %v, got %v", test.Expected.Value, value)
			}
		})
	}
}

func TestReadTime(t *testing.T) {
	tests := []struct {
		Name     string
		Input    []byte
		Expected time.Time
	}{
		{
			Name:     "zero time",
			Input:    []byte{0, 0, 0, 0},
			Expected: time.Unix(0, 0),
		},
		{
			Name:     "any ole time",
			Input:    []byte{139, 7, 142, 105},
			Expected: time.Date(2026, time.February, 12, 12, 2, 3, 0, time.Local),
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			value, err := readTime(bytes.NewReader(test.Input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if value != test.Expected {
				t.Fatalf("expected value: %v, got %v", test.Expected, value)
			}
		})
	}
}

func TestWriteTime(t *testing.T) {
	tests := []struct {
		Name     string
		Input    time.Time
		Expected []byte
	}{
		{
			Name:     "start of epoch",
			Input:    time.Unix(0, 0),
			Expected: []byte{0, 0, 0, 0},
		},
		{
			Name:     "zero time",
			Input:    time.Time{},
			Expected: []byte{0, 0, 0, 0},
		},
		{
			Name:     "any ole time",
			Input:    time.Date(2026, time.February, 12, 12, 2, 3, 0, time.Local),
			Expected: []byte{139, 7, 142, 105},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := writeTime(&buf, test.Input); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !bytes.Equal(buf.Bytes(), test.Expected) {
				t.Fatalf("expected value: %v, got %v", test.Expected, buf.Bytes())
			}
		})
	}
}
