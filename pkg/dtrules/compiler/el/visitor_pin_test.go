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
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Issue #803: many labeled alts in EL.g4 had no Visit<Label> override
// on PostfixEmitter and silently produced wrong or empty postfix — the
// dispatch fell through to BaseELVisitor, whose `VisitChildren` is a
// no-op in antlr's Go runtime (parser/antlr/v4/tree.go: returns nil
// without walking children). The contract from #687 is: every labeled
// alt needs an explicit override, or it must be on the
// inheritedAllowlist with a one-line rationale explaining why
// fall-through is OK.
//
// This file is the pin: it walks the source of postfix_emitter.go vs
// el_visitor.go, computes which Visit<Label> methods are inherited
// (not overridden), and asserts the set matches inheritedAllowlist.
// Drift in either direction fails the test:
//
//   - A new alt without an override → fails until allowlisted or fixed.
//   - An allowlisted alt gets a real override → fails until the
//     allowlist entry is removed (cleaning house as we go).

// inheritedAllowlist names every Visit<Label> method that PostfixEmitter
// intentionally inherits from BaseELVisitor. Each entry must be
// justified: either the rule is purely structural (no semantics to
// emit), or it has a sibling alt that handles the common case in
// practice. UNVERIFIED entries are tracked as TODOs against #803.
var inheritedAllowlist = map[string]string{
	// Structural / no-semantics rules (correct as no-op):
	"VisitCommonerror":          "antlr error-recovery context; not part of normal parse",
	"VisitOptSemi":               "optional semicolon terminal; nothing to emit",
	"VisitSeparator":             "literal separator token; nothing to emit",
	"VisitEmptyContext":          "empty context rule",
	"VisitEmptyPolicyStatement":  "empty policy statement",

	// Container rules whose VisitChildren-equivalent walking is done
	// elsewhere (or which never appear in compiled output because a
	// sibling alt wins parser-side):
	"VisitInthe":  "`in the` keyword sequence inside a larger rule; consumed by parent visitors",

	// =========================================================
	// TODO(#803): UNVERIFIED — needs reproducer + triage. Each
	// of these is an inherited Visit<Label> we haven't yet
	// confirmed is harmless. Listed here so this test passes
	// today; convert to explicit overrides or to a verified
	// fall-through rationale in follow-up PRs.
	// =========================================================
	// addtodest2 alts are reached only from addDestColon/subDestColon,
	// which type-switch and extract `GetText()` directly without calling
	// Visit. Verified by inspection (#803 batch 7).
	"VisitAddDestArray2":              "dead grammar; addDest/subDestColon extract via GetText, never Visit",
	"VisitAddDestDouble2":             "dead grammar; addDest/subDestColon extract via GetText, never Visit",
	"VisitAddDestLong2":               "dead grammar; addDest/subDestColon extract via GetText, never Visit",
	// blist / blistIc alts are traversed directly by the parent
	// visitors (VisitBoolStrEqList / VisitBoolStrEqIcList via
	// collectBlistStrexprs), not via the antlr Visit dispatch — so
	// these Visit* methods are never reached (#803 batch 10).
	"VisitBlistIcMulti":               "dead grammar; parent traverses via collectBlistStrexprs, never Visit",
	"VisitBlistIcOr":                  "dead grammar; parent traverses via collectBlistStrexprs, never Visit",
	"VisitBlistMulti":                 "dead grammar; parent traverses via collectBlistStrexprs, never Visit",
	"VisitBlistOr":                    "dead grammar; parent traverses via collectBlistStrexprs, never Visit",
	"VisitDateEarliestAfter":          "TODO(#803): triage; needs new `earliestafter` runtime op",
	"VisitEntityFirst":                "TODO(#803): triage",
	"VisitEntityFirstIn":              "TODO(#803): triage",
	// AddTo/SubFrom — these alts live inside fexpr/iexpr but the
	// action-statement form `add X to Y` / `subtract X from Y` parses
	// as addtostatement (rule `addtostatement`), not via the fexpr
	// path. Confirmed by tree-dump probe; `add 2 to a.x` matches
	// addNumToDest and emits the correct postfix through that path
	// (#803 batch 4).
	"VisitFloatAddTo":                 "dead grammar; addtostatement handles `add X to Y` as a statement",
	"VisitFloatSubFrom":               "dead grammar; addtostatement handles `subtract X from Y`",
	// Float/Int/Str Using are unreachable: ANTLR adaptive prediction
	// picks intUsingArray (in iexpr) first for the `using <ident>(<expr>)`
	// shape because both IDENT-typed sides match more broadly. The
	// actually-reached intUsingArray now has an override (#803 batch 6).
	"VisitFloatUsing":                 "dead grammar; intUsingArray wins parser-side for IDENT inputs",
	"VisitIfThen":                     "TODO(#803): triage; if/then in action statements has separate dispatch",
	"VisitIfThenElse":                 "TODO(#803): triage; same",
	"VisitIntAddTo":                   "dead grammar; addtostatement handles `add X to Y` (see VisitFloatAddTo)",
	"VisitIntSubFrom":                 "dead grammar; addtostatement handles `subtract X from Y`",
	"VisitIntUsing":                   "dead grammar; intUsingArray wins parser-side (see VisitFloatUsing)",
	"VisitLeftArrayColon":             "TODO(#803): triage",
	// leftTexpr alts are unreachable because the only SET form that
	// targets a typedTable (setTable) now emits an elstmterror
	// placeholder without visiting the leftTexpr (hash tables removed,
	// #803 batch 6).
	"VisitLeftTexprColon":             "dead grammar; setTable emits elstmterror without visiting leftTexpr",
	"VisitLeftTexprSimple":            "dead grammar; setTable emits elstmterror without visiting leftTexpr",
	"VisitOpListEntity":               "TODO(#803): triage",
	"VisitOpListEntitySingle":         "TODO(#803): triage",
	"VisitPerformName":                "dead grammar; ANTLR matches performDT/performDTExplicit first for any IDENT after PERFORM (verified by tree dump)",
	// setArray<Type> are unreachable for non-array RHS: ANTLR picks
	// setInt/setFloat/setString/setEntity/setDate first when the RHS
	// could be either a single typed value or an arrayExpr. The only
	// reachable setArray alt is setArrayArray (which now has an
	// override). Verified by tree-dump probe (#803 batch 3).
	"VisitSetArrayDate":               "dead grammar; setDate wins for IDENT/dexpr RHS",
	"VisitSetArrayEntity":             "dead grammar; setEntity wins for IDENT/eexpr RHS",
	"VisitSetArrayFloat":              "dead grammar; setFloat wins for IDENT/fexpr RHS",
	"VisitSetArrayInt":                "dead grammar; setInt wins for IDENT/iexpr RHS",
	"VisitSetArrayString":             "dead grammar; setString wins for IDENT/strexpr RHS",
	// setStringFromNumber/Name/Table are unreachable: ANTLR adaptive
	// prediction picks setInt/setFloat/setName/setTable first for
	// IDENT-prefixed RHS. Confirmed by parse-tree inspection (#803 batch 2).
	"VisitSetStringFromNumber":        "dead grammar; ANTLR picks setInt/setFloat for IDENT/number RHS",
	"VisitSetStringFromTable":         "dead grammar; ANTLR picks setTable for texpr RHS",
	// strConcat<Type> are all unreachable: ANTLR always picks the base
	// `strexpr PLUS strexpr` # strConcat first because the RHS IDENT
	// matches typedXmlValue inside strexpr. Confirmed by parse-tree
	// inspection across int/float/date/name/entity/array/null/invalid
	// RHS shapes (#803 batch 2).
	"VisitStrConcatArray":             "dead grammar; base strConcat wins parser-side",
	"VisitStrConcatDate":              "dead grammar; base strConcat wins parser-side",
	"VisitStrConcatEntity":            "dead grammar; base strConcat wins parser-side",
	"VisitStrConcatFloat":             "dead grammar; base strConcat wins parser-side",
	"VisitStrConcatInt":               "dead grammar; base strConcat wins parser-side",
	"VisitStrConcatInvalid":           "dead grammar; base strConcat wins parser-side",
	"VisitStrConcatNull":              "dead grammar; base strConcat wins parser-side",
	"VisitStrUsing":                   "dead grammar; intUsingArray wins parser-side (see VisitFloatUsing)",
	// tablelist / tableTyped are helper rules referenced from the
	// table-lookup alts; with the table-lookup parent emitting
	// elstmterror placeholders (#803 batch 6), the helpers are never
	// reached as visitors.
	"VisitTableListMulti":             "dead grammar; helper rule under table-lookup which emits elstmterror",
	"VisitTableListSingle":            "dead grammar; helper rule under table-lookup which emits elstmterror",
	"VisitTableTyped":                 "dead grammar; helper rule under table-lookup which emits elstmterror",
	"VisitThereis":                    "TODO(#803): triage",
	// typedXxx and undefinedIdent are IDENT-classification rules. Every
	// parent visitor that consumes them extracts the text via GetText()
	// directly (e.g. VisitOperatorstatements at line 4906 reads
	// `ctx.TypedOperator().GetText()`; VisitLocalEntityInit reads
	// `ctx.UndefinedIdent().GetText()`). The Visit() entry points are
	// never invoked. Verified by source inspection (#803 batch 7).
	"VisitTypedBoolFunction":          "dead grammar; consumers use TypedBoolFunction().GetText() directly",
	"VisitTypedInvalid":               "dead grammar; only used by dead-grammar strConcatInvalid",
	"VisitTypedNull":                  "dead grammar; only used by dead-grammar strConcatNull",
	"VisitTypedOperator":              "dead grammar; VisitOperatorstatements extracts via GetText",
	"VisitUndefinedIdent":             "dead grammar; CREATE/LOCAL parents extract via UndefinedIdent().GetText",
	"VisitUsingstatement":             "dead grammar; the rule's only alt wraps usingblock which is visited via children elsewhere",
}

