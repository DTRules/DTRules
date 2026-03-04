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

// Package trace parses DTRules XML trace files and reconstructs engine
// state at any point in the execution history. A trace file records every
// entity push/pop, attribute assignment, array mutation, and decision-table
// invocation that occurred during rule execution. This package replays
// those events to rebuild the full interpreter state (entity stack,
// attribute values, arrays) up to a chosen node, enabling post-mortem
// debugging and analysis.
package trace

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
	"github.com/PaulSnow/DTRules/go/pkg/dtrules/session"
)

// Trace provides analysis capabilities for DTRules trace files.
// It loads XML trace files produced during rule execution, reconstructs
// the engine state at any point in the trace, and tracks attribute changes.
//
// Typical usage:
//
//	t := trace.NewTrace()
//	root, _ := t.Load("trace.xml")
//	t.Print(os.Stdout)                        // print the trace tree
//	sess, _ := t.SetState(ruleSet, t.Find(n)) // reconstruct state at node n
//	changes := t.GetChanges()                  // inspect attribute changes
type Trace struct {
	root         *TraceNode                   // Root of the trace tree
	entityTable  map[string]dtrules.Entity    // Entities created during replay
	arrayTable   map[int]*dtrules.RArray      // Arrays created during replay
	executeTable *TraceNode                   // Current execute_table node
	changes      *ChangeTracker               // Tracks attribute changes
	session      dtrules.Session              // Session for state reconstruction
	position     *TraceNode                   // Current position in the trace
	ruleSet      *session.RuleSet             // Rule set for creating sessions
}

// NewTrace creates a new Trace instance.
func NewTrace() *Trace {
	return &Trace{
		entityTable: make(map[string]dtrules.Entity),
		arrayTable:  make(map[int]*dtrules.RArray),
		changes:     NewChangeTracker(),
	}
}

// Load loads a trace from a file path.
func (t *Trace) Load(filepath string) (*TraceNode, error) {
	root, err := LoadFile(filepath)
	if err != nil {
		return nil, err
	}
	t.root = root
	return root, nil
}

// LoadReader loads a trace from a reader.
func (t *Trace) LoadReader(r io.Reader) (*TraceNode, error) {
	root, err := Load(r)
	if err != nil {
		return nil, err
	}
	t.root = root
	return root, nil
}

// Root returns the root trace node.
func (t *Trace) Root() *TraceNode {
	return t.root
}

// Print outputs the trace tree.
func (t *Trace) Print(w io.Writer) {
	if t.root != nil {
		t.root.Print(w)
	} else {
		fmt.Fprintln(w, "No tree has been loaded")
	}
}

// Find finds a node by its number.
func (t *Trace) Find(n int) *TraceNode {
	if t.root == nil {
		return nil
	}
	return t.root.Find(n)
}

// SetState reconstructs the rules engine state at the given position.
// Returns the session representing that state, or nil on error.
func (t *Trace) SetState(rs *session.RuleSet, position *TraceNode) (dtrules.Session, error) {
	t.position = position
	t.ruleSet = rs
	t.executeTable = nil
	t.changes.Clear()
	t.entityTable = make(map[string]dtrules.Entity)
	t.arrayTable = make(map[int]*dtrules.RArray)

	// Create a new session
	sess, err := rs.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	t.session = sess

	// Replay the trace up to the position
	if t.root != nil {
		if err := t.replayNode(t.root, position); err != nil {
			// Return session even on error - it represents state up to the error
			return t.session, err
		}
	}

	return t.session, nil
}

