// Copyright 2025 DTRules contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !ui

// Package ui embeds the built editor bundle into Go binaries. This stub is
// compiled when the `ui` build tag is absent, so plain `go build ./...`
// works without npm or a built bundle.
package ui

import "io/fs"

// Dist reports that no UI bundle is embedded in this build.
func Dist() (fs.FS, bool) {
	return nil, false
}
