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

// Package project resolves a DTRules project's settings: defaults first, then
// whatever DTRules.xml overrides, then whatever the caller overrides on top.
//
// It exists because that resolution used to happen in four places at once.
// cmd/dtrules knew xml_dir, excel_dir and entry; apiserver knew xml_dir;
// authoring knew xml_dir and excel_dir; loader knew default_timezone and
// nothing else. Each had its own struct and its own idea of the defaults, so a
// project could be found by one command and not another, and a field added to
// the manifest reached only whichever reader happened to learn about it
// (#1031, #1049, #1052).
//
// One resolved Config is the whole answer. Add a setting here and every layer
// gets it.
//
// This package deliberately depends on nothing but the standard library, so
// any layer can import it without a cycle.
package project

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ManifestName is the file every project is described by.
const ManifestName = "DTRules.xml"

// maxManifestSize bounds a manifest read from an untrusted reader. A settings
// file is a few hundred bytes; anything near this is not one.
const maxManifestSize = 1 << 20

// Config is a project's resolved settings. Every path is absolute.
//
// Construct it with Load, never by hand: the zero value has no defaults
// applied and will send callers to the wrong directories.
type Config struct {
	// Root is the project directory — the one holding DTRules.xml.
	Root string
	// XMLDir holds the rules (*_dt.xml, *_edd.xml, *_map.xml).
	XMLDir string
	// ExcelDir holds the .xlsx workbooks that are the system of record.
	ExcelDir string
	// Entry is the default entry decision table, so `run` and `debug` need
	// no --entry. Empty when undeclared.
	Entry string
	// DefaultTimezone applies to date operators with no explicit zone.
	// Always set; "UTC" when undeclared.
	DefaultTimezone string

	// Declared records which settings came from the manifest rather than
	// from a default, so callers can tell "the project asked for this" from
	// "nobody said". Keys are the element names.
	Declared map[string]bool
}

// manifest is the on-disk shape. Unknown elements are ignored by
// encoding/xml, which is what lets a project carry legacy declarations
// without breaking.
type manifest struct {
	XMLDir          string `xml:"xml_dir"`
	ExcelDir        string `xml:"excel_dir"`
	Entry           string `xml:"entry"`
	DefaultTimezone string `xml:"default_timezone"`

	// RuleSetFilePath is the legacy spelling of xml_dir, written
	// root-relative with a leading slash ("/xml"). Honoured when xml_dir is
	// absent so projects predating xml_dir keep working; xml_dir wins.
	//
	// Deliberately not aliased: DTExcelFolder and EDDExcelFolder name the
	// historical .xls authoring directories, not the .xlsx workbooks
	// ExcelDir means. Mapping them across would point the toolchain at the
	// wrong files.
	RuleSetFilePath string `xml:"RuleSet>RuleSetFilePath"`
}

// Load resolves a project's settings: defaults, then DTRules.xml.
//
// A missing or malformed manifest is not an error — the conventional layout is
// a reasonable answer, and `dtrules verify` reports a broken manifest properly
// rather than every command failing obscurely.
func Load(root string) *Config {
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}

	c := &Config{
		Root:            abs,
		XMLDir:          filepath.Join(abs, "xml"),
		ExcelDir:        filepath.Join(abs, "excel"),
		DefaultTimezone: "UTC",
		Declared:        map[string]bool{},
	}

	data, err := os.ReadFile(filepath.Join(abs, ManifestName))
	if err != nil {
		return c
	}
	c.apply(data)
	return c
}

// Parse resolves settings from a manifest document rather than from disk, for
// callers holding the bytes already. Root is left empty, so declared relative
// paths stay relative; use Load when paths matter.
func Parse(r io.Reader) (*Config, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxManifestSize+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", ManifestName, err)
	}
	if int64(len(data)) > maxManifestSize {
		return nil, fmt.Errorf("%s exceeds the %d byte limit", ManifestName, maxManifestSize)
	}
	c := &Config{DefaultTimezone: "UTC", Declared: map[string]bool{}}
	if err := xml.Unmarshal(data, &manifest{}); err != nil {
		return nil, fmt.Errorf("parse %s: %w", ManifestName, err)
	}
	c.apply(data)
	return c, nil
}

// apply overlays a manifest document onto already-defaulted settings. A
// document that will not parse leaves the defaults in place: see Load.
func (c *Config) apply(data []byte) {
	var m manifest
	if err := xml.Unmarshal(data, &m); err != nil {
		return
	}

	if dir := c.resolve(m.XMLDir); dir != "" {
		c.XMLDir = dir
		c.Declared["xml_dir"] = true
	} else if dir := c.resolve(strings.TrimPrefix(strings.TrimSpace(m.RuleSetFilePath), "/")); dir != "" {
		c.XMLDir = dir
		c.Declared["RuleSetFilePath"] = true
	}
	if dir := c.resolve(m.ExcelDir); dir != "" {
		c.ExcelDir = dir
		c.Declared["excel_dir"] = true
	}
	if v := strings.TrimSpace(m.Entry); v != "" {
		c.Entry = v
		c.Declared["entry"] = true
	}
	if v := strings.TrimSpace(m.DefaultTimezone); v != "" {
		c.DefaultTimezone = v
		c.Declared["default_timezone"] = true
	}
}

// resolve turns a declared path into an absolute one, or "" when undeclared.
func (c *Config) resolve(rel string) string {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return ""
	}
	if filepath.IsAbs(rel) {
		return rel
	}
	return filepath.Join(c.Root, rel)
}

// WithDirs applies caller overrides — CLI flags — on top of the resolved
// settings. Empty strings leave the resolved value alone, so the precedence is
// flags, then manifest, then defaults.
func (c *Config) WithDirs(xmlDir, excelDir string) *Config {
	if d := c.resolve(xmlDir); d != "" {
		c.XMLDir = d
		c.Declared["xml_dir_flag"] = true
	}
	if d := c.resolve(excelDir); d != "" {
		c.ExcelDir = d
		c.Declared["excel_dir_flag"] = true
	}
	return c
}
