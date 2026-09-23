package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// #1222: table put reported every refusal as compile_error, "an EL expression
// failed to compile" -- a table number outside its file's range included. One
// row per kind of refusal, through the CLI and through MCP.
func TestTablePutNamesTheKindOfFailure(t *testing.T) {
	good := func() TableJSON {
		return TableJSON{
			Name: "Kind_Probe", Policy: "FIRST",
			Conditions: []ConditionJSON{
				{Number: 1, DSL: "client.age > 5", Columns: map[string]string{"1": "Y", "2": "N"}},
			},
			Actions: []ActionJSON{
				{Number: 1, DSL: "set client.eligible = true", Columns: map[string]bool{"1": true}},
			},
		}
	}
	cases := []struct {
		name     string
		edit     func(*TableJSON)
		kind     string
		hintHas  string
		detailHa string
	}{
		{"EL that does not compile",
			func(tj *TableJSON) { tj.Conditions[0].DSL = "!!! not EL !!!" },
			"compile_error", "EL expression", ""},
		{"number outside the file's range (the issue's repro)",
			func(tj *TableJSON) { tj.Number = 9000 },
			"invalid_number", "range", "outside file"},
		{"'*' outside the last column",
			func(tj *TableJSON) { tj.Conditions[0].Columns = map[string]string{"1": "*", "2": "Y"} },
			"otherwise_rule", "otherwise column", ""},
		{"a column key that is not a number",
			func(tj *TableJSON) { tj.Conditions[0].Columns = map[string]string{"one": "Y"} },
			"invalid_table", "refused", ""},
	}
	for _, c := range cases {
		tj := good()
		c.edit(&tj)
		payload, err := json.Marshal(tj)
		if err != nil {
			t.Fatal(err)
		}
		args := map[string]interface{}{
			"name": "Kind_Probe", "file": "probe_dt.xml", "range": "9500-9599", "reason": "error kinds",
		}
		var table map[string]interface{}
		if err := json.Unmarshal(payload, &table); err != nil {
			t.Fatal(err)
		}
		args["table"] = table

		t.Run("cli/"+c.name, func(t *testing.T) {
			dir := copyProject(t, "../../sampleprojects/CHIP")
			_, se, code := runTableCmd(t, dir, []string{"put", "Kind_Probe", "--file", "probe_dt.xml",
				"--range", "9500-9599", "--reason", "error kinds"}, string(payload))
			if code == 0 {
				t.Fatal("put accepted it")
			}
			var je jsonError
			if err := json.Unmarshal([]byte(se), &je); err != nil {
				t.Fatalf("stderr is not the error shape: %v\n%s", err, se)
			}
			if je.Error != c.kind || !strings.Contains(je.Hint, c.hintHas) || !strings.Contains(je.Detail, c.detailHa) {
				t.Errorf("got error=%q hint=%q detail=%q; want error=%q, hint containing %q, detail containing %q",
					je.Error, je.Hint, je.Detail, c.kind, c.hintHas, c.detailHa)
			}
		})
		t.Run("mcp/"+c.name, func(t *testing.T) {
			dir := copyProject(t, "../../sampleprojects/CHIP")
			rpc := newMCPRPC(t, dir)
			defer rpc.close()
			result := mustResult(t, rpc.call("tools/call", map[string]interface{}{"name": "table_put", "arguments": args}))
			structured, _ := result["structuredContent"].(map[string]interface{})
			if isErr, _ := result["isError"].(bool); !isErr || structured == nil {
				t.Fatalf("table_put accepted it: %v", result)
			}
			if kind, _ := structured["error"].(string); kind != c.kind {
				t.Errorf("error = %q, want %q (%v)", kind, c.kind, structured)
			}
		})
	}
}
