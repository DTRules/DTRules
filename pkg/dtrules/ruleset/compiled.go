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

// Package ruleset provides runtime-agnostic compiled rule structures.
//
// A CompiledRuleSet contains all the information needed to execute rules
// using any runtime (Go interpreter, x86-64 ASM, etc.). It stores only
// bytecode representations, not language-specific Object implementations.
//
// Key types:
//   - CompiledRuleSet: A complete set of decision tables and entity definitions
//   - CompiledTable: A single decision table with compiled conditions and actions
//   - CompiledNode: A node in the decision tree (condition or action)
//
// Usage:
//
//	// Load rules from XML
//	repo := repository.New()
//	rules, err := repo.LoadFromXML(eddReader, dtReader)
//
//	// Or load pre-compiled rules
//	rules, err := repo.LoadCompiled(reader)
//
//	// Execute with any runtime
//	sess := session.New(rules, goruntime.New())
//	err = sess.Execute("MyDecisionTable")
package ruleset

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// Format constants for serialization.
const (
	// RuleSetMagic is the magic number for serialized rulesets.
	RuleSetMagic = "DTRS"

	// RuleSetVersion is the current serialization format version.
	RuleSetVersion = 1
)

// CompiledRuleSet contains a complete set of compiled rules.
//
// This is the top-level structure that holds all decision tables and
// entity definitions needed to execute rules. It is runtime-agnostic
// and can be serialized for distribution.
type CompiledRuleSet struct {
	// Name identifies this ruleset.
	Name string

	// Version is an optional version string for the ruleset.
	Version string

	// EntityDefs defines the entity types and their attributes.
	EntityDefs []EntityDefinition

	// Tables contains all decision tables, keyed by name.
	Tables map[string]*CompiledTable

	// Metadata holds optional key-value metadata.
	Metadata map[string]string
}

// NewCompiledRuleSet creates a new empty ruleset.
func NewCompiledRuleSet(name string) *CompiledRuleSet {
	return &CompiledRuleSet{
		Name:       name,
		EntityDefs: make([]EntityDefinition, 0),
		Tables:     make(map[string]*CompiledTable),
		Metadata:   make(map[string]string),
	}
}

// GetTable returns a decision table by name, or nil if not found.
func (rs *CompiledRuleSet) GetTable(name string) *CompiledTable {
	return rs.Tables[name]
}

// AddTable adds a decision table to the ruleset.
func (rs *CompiledRuleSet) AddTable(table *CompiledTable) {
	rs.Tables[table.Name] = table
}

// TableNames returns the names of all tables in the ruleset.
func (rs *CompiledRuleSet) TableNames() []string {
	names := make([]string, 0, len(rs.Tables))
	for name := range rs.Tables {
		names = append(names, name)
	}
	return names
}

// EntityDefinition describes an entity type.
type EntityDefinition struct {
	// Name is the entity type name.
	Name string

	// Attributes defines the entity's fields.
	Attributes []AttributeDefinition

	// Readonly indicates if instances are immutable after creation.
	Readonly bool
}

// AttributeDefinition describes an entity attribute.
type AttributeDefinition struct {
	// Name is the attribute name.
	Name string

	// Type is the attribute's data type.
	Type AttributeType

	// Subtype provides additional type information (e.g., array element type).
	Subtype string

	// DefaultValue is the default value expression (as string).
	DefaultValue string

	// Input indicates if this attribute should be populated from input data.
	Input bool

	// Output indicates if this attribute should be included in output.
	Output bool
}

// AttributeType represents the type of an attribute.
type AttributeType int

const (
	// AttrTypeUnknown is the default/unspecified type.
	AttrTypeUnknown AttributeType = iota

	// AttrTypeInteger represents an integer value.
	AttrTypeInteger

	// AttrTypeDouble represents a floating-point value.
	AttrTypeDouble

	// AttrTypeString represents a string value.
	AttrTypeString

	// AttrTypeBoolean represents a boolean value.
	AttrTypeBoolean

	// AttrTypeDate represents a date value.
	AttrTypeDate

	// AttrTypeArray represents an array/list value.
	AttrTypeArray

	// AttrTypeEntity represents a reference to another entity.
	AttrTypeEntity

	// AttrTypeName represents an RName value.
	AttrTypeName
)

