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

// TestBytesLiteralValid tests that valid hex byte literals compile correctly.
func TestBytesLiteralValid(t *testing.T) {
	tests := []struct {
		name  string
		el    string
		wantS string
	}{
		{
			name:  "empty literal equality",
			el:    "0x is equal to 0x",
			wantS: "cvbytes",
		},
		{
			name:  "two-byte literal equality",
			el:    "0xdeadbeef is equal to 0xdeadbeef",
			wantS: "cvbytes",
		},
		{
			name:  "uppercase hex",
			el:    "0xDEADBEEF is equal to 0xDEADBEEF",
			wantS: "cvbytes",
		},
	}

	c := NewCompiler()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.CompileCondition(tt.el)
			if err != nil {
				t.Fatalf("CompileCondition(%q) error: %v", tt.el, err)
			}
			if !strings.Contains(got, tt.wantS) {
				t.Errorf("CompileCondition(%q) = %q; expected %q in output", tt.el, got, tt.wantS)
			}
		})
	}
}

// TestBytesConcat tests that bytes concatenation compiles to bytes+.
func TestBytesConcat(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"prefix": TypeBytes,
		"suffix": TypeBytes,
	})
	// Use equality with concat to force bytesexpr parsing
	got, err := c.CompileCondition("(prefix + suffix) is equal to (prefix + suffix)")
	if err != nil {
		t.Fatalf("CompileCondition error: %v", err)
	}
	if !strings.Contains(got, "bytes+") {
		t.Errorf("bytes concat should produce bytes+, got: %q", got)
	}
}

// TestBytesSlice tests that slice compiles to bytesslice with correct argument order.
func TestBytesSlice(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{"payload": TypeBytes})
	// bytes from <int> to <int> returns bytesexpr; check it compiles without error
	// and we can take length of the result
	got, err := c.CompileCondition("length of (payload from 0 to 2) is equal to 2")
	if err != nil {
		t.Fatalf("CompileCondition error: %v", err)
	}
	if !strings.Contains(got, "bytesslice") {
		t.Errorf("bytes slice should produce bytesslice, got: %q", got)
	}
}

// TestBytesEquality tests that bytes equality compiles to bytes==.
func TestBytesEquality(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"hash":     TypeBytes,
		"expected": TypeBytes,
	})
	got, err := c.CompileCondition("hash is equal to expected")
	if err != nil {
		t.Fatalf("CompileCondition error: %v", err)
	}
	if !strings.Contains(got, "bytes==") {
		t.Errorf("bytes equality should produce bytes==, got: %q", got)
	}
}

// TestBytesInequality tests that bytes inequality compiles to bytes!=.
func TestBytesInequality(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"hash":     TypeBytes,
		"expected": TypeBytes,
	})
	got, err := c.CompileCondition("hash is not equal to expected")
	if err != nil {
		t.Fatalf("CompileCondition error: %v", err)
	}
	if !strings.Contains(got, "bytes!=") {
		t.Errorf("bytes inequality should produce bytes!=, got: %q", got)
	}
}

// TestBytesLength tests that length of bytes compiles to byteslen.
func TestBytesLength(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{"data": TypeBytes})
	got, err := c.CompileCondition("length of data is equal to 0")
	if err != nil {
		t.Fatalf("CompileCondition error: %v", err)
	}
	if !strings.Contains(got, "byteslen") {
		t.Errorf("length of bytes should produce byteslen, got: %q", got)
	}
}

// TestBytesIndex tests that bytes[i] compiles to bytesidx.
func TestBytesIndex(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{"data": TypeBytes})
	got, err := c.CompileCondition("data[0] is equal to 255")
	if err != nil {
		t.Fatalf("CompileCondition error: %v", err)
	}
	if !strings.Contains(got, "bytesidx") {
		t.Errorf("bytes[i] should produce bytesidx, got: %q", got)
	}
}

// TestBytesLocalDecl tests local bytes variable declarations.
func TestBytesLocalDecl(t *testing.T) {
	c := NewCompiler()
	got, err := c.CompileContext("local bytes myHash")
	if err != nil {
		t.Fatalf("CompileContext error: %v", err)
	}
	if !strings.Contains(got, "null") {
		t.Errorf("local bytes decl should produce null allocate ..., got: %q", got)
	}
}

// TestBytesLocalDeclWithInit tests local bytes variable declaration with initializer.
func TestBytesLocalDeclWithInit(t *testing.T) {
	c := NewCompiler()
	got, err := c.CompileContext("local bytes myHash = 0xdeadbeef")
	if err != nil {
		t.Fatalf("CompileContext error: %v", err)
	}
	if !strings.Contains(got, "cvbytes") {
		t.Errorf("local bytes init should produce cvbytes, got: %q", got)
	}
}

