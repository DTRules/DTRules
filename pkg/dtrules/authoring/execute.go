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

package authoring

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/excel"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/operators"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// AttributeChange records a single attribute mutation caused by an action.
type AttributeChange struct {
	Entity    string
	Attribute string
	OldValue  string
	NewValue  string
}

// StepKind describes what a Step represents.
type StepKind string

const (
	StepCondition StepKind = "condition"
	StepAction    StepKind = "action"
)

// Step is one unit of execution during a table walk.
type Step struct {
	Kind      StepKind
	Condition string // EL text when Kind==StepCondition
	Result    bool   // evaluation result when Kind==StepCondition
	Column    int    // rule column this step belongs to
	Action    string // EL text when Kind==StepAction
	StateDiff []AttributeChange
	Error     error
}

// ExecutionTrace holds the complete record of a table execution.
type ExecutionTrace struct {
	Steps      []Step
	FinalState map[string]string // "Entity.Attribute" -> value
	Errors     []error
}

// execState holds the lazily-created rule session used for execution.
type execState struct {
	ruleSet *session.RuleSet
	sess    *session.RSession
	state   *interpreter.DTState
}

// ensureExecState builds the execState on Project if it doesn't exist yet.
func (p *Project) ensureExecState() error {
	if p.execSt != nil {
		return nil
	}

	rs := session.NewRuleSet("authoring")
	if rs == nil {
		return fmt.Errorf("failed to create rule set")
	}

	// Load EDD files
	eddFiles, _ := filepath.Glob(filepath.Join(p.xmlDir, "*_edd.xml"))
	for _, path := range eddFiles {
		if err := rs.LoadEDDFile(path); err != nil {
			return fmt.Errorf("LoadEDD %s: %w", path, err)
		}
	}

	// Write in-memory DT model to temp file and load it.
	// This ensures unsaved mutations are reflected in the session.
	for _, entry := range p.dtFiles {
		tmp, err := os.CreateTemp("", "authoring-dt-*.xml")
		if err != nil {
			return fmt.Errorf("create temp dt: %w", err)
		}
		tmpPath := tmp.Name()
		tmp.Close()

		imp := excel.NewDTImporter()
		if err := imp.WriteXML(entry.tables, tmpPath); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("write temp dt: %w", err)
		}
		if err := rs.LoadDecisionTablesFile(tmpPath); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("load temp dt %s: %w", tmpPath, err)
		}
		os.Remove(tmpPath)
	}

	rawSess, err := session.NewSession(rs)
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}

	dtState, ok := rawSess.GetState().(*interpreter.DTState)
	if !ok {
		return fmt.Errorf("unexpected state type %T", rawSess.GetState())
	}

	// Wire the global operator table so bytecode can call operators like xdef, cvb, etc.
	dtState.SetOperatorTable(operators.GetOperatorTable())

	p.execSt = &execState{
		ruleSet: rs,
		sess:    rawSess,
		state:   dtState,
	}
	return nil
}

// LoadTestData populates entity state from an XML test-data file using the
// project's _map.xml file. Accumulates across successive calls.
func (p *Project) LoadTestData(path string) error {
	if err := p.ensureExecState(); err != nil {
		return err
	}

	mapPath, err := p.findMapFile()
	if err != nil {
		return err
	}

	mapF, err := os.Open(mapPath)
	if err != nil {
		return fmt.Errorf("open map file: %w", err)
	}
	defer mapF.Close()

	dataF, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open test data: %w", err)
	}
	defer dataF.Close()

	return p.loadTestDataFromReaders(mapF, dataF)
}

// LoadTestDataReader populates entity state from readers for map+data XML.
func (p *Project) LoadTestDataReader(mapReader, dataReader io.Reader) error {
	if err := p.ensureExecState(); err != nil {
		return err
	}
	return p.loadTestDataFromReaders(mapReader, dataReader)
}

