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

package loader

import (
	"encoding/xml"
	"fmt"
	"log"
	"io"
	"strings"

	"github.com/DTRules/DTRules/go/pkg/dtrules"
	"github.com/DTRules/DTRules/go/pkg/dtrules/compiler"
	"github.com/DTRules/DTRules/go/pkg/dtrules/decisiontable"
	"github.com/DTRules/DTRules/go/pkg/dtrules/entity"
)

// DTLoader loads Decision Table XML files.
type DTLoader struct {
	session  dtrules.Session
	factory  *entity.Factory
	compiler *compiler.Compiler
	errors   []error
}

// NewDTLoader creates a new Decision Table loader.
func NewDTLoader(session dtrules.Session, factory *entity.Factory) *DTLoader {
	return &DTLoader{
		session:  session,
		factory:  factory,
		compiler: compiler.NewCompiler(session, factory),
		errors:   make([]error, 0),
	}
}

// XML structures matching the actual DTRules decision table format

// DTFile represents the root decision_tables element
type DTFile struct {
	XMLName xml.Name  `xml:"decision_tables"`
	Tables  []DTTable `xml:"decision_table"`
}

// DTTable represents a single decision table
type DTTable struct {
	TableName        string             `xml:"table_name"`
	XlsFile          string             `xml:"xls_file"`
	AttributeFields  DTAttributeFields  `xml:"attribute_fields"`
	Contexts         DTContexts         `xml:"contexts"`
	InitialActions   DTInitialActions   `xml:"initial_actions"`
	Conditions       DTConditions       `xml:"conditions"`
	Actions          DTActions          `xml:"actions"`
	PolicyStatements DTPolicyStatements `xml:"policy_statements"`
}

// DTAttributeFields represents metadata about the table
// Note: Type can be <Type>, <TYPE>, or <type> depending on the XML source
type DTAttributeFields struct {
	Type          string `xml:"Type"`
	TypeUppercase string `xml:"TYPE"`
	TypeLowercase string `xml:"type"`
	Comments      string `xml:"COMMENTS"`
	CommentsLower string `xml:"comments"`
	FileName      string `xml:"File_Name"`
	TableNumber   string `xml:"TABLE_NUMBER"`
}

// GetType returns the type field, checking all case variants
func (f *DTAttributeFields) GetType() string {
	if f.Type != "" {
		return f.Type
	}
	if f.TypeUppercase != "" {
		return f.TypeUppercase
	}
	return f.TypeLowercase
}

// DTContexts represents the context section
type DTContexts struct {
	Contexts []DTContextDetail `xml:"context_details"`
}

// DTContextDetail represents a single context entry
type DTContextDetail struct {
	Number      int    `xml:"context_number"`
	Comment     string `xml:"context_comment"`
	Description string `xml:"context_description"`
	Postfix     string `xml:"context_postfix"`
}

// DTInitialActions represents the initial actions section
type DTInitialActions struct {
	Actions []DTInitialAction `xml:"initial_action"`
}

// DTInitialAction represents a single initial action
type DTInitialAction struct {
	Number      int    `xml:"action_number"`
	Comment     string `xml:"action_comment"`
	Description string `xml:"action_description"`
	Postfix     string `xml:"action_postfix"`
}

// DTConditions represents the conditions section
type DTConditions struct {
	Conditions []DTConditionDetail `xml:"condition_details"`
}

// DTConditionDetail represents a single condition
type DTConditionDetail struct {
	Number      int              `xml:"condition_number"`
	Comment     string           `xml:"condition_comment"`
	Requirement string           `xml:"condition_requirement"`
	Description string           `xml:"condition_description"`
	Postfix     string           `xml:"condition_postfix"`
	Columns     []DTConditionCol `xml:"condition_column"`
}

// DTConditionCol represents a condition column entry
type DTConditionCol struct {
	ColumnNumber int    `xml:"column_number,attr"`
	ColumnValue  string `xml:"column_value,attr"`
}

// DTActions represents the actions section
type DTActions struct {
	Actions []DTActionDetail `xml:"action_details"`
}

// DTActionDetail represents a single action
type DTActionDetail struct {
	Number      int           `xml:"action_number"`
	Comment     string        `xml:"action_comment"`
	Requirement string        `xml:"initial_action_requirement"`
	Description string        `xml:"action_description"`
	Postfix     string        `xml:"action_postfix"`
	Columns     []DTActionCol `xml:"action_column"`
}

// DTActionCol represents an action column entry
type DTActionCol struct {
	ColumnNumber int    `xml:"column_number,attr"`
	ColumnValue  string `xml:"column_value,attr"`
}

// DTPolicyStatements represents policy statements
type DTPolicyStatements struct {
	Statements []DTPolicyStatement `xml:"policy_statement"`
}

// DTPolicyStatement represents a single policy statement
type DTPolicyStatement struct {
	Column      int    `xml:"column,attr"`
	Description string `xml:"policy_description"`
	Postfix     string `xml:"policy_statement_postfix"`
}

