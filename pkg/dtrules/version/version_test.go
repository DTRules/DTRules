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

package version

import (
	"strings"
	"testing"
)

func TestDefaultValues(t *testing.T) {
	if Version != "dev" {
		t.Errorf("Version default = %q, want %q", Version, "dev")
	}
	if Commit != "" {
		t.Errorf("Commit default = %q, want empty string", Commit)
	}
	if Date != "" {
		t.Errorf("Date default = %q, want empty string", Date)
	}
}

func TestInfo(t *testing.T) {
	info := Info()
	if !strings.Contains(info, "DTRules") {
		t.Errorf("Info() = %q, want it to contain 'DTRules'", info)
	}
}

func TestFull(t *testing.T) {
	full := Full()
	if !strings.Contains(full, "DTRules") {
		t.Errorf("Full() = %q, want it to contain 'DTRules'", full)
	}
}

func TestShort(t *testing.T) {
	short := Short()
	if short == "" {
		t.Error("Short() returned empty string, want at least 'dev'")
	}
}

func TestFullWithInjectedValues(t *testing.T) {
	orig := Commit
	origDate := Date
	defer func() {
		Commit = orig
		Date = origDate
	}()

	Commit = "abc1234"
	Date = "2024-01-01T00:00:00Z"

	full := Full()
	if !strings.Contains(full, "abc1234") {
		t.Errorf("Full() = %q, want it to contain commit 'abc1234'", full)
	}
	if !strings.Contains(full, "2024-01-01T00:00:00Z") {
		t.Errorf("Full() = %q, want it to contain date '2024-01-01T00:00:00Z'", full)
	}
}
