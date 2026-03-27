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

// Package repository provides interfaces and implementations for loading
// compiled rulesets without requiring a session or runtime.
//
// This package separates the concerns of rule loading from rule execution,
// allowing rules to be:
//   - Loaded from XML (EDD + DT files)
//   - Loaded from pre-compiled binary format
//   - Validated independently of execution
//   - Serialized for distribution
//
// Usage:
//
//	repo := repository.New()
//
//	// Load from XML
//	rules, err := repo.LoadFromXML(eddReader, dtReader)
//
//	// Or load pre-compiled
//	rules, err := repo.LoadCompiled(reader)
//
//	// Validate
//	errors := repo.Validate(rules)
//
//	// Save compiled
//	err = repo.Save(rules, writer)
package repository

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/DTRules/DTRules/go/pkg/dtrules"
	"github.com/DTRules/DTRules/go/pkg/dtrules/operators"
	"github.com/DTRules/DTRules/go/pkg/dtrules/ruleset"
)

// Repository provides methods for loading and managing compiled rulesets.
type Repository interface {
	// LoadFromXML loads rules from EDD and DT XML files.
	// eddReader provides the Entity Data Dictionary.
	// dtReader provides the Decision Tables.
	LoadFromXML(eddReader, dtReader io.Reader) (*ruleset.CompiledRuleSet, error)

	// LoadCompiled loads a pre-compiled ruleset from binary format.
	LoadCompiled(r io.Reader) (*ruleset.CompiledRuleSet, error)

	// Save writes a compiled ruleset to binary format.
	Save(rs *ruleset.CompiledRuleSet, w io.Writer) error

	// Validate checks a ruleset for errors.
	// Returns a list of validation errors, or empty slice if valid.
	Validate(rs *ruleset.CompiledRuleSet) []error
}

// XMLRepository implements Repository using XML file loading.
type XMLRepository struct {
	// strictMode causes validation errors to be fatal during loading
	strictMode bool
}

// New creates a new XMLRepository.
func New() *XMLRepository {
	return &XMLRepository{}
}

// NewStrict creates a new XMLRepository with strict validation.
func NewStrict() *XMLRepository {
	return &XMLRepository{strictMode: true}
}

// LoadFromXML loads rules from EDD and DT XML files.
func (r *XMLRepository) LoadFromXML(eddReader, dtReader io.Reader) (*ruleset.CompiledRuleSet, error) {
	// Parse EDD
	eddData, err := io.ReadAll(eddReader)
	if err != nil {
		return nil, fmt.Errorf("read EDD: %w", err)
	}

	var edd eddFile
	if err := xml.Unmarshal(eddData, &edd); err != nil {
		return nil, fmt.Errorf("parse EDD XML: %w", err)
	}

	// Parse DT
	dtData, err := io.ReadAll(dtReader)
	if err != nil {
		return nil, fmt.Errorf("read DT: %w", err)
	}

	var dt dtFile
	if err := xml.Unmarshal(dtData, &dt); err != nil {
		return nil, fmt.Errorf("parse DT XML: %w", err)
	}

	// Create compiled ruleset
	rs := ruleset.NewCompiledRuleSet(dt.Name)
	if rs.Name == "" {
		rs.Name = "unnamed"
	}

	// Convert EDD entities
	for _, entity := range edd.Entities {
		ed := convertEntityDef(&entity)
		rs.EntityDefs = append(rs.EntityDefs, ed)
	}

	// Convert DT tables
	for _, table := range dt.Tables {
		ct, err := r.convertTable(&table)
		if err != nil {
			if r.strictMode {
				return nil, fmt.Errorf("compile table %s: %w", table.Name, err)
			}
			// In non-strict mode, skip tables with errors
			continue
		}
		rs.AddTable(ct)
	}

	return rs, nil
}

// LoadCompiled loads a pre-compiled ruleset from binary format.
func (r *XMLRepository) LoadCompiled(reader io.Reader) (*ruleset.CompiledRuleSet, error) {
	return ruleset.Deserialize(reader)
}

// Save writes a compiled ruleset to binary format.
func (r *XMLRepository) Save(rs *ruleset.CompiledRuleSet, w io.Writer) error {
	return rs.Serialize(w)
}

