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

// Package apiserver implements the DTRules REST API server backing the UI.
// It provides HTTP endpoints to:
// - Open and manage projects
// - Edit entities (EDD)
// - Edit decision tables (DT)
// - Compile and validate expressions
// - Execute rules with optional tracing
//
// cmd/api runs it standalone; `dtrules edit` embeds it alongside the
// static UI bundle.
package apiserver

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/excel"
	"github.com/DTRules/DTRules/pkg/dtrules/compiler"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// Config carries the server's runtime options.
type Config struct {
	// Port to listen on (Run only).
	Port int
	// ProjectRoot, when set, restricts project access to this directory tree.
	ProjectRoot string
	// CORSOrigin is the allowed CORS origin ("*" for development; irrelevant
	// when the UI is served same-origin by `dtrules edit`).
	CORSOrigin string
	// MaxBodySize caps request body size in bytes (default 10MB when zero).
	MaxBodySize int64
	// ReadOnly rejects every mutating request and file browsing, for
	// publishing rules for review. Reads, compilation, and execution
	// (which persists nothing) stay available.
	ReadOnly bool
}

func (c Config) maxBodySize() int64 {
	if c.MaxBodySize > 0 {
		return c.MaxBodySize
	}
	return 10 << 20
}

// Server holds the API server state
type Server struct {
	cfg           Config
	mu            sync.RWMutex
	projectPath   string
	ruleSet       *session.RuleSet
	eddFiles      []string
	dtFiles       []string
	mapFiles      []string
	// entities and tables are slices, not maps: they preserve the authored
	// document order (which the XML and Excel sheets express), and at this
	// scale linear name lookup beats maintaining a parallel index.
	entities      []*EntityData
	tables        []*DecisionTableData
	modified      map[string]bool
	entityFactory *entity.Factory

	// debug is the active trace-debugging session, if any (one per server).
	debug *debugSession
}

// findEntity returns the entity with the given name, or nil. EL names are
// case-insensitive (authored case is preserved for display only).
func (s *Server) findEntity(name string) *EntityData {
	for _, e := range s.entities {
		if strings.EqualFold(e.Name, name) {
			return e
		}
	}
	return nil
}

// findTable returns the decision table with the given name, or nil. EL
// names are case-insensitive (authored case is preserved for display only).
func (s *Server) findTable(name string) *DecisionTableData {
	for _, t := range s.tables {
		if strings.EqualFold(t.TableName, name) {
			return t
		}
	}
	return nil
}

// upsertEntity replaces an existing entity in place (keeping its position and
// source file) or appends a new one recorded against the given source file.
func (s *Server) upsertEntity(e *EntityData, source string) {
	if existing := s.findEntity(e.Name); existing != nil {
		src := existing.Source
		*existing = *e
		existing.Source = src
		return
	}
	e.Source = source
	s.entities = append(s.entities, e)
}

// upsertTable replaces an existing table in place (keeping its position and
// source file) or appends a new one recorded against the given source file.
func (s *Server) upsertTable(t *DecisionTableData, source string) {
	if existing := s.findTable(t.TableName); existing != nil {
		src := existing.Source
		*existing = *t
		existing.Source = src
		return
	}
	t.Source = source
	s.tables = append(s.tables, t)
}

// removeEntity deletes the named entity, returning it (or nil if absent).
func (s *Server) removeEntity(name string) *EntityData {
	for i, e := range s.entities {
		if strings.EqualFold(e.Name, name) {
			s.entities = append(s.entities[:i], s.entities[i+1:]...)
			return e
		}
	}
	return nil
}

// removeTable deletes the named table, returning it (or nil if absent).
func (s *Server) removeTable(name string) *DecisionTableData {
	for i, t := range s.tables {
		if strings.EqualFold(t.TableName, name) {
			s.tables = append(s.tables[:i], s.tables[i+1:]...)
			return t
		}
	}
	return nil
}

// shiftTableNumbersFrom makes room at number n: when any other table already
// occupies n, every table (except keepName) with a numeric number >= n is
// shifted down by 100. Relative order is preserved, so no new collisions.
// Shifted tables' source files are marked modified.
func (s *Server) shiftTableNumbersFrom(n int, keepName string) {
	occupied := false
	for _, t := range s.tables {
		if !strings.EqualFold(t.TableName, keepName) {
			if num, err := strconv.Atoi(strings.TrimSpace(t.TableNumber)); err == nil && num == n {
				occupied = true
				break
			}
		}
	}
	if !occupied {
		return
	}
	for _, t := range s.tables {
		if strings.EqualFold(t.TableName, keepName) {
			continue
		}
		if num, err := strconv.Atoi(strings.TrimSpace(t.TableNumber)); err == nil && num >= n {
			t.TableNumber = strconv.Itoa(num + 100)
			if t.Source != "" {
				s.modified[t.Source] = true
			}
		}
	}
}

// reorderRequest is the body of the drag-and-drop reorder endpoints: the
// full list of names in the desired order.
type reorderRequest struct {
	Order []string `json:"order"`
}

// handleDTReorder renumbers tables to match the given order: 100, 200, ...
// POST /api/dt/reorder {"order": ["name", ...]}
func (s *Server) handleDTReorder(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req reorderRequest
	if err := s.limitedDecode(w, r, &req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	for i, name := range req.Order {
		if t := s.findTable(name); t != nil {
			num := strconv.Itoa((i + 1) * 100)
			if t.TableNumber != num {
				t.TableNumber = num
				if t.Source != "" {
					s.modified[t.Source] = true
				}
			}
		}
	}
	s.mu.Unlock()

	jsonResponse(w, map[string]interface{}{"success": true})
}

// handleEDDReorder renumbers entities to match the given order: 100, 200, ...
// POST /api/edd/reorder {"order": ["name", ...]}
func (s *Server) handleEDDReorder(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req reorderRequest
	if err := s.limitedDecode(w, r, &req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	for i, name := range req.Order {
		if e := s.findEntity(name); e != nil {
			num := strconv.Itoa((i + 1) * 100)
			if e.Number != num {
				e.Number = num
				if e.Source != "" {
					s.modified[e.Source] = true
				}
			}
		}
	}
	s.mu.Unlock()

	jsonResponse(w, map[string]interface{}{"success": true})
}

// shiftEntityNumbersFrom is the entity counterpart of shiftTableNumbersFrom.
func (s *Server) shiftEntityNumbersFrom(n int, keepName string) {
	occupied := false
	for _, e := range s.entities {
		if !strings.EqualFold(e.Name, keepName) {
			if num, err := strconv.Atoi(strings.TrimSpace(e.Number)); err == nil && num == n {
				occupied = true
				break
			}
		}
	}
	if !occupied {
		return
	}
	for _, e := range s.entities {
		if strings.EqualFold(e.Name, keepName) {
			continue
		}
		if num, err := strconv.Atoi(strings.TrimSpace(e.Number)); err == nil && num >= n {
			e.Number = strconv.Itoa(num + 100)
			if e.Source != "" {
				s.modified[e.Source] = true
			}
		}
	}
}

// EntityData represents an entity for the API
type EntityData struct {
	Name    string      `json:"name"`
	Number  string      `json:"number"`
	Access  string      `json:"access"`
	Comment string      `json:"comment"`
	Fields  []FieldData `json:"fields"`
	// Source is the relative path of the EDD file this entity came from.
	Source string `json:"-"`
}

// FieldData represents a field for the API
type FieldData struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Subtype      string `json:"subtype"`
	Access       string `json:"access"`
	Input        string `json:"input"`
	DefaultValue string `json:"defaultValue"`
	Comment      string `json:"comment"`
}

// DecisionTableData represents a decision table for the API
type DecisionTableData struct {
	TableName        string                   `json:"tableName"`
	XlsFile          string                   `json:"xlsFile"`
	Type             string                   `json:"type"`
	Comments         string                   `json:"comments"`
	TableNumber      string                   `json:"tableNumber"`
	Contexts         []ContextData            `json:"contexts"`
	InitialActions   string                   `json:"initialActions"`
	Conditions       []ConditionData          `json:"conditions"`
	Actions          []ActionData             `json:"actions"`
	PolicyStatements []PolicyStatementData    `json:"policyStatements"`
	ColumnCount      int                      `json:"columnCount"`
	// Source is the relative path of the DT file this table came from.
	Source string `json:"-"`
}

// ContextData represents a context for the API
type ContextData struct {
	Number      int    `json:"number"`
	Comment     string `json:"comment"`
	Description string `json:"description"`
	Postfix     string `json:"postfix"`
}

// ConditionData represents a condition for the API
type ConditionData struct {
	Number      int               `json:"number"`
	Comment     string            `json:"comment"`
	Requirement string            `json:"requirement"`
	Description string            `json:"description"`
	Postfix     string            `json:"postfix"`
	Columns     map[string]string `json:"columns"`
}

// ActionData represents an action for the API
type ActionData struct {
	Number      int               `json:"number"`
	Comment     string            `json:"comment"`
	Requirement string            `json:"requirement"`
	Description string            `json:"description"`
	Postfix     string            `json:"postfix"`
	Columns     map[string]string `json:"columns"`
}

// PolicyStatementData represents a policy statement for the API
type PolicyStatementData struct {
	Column      string `json:"column"`
	Description string `json:"description"`
	Postfix     string `json:"postfix"`
}

// TreeNode represents a node in the decision tree visualization
type TreeNode struct {
	ID          string      `json:"id"`
	Type        string      `json:"type"`
	Label       string      `json:"label"`
	Description string      `json:"description,omitempty"`
	Column      int         `json:"column,omitempty"`
	TrueChild   *TreeNode   `json:"trueChild,omitempty"`
	FalseChild  *TreeNode   `json:"falseChild,omitempty"`
	Children    []*TreeNode `json:"children,omitempty"`
}

// FileInfo represents file information for the API
type FileInfo struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Type     string `json:"type"`
	Modified bool   `json:"modified"`
}

