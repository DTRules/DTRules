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

package excel

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

var tableNameRE = regexp.MustCompile(`<table_name>([^<]*)</table_name>`)

// AssertRuleSetCovers refuses an export whose rule set is missing tables the
// XML declares.
//
// Exports build from a tolerantly-loaded rule set, deliberately: the export
// has to work while an operator has DSL written and postfix not yet compiled.
// The cost is that a table which will not load is skipped, and the workbook is
// written without it -- bootstrapping Cribbage produced a six-sheet workbook
// for an eleven-table project and reported success (#1081).
//
// While Excel merely sat beside the XML that was an incomplete artifact. Once
// Excel is the system of record and the XML is generated from it, the same
// skip deletes rules: the next build regenerates the XML from a workbook that
// never received them.
//
// So this is a refusal, not a warning. If Excel cannot represent every table
// the XML declares, that is a defect in DTRules -- the system of record has to
// be able to hold the system -- and the right response is to stop and say
// which tables, not to write a lossy workbook quietly.
func AssertRuleSetCovers(rs *session.RuleSet, xmlPaths ...string) error {
	declared := map[string]bool{}
	for _, p := range xmlPaths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue // a path that is not there declares nothing
		}
		for _, m := range tableNameRE.FindAllStringSubmatch(string(data), -1) {
			if name := strings.TrimSpace(m[1]); name != "" {
				declared[strings.ToLower(name)] = true
			}
		}
	}
	if len(declared) == 0 {
		return nil
	}

	for _, rn := range rs.GetDecisionTableNames() {
		delete(declared, strings.ToLower(rn.StringValue()))
	}
	if len(declared) == 0 {
		return nil
	}

	missing := make([]string, 0, len(declared))
	for name := range declared {
		missing = append(missing, name)
	}
	sort.Strings(missing)

	return fmt.Errorf(
		"refusing to export: %d table(s) declared in the XML did not load, so the "+
			"workbook would be written without them: %s\n"+
			"  Excel is the system of record, so a workbook that cannot hold every "+
			"table is a defect, not a partial success. Fix the load errors "+
			"(dtrules verify names them) and export again",
		len(missing), strings.Join(missing, ", "))
}
