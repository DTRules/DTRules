// Copyright 2026 DTRules contributors
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
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sort"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
	el "github.com/DTRules/DTRules/pkg/dtrules/compiler/el"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/excel"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
	"github.com/DTRules/DTRules/pkg/dtrules/trace"
)

// handleDebugSpeculate reruns the loaded trace's execution against a
// speculative edit of ONE decision table: the project rules are copied to
// a scratch overlay, the edited table's DSL is applied and compiled, the
// original trace's recorded initial data seeds a fresh session (the SAME
// inputs — no input file needed), and the entry table executes with
// tracing. The speculative trace becomes the active debug session; the
// original stays as the baseline for reports, diffs, and restore. Project
// files are never touched.
//
// POST /api/debug/speculate   {<DecisionTableData>}
// POST /api/debug/speculate/reset
func (s *Server) handleDebugSpeculate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var table DecisionTableData
	if err := s.limitedDecode(w, r, &table); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if table.TableName == "" {
		jsonError(w, "table needs a tableName", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.debug == nil {
		jsonError(w, "No trace loaded", http.StatusBadRequest)
		return
	}

	// Speculations always build on the ORIGINAL rules + this one edit —
	// re-speculating replaces the previous speculation, it doesn't stack.
	baselineTrace := s.debug.trace
	baselinePath := s.debug.tracePath
	if s.debug.baseline != nil {
		baselineTrace = s.debug.baseline
		baselinePath = s.debug.baselinePath
	}

	specPath, err := s.runSpeculation(baselinePath, &table)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	ds, err := s.loadDebugSessionLocked(specPath)
	if err != nil {
		jsonError(w, fmt.Sprintf("speculative trace failed to load: %v", err), http.StatusInternalServerError)
		return
	}
	ds.speculative = true
	ds.baseline = baselineTrace
	ds.baselinePath = baselinePath
	// The overlay's rules legitimately differ from the project's — the
	// SPECULATIVE flag carries that message; a scary mismatch chip would
	// mislead.
	ds.fingerprintMatch = "speculative"

	jsonResponse(w, debugSessionPayload(ds))
}

// handleDebugSpeculateReset restores the baseline trace session.
func (s *Server) handleDebugSpeculateReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.debug == nil || s.debug.baseline == nil {
		jsonError(w, "No speculation active", http.StatusBadRequest)
		return
	}
	ds, err := s.loadDebugSessionLocked(s.debug.baselinePath)
	if err != nil {
		jsonError(w, fmt.Sprintf("restore failed: %v", err), http.StatusInternalServerError)
		return
	}
	jsonResponse(w, debugSessionPayload(ds))
}

