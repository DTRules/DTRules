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

// Issue #712: `for all <array> as <alias>` binds each iteration entity to
// a local-entity slot instead of pushing it on the entity stack. The alias
// is the only way to reach the current iteration entity, which makes nested
// same-list iterations non-shadowing.

func TestForallAs_BasicEmit(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"taxpayer.incomes": TypeArray,
	})
	got, err := c.CompileContext("for all taxpayer.incomes as inc")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	want := "null allocate execute deallocate pop dup { 0 local! dup execute } taxpayer.incomes for pop"
	if got != want {
		t.Errorf("\nwant: %s\ngot:  %s", want, got)
	}
	// The alias path MUST NOT emit entitypush — that was the whole point.
	if strings.Contains(got, "entitypush") {
		t.Errorf("alias path emitted entitypush; got: %s", got)
	}
}

func TestForallAs_WhereClause(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"taxpayer.incomes": TypeArray,
		"income.amount":    TypeInteger,
	})
	got, err := c.CompileContext("for all taxpayer.incomes as inc where inc.amount > 0")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	// The body should only run when the predicate holds, and the alias
	// slot is populated BEFORE the predicate is evaluated so `inc.amount`
	// resolves via the local slot.
	want := "null allocate execute deallocate pop dup { 0 local! { dup execute } 0 local@ /amount get 0 > if } taxpayer.incomes for pop"
	if got != want {
		t.Errorf("\nwant: %s\ngot:  %s", want, got)
	}
	if strings.Contains(got, "entitypush") {
		t.Errorf("alias where path emitted entitypush; got: %s", got)
	}
}

// TestForallAs_NestedNonShadowing pins the motivating use case: two iterations
// over the same list with distinct aliases should both be reachable inside the
// inner body via local-slot access rather than the entity stack.
func TestForallAs_NestedNonShadowing(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"taxpayer.relatives": TypeArray,
		"person.parent_id":   TypeInteger,
		"person.id":          TypeInteger,
	})
	if _, err := c.CompileContext("for all taxpayer.relatives as parent"); err != nil {
		t.Fatalf("outer compile: %v", err)
	}
	// Inner iteration sees both `parent.*` (slot 0) and `child.*` (slot 1).
	got, err := c.CompileContext("for all taxpayer.relatives as child where child.parent_id == parent.id")
	if err != nil {
		t.Fatalf("inner compile: %v", err)
	}
	// The inner body references parent.id via slot 0 and stores child in slot 1.
	// Both resolve through local slots — no entitypush anywhere.
	if strings.Contains(got, "entitypush") {
		t.Errorf("nested alias path emitted entitypush; got: %s", got)
	}
	// parent.id should resolve as `0 local@ /id get` (slot 0 = parent).
	if !strings.Contains(got, "0 local@ /id get") {
		t.Errorf("expected `0 local@ /id get` for parent.id; got: %s", got)
	}
	// child.parent_id should resolve as `1 local@ /parent_id get` (slot 1 = child).
	if !strings.Contains(got, "1 local@ /parent_id get") {
		t.Errorf("expected `1 local@ /parent_id get` for child.parent_id; got: %s", got)
	}
	// child is stored into slot 1.
	if !strings.Contains(got, "1 local!") {
		t.Errorf("expected `1 local!` to store child element; got: %s", got)
	}
}

// TestForallAs_CollisionWithEntityType rejects aliases that shadow an existing
// EDD symbol because `<alias>.<field>` would be ambiguous with the registered
// entity-type path.
func TestForallAs_CollisionWithEntityType(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"income.amount": TypeInteger,
		"income":        TypeEntity,
	})
	_, err := c.CompileContext("for all taxpayer.incomes as income")
	if err == nil {
		t.Fatal("expected error for alias colliding with entity type")
	}
	if !strings.Contains(err.Error(), "collides") {
		t.Errorf("error should mention 'collides'; got: %v", err)
	}
}

// TestForallAs_NonAliasRegressionPin makes sure the existing `for all X`
// path (no alias) still pushes on the entity stack via `forall`, not the new
// local-slot `for` path.
func TestForallAs_NonAliasRegressionPin(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"taxpayer.incomes": TypeArray,
	})
	got, err := c.CompileContext("for all taxpayer.incomes")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	want := "dup taxpayer.incomes forall pop"
	if got != want {
		t.Errorf("\nwant: %s\ngot:  %s", want, got)
	}
}
