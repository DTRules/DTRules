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

package operators

import (
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// Behavior tests for the control-stack and frame ops:
// allocate, deallocate, local!, local@. These are how the compiler
// spills intermediate values and local variables for compiled EL
// expressions — a regression here would silently corrupt any
// decision-table expression that uses a local.
//
// Also covers xdef (same signature as def — assigns value to
// attribute on the entity stack).

// TestAllocateDeallocateRoundTrip — allocate moves top-of-data to
// the control stack, deallocate brings it back. After the pair the
// data stack must be unchanged.
func TestAllocateDeallocateRoundTrip(t *testing.T) {
	state := newTestState()
	state.DataPush(dtrules.GetRIntegerValue(777))

	alloc, _ := Get(dtrules.GetRName("allocate"))
	if err := alloc.Execute(state); err != nil {
		t.Fatalf("allocate: %v", err)
	}
	if state.DataStackDepth() != 0 {
		t.Errorf("allocate should clear data stack; depth=%d",
			state.DataStackDepth())
	}

	dealloc, _ := Get(dtrules.GetRName("deallocate"))
	if err := dealloc.Execute(state); err != nil {
		t.Fatalf("deallocate: %v", err)
	}
	if state.DataStackDepth() != 1 {
		t.Fatalf("deallocate should restore value; depth=%d",
			state.DataStackDepth())
	}
	top, _ := state.DataPop()
	v, _ := top.LongValue()
	if v != 777 {
		t.Errorf("allocate/deallocate round-trip: got %d, want 777", v)
	}
}

// TestLocalStoreFetch — local! stores a value at a frame-relative
// index; local@ reads it back. Uses PushFrame to set up a frame
// with pre-allocated slots (via allocate) so there's something to
// write into.
func TestLocalStoreFetch(t *testing.T) {
	state := newTestState()

	// Set up two frame slots by allocating placeholder values.
	if err := state.PushFrame(); err != nil {
		t.Fatalf("PushFrame: %v", err)
	}
	alloc, _ := Get(dtrules.GetRName("allocate"))
	state.DataPush(dtrules.GetRIntegerValue(0))
	alloc.Execute(state)
	state.DataPush(dtrules.GetRIntegerValue(0))
	alloc.Execute(state)

	// Store 42 into slot 0.
	// local!: ( value index -- )
	state.DataPush(dtrules.GetRIntegerValue(42))
	state.DataPush(dtrules.GetRIntegerValue(0))
	storeOp, _ := Get(dtrules.GetRName("local!"))
	if err := storeOp.Execute(state); err != nil {
		t.Fatalf("local!: %v", err)
	}
	// Store 99 into slot 1.
	state.DataPush(dtrules.GetRIntegerValue(99))
	state.DataPush(dtrules.GetRIntegerValue(1))
	if err := storeOp.Execute(state); err != nil {
		t.Fatalf("local! slot 1: %v", err)
	}

	// Fetch and verify slot 0.
	// local@: ( index -- value )
	state.DataPush(dtrules.GetRIntegerValue(0))
	fetchOp, _ := Get(dtrules.GetRName("local@"))
	if err := fetchOp.Execute(state); err != nil {
		t.Fatalf("local@: %v", err)
	}
	top, _ := state.DataPop()
	v, _ := top.LongValue()
	if v != 42 {
		t.Errorf("slot 0: got %d, want 42", v)
	}

	// Fetch and verify slot 1.
	state.DataPush(dtrules.GetRIntegerValue(1))
	if err := fetchOp.Execute(state); err != nil {
		t.Fatalf("local@ slot 1: %v", err)
	}
	top, _ = state.DataPop()
	v, _ = top.LongValue()
	if v != 99 {
		t.Errorf("slot 1: got %d, want 99", v)
	}
}

// TestXDefMatchesDef — xdef and def have the same ( value name -- )
// signature and assign to an attribute on the entity stack.
func TestXDefMatchesDef(t *testing.T) {
	state := newTestState()
	e := newTestEntity("Target", 1,
		map[string]dtrules.Object{"slot": dtrules.GetRIntegerValue(0)})
	state.EntityPush(e)

	state.DataPush(dtrules.GetRIntegerValue(123))
	state.DataPush(dtrules.GetRName("slot"))
	o, _ := Get(dtrules.GetRName("xdef"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("xdef: %v", err)
	}

	got, err := state.Find(dtrules.GetRName("slot"))
	if err != nil {
		t.Fatalf("Find after xdef: %v", err)
	}
	v, _ := got.LongValue()
	if v != 123 {
		t.Errorf("xdef set %d, want 123", v)
	}
}
