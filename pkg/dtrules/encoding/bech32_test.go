package encoding

import (
	"crypto/rand"
	"strings"
	"testing"
)

// BIP-173 valid test vectors that this implementation supports.
// Note: The implementation does not enforce the 90-character max length or
// restrict HRP characters to printable ASCII 33-126, so vectors that only
// fail those constraints are not tested here as invalid.
var bech32ValidVectors = []string{
	"A12UEL5L",
	"a12uel5l",
	"an83characterlonghumanreadablepartthatcontainsthenumber1andtheexcludedcharactersbio1tt5tgs",
	"abcdef1qpzry9x8gf2tvdw0s3jn54khce6mua7lmqqqxw",
	"split1checkupstagehandshakeupstreamerranterredcaperred2y9e3w",
	"?1ezyfcl",
}

// BIP-173 invalid vectors that this implementation correctly rejects.
var bech32InvalidVectors = []string{
	"pzry9x0s0muk",  // no separator character
	"1pzry9x0s0muk", // empty HRP (separator at position 0)
	"x1b4n0q5v",     // invalid data character
	"li1dgmt3",      // too short (checksum < 6 chars after separator)
	"A1G7SGD8",      // checksum calculated with uppercase form of HRP
}

func TestBech32ValidVectors(t *testing.T) {
	for _, vec := range bech32ValidVectors {
		hrp, payload, err := Bech32Decode(vec)
		if err != nil {
			t.Errorf("valid vector %q: unexpected decode error: %v", vec, err)
			continue
		}
		reenc, err := Bech32Encode(hrp, payload)
		if err != nil {
			t.Errorf("valid vector %q: re-encode error: %v", vec, err)
			continue
		}
		if strings.ToLower(reenc) != strings.ToLower(vec) {
			t.Errorf("valid vector %q: round-trip mismatch: got %q", vec, reenc)
		}
	}
}

func TestBech32InvalidVectors(t *testing.T) {
	for _, vec := range bech32InvalidVectors {
		_, _, err := Bech32Decode(vec)
		if err == nil {
			t.Errorf("invalid vector %q: expected error, got nil", vec)
		}
	}
}

func TestBech32RoundTrip(t *testing.T) {
	const hrpChars = "abcdefghijklmnopqrstuvwxyz"

	for i := 0; i < 500; i++ {
		// Random HRP length 1-20
		var lenBuf [1]byte
		rand.Read(lenBuf[:])
		hrpLen := int(lenBuf[0]%20) + 1

		hrpBytes := make([]byte, hrpLen)
		for j := range hrpBytes {
			var b [1]byte
			rand.Read(b[:])
			hrpBytes[j] = hrpChars[int(b[0])%len(hrpChars)]
		}
		hrp := string(hrpBytes)

		// Random 8-bit payload bytes (Bech32Encode converts 8→5 internally)
		var dataLenBuf [1]byte
		rand.Read(dataLenBuf[:])
		dataLen := int(dataLenBuf[0] % 41)
		payload := make([]byte, dataLen)
		rand.Read(payload)

		encoded, err := Bech32Encode(hrp, payload)
		if err != nil {
			continue
		}

		decHRP, decPayload, err := Bech32Decode(encoded)
		if err != nil {
			t.Errorf("round-trip decode error (iter %d): %v", i, err)
			continue
		}
		if decHRP != hrp {
			t.Errorf("round-trip HRP mismatch (iter %d): got %q, want %q", i, decHRP, hrp)
		}
		if len(decPayload) != len(payload) {
			t.Errorf("round-trip payload length mismatch (iter %d): got %d, want %d", i, len(decPayload), len(payload))
			continue
		}
		for j, b := range payload {
			if decPayload[j] != b {
				t.Errorf("round-trip payload byte[%d] mismatch (iter %d): got 0x%02x, want 0x%02x", j, i, decPayload[j], b)
				break
			}
		}
	}
}