// TableType indicates the execution mode of a decision table.
type TableType int

const (
	// TableTypeBalanced executes the decision tree, branching on conditions.
	TableTypeBalanced TableType = iota

	// TableTypeFirst finds and executes the first matching column.
	TableTypeFirst

	// TableTypeAll executes all matching columns.
	TableTypeAll
)

// String returns the string representation of the table type.
func (t TableType) String() string {
	switch t {
	case TableTypeBalanced:
		return "BALANCED"
	case TableTypeFirst:
		return "FIRST"
	case TableTypeAll:
		return "ALL"
	default:
		return "UNKNOWN"
	}
}

// CompiledTable represents a single compiled decision table.
type CompiledTable struct {
	// Name identifies this table.
	Name string

	// Type determines how the table is executed.
	Type TableType

	// Context is bytecode to establish the execution context (optional).
	Context *dtrules.BytecodeChunk

	// InitialActions are executed before the decision tree.
	InitialActions []*dtrules.BytecodeChunk

	// Conditions contains the condition bytecode for each row.
	// Index corresponds to condition number (0-based).
	Conditions []*dtrules.BytecodeChunk

	// Actions contains the action bytecode for each row.
	// Index corresponds to action number (0-based).
	Actions []*dtrules.BytecodeChunk

	// PolicyStatements are executed after all other actions (optional).
	PolicyStatements []*dtrules.BytecodeChunk

	// DecisionTree is the root of the decision tree.
	// For BALANCED type, this is the entry point for execution.
	DecisionTree *CompiledNode

	// Columns stores column-specific information for FIRST/ALL types.
	Columns []ColumnSpec
}

// NewCompiledTable creates a new compiled table.
func NewCompiledTable(name string, tableType TableType) *CompiledTable {
	return &CompiledTable{
		Name:             name,
		Type:             tableType,
		InitialActions:   make([]*dtrules.BytecodeChunk, 0),
		Conditions:       make([]*dtrules.BytecodeChunk, 0),
		Actions:          make([]*dtrules.BytecodeChunk, 0),
		PolicyStatements: make([]*dtrules.BytecodeChunk, 0),
		Columns:          make([]ColumnSpec, 0),
	}
}

// ColumnSpec describes a column in a FIRST or ALL type table.
type ColumnSpec struct {
	// ConditionMask is a bitmask of which conditions must be true for this column.
	// Bit i is set if condition i must be true.
	ConditionMask uint64

	// ConditionNegMask is a bitmask of which conditions must be false.
	// Bit i is set if condition i must be false.
	ConditionNegMask uint64

	// ActionIndices lists which actions to execute for this column.
	ActionIndices []int
}

// NodeType indicates whether a node is a condition or action node.
type NodeType int

const (
	// NodeTypeCondition is a condition node that branches based on a boolean result.
	NodeTypeCondition NodeType = iota

	// NodeTypeAction is a terminal node that executes actions.
	NodeTypeAction
)

// CompiledNode represents a node in the decision tree.
//
// For condition nodes:
//   - ConditionIndex points to the condition in CompiledTable.Conditions
//   - TrueBranch and FalseBranch point to child nodes
//
// For action nodes:
//   - ActionIndices lists which actions to execute from CompiledTable.Actions
type CompiledNode struct {
	// Type indicates if this is a condition or action node.
	Type NodeType

	// ConditionIndex is the index into CompiledTable.Conditions.
	// Only valid for NodeTypeCondition.
	ConditionIndex int

	// TrueBranch is the node to execute if the condition is true.
	// Only valid for NodeTypeCondition.
	TrueBranch *CompiledNode

	// FalseBranch is the node to execute if the condition is false.
	// Only valid for NodeTypeCondition.
	FalseBranch *CompiledNode

	// ActionIndices lists which actions to execute.
	// Only valid for NodeTypeAction.
	ActionIndices []int

	// Column indicates which decision table column this node represents.
	// Used for tracing and debugging.
	Column int
}

