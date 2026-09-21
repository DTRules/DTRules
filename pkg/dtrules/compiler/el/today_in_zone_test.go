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

// `today in zone <s>` used to lower through the generic rewrap, `today <s>
// dateinzone`: UTC midnight shown in the zone, which west of UTC is the
// previous evening. It lowers to `<s> todayinzone`, midnight today in that
// zone (#1273).
func TestTodayInZoneLowering(t *testing.T) {
	cases := []struct{ dsl, want string }{
		{`set d = today in zone "America/Chicago"`, `"America/Chicago" todayinzone cvdate /d xdef`},
		{`set d = Today in zone z`, `z todayinzone cvdate /d xdef`},
		{`set d = today`, `today cvdate /d xdef`},
		// Any other date keeps the rewrap.
		{`set d = e in zone "UTC"`, `e "UTC" dateinzone cvdate /d xdef`},
	}
	for _, c := range cases {
		comp := NewCompiler()
		comp.SetSymbols(map[string]string{"d": TypeDate, "e": TypeDate, "z": TypeString})
		got, err := comp.CompileAction(c.dsl)
		if err != nil {
			t.Fatalf("%s: %v", c.dsl, err)
		}
		if strings.TrimSpace(got) != c.want {
			t.Errorf("%s\n  got  %q\n  want %q", c.dsl, got, c.want)
		}
	}
}

// A field or local named today is the author's, and keeps its meaning.
func TestTodayInZoneYieldsToAFieldOrLocal(t *testing.T) {
	comp := NewCompiler()
	comp.SetSymbols(map[string]string{"d": TypeDate, "today": TypeDate})
	got, err := comp.CompileAction(`set d = today in zone "UTC"`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "dateinzone") || strings.Contains(got, "todayinzone") {
		t.Errorf("EDD field today: got %q, want the field rewrapped", got)
	}

	comp = NewCompiler()
	comp.SetSymbols(map[string]string{"d": TypeDate})
	got, err = comp.CompileAction(`{ local date today = d; set d = today in zone "UTC"; }`)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "todayinzone") {
		t.Errorf("local today: got %q, want the local rewrapped", got)
	}
}
