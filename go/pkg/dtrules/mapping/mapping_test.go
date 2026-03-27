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

package mapping

import (
	"strings"
	"testing"

	"github.com/DTRules/DTRules/go/pkg/dtrules"
	"github.com/DTRules/DTRules/go/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/go/pkg/dtrules/interpreter"
)

// mockSession implements minimal Session for testing
type mockSession struct {
	uniqueID int
	factory  *entity.Factory
	state    *interpreter.DTState
}

func newMockSession() *mockSession {
	factory := entity.NewFactory(nil)
	s := &mockSession{factory: factory}
	s.state = interpreter.NewDTState(s)
	return s
}

func (m *mockSession) GetState() dtrules.State                 { return m.state }
func (m *mockSession) GetEntityFactory() dtrules.EntityFactory { return m.factory }
func (m *mockSession) GetUniqueID() int                        { m.uniqueID++; return m.uniqueID }
func (m *mockSession) GetDateParser() dtrules.DateParser       { return nil }
func (m *mockSession) GetRuleSet() dtrules.RuleSet             { return nil }
func (m *mockSession) CreateEntity(name *dtrules.RName) (dtrules.Entity, error) {
	return m.factory.CreateEntity(m, name)
}
func (m *mockSession) Compile(expr string) (dtrules.Object, error) {
	return nil, nil
}
func (m *mockSession) GetEntityByID(id int) dtrules.Entity { return nil }

func TestLoadMapping(t *testing.T) {
	session := newMockSession()

	// First create the reference entities
	factory := session.factory
	personName := dtrules.GetRName("person")
	personRef, _ := factory.FindCreateRefEntity(false, personName)

	// Add attributes
	personRef.AddAttribute(dtrules.GetRName("id"), "", dtrules.GetRIntegerValue(0),
		true, true, dtrules.TypeInteger, "", "", "", "")
	personRef.AddAttribute(dtrules.GetRName("name"), "", dtrules.NewRString(""),
		true, true, dtrules.TypeString, "", "", "", "")
	personRef.AddAttribute(dtrules.GetRName("age"), "", dtrules.GetRIntegerValue(0),
		true, true, dtrules.TypeInteger, "", "", "", "")

	// Create mapping
	m := NewMapping(session)

	mapXML := `
<mapping>
	<XMLtoEDD>
		<map>
			<setattribute tag='id' RAttribute='id' enclosure='person' type='integer'/>
			<setattribute tag='name' RAttribute='name' enclosure='person' type='string'/>
			<setattribute tag='age' RAttribute='age' enclosure='person' type='integer'/>
			<createentity entity='person' tag='person' id='id'/>
		</map>
		<entities>
			<entity name='person' number='*'/>
		</entities>
		<initialization>
		</initialization>
	</XMLtoEDD>
</mapping>
`

	err := m.LoadMapping(strings.NewReader(mapXML))
	if err != nil {
		t.Fatalf("LoadMapping failed: %v", err)
	}

	// Verify entity info was loaded
	info := m.GetEntityInfo("person")
	if info == nil {
		t.Fatal("Expected entity info for 'person'")
	}
	if info.Name != "person" {
		t.Errorf("Expected entity name 'person', got '%s'", info.Name)
	}
	if info.ID != "id" {
		t.Errorf("Expected id attribute 'id', got '%s'", info.ID)
	}

	// Verify attribute info was loaded
	aInfo := m.GetAttributeInfo("name")
	if aInfo == nil {
		t.Fatal("Expected attribute info for 'name'")
	}
	attrib := aInfo.Lookup("person")
	if attrib == nil {
		t.Fatal("Expected attrib for 'person' enclosure")
	}
	if attrib.RAttribute != "name" {
		t.Errorf("Expected RAttribute 'name', got '%s'", attrib.RAttribute)
	}
	if attrib.Type != TypeString {
		t.Errorf("Expected TypeString, got %v", attrib.Type)
	}

	// Verify cardinality
	if m.GetEntityCardinality("person") != "*" {
		t.Errorf("Expected cardinality '*', got '%s'", m.GetEntityCardinality("person"))
	}
}

