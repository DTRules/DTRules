// Copyright 2026 Paul Snow
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
	"math/bits"
	"sort"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// The combinatorial primitives (#980). Each generator walks a source array
// of entities, discovers structures of a particular shape — subsets, key
// groups, consecutive runs — and materializes every structure as an entity
// of a caller-named EDD type appended to a destination array. Decision
// tables then iterate the materialized entities with ordinary `for all`
// contexts and score them with ordinary conditions: the combinatorial loop
// stays inside the operator, the policy stays in the table.
//
// All four are statement-form EL calls, e.g.
//
//	subsets(hand.cards, "combo", "value", hand.combos)
//
// which compiles to `hand.cards "combo" "value" hand.combos subsets`.

func init() {
	// Arity is recorded because these are the operators a short call misreads
	// silently: every argument after the source is a bare string or an array,
	// so `subsets(hand.cards)` pops whatever three values sit beneath it and
	// treats them as typename, sumfield and destination (#1105).
	RegisterWithArity("combinations", opCombinations, 5) // source, k, typename, sumfield, dest
	RegisterWithArity("subsets", opSubsets, 4)           // source, typename, sumfield, dest
	RegisterWithArity("groupby", opGroupBy, 4)           // source, keyfield, typename, dest
	RegisterWithArity("maximalruns", opMaximalRuns, 5)   // source, rankfield, minlen, typename, dest
	RegisterWithArity("suffixes", opSuffixes, 5)         // source, minlen, statfield, typename, dest
}

// subsetsCap bounds opSubsets: 2^12-1 = 4095 entities is the design ceiling.
const subsetsCap = 12

// combinationsCap bounds opCombinations' source size.
const combinationsCap = 20

// suffixesCap bounds opSuffixes' source size. Windows are linear in count
// but quadratic in total members (n windows share O(n²) member slots), and
// the family contract (#980) is a clear error, never an OOM. Pegging stacks
// peak at 13 cards; 64 leaves room for temporal streams like month series.
const suffixesCap = 64

// popEntityArray pops an object and unwraps it to a data-form RArray whose
// elements are all entities, returned alongside the raw array.
func popEntityArray(state dtrules.State, op string) (*dtrules.RArray, []dtrules.Entity, error) {
	obj, err := state.DataPop()
	if err != nil {
		return nil, nil, err
	}
	arr, err := obj.RArrayValue()
	if err != nil {
		return nil, nil, fmt.Errorf("%s: source must be an array: %w", op, err)
	}
	elems, err := arr.ArrayValue()
	if err != nil {
		return nil, nil, fmt.Errorf("%s: source array: %w", op, err)
	}
	ents := make([]dtrules.Entity, len(elems))
	for i, e := range elems {
		ent, err := e.REntityValue()
		if err != nil {
			return nil, nil, fmt.Errorf("%s: source element %d is not an entity: %w", op, i, err)
		}
		ents[i] = ent
	}
	return arr, ents, nil
}

// popString pops a string-valued argument.
func popString(state dtrules.State, op, what string) (string, error) {
	obj, err := state.DataPop()
	if err != nil {
		return "", fmt.Errorf("%s: missing %s argument: %w", op, what, err)
	}
	return obj.StringValue(), nil
}

// popInt pops an integer-valued argument.
func popInt(state dtrules.State, op, what string) (int, error) {
	obj, err := state.DataPop()
	if err != nil {
		return 0, err
	}
	v, err := obj.IntValue()
	if err != nil {
		return 0, fmt.Errorf("%s: %s must be an integer: %w", op, what, err)
	}
	return v, nil
}

// intField reads an integer attribute from an entity. Entity.Get reports a
// missing attribute as (nil, nil), so both paths are the same error here.
func intField(op string, ent dtrules.Entity, field *dtrules.RName) (int, error) {
	obj, err := ent.Get(field)
	if err != nil || obj == nil {
		return 0, fmt.Errorf("%s: entity %s has no attribute %q", op, ent.GetName().String(), field.String())
	}
	v, err := obj.IntValue()
	if err != nil {
		return 0, fmt.Errorf("%s: attribute %q is not an integer: %w", op, field.String(), err)
	}
	return v, nil
}

// structField is one stamped attribute of a materialized-structure entity.
type structField struct {
	name  string
	value dtrules.Object
}

