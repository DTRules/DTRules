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

package excel

import "testing"

// A table's contexts declare its locals. Initial actions were compiled first,
// so a compiler had not yet seen `local long clientAge` when it reached
// `set clientAge = 45` -- and with no local of that name in scope it emitted
// `/clientAge xdef`, defining a name in the entity dictionary, where the
// committed postfix says `45 cvi 2 local!`. The rebuilt table then failed at
// run time with `xdef: clientAge is undefined` (#1074).
//
// Contexts also run before initial actions, so compiling them first is what
// execution order asked for all along.

// orderRecordingCompiler notes the order in which sections were handed to it.
type orderRecordingCompiler struct{ seen []string }

func (c *orderRecordingCompiler) SetSymbols(map[string]string) {}

func (c *orderRecordingCompiler) CompileContext(el string) (string, error) {
	c.seen = append(c.seen, "context:"+el)
	return "ctx", nil
}

func (c *orderRecordingCompiler) CompileAction(el string) (string, error) {
	c.seen = append(c.seen, "action:"+el)
	return "act", nil
}

func (c *orderRecordingCompiler) CompileCondition(el string) (string, error) {
	c.seen = append(c.seen, "condition:"+el)
	return "cond", nil
}

func TestContextsCompileBeforeInitialActions(t *testing.T) {
	table := &DecisionTableXML{TableName: "Run_Test_15"}
	table.Contexts.Details = []ContextDetailXML{{Number: 1, DSL: "local long clientAge"}}
	table.InitialActions = []InitialActionXML{{DSL: "set clientAge = 45"}}
	table.Conditions = []ConditionXML{{DSL: "clientAge > 18"}}
	table.Actions = []ActionXML{{DSL: "set result.ok = true"}}

	rec := &orderRecordingCompiler{}
	imp := NewDTImporter()
	imp.SetELCompiler(rec)
	imp.SetSymbols(map[string]string{"result.ok": "boolean"})

	if err := imp.compileTableEL(table); err != nil {
		t.Fatalf("compileTableEL: %v", err)
	}

	if len(rec.seen) < 2 {
		t.Fatalf("expected every section compiled, got %v", rec.seen)
	}
	if rec.seen[0] != "context:local long clientAge" {
		t.Errorf("contexts must be compiled first, so a table's locals are in "+
			"scope for everything that references them; order was %v", rec.seen)
	}
	if rec.seen[1] != "action:set clientAge = 45" {
		t.Errorf("initial actions should follow the contexts; order was %v", rec.seen)
	}
}

// Conditions and actions still come after both, so the whole order matches the
// order the runtime executes them in.
func TestCompileOrderMatchesExecutionOrder(t *testing.T) {
	table := &DecisionTableXML{TableName: "T"}
	table.Contexts.Details = []ContextDetailXML{{Number: 1, DSL: "C"}}
	table.InitialActions = []InitialActionXML{{DSL: "I"}}
	table.Conditions = []ConditionXML{{DSL: "Q"}}
	table.Actions = []ActionXML{{DSL: "A"}}

	rec := &orderRecordingCompiler{}
	imp := NewDTImporter()
	imp.SetELCompiler(rec)
	imp.SetSymbols(map[string]string{"x": "integer"})

	if err := imp.compileTableEL(table); err != nil {
		t.Fatalf("compileTableEL: %v", err)
	}

	want := []string{"context:C", "action:I", "condition:Q", "action:A"}
	if len(rec.seen) != len(want) {
		t.Fatalf("compiled %v, want %v", rec.seen, want)
	}
	for i := range want {
		if rec.seen[i] != want[i] {
			t.Errorf("step %d = %q, want %q (full order %v)", i, rec.seen[i], want[i], rec.seen)
		}
	}
}