// replayNode recursively replays trace events to reconstruct state.
// Returns true when the target position is reached.
func (t *Trace) replayNode(node *TraceNode, target *TraceNode) error {
	// Found our position - we're done
	if node == target {
		return nil
	}

	state := t.session.GetState()

	switch node.Name {
	case "execute_table":
		t.executeTable = node
	case "entitypush":
		entity, err := t.getOrCreateEntity(node)
		if err != nil {
			return err
		}
		state.EntityPush(entity)
	case "entitypop":
		state.EntityPop()
	case "def":
		if err := t.handleDef(node); err != nil {
			return err
		}
	case "createentity":
		if _, err := t.getOrCreateEntity(node); err != nil {
			return err
		}
	case "newarray":
		t.handleNewArray(node)
	case "addto":
		if err := t.handleAddTo(node); err != nil {
			return err
		}
	case "remove":
		if err := t.handleRemove(node); err != nil {
			return err
		}
	}

	// Process children
	for _, child := range node.Children {
		if child == target {
			return nil
		}
		if err := t.replayNode(child, target); err != nil {
			return err
		}
	}

	return nil
}

// getOrCreateEntity gets an entity by ID, creating it if necessary.
func (t *Trace) getOrCreateEntity(node *TraceNode) (dtrules.Entity, error) {
	id := node.Attributes["id"]
	entityName := node.Attributes["entity"]
	if entityName == "" {
		entityName = node.Attributes["name"]
	}

	// Check if we already have this entity
	if entity, ok := t.entityTable[id]; ok {
		return entity, nil
	}

	// Create new entity
	name := dtrules.GetRName(entityName)
	if name == nil {
		return nil, fmt.Errorf("invalid entity name: %s", entityName)
	}

	entity, err := t.session.CreateEntity(name)
	if err != nil {
		return nil, fmt.Errorf("failed to create entity %s: %w", entityName, err)
	}

	t.entityTable[id] = entity
	return entity, nil
}

// handleDef handles attribute assignment.
func (t *Trace) handleDef(node *TraceNode) error {
	attrName := node.Attributes["name"]
	body := node.Body
	idStr := node.Attributes["id"]

	entity, ok := t.entityTable[idStr]
	if !ok {
		// Try to get/create the entity
		var err error
		entity, err = t.getOrCreateEntity(node)
		if err != nil {
			return err
		}
	}

	// Parse and execute the body to get the value
	var value dtrules.Object
	if body == "" {
		value = dtrules.GetRNull()
	} else {
		// Try to compile and execute the body
		compiled, err := t.session.Compile(body)
		if err != nil {
			// If compilation fails, try to interpret as literal value
			value = t.parseSimpleValue(body)
		} else {
			state := t.session.GetState()
			if err := compiled.Execute(state); err != nil {
				value = t.parseSimpleValue(body)
			} else {
				value, _ = state.DataPop()
			}
		}
	}

	// Set the attribute
	rname := dtrules.GetRName(attrName)
	if rname != nil {
		entity.Put(rname, value)
	}

	// Track the change
	if entityID, err := strconv.Atoi(idStr); err == nil {
		change := NewChange(entityID, strings.ToLower(attrName), t.executeTable)
		t.changes.Record(change)
	}

	return nil
}

// parseSimpleValue attempts to parse a simple literal value from the trace body.
func (t *Trace) parseSimpleValue(body string) dtrules.Object {
	body = strings.TrimSpace(body)

	// Try integer
	if i, err := strconv.ParseInt(body, 10, 64); err == nil {
		return dtrules.GetRIntegerValue(i)
	}

	// Try float
	if f, err := strconv.ParseFloat(body, 64); err == nil {
		return dtrules.GetRDoubleValue(f)
	}

	// Try boolean
	if body == "true" {
		return dtrules.GetRBoolean(true)
	}
	if body == "false" {
		return dtrules.GetRBoolean(false)
	}

	// Check for quoted string
	if len(body) >= 2 && body[0] == '"' && body[len(body)-1] == '"' {
		return dtrules.NewRString(body[1 : len(body)-1])
	}

	// Return as string
	return dtrules.NewRString(body)
}

