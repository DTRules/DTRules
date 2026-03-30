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

func TestCompileCondition(t *testing.T) {
	tests := []struct {
		name     string
		el       string
		expected string
	}{
		{
			name:     "simple equality",
			el:       "x is equal to 5",
			expected: "x 5 ==", // Integer comparison uses ==
		},
		{
			name:     "greater than",
			el:       "age is greater than 18",
			expected: "age 18 >", // Integer comparison uses >
		},
		{
			name:     "less than or equal",
			el:       "count is less than or equal to 10",
			expected: "count 10 <=", // Integer comparison uses <=
		},
		{
			name:     "boolean literal true",
			el:       "true",
			expected: "true",
		},
		{
			name:     "boolean literal false",
			el:       "false",
			expected: "false",
		},
		{
			name:     "otherwise",
			el:       "otherwise",
			expected: "otherwise",
		},
		{
			name:     "and expression",
			el:       "x is equal to 1 and y is equal to 2",
			expected: "x 1 == { pop y 2 == } over if", // Lazy evaluation
		},
		{
			name:     "or expression",
			el:       "x is equal to 1 or y is equal to 2",
			expected: "x 1 == { pop y 2 == } over not if", // Lazy evaluation
		},
		{
			name:     "not expression",
			el:       "not x is equal to 1",
			expected: "x 1 == not",
		},
		{
			name:     "string equality",
			el:       `status is equal to "active"`,
			expected: `status "active" streq`, // Name compared with string uses streq
		},
	}

	c := NewCompiler()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := c.CompileCondition(tt.el)
			if err != nil {
				t.Fatalf("CompileCondition(%q) error: %v", tt.el, err)
			}

			// Normalize whitespace for comparison
			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)

			if result != expected {
				t.Errorf("CompileCondition(%q)\n  got:  %q\n  want: %q", tt.el, result, expected)
			}
		})
	}
}

func TestCompileAction(t *testing.T) {
	tests := []struct {
		name     string
		el       string
		expected string
	}{
		{
			name:     "set integer",
			el:       "set count = 0;",
			expected: "0 cvi /count xdef", // Type conversion + left value format
		},
		{
			name:     "set boolean",
			el:       "set eligible = true;",
			expected: "true cvb /eligible xdef", // Type conversion + left value format
		},
		{
			name:     "perform table",
			el:       "perform Calculate_Tax;",
			expected: "Calculate_Tax", // Just table name, no executetable
		},
	}

	c := NewCompiler()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := c.CompileAction(tt.el)
			if err != nil {
				t.Fatalf("CompileAction(%q) error: %v", tt.el, err)
			}

			// Normalize whitespace for comparison
			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)

			if result != expected {
				t.Errorf("CompileAction(%q)\n  got:  %q\n  want: %q", tt.el, result, expected)
			}
		})
	}
}

func TestCompileArithmetic(t *testing.T) {
	tests := []struct {
		name     string
		el       string
		expected string
	}{
		{
			name:     "addition",
			el:       "x + y is equal to 10",
			expected: "x y + 10 ==", // Integer comparison uses ==
		},
		{
			name:     "subtraction",
			el:       "x - y is equal to 5",
			expected: "x y - 5 ==",
		},
		{
			name:     "multiplication",
			el:       "x * y is equal to 20",
			expected: "x y * 20 ==",
		},
		{
			name:     "division",
			el:       "x / y is equal to 2",
			expected: "x y / 2 ==",
		},
		{
			name:     "complex expression",
			el:       "x + y * z is equal to 10",
			expected: "x y z * + 10 ==",
		},
	}

	c := NewCompiler()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := c.CompileCondition(tt.el)
			if err != nil {
				t.Fatalf("CompileCondition(%q) error: %v", tt.el, err)
			}

			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)

			if result != expected {
				t.Errorf("CompileCondition(%q)\n  got:  %q\n  want: %q", tt.el, result, expected)
			}
		})
	}
}

func TestCompileErrors(t *testing.T) {
	tests := []struct {
		name string
		el   string
	}{
		{
			name: "invalid syntax",
			el:   "x is equal",
		},
		{
			name: "missing operand",
			el:   "+ 5",
		},
	}

	c := NewCompiler()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.CompileCondition(tt.el)
			if err == nil {
				t.Errorf("CompileCondition(%q) expected error, got none", tt.el)
			}
		})
	}
}

func TestEmptyExpression(t *testing.T) {
	c := NewCompiler()

	result, err := c.Compile("")
	if err != nil {
		t.Fatalf("Compile empty string error: %v", err)
	}

	if result != "" {
		t.Errorf("Compile empty string got %q, want empty", result)
	}
}
