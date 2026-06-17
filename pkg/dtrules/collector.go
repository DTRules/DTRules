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

package dtrules

// Collector is invoked at each entity-field read during execution to support
// interactive data collection (#850/#852). A non-nil collector turns on
// interactive mode; nil means batch execution — the read path is untouched
// and behaves exactly as today.
//
// The implementation lives outside the runtime (pkg/dtrules/collect) so the
// interpreter stays free of any UI dependency; this interface is the seam.
type Collector interface {
	// MaybeCollect is called for the field `attr` of entity `e` just before
	// its value is read. If `attr` is a `collect` field whose value has not
	// been collected on this instance yet, the collector obtains the value
	// (blocking — e.g. a CLI prompt or a web round-trip), writes it, and
	// marks it collected. For any other field it is a no-op. Returning an
	// error aborts execution.
	MaybeCollect(e Entity, attr *RName) error
}
