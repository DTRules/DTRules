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
	"fmt"
	"strings"
	"testing"
)

// Issue #703: `for all <type> entities [where <bexpr>]` rewrites a bare
// entity-type name to the EDD-declared `<owner>.<field>` path of the array
// that owns entities of that type. This keeps table authors from hand-
// coding collection paths they might forget to update when the EDD moves.

// stubResolver builds a fixed entityType → (owner, field) map and returns
// the resolver closure the EL compiler expects. Types not in the map return
// a "no collection" error; an empty owner marks an ambiguity sentinel.
func stubResolver(mapping map[string][][2]string) func(string) (string, string, error) {
	return func(entityType string) (string, string, error) {
		refs, ok := mapping[entityType]
		if !ok || len(refs) == 0 {
			return "", "", fmt.Errorf("no collection registered for type %q", entityType)
		}
		if len(refs) > 1 {
			cands := make([]string, len(refs))
			for i, r := range refs {
				cands[i] = r[0] + "." + r[1]
			}
			return "", "", fmt.Errorf("ambiguous collection for type %q: candidates [%s]", entityType, strings.Join(cands, ", "))
		}
		return refs[0][0], refs[0][1], nil
	}
}

func TestForallEntities_Simple(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"taxpayer.incomes": TypeArray,
	})
	c.SetCollectionResolver(stubResolver(map[string][][2]string{
		"income": {{"taxpayer", "incomes"}},
	}))
	got, err := c.CompileContext("for all income entities")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	want := "dup taxpayer.incomes forall pop"
	if got != want {
		t.Errorf("\nwant: %s\ngot:  %s", want, got)
	}
}

func TestForallEntities_Where(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"taxpayer.incomes": TypeArray,
		"income.amount":    TypeInteger,
	})
	c.SetCollectionResolver(stubResolver(map[string][][2]string{
		"income": {{"taxpayer", "incomes"}},
	}))
	got, err := c.CompileContext("for all income entities where income.amount > 0")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	want := "{ { dup execute } income.amount 0 > if } taxpayer.incomes forall pop"
	if got != want {
		t.Errorf("\nwant: %s\ngot:  %s", want, got)
	}
}

func TestForallEntities_NoCollectionError(t *testing.T) {
	c := NewCompiler()
	c.SetCollectionResolver(stubResolver(map[string][][2]string{}))
	_, err := c.CompileContext("for all orphan entities")
	if err == nil {
		t.Fatal("expected error when no collection is registered")
	}
	if !strings.Contains(err.Error(), "no collection") {
		t.Errorf("error should mention 'no collection'; got: %v", err)
	}
	if !strings.Contains(err.Error(), "orphan") {
		t.Errorf("error should name the unresolved type; got: %v", err)
	}
}

func TestForallEntities_AmbiguousListsCandidates(t *testing.T) {
	c := NewCompiler()
	c.SetCollectionResolver(stubResolver(map[string][][2]string{
		"income": {{"taxpayer", "incomes"}, {"spouse", "incomes"}},
	}))
	_, err := c.CompileContext("for all income entities")
	if err == nil {
		t.Fatal("expected ambiguity error when two owners declare the collection")
	}
	msg := err.Error()
	if !strings.Contains(msg, "ambiguous") {
		t.Errorf("error should mention 'ambiguous'; got: %v", err)
	}
	if !strings.Contains(msg, "taxpayer.incomes") || !strings.Contains(msg, "spouse.incomes") {
		t.Errorf("error should list both candidates; got: %v", err)
	}
}

func TestForallEntities_MissingResolverError(t *testing.T) {
	// No SetCollectionResolver — compiler should surface a clear error
	// rather than emitting bogus postfix.
	c := NewCompiler()
	_, err := c.CompileContext("for all income entities")
	if err == nil {
		t.Fatal("expected error when resolver is not configured")
	}
	if !strings.Contains(err.Error(), "resolver") {
		t.Errorf("error should mention 'resolver'; got: %v", err)
	}
}
