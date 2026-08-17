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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/operators"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// Dynamic dispatch forces a table into existence for every value the selector
// can take. CorporateTax dispatches on `apportionment.state_code` across 51
// states, and the states with no corporate income tax still need all three
// tables so the name resolves — SD's and WY's are seven action rows each,
// every one writing an audit line and zeroing a field.
//
// `perform table named (<expr>) with default <Table>` says where to go when
// the computed name is not there, so those tables need not exist (#776).

const dispatchEDD = `<entity_data_dictionary version="2">
<file_path>dispatch_edd</file_path>
<entity name="result" number="100" access="rw">
<field name="hit" type="integer" subtype="" access="rw" input="" default_value="0" comment="which table ran"></field>
</entity>
</entity_data_dictionary>`

const dispatchDT = `<decision_tables>
<decision_table><table_name>Handle_AA</table_name>
<initial_actions><initial_action><action_comment>mark</action_comment>
<initial_action_dsl>set result.hit = 1</initial_action_dsl>
<initial_action_postfix>1 /result.hit xdef</initial_action_postfix>
</initial_action></initial_actions>
<conditions></conditions><actions></actions></decision_table>
<decision_table><table_name>Handle_Default</table_name>
<initial_actions><initial_action><action_comment>mark</action_comment>
<initial_action_dsl>set result.hit = 2</initial_action_dsl>
<initial_action_postfix>2 /result.hit xdef</initial_action_postfix>
</initial_action></initial_actions>
<conditions></conditions><actions></actions></decision_table>
</decision_tables>`

// runDispatch executes `<name> /Handle_Default performtableordefault` and
// reports the marker the table that ran left on result.hit.
func runDispatch(t *testing.T, name, defaultName string) (int, error) {
	t.Helper()
	dir := t.TempDir()
	edd := filepath.Join(dir, "dispatch_edd.xml")
	dt := filepath.Join(dir, "dispatch_dt.xml")
	if err := os.WriteFile(edd, []byte(dispatchEDD), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dt, []byte(dispatchDT), 0o644); err != nil {
		t.Fatal(err)
	}

	rs := session.NewRuleSet("dispatch")
	if err := rs.LoadEDDFile(edd); err != nil {
		t.Fatalf("load edd: %v", err)
	}
	if err := rs.LoadDecisionTablesTolerantFile(dt); err != nil {
		t.Fatalf("load dt: %v", err)
	}
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	state := sess.GetState()

	result, err := sess.CreateEntity(dtrules.GetRName("result"))
	if err != nil {
		t.Fatalf("create result: %v", err)
	}
	state.EntityPush(result)

	state.DataPush(dtrules.GetRString(name))
	state.DataPush(dtrules.GetRString(defaultName))
	op, ok := operators.GetByString("performtableordefault")
	if !ok {
		t.Fatal("performtableordefault is not registered")
	}
	if err := op.Execute(state); err != nil {
		return 0, err
	}
	v, err := result.Get(dtrules.GetRName("hit"))
	if err != nil {
		return 0, err
	}
	return v.IntValue()
}

func TestDispatchFindsTheNamedTable(t *testing.T) {
	got, err := runDispatch(t, "Handle_AA", "Handle_Default")
	if err != nil {
		t.Fatalf("dispatch to an existing table failed: %v", err)
	}
	if got != 1 {
		t.Errorf("marker %d — the named table did not run", got)
	}
}

func TestDispatchFallsBackWhenTheTableIsAbsent(t *testing.T) {
	// No Handle_ZZ exists: the state nobody wrote a table for.
	got, err := runDispatch(t, "Handle_ZZ", "Handle_Default")
	if err != nil {
		t.Fatalf("dispatch should have fallen back to the default: %v", err)
	}
	if got != 2 {
		t.Errorf("marker %d — the default did not run", got)
	}
}

// Silence is the one thing this must not do.
func TestDispatchErrorNamesBothWhenNeitherExists(t *testing.T) {
	_, err := runDispatch(t, "Handle_ZZ", "No_Such_Default")
	if err == nil {
		t.Fatal("with neither table present this must fail, not run nothing")
	}
	for _, want := range []string{"Handle_ZZ", "No_Such_Default"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should name %q so the author sees what was looked for: %v", want, err)
		}
	}
}