// NewConditionNode creates a condition node.
func NewConditionNode(conditionIndex int) *CompiledNode {
	return &CompiledNode{
		Type:           NodeTypeCondition,
		ConditionIndex: conditionIndex,
		Column:         -1,
	}
}

// NewActionNode creates an action node.
func NewActionNode(actionIndices []int, column int) *CompiledNode {
	indices := make([]int, len(actionIndices))
	copy(indices, actionIndices)
	return &CompiledNode{
		Type:          NodeTypeAction,
		ActionIndices: indices,
		Column:        column,
	}
}

// Serialize writes the ruleset to binary format.
func (rs *CompiledRuleSet) Serialize(w io.Writer) error {
	// Magic number
	if _, err := w.Write([]byte(RuleSetMagic)); err != nil {
		return fmt.Errorf("write magic: %w", err)
	}

	// Version
	if err := binary.Write(w, binary.LittleEndian, uint8(RuleSetVersion)); err != nil {
		return fmt.Errorf("write version: %w", err)
	}

	// Name
	if err := writeString(w, rs.Name); err != nil {
		return fmt.Errorf("write name: %w", err)
	}

	// Version string
	if err := writeString(w, rs.Version); err != nil {
		return fmt.Errorf("write version string: %w", err)
	}

	// Entity definitions count
	if err := writeVarint(w, len(rs.EntityDefs)); err != nil {
		return fmt.Errorf("write entity count: %w", err)
	}

	// Entity definitions
	for i, ed := range rs.EntityDefs {
		if err := writeEntityDef(w, &ed); err != nil {
			return fmt.Errorf("write entity %d: %w", i, err)
		}
	}

	// Tables count
	if err := writeVarint(w, len(rs.Tables)); err != nil {
		return fmt.Errorf("write table count: %w", err)
	}

	// Tables
	for name, table := range rs.Tables {
		if err := writeString(w, name); err != nil {
			return fmt.Errorf("write table name %s: %w", name, err)
		}
		if err := writeTable(w, table); err != nil {
			return fmt.Errorf("write table %s: %w", name, err)
		}
	}

	// Metadata count
	if err := writeVarint(w, len(rs.Metadata)); err != nil {
		return fmt.Errorf("write metadata count: %w", err)
	}

	// Metadata
	for k, v := range rs.Metadata {
		if err := writeString(w, k); err != nil {
			return fmt.Errorf("write metadata key: %w", err)
		}
		if err := writeString(w, v); err != nil {
			return fmt.Errorf("write metadata value: %w", err)
		}
	}

	return nil
}

// Deserialize reads a ruleset from binary format.
func Deserialize(r io.Reader) (*CompiledRuleSet, error) {
	// Magic number
	magic := make([]byte, 4)
	if _, err := io.ReadFull(r, magic); err != nil {
		return nil, fmt.Errorf("read magic: %w", err)
	}
	if string(magic) != RuleSetMagic {
		return nil, errors.New("invalid ruleset magic number")
	}

	// Version
	var version uint8
	if err := binary.Read(r, binary.LittleEndian, &version); err != nil {
		return nil, fmt.Errorf("read version: %w", err)
	}
	if version != RuleSetVersion {
		return nil, fmt.Errorf("unsupported version: %d", version)
	}

	rs := &CompiledRuleSet{
		Tables:   make(map[string]*CompiledTable),
		Metadata: make(map[string]string),
	}

	// Name
	var err error
	rs.Name, err = readString(r)
	if err != nil {
		return nil, fmt.Errorf("read name: %w", err)
	}

	// Version string
	rs.Version, err = readString(r)
	if err != nil {
		return nil, fmt.Errorf("read version string: %w", err)
	}

	// Entity definitions
	entityCount, err := readVarint(r)
	if err != nil {
		return nil, fmt.Errorf("read entity count: %w", err)
	}

	rs.EntityDefs = make([]EntityDefinition, entityCount)
	for i := 0; i < entityCount; i++ {
		ed, err := readEntityDef(r)
		if err != nil {
			return nil, fmt.Errorf("read entity %d: %w", i, err)
		}
		rs.EntityDefs[i] = *ed
	}

	// Tables
	tableCount, err := readVarint(r)
	if err != nil {
		return nil, fmt.Errorf("read table count: %w", err)
	}

	for i := 0; i < tableCount; i++ {
		name, err := readString(r)
		if err != nil {
			return nil, fmt.Errorf("read table name: %w", err)
		}
		table, err := readTable(r)
		if err != nil {
			return nil, fmt.Errorf("read table %s: %w", name, err)
		}
		rs.Tables[name] = table
	}

	// Metadata
	metadataCount, err := readVarint(r)
	if err != nil {
		return nil, fmt.Errorf("read metadata count: %w", err)
	}

	for i := 0; i < metadataCount; i++ {
		k, err := readString(r)
		if err != nil {
			return nil, fmt.Errorf("read metadata key: %w", err)
		}
		v, err := readString(r)
		if err != nil {
			return nil, fmt.Errorf("read metadata value: %w", err)
		}
		rs.Metadata[k] = v
	}

	return rs, nil
}

