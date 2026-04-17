package encoding

import (
	"crypto/rand"
	"testing"
)

func TestBase58CheckKnownAnswers(t *testing.T) {
	// 20 zero bytes with version 0 → known canonical address
	zeroPayload := make([]byte, 20)
	got := Base58CheckEncode(zeroPayload, 0)
	want := "1111111111111111111114oLvT2"
	if got != want {
		t.Errorf("zero payload encode: got %q, want %q", got, want)
	}

	// P2PKH: 1BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN2
	// version=0, hash160 = 77bff20c60e522dfaa3350c39b030a5d004e839a
	p2pkhAddr := "1BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN2"
	expectedHash160 := []byte{
		0x77, 0xbf, 0xf2, 0x0c, 0x60, 0xe5, 0x22, 0xdf, 0xaa, 0x33,
		0x50, 0xc3, 0x9b, 0x03, 0x0a, 0x5d, 0x00, 0x4e, 0x83, 0x9a,
	}
	ver, payload, err := Base58CheckDecode(p2pkhAddr)
	if err != nil {
		t.Fatalf("P2PKH decode error: %v", err)
	}
	if ver != 0 {
		t.Errorf("P2PKH version: got %d, want 0", ver)
	}
	if len(payload) != 20 {
		t.Fatalf("P2PKH payload length: got %d, want 20", len(payload))
	}
	for i, b := range expectedHash160 {
		if payload[i] != b {
			t.Errorf("P2PKH payload byte[%d]: got 0x%02x, want 0x%02x", i, payload[i], b)
		}
	}

	// P2SH: version 5, re-encode and decode
	p2shPayload := []byte{
		0x54, 0x9a, 0x0e, 0x7a, 0x29, 0x5b, 0x6d, 0x78, 0xbb, 0xd5,
		0x0b, 0x6a, 0x99, 0xa4, 0x14, 0xa1, 0xb2, 0x4c, 0x60, 0x10,
	}
	p2shEncoded := Base58CheckEncode(p2shPayload, 5)
	decVer, decPayload, err := Base58CheckDecode(p2shEncoded)
	if err != nil {
		t.Fatalf("P2SH decode error: %v", err)
	}
	if decVer != 5 {
		t.Errorf("P2SH version: got %d, want 5", decVer)
	}
	if len(decPayload) != 20 {
		t.Fatalf("P2SH payload length: got %d, want 20", len(decPayload))
	}
	for i, b := range p2shPayload {
		if decPayload[i] != b {
			t.Errorf("P2SH payload byte[%d]: got 0x%02x, want 0x%02x", i, decPayload[i], b)
		}
	}
}

func TestBase58CheckRoundTrip(t *testing.T) {
	for i := 0; i < 1000; i++ {
		var versionBuf [1]byte
		rand.Read(versionBuf[:])
		version := versionBuf[0]

		payload := make([]byte, 20)
		rand.Read(payload)

		encoded := Base58CheckEncode(payload, version)
		decVer, decPayload, err := Base58CheckDecode(encoded)
		if err != nil {
			t.Fatalf("round-trip decode error (iter %d): %v", i, err)
		}
		if decVer != version {
			t.Errorf("round-trip version mismatch (iter %d): got %d, want %d", i, decVer, version)
		}
		if len(decPayload) != len(payload) {
			t.Fatalf("round-trip payload length mismatch (iter %d): got %d, want %d", i, len(decPayload), len(payload))
		}
		for j, b := range payload {
			if decPayload[j] != b {
				t.Errorf("round-trip payload byte[%d] mismatch (iter %d): got 0x%02x, want 0x%02x", j, i, decPayload[j], b)
				break
			}
		}
	}
}

func TestBase58CheckBadChecksum(t *testing.T) {
	payload := make([]byte, 20)
	rand.Read(payload)
	encoded := Base58CheckEncode(payload, 0)

	// Flip the last character to corrupt the checksum
	runes := []byte(encoded)
	last := runes[len(runes)-1]
	// Find a different valid base58 char
	for _, c := range base58Alphabet {
		if byte(c) != last {
			runes[len(runes)-1] = byte(c)
			break
		}
	}
	_, _, err := Base58CheckDecode(string(runes))
	if err == nil {
		t.Error("expected error for bad checksum, got nil")
	}
}

func TestBase58CheckInvalidCharacters(t *testing.T) {
	invalidChars := []string{"0", "O", "I", "l"}
	for _, c := range invalidChars {
		_, _, err := Base58CheckDecode("1" + c + "BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN2")
		if err == nil {
			t.Errorf("expected error for invalid char %q, got nil", c)
		}
	}
}

func TestBase58CheckEmptyString(t *testing.T) {
	_, _, err := Base58CheckDecode("")
	if err == nil {
		t.Error("expected error for empty string, got nil")
	}
}

func TestBase58CheckTooShort(t *testing.T) {
	// A valid base58 string but decodes to fewer than 5 bytes (not enough for version + 4-byte checksum)
	// "1" decodes to a single zero byte
	_, _, err := Base58CheckDecode("1111")
	if err == nil {
		t.Error("expected error for too-short decoded data, got nil")
	}
}
