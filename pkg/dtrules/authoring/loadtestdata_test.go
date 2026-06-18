//go:build archive

// ARCHIVED: LoadTestData fixtures with DSL but no compiled postfix; the loader
// rejects them. Run with `go test -tags archive`. Revisit.

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
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

func loadTestdataDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Join(filepath.Dir(file), "testdata", "loadtestdata")
}

func openLoadTestDataProject(t *testing.T) *authoring.Project {
	t.Helper()
	p, err := authoring.OpenProject(loadTestdataDir(t))
	if err != nil {
		t.Fatalf("OpenProject: %v", err)
	}
	return p
}

// jobData is minimal XML data for a job entity.
const jobData = `<?xml version="1.0" encoding="UTF-8"?>
<test><job><id>1</id><age>30</age></job></test>`

// TestLoadTestData_PushesInitEntities verifies that <initialentity epush='true'>
// for an entity NOT in <entities> still pushes it onto the entity stack (created fresh).
func TestLoadTestData_PushesInitEntities(t *testing.T) {
	p := openLoadTestDataProject(t)

	// 'params' is in the EDD but NOT in <entities> cardinality section — only epush drives the push.
	mapXML := `<mapping><XMLtoEDD>
		<map>
			<createentity entity='job' tag='job' id='id'></createentity>
		</map>
		<initialization>
			<initialentity entity='params' epush='true'></initialentity>
		</initialization>
	</XMLtoEDD></mapping>`

	err := p.LoadTestDataReader(strings.NewReader(mapXML), strings.NewReader(jobData))
	if err != nil {
		t.Fatalf("LoadTestDataReader: %v", err)
	}

	names := p.EntityStackNames()
	if !slices.Contains(names, "params") {
		t.Errorf("expected 'params' on entity stack after epush=true, got %v", names)
	}
}

// TestLoadTestData_EpushFalseNotPushed verifies that epush='false' leaves the
// entity off the entity stack.
func TestLoadTestData_EpushFalseNotPushed(t *testing.T) {
	p := openLoadTestDataProject(t)

	// 'params' is in the EDD but NOT in <entities> — epush=false means it stays off stack.
	mapXML := `<mapping><XMLtoEDD>
		<map>
			<createentity entity='job' tag='job' id='id'></createentity>
		</map>
		<initialization>
			<initialentity entity='params' epush='false'></initialentity>
		</initialization>
	</XMLtoEDD></mapping>`

	err := p.LoadTestDataReader(strings.NewReader(mapXML), strings.NewReader(jobData))
	if err != nil {
		t.Fatalf("LoadTestDataReader: %v", err)
	}

	names := p.EntityStackNames()
	if slices.Contains(names, "params") {
		t.Errorf("expected 'params' NOT on entity stack (epush=false), got %v", names)
	}
}

// TestLoadTestData_NoInitializationSection verifies that a mapping without any
// <initialization> section loads without error and does not push params.
func TestLoadTestData_NoInitializationSection(t *testing.T) {
	p := openLoadTestDataProject(t)

	// No <initialization> section at all; 'params' should not appear on the stack.
	mapXML := `<mapping><XMLtoEDD>
		<map>
			<createentity entity='job' tag='job' id='id'></createentity>
		</map>
	</XMLtoEDD></mapping>`

	err := p.LoadTestDataReader(strings.NewReader(mapXML), strings.NewReader(jobData))
	if err != nil {
		t.Fatalf("LoadTestDataReader: %v", err)
	}

	names := p.EntityStackNames()
	if slices.Contains(names, "params") {
		t.Errorf("expected 'params' absent from entity stack (no <initialization>), got %v", names)
	}
}

// TestLoadTestData_IntegrationWithExecuteEntry verifies that after LoadTestData
// pushes 'job' via epush='true', ExecuteEntry can resolve job.age in conditions.
func TestLoadTestData_IntegrationWithExecuteEntry(t *testing.T) {
	p := openLoadTestDataProject(t)

	dataPath := filepath.Join(loadTestdataDir(t), "inputs", "job_senior.xml")
	if err := p.LoadTestData(dataPath); err != nil {
		t.Fatalf("LoadTestData: %v", err)
	}

	// job_senior.xml sets age=45; Check_Job_Age column 1 fires when age >= 40.
	trace, err := p.ExecuteEntry("Check_Job_Age")
	if err != nil {
		t.Fatalf("ExecuteEntry: %v", err)
	}

	found := false
	for _, inv := range trace.Invocations {
		if inv.TableName == "Check_Job_Age" && inv.Column == 1 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected Check_Job_Age column 1 to fire for age=45, got invocations: %v", trace.Invocations)
	}
}
