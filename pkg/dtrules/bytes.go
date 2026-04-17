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

package dtrules

import (
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"
)

// RBytes represents an immutable byte-sequence value in the rules engine.
// Backed by []byte; equality uses constant-time comparison.
type RBytes struct {
	BaseObject
	value []byte
}

var bytesEmpty = &RBytes{value: []byte{}}

// NewRBytes creates a new RBytes from a byte slice (copied).
func NewRBytes(b []byte) *RBytes {
	if len(b) == 0 {
		return bytesEmpty
	}
	cp := make([]byte, len(b))
	copy(cp, b)
	return &RBytes{value: cp}
}

// NewRBytesFromHex parses a hex string like "0x4a5b6c" or "4a5b6c" and returns RBytes.
func NewRBytesFromHex(s string) (*RBytes, error) {
	s = strings.TrimSpace(s)
	// Strip optional 0x prefix (case-insensitive)
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "0x") {
		s = s[2:]
	}
	if s == "" {
		return bytesEmpty, nil
	}
	if len(s)%2 != 0 {
		return nil, ConversionError("RBytes.NewRBytesFromHex", fmt.Sprintf("hex string has odd length: %q", s))
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, ConversionError("RBytes.NewRBytesFromHex", fmt.Sprintf("invalid hex string: %q: %v", s, err))
	}
	return NewRBytes(b), nil
}

// Type returns the type for this object.
func (r *RBytes) Type() *RType {
	return TypeBytes
}

// Execute pushes this object onto the data stack.
func (r *RBytes) Execute(state State) error {
	return state.DataPush(r)
}

// ArrayExecute pushes this object onto the data stack.
func (r *RBytes) ArrayExecute(state State) error {
	return state.DataPush(r)
}

// GetExecutable returns this object.
func (r *RBytes) GetExecutable() Object {
	return r
}

// GetNonExecutable returns this object.
func (r *RBytes) GetNonExecutable() Object {
	return r
}

// IsExecutable returns false for bytes values.
func (r *RBytes) IsExecutable() bool {
	return false
}

// Equals compares this bytes value with another using constant-time comparison.
// Two bytes values are equal iff they have the same length and content.
func (r *RBytes) Equals(o Object) (bool, error) {
	if o.Type() == TypeNull {
		return false, nil
	}
	other, err := o.RBytesValue()
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare(r.value, other.value) == 1, nil
}

// Compare is not well-defined for bytes; returns lexicographic order for symmetry.
func (r *RBytes) Compare(o Object) (int, error) {
	other, err := o.RBytesValue()
	if err != nil {
		return 0, err
	}
	la, lb := len(r.value), len(other.value)
	min := la
	if lb < min {
		min = lb
	}
	for i := 0; i < min; i++ {
		if r.value[i] < other.value[i] {
			return -1, nil
		}
		if r.value[i] > other.value[i] {
			return 1, nil
		}
	}
	if la < lb {
		return -1, nil
	}
	if la > lb {
		return 1, nil
	}
	return 0, nil
}

// StringValue returns the hex representation with a 0x prefix.
func (r *RBytes) StringValue() string {
	return "0x" + hex.EncodeToString(r.value)
}

// PostFix returns the postfix representation.
func (r *RBytes) PostFix() string {
	return r.StringValue()
}

// Clone returns this object (bytes values are immutable).
func (r *RBytes) Clone(session Session) (Object, error) {
	return r, nil
}

// RClone returns this object.
func (r *RBytes) RClone() Object {
	return r
}

// String implements the Stringer interface.
func (r *RBytes) String() string {
	return r.StringValue()
}

// Len returns the number of bytes.
func (r *RBytes) Len() int {
	return len(r.value)
}

// Bytes returns a copy of the underlying byte slice.
func (r *RBytes) Bytes() []byte {
	cp := make([]byte, len(r.value))
	copy(cp, r.value)
	return cp
}

// Concat returns a new RBytes that is r followed by other.
func (r *RBytes) Concat(other *RBytes) *RBytes {
	if len(r.value) == 0 {
		return other
	}
	if len(other.value) == 0 {
		return r
	}
	buf := make([]byte, len(r.value)+len(other.value))
	copy(buf, r.value)
	copy(buf[len(r.value):], other.value)
	return &RBytes{value: buf}
}

// Slice returns r.value[from:to] as a new RBytes.
// from is inclusive, to is exclusive. Out-of-range is a runtime error.
func (r *RBytes) Slice(from, to int) (*RBytes, error) {
	n := len(r.value)
	if from < 0 || to < 0 || from > n || to > n || from > to {
		return nil, OutOfBoundsError("RBytes.Slice", fmt.Sprintf("slice [%d:%d] out of range for bytes of length %d", from, to, n))
	}
	if from == to {
		return bytesEmpty, nil
	}
	cp := make([]byte, to-from)
	copy(cp, r.value[from:to])
	return &RBytes{value: cp}, nil
}

// At returns the byte at index i as an integer (0-255). Out-of-range is a runtime error.
func (r *RBytes) At(i int) (int64, error) {
	if i < 0 || i >= len(r.value) {
		return 0, OutOfBoundsError("RBytes.At", fmt.Sprintf("index %d out of range for bytes of length %d", i, len(r.value)))
	}
	return int64(r.value[i]), nil
}

// ConstantTimeEqual compares r and other using crypto/subtle.ConstantTimeCompare.
func (r *RBytes) ConstantTimeEqual(other *RBytes) bool {
	return subtle.ConstantTimeCompare(r.value, other.value) == 1
}

// RBytesValue returns this object.
func (r *RBytes) RBytesValue() (*RBytes, error) {
	return r, nil
}

// RStringValue returns the hex string representation.
func (r *RBytes) RStringValue() *RString {
	return NewRString(r.StringValue())
}

// IntValue returns an error — bytes cannot be coerced to int.
func (r *RBytes) IntValue() (int, error) {
	return 0, ConversionError("RBytes.IntValue", "cannot convert bytes to integer")
}

// LongValue returns an error.
func (r *RBytes) LongValue() (int64, error) {
	return 0, ConversionError("RBytes.LongValue", "cannot convert bytes to long")
}

// DoubleValue returns an error.
func (r *RBytes) DoubleValue() (float64, error) {
	return 0, ConversionError("RBytes.DoubleValue", "cannot convert bytes to double")
}

// BooleanValue returns an error.
func (r *RBytes) BooleanValue() (bool, error) {
	return false, ConversionError("RBytes.BooleanValue", "cannot convert bytes to boolean")
}
