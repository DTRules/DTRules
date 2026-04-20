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
			expected: "/Calculate_Tax performtable",
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

// =============================================================================
// BigInt Expression Tests
// =============================================================================

// TestCompileBigIntKeywordRecognized tests that the BIGINT keyword is recognized
func TestCompileBigIntKeywordRecognized(t *testing.T) {
	c := NewCompiler()

	// Test that cvbi (convert to bigint) is generated for type casts
	tests := []struct {
		name     string
		el       string
		contains string
	}{
		{
			name:     "bigint type cast with cvbi",
			el:       `set x = (bigint) "12345";`,
			contains: "cvbi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := c.CompileAction(tt.el)
			if err != nil {
				t.Fatalf("CompileAction(%q) error: %v", tt.el, err)
			}

			if !strings.Contains(result, tt.contains) {
				t.Errorf("CompileAction(%q) = %q, expected to contain %q", tt.el, result, tt.contains)
			}
		})
	}
}

// TestCompileBigIntTypeConstant tests that the TypeBigInt constant is defined
func TestCompileBigIntTypeConstant(t *testing.T) {
	if TypeBigInt != "bigint" {
		t.Errorf("TypeBigInt = %q, want \"bigint\"", TypeBigInt)
	}
}

// =============================================================================
// BigInt Arithmetic Operators Tests
// =============================================================================

// TestBigIntArithmeticOperators tests that bigint arithmetic compiles to b+, b-, b*, b/ operators.
// Note: The grammar is syntax-based, so bigint operators are only used when the expression
// is explicitly parsed as a bigexpr (e.g., via explicit cast or typed field access).
// Local variables declared as bigint are stored as bigint, but when used in generic
// expressions like "x + y", the parser treats them as integer expressions.
func TestBigIntArithmeticOperators(t *testing.T) {
	tests := []struct {
		name     string
		action   string
		expected string
	}{
		{
			name:     "bigint addition with explicit casts",
			action:   "set result = (bigint) 10 + (bigint) 20;",
			expected: "10 cvbi 20 cvbi b+ cvbi /result xdef",
		},
		{
			name:     "bigint subtraction with explicit casts",
			action:   "set result = (bigint) 30 - (bigint) 10;",
			expected: "30 cvbi 10 cvbi b- cvbi /result xdef",
		},
		{
			name:     "bigint multiplication with explicit casts",
			action:   "set result = (bigint) 5 * (bigint) 4;",
			expected: "5 cvbi 4 cvbi b* cvbi /result xdef",
		},
		{
			name:     "bigint division with explicit casts",
			action:   "set result = (bigint) 20 / (bigint) 5;",
			expected: "20 cvbi 5 cvbi b/ cvbi /result xdef",
		},
		{
			name:     "bigint complex expression",
			action:   "set result = (bigint) 10 + (bigint) 5 * (bigint) 2;",
			expected: "10 cvbi 5 cvbi 2 cvbi b* b+ cvbi /result xdef",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompiler()

			result, err := c.CompileAction(tt.action)
			if err != nil {
				t.Fatalf("CompileAction(%q) error: %v", tt.action, err)
			}

			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)

			if result != expected {
				t.Errorf("CompileAction(%q)\n  got:  %q\n  want: %q", tt.action, result, expected)
			}
		})
	}
}

// TestBigIntArithmeticWithLocalVariables tests arithmetic operations on local bigint variables.
// The emitter is now type-aware and emits bigint operators (b+, b-, b*, b/) when
// operands are known to be bigint type, with automatic type promotion for mixed expressions.
func TestBigIntArithmeticWithLocalVariables(t *testing.T) {
	tests := []struct {
		name     string
		context  string
		action   string
		expected string
	}{
		{
			// The emitter detects that x is bigint and emits b+ operator
			name:     "local bigint variables use bigint operators",
			context:  "local bigint x = (bigint) 10;",
			action:   "set result = x + x;",
			expected: "0 local@ 0 local@ b+ cvi /result xdef",
		},
		{
			name:     "local bigint variables multiplication",
			context:  "local bigint x = (bigint) 10;",
			action:   "set result = x * x;",
			expected: "0 local@ 0 local@ b* cvi /result xdef",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompiler()

			// First compile the context to declare local variables
			_, err := c.CompileContext(tt.context)
			if err != nil {
				t.Fatalf("CompileContext(%q) error: %v", tt.context, err)
			}

			// Then compile the action
			result, err := c.CompileAction(tt.action)
			if err != nil {
				t.Fatalf("CompileAction(%q) error: %v", tt.action, err)
			}

			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)

			if result != expected {
				t.Errorf("CompileAction(%q)\n  got:  %q\n  want: %q", tt.action, result, expected)
			}
		})
	}
}

