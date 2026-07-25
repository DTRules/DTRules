// Copyright 2025 DTRules contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apiserver

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/trace"
)

// dirExists reports whether path exists and is a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// projectXMLDir resolves where a project keeps its rules XML: the xml_dir
// override from DTRules.xml when declared (how non-sample projects like
// staking point at nested rule dirs), else the xml/ subdirectory when it
// exists, else the project root. Mirrors cmd/dtrules resolveDirs precedence
// so the CLI and the editor agree on scope — fingerprints in particular
// must hash the same file set.
func projectXMLDir(projectPath string) string {
	if data, err := os.ReadFile(filepath.Join(projectPath, "DTRules.xml")); err == nil {
		var cfg struct {
			XMLDir string `xml:"xml_dir"`
		}
		if xml.Unmarshal(data, &cfg) == nil && cfg.XMLDir != "" {
			dir := cfg.XMLDir
			if !filepath.IsAbs(dir) {
				dir = filepath.Join(projectPath, dir)
			}
			if dirExists(dir) {
				return dir
			}
		}
	}
	if xd := filepath.Join(projectPath, "xml"); dirExists(xd) {
		return xd
	}
	return projectPath
}

// debugSession is the server-side state of one loaded trace: the parsed
// trace, and a replay session positioned at some node. v1 holds a single
// session per server; every viewer shares it (the web-conference model).
type debugSession struct {
	tracePath string
	trace     *trace.Trace
	nodeCount int
	position  int // node number the replay session is positioned at

	// replaySess is the session reconstructed by replaying to position.
	replaySess dtrules.Session

	provenance       trace.Provenance
	fingerprintMatch string // "match", "mismatch", or "unknown"
	verifyMismatches []string
}

// consoleBlocked are mutating operators the debug console refuses: the
// console is read-only, and raw postfix has no parse-time notion of
// mutation, so enforcement is by operator name.
var consoleBlocked = map[string]bool{
	"xdef": true, "def": true, "put": true, "xput": true,
	"addto": true, "addat": true, "remove": true, "removeat": true,
	"entitypush": true, "entitypop": true, "createentity": true,
	"newentity": true, "findcreateentity": true, "performtable": true,
	"executetable": true, "clear": true,
}

