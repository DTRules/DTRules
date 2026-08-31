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

package apiserver

import (
	"strings"
	"testing"
)

// The console has always taken postfix. `taxpayer.age 65 i>` is a
// transcription the author has to do in their head at the moment they are
// trying to think about something else, so it now takes EL too and says which
// way it read the input (#930).

func TestConsoleAcceptsEL(t *testing.T) {
	h := debugServer(t, true)

	status, body := do(t, h, "POST", "/api/debug/console", `{"postfix":"2 > 1"}`)
	if status != 200 {
		t.Fatalf("status %d, body %v", status, body)
	}
	if body["language"] != "el" {
		t.Errorf("language = %v, want el", body["language"])
	}
	if got, _ := body["postfix"].(string); strings.TrimSpace(got) != "2 1 >" {
		t.Errorf("postfix = %q, want the compiled form %q", got, "2 1 >")
	}
	results, _ := body["results"].([]any)
	if len(results) != 1 || results[0] != "true" {
		t.Errorf("results = %v, want [true]", results)
	}
}

// The construct that caught a bug in this change before it shipped.
// `there is <x> in <array> where ...` is a pure read, and it compiles to
// `... entitypush ... entitypop swap pop`. Both of those are on the raw-postfix
// blocklist, because a hand-typed push can be left unbalanced -- so checking
// compiled EL against that list refused a legitimate query.
func TestConsoleAllowsAnExistentialQuery(t *testing.T) {
	h := debugServer(t, true)

	status, body := do(t, h, "POST", "/api/debug/console",
		`{"postfix":"there is client in case.clients where client.age > 18"}`)
	// The assertion is about the guard, not the fixture's data: what must not
	// happen is a refusal for containing entitypush. An execution error over
	// the trace's own contents is a different thing and is fine here.
	msg, _ := body["error"].(string)
	if strings.Contains(strings.ToLower(msg), "read-only") ||
		strings.Contains(strings.ToLower(msg), "mutating") {
		t.Fatalf("a read-only existential was refused as a mutation: %v", body)
	}
	if got, _ := body["postfix"].(string); status == 200 && !strings.Contains(got, "entitypush") {
		t.Errorf("postfix = %q, expected the scope operators this test exists for", got)
	}
}

// Postfix still runs. EL has no spelling for bare stack manipulation, and a
// debugger is exactly where someone reaches for it — so the fallback is what
// makes trying EL first safe.
func TestConsoleStillAcceptsPostfix(t *testing.T) {
	h := debugServer(t, true)

	status, body := do(t, h, "POST", "/api/debug/console", `{"postfix":"2 3 +"}`)
	if status != 200 {
		t.Fatalf("status %d, body %v", status, body)
	}
	if body["language"] != "postfix" {
		t.Errorf("language = %v, want postfix", body["language"])
	}
	results, _ := body["results"].([]any)
	if len(results) != 1 || results[0] != "5" {
		t.Errorf("results = %v, want [5]", results)
	}
}

// Raw postfix mutation is still refused, unchanged. EL cannot reach these:
// a condition is an expression and the grammar has no spelling for
// assignment, which is why compiled EL is checked against a shorter list
// rather than trusted outright.
func TestConsoleStillRefusesPostfixMutation(t *testing.T) {
	h := debugServer(t, true)

	for _, expr := range []string{`1 /x xdef`, `client.age 1 addto`} {
		t.Run(expr, func(t *testing.T) {
			status, body := do(t, h, "POST", "/api/debug/console",
				`{"postfix":`+quote(expr)+`}`)
			if status == 200 {
				t.Fatalf("mutation accepted: %v", body)
			}
			msg, _ := body["error"].(string)
			if msg == "" {
				msg, _ = body["message"].(string)
			}
			if !strings.Contains(strings.ToLower(msg), "read-only") &&
				!strings.Contains(strings.ToLower(msg), "mutating") {
				t.Errorf("refused, but not as a mutation: %v", body)
			}
		})
	}
}

// Pinning the language stops the fallback, so "why will this not compile as
// EL" gets the EL error rather than a postfix error about the same text.
func TestConsoleLanguagePinReportsTheELError(t *testing.T) {
	h := debugServer(t, true)

	status, body := do(t, h, "POST", "/api/debug/console",
		`{"postfix":"2 3 +","language":"el"}`)
	if status == 200 {
		t.Fatalf("postfix pinned as EL should not compile: %v", body)
	}
	msg, _ := body["error"].(string)
	if msg == "" {
		msg, _ = body["message"].(string)
	}
	if !strings.Contains(msg, "EL compile error") {
		t.Errorf("error = %q, want it to name EL as the language that failed", msg)
	}
}

func TestConsolePostfixPinSkipsEL(t *testing.T) {
	h := debugServer(t, true)

	status, body := do(t, h, "POST", "/api/debug/console",
		`{"postfix":"2 3 +","language":"postfix"}`)
	if status != 200 {
		t.Fatalf("status %d, body %v", status, body)
	}
	if body["language"] != "postfix" {
		t.Errorf("language = %v, want postfix", body["language"])
	}
}

// quote is a minimal JSON string quoter for the table above.
func quote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}