// makeStructEntity creates one materialized-structure entity of the named
// EDD type and stamps the given fields, verifying each is declared. Fields
// arrive as an ordered slice, not a map: the generators run once per
// discovered structure — 31 times for a five-card subsets call — so the
// per-call map allocation was measurable (#1025), and ordered stamping keeps
// traces deterministic.
func makeStructEntity(state dtrules.State, op string, typeName *dtrules.RName, fields ...structField) (dtrules.Entity, error) {
	sess := state.GetSession()
	ent, err := sess.GetEntityFactory().CreateEntity(sess, typeName)
	if err != nil {
		return nil, fmt.Errorf("%s: cannot create entity of EDD type %q: %w", op, typeName.String(), err)
	}
	for _, f := range fields {
		rn := dtrules.GetRName(f.name)
		if !ent.ContainsAttribute(rn) {
			return nil, fmt.Errorf("%s: EDD type %q must declare attribute %q", op, typeName.String(), f.name)
		}
		if err := ent.Put(rn, f.value); err != nil {
			return nil, fmt.Errorf("%s: setting %s.%s: %w", op, typeName.String(), f.name, err)
		}
	}
	return ent, nil
}

// emitCombo materializes one subset as a combo entity and appends it to dest.
func emitCombo(state dtrules.State, op string, members []dtrules.Object, typeName *dtrules.RName, sumField string, dest *dtrules.RArray) error {
	sum := 0
	if sumField != "" {
		fieldName := dtrules.GetRName(sumField)
		for _, m := range members {
			ent, _ := m.REntityValue()
			v, err := intField(op, ent, fieldName)
			if err != nil {
				return err
			}
			sum += v
		}
	}
	membersArr, err := dtrules.NewArrayWithElements(state.GetSession(), true, members, false)
	if err != nil {
		return err
	}
	ent, err := makeStructEntity(state, op, typeName,
		structField{"members", membersArr},
		structField{"count", dtrules.GetRIntegerValue(int64(len(members)))},
		structField{"sum", dtrules.GetRIntegerValue(int64(sum))})
	if err != nil {
		return err
	}
	dest.Add(ent)
	dtrules.TraceArrayAdd(state, dest, ent)
	return nil
}

// opCombinations: ( src k typename sumfield dest -- ) for every k-element
// combination of the entities in src, create an entity of EDD type
// `typename` with fields members (the k source entities, by reference),
// count (= k), and sum (Σ member.<sumfield>, or 0 when sumfield is ""), and
// append it to dest. k = 0 or k > len(src) appends nothing. Sources larger
// than combinationsCap are an error, not an OOM. The field is `count`, not
// `size`: SIZE is an EL keyword, so a bare `size` reference will not parse.
func opCombinations(state dtrules.State) error {
	const op = "combinations"
	destObj, err := state.DataPop()
	if err != nil {
		return err
	}
	dest, err := destObj.RArrayValue()
	if err != nil {
		return fmt.Errorf("%s: dest must be an array: %w", op, err)
	}
	sumField, err := popString(state, op, "sumfield")
	if err != nil {
		return err
	}
	typeName, err := popRName(state, op, "typename")
	if err != nil {
		return err
	}
	k, err := popInt(state, op, "k")
	if err != nil {
		return err
	}
	src, ents, err := popEntityArray(state, op)
	if err != nil {
		return err
	}
	_ = src
	n := len(ents)
	if n > combinationsCap {
		return fmt.Errorf("%s: source has %d elements; the cap is %d", op, n, combinationsCap)
	}
	if k <= 0 || k > n {
		return nil
	}

	idx := make([]int, k)
	for i := range idx {
		idx[i] = i
	}
	for {
		members := make([]dtrules.Object, k)
		for i, j := range idx {
			members[i] = ents[j].(dtrules.Object)
		}
		if err := emitCombo(state, op, members, typeName, sumField, dest); err != nil {
			return err
		}
		// advance the combination indices
		i := k - 1
		for i >= 0 && idx[i] == n-k+i {
			i--
		}
		if i < 0 {
			break
		}
		idx[i]++
		for j := i + 1; j < k; j++ {
			idx[j] = idx[j-1] + 1
		}
	}
	return nil
}

