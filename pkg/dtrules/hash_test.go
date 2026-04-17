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

import "testing"

// Known-answer tests using canonical vectors (empty-string inputs).
func TestRBytes_Sha256_EmptyString(t *testing.T) {
	// SHA-256("") = e3b0c44298fc1c149afbf4c8996fb924...
	empty := NewRBytes([]byte{})
	got := empty.Sha256().StringValue()
	want := "0xe3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got != want {
		t.Errorf("Sha256(\"\") = %q, want %q", got, want)
	}
}

func TestRBytes_Sha3_EmptyString(t *testing.T) {
	// SHA3-256("") = a7ffc6f8bf1ed76651c14756a061d662...
	empty := NewRBytes([]byte{})
	got := empty.Sha3().StringValue()
	want := "0xa7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a"
	if got != want {
		t.Errorf("Sha3(\"\") = %q, want %q", got, want)
	}
}

func TestRBytes_Keccak256_EmptyString(t *testing.T) {
	// Keccak-256("") = c5d2460186f7233c927e7db2dcc703c0...
	empty := NewRBytes([]byte{})
	got := empty.Keccak256().StringValue()
	want := "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"
	if got != want {
		t.Errorf("Keccak256(\"\") = %q, want %q", got, want)
	}
}

func TestRBytes_Ripemd160_EmptyString(t *testing.T) {
	// RIPEMD-160("") = 9c1185a5c5e9fc54612808977ee8f548b2258d31
	empty := NewRBytes([]byte{})
	got := empty.Ripemd160().StringValue()
	want := "0x9c1185a5c5e9fc54612808977ee8f548b2258d31"
	if got != want {
		t.Errorf("Ripemd160(\"\") = %q, want %q", got, want)
	}
}

func TestRBytes_Sha256_OutputLength(t *testing.T) {
	b := NewRBytes([]byte("hello"))
	if got := b.Sha256().Len(); got != 32 {
		t.Errorf("Sha256 output length = %d, want 32", got)
	}
}

func TestRBytes_Keccak256_OutputLength(t *testing.T) {
	b := NewRBytes([]byte("hello"))
	if got := b.Keccak256().Len(); got != 32 {
		t.Errorf("Keccak256 output length = %d, want 32", got)
	}
}

func TestRBytes_Ripemd160_OutputLength(t *testing.T) {
	b := NewRBytes([]byte("hello"))
	if got := b.Ripemd160().Len(); got != 20 {
		t.Errorf("Ripemd160 output length = %d, want 20", got)
	}
}

func TestRBytes_Sha3_OutputLength(t *testing.T) {
	b := NewRBytes([]byte("hello"))
	if got := b.Sha3().Len(); got != 32 {
		t.Errorf("Sha3 output length = %d, want 32", got)
	}
}

// Sha256 and Keccak256 must differ (they are different algorithms).
func TestRBytes_Sha256VsKeccak256_Differ(t *testing.T) {
	b := NewRBytes([]byte("hello"))
	s256 := b.Sha256()
	k256 := b.Keccak256()
	if s256.ConstantTimeEqual(k256) {
		t.Error("SHA-256 and Keccak-256 produced identical output for non-empty input")
	}
}
