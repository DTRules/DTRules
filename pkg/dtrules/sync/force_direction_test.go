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

import "testing"

// `dtrules sync import` and `dtrules sync export` name a direction in the
// command itself, but both called SyncAll and let per-workbook timestamp
// detection decide. So each silently did nothing whenever the timestamps
// disagreed with the request -- "No files needed export (Excel is up to
// date)", exit 0 -- and --force did not help, because it waives the
// pending-user-edits check rather than direction detection (#1069).

func TestSetForceDirectionOverridesDetection(t *testing.T) {
	s := NewSyncerWithOptions("xml", "excel", DefaultOptions())

	if got := s.options.ForceDirection; got != NoSync {
		t.Fatalf("a fresh syncer should detect, not force; got %v", got)
	}

	s.SetForceDirection(XMLToExcel)
	if got := s.options.ForceDirection; got != XMLToExcel {
		t.Errorf("ForceDirection = %v, want XMLToExcel — `sync export` would "+
			"fall back to detection and export nothing", got)
	}

	s.SetForceDirection(ExcelToXML)
	if got := s.options.ForceDirection; got != ExcelToXML {
		t.Errorf("ForceDirection = %v, want ExcelToXML — `sync import` would "+
			"fall back to detection and import nothing", got)
	}
}

// NoSync is the "detect" value, so setting it back has to restore detection
// rather than pin some third behaviour.
func TestSetForceDirectionBackToNoSyncRestoresDetection(t *testing.T) {
	s := NewSyncerWithOptions("xml", "excel", DefaultOptions())
	s.SetForceDirection(XMLToExcel)
	s.SetForceDirection(NoSync)

	if got := s.options.ForceDirection; got != NoSync {
		t.Errorf("ForceDirection = %v, want NoSync (detect)", got)
	}
}
