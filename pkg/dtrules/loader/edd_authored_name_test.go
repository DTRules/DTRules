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

package loader

import (
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
)

// TestEDDFieldKeepsAuthoredCase is the EDD half of #1040.
//
// Decision-table names were fixed first; EDD fields go out through the same
// accessor and had the same defect. RNames intern case-insensitively and the
// cache keeps the first spelling seen process-wide, so a field `Status`
// declared on one entity reported as `status` once another entity had declared
// `status` earlier — and the exporter wrote that back, renaming the author's
// field on every build.
//
// The entities are ordered so the lower-case spelling interns first. Reverse
// them and the bug hides, which is why this fixture is written the way it is.
func TestEDDFieldKeepsAuthoredCase(t *testing.T) {
	factory := entity.NewFactory(nil)
	l := NewEDDLoader(tzTestSession{}, factory)

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="1.0">
  <entity name="alpha" access="rw">
    <field name="status" type="string" access="rw"/>
  </entity>
  <entity name="Bravo" access="rw">
    <field name="Status" type="string" access="rw"/>
  </entity>
</entity_data_dictionary>`

	if err := l.Load(strings.NewReader(xml)); err != nil {
		t.Fatalf("Load: %v", err)
	}

	ent := factory.FindRefEntity(dtrules.GetRName("Bravo"))
	if ent == nil {
		t.Fatal("Bravo entity not found")
	}
	attr := dtrules.GetRName("Status")
	entry := ent.GetEntry(attr)
	if entry == nil {
		t.Fatal("Bravo.Status entry not found")
	}

	if entry.AuthoredName != "Status" {
		t.Errorf("AuthoredName = %q, want %q — the author's case was replaced by the spelling "+
			"interned first on another entity", entry.AuthoredName, "Status")
	}
	// The interned name is allowed to differ; that is what makes lookup
	// case-insensitive and is not the bug.
	t.Logf("interned = %q, authored = %q", entry.Attribute.StringValue(), entry.AuthoredName)
}
