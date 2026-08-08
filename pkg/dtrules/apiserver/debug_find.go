// Copyright 2026 DTRules contributors
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

package apiserver

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules/trace"
)

// findMaxHits caps a single find response; the total match count is still
// reported so the UI can say "showing 200 of 1,431".
const findMaxHits = 200

// normalizeTraceValue reduces a def body's postfix to a comparable literal:
// quotes stripped, the fp suffix and cvdate/cvb conversion tails removed,
// whitespace trimmed. "689938.39835729fp" and `"2008-09-12" cvdate` become
// plain text a user would type.
func normalizeTraceValue(body string) string {
	s := strings.TrimSpace(body)
	for _, tail := range []string{" cvdate", " cvi", " cvb"} {
		s = strings.TrimSuffix(s, tail)
	}
	s = strings.TrimSuffix(s, "fp")
	s = strings.Trim(s, "\"")
	return strings.TrimSpace(s)
}

// valuesMatch compares a normalized trace value against the user's query
// value: case-insensitive text equality, or numeric equality when both
// parse (so "500" matches "500.00000000").
func valuesMatch(traceVal, queryVal string) bool {
	if strings.EqualFold(traceVal, queryVal) {
		return true
	}
	tv, terr := strconv.ParseFloat(traceVal, 64)
	qv, qerr := strconv.ParseFloat(queryVal, 64)
	return terr == nil && qerr == nil && tv == qv
}

// findConditionStep is one condition of a fired column in the why-chain:
// what the column REQUIRED and what actually happened.
type findConditionStep struct {
	Number   int    `json:"number"`   // ordinal position (matches every view)
	DSL      string `json:"dsl"`      // condition text from the table definition
	Required string `json:"required"` // the fired column's cell: Y / N / - (blank = don't care)
	Actual   string `json:"actual"`   // recorded result: true / false / "" (not evaluated)
}

// findChainLink is one frame of the why-chain, innermost first: the table
// whose fired column's action performed the write (or performed the table
// below it in the chain).
type findChainLink struct {
	Table      string              `json:"table"`
	Pass       int                 `json:"pass"`      // 1-based iteration ordinal
	PassCount  int                 `json:"passCount"` // total iterations of that table node
	Column     string              `json:"column"`    // fired column
	Action     string              `json:"action"`    // ordinal of the acting action
	PassNode   int                 `json:"passNode"`  // trace node of the pass (for jumping)
	Conditions []findConditionStep `json:"conditions"`
}

// findHit is one recorded write of the searched field.
type findHit struct {
	Node   int             `json:"node"` // the def node — jump target
	Entity string          `json:"entity"`
	ID     string          `json:"id"`
	Attr   string          `json:"attr"`
	Value  string          `json:"value"`
	Chain  []findChainLink `json:"chain"`
}

