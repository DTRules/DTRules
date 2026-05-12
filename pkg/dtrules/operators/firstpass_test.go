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

package operators_test

import (
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/operators"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// callFirstPass executes the registered `firstpass` op and returns the
// boolean it pushed. Going through the registry mirrors how the
// runtime actually invokes the op — opFirstPass is unexported, so we
// can't test it directly from this external package.
func callFirstPass(t *testing.T, state *interpreter.DTState) bool {
	t.Helper()
	op, ok := operators.Get(dtrules.GetRName("firstpass"))
	if !ok {
		t.Fatalf("firstpass operator not registered")
	}
	if err := op.Execute(state); err != nil {
		t.Fatalf("firstpass.Execute: %v", err)
	}
	got, err := state.DataPop()
	if err != nil {
		t.Fatalf("DataPop: %v", err)
	}
	if got.Type() != dtrules.TypeBoolean {
		t.Fatalf("expected boolean, got %v", got.Type())
	}
	b, _ := got.BooleanValue()
	return b
}

// TestFirstPass_NoActiveLoop: the predicate is meaningless outside a
// loop, so the safe answer is false. Anything else risks
// `first pass` firing on every invocation of a no-context table.
func TestFirstPass_NoActiveLoop(t *testing.T) {
	rs := session.NewRuleSet("test")
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	state := sess.GetState().(*interpreter.DTState)
	if got := callFirstPass(t, state); got {
		t.Errorf("no loop: want false, got true")
	}
}

// TestFirstPass_LoopStack walks the iteration tracking semantics the
// runtime contract is built on. The operators (for, forr, forall,
// forallr) push/bump/pop on this stack; opFirstPass is a pure read.
func TestFirstPass_LoopStack(t *testing.T) {
	rs := session.NewRuleSet("test")
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	state := sess.GetState().(*interpreter.DTState)

	if callFirstPass(t, state) {
		t.Errorf("outer scope: want false")
	}

	state.PushLoopFrame()
	if !callFirstPass(t, state) {
		t.Errorf("first iteration: want true")
	}

	state.BumpLoopIteration()
	if callFirstPass(t, state) {
		t.Errorf("second iteration: want false")
	}
	state.BumpLoopIteration()
	if callFirstPass(t, state) {
		t.Errorf("third iteration: want false")
	}

	state.PopLoopFrame()
	if callFirstPass(t, state) {
		t.Errorf("post-pop scope: want false")
	}
}

// TestFirstPass_NestedLoops: with two loops nested, the predicate
// tracks the INNERMOST one. Once the inner pops, the outer loop's own
// iteration state is what `first pass` reads.
func TestFirstPass_NestedLoops(t *testing.T) {
	rs := session.NewRuleSet("test")
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	state := sess.GetState().(*interpreter.DTState)

	state.PushLoopFrame() // outer
	if !callFirstPass(t, state) {
		t.Fatalf("outer.0: want true")
	}

	state.PushLoopFrame() // inner
	if !callFirstPass(t, state) {
		t.Errorf("inner.0 (under outer.0): want true")
	}
	state.BumpLoopIteration()
	if callFirstPass(t, state) {
		t.Errorf("inner.1: want false")
	}
	state.PopLoopFrame() // inner closes

	state.BumpLoopIteration() // outer → iteration 1
	if callFirstPass(t, state) {
		t.Errorf("outer.1 after inner pop: want false")
	}

	state.PushLoopFrame() // fresh inner under outer.1
	if !callFirstPass(t, state) {
		t.Errorf("fresh inner.0 under outer.1: want true")
	}
	state.PopLoopFrame()

	state.PopLoopFrame() // outer closes
	if callFirstPass(t, state) {
		t.Errorf("post-pop: want false")
	}
}
