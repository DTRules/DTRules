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

package decisiontable

import (
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// Builder provides a fluent interface for constructing decision tables.
type Builder struct {
	dt *RDecisionTable
}

// NewBuilder creates a new decision table builder.
// Returns nil if name has invalid syntax.
func NewBuilder(name string, session dtrules.Session) *Builder {
	rname := dtrules.GetRName(name)
	if rname == nil {
		return nil
	}
	dt := NewRDecisionTable(rname, session)
	dt.SetAuthoredName(name)
	return &Builder{dt: dt}
}

// SetType sets the table type (BALANCED, FIRST, or ALL).
func (b *Builder) SetType(t TableType) *Builder {
	b.dt.SetTableType(t)
	return b
}

// SetTypeFromString sets the table type from a string.
func (b *Builder) SetTypeFromString(t string) *Builder {
	switch strings.ToUpper(strings.TrimSpace(t)) {
	case "BALANCED":
		b.dt.SetTableType(BALANCED)
	case "FIRST":
		b.dt.SetTableType(FIRST)
	case "ALL":
		b.dt.SetTableType(ALL)
	}
	return b
}

// SetConditions sets the condition expressions.
func (b *Builder) SetConditions(conditions []string) *Builder {
	b.dt.conditions = conditions
	b.dt.conditionsComment = make([]string, len(conditions))
	b.dt.conditionsPostfix = make([]string, len(conditions))
	return b
}

// SetConditionsPostfix sets the compiled postfix for conditions.
func (b *Builder) SetConditionsPostfix(postfix []string) *Builder {
	b.dt.conditionsPostfix = postfix
	return b
}

// SetConditionsComment sets the comments for conditions.
func (b *Builder) SetConditionsComment(comments []string) *Builder {
	b.dt.conditionsComment = comments
	return b
}

// SetActions sets the action expressions.
func (b *Builder) SetActions(actions []string) *Builder {
	b.dt.actions = actions
	b.dt.actionsComment = make([]string, len(actions))
	b.dt.actionsPostfix = make([]string, len(actions))
	return b
}

// SetActionsPostfix sets the compiled postfix for actions.
func (b *Builder) SetActionsPostfix(postfix []string) *Builder {
	b.dt.actionsPostfix = postfix
	return b
}

// SetActionsComment sets the comments for actions.
func (b *Builder) SetActionsComment(comments []string) *Builder {
	b.dt.actionsComment = comments
	return b
}

// SetConditionTable sets the condition table (Y/N/- values).
// The table is organized as [row][col].
func (b *Builder) SetConditionTable(table [][]string) *Builder {
	b.dt.conditionTable = table
	return b
}

// SetActionTable sets the action table (X values).
// The table is organized as [row][col].
func (b *Builder) SetActionTable(table [][]string) *Builder {
	b.dt.actionTable = table
	return b
}

// SetMaxCol sets the number of columns.
func (b *Builder) SetMaxCol(n int) *Builder {
	b.dt.maxCol = n
	return b
}

// SetRConditions sets the compiled condition objects.
func (b *Builder) SetRConditions(conditions []dtrules.Object) *Builder {
	b.dt.rconditions = conditions
	return b
}

// SetRActions sets the compiled action objects.
func (b *Builder) SetRActions(actions []dtrules.Object) *Builder {
	b.dt.ractions = actions
	return b
}

// SetContexts sets the context entity names.
func (b *Builder) SetContexts(contexts []string) *Builder {
	b.dt.contexts = contexts
	return b
}

// SetContextsComment sets the per-context comments, indexed alongside
// SetContexts. The Excel exporter writes these into the CONTEXTS comment
// column, so a loader that skips them erases them on the next build.
func (b *Builder) SetContextsComment(comments []string) *Builder {
	b.dt.contextsComment = comments
	return b
}

// SetContextsPostfix sets the compiled context postfix.
func (b *Builder) SetContextsPostfix(postfix []string) *Builder {
	b.dt.contextsPostfix = postfix
	return b
}

// SetRContext sets the compiled context code.
func (b *Builder) SetRContext(context dtrules.Object) *Builder {
	b.dt.rcontext = context
	return b
}

// SetInitialActions sets the initial action expressions.
func (b *Builder) SetInitialActions(actions []string) *Builder {
	b.dt.initialActions = actions
	b.dt.initialActionsComment = make([]string, len(actions))
	b.dt.initialActionsPostfix = make([]string, len(actions))
	return b
}

// SetInitialActionsPostfix sets the compiled initial action postfix.
func (b *Builder) SetInitialActionsPostfix(postfix []string) *Builder {
	b.dt.initialActionsPostfix = postfix
	return b
}

// SetRInitialActions sets the compiled initial action objects.
func (b *Builder) SetRInitialActions(actions []dtrules.Object) *Builder {
	b.dt.rinitialActions = actions
	return b
}

// SetPolicyStatements sets the policy statement strings.
func (b *Builder) SetPolicyStatements(statements []string) *Builder {
	b.dt.policyStatements = statements
	b.dt.policyStatementsPostfix = make([]string, len(statements))
	return b
}

// SetPolicyStatementsPostfix sets the compiled policy statement postfix.
func (b *Builder) SetPolicyStatementsPostfix(postfix []string) *Builder {
	b.dt.policyStatementsPostfix = postfix
	return b
}

// SetRPolicyStatements sets the compiled policy statement objects.
func (b *Builder) SetRPolicyStatements(statements []dtrules.Object) *Builder {
	b.dt.rpolicyStatements = statements
	return b
}

// SetField sets a metadata field.
func (b *Builder) SetField(name, value string) *Builder {
	b.dt.SetField(name, value)
	return b
}

// SetFilename sets the source filename.
func (b *Builder) SetFilename(filename string) *Builder {
	b.dt.filename = filename
	return b
}

// SetFilePath sets the canonical file path.
func (b *Builder) SetFilePath(path string) *Builder {
	b.dt.filePath = path
	return b
}

// SetOptimize sets whether to optimize the decision tree.
func (b *Builder) SetOptimize(optimize bool) *Builder {
	b.dt.optimize = optimize
	return b
}

// Build builds and returns the decision table.
// Returns an error if the table cannot be built.
func (b *Builder) Build(state dtrules.State) (*RDecisionTable, error) {
	if err := b.dt.Build(state); err != nil {
		return nil, err
	}
	return b.dt, nil
}

// Get returns the decision table without building.
// Use this when you need to set up the decision table manually.
func (b *Builder) Get() *RDecisionTable {
	return b.dt
}

// ParseTableType parses a table type string.
func ParseTableType(s string) TableType {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "FIRST":
		return FIRST
	case "ALL":
		return ALL
	default:
		return BALANCED
	}
}
