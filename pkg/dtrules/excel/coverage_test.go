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

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// Exports build from a tolerantly-loaded rule set, so a table that will not
// load is skipped and the workbook written without it. While Excel merely sat
// beside the XML that was an incomplete artifact; once Excel is the system of
// record and the XML is generated from it, the same skip deletes rules
// (#1081).

const twoTableDT = `<decision_tables>
<decision_table><table_name>Alpha</table_name>
<attribute_fields><Type>FIRST</Type><TABLE_NUMBER>1</TABLE_NUMBER></attribute_fields>
<contexts></contexts><initial_actions></initial_actions>
<conditions></conditions><actions></actions>
</decision_table>
<decision_table><table_name>Beta</table_name>
<attribute_fields><Type>FIRST</Type><TABLE_NUMBER>2</TABLE_NUMBER></attribute_fields>
<contexts></contexts><initial_actions></initial_actions>
<conditions></conditions><actions></actions>
</decision_table>
</decision_tables>`

func writeDT(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "P_dt.xml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestExportRefusedWhenATableDidNotLoad(t *testing.T) {
	path := writeDT(t, twoTableDT)

	// A rule set that got only one of the two -- the shape a tolerant load
	// leaves behind when a table fails.
	rs := session.NewRuleSet("partial")
	if err := rs.LoadDecisionTablesTolerant(strings.NewReader(
		strings.Replace(twoTableDT, `<decision_table><table_name>Beta</table_name>
<attribute_fields><Type>FIRST</Type><TABLE_NUMBER>2</TABLE_NUMBER></attribute_fields>
<contexts></contexts><initial_actions></initial_actions>
<conditions></conditions><actions></actions>
</decision_table>
`, "", 1))); err != nil {
		t.Fatalf("load: %v", err)
	}

	err := AssertRuleSetCovers(rs, path)
	if err == nil {
		t.Fatal("a rule set missing a table the XML declares must refuse the " +
			"export; writing the workbook without it deletes the rule on the " +
			"next build")
	}
	if !strings.Contains(err.Error(), "beta") {
		t.Errorf("the error must name what is missing, got: %v", err)
	}
	if !strings.Contains(err.Error(), "system of record") {
		t.Errorf("the error should say why this is fatal, got: %v", err)
	}
}

func TestExportProceedsWhenEveryTableLoaded(t *testing.T) {
	path := writeDT(t, twoTableDT)

	rs := session.NewRuleSet("full")
	if err := rs.LoadDecisionTablesTolerant(strings.NewReader(twoTableDT)); err != nil {
		t.Fatalf("load: %v", err)
	}

	if err := AssertRuleSetCovers(rs, path); err != nil {
		t.Fatalf("every declared table loaded, so the export should proceed: %v", err)
	}
}

// Case is display-only in DTRules: two spellings are one name, so a rule set
// holding "alpha" covers an XML declaring "Alpha".
func TestCoverageIsCaseInsensitive(t *testing.T) {
	path := writeDT(t, strings.ReplaceAll(twoTableDT, "Alpha", "ALPHA"))

	rs := session.NewRuleSet("mixed")
	if err := rs.LoadDecisionTablesTolerant(strings.NewReader(twoTableDT)); err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := AssertRuleSetCovers(rs, path); err != nil {
		t.Errorf("ALPHA and Alpha are one name; coverage must not depend on "+
			"spelling: %v", err)
	}
}

// A path that is not there declares nothing, and an empty project is not a
// failure -- otherwise bootstrapping a project with no XML yet would refuse.
func TestNoDeclaredTablesIsNotAFailure(t *testing.T) {
	rs := session.NewRuleSet("empty")
	if err := AssertRuleSetCovers(rs, "/nonexistent/P_dt.xml"); err != nil {
		t.Errorf("nothing declared means nothing to lose: %v", err)
	}
}