// Load loads decision tables from an io.Reader.
// The input size is limited by MaxXMLSize (default 10 MB) to prevent memory exhaustion.
func (l *DTLoader) Load(r io.Reader) error {
	// Apply size limit if configured
	if MaxXMLSize > 0 {
		r = io.LimitReader(r, MaxXMLSize+1) // +1 to detect overflow
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read decision tables: %w", err)
	}

	// Check if we hit the size limit
	if MaxXMLSize > 0 && int64(len(data)) > MaxXMLSize {
		return fmt.Errorf("decision tables XML exceeds maximum size limit of %d bytes", MaxXMLSize)
	}

	var dtFile DTFile
	if err := xml.Unmarshal(data, &dtFile); err != nil {
		return fmt.Errorf("failed to parse decision tables XML: %w", err)
	}

	for _, table := range dtFile.Tables {
		if err := l.processTable(&table); err != nil {
			l.errors = append(l.errors, err)
		}
	}

	if len(l.errors) > 0 {
		for i, err := range l.errors {
			log.Printf("DT Load Error %d: %v", i+1, err)
		}
		return fmt.Errorf("decision table loading completed with %d errors", len(l.errors))
	}
	return nil
}

// processTable processes a single decision table.
func (l *DTLoader) processTable(table *DTTable) error {
	name := dtrules.GetRName(strings.TrimSpace(table.TableName))
	if name == nil {
		return fmt.Errorf("invalid decision table name syntax: %s", table.TableName)
	}

	// Create the decision table using the builder
	builder := decisiontable.NewBuilder(name.StringValue(), l.session)

	// Set table type
	builder.SetTypeFromString(table.AttributeFields.GetType())

	// Set metadata fields
	builder.SetField("TABLE_NUMBER", table.AttributeFields.TableNumber)
	builder.SetField("COMMENTS", table.AttributeFields.Comments)
	builder.SetFilename(table.AttributeFields.FileName)

	// Process contexts - use postfix if available
	contexts := make([]string, len(table.Contexts.Contexts))
	contextsPostfix := make([]string, len(table.Contexts.Contexts))
	contextsComment := make([]string, len(table.Contexts.Contexts))
	for i, ctx := range table.Contexts.Contexts {
		contexts[i] = ctx.Description
		contextsPostfix[i] = strings.TrimSpace(ctx.Postfix)
		contextsComment[i] = ctx.Comment
	}
	builder.SetContexts(contexts)

	// Process initial actions
	initialActions := make([]string, len(table.InitialActions.Actions))
	initialActionsPostfix := make([]string, len(table.InitialActions.Actions))
	for i, action := range table.InitialActions.Actions {
		initialActions[i] = action.Description
		initialActionsPostfix[i] = strings.TrimSpace(action.Postfix)
	}
	builder.SetInitialActions(initialActions)

	// Determine the number of columns
	maxCol := l.getMaxColumn(table)
	builder.SetMaxCol(maxCol)

	// Process conditions
	numConditions := len(table.Conditions.Conditions)
	conditions := make([]string, numConditions)
	conditionsPostfix := make([]string, numConditions)
	conditionsComment := make([]string, numConditions)
	conditionTable := make([][]string, numConditions)

	for i, cond := range table.Conditions.Conditions {
		conditions[i] = cond.Description
		conditionsPostfix[i] = strings.TrimSpace(cond.Postfix)
		conditionsComment[i] = cond.Comment

		// Initialize row with "-" (don't care)
		conditionTable[i] = make([]string, maxCol)
		for j := range conditionTable[i] {
			conditionTable[i][j] = "-"
		}

		// Fill in specified column values
		for _, col := range cond.Columns {
			if col.ColumnNumber > 0 && col.ColumnNumber <= maxCol {
				conditionTable[i][col.ColumnNumber-1] = col.ColumnValue
			}
		}
	}
	builder.SetConditions(conditions)
	builder.SetConditionsComment(conditionsComment)
	builder.SetConditionTable(conditionTable)

	// Process actions
	numActions := len(table.Actions.Actions)
	actions := make([]string, numActions)
	actionsPostfix := make([]string, numActions)
	actionsComment := make([]string, numActions)
	actionTable := make([][]string, numActions)

	for i, action := range table.Actions.Actions {
		actions[i] = action.Description
		actionsPostfix[i] = strings.TrimSpace(action.Postfix)
		actionsComment[i] = action.Comment

		// Initialize row with empty (no action)
		actionTable[i] = make([]string, maxCol)
		for j := range actionTable[i] {
			actionTable[i][j] = ""
		}

		// Fill in specified column values
		for _, col := range action.Columns {
			if col.ColumnNumber > 0 && col.ColumnNumber <= maxCol {
				actionTable[i][col.ColumnNumber-1] = col.ColumnValue
			}
		}
	}
	builder.SetActions(actions)
	builder.SetActionsComment(actionsComment)
	builder.SetActionTable(actionTable)

	// Process policy statements
	policyStatements := make([]string, maxCol+1)
	policyPostfix := make([]string, maxCol+1)
	for _, ps := range table.PolicyStatements.Statements {
		if ps.Column >= 0 && ps.Column <= maxCol {
			policyStatements[ps.Column] = ps.Description
			policyPostfix[ps.Column] = strings.TrimSpace(ps.Postfix)
		}
	}
	builder.SetPolicyStatements(policyStatements)

	// Compile conditions using postfix notation (already compiled in XML)
	rconditions, err := l.compilePostfixExpressions(conditionsPostfix)
	if err != nil {
		return fmt.Errorf("failed to compile conditions for %s: %w", name.StringValue(), err)
	}
	builder.SetRConditions(rconditions)

	// Compile actions using postfix notation
	ractions, err := l.compilePostfixExpressions(actionsPostfix)
	if err != nil {
		return fmt.Errorf("failed to compile actions for %s: %w", name.StringValue(), err)
	}
	builder.SetRActions(ractions)

	// Compile contexts using postfix notation
	if len(contextsPostfix) > 0 {
		contextCode, err := l.compileContextsPostfix(name.StringValue(), contextsPostfix)
		if err != nil {
			return fmt.Errorf("failed to compile contexts for %s: %w", name.StringValue(), err)
		}
		builder.SetRContext(contextCode)
	}

	// Compile initial actions using postfix notation
	if len(initialActionsPostfix) > 0 {
		rinitialActions, err := l.compilePostfixExpressions(initialActionsPostfix)
		if err != nil {
			return fmt.Errorf("failed to compile initial actions for %s: %w", name.StringValue(), err)
		}
		builder.SetRInitialActions(rinitialActions)
	}

	// Build the decision table (constructs the decision tree)
	state := l.session.GetState()
	dt, err := builder.Build(state)
	if err != nil {
		return fmt.Errorf("failed to build decision table %s: %w", name.StringValue(), err)
	}

	// Register the decision table with the factory
	return l.factory.AddDecisionTable(name, dt)
}

