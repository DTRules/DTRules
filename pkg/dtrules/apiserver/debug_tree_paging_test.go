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
	"testing"
)

// The tree ships as one blob. That is comfortable at 398k nodes and will not
// be past a million, so the endpoint can now be asked for part of it: `depth`
// bounds the descent, `node` names a subtree, and a node whose children were
// left out says so and reports how many it has (#930).
//
// Paging is opt-in. The tree view asks for the whole thing today, and a
// default that changed under it would be a regression dressed as an
// improvement.

func treeOf(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	tree, ok := body["tree"].(map[string]any)
	if !ok {
		t.Fatalf("no tree in response: %v", body)
	}
	return tree
}

func kids(t *testing.T, node map[string]any) []any {
	t.Helper()
	c, _ := node["children"].([]any)
	return c
}

// countNodes walks what actually came back.
func countNodes(node map[string]any) int {
	n := 1
	c, _ := node["children"].([]any)
	for _, k := range c {
		if m, ok := k.(map[string]any); ok {
			n += countNodes(m)
		}
	}
	return n
}

func TestTreeIsWholeByDefault(t *testing.T) {
	h := debugServer(t, true)

	status, body := do(t, h, "GET", "/api/debug/tree", "")
	if status != 200 {
		t.Fatalf("status %d, body %v", status, body)
	}
	tree := treeOf(t, body)
	if _, truncated := tree["truncated"]; truncated {
		t.Error("the default response was truncated — the tree view asks for the whole tree")
	}
	total, _ := body["nodes"].(float64)
	if got := countNodes(tree); float64(got) != total {
		t.Errorf("returned %d nodes, trace has %v — the default must still be the whole tree",
			got, body["nodes"])
	}
}

// depth=0 is the root alone: the shape a client starts from before anything
// is opened.
func TestTreeDepthZeroReturnsTheRootAlone(t *testing.T) {
	h := debugServer(t, true)

	status, body := do(t, h, "GET", "/api/debug/tree?depth=0", "")
	if status != 200 {
		t.Fatalf("status %d, body %v", status, body)
	}
	tree := treeOf(t, body)
	if n := countNodes(tree); n != 1 {
		t.Errorf("depth=0 returned %d nodes, want just the root", n)
	}
	if tree["truncated"] != true {
		t.Error("a root with children that were left out must say so")
	}
	cc, _ := tree["childCount"].(float64)
	if cc <= 0 {
		t.Errorf("childCount = %v — a client cannot show 'n more' without it", tree["childCount"])
	}
}

func TestTreeDepthOneReturnsOneLevel(t *testing.T) {
	h := debugServer(t, true)

	_, body := do(t, h, "GET", "/api/debug/tree?depth=1", "")
	tree := treeOf(t, body)
	children := kids(t, tree)
	if len(children) == 0 {
		t.Fatal("depth=1 returned no children")
	}
	if tree["truncated"] == true {
		t.Error("the root's own children were requested, so the root is not truncated")
	}
	// Every grandchild level is where truncation should appear instead.
	for _, c := range children {
		m, _ := c.(map[string]any)
		if cc, _ := m["childCount"].(float64); cc > 0 && m["truncated"] != true {
			t.Errorf("a child with %v of its own children was not marked truncated", cc)
		}
	}
}

// Naming a node fetches that subtree, which is how a client fills in a branch
// the user just opened.
func TestTreeCanFetchOneSubtree(t *testing.T) {
	h := debugServer(t, true)

	_, body := do(t, h, "GET", "/api/debug/tree?depth=1", "")
	children := kids(t, treeOf(t, body))
	if len(children) == 0 {
		t.Skip("trace root has no children")
	}
	first, _ := children[0].(map[string]any)
	num, _ := first["number"].(float64)

	status, sub := do(t, h, "GET", "/api/debug/tree?node="+itoa(int(num)), "")
	if status != 200 {
		t.Fatalf("status %d, body %v", status, sub)
	}
	subtree := treeOf(t, sub)
	if got, _ := subtree["number"].(float64); got != num {
		t.Errorf("asked for node %v, got %v", num, got)
	}
	if _, truncated := subtree["truncated"]; truncated {
		t.Error("a subtree requested with no depth should come back whole")
	}
}

func TestTreeRejectsNonsenseParameters(t *testing.T) {
	h := debugServer(t, true)

	for _, q := range []string{"?depth=-1", "?depth=abc", "?node=0", "?node=abc", "?node=999999"} {
		if status, _ := do(t, h, "GET", "/api/debug/tree"+q, ""); status == 200 {
			t.Errorf("%s was accepted", q)
		}
	}
}