func (p *Project) loadTestDataFromReaders(mapReader, dataReader io.Reader) error {
	m := mapping.NewMapping(p.execSt.sess)
	if err := m.LoadMapping(mapReader); err != nil {
		return fmt.Errorf("load mapping: %w", err)
	}
	// LoadDataAndPushSingletons creates entities from XML and pushes singletons
	// onto the entity stack so conditions/actions can resolve attributes.
	if err := m.LoadDataAndPushSingletons(dataReader); err != nil {
		return fmt.Errorf("load data: %w", err)
	}
	return nil
}

// SetAttribute sets an entity attribute programmatically.
// entityName is the entity type name (e.g. "applicant"), attribute is the
// field name (e.g. "age"), value is a Go primitive or string.
func (p *Project) SetAttribute(entityName, attribute string, value any) error {
	if err := p.ensureExecState(); err != nil {
		return err
	}

	eName := dtrules.GetRName(entityName)
	if eName == nil {
		return fmt.Errorf("invalid entity name: %s", entityName)
	}
	aName := dtrules.GetRName(attribute)
	if aName == nil {
		return fmt.Errorf("invalid attribute name: %s", attribute)
	}

	ent, err := p.findEntityOnStack(eName)
	if err != nil {
		return err
	}
	if ent == nil {
		// Entity not on stack — create one and push it
		newEnt, err := p.execSt.sess.CreateEntity(eName)
		if err != nil {
			return fmt.Errorf("create entity %s: %w", entityName, err)
		}
		if err := p.execSt.state.EntityPush(newEnt); err != nil {
			return fmt.Errorf("push entity %s: %w", entityName, err)
		}
		ent = newEnt
	}

	obj, err := goValueToDTRules(value)
	if err != nil {
		return fmt.Errorf("convert value: %w", err)
	}
	return ent.Put(aName, obj)
}

// ResetState drops the loaded entity state; the rule set is preserved so
// EDD/DT do not need to be reloaded from disk.
func (p *Project) ResetState() {
	if p.execSt == nil {
		return
	}
	rawSess, err := session.NewSession(p.execSt.ruleSet)
	if err != nil {
		p.execSt = nil
		return
	}
	dtState, ok := rawSess.GetState().(*interpreter.DTState)
	if !ok {
		p.execSt = nil
		return
	}
	dtState.SetOperatorTable(operators.GetOperatorTable())
	p.execSt.sess = rawSess
	p.execSt.state = dtState
}

// EvalCondition compiles an EL condition and evaluates it against current state.
// Uses the Object-based execution path so operators (cvb, xdef, etc.) work correctly.
func (p *Project) EvalCondition(el string) (bool, error) {
	if err := p.ensureExecState(); err != nil {
		return false, err
	}
	postfix, err := CheckCondition(el, p.symbols)
	if err != nil {
		return false, fmt.Errorf("compile condition: %w", err)
	}
	code, err := p.execSt.sess.Compile(postfix)
	if err != nil {
		return false, fmt.Errorf("compile postfix: %w", err)
	}
	state := p.execSt.state
	initialDepth := state.DataStackDepth()
	if err := code.Execute(state); err != nil {
		return false, err
	}
	if state.DataStackDepth() <= initialDepth {
		return false, fmt.Errorf("condition produced no result on stack")
	}
	result, err := state.DataPop()
	if err != nil {
		return false, err
	}
	// Clean up any extra values
	for state.DataStackDepth() > initialDepth {
		state.DataPop()
	}
	boolResult, err := result.BooleanValue()
	if err != nil {
		return false, fmt.Errorf("condition result is not boolean: %w", err)
	}
	return boolResult, nil
}

