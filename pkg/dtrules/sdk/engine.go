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

// Package sdk provides an embeddable DTRules engine for applications.
//
// The SDK allows applications to:
//   - Embed XML rules using Go's embed package
//   - Load rules from filesystem or embedded files
//   - Execute decision tables with typed input/output
//   - Run concurrent sessions safely
//
// Basic usage:
//
//	//go:embed rules/*.xml
//	var rulesFS embed.FS
//
//	func main() {
//	    engine, err := sdk.NewEngine("MyRules", sdk.WithFS(rulesFS, "rules"))
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    ctx := engine.NewContext()
//	    ctx.Set("income", 50000)
//	    ctx.Set("age", 35)
//
//	    result, err := engine.Execute("Calculate_Eligibility", ctx)
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    eligible := result.GetBool("eligible")
//	}
package sdk

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// Engine is the main entry point for executing decision tables.
// It holds a compiled RuleSet and can create concurrent execution contexts.
// Engine is safe for concurrent use.
type Engine struct {
	name    string
	ruleSet *session.RuleSet
	mu      sync.RWMutex
}

// EngineOption configures an Engine during creation.
type EngineOption func(*engineConfig) error

type engineConfig struct {
	eddFiles []io.Reader
	dtFiles  []io.Reader
	fsSource fs.FS
	fsPath   string
	dirPath  string
}

// WithEDD adds an EDD (Entity Data Dictionary) source.
func WithEDD(r io.Reader) EngineOption {
	return func(c *engineConfig) error {
		c.eddFiles = append(c.eddFiles, r)
		return nil
	}
}

// WithDT adds a Decision Tables source.
func WithDT(r io.Reader) EngineOption {
	return func(c *engineConfig) error {
		c.dtFiles = append(c.dtFiles, r)
		return nil
	}
}

// WithFS loads rules from an embedded filesystem (embed.FS).
// The path parameter specifies the subdirectory containing rule files.
//
// Example:
//
//	//go:embed rules/*.xml
//	var rulesFS embed.FS
//
//	engine, err := sdk.NewEngine("MyRules", sdk.WithFS(rulesFS, "rules"))
func WithFS(fsys fs.FS, path string) EngineOption {
	return func(c *engineConfig) error {
		c.fsSource = fsys
		c.fsPath = path
		return nil
	}
}

// WithDirectory loads rules from a filesystem directory.
func WithDirectory(path string) EngineOption {
	return func(c *engineConfig) error {
		c.dirPath = path
		return nil
	}
}

// NewEngine creates a new DTRules engine with the given name and options.
func NewEngine(name string, opts ...EngineOption) (*Engine, error) {
	cfg := &engineConfig{}
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("option error: %w", err)
		}
	}

	rs := session.NewRuleSet(name)

	// Load from embedded FS
	if cfg.fsSource != nil {
		if err := loadFromFS(rs, cfg.fsSource, cfg.fsPath); err != nil {
			return nil, fmt.Errorf("failed to load from embedded FS: %w", err)
		}
	}

	// Load from directory
	if cfg.dirPath != "" {
		if err := loadFromDirectory(rs, cfg.dirPath); err != nil {
			return nil, fmt.Errorf("failed to load from directory: %w", err)
		}
	}

	// Load from explicit readers
	for _, r := range cfg.eddFiles {
		if err := rs.LoadEDD(r); err != nil {
			return nil, fmt.Errorf("failed to load EDD: %w", err)
		}
	}
	for _, r := range cfg.dtFiles {
		if err := rs.LoadDecisionTables(r); err != nil {
			return nil, fmt.Errorf("failed to load DT: %w", err)
		}
	}

	return &Engine{
		name:    name,
		ruleSet: rs,
	}, nil
}