// =============================================================================
// BigInt Comparison Operators Tests
// =============================================================================

// TestBigIntComparisonOperators tests that bigint comparisons compile to b>, b<, b==, b>=, b<= operators.
// Note: Like arithmetic, bigint comparison operators are only used when expressions are explicitly
// parsed as bigexpr (via explicit casts). Local variables are parsed as generic expressions.
func TestBigIntComparisonOperators(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		expected  string
	}{
		{
			name:      "bigint equality with explicit casts",
			condition: "(bigint) 10 == (bigint) 10",
			expected:  "10 cvbi 10 cvbi b==",
		},
		{
			name:      "bigint not equal with explicit casts",
			condition: "(bigint) 10 != (bigint) 20",
			expected:  "10 cvbi 20 cvbi b== not",
		},
		{
			name:      "bigint greater than with explicit casts",
			condition: "(bigint) 20 > (bigint) 10",
			expected:  "20 cvbi 10 cvbi b>",
		},
		{
			name:      "bigint greater than or equal with explicit casts",
			condition: "(bigint) 10 >= (bigint) 10",
			expected:  "10 cvbi 10 cvbi b>=",
		},
		{
			name:      "bigint less than with explicit casts",
			condition: "(bigint) 5 < (bigint) 10",
			expected:  "5 cvbi 10 cvbi b<",
		},
		{
			name:      "bigint less than or equal with explicit casts",
			condition: "(bigint) 5 <= (bigint) 5",
			expected:  "5 cvbi 5 cvbi b<=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompiler()

			result, err := c.CompileCondition(tt.condition)
			if err != nil {
				t.Fatalf("CompileCondition(%q) error: %v", tt.condition, err)
			}

			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)

			if result != expected {
				t.Errorf("CompileCondition(%q)\n  got:  %q\n  want: %q", tt.condition, result, expected)
			}
		})
	}
}

// TestBigIntComparisonWithLocalVariables tests comparison operations on local bigint variables.
// The emitter is now type-aware across the whole comparison family — BoolName{Eq,Neq}
// dispatch to numeric compares when both operands resolve to a numeric type, so
// bigint equality now uses b== rather than the stringwise streq fallback.
func TestBigIntComparisonWithLocalVariables(t *testing.T) {
	tests := []struct {
		name      string
		context   string
		condition string
		expected  string
	}{
		{
			// BoolNameEq now detects the local-declared bigint type and emits
			// a bigint compare instead of the old streq fallback.
			name:      "local bigint equality uses bigint b==",
			context:   "local bigint x = (bigint) 10;",
			condition: "x == x",
			expected:  "0 local@ 0 local@ b==",
		},
		{
			// The emitter detects that x is bigint and emits b> operator
			name:      "local bigint greater than uses bigint b>",
			context:   "local bigint x = (bigint) 10;",
			condition: "x > x",
			expected:  "0 local@ 0 local@ b>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompiler()

			// First compile the context to declare local variables
			_, err := c.CompileContext(tt.context)
			if err != nil {
				t.Fatalf("CompileContext(%q) error: %v", tt.context, err)
			}

			// Then compile the condition
			result, err := c.CompileCondition(tt.condition)
			if err != nil {
				t.Fatalf("CompileCondition(%q) error: %v", tt.condition, err)
			}

			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)

			if result != expected {
				t.Errorf("CompileCondition(%q)\n  got:  %q\n  want: %q", tt.condition, result, expected)
			}
		})
	}
}

// =============================================================================
// BigInt Type Conversion Tests
// =============================================================================

