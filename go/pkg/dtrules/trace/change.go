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

package trace

import (
	"fmt"
)

// Change tracks when an entity attribute was modified during trace replay.
// It records which entity and attribute were changed, and the execute_table
// node that was responsible for the change.
type Change struct {
	EntityID     int        // Entity ID that was modified
	AttributeKey string     // Attribute name that was changed (lowercase)
	ExecuteTable *TraceNode // The execute_table node that caused this change
}

// changeKey is used for map lookups - uniquely identifies an entity/attribute pair.
type changeKey struct {
	entityID     int
	attributeKey string
}

// NewChange creates a new Change record.
func NewChange(entityID int, attribute string, executeTable *TraceNode) *Change {
	return &Change{
		EntityID:     entityID,
		AttributeKey: attribute,
		ExecuteTable: executeTable,
	}
}

// Key returns a comparable key for this change.
func (c *Change) Key() changeKey {
	return changeKey{
		entityID:     c.EntityID,
		attributeKey: c.AttributeKey,
	}
}

// String returns a string representation of the change.
func (c *Change) String() string {
	tableNum := -1
	if c.ExecuteTable != nil {
		tableNum = c.ExecuteTable.Number
	}
	return fmt.Sprintf("Change{entityID=%d, attribute=%s, executeTable=%d}",
		c.EntityID, c.AttributeKey, tableNum)
}

// ChangeTracker maintains a collection of changes made during trace replay.
type ChangeTracker struct {
	changes map[changeKey]*Change
}

// NewChangeTracker creates a new change tracker.
func NewChangeTracker() *ChangeTracker {
	return &ChangeTracker{
		changes: make(map[changeKey]*Change),
	}
}

// Record adds or updates a change record.
func (ct *ChangeTracker) Record(c *Change) {
	ct.changes[c.Key()] = c
}

// Get returns the change record for an entity/attribute pair, or nil if not found.
func (ct *ChangeTracker) Get(entityID int, attribute string) *Change {
	key := changeKey{entityID: entityID, attributeKey: attribute}
	return ct.changes[key]
}

// IsChanged returns true if the entity/attribute pair has been changed.
func (ct *ChangeTracker) IsChanged(entityID int, attribute string) bool {
	key := changeKey{entityID: entityID, attributeKey: attribute}
	_, ok := ct.changes[key]
	return ok
}

// GetExecuteTable returns the execute_table node that changed the given
// entity/attribute, or nil if not changed.
func (ct *ChangeTracker) GetExecuteTable(entityID int, attribute string) *TraceNode {
	c := ct.Get(entityID, attribute)
	if c != nil {
		return c.ExecuteTable
	}
	return nil
}

// Clear removes all tracked changes.
func (ct *ChangeTracker) Clear() {
	ct.changes = make(map[changeKey]*Change)
}

// All returns all tracked changes.
func (ct *ChangeTracker) All() []*Change {
	result := make([]*Change, 0, len(ct.changes))
	for _, c := range ct.changes {
		result = append(result, c)
	}
	return result
}

// Count returns the number of tracked changes.
func (ct *ChangeTracker) Count() int {
	return len(ct.changes)
}