// loadFromFS loads rules from an embedded or virtual filesystem.
func loadFromFS(rs *session.RuleSet, fsys fs.FS, basePath string) error {
	var eddFiles, dtFiles []string

	// Walk the filesystem to find rule files
	err := fs.WalkDir(fsys, basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		name := strings.ToLower(filepath.Base(path))
		if strings.HasSuffix(name, ".xml") {
			if strings.Contains(name, "_edd") || strings.Contains(name, "edd_") || name == "edd.xml" {
				eddFiles = append(eddFiles, path)
			} else if strings.Contains(name, "_dt") || strings.Contains(name, "dt_") ||
				strings.Contains(name, "decisiontables") {
				dtFiles = append(dtFiles, path)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Load EDD files first
	for _, path := range eddFiles {
		f, err := fsys.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", path, err)
		}
		if err := rs.LoadEDD(f); err != nil {
			f.Close()
			return fmt.Errorf("failed to load EDD %s: %w", path, err)
		}
		f.Close()
	}

	// Then load DT files
	for _, path := range dtFiles {
		f, err := fsys.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", path, err)
		}
		if err := rs.LoadDecisionTables(f); err != nil {
			f.Close()
			return fmt.Errorf("failed to load DT %s: %w", path, err)
		}
		f.Close()
	}

	return nil
}

// loadFromDirectory loads rules from a filesystem directory.
func loadFromDirectory(rs *session.RuleSet, dir string) error {
	var eddFiles, dtFiles []string

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		name := strings.ToLower(filepath.Base(path))
		if strings.HasSuffix(name, ".xml") {
			if strings.Contains(name, "_edd") || strings.Contains(name, "edd_") || name == "edd.xml" {
				eddFiles = append(eddFiles, path)
			} else if strings.Contains(name, "_dt") || strings.Contains(name, "dt_") ||
				strings.Contains(name, "decisiontables") {
				dtFiles = append(dtFiles, path)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Load EDD files first
	for _, path := range eddFiles {
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", path, err)
		}
		if err := rs.LoadEDD(f); err != nil {
			f.Close()
			return fmt.Errorf("failed to load EDD %s: %w", path, err)
		}
		f.Close()
	}

	// Then load DT files
	for _, path := range dtFiles {
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", path, err)
		}
		if err := rs.LoadDecisionTables(f); err != nil {
			f.Close()
			return fmt.Errorf("failed to load DT %s: %w", path, err)
		}
		f.Close()
	}

	return nil
}

// Name returns the engine name.
func (e *Engine) Name() string {
	return e.name
}

// ListTables returns the names of all available decision tables.
func (e *Engine) ListTables() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	names := e.ruleSet.GetDecisionTableNames()
	result := make([]string, len(names))
	for i, n := range names {
		result[i] = n.StringValue()
	}
	return result
}

// ListEntities returns the names of all defined entities.
func (e *Engine) ListEntities() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	names := e.ruleSet.GetEntityNames()
	result := make([]string, len(names))
	for i, n := range names {
		result[i] = n.StringValue()
	}
	return result
}

// NewContext creates a new execution context.
// Each context is independent and can be used concurrently.
func (e *Engine) NewContext() *Context {
	return &Context{
		engine: e,
		inputs: make(map[string]interface{}),
	}
}

// Execute runs a decision table with the given context.
// This is a convenience method that creates a session, populates inputs,
// executes the table, and extracts results.
func (e *Engine) Execute(tableName string, ctx *Context) (*Result, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Create session
	sess, err := e.ruleSet.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	rsess := sess.(*session.RSession)

	// Populate input entities
	if err := ctx.populateSession(rsess); err != nil {
		return nil, fmt.Errorf("failed to populate inputs: %w", err)
	}

	// Execute decision table
	if err := rsess.Execute(tableName); err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	// Extract results
	return newResult(rsess), nil
}

// Context holds input data for decision table execution.
type Context struct {
	engine       *Engine
	inputs       map[string]interface{}
	entities     map[string]map[string]interface{}
	entityArrays map[string][]map[string]interface{}
}

// Set sets a simple input value.
// For entity-specific values, use SetEntity.
func (c *Context) Set(name string, value interface{}) *Context {
	c.inputs[name] = value
	return c
}

// SetEntity sets a value on a specific entity.
func (c *Context) SetEntity(entityName, fieldName string, value interface{}) *Context {
	if c.entities == nil {
		c.entities = make(map[string]map[string]interface{})
	}
	if c.entities[entityName] == nil {
		c.entities[entityName] = make(map[string]interface{})
	}
	c.entities[entityName][fieldName] = value
	return c
}

// SetEntityArray sets an array of entities that can be iterated with forall.
// arrayName should be the plural entity name (e.g., "accounts" for "account" entities).
// items is a slice of maps where each map represents field name -> value for one entity.
func (c *Context) SetEntityArray(arrayName string, items []map[string]interface{}) *Context {
	if c.entityArrays == nil {
		c.entityArrays = make(map[string][]map[string]interface{})
	}
	c.entityArrays[arrayName] = items
	return c
}

