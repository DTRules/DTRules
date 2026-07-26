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
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/compiler"
	"github.com/DTRules/DTRules/pkg/dtrules/decisiontable"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
)

// DTLoader loads Decision Table files (XML or JSON).
//
// The loader is strictly a *consumer* of pre-compiled postfix. EL DSL is
// authoritative source text and is the authoring contract; postfix is the
// compiled artifact and is what the loader executes. Compilation lives in
// `dtrules build` and `dtrules compile` — never in the load path.
//
// When a table element has non-empty, non-comment EL DSL but its
// `<*_postfix>` is empty or comment-only, the loader returns an error
// directing the operator to run `dtrules build` (or `dtrules compile`)
// before embedding the XML. This refusal prevents silent stale-build
// drift and removes the EL compiler dependency from every consumer's
// runtime binary.
//
// Build-time tooling (the export step of the XML→Excel bootstrap, which
// loads partially-built XML so it can write Excel) sets
// `Tolerant=true` to disable the postfix-presence check. Runtime
// consumers should never use tolerant mode — it accepts XML that will
// crash at execution time.
type DTLoader struct {
	session  dtrules.Session
	factory  *entity.Factory
	compiler *compiler.Compiler
	errors   []error

	// Tolerant disables the "DSL with no compiled postfix" load error.
	// Used by build-time tooling that consumes XML mid-compile (export
	// → re-import pipeline). Default false; runtime consumers stay
	// strict.
	Tolerant bool
}

// NewDTLoader creates a new Decision Table loader in the default strict mode.
func NewDTLoader(session dtrules.Session, factory *entity.Factory) *DTLoader {
	return &DTLoader{
		session:  session,
		factory:  factory,
		compiler: compiler.NewCompiler(session, factory),
		errors:   make([]error, 0),
	}
}

// SetSymbols is retained as a no-op for source compatibility with callers
// (notably `session.RuleSet.LoadDecisionTables`). The loader no longer
// compiles EL — DSL must be pre-compiled to postfix by the build pipeline
// — so a symbol table here would have nothing to do. Symbol resolution
// belongs to `dtrules build` / `dtrules compile`, which already build
// the symbol map from the EDD they load alongside the tables.
func (l *DTLoader) SetSymbols(_ map[string]string) {}

// SetCollectionResolver is retained as a no-op for the same reason as
// SetSymbols. The `for all <type> entities` DSL form is resolved at
// authoring time by the EL compiler; at load time we only see the
// already-resolved postfix.
//
// The parameter is typed as `any` so the loader no longer needs to
// import `compiler/el`. Callers that were passing an
// `el.CollectionResolver` continue to compile unchanged.
func (l *DTLoader) SetCollectionResolver(_ any) {}

// Structures matching the DTRules decision table format (XML and JSON)

// DTFile represents the root decision_tables element
type DTFile struct {
	XMLName xml.Name  `xml:"decision_tables" json:"-"`
	Tables  []DTTable `xml:"decision_table" json:"decision_tables"`
}

// DTSource records the Excel workbook and sheet from which a decision table was imported.
type DTSource struct {
	RelativePath string `xml:"relative_path"`
	FileName     string `xml:"file_name"`
	SheetNumber  int    `xml:"sheet_number"`
}

// DTTable represents a single decision table
type DTTable struct {
	// NameAttr captures the 'name' attribute on <decision_table name="...">
	// Used by the EL XML format (e.g., CorporateTax, staking)
	NameAttr         string             `xml:"name,attr" json:"-"`
	// NumberAttr captures the 'number' attribute on <decision_table number="...">
	NumberAttr       string             `xml:"number,attr" json:"-"`
	Source           *DTSource          `xml:"source,omitempty" json:"-"`
	TableName        string             `xml:"table_name" json:"table_name"`
	XlsFile          string             `xml:"xls_file" json:"xls_file,omitempty"`
	AttributeFields  DTAttributeFields  `xml:"attribute_fields" json:"attribute_fields"`
	Contexts         DTContexts         `xml:"contexts" json:"contexts"`
	InitialActions   DTInitialActions   `xml:"initial_actions" json:"initial_actions"`
	Conditions       DTConditions       `xml:"conditions" json:"conditions"`
	Actions          DTActions          `xml:"actions" json:"actions"`
	PolicyStatements DTPolicyStatements `xml:"policy_statements" json:"policy_statements"`
}

