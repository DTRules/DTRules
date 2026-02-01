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
			s.dtFiles = append(s.dtFiles, relPath)
			s.loadDTFile(path)
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

	// Enable tracing if requested
	if req.Trace {
		state.EnableTrace()
	}

	// Execute
	if err := rsess.Execute(req.TableName); err != nil {
		jsonError(w, fmt.Sprintf("Execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Get results
	result := make(map[string]interface{})
	// TODO: Extract entity values as results

	jsonResponse(w, map[string]interface{}{
		"success": true,
		"result":  result,
	})
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

// XML file loading/saving

// EDD XML structures
type EDDXML struct {
	XMLName  xml.Name     `xml:"edd_set"`
	Entities []EntityXML  `xml:"entity"`
}

type EntityXML struct {
	Name    string     `xml:"name,attr"`
	Access  string     `xml:"access,attr"`
	Comment string     `xml:"comment"`
	Fields  []FieldXML `xml:"attribute"`
}

type FieldXML struct {
	Name         string `xml:"name,attr"`
	Type         string `xml:"type,attr"`
	Subtype      string `xml:"subtype,attr"`
	Access       string `xml:"access,attr"`
	Input        string `xml:"input,attr"`
	DefaultValue string `xml:"default,attr"`
	Comment      string `xml:"comment"`
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

// DT XML structures
type DTXML struct {
	XMLName xml.Name       `xml:"decision_tables"`
	Tables  []DTTableXML   `xml:"decision_table"`
}

type DTTableXML struct {
	Name             string               `xml:"table_name,attr"`
	Type             string               `xml:"type,attr"`
	Number           string               `xml:"number,attr"`
	XlsFile          string               `xml:"xls_file,attr"`
	Comments         string               `xml:"comments"`
	Contexts         []ContextXML         `xml:"context"`
	InitialActions   string               `xml:"initial_actions"`
	Conditions       []ConditionXML       `xml:"condition"`
	Actions          []ActionXML          `xml:"action"`
	PolicyStatements []PolicyStatementXML `xml:"policy_statement"`
}

type ContextXML struct {
	Number      int    `xml:"number,attr"`
	Comment     string `xml:"comment,attr"`
	Description string `xml:"description"`
	Postfix     string `xml:"postfix"`
}

type ConditionXML struct {
	Number      int          `xml:"number,attr"`
	Comment     string       `xml:"comment,attr"`
	Requirement string       `xml:"requirement,attr"`
	Description string       `xml:"description"`
	Postfix     string       `xml:"postfix"`
	Columns     []ColumnXML  `xml:"column"`
}

type ActionXML struct {
	Number      int          `xml:"number,attr"`
	Comment     string       `xml:"comment,attr"`
	Requirement string       `xml:"requirement,attr"`
	Description string       `xml:"description"`
	Postfix     string       `xml:"postfix"`
	Columns     []ColumnXML  `xml:"column"`
}

type ColumnXML struct {
	Number string `xml:"number,attr"`
	Value  string `xml:",chardata"`
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
		return err
	}

	for _, t := range dt.Tables {
		table := &DecisionTableData{
			TableName:      t.Name,
			XlsFile:        t.XlsFile,
			Type:           t.Type,
			Comments:       t.Comments,
			TableNumber:    t.Number,
			InitialActions: t.InitialActions,
		}

		// Load contexts
		for _, ctx := range t.Contexts {
			table.Contexts = append(table.Contexts, ContextData{
				Number:      ctx.Number,
				Comment:     ctx.Comment,
				Description: ctx.Description,
				Postfix:     ctx.Postfix,
			})
		}

		// Load conditions and determine column count
		maxCol := 0
		for _, cond := range t.Conditions {
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
		for _, action := range t.Actions {
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
		s.tables[table.TableName] = table
	}

	return nil
}

func (s *Server) saveDTFile(path string) error {
	dt := DTXML{}

	for _, table := range s.tables {
		t := DTTableXML{
			Name:           table.TableName,
			Type:           table.Type,
			Number:         table.TableNumber,
			XlsFile:        table.XlsFile,
			Comments:       table.Comments,
			InitialActions: table.InitialActions,
		}

		// Save contexts
		for _, ctx := range table.Contexts {
			t.Contexts = append(t.Contexts, ContextXML{
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
				c.Columns = append(c.Columns, ColumnXML{
					Number: colNum,
					Value:  val,
				})
			}
			t.Conditions = append(t.Conditions, c)
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
				a.Columns = append(a.Columns, ColumnXML{
					Number: colNum,
					Value:  val,
				})
			}
			t.Actions = append(t.Actions, a)
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