// Helper functions for serialization

func writeVarint(w io.Writer, v int) error {
	buf := make([]byte, 10)
	n := binary.PutUvarint(buf, uint64(v))
	_, err := w.Write(buf[:n])
	return err
}

func readVarint(r io.Reader) (int, error) {
	var buf [1]byte
	var result uint64
	var shift uint
	for {
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return 0, err
		}
		b := buf[0]
		result |= uint64(b&0x7F) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
		if shift >= 64 {
			return 0, errors.New("varint overflow")
		}
	}
	return int(result), nil
}

func writeString(w io.Writer, s string) error {
	if err := writeVarint(w, len(s)); err != nil {
		return err
	}
	_, err := w.Write([]byte(s))
	return err
}

func readString(r io.Reader) (string, error) {
	length, err := readVarint(r)
	if err != nil {
		return "", err
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func writeEntityDef(w io.Writer, ed *EntityDefinition) error {
	if err := writeString(w, ed.Name); err != nil {
		return err
	}

	var flags uint8
	if ed.Readonly {
		flags |= 1
	}
	if _, err := w.Write([]byte{flags}); err != nil {
		return err
	}

	if err := writeVarint(w, len(ed.Attributes)); err != nil {
		return err
	}

	for _, attr := range ed.Attributes {
		if err := writeAttrDef(w, &attr); err != nil {
			return err
		}
	}

	return nil
}

func readEntityDef(r io.Reader) (*EntityDefinition, error) {
	ed := &EntityDefinition{}

	var err error
	ed.Name, err = readString(r)
	if err != nil {
		return nil, err
	}

	var flags [1]byte
	if _, err := io.ReadFull(r, flags[:]); err != nil {
		return nil, err
	}
	ed.Readonly = flags[0]&1 != 0

	attrCount, err := readVarint(r)
	if err != nil {
		return nil, err
	}

	ed.Attributes = make([]AttributeDefinition, attrCount)
	for i := 0; i < attrCount; i++ {
		attr, err := readAttrDef(r)
		if err != nil {
			return nil, err
		}
		ed.Attributes[i] = *attr
	}

	return ed, nil
}

func writeAttrDef(w io.Writer, attr *AttributeDefinition) error {
	if err := writeString(w, attr.Name); err != nil {
		return err
	}

	if err := writeVarint(w, int(attr.Type)); err != nil {
		return err
	}

	if err := writeString(w, attr.Subtype); err != nil {
		return err
	}

	if err := writeString(w, attr.DefaultValue); err != nil {
		return err
	}

	var flags uint8
	if attr.Input {
		flags |= 1
	}
	if attr.Output {
		flags |= 2
	}
	_, err := w.Write([]byte{flags})
	return err
}

func readAttrDef(r io.Reader) (*AttributeDefinition, error) {
	attr := &AttributeDefinition{}

	var err error
	attr.Name, err = readString(r)
	if err != nil {
		return nil, err
	}

	typeVal, err := readVarint(r)
	if err != nil {
		return nil, err
	}
	attr.Type = AttributeType(typeVal)

	attr.Subtype, err = readString(r)
	if err != nil {
		return nil, err
	}

	attr.DefaultValue, err = readString(r)
	if err != nil {
		return nil, err
	}

	var flags [1]byte
	if _, err := io.ReadFull(r, flags[:]); err != nil {
		return nil, err
	}
	attr.Input = flags[0]&1 != 0
	attr.Output = flags[0]&2 != 0

	return attr, nil
}

func writeTable(w io.Writer, table *CompiledTable) error {
	// Name (already written by caller) - skip

	// Type
	if err := writeVarint(w, int(table.Type)); err != nil {
		return err
	}

	// Context
	hasContext := table.Context != nil
	if hasContext {
		if _, err := w.Write([]byte{1}); err != nil {
			return err
		}
		if err := writeBytecode(w, table.Context); err != nil {
			return err
		}
	} else {
		if _, err := w.Write([]byte{0}); err != nil {
			return err
		}
	}

	// InitialActions
	if err := writeBytecodeArray(w, table.InitialActions); err != nil {
		return err
	}

	// Conditions
	if err := writeBytecodeArray(w, table.Conditions); err != nil {
		return err
	}

	// Actions
	if err := writeBytecodeArray(w, table.Actions); err != nil {
		return err
	}

	// PolicyStatements
	if err := writeBytecodeArray(w, table.PolicyStatements); err != nil {
		return err
	}

	// DecisionTree
	if table.DecisionTree != nil {
		if _, err := w.Write([]byte{1}); err != nil {
			return err
		}
		if err := writeNode(w, table.DecisionTree); err != nil {
			return err
		}
	} else {
		if _, err := w.Write([]byte{0}); err != nil {
			return err
		}
	}

	// Columns
	if err := writeVarint(w, len(table.Columns)); err != nil {
		return err
	}
	for _, col := range table.Columns {
		if err := binary.Write(w, binary.LittleEndian, col.ConditionMask); err != nil {
			return err
		}
		if err := binary.Write(w, binary.LittleEndian, col.ConditionNegMask); err != nil {
			return err
		}
		if err := writeVarint(w, len(col.ActionIndices)); err != nil {
			return err
		}
		for _, idx := range col.ActionIndices {
			if err := writeVarint(w, idx); err != nil {
				return err
			}
		}
	}

	return nil
}

func readTable(r io.Reader) (*CompiledTable, error) {
	table := &CompiledTable{}

	// Type
	typeVal, err := readVarint(r)
	if err != nil {
		return nil, err
	}
	table.Type = TableType(typeVal)

	// Context
	var hasContext [1]byte
	if _, err := io.ReadFull(r, hasContext[:]); err != nil {
		return nil, err
	}
	if hasContext[0] == 1 {
		table.Context, err = readBytecode(r)
		if err != nil {
			return nil, err
		}
	}

	// InitialActions
	table.InitialActions, err = readBytecodeArray(r)
	if err != nil {
		return nil, err
	}

	// Conditions
	table.Conditions, err = readBytecodeArray(r)
	if err != nil {
		return nil, err
	}

	// Actions
	table.Actions, err = readBytecodeArray(r)
	if err != nil {
		return nil, err
	}

	// PolicyStatements
	table.PolicyStatements, err = readBytecodeArray(r)
	if err != nil {
		return nil, err
	}

	// DecisionTree
	var hasTree [1]byte
	if _, err := io.ReadFull(r, hasTree[:]); err != nil {
		return nil, err
	}
	if hasTree[0] == 1 {
		table.DecisionTree, err = readNode(r)
		if err != nil {
			return nil, err
		}
	}

	// Columns
	colCount, err := readVarint(r)
	if err != nil {
		return nil, err
	}
	table.Columns = make([]ColumnSpec, colCount)
	for i := 0; i < colCount; i++ {
		if err := binary.Read(r, binary.LittleEndian, &table.Columns[i].ConditionMask); err != nil {
			return nil, err
		}
		if err := binary.Read(r, binary.LittleEndian, &table.Columns[i].ConditionNegMask); err != nil {
			return nil, err
		}
		actionCount, err := readVarint(r)
		if err != nil {
			return nil, err
		}
		table.Columns[i].ActionIndices = make([]int, actionCount)
		for j := 0; j < actionCount; j++ {
			table.Columns[i].ActionIndices[j], err = readVarint(r)
			if err != nil {
				return nil, err
			}
		}
	}

	return table, nil
}

func writeBytecode(w io.Writer, bc *dtrules.BytecodeChunk) error {
	if bc == nil {
		return writeVarint(w, 0)
	}
	data := bc.Serialize()
	if err := writeVarint(w, len(data)); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}

func readBytecode(r io.Reader) (*dtrules.BytecodeChunk, error) {
	length, err := readVarint(r)
	if err != nil {
		return nil, err
	}
	if length == 0 {
		return nil, nil
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	return dtrules.DeserializeBytecode(data)
}

func writeBytecodeArray(w io.Writer, chunks []*dtrules.BytecodeChunk) error {
	if err := writeVarint(w, len(chunks)); err != nil {
		return err
	}
	for _, chunk := range chunks {
		if err := writeBytecode(w, chunk); err != nil {
			return err
		}
	}
	return nil
}

func readBytecodeArray(r io.Reader) ([]*dtrules.BytecodeChunk, error) {
	count, err := readVarint(r)
	if err != nil {
		return nil, err
	}
	chunks := make([]*dtrules.BytecodeChunk, count)
	for i := 0; i < count; i++ {
		chunks[i], err = readBytecode(r)
		if err != nil {
			return nil, err
		}
	}
	return chunks, nil
}

func writeNode(w io.Writer, node *CompiledNode) error {
	// Type
	if err := writeVarint(w, int(node.Type)); err != nil {
		return err
	}

	// Column
	if err := writeVarint(w, node.Column); err != nil {
		return err
	}

	if node.Type == NodeTypeCondition {
		// ConditionIndex
		if err := writeVarint(w, node.ConditionIndex); err != nil {
			return err
		}

		// TrueBranch
		if node.TrueBranch != nil {
			if _, err := w.Write([]byte{1}); err != nil {
				return err
			}
			if err := writeNode(w, node.TrueBranch); err != nil {
				return err
			}
		} else {
			if _, err := w.Write([]byte{0}); err != nil {
				return err
			}
		}

		// FalseBranch
		if node.FalseBranch != nil {
			if _, err := w.Write([]byte{1}); err != nil {
				return err
			}
			if err := writeNode(w, node.FalseBranch); err != nil {
				return err
			}
		} else {
			if _, err := w.Write([]byte{0}); err != nil {
				return err
			}
		}
	} else {
		// ActionIndices
		if err := writeVarint(w, len(node.ActionIndices)); err != nil {
			return err
		}
		for _, idx := range node.ActionIndices {
			if err := writeVarint(w, idx); err != nil {
				return err
			}
		}
	}

	return nil
}

func readNode(r io.Reader) (*CompiledNode, error) {
	node := &CompiledNode{}

	// Type
	typeVal, err := readVarint(r)
	if err != nil {
		return nil, err
	}
	node.Type = NodeType(typeVal)

	// Column
	node.Column, err = readVarint(r)
	if err != nil {
		return nil, err
	}

	if node.Type == NodeTypeCondition {
		// ConditionIndex
		node.ConditionIndex, err = readVarint(r)
		if err != nil {
			return nil, err
		}

		// TrueBranch
		var hasTrue [1]byte
		if _, err := io.ReadFull(r, hasTrue[:]); err != nil {
			return nil, err
		}
		if hasTrue[0] == 1 {
			node.TrueBranch, err = readNode(r)
			if err != nil {
				return nil, err
			}
		}

		// FalseBranch
		var hasFalse [1]byte
		if _, err := io.ReadFull(r, hasFalse[:]); err != nil {
			return nil, err
		}
		if hasFalse[0] == 1 {
			node.FalseBranch, err = readNode(r)
			if err != nil {
				return nil, err
			}
		}
	} else {
		// ActionIndices
		actionCount, err := readVarint(r)
		if err != nil {
			return nil, err
		}
		node.ActionIndices = make([]int, actionCount)
		for i := 0; i < actionCount; i++ {
			node.ActionIndices[i], err = readVarint(r)
			if err != nil {
				return nil, err
			}
		}
	}

	return node, nil
}