// EvalAction compiles an EL action, executes it, and returns attribute changes.
// Uses the Object-based execution path so operators (xdef, cvb, etc.) work correctly.
func (p *Project) EvalAction(el string) ([]AttributeChange, error) {
	if err := p.ensureExecState(); err != nil {
		return nil, err
	}
	postfix, err := CheckAction(el, p.symbols)
	if err != nil {
		return nil, fmt.Errorf("compile action: %w", err)
	}
	code, err := p.execSt.sess.Compile(postfix)
	if err != nil {
		return nil, fmt.Errorf("compile postfix: %w", err)
	}

	state := p.execSt.state
	before := p.snapshotState()
	if err := code.Execute(state); err != nil {
		return nil, err
	}
	return p.diffState(before), nil
}

// Execute runs the full decision table and returns an execution trace.
func (t *Table) Execute(p *Project) (*ExecutionTrace, error) {
	if err := p.ensureExecState(); err != nil {
		return nil, err
	}

	trace := &ExecutionTrace{}
	stepper := t.NewStepper(p)

	for stepper.Next() {
		step := stepper.Current()
		trace.Steps = append(trace.Steps, step)
		if step.Error != nil {
			trace.Errors = append(trace.Errors, step.Error)
		}
	}

	trace.FinalState = p.snapshotState()
	return trace, nil
}

// Stepper walks a decision table one step at a time.
type Stepper struct {
	table   *Table
	project *Project

	// condCache caches condition evaluations: conditionNumber -> bool result
	condCache map[int]bool

	// pending is a FIFO of steps yet to be returned from Next()
	pending []Step

	done    bool
	current Step

	// col is the next column to process (1-based)
	col int
}

// NewStepper creates a new Stepper for step-by-step execution.
func (t *Table) NewStepper(p *Project) *Stepper {
	return &Stepper{
		table:     t,
		project:   p,
		condCache: make(map[int]bool),
		col:       1,
	}
}

// Next advances to the next step. Returns false when exhausted.
func (s *Stepper) Next() bool {
	// Drain any pending steps before checking done — this ensures action steps
	// for the last matched column (e.g. under FIRST policy) are still yielded.
	if len(s.pending) > 0 {
		s.current = s.pending[0]
		s.pending = s.pending[1:]
		if s.current.Error != nil {
			s.done = true
		}
		return true
	}

	if s.done {
		return false
	}

	if err := s.project.ensureExecState(); err != nil {
		s.current = Step{Kind: StepCondition, Error: err}
		s.done = true
		return true
	}

	numCols := s.table.Columns()
	for s.col <= numCols {
		steps, matched, err := s.processColumn(s.col)
		if err != nil {
			s.current = Step{Kind: StepCondition, Column: s.col, Error: err}
			s.done = true
			return true
		}

		if len(steps) > 0 {
			s.pending = append(s.pending, steps[1:]...)
			s.current = steps[0]
			if matched && strings.ToUpper(s.table.Policy) == "FIRST" {
				s.done = true // stop after this column's actions drain
			}
			s.col++
			return true
		}

		// No steps for this column (no conditions matched, no actions)
		if matched && strings.ToUpper(s.table.Policy) == "FIRST" {
			s.done = true
			s.col++
			return false
		}
		s.col++
	}

	s.done = true
	return false
}

// processColumn evaluates conditions and, if matched, collects action steps
// for the given column. Returns (steps, columnMatched, error).
func (s *Stepper) processColumn(col int) ([]Step, bool, error) {
	var steps []Step

	// Evaluate conditions (cached), yield a step per relevant condition
	matched := true
	for _, cond := range s.table.Conditions {
		colVal, hasVal := cond.Columns[col]
		if !hasVal || colVal == "-" {
			continue
		}

		result, cached := s.condCache[cond.Number]
		if !cached {
			var evalErr error
			result, evalErr = s.project.EvalCondition(cond.DSL)
			if evalErr != nil {
				return nil, false, evalErr
			}
			s.condCache[cond.Number] = result
		}

		steps = append(steps, Step{
			Kind:      StepCondition,
			Condition: cond.DSL,
			Result:    result,
			Column:    col,
		})

		wantTrue := (colVal == "Y")
		if result != wantTrue {
			matched = false
		}
	}

	if !matched {
		return steps, false, nil
	}

	// Column matched — collect action steps
	for _, act := range s.table.Actions {
		if !act.Columns[col] || act.DSL == "" {
			continue
		}
		before := s.project.snapshotState()
		_, evalErr := s.project.EvalAction(act.DSL)
		var diff []AttributeChange
		if evalErr == nil {
			diff = s.project.diffState(before)
		}
		steps = append(steps, Step{
			Kind:      StepAction,
			Action:    act.DSL,
			Column:    col,
			StateDiff: diff,
			Error:     evalErr,
		})
		if evalErr != nil {
			return steps, true, nil
		}
	}

	return steps, true, nil
}