// GetTableName returns the table name, checking both the attribute and element forms.
// EL format uses name attribute: <decision_table name="My_Table">
// Traditional format uses element: <table_name>My_Table</table_name>
func (t *DTTable) GetTableName() string {
	if t.NameAttr != "" {
		return t.NameAttr
	}
	return t.TableName
}

// GetTableNumber returns the table number, checking both the attribute and element forms.
func (t *DTTable) GetTableNumber() string {
	if t.NumberAttr != "" {
		return t.NumberAttr
	}
	return t.AttributeFields.TableNumber
}

// DTAttributeFields represents metadata about the table
// Note: Type can be <Type>, <TYPE>, or <type> depending on the XML source
type DTAttributeFields struct {
	Type          string `xml:"Type" json:"type"`
	TypeUppercase string `xml:"TYPE" json:"-"`
	TypeLowercase string `xml:"type" json:"-"`
	Comments      string `xml:"COMMENTS" json:"comments,omitempty"`
	CommentsLower string `xml:"comments" json:"-"`
	FileName      string `xml:"File_Name" json:"file_name,omitempty"`
	TableNumber   string `xml:"TABLE_NUMBER" json:"table_number,omitempty"`
	FilePath      string `xml:"FILE_PATH" json:"file_path,omitempty"`
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
	Contexts []DTContextDetail `xml:"context_details" json:"context_details"`
	// Note: <context_entity> element is intentionally not parsed here.
	// The context_entity pattern (forall over entity arrays) doesn't handle
	// empty arrays well. Tables using context_entity should be converted to
	// use initial_actions with explicit forall loops that handle empty cases.
}

// DTContextDetail represents a single context entry
type DTContextDetail struct {
	Number      int    `xml:"context_number" json:"context_number"`
	Comment     string `xml:"context_comment" json:"context_comment,omitempty"`
	DSL         string `xml:"context_dsl" json:"context_dsl,omitempty"`
	Description string `xml:"context_description" json:"context_description"`
	Postfix     string `xml:"context_postfix" json:"context_postfix"`
}

// GetDSL returns the DSL expression, preferring DSL over Description for backward compatibility.
func (c *DTContextDetail) GetDSL() string {
	if c.DSL != "" {
		return c.DSL
	}
	return c.Description
}

// DTInitialActions represents the initial actions section
type DTInitialActions struct {
	Actions []DTInitialAction `xml:"initial_action" json:"initial_action"`
}

// DTInitialAction represents a single initial action. Two DSL/postfix tag
// shapes are accepted to cover both the original Excel-import convention
// (action_dsl/action_postfix on <initial_action>) and the EL-aware
// authoring SDK form (initial_action_dsl/initial_action_postfix).
type DTInitialAction struct {
	Number             int    `xml:"action_number" json:"action_number"`
	Comment            string `xml:"action_comment" json:"action_comment,omitempty"`
	DSL                string `xml:"initial_action_dsl" json:"initial_action_dsl,omitempty"`
	ActionDSL          string `xml:"action_dsl" json:"action_dsl,omitempty"`
	Description        string `xml:"action_description" json:"action_description"`
	Postfix            string `xml:"action_postfix" json:"action_postfix"`
	InitialActPostfix  string `xml:"initial_action_postfix" json:"initial_action_postfix,omitempty"`
}

// GetDSL returns the first non-empty DSL field, preferring initial_action_dsl
// over action_dsl over action_description for backward compatibility.
func (a *DTInitialAction) GetDSL() string {
	if a.DSL != "" {
		return a.DSL
	}
	if a.ActionDSL != "" {
		return a.ActionDSL
	}
	return a.Description
}

// GetPostfix returns the postfix expression from whichever XML tag form is
// populated.
func (a *DTInitialAction) GetPostfix() string {
	if a.Postfix != "" {
		return a.Postfix
	}
	return a.InitialActPostfix
}

// DTConditions represents the conditions section
type DTConditions struct {
	Conditions []DTConditionDetail `xml:"condition_details" json:"condition_details"`
}

