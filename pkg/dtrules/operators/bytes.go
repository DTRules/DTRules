// Copyright 2024 Paul Snow
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package operators

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/encoding"
)

func init() {
	// Bytes type conversion
	Register("cvbytes", opCvBytes)

	// Bytes arithmetic
	Register("bytes+", opBytesConcat)
	Alias("bytes+", "bytesconcat")

	// Bytes built-ins
	Register("byteslen", opBytesLen)
	Register("bytesslice", opBytesSlice)
	Register("bytesidx", opBytesIndex)

	// Bytes comparisons (constant-time)
	Register("bytes==", opBytesEqual)
	Register("bytes!=", opBytesNotEqual)

	// Encoding: hex
	Register("hex", opHex)     // ( bytes -- string )  lowercase hex, no 0x prefix
	Register("cvhex", opCvHex) // ( string -- bytes )  accepts optional 0x prefix

	// Encoding: base58check
	Register("b58check", opB58Check)     // ( bytes version -- string )
	Register("cvb58check", opCvB58Check) // ( string -- bytes )  also pushes version as integer

	// Encoding: bech32
	Register("bech32", opBech32)     // ( bytes hrp -- string )
	Register("cvbech32", opCvBech32) // ( string -- bytes )  also pushes hrp as string

	// Bigint <-> bytes
	Register("bigintbytes", opBigIntBytes) // ( bigint size -- bytes )
	Register("bytesbigint", opBytesBigInt) // ( bytes -- bigint )
}

// opCvBytes converts a value to RBytes: ( value -- bytes )
// Accepts hex string ("0x...") or RBytes identity.
func opCvBytes(state dtrules.State) error {
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	b, err := a.RBytesValue()
	if err != nil {
		// Try from string (hex literal emitted by compiler is already RBytes after cvbytes,
		// but a plain string like "0xdeadbeef" should also work)
		s := a.StringValue()
		b, err = dtrules.NewRBytesFromHex(s)
		if err != nil {
			return dtrules.ConversionError("cvbytes", "cannot convert value to bytes: "+err.Error())
		}
	}
	return state.DataPush(b)
}

// opBytesConcat concatenates two byte sequences: ( a b -- a||b )
func opBytesConcat(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.RBytesValue()
	if err != nil {
		return err
	}
	bVal, err := b.RBytesValue()
	if err != nil {
		return err
	}
	return state.DataPush(aVal.Concat(bVal))
}

// opBytesLen returns the length in bytes: ( bytes -- int )
func opBytesLen(state dtrules.State) error {
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.RBytesValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRIntegerValue(int64(aVal.Len())))
}

// opBytesSlice returns a sub-sequence: ( bytes from to -- bytes[from:to] )
func opBytesSlice(state dtrules.State) error {
	toObj, err := state.DataPop()
	if err != nil {
		return err
	}
	fromObj, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.RBytesValue()
	if err != nil {
		return err
	}
	fromInt, err := fromObj.IntValue()
	if err != nil {
		return err
	}
	toInt, err := toObj.IntValue()
	if err != nil {
		return err
	}
	result, err := aVal.Slice(fromInt, toInt)
	if err != nil {
		return err
	}
	return state.DataPush(result)
}

// opBytesIndex returns byte at index as integer 0-255: ( bytes i -- int )
func opBytesIndex(state dtrules.State) error {
	idxObj, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.RBytesValue()
	if err != nil {
		return err
	}
	idx, err := idxObj.IntValue()
	if err != nil {
		return err
	}
	byteVal, err := aVal.At(idx)
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRIntegerValue(byteVal))
}

// opBytesEqual compares two byte sequences using constant-time comparison: ( a b -- bool )
func opBytesEqual(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.RBytesValue()
	if err != nil {
		return err
	}
	bVal, err := b.RBytesValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(aVal.ConstantTimeEqual(bVal)))
}

// opBytesNotEqual compares two byte sequences for inequality using constant-time comparison: ( a b -- bool )
func opBytesNotEqual(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.RBytesValue()
	if err != nil {
		return err
	}
	bVal, err := b.RBytesValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(!aVal.ConstantTimeEqual(bVal)))
}

