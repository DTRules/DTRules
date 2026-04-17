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
)

// ExtractExcel walks src and writes every *_dt.xml, *_edd.xml, and *_map.xml
// to an equivalent xlsx under dst, preserving subdirectory layout.
// This is a pure XML→xlsx representation conversion; no session or RuleSet is required.
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

	edd, err := UnmarshalEDDXML(data)
	if err != nil {
		return fmt.Errorf("extract %s: %w", relPath, err)
	}

	out := xlsxPath(dst, relPath)
	if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
		return fmt.Errorf("extract %s: %w", relPath, err)
	}

	if err := WriteEDDXMLToExcel(edd, out); err != nil {
		return fmt.Errorf("extract %s: %w", relPath, err)
	}
	return nil
}

func extractDT(src fs.FS, relPath, dst string) error {
	data, err := fs.ReadFile(src, relPath)
	if err != nil {
		return fmt.Errorf("extract %s: %w", relPath, err)
	}

	dt, err := UnmarshalDecisionTablesXML(data)
	if err != nil {
		return fmt.Errorf("extract %s: %w", relPath, err)
	}

	out := xlsxPath(dst, relPath)
	if err := os.MkdirAll(filepath.Dir(out), 0755); err != nil {
		return fmt.Errorf("extract %s: %w", relPath, err)
	}

	if err := WriteDTXMLToExcel(dt, out); err != nil {
		return fmt.Errorf("extract %s: %w", relPath, err)
	}
	return nil
}
