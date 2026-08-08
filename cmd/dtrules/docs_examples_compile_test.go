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

	el "github.com/DTRules/DTRules/pkg/dtrules/compiler/el"
)

// mutatingShortcutExamples are the action forms `dtrules docs operators`
// presents as mutating shortcuts. Every one must compile.
//
// The docs listed eight forms that do not parse (#985): `add to myLong 5` and
// `subtract from myLong 3` have the operand and target the wrong way round,
// and `multiply myLong by 2` / `divide myLong by 4` do not exist at all — in
// any spelling. A rule author following the docs got
// `no viable alternative at input 'addto'` and no hint that the docs were
// wrong rather than their rule.
//
// This is the `dtrules docs` counterpart of the el-reference gate (#961,
// extended in #1021): documentation that shows syntax the compiler rejects is
// worse than no documentation, because the reader trusts it.
var mutatingShortcutExamples = []string{
	"add 5 to myLong",
	"subtract 3 from myLong",
	"increment myLong",
	"decrement myLong",
	"add 1.5 to myDouble",
	"subtract 0.5 from myDouble",
	"set myLong = myLong * 2",
	"set myLong = myLong / 4",
	"set myDouble = myDouble * 1.1",
	"set myDouble = myDouble / 2.0",
}

func TestDocsMutatingShortcutsCompile(t *testing.T) {
	for _, dsl := range mutatingShortcutExamples {
		c := el.NewCompiler()
		c.SetSymbols(map[string]string{"myLong": "long", "myDouble": "double"})
		if _, err := c.CompileAction(dsl); err != nil {
			t.Errorf("`dtrules docs` shows %q but it does not compile:\n  %v", dsl, err)
		}
	}
}

// TestDocsDoNotShowRejectedForms pins the specific forms that were wrong, so
// they cannot come back. Listing them by name is deliberate: a reader who
// remembers the old spelling should find it absent, not silently changed.
func TestDocsDoNotShowRejectedForms(t *testing.T) {
	body := docsOperatorsBody(t)
	for _, dead := range []string{
		"add to myLong",
		"subtract from myLong",
		"multiply myLong by",
		"divide myLong by",
		"add to myDouble",
		"subtract from myDouble",
		"multiply myDouble by",
		"divide myDouble by",
	} {
		if strings.Contains(body, dead) {
			t.Errorf("docs still show %q, which the compiler rejects (#985)", dead)
		}
	}
}

// docsOperatorsBody returns every docs page that carries the mutating
// shortcuts. Both `dtrules docs el` and `dtrules docs operators` listed them,
// and only one of the two was reported — fixing a page at a time is how the
// other survives.
func docsOperatorsBody(t *testing.T) string {
	t.Helper()
	body := docOperators + "\n" + docEL
	if len(body) < 1000 {
		t.Fatalf("docs body is %d bytes — the pages are not being read", len(body))
	}
	return body
}