// TestBigIntTypeConversions tests that (bigint) cast syntax emits cvbi
func TestBigIntTypeConversions(t *testing.T) {
	tests := []struct {
		name     string
		action   string
		contains string
	}{
		{
			name:     "bigint cast from string literal",
			action:   `set x = (bigint) "12345";`,
			contains: "cvbi",
		},
		{
			name:     "bigint cast from integer",
			action:   `set x = (bigint) 100;`,
			contains: "cvbi",
		},
		{
			name:     "bigint cast from float",
			action:   `set x = (bigint) 3.14;`,
			contains: "cvbi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompiler()

			result, err := c.CompileAction(tt.action)
			if err != nil {
				t.Fatalf("CompileAction(%q) error: %v", tt.action, err)
			}

			if !strings.Contains(result, tt.contains) {
				t.Errorf("CompileAction(%q) = %q, expected to contain %q", tt.action, result, tt.contains)
			}
		})
	}
}

// TestBigIntTypeConversionsExact tests exact postfix output for type conversions
func TestBigIntTypeConversionsExact(t *testing.T) {
	tests := []struct {
		name     string
		action   string
		expected string
	}{
		{
			name:     "bigint cast from string literal exact",
			action:   `set x = (bigint) "12345";`,
			expected: `"12345" cvbi cvbi /x xdef`,
		},
		{
			name:     "bigint cast from integer exact",
			action:   `set x = (bigint) 100;`,
			expected: `100 cvbi cvbi /x xdef`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompiler()

			result, err := c.CompileAction(tt.action)
			if err != nil {
				t.Fatalf("CompileAction(%q) error: %v", tt.action, err)
			}

			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)

			if result != expected {
				t.Errorf("CompileAction(%q)\n  got:  %q\n  want: %q", tt.action, result, expected)
			}
		})
	}
}

// =============================================================================
// BigInt Local Variable Declaration Tests
// =============================================================================

// TestBigIntLocalVariableDeclarations tests local bigint variable declarations
func TestBigIntLocalVariableDeclarations(t *testing.T) {
	tests := []struct {
		name     string
		context  string
		expected string
	}{
		{
			name:     "local bigint uninitialized",
			context:  "local bigint x;",
			expected: "null allocate execute deallocate pop",
		},
		{
			name:     "local bigint with integer initialization",
			context:  "local bigint x = (bigint) 100;",
			expected: "100 cvbi cvbi allocate execute deallocate pop",
		},
		{
			name:     "local bigint with string initialization",
			context:  `local bigint x = (bigint) "999";`,
			expected: `"999" cvbi cvbi allocate execute deallocate pop`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompiler()

			result, err := c.CompileContext(tt.context)
			if err != nil {
				t.Fatalf("CompileContext(%q) error: %v", tt.context, err)
			}

			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)

			if result != expected {
				t.Errorf("CompileContext(%q)\n  got:  %q\n  want: %q", tt.context, result, expected)
			}
		})
	}
}

// TestBigIntLocalVariableReference tests that local bigint variables are referenced correctly
func TestBigIntLocalVariableReference(t *testing.T) {
	c := NewCompiler()

	// Declare a local bigint variable
	_, err := c.CompileContext("local bigint x = (bigint) 100;")
	if err != nil {
		t.Fatalf("CompileContext error: %v", err)
	}

	// Now reference it - should use "0 local@" pattern
	result, err := c.CompileAction("set result = x;")
	if err != nil {
		t.Fatalf("CompileAction error: %v", err)
	}

	// Check that local variable reference is used
	if !strings.Contains(result, "0 local@") {
		t.Errorf("CompileAction('set result = x') = %q, expected to contain '0 local@'", result)
	}
}

// =============================================================================
// BigInt Set Statement Tests
// =============================================================================

// TestBigIntSetStatements tests set statements with bigint values using explicit casts
func TestBigIntSetStatements(t *testing.T) {
	tests := []struct {
		name     string
		action   string
		contains []string
	}{
		{
			name:     "set bigint from addition with casts",
			action:   "set result = (bigint) 10 + (bigint) 20;",
			contains: []string{"b+", "cvbi", "xdef"},
		},
		{
			name:     "set bigint from multiplication with casts",
			action:   "set result = (bigint) 5 * (bigint) 4;",
			contains: []string{"b*", "cvbi", "xdef"},
		},
		{
			name:     "set bigint from subtraction with casts",
			action:   "set result = (bigint) 100 - (bigint) 50;",
			contains: []string{"b-", "cvbi", "xdef"},
		},
		{
			name:     "set bigint from division with casts",
			action:   "set result = (bigint) 100 / (bigint) 10;",
			contains: []string{"b/", "cvbi", "xdef"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompiler()

			result, err := c.CompileAction(tt.action)
			if err != nil {
				t.Fatalf("CompileAction(%q) error: %v", tt.action, err)
			}

			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("CompileAction(%q) = %q, expected to contain %q", tt.action, result, want)
				}
			}
		})
	}
}

