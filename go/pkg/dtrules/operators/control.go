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
	"fmt"

	"github.com/DTRules/DTRules/go/pkg/dtrules"
)

// DefaultMaxIterations is the maximum iterations for loop operators.
// This prevents infinite loops from consuming unbounded CPU.
const DefaultMaxIterations = 1000000

// ErrMaxIterationsExceeded returns an error indicating the loop exceeded the maximum iterations.
func ErrMaxIterationsExceeded(limit int) error {
	return fmt.Errorf("loop exceeded maximum iterations (%d)", limit)
}

func init() {
	Register("if", opIf)
	Register("ifelse", opIfelse)
	Register("while", opWhile)
	Register("execute", opExecute)
	Register("for", opFor)
	Register("forr", opForr)
	Register("forall", opForall)
	Register("forallr", opForallr)
	Register("doloop", opDoloop)
	Register("lookup", opLookup)
	Register("throwexception", opThrowException)
	Register("forfirst", opForFirst)
	Register("forfirstelse", opForFirstElse)
	Register("entityforall", opEntityForAll)
	Register("allocate", opAllocate)
	Alias("allocate", "cpush")
	Register("deallocate", opDeallocate)
	Alias("deallocate", "cpop")
	Register("local@", opLocalFetch)
	Register("local!", opLocalStore)
	Register("ignore", opIgnore)
	Alias("ignore", "nop")
	Alias("ignore", "performaliased")
	Register("executetable", opExecuteTable)
	Register("performcatcherror", opPerformCatchError)
}

// opIf: ( body boolean -- ) executes body if boolean is true
func opIf(state dtrules.State) error {
	testObj, err := state.DataPop()
	if err != nil {
		return err
	}
	body, err := state.DataPop()
	if err != nil {
		return err
	}
	test, err := testObj.BooleanValue()
	if err != nil {
		return err
	}
	if test {
		return body.Execute(state)
	}
	return nil
}

// opIfelse: ( truebody falsebody test -- ) executes truebody if test is true, else falsebody
func opIfelse(state dtrules.State) error {
	testObj, err := state.DataPop()
	if err != nil {
		return err
	}
	falsebody, err := state.DataPop()
	if err != nil {
		return err
	}
	truebody, err := state.DataPop()
	if err != nil {
		return err
	}
	test, err := testObj.BooleanValue()
	if err != nil {
		return err
	}
	if test {
		return truebody.Execute(state)
	}
	return falsebody.Execute(state)
}

// opWhile: ( body test -- ) executes body while test returns true.
// Has an iteration limit (DefaultMaxIterations = 1M) to prevent infinite loops.
func opWhile(state dtrules.State) error {
	test, err := state.DataPop()
	if err != nil {
		return err
	}
	body, err := state.DataPop()
	if err != nil {
		return err
	}

	// Evaluate test first
	if err := test.Execute(state); err != nil {
		return err
	}
	resultObj, err := state.DataPop()
	if err != nil {
		return err
	}
	result, err := resultObj.BooleanValue()
	if err != nil {
		return err
	}

	iterations := 0
	for result {
		// Check iteration limit
		iterations++
		if iterations > DefaultMaxIterations {
			return fmt.Errorf("while: %w", ErrMaxIterationsExceeded(DefaultMaxIterations))
		}

		if err := body.Execute(state); err != nil {
			return err
		}
		if err := test.Execute(state); err != nil {
			return err
		}
		resultObj, err = state.DataPop()
		if err != nil {
			return err
		}
		result, err = resultObj.BooleanValue()
		if err != nil {
			return err
		}
	}
	return nil
}

// opExecute: ( obj -- ? ) executes the object on top of data stack
func opExecute(state dtrules.State) error {
	obj, err := state.DataPop()
	if err != nil {
		return err
	}
	return obj.GetExecutable().Execute(state)
}

// opFor: ( body array -- ) pushes each element and executes body
func opFor(state dtrules.State) error {
	arrayObj, err := state.DataPop()
	if err != nil {
		return err
	}
	body, err := state.DataPop()
	if err != nil {
		return err
	}
	list, err := arrayObj.RArrayValue()
	if err != nil {
		return err
	}

	for _, obj := range list.GetIterator() {
		if err := state.DataPush(obj); err != nil {
			return err
		}
		if err := body.Execute(state); err != nil {
			return err
		}
	}
	return nil
}

