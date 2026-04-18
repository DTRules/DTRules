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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/operators"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// TableInvocation records one decision table invocation in a RunTrace.
type TableInvocation struct {
	Index      int               // Position in the flat pre-order list
	TableName  string            // Name of the invoked table
	Depth      int               // Call depth (0 = entry table, 1 = called by entry, etc.)
	StartState map[string]string // Entity attribute snapshot before execution
	EndState   map[string]string // Entity attribute snapshot after execution
}

// RunTrace is the complete record of an ExecuteEntry call.
type RunTrace struct {
	EntryTable  string
	InputState  map[string]string  // Snapshot of entity state before any execution
	Invocations []*TableInvocation // Pre-order flattened list of all table invocations
	FinalState  map[string]string  // Entity attribute snapshot after all execution
}

// EntityView is a read-only view of a single entity on the stack.
type EntityView struct {
	Name       string
	Attributes map[string]string
}

// DebugSession is a live session paused before a specific invocation.
type DebugSession struct {
	project    *Project
	trace      *RunTrace     // Original trace from which we replayed
	pauseIndex int           // The index we are paused before
	execSt     *execState    // Dedicated exec state for this session
}

// traceBuilder accumulates TableInvocation records during ExecuteEntry.
type traceBuilder struct {
	invocations  []*TableInvocation
	project      *Project
	currentDepth int
	stopAt       int  // if > 0, stop (via sentinel error) after this many invocations
	execSt       *execState // if set, used instead of project.execSt for snapshots
}

func (b *traceBuilder) snapshotState() map[string]string {
	if b.execSt != nil {
		return snapshotExecState(b.execSt)
	}
	return b.project.snapshotState()
}

