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
	"net/http"
	"sort"
	"strconv"

	"github.com/DTRules/DTRules/pkg/dtrules"
)

// entityValue is one classified value for the entity explorer: a scalar
// with display text, an entity reference (drill via its id), or an array
// (drill via its arrayId). Nested composition is fetched lazily by the
// UI — one request per expansion, so arbitrarily deep structures stay
// cheap.
type entityValue struct {
	Kind  string `json:"kind"` // "value" | "entity" | "array"
	Type  string `json:"type,omitempty"`
	Value string `json:"value,omitempty"` // scalars: display text

	// Entity references
	Entity string `json:"entity,omitempty"` // entity name
	ID     int    `json:"id,omitempty"`     // instance id (fetch /api/debug/entity?id=)

	// Arrays
	ArrayID int `json:"arrayId,omitempty"` // fetch /api/debug/array?id=
	Length  int `json:"length,omitempty"`
}

// entityField is one attribute of an inspected instance.
type entityField struct {
	Name string `json:"name"`
	entityValue
}

// classifyObject renders a value for the explorer.
func classifyObject(v dtrules.Object) entityValue {
	if v == nil {
		return entityValue{Kind: "value", Value: ""}
	}
	if e, ok := v.(dtrules.Entity); ok {
		return entityValue{Kind: "entity", Entity: e.GetName().StringValue(), ID: e.GetID()}
	}
	if ar, ok := v.(*dtrules.RArray); ok {
		return entityValue{Kind: "array", ArrayID: ar.GetID(), Length: ar.Size()}
	}
	typ := ""
	if t := v.Type(); t != nil {
		typ = t.String()
	}
	return entityValue{Kind: "value", Type: typ, Value: v.StringValue()}
}

// handleDebugEntity inspects one entity instance at the current replay
// position: every attribute, classified for drilling.
// GET /api/debug/entity?id=<instance id>
func (s *Server) handleDebugEntity(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		jsonError(w, "id is required", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.debug == nil {
		jsonError(w, "No trace loaded", http.StatusBadRequest)
		return
	}
	e := s.debug.trace.EntityByID(id)
	if e == nil {
		jsonError(w, "No such entity instance at the current position", http.StatusNotFound)
		return
	}

	fields := []entityField{}
	for _, n := range e.GetAttributeNames() {
		name := n.StringValue()
		if name == "" {
			continue
		}
		v, err := e.Get(n)
		if err != nil {
			continue
		}
		fields = append(fields, entityField{Name: name, entityValue: classifyObject(v)})
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })

	jsonResponse(w, map[string]interface{}{
		"success": true,
		"name":    e.GetName().StringValue(),
		"id":      e.GetID(),
		"fields":  fields,
	})
}

// handleDebugArray lists an array's elements at the current replay
// position, classified for drilling, with offset/limit paging.
// GET /api/debug/array?id=<arrayId>[&offset=0][&limit=200]
func (s *Server) handleDebugArray(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		jsonError(w, "numeric id is required", http.StatusBadRequest)
		return
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.debug == nil {
		jsonError(w, "No trace loaded", http.StatusBadRequest)
		return
	}
	ar := s.debug.trace.ArrayByID(id)
	if ar == nil {
		jsonError(w, "No such array at the current position", http.StatusNotFound)
		return
	}

	total := ar.Size()
	elements := []entityValue{}
	for i := offset; i < total && i < offset+limit; i++ {
		v, gerr := ar.Get(i)
		if gerr != nil {
			elements = append(elements, entityValue{Kind: "value", Value: "(error)"})
			continue
		}
		elements = append(elements, classifyObject(v))
	}

	jsonResponse(w, map[string]interface{}{
		"success":  true,
		"id":       id,
		"total":    total,
		"offset":   offset,
		"elements": elements,
	})
}