// =============================================================================
// BigInt Negation Test
// =============================================================================

// TestBigIntNegation tests that bigint negation compiles to bnegate when using explicit cast
func TestBigIntNegation(t *testing.T) {
	c := NewCompiler()

	// Negate a bigint with explicit cast
	result, err := c.CompileAction("set result = -(bigint) 100;")
	if err != nil {
		t.Fatalf("CompileAction error: %v", err)
	}

	if !strings.Contains(result, "bnegate") {
		t.Errorf("CompileAction('set result = -(bigint) 100') = %q, expected to contain 'bnegate'", result)
	}
}

// TestBigIntNegationLocalVariable tests negation of local bigint variables.
// Due to grammar limitations, negation of a local variable uses integer negation.
func TestBigIntNegationLocalVariable(t *testing.T) {
	c := NewCompiler()

	// Declare a local bigint variable
	_, err := c.CompileContext("local bigint x = (bigint) 100;")
	if err != nil {
		t.Fatalf("CompileContext error: %v", err)
	}

	// Negate the bigint - uses integer negation due to grammar
	result, err := c.CompileAction("set result = -x;")
	if err != nil {
		t.Fatalf("CompileAction error: %v", err)
	}

	// Uses 'neg' (integer negation) not 'bnegate' due to grammar limitations
	if !strings.Contains(result, "neg") {
		t.Errorf("CompileAction('set result = -x') = %q, expected to contain 'neg'", result)
	}
}

// =============================================================================
// BigInt Absolute Value Test
// =============================================================================

// TestBigIntAbsoluteValue tests that bigint absolute value compiles to babs
func TestBigIntAbsoluteValue(t *testing.T) {
	c := NewCompiler()

	// Get absolute value of an explicit bigint expression
	result, err := c.CompileAction("set result = absolute value of (bigint) -100;")
	if err != nil {
		t.Fatalf("CompileAction error: %v", err)
	}

	if !strings.Contains(result, "babs") {
		t.Errorf("CompileAction('set result = absolute value of (bigint) -100') = %q, expected to contain 'babs'", result)
	}
}

// =============================================================================
// BigInt Multiple Local Variables Test
// =============================================================================

// TestBigIntMultipleLocalVariables tests that multiple bigint local variables get correct indices.
// The emitter is type-aware and emits bigint operators when operands are known to be bigint.
func TestBigIntMultipleLocalVariables(t *testing.T) {
	c := NewCompiler()

	// Declare first local bigint variable (index 0)
	_, err := c.CompileContext("local bigint x = (bigint) 10;")
	if err != nil {
		t.Fatalf("CompileContext(x) error: %v", err)
	}

	// Declare second local bigint variable (index 1)
	_, err = c.CompileContext("local bigint y = (bigint) 20;")
	if err != nil {
		t.Fatalf("CompileContext(y) error: %v", err)
	}

	// Use both variables - x should be "0 local@", y should be "1 local@"
	result, err := c.CompileAction("set result = x + y;")
	if err != nil {
		t.Fatalf("CompileAction error: %v", err)
	}

	if !strings.Contains(result, "0 local@") {
		t.Errorf("CompileAction('set result = x + y') = %q, expected to contain '0 local@' for x", result)
	}

	if !strings.Contains(result, "1 local@") {
		t.Errorf("CompileAction('set result = x + y') = %q, expected to contain '1 local@' for y", result)
	}

	// Uses bigint 'b+' operator because emitter detects x and y are bigint
	expected := "0 local@ 1 local@ b+ cvi /result xdef"
	result = strings.TrimSpace(result)
	if result != expected {
		t.Errorf("CompileAction('set result = x + y')\n  got:  %q\n  want: %q", result, expected)
	}
}

// =============================================================================
// BigInt Parenthesized Expression Test
// =============================================================================

// TestBigIntParenthesizedExpression tests that parentheses work correctly with explicit bigint casts
func TestBigIntParenthesizedExpression(t *testing.T) {
	c := NewCompiler()

	// Test ((bigint)10 + (bigint)20) * (bigint)5
	result, err := c.CompileAction("set result = ((bigint) 10 + (bigint) 20) * (bigint) 5;")
	if err != nil {
		t.Fatalf("CompileAction error: %v", err)
	}

	// Expected: first add 10 + 20, then multiply by 5
	expected := "10 cvbi 20 cvbi b+ 5 cvbi b* cvbi /result xdef"
	result = strings.TrimSpace(result)
	if result != expected {
		t.Errorf("CompileAction('set result = ((bigint) 10 + (bigint) 20) * (bigint) 5')\n  got:  %q\n  want: %q", result, expected)
	}
}

