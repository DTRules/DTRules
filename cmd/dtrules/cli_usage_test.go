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

package main

import (
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
)

// #1224: `dtrules map` -- the authoring surface for mappings -- existed and
// was listed nowhere in `dtrules --help`, so a session that read the help
// concluded there was no mapping API and wrote the mapping by hand. debug and
// report were missing too. Every command the dispatcher accepts is in the
// help, except the ones hidden on purpose.
func TestUsageListsEveryCommand(t *testing.T) {
	hidden := map[string]bool{
		"internal": true, // backward-compat script entry points, see runInternal
		"help":     true, "-h": true, "--help": true,
	}

	src, err := os.ReadFile("cli.go")
	if err != nil {
		t.Fatal(err)
	}
	body := string(src)
	start := strings.Index(body, "func (c *CLI) Run(")
	if start < 0 {
		t.Fatal("CLI.Run not found in cli.go")
	}
	body = body[start:]
	if end := strings.Index(body, "\nfunc "); end > 0 {
		body = body[:end]
	}
	var commands []string
	for _, m := range regexp.MustCompile(`case ([^:]+):`).FindAllStringSubmatch(body, -1) {
		for _, q := range regexp.MustCompile(`"([^"]+)"`).FindAllStringSubmatch(m[1], -1) {
			if !hidden[q[1]] {
				commands = append(commands, q[1])
			}
		}
	}
	if len(commands) < 10 {
		t.Fatalf("found only %d commands in CLI.Run; the parse is wrong: %v", len(commands), commands)
	}

	r, w, _ := os.Pipe()
	stdout := os.Stdout
	os.Stdout = w
	NewCLI().printUsage()
	w.Close()
	os.Stdout = stdout
	usage, _ := io.ReadAll(r)

	for _, cmd := range commands {
		if !regexp.MustCompile(`(?m)^  ` + regexp.QuoteMeta(cmd) + `\s`).Match(usage) {
			t.Errorf("command %q is dispatched but not listed in `dtrules --help`", cmd)
		}
	}
}
