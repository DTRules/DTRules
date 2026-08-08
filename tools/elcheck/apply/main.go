// Command apply writes an elcheck overrides file into a project through the
// authoring SDK — the bulk equivalent of `dtrules table put`, opening the
// project once and saving once instead of a get/put round trip per table.
//
// Keys are 1-based row POSITIONS (`context 3`, `initial action 6`,
// `condition@12`, `action@37`), never the numbers written in the XML: the
// authoring view renumbers rows to their position on load, and a number-keyed
// patch lands one row off wherever the stored numbering has a gap.
//
// Safety: every mutation goes through the SDK's validating methods, and after
// all patches are applied the tool refuses to save if any patched table still
// reports hand-coded rows — the rows a Save would silently delete (#974).
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "apply: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	if len(os.Args) != 3 {
		fatal("usage: apply <project> <overrides.json>")
	}
	project, ovpath := os.Args[1], os.Args[2]

	raw, err := os.ReadFile(ovpath)
	if err != nil {
		fatal("%v", err)
	}
	var overrides map[string]map[string]string
	if err := json.Unmarshal(raw, &overrides); err != nil {
		fatal("%v", err)
	}

	p, err := authoring.OpenProject(project)
	if err != nil {
		fatal("open: %v", err)
	}

	tables := make([]string, 0, len(overrides))
	for name := range overrides {
		tables = append(tables, name)
	}
	sort.Strings(tables)

	rows := 0
	for _, name := range tables {
		t := p.Table(name)
		if t == nil {
			fatal("table %q not found", name)
		}
		// Number-keyed SDK calls need unique numbers to be unambiguous.
		seen := map[int]bool{}
		for _, c := range t.Conditions {
			if seen[c.Number] {
				fatal("%s: duplicate condition number %d — cannot patch safely", name, c.Number)
			}
			seen[c.Number] = true
		}
		seen = map[int]bool{}
		for _, a := range t.Actions {
			if seen[a.Number] {
				fatal("%s: duplicate action number %d — cannot patch safely", name, a.Number)
			}
			seen[a.Number] = true
		}

		keys := make([]string, 0, len(overrides[name]))
		for k := range overrides[name] {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			dsl := overrides[name][key]
			if err := patch(t, key, dsl); err != nil {
				fatal("%s %s: %v", name, key, err)
			}
			rows++
		}
	}

	// The gate: a row that still carries postfix without DSL is deleted by
	// Save. Nothing may be saved while any patched table has one.
	var dirty []string
	for _, name := range tables {
		if hand := p.Table(name).HandCodedRows(); len(hand) > 0 {
			dirty = append(dirty, fmt.Sprintf("%s: %s", name, strings.Join(hand, ", ")))
		}
	}
	if len(dirty) > 0 {
		fatal("refusing to save — hand-coded rows would be deleted:\n  %s",
			strings.Join(dirty, "\n  "))
	}

	if err := p.Save(); err != nil {
		fatal("save: %v", err)
	}
	fmt.Printf("applied %d rows across %d tables\n", rows, len(tables))
}

func patch(t *authoring.Table, key, dsl string) error {
	switch {
	case strings.HasPrefix(key, "context "):
		i, err := strconv.Atoi(strings.TrimPrefix(key, "context "))
		if err != nil || i < 1 || i > len(t.Contexts) {
			return fmt.Errorf("bad context position %q", key)
		}
		c := t.Contexts[i-1]
		c.DSL = dsl
		return t.UpdateContext(i-1, c)
	case strings.HasPrefix(key, "initial action "):
		i, err := strconv.Atoi(strings.TrimPrefix(key, "initial action "))
		if err != nil || i < 1 || i > len(t.InitialActions) {
			return fmt.Errorf("bad initial action position %q", key)
		}
		a := t.InitialActions[i-1]
		a.DSL = dsl
		return t.UpdateInitialAction(i-1, a)
	case strings.HasPrefix(key, "condition@"):
		i, err := strconv.Atoi(strings.TrimPrefix(key, "condition@"))
		if err != nil || i < 1 || i > len(t.Conditions) {
			return fmt.Errorf("bad condition position %q", key)
		}
		c := t.Conditions[i-1]
		c.DSL = dsl
		return t.UpdateCondition(c.Number, c)
	case strings.HasPrefix(key, "action@"):
		i, err := strconv.Atoi(strings.TrimPrefix(key, "action@"))
		if err != nil || i < 1 || i > len(t.Actions) {
			return fmt.Errorf("bad action position %q", key)
		}
		a := t.Actions[i-1]
		a.DSL = dsl
		return t.UpdateAction(a.Number, a)
	}
	return fmt.Errorf("unrecognized key %q", key)
}
