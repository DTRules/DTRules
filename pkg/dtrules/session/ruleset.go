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
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/compiler"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/loader"
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

// LoadEDDJSON loads an EDD from a JSON reader.
func (rs *RuleSet) LoadEDDJSON(r io.Reader) error {
	tempSession := &loadSession{
		factory:    rs.entityFactory,
		dateParser: NewDateParser(),
	}

	jsonLoader := loader.NewJSONEDDLoader(tempSession, rs.entityFactory)
	return jsonLoader.Load(r)
}

// LoadDecisionTables loads decision tables from a reader.
func (rs *RuleSet) LoadDecisionTables(r io.Reader) error {
	// Create a temporary session for loading
	tempSession := &loadSession{
		factory:    rs.entityFactory,
		dateParser: NewDateParser(),
	}

	dtLoader := loader.NewDTLoader(tempSession, rs.entityFactory)
	// Build the symbol table from the EDD so the EL compiler can pick the
	// right arithmetic dispatch (bigint × int → bigint, etc.) when the DSL
	// references typed fields. Without this the compiler defaults to integer
	// arithmetic for every iexpr × iexpr multiply and loses type promotion.
	dtLoader.SetSymbols(rs.buildSymbolTable())
	return dtLoader.Load(r)
}

// buildSymbolTable walks the entity factory's registered reference entities
// and returns a flat map of "entity.field" and "field" → type-name suitable
// for the EL compiler's SetSymbols.
func (rs *RuleSet) buildSymbolTable() map[string]string {
	symbols := map[string]string{}
	for _, ent := range rs.entityFactory.GetRefEntities() {
		entityName := ent.GetName().StringValue()
		for _, entry := range ent.GetEntries() {
			if entry.Type == nil {
				continue
			}
			fieldName := entry.Attribute.StringValue()
			typeName := entry.Type.GetName().StringValue()
			symbols[fieldName] = typeName
			symbols[entityName+"."+fieldName] = typeName
		}
	}
	return symbols
}

// LoadFromDirectory loads all XML files from a directory.
// EDDs are loaded first (ordered by their numbering), then decision tables.
// This method scans the directory recursively for *.xml files and loads them
// in the correct order based on TABLE_NUMBER (for DTs) and file_path numbering (for EDDs).
//
// Example:
//
//	rs := session.NewRuleSet("TaxReturn")
//	err := rs.LoadFromDirectory("./sampleprojects/TaxReturn/xml")
func (rs *RuleSet) LoadFromDirectory(dirPath string) error {
	return loader.LoadRulesFromDirectory(rs, dirPath)
}

// LoadFromFS loads all XML files from an fs.FS under the given root path.
// Use "." as root to load from the FS root, or a subdirectory like "rules/xml".
//
// Example:
//
//	//go:embed rules/xml
//	var rulesFS embed.FS
//
//	rs := session.NewRuleSet("TaxReturn")
//	err := rs.LoadFromFS(rulesFS, "rules/xml")
func (rs *RuleSet) LoadFromFS(fsys fs.FS, root string) error {
	return loader.LoadRulesFromFS(rs, fsys, root)
}

// LoadEDDFile loads an EDD from a file path.
func (rs *RuleSet) LoadEDDFile(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open EDD file %s: %w", filePath, err)
	}
	defer f.Close()
	return rs.LoadEDD(f)
}

// LoadDecisionTablesFile loads decision tables from a file path.
func (rs *RuleSet) LoadDecisionTablesFile(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open DT file %s: %w", filePath, err)
	}
	defer f.Close()
	return rs.LoadDecisionTables(f)
}

// LoadFromPath loads rules from a path (either a directory or individual files).
// If the path is a directory, it loads all XML files recursively.
// If the path is a file, it determines whether it's an EDD or DT and loads it.
// If eddPath and dtPath are both provided, they are loaded as individual files.
//
// Examples:
//   // Load from directory
//   rs.LoadFromPath("./sampleprojects/TaxReturn/xml", "", "")
//
//   // Load from individual files
//   rs.LoadFromPath("", "./xml/TaxReturn_edd.xml", "./xml/TaxReturn_dt.xml")
//
//   // Auto-detect from directory
//   rs.LoadFromPath("./xml", "", "")
func (rs *RuleSet) LoadFromPath(path, eddPath, dtPath string) error {
	// If individual files are specified, load them
	if eddPath != "" && dtPath != "" {
		if err := rs.LoadEDDFile(eddPath); err != nil {
			return err
		}
		return rs.LoadDecisionTablesFile(dtPath)
	}

	// If only path is specified, check if it's a directory or file
	if path != "" {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("failed to stat path %s: %w", path, err)
		}

		if info.IsDir() {
			// Load from directory
			return rs.LoadFromDirectory(path)
		}

		// It's a file - try to determine type and load
		return fmt.Errorf("single file loading requires both -edd and -dt paths, or use a directory path")
	}

	return fmt.Errorf("must specify either path or both eddPath and dtPath")
}

// LoadRulesFromFS loads rules from an fs.FS into a new RuleSet.
// root is the subdirectory inside fsys that contains the XML tree (use "." for the FS root).
//
// Example:
//
//	//go:embed rules/xml
//	var rulesFS embed.FS
//
//	rs, err := session.LoadRulesFromFS("TaxReturn", rulesFS, "rules/xml")
func LoadRulesFromFS(name string, fsys fs.FS, root string) (*RuleSet, error) {
	rs := NewRuleSet(name)
	if rs == nil {
		return nil, fmt.Errorf("invalid rule set name: %s", name)
	}
	if err := rs.LoadFromFS(fsys, root); err != nil {
		return nil, err
	}
	return rs, nil
}

// LoadRulesFromDirectory is a package-level convenience function for loading
// rules from a directory into a new RuleSet.
//
// Example:
//
//	rs, err := session.LoadRulesFromDirectory("TaxReturn", "./sampleprojects/TaxReturn/xml")
func LoadRulesFromDirectory(name, dirPath string) (*RuleSet, error) {
	rs := NewRuleSet(name)
	if rs == nil {
		return nil, fmt.Errorf("invalid rule set name: %s", name)
	}
	err := rs.LoadFromDirectory(dirPath)
	if err != nil {
		return nil, err
	}
	return rs, nil
}

// LoadRulesFromFiles is a package-level convenience function for loading
// rules from EDD and DT files into a new RuleSet.
//
// Example:
//   rs, err := session.LoadRulesFromFiles("TaxReturn",
//       "./xml/TaxReturn_edd.xml",
//       "./xml/TaxReturn_dt.xml")
func LoadRulesFromFiles(name, eddPath, dtPath string) (*RuleSet, error) {
	rs := NewRuleSet(name)
	if rs == nil {
		return nil, fmt.Errorf("invalid rule set name: %s", name)
	}
	if err := rs.LoadEDDFile(eddPath); err != nil {
		return nil, err
	}
	if err := rs.LoadDecisionTablesFile(dtPath); err != nil {
		return nil, err
	}
	return rs, nil
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
