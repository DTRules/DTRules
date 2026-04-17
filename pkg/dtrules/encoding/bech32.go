package encoding

import (
	"errors"
	"strings"
)

// bech32Charset is the BIP-173 character set.
const bech32Charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

var bech32CharsetIndex [256]int8

func init() {
	for i := range bech32CharsetIndex {
		bech32CharsetIndex[i] = -1
	}
	for i, c := range bech32Charset {
		bech32CharsetIndex[c] = int8(i)
	}
}

func bech32PolymodStep(pre uint32) uint32 {
	b := pre >> 25
	return ((pre & 0x1ffffff) << 5) ^
		(-((b >> 0) & 1) & 0x3b6a57b2) ^
		(-((b >> 1) & 1) & 0x26508e6d) ^
		(-((b >> 2) & 1) & 0x1ea119fa) ^
		(-((b >> 3) & 1) & 0x3d4233dd) ^
		(-((b >> 4) & 1) & 0x2a1462b3)
}

func bech32HRPExpand(hrp string) []byte {
	result := make([]byte, len(hrp)*2+1)
	for i := 0; i < len(hrp); i++ {
		result[i] = hrp[i] >> 5
		result[len(hrp)+1+i] = hrp[i] & 0x1f
	}
	result[len(hrp)] = 0
	return result
}

func bech32VerifyChecksum(hrp string, data []byte) bool {
	values := append(bech32HRPExpand(hrp), data...)
	return bech32Polymod(values) == 1
}

func bech32Polymod(values []byte) uint32 {
	chk := uint32(1)
	for _, v := range values {
		chk = bech32PolymodStep(chk) ^ uint32(v)
	}
	return chk
}

func bech32CreateChecksum(hrp string, data []byte) []byte {
	// BIP-173: polymod(hrp_expand || data || [0,0,0,0,0,0]) XOR 1.
	// Process hrp_expand and data with their values XOR'd in, then 6 more
	// polymod steps for the trailing zeros (no XOR since value is 0),
	// then XOR 1 at the end.
	// Append 6 zero bytes for checksum placeholder, then XOR with 1
	values := append(bech32HRPExpand(hrp), data...)
	zeros := []byte{0, 0, 0, 0, 0, 0}
	polymod := bech32Polymod(append(values, zeros...)) ^ 1
	checksum := make([]byte, 6)
	for i := range checksum {
		checksum[i] = byte((polymod >> (5 * (5 - i))) & 0x1f)
	}
	return checksum
}

// convertBits converts data from fromBits to toBits bits-per-byte encoding.
// If pad is true, pad the output; if false, strip padding (and error if non-zero padding).
func convertBits(data []byte, fromBits, toBits uint, pad bool) ([]byte, error) {
	acc := 0
	bits := uint(0)
	var result []byte
	maxv := (1 << toBits) - 1
	for _, b := range data {
		acc = (acc << fromBits) | int(b)
		bits += fromBits
		for bits >= toBits {
			bits -= toBits
			result = append(result, byte((acc>>bits)&maxv))
		}
	}
	if pad {
		if bits > 0 {
			result = append(result, byte((acc<<(toBits-bits))&maxv))
		}
	} else if bits >= fromBits || ((acc<<(toBits-bits))&maxv) != 0 {
		return nil, errors.New("bech32: non-zero padding")
	}
	return result, nil
}

// Bech32Encode encodes payload bytes with the given human-readable part (hrp)
// using BIP-173 bech32 encoding.
func Bech32Encode(hrp string, payload []byte) (string, error) {
	hrp = strings.ToLower(hrp)
	converted, err := convertBits(payload, 8, 5, true)
	if err != nil {
		return "", err
	}
	checksum := bech32CreateChecksum(hrp, converted)
	combined := append(converted, checksum...)
	var sb strings.Builder
	sb.WriteString(hrp)
	sb.WriteByte('1')
	for _, b := range combined {
		sb.WriteByte(bech32Charset[b])
	}
	return sb.String(), nil
}

// Bech32Decode decodes a bech32 string and returns (hrp, payload, error).
func Bech32Decode(bech string) (hrp string, payload []byte, err error) {
	// Validate all characters are printable ASCII (33–126, no mixed case)
	hasLower, hasUpper := false, false
	for _, c := range bech {
		if c < 33 || c > 126 {
			return "", nil, errors.New("bech32: character out of printable ASCII range")
		}
		if c >= 'a' && c <= 'z' {
			hasLower = true
		}
		if c >= 'A' && c <= 'Z' {
			hasUpper = true
		}
	}
	if hasLower && hasUpper {
		return "", nil, errors.New("bech32: mixed case string")
	}
	lower := strings.ToLower(bech)
	sep := strings.LastIndex(lower, "1")
	if sep < 1 || sep+7 > len(lower) {
		return "", nil, errors.New("bech32: invalid format (no separator or too short)")
	}
	hrp = lower[:sep]
	data := make([]byte, len(lower)-sep-1)
	for i := 0; i < len(data); i++ {
		c := lower[sep+1+i]
		if c > 127 || bech32CharsetIndex[c] < 0 {
			return "", nil, errors.New("bech32: invalid data character")
		}
		data[i] = byte(bech32CharsetIndex[c])
	}
	if !bech32VerifyChecksum(hrp, data) {
		return "", nil, errors.New("bech32: invalid checksum")
	}
	payload, err = convertBits(data[:len(data)-6], 5, 8, false)
	if err != nil {
		return "", nil, err
	}
	return hrp, payload, nil
}
