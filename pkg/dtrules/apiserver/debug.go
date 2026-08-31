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
	"fmt"
	"github.com/DTRules/DTRules/pkg/dtrules/project"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
	"github.com/DTRules/DTRules/pkg/dtrules/compiler/el"
	"github.com/DTRules/DTRules/pkg/dtrules/operators"
	"github.com/DTRules/DTRules/pkg/dtrules/trace"
)

// dirExists reports whether path exists and is a directory.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// projectConfig is what a project's DTRules.xml declares (all optional).
// projectXMLDir resolves where a project keeps its rules XML: the xml_dir
// override from DTRules.xml when declared (how non-sample projects like
// staking point at nested rule dirs), else the xml/ subdirectory when it
// exists, else the project root. Mirrors cmd/dtrules resolveDirs precedence
// so the CLI and the editor agree on scope — fingerprints in particular
// must hash the same file set.
func projectXMLDir(projectPath string) string {
	if cfg := project.Load(projectPath); dirExists(cfg.XMLDir) {
		return cfg.XMLDir
	}
	if xd := filepath.Join(projectPath, "xml"); dirExists(xd) {
		return xd
	}
	return projectPath
}

// configPayload describes the project's effective configuration for the UI:
// where the rules actually load from (relative to the project root) and the
// declared entry table. Caller holds s.mu (read).
func (s *Server) configPayload() map[string]interface{} {
	if s.projectPath == "" {
		return nil
	}
	cfg := project.Load(s.projectPath)
	xmlDir := projectXMLDir(s.projectPath)
	rel, err := filepath.Rel(s.projectPath, xmlDir)
	if err != nil {
		rel = xmlDir
	}
	// A fallback-to-root scan that found decision tables under more than
	// one top-level subdirectory almost certainly swept several unrelated
	// projects (e.g. the editor was launched from a repo root). Surface
	// that so the UI can warn instead of showing a truthful-but-useless
	// "rules: .".
	multiRoot := false
	if rel == "." {
		tops := map[string]bool{}
		rootFiles := 0
		for _, f := range s.dtFiles {
			parts := strings.SplitN(filepath.ToSlash(f), "/", 2)
			if len(parts) == 2 {
				tops[parts[0]] = true
			} else {
				rootFiles++
			}
		}
		// A real project keeps its main *_dt.xml at the scan root
		// (subfolders like states/ are fine); a repo-root sweep has
		// everything nested under several unrelated directories.
		multiRoot = rootFiles == 0 && len(tops) > 1
	}
	return map[string]interface{}{
		"xmlDir":    rel,
		"entry":     cfg.Entry,
		"declared":  cfg.XMLDir != "" || cfg.Entry != "",
		"multiRoot": multiRoot,
	}
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

	// baseline holds the original trace while a speculative run is the
	// active session (nil otherwise). Reports diff against it; reset
	// restores it.
	baseline     *trace.Trace
	baselinePath string
	speculative  bool
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

// consoleBlockedAfterEL is consoleBlocked without the two scope operators.
//
// EL cannot express mutation in a condition, so this is a backstop rather than
// the guard -- but it must not refuse what the compiler legitimately emits.
// entitypush/entitypop come out in balanced pairs for `there is <x> in <array>
// where ...`, which is a read.
var consoleBlockedAfterEL = func() map[string]bool {
	m := make(map[string]bool, len(consoleBlocked))
	for k, v := range consoleBlocked {
		m[k] = v
	}
	delete(m, "entitypush")
	delete(m, "entitypop")
	return m
}()

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

	ds, err := s.loadDebugSessionLocked(validated)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonResponse(w, debugSessionPayload(ds))
}

