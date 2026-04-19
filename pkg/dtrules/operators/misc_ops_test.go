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

// Fills remaining gaps in per-op behavior coverage. These families
// previously had only registration tests:
//   - String comparison: s+, s<, s<=, s>, s>=
//   - Array: sortarray, copyelements, deepcopy
//   - Entity/name lookup: lookup, get
//   - Date: getdate, gettimestamp
//   - Error throwing: error, throwexception

// -- String comparison ops --------------------------------------------------

// TestStringConcat — s+ concatenates two strings.
func TestStringConcat(t *testing.T) {
	state := newTestState()
	state.DataPush(dtrules.NewRString("hello "))
	state.DataPush(dtrules.NewRString("world"))
	o, _ := Get(dtrules.GetRName("s+"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("s+: %v", err)
	}
	top, _ := state.DataPop()
	if top.StringValue() != "hello world" {
		t.Errorf("s+: got %q, want %q", top.StringValue(), "hello world")
	}
}

// TestStringOrdering — s<, s<=, s>, s>= pinned against ASCIIbetical
// ordering. Pins the operand order: bottom < top → true for s<.
func TestStringOrdering(t *testing.T) {
	cases := []struct {
		op      string
		a, b    string
		want    bool
	}{
		{"s<", "apple", "banana", true},
		{"s<", "banana", "apple", false},
		{"s<", "apple", "apple", false},
		{"s<=", "apple", "apple", true},
		{"s<=", "apple", "banana", true},
		{"s<=", "banana", "apple", false},
		{"s>", "banana", "apple", true},
		{"s>", "apple", "banana", false},
		{"s>=", "apple", "apple", true},
		{"s>=", "banana", "apple", true},
	}
	for _, c := range cases {
		t.Run(c.op+"_"+c.a+"_"+c.b, func(t *testing.T) {
			state := newTestState()
			state.DataPush(dtrules.NewRString(c.a))
			state.DataPush(dtrules.NewRString(c.b))
			o, _ := Get(dtrules.GetRName(c.op))
			if err := o.Execute(state); err != nil {
				t.Fatalf("%s: %v", c.op, err)
			}
			top, _ := state.DataPop()
			v, _ := top.BooleanValue()
			if v != c.want {
				t.Errorf("%s(%q, %q) = %v, want %v",
					c.op, c.a, c.b, v, c.want)
			}
		})
	}
}

// -- Array ops ---------------------------------------------------------------

// mustArray builds a writable RArray seeded with integers.
func mustArray(t *testing.T, state dtrules.State, vs ...int64) *dtrules.RArray {
	t.Helper()
	arr, err := dtrules.NewArray(state.GetSession(), true, false)
	if err != nil {
		t.Fatalf("NewArray: %v", err)
	}
	for _, v := range vs {
		arr.Add(dtrules.GetRIntegerValue(v))
	}
	return arr
}

func arrayInts(t *testing.T, arr *dtrules.RArray) []int64 {
	t.Helper()
	out := make([]int64, 0, arr.Size())
	for _, e := range arr.GetIterator() {
		v, err := e.LongValue()
		if err != nil {
			t.Fatalf("LongValue: %v", err)
		}
		out = append(out, v)
	}
	return out
}

// TestSortArrayAscendingDescending pins bubble-sort output for both
// directions. Note the parameter sense is inverted from its name: the
// sampleproject passes `false` to get ascending order (see
// SyntaxTests/xml/syntaxexample_dt.xml: "earliest of cases after
// currentDate" → `false sortarray for`). Pin that sense — a rewrite
// that "fixes" the parameter name without updating the .xml would
// silently flip sampleproject output.
func TestSortArrayAscendingDescending(t *testing.T) {
	state := newTestState()
	for _, tc := range []struct {
		name  string
		param bool
		want  []int64
	}{
		{"false_produces_ascending", false, []int64{1, 2, 3, 5, 8}},
		{"true_produces_descending", true, []int64{8, 5, 3, 2, 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			arr := mustArray(t, state, 5, 3, 8, 1, 2)
			state.DataPush(arr)
			state.DataPush(dtrules.GetRBoolean(tc.param))
			o, _ := Get(dtrules.GetRName("sortarray"))
			if err := o.Execute(state); err != nil {
				t.Fatalf("sortarray: %v", err)
			}
			got := arrayInts(t, arr)
			if !intSliceEqual(got, tc.want) {
				t.Errorf("sortarray %s: got %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

// TestCopyElements — copyelements appends src into dest.
func TestCopyElements(t *testing.T) {
	state := newTestState()
	dest := mustArray(t, state, 1, 2)
	src := mustArray(t, state, 3, 4, 5)
	state.DataPush(dest)
	state.DataPush(src)
	o, _ := Get(dtrules.GetRName("copyelements"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("copyelements: %v", err)
	}
	want := []int64{1, 2, 3, 4, 5}
	got := arrayInts(t, dest)
	if !intSliceEqual(got, want) {
		t.Errorf("copyelements: dest=%v, want %v", got, want)
	}
}

// TestDeepCopyProducesIndependentArray — deepcopy produces a new array
// whose elements are clones of the original, not shared references.
func TestDeepCopyProducesIndependentArray(t *testing.T) {
	state := newTestState()
	orig := mustArray(t, state, 10, 20, 30)
	state.DataPush(orig)
	o, _ := Get(dtrules.GetRName("deepcopy"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("deepcopy: %v", err)
	}
	top, _ := state.DataPop()
	clone, err := top.RArrayValue()
	if err != nil {
		t.Fatalf("RArrayValue: %v", err)
	}
	if clone.Size() != orig.Size() {
		t.Fatalf("deepcopy size mismatch: got %d, want %d",
			clone.Size(), orig.Size())
	}
	// Append to original; clone must stay the same size.
	orig.Add(dtrules.GetRIntegerValue(99))
	if clone.Size() == orig.Size() {
		t.Errorf("deepcopy did not detach: mutation to original grew clone too")
	}
	wantClone := []int64{10, 20, 30}
	if !intSliceEqual(arrayInts(t, clone), wantClone) {
		t.Errorf("deepcopy contents: got %v, want %v",
			arrayInts(t, clone), wantClone)
	}
}

// -- Lookup / get ------------------------------------------------------------

// TestLookupOnEntityStack — lookup finds a name on the entity stack
// and pushes its value.
func TestLookupOnEntityStack(t *testing.T) {
	state := newTestState()
	e := newTestEntity("Box", 1, map[string]dtrules.Object{
		"size": dtrules.GetRIntegerValue(10),
	})
	// Seed the attribute value via def (else default is 0).
	state.EntityPush(e)
	state.DataPush(dtrules.GetRIntegerValue(10))
	state.DataPush(dtrules.GetRName("size"))
	defOp, _ := Get(dtrules.GetRName("def"))
	defOp.Execute(state)

	// Lookup should return 10.
	state.DataPush(dtrules.GetRName("size"))
	o, _ := Get(dtrules.GetRName("lookup"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("lookup: %v", err)
	}
	top, _ := state.DataPop()
	v, _ := top.LongValue()
	if v != 10 {
		t.Errorf("lookup: got %d, want 10", v)
	}

	// Missing name should return an UndefinedError.
	state.DataPush(dtrules.GetRName("nope"))
	if err := o.Execute(state); err == nil {
		t.Errorf("lookup of undefined name should error")
	}
}

// TestGetReadsAttributeFromEntity — get: ( entity name -- value )
// reads an attribute directly from the given entity (no entity-stack
// search).
func TestGetReadsAttributeFromEntity(t *testing.T) {
	state := newTestState()
	e := newTestEntity("Box", 1, map[string]dtrules.Object{
		"depth": dtrules.GetRIntegerValue(7),
	})
	// get uses entity.Get which returns the default if unset.
	state.DataPush(e)
	state.DataPush(dtrules.GetRName("depth"))
	o, _ := Get(dtrules.GetRName("get"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("get: %v", err)
	}
	top, _ := state.DataPop()
	v, _ := top.LongValue()
	if v != 7 {
		t.Errorf("get(depth): got %d, want 7", v)
	}
}

// -- Date ops ----------------------------------------------------------------

// TestGetDateReturnsDate — getdate pushes now() as a date object.
func TestGetDateReturnsDate(t *testing.T) {
	state := newTestState()
	o, _ := Get(dtrules.GetRName("getdate"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("getdate: %v", err)
	}
	top, _ := state.DataPop()
	if top.Type() != dtrules.TypeDate {
		t.Errorf("getdate: got type %s, want Date", top.Type().String())
	}
}

// TestGetTimestampFormat — gettimestamp returns a string formatted
// "YYYY-MM-DD HH:MM:SS.nnnnnnnnn". Verify the shape.
func TestGetTimestampFormat(t *testing.T) {
	state := newTestState()
	pushDate(t, state, 2024, 7, 15)
	o, _ := Get(dtrules.GetRName("gettimestamp"))
	if err := o.Execute(state); err != nil {
		t.Fatalf("gettimestamp: %v", err)
	}
	top, _ := state.DataPop()
	s := top.StringValue()
	// Expected prefix for 2024-07-15 midnight UTC.
	want := "2024-07-15 00:00:00"
	if len(s) < len(want) || s[:len(want)] != want {
		t.Errorf("gettimestamp: got %q, want prefix %q", s, want)
	}
}

// -- Error ops ---------------------------------------------------------------

// TestThrowException pushes an exception message; the op must return
// an error.
func TestThrowException(t *testing.T) {
	state := newTestState()
	state.DataPush(dtrules.NewRString("boom"))
	o, _ := Get(dtrules.GetRName("throwexception"))
	if err := o.Execute(state); err == nil {
		t.Error("throwexception should return an error")
	}
}

// TestErrorOperator — error: ( exceptionName message -- ) always errors.
func TestErrorOperator(t *testing.T) {
	state := newTestState()
	state.DataPush(dtrules.NewRString("MyException"))
	state.DataPush(dtrules.NewRString("something broke"))
	o, _ := Get(dtrules.GetRName("error"))
	if err := o.Execute(state); err == nil {
		t.Error("error op should return an error")
	}
}