// handleDebugLoad loads a trace file and prepares a replay session.
// POST /api/debug/load {"path": "..."} — path is validated against the
// project root like every other file access.
func (s *Server) handleDebugLoad(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := s.limitedDecode(w, r, &req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	validated, err := s.validateProjectPath(req.Path)
	if err != nil {
		jsonError(w, fmt.Sprintf("Invalid trace path: %v", err), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ruleSet == nil {
		jsonError(w, "Open a project before loading a trace", http.StatusBadRequest)
		return
	}

	tr := trace.NewTrace()
	root, err := tr.Load(validated)
	if err != nil {
		jsonError(w, fmt.Sprintf("Failed to load trace: %v", err), http.StatusBadRequest)
		return
	}

	ds := &debugSession{
		tracePath:  validated,
		trace:      tr,
		nodeCount:  root.Count(),
		provenance: tr.Provenance(),
	}

	// Compare the trace's rules fingerprint against the open project. The
	// CLI fingerprints the resolved rules directory, so match that scope —
	// fingerprinting the project root would also hash input/output XML and
	// never match.
	fpDir := projectXMLDir(s.projectPath)
	ds.fingerprintMatch = "unknown"
	if ds.provenance.RulesFingerprint != "" {
		if fp, err := trace.FingerprintRules(fpDir); err == nil {
			if fp == ds.provenance.RulesFingerprint {
				ds.fingerprintMatch = "match"
			} else {
				ds.fingerprintMatch = "mismatch"
			}
		}
	}

	// Always verify: replay to the recorded final state and compare.
	if fs := tr.FinalState(); fs != nil {
		if sess, err := tr.SetState(s.ruleSet, fs); err == nil {
			ds.verifyMismatches = tr.VerifyFinalState(sess.GetState())
		} else {
			ds.verifyMismatches = []string{fmt.Sprintf("replay failed: %v", err)}
		}
	} else {
		ds.verifyMismatches = []string{"trace records no finalState"}
	}

	if ds.verifyMismatches == nil {
		ds.verifyMismatches = []string{}
	}

	// Start positioned at the beginning.
	ds.position = 1
	if sess, err := tr.SetState(s.ruleSet, root); err == nil {
		ds.replaySess = sess
	}

	s.debug = ds

	jsonResponse(w, map[string]interface{}{
		"success":          true,
		"tracePath":        validated,
		"nodes":            ds.nodeCount,
		"dtrulesVersion":   ds.provenance.DTRulesVersion,
		"rulesFingerprint": ds.provenance.RulesFingerprint,
		"fingerprintMatch": ds.fingerprintMatch,
		"verifyMismatches": ds.verifyMismatches,
	})
}

// debugNodeJSON serializes a trace subtree for the UI's tree view.
func debugNodeJSON(n *trace.TraceNode) map[string]interface{} {
	children := make([]map[string]interface{}, 0, len(n.Children))
	for _, c := range n.Children {
		children = append(children, debugNodeJSON(c))
	}
	out := map[string]interface{}{
		"number":   n.Number,
		"name":     n.Name,
		"children": children,
	}
	if len(n.Attributes) > 0 {
		out["attrs"] = n.Attributes
	}
	if n.Body != "" {
		out["body"] = n.Body
	}
	return out
}

// handleDebugTree returns the loaded trace as a tree.
// GET /api/debug/tree
func (s *Server) handleDebugTree(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.debug == nil {
		jsonError(w, "No trace loaded", http.StatusBadRequest)
		return
	}
	jsonResponse(w, map[string]interface{}{
		"success": true,
		"tree":    debugNodeJSON(s.debug.trace.Root()),
	})
}

// debugStack serializes the replay session's entity stack, bottom first.
func debugStack(state trace.StackState) []map[string]interface{} {
	frames := []map[string]interface{}{}
	depth := state.EntityDepth()
	for i := 0; i < depth; i++ {
		e, err := state.EntityFetch(i)
		if err != nil || e == nil {
			continue
		}
		attrs := map[string]string{}
		for _, n := range e.GetAttributeNames() {
			name := n.StringValue()
			if name == "" {
				continue
			}
			if v, err := e.Get(n); err == nil && v != nil {
				attrs[name] = v.StringValue()
			}
		}
		frames = append(frames, map[string]interface{}{
			"name":  e.GetName().StringValue(),
			"id":    e.GetID(),
			"attrs": attrs,
		})
	}
	return frames
}

// debugContext walks up from a node to name the enclosing decision table,
// column, and action — the UI's breadcrumb and grid highlight.
func debugContext(n *trace.TraceNode) map[string]string {
	ctx := map[string]string{}
	for cur := n; cur != nil; cur = cur.Parent {
		switch cur.Name {
		case "action":
			if _, ok := ctx["action"]; !ok {
				ctx["action"] = cur.Attributes["n"]
			}
		case "column":
			if _, ok := ctx["column"]; !ok {
				ctx["column"] = cur.Attributes["n"]
			}
		case "decisiontable":
			if _, ok := ctx["table"]; !ok {
				ctx["table"] = cur.Attributes["name"]
			}
		}
	}
	return ctx
}

// handleDebugPosition replays to a node ("run to here" / stepping).
// POST /api/debug/position {"node": 123}
func (s *Server) handleDebugPosition(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Node int `json:"node"`
	}
	if err := s.limitedDecode(w, r, &req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.debug == nil {
		jsonError(w, "No trace loaded", http.StatusBadRequest)
		return
	}
	if req.Node < 1 || req.Node > s.debug.nodeCount {
		jsonError(w, fmt.Sprintf("Node %d out of range (1..%d)", req.Node, s.debug.nodeCount), http.StatusBadRequest)
		return
	}
	target := s.debug.trace.Find(req.Node)
	if target == nil {
		jsonError(w, fmt.Sprintf("Node %d not found", req.Node), http.StatusBadRequest)
		return
	}

	sess, err := s.debug.trace.SetState(s.ruleSet, target)
	if err != nil {
		jsonError(w, fmt.Sprintf("Replay failed: %v", err), http.StatusInternalServerError)
		return
	}
	s.debug.replaySess = sess
	s.debug.position = req.Node

	jsonResponse(w, map[string]interface{}{
		"success":  true,
		"position": req.Node,
		"nodes":    s.debug.nodeCount,
		"context":  debugContext(target),
		"node":     map[string]interface{}{"name": target.Name, "attrs": target.Attributes, "body": target.Body},
		"stack":    debugStack(sess.GetState()),
	})
}

// handleDebugConsole executes read-only postfix at the current position and
// returns whatever is left on the data stack.
// POST /api/debug/console {"postfix": "..."}
func (s *Server) handleDebugConsole(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Postfix string `json:"postfix"`
	}
	if err := s.limitedDecode(w, r, &req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.debug == nil || s.debug.replaySess == nil {
		jsonError(w, "No trace loaded", http.StatusBadRequest)
		return
	}

	// Read-only enforcement: refuse mutating operators by name.
	for _, tok := range strings.Fields(req.Postfix) {
		if consoleBlocked[strings.ToLower(tok)] {
			jsonError(w, fmt.Sprintf("%s is a mutating operator — console is read-only", tok), http.StatusBadRequest)
			return
		}
	}

	compiled, err := s.debug.replaySess.Compile(req.Postfix)
	if err != nil {
		jsonError(w, fmt.Sprintf("Compile error: %v", err), http.StatusBadRequest)
		return
	}
	state := s.debug.replaySess.GetState()
	if err := compiled.Execute(state); err != nil {
		// Drain anything the failed execution left behind.
		for state.DataStackDepth() > 0 {
			state.DataPop()
		}
		jsonError(w, fmt.Sprintf("Execution error: %v", err), http.StatusBadRequest)
		return
	}

	// Whatever is left on the data stack is the result, printed top-last.
	var results []string
	var stack []dtrules.Object
	for state.DataStackDepth() > 0 {
		if v, err := state.DataPop(); err == nil {
			stack = append(stack, v)
		}
	}
	for i := len(stack) - 1; i >= 0; i-- {
		results = append(results, stack[i].StringValue())
	}

	jsonResponse(w, map[string]interface{}{
		"success": true,
		"results": results,
	})
}
