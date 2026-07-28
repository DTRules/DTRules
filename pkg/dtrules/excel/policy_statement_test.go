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

package excel

import "testing"

func TestCompilePolicyStatement(t *testing.T) {
	tests := []struct {
		name string
		desc string
		want string
	}{
		{
			name: "plain literal",
			desc: "thing.value == 1",
			want: `"thing.value == 1"`,
		},
		{
			name: "empty",
			desc: "",
			want: `""`,
		},
		{
			// The shape the legacy (Java) compiler emitted; TestProject's
			// checked-in XML carries exactly this string.
			name: "trailing substitution",
			desc: "thing.value is out of range, i.e.  {thing.value}",
			want: `"thing.value is out of range, i.e.  " thing.value cvs strconcat "" strconcat`,
		},
		{
			name: "substitution in the middle",
			desc: "income {job.income} exceeds the limit",
			want: `"income " job.income cvs strconcat " exceeds the limit" strconcat`,
		},
		{
			name: "two substitutions",
			desc: "{job.name} owes {job.tax}",
			want: `"" job.name cvs strconcat " owes " strconcat job.tax cvs strconcat "" strconcat`,
		},
		{
			name: "braces are trimmed",
			desc: "value is { thing.value }",
			want: `"value is " thing.value cvs strconcat "" strconcat`,
		},
		{
			name: "unterminated brace stays literal",
			desc: "a { b",
			want: `"a { b"`,
		},
		{
			name: "empty group stays literal",
			desc: "a {} b",
			want: `"a {} b"`,
		},
		{
			name: "embedded quote is escaped",
			desc: `say "hi"`,
			want: `"say \"hi\""`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CompilePolicyStatement(tt.desc); got != tt.want {
				t.Errorf("CompilePolicyStatement(%q):\n got %s\nwant %s", tt.desc, got, tt.want)
			}
		})
	}
}

func TestIsColumnNumberRow(t *testing.T) {
	tests := []struct {
		name string
		row  []string
		want bool
	}{
		{"bare column numbers", []string{"", "", "", "1", "2", "3"}, true},
		{"policy data row", []string{"1", "thing.value == 1", "", "", ""}, false},
		{"old-format statements", []string{"", "Column Policy", "", "a", "b"}, false},
		{"empty row", []string{"", "", ""}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isColumnNumberRow(tt.row); got != tt.want {
				t.Errorf("isColumnNumberRow(%q) = %v, want %v", tt.row, got, tt.want)
			}
		})
	}
}

// TestParseExporterFormatKeepsFirstPolicyStatement guards the round-trip
// off-by-one: the exporter writes the section title, the "Policy Statements"
// label, and the rule-column numbers on ONE row, so the row after it is data.
// Routing through a separate header state consumed it and column 1's policy
// statement disappeared on every Excel→XML build.
func TestParseExporterFormatKeepsFirstPolicyStatement(t *testing.T) {
	rows := [][]string{
		{"DT: Test Entry Point"},
		{"Type: FIRST"},
		{"COMMENTS: "},
		{"TABLE_NUMBER: "},
		{},
		{"CONDITIONS: COMMENTS", "", "DSL: CONDITIONS", "1", "2"},
		{"1", "", "thing.value == 1", "Y", ""},
		{"2", "", "otherwise", "", "*"},
		{},
		{"ACTIONS: COMMENTS", "", "DSL: ACTIONS", "1", "2"},
		{"1", "", "add the policy statements to the job.notes", "X", "X"},
		{},
		{"Policy:", "", "Policy Statements", "1", "2"},
		{"1", "thing.value == 1", "", "", ""},
		{"2", "thing.value is out of range, i.e.  {thing.value}", "", "", ""},
	}

	imp := NewDTImporter()
	table, err := imp.parseExporterFormat(rows, "Test Entry Point", &DecisionTableXML{})
	if err != nil {
		t.Fatalf("parseExporterFormat: %v", err)
	}

	if len(table.PolicyStatements) != 2 {
		t.Fatalf("got %d policy statements, want 2: %+v", len(table.PolicyStatements), table.PolicyStatements)
	}
	if got := table.PolicyStatements[0].Column; got != "1" {
		t.Errorf("first policy statement is for column %s, want 1", got)
	}
	if got := table.PolicyStatements[0].Description; got != "thing.value == 1" {
		t.Errorf("first policy description = %q", got)
	}
	want := `"thing.value is out of range, i.e.  " thing.value cvs strconcat "" strconcat`
	if got := table.PolicyStatements[1].Postfix; got != want {
		t.Errorf("interpolated policy postfix:\n got %s\nwant %s", got, want)
	}
}

// TestParseExporterFormatSkipsLegacyColumnNumberRow covers the two-row layout
// (title row, then a bare column-number row) that the straight-to-data fix
// must not mistake for policy content.
func TestParseExporterFormatSkipsLegacyColumnNumberRow(t *testing.T) {
	rows := [][]string{
		{"DT: Test Entry Point"},
		{"Policy:", "", "Policy Statements"},
		{"", "", "", "1", "2", "3"},
		{"1", "first statement", "", "", ""},
	}

	imp := NewDTImporter()
	table, err := imp.parseExporterFormat(rows, "Test Entry Point", &DecisionTableXML{})
	if err != nil {
		t.Fatalf("parseExporterFormat: %v", err)
	}
	if len(table.PolicyStatements) != 1 {
		t.Fatalf("got %d policy statements, want 1: %+v", len(table.PolicyStatements), table.PolicyStatements)
	}
	if got := table.PolicyStatements[0].Description; got != "first statement" {
		t.Errorf("policy description = %q, want %q", got, "first statement")
	}
}
