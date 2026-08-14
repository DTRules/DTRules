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

package operators

import (
	"fmt"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// argmax / argmin — which element wins, not what the winning value is.
//
// `max of <field> in <array>` gives the value. The piece that keeps a choice
// rule inside the tables is the element that attains it, because the rule then
// goes on to read that element's other fields: the highest-EV discard, the most
// favourable filing status, the cheapest qualifying plan. Without it the table
// can score the options and not say which one won, and the choice moves to host
// code even when the criterion is pure policy (#1024).
//
// Statement-form EL calls, in the shape the other selection operators use:
//
//	argmax(hand.cards, "rank", hand.best)
//
// which compiles to `hand.cards "rank" hand.best argmax`.
//
// The destination is an array, matching the generators, and it receives the
// winning element itself rather than a copy — reading a field off the result
// reads the original entity. It is cleared first, so a table that runs twice
// does not accumulate winners.
//
// Ties go to the first element in array order. That is a decision, not an
// accident: it makes the result stable under re-running the same rules on the
// same data, which matters more for an advice rule than any rule about which
// of two equal options is better.
//
// An empty source leaves the destination empty rather than failing. "No options
// to choose between" is a state a rule can test with `number of`, and a table
// that has already established there are options should not have to defend
// against the operator throwing.
func init() {
	RegisterWithArity("argmax", opArgMax, 3) // source, field, dest
	RegisterWithArity("argmin", opArgMin, 3) // source, field, dest
}

func opArgMax(state dtrules.State) error { return argSelect(state, "argmax", true) }
func opArgMin(state dtrules.State) error { return argSelect(state, "argmin", false) }

// argSelect: ( src field dest -- ) put the element of src with the largest
// (or smallest) value of the named integer field into dest.
func argSelect(state dtrules.State, op string, wantMax bool) error {
	destObj, err := state.DataPop()
	if err != nil {
		return err
	}
	dest, err := destObj.RArrayValue()
	if err != nil {
		return fmt.Errorf("%s: dest must be an array: %w", op, err)
	}
	fieldText, err := popString(state, op, "field")
	if err != nil {
		return err
	}
	_, ents, err := popEntityArray(state, op)
	if err != nil {
		return err
	}

	// Clear first: a table that runs twice must select, not accumulate.
	dest.Clear()
	if len(ents) == 0 {
		return nil
	}

	field := dtrules.GetRName(fieldText)
	if field == nil {
		return fmt.Errorf("%s: invalid field name syntax: %q", op, fieldText)
	}

	best := 0
	bestSet := false
	var winner dtrules.Entity
	for _, ent := range ents {
		v, err := intField(op, ent, field)
		if err != nil {
			return err
		}
		// Strictly better only, so the first of equal values keeps the win.
		if !bestSet || (wantMax && v > best) || (!wantMax && v < best) {
			best, bestSet, winner = v, true, ent
		}
	}
	dest.Add(winner.(dtrules.Object))
	dtrules.TraceArrayAdd(state, dest, winner.(dtrules.Object))
	return nil
}
