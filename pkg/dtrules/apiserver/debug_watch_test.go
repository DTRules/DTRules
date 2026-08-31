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

// "Run until X is true" — the predicate is re-evaluated as the replay walks
// forward, and the run stops where it first becomes true (#930).

func TestWatchStopsWhereThePredicateBecomesTrue(t *testing.T) {
	h := debugServer(t, true)

	// client.age is only on the entity stack once the run descends into a
	// client, so the predicate is genuinely false-then-true across the trace
	// rather than constant.
	status, body := do(t, h, "POST", "/api/debug/watch",
		`{"expression":"client.age > 0","from":1,"limit":140}`)
	if status != 200 {
		t.Fatalf("status %d, body %v", status, body)
	}
	if body["hit"] != true {
		t.Fatalf("the predicate never fired anywhere in the trace: %v", body)
	}
	pos, _ := body["position"].(float64)
	if pos < 2 {
		t.Errorf("position = %v, want a node after the one it started from", body["position"])
	}
	if body["language"] != "el" {
		t.Errorf("language = %v, want el — the watch takes EL like the console", body["language"])
	}
	// The debugger is left where the predicate fired, which is the point of
	// "run until".
	_, st := do(t, h, "GET", "/api/debug/status", "")
	if got, _ := st["position"].(float64); got != pos {
		t.Errorf("after a hit the session sits at %v, want the node that fired (%v)", got, pos)
	}
}

// A predicate that is already true where the search starts has not *become*
// true. Constant-true is the sharpest case: it must never fire, because
// otherwise every step through a run of satisfying nodes stops on the spot and
// the debugger cannot be walked forward.
func TestWatchDoesNotFireOnAPredicateAlreadyTrue(t *testing.T) {
	h := debugServer(t, true)

	status, body := do(t, h, "POST", "/api/debug/watch",
		`{"expression":"1 > 0","from":1,"limit":60}`)
	if status != 200 {
		t.Fatalf("status %d, body %v", status, body)
	}
	if body["hit"] != false {
		t.Errorf("a predicate true at every node reported a transition at %v — "+
			"the watch is level-triggered", body["position"])
	}
}

// Edge-triggered, not level-triggered. A predicate that is already true where
// the search starts has not just become true, so the search must walk past it
// rather than stopping on the spot -- otherwise stepping through a run of
// nodes that all satisfy it is impossible, because every step stops at once.
func TestWatchIsEdgeTriggered(t *testing.T) {
	h := debugServer(t, true)

	first, body := do(t, h, "POST", "/api/debug/watch",
		`{"expression":"client.age > 0","from":1,"limit":140}`)
	if first != 200 || body["hit"] != true {
		t.Fatalf("setup: %v", body)
	}
	at, _ := body["position"].(float64)

	// Resuming from where it stopped must not return the same node: at that
	// node the predicate is true, so it has not just become true.
	status, body2 := do(t, h, "POST", "/api/debug/watch",
		`{"expression":"client.age > 0","from":`+itoa(int(at))+`,"limit":140}`)
	if status != 200 {
		t.Fatalf("status %d, body %v", status, body2)
	}
	if body2["hit"] == true {
		if again, _ := body2["position"].(float64); again == at {
			t.Errorf("stopped again at node %v — the watch is level-triggered, so a run "+
				"of nodes satisfying the predicate cannot be walked through", at)
		}
	}
}

// "Not yet" is an answer, not an error: the caller needs to know how far the
// search reached so it can resume, and an unbounded run-until would be a
// request that never returns.
func TestWatchReportsNoHitWithWhereItReached(t *testing.T) {
	h := debugServer(t, true)

	status, body := do(t, h, "POST", "/api/debug/watch",
		`{"expression":"1 > 2","from":1,"limit":25}`)
	if status != 200 {
		t.Fatalf("a search that found nothing should not be an error: %d %v", status, body)
	}
	if body["hit"] != false {
		t.Fatalf("hit = %v, want false", body["hit"])
	}
	ex, _ := body["examined"].(float64)
	if ex <= 0 || ex > 25 {
		t.Errorf("examined = %v, want it bounded by the limit", body["examined"])
	}
	if _, ok := body["searched_to"]; !ok {
		t.Error("no searched_to — the caller cannot resume without knowing where it got to")
	}
}

// A watch expression is read-only, on the same terms as the console.
func TestWatchRefusesAMutatingExpression(t *testing.T) {
	h := debugServer(t, true)

	status, body := do(t, h, "POST", "/api/debug/watch",
		`{"expression":"1 /x xdef","language":"postfix","from":1}`)
	if status == 200 {
		t.Fatalf("a mutating watch was accepted: %v", body)
	}
	msg, _ := body["error"].(string)
	if !strings.Contains(strings.ToLower(msg), "read-only") {
		t.Errorf("refused, but not as read-only: %v", body)
	}
}

func TestWatchRequiresAnExpression(t *testing.T) {
	h := debugServer(t, true)
	status, _ := do(t, h, "POST", "/api/debug/watch", `{"from":1}`)
	if status == 200 {
		t.Error("an empty watch expression should be refused")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