// New creates an API server with the given configuration.
func New(cfg Config) *Server {
	return &Server{
		cfg:      cfg,
		modified: make(map[string]bool),
	}
}

// Routes returns the server's /api/* handler, CORS-wrapped.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/api/health", s.handleHealth)

	// Sample projects discovery
	mux.HandleFunc("/api/samples", s.handleSamples)

	// Directory browsing for the UI's project picker
	mux.HandleFunc("/api/browse", s.handleBrowse)

	// Project endpoints
	mux.HandleFunc("/api/project/open", s.handleProjectOpen)
	mux.HandleFunc("/api/project/save", s.handleProjectSave)
	mux.HandleFunc("/api/project/files", s.handleProjectFiles)
	mux.HandleFunc("/api/project/current", s.handleProjectCurrent)

	// EDD endpoints
	mux.HandleFunc("/api/edd", s.handleEDD)
	mux.HandleFunc("/api/edd/entity/", s.handleEntity)
	mux.HandleFunc("/api/edd/reorder", s.handleEDDReorder)

	// DT endpoints
	mux.HandleFunc("/api/dt", s.handleDTList)
	mux.HandleFunc("/api/dt/", s.handleDT)
	mux.HandleFunc("/api/dt/reorder", s.handleDTReorder)

	// Compile endpoints
	mux.HandleFunc("/api/compile/expression", s.handleCompileExpression)
	mux.HandleFunc("/api/compile/operators", s.handleGetOperators)
	mux.HandleFunc("/api/compile/fields", s.handleGetFields)

	// Execute endpoints
	mux.HandleFunc("/api/execute", s.handleExecute)
	mux.HandleFunc("/api/execute/validate", s.handleValidateExecution)

	// Trace debugger endpoints
	mux.HandleFunc("/api/debug/load", s.handleDebugLoad)
	mux.HandleFunc("/api/debug/status", s.handleDebugStatus)
	mux.HandleFunc("/api/debug/tree", s.handleDebugTree)
	mux.HandleFunc("/api/debug/position", s.handleDebugPosition)
	mux.HandleFunc("/api/debug/console", s.handleDebugConsole)

	origin := s.cfg.CORSOrigin
	if origin == "" {
		origin = "*"
	}
	var handler http.Handler = mux
	if s.cfg.ReadOnly {
		handler = readOnlyGuard(handler)
	}
	return corsMiddleware(handler, origin)
}

// readOnlyGuard rejects mutating requests and file browsing. Non-GET
// requests are denied except execution and expression compilation, which
// persist nothing and are part of reviewing rules.
func readOnlyGuard(next http.Handler) http.Handler {
	allowedPost := map[string]bool{
		"/api/execute":            true,
		"/api/execute/validate":   true,
		"/api/compile/expression": true,
		// Trace debugging is a read operation: replay mutates only an
		// in-memory sandbox session, and the console blocks mutating ops.
		"/api/debug/load":     true,
		"/api/debug/position": true,
		"/api/debug/console":  true,
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/browse" {
			jsonError(w, "Server is read-only", http.StatusForbidden)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodOptions && !allowedPost[r.URL.Path] {
			jsonError(w, "Server is read-only", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Run serves the API on cfg.Port until the listener fails.
func Run(cfg Config) error {
	s := New(cfg)
	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("DTRules API server starting on http://localhost%s", addr)
	return http.ListenAndServe(addr, s.Routes())
}

// CORS middleware
func corsMiddleware(next http.Handler, origin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// JSON response helpers
func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
	}
}

func jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// limitedDecode decodes JSON from a size-limited request body
func (s *Server) limitedDecode(w http.ResponseWriter, r *http.Request, v interface{}) error {
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.maxBodySize())
	return json.NewDecoder(r.Body).Decode(v)
}

// validateProjectPath validates and resolves a project path, checking for traversal attacks
func (s *Server) validateProjectPath(path string) (string, error) {
	// Resolve to absolute path and clean it
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Resolve symlinks to prevent symlink-based traversal
	resolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", fmt.Errorf("path does not exist: %w", err)
	}

	// If --project-root is set, verify the path is within it
	if s.cfg.ProjectRoot != "" {
		root, err := filepath.Abs(s.cfg.ProjectRoot)
		if err != nil {
			return "", fmt.Errorf("invalid project root: %w", err)
		}
		root, err = filepath.EvalSymlinks(root)
		if err != nil {
			return "", fmt.Errorf("project root does not exist: %w", err)
		}
		// Ensure resolved path is within root (add separator to prevent prefix attacks like /rootExtra)
		if !strings.HasPrefix(resolved+string(filepath.Separator), root+string(filepath.Separator)) && resolved != root {
			return "", fmt.Errorf("path %q is outside allowed project root %q", path, s.cfg.ProjectRoot)
		}
	}

	return resolved, nil
}

// Health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]string{"status": "ok"})
}

// handleBrowse lists a directory for the UI's project picker.
// GET /api/browse?path=<dir> — defaults to the user's home directory.
// Honors --project-root confinement via validateProjectPath.
func (s *Server) handleBrowse(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			jsonError(w, "Cannot determine home directory", http.StatusInternalServerError)
			return
		}
		path = home
	}

	resolved, err := s.validateProjectPath(path)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		jsonError(w, fmt.Sprintf("Not a directory: %s", path), http.StatusBadRequest)
		return
	}

	dirEntries, err := os.ReadDir(resolved)
	if err != nil {
		jsonError(w, fmt.Sprintf("Cannot read directory: %v", err), http.StatusBadRequest)
		return
	}

	type browseEntry struct {
		Name  string `json:"name"`
		Path  string `json:"path"`
		IsDir bool   `json:"isDir"`
		// Size in bytes for files — a picker showing sizes lets the user
		// tell a real trace from a header-only one at a glance.
		Size int64 `json:"size,omitempty"`
	}

	entries := []browseEntry{}
	if parent := filepath.Dir(resolved); parent != resolved {
		if _, err := s.validateProjectPath(parent); err == nil {
			entries = append(entries, browseEntry{Name: "..", Path: parent, IsDir: true})
		}
	}

	var dirs, files []browseEntry
	isProject := false
	for _, e := range dirEntries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		entry := browseEntry{Name: name, Path: filepath.Join(resolved, name), IsDir: e.IsDir()}
		if e.IsDir() {
			dirs = append(dirs, entry)
		} else {
			if fi, err := e.Info(); err == nil {
				entry.Size = fi.Size()
			}
			files = append(files, entry)
			lower := strings.ToLower(name)
			if strings.HasSuffix(lower, "_dt.xml") || strings.HasSuffix(lower, "_edd.xml") {
				isProject = true
			}
		}
	}
	entries = append(entries, dirs...)
	entries = append(entries, files...)

	jsonResponse(w, map[string]interface{}{
		"success":     true,
		"currentPath": resolved,
		"entries":     entries,
		"isProject":   isProject,
	})
}

