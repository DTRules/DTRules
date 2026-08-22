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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/loader"
)

// DatedConstantWarning is one field whose comment cites a year other than the
// project's declared tax year.
type DatedConstantWarning struct {
	Field   string // entity.field
	Years   []string
	TaxYear string
	EddFile string
}

func (w DatedConstantWarning) String() string {
	return fmt.Sprintf("INFO dated constant: %s cites %s but the project's tax year is %s — "+
		"verify the figure against its source (%s)",
		w.Field, strings.Join(w.Years, ", "), w.TaxYear, w.EddFile)
}

var yearPattern = regexp.MustCompile(`\b(19|20)\d{2}\b`)

// AnalyzeDatedConstants flags EDD fields whose comment cites a year that is
// not the project's declared tax year.
//
// A stale constant produces a plausible answer: rules run, verify is clean,
// and the scenarios agree with the constant they were derived from. Six were
// found by hand before this existed — a 2024 SS threshold, three adoption
// figures, the pre-OBBBA MFS standard deduction ($750 of taxable income per
// return), and, on the day this check was written, the 2024 FEIE cap and the
// 2024 underpayment rate. Not one was found by a failing test (#1140).
//
// The rule: a comment citing any year must also cite the project's year.
// "2025: $2,800. Was 2,700, the 2024 figure" passes — the history is welcome
// — while "(2024)" alone is exactly the shape every stale constant carried.
//
// Exemptions, because a year is sometimes the fact itself: fields whose NAME
// contains the cited year (nol_2019_remaining), and projects declaring no
// tax_year field at all, which have nothing to check against.
func AnalyzeDatedConstants(xmlDir string) ([]DatedConstantWarning, error) {
	taxYear := findDeclaredTaxYear(xmlDir)
	if taxYear == "" {
		return nil, nil
	}

	var warnings []DatedConstantWarning
	err := filepath.WalkDir(xmlDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), "_edd.xml") || loader.SkipRuleFile(p) {
			return nil
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		var doc eddDoc
		if xml.Unmarshal(data, &doc) != nil {
			return nil
		}
		for _, ent := range doc.Entities {
			for _, f := range ent.Fields {
				years := yearPattern.FindAllString(f.Comment, -1)
				if len(years) == 0 {
					continue
				}
				cited := dedupeSorted(years)
				ok := false
				for _, y := range cited {
					if y == taxYear || strings.Contains(f.Name, y) {
						ok = true
						break
					}
				}
				if !ok {
					warnings = append(warnings, DatedConstantWarning{
						Field:   ent.Name + "." + f.Name,
						Years:   cited,
						TaxYear: taxYear,
						EddFile: filepath.Base(p),
					})
				}
			}
		}
		return nil
	})
	sort.Slice(warnings, func(i, j int) bool { return warnings[i].Field < warnings[j].Field })
	return warnings, err
}

type eddDoc struct {
	Entities []struct {
		Name   string `xml:"name,attr"`
		Fields []struct {
			Name    string `xml:"name,attr"`
			Default string `xml:"default_value,attr"`
			Comment string `xml:"comment,attr"`
		} `xml:"field"`
	} `xml:"entity"`
}

// findDeclaredTaxYear reads the project's tax year from a field named
// tax_year on any entity — TaxReturn declares `job.tax_year` default 2025.
func findDeclaredTaxYear(xmlDir string) string {
	year := ""
	_ = filepath.WalkDir(xmlDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), "_edd.xml") || loader.SkipRuleFile(p) {
			return nil
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		var doc eddDoc
		if xml.Unmarshal(data, &doc) != nil {
			return nil
		}
		for _, ent := range doc.Entities {
			for _, f := range ent.Fields {
				if strings.EqualFold(f.Name, "tax_year") && yearPattern.MatchString(f.Default) {
					year = f.Default
				}
			}
		}
		return nil
	})
	return year
}

func dedupeSorted(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}
