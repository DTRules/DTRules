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
	"math/big"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
)

func newEncodingTestState() *interpreter.DTState {
	return interpreter.NewDTState(&mockSession{})
}

func runOp(t *testing.T, state *interpreter.DTState, opName string) {
	t.Helper()
	op, ok := Get(dtrules.GetRName(opName))
	if !ok {
		t.Fatalf("operator %q not found", opName)
	}
	if err := op.Execute(state); err != nil {
		t.Fatalf("operator %q failed: %v", opName, err)
	}
}

// ============================================================================
// Hex round-trip
// ============================================================================

func TestHexRoundTrip(t *testing.T) {
	state := newEncodingTestState()

	original := dtrules.NewRBytes([]byte{0xde, 0xad, 0xbe, 0xef})
	state.DataPush(original)
	runOp(t, state, "hex")

	strVal, err := state.DataPop()
	if err != nil {
		t.Fatalf("pop failed: %v", err)
	}
	if strVal.StringValue() != "deadbeef" {
		t.Errorf("hex: expected deadbeef, got %q", strVal.StringValue())
	}

	// Round-trip: cvhex
	state.DataPush(strVal)
	runOp(t, state, "cvhex")

	result, err := state.DataPop()
	if err != nil {
		t.Fatalf("pop failed: %v", err)
	}
	rb, err := result.RBytesValue()
	if err != nil {
		t.Fatalf("RBytesValue: %v", err)
	}
	if !rb.ConstantTimeEqual(original) {
		t.Errorf("hex round-trip failed: got %v, want %v", rb, original)
	}
}

func TestHexNoPrefix(t *testing.T) {
	state := newEncodingTestState()
	state.DataPush(dtrules.NewRBytes([]byte{0xab, 0xcd}))
	runOp(t, state, "hex")
	val, _ := state.DataPop()
	if got := val.StringValue(); got != "abcd" {
		t.Errorf("hex: expected 'abcd' (no 0x prefix), got %q", got)
	}
}

func TestCvHexAccepts0xPrefix(t *testing.T) {
	state := newEncodingTestState()
	state.DataPush(dtrules.NewRString("0xdeadbeef"))
	runOp(t, state, "cvhex")
	val, _ := state.DataPop()
	rb, _ := val.RBytesValue()
	expected := dtrules.NewRBytes([]byte{0xde, 0xad, 0xbe, 0xef})
	if !rb.ConstantTimeEqual(expected) {
		t.Errorf("cvhex with 0x prefix: got %v, want %v", rb, expected)
	}
}

func TestCvHexOddLength(t *testing.T) {
	state := newEncodingTestState()
	state.DataPush(dtrules.NewRString("abc"))
	op, _ := Get(dtrules.GetRName("cvhex"))
	if err := op.Execute(state); err == nil {
		t.Error("cvhex should fail on odd-length hex string")
	}
}

func TestCvHexNonHexChar(t *testing.T) {
	state := newEncodingTestState()
	state.DataPush(dtrules.NewRString("zzzz"))
	op, _ := Get(dtrules.GetRName("cvhex"))
	if err := op.Execute(state); err == nil {
		t.Error("cvhex should fail on non-hex characters")
	}
}

// ============================================================================
// Base58check round-trip
// ============================================================================

func TestBase58CheckRoundTrip(t *testing.T) {
	state := newEncodingTestState()

	payload := dtrules.NewRBytes(make([]byte, 20))
	version := dtrules.GetRIntegerValue(0)

	state.DataPush(payload)
	state.DataPush(version)
	runOp(t, state, "b58check")

	encoded, _ := state.DataPop()

	// Decode
	state.DataPush(encoded)
	runOp(t, state, "cvb58check")

	versionOut, _ := state.DataPop()
	payloadOut, _ := state.DataPop()

	if v, _ := versionOut.IntValue(); v != 0 {
		t.Errorf("base58check round-trip: version mismatch, got %d, want 0", v)
	}
	rb, _ := payloadOut.RBytesValue()
	if !rb.ConstantTimeEqual(payload) {
		t.Errorf("base58check round-trip: payload mismatch")
	}
}

func TestBase58CheckKnownAnswer(t *testing.T) {
	// Bitcoin "null address": 20 zero bytes, version 0 → "1111111111111111111114oLvT2"
	state := newEncodingTestState()
	state.DataPush(dtrules.NewRBytes(make([]byte, 20)))
	state.DataPush(dtrules.GetRIntegerValue(0))
	runOp(t, state, "b58check")

	encoded, _ := state.DataPop()
	const want = "1111111111111111111114oLvT2"
	if encoded.StringValue() != want {
		t.Errorf("base58check known-answer: got %q, want %q", encoded.StringValue(), want)
	}
}

func TestBase58CheckBadChecksum(t *testing.T) {
	state := newEncodingTestState()
	state.DataPush(dtrules.NewRString("1111111111111111111114oLvT3")) // last char changed
	op, _ := Get(dtrules.GetRName("cvb58check"))
	if err := op.Execute(state); err == nil {
		t.Error("cvb58check should fail on bad checksum")
	}
}

// ============================================================================
// Bech32 round-trip
// ============================================================================

