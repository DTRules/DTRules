// Copyright 2026 Paul Snow
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package dtrules_test

import (
	"math"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// revProc202440 is Rev. Proc. 2024-40 section 3.01's 2025 rate schedules
// (Tables 1-4): each row is "over this amount, the tax is base plus rate of
// the excess".
var revProc202440 = map[string][][3]float64{
	"MFJ":    {{0, 0, .10}, {23850, 2385, .12}, {96950, 11157, .22}, {206700, 35302, .24}, {394600, 80398, .32}, {501050, 114462, .35}, {751600, 202154.50, .37}},
	"HOH":    {{0, 0, .10}, {17000, 1700, .12}, {64850, 7442, .22}, {103350, 15912, .24}, {197300, 38460, .32}, {250500, 55484, .35}, {626350, 187031.50, .37}},
	"Single": {{0, 0, .10}, {11925, 1192.50, .12}, {48475, 5578.50, .22}, {103350, 17651, .24}, {197300, 40199, .32}, {250525, 57231, .35}, {626350, 188769.75, .37}},
	"MFS":    {{0, 0, .10}, {11925, 1192.50, .12}, {48475, 5578.50, .22}, {103350, 17651, .24}, {197300, 40199, .32}, {250525, 57231, .35}, {375800, 101077.25, .37}},
}

func scheduleTax(rows [][3]float64, income float64) float64 {
	tax := 0.0
	for _, r := range rows {
		if income > r[0] {
			tax = r[1] + r[2]*(income-r[0])
		}
	}
	return tax
}

// TestFederalRateSchedules runs Apply_Tax_Brackets for each filing status at
// points inside every bracket and at every threshold, against Rev. Proc.
// 2024-40. Head of Household and MFS used the Single schedule (#1324).
func TestFederalRateSchedules(t *testing.T) {
	rs, _ := loadTaxReturn(t)
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	state := sess.GetState()
	job, _ := sess.CreateEntity(dtrules.GetRName("job"))
	res, _ := sess.CreateEntity(dtrules.GetRName("result"))
	constants, _ := sess.CreateEntity(dtrules.GetRName("constants"))
	dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Apply_Tax_Brackets"))
	if err != nil || dt == nil {
		t.Fatalf("Apply_Tax_Brackets: %v", err)
	}
	for status, rows := range revProc202440 {
		var incomes []float64
		for i, r := range rows {
			incomes = append(incomes, r[0], r[0]+1)
			if i+1 < len(rows) {
				incomes = append(incomes, (r[0]+rows[i+1][0])/2)
			} else {
				incomes = append(incomes, r[0]*2)
			}
		}
		for _, income := range incomes {
			job.Put(dtrules.GetRName("filing_status"), dtrules.NewRString(status))
			res.Put(dtrules.GetRName("taxable_income"), dtrules.GetRDoubleValue(income))
			state.EntityPush(constants)
			state.EntityPush(job)
			state.EntityPush(res)
			err := dt.Execute(state)
			state.EntityPop()
			state.EntityPop()
			state.EntityPop()
			if err != nil {
				t.Fatalf("%s $%.0f: %v", status, income, err)
			}
			v, _ := res.Get(dtrules.GetRName("regular_tax"))
			got, _ := v.DoubleValue()
			if want := scheduleTax(rows, income); math.Abs(got-want) > 0.01 {
				t.Errorf("%s, taxable income $%.0f: tax $%.2f, Rev. Proc. 2024-40 gives $%.2f", status, income, got, want)
			}
		}
	}
}

// #1321: the child tax credit and the EITC count compare relationship with
// "child"; Normalize_Dependent_Relationship maps a son, daughter, stepchild,
// foster or adopted child to it, ignoring case.
func TestNormalizeDependentRelationship(t *testing.T) {
	rs, _ := loadTaxReturn(t)
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	state := sess.GetState()
	job, _ := sess.CreateEntity(dtrules.GetRName("job"))
	dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName("Normalize_Dependent_Relationship"))
	if err != nil || dt == nil {
		t.Fatalf("Normalize_Dependent_Relationship: %v", err)
	}
	cases := map[string]string{
		"child": "child", "Child": "child", "son": "child", "Son": "child", "daughter": "child",
		"Daughter": "child", "stepdaughter": "child", "foster child": "child", "adopted_child": "child",
		"parent": "parent", "grandchild": "grandchild",
	}
	var deps []dtrules.Object
	var order []string
	for in := range cases {
		d, _ := sess.CreateEntity(dtrules.GetRName("dependent"))
		d.Put(dtrules.GetRName("relationship"), dtrules.NewRString(in))
		deps = append(deps, d)
		order = append(order, in)
	}
	arr, err := dtrules.NewArrayWithElements(sess, true, deps, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := job.Put(dtrules.GetRName("dependents"), arr); err != nil {
		t.Fatal(err)
	}
	state.EntityPush(job)
	defer state.EntityPop()
	if err := dt.Execute(state); err != nil {
		t.Fatal(err)
	}
	for i, in := range order {
		v, _ := deps[i].(dtrules.Entity).Get(dtrules.GetRName("relationship"))
		if got := v.StringValue(); got != cases[in] {
			t.Errorf("relationship %q normalised to %q, want %q", in, got, cases[in])
		}
	}
}
