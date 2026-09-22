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
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// TestRegexMatchReusesCompiledPattern (#1262): matching the same pattern
// again does not recompile it. Compiling this pattern allocates dozens of
// objects; a cached match allocates only the boolean push's bookkeeping.
func TestRegexMatchReusesCompiledPattern(t *testing.T) {
	op, _ := Get(dtrules.GetRName("regexmatch"))
	state := newTestState()
	subject := dtrules.NewRString("e11.4362a")
	pattern := dtrules.NewRString(`^[A-Za-z0-9._-]+(\.[0-9]+)?[a-z]*$`)
	match := func() {
		state.DataPush(subject)
		state.DataPush(pattern)
		if err := op.Execute(state); err != nil {
			t.Fatal(err)
		}
		if _, err := state.DataPop(); err != nil {
			t.Fatal(err)
		}
	}
	match() // first call may compile
	if allocs := testing.AllocsPerRun(100, match); allocs > 4 {
		t.Errorf("regexmatch allocates %.0f objects per call on a repeated pattern; it is recompiling", allocs)
	}
}