// opForr: ( body array -- ) pushes each element in reverse and executes body
func opForr(state dtrules.State) error {
	arrayObj, err := state.DataPop()
	if err != nil {
		return err
	}
	body, err := state.DataPop()
	if err != nil {
		return err
	}
	arr, err := arrayObj.RArrayValue()
	if err != nil {
		return err
	}

	for i := arr.Size() - 1; i >= 0; i-- {
		elem, err := arr.Get(i)
		if err != nil {
			return err
		}
		if err := state.DataPush(elem); err != nil {
			return err
		}
		if err := body.Execute(state); err != nil {
			return err
		}
	}
	return nil
}

// opForall: ( body array -- ) executes body for each entity in array
// Entities are automatically pushed to entity stack before executing body
func opForall(state dtrules.State) error {
	arrayObj, err := state.DataPop()
	if err != nil {
		return err
	}
	body, err := state.DataPop()
	if err != nil {
		return err
	}
	arr, err := arrayObj.RArrayValue()
	if err != nil {
		return err
	}

	for _, obj := range arr.GetIterator() {
		if obj.Type() == dtrules.TypeNull {
			continue
		}
		if obj.Type() != dtrules.TypeEntity {
			return dtrules.TypeCheckError("Forall", "Encountered a non-Entity entry in array: "+obj.StringValue())
		}
		entity, err := obj.REntityValue()
		if err != nil {
			return err
		}
		state.EntityPush(entity)
		if err := body.Execute(state); err != nil {
			state.EntityPop()
			return err
		}
		state.EntityPop()
	}
	return nil
}

// opForallr: ( body array -- ) executes body for each entity in reverse order
// Entities are automatically pushed to entity stack before executing body
func opForallr(state dtrules.State) error {
	arrayObj, err := state.DataPop()
	if err != nil {
		return err
	}
	body, err := state.DataPop()
	if err != nil {
		return err
	}
	arr, err := arrayObj.RArrayValue()
	if err != nil {
		return err
	}

	list := arr.GetIterator()
	for i := len(list) - 1; i >= 0; i-- {
		obj := list[i]
		if obj.Type() == dtrules.TypeNull {
			continue
		}
		if obj.Type() != dtrules.TypeEntity {
			return dtrules.TypeCheckError("Forallr", "Encountered a non-Entity entry in array: "+obj.StringValue())
		}
		entity, err := obj.REntityValue()
		if err != nil {
			return err
		}
		state.EntityPush(entity)
		if err := body.Execute(state); err != nil {
			state.EntityPop()
			return err
		}
		state.EntityPop()
	}
	return nil
}

// opDoloop: ( body start increment limit -- )
// Has an iteration limit (DefaultMaxIterations) to prevent infinite loops.
func opDoloop(state dtrules.State) error {
	limitObj, err := state.DataPop()
	if err != nil {
		return err
	}
	incrementObj, err := state.DataPop()
	if err != nil {
		return err
	}
	startObj, err := state.DataPop()
	if err != nil {
		return err
	}
	body, err := state.DataPop()
	if err != nil {
		return err
	}

	limit, err := limitObj.IntValue()
	if err != nil {
		return err
	}
	increment, err := incrementObj.IntValue()
	if err != nil {
		return err
	}
	start, err := startObj.IntValue()
	if err != nil {
		return err
	}

	// Check for zero increment (would cause infinite loop)
	if increment == 0 {
		return dtrules.UndefinedError("doloop", "increment cannot be zero")
	}

	// Note: This uses a simplified approach without control stack frames
	// The full implementation would use the control stack for the loop variable
	iterations := 0
	if increment > 0 {
		for i := start; i < limit; i += increment {
			iterations++
			if iterations > DefaultMaxIterations {
				return fmt.Errorf("doloop: %w", ErrMaxIterationsExceeded(DefaultMaxIterations))
			}
			if err := state.DataPush(dtrules.GetRIntegerValueFromInt(i)); err != nil {
				return err
			}
			if err := body.Execute(state); err != nil {
				return err
			}
		}
	} else {
		for i := start; i > limit; i += increment {
			iterations++
			if iterations > DefaultMaxIterations {
				return fmt.Errorf("doloop: %w", ErrMaxIterationsExceeded(DefaultMaxIterations))
			}
			if err := state.DataPush(dtrules.GetRIntegerValueFromInt(i)); err != nil {
				return err
			}
			if err := body.Execute(state); err != nil {
				return err
			}
		}
	}
	return nil
}

// opLookup: ( name -- value ) looks up name and pushes value
func opLookup(state dtrules.State) error {
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
		return dtrules.UndefinedError("Lookup", "Could not find a value for "+name.StringValue()+" in the current context")
	}
	return state.DataPush(value)
}

// opThrowException: ( string -- ) throws an exception with the message
func opThrowException(state dtrules.State) error {
	value, err := state.DataPop()
	if err != nil {
		return err
	}
	return dtrules.NewRulesError("throwException", "ThrowException.execute()", value.StringValue())
}

