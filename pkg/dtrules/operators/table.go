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
	"github.com/DTRules/DTRules/pkg/dtrules"
)

func init() {
	Register("newtable", opNewTable)
	Register("tableget", opTableGet)
	Alias("tableget", "tget")
	Register("tableput", opTablePut)
	Alias("tableput", "tput")
	Register("tablekeys", opTableKeys)
	Register("tablevalues", opTableValues)
	Register("tablecontains", opTableContains)
	Register("tableremove", opTableRemove)
	Register("tablesize", opTableSize)
}

// opNewTable: ( -- table ) creates a new empty table
func opNewTable(state dtrules.State) error {
	table, err := dtrules.NewRTable(state.GetSession().GetEntityFactory(), nil, "")
	if err != nil {
		return err
	}
	return state.DataPush(table)
}

// opTableGet: ( table key -- value ) gets value for key from table
func opTableGet(state dtrules.State) error {
	keyObj, err := state.DataPop()
	if err != nil {
		return err
	}
	tableObj, err := state.DataPop()
	if err != nil {
		return err
	}

	table, err := tableObj.RTableValue()
	if err != nil {
		return err
	}

	value, err := table.GetValue(keyObj)
	if err != nil {
		return err
	}
	return state.DataPush(value)
}

// opTablePut: ( table key value -- ) puts value for key in table
func opTablePut(state dtrules.State) error {
	value, err := state.DataPop()
	if err != nil {
		return err
	}
	keyObj, err := state.DataPop()
	if err != nil {
		return err
	}
	tableObj, err := state.DataPop()
	if err != nil {
		return err
	}

	table, err := tableObj.RTableValue()
	if err != nil {
		return err
	}

	return table.SetValue(keyObj, value)
}

// opTableKeys: ( table -- array ) returns array of keys
func opTableKeys(state dtrules.State) error {
	tableObj, err := state.DataPop()
	if err != nil {
		return err
	}

	table, err := tableObj.RTableValue()
	if err != nil {
		return err
	}

	keys, err := table.GetKeys(state.GetSession())
	if err != nil {
		return err
	}
	return state.DataPush(keys)
}

// opTableValues: ( table -- array ) returns array of values
func opTableValues(state dtrules.State) error {
	tableObj, err := state.DataPop()
	if err != nil {
		return err
	}

	table, err := tableObj.RTableValue()
	if err != nil {
		return err
	}

	// Get the raw map and extract values
	rawMap, err := table.TableValue()
	if err != nil {
		return err
	}
	values := make([]dtrules.Object, 0, len(rawMap))
	for _, v := range rawMap {
		values = append(values, v)
	}
	arr, err := dtrules.NewArrayWithElements(state.GetSession(), false, values, false)
	if err != nil {
		return err
	}
	return state.DataPush(arr)
}

// opTableContains: ( table key -- boolean ) checks if key exists in table
func opTableContains(state dtrules.State) error {
	keyObj, err := state.DataPop()
	if err != nil {
		return err
	}
	tableObj, err := state.DataPop()
	if err != nil {
		return err
	}

	table, err := tableObj.RTableValue()
	if err != nil {
		return err
	}

	contains := table.ContainsKey(keyObj)
	return state.DataPush(dtrules.GetRBoolean(contains))
}

// opTableRemove: ( table key -- ) removes key from table
func opTableRemove(state dtrules.State) error {
	keyObj, err := state.DataPop()
	if err != nil {
		return err
	}
	tableObj, err := state.DataPop()
	if err != nil {
		return err
	}

	table, err := tableObj.RTableValue()
	if err != nil {
		return err
	}

	// Get the value first, then set to null to effectively remove
	// (The RTable implementation doesn't have a Remove method,
	// but we can set to null which achieves similar behavior)
	return table.SetValue(keyObj, dtrules.GetRNull())
}

// opTableSize: ( table -- size ) returns number of entries in table
func opTableSize(state dtrules.State) error {
	tableObj, err := state.DataPop()
	if err != nil {
		return err
	}

	table, err := tableObj.RTableValue()
	if err != nil {
		return err
	}

	return state.DataPush(dtrules.GetRIntegerValueFromInt(table.Size()))
}