// snapshotExecState captures entity state from an arbitrary execState.
func snapshotExecState(est *execState) map[string]string {
	snap := make(map[string]string)
	if est == nil {
		return snap
	}
	state := est.state
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

// fastForwardStop is a sentinel error to halt fast-forward replay.
type fastForwardStop struct{}

func (e *fastForwardStop) Error() string { return "fast-forward stop" }

func (e *fastForwardStop) Is(target error) bool {
	_, ok := target.(*fastForwardStop)
	return ok
}

var errFastForwardStop = &fastForwardStop{}

func isFastForwardStop(err error) bool {
	return errors.Is(err, errFastForwardStop)
}

// dtNameGetter is a local interface for factories that expose decision table names.
type dtNameGetter interface {
	GetDecisionTableNames() []*dtrules.RName
}

// installTracingWrappers replaces all decision tables in the factory's entity with
// depthAwareWrappers that record invocations into builder. Returns a restore func.
func installTracingWrappers(ef dtrules.EntityFactory, builder *traceBuilder) (func(), error) {
	dtEntity := ef.GetDecisionTablesEntity()
	re, ok := dtEntity.(*entity.REntity)
	if !ok {
		return func() {}, fmt.Errorf("decision tables entity is not *entity.REntity")
	}

	ng, ok := ef.(dtNameGetter)
	if !ok {
		return func() {}, fmt.Errorf("entity factory does not expose GetDecisionTableNames")
	}

	tableNames := ng.GetDecisionTableNames()
	originals := make(map[*dtrules.RName]dtrules.Object, len(tableNames))

	for _, name := range tableNames {
		orig, err := re.Get(name)
		if err != nil || orig == nil {
			continue
		}
		dt, ok := orig.(dtrules.DecisionTable)
		if !ok {
			continue
		}
		originals[name] = orig

		wrapper := &depthAwareWrapper{
			inner:   dt,
			name:    name,
			builder: builder,
		}
		if err := re.Put(name, wrapper); err != nil {
			for rname, obj := range originals {
				re.Put(rname, obj) //nolint:errcheck
			}
			return func() {}, fmt.Errorf("install wrapper for %s: %w", name.StringValue(), err)
		}
	}

	restore := func() {
		for name, obj := range originals {
			re.Put(name, obj) //nolint:errcheck
		}
	}
	return restore, nil
}

// depthAwareWrapper wraps a DecisionTable and records each invocation into a traceBuilder.
type depthAwareWrapper struct {
	inner   dtrules.DecisionTable
	name    *dtrules.RName
	builder *traceBuilder
}

func (w *depthAwareWrapper) Type() *dtrules.RType { return dtrules.TypeDecisionTable }

func (w *depthAwareWrapper) Execute(state dtrules.State) error {
	return w.executeTracked(state, false)
}

func (w *depthAwareWrapper) ExecuteTable(state dtrules.State) error {
	return w.executeTracked(state, true)
}

func (w *depthAwareWrapper) ArrayExecute(state dtrules.State) error {
	return w.Execute(state)
}

func (w *depthAwareWrapper) GetExecutable() dtrules.Object    { return w }
func (w *depthAwareWrapper) GetNonExecutable() dtrules.Object { return w }
func (w *depthAwareWrapper) IsExecutable() bool               { return true }

func (w *depthAwareWrapper) Equals(o dtrules.Object) (bool, error) {
	other, ok := o.(*depthAwareWrapper)
	return ok && w == other, nil
}

func (w *depthAwareWrapper) Compare(o dtrules.Object) (int, error) {
	return 0, fmt.Errorf("compare not supported on depthAwareWrapper")
}

func (w *depthAwareWrapper) StringValue() string { return w.name.StringValue() }
func (w *depthAwareWrapper) PostFix() string     { return w.name.StringValue() + " performaliased " }

func (w *depthAwareWrapper) Clone(s dtrules.Session) (dtrules.Object, error) { return w, nil }
func (w *depthAwareWrapper) RClone() dtrules.Object                           { return w }
func (w *depthAwareWrapper) IntValue() (int, error)                           { return 0, unsupported("IntValue") }
func (w *depthAwareWrapper) LongValue() (int64, error)                        { return 0, unsupported("LongValue") }
func (w *depthAwareWrapper) DoubleValue() (float64, error)                    { return 0, unsupported("DoubleValue") }
func (w *depthAwareWrapper) BooleanValue() (bool, error)                      { return false, unsupported("BooleanValue") }
func (w *depthAwareWrapper) TimeValue() (time.Time, error) {
	return time.Time{}, unsupported("TimeValue")
}
func (w *depthAwareWrapper) ArrayValue() ([]dtrules.Object, error) {
	return nil, unsupported("ArrayValue")
}
func (w *depthAwareWrapper) TableValue() (map[dtrules.Object]dtrules.Object, error) {
	return nil, unsupported("TableValue")
}
func (w *depthAwareWrapper) RIntegerValue() (*dtrules.RInteger, error) {
	return nil, unsupported("RIntegerValue")
}
func (w *depthAwareWrapper) RDoubleValue() (*dtrules.RDouble, error) {
	return nil, unsupported("RDoubleValue")
}
func (w *depthAwareWrapper) RBooleanValue() (*dtrules.RBoolean, error) {
	return nil, unsupported("RBooleanValue")
}
func (w *depthAwareWrapper) RStringValue() *dtrules.RString {
	return dtrules.NewRString(w.name.StringValue())
}
func (w *depthAwareWrapper) RNameValue() (*dtrules.RName, error) { return w.name, nil }
func (w *depthAwareWrapper) RArrayValue() (*dtrules.RArray, error) {
	return nil, unsupported("RArrayValue")
}
func (w *depthAwareWrapper) RTableValue() (*dtrules.RTable, error) {
	return nil, unsupported("RTableValue")
}
func (w *depthAwareWrapper) RTimeValue() (*dtrules.RDate, error) {
	return nil, unsupported("RTimeValue")
}
func (w *depthAwareWrapper) REntityValue() (dtrules.Entity, error) {
	return nil, unsupported("REntityValue")
}
func (w *depthAwareWrapper) RBigIntValue() (*dtrules.RBigInt, error) {
	return nil, unsupported("RBigIntValue")
}
func (w *depthAwareWrapper) RBytesValue() (*dtrules.RBytes, error) {
	return nil, unsupported("RBytesValue")
}

func unsupported(method string) error {
	return fmt.Errorf("depthAwareWrapper: %s not supported", method)
}

func (w *depthAwareWrapper) executeTracked(state dtrules.State, tableOnly bool) error {
	b := w.builder

	// Check stop condition for fast-forward.
	if b.stopAt > 0 && len(b.invocations) >= b.stopAt {
		return &fastForwardStop{}
	}

	depth := b.currentDepth
	idx := len(b.invocations)
	inv := &TableInvocation{
		Index:      idx,
		TableName:  w.name.StringValue(),
		Depth:      depth,
		StartState: b.snapshotState(),
	}
	b.invocations = append(b.invocations, inv)
	b.currentDepth++

	var err error
	if tableOnly {
		err = w.inner.ExecuteTable(state)
	} else {
		err = w.inner.Execute(state)
	}

	b.currentDepth--
	if err != nil && !isFastForwardStop(err) {
		inv.EndState = b.snapshotState()
		return err
	}
	inv.EndState = b.snapshotState()
	return err
}

// ExecuteEntry runs the named decision table and returns a RunTrace of all
// table invocations (entry + any sub-table calls), in pre-order.
func (p *Project) ExecuteEntry(tableName string) (*RunTrace, error) {
	if err := p.ensureExecState(); err != nil {
		return nil, err
	}

	inputState := p.snapshotState()
	builder := &traceBuilder{project: p}

	restore, err := installTracingWrappers(p.execSt.sess.GetEntityFactory(), builder)
	if err != nil {
		return nil, fmt.Errorf("install tracing wrappers: %w", err)
	}
	defer restore()

	if err := p.execSt.sess.Execute(tableName); err != nil {
		return nil, fmt.Errorf("execute %s: %w", tableName, err)
	}

	finalState := p.snapshotState()
	return &RunTrace{
		EntryTable:  tableName,
		InputState:  inputState,
		Invocations: builder.invocations,
		FinalState:  finalState,
	}, nil
}

// newDebugExecState creates a fresh execState for replay, sharing the same ruleSet.
func newDebugExecState(rs *session.RuleSet) (*execState, error) {
	rawSess, err := session.NewSession(rs)
	if err != nil {
		return nil, fmt.Errorf("new session: %w", err)
	}

	dtState, ok := rawSess.GetState().(*interpreter.DTState)
	if !ok {
		return nil, fmt.Errorf("unexpected state type %T", rawSess.GetState())
	}
	dtState.SetOperatorTable(operators.GetOperatorTable())

	return &execState{
		ruleSet: rs,
		sess:    rawSess,
		state:   dtState,
	}, nil
}

// applySnapshot sets entity attributes in est to match snap.
// Entities that don't exist yet are created and pushed.
func applySnapshot(est *execState, snap map[string]string) error {
	type entry struct{ entityName, attrName, value string }
	var entries []entry
	for key, val := range snap {
		parts := strings.SplitN(key, ".", 2)
		if len(parts) != 2 {
			continue
		}
		entries = append(entries, entry{parts[0], parts[1], val})
	}

	for _, e := range entries {
		eName := dtrules.GetRName(e.entityName)
		aName := dtrules.GetRName(e.attrName)
		if eName == nil || aName == nil {
			continue
		}

		var ent dtrules.Entity
		depth := est.state.EntityDepth()
		for i := 0; i < depth; i++ {
			candidate, err := est.state.EntityFetch(i)
			if err != nil {
				continue
			}
			if candidate.GetName() == eName {
				ent = candidate
				break
			}
		}
		if ent == nil {
			newEnt, err := est.sess.CreateEntity(eName)
			if err != nil {
				continue
			}
			if err2 := est.state.EntityPush(newEnt); err2 != nil {
				continue
			}
			ent = newEnt
		}

		obj := parseSnapshotValue(e.value)
		ent.Put(aName, obj) //nolint:errcheck
	}
	return nil
}

// parseSnapshotValue converts a snapshot string value to a dtrules.Object.
func parseSnapshotValue(val string) dtrules.Object {
	switch strings.ToLower(val) {
	case "true":
		return dtrules.GetRBoolean(true)
	case "false":
		return dtrules.GetRBoolean(false)
	}
	var iv int64
	if n, _ := fmt.Sscanf(val, "%d", &iv); n == 1 {
		return dtrules.GetRIntegerValue(iv)
	}
	var fv float64
	if n, _ := fmt.Sscanf(val, "%g", &fv); n == 1 {
		return dtrules.GetRDoubleValue(fv)
	}
	return dtrules.NewRString(val)
}

// ResumeAt replays the given trace up to (but not including) the invocation at
// idx, then returns a DebugSession paused before that invocation.
func (p *Project) ResumeAt(trace *RunTrace, idx int) (*DebugSession, error) {
	if idx < 0 || idx >= len(trace.Invocations) {
		return nil, fmt.Errorf("index %d out of range [0, %d)", idx, len(trace.Invocations))
	}

	est, err := newDebugExecState(p.execSt.ruleSet)
	if err != nil {
		return nil, err
	}

	// Restore input state.
	if err := applySnapshot(est, trace.InputState); err != nil {
		return nil, fmt.Errorf("restore input state: %w", err)
	}

	sess := &DebugSession{
		project:    p,
		trace:      trace,
		pauseIndex: idx,
		execSt:     est,
	}

	// Fast-forward to idx by executing with a stop-after-N wrapper.
	if idx > 0 {
		if err := sess.fastForward(idx); err != nil {
			return nil, fmt.Errorf("fast-forward to index %d: %w", idx, err)
		}
	}

	return sess, nil
}

// fastForward executes the entry table on the debug session's execSt, stopping
// after exactly `count` invocations have been recorded.
func (sess *DebugSession) fastForward(count int) error {
	builder := &traceBuilder{
		project: sess.project,
		stopAt:  count,
		execSt:  sess.execSt,
	}

	restore, err := installTracingWrappers(sess.execSt.sess.GetEntityFactory(), builder)
	if err != nil {
		return err
	}
	defer restore()

	// Temporarily swap project.execSt so snapshotState operates on the debug session.
	origExecSt := sess.project.execSt
	sess.project.execSt = sess.execSt
	execErr := sess.execSt.sess.Execute(sess.trace.EntryTable)
	sess.project.execSt = origExecSt

	if isFastForwardStop(execErr) {
		return nil
	}
	return execErr
}

// EntityStack returns a snapshot of the entity stack in the live debug session.
func (sess *DebugSession) EntityStack() []EntityView {
	state := sess.execSt.state
	depth := state.EntityDepth()
	views := make([]EntityView, 0, depth)
	for i := 0; i < depth; i++ {
		ent, err := state.EntityFetch(i)
		if err != nil {
			continue
		}
		re, ok := ent.(*entity.REntity)
		if !ok {
			continue
		}
		view := EntityView{
			Name:       re.GetName().StringValue(),
			Attributes: make(map[string]string),
		}
		for _, attrName := range re.GetAttributeNames() {
			val, err := re.Get(attrName)
			if err != nil || val == nil {
				continue
			}
			view.Attributes[attrName.StringValue()] = val.StringValue()
		}
		views = append(views, view)
	}
	return views
}

// Resolve looks up an attribute by name on the live debug session's entity stack.
// Returns (value, entityName, error).
func (sess *DebugSession) Resolve(name string) (any, string, error) {
	rname := dtrules.GetRName(name)
	if rname == nil {
		return nil, "", fmt.Errorf("invalid name: %s", name)
	}

	state := sess.execSt.state
	obj, err := state.Find(rname)
	if err != nil {
		return nil, "", err
	}
	if obj == nil {
		return nil, "", fmt.Errorf("name not found: %s", name)
	}

	ent, err := state.FindEntity(rname)
	if err != nil || ent == nil {
		return obj.StringValue(), "", nil
	}
	return obj.StringValue(), ent.GetName().StringValue(), nil
}

// NextInvocation returns the TableInvocation we are paused before, or nil if done.
func (sess *DebugSession) NextInvocation() *TableInvocation {
	if sess.pauseIndex >= len(sess.trace.Invocations) {
		return nil
	}
	return sess.trace.Invocations[sess.pauseIndex]
}

// Step runs the paused invocation and re-pauses before the next one.
// Returns a live TableInvocation with EndState reflecting what actually happened.
func (sess *DebugSession) Step() (*TableInvocation, error) {
	if sess.pauseIndex >= len(sess.trace.Invocations) {
		return nil, fmt.Errorf("no more invocations to step")
	}
	inv := sess.trace.Invocations[sess.pauseIndex]
	tableName := inv.TableName

	// Snapshot before so we can report StartState accurately.
	startSnap := snapshotExecState(sess.execSt)

	// Execute this one table on the debug session. Use a builder that stops after
	// one invocation so sub-tables don't bleed into subsequent steps.
	// However, sub-tables are separate invocations in the trace. Step runs exactly
	// the invocation at pauseIndex. If it's at depth 0, running it will also run
	// its children (which are depth > 0 invocations following it). We skip those.
	builder := &traceBuilder{
		project: sess.project,
		stopAt:  0, // no stop — run full invocation
		execSt:  sess.execSt,
	}

	restore, err := installTracingWrappers(sess.execSt.sess.GetEntityFactory(), builder)
	if err != nil {
		return nil, err
	}

	origExecSt := sess.project.execSt
	sess.project.execSt = sess.execSt
	execErr := sess.execSt.sess.Execute(tableName)
	sess.project.execSt = origExecSt
	restore()

	if execErr != nil {
		return nil, fmt.Errorf("step %s: %w", tableName, execErr)
	}

	endSnap := snapshotExecState(sess.execSt)
	liveInv := &TableInvocation{
		Index:      inv.Index,
		TableName:  inv.TableName,
		Depth:      inv.Depth,
		StartState: startSnap,
		EndState:   endSnap,
	}

	// Advance pauseIndex past this invocation and all its children.
	sess.pauseIndex++
	for sess.pauseIndex < len(sess.trace.Invocations) &&
		sess.trace.Invocations[sess.pauseIndex].Depth > inv.Depth {
		sess.pauseIndex++
	}

	return liveInv, nil
}

// Continue runs from the current pause point to the end, returning a RunTrace
// of what actually happened (may diverge from original if state was mutated).
func (sess *DebugSession) Continue() (*RunTrace, error) {
	inputSnap := snapshotExecState(sess.execSt)
	builder := &traceBuilder{
		project: sess.project,
		execSt:  sess.execSt,
	}

	restore, err := installTracingWrappers(sess.execSt.sess.GetEntityFactory(), builder)
	if err != nil {
		return nil, err
	}
	defer restore()

	origExecSt := sess.project.execSt
	sess.project.execSt = sess.execSt

	// Execute remaining invocations. We execute each invocation that would NOT
	// be triggered automatically by its parent — i.e., all remaining invocations
	// that don't have an ancestor already in our execution list.
	// We walk the remaining trace and execute any invocation directly whose parent
	// has already completed. If we're paused at depth > 0, that invocation needs
	// to be executed directly (its parent already ran partially).
	var execErr error
	idx := sess.pauseIndex
	for idx < len(sess.trace.Invocations) {
		inv := sess.trace.Invocations[idx]

		// Execute this invocation directly (sub-tables will be captured by wrappers).
		if err2 := sess.execSt.sess.Execute(inv.TableName); err2 != nil {
			execErr = err2
			break
		}

		// Advance past this invocation and any sub-invocations it produced.
		idx++
		for idx < len(sess.trace.Invocations) &&
			sess.trace.Invocations[idx].Depth > inv.Depth {
			idx++
		}
	}
	sess.pauseIndex = idx

	sess.project.execSt = origExecSt

	if execErr != nil {
		return nil, execErr
	}

	finalState := snapshotExecState(sess.execSt)
	return &RunTrace{
		EntryTable:  sess.trace.EntryTable,
		InputState:  inputSnap,
		Invocations: builder.invocations,
		FinalState:  finalState,
	}, nil
}

// SetAttribute sets an attribute on an entity in the live debug session.
func (sess *DebugSession) SetAttribute(entityName, attribute string, value any) error {
	eName := dtrules.GetRName(entityName)
	if eName == nil {
		return fmt.Errorf("invalid entity name: %s", entityName)
	}
	aName := dtrules.GetRName(attribute)
	if aName == nil {
		return fmt.Errorf("invalid attribute name: %s", attribute)
	}

	state := sess.execSt.state
	var ent dtrules.Entity
	depth := state.EntityDepth()
	for i := 0; i < depth; i++ {
		e, err := state.EntityFetch(i)
		if err != nil {
			continue
		}
		if e.GetName() == eName {
			ent = e
			break
		}
	}
	if ent == nil {
		newEnt, err := sess.execSt.sess.CreateEntity(eName)
		if err != nil {
			return fmt.Errorf("create entity %s: %w", entityName, err)
		}
		if err := state.EntityPush(newEnt); err != nil {
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

// Close releases resources held by this debug session.
func (sess *DebugSession) Close() {
	sess.execSt = nil
}