// Validate checks a ruleset for errors.
func (r *XMLRepository) Validate(rs *ruleset.CompiledRuleSet) []error {
	var errs []error

	if rs.Name == "" {
		errs = append(errs, errors.New("ruleset name is empty"))
	}

	// Check for duplicate table names
	tableNames := make(map[string]bool)
	for name := range rs.Tables {
		if tableNames[name] {
			errs = append(errs, fmt.Errorf("duplicate table name: %s", name))
		}
		tableNames[name] = true
	}

	// Validate each table
	for name, table := range rs.Tables {
		tableErrs := r.validateTable(name, table)
		errs = append(errs, tableErrs...)
	}

	// Check entity definitions
	entityNames := make(map[string]bool)
	for _, ed := range rs.EntityDefs {
		if entityNames[ed.Name] {
			errs = append(errs, fmt.Errorf("duplicate entity name: %s", ed.Name))
		}
		entityNames[ed.Name] = true
	}

	return errs
}

func (r *XMLRepository) validateTable(name string, table *ruleset.CompiledTable) []error {
	var errs []error

	if table.Name == "" {
		errs = append(errs, fmt.Errorf("table %s: name is empty", name))
	}

	// Check that condition indices in the tree are valid
	if table.DecisionTree != nil {
		errs = append(errs, r.validateNode(name, table, table.DecisionTree)...)
	}

	// Check that action indices in columns are valid
	for i, col := range table.Columns {
		for _, idx := range col.ActionIndices {
			if idx < 0 || idx >= len(table.Actions) {
				errs = append(errs, fmt.Errorf("table %s column %d: invalid action index %d", name, i, idx))
			}
		}
	}

	return errs
}

func (r *XMLRepository) validateNode(tableName string, table *ruleset.CompiledTable, node *ruleset.CompiledNode) []error {
	var errs []error

	if node.Type == ruleset.NodeTypeCondition {
		if node.ConditionIndex < 0 || node.ConditionIndex >= len(table.Conditions) {
			errs = append(errs, fmt.Errorf("table %s: invalid condition index %d", tableName, node.ConditionIndex))
		}
		if node.TrueBranch != nil {
			errs = append(errs, r.validateNode(tableName, table, node.TrueBranch)...)
		}
		if node.FalseBranch != nil {
			errs = append(errs, r.validateNode(tableName, table, node.FalseBranch)...)
		}
	} else {
		for _, idx := range node.ActionIndices {
			if idx < 0 || idx >= len(table.Actions) {
				errs = append(errs, fmt.Errorf("table %s: invalid action index %d", tableName, idx))
			}
		}
	}

	return errs
}

// convertEntityDef converts XML entity to EntityDefinition.
func convertEntityDef(entity *eddEntity) ruleset.EntityDefinition {
	ed := ruleset.EntityDefinition{
		Name:       entity.Name,
		Readonly:   entity.Readonly == "true",
		Attributes: make([]ruleset.AttributeDefinition, 0, len(entity.Fields)),
	}

	for _, field := range entity.Fields {
		attr := ruleset.AttributeDefinition{
			Name:         field.Name,
			Type:         parseAttributeType(field.Type),
			Subtype:      field.Subtype,
			DefaultValue: field.DefaultValue,
			Input:        field.Input == "true",
			Output:       field.Output == "true",
		}
		ed.Attributes = append(ed.Attributes, attr)
	}

	return ed
}

func parseAttributeType(typeStr string) ruleset.AttributeType {
	switch strings.ToLower(typeStr) {
	case "integer", "int", "long":
		return ruleset.AttrTypeInteger
	case "double", "float", "number":
		return ruleset.AttrTypeDouble
	case "string", "text":
		return ruleset.AttrTypeString
	case "boolean", "bool":
		return ruleset.AttrTypeBoolean
	case "date":
		return ruleset.AttrTypeDate
	case "array", "list":
		return ruleset.AttrTypeArray
	case "entity":
		return ruleset.AttrTypeEntity
	case "table", "map", "hashtable":
		return ruleset.AttrTypeTable
	case "name":
		return ruleset.AttrTypeName
	default:
		return ruleset.AttrTypeUnknown
	}
}

