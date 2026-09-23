package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules/authoring"
)

// #1225: moving a table to another file reported success and left two copies.
// The moved table kept its old workbook, so the Excel refresh exported it back
// into the source workbook and recompiled that into the source _dt.xml; the
// new file's workbook was never created at all. Every case here ends with the
// two checks `verify` runs, because "the XML looks right" is what the bug
// already reported.

const pokerSample = "../../sampleprojects/Poker"

func mustRun(t *testing.T, proj string, args []string, stdin string) string {
	t.Helper()
	out, errOut, code := runTableCmd(t, proj, args, stdin)
	if code != 0 {
		t.Fatalf("table %s: exit %d\nstdout: %s\nstderr: %s", strings.Join(args, " "), code, out, errOut)
	}
	return out
}

// filesOf maps each table name to the files holding it, so a duplicate shows
// as two files rather than as a `-1` the loader invented.
func filesOf(t *testing.T, proj string) map[string][]string {
	t.Helper()
	p, err := authoring.OpenProject(proj)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][]string{}
	for _, f := range p.Files() {
		for _, name := range f.Tables {
			base := strings.TrimSuffix(name, "-1")
			got[base] = append(got[base], f.Path)
		}
	}
	return got
}

func assertVerifies(t *testing.T, proj string) {
	t.Helper()
	xmlDir, excelDir := filepath.Join(proj, "xml"), filepath.Join(proj, "excel")
	for _, f := range checkBuildIdempotency(proj, xmlDir, excelDir, &verifyOptions{}) {
		t.Errorf("verify [build]: %s", f.message)
	}
	for _, f := range checkWorkbookProvenance(proj, xmlDir, excelDir) {
		t.Errorf("verify [provenance]: %s", f.message)
	}
}

func assertOnlyIn(t *testing.T, proj, table, file string) {
	t.Helper()
	if got := filesOf(t, proj)[table]; len(got) != 1 || got[0] != file {
		t.Fatalf("%s is in %v, want only %s", table, got, file)
	}
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}

func TestSetFileMovesTheTable(t *testing.T) {
	proj := copyProject(t, pokerSample)
	mustRun(t, proj, []string{"patch", "Rock_Decision"},
		`{"op":"set-file","file":"rock_dt.xml","range":"500-599","reason":"own file"}`)

	assertOnlyIn(t, proj, "Rock_Decision", "rock_dt.xml")
	assertExists(t, filepath.Join(proj, "excel", "rock.xlsx"))
	assertVerifies(t, proj)
}

// The issue's second trap: `table get` output carries the old number and
// workbook, and put of it into a new file failed on the number and, with the
// number removed, duplicated the table through the workbook.
func TestPutOfGetOutputIntoANewFileMovesTheTable(t *testing.T) {
	proj := copyProject(t, pokerSample)
	got := mustRun(t, proj, []string{"get", "Rock_Decision"}, "")
	mustRun(t, proj, []string{"put", "Rock_Decision", "--file", "rock_dt.xml",
		"--range", "500-599", "--reason", "own file"}, got)

	assertOnlyIn(t, proj, "Rock_Decision", "rock_dt.xml")
	assertExists(t, filepath.Join(proj, "excel", "rock.xlsx"))
	assertVerifies(t, proj)
}

func TestNewTableInANewFileGetsItsWorkbook(t *testing.T) {
	proj := copyProject(t, pokerSample)
	mustRun(t, proj, []string{"put", "Brand_New", "--file", "fresh_dt.xml",
		"--range", "500-599", "--reason", "new"},
		`{"name":"Brand_New","policy":"FIRST","conditions":[],
		  "actions":[{"number":1,"dsl":"set player.action = \"fold\"","columns":{"1":true}}]}`)

	assertExists(t, filepath.Join(proj, "excel", "fresh.xlsx"))
	assertVerifies(t, proj)
}

