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

package authoring

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/DTRules/DTRules/pkg/dtrules/excel"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
	"github.com/DTRules/DTRules/pkg/dtrules/sync"
)

// Package-level helpers that implement the "Excel is system of record"
// contract for any caller working with a DTRules project on disk.
// Project.Save / SaveEDD wrap them; dtrules compile (which does
// byte-level XML rewriting outside the Project model) calls them
// directly around its compile loop.
//
// Both helpers are no-ops on projects with no `.sync-manifest.json`
// reachable from the XML directory — that's the legacy / flat layout
// where Excel isn't in play, and the legacy Save behavior must stay
// preserved.

// GuardExcelInDir runs the manifest-driven Excel-mtime guard against
// the project rooted at xmlDir. If `overwrite` is false (the default)
// and any covered Excel file has been touched since the last export,
// the guard returns a `*sync.ExcelModifiedError` naming the file. Pass
// true to bypass the mtime guard while still running the lock-file
// detection — lock files always block writes because writing through
// an open spreadsheet corrupts the on-disk state from the other app's
// perspective.
//
// Returns nil on no-manifest projects.
func GuardExcelInDir(xmlDir string, overwrite bool) error {
	return GuardExcelIn(xmlDir, "", overwrite)
}

// GuardExcelIn is GuardExcelInDir with the project's declared Excel directory.
// Pass "" to fall back to searching the conventional layouts.
//
// The search is adjacency-based, so a project whose rules sit beside a second,
// unused workbook directory guards — and writes — the wrong one. Staking has
// exactly that shape: excel/ is declared and used by build and verify, while a
// stale pkg/dtrules/excel/ sits next to the rules and won the search (#1049).
func GuardExcelIn(xmlDir, excelDir string, overwrite bool) error {
	m, manifestDir := loadSyncManifest(xmlDir, excelDir)
	if m == nil {
		return nil
	}
	for excelRel := range m.Files {
		excelPath := filepath.Join(manifestDir, excelRel)
		if err := excelLockError(excelPath); err != nil {
			return err
		}
		if overwrite {
			continue
		}
		if err := m.ExportGuard(excelPath); err != nil {
			return err
		}
	}
	return nil
}

// RefreshExcelInDir re-exports every Excel file the project's sync
// manifest covers from the current on-disk XML state. Loads a
// tolerant RuleSet so the export works even when the operator is
// mid-edit (DSL added but not yet compiled to postfix).
//
// Idempotent: a second call right after the first writes the same
// bytes. Called by Save / SaveEDD / dtrules compile after their
// respective XML writes; the manifest's RecordExport refreshes
// LastExportTime so the next GuardExcelInDir starts from a clean
// baseline.
//
// Returns nil on no-manifest projects.
func RefreshExcelInDir(xmlDir string) error {
	return RefreshExcelIn(xmlDir, "")
}

// RefreshExcelIn is RefreshExcelInDir with the project's declared Excel
// directory. Pass "" to search the conventional layouts (#1049).
func RefreshExcelIn(xmlDir, excelDir string) error {
	m, manifestDir := loadSyncManifest(xmlDir, excelDir)
	if m == nil {
		return nil
	}
	rs, err := loadRuleSetForExportInDir(xmlDir)
	if err != nil {
		return fmt.Errorf("excel refresh: load ruleset: %w", err)
	}
	exporter := excel.NewExporter(rs)
	for excelRel, entry := range m.Files {
		excelPath := filepath.Join(manifestDir, excelRel)
		if err := exporter.ExportDecisionTables(excelPath); err != nil {
			return fmt.Errorf("excel refresh: export %s: %w", excelPath, err)
		}
		if err := m.RecordExport(excelPath, entry.XMLFiles); err != nil {
			return fmt.Errorf("excel refresh: record manifest for %s: %w", excelPath, err)
		}
	}
	return nil
}

