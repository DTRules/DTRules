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

package loader

import (
	"strings"
	"testing"
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
)

// Phase 2 of #743: <field type="date" timezone="..."> instructs the loader
// to interpret a tz-naïve default-value string in the declared zone instead
// of UTC. Without the attribute the existing UTC behavior must be preserved.

// tzTestSession satisfies the bits of dtrules.Session that the EDD loader
// uses for date parsing — only GetDateParser is exercised here, so the rest
// can be left as zero-value stubs.
type tzTestSession struct{}

func (tzTestSession) GetState() dtrules.State                                    { return nil }
func (tzTestSession) GetEntityFactory() dtrules.EntityFactory                    { return nil }
func (tzTestSession) GetUniqueID() int                                           { return 0 }
func (tzTestSession) GetDateParser() dtrules.DateParser                          { return tzTestParser{} }
func (tzTestSession) SetDateParser(p dtrules.DateParser)                         {}
func (tzTestSession) GetRuleSet() dtrules.RuleSet                                { return nil }
func (tzTestSession) CreateEntity(*dtrules.RName) (dtrules.Entity, error)        { return nil, nil }
func (tzTestSession) GetEntityByID(int) dtrules.Entity                           { return nil }
func (tzTestSession) GetAttribute(string) interface{}                            { return nil }
func (tzTestSession) SetAttribute(string, interface{})                           {}
func (tzTestSession) Execute(string) error                                       { return nil }
func (tzTestSession) ExecuteAt(string, string) error                             { return nil }
func (tzTestSession) Compile(string) (dtrules.Object, error)                     { return nil, nil }
func (tzTestSession) CompileExpressionToBytecode(string) (*dtrules.BytecodeChunk, error) {
	return nil, nil
}

type tzTestParser struct{}

func (tzTestParser) GetDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}
func (tzTestParser) Parse(s string) (time.Time, error) { return tzTestParser{}.GetDate(s) }

func TestEDDLoader_FieldTimezoneRebasesDefault(t *testing.T) {
	factory := entity.NewFactory(nil)
	l := NewEDDLoader(tzTestSession{}, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <entity name="job" access="rw">
    <field name="filing_date" type="date" timezone="America/New_York" default_value="2026-04-15" access="rw"/>
  </entity>
</entity_data_dictionary>`

	if err := l.Load(strings.NewReader(xml)); err != nil {
		t.Fatalf("Load: %v", err)
	}

	jobEnt := factory.FindRefEntity(dtrules.GetRName("job"))
	if jobEnt == nil {
		t.Fatal("job entity not created")
	}
	entry := jobEnt.GetEntry(dtrules.GetRName("filing_date"))
	if entry == nil {
		t.Fatal("filing_date attribute missing")
	}
	tv, err := entry.DefaultValue.TimeValue()
	if err != nil {
		t.Fatalf("DefaultValue.TimeValue: %v", err)
	}
	if tv.Location().String() != "America/New_York" {
		t.Fatalf("location = %v, want America/New_York", tv.Location())
	}
	if tv.Year() != 2026 || tv.Month() != time.April || tv.Day() != 15 {
		t.Fatalf("date = %s, want 2026-04-15 NY", tv.Format(time.RFC3339))
	}
	if tv.Hour() != 0 || tv.Minute() != 0 {
		t.Fatalf("time-of-day = %02d:%02d, want 00:00 NY", tv.Hour(), tv.Minute())
	}
}

func TestEDDLoader_NoTimezone_StaysUTC(t *testing.T) {
	factory := entity.NewFactory(nil)
	l := NewEDDLoader(tzTestSession{}, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <entity name="job" access="rw">
    <field name="filing_date" type="date" default_value="2026-04-15" access="rw"/>
  </entity>
</entity_data_dictionary>`

	if err := l.Load(strings.NewReader(xml)); err != nil {
		t.Fatalf("Load: %v", err)
	}

	jobEnt := factory.FindRefEntity(dtrules.GetRName("job"))
	entry := jobEnt.GetEntry(dtrules.GetRName("filing_date"))
	tv, err := entry.DefaultValue.TimeValue()
	if err != nil {
		t.Fatalf("TimeValue: %v", err)
	}
	if tv.Location() != time.UTC {
		t.Fatalf("location = %v, want UTC", tv.Location())
	}
}
