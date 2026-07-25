// Copyright 2025 DTRules contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package excel

import "testing"

func TestNormalizeEntityNumbers_Backfill(t *testing.T) {
	edd := &EDDXML{Entities: []*EDDXMLEntity{
		{Name: "a"},                  // missing -> 400
		{Name: "b", Number: "250"},   // kept
		{Name: "c", Number: "bogus"}, // non-numeric -> 500
		{Name: "d", Number: "300"},   // kept (max)
	}}
	normalizeEntityNumbers(edd)

	want := map[string]string{"a": "400", "b": "250", "c": "500", "d": "300"}
	for _, e := range edd.Entities {
		if e.Number != want[e.Name] {
			t.Errorf("entity %s: number = %q, want %q", e.Name, e.Number, want[e.Name])
		}
	}
}

func TestNormalizeEntityNumbers_FromScratch(t *testing.T) {
	edd := &EDDXML{Entities: []*EDDXMLEntity{{Name: "x"}, {Name: "y"}, {Name: "z"}}}
	normalizeEntityNumbers(edd)
	for i, want := range []string{"100", "200", "300"} {
		if got := edd.Entities[i].Number; got != want {
			t.Errorf("entity %d: number = %q, want %q", i, got, want)
		}
	}
}

func TestNormalizeTableNumbers_Backfill(t *testing.T) {
	tables := &DecisionTablesXML{Tables: []DecisionTableXML{
		{TableName: "First", AttributeFields: AttributeFieldsXML{TableNumber: "100"}},
		{TableName: "NoNumber"},
		{TableName: "High", AttributeFields: AttributeFieldsXML{TableNumber: "1550"}},
		{TableName: "AlsoMissing"},
	}}
	normalizeTableNumbers(tables)

	// Continues above 1550, aligned to the 100 grid: 1600, 1700.
	want := map[string]string{"First": "100", "NoNumber": "1600", "High": "1550", "AlsoMissing": "1700"}
	for i := range tables.Tables {
		name := tables.Tables[i].TableName
		if got := tables.Tables[i].AttributeFields.TableNumber; got != want[name] {
			t.Errorf("table %s: number = %q, want %q", name, got, want[name])
		}
	}
}
