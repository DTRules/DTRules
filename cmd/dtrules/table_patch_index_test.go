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

package main

import (
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

// `index` is 1-based, like action_number and condition_number (#1297). It was
// 0-based, so a caller using the same numbering as every other op rewrote the
// *next* row and got `"status": "patched"` back — that is how
// `set street = using client (street)` was destroyed in SyntaxTests.

func TestSlotIndexIsOneBased(t *testing.T) {
	for _, c := range []struct {
		in   int
		want int
	}{{1, 0}, {2, 1}, {3, 2}, {99, 98}} {
		got, err := slotIndex(c.in, "update-initial-action")
		if err != nil {
			t.Fatalf("index %d: %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("index %d -> slot %d, want %d", c.in, got, c.want)
		}
	}
}

// `index` is omitempty, so a payload that forgot the field and one that asks
// for slot 0 arrive identically. Both are refused, and the error states the
// numbering rather than quietly writing to the first row.
func TestSlotIndexRejectsZeroAndNegative(t *testing.T) {
	for _, in := range []int{0, -1} {
		_, err := slotIndex(in, "delete-context")
		if err == nil {
			t.Fatalf("index %d must be refused", in)
		}
		if !strings.Contains(err.Error(), "1-based") {
			t.Errorf("index %d: the error should state the numbering, got: %v", in, err)
		}
		if !strings.Contains(err.Error(), "delete-context") {
			t.Errorf("index %d: the error should name the op, got: %v", in, err)
		}
	}
}

// Every op that takes `index` goes through the conversion: with the field
// missing, each is refused before it touches the table.
func TestIndexOpsRefuseAMissingIndex(t *testing.T) {
	dsl := "set x = 1"
	for _, op := range []string{
		"update-initial-action", "delete-initial-action",
		"update-context", "delete-context",
	} {
		t.Run(op, func(t *testing.T) {
			tbl := &authoring.Table{
				Name:           "T",
				InitialActions: []authoring.InitialAction{{DSL: "first"}, {DSL: "second"}},
				Contexts:       []authoring.Context{{DSL: "ctx one"}, {DSL: "ctx two"}},
			}
			p := tablePatch{Op: op, DSL: &dsl} // Index left at its zero value
			err := p.apply(nil, tbl)
			if err == nil {
				t.Fatalf("%s with no index must be refused", op)
			}
			if !strings.Contains(err.Error(), "1-based") {
				t.Errorf("the error should state the numbering, got: %v", err)
			}
			if len(tbl.InitialActions) != 2 || tbl.InitialActions[0].DSL != "first" {
				t.Errorf("a refused patch changed the initial actions: %+v", tbl.InitialActions)
			}
			if len(tbl.Contexts) != 2 || tbl.Contexts[0].DSL != "ctx one" {
				t.Errorf("a refused patch changed the contexts: %+v", tbl.Contexts)
			}
		})
	}
}