// runSpeculation builds the overlay rules with the table edit applied,
// seeds a session from the baseline trace's initial data, executes the
// entry table with tracing, and returns the speculative trace path.
// Caller holds s.mu.
func (s *Server) runSpeculation(baselinePath string, table *DecisionTableData) (string, error) {
	// 1. Overlay: copy the resolved rules dir.
	srcDir := projectXMLDir(s.projectPath)
	overlay, err := os.MkdirTemp("", "dtrules-spec-*")
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".xml") {
			continue
		}
		data, rerr := os.ReadFile(filepath.Join(srcDir, e.Name()))
		if rerr != nil {
			return "", rerr
		}
		if werr := os.WriteFile(filepath.Join(overlay, e.Name()), data, 0o644); werr != nil {
			return "", werr
		}
	}

	// 2. Apply the edit (and compile its DSL) into the overlay.
	if err := applySpeculativeEdit(overlay, table); err != nil {
		return "", err
	}

	// 3. Modified ruleset.
	rs := session.NewRuleSet("speculative")
	if err := rs.LoadFromDirectory(overlay); err != nil {
		return "", fmt.Errorf("load modified rules: %v", err)
	}

	// 4. Seed from the baseline trace's recorded initial data: a fresh
	// copy of the trace replayed to the first decisiontable node.
	tr := trace.NewTrace()
	root, err := tr.Load(baselinePath)
	if err != nil {
		return "", fmt.Errorf("reload baseline trace: %v", err)
	}
	entryNode := firstDecisionTable(root)
	if entryNode == nil {
		return "", fmt.Errorf("baseline trace has no decision table execution")
	}
	entry := entryNode.Attributes["name"]
	sess, err := tr.SetState(rs, entryNode)
	if err != nil {
		return "", fmt.Errorf("seed initial data: %v", err)
	}
	state, ok := sess.GetState().(*interpreter.DTState)
	if !ok {
		return "", fmt.Errorf("unexpected state type %T", sess.GetState())
	}

	// The seeded session must allocate NEW ids above everything the
	// baseline recorded — otherwise entities/arrays created during the
	// speculative run collide with spliced initial-data ids and the trace
	// cannot replay.
	if fac, ok := sess.GetEntityFactory().(*entity.Factory); ok {
		fac.EnsureUniqueIDAbove(maxRecordedID(root))
	}

	// 5. Execute the entry table with tracing into the overlay. The
	// baseline's initial-data section (everything between the header and
	// the first decisiontable) is spliced in verbatim: the seeding replay
	// above ran untraced, and without those defs the speculative trace
	// could neither verify nor replay the input data.
	specPath := filepath.Join(overlay, "speculative.trace.xml")
	f, err := os.Create(specPath)
	if err != nil {
		return "", err
	}
	fingerprint, _ := trace.FingerprintRules(overlay)
	trace.WriteHeader(f, trace.Provenance{
		DTRulesVersion:   s.debug.provenance.DTRulesVersion,
		RulesFingerprint: fingerprint,
	})
	if raw, rerr := os.ReadFile(baselinePath); rerr == nil {
		text := string(raw)
		start := strings.Index(text, "\n")
		end := strings.Index(text, "<decisiontable")
		if start >= 0 && end > start {
			if _, werr := f.WriteString(text[start+1 : end]); werr != nil {
				f.Close()
				return "", werr
			}
		}
	}
	state.SetOutput(f, nil)
	state.EnableTrace()

	rsess, ok := sess.(*session.RSession)
	if !ok {
		f.Close()
		return "", fmt.Errorf("unexpected session type %T", sess)
	}
	execErr := rsess.Execute(entry)
	trace.WriteFinalState(f, state)
	trace.WriteFooter(f)
	f.Close()
	if execErr != nil {
		return "", fmt.Errorf("speculative run failed: %v", execErr)
	}
	return specPath, nil
}

// applySpeculativeEdit replaces one table's editable fields in the overlay
// XML and recompiles its DSL rows (conditions and actions) so the rerun
// executes the edit, not the stale postfix.
func applySpeculativeEdit(overlay string, table *DecisionTableData) error {
	files, _ := filepath.Glob(filepath.Join(overlay, "*_dt.xml"))
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		doc, err := excel.UnmarshalDecisionTablesXML(data)
		if err != nil {
			continue
		}
		for i := range doc.Tables {
			if !strings.EqualFold(doc.Tables[i].TableName, table.TableName) {
				continue
			}
			applyTableEdits(&doc.Tables[i], table)
			if err := compileTableDSL(overlay, &doc.Tables[i]); err != nil {
				return err
			}
			imp := excel.NewDTImporter()
			return imp.WriteXML(doc, path)
		}
	}
	return fmt.Errorf("table %q not found in the rules", table.TableName)
}

// compileTableDSL recompiles every row of one table from its DSL, with the
// overlay's EDD symbols for type-aware dispatch. Rows with no DSL keep their
// existing postfix — that is how a hand-coded row survives, and it is the only
// case where keeping the stored postfix is correct.
//
// All four sections are covered. Contexts and initial actions were not, which
// was survivable while this only served speculation but is not once a save
// depends on it (#928): a table whose iteration lives in a context would have
// been written with its edited context DSL against the previous postfix.
func compileTableDSL(overlay string, x *excel.DecisionTableXML) error {
	c := el.NewCompiler()
	if syms := authoring.LoadEDDSymbols(overlay); len(syms) > 0 {
		c.SetSymbols(syms)
	}
	for i := range x.Contexts.Details {
		dsl := strings.TrimSpace(x.Contexts.Details[i].DSL)
		if dsl == "" {
			continue
		}
		pf, err := c.CompileContext(dsl)
		if err != nil {
			return fmt.Errorf("context %d: %v", i+1, err)
		}
		x.Contexts.Details[i].Postfix = pf
	}
	initial := x.EffectiveInitialActions()
	for i := range initial {
		dsl := strings.TrimSpace(initial[i].DSL)
		if dsl == "" {
			continue
		}
		pf, err := c.CompileAction(dsl)
		if err != nil {
			return fmt.Errorf("initial action %d: %v", i+1, err)
		}
		initial[i].Postfix = pf
	}
	for i := range x.Conditions {
		dsl := strings.TrimSpace(x.Conditions[i].DSL)
		if dsl == "" {
			continue
		}
		pf, err := c.CompileCondition(dsl)
		if err != nil {
			return fmt.Errorf("condition %d: %v", i+1, err)
		}
		x.Conditions[i].Postfix = pf
	}
	for i := range x.Actions {
		dsl := strings.TrimSpace(x.Actions[i].DSL)
		if dsl == "" {
			continue
		}
		pf, err := c.CompileAction(dsl)
		if err != nil {
			return fmt.Errorf("action %d: %v", i+1, err)
		}
		x.Actions[i].Postfix = pf
	}
	return nil
}

