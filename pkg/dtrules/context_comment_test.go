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

package dtrules_test

import (
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/decisiontable"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

const contextCommentDT = `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
<decision_table>
<table_name>Calculate_Gross_Income</table_name>
<attribute_fields><Type>FIRST</Type><TABLE_NUMBER>100</TABLE_NUMBER></attribute_fields>
<contexts>
<context_details>
<context_number>1</context_number>
<context_comment>Iterate over all income entries</context_comment>
<context_dsl>for all incomes</context_dsl>
<context_postfix>dup incomes forall pop</context_postfix>
</context_details>
</contexts>
<initial_actions></initial_actions>
<conditions>
<condition_details>
<condition_number>1</condition_number>
<condition_comment></condition_comment>
<condition_dsl>true</condition_dsl>
<condition_postfix>true</condition_postfix>
<condition_column column_number="1" column_value="Y" />
</condition_details>
</conditions>
<actions></actions>
<policy_statements></policy_statements>
</decision_table>
</decision_tables>
`

// TestContextCommentReachesTheTable guards a comment that used to be read from
// the XML and then dropped before the table was built. The Excel exporter
// reads context comments off the table, so a table that never received them
// wrote blank comment cells — and the next Excel→XML build stored the blanks,
// erasing every context comment in the project.
func TestContextCommentReachesTheTable(t *testing.T) {
	rs := session.NewRuleSet("ContextComments")
	if err := rs.LoadDecisionTables(strings.NewReader(contextCommentDT)); err != nil {
		t.Fatalf("LoadDecisionTables: %v", err)
	}

	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	dtObj, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Calculate_Gross_Income"))
	if err != nil {
		t.Fatalf("GetDecisionTable: %v", err)
	}
	table, ok := dtObj.(*decisiontable.RDecisionTable)
	if !ok {
		t.Fatalf("got %T, want *decisiontable.RDecisionTable", dtObj)
	}

	comments := table.GetContextsComment()
	if len(comments) != 1 {
		t.Fatalf("got %d context comments, want 1", len(comments))
	}
	if want := "Iterate over all income entries"; comments[0] != want {
		t.Errorf("context comment = %q, want %q", comments[0], want)
	}
}
