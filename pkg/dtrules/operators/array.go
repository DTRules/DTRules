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
	"math/rand"
	"sort"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

func init() {
	Register("newarray", opNewArray)
	Register("addto", opAddTo)
	Register("addat", opAddAt)
	Register("length", opLength)
	Register("getat", opGetAt)
	Register("removeat", opRemoveAt)
	Register("remove", opRemove)
	Register("memberof", opMemberOf)
	Register("copy", opCopy)
	Register("first", opFirst)
	Register("last", opLast)
	Register("copyelements", opCopyElements)
	Register("sortarray", opSortArray)
	Register("sortentities", opSortEntities)
	Register("add_no_dups", opAddNoDups)
	Register("merge", opMerge)
	Register("randomize", opRandomize)
	Register("intersection", opIntersection)
	Register("intersects", opIntersects)
	Register("addarray", opAddArray)
	Register("deepcopy", opDeepCopy)
	Register("cleararray", opClearArray)
}

// opNewArray: ( -- array ) creates a new empty array
func opNewArray(state dtrules.State) error {
	arr, err := dtrules.NewArray(state.GetSession(), true, false)
	if err != nil {
		return err
	}
	return state.DataPush(arr)
}

// opAddTo: ( array element -- ) adds element to array
func opAddTo(state dtrules.State) error {
	element, err := state.DataPop()
	if err != nil {
		return err
	}
	arrayObj, err := state.DataPop()
	if err != nil {
		return err
	}
	arr, err := arrayObj.RArrayValue()
	if err != nil {
		return err
	}
	arr.Add(element)
	dtrules.TraceArrayAdd(state, arr, element)
	return nil
}

// opAddAt: ( array element index -- ) adds element at index
func opAddAt(state dtrules.State) error {
	indexObj, err := state.DataPop()
	if err != nil {
		return err
	}
	element, err := state.DataPop()
	if err != nil {
		return err
	}
	arrayObj, err := state.DataPop()
	if err != nil {
		return err
	}

	index, err := indexObj.IntValue()
	if err != nil {
		return err
	}
	arr, err := arrayObj.RArrayValue()
	if err != nil {
		return err
	}
	if err := arr.AddAt(index, element); err != nil {
		return err
	}
	dtrules.TraceArrayAddAt(state, arr, element, index)
	return nil
}

// opLength: ( array -- length ) returns array length
func opLength(state dtrules.State) error {
	arrayObj, err := state.DataPop()
	if err != nil {
		return err
	}
	arr, err := arrayObj.RArrayValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRIntegerValueFromInt(arr.Size()))
}

// opGetAt: ( array index -- element ) gets element at index
func opGetAt(state dtrules.State) error {
	indexObj, err := state.DataPop()
	if err != nil {
		return err
	}
	arrayObj, err := state.DataPop()
	if err != nil {
		return err
	}

	index, err := indexObj.IntValue()
	if err != nil {
		return err
	}
	arr, err := arrayObj.RArrayValue()
	if err != nil {
		return err
	}

	element, err := arr.Get(index)
	if err != nil {
		return err
	}
	return state.DataPush(element)
}

// opRemoveAt: ( array index -- ) removes element at index
func opRemoveAt(state dtrules.State) error {
	indexObj, err := state.DataPop()
	if err != nil {
		return err
	}
	arrayObj, err := state.DataPop()
	if err != nil {
		return err
	}

	index, err := indexObj.IntValue()
	if err != nil {
		return err
	}
	arr, err := arrayObj.RArrayValue()
	if err != nil {
		return err
	}
	arr.Delete(index)
	dtrules.TraceArrayRemoveAt(state, arr, index)
	return nil
}

// opRemove: ( array element -- ) removes first occurrence of element
func opRemove(state dtrules.State) error {
	element, err := state.DataPop()
	if err != nil {
		return err
	}
	arrayObj, err := state.DataPop()
	if err != nil {
		return err
	}
	arr, err := arrayObj.RArrayValue()
	if err != nil {
		return err
	}
	arr.Remove(element)
	dtrules.TraceArrayRemove(state, arr, element)
	return nil
}

// opMemberOf: ( array element -- boolean ) checks if element is in array
func opMemberOf(state dtrules.State) error {
	element, err := state.DataPop()
	if err != nil {
		return err
	}
	arrayObj, err := state.DataPop()
	if err != nil {
		return err
	}
	arr, err := arrayObj.RArrayValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRBoolean(arr.Contains(element)))
}

