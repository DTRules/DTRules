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

package sync

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/excel"
)

// EDDConstraintFinding is one field whose declaration contradicts itself:
// the value constraints it carries (#1209) cannot be satisfied by its own
// default, or the limits themselves are not numbers.
//
// A default outside a field's own vocabulary is a genuine defect rather than
// a style question — the field starts every run holding a value the rules
// were told it can never hold, and no input can correct it, because a
// supplied value is the one thing that is checked.
type EDDConstraintFinding struct {
	File   string // EDD file that declares the field
	Entity string // declaring entity
	Field  string // field name, in its authored spelling
	Detail string // what is wrong, naming the value and the limit
}

func (f EDDConstraintFinding) String() string {
	return fmt.Sprintf("%s: %s.%s: %s", filepath.Base(f.File), f.Entity, f.Field, f.Detail)
}

// ValidateEDDConstraints checks every EDD under xmlDir for fields whose
// declared default cannot satisfy the constraints declared beside it. It
// reads the XML directly rather than loading the rule set, so a project that
// does not yet compile is still checked.
func ValidateEDDConstraints(xmlDir string) ([]EDDConstraintFinding, error) {
	files, err := findEDDFiles(xmlDir)
	if err != nil {
		return nil, err
	}
	var findings []EDDConstraintFinding
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", file, err)
		}
		var edd excel.EDDXML
		if err := xml.Unmarshal(data, &edd); err != nil {
			// Malformed XML is somebody else's finding; the structure and EL
			// checks already report it and reporting it twice helps nobody.
			continue
		}
		for _, ent := range edd.Entities {
			if ent == nil {
				continue
			}
			for _, f := range ent.Fields {
				if f == nil {
					continue
				}
				if detail := checkFieldConstraints(f); detail != "" {
					findings = append(findings, EDDConstraintFinding{
						File: file, Entity: ent.Name, Field: f.Name, Detail: detail,
					})
				}
			}
		}
	}
	return findings, nil
}

// checkFieldConstraints returns what is wrong with one field's constraints,
// or "" when nothing is.
func checkFieldConstraints(f *excel.EDDXMLField) string {
	maxLength, ok := parseEDDLimit(f.MaxLength)
	if !ok {
		return fmt.Sprintf("max_length %q is not a non-negative whole number", f.MaxLength)
	}
	maxWords, ok := parseEDDLimit(f.MaxWords)
	if !ok {
		return fmt.Sprintf("max_words %q is not a non-negative whole number", f.MaxWords)
	}
	c := &entity.FieldConstraints{MaxLength: maxLength, MaxWords: maxWords}
	for _, v := range f.AllowedValues {
		if v != nil && strings.TrimSpace(v.Value) != "" {
			c.AllowedValues = append(c.AllowedValues, strings.TrimSpace(v.Value))
		}
	}
	if c.IsEmpty() || strings.TrimSpace(f.DefaultValue) == "" {
		return ""
	}
	if err := c.Check("default", f.DefaultValue); err != nil {
		// Check names the subject first; here the subject is the default, and
		// the caller already names entity and field.
		return strings.TrimPrefix(err.Error(), "default: ")
	}
	return ""
}

// parseEDDLimit reads a max_length / max_words attribute. Empty is no limit.
func parseEDDLimit(v string) (int, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, true
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

// findEDDFiles finds all EDD XML files in a directory.
func findEDDFiles(xmlDir string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(xmlDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); name == "testfiles" || name == "schemas" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), "_edd.xml") {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}