// DTConditionDetail represents a single condition
type DTConditionDetail struct {
	Number      int              `xml:"condition_number" json:"condition_number"`
	Comment     string           `xml:"condition_comment" json:"condition_comment,omitempty"`
	Requirement string           `xml:"condition_requirement" json:"condition_requirement,omitempty"`
	DSL         string           `xml:"condition_dsl" json:"condition_dsl,omitempty"`
	Description string           `xml:"condition_description" json:"condition_description"`
	Postfix     string           `xml:"condition_postfix" json:"condition_postfix"`
	Columns     []DTConditionCol `xml:"condition_column" json:"condition_columns"`
}

// GetDSL returns the DSL expression, preferring DSL over Description for backward compatibility.
func (c *DTConditionDetail) GetDSL() string {
	if c.DSL != "" {
		return c.DSL
	}
	return c.Description
}

// DTConditionCol represents a condition column entry
type DTConditionCol struct {
	ColumnNumber int    `xml:"column_number,attr" json:"column_number"`
	ColumnValue  string `xml:"column_value,attr" json:"column_value"`
}

// DTActions represents the actions section
type DTActions struct {
	Actions []DTActionDetail `xml:"action_details" json:"action_details"`
}

// DTActionDetail represents a single action
type DTActionDetail struct {
	Number      int           `xml:"action_number" json:"action_number"`
	Comment     string        `xml:"action_comment" json:"action_comment,omitempty"`
	Requirement string        `xml:"initial_action_requirement" json:"action_requirement,omitempty"`
	DSL         string        `xml:"action_dsl" json:"action_dsl,omitempty"`
	Description string        `xml:"action_description" json:"action_description"`
	Postfix     string        `xml:"action_postfix" json:"action_postfix"`
	Columns     []DTActionCol `xml:"action_column" json:"action_columns"`
}

// GetDSL returns the DSL expression, preferring DSL over Description for backward compatibility.
func (a *DTActionDetail) GetDSL() string {
	if a.DSL != "" {
		return a.DSL
	}
	return a.Description
}

// DTActionCol represents an action column entry
type DTActionCol struct {
	ColumnNumber int    `xml:"column_number,attr" json:"column_number"`
	ColumnValue  string `xml:"column_value,attr" json:"column_value"`
}

// DTPolicyStatements represents policy statements
type DTPolicyStatements struct {
	Statements []DTPolicyStatement `xml:"policy_statement" json:"policy_statements"`
}

// DTPolicyStatement represents a single policy statement
type DTPolicyStatement struct {
	Column      int    `xml:"column,attr" json:"column"`
	Description string `xml:"policy_description" json:"policy_description"`
	Postfix     string `xml:"policy_statement_postfix" json:"policy_statement_postfix"`
}

// Load loads decision tables from an io.Reader. The input may be XML or JSON;
// the format is detected automatically from the content.
// The input size is limited by MaxXMLSize (default 10 MB) to prevent memory exhaustion.
func (l *DTLoader) Load(r io.Reader) error {
	// Detect format
	format, r, err := DetectFormat(r)
	if err != nil {
		return fmt.Errorf("failed to read decision tables: %w", err)
	}

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
		return fmt.Errorf("decision tables input exceeds maximum size limit of %d bytes", MaxXMLSize)
	}

	var dtFile DTFile
	switch format {
	case FormatJSON:
		if err := json.Unmarshal(data, &dtFile); err != nil {
			return fmt.Errorf("failed to parse decision tables JSON: %w", err)
		}
	default:
		if err := xml.Unmarshal(data, &dtFile); err != nil {
			return fmt.Errorf("failed to parse decision tables XML: %w", err)
		}
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
		// Embed the first error's text in the returned aggregate so
		// callers and tests see the actionable detail (which table,
		// which row, "run `dtrules build`") instead of just a count.
		// Multi-error case keeps the count too.
		if len(l.errors) == 1 {
			return fmt.Errorf("decision table loading failed: %w", l.errors[0])
		}
		return fmt.Errorf("decision table loading failed with %d errors; first: %w", len(l.errors), l.errors[0])
	}
	return nil
}

