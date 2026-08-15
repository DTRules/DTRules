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

package analysis

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// initialEntities are the entities a project's mapping pushes onto the entity
// stack before any table runs.
//
// The analyzer resolved bare identifiers only against entities a table's own
// `for all` / `using` context pushed, and gave up entirely when that context
// was empty — which is most tables. But a bare name in such a table is not
// unresolvable: it resolves against whatever the mapping pushed at
// initialization, and the mapping says exactly what that is:
//
//	<initialization>
//	  <initialentity entity='job' epush='true'></initialentity>
//	</initialization>
//
// Without this, CHIP's `lostInsuranceDate + 90 days is greater than the
// currentdate` — which compiles to `currentdate d>` and reads job.currentdate
// — was reported as "unused EDD field: job.currentdate". The field is read on
// every run (#776).
//
// This is precise rather than an over-approximation: these entities really are
// on the stack for every table, so a bare name really can resolve to them.
func initialEntities(xmlDir string) []string {
	var doc struct {
		Initial []struct {
			Entity string `xml:"entity,attr"`
			EPush  string `xml:"epush,attr"`
		} `xml:"initialization>initialentity"`
	}

	seen := make(map[string]bool)
	_ = filepath.WalkDir(xmlDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), "_map.xml") {
			return nil
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		doc.Initial = nil
		if xml.Unmarshal(data, &doc) != nil {
			return nil
		}
		for _, e := range doc.Initial {
			// epush='false' declares the entity without putting it on the
			// stack, so a bare name cannot resolve to it.
			if !strings.EqualFold(strings.TrimSpace(e.EPush), "true") {
				continue
			}
			if name := strings.ToLower(strings.TrimSpace(e.Entity)); name != "" {
				seen[name] = true
			}
		}
		return nil
	})

	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
