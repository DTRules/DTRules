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
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

const sortFieldEDD = `<entity_data_dictionary version="2">
<file_path>sortfield_edd</file_path>
<entity name="node" number="100" access="rw">
<field name="val" type="integer" subtype="" access="rw" input="" default_value="0" comment="sort key"></field>
</entity>
<entity name="state" number="200" access="rw">
<field name="nodes" type="array" subtype="node" access="rw" input="" default_value="" comment="the list"></field>
</entity>
</entity_data_dictionary>`

func sortFieldTable(name, dsl, postfix string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return `<decision_table><table_name>` + name + `</table_name>
<initial_actions><initial_action><action_comment>sort</action_comment>
<initial_action_dsl>` + r.Replace(dsl) + `</initial_action_dsl>
<initial_action_postfix>` + r.Replace(postfix) + `</initial_action_postfix>
</initial_action></initial_actions>
<conditions></conditions><actions></actions></decision_table>
`
}

// sortentities takes the field as a literal name, /val — the non-executable
// form. Attributes are keyed by the executable form, so looking the field up
// with the literal as given found nothing, and comparing the missing value
// dereferenced nil and crashed the process.
func TestSortEntities_LiteralFieldName(t *testing.T) {
	dt := "<decision_tables>\n" +
		sortFieldTable("Sort_Up", "sort state.nodes by val ascending", "state.nodes /val true sortentities") +
		sortFieldTable("Sort_Down", "sort state.nodes by val descending", "state.nodes /val false sortentities") +
		sortFieldTable("Sort_Missing", "sort state.nodes by nosuch", "state.nodes /nosuch true sortentities") +
		"</decision_tables>"

	setup := func(t *testing.T) (*session.RSession, *dtrules.RArray) {
		t.Helper()
		rs := session.NewRuleSet("sortfield")
		if err := rs.LoadEDD(strings.NewReader(sortFieldEDD)); err != nil {
			t.Fatal(err)
		}
		if err := rs.LoadDecisionTables(strings.NewReader(dt)); err != nil {
			t.Fatal(err)
		}
		s, err := rs.NewSession()
		if err != nil {
			t.Fatal(err)
		}
		sess := s.(*session.RSession)
		st, _ := sess.CreateEntity(dtrules.GetRName("state"))
		nodes, _ := dtrules.NewArray(sess, true, false)
		for _, v := range []int64{2, 3, 1} {
			n, _ := sess.CreateEntity(dtrules.GetRName("node"))
			n.Put(dtrules.GetRName("val"), dtrules.GetRIntegerValue(v))
			nodes.Add(n.(dtrules.Object))
		}
		st.Put(dtrules.GetRName("nodes"), nodes)
		sess.GetState().EntityPush(st)
		return sess, nodes
	}
	vals := func(nodes *dtrules.RArray) []int {
		var out []int
		for _, o := range nodes.GetIterator() {
			e, _ := o.REntityValue()
			v, _ := e.Get(dtrules.GetRName("val"))
			n, _ := v.IntValue()
			out = append(out, n)
		}
		return out
	}

	for _, tc := range []struct {
		table string
		want  string
	}{{"Sort_Up", "[1 2 3]"}, {"Sort_Down", "[3 2 1]"}} {
		t.Run(tc.table, func(t *testing.T) {
			sess, nodes := setup(t)
			if err := sess.Execute(tc.table); err != nil {
				t.Fatal(err)
			}
			if got := fmt.Sprint(vals(nodes)); got != tc.want {
				t.Errorf("sorted %s, want %s", got, tc.want)
			}
		})
	}
	t.Run("Sort_Missing", func(t *testing.T) {
		sess, _ := setup(t)
		err := sess.Execute("Sort_Missing")
		if err == nil || !strings.Contains(err.Error(), "nosuch") {
			t.Errorf("sorting by a field the entities lack must be an error naming it, got %v", err)
		}
	})
}