// opSubsets: ( src typename sumfield dest -- ) like combinations, for every
// non-empty subset of src: 2^n − 1 entities. Sources larger than subsetsCap
// are an error by design.
func opSubsets(state dtrules.State) error {
	const op = "subsets"
	destObj, err := state.DataPop()
	if err != nil {
		return err
	}
	dest, err := destObj.RArrayValue()
	if err != nil {
		return fmt.Errorf("%s: dest must be an array: %w", op, err)
	}
	sumField, err := popString(state, op, "sumfield")
	if err != nil {
		return err
	}
	typeName, err := popRName(state, op, "typename")
	if err != nil {
		return err
	}
	_, ents, err := popEntityArray(state, op)
	if err != nil {
		return err
	}
	n := len(ents)
	if n > subsetsCap {
		return fmt.Errorf("%s: source has %d elements; the cap is %d (2^%d−1 subsets)", op, n, subsetsCap, subsetsCap)
	}
	for mask := 1; mask < 1<<n; mask++ {
		members := make([]dtrules.Object, 0, bits.OnesCount(uint(mask)))
		for i := 0; i < n; i++ {
			if mask&(1<<i) != 0 {
				members = append(members, ents[i].(dtrules.Object))
			}
		}
		if err := emitCombo(state, op, members, typeName, sumField, dest); err != nil {
			return err
		}
	}
	return nil
}

// opGroupBy: ( src keyfield typename dest -- ) partition the entities of
// src by the integer value of their `keyfield` attribute. For each distinct
// key, in first-seen order, create an entity of EDD type `typename` with
// fields key, count, and members (the sharing entities, by reference), and
// append it to dest. First-seen order is part of the contract: it keeps
// table traces deterministic.
func opGroupBy(state dtrules.State) error {
	const op = "groupby"
	destObj, err := state.DataPop()
	if err != nil {
		return err
	}
	dest, err := destObj.RArrayValue()
	if err != nil {
		return fmt.Errorf("%s: dest must be an array: %w", op, err)
	}
	typeName, err := popRName(state, op, "typename")
	if err != nil {
		return err
	}
	keyField, err := popString(state, op, "keyfield")
	if err != nil {
		return err
	}
	_, ents, err := popEntityArray(state, op)
	if err != nil {
		return err
	}

	fieldName := dtrules.GetRName(keyField)
	order := make([]int, 0, len(ents))
	groups := make(map[int][]dtrules.Object)
	for _, ent := range ents {
		key, err := intField(op, ent, fieldName)
		if err != nil {
			return err
		}
		if _, seen := groups[key]; !seen {
			order = append(order, key)
		}
		groups[key] = append(groups[key], ent.(dtrules.Object))
	}
	for _, key := range order {
		members := groups[key]
		membersArr, err := dtrules.NewArrayWithElements(state.GetSession(), true, members, false)
		if err != nil {
			return err
		}
		ent, err := makeStructEntity(state, op, typeName,
			structField{"key", dtrules.GetRIntegerValue(int64(key))},
			structField{"count", dtrules.GetRIntegerValue(int64(len(members)))},
			structField{"members", membersArr})
		if err != nil {
			return err
		}
		dest.Add(ent)
		dtrules.TraceArrayAdd(state, dest, ent)
	}
	return nil
}

// opMaximalRuns: ( src rankfield minlen typename dest -- ) scan the multiset
// of `rankfield` values in src, order-independent. For every maximal
// interval of consecutive values all present at least once, with length ≥
// minlen, create an entity of EDD type `typename` with fields start (lowest
// value), span (number of consecutive values), and multiplicity (product
// of the value counts — 1 = single run, 2 = double run, 4 = double-double),
// and append it to dest. The field is `span`, not `length`: LENGTH is an EL
// keyword, so a bare `length` reference will not parse.
func opMaximalRuns(state dtrules.State) error {
	const op = "maximalruns"
	destObj, err := state.DataPop()
	if err != nil {
		return err
	}
	dest, err := destObj.RArrayValue()
	if err != nil {
		return fmt.Errorf("%s: dest must be an array: %w", op, err)
	}
	typeName, err := popRName(state, op, "typename")
	if err != nil {
		return err
	}
	minLen, err := popInt(state, op, "minlen")
	if err != nil {
		return err
	}
	rankField, err := popString(state, op, "rankfield")
	if err != nil {
		return err
	}
	_, ents, err := popEntityArray(state, op)
	if err != nil {
		return err
	}

	fieldName := dtrules.GetRName(rankField)
	counts := make(map[int]int)
	for _, ent := range ents {
		v, err := intField(op, ent, fieldName)
		if err != nil {
			return err
		}
		counts[v]++
	}
	ranks := make([]int, 0, len(counts))
	for r := range counts {
		ranks = append(ranks, r)
	}
	sort.Ints(ranks)

	for i := 0; i < len(ranks); {
		j := i
		mult := counts[ranks[i]]
		for j+1 < len(ranks) && ranks[j+1] == ranks[j]+1 {
			j++
			mult *= counts[ranks[j]]
		}
		length := j - i + 1
		if length >= minLen {
			ent, err := makeStructEntity(state, op, typeName,
				structField{"start", dtrules.GetRIntegerValue(int64(ranks[i]))},
				structField{"span", dtrules.GetRIntegerValue(int64(length))},
				structField{"multiplicity", dtrules.GetRIntegerValue(int64(mult))})
			if err != nil {
				return err
			}
			dest.Add(ent)
			dtrules.TraceArrayAdd(state, dest, ent)
		}
		i = j + 1
	}
	return nil
}

