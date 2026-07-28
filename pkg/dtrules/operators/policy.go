// Copyright 2004-2011 DTRules.com, Inc.
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

package operators

import (
	"github.com/DTRules/DTRules/pkg/dtrules"
)

func init() {
	Register("policystatements", opPolicyStatements)
}

// policyStatementTable is the slice of RDecisionTable the operator needs. It
// is declared structurally so the operators package does not have to import
// decisiontable (which reaches operators through the interpreter).
type policyStatementTable interface {
	// GetPolicyStatements returns the authored statement text per column,
	// indexed by 1-based column number (index 0 unused).
	GetPolicyStatements() []string
	// GetRPolicyStatements returns the compiled form of the same slice.
	GetRPolicyStatements() []dtrules.Object
}

// firedColumns is the slice of ANode the operator needs: which rule columns
// led to the action node currently executing.
type firedColumns interface {
	// Columns returns the 1-based rule-column numbers of this node.
	Columns() []int
}

// opPolicyStatements: ( -- array ) pushes the policy statements of the columns
// that fired into the action node currently executing.
//
// Each statement is a template compiled to postfix at build time (see
// excel.CompilePolicyStatement), so `{expr}` substitutions are evaluated here
// against live data rather than emitted as literal braces. Statements come out
// in column order; a column with no statement contributes nothing, and a table
// with no statements at all yields an empty array.
//
// The EL form is `policy statements` — an array expression, so it composes
// with the array operators:
//
//	add the policy statements to the job.notes
//	  -> policystatements job.notes swap addto
func opPolicyStatements(state dtrules.State) error {
	arr, err := dtrules.NewArray(state.GetSession(), true, false)
	if err != nil {
		return err
	}

	table, _ := state.GetCurrentTable().(policyStatementTable)
	node, _ := state.GetANode().(firedColumns)
	if table == nil || node == nil {
		// Outside a decision-table action there is no column to speak of.
		// An empty array keeps `add the policy statements to ...` a no-op
		// rather than a crash.
		return state.DataPush(arr)
	}

	text := table.GetPolicyStatements()
	compiled := table.GetRPolicyStatements()

	for _, col := range node.Columns() {
		if col < 0 || col >= len(text) || text[col] == "" {
			continue
		}
		if col < len(compiled) && compiled[col] != nil {
			// Execute directly rather than through State.Evaluate: Evaluate
			// discards whatever the code leaves on the stack, and the value
			// it leaves is the whole point here.
			depth := state.DataStackDepth()
			if err := compiled[col].Execute(state); err != nil {
				return err
			}
			if state.DataStackDepth() > depth {
				value, err := state.DataPop()
				if err != nil {
					return err
				}
				arr.Add(value)
				// Drop anything else the statement left behind so the
				// operator's own net effect stays "push one array".
				for state.DataStackDepth() > depth {
					if _, err := state.DataPop(); err != nil {
						return err
					}
				}
				continue
			}
		}
		// No compiled form (older XML): the authored text is the statement.
		arr.Add(dtrules.NewRString(text[col]))
	}

	return state.DataPush(arr)
}