// opForFirst: ( body test array -- ) executes body for first entity matching test
func opForFirst(state dtrules.State) error {
	arrayObj, err := state.DataPop()
	if err != nil {
		return err
	}
	test, err := state.DataPop()
	if err != nil {
		return err
	}
	body, err := state.DataPop()
	if err != nil {
		return err
	}

	arr, err := arrayObj.RArrayValue()
	if err != nil {
		return err
	}

	for _, obj := range arr.GetIterator() {
		entity, err := obj.REntityValue()
		if err != nil {
			continue
		}
		state.EntityPush(entity)
		if err := test.Execute(state); err != nil {
			state.EntityPop()
			return err
		}
		resultObj, err := state.DataPop()
		if err != nil {
			state.EntityPop()
			return err
		}
		result, err := resultObj.BooleanValue()
		if err != nil {
			state.EntityPop()
			return err
		}
		if result {
			if err := body.Execute(state); err != nil {
				state.EntityPop()
				return err
			}
			state.EntityPop()
			return nil
		}
		state.EntityPop()
	}
	return nil
}

// opForFirstElse: ( body1 body2 test array -- ) executes body1 for first match, else body2
func opForFirstElse(state dtrules.State) error {
	arrayObj, err := state.DataPop()
	if err != nil {
		return err
	}
	test, err := state.DataPop()
	if err != nil {
		return err
	}
	body2, err := state.DataPop()
	if err != nil {
		return err
	}
	body1, err := state.DataPop()
	if err != nil {
		return err
	}

	arr, err := arrayObj.RArrayValue()
	if err != nil {
		return err
	}

	for _, obj := range arr.GetIterator() {
		entity, err := obj.REntityValue()
		if err != nil {
			continue
		}
		state.EntityPush(entity)
		if err := test.Execute(state); err != nil {
			state.EntityPop()
			return err
		}
		resultObj, err := state.DataPop()
		if err != nil {
			state.EntityPop()
			return err
		}
		result, err := resultObj.BooleanValue()
		if err != nil {
			state.EntityPop()
			return err
		}
		if result {
			if err := body1.Execute(state); err != nil {
				state.EntityPop()
				return err
			}
			state.EntityPop()
			return nil
		}
		state.EntityPop()
	}

	// No match found, execute body2
	return body2.Execute(state)
}

// opEntityForAll: ( body entity -- ) executes body for each attribute in entity
func opEntityForAll(state dtrules.State) error {
	entityObj, err := state.DataPop()
	if err != nil {
		return err
	}
	body, err := state.DataPop()
	if err != nil {
		return err
	}

	entity, err := entityObj.REntityValue()
	if err != nil {
		return err
	}

	names := entity.GetAttributeNames()
	for _, name := range names {
		value, err := entity.Get(name)
		if err != nil {
			continue
		}
		if value != nil {
			if err := state.DataPush(name); err != nil {
				return err
			}
			if err := state.DataPush(value); err != nil {
				return err
			}
			if err := body.Execute(state); err != nil {
				return err
			}
		}
	}
	return nil
}

// opAllocate: ( value -- ) pushes value to control stack
func opAllocate(state dtrules.State) error {
	v, err := state.DataPop()
	if err != nil {
		return err
	}
	state.CtrlPush(v)
	return nil
}

// opDeallocate: ( -- value ) pops value from control stack to data stack
func opDeallocate(state dtrules.State) error {
	v, err := state.CtrlPop()
	if err != nil {
		return err
	}
	return state.DataPush(v)
}

// opLocalFetch: ( index -- value ) fetches value from control stack frame
func opLocalFetch(state dtrules.State) error {
	indexObj, err := state.DataPop()
	if err != nil {
		return err
	}
	index, err := indexObj.IntValue()
	if err != nil {
		return err
	}
	value, err := state.GetFrameValue(index)
	if err != nil {
		return err
	}
	return state.DataPush(value)
}

// opLocalStore: ( value index -- ) stores value in control stack frame
func opLocalStore(state dtrules.State) error {
	indexObj, err := state.DataPop()
	if err != nil {
		return err
	}
	value, err := state.DataPop()
	if err != nil {
		return err
	}
	index, err := indexObj.IntValue()
	if err != nil {
		return err
	}
	return state.SetFrameValue(index, value)
}

// opIgnore: ( -- ) does nothing (noop)
func opIgnore(state dtrules.State) error {
	return nil
}

