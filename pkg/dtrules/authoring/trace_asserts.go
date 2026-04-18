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

package authoring

import "fmt"

// AssertVisited returns an error if no invocation of table was found in the trace.
// If column > 0, the invocation must have been triggered from that specific column.
// If column == 0, any column matches.
func (t *RunTrace) AssertVisited(table string, column int) error {
	for _, inv := range t.Invocations {
		if inv.TableName != table {
			continue
		}
		if column == 0 || inv.Column == column {
			return nil
		}
	}
	if column == 0 {
		return fmt.Errorf("AssertVisited: table %q was not visited", table)
	}
	return fmt.Errorf("AssertVisited: table %q was not visited at column %d", table, column)
}

// AssertNotVisited returns an error if table was invoked at all in the trace.
func (t *RunTrace) AssertNotVisited(table string) error {
	for _, inv := range t.Invocations {
		if inv.TableName == table {
			return fmt.Errorf("AssertNotVisited: table %q was visited (index %d)", table, inv.Index)
		}
	}
	return nil
}

// AssertSequence returns an error if the tables do not appear in the given order
// within the trace. The tables may be non-contiguous but must appear in order.
func (t *RunTrace) AssertSequence(tables []string) error {
	if len(tables) == 0 {
		return nil
	}

	pos := 0 // current scan position in t.Invocations
	for i, want := range tables {
		found := false
		for pos < len(t.Invocations) {
			if t.Invocations[pos].TableName == want {
				pos++
				found = true
				break
			}
			pos++
		}
		if !found {
			if i == 0 {
				return fmt.Errorf("AssertSequence: table %q not found in trace", want)
			}
			prev := tables[i-1]
			return fmt.Errorf("AssertSequence: table %q not found after %q", want, prev)
		}
	}
	return nil
}
