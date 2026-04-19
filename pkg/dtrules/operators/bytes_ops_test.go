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
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// Behavior matrix for the byte-sequence manipulation ops. The
// encoding_test.go suite covers the encoding family (hex, b58check,
// bech32, bigint<->bytes); these ops are the raw byte manipulators
// the staking code uses to build/slice transaction payloads.
//
// Covered: bytes+, byteslen, bytesslice, bytesidx, bytes==, bytes!=
// Not covered here (already in encoding_test.go): hex, cvhex,
// b58check, cvb58check, bech32, cvbech32, bigintbytes, bytesbigint.

// pushBytes helper pushes a byte slice as RBytes.
func pushBytes(t *testing.T, state dtrules.State, b []byte) {
	t.Helper()
	if err := state.DataPush(dtrules.NewRBytes(b)); err != nil {
		t.Fatalf("DataPush RBytes: %v", err)
	}
}

// runBytesOp runs an op and returns the top-of-stack as RBytes bytes.
func runBytesOp(t *testing.T, op string, pushes func(dtrules.State)) []byte {
	t.Helper()
	state := newTestState()
	pushes(state)
	o, ok := Get(dtrules.GetRName(op))
	if !ok {
		t.Fatalf("op %q not registered", op)
	}
	if err := o.Execute(state); err != nil {
		t.Fatalf("%s: %v", op, err)
	}
	top, _ := state.DataPop()
	b, err := top.RBytesValue()
	if err != nil {
		t.Fatalf("RBytesValue: %v", err)
	}
	return b.Bytes()
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestBytesConcat — bytes+ joins two byte sequences.
func TestBytesConcat(t *testing.T) {
	cases := []struct {
		a, b []byte
		want []byte
	}{
		{[]byte{0xDE, 0xAD}, []byte{0xBE, 0xEF}, []byte{0xDE, 0xAD, 0xBE, 0xEF}},
		{[]byte{}, []byte{0x01}, []byte{0x01}},
		{[]byte{0x01}, []byte{}, []byte{0x01}},
		{[]byte{}, []byte{}, []byte{}},
	}
	for _, c := range cases {
		got := runBytesOp(t, "bytes+", func(s dtrules.State) {
			pushBytes(t, s, c.a)
			pushBytes(t, s, c.b)
		})
		if !bytesEqual(got, c.want) {
			t.Errorf("bytes+ %x %x: got %x, want %x", c.a, c.b, got, c.want)
		}
	}
}

// TestBytesLen — byteslen returns the number of bytes as an integer.
func TestBytesLen(t *testing.T) {
	cases := []struct {
		input []byte
		want  int64
	}{
		{[]byte{}, 0},
		{[]byte{0x01}, 1},
		{[]byte{0xDE, 0xAD, 0xBE, 0xEF}, 4},
		{make([]byte, 32), 32},
	}
	for _, c := range cases {
		state := newTestState()
		pushBytes(t, state, c.input)
		o, _ := Get(dtrules.GetRName("byteslen"))
		if err := o.Execute(state); err != nil {
			t.Fatalf("byteslen: %v", err)
		}
		top, _ := state.DataPop()
		v, _ := top.LongValue()
		if v != c.want {
			t.Errorf("byteslen(%x) = %d, want %d", c.input, v, c.want)
		}
	}
}

// TestBytesSlice — bytesslice returns a sub-sequence, Go-slice semantics.
func TestBytesSlice(t *testing.T) {
	src := []byte{0x00, 0x11, 0x22, 0x33, 0x44}
	cases := []struct {
		from, to int64
		want     []byte
	}{
		{0, 5, []byte{0x00, 0x11, 0x22, 0x33, 0x44}},
		{0, 0, []byte{}},
		{1, 4, []byte{0x11, 0x22, 0x33}},
		{2, 3, []byte{0x22}},
		{5, 5, []byte{}},
	}
	for _, c := range cases {
		got := runBytesOp(t, "bytesslice", func(s dtrules.State) {
			pushBytes(t, s, src)
			s.DataPush(dtrules.GetRIntegerValue(c.from))
			s.DataPush(dtrules.GetRIntegerValue(c.to))
		})
		if !bytesEqual(got, c.want) {
			t.Errorf("bytesslice[%d:%d] of %x = %x, want %x",
				c.from, c.to, src, got, c.want)
		}
	}
}

// TestBytesSliceOutOfRange — bytesslice should error on bad indices.
func TestBytesSliceOutOfRange(t *testing.T) {
	src := []byte{0x00, 0x11, 0x22}
	cases := []struct {
		from, to int64
	}{
		{-1, 2},
		{0, 10}, // past end
		{2, 1},  // inverted
	}
	for _, c := range cases {
		state := newTestState()
		pushBytes(t, state, src)
		state.DataPush(dtrules.GetRIntegerValue(c.from))
		state.DataPush(dtrules.GetRIntegerValue(c.to))
		o, _ := Get(dtrules.GetRName("bytesslice"))
		if err := o.Execute(state); err == nil {
			t.Errorf("bytesslice[%d:%d] should error on %x", c.from, c.to, src)
		}
	}
}

// TestBytesIndex — bytesidx returns the byte at an index as integer 0-255.
func TestBytesIndex(t *testing.T) {
	src := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	cases := []struct {
		idx  int64
		want int64
	}{
		{0, 0xDE},
		{1, 0xAD},
		{2, 0xBE},
		{3, 0xEF},
	}
	for _, c := range cases {
		state := newTestState()
		pushBytes(t, state, src)
		state.DataPush(dtrules.GetRIntegerValue(c.idx))
		o, _ := Get(dtrules.GetRName("bytesidx"))
		if err := o.Execute(state); err != nil {
			t.Fatalf("bytesidx: %v", err)
		}
		top, _ := state.DataPop()
		v, _ := top.LongValue()
		if v != c.want {
			t.Errorf("bytesidx(%x, %d) = %d, want %d", src, c.idx, v, c.want)
		}
	}
}

// TestBytesIndexOutOfRange pins the error boundary.
func TestBytesIndexOutOfRange(t *testing.T) {
	src := []byte{0x01, 0x02, 0x03}
	for _, idx := range []int64{-1, 3, 100} {
		state := newTestState()
		pushBytes(t, state, src)
		state.DataPush(dtrules.GetRIntegerValue(idx))
		o, _ := Get(dtrules.GetRName("bytesidx"))
		if err := o.Execute(state); err == nil {
			t.Errorf("bytesidx(%x, %d) should error", src, idx)
		}
	}
}

// TestBytesEquality — bytes== and bytes!= both exercise the same
// constant-time compare. Pin both polarity and length-mismatch.
func TestBytesEquality(t *testing.T) {
	cases := []struct {
		name   string
		a, b   []byte
		wantEq bool
	}{
		{"identical", []byte{0x01, 0x02}, []byte{0x01, 0x02}, true},
		{"empty", []byte{}, []byte{}, true},
		{"different_value", []byte{0x01, 0x02}, []byte{0x01, 0x03}, false},
		{"different_length", []byte{0x01}, []byte{0x01, 0x00}, false},
		{"a_empty", []byte{}, []byte{0x00}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, pair := range []struct {
				op   string
				want bool
			}{
				{"bytes==", c.wantEq},
				{"bytes!=", !c.wantEq},
			} {
				state := newTestState()
				pushBytes(t, state, c.a)
				pushBytes(t, state, c.b)
				o, _ := Get(dtrules.GetRName(pair.op))
				if err := o.Execute(state); err != nil {
					t.Fatalf("%s: %v", pair.op, err)
				}
				top, _ := state.DataPop()
				v, _ := top.BooleanValue()
				if v != pair.want {
					t.Errorf("%s(%x, %x) = %v, want %v",
						pair.op, c.a, c.b, v, pair.want)
				}
			}
		})
	}
}