func TestBech32RoundTrip(t *testing.T) {
	state := newEncodingTestState()

	payload := dtrules.NewRBytes([]byte{0x75, 0x1e, 0x76, 0xe8, 0x19, 0x91, 0x96, 0xd4, 0x54, 0x94, 0x1c, 0x45, 0xd1, 0xb3, 0xa3, 0x23, 0xf1, 0x43, 0x3b, 0xd6})
	hrp := dtrules.NewRString("bc")

	state.DataPush(payload)
	state.DataPush(hrp)
	runOp(t, state, "bech32")

	encoded, _ := state.DataPop()

	// Decode
	state.DataPush(encoded)
	runOp(t, state, "cvbech32")

	hrpOut, _ := state.DataPop()
	payloadOut, _ := state.DataPop()

	if hrpOut.StringValue() != "bc" {
		t.Errorf("bech32 round-trip: hrp mismatch, got %q, want bc", hrpOut.StringValue())
	}
	rb, _ := payloadOut.RBytesValue()
	if !rb.ConstantTimeEqual(payload) {
		t.Errorf("bech32 round-trip: payload mismatch")
	}
}

// Known-answer: 20-byte hash with hrp "bc" → bc1w508d6qejxtdg4y5r3zarvary0c5xw7kj7gz7z
// (raw bech32, not SegWit witness-version format)
func TestBech32KnownVector(t *testing.T) {
	payload := []byte{0x75, 0x1e, 0x76, 0xe8, 0x19, 0x91, 0x96, 0xd4, 0x54, 0x94, 0x1c, 0x45, 0xd1, 0xb3, 0xa3, 0x23, 0xf1, 0x43, 0x3b, 0xd6}
	state := newEncodingTestState()
	state.DataPush(dtrules.NewRBytes(payload))
	state.DataPush(dtrules.NewRString("bc"))
	runOp(t, state, "bech32")

	encoded, _ := state.DataPop()
	const want = "bc1w508d6qejxtdg4y5r3zarvary0c5xw7kj7gz7z"
	if encoded.StringValue() != want {
		t.Errorf("bech32 known vector: got %q, want %q", encoded.StringValue(), want)
	}

	// Decode it back
	state.DataPush(encoded)
	runOp(t, state, "cvbech32")
	hrp, _ := state.DataPop()
	payloadOut, _ := state.DataPop()

	if hrp.StringValue() != "bc" {
		t.Errorf("bech32 vector: hrp = %q, want bc", hrp.StringValue())
	}
	rb, _ := payloadOut.RBytesValue()
	expected := dtrules.NewRBytes(payload)
	if !rb.ConstantTimeEqual(expected) {
		t.Errorf("bech32 vector: decoded payload mismatch")
	}
}

func TestBech32BadChecksum(t *testing.T) {
	state := newEncodingTestState()
	// Corrupt the last char of a valid bech32 string
	state.DataPush(dtrules.NewRString("bc1w508d6qejxtdg4y5r3zarvary0c5xw7kj7gz7q")) // last char changed
	op, _ := Get(dtrules.GetRName("cvbech32"))
	if err := op.Execute(state); err == nil {
		t.Error("cvbech32 should fail on bad checksum")
	}
}

// ============================================================================
// BigInt <-> bytes round-trip
// ============================================================================

func TestBigIntBytesRoundTrip(t *testing.T) {
	state := newEncodingTestState()

	n := new(big.Int).SetInt64(256)
	state.DataPush(dtrules.NewRBigInt(n))
	state.DataPush(dtrules.GetRIntegerValue(2))
	runOp(t, state, "bigintbytes")

	bVal, _ := state.DataPop()
	rb, _ := bVal.RBytesValue()
	if rb.Len() != 2 {
		t.Errorf("bigintbytes: length = %d, want 2", rb.Len())
	}
	// 256 = 0x0100
	b := rb.Bytes()
	if b[0] != 0x01 || b[1] != 0x00 {
		t.Errorf("bigintbytes: got %x, want 0100", b)
	}

	// Round-trip
	state.DataPush(bVal)
	runOp(t, state, "bytesbigint")
	result, _ := state.DataPop()
	bi, _ := result.RBigIntValue()
	if bi.BigIntValue().Cmp(n) != 0 {
		t.Errorf("bytesbigint round-trip: got %v, want %v", bi.BigIntValue(), n)
	}
}

func TestBigIntBytesKnownAnswer(t *testing.T) {
	// 256 in 2 bytes → 0x0100
	state := newEncodingTestState()
	state.DataPush(dtrules.NewRBigInt(big.NewInt(256)))
	state.DataPush(dtrules.GetRIntegerValue(2))
	runOp(t, state, "bigintbytes")

	val, _ := state.DataPop()
	rb, _ := val.RBytesValue()
	b := rb.Bytes()
	if b[0] != 0x01 || b[1] != 0x00 {
		t.Errorf("bigintbytes 256 size 2: got %x, want 0100", b)
	}
}

func TestBigIntBytesTooLarge(t *testing.T) {
	state := newEncodingTestState()
	// 256 doesn't fit in 1 byte
	state.DataPush(dtrules.NewRBigInt(big.NewInt(256)))
	state.DataPush(dtrules.GetRIntegerValue(1))
	op, _ := Get(dtrules.GetRName("bigintbytes"))
	if err := op.Execute(state); err == nil {
		t.Error("bigintbytes should fail when value doesn't fit")
	}
}
