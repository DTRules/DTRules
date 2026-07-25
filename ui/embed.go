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

//go:build ui

// Package ui embeds the built editor bundle (ui/dist) into Go binaries.
// Build the bundle first (`make ui-dist`), then build with `-tags ui`.
// Without the tag, the stub in embed_stub.go reports the UI as unavailable
// so plain `go build ./...` never requires npm.
package ui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// Dist returns the built UI bundle as a filesystem rooted at its index.html,
// and true. (The ui build tag guarantees the bundle was embedded.)
func Dist() (fs.FS, bool) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, false
	}
	return sub, true
}