// handleSamples returns available sample projects
func (s *Server) handleSamples(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Find the DTRules root directory by looking for sampleprojects
	// Start from the executable location and search upward
	execPath, err := os.Executable()
	if err != nil {
		execPath, _ = os.Getwd()
	}

	// Try to find sampleprojects directory
	// When running from go/cmd/api, we need to go up 3 levels to reach DTRules root
	cwd, _ := os.Getwd()
	searchPaths := []string{
		filepath.Dir(execPath),                              // Same dir as executable
		filepath.Join(filepath.Dir(execPath), ".."),         // Parent of executable
		filepath.Join(filepath.Dir(execPath), "../.."),      // Grandparent (go/cmd/api -> go -> DTRules)
		filepath.Join(filepath.Dir(execPath), "../../.."),   // Great-grandparent
		cwd,                                                 // Current working directory
		filepath.Join(cwd, ".."),                            // Parent of cwd
		filepath.Join(cwd, "../.."),                         // Grandparent of cwd
		filepath.Join(cwd, "../../.."),                      // Great-grandparent of cwd (go/cmd/api -> go -> DTRules)
	}

	var samplesDir string
	for _, base := range searchPaths {
		candidate := filepath.Join(base, "sampleprojects")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			samplesDir, _ = filepath.Abs(candidate)
			break
		}
	}

	if samplesDir == "" {
		jsonResponse(w, map[string]interface{}{
			"success": true,
			"samples": []interface{}{},
			"message": "No sample projects found",
		})
		return
	}

	// Scan for sample projects (directories containing xml subdirectory with *_edd.xml files)
	type SampleProject struct {
		Name string `json:"name"`
		Path string `json:"path"`
		Description string `json:"description"`
	}

	var samples []SampleProject

	entries, err := os.ReadDir(samplesDir)
	if err != nil {
		jsonError(w, "Failed to read samples directory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectName := entry.Name()
		xmlDir := filepath.Join(samplesDir, projectName, "xml")

		// Check if xml directory exists and contains EDD files
		if info, err := os.Stat(xmlDir); err == nil && info.IsDir() {
			files, _ := os.ReadDir(xmlDir)
			hasEDD := false
			for _, f := range files {
				if strings.HasSuffix(f.Name(), "_edd.xml") {
					hasEDD = true
					break
				}
			}
			if hasEDD {
				description := ""
				switch projectName {
				case "CHIP":
					description = "Children's Health Insurance Program eligibility rules"
				case "KidAid":
					description = "Child assistance program eligibility rules"
				case "TestProject":
					description = "Minimal template for new projects"
				case "SyntaxTests":
					description = "Expression Language syntax examples"
				}
				samples = append(samples, SampleProject{
					Name: projectName,
					Path: xmlDir,
					Description: description,
				})
			}
		}
	}

	jsonResponse(w, map[string]interface{}{
		"success": true,
		"samples": samples,
	})
}

// Project endpoints
func (s *Server) handleProjectOpen(w http.ResponseWriter, r *http.Request) {
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

	if err := s.LoadProject(req.Path); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	jsonResponse(w, map[string]interface{}{
		"success":  true,
		"config":   s.configPayload(),
		"eddFiles": s.eddFiles,
		"dtFiles":  s.dtFiles,
		"mapFiles": s.mapFiles,
	})
}

// handleProjectCurrent reports the project the server already has loaded
// (e.g. one passed to `dtrules edit` at startup), so a fresh browser session
// can adopt it instead of showing the welcome screen.
func (s *Server) handleProjectCurrent(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	jsonResponse(w, map[string]interface{}{
		"success":  true,
		"path":     s.projectPath,
		"config":   s.configPayload(),
		"eddFiles": s.eddFiles,
		"dtFiles":  s.dtFiles,
		"mapFiles": s.mapFiles,
		"readOnly": s.cfg.ReadOnly,
	})
}

