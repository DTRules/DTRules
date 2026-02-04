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

// Package main implements the DTRules REST API server.
// This server provides HTTP endpoints for the DTRules UI to:
// - Open and manage projects
// - Edit entities (EDD)
// - Edit decision tables (DT)
// - Compile and validate expressions
// - Execute rules with optional tracing
package main

import (
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/compiler"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/entity"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/interpreter"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/session"
)

var (
	port    = flag.Int("port", 8080, "Port to listen on")
	verbose = flag.Bool("v", false, "Verbose logging")
)

// Server holds the API server state
type Server struct {
	mu            sync.RWMutex
	projectPath   string
	ruleSet       *session.RuleSet
	eddFiles      []string
	dtFiles       []string
	mapFiles      []string
	entities      map[string]*EntityData
	tables        map[string]*DecisionTableData
	modified      map[string]bool
	entityFactory *entity.Factory
}

// EntityData represents an entity for the API
type EntityData struct {
	Name    string      `json:"name"`
	Access  string      `json:"access"`
	Comment string      `json:"comment"`
	Fields  []FieldData `json:"fields"`
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

func main() {
	flag.Parse()

	server := &Server{
		entities: make(map[string]*EntityData),
		tables:   make(map[string]*DecisionTableData),
		modified: make(map[string]bool),
	}

	// Set up routes
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/api/health", server.handleHealth)

	// Sample projects discovery
	mux.HandleFunc("/api/samples", server.handleSamples)

	// Project endpoints
	mux.HandleFunc("/api/project/open", server.handleProjectOpen)
	mux.HandleFunc("/api/project/save", server.handleProjectSave)
	mux.HandleFunc("/api/project/files", server.handleProjectFiles)

	// EDD endpoints
	mux.HandleFunc("/api/edd", server.handleEDD)
	mux.HandleFunc("/api/edd/entity/", server.handleEntity)

	// DT endpoints
	mux.HandleFunc("/api/dt", server.handleDTList)
	mux.HandleFunc("/api/dt/", server.handleDT)

	// Compile endpoints
	mux.HandleFunc("/api/compile/expression", server.handleCompileExpression)
	mux.HandleFunc("/api/compile/operators", server.handleGetOperators)
	mux.HandleFunc("/api/compile/fields", server.handleGetFields)

	// Execute endpoints
	mux.HandleFunc("/api/execute", server.handleExecute)
	mux.HandleFunc("/api/execute/validate", server.handleValidateExecution)

	// Wrap with CORS middleware
	handler := corsMiddleware(mux)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("DTRules API server starting on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}

// CORS middleware for development
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
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
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// Health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]string{"status": "ok"})
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Scan directory for XML files
	s.projectPath = req.Path
	s.eddFiles = []string{}
	s.dtFiles = []string{}
	s.mapFiles = []string{}
	s.entities = make(map[string]*EntityData)
	s.tables = make(map[string]*DecisionTableData)
	s.modified = make(map[string]bool)

	err := filepath.Walk(req.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if !strings.HasSuffix(strings.ToLower(path), ".xml") {
			return nil
		}

		relPath, _ := filepath.Rel(req.Path, path)
		name := filepath.Base(path)

		// Categorize by name pattern
		nameLower := strings.ToLower(name)
		if strings.Contains(nameLower, "edd") {
			s.eddFiles = append(s.eddFiles, relPath)
			s.loadEDDFile(path)
		} else if strings.Contains(nameLower, "_dt") || strings.Contains(nameLower, "decisiontable") {
			// Skip "Uncompiled" files - they have no postfix expressions and would overwrite valid tables
			if strings.Contains(nameLower, "uncompiled") {
				log.Printf("Skipping uncompiled DT file: %s", relPath)
			} else {
				s.dtFiles = append(s.dtFiles, relPath)
				s.loadDTFile(path)
			}
		} else if strings.Contains(nameLower, "map") {
			s.mapFiles = append(s.mapFiles, relPath)
		}

		return nil
	})

	if err != nil {
		jsonError(w, fmt.Sprintf("Failed to scan directory: %v", err), http.StatusInternalServerError)
		return
	}

	// Create rule set
	s.ruleSet = session.NewRuleSet("ui-project")

	// Load EDD files into the rule set for execution
	for _, eddFile := range s.eddFiles {
		path := filepath.Join(s.projectPath, eddFile)
		f, err := os.Open(path)
		if err != nil {
			log.Printf("Warning: Failed to open EDD file %s for rule set: %v", eddFile, err)
			continue
		}
		if err := s.ruleSet.LoadEDD(f); err != nil {
			log.Printf("Warning: Failed to load EDD file %s into rule set: %v", eddFile, err)
		}
		f.Close()
	}

	// Load decision table files into the rule set for execution
	for _, dtFile := range s.dtFiles {
		path := filepath.Join(s.projectPath, dtFile)
		f, err := os.Open(path)
		if err != nil {
			log.Printf("Warning: Failed to open DT file %s for rule set: %v", dtFile, err)
			continue
		}
		if err := s.ruleSet.LoadDecisionTables(f); err != nil {
			log.Printf("Warning: Failed to load DT file %s into rule set: %v", dtFile, err)
		}
		f.Close()
	}

	jsonResponse(w, map[string]interface{}{
		"success":  true,
		"eddFiles": s.eddFiles,
		"dtFiles":  s.dtFiles,
		"mapFiles": s.mapFiles,
	})
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
			if err := s.saveEDDFile(path); err != nil {
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
			if err := s.saveDTFile(path); err != nil {
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

	entities := make([]*EntityData, 0, len(s.entities))
	for _, e := range s.entities {
		entities = append(entities, e)
	}

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
		entity, ok := s.entities[entityName]
		s.mu.RUnlock()

		if !ok {
			jsonError(w, "Entity not found", http.StatusNotFound)
			return
		}
		jsonResponse(w, map[string]interface{}{
			"success": true,
			"entity":  entity,
		})

	case "POST":
		var entity EntityData
		if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
			jsonError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		s.mu.Lock()
		s.entities[entity.Name] = &entity
		if len(s.eddFiles) > 0 {
			s.modified[s.eddFiles[0]] = true
		}
		s.mu.Unlock()

		jsonResponse(w, map[string]interface{}{
			"success": true,
			"message": "Entity created",
		})

	case "PUT":
		var entity EntityData
		if err := json.NewDecoder(r.Body).Decode(&entity); err != nil {
			jsonError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		s.mu.Lock()
		s.entities[entityName] = &entity
		if len(s.eddFiles) > 0 {
			s.modified[s.eddFiles[0]] = true
		}
		s.mu.Unlock()

		jsonResponse(w, map[string]interface{}{
			"success": true,
			"message": "Entity updated",
		})

	case "DELETE":
		s.mu.Lock()
		delete(s.entities, entityName)
		if len(s.eddFiles) > 0 {
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
		if err := json.NewDecoder(r.Body).Decode(&table); err != nil {
			jsonError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		s.mu.Lock()
		s.tables[table.TableName] = &table
		if len(s.dtFiles) > 0 {
			s.modified[s.dtFiles[0]] = true
		}
		s.mu.Unlock()

		jsonResponse(w, map[string]interface{}{
			"success": true,
			"message": "Table created",
		})
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	tables := make([]map[string]interface{}, 0, len(s.tables))
	for _, t := range s.tables {
		tables = append(tables, map[string]interface{}{
			"name":           t.TableName,
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
		table, ok := s.tables[tableName]
		s.mu.RUnlock()

		if !ok {
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
		if err := json.NewDecoder(r.Body).Decode(&table); err != nil {
			jsonError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		s.mu.Lock()
		s.tables[tableName] = &table
		if len(s.dtFiles) > 0 {
			s.modified[s.dtFiles[0]] = true
		}
		s.mu.Unlock()

		jsonResponse(w, map[string]interface{}{
			"success": true,
			"message": "Table updated",
		})

	case "DELETE":
		s.mu.Lock()
		delete(s.tables, tableName)
		if len(s.dtFiles) > 0 {
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
	table, ok := s.tables[tableName]
	s.mu.RUnlock()

	if !ok {
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	_, tableExists := s.tables[req.TableName]
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

func (s *Server) loadEDDFile(path string) error {
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
		return s.loadEDDAlternative(data)
	}

	for _, e := range edd.Entities {
		entity := &EntityData{
			Name:    e.Name,
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
		s.entities[entity.Name] = entity
	}

	return nil
}

func (s *Server) loadEDDAlternative(data []byte) error {
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
		s.entities[entity.Name] = entity
	}

	return nil
}

func (s *Server) saveEDDFile(path string) error {
	// Build EDD XML
	edd := EDDXML{}
	for _, entity := range s.entities {
		e := EntityXML{
			Name:    entity.Name,
			Access:  entity.Access,
			Comment: entity.Comment,
		}
		for _, field := range entity.Fields {
			e.Fields = append(e.Fields, FieldXML{
				Name:         field.Name,
				Type:         field.Type,
				Subtype:      field.Subtype,
				Access:       field.Access,
				Input:        field.Input,
				DefaultValue: field.DefaultValue,
				Comment:      field.Comment,
			})
		}
		edd.Entities = append(edd.Entities, e)
	}

	data, err := xml.MarshalIndent(edd, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, append([]byte(xml.Header), data...), 0644)
}

// DT XML structures - matches the actual DTRules XML format
type DTXML struct {
	XMLName xml.Name       `xml:"decision_tables"`
	Tables  []DTTableXML   `xml:"decision_table"`
}

type DTTableXML struct {
	Name             string               `xml:"table_name"`
	XlsFile          string               `xml:"xls_file"`
	AttributeFields  AttributeFieldsXML   `xml:"attribute_fields"`
	Contexts         ContextsXML          `xml:"contexts"`
	InitialActions   string               `xml:"initial_actions"`
	Conditions       ConditionsXML        `xml:"conditions"`
	Actions          ActionsXML           `xml:"actions"`
	PolicyStatements []PolicyStatementXML `xml:"policy_statement"`
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
	Postfix     string          `xml:"action_postfix"`
	Columns     []ActionColXML  `xml:"action_column"`
}

type ActionColXML struct {
	Number string `xml:"column_number,attr"`
	Value  string `xml:"column_value,attr"`
}

type PolicyStatementXML struct {
	Column      string `xml:"column,attr"`
	Description string `xml:"description"`
	Postfix     string `xml:"postfix"`
}

func (s *Server) loadDTFile(path string) error {
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
		// Skip tables with no name
		if t.Name == "" {
			continue
		}

		table := &DecisionTableData{
			TableName:      t.Name,
			XlsFile:        t.XlsFile,
			Type:           t.AttributeFields.Type,
			Comments:       t.AttributeFields.Comments,
			TableNumber:    t.AttributeFields.TableNumber,
			InitialActions: t.InitialActions,
		}

		// Load contexts
		for _, ctx := range t.Contexts.Contexts {
			table.Contexts = append(table.Contexts, ContextData{
				Number:      ctx.Number,
				Comment:     ctx.Comment,
				Description: ctx.Description,
				Postfix:     ctx.Postfix,
			})
		}

		// Load conditions and determine column count
		maxCol := 0
		for _, cond := range t.Conditions.Conditions {
			condData := ConditionData{
				Number:      cond.Number,
				Comment:     cond.Comment,
				Requirement: cond.Requirement,
				Description: cond.Description,
				Postfix:     cond.Postfix,
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
				Description: action.Description,
				Postfix:     action.Postfix,
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
		for _, ps := range t.PolicyStatements {
			table.PolicyStatements = append(table.PolicyStatements, PolicyStatementData{
				Column:      ps.Column,
				Description: ps.Description,
				Postfix:     ps.Postfix,
			})
		}

		table.ColumnCount = maxCol
		if table.TableName != "" {
			s.tables[table.TableName] = table
		}
	}

	return nil
}

func (s *Server) saveDTFile(path string) error {
	dt := DTXML{}

	for _, table := range s.tables {
		t := DTTableXML{
			Name:           table.TableName,
			XlsFile:        table.XlsFile,
			InitialActions: table.InitialActions,
			AttributeFields: AttributeFieldsXML{
				Type:        table.Type,
				Comments:    table.Comments,
				TableNumber: table.TableNumber,
			},
		}

		// Save contexts
		for _, ctx := range table.Contexts {
			t.Contexts.Contexts = append(t.Contexts.Contexts, ContextXML{
				Number:      ctx.Number,
				Comment:     ctx.Comment,
				Description: ctx.Description,
				Postfix:     ctx.Postfix,
			})
		}

		// Save conditions
		for _, cond := range table.Conditions {
			c := ConditionXML{
				Number:      cond.Number,
				Comment:     cond.Comment,
				Requirement: cond.Requirement,
				Description: cond.Description,
				Postfix:     cond.Postfix,
			}
			for colNum, val := range cond.Columns {
				c.Columns = append(c.Columns, ConditionColXML{
					Number: colNum,
					Value:  val,
				})
			}
			t.Conditions.Conditions = append(t.Conditions.Conditions, c)
		}

		// Save actions
		for _, action := range table.Actions {
			a := ActionXML{
				Number:      action.Number,
				Comment:     action.Comment,
				Requirement: action.Requirement,
				Description: action.Description,
				Postfix:     action.Postfix,
			}
			for colNum, val := range action.Columns {
				a.Columns = append(a.Columns, ActionColXML{
					Number: colNum,
					Value:  val,
				})
			}
			t.Actions.Actions = append(t.Actions.Actions, a)
		}

		// Save policy statements
		for _, ps := range table.PolicyStatements {
			t.PolicyStatements = append(t.PolicyStatements, PolicyStatementXML{
				Column:      ps.Column,
				Description: ps.Description,
				Postfix:     ps.Postfix,
			})
		}

		dt.Tables = append(dt.Tables, t)
	}

	data, err := xml.MarshalIndent(dt, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, append([]byte(xml.Header), data...), 0644)
}
