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

func dollarSymbols() map[string]string {
	return map[string]string{
		"clients": "array", "cases": "array", "codes": "array", "client": "entity",
		"street": "string", "zip": "string", "nm": "name", "eligible": "boolean",
		"program": "string", "job": "string",
	}
}

// `$foo` is removed (#1280). It lexed as a name, but the sigil rode into the
// postfix, where `$foo` is an executable lookup of an attribute literally
// called "$foo" — which no EDD declares, so every one of these failed at
// runtime. Each form is a compile error now, and the error names the
// spelling that works.
func TestDollarNamesAreRejected(t *testing.T) {
	forms := []string{
		"set nm = $foo",
		"set zip = $unknown",
		"set street = street + $Oak",
		"sort clients in ascending order by $steve",
		"sort clients in descending order by $steve",
		"remove $steve from clients array",
		"set client = new $theClient entity",
		"set codes = array of values [ job, $boy ]",
		"set eligible = $aName is equal to $anotherName",
		"set eligible = $aName != program",
		"set eligible = there is match for all clients to $applying in cases",
		"set street = using client ( $x )",
		"set codes = (array) $kids",
		"perform $x",
	}
	for _, form := range forms {
		t.Run(form, func(t *testing.T) {
			c := NewCompiler()
			c.SetSymbols(dollarSymbols())
			got, err := c.CompileAction(form)
			if err == nil {
				t.Fatalf("compiled to %q; `$name` must be refused", got)
			}
			if !strings.Contains(err.Error(), "#1280") {
				t.Errorf("error should cite the removal, got: %v", err)
			}
		})
	}
}

// The error points at the spelling that works, with the author's own name
// filled in rather than a generic placeholder.
func TestDollarNameErrorNamesTheReplacement(t *testing.T) {
	c := NewCompiler()
	c.SetSymbols(dollarSymbols())
	_, err := c.CompileAction("set nm = $sort_field")
	if err == nil {
		t.Fatal("expected a compile error")
	}
	for _, want := range []string{"`$sort_field` is not a name", `the name "sort_field"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should contain %q, got: %v", want, err)
		}
	}
}

// The spellings that replace it, and what they compile to. `cvn` makes a
// real name; the operators that want one (sortentities, createentity,
// performtable) re-intern whatever they are handed.
func TestNameSpellingsThatWork(t *testing.T) {
	cases := []struct{ dsl, want string }{
		{`set nm = the name "foo"`, `"foo" cvn cvs /nm xdef`},
		{`set nm = (name) "foo"`, `"foo" cvn cvs /nm xdef`},
		{`sort clients in ascending order by the name "steve"`, `clients "steve" cvn true sortentities`},
		{`remove the name "steve" from clients array`, `clients "steve" cvn remove`},
		{`set street = street + the name "Oak"`, `street "Oak" cvn cvs concat cvs /street xdef`},
		{`set client = new client entity`, `/client createentity cve /client xdef`},
	}
	for _, c := range cases {
		t.Run(c.dsl, func(t *testing.T) {
			comp := NewCompiler()
			comp.SetSymbols(dollarSymbols())
			got, err := comp.CompileAction(c.dsl)
			if err != nil {
				t.Fatalf("%q: %v", c.dsl, err)
			}
			if strings.TrimSpace(got) != c.want {
				t.Errorf("%q\n got %q\nwant %q", c.dsl, got, c.want)
			}
		})
	}
}
