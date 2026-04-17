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
	"github.com/DTRules/DTRules/pkg/dtrules"
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
