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
	"VisitAddDestArray2":              "TODO(#803): triage",
	"VisitAddDestDouble2":             "TODO(#803): triage",
	"VisitAddDestLong2":               "TODO(#803): triage",
	"VisitArrayColonRef":              "TODO(#803): triage",
	"VisitArrayDeepCopy":              "TODO(#803): triage",
	"VisitArrayDeepCopySimple":        "TODO(#803): triage",
	"VisitArrayMap":                   "TODO(#803): triage",
	"VisitArrayName":                  "TODO(#803): triage",
	"VisitArrayTokenize":              "TODO(#803): triage",
	"VisitBlistIcMulti":               "TODO(#803): triage",
	"VisitBlistIcOr":                  "TODO(#803): triage",
	"VisitBlistMulti":                 "TODO(#803): triage",
	"VisitBlistOr":                    "TODO(#803): triage",
	"VisitBoolColonRef":               "TODO(#803): triage",
	"VisitBoolFunction":               "TODO(#803): triage",
	"VisitBoolStrEqIcList":            "TODO(#803): triage",
	"VisitBoolStrEqList":              "TODO(#803): triage",
	"VisitDateColonRef":               "TODO(#803): triage",
	"VisitDateEarliestAfter":          "TODO(#803): triage",
	"VisitDateFromArrayAt":            "TODO(#803): triage",
	"VisitDateFromIndex":              "TODO(#803): triage",
	"VisitDateFromStrCast":            "TODO(#803): triage",
	"VisitDateFromStrFunc":            "TODO(#803): triage",
	"VisitDateTableLookup":            "TODO(#803): triage",
	"VisitDateUsing":                  "TODO(#803): triage",
	"VisitEntityFirst":                "TODO(#803): triage",
	"VisitEntityFirstIn":              "TODO(#803): triage",
	"VisitEntityTableLookup":          "TODO(#803): triage",
	"VisitFloatAddTo":                 "TODO(#803): triage; in practice action statement `add X to Y` uses a different dispatch and emits correctly",
	"VisitFloatColonRef":              "TODO(#803): triage",
	"VisitFloatSubFrom":               "TODO(#803): triage; same caveat as VisitFloatAddTo",
	"VisitFloatSumOf":                 "TODO(#803): triage",
	"VisitFloatTableLookup":           "TODO(#803): triage",
	"VisitFloatUsing":                 "TODO(#803): triage",
	"VisitIfThen":                     "TODO(#803): triage; if/then in action statements has separate dispatch",
	"VisitIfThenElse":                 "TODO(#803): triage; same",
	"VisitIntAddTo":                   "TODO(#803): triage; same caveat as VisitFloatAddTo",
	"VisitIntColonRef":                "TODO(#803): triage",
	"VisitIntIndexOf":                 "TODO(#803): triage",
	"VisitIntSubFrom":                 "TODO(#803): triage",
	"VisitIntSumOf":                   "TODO(#803): triage",
	"VisitIntTableLookup":             "TODO(#803): triage",
	"VisitIntUsing":                   "TODO(#803): triage",
	"VisitIntUsingArray":              "TODO(#803): triage",
	"VisitLeftArrayColon":             "TODO(#803): triage",
	"VisitLeftTexprColon":             "TODO(#803): triage",
	"VisitLeftTexprSimple":            "TODO(#803): triage",
	"VisitNameArrayAt":                "TODO(#803): triage",
	"VisitNameColonRef":               "TODO(#803): triage",
	"VisitNameFromStr":                "TODO(#803): triage",
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
	"VisitSetTable":                   "TODO(#803): triage",
	"VisitStrAttrOf":                  "TODO(#803): triage",
	"VisitStrColonRef":                "TODO(#803): triage",
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
	"VisitStrMappingKey":              "TODO(#803): triage",
	"VisitStrRelationship":            "TODO(#803): triage",
	"VisitStrTableInfo":               "TODO(#803): triage",
	"VisitStrTimestamp":               "TODO(#803): triage",
	"VisitStrUsing":                   "TODO(#803): triage",
	"VisitStrXmlAttr":                 "TODO(#803): triage",
	"VisitSubDestColon":               "TODO(#803): triage",
	"VisitTableListMulti":             "TODO(#803): triage",
	"VisitTableListSingle":            "TODO(#803): triage",
	"VisitTableTyped":                 "TODO(#803): triage",
	"VisitThereis":                    "TODO(#803): triage",
	"VisitTypedBoolFunction":          "TODO(#803): triage",
	"VisitTypedInvalid":               "TODO(#803): triage",
	"VisitTypedNull":                  "TODO(#803): triage",
	"VisitTypedOperator":              "TODO(#803): triage",
	"VisitUndefinedIdent":             "TODO(#803): triage",
	"VisitUsingstatement":             "TODO(#803): triage",
	"VisitXmlvalues":                  "TODO(#803): triage",
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
