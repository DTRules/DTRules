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
	"os"
	"strings"
)

// WriteMapXML is the exported wrapper for writeMapXML.
func WriteMapXML(m *MapXML, path string) error {
	return writeMapXML(m, path)
}

// writeMapXML serializes a MapXML back to a _map.xml file.
// Section entries become XML comments; attribute entries become <setattribute> elements.
func writeMapXML(m *MapXML, path string) error {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString("<mapping>\n")
	sb.WriteString("\t<XMLtoEDD>\n")
	sb.WriteString("\t\t<map>\n")

	for _, entry := range m.Entries {
		if entry.IsSection {
			sb.WriteString(fmt.Sprintf("\t\t\t<!-- %s -->\n", xmlEscapeComment(entry.Comment)))
		} else {
			sb.WriteString(fmt.Sprintf(
				"\t\t\t<setattribute tag='%s' RAttribute='%s' enclosure='%s' type='%s'></setattribute>\n",
				escAttr(entry.Tag), escAttr(entry.RAttribute),
				escAttr(entry.Enclosure), escAttr(entry.Type),
			))
		}
	}

	for _, ce := range m.CreateEntities {
		list := ""
		if ce.List != "" {
			list = fmt.Sprintf(" list='%s'", escAttr(ce.List))
		}
		sb.WriteString(fmt.Sprintf(
			"\t\t\t<createentity entity='%s' tag='%s' id='%s'%s></createentity>\n",
			escAttr(ce.Entity), escAttr(ce.Tag), escAttr(ce.ID), list,
		))
	}

	sb.WriteString("\t\t</map>\n")

	if len(m.EntityDecls) > 0 {
		sb.WriteString("\t\t<entities>\n")
		for _, ed := range m.EntityDecls {
			sb.WriteString(fmt.Sprintf(
				"\t\t\t<entity name='%s' number='%s'></entity>\n",
				escAttr(ed.Name), escAttr(ed.Number),
			))
		}
		sb.WriteString("\t\t</entities>\n")
	}

	if len(m.InitialEntities) > 0 {
		sb.WriteString("\t\t<initialization>\n")
		for _, ie := range m.InitialEntities {
			sb.WriteString(fmt.Sprintf(
				"\t\t\t<initialentity entity='%s' epush='%t'></initialentity>\n",
				escAttr(ie.Entity), ie.EPush,
			))
		}
		sb.WriteString("\t\t</initialization>\n")
	}

	sb.WriteString("\t</XMLtoEDD>\n")
	sb.WriteString("</mapping>\n")

	return os.WriteFile(path, []byte(sb.String()), 0644)
}

// xmlEscapeComment escapes "--" sequences in comment text so the XML remains valid.
func xmlEscapeComment(s string) string {
	return strings.ReplaceAll(s, "--", "- -")
}

// escAttr escapes a string for use as an XML attribute value.
func escAttr(s string) string {
	var buf strings.Builder
	if err := xml.EscapeText(&buf, []byte(s)); err != nil {
		return s
	}
	return buf.String()
}
