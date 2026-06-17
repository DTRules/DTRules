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

package main

import (
	"bufio"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/collect"
)

func askWith(input string, req collect.Request) (string, bool) {
	a := &cliAsker{in: bufio.NewReader(strings.NewReader(input))}
	v, ok, _ := a.Ask(req)
	if v == nil {
		return "", ok
	}
	return v.StringValue(), ok
}

func TestCLIAsker_NumberAndAscii(t *testing.T) {
	// A typed value is returned verbatim (entity.Put coerces later).
	if v, ok := askWith("58\n", collect.Request{Field: "age", QType: "number"}); !ok || v != "58" {
		t.Errorf("number: got (%q,%v), want (58,true)", v, ok)
	}
	// Empty line keeps the default (ok=false).
	if v, ok := askWith("\n", collect.Request{Field: "age", QType: "number"}); ok || v != "" {
		t.Errorf("empty: got (%q,%v), want (\"\",false)", v, ok)
	}
	if v, ok := askWith("Acute Sinusitis\n", collect.Request{Field: "dx", QType: "ascii"}); !ok || v != "Acute Sinusitis" {
		t.Errorf("ascii: got (%q,%v)", v, ok)
	}
}

func TestCLIAsker_MultipleChoice(t *testing.T) {
	req := collect.Request{
		Field: "allergic", QType: "multiple_choice",
		Options: []collect.Option{{Value: "true", Label: "Yes"}, {Value: "false", Label: "No"}},
	}
	// Numbered choice maps to the option's value.
	if v, ok := askWith("1\n", req); !ok || v != "true" {
		t.Errorf("choice 1: got (%q,%v), want (true,true)", v, ok)
	}
	if v, ok := askWith("2\n", req); !ok || v != "false" {
		t.Errorf("choice 2: got (%q,%v), want (false,true)", v, ok)
	}
	// A literal value is accepted too.
	if v, ok := askWith("false\n", req); !ok || v != "false" {
		t.Errorf("literal: got (%q,%v)", v, ok)
	}
	// Empty keeps the default.
	if _, ok := askWith("\n", req); ok {
		t.Errorf("empty choice should keep default (ok=false)")
	}
}