// convertTable converts XML table to CompiledTable.
func (r *XMLRepository) convertTable(table *dtTable) (*ruleset.CompiledTable, error) {
	tableType := parseTableType(table.Type)
	ct := ruleset.NewCompiledTable(table.Name, tableType)

	// Compile context
	if table.Contexts != nil && len(table.Contexts.Items) > 0 {
		bc, err := compileExpression(table.Contexts.Items[0].Value)
		if err != nil {
			return nil, fmt.Errorf("compile context: %w", err)
		}
		ct.Context = bc
	}

	// Compile initial actions
	if table.InitialActions != nil {
		for i, ia := range table.InitialActions.Items {
			if ia.Value == "" {
				ct.InitialActions = append(ct.InitialActions, nil)
				continue
			}
			bc, err := compileExpression(ia.Value)
			if err != nil {
				return nil, fmt.Errorf("compile initial action %d: %w", i, err)
			}
			ct.InitialActions = append(ct.InitialActions, bc)
		}
	}

	// Compile conditions
	if table.Conditions != nil {
		for i, cond := range table.Conditions.Items {
			if cond.Value == "" {
				ct.Conditions = append(ct.Conditions, nil)
				continue
			}
			bc, err := compileExpression(cond.Value)
			if err != nil {
				return nil, fmt.Errorf("compile condition %d: %w", i, err)
			}
			ct.Conditions = append(ct.Conditions, bc)
		}
	}

	// Compile actions
	if table.Actions != nil {
		for i, action := range table.Actions.Items {
			if action.Value == "" {
				ct.Actions = append(ct.Actions, nil)
				continue
			}
			bc, err := compileExpression(action.Value)
			if err != nil {
				return nil, fmt.Errorf("compile action %d: %w", i, err)
			}
			ct.Actions = append(ct.Actions, bc)
		}
	}

	// Compile policy statements
	if table.PolicyStatements != nil {
		for i, ps := range table.PolicyStatements.Items {
			if ps.Value == "" {
				ct.PolicyStatements = append(ct.PolicyStatements, nil)
				continue
			}
			bc, err := compileExpression(ps.Value)
			if err != nil {
				return nil, fmt.Errorf("compile policy statement %d: %w", i, err)
			}
			ct.PolicyStatements = append(ct.PolicyStatements, bc)
		}
	}

	// Build decision tree based on table structure
	// This requires analyzing the column specifications
	if tableType == ruleset.TableTypeBalanced {
		ct.DecisionTree = r.buildDecisionTree(table, ct)
	} else {
		ct.Columns = r.buildColumnSpecs(table, ct)
	}

	return ct, nil
}

func parseTableType(typeStr string) ruleset.TableType {
	switch strings.ToUpper(typeStr) {
	case "FIRST":
		return ruleset.TableTypeFirst
	case "ALL":
		return ruleset.TableTypeAll
	default:
		return ruleset.TableTypeBalanced
	}
}

// compileExpression compiles a postfix expression to bytecode.
// This is a standalone bytecode compiler that doesn't require a session.
func compileExpression(expr string) (*dtrules.BytecodeChunk, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, nil
	}

	// Tokenize the expression
	tokens := tokenize(expr)
	if len(tokens) == 0 {
		return nil, nil
	}

	bc := dtrules.NewBytecodeChunk()

	for _, token := range tokens {
		if err := compileTokenToBytecode(bc, token); err != nil {
			return nil, fmt.Errorf("failed to compile token '%s': %w", token, err)
		}
	}

	return bc, nil
}