// processTable processes a single decision table.
func (l *DTLoader) processTable(table *DTTable) error {
	// Use GetTableName() which checks both attribute and element forms
	tableName := strings.TrimSpace(table.GetTableName())

	// Check for legacy postfix (hand-coded without EL descriptions). The
	// flag is propagated to the built RDecisionTable below; the runtime
	// refuses to execute tables with this flag set.
	legacy := l.detectLegacyPostfix(table)
	if legacy {
		l.warnLegacyPostfix(tableName, table.XlsFile)
	}
	name := dtrules.GetRName(tableName)
	if name == nil {
		return fmt.Errorf("invalid decision table name syntax: %s", tableName)
	}

	// Create the decision table using the builder
	builder := decisiontable.NewBuilder(name.StringValue(), l.session)

	// Set table type
	builder.SetTypeFromString(table.AttributeFields.GetType())

	// Set metadata fields - use GetTableNumber() for attribute/element fallback
	builder.SetField("TABLE_NUMBER", table.GetTableNumber())
	builder.SetField("COMMENTS", table.AttributeFields.Comments)
	builder.SetFilename(table.XlsFile)

	// Set FILE_PATH if present (with fallback to xls_file)
	if table.AttributeFields.FilePath != "" {
		builder.SetFilePath(table.AttributeFields.FilePath)
	}

	// Process contexts. The loader consumes pre-compiled postfix; EL DSL
	// is informational source text only. If DSL is present (non-comment)
	// but stored postfix is empty, that's a stale build — refuse to load
	// rather than silently no-op. Comment-only DSL is allowed with empty
	// postfix (the runtime treats it as a no-op context).
	contexts := make([]string, len(table.Contexts.Contexts))
	contextsPostfix := make([]string, len(table.Contexts.Contexts))
	contextsComment := make([]string, len(table.Contexts.Contexts))
	for i, ctx := range table.Contexts.Contexts {
		dsl := ctx.GetDSL()
		contexts[i] = dsl
		stored := strings.TrimSpace(ctx.Postfix)
		dslTrimmed := strings.TrimSpace(dsl)

		if !l.Tolerant && dslTrimmed != "" && !isCommentLine(dslTrimmed) && stored == "" {
			return fmt.Errorf("context %d ('%s') has DSL but no compiled postfix in table %s — run `dtrules build` or `dtrules compile` before loading",
				i+1, dsl, name.StringValue())
		}
		contextsPostfix[i] = stored
		contextsComment[i] = ctx.Comment
	}

	builder.SetContexts(contexts)

	// Process initial actions. Same strict policy as contexts: non-comment
	// DSL paired with an empty postfix is a stale build — refuse to load.
	initialActions := make([]string, len(table.InitialActions.Actions))
	initialActionsPostfix := make([]string, len(table.InitialActions.Actions))
	for i, action := range table.InitialActions.Actions {
		dsl := action.GetDSL()
		initialActions[i] = dsl
		postfix := strings.TrimSpace(action.GetPostfix())

		dslTrimmed := strings.TrimSpace(dsl)
		commentTrimmed := strings.TrimSpace(action.Comment)
		if !l.Tolerant && isEmptyOrCommentOnly(postfix) && dslTrimmed != "" && !isCommentLine(dslTrimmed) && dslTrimmed != commentTrimmed {
			return fmt.Errorf("initial action %d ('%s') has DSL but no compiled postfix in table %s — run `dtrules build` or `dtrules compile` before loading",
				i+1, dsl, name.StringValue())
		}
		initialActionsPostfix[i] = postfix
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
		dsl := cond.GetDSL()
		conditions[i] = dsl
		postfix := strings.TrimSpace(cond.Postfix)

		dslTrimmed := strings.TrimSpace(dsl)
		commentTrimmed := strings.TrimSpace(cond.Comment)
		if !l.Tolerant && isEmptyOrCommentOnly(postfix) && dslTrimmed != "" && !isCommentLine(dslTrimmed) && dslTrimmed != commentTrimmed {
			return fmt.Errorf("condition %d ('%s') has DSL but no compiled postfix in table %s — run `dtrules build` or `dtrules compile` before loading",
				i+1, dsl, name.StringValue())
		}
		// Comment-only DSL with no stored postfix → "true always" so the
		// table builder still has a parseable condition expression to
		// emit. This preserves the editor convenience of leaving
		// documentation rows without postfix.
		if isEmptyOrCommentOnly(postfix) && isCommentLine(dslTrimmed) && postfix == "" {
			postfix = "true always"
		}
		conditionsPostfix[i] = postfix
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
		dsl := action.GetDSL()
		actions[i] = dsl
		postfix := strings.TrimSpace(action.Postfix)

		dslTrimmed := strings.TrimSpace(dsl)
		commentTrimmed := strings.TrimSpace(action.Comment)
		if !l.Tolerant && isEmptyOrCommentOnly(postfix) && dslTrimmed != "" && !isCommentLine(dslTrimmed) && dslTrimmed != commentTrimmed {
			return fmt.Errorf("action %d ('%s') has DSL but no compiled postfix in table %s — run `dtrules build` or `dtrules compile` before loading",
				i+1, dsl, name.StringValue())
		}
		actionsPostfix[i] = postfix
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

	// Mark the built table if it's legacy-postfix-only so Execute /
	// ExecuteTable will refuse to run it. The runtime treats this as a
	// hard block; the operator must author EL DSL for at least one row to
	// unblock.
	if legacy {
		reason := "loaded from " + table.XlsFile
		if first := decisiontable.FirstHandCodedElement(collectPostfixEntries(table)); first != "" {
			reason = reason + "; " + first + " has hand-coded postfix without EL DSL"
		}
		dt.SetHandCodedPostfix(true, reason)
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
			// Transform legacy if...endif syntax to { body } condition if syntax
			expr = transformIfEndif(expr)
			compiled, err := l.compiler.CompilePostfix(expr)
			if err != nil {
				return nil, fmt.Errorf("expression %d ('%s'): %w", i+1, expr, err)
			}
			result[i] = compiled
		}
	}

	return result, nil
}