// LoadProject validates, scans, and loads the project directory at path.
// It is used by the project/open endpoint and by `dtrules edit` to open a
// project at startup.
func (s *Server) LoadProject(reqPath string) error {
	// Validate project path (prevents path traversal)
	validatedPath, err := s.validateProjectPath(reqPath)
	if err != nil {
		return fmt.Errorf("invalid project path: %v", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Scan directory for XML files
	s.projectPath = validatedPath
	s.eddFiles = []string{}
	s.dtFiles = []string{}
	s.mapFiles = []string{}
	s.entities = nil
	s.tables = nil
	s.modified = make(map[string]bool)

	// Scan the project's resolved rules directory (DTRules.xml xml_dir
	// override, else xml/, else the root). Walking the whole tree from a
	// repo root would sweep in test fixtures and generated copies.
	scanRoot := projectXMLDir(validatedPath)
	err = filepath.Walk(scanRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if !strings.HasSuffix(strings.ToLower(path), ".xml") {
			return nil
		}

		relPath, _ := filepath.Rel(validatedPath, path)
		name := filepath.Base(path)

		// Categorize by name pattern
		nameLower := strings.ToLower(name)
		if strings.Contains(nameLower, "edd") {
			s.eddFiles = append(s.eddFiles, relPath)
			s.loadEDDFile(path, relPath)
		} else if strings.Contains(nameLower, "_dt") || strings.Contains(nameLower, "decisiontable") {
			// Skip "Uncompiled" files - they have no postfix expressions and would overwrite valid tables
			if strings.Contains(nameLower, "uncompiled") {
				log.Printf("Skipping uncompiled DT file: %s", relPath)
			} else {
				s.dtFiles = append(s.dtFiles, relPath)
				s.loadDTFile(path, relPath)
			}
		} else if strings.Contains(nameLower, "map") {
			s.mapFiles = append(s.mapFiles, relPath)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan directory: %v", err)
	}

	// Build the executable rule set from the discovered files. Failures
	// are logged but non-fatal — the API server prefers to load whatever
	// is loadable so the UI can show partial state. This is intentionally
	// different from cmd/dtrules (which fatals); the parity tests at
	// load_parity_test.go pin both load surfaces to a common contract.
	s.ruleSet = buildRuleSetFromXML("ui-project", s.projectPath, s.eddFiles, s.dtFiles, logLoadWarning)

	return nil
}

func (s *Server) handleProjectSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	savedFiles := []string{}

	// Save modified EDD files
	for _, eddFile := range s.eddFiles {
		if s.modified[eddFile] {
			path := filepath.Join(s.projectPath, eddFile)
			if err := s.saveEDDFile(path, eddFile); err != nil {
				jsonError(w, fmt.Sprintf("Failed to save %s: %v", eddFile, err), http.StatusInternalServerError)
				return
			}
			savedFiles = append(savedFiles, eddFile)
			s.modified[eddFile] = false
		}
	}

	// Save modified DT files
	for _, dtFile := range s.dtFiles {
		if s.modified[dtFile] {
			path := filepath.Join(s.projectPath, dtFile)
			if err := s.saveDTFile(path, dtFile); err != nil {
				jsonError(w, fmt.Sprintf("Failed to save %s: %v", dtFile, err), http.StatusInternalServerError)
				return
			}
			savedFiles = append(savedFiles, dtFile)
			s.modified[dtFile] = false
		}
	}

	jsonResponse(w, map[string]interface{}{
		"success":    true,
		"savedFiles": savedFiles,
	})
}

func (s *Server) handleProjectFiles(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files := []FileInfo{}

	for _, f := range s.eddFiles {
		files = append(files, FileInfo{
			Name:     filepath.Base(f),
			Path:     f,
			Type:     "edd",
			Modified: s.modified[f],
		})
	}

	for _, f := range s.dtFiles {
		files = append(files, FileInfo{
			Name:     filepath.Base(f),
			Path:     f,
			Type:     "dt",
			Modified: s.modified[f],
		})
	}

	for _, f := range s.mapFiles {
		files = append(files, FileInfo{
			Name:     filepath.Base(f),
			Path:     f,
			Type:     "map",
			Modified: s.modified[f],
		})
	}

	jsonResponse(w, map[string]interface{}{
		"success": true,
		"files":   files,
	})
}

// EDD endpoints
func (s *Server) handleEDD(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Present entities sorted by number (then document order for the
	// unnumbered); sort a copy so s.entities keeps its document order.
	entities := append([]*EntityData{}, s.entities...)
	sort.SliceStable(entities, func(i, j int) bool {
		ni, ei := strconv.Atoi(strings.TrimSpace(entities[i].Number))
		nj, ej := strconv.Atoi(strings.TrimSpace(entities[j].Number))
		switch {
		case ei == nil && ej == nil:
			return ni < nj
		case ei == nil:
			return true
		default:
			return false
		}
	})

	jsonResponse(w, map[string]interface{}{
		"success":  true,
		"entities": entities,
	})
}

func (s *Server) handleEntity(w http.ResponseWriter, r *http.Request) {
	// Extract entity name from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/edd/entity/")
	entityName := strings.TrimSuffix(path, "/")

	switch r.Method {
	case "GET":
		s.mu.RLock()
		entity := s.findEntity(entityName)
		s.mu.RUnlock()

		if entity == nil {
			jsonError(w, "Entity not found", http.StatusNotFound)
			return
		}
		jsonResponse(w, map[string]interface{}{
			"success": true,
			"entity":  entity,
		})

	case "POST":
		var entity EntityData
		if err := s.limitedDecode(w, r, &entity); err != nil {
			jsonError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		s.mu.Lock()
		// New entities are recorded against the first EDD file
		source := ""
		if len(s.eddFiles) > 0 {
			source = s.eddFiles[0]
			s.modified[source] = true
		}
		s.upsertEntity(&entity, source)
		s.mu.Unlock()

		jsonResponse(w, map[string]interface{}{
			"success": true,
			"message": "Entity created",
		})

	case "PUT":
		var entity EntityData
		if err := s.limitedDecode(w, r, &entity); err != nil {
			jsonError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		s.mu.Lock()
		if existing := s.findEntity(entityName); existing != nil {
			oldNumber := existing.Number
			src := existing.Source
			*existing = entity
			existing.Source = src
			if src != "" {
				s.modified[src] = true
			} else if len(s.eddFiles) > 0 {
				existing.Source = s.eddFiles[0]
				s.modified[s.eddFiles[0]] = true
			}
			// Renumbering makes room: entities at/after the new number shift down
			if entity.Number != oldNumber {
				if n, err := strconv.Atoi(strings.TrimSpace(entity.Number)); err == nil {
					s.shiftEntityNumbersFrom(n, entityName)
				}
			}
		} else {
			source := ""
			if len(s.eddFiles) > 0 {
				source = s.eddFiles[0]
				s.modified[source] = true
			}
			s.upsertEntity(&entity, source)
		}
		s.mu.Unlock()

		jsonResponse(w, map[string]interface{}{
			"success": true,
			"message": "Entity updated",
		})

	case "DELETE":
		s.mu.Lock()
		removed := s.removeEntity(entityName)
		if removed != nil && removed.Source != "" {
			s.modified[removed.Source] = true
		} else if len(s.eddFiles) > 0 {
			s.modified[s.eddFiles[0]] = true
		}
		s.mu.Unlock()

		jsonResponse(w, map[string]interface{}{
			"success": true,
			"message": "Entity deleted",
		})

	default:
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// DT endpoints
func (s *Server) handleDTList(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Create new table
		var table DecisionTableData
		if err := s.limitedDecode(w, r, &table); err != nil {
			jsonError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		s.mu.Lock()
		// New tables are recorded against the first DT file
		source := ""
		if len(s.dtFiles) > 0 {
			source = s.dtFiles[0]
			s.modified[source] = true
		}
		s.upsertTable(&table, source)
		s.mu.Unlock()

		jsonResponse(w, map[string]interface{}{
			"success": true,
			"message": "Table created",
		})
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Present tables sorted by TABLE_NUMBER (then name), like the Excel
	// export; sort a copy so s.tables keeps its document order.
	ordered := append([]*DecisionTableData(nil), s.tables...)
	sort.Slice(ordered, func(i, j int) bool {
		ni, ei := strconv.Atoi(strings.TrimSpace(ordered[i].TableNumber))
		nj, ej := strconv.Atoi(strings.TrimSpace(ordered[j].TableNumber))
		switch {
		case ei == nil && ej == nil && ni != nj:
			return ni < nj
		case ei == nil && ej != nil:
			return true // numbered tables before unnumbered
		case ei != nil && ej == nil:
			return false
		default:
			return ordered[i].TableName < ordered[j].TableName
		}
	})

	tables := make([]map[string]interface{}, 0, len(ordered))
	for _, t := range ordered {
		tables = append(tables, map[string]interface{}{
			"name":           t.TableName,
			"tableNumber":    t.TableNumber,
			"type":           t.Type,
			"comments":       t.Comments,
			"conditionCount": len(t.Conditions),
			"actionCount":    len(t.Actions),
			"columnCount":    t.ColumnCount,
		})
	}

	jsonResponse(w, map[string]interface{}{
		"success": true,
		"tables":  tables,
	})
}

func (s *Server) handleDT(w http.ResponseWriter, r *http.Request) {
	// Extract table name and action from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/dt/")
	parts := strings.Split(path, "/")
	tableName := parts[0]

	// Check for tree endpoint
	if len(parts) > 1 && parts[1] == "tree" {
		s.handleDTTree(w, r, tableName)
		return
	}

	switch r.Method {
	case "GET":
		s.mu.RLock()
		table := s.findTable(tableName)
		s.mu.RUnlock()

		if table == nil {
			jsonError(w, "Table not found", http.StatusNotFound)
			return
		}

		response := map[string]interface{}{
			"success":          true,
			"tableName":        table.TableName,
			"xlsFile":          table.XlsFile,
			"type":             table.Type,
			"comments":         table.Comments,
			"tableNumber":      table.TableNumber,
			"contexts":         table.Contexts,
			"initialActions":   table.InitialActions,
			"conditions":       table.Conditions,
			"actions":          table.Actions,
			"policyStatements": table.PolicyStatements,
			"columnCount":      table.ColumnCount,
		}
		jsonResponse(w, response)

	case "PUT":
		var table DecisionTableData
		if err := s.limitedDecode(w, r, &table); err != nil {
			jsonError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		s.mu.Lock()
		if existing := s.findTable(tableName); existing != nil {
			oldNumber := existing.TableNumber
			src := existing.Source
			*existing = table
			existing.Source = src
			if src != "" {
				s.modified[src] = true
			} else if len(s.dtFiles) > 0 {
				existing.Source = s.dtFiles[0]
				s.modified[s.dtFiles[0]] = true
			}
			// Renumbering makes room: tables at/after the new number shift down
			if table.TableNumber != oldNumber {
				if n, err := strconv.Atoi(strings.TrimSpace(table.TableNumber)); err == nil {
					s.shiftTableNumbersFrom(n, tableName)
				}
			}
		} else {
			source := ""
			if len(s.dtFiles) > 0 {
				source = s.dtFiles[0]
				s.modified[source] = true
			}
			s.upsertTable(&table, source)
		}
		s.mu.Unlock()

		jsonResponse(w, map[string]interface{}{
			"success": true,
			"message": "Table updated",
		})

	case "DELETE":
		s.mu.Lock()
		removed := s.removeTable(tableName)
		if removed != nil && removed.Source != "" {
			s.modified[removed.Source] = true
		} else if len(s.dtFiles) > 0 {
			s.modified[s.dtFiles[0]] = true
		}
		s.mu.Unlock()

		jsonResponse(w, map[string]interface{}{
			"success": true,
			"message": "Table deleted",
		})

	default:
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleDTTree(w http.ResponseWriter, r *http.Request, tableName string) {
	s.mu.RLock()
	table := s.findTable(tableName)
	s.mu.RUnlock()

	if table == nil {
		jsonError(w, "Table not found", http.StatusNotFound)
		return
	}

	// Generate tree visualization
	tree := s.generateTree(table)

	jsonResponse(w, map[string]interface{}{
		"success": true,
		"tree":    tree,
	})
}

func (s *Server) generateTree(table *DecisionTableData) *TreeNode {
	root := &TreeNode{
		ID:    "start",
		Type:  "start",
		Label: table.TableName,
	}

	// Build tree from conditions and actions
	if len(table.Conditions) == 0 {
		// No conditions, just actions
		if len(table.Actions) > 0 {
			actionsNode := &TreeNode{
				ID:    "actions",
				Type:  "actions",
				Label: "Actions",
			}
			for i, action := range table.Actions {
				actionsNode.Children = append(actionsNode.Children, &TreeNode{
					ID:          fmt.Sprintf("action_%d", i),
					Type:        "action",
					Label:       action.Description,
					Description: action.Postfix,
				})
			}
			root.Children = []*TreeNode{actionsNode}
		}
		return root
	}

	// Build tree for each column
	for col := 1; col <= table.ColumnCount; col++ {
		colNode := s.buildColumnTree(table, col)
		root.Children = append(root.Children, colNode)
	}

	return root
}

func (s *Server) buildColumnTree(table *DecisionTableData, col int) *TreeNode {
	colStr := fmt.Sprintf("%d", col)
	colNode := &TreeNode{
		ID:     fmt.Sprintf("col_%d", col),
		Type:   "condition",
		Label:  fmt.Sprintf("Column %d", col),
		Column: col,
	}

	// Add conditions for this column
	var current *TreeNode = colNode
	for i, cond := range table.Conditions {
		condVal := cond.Columns[colStr]
		if condVal == "" {
			condVal = "-"
		}

		condNode := &TreeNode{
			ID:          fmt.Sprintf("cond_%d_col_%d", i, col),
			Type:        "condition",
			Label:       cond.Description,
			Description: fmt.Sprintf("%s = %s", cond.Postfix, condVal),
		}

		if condVal == "Y" {
			current.TrueChild = condNode
			current = condNode
		} else if condVal == "N" {
			current.FalseChild = condNode
			current = condNode
		} else {
			current.Children = append(current.Children, condNode)
			current = condNode
		}
	}

	// Add actions for this column
	actionsNode := &TreeNode{
		ID:    fmt.Sprintf("actions_col_%d", col),
		Type:  "actions",
		Label: "Actions",
	}

	for i, action := range table.Actions {
		actionVal := action.Columns[colStr]
		if actionVal == "X" {
			actionsNode.Children = append(actionsNode.Children, &TreeNode{
				ID:          fmt.Sprintf("action_%d_col_%d", i, col),
				Type:        "action",
				Label:       action.Description,
				Description: action.Postfix,
			})
		}
	}

	if len(actionsNode.Children) > 0 {
		if current.TrueChild == nil {
			current.TrueChild = actionsNode
		} else {
			current.Children = append(current.Children, actionsNode)
		}
	}

	return colNode
}

// Compile endpoints
func (s *Server) handleCompileExpression(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Expression string `json:"expression"`
		EntityName string `json:"entityName"`
	}
	if err := s.limitedDecode(w, r, &req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	ruleSet := s.ruleSet
	s.mu.RUnlock()

	if ruleSet == nil {
		// No project loaded, do basic syntax check
		valid := true
		errorMsg := ""

		// Basic validation - check for balanced brackets
		brackets := 0
		braces := 0
		for _, c := range req.Expression {
			switch c {
			case '[':
				brackets++
			case ']':
				brackets--
			case '{':
				braces++
			case '}':
				braces--
			}
		}

		if brackets != 0 {
			valid = false
			errorMsg = "Unbalanced brackets"
		} else if braces != 0 {
			valid = false
			errorMsg = "Unbalanced braces"
		}

		jsonResponse(w, map[string]interface{}{
			"valid": valid,
			"error": errorMsg,
		})
		return
	}

	// Try to compile with actual compiler
	sess, err := ruleSet.NewSession()
	if err != nil {
		jsonResponse(w, map[string]interface{}{
			"valid": true, // Assume valid if we can't compile
		})
		return
	}

	rsess := sess.(*session.RSession)
	c := compiler.NewCompiler(rsess, rsess.GetEntityFactory().(*entity.Factory))

	_, err = c.Compile(req.Expression)
	if err != nil {
		jsonResponse(w, map[string]interface{}{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	jsonResponse(w, map[string]interface{}{
		"valid": true,
	})
}

func (s *Server) handleGetOperators(w http.ResponseWriter, r *http.Request) {
	// Return list of DTRules operators
	operators := []string{
		// Stack operations
		"dup", "pop", "swap", "exch", "roll", "copy", "index", "clear", "count",
		// Math operations
		"+", "-", "*", "/", "mod", "abs", "neg", "min", "max",
		// Comparison operations
		"<", "<=", ">", ">=", "==", "!=", "eq", "ne", "lt", "le", "gt", "ge",
		// Boolean operations
		"and", "or", "not", "xor",
		// Control flow
		"if", "ifelse", "forall", "loop", "exit", "repeat", "for",
		// Entity operations
		"entitypush", "entitypop", "createentity", "findmatch", "findall",
		"allocate", "deallocate",
		// String operations
		"concat", "substring", "length", "uppercase", "lowercase", "trim",
		// Array operations
		"newarray", "addto", "removefrom", "member", "first", "last",
		// Type operations
		"cvs", "cvi", "cvd", "cvb",
		// Misc
		"def", "set", "get", "exec", "print", "debug", "trace",
	}

	jsonResponse(w, map[string]interface{}{
		"operators": operators,
	})
}

func (s *Server) handleGetFields(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fields := []string{}
	for _, entity := range s.entities {
		for _, field := range entity.Fields {
			fields = append(fields, fmt.Sprintf("%s.%s", entity.Name, field.Name))
		}
	}

	jsonResponse(w, map[string]interface{}{
		"fields": fields,
	})
}

// Execute endpoints
func (s *Server) handleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TableName string                 `json:"tableName"`
		Data      map[string]interface{} `json:"data"`
		Trace     bool                   `json:"trace"`
	}
	if err := s.limitedDecode(w, r, &req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	ruleSet := s.ruleSet
	s.mu.RUnlock()

	if ruleSet == nil {
		jsonError(w, "No project loaded", http.StatusBadRequest)
		return
	}

	// Create session
	sess, err := ruleSet.NewSession()
	if err != nil {
		jsonError(w, fmt.Sprintf("Failed to create session: %v", err), http.StatusInternalServerError)
		return
	}

	rsess := sess.(*session.RSession)
	state := rsess.GetState().(*interpreter.DTState)
	factory := rsess.GetEntityFactory().(*entity.Factory)

	// Enable tracing if requested
	if req.Trace {
		state.EnableTrace()
	}

	// Create and push the constants entity if it exists (needed for lookups like CHIP, MEDICAID, etc.)
	constantsName := dtrules.GetRName("constants")
	if constantsName != nil {
		if constantsEntity, err := factory.CreateEntity(rsess, constantsName); err == nil && constantsEntity != nil {
			state.EntityPush(constantsEntity)
		}
	}

	// Load input data into entities
	warnings, err := loadInputData(rsess, state, factory, req.Data)
	if err != nil {
		jsonError(w, fmt.Sprintf("Failed to load input data: %v", err), http.StatusInternalServerError)
		return
	}

	// Log entity stack for debugging
	log.Printf("Entity stack before execute (depth=%d):", state.EntityDepth())
	for i := 0; i < state.EntityDepth(); i++ {
		if ent, err := state.GetEntityStack(i); err == nil && ent != nil {
			log.Printf("  [%d] %s", i, ent.GetName().StringValue())
		}
	}

	// Execute
	if err := rsess.Execute(req.TableName); err != nil {
		// Include more context in error response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Execution failed: %v", err),
			"context": map[string]interface{}{
				"tableName":   req.TableName,
				"stackDepth":  state.DataStackDepth(),
				"entityDepth": state.EntityDepth(),
			},
			"warnings": warnings,
		})
		return
	}

	// Get results - extract entity values from the state
	result := make(map[string]interface{})

	// Extract values from all entities on the entity stack
	for i := 0; i < state.EntityDepth(); i++ {
		ent, err := state.GetEntityStack(i)
		if err != nil || ent == nil {
			continue
		}

		// Get entity as REntity to access attributes
		rentity, ok := ent.(*entity.REntity)
		if !ok {
			continue
		}

		entityName := rentity.GetName().StringValue()
		entityData := make(map[string]interface{})

		// Extract all attribute values
		for _, attrName := range rentity.GetAttributeNames() {
			val, err := rentity.Get(attrName)
			if err != nil || val == nil {
				continue
			}

			// Convert DTRules value to Go value
			entityData[attrName.StringValue()] = convertToGoValue(val)
		}

		if len(entityData) > 0 {
			result[entityName] = entityData
		}
	}

	response := map[string]interface{}{
		"success": true,
		"result":  result,
	}
	if len(warnings) > 0 {
		response["warnings"] = warnings
	}
	jsonResponse(w, response)
}

func (s *Server) handleValidateExecution(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TableName string                 `json:"tableName"`
		Data      map[string]interface{} `json:"data"`
	}
	if err := s.limitedDecode(w, r, &req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	tableExists := s.findTable(req.TableName) != nil
	s.mu.RUnlock()

	if !tableExists {
		jsonError(w, "Table not found", http.StatusNotFound)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"success": true,
		"message": "Validation passed",
	})
}

// loadInputData loads the input JSON data into DTRules entities and pushes them onto the entity stack
// Returns a slice of warnings encountered during loading (non-fatal issues)
func loadInputData(sess *session.RSession, state *interpreter.DTState, factory *entity.Factory, data map[string]interface{}) ([]string, error) {
	var warnings []string

	for key, value := range data {
		switch v := value.(type) {
		case map[string]interface{}:
			// Single entity - load it with its nested arrays
			rentity, err := loadEntityWithArrays(sess, state, factory, key, v, &warnings)
			if err != nil {
				warning := fmt.Sprintf("Failed to load entity '%s': %v", key, err)
				log.Printf("Warning: %s", warning)
				warnings = append(warnings, warning)
				continue
			}
			// Push the entity onto the entity stack
			if err := state.EntityPush(rentity); err != nil {
				warning := fmt.Sprintf("Failed to push entity '%s': %v", key, err)
				log.Printf("Warning: %s", warning)
				warnings = append(warnings, warning)
			}
		case []interface{}:
			// Top-level array of entities (e.g., standalone "clients" array)
			// Try to determine singular form (e.g., "clients" -> "client")
			entityName := key
			if len(key) > 1 && key[len(key)-1] == 's' {
				entityName = key[:len(key)-1]
			}
			for i, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					rentity, err := loadEntityWithArrays(sess, state, factory, entityName, itemMap, &warnings)
					if err != nil {
						warning := fmt.Sprintf("Failed to load %s[%d]: %v", key, i, err)
						log.Printf("Warning: %s", warning)
						warnings = append(warnings, warning)
						continue
					}
					if err := state.EntityPush(rentity); err != nil {
						warning := fmt.Sprintf("Failed to push %s[%d]: %v", key, i, err)
						log.Printf("Warning: %s", warning)
						warnings = append(warnings, warning)
					}
				}
			}
		default:
			// Skip non-object values at top level
			warning := fmt.Sprintf("Skipping top-level non-object value: %s", key)
			log.Printf("%s", warning)
			warnings = append(warnings, warning)
		}
	}
	return warnings, nil
}

// loadEntityWithArrays creates a DTRules entity from a map, handling nested entity arrays
// It returns the created entity but does NOT push it to the stack (caller decides)
func loadEntityWithArrays(sess *session.RSession, state *interpreter.DTState, factory *entity.Factory, entityName string, data map[string]interface{}, warnings *[]string) (*entity.REntity, error) {
	// Get the RName for the entity
	name := dtrules.GetRName(entityName)
	if name == nil {
		return nil, fmt.Errorf("invalid entity name: %s", entityName)
	}

	// Create an entity instance
	ent, err := factory.CreateEntity(sess, name)
	if err != nil {
		return nil, fmt.Errorf("failed to create entity %s: %w", entityName, err)
	}
	if ent == nil {
		return nil, fmt.Errorf("entity type not found: %s", entityName)
	}

	// Get as REntity to set values
	rentity, ok := ent.(*entity.REntity)
	if !ok {
		return nil, fmt.Errorf("entity is not an REntity: %s", entityName)
	}

	// Set attribute values
	for attrKey, attrValue := range data {
		attrName := dtrules.GetRName(attrKey)
		if attrName == nil {
			warning := fmt.Sprintf("Invalid attribute name: %s", attrKey)
			log.Printf("Warning: %s", warning)
			*warnings = append(*warnings, warning)
			continue
		}

		// Check if this is an array of objects (potential entity array)
		if arr, isArray := attrValue.([]interface{}); isArray && len(arr) > 0 {
			if _, isObject := arr[0].(map[string]interface{}); isObject {
				// This is an array of entities - create a DTRules array with entity references
				dtArray, err := loadEntityArray(sess, state, factory, attrKey, arr, warnings)
				if err != nil {
					warning := fmt.Sprintf("Failed to load array %s.%s: %v", entityName, attrKey, err)
					log.Printf("Warning: %s", warning)
					*warnings = append(*warnings, warning)
					continue
				}
				if err := rentity.Put(attrName, dtArray); err != nil {
					warning := fmt.Sprintf("Failed to set array %s.%s: %v", entityName, attrKey, err)
					log.Printf("Warning: %s", warning)
					*warnings = append(*warnings, warning)
				}
				continue
			}
		}

		// Check if this attribute expects a date type
		var dtValue dtrules.Object
		entry := rentity.GetEntry(attrName)
		if entry != nil && entry.Type == dtrules.TypeDate {
			// Try to parse the value as a date
			if strVal, ok := attrValue.(string); ok {
				dateVal, err := dtrules.GetRDate(sess, strVal)
				if err != nil {
					warning := fmt.Sprintf("Failed to parse date %s.%s: %v", entityName, attrKey, err)
					log.Printf("Warning: %s", warning)
					*warnings = append(*warnings, warning)
					continue
				}
				dtValue = dateVal
			} else {
				dtValue = goValueToDTRules(attrValue)
			}
		} else {
			// Convert Go value to DTRules object for simple values
			dtValue = goValueToDTRules(attrValue)
		}
		if dtValue == nil {
			continue
		}

		// Set the value on the entity
		if err := rentity.Put(attrName, dtValue); err != nil {
			warning := fmt.Sprintf("Failed to set %s.%s: %v", entityName, attrKey, err)
			log.Printf("Warning: %s", warning)
			*warnings = append(*warnings, warning)
		}
	}

	return rentity, nil
}

// loadEntityArray creates a DTRules array containing entity references
// It also pushes each created entity onto the entity stack so they're accessible
func loadEntityArray(sess *session.RSession, state *interpreter.DTState, factory *entity.Factory, arrayName string, items []interface{}, warnings *[]string) (dtrules.Object, error) {
	// Determine the entity type name from the array name
	// e.g., "clients" -> "client", "incomes" -> "income"
	entityTypeName := arrayName
	if len(arrayName) > 1 && arrayName[len(arrayName)-1] == 's' {
		entityTypeName = arrayName[:len(arrayName)-1]
	}

	// Create a DTRules array to hold the entity references
	dtArray, err := dtrules.NewArray(sess, true, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create array: %w", err)
	}

	// Process each item in the array
	for i, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			warning := fmt.Sprintf("Array item %d is not an object", i)
			log.Printf("Warning: %s", warning)
			*warnings = append(*warnings, warning)
			continue
		}

		// Create the child entity (with its own nested arrays)
		childEntity, err := loadEntityWithArrays(sess, state, factory, entityTypeName, itemMap, warnings)
		if err != nil {
			warning := fmt.Sprintf("Failed to create %s[%d]: %v", arrayName, i, err)
			log.Printf("Warning: %s", warning)
			*warnings = append(*warnings, warning)
			continue
		}

		// Add the entity reference to the array
		dtArray.Add(childEntity)

		// Push the entity onto the entity stack so it's accessible for lookups
		if err := state.EntityPush(childEntity); err != nil {
			warning := fmt.Sprintf("Failed to push %s[%d] to stack: %v", arrayName, i, err)
			log.Printf("Warning: %s", warning)
			*warnings = append(*warnings, warning)
		}
	}

	return dtArray, nil
}

// goValueToDTRules converts a Go value to a DTRules object
// Note: For arrays of objects (entities), use loadEntityArray instead
func goValueToDTRules(value interface{}) dtrules.Object {
	if value == nil {
		return dtrules.GetRNull()
	}

	switch v := value.(type) {
	case bool:
		return dtrules.GetRBoolean(v)
	case float64:
		// JSON numbers are float64
		// Check if it's actually an integer
		if v == float64(int64(v)) {
			return dtrules.GetRIntegerValue(int64(v))
		}
		return dtrules.GetRDoubleValue(v)
	case int:
		return dtrules.GetRIntegerValue(int64(v))
	case int64:
		return dtrules.GetRIntegerValue(v)
	case string:
		return dtrules.GetRString(v)
	case []interface{}:
		// Simple arrays (strings, numbers) - not entity arrays
		// Note: Entity arrays should be handled by loadEntityArray before this point
		// This handles cases like ["AA", "BB", "CC"]
		arr, err := dtrules.NewArray(nil, true, false)
		if err != nil {
			// Fallback to JSON string
			jsonBytes, _ := json.Marshal(v)
			return dtrules.GetRString(string(jsonBytes))
		}
		for _, item := range v {
			dtItem := goValueToDTRules(item)
			if dtItem != nil {
				arr.Add(dtItem)
			}
		}
		return arr
	case map[string]interface{}:
		// Nested objects that aren't entities - convert to JSON string
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return dtrules.GetRString(fmt.Sprintf("%v", v))
		}
		return dtrules.GetRString(string(jsonBytes))
	default:
		// Try to convert to string
		return dtrules.GetRString(fmt.Sprintf("%v", v))
	}
}

// convertToGoValue converts a DTRules object to a Go value for JSON serialization
func convertToGoValue(obj dtrules.Object) interface{} {
	if obj == nil {
		return nil
	}

	switch obj.Type() {
	case dtrules.TypeInteger:
		if v, err := obj.IntValue(); err == nil {
			return v
		}
		return obj.StringValue()
	case dtrules.TypeDouble:
		if v, err := obj.DoubleValue(); err == nil {
			return v
		}
		return obj.StringValue()
	case dtrules.TypeBoolean:
		if v, err := obj.BooleanValue(); err == nil {
			return v
		}
		return obj.StringValue()
	case dtrules.TypeString, dtrules.TypeName:
		return obj.StringValue()
	case dtrules.TypeDate:
		return obj.StringValue() // Return date as string
	case dtrules.TypeNull:
		return nil
	case dtrules.TypeArray:
		// Handle arrays
		if arr, err := obj.ArrayValue(); err == nil {
			result := make([]interface{}, len(arr))
			for i, item := range arr {
				result[i] = convertToGoValue(item)
			}
			return result
		}
		return obj.StringValue()
	default:
		// For other types, return string representation
		return obj.StringValue()
	}
}

// XML file loading/saving

// EDD XML structures
type EDDXML struct {
	XMLName  xml.Name     `xml:"entity_data_dictionary"`
	Entities []EntityXML  `xml:"entity"`
}

type EntityXML struct {
	Name    string     `xml:"name,attr"`
	Number  string     `xml:"number,attr"`
	Access  string     `xml:"access,attr"`
	Comment string     `xml:"comment,attr"`
	Fields  []FieldXML `xml:"field"`
}

type FieldXML struct {
	Name         string `xml:"name,attr"`
	Type         string `xml:"type,attr"`
	Subtype      string `xml:"subtype,attr"`
	Access       string `xml:"access,attr"`
	Input        string `xml:"input,attr"`
	DefaultValue string `xml:"default_value,attr"`
	Comment      string `xml:"comment,attr"`
}

func (s *Server) loadEDDFile(path, relPath string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	var edd EDDXML
	if err := xml.Unmarshal(data, &edd); err != nil {
		// Try alternative format
		return s.loadEDDAlternative(data, relPath)
	}

	for _, e := range edd.Entities {
		entity := &EntityData{
			Name:    e.Name,
			Number:  e.Number,
			Access:  e.Access,
			Comment: e.Comment,
			Fields:  make([]FieldData, len(e.Fields)),
		}
		for i, f := range e.Fields {
			entity.Fields[i] = FieldData{
				Name:         f.Name,
				Type:         f.Type,
				Subtype:      f.Subtype,
				Access:       f.Access,
				Input:        f.Input,
				DefaultValue: f.DefaultValue,
				Comment:      f.Comment,
			}
		}
		s.upsertEntity(entity, relPath)
	}

	return nil
}

func (s *Server) loadEDDAlternative(data []byte, relPath string) error {
	// Try parsing as different EDD format
	type AltEDD struct {
		XMLName  xml.Name    `xml:"edd"`
		Entities []EntityXML `xml:"entity"`
	}

	var edd AltEDD
	if err := xml.Unmarshal(data, &edd); err != nil {
		return err
	}

	for _, e := range edd.Entities {
		entity := &EntityData{
			Name:    e.Name,
			Number:  e.Number,
			Access:  e.Access,
			Comment: e.Comment,
			Fields:  make([]FieldData, len(e.Fields)),
		}
		for i, f := range e.Fields {
			entity.Fields[i] = FieldData{
				Name:         f.Name,
				Type:         f.Type,
				Subtype:      f.Subtype,
				Access:       f.Access,
				Input:        f.Input,
				DefaultValue: f.DefaultValue,
				Comment:      f.Comment,
			}
		}
		s.upsertEntity(entity, relPath)
	}

	return nil
}

// saveEDDFile persists the entities belonging to relPath by merging the
// API's editable fields into the canonical on-disk model and emitting through
// the excel package's WriteXML — which backfills missing entity numbers.
// Elements the API model doesn't carry (sources, collect/question metadata)
// survive the round-trip.
func (s *Server) saveEDDFile(path, relPath string) error {
	doc := &excel.EDDXML{}
	if data, err := os.ReadFile(path); err == nil {
		if perr := xml.Unmarshal(data, doc); perr != nil {
			return fmt.Errorf("parse %s: %w", path, perr)
		}
	}

	// Entities that should be in this file now, by name.
	keep := make(map[string]*EntityData)
	for _, e := range s.entities {
		if e.Source == relPath {
			keep[e.Name] = e
		}
	}

	// Drop entities deleted through the API, keep everything else.
	filtered := doc.Entities[:0]
	for _, x := range doc.Entities {
		if keep[x.Name] != nil {
			filtered = append(filtered, x)
		}
	}
	doc.Entities = filtered

	// Append entities created through the API, in document order.
	inFile := make(map[string]bool, len(doc.Entities))
	for _, x := range doc.Entities {
		inFile[x.Name] = true
	}
	for _, e := range s.entities {
		if e.Source == relPath && !inFile[e.Name] {
			doc.Entities = append(doc.Entities, &excel.EDDXMLEntity{Name: e.Name})
		}
	}

	// Apply the API-editable fields onto the canonical model.
	for _, x := range doc.Entities {
		applyEntityEdits(x, keep[x.Name])
	}

	importer := excel.NewEDDImporter()
	return importer.WriteXML(doc, path)
}

// applyEntityEdits overwrites the fields the API can edit — number, access,
// comment, and the field rows — while preserving per-field metadata the API
// model doesn't carry (collect flags, question definitions) by index.
func applyEntityEdits(x *excel.EDDXMLEntity, e *EntityData) {
	if e == nil {
		return
	}
	x.Number = e.Number
	x.Access = e.Access
	x.Comment = e.Comment

	fields := make([]*excel.EDDXMLField, len(e.Fields))
	for i, f := range e.Fields {
		var xf excel.EDDXMLField
		if i < len(x.Fields) && x.Fields[i] != nil {
			xf = *x.Fields[i] // preserve collect/question metadata
		}
		xf.Name = f.Name
		xf.Type = f.Type
		xf.SubType = f.Subtype
		xf.Access = f.Access
		xf.Input = f.Input
		xf.DefaultValue = f.DefaultValue
		xf.Comment = f.Comment
		fields[i] = &xf
	}
	x.Fields = fields
}

// DT XML structures - matches the actual DTRules XML format
type DTXML struct {
	XMLName xml.Name       `xml:"decision_tables"`
	Tables  []DTTableXML   `xml:"decision_table"`
}

type DTTableXML struct {
	// NameAttr captures EL format: <decision_table name="...">
	NameAttr         string               `xml:"name,attr"`
	// NumberAttr captures EL format: <decision_table number="...">
	NumberAttr       string               `xml:"number,attr"`
	// Name captures traditional format: <table_name>...</table_name>
	Name             string               `xml:"table_name"`
	XlsFile          string               `xml:"xls_file"`
	AttributeFields  AttributeFieldsXML   `xml:"attribute_fields"`
	Contexts         ContextsXML          `xml:"contexts"`
	InitialActions   InitialActionsXML    `xml:"initial_actions"`
	Conditions       ConditionsXML        `xml:"conditions"`
	Actions          ActionsXML           `xml:"actions"`
	PolicyStatements []PolicyStatementXML `xml:"policy_statements>policy_statement"`
	// LegacyPolicyStatements captures the old flat form: <policy_statement>
	// as a direct child of <decision_table>.
	LegacyPolicyStatements []PolicyStatementXML `xml:"policy_statement"`
}

// AllPolicyStatements returns whichever form the document used.
func (t *DTTableXML) AllPolicyStatements() []PolicyStatementXML {
	if len(t.PolicyStatements) > 0 {
		return t.PolicyStatements
	}
	return t.LegacyPolicyStatements
}

// InitialActionsXML supports both formats: legacy raw text content, and the
// structured <initial_action> entries carrying DSL + postfix.
type InitialActionsXML struct {
	Raw     string                    `xml:",chardata"`
	Details []InitialActionDetailXML  `xml:"initial_action"`
}

type InitialActionDetailXML struct {
	Number  int    `xml:"initial_action_number"`
	Comment string `xml:"initial_action_comment"`
	DSL     string `xml:"initial_action_dsl"`
	Postfix string `xml:"initial_action_postfix"`
}

// Text returns the display form of the initial actions: structured DSL lines
// when present, otherwise the legacy raw text.
func (ia *InitialActionsXML) Text() string {
	if len(ia.Details) == 0 {
		return strings.TrimSpace(ia.Raw)
	}
	lines := make([]string, 0, len(ia.Details))
	for _, d := range ia.Details {
		text := strings.TrimSpace(d.DSL)
		if text == "" {
			text = strings.TrimSpace(d.Postfix)
		}
		if text != "" {
			lines = append(lines, text)
		}
	}
	return strings.Join(lines, "\n")
}

// GetTableName returns the table name from either attribute or element form.
func (t *DTTableXML) GetTableName() string {
	if t.NameAttr != "" {
		return t.NameAttr
	}
	return t.Name
}

// GetTableNumber returns the table number from either attribute or element form.
func (t *DTTableXML) GetTableNumber() string {
	if t.NumberAttr != "" {
		return t.NumberAttr
	}
	return t.AttributeFields.TableNumber
}

type AttributeFieldsXML struct {
	Type        string `xml:"Type"`
	Comments    string `xml:"COMMENTS"`
	FileName    string `xml:"File_Name"`
	TableNumber string `xml:"TABLE_NUMBER"`
}

type ContextsXML struct {
	Contexts []ContextXML `xml:"context_details"`
}

type ContextXML struct {
	Number      int    `xml:"context_number"`
	Comment     string `xml:"context_comment"`
	Description string `xml:"context_description"`
	DSL         string `xml:"context_dsl"`
	Postfix     string `xml:"context_postfix"`
}

type ConditionsXML struct {
	Conditions []ConditionXML `xml:"condition_details"`
}

type ConditionXML struct {
	Number      int                `xml:"condition_number"`
	Comment     string             `xml:"condition_comment"`
	Requirement string             `xml:"condition_requirement"`
	Description string             `xml:"condition_description"`
	DSL         string             `xml:"condition_dsl"`
	Postfix     string             `xml:"condition_postfix"`
	Columns     []ConditionColXML  `xml:"condition_column"`
}

type ConditionColXML struct {
	Number string `xml:"column_number,attr"`
	Value  string `xml:"column_value,attr"`
}

type ActionsXML struct {
	Actions []ActionXML `xml:"action_details"`
}

type ActionXML struct {
	Number      int             `xml:"action_number"`
	Comment     string          `xml:"action_comment"`
	Requirement string          `xml:"action_requirement"`
	Description string          `xml:"action_description"`
	DSL         string          `xml:"action_dsl"`
	Postfix     string          `xml:"action_postfix"`
	Columns     []ActionColXML  `xml:"action_column"`
}

type ActionColXML struct {
	Number string `xml:"column_number,attr"`
	Value  string `xml:"column_value,attr"`
}

// PolicyStatementXML matches the canonical element names used by the engine
// loader (pkg/dtrules/loader): policy_description / policy_statement_postfix.
type PolicyStatementXML struct {
	Column      string `xml:"column,attr"`
	Description string `xml:"policy_description"`
	Postfix     string `xml:"policy_statement_postfix"`
}

// dslOrDescription returns the authored EL DSL when present, falling back to
// the legacy description element.
func dslOrDescription(dsl, description string) string {
	if s := strings.TrimSpace(dsl); s != "" {
		return s
	}
	return strings.TrimSpace(description)
}

func (s *Server) loadDTFile(path, relPath string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	var dt DTXML
	if err := xml.Unmarshal(data, &dt); err != nil {
		log.Printf("Failed to parse DT file %s: %v", path, err)
		return err
	}

	for _, t := range dt.Tables {
		// Skip tables with no name - check both attribute and element forms
		tableName := t.GetTableName()
		if tableName == "" {
			continue
		}

		table := &DecisionTableData{
			TableName:      tableName,
			XlsFile:        t.XlsFile,
			Type:           t.AttributeFields.Type,
			Comments:       t.AttributeFields.Comments,
			TableNumber:    t.GetTableNumber(),
			InitialActions: t.InitialActions.Text(),
		}

		// Load contexts
		for _, ctx := range t.Contexts.Contexts {
			table.Contexts = append(table.Contexts, ContextData{
				Number:      ctx.Number,
				Comment:     ctx.Comment,
				Description: dslOrDescription(ctx.DSL, ctx.Description),
				Postfix:     strings.TrimSpace(ctx.Postfix),
			})
		}

		// Load conditions and determine column count
		maxCol := 0
		for _, cond := range t.Conditions.Conditions {
			condData := ConditionData{
				Number:      cond.Number,
				Comment:     cond.Comment,
				Requirement: cond.Requirement,
				Description: dslOrDescription(cond.DSL, cond.Description),
				Postfix:     strings.TrimSpace(cond.Postfix),
				Columns:     make(map[string]string),
			}
			for _, col := range cond.Columns {
				condData.Columns[col.Number] = strings.TrimSpace(col.Value)
				// Track max column
				var colNum int
				fmt.Sscanf(col.Number, "%d", &colNum)
				if colNum > maxCol {
					maxCol = colNum
				}
			}
			table.Conditions = append(table.Conditions, condData)
		}

		// Load actions
		for _, action := range t.Actions.Actions {
			actionData := ActionData{
				Number:      action.Number,
				Comment:     action.Comment,
				Requirement: action.Requirement,
				Description: dslOrDescription(action.DSL, action.Description),
				Postfix:     strings.TrimSpace(action.Postfix),
				Columns:     make(map[string]string),
			}
			for _, col := range action.Columns {
				actionData.Columns[col.Number] = strings.TrimSpace(col.Value)
				// Track max column
				var colNum int
				fmt.Sscanf(col.Number, "%d", &colNum)
				if colNum > maxCol {
					maxCol = colNum
				}
			}
			table.Actions = append(table.Actions, actionData)
		}

		// Load policy statements
		for _, ps := range t.AllPolicyStatements() {
			table.PolicyStatements = append(table.PolicyStatements, PolicyStatementData{
				Column:      ps.Column,
				Description: ps.Description,
				Postfix:     ps.Postfix,
			})
		}

		table.ColumnCount = maxCol
		if table.TableName != "" {
			s.upsertTable(table, relPath)
		}
	}

	return nil
}

// saveDTFile persists the tables belonging to relPath. It merges the API's
// editable fields into the canonical on-disk model and emits through the
// excel package's WriteXML — the single DT-XML emission funnel — so workbook
// provenance (<xls_file>/<source>) is normalized and every element the API
// model doesn't carry (DSL/postfix pairs it didn't touch, structured initial
// actions, legacy tag forms, table descriptions) survives the round-trip.
func (s *Server) saveDTFile(path, relPath string) error {
	doc := &excel.DecisionTablesXML{}
	if data, err := os.ReadFile(path); err == nil {
		parsed, perr := excel.UnmarshalDecisionTablesXML(data)
		if perr != nil {
			return fmt.Errorf("parse %s: %w", path, perr)
		}
		doc = parsed
	}

	// Tables that should be in this file now, by name.
	keep := make(map[string]*DecisionTableData)
	for _, t := range s.tables {
		if t.Source == relPath {
			keep[t.TableName] = t
		}
	}

	// Drop tables deleted through the API, keep everything else.
	filtered := doc.Tables[:0]
	for _, x := range doc.Tables {
		if _, ok := keep[x.TableName]; ok {
			filtered = append(filtered, x)
		}
	}
	doc.Tables = filtered

	// Append tables created through the API that aren't in the file yet.
	inFile := make(map[string]bool, len(doc.Tables))
	for i := range doc.Tables {
		inFile[doc.Tables[i].TableName] = true
	}
	for _, t := range keep {
		if !inFile[t.TableName] {
			doc.Tables = append(doc.Tables, excel.DecisionTableXML{TableName: t.TableName})
		}
	}

	// Apply the API-editable fields onto the canonical model.
	for i := range doc.Tables {
		applyTableEdits(&doc.Tables[i], keep[doc.Tables[i].TableName])
	}

	importer := excel.NewDTImporter()
	return importer.WriteXML(doc, path)
}

// applyTableEdits overwrites the fields the API can edit — type, comments,
// table number, and the condition/action rows (comment, DSL, rule cells) —
// while preserving everything else the file carries. Contexts, initial
// actions, and policy statements are read-only in the API and left untouched
// when present.
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

// columnValues converts the API's cell map to the XML column list, sorted by
// column number so output is deterministic.
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
