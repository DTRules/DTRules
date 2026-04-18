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

package authoring_test

import (
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

// makeTrace builds a RunTrace with the given invocations for use in assertion tests.
func makeTrace(invs ...*authoring.TableInvocation) *authoring.RunTrace {
	return &authoring.RunTrace{
		EntryTable:  "Entry",
		Invocations: invs,
		InputState:  map[string]string{},
		FinalState:  map[string]string{},
	}
}

func inv(index int, name string, depth int, column int) *authoring.TableInvocation {
	return &authoring.TableInvocation{
		Index:      index,
		TableName:  name,
		Depth:      depth,
		Column:     column,
		StartState: map[string]string{},
		EndState:   map[string]string{},
	}
}

// --- AssertVisited ---

func TestAssertVisited_Found(t *testing.T) {
	tr := makeTrace(
		inv(0, "Entry", 0, 0),
		inv(1, "Check_Eligibility", 1, 1),
	)
	if err := tr.AssertVisited("Check_Eligibility", 1); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestAssertVisited_NotFound(t *testing.T) {
	tr := makeTrace(
		inv(0, "Entry", 0, 0),
		inv(1, "Score_Tiers", 1, 2),
	)
	err := tr.AssertVisited("Check_Eligibility", 1)
	if err == nil {
		t.Fatal("expected error for missing table, got nil")
	}
	if !strings.Contains(err.Error(), "Check_Eligibility") {
		t.Errorf("error should name the table, got: %v", err)
	}
	if !strings.Contains(err.Error(), "1") {
		t.Errorf("error should name the column, got: %v", err)
	}
}

func TestAssertVisited_AnyColumn(t *testing.T) {
	tr := makeTrace(
		inv(0, "Entry", 0, 0),
		inv(1, "Check_Eligibility", 1, 3),
	)
	// column == 0 should match regardless of the actual column
	if err := tr.AssertVisited("Check_Eligibility", 0); err != nil {
		t.Errorf("column==0 should match any invocation of the table, got: %v", err)
	}
}

// --- AssertNotVisited ---

func TestAssertNotVisited_Pass(t *testing.T) {
	tr := makeTrace(
		inv(0, "Entry", 0, 0),
		inv(1, "Score_Tiers", 1, 1),
	)
	if err := tr.AssertNotVisited("Check_Eligibility"); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestAssertNotVisited_Fail(t *testing.T) {
	tr := makeTrace(
		inv(0, "Entry", 0, 0),
		inv(1, "Check_Eligibility", 1, 1),
	)
	err := tr.AssertNotVisited("Check_Eligibility")
	if err == nil {
		t.Fatal("expected error when table was visited, got nil")
	}
	if !strings.Contains(err.Error(), "Check_Eligibility") {
		t.Errorf("error should name the table, got: %v", err)
	}
}

// --- AssertSequence ---

func TestAssertSequence_InOrder(t *testing.T) {
	tr := makeTrace(
		inv(0, "Entry", 0, 0),
		inv(1, "Check_Eligibility", 1, 1),
		inv(2, "Score_Tiers", 1, 1),
		inv(3, "Finalize", 1, 1),
	)
	if err := tr.AssertSequence([]string{"Check_Eligibility", "Score_Tiers", "Finalize"}); err != nil {
		t.Errorf("expected no error for in-order sequence, got: %v", err)
	}
}

func TestAssertSequence_OutOfOrder(t *testing.T) {
	tr := makeTrace(
		inv(0, "Entry", 0, 0),
		inv(1, "Score_Tiers", 1, 1),
		inv(2, "Check_Eligibility", 1, 1),
	)
	// Requesting Check_Eligibility before Score_Tiers — that's reversed in the trace.
	err := tr.AssertSequence([]string{"Check_Eligibility", "Score_Tiers"})
	if err == nil {
		t.Fatal("expected error for out-of-order sequence, got nil")
	}
	if !strings.Contains(err.Error(), "Score_Tiers") {
		t.Errorf("error should name the out-of-order table, got: %v", err)
	}
	if !strings.Contains(err.Error(), "Check_Eligibility") {
		t.Errorf("error should name the preceding table, got: %v", err)
	}
}

func TestAssertSequence_Missing(t *testing.T) {
	tr := makeTrace(
		inv(0, "Entry", 0, 0),
		inv(1, "Check_Eligibility", 1, 1),
	)
	err := tr.AssertSequence([]string{"Check_Eligibility", "Finalize"})
	if err == nil {
		t.Fatal("expected error for missing table in sequence, got nil")
	}
	if !strings.Contains(err.Error(), "Finalize") {
		t.Errorf("error should name the missing table, got: %v", err)
	}
}