// populateSession populates a session with context data.
func (c *Context) populateSession(sess *session.RSession) error {
	state := sess.GetState()

	// For each entity with data, create an instance and populate it
	for entityName, fields := range c.entities {
		entity, err := sess.CreateEntity(dtrules.GetRName(entityName))
		if err != nil {
			return fmt.Errorf("failed to create entity %s: %w", entityName, err)
		}

		for fieldName, value := range fields {
			obj, err := toObject(value)
			if err != nil {
				return fmt.Errorf("failed to convert %s.%s: %w", entityName, fieldName, err)
			}
			if err := entity.Put(dtrules.GetRName(fieldName), obj); err != nil {
				return fmt.Errorf("failed to set %s.%s: %w", entityName, fieldName, err)
			}
		}

		// Push entity onto stack so decision tables can access it
		state.EntityPush(entity)
	}

	// Process entity arrays
	for arrayName, items := range c.entityArrays {
		if len(items) == 0 {
			continue
		}

		// Derive singular entity name from plural array name
		entityName := singularize(arrayName)

		// Create a DTRules array to hold the entities
		arr, err := dtrules.NewArray(sess, true, false)
		if err != nil {
			return fmt.Errorf("failed to create array for %s: %w", arrayName, err)
		}

		// Create entity instances and add them to the array
		for i, itemFields := range items {
			if itemFields == nil {
				continue
			}

			entity, err := sess.CreateEntity(dtrules.GetRName(entityName))
			if err != nil {
				return fmt.Errorf("failed to create entity %s[%d]: %w", arrayName, i, err)
			}

			for fieldName, value := range itemFields {
				obj, err := toObject(value)
				if err != nil {
					return fmt.Errorf("failed to convert %s[%d].%s: %w", arrayName, i, fieldName, err)
				}
				if err := entity.Put(dtrules.GetRName(fieldName), obj); err != nil {
					return fmt.Errorf("failed to set %s[%d].%s: %w", arrayName, i, fieldName, err)
				}
			}

			// Add entity to the array
			arr.Add(entity)

			// Push entity to entity stack so it's accessible
			state.EntityPush(entity)
		}

		// Store the array as a session attribute so it can be accessed by name
		sess.SetAttribute(arrayName, arr)
	}

	// Simple inputs go to session attributes or a default entity
	// This depends on the specific rule set's design
	for name, value := range c.inputs {
		sess.SetAttribute(name, value)
	}

	return nil
}

// toObject converts a Go value to a DTRules Object.
func toObject(v interface{}) (dtrules.Object, error) {
	switch val := v.(type) {
	case nil:
		return dtrules.GetRNull(), nil
	case bool:
		return dtrules.GetRBoolean(val), nil
	case int:
		return dtrules.GetRIntegerValue(int64(val)), nil
	case int32:
		return dtrules.GetRIntegerValue(int64(val)), nil
	case int64:
		return dtrules.GetRIntegerValue(val), nil
	case float32:
		return dtrules.GetRDoubleValue(float64(val)), nil
	case float64:
		return dtrules.GetRDoubleValue(val), nil
	case string:
		return dtrules.GetRString(val), nil
	case dtrules.Object:
		return val, nil
	default:
		return nil, fmt.Errorf("unsupported type: %T", v)
	}
}

// singularize returns the singular form of a plural noun.
// Handles common cases: "orders" -> "order", "entities" -> "entity".
func singularize(s string) string {
	if len(s) <= 1 {
		return s
	}
	lower := strings.ToLower(s)
	if strings.HasSuffix(lower, "ies") && len(s) > 3 {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(lower, "ses") || strings.HasSuffix(lower, "xes") {
		return s[:len(s)-2]
	}
	if strings.HasSuffix(lower, "s") && !strings.HasSuffix(lower, "ss") {
		return s[:len(s)-1]
	}
	return s
}

// Result holds the output from a decision table execution.
type Result struct {
	session  *session.RSession
	entities map[string]dtrules.Entity
}

// newResult creates a Result from a session.
func newResult(sess *session.RSession) *Result {
	return &Result{
		session:  sess,
		entities: make(map[string]dtrules.Entity),
	}
}

// GetEntity returns an entity by name from the result.
func (r *Result) GetEntity(name string) (dtrules.Entity, bool) {
	// Try to find entity by name in the session
	// This requires iterating through created entities
	// For now, return from cache if available
	e, ok := r.entities[name]
	return e, ok
}

// Get retrieves a value from the result.
// It searches session attributes first, then entities.
func (r *Result) Get(name string) (interface{}, bool) {
	// Check session attributes
	if v := r.session.GetAttribute(name); v != nil {
		return v, true
	}
	return nil, false
}

// GetString retrieves a string value.
func (r *Result) GetString(name string) string {
	if v, ok := r.Get(name); ok {
		if s, ok := v.(string); ok {
			return s
		}
		if obj, ok := v.(dtrules.Object); ok {
			return obj.StringValue()
		}
	}
	return ""
}

// GetInt retrieves an integer value.
func (r *Result) GetInt(name string) int64 {
	if v, ok := r.Get(name); ok {
		switch val := v.(type) {
		case int:
			return int64(val)
		case int64:
			return val
		case dtrules.Object:
			if i, err := val.IntValue(); err == nil {
				return int64(i)
			}
		}
	}
	return 0
}

// GetFloat retrieves a float value.
func (r *Result) GetFloat(name string) float64 {
	if v, ok := r.Get(name); ok {
		switch val := v.(type) {
		case float64:
			return val
		case float32:
			return float64(val)
		case dtrules.Object:
			if f, err := val.DoubleValue(); err == nil {
				return f
			}
		}
	}
	return 0
}

// GetBool retrieves a boolean value.
func (r *Result) GetBool(name string) bool {
	if v, ok := r.Get(name); ok {
		switch val := v.(type) {
		case bool:
			return val
		case dtrules.Object:
			if b, err := val.BooleanValue(); err == nil {
				return b
			}
		}
	}
	return false
}
