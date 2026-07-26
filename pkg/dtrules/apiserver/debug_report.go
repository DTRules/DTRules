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
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/trace"
)

// handleDebugReport runs a report spec against the active debug trace —
// and, when a speculative session is active, against the baseline too,
// returning the row-level diff.
// POST /api/debug/report {spec}
func (s *Server) handleDebugReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var spec trace.ReportSpec
	if err := s.limitedDecode(w, r, &spec); err != nil {
		jsonError(w, "Invalid report spec", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.debug == nil {
		jsonError(w, "No trace loaded", http.StatusBadRequest)
		return
	}

	report, err := s.runReportLocked(s.debug.trace, spec)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := map[string]interface{}{"success": true, "report": report}

	// Speculative session: run the same spec on the baseline and diff.
	if s.debug.baseline != nil {
		if base, err := s.runReportLocked(s.debug.baseline, spec); err == nil {
			resp["baseline"] = base
			resp["diff"] = trace.DiffReports(base, report)
		}
	}
	jsonResponse(w, resp)
}

// runReportLocked replays tr to its end in a fresh session and generates
// the report. Caller holds s.mu (read).
func (s *Server) runReportLocked(tr *trace.Trace, spec trace.ReportSpec) (*trace.Report, error) {
	fs := tr.FinalState()
	if fs == nil {
		return nil, fmt.Errorf("trace records no finalState")
	}
	sess, err := tr.SetState(s.ruleSet, fs)
	if err != nil {
		return nil, fmt.Errorf("replay failed: %v", err)
	}
	return tr.GenerateReport(sess, spec), nil
}

// reportSpecName validates saveable spec names: word characters and
// dashes only, no path structure.
var reportSpecName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// handleReportSpecs lists and saves report specs under <project>/reports/.
// GET  /api/reports          → {"specs": [{"name", "spec"}]}
// POST /api/reports          → {"name": "...", "spec": {...}} saves
func (s *Server) handleReportSpecs(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	projectPath := s.projectPath
	s.mu.RUnlock()
	if projectPath == "" {
		jsonError(w, "No project open", http.StatusBadRequest)
		return
	}
	dir := filepath.Join(projectPath, "reports")

	switch r.Method {
	case "GET":
		specs := []map[string]interface{}{}
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".report.json") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			var spec trace.ReportSpec
			if json.Unmarshal(data, &spec) != nil {
				continue
			}
			specs = append(specs, map[string]interface{}{
				"name": strings.TrimSuffix(e.Name(), ".report.json"),
				"spec": spec,
			})
		}
		jsonResponse(w, map[string]interface{}{"success": true, "specs": specs})

	case "POST":
		if s.cfg.ReadOnly {
			jsonError(w, "Server is read-only", http.StatusForbidden)
			return
		}
		var req struct {
			Name string           `json:"name"`
			Spec trace.ReportSpec `json:"spec"`
		}
		if err := s.limitedDecode(w, r, &req); err != nil {
			jsonError(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if !reportSpecName.MatchString(req.Name) {
			jsonError(w, "Report name must be letters, digits, _ or -", http.StatusBadRequest)
			return
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data, _ := json.MarshalIndent(req.Spec, "", "  ")
		if err := os.WriteFile(filepath.Join(dir, req.Name+".report.json"), data, 0o644); err != nil {
			jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonResponse(w, map[string]interface{}{"success": true, "name": req.Name})

	default:
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
