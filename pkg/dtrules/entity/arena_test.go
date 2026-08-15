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

package entity_test

import (
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

func newArenaHarness(t *testing.T) (dtrules.Session, dtrules.ScratchAllocator) {
	t.Helper()
	rs := session.NewRuleSet("arena")
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	ef := sess.GetEntityFactory().(*entity.Factory)
	ref, err := ef.FindCreateRefEntity(true, dtrules.GetRName("combo"))
	if err != nil {
		t.Fatal(err)
	}
	ref.AddAttribute(dtrules.GetRName("count"), "", dtrules.GetRIntegerValue(0), true, true, dtrules.TypeInteger, "", "", "", "")
	ref.AddAttribute(dtrules.GetRName("sum"), "", dtrules.GetRIntegerValue(0), true, true, dtrules.TypeInteger, "", "", "", "")

	sa, ok := sess.(dtrules.ScratchAllocator)
	if !ok {
		t.Fatal("RSession must implement dtrules.ScratchAllocator")
	}
	return sess, sa
}

// TestArenaRecyclesInstances: after a reset, the next allocation of the same
// type is the same Go object — with a fresh unique ID and fully re-defaulted
// values. Indistinguishable from a new clone, minus the allocations.
func TestArenaRecyclesInstances(t *testing.T) {
	_, sa := newArenaHarness(t)
	sa.EnableScratch()

	comboName := dtrules.GetRName("combo")
	first, err := sa.CreateScratchEntity(comboName)
	if err != nil {
		t.Fatalf("CreateScratchEntity: %v", err)
	}
	firstID := first.GetID()
	first.Put(dtrules.GetRName("sum"), dtrules.GetRIntegerValue(15))

	sa.ResetScratch()

	second, err := sa.CreateScratchEntity(comboName)
	if err != nil {
		t.Fatalf("CreateScratchEntity after reset: %v", err)
	}
	if second.(*entity.REntity) != first.(*entity.REntity) {
		t.Error("expected the retired instance to be recycled")
	}
	if second.GetID() == firstID {
		t.Errorf("recycled instance kept its old ID %d — traces would see two incarnations under one ID", firstID)
	}
	v, err := second.Get(dtrules.GetRName("sum"))
	if err != nil {
		t.Fatalf("Get on recycled: %v", err)
	}
	if n, _ := v.IntValue(); n != 0 {
		t.Errorf("recycled instance leaked a previous value: sum = %d, want the default 0", n)
	}
}

// TestArenaStaleUseFailsLoudly: the escape hatch. A reference held across
// ResetScratch must error at next use — never silently read recycled
// storage.
func TestArenaStaleUseFailsLoudly(t *testing.T) {
	_, sa := newArenaHarness(t)
	sa.EnableScratch()

	combo, err := sa.CreateScratchEntity(dtrules.GetRName("combo"))
	if err != nil {
		t.Fatal(err)
	}
	sa.ResetScratch()

	if _, err := combo.Get(dtrules.GetRName("sum")); err == nil {
		t.Error("Get on a stale scratch entity must error")
	} else if !strings.Contains(err.Error(), "ResetScratch") {
		t.Errorf("the error should name the cause, got: %v", err)
	}
	if err := combo.Put(dtrules.GetRName("sum"), dtrules.GetRIntegerValue(1)); err == nil {
		t.Error("Put on a stale scratch entity must error")
	}
}

// TestArenaDisabledFallsBack: without EnableScratch, the scratch methods
// allocate ordinarily, nothing errors, and ResetScratch is a no-op — the
// zero-change path for existing hosts.
func TestArenaDisabledFallsBack(t *testing.T) {
	_, sa := newArenaHarness(t)

	if sa.ScratchEnabled() {
		t.Fatal("scratch must be off by default")
	}
	combo, err := sa.CreateScratchEntity(dtrules.GetRName("combo"))
	if err != nil {
		t.Fatalf("fallback CreateScratchEntity: %v", err)
	}
	sa.ResetScratch() // no-op
	if _, err := combo.Get(dtrules.GetRName("sum")); err != nil {
		t.Errorf("an ordinary entity must survive a no-op reset: %v", err)
	}
	arr, err := sa.CreateScratchArray()
	if err != nil || arr == nil {
		t.Fatalf("fallback CreateScratchArray: %v", err)
	}
}

// TestArenaRecyclesArrays: retired arrays come back empty.
func TestArenaRecyclesArrays(t *testing.T) {
	sess, sa := newArenaHarness(t)
	sa.EnableScratch()
	_ = sess

	arr, err := sa.CreateScratchArray()
	if err != nil {
		t.Fatal(err)
	}
	arr.Add(dtrules.GetRIntegerValue(42))
	sa.ResetScratch()

	again, err := sa.CreateScratchArray()
	if err != nil {
		t.Fatal(err)
	}
	if again != arr {
		t.Error("expected the retired array to be recycled")
	}
	if again.Size() != 0 {
		t.Errorf("recycled array not empty: %d elements", again.Size())
	}
}