// handleDebugFind searches the loaded trace for writes of a field.
// GET /api/debug/find?attr=<name>&entity=<name>&value=<v>
//   - attr is required; entity and value are optional narrowing filters.
//   - All name matching is case-insensitive (EL semantics).
//
// Each hit carries the why-chain: for the write's table and every caller up
// the stack, the fired column's conditions with their required cells and
// actual results — "all the conditions that had to be met" for the write.
func (s *Server) handleDebugFind(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	attr := strings.TrimSpace(r.URL.Query().Get("attr"))
	entity := strings.TrimSpace(r.URL.Query().Get("entity"))
	value := strings.TrimSpace(r.URL.Query().Get("value"))
	instanceID := strings.TrimSpace(r.URL.Query().Get("id"))
	keyField := strings.TrimSpace(r.URL.Query().Get("keyField"))
	keyValue := strings.TrimSpace(r.URL.Query().Get("keyValue"))
	if attr == "" {
		jsonError(w, "attr is required (optionally entity, value, id, keyField/keyValue)", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.debug == nil {
		jsonError(w, "No trace loaded", http.StatusBadRequest)
		return
	}

	root := s.debug.trace.Root()

	// Instance scoping — the usual question is about ONE entity out of
	// all of them ("why is THIS account ineligible?"). A key field pins
	// the instance: every id whose keyField was set to keyValue.
	var wantIDs map[string]bool
	if instanceID != "" {
		wantIDs = map[string]bool{instanceID: true}
	} else if keyField != "" && keyValue != "" {
		wantIDs = map[string]bool{}
		var scan func(n *trace.TraceNode)
		scan = func(n *trace.TraceNode) {
			if n.Name == "def" &&
				strings.EqualFold(n.Attributes["name"], keyField) &&
				(entity == "" || strings.EqualFold(n.Attributes["entity"], entity)) &&
				valuesMatch(normalizeTraceValue(n.Body), keyValue) {
				if id := n.Attributes["id"]; id != "" {
					wantIDs[id] = true
				}
			}
			for _, c := range n.Children {
				scan(c)
			}
		}
		scan(root)
		if len(wantIDs) == 0 {
			jsonResponse(w, map[string]interface{}{
				"success": true, "total": 0, "hits": []findHit{},
				"note": fmt.Sprintf("no %s has %s = %s", entityOr(entity, "entity"), keyField, keyValue),
			})
			return
		}
	}

	hits := []findHit{}
	total := 0

	var walk func(n *trace.TraceNode)
	walk = func(n *trace.TraceNode) {
		if n.Name == "def" &&
			strings.EqualFold(n.Attributes["name"], attr) &&
			(entity == "" || strings.EqualFold(n.Attributes["entity"], entity)) &&
			(wantIDs == nil || wantIDs[n.Attributes["id"]]) {
			v := normalizeTraceValue(n.Body)
			if value == "" || valuesMatch(v, value) {
				total++
				if len(hits) < findMaxHits {
					hits = append(hits, findHit{
						Node:   n.Number,
						Entity: n.Attributes["entity"],
						ID:     n.Attributes["id"],
						Attr:   n.Attributes["name"],
						Value:  v,
						Chain:  s.whyChain(n),
					})
				}
			}
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)

	jsonResponse(w, map[string]interface{}{
		"success": true,
		"total":   total,
		"hits":    hits,
	})
}

// entityOr returns name or the fallback when name is empty.
func entityOr(name, fallback string) string {
	if name == "" {
		return fallback
	}
	return name
}

// whyChain walks up from a trace node collecting, for each enclosing
// decision-table pass, the fired column's condition requirements joined
// with their actual recorded results. Innermost table first. Caller holds
// s.mu (read).
func (s *Server) whyChain(n *trace.TraceNode) []findChainLink {
	chain := []findChainLink{}
	var action, column *trace.TraceNode
	for cur := n.Parent; cur != nil; cur = cur.Parent {
		switch cur.Name {
		case "action", "initialaction":
			if action == nil {
				action = cur
			}
		case "column":
			if column == nil {
				column = cur
			}
		case "execute_table":
			link := s.chainLink(cur, column, action)
			chain = append(chain, link)
			// Reset per-frame context; the next levels up describe the caller.
			action, column = nil, nil
		}
	}
	return chain
}

// chainLink builds one why-chain frame from a pass node and the acting
// column/action within it.
func (s *Server) chainLink(pass, column, action *trace.TraceNode) findChainLink {
	link := findChainLink{Pass: 1, PassCount: 1, PassNode: pass.Number}
	if column != nil {
		link.Column = column.Attributes["n"]
	}
	if action != nil {
		link.Action = action.Attributes["n"]
	}

	dt := pass.Parent
	if dt != nil && dt.Name == "decisiontable" {
		link.Table = dt.Attributes["name"]
		ord, count := 0, 0
		for _, c := range dt.Children {
			if c.Name == "execute_table" {
				count++
				if c.Number == pass.Number {
					ord = count
				}
			}
		}
		link.Pass, link.PassCount = ord, count
	}

	// Actual condition results recorded in this pass, by ordinal.
	actual := map[int]string{}
	for _, c := range pass.Children {
		if c.Name == "condition" {
			if num, err := strconv.Atoi(c.Attributes["n"]); err == nil {
				actual[num] = c.Attributes["result"]
			}
		}
	}

	// Join with the table definition: the fired column's required cells.
	if td := s.findTable(link.Table); td != nil && link.Column != "" {
		for i, cond := range td.Conditions {
			required := strings.TrimSpace(strings.ToUpper(cellValue(cond.Columns, link.Column)))
			if required == "" || required == "-" || required == "*" {
				continue // don't-care cells put no requirement on the column
			}
			dsl := cond.Description
			if dsl == "" {
				dsl = cond.Postfix
			}
			link.Conditions = append(link.Conditions, findConditionStep{
				Number:   i + 1,
				DSL:      dsl,
				Required: required,
				Actual:   actual[i+1],
			})
		}
	}
	return link
}

// cellValue reads a column cell from the {"1": "Y", ...} map, tolerating
// numeric-string key variants.
func cellValue(columns map[string]string, col string) string {
	if v, ok := columns[col]; ok {
		return v
	}
	if n, err := strconv.Atoi(col); err == nil {
		if v, ok := columns[strconv.Itoa(n)]; ok {
			return v
		}
	}
	return ""
}
