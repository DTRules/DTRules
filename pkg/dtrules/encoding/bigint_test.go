package encoding

import (
	"crypto/rand"
	"math/big"
	"testing"
)

func TestBigIntToBytesKnownValues(t *testing.T) {
	cases := []struct {
		n    int64
		size int
		want []byte
		err  bool
	}{
		{0, 4, []byte{0, 0, 0, 0}, false},
		{256, 2, []byte{0x01, 0x00}, false},
		{0xFF, 1, []byte{0xFF}, false},
		{256, 1, nil, true},  // doesn't fit in 1 byte
		{-1, 4, nil, true},   // negative
		{0, 0, []byte{}, false}, // size=0, value=0 fits (zero bytes of big.Int)
	}
	for _, c := range cases {
		n := big.NewInt(c.n)
		got, err := BigIntToBytes(n, c.size)
		if c.err {
			if err == nil {
				t.Errorf("BigIntToBytes(%d, %d): expected error, got nil (result=%v)", c.n, c.size, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("BigIntToBytes(%d, %d): unexpected error: %v", c.n, c.size, err)
			continue
		}
		if len(got) != len(c.want) {
			t.Errorf("BigIntToBytes(%d, %d): length got %d, want %d", c.n, c.size, len(got), len(c.want))
			continue
		}
		for i, b := range c.want {
			if got[i] != b {
				t.Errorf("BigIntToBytes(%d, %d): byte[%d] got 0x%02x, want 0x%02x", c.n, c.size, i, got[i], b)
			}
		}
	}
}

func TestBytesToBigIntKnownValues(t *testing.T) {
	cases := []struct {
		input []byte
		want  int64
	}{
		{[]byte{}, 0},
		{[]byte{0x01, 0x00}, 256},
		{[]byte{0xFF, 0xFF}, 65535},
	}
	for _, c := range cases {
		got := BytesToBigInt(c.input)
		want := big.NewInt(c.want)
		if got.Cmp(want) != 0 {
			t.Errorf("BytesToBigInt(%v): got %s, want %d", c.input, got, c.want)
		}
	}
}

func TestBigIntRoundTrip(t *testing.T) {
	for i := 0; i < 1000; i++ {
		// Random size 1..64
		var sizeBuf [1]byte
		rand.Read(sizeBuf[:])
		size := int(sizeBuf[0]%64) + 1

		// Random n that fits in size bytes: generate (size) random bytes, treat as big-endian unsigned
		nBytes := make([]byte, size)
		rand.Read(nBytes)
		n := new(big.Int).SetBytes(nBytes)

		encoded, err := BigIntToBytes(n, size)
		if err != nil {
			t.Fatalf("round-trip BigIntToBytes error (iter %d, size %d): %v", i, size, err)
		}
		decoded := BytesToBigInt(encoded)
		if decoded.Cmp(n) != 0 {
			t.Errorf("round-trip mismatch (iter %d): got %s, want %s", i, decoded, n)
		}
	}
}
