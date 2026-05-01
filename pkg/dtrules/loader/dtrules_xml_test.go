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
)

// Phase 2 of #743: <default_timezone> in DTRules.xml is the project-level
// fallback when an op has no `in zone` clause and no field-level zone.

func TestLoadProjectConfig_ExplicitDefaultTimezone(t *testing.T) {
	xml := `<DTRules>
  <compiler>EL</compiler>
  <default_timezone>America/New_York</default_timezone>
  <RuleSet name="X" source="file"/>
</DTRules>`
	cfg, err := LoadProjectConfig(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}
	if got := cfg.GetDefaultTimezone(); got != "America/New_York" {
		t.Fatalf("default_timezone = %q, want America/New_York", got)
	}
}

func TestLoadProjectConfig_AbsentDefaultsToUTC(t *testing.T) {
	xml := `<DTRules>
  <compiler>EL</compiler>
  <RuleSet name="X" source="file"/>
</DTRules>`
	cfg, err := LoadProjectConfig(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}
	if got := cfg.GetDefaultTimezone(); got != "UTC" {
		t.Fatalf("absent default = %q, want UTC", got)
	}
}

func TestLoadProjectConfig_EmptyStringDefaultsToUTC(t *testing.T) {
	xml := `<DTRules>
  <default_timezone>   </default_timezone>
</DTRules>`
	cfg, err := LoadProjectConfig(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}
	if got := cfg.GetDefaultTimezone(); got != "UTC" {
		t.Fatalf("blank default = %q, want UTC", got)
	}
}
