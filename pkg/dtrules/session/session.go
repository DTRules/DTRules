// Copyright 2004-2011 DTRules.com, Inc.
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

// Package session implements the DTRules session and rule set management.
package session

import (
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/compiler"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/operators"
)

// RSession implements the Session interface.
// Each session is associated with a particular RuleSet.
type RSession struct {
	ruleSet         *RuleSet
	state           *interpreter.DTState
	entityFactory   *entity.Factory
	uniqueID        int
	entityInstances map[int]dtrules.Entity
	dateParser      dtrules.DateParser
	attributes      map[string]interface{}
}

// NewSession creates a new session for the given rule set.
func NewSession(rs *RuleSet) (*RSession, error) {
	s := &RSession{
		ruleSet:         rs,
		entityInstances: make(map[int]dtrules.Entity),
		dateParser:      NewDateParser(),
		attributes:      make(map[string]interface{}),
	}

	// Create state
	s.state = interpreter.NewDTState(s)

	// Get entity factory
	s.entityFactory = rs.GetEntityFactory()
	s.uniqueID = s.entityFactory.GetUniqueID() + 10000

	// Add reference entities to session
	for _, entityName := range s.entityFactory.GetEntityNames() {
		refEntity := s.entityFactory.FindRefEntity(entityName)
		if refEntity != nil {
			id := refEntity.GetID()
			s.entityInstances[id] = refEntity
		}
	}

	// Push primitives and decision tables entities onto entity stack
	if primitives := operators.GetPrimitives(); primitives != nil {
		s.state.EntityPush(primitives)
	}
	// Push decision tables entity so decision tables can call each other
	if dtEntity := s.entityFactory.GetDecisionTablesEntity(); dtEntity != nil {
		s.state.EntityPush(dtEntity)
	}

	return s, nil
}

// GetState returns the interpreter state.
func (s *RSession) GetState() dtrules.State {
	return s.state
}

// GetEntityFactory returns the entity factory.
func (s *RSession) GetEntityFactory() dtrules.EntityFactory {
	return s.entityFactory
}

// GetUniqueID returns a unique ID for this session.
func (s *RSession) GetUniqueID() int {
	id := s.uniqueID
	s.uniqueID++
	return id
}

// GetDateParser returns the date parser.
func (s *RSession) GetDateParser() dtrules.DateParser {
	return s.dateParser
}

// SetDateParser sets the date parser.
func (s *RSession) SetDateParser(parser dtrules.DateParser) {
	s.dateParser = parser
}

// GetRuleSet returns the rule set.
func (s *RSession) GetRuleSet() dtrules.RuleSet {
	return s.ruleSet
}

// CreateEntity creates a new entity instance.
func (s *RSession) CreateEntity(name *dtrules.RName) (dtrules.Entity, error) {
	entity, err := s.entityFactory.CreateEntity(s, name)
	if err != nil {
		return nil, err
	}
	s.entityInstances[entity.GetID()] = entity
	return entity, nil
}

// GetEntityByID returns an entity by its ID.
func (s *RSession) GetEntityByID(id int) dtrules.Entity {
	return s.entityInstances[id]
}

// GetAttribute returns a session attribute.
func (s *RSession) GetAttribute(key string) interface{} {
	return s.attributes[key]
}

// SetAttribute sets a session attribute.
func (s *RSession) SetAttribute(key string, value interface{}) {
	s.attributes[key] = value
}

// Execute executes a decision table by name.
func (s *RSession) Execute(tableName string) error {
	name := dtrules.GetRName(tableName)
	if name == nil {
		return dtrules.UndefinedError("Execute", "Invalid decision table name syntax: "+tableName)
	}
	dt, err := s.entityFactory.GetDecisionTable(name)
	if err != nil {
		return err
	}
	if dt == nil {
		return dtrules.UndefinedError("Execute", "Decision table not found: "+tableName)
	}
	return dt.Execute(s.state)
}

// ExecuteAt executes a decision table with a given entity context.
func (s *RSession) ExecuteAt(tableName string, entityName string) error {
	// Find the entity
	eName := dtrules.GetRName(entityName)
	if eName == nil {
		return dtrules.UndefinedError("ExecuteAt", "Invalid entity name syntax: "+entityName)
	}
	entity, err := s.state.FindEntity(eName)
	if err != nil || entity == nil {
		return dtrules.UndefinedError("ExecuteAt", "Entity not found: "+entityName)
	}

	// Push entity onto stack
	s.state.EntityPush(entity)

	// Execute
	err = s.Execute(tableName)

	// Pop entity (even on error)
	s.state.EntityPop()

	return err
}

// Compile compiles a postfix expression string into executable code.
func (s *RSession) Compile(expr string) (dtrules.Object, error) {
	c := compiler.NewCompiler(s, s.entityFactory)
	return c.Compile(expr)
}

// CompileExpressionToBytecode compiles a postfix expression to bytecode.
func (s *RSession) CompileExpressionToBytecode(expr string) (*dtrules.BytecodeChunk, error) {
	c := compiler.NewCompiler(s, s.entityFactory)
	return c.CompileToBytecode(expr)
}

// DateParser implements the DateParser interface with common date formats.
type DateParser struct {
	formats []string
}

// NewDateParser creates a new date parser with common formats.
// Most-specific formats are tried first; RFC3339 variants precede date-only formats
// so that timestamps like "2026-04-17T21:05:30Z" are parsed correctly.
func NewDateParser() *DateParser {
	return &DateParser{
		formats: []string{
			time.RFC3339Nano,      // 2006-01-02T15:04:05.999999999Z07:00
			time.RFC3339,          // 2006-01-02T15:04:05Z07:00
			"2006-01-02 15:04:05", // YYYY-MM-DD HH:MM:SS (space-separated)
			"2006-01-02",          // YYYY-MM-DD
			"01/02/2006",          // MM/DD/YYYY
			"1/2/2006",            // M/D/YYYY
			"01/02/2006 15:04:05", // MM/DD/YYYY HH:MM:SS
			"Jan 2, 2006",         // Month D, YYYY
			"January 2, 2006",     // Month D, YYYY
			dtrules.DateFormat,    // Default format
		},
	}
}

// GetDate parses a string into a time.Time.
func (p *DateParser) GetDate(s string) (time.Time, error) {
	for _, format := range p.formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, dtrules.NewRulesError("DateParser", "GetDate",
		"Could not parse date: "+s)
}

// Parse is an alias for GetDate.
func (p *DateParser) Parse(s string) (time.Time, error) {
	return p.GetDate(s)
}

// AddFormat adds a date format to the parser.
func (p *DateParser) AddFormat(format string) {
	p.formats = append([]string{format}, p.formats...)
}
