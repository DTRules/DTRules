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

package dtrules

// ScratchAllocator is the scratch-generation arena a session may offer
// (#1025). Everything the combinatorial generators materialize — combo,
// group, and run entities plus their members arrays — lives exactly as long
// as one table execution, which is what makes ~90% of a small table's cost
// recyclable. A host that scores in a loop opts in once, then resets between
// executions:
//
//	sess.EnableScratch()
//	for _, hand := range hands {
//	    … execute …
//	    sess.ResetScratch()
//	}
//
// The contract:
//
//   - Opt-in. Until EnableScratch is called, CreateScratch* fall back to
//     ordinary allocation and ResetScratch is a no-op, so nothing changes
//     for existing hosts — and nothing is pinned in memory by an arena
//     nobody resets.
//   - ResetScratch invalidates every scratch object handed out since the
//     previous reset. Using a stale scratch entity afterwards is a runtime
//     error naming the cause, never a silent read of recycled storage —
//     that is the answer to the escape hatch where a table stores a combo
//     into durable state: it fails loudly at next use.
//   - Recycled entities get a fresh unique ID on every reuse, so entity IDs
//     in traces never refer to two incarnations.
//   - Sessions are single-threaded; the arena inherits that and takes no
//     locks.
type ScratchAllocator interface {
	// ScratchEnabled reports whether EnableScratch has been called.
	ScratchEnabled() bool

	// EnableScratch activates the arena for this session. Idempotent.
	EnableScratch()

	// CreateScratchEntity allocates an entity of the named EDD type from
	// the arena (recycled when possible), or ordinarily when the arena is
	// not enabled.
	CreateScratchEntity(name *RName) (Entity, error)

	// CreateScratchArray allocates a data array from the arena (recycled
	// when possible), or ordinarily when the arena is not enabled.
	CreateScratchArray() (*RArray, error)

	// ResetScratch recycles everything handed out since the last reset and
	// advances the generation. No-op when the arena is not enabled.
	ResetScratch()
}