// TestHashBuiltinsPostfix tests that hash built-ins compile to their postfix operators.
func TestHashBuiltinsPostfix(t *testing.T) {
	tests := []struct {
		name     string
		el       string
		contains string
	}{
		{
			name:     "sha256 literal",
			el:       "sha256 of 0xdeadbeef is equal to 0xdeadbeef",
			contains: "sha256",
		},
		{
			name:     "keccak256 literal",
			el:       "keccak256 of 0xdeadbeef is equal to 0xdeadbeef",
			contains: "keccak256",
		},
		{
			name:     "ripemd160 literal",
			el:       "ripemd160 of 0xdeadbeef is equal to 0xdeadbeef",
			contains: "ripemd160",
		},
		{
			name:     "sha3 literal",
			el:       "sha3 of 0xdeadbeef is equal to 0xdeadbeef",
			contains: "sha3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompiler()
			got, err := c.CompileCondition(tt.el)
			if err != nil {
				t.Fatalf("CompileCondition(%q) error: %v", tt.el, err)
			}
			if !strings.Contains(got, tt.contains) {
				t.Errorf("CompileCondition(%q) = %q; expected %q in output", tt.el, got, tt.contains)
			}
		})
	}
}

// TestHashBuiltinsSymbolTyped tests hash built-ins with symbol-typed bytes variables.
func TestHashBuiltinsSymbolTyped(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{"payload": TypeBytes, "expected": TypeBytes})
	got, err := c.CompileCondition("sha256 of payload is equal to expected")
	if err != nil {
		t.Fatalf("CompileCondition error: %v", err)
	}
	if !strings.Contains(got, "sha256") {
		t.Errorf("sha256 of typed bytes should produce sha256, got: %q", got)
	}
}

// TestHashBuiltinsNonBytesError tests that applying a hash to a non-bytes expression fails.
func TestHashBuiltinsNonBytesError(t *testing.T) {
	c := NewCompiler()
	_, err := c.CompileCondition("sha256 of 42 is equal to 0x00")
	if err == nil {
		t.Error("sha256 of non-bytes integer should fail at compile time")
	}
}

// TestBytesPostfixGolden tests postfix round-trip for key bytes constructs.
func TestBytesPostfixGolden(t *testing.T) {
	tests := []struct {
		name     string
		compile  func(c *Compiler) (string, error)
		contains string
	}{
		{
			name: "literal",
			compile: func(c *Compiler) (string, error) {
				return c.CompileCondition("0xaabbcc is equal to 0xaabbcc")
			},
			contains: "cvbytes",
		},
		{
			name: "equality",
			compile: func(c *Compiler) (string, error) {
				c.SetSymbols(map[string]string{"x": TypeBytes, "y": TypeBytes})
				return c.CompileCondition("x is equal to y")
			},
			contains: "bytes==",
		},
		{
			name: "concat",
			compile: func(c *Compiler) (string, error) {
				c.SetSymbols(map[string]string{"prefix": TypeBytes, "suffix": TypeBytes})
				return c.CompileCondition("(prefix + suffix) is equal to (prefix + suffix)")
			},
			contains: "bytes+",
		},
		{
			name: "slice",
			compile: func(c *Compiler) (string, error) {
				c.SetSymbols(map[string]string{"data": TypeBytes})
				return c.CompileCondition("length of (data from 0 to 4) is equal to 4")
			},
			contains: "bytesslice",
		},
		{
			name: "length",
			compile: func(c *Compiler) (string, error) {
				c.SetSymbols(map[string]string{"data": TypeBytes})
				return c.CompileCondition("length of data is equal to 32")
			},
			contains: "byteslen",
		},
		{
			name: "index",
			compile: func(c *Compiler) (string, error) {
				c.SetSymbols(map[string]string{"data": TypeBytes})
				return c.CompileCondition("data[0] is equal to 255")
			},
			contains: "bytesidx",
		},
		{
			name: "sha256",
			compile: func(c *Compiler) (string, error) {
				return c.CompileCondition("sha256 of 0xdeadbeef is equal to 0xdeadbeef")
			},
			contains: "sha256",
		},
		{
			name: "keccak256",
			compile: func(c *Compiler) (string, error) {
				return c.CompileCondition("keccak256 of 0xdeadbeef is equal to 0xdeadbeef")
			},
			contains: "keccak256",
		},
		{
			name: "ripemd160",
			compile: func(c *Compiler) (string, error) {
				return c.CompileCondition("ripemd160 of 0xdeadbeef is equal to 0xdeadbeef")
			},
			contains: "ripemd160",
		},
		{
			name: "sha3",
			compile: func(c *Compiler) (string, error) {
				return c.CompileCondition("sha3 of 0xdeadbeef is equal to 0xdeadbeef")
			},
			contains: "sha3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompiler()
			got, err := tt.compile(c)
			if err != nil {
				t.Fatalf("compile error: %v", err)
			}
			if !strings.Contains(got, tt.contains) {
				t.Errorf("postfix %q does not contain %q", got, tt.contains)
			}
		})
	}
}
