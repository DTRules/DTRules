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

package dtrules_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/decisiontable"
	"github.com/DTRules/DTRules/pkg/dtrules/excel"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// actionList pulls "1, 3" out of "... action(s) 1, 3 marked ...".
var actionList = regexp.MustCompile(`actions? ([0-9, ]+) marked`)

// TestNoConditionWarningMatchesTheEngine (#1230): the advisory pass says which
// column actions of a table with no conditions never run. That claim is only
// worth making if it is what the engine does, and the engine differs by
// policy: FIRST and ALL build no tree and run no column; BALANCED -- also
// what an empty Type loads as -- runs column 1 and nothing else. Each table is
// loaded through the real XML loader and executed in a session, then the
// same XML goes through the advisory pass the way `build`/`review` feed it,
// and the two must agree action by action.
func TestNoConditionWarningMatchesTheEngine(t *testing.T) {
	const edd = `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
	<entity name="probe" access="rw" comment="">
		<field name="probe" type="entity" subtype="" access="r" input="" default_value="" comment=""></field>
		<field name="a1" type="boolean" subtype="" access="rw" input="" default_value="false" comment=""></field>
		<field name="a2" type="boolean" subtype="" access="rw" input="" default_value="false" comment=""></field>
		<field name="a3" type="boolean" subtype="" access="rw" input="" default_value="false" comment=""></field>
	</entity>
</entity_data_dictionary>`
	// action 1 in column 1; action 2 in column 2; action 3 in both.
	action := func(n int, cols string) string {
		var cells strings.Builder
		for i, c := range cols {
			if c == 'X' {
				fmt.Fprintf(&cells, `<action_column column_number="%d" column_value="X"></action_column>`, i+1)
			}
		}
		return fmt.Sprintf(`<action_details><action_number>%d</action_number>
<action_dsl>set probe.a%d = true</action_dsl><action_postfix>true /probe.a%d xdef</action_postfix>
%s</action_details>`, n, n, n, cells.String())
	}
	table := func(policy string) string {
		return `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables><decision_table>
<table_name>NoCond</table_name>
<attribute_fields><Type>` + policy + `</Type><TABLE_NUMBER>100</TABLE_NUMBER></attribute_fields>
<contexts></contexts><initial_actions></initial_actions>
<conditions></conditions>
<actions>` + action(1, "X-") + action(2, "-X") + action(3, "XX") + `</actions>
<policy_statements></policy_statements>
</decision_table></decision_tables>`
	}

	for _, policy := range []string{"", "BALANCED", "FIRST", "ALL"} {
		t.Run("Type="+policy, func(t *testing.T) {
			xml := table(policy)

			// What the engine does.
			rs := session.NewRuleSet("nocond")
			if err := rs.LoadEDD(strings.NewReader(edd)); err != nil {
				t.Fatal(err)
			}
			if err := rs.LoadDecisionTables(strings.NewReader(xml)); err != nil {
				t.Fatal(err)
			}
			sess, err := rs.NewSession()
			if err != nil {
				t.Fatal(err)
			}
			probe, err := sess.(*session.RSession).CreateEntity(dtrules.GetRName("probe"))
			if err != nil {
				t.Fatal(err)
			}
			state := sess.GetState()
			state.EntityPush(probe)
			dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("NoCond"))
			if err != nil {
				t.Fatal(err)
			}
			if err := dt.Execute(state); err != nil {
				t.Fatal(err)
			}
			ran := map[int]bool{}
			for n := 1; n <= 3; n++ {
				v, _ := probe.Get(dtrules.GetRName(fmt.Sprintf("a%d", n)))
				ran[n], _ = v.BooleanValue()
			}

			// What the advisory pass says, fed as build/review feed it.
			parsed, err := excel.UnmarshalDecisionTablesXML([]byte(xml))
			if err != nil {
				t.Fatal(err)
			}
			tx := &parsed.Tables[0]
			width := excel.TableWidth(tx)
			in := decisiontable.Inputs{Name: tx.TableName, Policy: tx.AttributeFields.Type, MaxCol: width}
			for i := range tx.Actions {
				in.Actions = append(in.Actions, decisiontable.ActionRow{DSL: tx.Actions[i].DSL, Columns: tx.Actions[i].Row(width)})
			}
			stranded := map[int]bool{}
			for _, w := range decisiontable.Analyze(in) {
				if w.Kind != decisiontable.KindColumnActionsWithoutConditions {
					continue
				}
				m := actionList.FindStringSubmatch(w.Reason)
				if m == nil {
					t.Fatalf("warning names no actions: %q", w.Reason)
				}
				for _, f := range strings.Split(m[1], ",") {
					var n int
					fmt.Sscan(strings.TrimSpace(f), &n)
					stranded[n] = true
				}
			}
			for n := 1; n <= 3; n++ {
				if ran[n] == stranded[n] {
					t.Errorf("action %d: engine ran=%v, advisory says never runs=%v", n, ran[n], stranded[n])
				}
			}
		})
	}
}