// transformIfEndif converts legacy if syntax to Go-style syntax.
// It handles:
// - "condition if body endif" -> "{ body } condition if"
// - "condition if truebody else falsebody endif" -> "{ truebody } { falsebody } condition ifelse"
// - "condition if body then" -> "{ body } condition if" (then is alias for endif)
// - "condition if truebody else falsebody then" -> "{ truebody } { falsebody } condition ifelse"
// - "condition iftrue body then" -> "{ body } condition if"
// - "condition iftrue truebody else falsebody then" -> "{ truebody } { falsebody } condition ifelse"
func transformIfEndif(postfix string) string {
	// Quick check - if no "endif" or "then", nothing to transform
	if !strings.Contains(postfix, "endif") && !strings.Contains(postfix, "then") {
		return postfix
	}

	tokens := tokenizePostfix(postfix)

	// Transform iftrue/then patterns first (they use "then" as terminator)
	tokens = transformIftrueTokens(tokens)

	// Normalize "then" to "endif" for if...then patterns (when paired with "if", not "iftrue")
	tokens = normalizeIfThen(tokens)

	// Transform if/endif patterns
	tokens = transformIfEndifTokens(tokens)

	return strings.Join(tokens, " ")
}

// normalizeIfThen converts "then" tokens to "endif" when they are paired with "if" (not "iftrue").
// This handles the legacy "if...else...then" pattern.
func normalizeIfThen(tokens []string) []string {
	// Find if...then pairs and convert then to endif
	for i := 0; i < len(tokens); i++ {
		if tokens[i] == "if" {
			// Find matching then or endif
			depth := 1
			for j := i + 1; j < len(tokens); j++ {
				switch tokens[j] {
				case "if", "iftrue":
					depth++
				case "endif":
					depth--
				case "then":
					depth--
					if depth == 0 {
						// Convert then to endif
						tokens[j] = "endif"
					}
				}
				if depth == 0 {
					break
				}
			}
		}
	}
	return tokens
}