// tokenize splits an expression into tokens.
func tokenize(expr string) []string {
	var tokens []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)
	braceDepth := 0

	for i := 0; i < len(expr); i++ {
		ch := expr[i]

		// Handle quoted strings
		if !inQuote && (ch == '"' || ch == '\'') {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			inQuote = true
			quoteChar = ch
			current.WriteByte(ch)
			continue
		}

		if inQuote {
			current.WriteByte(ch)
			if ch == quoteChar && (i == 0 || expr[i-1] != '\\') {
				tokens = append(tokens, current.String())
				current.Reset()
				inQuote = false
			}
			continue
		}

		// Handle braces (code blocks)
		if ch == '{' {
			if braceDepth == 0 && current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			braceDepth++
			current.WriteByte(ch)
			continue
		}

		if ch == '}' {
			current.WriteByte(ch)
			braceDepth--
			if braceDepth == 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}

		// Handle whitespace
		if braceDepth == 0 && (ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r') {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteByte(ch)
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// compileTokenToBytecode compiles a single token to bytecode.
func compileTokenToBytecode(bc *dtrules.BytecodeChunk, token string) error {
	if token == "" {
		return nil
	}

	// Handle quoted strings
	if len(token) >= 2 && (token[0] == '"' || token[0] == '\'') {
		str := token[1 : len(token)-1]
		bc.EmitPushConstant(dtrules.NewValueString(str))
		return nil
	}

	// Handle code blocks - compile inner expression
	if len(token) >= 2 && token[0] == '{' && token[len(token)-1] == '}' {
		inner := token[1 : len(token)-1]
		innerBC, err := compileExpression(inner)
		if err != nil {
			return err
		}
		// Store the inner bytecode as a constant
		// For execution, the runtime will need to handle this
		if innerBC != nil {
			// For now, we'll store a placeholder - actual block execution
			// would require the Value type to support bytecode chunks
			bc.EmitPushConstant(dtrules.NewValueString("{" + inner + "}"))
		}
		return nil
	}

	// Handle special literals
	switch strings.ToLower(token) {
	case "true":
		bc.Emit(dtrules.OpPushTrue)
		return nil
	case "false":
		bc.Emit(dtrules.OpPushFalse)
		return nil
	case "null":
		bc.Emit(dtrules.OpPushNull)
		return nil
	}

	// Try as integer
	if i, err := dtrules.GetRIntegerValueFromString(token); err == nil {
		v, _ := i.LongValue()
		bc.EmitPushConstant(dtrules.NewValueInteger(v))
		return nil
	}

	// Try as double
	if d, err := dtrules.GetRDoubleValueFromString(token); err == nil {
		v, _ := d.DoubleValue()
		bc.EmitPushConstant(dtrules.NewValueDouble(v))
		return nil
	}

	// Handle literal names starting with /
	if strings.HasPrefix(token, "/") {
		literalName := token[1:]
		name := dtrules.GetRName(literalName)
		if name == nil {
			return fmt.Errorf("invalid name syntax: %s", literalName)
		}
		bc.EmitPushName(name)
		return nil
	}

	// Get the name for this token
	name := dtrules.GetRName(token)
	if name == nil {
		return fmt.Errorf("invalid name syntax: %s", token)
	}

	// Check if it's an operator - use indexed lookup
	if idx, ok := operators.GetIndex(name); ok {
		// Map common operators to dedicated opcodes
		switch token {
		case "+":
			bc.Emit(dtrules.OpAdd)
		case "-":
			bc.Emit(dtrules.OpSub)
		case "*":
			bc.Emit(dtrules.OpMul)
		case "/":
			bc.Emit(dtrules.OpDiv)
		case "=", "eq":
			bc.Emit(dtrules.OpEq)
		case "!=", "ne":
			bc.Emit(dtrules.OpNe)
		case "<", "lt":
			bc.Emit(dtrules.OpLt)
		case "<=", "le":
			bc.Emit(dtrules.OpLe)
		case ">", "gt":
			bc.Emit(dtrules.OpGt)
		case ">=", "ge":
			bc.Emit(dtrules.OpGe)
		case "and":
			bc.Emit(dtrules.OpAnd)
		case "or":
			bc.Emit(dtrules.OpOr)
		case "not":
			bc.Emit(dtrules.OpNot)
		case "xor":
			bc.Emit(dtrules.OpXor)
		case "dup":
			bc.Emit(dtrules.OpDup)
		case "swap":
			bc.Emit(dtrules.OpSwap)
		case "pop":
			bc.Emit(dtrules.OpPop)
		case "def":
			bc.Emit(dtrules.OpDef)
		case "negate":
			bc.Emit(dtrules.OpNeg)
		default:
			bc.EmitOperator(idx)
		}
		return nil
	}

	// Otherwise, treat as a name to be looked up at runtime
	bc.EmitPushName(name)
	bc.Emit(dtrules.OpLookup)
	return nil
}

// buildDecisionTree builds the decision tree for a BALANCED table.
func (r *XMLRepository) buildDecisionTree(table *dtTable, ct *ruleset.CompiledTable) *ruleset.CompiledNode {
	// Parse column entries to understand the tree structure
	numConditions := len(ct.Conditions)
	if numConditions == 0 {
		// No conditions - just an action node
		actionIndices := make([]int, 0)
		for i := range ct.Actions {
			if ct.Actions[i] != nil {
				actionIndices = append(actionIndices, i)
			}
		}
		return ruleset.NewActionNode(actionIndices, 0)
	}

	// Build tree recursively starting from condition 0
	return r.buildNode(table, ct, 0, nil, 0)
}

// buildNode recursively builds decision tree nodes.
func (r *XMLRepository) buildNode(table *dtTable, ct *ruleset.CompiledTable, condIndex int, columnMask []bool, depth int) *ruleset.CompiledNode {
	if condIndex >= len(ct.Conditions) || ct.Conditions[condIndex] == nil {
		// Reached end of conditions - create action node
		actionIndices := r.findMatchingActions(table, ct, columnMask)
		return ruleset.NewActionNode(actionIndices, depth)
	}

	// Create condition node
	node := ruleset.NewConditionNode(condIndex)

	// Build true branch (condition is true)
	trueMask := make([]bool, len(columnMask)+1)
	copy(trueMask, columnMask)
	trueMask = append(trueMask, true)
	node.TrueBranch = r.buildNode(table, ct, condIndex+1, trueMask, depth+1)

	// Build false branch (condition is false)
	falseMask := make([]bool, len(columnMask)+1)
	copy(falseMask, columnMask)
	falseMask = append(falseMask, false)
	node.FalseBranch = r.buildNode(table, ct, condIndex+1, falseMask, depth+1)

	return node
}

// findMatchingActions finds action indices that match the column mask.
func (r *XMLRepository) findMatchingActions(table *dtTable, ct *ruleset.CompiledTable, columnMask []bool) []int {
	// For now, return all non-nil actions
	// A full implementation would parse the column entries from the XML
	actionIndices := make([]int, 0)
	for i := range ct.Actions {
		if ct.Actions[i] != nil {
			actionIndices = append(actionIndices, i)
		}
	}
	return actionIndices
}

// buildColumnSpecs builds column specifications for FIRST/ALL tables.
func (r *XMLRepository) buildColumnSpecs(table *dtTable, ct *ruleset.CompiledTable) []ruleset.ColumnSpec {
	// Parse column entries from the XML
	// For now, create a simple single-column spec
	specs := make([]ruleset.ColumnSpec, 0)

	actionIndices := make([]int, 0)
	for i := range ct.Actions {
		if ct.Actions[i] != nil {
			actionIndices = append(actionIndices, i)
		}
	}

	if len(actionIndices) > 0 {
		specs = append(specs, ruleset.ColumnSpec{
			ConditionMask:    0,
			ConditionNegMask: 0,
			ActionIndices:    actionIndices,
		})
	}

	return specs
}

// XML structure types for parsing

type eddFile struct {
	XMLName  xml.Name    `xml:"edd"`
	Name     string      `xml:"name,attr"`
	Entities []eddEntity `xml:"entity"`
}

type eddEntity struct {
	Name     string     `xml:"entityname,attr"`
	Readonly string     `xml:"readonly,attr"`
	Fields   []eddField `xml:"field"`
}

type eddField struct {
	Name         string `xml:"fieldname,attr"`
	Type         string `xml:"type,attr"`
	Subtype      string `xml:"subtype,attr"`
	DefaultValue string `xml:"defaultvalue,attr"`
	Input        string `xml:"input,attr"`
	Output       string `xml:"output,attr"`
}

type dtFile struct {
	XMLName xml.Name  `xml:"decision_tables"`
	Name    string    `xml:"name,attr"`
	Tables  []dtTable `xml:"decision_table"`
}

type dtTable struct {
	Name             string            `xml:"name,attr"`
	Type             string            `xml:"type,attr"`
	Contexts         *dtSection        `xml:"contexts"`
	InitialActions   *dtSection        `xml:"initialactions"`
	Conditions       *dtSection        `xml:"conditions"`
	Actions          *dtSection        `xml:"actions"`
	PolicyStatements *dtSection        `xml:"policystatements"`
}

type dtSection struct {
	Items []dtItem `xml:"item"`
}

type dtItem struct {
	Num   int    `xml:"num,attr"`
	Value string `xml:",chardata"`
}

// Verify interface compliance at compile time
var _ Repository = (*XMLRepository)(nil)
