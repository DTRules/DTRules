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

//go:build !amd64

package interpreter

import "github.com/DTRules/DTRules/pkg/dtrules"

// ExecuteBytecodeASM is provided as a fallback on non-amd64 builds so the
// nativeasm runtime package still links. The actual SSE2 dispatch loop lives
// in `asm_amd64.s` + companion Go declarations under `//go:build amd64` —
// without this file, arm64 builds (linux-arm64 / darwin-arm64 release
// targets) would fail with `undefined: asmAdd` and similar link errors.
//
// The fallback delegates to the pure-Go ExecuteBytecode path in vm.go, so
// nativeasm callers get the same observable behaviour as on amd64, just
// without the assembly speedup. Callers that need the speedup can detect
// the architecture and choose a different BytecodeExecutor; nothing in
// the engine forces them through this path.
func (s *DTState) ExecuteBytecodeASM(bc *dtrules.BytecodeChunk) error {
	return s.ExecuteBytecode(bc)
}