// LoadSyncManifestForXMLDir searches the conventional layouts for the
// `.sync-manifest.json` that pairs Excel files with their XML exports.
// Returns the manifest plus the directory it lives in (manifest paths
// are relative to that directory), or (nil, "") if no manifest is
// reachable.
//
// Search order:
//  1. `<xmlDir>/.sync-manifest.json` — canonical xml/+excel/-as-siblings layout.
//  2. `<xmlDir>/../source/.sync-manifest.json` — staking's convention
//     (rules/ and source/ as siblings under pkg/dtrules).
//  3. `<xmlDir>/../excel/.sync-manifest.json` — TaxReturn-style layout.
//  4. `<xmlDir>/../.sync-manifest.json` — flat catch-all.
//
// First match wins. Exported so callers in cmd/dtrules can probe
// before deciding whether to enforce the Excel contract.
func LoadSyncManifestForXMLDir(xmlDir string) (*sync.Manifest, string) {
	parent := filepath.Dir(xmlDir)
	candidates := []string{
		xmlDir,
		filepath.Join(parent, "source"),
		filepath.Join(parent, "excel"),
		parent,
	}
	for _, dir := range candidates {
		manifestPath := filepath.Join(dir, ".sync-manifest.json")
		if _, err := os.Stat(manifestPath); err != nil {
			continue
		}
		m, err := sync.LoadManifest(manifestPath)
		if err != nil {
			continue
		}
		return m, dir
	}
	return nil, ""
}

// excelLockError returns a non-nil error when an Excel file appears
// to be open in another process. Two conventions are recognized:
//   - `~$<name>.xlsx`        Microsoft Excel (Windows / macOS)
//   - `.~lock.<name>.xlsx#`  LibreOffice
//
// Writing to a workbook that's open in either application from outside
// the app corrupts it from the app's point of view. Refusing the
// write here is friendlier than the post-hoc cleanup users would
// otherwise face.
func excelLockError(excelPath string) error {
	dir := filepath.Dir(excelPath)
	base := filepath.Base(excelPath)
	candidates := []string{
		filepath.Join(dir, "~$"+base),
		filepath.Join(dir, ".~lock."+base+"#"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return fmt.Errorf("Excel file %q appears to be open (lock file %q present); close the spreadsheet app and retry, or remove the lock file if stale",
				excelPath, c)
		}
	}
	return nil
}

// loadRuleSetForExportInDir is the file-system-driven function used
// by RefreshExcelInDir. It globs EDD and DT files directly
// under xmlDir (the same set Project.loadDTFiles + Project.loadEDD
// would have produced) and loads them into a tolerant RuleSet. Used
// by callers that don't already have a Project in memory — chiefly
// dtrules compile, which does byte-level XML rewriting outside the
// Project model.
func loadRuleSetForExportInDir(xmlDir string) (*session.RuleSet, error) {
	rs := session.NewRuleSet("authoring-export")
	if rs == nil {
		return nil, fmt.Errorf("failed to create ruleset")
	}
	eddMatches, _ := filepath.Glob(filepath.Join(xmlDir, "*_edd.xml"))
	for _, eddPath := range eddMatches {
		if err := rs.LoadEDDFile(eddPath); err != nil {
			return nil, fmt.Errorf("load edd %s: %w", eddPath, err)
		}
	}
	// DT files are loaded recursively so nested layouts (e.g.
	// states/CA_dt.xml under xml/) get picked up the way the loader
	// would discover them at runtime.
	var dtPaths []string
	_ = filepath.WalkDir(xmlDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if filepath.Ext(name) != ".xml" {
			return nil
		}
		if filepath.Base(name) == "" {
			return nil
		}
		// _dt.xml only; skip _edd.xml (already loaded) and _map.xml.
		if len(name) >= 7 && name[len(name)-7:] == "_dt.xml" {
			dtPaths = append(dtPaths, p)
		}
		return nil
	})
	for _, dtPath := range dtPaths {
		if err := rs.LoadDecisionTablesTolerantFile(dtPath); err != nil {
			return nil, fmt.Errorf("load dt %s: %w", dtPath, err)
		}
	}
	return rs, nil
}

// loadSyncManifest prefers an explicitly declared Excel directory over the
// adjacency search, so a project that says where its workbooks live is
// believed (#1049).
func loadSyncManifest(xmlDir, excelDir string) (*sync.Manifest, string) {
	if excelDir != "" {
		path := filepath.Join(excelDir, ".sync-manifest.json")
		if _, err := os.Stat(path); err == nil {
			if m, err := sync.LoadManifest(path); err == nil {
				return m, excelDir
			}
		}
		// Declared but with no manifest: that is a project whose Excel has
		// never been exported, not an invitation to guard a different
		// directory's workbooks.
		return nil, ""
	}
	return LoadSyncManifestForXMLDir(xmlDir)
}
