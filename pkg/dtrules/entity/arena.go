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

package entity

import (
	"github.com/DTRules/DTRules/pkg/dtrules"
)

// Arena is the per-session scratch pool behind dtrules.ScratchAllocator
// (#1025). Instance construction is cheap to recycle because CloneEntity
// shares attribute definitions with the reference entity — per-instance
// state is one values slice and an ID — so reuse is a default re-copy plus
// a fresh unique ID, skipping the struct, map-header, and slice allocations
// that dominate a generator-heavy table execution.
//
// Sessions are single-threaded; the arena takes no locks.
type Arena struct {
	session dtrules.Session
	factory *Factory
	gen     uint64

	liveEnts []*REntity
	entPool  map[*dtrules.RName][]*REntity

	liveArrs []*dtrules.RArray
	arrPool  []*dtrules.RArray
}

// NewArena creates an empty arena bound to one session.
func NewArena(session dtrules.Session, factory *Factory) *Arena {
	return &Arena{
		session: session,
		factory: factory,
		entPool: make(map[*dtrules.RName][]*REntity),
	}
}

// Generation returns the current scratch generation. Entities stamped with
// an older generation are stale: they were recycled by a ResetScratch and
// must not be read or written.
func (a *Arena) Generation() uint64 { return a.gen }

// CreateEntity allocates a scratch entity of the named EDD type, recycling
// a retired instance when one is available. Every allocation — fresh or
// recycled — carries a new unique ID, so an ID in a trace never refers to
// two incarnations.
func (a *Arena) CreateEntity(name *dtrules.RName) (dtrules.Entity, error) {
	execName := name.GetExecutable().(*dtrules.RName)
	ref := a.factory.FindRefEntity(execName)
	if ref == nil {
		return nil, dtrules.UndefinedError("CreateScratchEntity", "Reference entity not found: "+name.StringValue())
	}

	if pool := a.entPool[execName]; len(pool) > 0 {
		e := pool[len(pool)-1]
		a.entPool[execName] = pool[:len(pool)-1]
		// Stamp the current generation before reinit: reinitFrom goes
		// through Put, and Put refuses stale scratch entities.
		e.scratchGen = a.gen
		e.id = a.session.GetUniqueID()
		e.readonly = false
		if err := e.reinitFrom(ref, a.session); err != nil {
			return nil, err
		}
		a.liveEnts = append(a.liveEnts, e)
		return e, nil
	}

	inst, err := CloneEntity(false, ref, a.session)
	if err != nil {
		return nil, err
	}
	inst.scratchArena = a
	inst.scratchGen = a.gen
	a.liveEnts = append(a.liveEnts, inst)
	return inst, nil
}

// CreateArray allocates a scratch data array (duplicates allowed, the shape
// every generator uses for members), recycling a retired one when possible.
func (a *Arena) CreateArray() (*dtrules.RArray, error) {
	if n := len(a.arrPool); n > 0 {
		arr := a.arrPool[n-1]
		a.arrPool = a.arrPool[:n-1]
		a.liveArrs = append(a.liveArrs, arr)
		return arr, nil
	}
	arr, err := dtrules.NewArray(a.session, true, false)
	if err != nil {
		return nil, err
	}
	a.liveArrs = append(a.liveArrs, arr)
	return arr, nil
}

// Reset retires everything handed out since the last reset and advances the
// generation. Retired entities keep their old generation stamp, which is
// exactly what makes any lingering reference to them fail loudly at next
// Get/Put instead of reading recycled storage.
func (a *Arena) Reset() {
	a.gen++
	for _, e := range a.liveEnts {
		a.entPool[e.name] = append(a.entPool[e.name], e)
	}
	a.liveEnts = a.liveEnts[:0]
	for _, arr := range a.liveArrs {
		arr.Clear()
		a.arrPool = append(a.arrPool, arr)
	}
	a.liveArrs = a.liveArrs[:0]
}
