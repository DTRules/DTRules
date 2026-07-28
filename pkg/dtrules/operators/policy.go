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

// opPolicyStatements: ( -- array ) pushes the run's policy-statement
// accumulator — the statement of every column that has fired since the last
// clear, in fire order.
//
// Statements collect on their own as columns fire (see
// ANode.collectPolicyStatements), so this reports what the tables a rule
// performed concluded, not just the column the rule is standing in. That is
// what documents a whole run: evaluate a household member against every
// program, then read back everything those tables concluded for them.
//
// The EL forms:
//
//	add the policy statements to person_report.findings   copy into a report
//	clear the policy statements                           start the next phase
//
// The array is the live accumulator, not a copy, which is what makes `clear`
// work — it compiles to `policystatements cleararray`, and emptying this
// object is the reset.
func opPolicyStatements(state dtrules.State) error {
	return state.DataPush(state.PolicyStatements())
}