// LoadDebugTrace loads a trace into the server's debug session — the
// programmatic form of POST /api/debug/load, used by `dtrules debug` /
// `dtrules edit --trace` to open the editor with the trace ready.
func (s *Server) LoadDebugTrace(path string) error {
	validated, err := s.validateProjectPath(path)
	if err != nil {
		return fmt.Errorf("invalid trace path: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err = s.loadDebugSessionLocked(validated)
	return err
}

// handleDebugStatus reports the current debug session, so a UI that mounts
// after a server-side preload (dtrules debug) can adopt it.
// GET /api/debug/status
func (s *Server) handleDebugStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.debug == nil {
		jsonResponse(w, map[string]interface{}{"success": true, "loaded": false})
		return
	}
	payload := debugSessionPayload(s.debug)
	payload["loaded"] = true
	jsonResponse(w, payload)
}

// debugSessionPayload is the JSON shape shared by load and status.
func debugSessionPayload(ds *debugSession) map[string]interface{} {
	return map[string]interface{}{
		"success":          true,
		"tracePath":        ds.tracePath,
		"nodes":            ds.nodeCount,
		"dtrulesVersion":   ds.provenance.DTRulesVersion,
		"rulesFingerprint": ds.provenance.RulesFingerprint,
		"fingerprintMatch": ds.fingerprintMatch,
		"verifyMismatches": ds.verifyMismatches,
		"speculative":      ds.speculative,
	}
}

// loadDebugSessionLocked parses the trace at validated, verifies it against
// the open project, and installs it as the server's debug session. Caller
// holds s.mu.
func (s *Server) loadDebugSessionLocked(validated string) (*debugSession, error) {
	if s.ruleSet == nil {
		return nil, fmt.Errorf("open a project before loading a trace")
	}

	tr := trace.NewTrace()
	root, err := tr.Load(validated)
	if err != nil {
		return nil, fmt.Errorf("failed to load trace: %w", err)
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
	return ds, nil
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
		// Language pins the input: "el", "postfix", or "" to try EL first
		// and fall back. Pinning is for callers that already know, and for
		// asking why something did not compile as EL.
		Language string `json:"language"`
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

	// Type EL at the console, not postfix. `taxpayer.age > 65` is what an
	// author has in front of them; `taxpayer.age 65 i>` is a transcription
	// they have to do in their head, at the moment they are trying to think
	// about something else (#930).
	//
	// EL first, raw postfix if that does not parse. The fallback is what makes
	// it safe to try: the console has always been postfix and stays so for
	// anything EL cannot express, including bare stack manipulation.
	source, language, elErr := s.consoleCompileSource(req.Postfix, req.Language)
	if elErr != nil {
		jsonError(w, elErr.Error(), http.StatusBadRequest)
		return
	}

	// Read-only enforcement, against the right list for how the input was
	// read. The blocklist exists because "raw postfix has no parse-time notion
	// of mutation" -- EL does: a condition is an expression, and the grammar
	// has no spelling for assignment.
	//
	// So compiled EL is checked against a shorter list. entitypush/entitypop
	// are blocked in raw postfix because a hand-typed push can be left
	// unbalanced; EL emits them in balanced pairs for scoping, and
	// `there is person in household.members where person.age > 18` -- a pure
	// read -- compiles to exactly that. Checking compiled EL against the raw
	// list would refuse it.
	blocked := consoleBlocked
	if language == "el" {
		blocked = consoleBlockedAfterEL
	}
	for _, tok := range strings.Fields(source) {
		if blocked[strings.ToLower(tok)] {
			jsonError(w, fmt.Sprintf("%s is a mutating operator — console is read-only", tok), http.StatusBadRequest)
			return
		}
	}

	compiled, err := s.debug.replaySess.Compile(source)
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
		// Which way the input was read, and what actually ran. An EL
		// expression that compiles to surprising postfix is exactly the thing
		// a debugger user wants to see.
		"language": language,
		"postfix":  source,
	})
}

// consoleCompileSource resolves console input to the postfix that will run,
// and says which language it was read as.
//
// With no language pinned it tries EL and falls back to treating the input as
// postfix, because that is what the console accepted before and still must:
// EL has no spelling for bare stack manipulation, and a debugger is exactly
// where someone reaches for it.
//
// A pinned language does not fall back — asking "why does this not compile as
// EL" deserves the EL error, not a postfix error about the same text.
func (s *Server) consoleCompileSource(input, language string) (source, used string, err error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", "postfix", nil
	}

	switch strings.ToLower(strings.TrimSpace(language)) {
	case "postfix":
		return trimmed, "postfix", nil
	case "el":
		compiled, cerr := s.compileConsoleEL(trimmed)
		if cerr != nil {
			return "", "", fmt.Errorf("EL compile error: %v", cerr)
		}
		return compiled, "el", nil
	}

	if compiled, cerr := s.compileConsoleEL(trimmed); cerr == nil && strings.TrimSpace(compiled) != "" {
		return compiled, "el", nil
	}
	return trimmed, "postfix", nil
}

// compileConsoleEL compiles one EL expression, typed against the project's own
// EDD so `taxpayer.age` is an integer rather than defaulting to one.
func (s *Server) compileConsoleEL(expr string) (string, error) {
	c := el.NewCompiler()
	c.SetOperatorChecker(func(name string) bool {
		_, ok := operators.GetByString(name)
		return ok
	})
	if syms := s.consoleSymbols(); len(syms) > 0 {
		c.SetSymbols(syms)
	}
	// A console line is an expression to evaluate, which is the condition
	// form: it leaves its value on the data stack, which is what the console
	// then prints.
	return c.CompileCondition(expr)
}

// consoleSymbols is the project's EDD, so a console expression is typed the
// way the rules are. Without it every field defaults to integer and
// `taxpayer.agi > 1000.5` compiles to an integer comparison.
//
// Loaded per call rather than cached: a console line is typed by a human, so
// the read is free at this rate, and the debugger should not go stale against
// an EDD edited in another tab.
func (s *Server) consoleSymbols() map[string]string {
	if s.projectPath == "" {
		return nil
	}
	return authoring.LoadEDDSymbols(projectXMLDir(s.projectPath))
}