// TestPostfixEmitterVisitorCoverage asserts that the inherited
// (non-overridden) Visit<Label> methods on PostfixEmitter exactly
// match inheritedAllowlist — failing on drift in either direction.
//
// Adding a new labeled alt requires either (a) a Visit<Label> in
// postfix_emitter.go or (b) an allowlist entry with a rationale.
// Adding an override for an existing allowlisted entry requires
// removing the entry. The test catches both at CI time.
func TestPostfixEmitterVisitorCoverage(t *testing.T) {
	declared := readDeclaredVisitMethods(t)
	overridden := readOverriddenVisitMethods(t)

	inherited := []string{}
	for name := range declared {
		if !overridden[name] {
			inherited = append(inherited, name)
		}
	}
	sort.Strings(inherited)

	allowed := map[string]bool{}
	for k := range inheritedAllowlist {
		allowed[k] = true
	}

	// Extras: inherited but not in allowlist → new silent-failure
	// risk; add a Visit<Label> override or document why fall-through
	// is OK.
	extras := []string{}
	for _, name := range inherited {
		if !allowed[name] {
			extras = append(extras, name)
		}
	}
	if len(extras) > 0 {
		t.Errorf("New inherited Visit methods without allowlist entry (#803). "+
			"Either add a Visit<Label> override in postfix_emitter.go or "+
			"add the method to inheritedAllowlist with a rationale:\n  %s",
			strings.Join(extras, "\n  "))
	}

	// Stale: allowlisted but no longer inherited → an override was
	// added; remove the allowlist entry to keep this honest.
	stale := []string{}
	for name := range inheritedAllowlist {
		if overridden[name] || !declared[name] {
			stale = append(stale, name)
		}
	}
	sort.Strings(stale)
	if len(stale) > 0 {
		t.Errorf("Allowlist entries that are no longer inherited. "+
			"Remove these from inheritedAllowlist:\n  %s",
			strings.Join(stale, "\n  "))
	}
}

