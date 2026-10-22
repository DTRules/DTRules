package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #1306, steady state: `edd put` with `entities[].comment` sets an entity's
// comment, the refresh exports it to the EDD sheet and recompiles the
// workbook, and the comment is still there afterwards. `edd get` emits it,
// so a put of get's output loses nothing.

const commentedEDDPut = `{"entities":[
  {"name":"policy","comment":"the policy under test","fields":[
    {"name":"flag","type":"integer","access":"rw","default":"1","comment":"always 1 in these vectors"}]},
  {"name":"result","fields":[
    {"name":"fired","type":"string","access":"rw","comment":"each fired column appends its label"}]}
]}`

func TestEDDPutKeepsAnEntityComment(t *testing.T) {
	proj := bootstrap(t, nil)

	out, errOut, code := runEDDCmd(t, proj, []string{"put"}, commentedEDDPut)
	if code != 0 {
		t.Fatalf("edd put: exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}

	data, err := os.ReadFile(filepath.Join(proj, "xml", "v_edd.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `<entity name="policy" number="100" xls_file="v.xlsx" access="rw" comment="the policy under test">`) {
		t.Errorf("edd put did not keep the entity comment:\n%s", data)
	}
	if strings.Contains(string(data), `name="result" number="200" xls_file="v.xlsx" access="rw" comment=`) {
		t.Errorf("result grew a comment:\n%s", data)
	}
	assertVerifies(t, proj)

	// get emits it at entity level, beside the fields.
	got, errOut, code := runEDDCmd(t, proj, []string{"get"}, "")
	if code != 0 {
		t.Fatalf("edd get: exit %d\nstderr: %s", code, errOut)
	}
	var doc struct {
		Entities []struct {
			Name    string `json:"name"`
			Comment string `json:"comment"`
		} `json:"entities"`
	}
	if err := json.Unmarshal([]byte(got), &doc); err != nil {
		t.Fatalf("edd get output: %v\n%s", err, got)
	}
	comments := map[string]string{}
	for _, e := range doc.Entities {
		comments[e.Name] = e.Comment
	}
	if comments["policy"] != "the policy under test" || comments["result"] != "" {
		t.Errorf("edd get comments = %v", comments)
	}
}
