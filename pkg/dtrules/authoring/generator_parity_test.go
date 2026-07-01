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

package authoring_test

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
	"github.com/DTRules/DTRules/pkg/dtrules/compiler/el"
	"github.com/DTRules/DTRules/pkg/dtrules/excel"
)

const parityEDD = `<entity_data_dictionary version='2'>
	<entity name='budget'>
		<field name='supply_limit' type='fixed'></field>
		<field name='acme_issued' type='fixed'></field>
		<field name='rate' type='double'></field>
		<field name='count' type='integer'></field>
		<field name='name' type='string'></field>
		<field name='start' type='date'></field>
	</entity>
</entity_data_dictionary>`

// parityCorpus are DSL snippets whose postfix depends on operand types, so they
// exercise the symbol map. Bare and entity-qualified references are both used.
var parityCorpus = []string{
	"supply_limit >= acme_issued",               // fixed comparison -> fp>=
	"budget.supply_limit >= budget.acme_issued", // entity-qualified keys
	"rate > 1.0",                                // double comparison -> f>
	"count > 1",                                 // integer comparison
	"supply_limit != acme_issued",               // fixed inequality -> fp!=
}

// TestGeneratorParity (#898) pins that the two independent EDD→symbol builders
// agree: the file-based authoring.LoadEDDSymbols (used by authoring Save and
// `dtrules build`) and the content-based excel.EDDSymbols (workbook import).
// It asserts identical symbol maps AND identical compiled postfix for a DSL
// corpus — the guard that would have caught #874 (bare-key divergence).
func TestGeneratorParity(t *testing.T) {
	dir := t.TempDir()
	xmlDir := filepath.Join(dir, "xml")
	if err := os.MkdirAll(xmlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	eddPath := filepath.Join(xmlDir, "budget_edd.xml")
	if err := os.WriteFile(eddPath, []byte(parityEDD), 0o644); err != nil {
		t.Fatal(err)
	}

	// File-based builder (authoring Save + build).
	fileSyms := authoring.LoadEDDSymbols(xmlDir)

	// Content-based builder (workbook import), from the same EDD parsed.
	var edd excel.EDDXML
	if err := xml.Unmarshal([]byte(parityEDD), &edd); err != nil {
		t.Fatalf("parse EDD: %v", err)
	}
	contentSyms := excel.EDDSymbols(&edd)

	if !reflect.DeepEqual(fileSyms, contentSyms) {
		t.Fatalf("symbol maps diverge:\n file    = %v\n content = %v", fileSyms, contentSyms)
	}

	// End-to-end: identical postfix for every snippet under each map.
	compile := func(syms map[string]string, dsl string) string {
		c := el.NewCompiler()
		c.SetSymbols(syms)
		pf, err := c.CompileCondition(dsl)
		if err != nil {
			t.Fatalf("compile %q: %v", dsl, err)
		}
		return pf
	}
	for _, dsl := range parityCorpus {
		a := compile(fileSyms, dsl)
		b := compile(contentSyms, dsl)
		if a != b {
			t.Errorf("postfix diverges for %q:\n file    = %q\n content = %q", dsl, a, b)
		}
	}
}
