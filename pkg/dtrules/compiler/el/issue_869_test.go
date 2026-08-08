// Copyright 2026 Paul Snow
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
	"testing"
)

// Issue #869: the grammar's entity alternatives (boolThereIsInEntityWhere /
// boolThereIsNoInEntityWhere) shadow the array alternatives entirely —
// typedEntity and typedArray both come from IDENT — so `there is x in <array>
// where p` compiled to `<arr> entitypush …`, which crashes at runtime when
// entitypush calls REntityValue on the array. The emitter now routes by the
// operand's declared type. Execution coverage lives in
// pkg/dtrules/issue_869_exec_test.go; these pin the compile shapes.

func TestIssue869_ArrayOperandRoutesToForallFold(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{"kids": "array", "tax": "integer", "house": "entity"})

	got, err := c.CompileCondition("there is k in kids where tax > 5")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	want := "false { tax 5 > or } kids forall"
	if got != want {
		t.Errorf("array operand shape\n got: %q\nwant: %q", got, want)
	}

	got, err = c.CompileCondition("there is no k in kids where tax > 5")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	want = "false { tax 5 > or } kids forall not"
	if got != want {
		t.Errorf("negated array operand shape\n got: %q\nwant: %q", got, want)
	}
}

func TestIssue869_EntityOperandKeepsEntityScope(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{"kids": "array", "tax": "integer", "house": "entity"})

	got, err := c.CompileCondition("there is k in house where tax > 5")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	want := "house entitypush tax 5 > entitypop swap pop"
	if got != want {
		t.Errorf("entity operand shape\n got: %q\nwant: %q", got, want)
	}
}