// readDeclaredVisitMethods extracts every `VisitXxx(ctx *XxxContext) interface{}`
// declared in the generated el_visitor.go interface. Each declared
// method corresponds to a labeled alt (or an unlabeled rule that
// antlr nonetheless emits a Visit for).
func readDeclaredVisitMethods(t *testing.T) map[string]bool {
	t.Helper()
	src, err := os.ReadFile("el_visitor.go")
	if err != nil {
		t.Fatalf("read el_visitor.go: %v", err)
	}
	re := regexp.MustCompile(`\b(Visit\w+)\(ctx \*\w+Context\) interface\{\}`)
	out := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(string(src), -1) {
		out[m[1]] = true
	}
	return out
}

// readOverriddenVisitMethods extracts every `func (e *PostfixEmitter) VisitXxx(`
// implemented in postfix_emitter.go. Helper functions on PostfixEmitter
// that don't follow the Visit convention are ignored.
func readOverriddenVisitMethods(t *testing.T) map[string]bool {
	t.Helper()
	src, err := os.ReadFile("postfix_emitter.go")
	if err != nil {
		t.Fatalf("read postfix_emitter.go: %v", err)
	}
	re := regexp.MustCompile(`(?m)^func \(e \*PostfixEmitter\) (Visit\w+)\(`)
	out := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(string(src), -1) {
		out[m[1]] = true
	}
	return out
}
