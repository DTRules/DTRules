// Package encoding provides encoding helpers for the DTRules EL runtime:
// base58check, bech32, hex, and bigint↔bytes conversions.
package encoding

import (
	"crypto/sha256"
	"errors"
	"math/big"
)

// base58Alphabet is the Bitcoin base58 alphabet.
const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

var base58AlphabetIndex [256]int8

func init() {
	for i := range base58AlphabetIndex {
		base58AlphabetIndex[i] = -1
	}
	for i, c := range base58Alphabet {
		base58AlphabetIndex[c] = int8(i)
	}
}

var bigZero = big.NewInt(0)
var bigBase = big.NewInt(58)

// base58Encode encodes raw bytes to a base58 string.
func base58Encode(input []byte) string {
	n := new(big.Int).SetBytes(input)
	var result []byte
	mod := new(big.Int)
	for n.Cmp(bigZero) > 0 {
		n.DivMod(n, bigBase, mod)
		result = append(result, base58Alphabet[mod.Int64()])
	}
	// Leading zero bytes become '1'
	for _, b := range input {
		if b != 0 {
			break
		}
		result = append(result, '1')
	}
	// Reverse
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return string(result)
}

// base58Decode decodes a base58 string to raw bytes.
func base58Decode(s string) ([]byte, error) {
	n := new(big.Int)
	for _, c := range s {
		if c > 255 || base58AlphabetIndex[c] < 0 {
			return nil, errors.New("invalid base58 character")
		}
		n.Mul(n, bigBase)
		n.Add(n, big.NewInt(int64(base58AlphabetIndex[c])))
	}
	decoded := n.Bytes()
	// Restore leading zeroes
	leadingZeros := 0
	for _, c := range s {
		if c != '1' {
			break
		}
		leadingZeros++
	}
	result := make([]byte, leadingZeros+len(decoded))
	copy(result[leadingZeros:], decoded)
	return result, nil
}

func checksum(data []byte) []byte {
	h1 := sha256.Sum256(data)
	h2 := sha256.Sum256(h1[:])
	return h2[:4]
}

// Base58CheckEncode encodes payload with a one-byte version prefix using base58check.
func Base58CheckEncode(payload []byte, version byte) string {
	versioned := append([]byte{version}, payload...)
	cs := checksum(versioned)
	full := append(versioned, cs...)
	return base58Encode(full)
}

// Base58CheckDecode decodes a base58check string and returns (version, payload, error).
func Base58CheckDecode(s string) (version byte, payload []byte, err error) {
	decoded, err := base58Decode(s)
	if err != nil {
		return 0, nil, err
	}
	if len(decoded) < 5 {
		return 0, nil, errors.New("base58check: decoded data too short")
	}
	body := decoded[:len(decoded)-4]
	got := decoded[len(decoded)-4:]
	want := checksum(body)
	if got[0] != want[0] || got[1] != want[1] || got[2] != want[2] || got[3] != want[3] {
		return 0, nil, errors.New("base58check: invalid checksum")
	}
	return body[0], body[1:], nil
}