// opCopy: ( array -- newarray ) creates a shallow copy of array
func opCopy(state dtrules.State) error {
	arrayObj, err := state.DataPop()
	if err != nil {
		return err
	}
	arr, err := arrayObj.RArrayValue()
	if err != nil {
		return err
	}
	copied, err := arr.Clone(state.GetSession())
	if err != nil {
		return err
	}
	return state.DataPush(copied)
}

// opFirst: ( array -- element ) returns first element
func opFirst(state dtrules.State) error {
	arrayObj, err := state.DataPop()
	if err != nil {
		return err
	}
	arr, err := arrayObj.RArrayValue()
	if err != nil {
		return err
	}
	if arr.Size() == 0 {
		return state.DataPush(dtrules.GetRNull())
	}
	element, err := arr.Get(0)
	if err != nil {
		return err
	}
	return state.DataPush(element)
}

// opLast: ( array -- element ) returns last element
func opLast(state dtrules.State) error {
	arrayObj, err := state.DataPop()
	if err != nil {
		return err
	}
	arr, err := arrayObj.RArrayValue()
	if err != nil {
		return err
	}
	size := arr.Size()
	if size == 0 {
		return state.DataPush(dtrules.GetRNull())
	}
	element, err := arr.Get(size - 1)
	if err != nil {
		return err
	}
	return state.DataPush(element)
}

// opCopyElements: ( destarray srcarray -- ) copies elements from src to dest
func opCopyElements(state dtrules.State) error {
	srcObj, err := state.DataPop()
	if err != nil {
		return err
	}
	destObj, err := state.DataPop()
	if err != nil {
		return err
	}
	src, err := srcObj.RArrayValue()
	if err != nil {
		return err
	}
	dest, err := destObj.RArrayValue()
	if err != nil {
		return err
	}

	for _, elem := range src.GetIterator() {
		dest.Add(elem)
	}
	return nil
}

// opSortArray: ( array boolean -- ) sorts array elements in place.
// asc=true → ascending, asc=false → descending.
func opSortArray(state dtrules.State) error {
	ascObj, err := state.DataPop()
	if err != nil {
		return err
	}
	arrayObj, err := state.DataPop()
	if err != nil {
		return err
	}
	asc, err := ascObj.BooleanValue()
	if err != nil {
		return err
	}
	arr, err := arrayObj.RArrayValue()
	if err != nil {
		return err
	}

	elements := arr.GetIterator()
	var cmpErr error
	sort.SliceStable(elements, func(i, j int) bool {
		if cmpErr != nil {
			return false
		}
		cmp, err := elements[i].Compare(elements[j])
		if err != nil {
			cmpErr = err
			return false
		}
		if asc {
			return cmp < 0
		}
		return cmp > 0
	})
	return cmpErr
}

// opSortEntities: ( array field boolean -- ) sorts entities by field value in place.
// asc=true → ascending, asc=false → descending.
func opSortEntities(state dtrules.State) error {
	ascObj, err := state.DataPop()
	if err != nil {
		return err
	}
	nameObj, err := state.DataPop()
	if err != nil {
		return err
	}
	arrayObj, err := state.DataPop()
	if err != nil {
		return err
	}

	asc, err := ascObj.BooleanValue()
	if err != nil {
		return err
	}
	name, err := nameObj.RNameValue()
	if err != nil {
		return err
	}
	arr, err := arrayObj.RArrayValue()
	if err != nil {
		return err
	}

	elements := arr.GetIterator()
	var cmpErr error
	sort.SliceStable(elements, func(i, j int) bool {
		if cmpErr != nil {
			return false
		}
		e1, err := elements[i].REntityValue()
		if err != nil {
			cmpErr = err
			return false
		}
		e2, err := elements[j].REntityValue()
		if err != nil {
			cmpErr = err
			return false
		}
		v1, err := e1.Get(name)
		if err != nil {
			cmpErr = err
			return false
		}
		v2, err := e2.Get(name)
		if err != nil {
			cmpErr = err
			return false
		}
		cmp, err := v1.Compare(v2)
		if err != nil {
			cmpErr = err
			return false
		}
		if asc {
			return cmp < 0
		}
		return cmp > 0
	})
	return cmpErr
}

// opAddNoDups: ( array item -- ) adds item to array if not already present
func opAddNoDups(state dtrules.State) error {
	value, err := state.DataPop()
	if err != nil {
		return err
	}
	arrayObj, err := state.DataPop()
	if err != nil {
		return err
	}
	arr, err := arrayObj.RArrayValue()
	if err != nil {
		return err
	}

	if !arr.Contains(value) {
		arr.Add(value)
	}
	return nil
}