// A new file in a directory that does not exist yet. Save used to write the
// source file first and then fail on the target, and the table was in neither.
func TestMoveIntoANewDirectory(t *testing.T) {
	proj := copyProject(t, pokerSample)
	// --range as a flag: patch used to drop it silently.
	mustRun(t, proj, []string{"patch", "Rock_Decision", "--range", "500-599"},
		`{"op":"set-file","file":"styles/rock_dt.xml","reason":"styles"}`)

	assertOnlyIn(t, proj, "Rock_Decision", "styles/rock_dt.xml")
	assertExists(t, filepath.Join(proj, "excel", "styles", "rock.xlsx"))
	assertVerifies(t, proj)

	// Into a file that now exists: it takes that file's workbook.
	mustRun(t, proj, []string{"patch", "TAG_Decision"},
		`{"op":"set-file","file":"styles/rock_dt.xml","reason":"styles"}`)
	assertOnlyIn(t, proj, "TAG_Decision", "styles/rock_dt.xml")
	assertVerifies(t, proj)

	// And back out: the emptied file and its workbook go.
	for _, name := range []string{"Rock_Decision", "TAG_Decision"} {
		mustRun(t, proj, []string{"patch", name}, `{"op":"set-file","file":"Poker_dt.xml","reason":"back"}`)
		assertOnlyIn(t, proj, name, "Poker_dt.xml")
	}
	for _, gone := range []string{
		filepath.Join(proj, "xml", "styles", "rock_dt.xml"),
		filepath.Join(proj, "excel", "styles", "rock.xlsx"),
	} {
		if _, err := os.Stat(gone); !os.IsNotExist(err) {
			t.Errorf("%s should be removed once its last table left (stat err=%v)", gone, err)
		}
	}
	assertVerifies(t, proj)
}

// A new file's workbook can only be the one named after it: the build
// compiles X.xlsx to X_dt.xml. Anything else is refused before any write.
func TestPutRefusesAForeignWorkbookForANewFile(t *testing.T) {
	proj := copyProject(t, pokerSample)
	before, err := os.ReadFile(filepath.Join(proj, "xml", "Poker_dt.xml"))
	if err != nil {
		t.Fatal(err)
	}
	got := mustRun(t, proj, []string{"get", "LAG_Decision"}, "")
	got = strings.Replace(got, `"workbook": "Poker.xlsx"`, `"workbook": "other.xlsx"`, 1)
	if !strings.Contains(got, "other.xlsx") {
		t.Fatalf("could not set the workbook in get output:\n%s", got)
	}
	_, errOut, code := runTableCmd(t, proj, []string{"put", "LAG_Decision", "--file", "lag_dt.xml",
		"--range", "700-799", "--reason", "x"}, got)
	if code == 0 || !strings.Contains(errOut, "invalid_input") || !strings.Contains(errOut, "lag.xlsx") {
		t.Fatalf("want invalid_input naming lag.xlsx; exit %d, stderr %s", code, errOut)
	}
	after, _ := os.ReadFile(filepath.Join(proj, "xml", "Poker_dt.xml"))
	if string(before) != string(after) {
		t.Error("a refused put still rewrote Poker_dt.xml")
	}
	if _, err := os.Stat(filepath.Join(proj, "xml", "lag_dt.xml")); !os.IsNotExist(err) {
		t.Error("a refused put still wrote lag_dt.xml")
	}
}

func TestPatchFlagThatDisagreesWithTheBodyIsRefused(t *testing.T) {
	proj := copyProject(t, pokerSample)
	_, errOut, code := runTableCmd(t, proj, []string{"patch", "LAG_Decision", "--range", "600-699"},
		`{"op":"set-file","file":"lag_dt.xml","range":"500-599","reason":"x"}`)
	if code == 0 || !strings.Contains(errOut, "disagrees") {
		t.Fatalf("want a disagreement error; exit %d, stderr %s", code, errOut)
	}
}

// MCP's table_put has its own copy of the put logic; it must move the same way.
func TestMCPTablePutOfGetOutputMovesTheTable(t *testing.T) {
	proj := copyProject(t, pokerSample)
	var table map[string]interface{}
	if err := json.Unmarshal([]byte(mustRun(t, proj, []string{"get", "Rock_Decision"}, "")), &table); err != nil {
		t.Fatal(err)
	}
	rpc := newMCPRPC(t, proj)
	defer rpc.close()
	result := mustResult(t, rpc.call("tools/call", map[string]interface{}{
		"name": "table_put",
		"arguments": map[string]interface{}{
			"name": "Rock_Decision", "file": "rock_dt.xml", "range": "500-599",
			"reason": "own file", "table": table,
		},
	}))
	if isErr, _ := result["isError"].(bool); isErr {
		t.Fatalf("table_put failed: %v", result)
	}
	assertOnlyIn(t, proj, "Rock_Decision", "rock_dt.xml")
	assertExists(t, filepath.Join(proj, "excel", "rock.xlsx"))
	assertVerifies(t, proj)
}
