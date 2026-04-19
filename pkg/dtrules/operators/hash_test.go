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
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// Behavior tests for the hash family: sha256, keccak256, ripemd160, sha3.
// These ops are used in blockchain / crypto rules — a silent-wrong-answer
// on a hash is catastrophic (would break signatures, invalidate Merkle
// proofs, etc.). Pinning against the known-standard test vectors is the
// correct insurance.

// runHash pushes a byte-string input and runs the named hash op,
// returning the result as a hex string for comparison.
func runHash(t *testing.T, op string, input []byte) string {
	t.Helper()
	state := newTestState()
	bytes, err := dtrules.NewRBytesFromHex("0x" + hex.EncodeToString(input))
	if err != nil {
		t.Fatalf("NewRBytesFromHex: %v", err)
	}
	if err := state.DataPush(bytes); err != nil {
		t.Fatalf("DataPush: %v", err)
	}
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
	// RBytes has a PostFix form but the hex string is more comparable.
	return b.StringValue()
}

// TestSha256KnownVector — empty input and a well-known "abc" vector.
// https://www.di-mgt.com.au/sha_testvectors.html
func TestSha256KnownVector(t *testing.T) {
	cases := []struct {
		input []byte
		want  string
	}{
		{
			input: []byte{}, // empty
			want:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			input: []byte("abc"),
			want:  "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
		},
	}
	for _, c := range cases {
		got := runHash(t, "sha256", c.input)
		// Strip leading 0x if present.
		if len(got) >= 2 && got[0:2] == "0x" {
			got = got[2:]
		}
		if got != c.want {
			t.Errorf("sha256(%q) = %s, want %s", c.input, got, c.want)
		}
	}
}

// TestKeccak256KnownVector — Keccak-256 of empty string (the EVM hash).
func TestKeccak256KnownVector(t *testing.T) {
	// keccak256("") = c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470
	got := runHash(t, "keccak256", []byte{})
	if len(got) >= 2 && got[0:2] == "0x" {
		got = got[2:]
	}
	want := "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"
	if got != want {
		t.Errorf("keccak256(empty) = %s, want %s", got, want)
	}
}

// TestRipemd160KnownVector — RIPEMD-160 of "abc".
func TestRipemd160KnownVector(t *testing.T) {
	// ripemd160("abc") = 8eb208f7e05d987a9b044a8e98c6b087f15a0bfc
	got := runHash(t, "ripemd160", []byte("abc"))
	if len(got) >= 2 && got[0:2] == "0x" {
		got = got[2:]
	}
	want := "8eb208f7e05d987a9b044a8e98c6b087f15a0bfc"
	if got != want {
		t.Errorf("ripemd160(\"abc\") = %s, want %s", got, want)
	}
}

// TestSha3KnownVector — SHA3-256 of empty.
func TestSha3KnownVector(t *testing.T) {
	// sha3_256("") = a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a
	got := runHash(t, "sha3", []byte{})
	if len(got) >= 2 && got[0:2] == "0x" {
		got = got[2:]
	}
	want := "a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a"
	if got != want {
		t.Errorf("sha3(empty) = %s, want %s", got, want)
	}
}

// TestHashOutputLengths — all hashes should produce fixed-width output.
func TestHashOutputLengths(t *testing.T) {
	cases := map[string]int{
		"sha256":    32, // 32 bytes = 64 hex chars
		"keccak256": 32,
		"ripemd160": 20,
		"sha3":      32,
	}
	for op, wantLen := range cases {
		t.Run(op, func(t *testing.T) {
			got := runHash(t, op, []byte("any input"))
			hexStr := got
			if len(hexStr) >= 2 && hexStr[0:2] == "0x" {
				hexStr = hexStr[2:]
			}
			if len(hexStr) != wantLen*2 {
				t.Errorf("%s produced %d hex chars, want %d", op, len(hexStr), wantLen*2)
			}
		})
	}
}
