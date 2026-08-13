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

package el

import (
	"strings"
	"testing"
)

// The fold family stopped at `sum of`, so "the best of a set of options" --
// the shape of most advice rules -- had to be a host-side loop even when the
// criterion was pure policy (#1024).

func compileCond(t *testing.T, src string) string {
	t.Helper()
	out, err := NewCompiler().CompileCondition(src)
	if err != nil {
		t.Fatalf("compiling %q: %v", src, err)
	}
	return out
}

func TestMaxOfArrayCompiles(t *testing.T) {
	got := compileCond(t, "max of card.rank in stack.cards == 5")
	for _, want := range []string{"0", "card.rank", "max", "stack.cards", "forall"} {
		if !strings.Contains(got, want) {
			t.Errorf("postfix missing %q: %s", want, got)
		}
	}
}

func TestMinOfArrayCompiles(t *testing.T) {
	if got := compileCond(t, "min of card.rank in stack.cards == 5"); !strings.Contains(got, "min") {
		t.Errorf("postfix does not fold with min: %s", got)
	}
}

// The where form guards the body with `if`, inner block before the predicate,
// because `if` pops the boolean off the top.
func TestMaxOfArrayWhereCompiles(t *testing.T) {
	got := compileCond(t, "max of card.rank in stack.cards where card.suit == 1 == 5")
	if !strings.Contains(got, "if") {
		t.Errorf("the where clause did not lower to a guarded body: %s", got)
	}
	if strings.Index(got, "max") > strings.Index(got, "if") {
		t.Errorf("the accumulating operator must be inside the guarded block: %s", got)
	}
}

// `maximum of a and b` is two scalars and must keep lexing as MAXIMUM: 'max'
// requires whitespace after it, so the longer keyword cannot be split.
func TestScalarMaximumStillParses(t *testing.T) {
	if got := compileCond(t, "the maximum of taxpayer.a and taxpayer.b == 5"); !strings.Contains(got, "max") {
		t.Errorf("the scalar form stopped compiling: %s", got)
	}
	if strings.Contains(compileCond(t, "the maximum of taxpayer.a and taxpayer.b == 5"), "forall") {
		t.Error("the scalar form was folded over an array")
	}
}
