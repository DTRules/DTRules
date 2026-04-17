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
// <mapping><XMLtoEDD><map> structure. Section comments in the XML
// are preserved as SectionComment entries between SetAttribute groups.
type MapXML struct {
	MapName string        // from the MAP: <name> marker in A1
	Entries []MapEntry    // ordered list of entries (attributes and section headers)
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