// opMerge: ( array1 array2 -- array3 ) merges two arrays into new array
func opMerge(state dtrules.State) error {
	array2Obj, err := state.DataPop()
	if err != nil {
		return err
	}
	array1Obj, err := state.DataPop()
	if err != nil {
		return err
	}
	arr1, err := array1Obj.RArrayValue()
	if err != nil {
		return err
	}
	arr2, err := array2Obj.RArrayValue()
	if err != nil {
		return err
	}

	newArr, err := dtrules.NewArray(state.GetSession(), false, false)
	if err != nil {
		return err
	}

	for _, elem := range arr1.GetIterator() {
		newArr.Add(elem)
	}
	for _, elem := range arr2.GetIterator() {
		newArr.Add(elem)
	}

	return state.DataPush(newArr)
}

// opRandomize: ( array -- ) randomizes the order of elements in the array
func opRandomize(state dtrules.State) error {
	arrayObj, err := state.DataPop()
	if err != nil {
		return err
	}
	arr, err := arrayObj.RArrayValue()
	if err != nil {
		return err
	}

	elements := arr.GetIterator()
	size := len(elements)
	for i := 0; i < 10; i++ {
		for j := 0; j < size; j++ {
			x := rand.Intn(size)
			elements[j], elements[x] = elements[x], elements[j]
		}
	}
	return nil
}

// opIntersection: ( array1 array2 -- array3 ) returns intersection of arrays
func opIntersection(state dtrules.State) error {
	array1Obj, err := state.DataPop()
	if err != nil {
		return err
	}
	array2Obj, err := state.DataPop()
	if err != nil {
		return err
	}
	arr1, err := array1Obj.RArrayValue()
	if err != nil {
		return err
	}
	arr2, err := array2Obj.RArrayValue()
	if err != nil {
		return err
	}

	result, err := dtrules.NewArray(state.GetSession(), false, false)
	if err != nil {
		return err
	}

	for _, v := range arr1.GetIterator() {
		if arr2.Contains(v) {
			result.Add(v)
		}
	}

	return state.DataPush(result)
}

// opIntersects: ( array1 array2 -- boolean ) returns true if arrays have common elements
func opIntersects(state dtrules.State) error {
	array1Obj, err := state.DataPop()
	if err != nil {
		return err
	}
	array2Obj, err := state.DataPop()
	if err != nil {
		return err
	}
	arr1, err := array1Obj.RArrayValue()
	if err != nil {
		return err
	}
	arr2, err := array2Obj.RArrayValue()
	if err != nil {
		return err
	}

	for _, v := range arr1.GetIterator() {
		if arr2.Contains(v) {
			return state.DataPush(dtrules.GetRBoolean(true))
		}
	}

	return state.DataPush(dtrules.GetRBoolean(false))
}

// opAddArray: ( array1 array2 boolean -- ) adds elements of array1 to array2
// If boolean is true, duplicates are allowed; if false, no duplicates added
func opAddArray(state dtrules.State) error {
	dupsObj, err := state.DataPop()
	if err != nil {
		return err
	}
	a2Obj, err := state.DataPop()
	if err != nil {
		return err
	}
	a1Obj, err := state.DataPop()
	if err != nil {
		return err
	}

	dups, err := dupsObj.BooleanValue()
	if err != nil {
		return err
	}
	a1, err := a1Obj.RArrayValue()
	if err != nil {
		return err
	}
	a2, err := a2Obj.RArrayValue()
	if err != nil {
		return err
	}

	for _, o := range a1.GetIterator() {
		if dups || !a2.Contains(o) {
			a2.Add(o)
		}
	}
	return nil
}

// opDeepCopy: ( array -- newarray ) deep copies array and all elements
func opDeepCopy(state dtrules.State) error {
	arrayObj, err := state.DataPop()
	if err != nil {
		return err
	}
	arr, err := arrayObj.RArrayValue()
	if err != nil {
		return err
	}

	newArr, err := dtrules.NewArray(state.GetSession(), arr.IsExecutable(), false)
	if err != nil {
		return err
	}

	for _, elem := range arr.GetIterator() {
		cloned, err := elem.Clone(state.GetSession())
		if err != nil {
			return err
		}
		newArr.Add(cloned)
	}

	return state.DataPush(newArr)
}

// opClearArray: ( array -- ) removes all elements from array
func opClearArray(state dtrules.State) error {
	arrayObj, err := state.DataPop()
	if err != nil {
		return err
	}
	arr, err := arrayObj.RArrayValue()
	if err != nil {
		return err
	}
	arr.Clear()
	return nil
}
