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
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/collect"
	"github.com/DTRules/DTRules/pkg/dtrules/interpreter"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// runRun handles `dtrules run [path] --entry <table> [--input f] [--interactive]`.
// It loads a project, runs a decision table, and prints the result entity.
// With --interactive, a CLI prompt collects any reached `collect` field whose
// value hasn't been provided (#850/#854).
func (c *CLI) runRun(args []string) int {
	path, entry, input, resultEntity := ".", "", "", "result"
	interactive := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--entry":
			if i+1 < len(args) {
				entry = args[i+1]
				i++
			}
		case "--input":
			if i+1 < len(args) {
				input = args[i+1]
				i++
			}
		case "--result-entity":
			if i+1 < len(args) {
				resultEntity = args[i+1]
				i++
			}
		case "--interactive", "-i":
			interactive = true
		case "-h", "--help":
			c.printRunUsage()
			return 0
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(os.Stderr, "Unknown flag: %s\n", args[i])
				c.printRunUsage()
				return 1
			}
			path = args[i]
		}
	}
	if entry == "" {
		fmt.Fprintln(os.Stderr, "Error: --entry <table> is required")
		c.printRunUsage()
		return 1
	}

	xmlDir, _, err := resolveDirs(mustAbs(path), "", "")
	if err != nil || !dirExists(xmlDir) {
		fmt.Fprintf(os.Stderr, "Error: could not find xml/ directory under %s\n", path)
		return 1
	}

	rs := session.NewRuleSet(filepath.Base(mustAbs(path)))
	if err := rs.LoadFromDirectory(xmlDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading rules: %v\n", err)
		return 1
	}
	sess, err := rs.NewSession()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating session: %v\n", err)
		return 1
	}

	// Initialize entities (and optionally load input data) via the mapping.
	if err := initMapping(sess, xmlDir, input); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing data: %v\n", err)
		return 1
	}

	state := sess.GetState()
	if interactive {
		if dts, ok := state.(*interpreter.DTState); ok {
			dts.SetCollector(collect.New(newCLIAsker()))
		} else {
			fmt.Fprintln(os.Stderr, "Warning: interactive collection unavailable for this state")
		}
	}

	dt, err := sess.GetEntityFactory().GetDecisionTable(dtrules.GetRName(entry))
	if err != nil || dt == nil {
		fmt.Fprintf(os.Stderr, "Error: decision table %q not found\n", entry)
		return 1
	}
	if err := dt.Execute(state); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing %q: %v\n", entry, err)
		return 1
	}

	renderResult(state, resultEntity)
	return 0
}

// initMapping loads the project's *_map.xml (if any), initializes the entity
// stack, and loads input data when a file is given.
func initMapping(sess dtrules.Session, xmlDir, input string) error {
	maps, _ := filepath.Glob(filepath.Join(xmlDir, "*_map.xml"))
	if len(maps) == 0 {
		return nil // no mapping; rely on whatever the session set up
	}
	mapFile, err := os.Open(maps[0])
	if err != nil {
		return err
	}
	defer mapFile.Close()

	m := mapping.NewMapping(sess)
	if err := m.LoadMapping(mapFile); err != nil {
		return err
	}
	if err := m.Initialize(); err != nil {
		return err
	}
	if input != "" {
		f, err := os.Open(input)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := m.LoadData(f); err != nil {
			return err
		}
	}
	return nil
}

// renderResult prints the named output entity's fields after execution.
func renderResult(state dtrules.State, entityName string) {
	result, err := state.FindEntity(dtrules.GetRName(entityName))
	if err != nil || result == nil {
		fmt.Printf("\n(no %q entity to display)\n", entityName)
		return
	}
	fmt.Printf("\n=== %s ===\n", entityName)
	for _, attr := range result.GetAttributeNames() {
		name := attr.GetName()
		if name == entityName || name == "mapping*key" {
			continue
		}
		v, err := result.Get(attr)
		if err != nil || v == nil {
			continue
		}
		if arr, err := v.ArrayValue(); err == nil {
			if len(arr) == 0 {
				continue
			}
			fmt.Printf("  %s:\n", name)
			for _, e := range arr {
				fmt.Printf("    - %s\n", e.StringValue())
			}
			continue
		}
		fmt.Printf("  %-22s %s\n", name+":", v.StringValue())
	}
}

func (c *CLI) printRunUsage() {
	fmt.Println(`Usage: dtrules run [path] --entry <table> [options]

Loads a project, runs a decision table, and prints the result entity.

Options:
  --entry <table>        Decision table to run (required)
  --input <file.xml>     Input data to load via the project mapping
  --interactive, -i      Prompt for any reached collect field not supplied
  --result-entity <name> Output entity to print (default: result)

Examples:
  dtrules run ./sampleprojects/SinusitisTherapy --entry Determine_Therapy --interactive
  dtrules run . --entry Determine_Therapy --input case.xml`)
}

func mustAbs(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}

// cliAsker prompts for collect-field values on the terminal.
type cliAsker struct {
	in *bufio.Reader
}

func newCLIAsker() *cliAsker { return &cliAsker{in: bufio.NewReader(os.Stdin)} }

// Ask implements collect.Asker. An empty line keeps the current default.
func (a *cliAsker) Ask(req collect.Request) (dtrules.Object, bool, error) {
	cur := ""
	if req.Current != nil {
		cur = req.Current.StringValue()
	}
	prompt := req.Text
	if prompt == "" {
		prompt = fmt.Sprintf("%s.%s", req.Entity, req.Field)
	}
	fmt.Printf("\n%s\n", prompt)

	if req.QType == "multiple_choice" {
		for i, o := range req.Options {
			label := o.Label
			if label == "" {
				label = o.Value
			}
			fmt.Printf("  %d) %s\n", i+1, label)
		}
		fmt.Printf("  [default %s] > ", cur)
		line, _ := a.in.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return nil, false, nil
		}
		if n, err := strconv.Atoi(line); err == nil && n >= 1 && n <= len(req.Options) {
			return dtrules.GetRString(req.Options[n-1].Value), true, nil
		}
		return dtrules.GetRString(line), true, nil // accept a literal value too
	}

	fmt.Printf("  (%s) [default %s] > ", req.QType, cur)
	line, _ := a.in.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, false, nil
	}
	return dtrules.GetRString(line), true, nil
}
