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

package dtrules

// ChangeTracker is implemented by a State that records whether execution
// changed entity data (#1233). A host that runs rules on a timer resets the
// flag, executes, and skips journaling when Changed reports false.
//
// A change is a write by the running rules that alters what a save would
// write: an attribute set to a value that saves differently (SameValue), an
// array that gained, lost or reordered elements, or an entity stack left
// different from the one ResetChanged recorded. Writing the value a field
// already holds is not a change. Writes that bypass the rules (entity.Put
// from a host, the loaders, trace replay) are not counted. The flag is
// sticky: it stays set across executions until ResetChanged.
type ChangeTracker interface {
	// Changed reports whether entity data changed since the state was
	// created or last reset.
	Changed() bool
	// ResetChanged clears the flag.
	ResetChanged()
	// MarkChanged sets the flag.
	MarkChanged()
}

// MarkChanged records on state, if it tracks changes, that entity data
// changed.
func MarkChanged(state State) {
	if t, ok := state.(ChangeTracker); ok {
		t.MarkChanged()
	}
}

// MarkArrayChanged records an in-place change to arr, unless arr is a fresh
// array that nothing holds yet: building an array literal is not a change;
// storing it is, when it differs from what it replaces.
func MarkArrayChanged(state State, arr *RArray) {
	if arr != nil && !arr.fresh {
		MarkChanged(state)
	}
}

// WatchArray is for operators that rewrite an array in place (sort, shuffle,
// clear and refill): call it before, and the returned function after. The
// array counts as changed when its elements differ afterwards. It costs
// nothing when the answer is already known: a state that does not track
// changes, a flag already set, or a fresh array.
func WatchArray(state State, arr *RArray) func() {
	t, ok := state.(ChangeTracker)
	if !ok || arr == nil || arr.fresh || t.Changed() {
		return func() {}
	}
	before := append([]Object(nil), arr.GetIterator()...)
	return func() {
		after := arr.GetIterator()
		if len(after) != len(before) {
			t.MarkChanged()
			return
		}
		for i := range after {
			if !SameValue(before[i], after[i]) {
				t.MarkChanged()
				return
			}
		}
	}
}

// SameValue reports whether a write that replaced old with new left the
// value unchanged, as a save would write it: scalars of the same type with
// the same text, the same entity (by identity), or arrays whose elements
// are pairwise the same. It is deliberately stricter than Equals, which
// lets doubles differ by 1e-9 and names differ in case, so a value whose
// saved form changed is never reported as unchanged.
func SameValue(old, new Object) bool {
	if old == nil || new == nil {
		return old == nil && new == nil
	}
	if old == new {
		return true
	}
	if old.Type().GetID() != new.Type().GetID() {
		return false
	}
	if oa, ok := old.(*RArray); ok {
		na, ok := new.(*RArray)
		if !ok {
			return false
		}
		a, b := oa.GetIterator(), na.GetIterator()
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if !SameValue(a[i], b[i]) {
				return false
			}
		}
		return true
	}
	if _, ok := old.(Entity); ok {
		return false // distinct entities (identity was checked above)
	}
	return old.StringValue() == new.StringValue()
}
