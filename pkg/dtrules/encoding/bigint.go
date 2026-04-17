package encoding

import (
	"errors"
	"math/big"
)

// BigIntToBytes encodes n as a big-endian byte slice, zero-padded to size bytes.
// Returns an error if n is negative or does not fit in size bytes.
func BigIntToBytes(n *big.Int, size int) ([]byte, error) {
	if n.Sign() < 0 {
		return nil, errors.New("bigint to bytes: negative value not supported")
	}
	b := n.Bytes()
	if len(b) > size {
		return nil, errors.New("bigint to bytes: value does not fit in requested size")
	}
	result := make([]byte, size)
	copy(result[size-len(b):], b)
	return result, nil
}

// BytesToBigInt interprets b as an unsigned big-endian integer and returns a *big.Int.
func BytesToBigInt(b []byte) *big.Int {
	return new(big.Int).SetBytes(b)
}
