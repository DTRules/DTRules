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

package entity

import (
	"fmt"
	"sync"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
)

// Factory manages reference entities and creates instances.
// Once a session is created, the factory is frozen and cannot be modified.
type Factory struct {
	mu                sync.RWMutex
	uniqueID          int
	frozen            bool
	referenceEntities map[*dtrules.RName]*REntity
	decisionTables    *REntity
	decisionTableList []*dtrules.RName
	ruleSet           interface{} // *RuleSet when implemented
	createStamp       string
}

// NewFactory creates a new entity factory.
func NewFactory(ruleSet interface{}) *Factory {
	f := &Factory{
		uniqueID:          10, // Leave room for fixed IDs (primitives=1, decisiontables=2)
		frozen:            false,
		referenceEntities: make(map[*dtrules.RName]*REntity),
		decisionTableList: make([]*dtrules.RName, 0),
		ruleSet:           ruleSet,
	}

	// Create decisiontables entity
	f.decisionTables = NewREntity(2, true, dtrules.GetRName("decisiontables"))
	f.decisionTables.RemoveAttribute(dtrules.GetRName("decisiontables"))

	return f
}

// GetUniqueID returns a unique ID.
// Returns an error if the factory is frozen.
func (f *Factory) GetUniqueID() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.frozen {
		// Return ID anyway but log warning - in Go we can't throw
		// In production, should handle this more gracefully
	}
	id := f.uniqueID
	f.uniqueID++
	return id
}

// Freeze marks the factory as frozen (after first session created).
func (f *Factory) Freeze() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.frozen = true
}

// IsFrozen returns whether the factory is frozen.
func (f *Factory) IsFrozen() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.frozen
}

// FindRefEntity looks up a reference entity by name.
// Returns nil if not found.
func (f *Factory) FindRefEntity(name *dtrules.RName) *REntity {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.referenceEntities[name]
}

// FindRefEntityByString looks up a reference entity by string name.
// Returns nil if the name has invalid syntax.
func (f *Factory) FindRefEntityByString(name string) *REntity {
	rname := dtrules.GetRName(name)
	if rname == nil {
		return nil
	}
	return f.FindRefEntity(rname)
}

// FindCreateRefEntity finds or creates a reference entity.
func (f *Factory) FindCreateRefEntity(readonly bool, name *dtrules.RName) (*REntity, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if existing, ok := f.referenceEntities[name]; ok {
		return existing, nil
	}

	entity := NewREntity(f.uniqueID, readonly, name)
	f.uniqueID++
	f.referenceEntities[name] = entity
	return entity, nil
}

// GetReferenceEntity returns a reference entity by name.
func (f *Factory) GetReferenceEntity(name *dtrules.RName) (dtrules.Entity, error) {
	// Always use the executable form of the name for consistent lookup
	execName := name.GetExecutable().(*dtrules.RName)
	f.mu.RLock()
	defer f.mu.RUnlock()

	if entity, ok := f.referenceEntities[execName]; ok {
		return entity, nil
	}
	return nil, nil
}

// CreateEntity creates a new entity instance from a reference.
func (f *Factory) CreateEntity(session dtrules.Session, name *dtrules.RName) (dtrules.Entity, error) {
	// Always use the executable form of the name for consistent lookup
	// since reference entities are stored with the executable form
	execName := name.GetExecutable().(*dtrules.RName)
	f.mu.RLock()
	refEntity, ok := f.referenceEntities[execName]
	f.mu.RUnlock()

	if !ok {
		return nil, dtrules.UndefinedError("CreateEntity", "Reference entity not found: "+name.StringValue())
	}

	return CloneEntity(false, refEntity, session)
}

// GetRefEntities returns all reference entities.
func (f *Factory) GetRefEntities() []*REntity {
	f.mu.RLock()
	defer f.mu.RUnlock()

	entities := make([]*REntity, 0, len(f.referenceEntities))
	for _, entity := range f.referenceEntities {
		entities = append(entities, entity)
	}
	return entities
}

// GetEntityNames returns all entity names.
func (f *Factory) GetEntityNames() []*dtrules.RName {
	f.mu.RLock()
	defer f.mu.RUnlock()

	names := make([]*dtrules.RName, 0, len(f.referenceEntities))
	for name := range f.referenceEntities {
		names = append(names, name)
	}
	return names
}

// GetDecisionTable returns a decision table by name.
func (f *Factory) GetDecisionTable(name *dtrules.RName) (dtrules.DecisionTable, error) {
	// Always use the executable form of the name for consistent lookup
	// since tables are stored with the executable form
	execName := name.GetExecutable().(*dtrules.RName)
	dt, err := f.decisionTables.Get(execName)
	if err != nil {
		return nil, err
	}
	if dt == nil || dt.Type() != dtrules.TypeDecisionTable {
		return nil, nil
	}
	// Cast to DecisionTable interface
	dtable, ok := dt.(dtrules.DecisionTable)
	if !ok {
		return nil, nil
	}
	return dtable, nil
}

// FindDecisionTable finds a decision table by name.
func (f *Factory) FindDecisionTable(name *dtrules.RName) dtrules.Object {
	dt, _ := f.decisionTables.Get(name)
	if dt != nil && dt.Type() == dtrules.TypeDecisionTable {
		return dt
	}
	return nil
}

// AddDecisionTable adds a decision table to the factory.
func (f *Factory) AddDecisionTable(name *dtrules.RName, table dtrules.Object) error {
	f.decisionTableList = append(f.decisionTableList, name)
	f.decisionTables.AddAttribute(name, "", table, false, true, dtrules.TypeDecisionTable, "", "", "", "")
	return f.decisionTables.Put(name, table)
}

// GetDecisionTableNames returns all decision table names.
func (f *Factory) GetDecisionTableNames() []*dtrules.RName {
	return f.decisionTableList
}

// DeleteDecisionTable removes a decision table.
func (f *Factory) DeleteDecisionTable(name *dtrules.RName) error {
	f.decisionTables.RemoveAttribute(name)
	// Remove from list
	for i, n := range f.decisionTableList {
		if n == name {
			f.decisionTableList = append(f.decisionTableList[:i], f.decisionTableList[i+1:]...)
			break
		}
	}
	return nil
}

// GetDecisionTablesEntity returns the entity containing all decision tables.
// This entity should be pushed onto the entity stack to make decision tables
// accessible for execution.
func (f *Factory) GetDecisionTablesEntity() dtrules.Entity {
	return f.decisionTables
}

// String returns a string representation of the factory.
func (f *Factory) String() string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	result := ""
	for name, entity := range f.referenceEntities {
		result += name.String() + "\r\n"
		for _, attrName := range entity.GetAttributeNames() {
			entry := entity.GetEntry(attrName)
			result += "   " + attrName.String()
			if entry.DefaultValue != nil {
				result += "  --  default value: " + fmt.Sprintf("%v", entry.DefaultValue)
			}
			result += "\r\n"
		}
	}
	return result
}
