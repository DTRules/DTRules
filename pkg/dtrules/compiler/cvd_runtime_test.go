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

package compiler

import (
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"

	// Pull in operator registrations.
	_ "github.com/DTRules/DTRules/pkg/dtrules/operators"
)

// Issue #694: before the rename, the EL emitter's typeConverter emitted
// `"cvd"` for TypeDouble but the registered `cvd` op was the Date
// converter. Every `set <double_field> = <expr>` rule silently stored
// null. These runtime integration tests would have caught the bug —
// the earlier compile-level emission tests wouldn't, because they
// never executed the compiled code and read back the stored value.
//
// Each test below compiles a one-liner that pushes a literal and
// applies the target-type conversion via the `cv*` op family, then
// reads the resulting value off the stack. One test per numeric
// target type (int / bigint / double / fixed), plus a smoke test
// for the renamed `cvdate` op.

func execToTop(t *testing.T, src string) dtrules.Object {
	t.Helper()
	c := newTestCompiler()
	code, err := c.Compile(src)
	if err != nil {
		t.Fatalf("compile %q: %v", src, err)
	}
	state := interpreter.NewDTState(&mockSession{})
	if err := code.Execute(state); err != nil {
		t.Fatalf("execute %q: %v", src, err)
	}
	top, err := state.DataPop()
	if err != nil {
		t.Fatalf("DataPop %q: %v", src, err)
	}
	return top
}

// TestCvd_StoresDouble is the #694 reproducer: before the rename, this
// test would have stored null on the stack (cvd was the date converter
// and fell through to GetRDate which fails to parse "3.14").
func TestCvd_StoresDouble(t *testing.T) {
	top := execToTop(t, "3.14 cvd")
	if top.Type() != dtrules.TypeDouble {
		t.Fatalf("cvd should produce a double, got %s (value=%q)",
			top.Type().String(), top.StringValue())
	}
	v, err := top.DoubleValue()
	if err != nil {
		t.Fatalf("DoubleValue: %v", err)
	}
	if v != 3.14 {
		t.Errorf("cvd(3.14) = %v, want 3.14", v)
	}
}

// TestCvi_StoresInt guards the integer path — regression guard, should
// have worked before and after the rename.
func TestCvi_StoresInt(t *testing.T) {
	top := execToTop(t, "42 cvi")
	if top.Type() != dtrules.TypeInteger {
		t.Fatalf("cvi should produce integer, got %s", top.Type().String())
	}
	v, err := top.IntValue()
	if err != nil {
		t.Fatalf("IntValue: %v", err)
	}
	if v != 42 {
		t.Errorf("cvi(42) = %d, want 42", v)
	}
}

// TestCvfp_StoresFixed guards the fixed path.
func TestCvfp_StoresFixed(t *testing.T) {
	top := execToTop(t, "1.5fp cvfp")
	if top.Type() != dtrules.TypeFixed {
		t.Fatalf("cvfp should produce fixed, got %s", top.Type().String())
	}
	if got := top.StringValue(); got != "1.50000000" {
		t.Errorf("cvfp(1.5fp) = %q, want 1.50000000", got)
	}
}

// TestCvbi_StoresBigInt guards the bigint path.
func TestCvbi_StoresBigInt(t *testing.T) {
	top := execToTop(t, "42 cvbi")
	if top.Type() != dtrules.TypeBigInt {
		t.Fatalf("cvbi should produce bigint, got %s", top.Type().String())
	}
	if got := top.StringValue(); got != "42" {
		t.Errorf("cvbi(42) = %q, want 42", got)
	}
}

// TestCvrLegacyAlias ensures the old `cvr` name still works as an
// alias for `cvd`. Stored postfix in external rule projects may use
// `cvr` (the previous name for the double converter); during the
// migration those rules should continue to execute correctly.
func TestCvrLegacyAlias(t *testing.T) {
	top := execToTop(t, "3.14 cvr")
	if top.Type() != dtrules.TypeDouble {
		t.Fatalf("cvr (legacy alias) should still produce a double, got %s",
			top.Type().String())
	}
	v, _ := top.DoubleValue()
	if v != 3.14 {
		t.Errorf("cvr(3.14) = %v, want 3.14", v)
	}
}

// TestCvdate_Registered ensures the renamed `cvdate` op is findable
// via the operator registry. End-to-end execution with a non-date
// input would need a real DateParser on the session (the mock
// session returns nil, and the op panics on that path — pre-existing
// behavior, not caused by the rename). A simple registry-lookup
// test guards the rename without triggering the unrelated nil-parser
// panic.
func TestCvdate_Registered(t *testing.T) {
	// Compile succeeds iff cvdate resolves. The old name `cvd` for the
	// date converter no longer exists (reassigned to double); only
	// `cvdate` should work.
	c := newTestCompiler()
	_, err := c.Compile("cvdate")
	if err != nil {
		t.Fatalf("cvdate should be registered: %v", err)
	}
}

// Guard against the unused-import lint after removing the non-date test.
var _ = strings.Contains
