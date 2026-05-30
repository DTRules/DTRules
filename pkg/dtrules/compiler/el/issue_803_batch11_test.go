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

// Issue #803 batch 11: str-family and xmlvalues container.
// All six rules previously emitted empty postfix.

func issue803Batch11Symbols() map[string]string {
	return map[string]string{
		"client.code": "string",
		"client.ts":   "string",
		"family":      "entity",
		"family.head": "entity",
	}
}

// TestIssue803_StrTimestamp: `get current_timestamp` is a niladic
// runtime op call. The op is registered as `gettimestamp`.
func TestIssue803_StrTimestamp(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch11Symbols())
	got, err := c.CompileAction(`set client.ts = get current timestamp`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !strings.Contains(got, "gettimestamp") {
		t.Errorf("expected gettimestamp in postfix, got: %s", got)
	}
}

// TestIssue803_StrAttrOf_ErrorsLoudly: `attribute <name> of <entity>`
// has no runtime op; the visitor must emit elstmterror so the rule
// fails loudly instead of silently dropping the access.
func TestIssue803_StrAttrOf_ErrorsLoudly(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch11Symbols())
	got, err := c.CompileAction(`set client.code = attribute "name" of family`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !strings.Contains(got, "elstmterror") {
		t.Errorf("expected elstmterror in postfix, got: %s", got)
	}
}

// TestIssue803_StrMappingKey_ErrorsLoudly: bare `mappingkey` keyword
// has no runtime op in the Go engine. Must emit elstmterror.
func TestIssue803_StrMappingKey_ErrorsLoudly(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch11Symbols())
	got, err := c.CompileAction(`set client.code = mapping key`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !strings.Contains(got, "elstmterror") {
		t.Errorf("expected elstmterror in postfix, got: %s", got)
	}
}

// TestIssue803_StrRelationship_ErrorsLoudly: `relationship between
// <e1> and <e2>` has no runtime op. Must emit elstmterror.
func TestIssue803_StrRelationship_ErrorsLoudly(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(issue803Batch11Symbols())
	got, err := c.CompileAction(`set client.code = relationship between family and family.head`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !strings.Contains(got, "elstmterror") {
		t.Errorf("expected elstmterror in postfix, got: %s", got)
	}
}

// TestIssue803_StrXmlAttr_ErrorsLoudly: `<xmlval> : get attribute
// <name>` has no XML runtime op. Must emit elstmterror.
func TestIssue803_StrXmlAttr_ErrorsLoudly(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"client.code": "string",
		"client.xml":  "xmlvalue",
	})
	got, err := c.CompileAction(`set client.code = client.xml: get attribute "id"`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !strings.Contains(got, "elstmterror") {
		t.Errorf("expected elstmterror in postfix, got: %s", got)
	}
}

// TestIssue803_Xmlvalues_UnreachableViaXmlMutation pins the
// finding that VisitXmlvalues' override is defensive — every
// xmlvaluestatements alt emits an elstmterror without visiting the
// RHS xmlvalues. The whole mutation collapses to a placeholder. If
// the XML runtime is ever wired up and the parents stop short-
// circuiting, this test will fail loudly and the body should be
// replaced with a positive assertion on the dispatched child.
func TestIssue803_Xmlvalues_UnreachableViaXmlMutation(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"client.xml":  "xmlvalue",
		"client.code": "string",
	})
	got, err := c.CompileAction(`client.xml: set attribute "id" = client.code`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !strings.Contains(got, "elstmterror") {
		t.Errorf("expected xml mutation to short-circuit to elstmterror, got: %s", got)
	}
	if strings.Contains(got, "client.code") {
		t.Errorf("xmlvalues now reachable; replace this test with a positive child-dispatch assertion. got: %s", got)
	}
}