// opHex converts bytes to lowercase hex string without 0x prefix: ( bytes -- string )
func opHex(state dtrules.State) error {
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.RBytesValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.NewRString(hex.EncodeToString(aVal.Bytes())))
}

// opCvHex converts a hex string (with optional 0x prefix) to bytes: ( string -- bytes )
func opCvHex(state dtrules.State) error {
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	s := strings.TrimSpace(a.StringValue())
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "0x") {
		s = s[2:]
	}
	if len(s)%2 != 0 {
		return dtrules.ConversionError("cvhex", fmt.Sprintf("hex string has odd length: %q", s))
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return dtrules.ConversionError("cvhex", fmt.Sprintf("invalid hex string: %v", err))
	}
	return state.DataPush(dtrules.NewRBytes(b))
}

// opB58Check encodes bytes with version byte using base58check: ( bytes version -- string )
func opB58Check(state dtrules.State) error {
	versionObj, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.RBytesValue()
	if err != nil {
		return err
	}
	version, err := versionObj.IntValue()
	if err != nil {
		return err
	}
	if version < 0 || version > 255 {
		return dtrules.ConversionError("b58check", fmt.Sprintf("version byte %d out of range [0,255]", version))
	}
	encoded := encoding.Base58CheckEncode(aVal.Bytes(), byte(version))
	return state.DataPush(dtrules.NewRString(encoded))
}

// opCvB58Check decodes a base58check string: ( string -- bytes version )
// Pushes bytes first, then version integer (top of stack).
func opCvB58Check(state dtrules.State) error {
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	version, payload, err := encoding.Base58CheckDecode(a.StringValue())
	if err != nil {
		return dtrules.ConversionError("cvb58check", err.Error())
	}
	if err := state.DataPush(dtrules.NewRBytes(payload)); err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRIntegerValue(int64(version)))
}

// opBech32 encodes bytes with hrp using bech32: ( bytes hrp -- string )
func opBech32(state dtrules.State) error {
	hrpObj, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.RBytesValue()
	if err != nil {
		return err
	}
	hrp := hrpObj.StringValue()
	encoded, err := encoding.Bech32Encode(hrp, aVal.Bytes())
	if err != nil {
		return dtrules.ConversionError("bech32", err.Error())
	}
	return state.DataPush(dtrules.NewRString(encoded))
}

// opCvBech32 decodes a bech32 string: ( string -- bytes hrp )
// Pushes bytes first, then hrp string (top of stack).
func opCvBech32(state dtrules.State) error {
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	hrp, payload, err := encoding.Bech32Decode(a.StringValue())
	if err != nil {
		return dtrules.ConversionError("cvbech32", err.Error())
	}
	if err := state.DataPush(dtrules.NewRBytes(payload)); err != nil {
		return err
	}
	return state.DataPush(dtrules.NewRString(hrp))
}

// opBigIntBytes encodes a bigint as big-endian bytes of the given size: ( bigint size -- bytes )
func opBigIntBytes(state dtrules.State) error {
	sizeObj, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	size, err := sizeObj.IntValue()
	if err != nil {
		return err
	}
	if size <= 0 {
		return dtrules.ConversionError("bigintbytes", fmt.Sprintf("size must be positive, got %d", size))
	}
	bi, err := a.RBigIntValue()
	if err != nil {
		return err
	}
	b, err := encoding.BigIntToBytes(bi.BigIntValue(), size)
	if err != nil {
		return dtrules.ConversionError("bigintbytes", err.Error())
	}
	return state.DataPush(dtrules.NewRBytes(b))
}

// opBytesBigInt interprets bytes as an unsigned big-endian integer: ( bytes -- bigint )
func opBytesBigInt(state dtrules.State) error {
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	aVal, err := a.RBytesValue()
	if err != nil {
		return err
	}
	result := encoding.BytesToBigInt(aVal.Bytes())
	return state.DataPush(dtrules.NewRBigInt(result))
}
