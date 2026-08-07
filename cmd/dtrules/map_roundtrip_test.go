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

package main

import (
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/excel"
)

// #993: a workbook written before the `list` column existed reads back with
// createentity rows present but every List empty. The old guard counted rows,
// so it passed that through and the rewritten map lost `list=` on every
// createentity — which is what appends each parsed entity onto the collection
// the tables iterate. Every `for all` then ran over an empty array and the
// Accumulate staking ruleset returned 0 for every staked balance, while the
// build printed "Result: OK — no drops" and exited 0.
func TestListsLostDetectsPreListSheet(t *testing.T) {
	existing := []excel.MapCreateEntity{
		{Entity: "staking_account", Tag: "staking_account", ID: "account_url", List: "accounts"},
		{Entity: "balance_tx", Tag: "balance_tx", ID: "btx_seq", List: "balance_history"},
		{Entity: "job", Tag: "job", ID: "id"}, // legitimately has no list
	}

	preList := []excel.MapCreateEntity{
		{Entity: "staking_account", Tag: "staking_account", ID: "account_url"},
		{Entity: "balance_tx", Tag: "balance_tx", ID: "btx_seq"},
		{Entity: "job", Tag: "job", ID: "id"},
	}
	if !listsLost(preList, existing) {
		t.Fatal("a sheet that dropped every list= was not detected as lossy")
	}

	merged := mergeCreateEntityLists(preList, existing)
	if merged[0].List != "accounts" || merged[1].List != "balance_history" {
		t.Errorf("lists not restored: %+v", merged)
	}
	if merged[2].List != "" {
		t.Errorf("invented a list for a row that never had one: %+v", merged[2])
	}

	// A current sheet round-trips intact and must not be flagged.
	if listsLost(existing, existing) {
		t.Error("an intact sheet was reported as lossy")
	}

	// An author who deliberately removes a list in Excel is indistinguishable
	// from a lossy sheet, so the conservative choice is to keep the value and
	// warn — losing a production ruleset's data loading is the worse failure.
	// This pins that choice so a change to it is deliberate.
	oneRemoved := []excel.MapCreateEntity{
		{Entity: "staking_account", Tag: "staking_account", ID: "account_url"},
		{Entity: "balance_tx", Tag: "balance_tx", ID: "btx_seq", List: "balance_history"},
		{Entity: "job", Tag: "job", ID: "id"},
	}
	if !listsLost(oneRemoved, existing) {
		t.Error("a single dropped list= should still be caught")
	}
	if got := mergeCreateEntityLists(oneRemoved, existing)[0].List; got != "accounts" {
		t.Errorf("kept value = %q, want the existing \"accounts\"", got)
	}
}

// A renamed row is a different row: nothing should be carried onto it.
func TestMergeCreateEntityListsKeysOnIdentity(t *testing.T) {
	existing := []excel.MapCreateEntity{
		{Entity: "staking_account", Tag: "staking_account", ID: "account_url", List: "accounts"},
	}
	renamed := []excel.MapCreateEntity{
		{Entity: "staking_account", Tag: "renamed_tag", ID: "account_url"},
	}
	if listsLost(renamed, existing) {
		t.Error("a renamed tag must not look like a lost list")
	}
	if got := mergeCreateEntityLists(renamed, existing)[0].List; got != "" {
		t.Errorf("carried %q onto a renamed row", got)
	}
}
