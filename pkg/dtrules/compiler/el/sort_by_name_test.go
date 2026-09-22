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

// `sort <array> in … order by <nexpr>` sorts by the field the name
// expression evaluates to. `by entry.key`, `by key` and `by $key`, where
// `key` is a string field, are field reads: they evaluate to the field's
// value, before sortentities runs and with no entity on the stack, and fail
// at run time with "The Name 'key' was not defined by any Entity". They used
// to compile clean (#1227).

func sortSymbols() map[string]string {
	return map[string]string{
		"state.entries": "array", "entries": "array",
		"entry.key": "string", "key": "string",
		"entry.when": "date", "when": "date",
		"state.sort_field": "name", "sort_field": "name",
	}
}

func TestSortByFieldReadRejected(t *testing.T) {
	cases := []struct{ dsl, field string }{
		{"sort state.entries in ascending order by entry.key", "key"},
		{"sort state.entries in descending order by entry.key", "key"},
		{"sort state.entries in ascending order by key", "key"},
		{"sort state.entries in ascending order by $key", "key"},
		{"sort state.entries in ascending order by entry.when", "when"},
	}
	for _, tc := range cases {
		t.Run(tc.dsl, func(t *testing.T) {
			c := NewCompiler()
			c.SetSymbols(sortSymbols())
			pf, err := c.CompileAction(tc.dsl)
			if err == nil {
				t.Fatalf("compiled to %q; want an error", pf)
			}
			hint := `the name "` + tc.field + `"`
			if !strings.Contains(err.Error(), hint) {
				t.Errorf("error does not carry the hint %s: %v", hint, err)
			}
		})
	}
}

// The name forms still compile: a literal name (`the name "key"`, `(name)
// "key"`, the keyword `name`) and a name-typed field holding the sort key.
func TestSortByNameAccepted(t *testing.T) {
	cases := []struct{ dsl, want string }{
		{`sort state.entries in ascending order by the name "key"`, `state.entries "key" cvn true sortentities`},
		{`sort state.entries in descending order by (name) "key"`, `state.entries "key" cvn false sortentities`},
		{`sort state.entries in ascending order by name`, `state.entries /name true sortentities`},
		{`sort state.entries in ascending order by state.sort_field`, `state.entries state.sort_field true sortentities`},
		{`sort state.entries in ascending order by $sort_field`, `state.entries $sort_field true sortentities`},
	}
	for _, tc := range cases {
		t.Run(tc.dsl, func(t *testing.T) {
			c := NewCompiler()
			c.SetSymbols(sortSymbols())
			pf, err := c.CompileAction(tc.dsl)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			if strings.TrimSpace(pf) != tc.want {
				t.Errorf("postfix = %q, want %q", pf, tc.want)
			}
		})
	}
}

// A bare field name shared by two entities with different types (review of
// #1240). The EDD loader fills the bare key last-entity-wins, so the bare
// type is whichever entity came last; the check must not trust it. The
// qualified names are exact and still decide.
func TestSortByBareNameCollision(t *testing.T) {
	// Both orders the loader could have filled the bare key in.
	for _, bare := range []string{"name", "string"} {
		c := NewCompiler()
		c.SetSymbols(map[string]string{
			"orders":        "array",
			"order.sortkey": "name", "line.sortkey": "string",
			"sortkey": bare,
		})
		if pf, err := c.CompileAction("sort orders in ascending order by sortkey"); err != nil {
			t.Errorf("bare sortkey (filled as %s) refused though order.sortkey is a name: %v", bare, err)
		} else if !strings.Contains(pf, "sortkey true sortentities") {
			t.Errorf("postfix = %q", pf)
		}
		// Qualified: order.sortkey is a name, line.sortkey is not.
		if _, err := c.CompileAction("sort orders in ascending order by order.sortkey"); err != nil {
			t.Errorf("order.sortkey is a name field, refused: %v", err)
		}
		_, err := c.CompileAction("sort orders in ascending order by line.sortkey")
		if err == nil || !strings.Contains(err.Error(), `the name "sortkey"`) {
			t.Errorf("line.sortkey is a string field; want the hint, got %v", err)
		}
	}
	// Every entity that declares the bare name agrees it is a string: the
	// bare name is still refused, and the message names where it comes from.
	c := NewCompiler()
	c.SetSymbols(map[string]string{
		"orders": "array", "order.key": "string", "line.key": "string", "key": "string",
	})
	_, err := c.CompileAction("sort orders in ascending order by key")
	if err == nil || !strings.Contains(err.Error(), `the name "key"`) {
		t.Errorf("bare key, string everywhere: want the hint, got %v", err)
	}
}
