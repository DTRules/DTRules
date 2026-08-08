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

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
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
		{"bigint", TypeBigInt},
		{"biginteger", TypeBigInt},
		{"BIGINT", TypeBigInt},
		{"fixed", TypeFixed},
		{"FIXED", TypeFixed},
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

// =============================================================================
// BigInt Mapping Tests
// =============================================================================

func TestBigIntAttributeMapping(t *testing.T) {
	session := newMockSession()

	// Create reference entity with bigint field
	factory := session.factory
	blockchainName := dtrules.GetRName("blockchain")
	blockchainRef, _ := factory.FindCreateRefEntity(false, blockchainName)

	// Add BigInt attribute
	blockchainRef.AddAttribute(dtrules.GetRName("balance"), "", dtrules.GetRBigIntFromInt64(0),
		true, true, dtrules.TypeBigInt, "", "", "", "")

	// Create mapping
	m := NewMapping(session)

	mapXML := `
<mapping>
	<XMLtoEDD>
		<map>
			<setattribute tag='balance' RAttribute='balance' enclosure='blockchain' type='bigint'/>
			<createentity entity='blockchain' tag='blockchain' id='id'/>
		</map>
		<entities>
			<entity name='blockchain' number='1'/>
		</entities>
	</XMLtoEDD>
</mapping>
`

	err := m.LoadMapping(strings.NewReader(mapXML))
	if err != nil {
		t.Fatalf("LoadMapping failed: %v", err)
	}

	// Verify attribute info was loaded with BigInt type
	aInfo := m.GetAttributeInfo("balance")
	if aInfo == nil {
		t.Fatal("Expected attribute info for 'balance'")
	}
	attrib := aInfo.Lookup("blockchain")
	if attrib == nil {
		t.Fatal("Expected attrib for 'blockchain' enclosure")
	}
	if attrib.Type != TypeBigInt {
		t.Errorf("Expected TypeBigInt, got %v", attrib.Type)
	}
}

func TestBigIntJSONDataLoading(t *testing.T) {
	session := newMockSession()

	// Create reference entity with bigint field
	factory := session.factory
	walletName := dtrules.GetRName("wallet")
	walletRef, _ := factory.FindCreateRefEntity(false, walletName)

	// Add fields including BigInt
	walletRef.AddAttribute(dtrules.GetRName("id"), "", dtrules.GetRIntegerValue(0),
		true, true, dtrules.TypeInteger, "", "", "", "")
	walletRef.AddAttribute(dtrules.GetRName("balance"), "", dtrules.GetRBigIntFromInt64(0),
		true, true, dtrules.TypeBigInt, "", "", "", "")

	// Create mapping
	m := NewMapping(session)

	mapXML := `
<mapping>
	<XMLtoEDD>
		<map>
			<setattribute tag='id' RAttribute='id' enclosure='wallet' type='integer'/>
			<setattribute tag='balance' RAttribute='balance' enclosure='wallet' type='bigint'/>
			<createentity entity='wallet' tag='wallet' id='id'/>
		</map>
		<entities>
			<entity name='wallet' number='*'/>
		</entities>
	</XMLtoEDD>
</mapping>
`

	err := m.LoadMapping(strings.NewReader(mapXML))
	if err != nil {
		t.Fatalf("LoadMapping failed: %v", err)
	}

	// Load JSON data with a large number as string (to preserve precision)
	dataJSON := `{
		"wallet": {
			"id": 1,
			"balance": "123456789012345678901234567890"
		}
	}`

	err = m.LoadDataJSON(strings.NewReader(dataJSON))
	if err != nil {
		t.Fatalf("LoadDataJSON failed: %v", err)
	}
}

// TestFixedJSONConvertToAttributeType directly exercises the TypeFixed arm of
// jsonDataLoader.convertToAttributeType — the runtime JSON data path that
// parses fp values from strings to preserve the 10^-8 grid.
func TestFixedJSONConvertToAttributeType(t *testing.T) {
	session := newMockSession()
	m := NewMapping(session)
	l := newJSONDataLoader(m)

	cases := []struct {
		body string
		want string
	}{
		{"1680748.45091643", "1680748.45091643"},
		{"-0.00000001", "-0.00000001"},
		{"0", "0.00000000"},
	}
	for _, c := range cases {
		t.Run(c.body, func(t *testing.T) {
			obj := l.convertToAttributeType(TypeFixed, c.body, c.body)
			fp, ok := obj.(*dtrules.RFixed)
			if !ok {
				t.Fatalf("expected *RFixed, got %T", obj)
			}
			if got := fp.StringValue(); got != c.want {
				t.Errorf("convertToAttributeType(%q) = %q, want %q", c.body, got, c.want)
			}
		})
	}
}

func TestBigIntConversionFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"small number", "42", "42"},
		{"large number", "123456789012345678901234567890", "123456789012345678901234567890"},
		{"negative large", "-999999999999999999999", "-999999999999999999999"},
		{"max int64", "9223372036854775807", "9223372036854775807"},
		{"beyond int64", "9223372036854775808", "9223372036854775808"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bi, err := dtrules.GetRBigIntFromString(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse %q: %v", tt.input, err)
			}
			if bi.StringValue() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, bi.StringValue())
			}
		})
	}
}

func TestBigIntSerializesToString(t *testing.T) {
	// Verify that BigInt values serialize to strings that preserve full precision
	largeNum := "123456789012345678901234567890"
	bi, err := dtrules.GetRBigIntFromString(largeNum)
	if err != nil {
		t.Fatalf("Failed to create BigInt: %v", err)
	}

	// StringValue() is what would be used for JSON output
	serialized := bi.StringValue()
	if serialized != largeNum {
		t.Errorf("Serialization lost precision: expected %s, got %s", largeNum, serialized)
	}

	// PostFix() should also return the string representation
	postfix := bi.PostFix()
	if postfix != largeNum {
		t.Errorf("PostFix() lost precision: expected %s, got %s", largeNum, postfix)
	}
}

// TestSingletonCreateEntityBindsToInitializedInstance covers the two load
// orders agreeing.
//
// `dtrules run --input` loads data and then pushes singletons, so a root tag
// like <patient> needs a createentity or its fields have no entity to attach
// to. The interview path does the reverse — Initialize, then LoadData — and
// with that same createentity it used to build a SECOND patient, leaving the
// pushed singleton empty and the loaded values unreachable. SinusitisTherapy
// divided by a zero plasma creatinine that way.
func TestSingletonCreateEntityBindsToInitializedInstance(t *testing.T) {
	const mapXML = `<?xml version="1.0" encoding="UTF-8"?>
<mapping>
	<XMLtoEDD>
		<map>
			<setattribute tag='pcr' RAttribute='pcr' enclosure='patient' type='double'></setattribute>
			<createentity entity='patient' tag='patient' id='id'></createentity>
		</map>
		<entities>
			<entity name='patient' number='1'></entity>
		</entities>
		<initialization>
			<initialentity entity='patient' epush='true'></initialentity>
		</initialization>
	</XMLtoEDD>
</mapping>`
	const data = `<patient><pcr>0.9</pcr></patient>`

	sess := newMockSession()
	patientRef, _ := sess.factory.FindCreateRefEntity(false, dtrules.GetRName("patient"))
	patientRef.AddAttribute(dtrules.GetRName("pcr"), "", dtrules.GetRDoubleValue(0),
		true, true, dtrules.TypeDouble, "", "", "", "")

	m := NewMapping(sess)
	if err := m.LoadMapping(strings.NewReader(mapXML)); err != nil {
		t.Fatalf("LoadMapping: %v", err)
	}
	if err := m.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if err := m.LoadData(strings.NewReader(data)); err != nil {
		t.Fatalf("LoadData: %v", err)
	}

	// The value has to be readable through the entity that is actually on
	// the stack — that is what the rules resolve against.
	state := sess.GetState()
	ent, err := state.FindEntity(dtrules.GetRName("pcr"))
	if err != nil {
		t.Fatalf("no entity on the stack carries pcr: %v", err)
	}
	got, err := ent.Get(dtrules.GetRName("pcr"))
	if err != nil {
		t.Fatalf("patient.pcr: %v", err)
	}
	v, err := got.DoubleValue()
	if err != nil {
		t.Fatalf("patient.pcr is not a double: %v", err)
	}
	if v != 0.9 {
		t.Errorf("patient.pcr = %v, want 0.9 — the loaded value landed on a different instance", v)
	}
}