// TestBigIntParenthesizedWithLocalVariables tests parentheses with local variables.
// The emitter is type-aware and emits bigint operators when operands are known to be bigint.
func TestBigIntParenthesizedWithLocalVariables(t *testing.T) {
	c := NewCompiler()

	// Declare local variables
	_, err := c.CompileContext("local bigint x = (bigint) 10;")
	if err != nil {
		t.Fatalf("CompileContext(x) error: %v", err)
	}

	_, err = c.CompileContext("local bigint y = (bigint) 20;")
	if err != nil {
		t.Fatalf("CompileContext(y) error: %v", err)
	}

	// Test (x + y) * x - emitter detects bigint types
	result, err := c.CompileAction("set result = (x + y) * x;")
	if err != nil {
		t.Fatalf("CompileAction error: %v", err)
	}

	// Uses bigint operators because emitter detects x and y are bigint
	expected := "0 local@ 1 local@ b+ 0 local@ b* cvi /result xdef"
	result = strings.TrimSpace(result)
	if result != expected {
		t.Errorf("CompileAction('set result = (x + y) * x')\n  got:  %q\n  want: %q", result, expected)
	}
}

// TestBigIntMixedTypePromotion tests that mixing int and bigint auto-promotes int to bigint.
func TestBigIntMixedTypePromotion(t *testing.T) {
	tests := []struct {
		name     string
		contexts []string
		action   string
		expected string
	}{
		{
			// When mixing int and bigint, the int should be promoted to bigint
			name:     "int + bigint promotes int to bigint",
			contexts: []string{"local int x = 10;", "local bigint y = (bigint) 20;"},
			action:   "set result = x + y;",
			expected: "0 local@ cvbi 1 local@ b+ cvi /result xdef",
		},
		{
			name:     "bigint + int promotes int to bigint",
			contexts: []string{"local bigint x = (bigint) 10;", "local int y = 20;"},
			action:   "set result = x + y;",
			expected: "0 local@ 1 local@ cvbi b+ cvi /result xdef",
		},
		{
			name:     "int * bigint promotes int to bigint",
			contexts: []string{"local int x = 10;", "local bigint y = (bigint) 20;"},
			action:   "set result = x * y;",
			expected: "0 local@ cvbi 1 local@ b* cvi /result xdef",
		},
		{
			name:     "int - bigint promotes int to bigint",
			contexts: []string{"local int x = 10;", "local bigint y = (bigint) 20;"},
			action:   "set result = x - y;",
			expected: "0 local@ cvbi 1 local@ b- cvi /result xdef",
		},
		{
			name:     "int / bigint promotes int to bigint",
			contexts: []string{"local int x = 10;", "local bigint y = (bigint) 20;"},
			action:   "set result = x / y;",
			expected: "0 local@ cvbi 1 local@ b/ cvi /result xdef",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCompiler()

			// Compile all contexts
			for _, ctx := range tt.contexts {
				_, err := c.CompileContext(ctx)
				if err != nil {
					t.Fatalf("CompileContext(%q) error: %v", ctx, err)
				}
			}

			// Compile the action
			result, err := c.CompileAction(tt.action)
			if err != nil {
				t.Fatalf("CompileAction(%q) error: %v", tt.action, err)
			}

			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)

			if result != expected {
				t.Errorf("CompileAction(%q)\n  got:  %q\n  want: %q", tt.action, result, expected)
			}
		})
	}
}

// TestCompile_RejectsTrailingTokens pins that extra tokens after a successful
// parse surface as a compile error instead of being silently dropped.
func TestCompile_RejectsTrailingTokens(t *testing.T) {
	c := NewCompiler()
	out, err := c.CompileContext("for all income extra junk")
	if err == nil {
		t.Fatalf("expected error on trailing tokens, got postfix %q", out)
	}
	if !strings.Contains(err.Error(), "unexpected tokens") {
		t.Errorf("expected 'unexpected tokens' in error, got: %v", err)
	}
}
