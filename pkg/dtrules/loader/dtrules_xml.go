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

package loader

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

// ProjectConfig holds top-level settings parsed from a DTRules.xml file.
//
// Phase 2 of #743 introduces the <default_timezone> element. Operators that
// have no explicit `in zone <strexpr>` clause and no field-level zone fall
// back to this value. If absent, callers should default to UTC.
type ProjectConfig struct {
	XMLName         xml.Name `xml:"DTRules"`
	DefaultTimezone string   `xml:"default_timezone"`
}

// GetDefaultTimezone returns the configured default timezone, or "UTC" when
// the file did not declare one.
func (c *ProjectConfig) GetDefaultTimezone() string {
	tz := strings.TrimSpace(c.DefaultTimezone)
	if tz == "" {
		return "UTC"
	}
	return tz
}

// LoadProjectConfig parses a DTRules.xml document from r and returns the
// extracted ProjectConfig. Unrelated elements (RuleSet, compileralias, etc.)
// are ignored — only the keys this package cares about are populated.
func LoadProjectConfig(r io.Reader) (*ProjectConfig, error) {
	if MaxXMLSize > 0 {
		r = io.LimitReader(r, MaxXMLSize+1)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read DTRules.xml: %w", err)
	}
	if MaxXMLSize > 0 && int64(len(data)) > MaxXMLSize {
		return nil, fmt.Errorf("DTRules.xml exceeds maximum size limit of %d bytes", MaxXMLSize)
	}
	cfg := &ProjectConfig{}
	if err := xml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse DTRules.xml: %w", err)
	}
	return cfg, nil
}

// LoadProjectConfigFile is a convenience wrapper around LoadProjectConfig
// that opens a file by path.
func LoadProjectConfigFile(path string) (*ProjectConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open DTRules.xml at %s: %w", path, err)
	}
	defer f.Close()
	return LoadProjectConfig(f)
}
