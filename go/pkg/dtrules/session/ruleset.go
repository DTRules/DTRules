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

package session

import (
	"io"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/compiler"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/entity"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/loader"
)

// RuleSet defines a set of artifacts that make up a logical set of rules.
// This includes the EDD (Entity Data Dictionary), decision tables,
// and mappings.
type RuleSet struct {
	name          *dtrules.RName
	entityFactory *entity.Factory
	eddNames      []string
	dtNames       []string
	mapPaths      []string
	entryPoints   map[string]string
	attributes    map[string]interface{}
}

// NewRuleSet creates a new rule set with the given name.
// Returns nil if name has invalid syntax.
func NewRuleSet(name string) *RuleSet {
	rname := dtrules.GetRName(name)
	if rname == nil {
		return nil
	}
	return &RuleSet{
		name:          rname,
		entityFactory: entity.NewFactory(nil),
		eddNames:      make([]string, 0),
		dtNames:       make([]string, 0),
		mapPaths:      make([]string, 0),
		entryPoints:   make(map[string]string),
		attributes:    make(map[string]interface{}),
	}
}

// GetName returns the rule set name.
func (rs *RuleSet) GetName() *dtrules.RName {
	return rs.name
}

// GetEntityFactory returns the entity factory.
func (rs *RuleSet) GetEntityFactory() *entity.Factory {
	return rs.entityFactory
}

// LoadEDD loads an EDD from a reader.
func (rs *RuleSet) LoadEDD(r io.Reader) error {
	// Create a temporary session for loading
	tempSession := &loadSession{
		factory:    rs.entityFactory,
		dateParser: NewDateParser(),
	}

	eddLoader := loader.NewEDDLoader(tempSession, rs.entityFactory)
	return eddLoader.Load(r)
}

// LoadDecisionTables loads decision tables from a reader.
func (rs *RuleSet) LoadDecisionTables(r io.Reader) error {
	// Create a temporary session for loading
	tempSession := &loadSession{
		factory:    rs.entityFactory,
		dateParser: NewDateParser(),
	}

	dtLoader := loader.NewDTLoader(tempSession, rs.entityFactory)
	return dtLoader.Load(r)
}

// NewSession creates a new session for this rule set.
func (rs *RuleSet) NewSession() (dtrules.Session, error) {
	return NewSession(rs)
}

// AddEntryPoint adds a named entry point.
func (rs *RuleSet) AddEntryPoint(name, tableName string) {
	rs.entryPoints[name] = tableName
}

// GetEntryPoint returns the decision table name for an entry point.
func (rs *RuleSet) GetEntryPoint(name string) string {
	return rs.entryPoints[name]
}

// GetAttribute returns an attribute.
func (rs *RuleSet) GetAttribute(key string) interface{} {
	return rs.attributes[key]
}

// SetAttribute sets an attribute.
func (rs *RuleSet) SetAttribute(key string, value interface{}) {
	rs.attributes[key] = value
}

// GetDecisionTableNames returns all decision table names.
func (rs *RuleSet) GetDecisionTableNames() []*dtrules.RName {
	return rs.entityFactory.GetDecisionTableNames()
}

// GetEntityNames returns all entity names.
func (rs *RuleSet) GetEntityNames() []*dtrules.RName {
	return rs.entityFactory.GetEntityNames()
}

// loadSession is a minimal session implementation for loading.
type loadSession struct {
	factory    *entity.Factory
	dateParser dtrules.DateParser
	uniqueID   int
}

func (s *loadSession) GetState() dtrules.State {
	return nil
}

func (s *loadSession) GetEntityFactory() dtrules.EntityFactory {
	return s.factory
}

func (s *loadSession) GetUniqueID() int {
	id := s.uniqueID
	s.uniqueID++
	return id
}

func (s *loadSession) GetDateParser() dtrules.DateParser {
	return s.dateParser
}

func (s *loadSession) GetRuleSet() dtrules.RuleSet {
	return nil
}

func (s *loadSession) CreateEntity(name *dtrules.RName) (dtrules.Entity, error) {
	return s.factory.CreateEntity(s, name)
}

func (s *loadSession) GetEntityByID(id int) dtrules.Entity {
	return nil
}

func (s *loadSession) Compile(expr string) (dtrules.Object, error) {
	c := compiler.NewCompiler(s, s.factory)
	return c.Compile(expr)
}
