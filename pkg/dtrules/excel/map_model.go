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

// MapXML holds the in-memory model for a MAP sheet, mirroring the XML
// <mapping><XMLtoEDD> structure. Section comments in the XML
// are preserved as SectionComment entries between SetAttribute groups.
//
// Beyond the <map> attribute entries, the model carries the three sections
// the engine's mapping loader depends on — createentity, the entities
// cardinality declarations, and the initialization stack. Dropping them on
// an Excel round-trip silently broke projects (KidAid's regenerated map
// lost its entity stack and nothing could resolve job.program).
type MapXML struct {
	MapName string     // from the MAP: <name> marker in A1
	Entries []MapEntry // ordered list of entries (attributes and section headers)

	// CreateEntities are the <createentity entity tag id> declarations:
	// which XML tags create which entity instances.
	CreateEntities []MapCreateEntity
	// EntityDecls are the <entities><entity name number> declarations:
	// entity cardinality ("1" singleton, "*" many).
	EntityDecls []MapEntityDecl
	// InitialEntities are the <initialization><initialentity> stack:
	// entities pushed (in order) before execution.
	InitialEntities []MapInitialEntity
}

// MapCreateEntity is one <createentity entity='X' tag='t' id='id'/> row.
type MapCreateEntity struct {
	Entity string
	Tag    string
	ID     string
	// List is the optional list='<array field>' attribute: the created
	// entity is appended to that array on its enclosing entity. Dropping it
	// leaves the array empty at runtime, and the rules that iterate it
	// silently compute nothing — StateTax's brackets went this way.
	List string
}

// MapEntityDecl is one <entities><entity name='X' number='1|*'/> row.
type MapEntityDecl struct {
	Name   string
	Number string
}

// MapInitialEntity is one <initialization><initialentity entity='X'
// epush='true'/> row.
type MapInitialEntity struct {
	Entity string
	EPush  bool
}

// MapEntry is a single row in the MAP sheet.
// If IsSection is true, Comment holds the section label and all other fields are empty.
type MapEntry struct {
	IsSection  bool   // true → blank separator row; Comment = section label
	Tag        string // XML tag name
	RAttribute string // RAttribute value (defaults to Tag if empty in XML)
	Enclosure  string // enclosure entity name
	Type       string // attribute type (string, integer, double, boolean, date, …)
	Comment    string // section label when IsSection=true
}