// Current returns the most recently yielded step.
func (s *Stepper) Current() Step {
	return s.current
}

// snapshotState captures the current string value of all entity attributes
// on the entity stack.
func (p *Project) snapshotState() map[string]string {
	snap := make(map[string]string)
	if p.execSt == nil {
		return snap
	}
	state := p.execSt.state
	depth := state.EntityDepth()
	for i := 0; i < depth; i++ {
		ent, err := state.EntityFetch(i)
		if err != nil {
			continue
		}
		re, ok := ent.(*entity.REntity)
		if !ok {
			continue
		}
		if re.IsReadOnly() {
			continue
		}
		entName := re.GetName().StringValue()
		for _, attrName := range re.GetAttributeNames() {
			val, err := re.Get(attrName)
			if err != nil || val == nil {
				continue
			}
			key := entName + "." + attrName.StringValue()
			snap[key] = val.StringValue()
		}
	}
	return snap
}

// diffState compares before snapshot with current state and returns changes.
func (p *Project) diffState(before map[string]string) []AttributeChange {
	after := p.snapshotState()
	var changes []AttributeChange
	for key, newVal := range after {
		oldVal := before[key]
		if oldVal != newVal {
			parts := strings.SplitN(key, ".", 2)
			if len(parts) != 2 {
				continue
			}
			changes = append(changes, AttributeChange{
				Entity:    parts[0],
				Attribute: parts[1],
				OldValue:  oldVal,
				NewValue:  newVal,
			})
		}
	}
	return changes
}

// findEntityOnStack searches the entity stack for an entity with the given name.
func (p *Project) findEntityOnStack(name *dtrules.RName) (dtrules.Entity, error) {
	state := p.execSt.state
	depth := state.EntityDepth()
	for i := 0; i < depth; i++ {
		ent, err := state.EntityFetch(i)
		if err != nil {
			continue
		}
		if ent.GetName() == name {
			return ent, nil
		}
	}
	return nil, nil
}

// findMapFile looks for a *_map.xml file in the project xml dir.
func (p *Project) findMapFile() (string, error) {
	maps, err := filepath.Glob(filepath.Join(p.xmlDir, "*_map.xml"))
	if err != nil {
		return "", err
	}
	if len(maps) == 0 {
		return "", fmt.Errorf("no *_map.xml file found in %s", p.xmlDir)
	}
	return maps[0], nil
}

// goValueToDTRules converts a Go primitive to a dtrules.Object.
func goValueToDTRules(v any) (dtrules.Object, error) {
	switch val := v.(type) {
	case bool:
		return dtrules.GetRBoolean(val), nil
	case int:
		return dtrules.GetRIntegerValue(int64(val)), nil
	case int64:
		return dtrules.GetRIntegerValue(val), nil
	case float64:
		return dtrules.GetRDoubleValue(val), nil
	case float32:
		return dtrules.GetRDoubleValue(float64(val)), nil
	case string:
		return dtrules.NewRString(val), nil
	case dtrules.Object:
		return val, nil
	default:
		return nil, fmt.Errorf("unsupported value type %T", v)
	}
}
