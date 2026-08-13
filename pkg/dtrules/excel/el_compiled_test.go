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

// `el_compiled` has to describe the file, not the run that produced it.
//
// Written straight from the in-memory flag, the two build directions gave
// different answers and nothing could reconcile them: verify imports into an
// empty directory so every table is freshly compiled and the flag is true,
// while a build in place reads the flag back from XML predating the attribute
// and writes false again. A project maintained with `build` was permanently
// red on `content differs from build output`, differing by this attribute and
// nothing else -- and the only available fix, hand-editing it in, is what the
// authoring contract forbids (#1051).

func TestELCompiledIsDerivedFromTheRows(t *testing.T) {
	// A table whose flag was lost, but whose rows plainly came from EL.
	table := &DecisionTableXML{
		TableName:  "Recovered",
		Conditions: []ConditionXML{{DSL: "a is equal to b", Postfix: "a b eq"}},
	}
	if !elCompiled(table) {
		t.Error("a row with both DSL and postfix says the postfix came from the DSL; " +
			"reporting otherwise leaves the project permanently red")
	}
}

func TestHandWrittenPostfixIsNotMarked(t *testing.T) {
	// Postfix with no DSL is exactly what the attribute must not claim.
	table := &DecisionTableXML{
		TableName: "HandAuthored",
		Actions:   []ActionXML{{Postfix: "1 2 add"}},
	}
	if elCompiled(table) {
		t.Error("postfix with no DSL was marked as EL-compiled")
	}
}

func TestInitialActionsCountThroughBothSpellings(t *testing.T) {
	// Initial actions carry two element spellings; reading the field directly
	// sees only one of them.
	legacy := &DecisionTableXML{
		TableName:            "LegacySpelling",
		InitialActionsLegacy: []InitialActionXML{{ActionDSL: "perform X", ActionPostfix: "/X perform"}},
	}
	if !elCompiled(legacy) {
		t.Error("the legacy initial-action spelling was not consulted")
	}
}

// The in-memory flag still wins when it is set, so a fresh compile of a table
// whose rows are all empty is still marked.
func TestExplicitFlagStillCounts(t *testing.T) {
	if !elCompiled(&DecisionTableXML{TableName: "Empty", ELCompiled: true}) {
		t.Error("an explicitly compiled table lost its marker")
	}
}
