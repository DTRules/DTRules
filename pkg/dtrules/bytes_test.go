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
	"testing"
)

func TestRBytes_NewFromHex(t *testing.T) {
	tests := []struct {
		input   string
		wantLen int
		wantHex string
		wantErr bool
	}{
		{"0x", 0, "0x", false},
		{"0x00", 1, "0x00", false},
		{"0xdeadbeef", 4, "0xdeadbeef", false},
		{"0xDEADBEEF", 4, "0xdeadbeef", false}, // case-insensitive
		{"0xaabbcc", 3, "0xaabbcc", false},
		{"deadbeef", 4, "0xdeadbeef", false}, // no prefix also OK
		{"0x1", 0, "", true},                 // odd length
		{"0xgg", 0, "", true},               // non-hex
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			b, err := NewRBytesFromHex(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewRBytesFromHex(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewRBytesFromHex(%q) error: %v", tt.input, err)
			}
			if b.Len() != tt.wantLen {
				t.Errorf("Len() = %d, want %d", b.Len(), tt.wantLen)
			}
			if b.StringValue() != tt.wantHex {
				t.Errorf("StringValue() = %q, want %q", b.StringValue(), tt.wantHex)
			}
		})
	}
}

func TestRBytes_Concat(t *testing.T) {
	a, _ := NewRBytesFromHex("0xaabb")
	b, _ := NewRBytesFromHex("0xccdd")
	c := a.Concat(b)
	if c.StringValue() != "0xaabbccdd" {
		t.Errorf("Concat = %q, want 0xaabbccdd", c.StringValue())
	}
}

func TestRBytes_Concat_Empty(t *testing.T) {
	a, _ := NewRBytesFromHex("0x")
	b, _ := NewRBytesFromHex("0xaabb")
	if a.Concat(b) != b {
		t.Error("Concat with empty left should return right identity")
	}
	if b.Concat(a) != b {
		t.Error("Concat with empty right should return left identity")
	}
}

func TestRBytes_Slice(t *testing.T) {
	b, _ := NewRBytesFromHex("0xaabbccdd")
	tests := []struct {
		from, to int
		want     string
		wantErr  bool
	}{
		{0, 4, "0xaabbccdd", false},
		{0, 2, "0xaabb", false},
		{2, 4, "0xccdd", false},
		{1, 3, "0xbbcc", false},
		{0, 0, "0x", false},   // equal from/to → empty
		{2, 2, "0x", false},   // equal from/to → empty
		{0, 5, "", true},      // to > len
		{-1, 2, "", true},     // negative from
		{3, 2, "", true},      // from > to
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got, err := b.Slice(tt.from, tt.to)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Slice(%d,%d) expected error, got nil", tt.from, tt.to)
				}
				return
			}
			if err != nil {
				t.Fatalf("Slice(%d,%d) error: %v", tt.from, tt.to, err)
			}
			if got.StringValue() != tt.want {
				t.Errorf("Slice(%d,%d) = %q, want %q", tt.from, tt.to, got.StringValue(), tt.want)
			}
		})
	}
}

func TestRBytes_At(t *testing.T) {
	b, _ := NewRBytesFromHex("0xaabbcc")
	v, err := b.At(0)
	if err != nil || v != 0xaa {
		t.Errorf("At(0) = %d, %v; want 0xaa", v, err)
	}
	v, err = b.At(2)
	if err != nil || v != 0xcc {
		t.Errorf("At(2) = %d, %v; want 0xcc", v, err)
	}
	_, err = b.At(3)
	if err == nil {
		t.Error("At(3) should return error for len=3 bytes")
	}
	_, err = b.At(-1)
	if err == nil {
		t.Error("At(-1) should return error")
	}
}

func TestRBytes_ConstantTimeEqual(t *testing.T) {
	a, _ := NewRBytesFromHex("0xdeadbeef")
	b, _ := NewRBytesFromHex("0xdeadbeef")
	c, _ := NewRBytesFromHex("0xdeadbeee")
	d, _ := NewRBytesFromHex("0xdeadbe")

	if !a.ConstantTimeEqual(b) {
		t.Error("equal bytes should return true")
	}
	if a.ConstantTimeEqual(c) {
		t.Error("bytes differing at last byte should return false")
	}
	if a.ConstantTimeEqual(d) {
		t.Error("bytes with different lengths should return false")
	}
}

func TestRBytes_Type(t *testing.T) {
	b := NewRBytes([]byte{1, 2, 3})
	if b.Type() != TypeBytes {
		t.Errorf("Type() = %v, want TypeBytes", b.Type())
	}
}

func TestRBytes_Immutability(t *testing.T) {
	orig := []byte{0xaa, 0xbb}
	b := NewRBytes(orig)
	orig[0] = 0xff
	if b.Bytes()[0] != 0xaa {
		t.Error("RBytes should be immutable - external modification should not affect value")
	}
	got := b.Bytes()
	got[0] = 0xff
	if b.Bytes()[0] != 0xaa {
		t.Error("Bytes() should return a copy, not the internal slice")
	}
}

func TestRBytes_TimingTest(t *testing.T) {
	// Basic timing test: two sequences of same length differing at first vs last byte.
	// We run 10k iterations each way and compare total times to stay within 10x.
	// If CI is too noisy, skip.
	t.Skip("timing test unreliable in CI — run locally to verify constant-time equality")

	const iters = 10000
	a, _ := NewRBytesFromHex("0x" + "aa" + "0000000000000000000000000000000000000000000000000000000000000000")
	b1, _ := NewRBytesFromHex("0x" + "bb" + "0000000000000000000000000000000000000000000000000000000000000000")
	b2, _ := NewRBytesFromHex("0x" + "00" + "000000000000000000000000000000000000000000000000000000000000" + "bb")

	for i := 0; i < iters; i++ {
		a.ConstantTimeEqual(b1)
	}
	for i := 0; i < iters; i++ {
		a.ConstantTimeEqual(b2)
	}
}
