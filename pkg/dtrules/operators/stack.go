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

package operators

import (
	"log"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

func init() {
	// Stack manipulation
	Register("pop", opPop)
	Alias("pop", "drop")
	Register("dup", opDup)
	Register("swap", opSwap)
	Alias("swap", "exch")
	Register("over", opOver)
	Register("rot", opRot)
	Register("pick", opPick)
	Register("roll", opRoll)
	Register("clear", opClear)

	// Null
	Register("null", opNull)

	// Mark
	Register("mark", opMark)
	Register("arraytomark", opArrayToMark)
	Register("counttomark", opCountToMark)
	Register("cleartomark", opClearToMark)

	// Debug and trace
	Register("debug", opDebug)
	Register("print", opPrint)
	Register("traceon", opTraceOn)
	Register("traceoff", opTraceOff)
	Register("setdebug", opSetDebug)
	Register("pstack", opPStack)
	Register("printtos", opPrintTOS)

	// Clone
	Register("clone", opClone)

	// Control stack
	Register(">r", opToR)
	Register("r>", opFromR)
	Register("i", opI)
	Alias("i", "r@")
	Register("j", opJ)
	Register("k", opK)

	// Entity operations
	Register("entityfetch", opEntityFetch)
	Register("get", opGet)
	Register("find", opFind)
	Register("xdef", opXDef)
	// `createentity` is an alias for `newentity` — they construct an entity
	// instance by name through the session's factory. Same runtime behavior.
	Alias("newentity", "createentity")
	Register("findcreateentity", opFindCreateEntity)
	Alias("findcreateentity", "fce")

	// Type conversions.
	//
	// Naming: the letter after `cv` is the target type's short code. Until
	// #694 was fixed, `cvd` was misleadingly the date converter and the
	// emitter was emitting `cvd` for double-typed fields — so every
	// `set <double> = <expr>` stored null. Now `cvd` is double and the
	// explicit `cvdate` is the date converter. `cvr` remains as a
	// back-compat alias of `cvd` (some stored postfix predates this
	// rename).
	Register("cvi", opCvi)
	Register("cvd", opCvd) // double
	Alias("cvd", "cvr")
	Register("cvb", opCvb)
	Register("cve", opCve)
	Register("cvn", opCvn)
	Register("cvdate", opCvDate) // date

	// Error handling (legacy two-arg form)
	Register("error", opError)

	// EL statement opcodes — registered so the compiler can resolve the token
	// at compile time; runtime dispatch goes through OpError/OpWarn opcodes
	Register("elstmterror", opELStmtError)
	Register("elstmtwarn", opELStmtWarn)
}

// opPop: ( a -- ) removes top element
func opPop(state dtrules.State) error {
	_, err := state.DataPop()
	return err
}

// opDup: ( a -- a a ) duplicates top element
func opDup(state dtrules.State) error {
	a, err := state.DataPeek()
	if err != nil {
		return err
	}
	return state.DataPush(a)
}

// opSwap: ( a b -- b a ) swaps top two elements
func opSwap(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	if err := state.DataPush(b); err != nil {
		return err
	}
	return state.DataPush(a)
}

// opOver: ( a b -- a b a ) copies second element to top
func opOver(state dtrules.State) error {
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPeek()
	if err != nil {
		return err
	}
	if err := state.DataPush(b); err != nil {
		return err
	}
	return state.DataPush(a)
}

// opRot: ( a b c -- b c a ) rotates third element to top
func opRot(state dtrules.State) error {
	c, err := state.DataPop()
	if err != nil {
		return err
	}
	b, err := state.DataPop()
	if err != nil {
		return err
	}
	a, err := state.DataPop()
	if err != nil {
		return err
	}
	if err := state.DataPush(b); err != nil {
		return err
	}
	if err := state.DataPush(c); err != nil {
		return err
	}
	return state.DataPush(a)
}

// opPick: ( ... n -- ... nth ) copies nth element to top (0 = top)
func opPick(state dtrules.State) error {
	nObj, err := state.DataPop()
	if err != nil {
		return err
	}
	n, err := nObj.IntValue()
	if err != nil {
		return err
	}

	depth := state.DataStackDepth()
	if n < 0 || n >= depth {
		return dtrules.OutOfBoundsError("pick", "Index out of bounds")
	}

	// GetDataStack uses 0 = bottom, so convert from top-relative
	picked, err := state.GetDataStack(depth - 1 - n)
	if err != nil {
		return err
	}
	return state.DataPush(picked)
}

// opRoll: ( ... n -- ... ) rotates top n elements
func opRoll(state dtrules.State) error {
	nObj, err := state.DataPop()
	if err != nil {
		return err
	}
	n, err := nObj.IntValue()
	if err != nil {
		return err
	}

	if n <= 0 {
		return nil
	}

	depth := state.DataStackDepth()
	if n > depth {
		return dtrules.OutOfBoundsError("roll", "Not enough elements on stack")
	}

	// Pop all elements we're rolling
	temp := make([]dtrules.Object, n)
	for i := 0; i < n; i++ {
		temp[i], err = state.DataPop()
		if err != nil {
			return err
		}
	}

	// Push bottom element first, then rest
	if err := state.DataPush(temp[0]); err != nil {
		return err
	}
	for i := n - 1; i >= 1; i-- {
		if err := state.DataPush(temp[i]); err != nil {
			return err
		}
	}
	return nil
}

// opClear: ( ... -- ) clears the data stack
func opClear(state dtrules.State) error {
	for state.DataStackDepth() > 0 {
		if _, err := state.DataPop(); err != nil {
			return err
		}
	}
	return nil
}

// opNull: ( -- null ) pushes null onto stack
func opNull(state dtrules.State) error {
	return state.DataPush(dtrules.GetRNull())
}

// opMark: ( -- mark ) pushes mark onto stack
func opMark(state dtrules.State) error {
	return state.DataPush(dtrules.GetMark())
}

// opArrayToMark: ( mark ... -- array ) collects elements to mark into array
func opArrayToMark(state dtrules.State) error {
	elements := make([]dtrules.Object, 0)

	for {
		obj, err := state.DataPop()
		if err != nil {
			return dtrules.UndefinedError("arraytomark", "No mark found on stack")
		}
		if obj == dtrules.GetMark() {
			break
		}
		// Prepend since we're popping in reverse order
		elements = append([]dtrules.Object{obj}, elements...)
	}

	arr, err := dtrules.NewArrayWithElements(state.GetSession(), true, elements, false)
	if err != nil {
		return err
	}
	return state.DataPush(arr)
}

// opCountToMark: ( mark ... -- mark ... count ) counts elements to mark
func opCountToMark(state dtrules.State) error {
	count := 0
	temp := make([]dtrules.Object, 0)

	for {
		obj, err := state.DataPop()
		if err != nil {
			return dtrules.UndefinedError("counttomark", "No mark found on stack")
		}
		if obj == dtrules.GetMark() {
			if err := state.DataPush(obj); err != nil {
				return err
			}
			break
		}
		temp = append(temp, obj)
		count++
	}

	// Restore elements
	for i := len(temp) - 1; i >= 0; i-- {
		if err := state.DataPush(temp[i]); err != nil {
			return err
		}
	}
	return state.DataPush(dtrules.GetRIntegerValueFromInt(count))
}

// opClearToMark: ( mark ... -- ) removes elements up to and including mark
func opClearToMark(state dtrules.State) error {
	for {
		obj, err := state.DataPop()
		if err != nil {
			return dtrules.UndefinedError("cleartomark", "No mark found on stack")
		}
		if obj == dtrules.GetMark() {
			return nil
		}
	}
}

// opDebug: ( string -- ) prints debug message if debug is enabled
func opDebug(state dtrules.State) error {
	msg, err := state.DataPop()
	if err != nil {
		return err
	}
	if state.TestState(dtrules.StateDebug) {
		state.Debug(msg.StringValue())
	}
	return nil
}

// opPrint: ( string -- ) prints message
func opPrint(state dtrules.State) error {
	msg, err := state.DataPop()
	if err != nil {
		return err
	}
	state.Print(msg.StringValue())
	return nil
}

// opTraceOn: ( -- ) turns on trace
func opTraceOn(state dtrules.State) error {
	state.SetState(dtrules.StateTrace)
	return nil
}

// opTraceOff: ( -- ) turns off trace
func opTraceOff(state dtrules.State) error {
	state.ClearState(dtrules.StateTrace)
	return nil
}

// opSetDebug: ( boolean -- ) sets debug state
func opSetDebug(state dtrules.State) error {
	flgObj, err := state.DataPop()
	if err != nil {
		return err
	}
	flg, err := flgObj.BooleanValue()
	if err != nil {
		return err
	}
	if flg {
		state.SetState(dtrules.StateDebug)
	} else {
		state.ClearState(dtrules.StateDebug)
	}
	return nil
}

// opPStack: ( -- ) prints all stacks for debugging
func opPStack(state dtrules.State) error {
	state.PStack()
	return nil
}

// opPrintTOS: ( obj -- ) prints top of stack
func opPrintTOS(state dtrules.State) error {
	obj, err := state.DataPop()
	if err != nil {
		return err
	}
	state.Debug(obj.StringValue())
	return nil
}

// opClone: ( obj -- obj2 ) creates a clone of the object
func opClone(state dtrules.State) error {
	obj, err := state.DataPop()
	if err != nil {
		return err
	}
	cloned, err := obj.Clone(state.GetSession())
	if err != nil {
		return err
	}
	return state.DataPush(cloned)
}

// opToR: ( obj -- ) pushes from data stack to control stack
func opToR(state dtrules.State) error {
	obj, err := state.DataPop()
	if err != nil {
		return err
	}
	state.CtrlPush(obj)
	return nil
}

// opFromR: ( -- obj ) pops from control stack to data stack
func opFromR(state dtrules.State) error {
	obj, err := state.CtrlPop()
	if err != nil {
		return err
	}
	return state.DataPush(obj)
}

// opI: ( -- obj ) returns top of control stack
func opI(state dtrules.State) error {
	depth := state.CtrlStackDepth()
	if depth == 0 {
		return dtrules.OutOfBoundsError("i", "Control stack is empty")
	}
	obj, err := state.GetCtrlStack(depth - 1)
	if err != nil {
		return err
	}
	return state.DataPush(obj)
}

// opJ: ( -- obj ) returns second element from control stack
func opJ(state dtrules.State) error {
	depth := state.CtrlStackDepth()
	if depth < 2 {
		return dtrules.OutOfBoundsError("j", "Control stack too shallow")
	}
	obj, err := state.GetCtrlStack(depth - 2)
	if err != nil {
		return err
	}
	return state.DataPush(obj)
}

// opK: ( -- obj ) returns third element from control stack
func opK(state dtrules.State) error {
	depth := state.CtrlStackDepth()
	if depth < 3 {
		return dtrules.OutOfBoundsError("k", "Control stack too shallow")
	}
	obj, err := state.GetCtrlStack(depth - 3)
	if err != nil {
		return err
	}
	return state.DataPush(obj)
}

// opEntityFetch: ( index -- entity ) returns nth entity from entity stack
func opEntityFetch(state dtrules.State) error {
	indexObj, err := state.DataPop()
	if err != nil {
		return err
	}
	index, err := indexObj.IntValue()
	if err != nil {
		return err
	}
	entity, err := state.EntityFetch(index)
	if err != nil {
		return err
	}
	return state.DataPush(entity.(dtrules.Object))
}

// opGet: ( entity attribute -- value ) gets attribute from entity
func opGet(state dtrules.State) error {
	nameObj, err := state.DataPop()
	if err != nil {
		return err
	}
	entityObj, err := state.DataPop()
	if err != nil {
		return err
	}
	name, err := nameObj.RNameValue()
	if err != nil {
		return err
	}
	entity, err := entityObj.REntityValue()
	if err != nil {
		return err
	}
	value, err := entity.Get(name)
	if err != nil {
		return err
	}
	if value == nil {
		return state.DataPush(dtrules.GetRNull())
	}
	return state.DataPush(value)
}

// opFind: ( name -- value ) looks up name on entity stack
func opFind(state dtrules.State) error {
	nameObj, err := state.DataPop()
	if err != nil {
		return err
	}
	name, err := nameObj.RNameValue()
	if err != nil {
		return err
	}
	value, err := state.Find(name)
	if err != nil {
		return err
	}
	if value == nil {
		return dtrules.UndefinedError("find", name.StringValue()+" is undefined")
	}
	return state.DataPush(value)
}

// opXDef: ( value name -- ) defines name with value (opposite order of def)
func opXDef(state dtrules.State) error {
	nameObj, err := state.DataPop()
	if err != nil {
		return err
	}
	value, err := state.DataPop()
	if err != nil {
		return err
	}
	name, err := nameObj.RNameValue()
	if err != nil {
		return err
	}
	ok, err := state.Def(name, value, true)
	if err != nil {
		return err
	}
	if !ok {
		return dtrules.UndefinedError("xdef", name.StringValue()+" is undefined")
	}
	return nil
}

// opFindCreateEntity: ( name id -- entity ) finds existing entity by id, or creates a new one
func opFindCreateEntity(state dtrules.State) error {
	idObj, err := state.DataPop()
	if err != nil {
		return err
	}
	nameObj, err := state.DataPop()
	if err != nil {
		return err
	}
	id, err := idObj.IntValue()
	if err != nil {
		return err
	}
	name, err := nameObj.RNameValue()
	if err != nil {
		return err
	}
	// Look up existing entity by ID
	if existing := state.GetSession().GetEntityByID(int(id)); existing != nil {
		return state.DataPush(existing.(dtrules.Object))
	}
	// Not found — create a new entity (automatically registered by session)
	entity, err := state.GetSession().CreateEntity(name)
	if err != nil {
		return err
	}
	return state.DataPush(entity.(dtrules.Object))
}

// opCvi: ( obj -- integer ) converts to integer
func opCvi(state dtrules.State) error {
	obj, err := state.DataPop()
	if err != nil {
		return err
	}
	v, err := obj.RIntegerValue()
	if err != nil {
		return state.DataPush(dtrules.GetRNull())
	}
	return state.DataPush(v)
}

// opCvd: ( obj -- double ) converts to double. The `d` is "double"; the
// date counterpart is spelled `cvdate`. Before #694 this function was
// named opCvr and `cvd` was mis-registered to the date converter below,
// so every `set <double> = <expr>` rule was silently storing null.
func opCvd(state dtrules.State) error {
	obj, err := state.DataPop()
	if err != nil {
		return err
	}
	v, err := obj.RDoubleValue()
	if err != nil {
		return state.DataPush(dtrules.GetRNull())
	}
	return state.DataPush(v)
}

// opCvb: ( obj -- boolean ) converts to boolean
// opCvb: ( obj -- boolean ) coerces to boolean with strict rules:
//   - RBoolean: passthrough
//   - string: case-insensitive "true"/"false"/"yes"/"no"/"y"/"n"/"t"/"f"/"0"/"1" → bool; else error
//   - numeric (int / bigint / double): 0 → false, non-zero → true
//   - any other type: error
//
// Replaces the older behavior that silently returned null on any coercion
// failure. `(boolean) <expr>` now fails loudly on a value it can't interpret
// rather than producing a mystery null downstream.
func opCvb(state dtrules.State) error {
	obj, err := state.DataPop()
	if err != nil {
		return err
	}
	t := obj.Type()
	switch t {
	case dtrules.TypeBoolean:
		return state.DataPush(obj)
	case dtrules.TypeString:
		s := strings.ToLower(strings.TrimSpace(obj.StringValue()))
		switch s {
		case "true", "yes", "y", "t", "1":
			return state.DataPush(dtrules.True)
		case "false", "no", "n", "f", "0":
			return state.DataPush(dtrules.False)
		}
		return dtrules.TypeCheckError("cvb", "cannot coerce string "+obj.StringValue()+" to boolean")
	case dtrules.TypeInteger:
		i, err := obj.LongValue()
		if err != nil {
			return err
		}
		return state.DataPush(dtrules.GetRBoolean(i != 0))
	case dtrules.TypeDouble:
		d, err := obj.DoubleValue()
		if err != nil {
			return err
		}
		return state.DataPush(dtrules.GetRBoolean(d != 0))
	case dtrules.TypeBigInt:
		b, err := obj.RBigIntValue()
		if err != nil {
			return err
		}
		return state.DataPush(dtrules.GetRBoolean(b.BigIntValue().Sign() != 0))
	case dtrules.TypeFixed:
		v, err := obj.BooleanValue()
		if err != nil {
			return err
		}
		return state.DataPush(dtrules.GetRBoolean(v))
	}
	return dtrules.TypeCheckError("cvb", "cannot coerce "+t.GetName().StringValue()+" to boolean")
}

// opCve: ( obj -- entity ) converts to entity
func opCve(state dtrules.State) error {
	obj, err := state.DataPop()
	if err != nil {
		return err
	}
	v, err := obj.REntityValue()
	if err != nil {
		return state.DataPush(dtrules.GetRNull())
	}
	return state.DataPush(v.(dtrules.Object))
}

// opCvn: ( obj -- name ) converts to name
func opCvn(state dtrules.State) error {
	obj, err := state.DataPop()
	if err != nil {
		return err
	}
	v, err := obj.RNameValue()
	if err != nil {
		return state.DataPush(dtrules.GetRNull())
	}
	return state.DataPush(v)
}

// opCvDate: ( obj -- date ) converts to date. Renamed from opCvd per
// #694; the symbol `cvd` now refers to the double converter above.
func opCvDate(state dtrules.State) error {
	obj, err := state.DataPop()
	if err != nil {
		return err
	}
	v, err := obj.RTimeValue()
	if err != nil {
		// Try parsing as string
		str := obj.StringValue()
		date, err := dtrules.GetRDate(state.GetSession(), str)
		if err != nil {
			return state.DataPush(dtrules.GetRNull())
		}
		return state.DataPush(date)
	}
	return state.DataPush(v)
}

// opError: ( exception message -- ) throws an error
func opError(state dtrules.State) error {
	msgObj, err := state.DataPop()
	if err != nil {
		return err
	}
	excObj, err := state.DataPop()
	if err != nil {
		return err
	}
	return dtrules.NewRulesError(excObj.StringValue(), "User Exception", msgObj.StringValue())
}

// opELStmtError: ( string -- ) halts execution with an ELStatementError.
// Used by the `error <strexpr>` EL statement; runtime dispatch uses OpError.
func opELStmtError(state dtrules.State) error {
	msgObj, err := state.DataPop()
	if err != nil {
		return err
	}
	return dtrules.NewELStatementError(msgObj.StringValue())
}

// opELStmtWarn: ( string -- ) logs a non-fatal warning and continues.
// Used by the `warn <strexpr>` EL statement; runtime dispatch uses OpWarn.
func opELStmtWarn(state dtrules.State) error {
	msgObj, err := state.DataPop()
	if err != nil {
		return err
	}
	// TODO: wire to a real logger-injection point (see future issue)
	log.Printf("[warn] %s", msgObj.StringValue())
	return nil
}
