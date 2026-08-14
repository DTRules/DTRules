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
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// WorkbookHash is the provenance stamp a compiled XML file carries: the hash
// of the workbook bytes it was compiled from.
//
// Hashing the file rather than its contents-as-parsed is deliberate. The
// question is "did this workbook change since the XML was generated", and the
// bytes answer it without needing to agree with the importer about what counts
// as a meaningful change. The exporter is byte-deterministic -- consecutive
// exports of the same rule set produce identical files -- so the stamp is
// stable across rebuilds that changed nothing (#1091).
//
// Returns "" when the file cannot be read. A missing hash means "unknown
// provenance", which callers must treat as "fall back to timestamps" rather
// than as "unchanged".
func WorkbookHash(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// RecordedWorkbookHash returns the provenance stamp the tables in this file
// agree on, or "" when they carry none or disagree.
//
// Disagreement is not an error to report here: a file assembled from more than
// one workbook has no single provenance, and the caller's fallback is the
// right answer for it.
func RecordedWorkbookHash(tables []DecisionTableXML) string {
	seen := ""
	for i := range tables {
		src := tables[i].Source
		if src == nil || src.SourceHash == "" {
			continue
		}
		if seen == "" {
			seen = src.SourceHash
			continue
		}
		if seen != src.SourceHash {
			return ""
		}
	}
	return seen
}