func TestLoadData(t *testing.T) {
	session := newMockSession()

	// First create the reference entities
	factory := session.factory
	personName := dtrules.GetRName("person")
	personRef, _ := factory.FindCreateRefEntity(false, personName)

	// Add attributes
	personRef.AddAttribute(dtrules.GetRName("id"), "", dtrules.GetRIntegerValue(0),
		true, true, dtrules.TypeInteger, "", "", "", "")
	personRef.AddAttribute(dtrules.GetRName("name"), "", dtrules.NewRString(""),
		true, true, dtrules.TypeString, "", "", "", "")
	personRef.AddAttribute(dtrules.GetRName("age"), "", dtrules.GetRIntegerValue(0),
		true, true, dtrules.TypeInteger, "", "", "", "")

	// Create and load mapping
	m := NewMapping(session)

	mapXML := `
<mapping>
	<XMLtoEDD>
		<map>
			<setattribute tag='id' RAttribute='id' enclosure='person' type='integer'/>
			<setattribute tag='personname' RAttribute='name' enclosure='person' type='string'/>
			<setattribute tag='age' RAttribute='age' enclosure='person' type='integer'/>
			<createentity entity='person' tag='person' id='id'/>
		</map>
		<entities>
			<entity name='person' number='*'/>
		</entities>
	</XMLtoEDD>
</mapping>
`

	err := m.LoadMapping(strings.NewReader(mapXML))
	if err != nil {
		t.Fatalf("LoadMapping failed: %v", err)
	}

	// Now load data
	dataXML := `
<data>
	<person id="1">
		<personname>Alice</personname>
		<age>30</age>
	</person>
	<person id="2">
		<personname>Bob</personname>
		<age>25</age>
	</person>
</data>
`

	err = m.LoadData(strings.NewReader(dataXML))
	if err != nil {
		t.Fatalf("LoadData failed: %v", err)
	}
}

func TestLoadDataWithInitialEntity(t *testing.T) {
	session := newMockSession()

	// Create reference entities
	factory := session.factory

	// Create job entity (singleton)
	jobName := dtrules.GetRName("job")
	jobRef, _ := factory.FindCreateRefEntity(false, jobName)
	jobRef.AddAttribute(dtrules.GetRName("id"), "", dtrules.GetRIntegerValue(0),
		true, true, dtrules.TypeInteger, "", "", "", "")
	jobRef.AddAttribute(dtrules.GetRName("name"), "", dtrules.NewRString(""),
		true, true, dtrules.TypeString, "", "", "", "")

	// Create person entity (multiple)
	personName := dtrules.GetRName("person")
	personRef, _ := factory.FindCreateRefEntity(false, personName)
	personRef.AddAttribute(dtrules.GetRName("id"), "", dtrules.GetRIntegerValue(0),
		true, true, dtrules.TypeInteger, "", "", "", "")
	personRef.AddAttribute(dtrules.GetRName("name"), "", dtrules.NewRString(""),
		true, true, dtrules.TypeString, "", "", "", "")
	personRef.AddAttribute(dtrules.GetRName("age"), "", dtrules.GetRIntegerValue(0),
		true, true, dtrules.TypeInteger, "", "", "", "")

	// Create and load mapping
	m := NewMapping(session)

	mapXML := `
<mapping>
	<XMLtoEDD>
		<map>
			<setattribute tag='id' RAttribute='id' enclosure='job' type='integer'/>
			<setattribute tag='jobname' RAttribute='name' enclosure='job' type='string'/>
			<createentity entity='job' tag='job' id='id'/>

			<setattribute tag='id' RAttribute='id' enclosure='person' type='integer'/>
			<setattribute tag='personname' RAttribute='name' enclosure='person' type='string'/>
			<setattribute tag='age' RAttribute='age' enclosure='person' type='integer'/>
			<createentity entity='person' tag='person' id='id'/>
		</map>
		<entities>
			<entity name='job' number='1'/>
			<entity name='person' number='*'/>
		</entities>
		<initialization>
			<initialentity entity='job'/>
		</initialization>
	</XMLtoEDD>
</mapping>
`

	err := m.LoadMapping(strings.NewReader(mapXML))
	if err != nil {
		t.Fatalf("LoadMapping failed: %v", err)
	}

	// Initialize creates the job entity and pushes it onto the entity stack
	err = m.Initialize()
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Verify job entity is on entity stack
	jobEntity, err := session.state.EntityFetch(0)
	if err != nil {
		t.Fatalf("EntityFetch failed: %v", err)
	}
	if jobEntity == nil {
		t.Fatal("Expected job entity on stack")
	}
	if jobEntity.GetName().StringValue() != "job" {
		t.Errorf("Expected entity name 'job', got '%s'", jobEntity.GetName().StringValue())
	}

	// Now load data
	dataXML := `
<data>
	<job id="100">
		<jobname>Test Job</jobname>
	</job>
	<person id="1">
		<personname>Alice</personname>
		<age>30</age>
	</person>
	<person id="2">
		<personname>Bob</personname>
		<age>25</age>
	</person>
</data>
`

	err = m.LoadData(strings.NewReader(dataXML))
	if err != nil {
		t.Fatalf("LoadData failed: %v", err)
	}
}

func TestAttributeTypes(t *testing.T) {
	tests := []struct {
		input    string
		expected AttributeType
	}{
		{"string", TypeString},
		{"STRING", TypeString},
		{"integer", TypeInteger},
		{"int", TypeInteger},
		{"double", TypeDouble},
		{"float", TypeDouble},
		{"boolean", TypeBoolean},
		{"bool", TypeBoolean},
		{"date", TypeDate},
		{"entity", TypeEntity},
		{"array", TypeArray},
		{"list", TypeArray},
		{"xmlvalue", TypeXMLValue},
		{"unknown", TypeNone},
		{"", TypeNone},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseAttributeType(tt.input)
			if result != tt.expected {
				t.Errorf("ParseAttributeType(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
