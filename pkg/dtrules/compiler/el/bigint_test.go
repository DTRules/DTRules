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

// TestBigIntSubtraction_NoCvi is the reproducer for issue #624: bigint subtraction
// assignment must not emit cvi (which would truncate the result to int64).
func TestBigIntSubtraction_NoCvi(t *testing.T) {
	syms := map[string]string{
		"bp.weekly_budget":         "bigint",
		"wp.effective_withholding": "bigint",
		"wp.staker_budget":         "bigint",
	}
	c := NewCompiler()
	c.SetSymbols(syms)
	postfix, err := c.CompileAction("set wp.staker_budget = bp.weekly_budget - wp.effective_withholding")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if strings.Contains(postfix, " cvi ") || strings.HasSuffix(postfix, " cvi") {
		t.Errorf("postfix contains spurious 'cvi': %s", postfix)
	}
}

// TestBigIntAddition_NoCvi mirrors the subtraction test for the + operator.
func TestBigIntAddition_NoCvi(t *testing.T) {
	syms := map[string]string{
		"bp.weekly_budget":         "bigint",
		"wp.effective_withholding": "bigint",
		"wp.staker_budget":         "bigint",
	}
	c := NewCompiler()
	c.SetSymbols(syms)
	postfix, err := c.CompileAction("set wp.staker_budget = bp.weekly_budget + wp.effective_withholding")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if strings.Contains(postfix, " cvi ") || strings.HasSuffix(postfix, " cvi") {
		t.Errorf("postfix contains spurious 'cvi': %s", postfix)
	}
}

// TestBigIntMultiplication_NoCvi is a regression guard: multiplication was already correct,
// confirm it stays correct.
func TestBigIntMultiplication_NoCvi(t *testing.T) {
	syms := map[string]string{
		"bp.weekly_budget":         "bigint",
		"wp.effective_withholding": "bigint",
		"wp.staker_budget":         "bigint",
	}
	c := NewCompiler()
	c.SetSymbols(syms)
	postfix, err := c.CompileAction("set wp.staker_budget = bp.weekly_budget * wp.effective_withholding")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if strings.Contains(postfix, " cvi ") || strings.HasSuffix(postfix, " cvi") {
		t.Errorf("postfix contains spurious 'cvi' for bigint multiplication: %s", postfix)
	}
}

// TestIntSubtraction_HasCvi verifies that assigning integer subtraction into a long field
// still emits cvi — guards against accidentally switching integer ops to bigint ops.
func TestIntSubtraction_HasCvi(t *testing.T) {
	syms := map[string]string{
		"a": "long",
		"b": "long",
		"c": "long",
	}
	c := NewCompiler()
	c.SetSymbols(syms)
	postfix, err := c.CompileAction("set c = a - b")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if !strings.Contains(postfix, "cvi") {
		t.Errorf("postfix missing expected 'cvi' for long-target assignment: %s", postfix)
	}
}
