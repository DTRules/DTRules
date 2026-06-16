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

package authoring_test

import (
	"reflect"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

func newTestProject(t *testing.T) *authoring.Project {
	t.Helper()
	p, err := authoring.NewInMemoryProject(t.TempDir())
	if err != nil {
		t.Fatalf("NewInMemoryProject: %v", err)
	}
	return p
}

func TestDependencies_DirectPerforms(t *testing.T) {
	p := newTestProject(t)
	tbl, err := p.AddTable("Root", "tables_dt.xml", "test")
	if err != nil {
		t.Fatal(err)
	}
	_ = tbl.AddAction(authoring.Action{DSL: "perform TableA"})
	_ = tbl.AddAction(authoring.Action{DSL: "perform TableB"})

	got := p.Table("Root").Dependencies()
	want := []string{"TableA", "TableB"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Dependencies() = %v, want %v", got, want)
	}
}

func TestDependencies_Deduplicated(t *testing.T) {
	p := newTestProject(t)
	tbl, err := p.AddTable("Root", "tables_dt.xml", "test")
	if err != nil {
		t.Fatal(err)
	}
	_ = tbl.AddAction(authoring.Action{DSL: "perform TableA"})
	_ = tbl.AddAction(authoring.Action{DSL: "perform TableA"})

	got := p.Table("Root").Dependencies()
	want := []string{"TableA"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Dependencies() = %v, want %v", got, want)
	}
}

func TestDependencies_NoPerforms(t *testing.T) {
	p := newTestProject(t)
	tbl, err := p.AddTable("Root", "tables_dt.xml", "test")
	if err != nil {
		t.Fatal(err)
	}
	_ = tbl.AddAction(authoring.Action{DSL: "set applicant.eligible = true"})

	got := p.Table("Root").Dependencies()
	if len(got) != 0 {
		t.Errorf("Dependencies() = %v, want empty", got)
	}
}

func TestCallers_CrossTableSearch(t *testing.T) {
	// TableA -> TableB -> TableC
	p := newTestProject(t)

	tblA, err := p.AddTable("TableA", "tables_dt.xml", "test")
	if err != nil {
		t.Fatal(err)
	}
	_ = tblA.AddAction(authoring.Action{DSL: "perform TableB"})

	tblB, err := p.AddTable("TableB", "tables_dt.xml", "test")
	if err != nil {
		t.Fatal(err)
	}
	_ = tblB.AddAction(authoring.Action{DSL: "perform TableC"})

	_, err = p.AddTable("TableC", "tables_dt.xml", "test")
	if err != nil {
		t.Fatal(err)
	}

	got := p.Table("TableB").Callers()
	want := []string{"TableA"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Callers() = %v, want %v", got, want)
	}
}

func TestCallers_None(t *testing.T) {
	p := newTestProject(t)
	_, err := p.AddTable("TableA", "tables_dt.xml", "test")
	if err != nil {
		t.Fatal(err)
	}
	tblB, err := p.AddTable("TableB", "tables_dt.xml", "test")
	if err != nil {
		t.Fatal(err)
	}
	_ = tblB.AddAction(authoring.Action{DSL: "set applicant.eligible = true"})

	got := p.Table("TableA").Callers()
	if len(got) != 0 {
		t.Errorf("Callers() = %v, want empty", got)
	}
}
