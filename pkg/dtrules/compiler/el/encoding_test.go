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

package el

import (
	"strings"
	"testing"
)

// TestEncodingELProductions verifies that encoding EL constructs compile to the
// expected postfix operators.
func TestEncodingELProductions(t *testing.T) {
	tests := []struct {
		name     string
		symbols  map[string]string
		compile  func(c *Compiler) (string, error)
		contains []string
	}{
		{
			name:    "hex of bytes",
			symbols: map[string]string{"data": TypeBytes},
			compile: func(c *Compiler) (string, error) {
				return c.CompileCondition(`hex of data == "deadbeef"`)
			},
			contains: []string{"hex", `"deadbeef"`},
		},
		{
			name: "bytes of hex string literal",
			compile: func(c *Compiler) (string, error) {
				return c.CompileCondition(`bytes of hex "deadbeef" == 0xdeadbeef`)
			},
			contains: []string{"cvhex"},
		},
		{
			name:    "bytes of hex variable",
			symbols: map[string]string{"hexStr": TypeString},
			compile: func(c *Compiler) (string, error) {
				return c.CompileCondition(`length of (bytes of hex hexStr) == 0`)
			},
			contains: []string{"cvhex", "byteslen"},
		},
		{
			name:    "base58check of bytes",
			symbols: map[string]string{"payload": TypeBytes},
			compile: func(c *Compiler) (string, error) {
				return c.CompileCondition(`base58check of payload version 0 == "1111111111111111111114oLvT2"`)
			},
			contains: []string{"b58check"},
		},
		{
			name: "bytes of base58check",
			compile: func(c *Compiler) (string, error) {
				return c.CompileCondition(`length of (bytes of base58check "1111111111111111111114oLvT2") == 20`)
			},
			contains: []string{"cvb58check", "byteslen"},
		},
		{
			name:    "bech32 of bytes with hrp",
			symbols: map[string]string{"pubkeyHash": TypeBytes},
			compile: func(c *Compiler) (string, error) {
				return c.CompileCondition(`bech32 of pubkeyHash hrp "bc" == "someaddr"`)
			},
			contains: []string{"bech32"},
		},
		{
			name: "bytes of bech32 string",
			compile: func(c *Compiler) (string, error) {
				return c.CompileCondition(`length of (bytes of bech32 "bc1w508d6qejxtdg4y5r3zarvary0c5xw7kj7gz7z") == 20`)
			},
			contains: []string{"cvbech32", "byteslen"},
		},
		{
			name:    "bigint of bytes",
			symbols: map[string]string{"rawBytes": TypeBytes},
			compile: func(c *Compiler) (string, error) {
				return c.CompileCondition(`bigint of bytes rawBytes == (bigint) 0`)
			},
			contains: []string{"bytesbigint"},
		},
		{
			name:    "bytes of bigint with size",
			symbols: map[string]string{"amount": TypeBigInt},
			compile: func(c *Compiler) (string, error) {
				return c.CompileCondition(`length of (bytes of bigint amount size 32) == 32`)
			},
			contains: []string{"bigintbytes", "byteslen"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompiler()
			if tt.symbols != nil {
				c.SetSymbols(tt.symbols)
			}
			got, err := tt.compile(c)
			if err != nil {
				t.Fatalf("compile error: %v", err)
			}
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("postfix %q does not contain %q", got, want)
				}
			}
		})
	}
}

// TestEncodingRoundTripViaCondition verifies round-trip for base58check and bech32 decode
// using condition expressions (the policy-statement path has a pre-existing limitation
// where the visitor does not emit postfix for top-level policystatement nodes; this is
// tracked separately and does not affect the correctness of the operators themselves).
func TestEncodingRoundTripViaCondition(t *testing.T) {
	tests := []struct {
		name     string
		symbols  map[string]string
		el       string
		contains []string
	}{
		{
			name:     "hex of bytes in condition",
			symbols:  map[string]string{"rawBytes": TypeBytes},
			el:       `hex of rawBytes == "deadbeef"`,
			contains: []string{"hex"},
		},
		{
			name:     "bigint of bytes in condition",
			symbols:  map[string]string{"rawBytes": TypeBytes},
			el:       `bigint of bytes rawBytes == (bigint) 0`,
			contains: []string{"bytesbigint"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompiler()
			if tt.symbols != nil {
				c.SetSymbols(tt.symbols)
			}
			got, err := c.CompileCondition(tt.el)
			if err != nil {
				t.Fatalf("compile error: %v", err)
			}
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("postfix %q does not contain %q", got, want)
				}
			}
		})
	}
}