// tokenizePostfix splits postfix into tokens, preserving string literals.
func tokenizePostfix(postfix string) []string {
	var tokens []string
	var current strings.Builder
	inString := false
	stringChar := byte(0)

	for i := 0; i < len(postfix); i++ {
		ch := postfix[i]

		if inString {
			current.WriteByte(ch)
			if ch == stringChar {
				inString = false
				tokens = append(tokens, current.String())
				current.Reset()
			}
		} else if ch == '"' || ch == '\'' {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			inString = true
			stringChar = ch
			current.WriteByte(ch)
		} else if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

// transformIfEndifTokens transforms if/endif patterns in token slice.
func transformIfEndifTokens(tokens []string) []string {
	// Find innermost if...endif pairs and transform them from inside out
	for {
		ifIdx, elseIdx, endifIdx := findInnermostIfEndif(tokens)
		if ifIdx == -1 {
			break
		}

		if elseIdx != -1 {
			// if...else...endif -> { truebody } { falsebody } condition ifelse
			condition := tokens[:ifIdx]
			truebody := tokens[ifIdx+1 : elseIdx]
			falsebody := tokens[elseIdx+1 : endifIdx]
			rest := tokens[endifIdx+1:]

			// Build new token list
			var newTokens []string
			newTokens = append(newTokens, "{")
			newTokens = append(newTokens, truebody...)
			newTokens = append(newTokens, "}")
			newTokens = append(newTokens, "{")
			newTokens = append(newTokens, falsebody...)
			newTokens = append(newTokens, "}")
			newTokens = append(newTokens, condition...)
			newTokens = append(newTokens, "ifelse")
			newTokens = append(newTokens, rest...)
			tokens = newTokens
		} else {
			// if...endif -> { body } condition if
			condition := tokens[:ifIdx]
			body := tokens[ifIdx+1 : endifIdx]
			rest := tokens[endifIdx+1:]

			// Build new token list
			var newTokens []string
			newTokens = append(newTokens, "{")
			newTokens = append(newTokens, body...)
			newTokens = append(newTokens, "}")
			newTokens = append(newTokens, condition...)
			newTokens = append(newTokens, "if")
			newTokens = append(newTokens, rest...)
			tokens = newTokens
		}
	}
	return tokens
}

// findInnermostIfEndif finds the innermost if...endif pair.
// Returns ifIdx, elseIdx (-1 if no else), endifIdx.
// Returns -1, -1, -1 if no if...endif found.
func findInnermostIfEndif(tokens []string) (int, int, int) {
	// Find the last "if" that has a matching "endif"
	for i := len(tokens) - 1; i >= 0; i-- {
		if tokens[i] == "if" {
			// Find matching endif (and optional else)
			depth := 1
			elseIdx := -1
			for j := i + 1; j < len(tokens); j++ {
				switch tokens[j] {
				case "if":
					depth++
				case "endif":
					depth--
					if depth == 0 {
						return i, elseIdx, j
					}
				case "else":
					if depth == 1 {
						elseIdx = j
					}
				}
			}
		}
	}
	return -1, -1, -1
}

// transformIftrueTokens transforms iftrue/then patterns in token slice.
// Converts:
// - "condition iftrue body then" -> "{ body } condition if"
// - "condition iftrue truebody else falsebody then" -> "{ truebody } { falsebody } condition ifelse"
func transformIftrueTokens(tokens []string) []string {
	// Find innermost iftrue...then pairs and transform them from inside out
	for {
		iftrueIdx, elseIdx, thenIdx := findInnermostIftrue(tokens)
		if iftrueIdx == -1 {
			break
		}

		if elseIdx != -1 {
			// iftrue...else...then -> { truebody } { falsebody } condition ifelse
			condition := tokens[:iftrueIdx]
			truebody := tokens[iftrueIdx+1 : elseIdx]
			falsebody := tokens[elseIdx+1 : thenIdx]
			rest := tokens[thenIdx+1:]

			// Build new token list
			var newTokens []string
			newTokens = append(newTokens, "{")
			newTokens = append(newTokens, truebody...)
			newTokens = append(newTokens, "}")
			newTokens = append(newTokens, "{")
			newTokens = append(newTokens, falsebody...)
			newTokens = append(newTokens, "}")
			newTokens = append(newTokens, condition...)
			newTokens = append(newTokens, "ifelse")
			newTokens = append(newTokens, rest...)
			tokens = newTokens
		} else {
			// iftrue...then -> { body } condition if
			condition := tokens[:iftrueIdx]
			body := tokens[iftrueIdx+1 : thenIdx]
			rest := tokens[thenIdx+1:]

			// Build new token list
			var newTokens []string
			newTokens = append(newTokens, "{")
			newTokens = append(newTokens, body...)
			newTokens = append(newTokens, "}")
			newTokens = append(newTokens, condition...)
			newTokens = append(newTokens, "if")
			newTokens = append(newTokens, rest...)
			tokens = newTokens
		}
	}
	return tokens
}

// findInnermostIftrue finds the innermost iftrue...then pair.
// Returns iftrueIdx, elseIdx (-1 if no else), thenIdx.
// Returns -1, -1, -1 if no iftrue...then found.
func findInnermostIftrue(tokens []string) (int, int, int) {
	// Find the last "iftrue" that has a matching "then"
	for i := len(tokens) - 1; i >= 0; i-- {
		if tokens[i] == "iftrue" {
			// Find matching then (and optional else)
			depth := 1
			elseIdx := -1
			for j := i + 1; j < len(tokens); j++ {
				switch tokens[j] {
				case "iftrue":
					depth++
				case "then":
					depth--
					if depth == 0 {
						return i, elseIdx, j
					}
				case "else":
					if depth == 1 {
						elseIdx = j
					}
				}
			}
		}
	}
	return -1, -1, -1
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

// collectPostfixEntries builds the analysis-package PostfixEntry slice for
// every element of a DTTable: contexts, initial actions, conditions, and
// actions. Used to feed the canonical detector in
// decisiontable.HasAnyHandCodedPostfix / CheckHandCodedPostfix.
func collectPostfixEntries(table *DTTable) []decisiontable.PostfixEntry {
	var entries []decisiontable.PostfixEntry
	for i, ctx := range table.Contexts.Contexts {
		entries = append(entries, decisiontable.PostfixEntry{
			Kind: "context", Number: i + 1,
			DSL:     ctx.GetDSL(),
			Postfix: ctx.Postfix,
		})
	}
	for i, action := range table.InitialActions.Actions {
		entries = append(entries, decisiontable.PostfixEntry{
			Kind: "initial_action", Number: i + 1,
			DSL:     action.GetDSL(),
			Postfix: action.GetPostfix(),
		})
	}
	for _, cond := range table.Conditions.Conditions {
		entries = append(entries, decisiontable.PostfixEntry{
			Kind: "condition", Number: cond.Number,
			DSL:     cond.GetDSL(),
			Postfix: cond.Postfix,
		})
	}
	for _, act := range table.Actions.Actions {
		entries = append(entries, decisiontable.PostfixEntry{
			Kind: "action", Number: act.Number,
			DSL:     act.GetDSL(),
			Postfix: act.Postfix,
		})
	}
	return entries
}

// detectLegacyPostfix reports whether a table has hand-coded postfix without
// any EL DSL — i.e. it would refuse to execute under the runtime's
// hand-coded-postfix gate. Delegates to decisiontable.HasAnyHandCodedPostfix
// so the loader and analyzer agree on the rule.
func (l *DTLoader) detectLegacyPostfix(table *DTTable) bool {
	return decisiontable.HasAnyHandCodedPostfix(collectPostfixEntries(table))
}

// warnLegacyPostfix logs a warning about a table with hand-coded postfix.
func (l *DTLoader) warnLegacyPostfix(tableName, filePath string) {
	log.Printf("WARNING: %s in %s has hand-coded postfix without EL DSL.", tableName, filePath)
	log.Printf("         All postfix should come from EL compilation. Use 'dtrules sync import' to compile EL.")
}

// isEmptyOrCommentOnly checks if a postfix string is effectively empty.
// It returns true if the string is empty, whitespace only, or contains only comments.
// Comments start with "//" or "#" and extend to the end of the line.
func isEmptyOrCommentOnly(postfix string) bool {
	lines := strings.Split(postfix, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Found a non-empty, non-comment line
		return false
	}
	return true
}

// isCommentLine checks if a string is a comment line (starts with // or #).
func isCommentLine(s string) bool {
	trimmed := strings.TrimSpace(s)
	// Block comments (/* ... */) count only when the WHOLE line is the
	// comment — a trailing block comment after real DSL is still DSL.
	if strings.HasPrefix(trimmed, "/*") && strings.HasSuffix(trimmed, "*/") {
		return true
	}
	return strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#")
}
