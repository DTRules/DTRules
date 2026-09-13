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

package dtrules_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A state's rate logic used to read result.agi directly, which is the federal
// AGI and the one number every state shares. That is why only one state could
// ever be computed: there was no way to run a state's table against anything
// but the taxpayer's whole income.
//
// They now read result.state_calc_agi — the AGI *this* calculation runs
// against. The dispatcher sets it to result.agi before the resident pass, so
// nothing changed; what changed is that the tables became addressable, which
// is the prerequisite for computing a state the taxpayer does not live in
// (#1177).

var dslTag = regexp.MustCompile(`(?s)<(?:condition|action|initial_action|context)_dsl>(.*?)</`)

func TestStateTablesReadTheCalculationAGI(t *testing.T) {
	cwd, _ := os.Getwd()
	statesDir := filepath.Join(cwd, "..", "..", "sampleprojects", "TaxReturn", "xml", "states")
	entries, err := os.ReadDir(statesDir)
	if err != nil {
		t.Skip("TaxReturn state rules not present")
	}

	offenders := map[string]int{}
	sawIndirection := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), "_dt.xml") {
			continue
		}
		data, rerr := os.ReadFile(filepath.Join(statesDir, e.Name()))
		if rerr != nil {
			continue
		}
		for _, m := range dslTag.FindAllStringSubmatch(string(data), -1) {
			dsl := m[1]
			sawIndirection += strings.Count(dsl, "result.state_calc_agi")
			// result.state_calc_agi contains "result.agi" as no substring, so
			// a word-boundary match is enough to separate them.
			for _, ref := range regexp.MustCompile(`result\.agi\b`).FindAllString(dsl, -1) {
				_ = ref
				offenders[e.Name()]++
			}
		}
	}

	if sawIndirection == 0 {
		t.Fatal("no state table reads result.state_calc_agi; the fixture or the refactor is wrong")
	}
	for file, n := range offenders {
		t.Errorf("%s reads result.agi in %d place(s) — a state table reading the federal AGI "+
			"directly can only ever be run for the state the taxpayer lives in", file, n)
	}
	t.Logf("%d state DSL rows read result.state_calc_agi, 0 read result.agi", sawIndirection)
}
