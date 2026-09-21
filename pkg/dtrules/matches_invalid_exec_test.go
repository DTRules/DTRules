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

package dtrules_test

import (
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/compiler/el"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// TestMatchesInvalidPatternIsAnError (#1262): a pattern that does not
// compile makes the rule fail, naming the pattern, instead of reading as
// "does not match". And `matches` is an unanchored search: the pattern may
// match anywhere in the string unless it says ^…$.
func TestMatchesInvalidPatternIsAnError(t *testing.T) {
	rs := session.NewRuleSet("matches-invalid")
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	ef := sess.GetEntityFactory().(*entity.Factory)
	rootName := dtrules.GetRName("root")
	rootRef, err := ef.FindCreateRefEntity(true, rootName)
	if err != nil {
		t.Fatal(err)
	}
	rootRef.AddAttribute(dtrules.GetRName("id"), "", dtrules.NewRString(""), true, true, dtrules.TypeString, "", "", "", "")
	rootRef.AddAttribute(dtrules.GetRName("b"), "", dtrules.GetRBoolean(false), true, true, dtrules.TypeBoolean, "", "", "", "")
	root, err := ef.CreateEntity(sess, rootName)
	if err != nil {
		t.Fatal(err)
	}
	root.Put(dtrules.GetRName("id"), dtrules.NewRString("xabcx"))

	state := sess.GetState().(*interpreter.DTState)
	elc := el.NewCompiler()
	elc.SetSymbols(map[string]string{"id": "string", "b": "boolean"})
	run := func(src string) (bool, error) {
		t.Helper()
		pf, err := elc.CompileAction("set b = " + src)
		if err != nil {
			t.Fatalf("%q compile: %v", src, err)
		}
		obj, err := sess.Compile(pf)
		if err != nil {
			t.Fatalf("%q assemble %q: %v", src, pf, err)
		}
		root.Put(dtrules.GetRName("b"), dtrules.GetRBoolean(false))
		state.EntityPush(root)
		defer state.EntityPop()
		if err := obj.Execute(state); err != nil {
			return false, err
		}
		v, _ := root.Get(dtrules.GetRName("b"))
		got, _ := v.BooleanValue()
		return got, nil
	}

	for _, pattern := range []string{`(`, `[a-`, `a**`} {
		src := `id matches "` + pattern + `"`
		got, err := run(src)
		if err == nil {
			t.Errorf("%s: want an error for an invalid pattern, got %v", src, got)
			continue
		}
		if !strings.Contains(err.Error(), pattern) {
			t.Errorf("%s: error should name the pattern %q: %v", src, pattern, err)
		}
	}

	for _, tc := range []struct {
		src  string
		want bool
	}{
		{`id matches "abc"`, true},
		{`id matches "^abc$"`, false},
		{`id matches "^xabcx$"`, true},
	} {
		got, err := run(tc.src)
		if err != nil {
			t.Errorf("%s: %v", tc.src, err)
		} else if got != tc.want {
			t.Errorf("%s = %v, want %v (matches is an unanchored search)", tc.src, got, tc.want)
		}
	}
}