// opExecuteTable: ( name -- ) executes a decision table's conditions and actions
// This is used by contexts to call the table's executeTable method directly,
// bypassing the context setup (which would cause infinite recursion).
func opExecuteTable(state dtrules.State) error {
	nameObj, err := state.DataPop()
	if err != nil {
		return err
	}
	name, err := nameObj.RNameValue()
	if err != nil {
		return err
	}
	// Get the decision table from the entity factory
	dtObj, err := state.GetSession().GetEntityFactory().GetDecisionTable(name)
	if err != nil {
		return err
	}
	if dtObj == nil {
		return dtrules.UndefinedError("ExecuteTable", "Decision table not found: "+name.StringValue())
	}
	// Call ExecuteTable which runs the decision tree directly without context
	return dtObj.ExecuteTable(state)
}

// opPerformCatchError: ( table error_table error_entity -- )
// Executes the main table. If a RulesException is thrown, creates an error entity,
// populates it with error details, pushes it to the entity stack, and executes
// the error handler table.
//
// The error entity is populated with the following fields (if defined):
// - errortype: The type of error (e.g., "undefined", "typecheck")
// - location: Where the error occurred
// - message: The error message
// - postfix: The postfix context if available
//
// This is critical for production rule systems that need to handle errors
// gracefully without terminating execution.
func opPerformCatchError(state dtrules.State) error {
	// Pop error_entity name
	errorEntityNameObj, err := state.DataPop()
	if err != nil {
		return err
	}
	errorEntityName, err := errorEntityNameObj.RNameValue()
	if err != nil {
		return err
	}

	// Pop error_table name
	errorTableNameObj, err := state.DataPop()
	if err != nil {
		return err
	}
	errorTableName, err := errorTableNameObj.RNameValue()
	if err != nil {
		return err
	}

	// Pop main_table name
	mainTableNameObj, err := state.DataPop()
	if err != nil {
		return err
	}
	mainTableName, err := mainTableNameObj.RNameValue()
	if err != nil {
		return err
	}

	// Look up and execute the main table
	mainTable, err := state.Find(mainTableName)
	if err != nil {
		return err
	}
	if mainTable == nil {
		return dtrules.UndefinedError("PerformCatchError",
			"The table '"+mainTableName.StringValue()+"' is undefined")
	}

	// Execute main table and catch errors
	mainErr := mainTable.Execute(state)
	if mainErr != nil {
		// Create an instance of the error entity
		session := state.GetSession()
		ef := session.GetEntityFactory()
		errorEntity, createErr := ef.CreateEntity(session, errorEntityName)
		if createErr != nil {
			// If we can't create the error entity, return the original error
			return mainErr
		}

		// Push the error entity to the entity stack
		state.EntityPush(errorEntity)

		// Populate error entity with error details
		// If any of the puts fail (because the entity doesn't define the attribute),
		// simply carry on - the user can define these fields if they need them
		populateErrorEntity(errorEntity, mainErr)

		// Look up and execute the error handler table
		errorTable, err := state.Find(errorTableName)
		if err != nil {
			// Pop the error entity since we're returning
			state.EntityPop()
			return err
		}
		if errorTable == nil {
			state.EntityPop()
			return dtrules.UndefinedError("PerformCatchError",
				"The error table '"+errorTableName.StringValue()+"' is undefined")
		}

		// Execute the error handler table
		handlerErr := errorTable.Execute(state)
		if handlerErr != nil {
			// Pop the error entity and return the handler error
			state.EntityPop()
			return handlerErr
		}

		// Leave the error entity on the entity stack for the caller to use
		// (they can pop it when done if needed)
		return nil
	}

	return nil
}

// populateErrorEntity fills in the error details on an error entity.
// Silently ignores errors from Put (attribute not defined).
func populateErrorEntity(entity dtrules.Entity, err error) {
	// Try to extract structured error info from RulesError
	var rulesErr *dtrules.RulesError
	if re, ok := err.(*dtrules.RulesError); ok {
		rulesErr = re
	}

	// Helper to safely put a string value
	putString := func(attrName string, value string) {
		_ = entity.Put(dtrules.GetRName(attrName), dtrules.NewRString(value))
	}

	if rulesErr != nil {
		// Populate from RulesError fields
		putString("errortype", rulesErr.ErrorType)
		putString("location", rulesErr.Location)
		putString("message", rulesErr.Message)
		putString("postfix", rulesErr.PostFix)
		// 'formal' is the same as errortype in Java implementation
		putString("formal", rulesErr.ErrorType)
	} else {
		// Generic error - populate what we can
		putString("errortype", "execution")
		putString("location", "")
		putString("message", err.Error())
		putString("postfix", "")
		putString("formal", "execution")
	}
}
