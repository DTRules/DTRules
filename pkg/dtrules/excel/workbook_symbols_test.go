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

// A multi-file project's decision tables read entities declared in other
// workbooks. CorporateTax is the shape: 51 state workbooks whose tables all
// read the shared `apportionment` entity, which is declared once in the core
// workbook.
//
// The symbol table decides the arithmetic. A field the compiler cannot type is
// assumed to be an integer, so `apportionment.state_credits f-` compiles to
// `-` and `cvd` to `cvi`: money subtracted and stored as whole units, with no
// error and no warning. That is a wrong answer, not a rounded one (#1094).

func ownEDD() *EDDXML {
	return &EDDXML{Entities: []*EDDXMLEntity{{
		Name:   "sc_apportionment",
		Fields: []*EDDXMLField{{Name: "state_tax_rate", Type: "double"}},
	}}}
}

// The project-wide types must survive a workbook that carries its own EDD.
func TestWorkbookSymbolsKeepTheRestOfTheProject(t *testing.T) {
	w := NewWorkbookImporter()
	w.SetSymbols(map[string]string{
		"apportionment.state_credits": "double",
		"state_credits":               "double",
	})

	got := w.symbolsFor(ownEDD())

	if got["apportionment.state_credits"] != "double" {
		t.Errorf("a shared entity from another workbook lost its type (got %q); "+
			"its arithmetic compiles as integer", got["apportionment.state_credits"])
	}
	if got["sc_apportionment.state_tax_rate"] != "double" {
		t.Error("the workbook's own EDD is missing from the symbol table")
	}
}

// The workbook that ships an entity is the authority on it.
func TestWorkbookOwnEDDWinsOnConflict(t *testing.T) {
	w := NewWorkbookImporter()
	w.SetSymbols(map[string]string{"sc_apportionment.state_tax_rate": "integer"})

	if got := w.symbolsFor(ownEDD())["sc_apportionment.state_tax_rate"]; got != "double" {
		t.Errorf("own EDD did not win: got %q, want double", got)
	}
}

// One importer is reused for every workbook in the build loop, with no
// save/restore around the per-workbook layer. When that layer was written over
// the project map, it leaked forward: a DT-only workbook compiled against
// whichever EDD happened to be imported before it.
func TestWorkbookSymbolsDoNotLeakToTheNextWorkbook(t *testing.T) {
	w := NewWorkbookImporter()
	w.SetSymbols(map[string]string{"apportionment.state_credits": "double"})

	w.symbolsFor(ownEDD()) // a workbook with its own EDD

	// The next workbook has no EDD sheet of its own and falls back.
	next := w.symbolsFor(&EDDXML{})
	if next["apportionment.state_credits"] != "double" {
		t.Error("the project-wide map was consumed by the previous workbook")
	}
	if _, leaked := next["sc_apportionment.state_tax_rate"]; leaked {
		t.Error("the previous workbook's entities leaked into this one")
	}
}
