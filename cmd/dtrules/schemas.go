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

// JSON Schemas (hand-written, draft-07) describing the DTO shapes and patch
// op unions exposed by `dtrules table` and `dtrules edd`. Emitted verbatim
// by the `schema` subcommand — keep these in sync with table_json.go /
// table_patch.go when fields change.

const tableSchemaJSON = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "DTRulesTable",
  "type": "object",
  "required": ["name"],
  "properties": {
    "name":   {"type": "string"},
    "number": {"type": "integer", "minimum": 1, "description": "TABLE_NUMBER for load/sheet ordering; omit to auto-assign within the file's range"},
    "file":   {"type": "string", "description": "DT file (relative to xml/, _dt.xml auto-suffixed) the table lives in; required when creating a table"},
    "policy": {"type": "string", "enum": ["FIRST", "ALL", "NONE", ""]},
    "contexts": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["dsl"],
        "properties": {"dsl": {"type": "string"}}
      }
    },
    "initial_actions": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["dsl"],
        "properties": {"dsl": {"type": "string"}}
      }
    },
    "conditions": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["number", "dsl"],
        "properties": {
          "number":  {"type": "integer", "minimum": 1},
          "comment": {"type": "string"},
          "dsl":     {"type": "string"},
          "columns": {
            "type": "object",
            "description": "map of column number (stringified) to Y/N/-",
            "additionalProperties": {"type": "string", "enum": ["Y", "N", "-"]}
          }
        }
      }
    },
    "actions": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["number", "dsl"],
        "properties": {
          "number":  {"type": "integer", "minimum": 1},
          "comment": {"type": "string"},
          "dsl":     {"type": "string"},
          "columns": {
            "type": "object",
            "description": "map of column number (stringified) to bool (true = action fires on this column)",
            "additionalProperties": {"type": "boolean"}
          }
        }
      }
    }
  }
}
`

const tablePatchSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "DTRulesTablePatchOp",
  "type": "object",
  "required": ["op"],
  "properties": {
    "op": {
      "type": "string",
      "enum": [
        "set-name",
        "set-number",
        "set-file",
        "set-range",
        "set-policy",
        "set-condition-cell",
        "set-action-cell",
        "add-column",
        "update-column",
        "delete-column",
        "add-condition",
        "update-condition",
        "update-condition-dsl",
        "delete-condition",
        "add-action",
        "update-action",
        "update-action-dsl",
        "delete-action",
        "add-initial-action",
        "update-initial-action",
        "delete-initial-action",
        "add-context",
        "update-context",
        "delete-context"
      ]
    },
    "column":           {"type": "integer", "minimum": 1},
    "condition_number": {"type": "integer", "minimum": 1},
    "action_number":    {"type": "integer", "minimum": 1},
    "index":            {"type": "integer", "minimum": 0},
    "value":            {"type": "string", "enum": ["Y", "N", "-"]},
    "on":               {"type": "boolean"},
    "name":             {"type": "string"},
    "number":           {"type": "integer", "minimum": 1},
    "file":             {"type": "string", "description": "set-file: target DT file (move); set-range: file to resize"},
    "range":            {"type": "string", "description": "LO-HI, e.g. 3000-3500 (set-range; set-file when target file is new)"},
    "reason":           {"type": "string", "description": "required for set-file / set-range; recorded in authoring-notes.md"},
    "policy":           {"type": "string"},
    "dsl":              {"type": "string"},
    "comment":          {"type": "string"},
    "conditions": {
      "type": "object",
      "description": "add-column / update-column: condition-number (stringified) to Y/N/-",
      "additionalProperties": {"type": "string", "enum": ["Y", "N", "-"]}
    },
    "actions": {
      "type": "array",
      "description": "add-column / update-column: list of action numbers that fire on this column",
      "items": {"type": "integer"}
    },
    "columns": {
      "type": "object",
      "description": "add-condition / update-condition: map of column number (stringified) to Y/N/-",
      "additionalProperties": {"type": "string", "enum": ["Y", "N", "-"]}
    },
    "action_columns": {
      "type": "object",
      "description": "add-action / update-action: map of column number (stringified) to bool",
      "additionalProperties": {"type": "boolean"}
    }
  }
}
`

const eddSchemaJSON = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "DTRulesEDD",
  "type": "object",
  "required": ["entities"],
  "properties": {
    "entities": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["name"],
        "properties": {
          "name": {"type": "string"},
          "fields": {
            "type": "array",
            "items": {
              "type": "object",
              "required": ["name", "type"],
              "properties": {
                "name":    {"type": "string"},
                "type":    {"type": "string", "enum": ["string", "integer", "long", "double", "boolean", "date", "bigint", "array", "entity"]},
                "subtype": {"type": "string"},
                "default": {"type": "string"},
                "access":  {"type": "string", "enum": ["r", "w", "rw", ""]},
                "input":   {"type": "string"},
                "comment": {"type": "string"},
                "collect":       {"type": "string", "enum": ["true", "false", ""], "description": "ask the user for this field rather than use its default (#850); requires a writable field"},
                "question_text": {"type": "string"},
                "question_type": {"type": "string", "enum": ["multiple_choice", "ascii", "number", "date", ""]},
                "options": {
                  "type": "array",
                  "description": "choices for a multiple_choice question",
                  "items": {
                    "type": "object",
                    "required": ["value"],
                    "properties": {
                      "value": {"type": "string"},
                      "label": {"type": "string"}
                    }
                  }
                }
              }
            }
          }
        }
      }
    }
  }
}
`

const eddPatchSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "DTRulesEDDPatchOp",
  "type": "object",
  "required": ["op"],
  "properties": {
    "op": {
      "type": "string",
      "enum": [
        "add-entity",
        "delete-entity",
        "add-field",
        "update-field",
        "delete-field",
        "set-comment"
      ]
    },
    "entity":  {"type": "string"},
    "name":    {"type": "string"},
    "comment": {"type": "string"},
    "field": {
      "type": "object",
      "required": ["name"],
      "properties": {
        "name":    {"type": "string"},
        "type":    {"type": "string"},
        "subtype": {"type": "string"},
        "default": {"type": "string"},
        "access":  {"type": "string"},
        "input":   {"type": "string"},
        "comment": {"type": "string"}
      }
    }
  }
}
`
