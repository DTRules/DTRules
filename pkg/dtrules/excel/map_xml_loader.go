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

package excel

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

// LoadMapXMLFromFile parses a TaxReturn_map.xml-style file into a MapXML model.
// XML comments between <setattribute> elements are preserved as section entries.
func LoadMapXMLFromFile(path string) (*MapXML, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open map xml: %w", err)
	}
	defer f.Close()
	return loadMapXMLFromReader(f, path)
}

func loadMapXMLFromReader(r io.Reader, name string) (*MapXML, error) {
	m := &MapXML{}
	// Derive map name from filename (strip dir + extension)
	base := name
	for i := len(base) - 1; i >= 0; i-- {
		if base[i] == '/' || base[i] == '\\' {
			base = base[i+1:]
			break
		}
	}
	// Strip _map.xml or .xml
	if strings.HasSuffix(base, "_map.xml") {
		m.MapName = strings.TrimSuffix(base, "_map.xml")
	} else {
		m.MapName = strings.TrimSuffix(base, ".xml")
	}

	decoder := xml.NewDecoder(r)
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse map xml: %w", err)
		}

		switch t := tok.(type) {
		case xml.Comment:
			text := strings.TrimSpace(string(t))
			if text != "" {
				m.Entries = append(m.Entries, MapEntry{IsSection: true, Comment: text})
			}
		case xml.StartElement:
			switch strings.ToLower(t.Name.Local) {
			case "setattribute":
				entry := MapEntry{}
				for _, attr := range t.Attr {
					switch strings.ToLower(attr.Name.Local) {
					case "tag":
						entry.Tag = attr.Value
					case "rattribute":
						entry.RAttribute = attr.Value
					case "enclosure", "entity":
						if entry.Enclosure == "" {
							entry.Enclosure = attr.Value
						}
					case "type":
						entry.Type = attr.Value
					}
				}
				if entry.RAttribute == "" {
					entry.RAttribute = entry.Tag
				}
				if entry.Tag != "" {
					m.Entries = append(m.Entries, entry)
				}
			case "createentity":
				ce := MapCreateEntity{}
				for _, attr := range t.Attr {
					switch strings.ToLower(attr.Name.Local) {
					case "entity":
						ce.Entity = attr.Value
					case "tag":
						ce.Tag = attr.Value
					case "id":
						ce.ID = attr.Value
					}
				}
				if ce.Entity != "" {
					m.CreateEntities = append(m.CreateEntities, ce)
				}
			case "entity":
				// <entity name number> only appears inside <entities>.
				ed := MapEntityDecl{}
				for _, attr := range t.Attr {
					switch strings.ToLower(attr.Name.Local) {
					case "name":
						ed.Name = attr.Value
					case "number":
						ed.Number = attr.Value
					}
				}
				if ed.Name != "" {
					m.EntityDecls = append(m.EntityDecls, ed)
				}
			case "initialentity":
				ie := MapInitialEntity{}
				for _, attr := range t.Attr {
					switch strings.ToLower(attr.Name.Local) {
					case "entity":
						ie.Entity = attr.Value
					case "epush":
						ie.EPush = strings.EqualFold(strings.TrimSpace(attr.Value), "true")
					}
				}
				if ie.Entity != "" {
					m.InitialEntities = append(m.InitialEntities, ie)
				}
			}
		}
	}
	return m, nil
}
