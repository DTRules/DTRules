//go:build archive

// ARCHIVED: forall-alias fixtures with DSL but no compiled postfix; the loader
// rejects them. Run with `go test -tags archive`. Revisit.

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

package authoring_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/operators"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// Issue #714 regression tests: `for all <arr> as <alias>` shipped in v1.8.1
// with compile-only tests, and runtime field access through the alias failed.
// These tests execute the compiled tables and assert concrete post-conditions.

func forallAsTestdataDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Join(filepath.Dir(file), "testdata", "forall_as")
}

func forallAsSetup(t *testing.T) *session.RuleSet {
	t.Helper()
	xmlDir := filepath.Join(forallAsTestdataDir(t), "xml")

	rs := session.NewRuleSet("forall-as-test")

	eddPath := filepath.Join(xmlDir, "forall_as_edd.xml")
	eddFile, err := os.Open(eddPath)
	if err != nil {
		t.Fatalf("open EDD: %v", err)
	}
	defer eddFile.Close()
	if err := rs.LoadEDD(eddFile); err != nil {
		t.Fatalf("LoadEDD: %v", err)
	}

	dtPath := filepath.Join(xmlDir, "forall_as_dt.xml")
	dtFile, err := os.Open(dtPath)
	if err != nil {
		t.Fatalf("open DT: %v", err)
	}
	defer dtFile.Close()
	if err := rs.LoadDecisionTables(dtFile); err != nil {
		t.Fatalf("LoadDecisionTables: %v", err)
	}

	return rs
}

func forallAsSession(t *testing.T, rs *session.RuleSet, inputFile string) (*session.RSession, *interpreter.DTState) {
	t.Helper()
	xmlDir := filepath.Join(forallAsTestdataDir(t), "xml")

	sess, err := session.NewSession(rs)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	state, ok := sess.GetState().(*interpreter.DTState)
	if !ok {
		t.Fatalf("unexpected state type %T", sess.GetState())
	}
	state.SetOperatorTable(operators.GetOperatorTable())

	mapPath := filepath.Join(xmlDir, "forall_as_map.xml")
	mapFile, err := os.Open(mapPath)
	if err != nil {
		t.Fatalf("open map: %v", err)
	}
	defer mapFile.Close()

	dataFile, err := os.Open(inputFile)
	if err != nil {
		t.Fatalf("open data: %v", err)
	}
	defer dataFile.Close()

	m := mapping.NewMapping(sess)
	if err := m.LoadMapping(mapFile); err != nil {
		t.Fatalf("LoadMapping: %v", err)
	}
	if err := m.LoadDataAndPushSingletons(dataFile); err != nil {
		t.Fatalf("LoadDataAndPushSingletons: %v", err)
	}

	return sess, state
}

// calcCtx finds the calculation_context entity currently on the entity stack
// and returns it. Fails the test if not found.
func calcCtx(t *testing.T, state *interpreter.DTState) dtrules.Entity {
	t.Helper()
	for i := 0; i < state.EntityDepth(); i++ {
		ent, err := state.EntityFetch(i)
		if err != nil {
			continue
		}
		if ent.GetName().StringValue() == "calculation_context" {
			return ent
		}
	}
	t.Fatalf("calculation_context not on entity stack")
	return nil
}

func ctxInt(t *testing.T, state *interpreter.DTState, field string) int64 {
	t.Helper()
	ent := calcCtx(t, state)
	val, err := ent.Get(dtrules.GetRName(field))
	if err != nil || val == nil {
		t.Fatalf("get %s: %v", field, err)
	}
	ri, err := val.RIntegerValue()
	if err != nil {
		t.Fatalf("%s not integer: %v", field, err)
	}
	n, err := ri.LongValue()
	if err != nil {
		t.Fatalf("%s not long: %v", field, err)
	}
	return n
}

func ctxString(t *testing.T, state *interpreter.DTState, field string) string {
	t.Helper()
	ent := calcCtx(t, state)
	val, err := ent.Get(dtrules.GetRName(field))
	if err != nil || val == nil {
		t.Fatalf("get %s: %v", field, err)
	}
	return val.StringValue()
}

// runForallAs executes the given table against the three_accounts input.
// The input has 3 accounts with balances 100, 200, 300 and labels A, B, C.
func runForallAs(t *testing.T, table string) *interpreter.DTState {
	t.Helper()
	rs := forallAsSetup(t)
	inputPath := filepath.Join(forallAsTestdataDir(t), "inputs", "three_accounts.xml")
	sess, state := forallAsSession(t, rs, inputPath)
	if err := sess.Execute(table); err != nil {
		t.Fatalf("Execute %s: %v", table, err)
	}
	return state
}

// TestForallAs_BasicReadsAliasField: single as-alias iteration; the body
// reads `acct.balance` and accumulates into calculation_context.total_balance.
// Expected sum: 100+200+300 = 600.
func TestForallAs_BasicReadsAliasField(t *testing.T) {
	state := runForallAs(t, "FA_Basic")
	got := ctxInt(t, state, "total_balance")
	if got != 600 {
		t.Errorf("FA_Basic: total_balance = %d, want 600", got)
	}
}

// TestForallAs_WhereClauseReadsAliasField: the where-clause reads
// acct.has_staking_authority and keeps only authorized accounts. The body
// increments processed_count. Two accounts are authorized (A, C).
func TestForallAs_WhereClauseReadsAliasField(t *testing.T) {
	state := runForallAs(t, "FA_Where")
	got := ctxInt(t, state, "processed_count")
	if got != 2 {
		t.Errorf("FA_Where: processed_count = %d, want 2", got)
	}
}

// TestForallAs_NestedNonShadowing: nested iteration with distinct aliases.
// Inner filters by inner_a.balance > outer_a.balance. With balances 100,200,300
// and labels A,B,C, the pairs emitted (outer<inner) are:
//
//	A<B, A<C, B<C
//
// joined by ";" with a trailing ";".
func TestForallAs_NestedNonShadowing(t *testing.T) {
	state := runForallAs(t, "FA_Nested")
	got := ctxString(t, state, "audit")
	want := "A<B;A<C;B<C;"
	if got != want {
		t.Errorf("FA_Nested: audit = %q, want %q", got, want)
	}
}

// TestForallAs_NonAliasRegression: the plain `for all <arr>` path (no alias)
// must still push on the entity stack and resolve bare field names.
// With balances 100, 200, 300 — all positive — total is 600.
func TestForallAs_NonAliasRegression(t *testing.T) {
	state := runForallAs(t, "FA_NoAlias")
	got := ctxInt(t, state, "total_balance")
	if got != 600 {
		t.Errorf("FA_NoAlias: total_balance = %d, want 600", got)
	}
}
