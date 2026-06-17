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

// Package datafile reads and writes canonical DTRules data XML — a
// mapping-free, EDD-shaped serialization of a session's entity values
// (#850/#853). Unlike a project mapping (which reconciles foreign tag names),
// canonical data uses tags that are 1:1 with the EDD: <entity><field>value</field></entity>,
// arrays as repeated subtype elements, nested entities nested. So load and
// save are inverse, generic walks of the EDD schema.
//
// A saved dataset is both the audit record of an interactive session and a
// replay fixture: authoritative-load it for a batch run, or review-load it to
// re-interview with each collect field pre-filled.
package datafile

import (
	"encoding/xml"
	"io"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/entity"
)

const rootElement = "dtrules-data"

// reservedAttr reports attribute names that are runtime bookkeeping, not data.
func reservedAttr(entityName, attr string) bool {
	return attr == entityName || attr == "mapping*key" || strings.Contains(attr, "*")
}

// Write serializes the given entities to canonical data XML.
func Write(w io.Writer, entities []*entity.REntity) error {
	var b strings.Builder
	b.WriteString(xml.Header)
	b.WriteString("<" + rootElement + ">\n")
	for _, e := range entities {
		if err := writeEntity(&b, e, e.GetName().GetName(), 1); err != nil {
			return err
		}
	}
	b.WriteString("</" + rootElement + ">\n")
	_, err := io.WriteString(w, b.String())
	return err
}

func indent(b *strings.Builder, depth int) {
	for i := 0; i < depth; i++ {
		b.WriteString("  ")
	}
}

func esc(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// writeEntity emits one entity as <elemName>...fields...</elemName>.
func writeEntity(b *strings.Builder, e *entity.REntity, elemName string, depth int) error {
	indent(b, depth)
	b.WriteString("<" + elemName + ">\n")
	entityName := e.GetName().GetName()
	for _, attr := range e.GetAttributeNames() {
		name := attr.GetName()
		if reservedAttr(entityName, name) {
			continue
		}
		entry := e.GetEntry(attr)
		if entry == nil {
			continue
		}
		v, err := e.Get(attr)
		if err != nil || v == nil || v.Type() == dtrules.TypeNull {
			continue
		}
		if err := writeField(b, name, entry, v, depth+1); err != nil {
			return err
		}
	}
	indent(b, depth)
	b.WriteString("</" + elemName + ">\n")
	return nil
}

func writeField(b *strings.Builder, name string, entry *entity.EntityEntry, v dtrules.Object, depth int) error {
	switch entry.Type {
	case dtrules.TypeArray:
		arr, err := v.ArrayValue()
		if err != nil || len(arr) == 0 {
			return nil
		}
		indent(b, depth)
		b.WriteString("<" + name + ">\n")
		for _, el := range arr {
			if sub, ok := el.(*entity.REntity); ok {
				// array of entities: each element named by its entity type
				if err := writeEntity(b, sub, sub.GetName().GetName(), depth+1); err != nil {
					return err
				}
			} else {
				// array of scalars
				indent(b, depth+1)
				b.WriteString("<item>" + esc(el.StringValue()) + "</item>\n")
			}
		}
		indent(b, depth)
		b.WriteString("</" + name + ">\n")
	case dtrules.TypeEntity:
		if sub, ok := v.(*entity.REntity); ok {
			return writeEntity(b, sub, name, depth)
		}
	default: // scalar
		indent(b, depth)
		b.WriteString("<" + name + ">" + esc(v.StringValue()) + "</" + name + ">\n")
	}
	return nil
}

// LoadMode selects how loaded values mark collection state (#850/#853).
type LoadMode int

const (
	// Authoritative marks every loaded value as collected — the data is taken
	// as final, so an interactive run won't re-ask for it (and batch is
	// identical to today).
	Authoritative LoadMode = iota
	// Review loads the values but leaves them defaulted, so an interactive run
	// re-asks every collect field with the loaded value pre-filled.
	Review
)

// EntityFinder returns the top-level entity instance for an entity-type name,
// or nil if the session has no such entity.
type EntityFinder func(name string) *entity.REntity

// EntityCreator makes a fresh instance of the given entity type, used for
// array elements. May return (nil, nil) to skip.
type EntityCreator func(subtype string) (*entity.REntity, error)

// Read parses canonical data XML and applies it to the session's entities.
// `find` locates top-level entities by type name; `create` instantiates array
// elements. Unknown entities/fields are skipped.
func Read(r io.Reader, find EntityFinder, create EntityCreator, mode LoadMode) error {
	dec := xml.NewDecoder(r)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == rootElement {
			return readContainer(dec, find, create, mode, func(name string) (*entity.REntity, error) {
				return find(name), nil
			})
		}
	}
}

