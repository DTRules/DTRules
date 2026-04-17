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

package excel

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// ExtractExcel walks src and writes every *_dt.xml, *_edd.xml, and *_map.xml
// back to an equivalent xlsx under dst, preserving subdirectory layout.
// src is typically an embed.FS; dst is a filesystem directory path.
// Extracted trees can be re-authored and round-tripped back through
// `dtrules build`.
//
// Limitation: the loader currently requires a session/entity context to compile
// postfix expressions, so each _edd.xml and _dt.xml is loaded into an isolated
// RuleSet. Tables that reference entities defined in other files may produce
// loader warnings but will still be extracted. See issue #541 for the fs.FS
// loader follow-up that will remove this limitation.
func ExtractExcel(src fs.FS, dst string) error {
	return fs.WalkDir(src, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		name := d.Name()
		switch {
		case strings.HasSuffix(name, "_map.xml"):
			return extractMap(src, path, dst)
		case strings.HasSuffix(name, "_edd.xml"):
			return extractEDD(src, path, dst)
		case strings.HasSuffix(name, "_dt.xml"):
			return extractDT(src, path, dst)
		}
		return nil
	})
}

// xlsxPath converts a source relative path (foo/bar_dt.xml) to the
// destination xlsx path under dst (dst/foo/bar_dt.xlsx).
func xlsxPath(dst, relPath string) string {
	noExt := strings.TrimSuffix(relPath, ".xml")
	return filepath.Join(dst, filepath.FromSlash(noExt+".xlsx"))
}

func extractMap(src fs.FS, relPath, dst string) error {
	data, err := fs.ReadFile(src, relPath)
	if err != nil {
		return fmt.Errorf("extract %s: %w", relPath, err)
	}

	m, err := loadMapXMLFromReader(bytes.NewReader(data), relPath)
	if err != nil {
		return fmt.Errorf("extract %s: %w", relPath, err)
	}

	out := xlsxPath(dst, relPath)
	if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
		return fmt.Errorf("extract %s: %w", relPath, err)
	}

	exp := NewMapExporter()
	if err := exp.ExportToFile(m, out); err != nil {
		return fmt.Errorf("extract %s: %w", relPath, err)
	}
	return nil
}

func extractEDD(src fs.FS, relPath, dst string) error {
	data, err := fs.ReadFile(src, relPath)
	if err != nil {
		return fmt.Errorf("extract %s: %w", relPath, err)
	}

	rs := session.NewRuleSet("extract")
	if rs == nil {
		return fmt.Errorf("extract %s: failed to create rule set", relPath)
	}

	if err := rs.LoadEDD(bytes.NewReader(data)); err != nil {
		return fmt.Errorf("extract %s: %w", relPath, err)
	}

	out := xlsxPath(dst, relPath)
	if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
		return fmt.Errorf("extract %s: %w", relPath, err)
	}

	exp := NewExporter(rs)
	if err := exp.ExportEDD(out); err != nil {
		return fmt.Errorf("extract %s: %w", relPath, err)
	}
	return nil
}

func extractDT(src fs.FS, relPath, dst string) error {
	data, err := fs.ReadFile(src, relPath)
	if err != nil {
		return fmt.Errorf("extract %s: %w", relPath, err)
	}

	rs := session.NewRuleSet("extract")
	if rs == nil {
		return fmt.Errorf("extract %s: failed to create rule set", relPath)
	}

	// The DT loader compiles EL expressions; warnings may appear for tables
	// referencing entities not in this isolated RuleSet (issue #541).
	if err := rs.LoadDecisionTables(bytes.NewReader(data)); err != nil {
		return fmt.Errorf("extract %s: %w", relPath, err)
	}

	out := xlsxPath(dst, relPath)
	if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
		return fmt.Errorf("extract %s: %w", relPath, err)
	}

	exp := NewExporter(rs)
	if err := exp.ExportDecisionTables(out); err != nil {
		return fmt.Errorf("extract %s: %w", relPath, err)
	}
	return nil
}