// handleNewArray creates a new array from a trace node.
func (t *Trace) handleNewArray(node *TraceNode) {
	id := node.GetArrayID()
	if id == 0 {
		return
	}
	if _, ok := t.arrayTable[id]; !ok {
		t.arrayTable[id] = dtrules.NewArrayTraceInterface(id, true, false)
	}
}

// handleAddTo adds an element to an array.
func (t *Trace) handleAddTo(node *TraceNode) error {
	id := node.GetArrayID()
	if id == 0 {
		return nil
	}

	ar, ok := t.arrayTable[id]
	if !ok {
		// Create the array if it doesn't exist
		ar = dtrules.NewArrayTraceInterface(id, true, false)
		t.arrayTable[id] = ar
	}

	// Get the value to add
	body := node.Body
	var value dtrules.Object
	if body == "" {
		value = dtrules.GetRNull()
	} else {
		compiled, err := t.session.Compile(body)
		if err != nil {
			value = t.parseSimpleValue(body)
		} else {
			state := t.session.GetState()
			if err := compiled.Execute(state); err != nil {
				value = t.parseSimpleValue(body)
			} else {
				value, _ = state.DataPop()
			}
		}
	}

	ar.Add(value)
	return nil
}

// handleRemove removes an element from an array.
func (t *Trace) handleRemove(node *TraceNode) error {
	id := node.GetArrayID()
	if id == 0 {
		return nil
	}

	ar, ok := t.arrayTable[id]
	if !ok {
		return nil
	}

	body := node.Body
	if body == "" {
		return nil
	}

	compiled, err := t.session.Compile(body)
	if err != nil {
		return nil
	}

	state := t.session.GetState()
	if err := compiled.Execute(state); err != nil {
		return nil
	}

	value, _ := state.DataPop()
	ar.Remove(value)
	return nil
}

// InstancesOf returns all entities of the given type that were created
// up to the current position.
func (t *Trace) InstancesOf(entityName string) []dtrules.Entity {
	if t.session == nil || t.root == nil {
		return nil
	}

	var entityIDs []int
	t.root.SearchTree(t, entityName, &entityIDs)

	result := make([]dtrules.Entity, 0, len(entityIDs))
	for _, id := range entityIDs {
		if entity, ok := t.entityTable[strconv.Itoa(id)]; ok {
			result = append(result, entity)
		}
	}
	return result
}

// GetActions returns the actions executed at the current position.
func (t *Trace) GetActions() []int {
	if t.position == nil {
		return nil
	}
	return t.position.GetActions()
}

// GetActionsAt returns the actions executed at the given position.
func (t *Trace) GetActionsAt(position *TraceNode) []int {
	if position == nil {
		return nil
	}
	return position.GetActions()
}

// IsChanged returns the execute_table node where the entity's attribute
// was changed, or nil if not changed.
func (t *Trace) IsChanged(entityID int, attribute string) *TraceNode {
	return t.changes.GetExecuteTable(entityID, strings.ToLower(attribute))
}

// IsDefaultValue returns true if the attribute hasn't been changed from
// its default value.
func (t *Trace) IsDefaultValue(entityID int, attribute string) bool {
	return !t.changes.IsChanged(entityID, strings.ToLower(attribute))
}

// GetPosition returns the current trace position.
func (t *Trace) GetPosition() *TraceNode {
	return t.position
}

// GetSession returns the current session.
func (t *Trace) GetSession() dtrules.Session {
	return t.session
}

// GetEntityTable returns the entity table.
func (t *Trace) GetEntityTable() map[string]dtrules.Entity {
	return t.entityTable
}

// GetArrayTable returns the array table.
func (t *Trace) GetArrayTable() map[int]*dtrules.RArray {
	return t.arrayTable
}

// GetExecuteTable returns the current execute_table node.
func (t *Trace) GetExecuteTable() *TraceNode {
	return t.executeTable
}

// GetChanges returns all tracked changes.
func (t *Trace) GetChanges() []*Change {
	return t.changes.All()
}
