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

package analysis

import "testing"

// Two ways a field that is read on every run was reported as unused, both from
// CHIP's `is there relationship in case.relationships where (type == "parent"
// and source == client and target == ApplyingClient)` (#776).

func relSchema() *eddSchema {
	return &eddSchema{
		FieldsByEntity: map[string]map[string]bool{
			"relationship": {"type": true, "source": true, "target": true},
			"case":         {"relationships": true},
		},
	}
}

// The existential quantifier binds its element exactly as `for all` does. It
// names the entity directly, which matters because the array it iterates
// cannot be relied on to say: CHIP declares `relationships` as
// `type="array" subtype=""`, so nothing recovers the element type from it.
func TestExistentialBindsItsElement(t *testing.T) {
	got := inlinePushes(`is there relationship in case.relationships where (type == "parent")`, relSchema())

	var found bool
	for _, e := range got {
		if e == "relationship" {
			found = true
		}
	}
	if !found {
		t.Errorf("pushed %v, want relationship — its `where` clause reads the element's fields", got)
	}
}

func TestThereIsNoAlsoBinds(t *testing.T) {
	if got := inlinePushes(`there is no relationship in case.relationships where source == client`, relSchema()); len(got) == 0 {
		t.Error("the negated spelling binds its element too")
	}
}

// The DSL pass must skip EL keywords, because a word like `type` is syntax
// there as often as it is a field. Postfix has no syntax left — the compiler
// already decided what every word meant — so a bare token naming a declared
// field is a read, keyword or not.
func TestKeywordNamedFieldIsSeenInPostfix(t *testing.T) {
	reads := map[string]bool{}
	extractBarePostfixReads(`false { type "parent" streq { pop source client req } over if } case.relationships forall`,
		[]string{"relationship"}, relSchema(), reads)

	for _, want := range []string{"relationship.type", "relationship.source"} {
		if !reads[want] {
			t.Errorf("%s was not counted as read; reads=%v", want, reads)
		}
	}
}

// Literals and block delimiters are not field reads.
func TestPostfixLiteralsAreNotReads(t *testing.T) {
	reads := map[string]bool{}
	extractBarePostfixReads(`{ } "type" 42 3.14`, []string{"relationship"}, relSchema(), reads)
	if len(reads) != 0 {
		t.Errorf("literals and delimiters were read as fields: %v", reads)
	}
}