// maxRecordedID scans every id / arrayId attribute in the trace and
// returns the largest numeric value.
func maxRecordedID(n *trace.TraceNode) int {
	max := 0
	var walk func(x *trace.TraceNode)
	walk = func(x *trace.TraceNode) {
		for _, key := range []string{"id", "arrayId"} {
			if v := x.Attributes[key]; v != "" {
				if id, err := strconv.Atoi(v); err == nil && id > max {
					max = id
				}
			}
		}
		for _, c := range x.Children {
			walk(c)
		}
	}
	walk(n)
	return max
}

// firstDecisionTable finds the first decisiontable node in the trace —
// the entry table of the recorded run.
func firstDecisionTable(n *trace.TraceNode) *trace.TraceNode {
	if n.Name == "decisiontable" {
		return n
	}
	for _, c := range n.Children {
		if r := firstDecisionTable(c); r != nil {
			return r
		}
	}
	return nil
}

// compileError marks a save that failed because the author's DSL would not
// compile, as opposed to an I/O or parse failure. The distinction is what lets
// the editor answer "fix your rule" rather than "the server broke" (#928).
type compileError struct {
	Table string
	Err   error
}

func (e *compileError) Error() string { return e.Table + ": " + e.Err.Error() }
func (e *compileError) Unwrap() error { return e.Err }

// The merge helpers below serve ONLY speculation: they overlay the editor's
// unsaved rows onto a copy of the canonical table so "Run speculation" sees
// what the author sees. The save path stopped using them when it collapsed
// onto authoring.Project (#1084) — a real save reconciles through the
// authoring mutations instead.

// applyTableEdits overwrites the fields the editor can edit — type, comments,
// table number, and the condition/action rows — on a speculation copy.
func applyTableEdits(x *excel.DecisionTableXML, t *DecisionTableData) {
	if t == nil {
		return
	}
	x.AttributeFields.Type = t.Type
	x.AttributeFields.Comments = t.Comments
	x.AttributeFields.TableNumber = t.TableNumber

	x.Conditions = mergeConditionRows(x.Conditions, t.Conditions)
	x.Actions = mergeActionRows(x.Actions, t.Actions)
}

func mergeConditionRows(existing []excel.ConditionXML, rows []ConditionData) []excel.ConditionXML {
	out := make([]excel.ConditionXML, len(rows))
	for i, r := range rows {
		if i < len(existing) {
			out[i] = existing[i] // preserve postfix and any legacy fields
		}
		out[i].Number = strconv.Itoa(r.Number)
		out[i].Comment = r.Comment
		out[i].DSL = r.Description
		out[i].Columns = columnValues(r.Columns)
	}
	return out
}

func mergeActionRows(existing []excel.ActionXML, rows []ActionData) []excel.ActionXML {
	out := make([]excel.ActionXML, len(rows))
	for i, r := range rows {
		if i < len(existing) {
			out[i] = existing[i] // preserve postfix and any legacy fields
		}
		out[i].Number = strconv.Itoa(r.Number)
		out[i].Comment = r.Comment
		out[i].DSL = r.Description
		out[i].Columns = columnValues(r.Columns)
	}
	return out
}

// columnValues converts the editor's cell map to the XML column list, sorted
// by column number so output is deterministic.
func columnValues(cols map[string]string) []excel.ColumnValueXML {
	nums := make([]int, 0, len(cols))
	for k := range cols {
		if n, err := strconv.Atoi(strings.TrimSpace(k)); err == nil {
			nums = append(nums, n)
		}
	}
	sort.Ints(nums)
	out := make([]excel.ColumnValueXML, 0, len(nums))
	for _, n := range nums {
		v := strings.TrimSpace(cols[strconv.Itoa(n)])
		if v == "" {
			continue
		}
		out = append(out, excel.ColumnValueXML{Number: n, Value: v})
	}
	return out
}
