package meshcore

import (
	"bytes"
	"testing"

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