// getMaxColumn determines the maximum column number in the table.
func (l *DTLoader) getMaxColumn(table *DTTable) int {
	maxCol := 0

	for _, cond := range table.Conditions.Conditions {
		for _, col := range cond.Columns {
			if col.ColumnNumber > maxCol {
				maxCol = col.ColumnNumber
			}
		}
	}

	for _, action := range table.Actions.Actions {
		for _, col := range action.Columns {
			if col.ColumnNumber > maxCol {
				maxCol = col.ColumnNumber
			}
		}
	}

	if maxCol == 0 {
		maxCol = 1
	}

	return maxCol
}

// compilePostfixExpressions compiles a list of postfix expressions.
func (l *DTLoader) compilePostfixExpressions(expressions []string) ([]dtrules.Object, error) {
	result := make([]dtrules.Object, len(expressions))

	for i, expr := range expressions {
		if expr == "" {
			// Empty expression - create no-op
			compiled, err := l.compiler.Compile("")
			if err != nil {
				return nil, fmt.Errorf("expression %d: %w", i+1, err)
			}
			result[i] = compiled
		} else {
			compiled, err := l.compiler.CompilePostfix(expr)
			if err != nil {
				return nil, fmt.Errorf("expression %d ('%s'): %w", i+1, expr, err)
			}
			result[i] = compiled
		}
	}

	return result, nil
}

// compileContextsPostfix compiles context postfix expressions into a single code block.
// Following the Java pattern, the table call is wrapped inside the context:
// Start with "/" + tableName + " executetable "
// Then for each context (in reverse order): contextsrc = "{ " + contextsrc + " } " + contextsPostfix[i]
func (l *DTLoader) compileContextsPostfix(tableName string, contexts []string) (dtrules.Object, error) {
	if len(contexts) == 0 {
		return nil, nil
	}

	// Check if any contexts have meaningful content
	// Skip contexts that are just "execute" (placeholder for "no longer supported")
	hasContent := false
	for _, ctx := range contexts {
		ctx = strings.TrimSpace(ctx)
		if ctx != "" && ctx != "execute" {
			hasContent = true
			break
		}
	}
	if !hasContent {
		return nil, nil
	}

	// Start with the table call (like Java: "/" + getName().stringValue() + " executeTable ")
	contextsrc := "/" + tableName + " executetable "

	// Wrap with each context postfix (in reverse order, like Java)
	for i := len(contexts) - 1; i >= 0; i-- {
		ctx := strings.TrimSpace(contexts[i])
		// Skip empty contexts and placeholder "execute" (used for "no longer supported" entries)
		if ctx != "" && ctx != "execute" {
			contextsrc = "{ " + contextsrc + " } " + ctx
		}
	}

	return l.compiler.CompilePostfix(contextsrc)
}

// GetErrors returns any errors encountered during loading.
func (l *DTLoader) GetErrors() []error {
	return l.errors
}
