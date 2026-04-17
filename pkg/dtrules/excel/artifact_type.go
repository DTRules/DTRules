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
	"path/filepath"
	"strings"
)

// ArtifactTypeFromFilename is the exported version of artifactTypeFromFilename.
func ArtifactTypeFromFilename(path string) (string, bool) {
	return artifactTypeFromFilename(path)
}

// ArtifactTypeMismatch is the exported version of artifactTypeMismatch.
func ArtifactTypeMismatch(suffix string, hasdt, hasedd, hasmap bool) string {
	return artifactTypeMismatch(suffix, hasdt, hasedd, hasmap)
}

// artifactTypeFromFilename returns the artifact type encoded in a workbook filename.
// Returns ("dt", true), ("edd", true), ("map", true), or ("", false) for mixed/unknown.
//
// Examples:
//
//	"001_Compute_Tax_Return_dt.xlsx"  → ("dt", true)
//	"TaxReturn_edd.xlsx"              → ("edd", true)
//	"TaxReturn_map.xlsx"              → ("map", true)
//	"TaxReturn.xlsx"                  → ("", false)   — mixed artifact
//	"states/CO.xlsx"                  → ("", false)   — mixed artifact
func artifactTypeFromFilename(path string) (string, bool) {
	base := strings.ToLower(strings.TrimSuffix(filepath.Base(path), ".xlsx"))
	switch {
	case strings.HasSuffix(base, "_dt"):
		return "dt", true
	case strings.HasSuffix(base, "_edd"):
		return "edd", true
	case strings.HasSuffix(base, "_map"):
		return "map", true
	default:
		return "", false
	}
}

// artifactTypeMismatch returns an error message if the filename suffix disagrees
// with the detected sheet content, or empty string if they agree.
// suffix is one of "dt", "edd", "map", "". sheetTypes is a set of detected types.
func artifactTypeMismatch(suffix string, hasdt, hasedd, hasmap bool) string {
	switch suffix {
	case "dt":
		if hasedd || hasmap {
			return "workbook has _dt suffix but contains non-DT sheets (EDD or MAP)"
		}
	case "edd":
		if hasdt || hasmap {
			return "workbook has _edd suffix but contains non-EDD sheets (DT or MAP)"
		}
	case "map":
		if hasdt || hasedd {
			return "workbook has _map suffix but contains non-MAP sheets (DT or EDD)"
		}
	}
	return ""
}

// suffixForResult returns the appropriate filename suffix for exporting a single-
// artifact WorkbookResult. Returns "" for mixed-artifact results.
func suffixForResult(r *WorkbookResult) string {
	hasDT := r.DTables != nil && len(r.DTables.Tables) > 0
	hasEDD := r.EDD != nil && len(r.EDD.Entities) > 0
	hasMAP := r.Map != nil && len(r.Map.Entries) > 0
	count := 0
	if hasDT {
		count++
	}
	if hasEDD {
		count++
	}
	if hasMAP {
		count++
	}
	if count != 1 {
		return "" // mixed — no suffix
	}
	switch {
	case hasDT:
		return "_dt"
	case hasEDD:
		return "_edd"
	case hasMAP:
		return "_map"
	}
	return ""
}