// readContainer reads a sequence of entity elements until the closing tag of
// the current element. resolve maps an element name to the target entity.
func readContainer(dec *xml.Decoder, find EntityFinder, create EntityCreator, mode LoadMode,
	resolve func(string) (*entity.REntity, error)) error {
	for {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			e, err := resolve(t.Name.Local)
			if err != nil {
				return err
			}
			if e == nil {
				if err := dec.Skip(); err != nil {
					return err
				}
				continue
			}
			if err := readEntityBody(dec, e, find, create, mode); err != nil {
				return err
			}
		case xml.EndElement:
			return nil
		}
	}
}

// readEntityBody fills entity e from its field elements until e's closing tag.
func readEntityBody(dec *xml.Decoder, e *entity.REntity, find EntityFinder, create EntityCreator, mode LoadMode) error {
	if mode == Authoritative {
		// Turn tracking on first so each Put marks the field collected.
		e.EnableCollectTracking()
	}
	for {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			field := dtrules.GetRName(t.Name.Local)
			entry := e.GetEntry(field)
			if entry == nil {
				if err := dec.Skip(); err != nil {
					return err
				}
				continue
			}
			switch entry.Type {
			case dtrules.TypeArray:
				if err := readArray(dec, e, field, entry, find, create, mode); err != nil {
					return err
				}
			case dtrules.TypeEntity:
				nested, _ := e.Get(field)
				if ne, ok := nested.(*entity.REntity); ok {
					if err := readEntityBody(dec, ne, find, create, mode); err != nil {
						return err
					}
				} else if err := dec.Skip(); err != nil {
					return err
				}
			default: // scalar
				txt, err := readText(dec)
				if err != nil {
					return err
				}
				// Put coerces the text to the field's declared type, and (when
				// tracking is on, i.e. authoritative) marks it collected.
				_ = e.Put(field, dtrules.GetRString(txt))
			}
		case xml.EndElement:
			return nil
		}
	}
}

// readArray reads the elements of an array field until its closing tag.
func readArray(dec *xml.Decoder, e *entity.REntity, field *dtrules.RName, entry *entity.EntityEntry,
	find EntityFinder, create EntityCreator, mode LoadMode) error {
	cur, _ := e.Get(field)
	arr, _ := cur.(*dtrules.RArray)
	for {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "item" {
				txt, err := readText(dec)
				if err != nil {
					return err
				}
				if arr != nil {
					arr.Add(dtrules.GetRString(txt))
				}
				continue
			}
			// An entity element: create an instance of the array's subtype and
			// populate it, then append.
			if create == nil {
				if err := dec.Skip(); err != nil {
					return err
				}
				continue
			}
			el, err := create(entry.SubType)
			if err != nil {
				return err
			}
			if el == nil {
				if err := dec.Skip(); err != nil {
					return err
				}
				continue
			}
			if err := readEntityBody(dec, el, find, create, mode); err != nil {
				return err
			}
			if arr != nil {
				arr.Add(el)
			}
		case xml.EndElement:
			return nil
		}
	}
}

// readText returns the character data of the current element, consuming up to
// and including its end tag.
func readText(dec *xml.Decoder) (string, error) {
	var sb strings.Builder
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", err
		}
		switch t := tok.(type) {
		case xml.CharData:
			sb.Write(t)
		case xml.EndElement:
			return strings.TrimSpace(sb.String()), nil
		case xml.StartElement:
			// Unexpected nested element in a scalar; skip it.
			if err := dec.Skip(); err != nil {
				return "", err
			}
		}
	}
}

