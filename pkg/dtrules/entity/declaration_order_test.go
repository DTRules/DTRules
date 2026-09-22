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

package entity

import (
	"reflect"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// #1231: reference entities and their attributes are listed in declaration
// order, every time. The names are declared out of alphabetical order so
// that a sorted listing would fail too.
func TestDeclarationOrder_RefEntitiesAndAttributes(t *testing.T) {
	names := []string{"zeta", "alpha", "mid", "beta", "omega", "gamma", "kappa", "delta"}
	fields := []string{"zz", "aa", "mm", "bb", "oo", "gg", "kk", "dd"}

	for run := 0; run < 20; run++ {
		f := NewFactory(nil)
		for _, n := range names {
			e, err := f.FindCreateRefEntity(false, dtrules.GetRName(n))
			if err != nil {
				t.Fatal(err)
			}
			for _, fld := range fields {
				e.AddAttribute(dtrules.GetRName(fld), "", nil, true, true, dtrules.TypeString, "", "", "", "")
			}
		}

		var gotEntities []string
		for _, e := range f.GetRefEntities() {
			gotEntities = append(gotEntities, e.GetName().GetName())
		}
		if !reflect.DeepEqual(gotEntities, names) {
			t.Fatalf("run %d: GetRefEntities = %v, want declaration order %v", run, gotEntities, names)
		}

		var gotFields []string
		ref := f.GetRefEntities()[0]
		for _, a := range ref.GetAttributeNames() {
			if a.GetName() == "zeta" || a.GetName() == MappingKeyName.GetName() {
				continue // self-reference and mapping key bookkeeping
			}
			gotFields = append(gotFields, a.GetName())
		}
		if !reflect.DeepEqual(gotFields, fields) {
			t.Fatalf("run %d: GetAttributeNames = %v, want declaration order %v", run, gotFields, fields)
		}
	}
}