// opSuffixes: ( src minlen statfield typename dest -- ) for every trailing
// window of src with length ≥ minlen — the last 2 elements, the last 3, … —
// create an entity of EDD type `typename` with fields
//
//	members  : the window's entities, in source order, by reference
//	count    : window length
//	sum      : Σ member.<statfield>
//	distinct : number of distinct <statfield> values in the window
//	spread   : max − min of <statfield> over the window
//
// and append it to dest, LONGEST WINDOW FIRST. The emission order is part
// of the contract: order-dependent policies like cribbage's longest-run-
// only rule read as "the first qualifying window" — a plain zero-guard
// condition in a table iterating the destination (#1023).
//
// The window shape makes order-dependent structure testable with plain
// conditions: a window is a run iff distinct == count and spread ==
// count−1 (any lay order), and a trailing pair block iff distinct == 1.
func opSuffixes(state dtrules.State) error {
	const op = "suffixes"
	destObj, err := state.DataPop()
	if err != nil {
		return err
	}
	dest, err := destObj.RArrayValue()
	if err != nil {
		return fmt.Errorf("%s: dest must be an array: %w", op, err)
	}
	typeName, err := popRName(state, op, "typename")
	if err != nil {
		return err
	}
	statField, err := popString(state, op, "statfield")
	if err != nil {
		return err
	}
	if statField == "" {
		return fmt.Errorf("%s: statfield must name an integer attribute", op)
	}
	minLen, err := popInt(state, op, "minlen")
	if err != nil {
		return err
	}
	if minLen < 1 {
		minLen = 1
	}
	_, ents, err := popEntityArray(state, op)
	if err != nil {
		return err
	}
	if len(ents) > suffixesCap {
		return fmt.Errorf("%s: source has %d elements; the cap is %d", op, len(ents), suffixesCap)
	}

	fieldName := dtrules.GetRName(statField)
	vals := make([]int, len(ents))
	for i, ent := range ents {
		v, err := intField(op, ent, fieldName)
		if err != nil {
			return err
		}
		vals[i] = v
	}

	var distinctBuf [suffixesCap]int
	for l := len(ents); l >= minLen; l-- {
		start := len(ents) - l
		sum, mn, mx := 0, vals[start], vals[start]
		distinct := 0
		members := make([]dtrules.Object, 0, l)
		for i := start; i < len(ents); i++ {
			members = append(members, ents[i].(dtrules.Object))
			sum += vals[i]
			fresh := true
			for _, v := range distinctBuf[:distinct] {
				if v == vals[i] {
					fresh = false
					break
				}
			}
			if fresh {
				distinctBuf[distinct] = vals[i]
				distinct++
			}
			if vals[i] < mn {
				mn = vals[i]
			}
			if vals[i] > mx {
				mx = vals[i]
			}
		}
		membersArr, err := dtrules.NewArrayWithElements(state.GetSession(), true, members, false)
		if err != nil {
			return err
		}
		ent, err := makeStructEntity(state, op, typeName,
			structField{"members", membersArr},
			structField{"count", dtrules.GetRIntegerValue(int64(l))},
			structField{"sum", dtrules.GetRIntegerValue(int64(sum))},
			structField{"distinct", dtrules.GetRIntegerValue(int64(distinct))},
			structField{"spread", dtrules.GetRIntegerValue(int64(mx - mn))})
		if err != nil {
			return err
		}
		dest.Add(ent)
		dtrules.TraceArrayAdd(state, dest, ent)
	}
	return nil
}

// popRName pops an argument usable as an EDD type name.
func popRName(state dtrules.State, op, what string) (*dtrules.RName, error) {
	obj, err := state.DataPop()
	if err != nil {
		return nil, err
	}
	name, err := obj.RNameValue()
	if err != nil {
		return nil, fmt.Errorf("%s: %s must name an EDD type: %w", op, what, err)
	}
	return name, nil
}
