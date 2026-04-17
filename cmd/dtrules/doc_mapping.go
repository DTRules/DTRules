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

const docMapping = `Mapping - XML and xlsx Schema
==============================

The mapping file translates input data (XML or JSON) into DTRules entities.
It lives alongside the EDD and DT files, named *_map.xml or map_*.xml.


Overview
--------
When DTRules receives input data (typically an XML document), it needs to know
which XML tag maps to which entity field. The mapping file provides this
translation layer so that your input data structure and your entity structure
can differ independently.

The mapping file is loaded once at session initialization. It is not a decision
table — it does not execute rules. It only defines how to read input data.


XML Schema
----------

Top-level structure:

  <mapping>
    <XMLtoEDD>
      <map>
        <!-- attribute mappings and entity creation rules -->
      </map>

      <entities>
        <!-- cardinality declarations -->
      </entities>

      <initialization>
        <!-- singleton entities to push onto the context stack -->
      </initialization>
    </XMLtoEDD>
  </mapping>


<setattribute> — Map an XML tag to an entity field
---------------------------------------------------
Each <setattribute> tells the engine: "when you see this XML tag, set this
field on this entity."

  <setattribute
    tag="xml_tag_name"
    RAttribute="entity_field_name"
    enclosure="entity_type_name"
    type="field_type">
  </setattribute>

Attributes:

  tag         Required. The XML element name in the input data.
  RAttribute  The entity field name to populate. Defaults to tag if omitted.
  enclosure   The entity type that owns this field. Without an enclosure the
              mapping applies to whichever entity is current in the context
              (rarely used — always specify enclosure explicitly).
  type        The field type. Supported values:
                string    — text
                integer   — whole number (int)
                double    — decimal number (float64)
                boolean   — true/false
                date      — date value (ISO 8601: YYYY-MM-DD)
                bigint    — arbitrary-precision integer
                entity    — reference to another entity
                array     — list of values

Example — map an XML <filing_status> tag to job.filing_status:

  <setattribute tag='filing_status' RAttribute='filing_status'
                enclosure='job' type='string'></setattribute>

Example — rename the XML tag: input has <dob>, entity has date_of_birth:

  <setattribute tag='dob' RAttribute='date_of_birth'
                enclosure='taxpayer' type='date'></setattribute>

When the same XML tag appears under multiple enclosures (e.g., both taxpayer
and dependent have an 'id' field), list both <setattribute> elements. The
engine resolves which one to use based on the current enclosure context:

  <setattribute tag='id' RAttribute='id' enclosure='taxpayer' type='integer'></setattribute>
  <setattribute tag='id' RAttribute='id' enclosure='dependent' type='integer'></setattribute>


<createentity> — Create an entity instance from an XML element
--------------------------------------------------------------
When the engine encounters the named XML open-tag, it creates a new entity
instance and begins collecting <setattribute> values into it.

  <createentity
    entity="entity_type"
    tag="xml_tag_name"
    id="id_field_name"
    list="list_field_name">
  </createentity>

Attributes:

  entity   Required. The entity type to create (must be defined in the EDD).
  tag      Required. The XML element name that triggers entity creation.
  id       The entity field that holds the unique identifier (optional but
           strongly recommended for multi-instance entities).
  list     The array field on the parent entity to add the new instance to.
           Omit for singleton entities that are not collected into a list.

Example — create a taxpayer entity from each <taxpayer> XML element and
add it to job.taxpayers:

  <createentity entity='taxpayer' tag='taxpayer' id='id' list='taxpayers'>
  </createentity>

Example — create a dependent entity (collected into dependents list):

  <createentity entity='dependent' tag='dependent' id='id' list='dependents'>
  </createentity>


<entity> — Declare entity cardinality
--------------------------------------
The <entities> block declares how many instances of each entity can exist:

  <entity name="job"       number="1"></entity>   <!-- singleton -->
  <entity name="taxpayer"  number="*"></entity>   <!-- zero or more -->
  <entity name="dependent" number="*"></entity>   <!-- zero or more -->

  number="1"   Singleton — exactly one instance exists.
  number="*"   Multi-instance — zero or more instances.
  number="+"   Multi-instance — one or more instances (at least one required).


<initialentity> — Push singletons onto the context stack
---------------------------------------------------------
Singleton entities (number="1") that are NOT created from input XML must be
initialized at startup. The <initialization> block lists them:

  <initialization>
    <initialentity entity='result'    epush='true'></initialentity>
    <initialentity entity='constants' epush='true'></initialentity>
    <initialentity entity='job'       epush='true'></initialentity>
  </initialization>

epush='true' pushes the entity onto the entity stack so decision tables can
resolve its fields without explicit qualification.


xlsx Schema (v1.5.0)
--------------------
The mapping file round-trips to and from an Excel MAP sheet. This allows
business users to review mappings in spreadsheet form.

Sheet structure:

  Row 1:   A1 = "MAP: <map_name>"  (merged across columns A–D)
  Row 2:   Column headers: Tag | RAttribute | Enclosure | Type
  Row 3+:  Data rows (one per <setattribute>) or section comment rows.

Column layout:

  Col A   Tag         XML element name
  Col B   RAttribute  Entity field name
  Col C   Enclosure   Entity type name
  Col D   Type        Field type (string, integer, double, boolean, date, …)

Section comment rows:
  XML <!-- comments --> between <setattribute> groups become separator rows
  in the xlsx. A section row has col A starting with "# " followed by the
  comment text; columns B–D are blank.

  On import the "# " prefix is stripped to recover the original comment.

A1 marker detection:
  Any sheet whose A1 cell starts with "MAP:" (case-insensitive) is treated
  as a MAP sheet by the importer.


Round-Trip Guarantees
---------------------
The XML ↔ xlsx round-trip preserves:

  - All <setattribute> entries (Tag, RAttribute, Enclosure, Type).
  - Section comment rows (stripped of "# " on import, re-added on export).
  - Ordering within a section is stable (insertion order preserved).
  - RAttribute defaults to Tag when blank in xlsx (same as XML behavior).

What is NOT round-tripped:
  - <createentity>, <entity>, <initialentity> blocks live only in XML.
    The xlsx MAP sheet covers only the <setattribute> entries.
  - XML comments outside the <map> element.


Worked Example
--------------
Suppose your input XML looks like this:

  <taxdata>
    <job>
      <tax_year>2025</tax_year>
      <filing_status>single</filing_status>
    </job>
    <taxpayer>
      <id>1</id>
      <name>Jane Doe</name>
      <w2_wages>85000.00</w2_wages>
    </taxpayer>
    <dependent>
      <id>1</id>
      <name>Alex Doe</name>
      <age>10</age>
    </dependent>
  </taxdata>

The corresponding mapping file:

  <mapping>
    <XMLtoEDD>
      <map>
        <!-- Job (root entity) mappings -->
        <setattribute tag='tax_year'      RAttribute='tax_year'      enclosure='job' type='integer'></setattribute>
        <setattribute tag='filing_status' RAttribute='filing_status' enclosure='job' type='string'></setattribute>

        <!-- Taxpayer mappings -->
        <setattribute tag='id'       RAttribute='id'       enclosure='taxpayer' type='integer'></setattribute>
        <setattribute tag='name'     RAttribute='name'     enclosure='taxpayer' type='string'></setattribute>
        <setattribute tag='w2_wages' RAttribute='w2_wages' enclosure='taxpayer' type='double'></setattribute>

        <!-- Dependent mappings -->
        <setattribute tag='id'   RAttribute='id'   enclosure='dependent' type='integer'></setattribute>
        <setattribute tag='name' RAttribute='name' enclosure='dependent' type='string'></setattribute>
        <setattribute tag='age'  RAttribute='age'  enclosure='dependent' type='integer'></setattribute>

        <!-- Create entities from XML tags -->
        <createentity entity='taxpayer'  tag='taxpayer'  id='id' list='taxpayers'></createentity>
        <createentity entity='dependent' tag='dependent' id='id' list='dependents'></createentity>
      </map>

      <entities>
        <entity name='job'       number='1'></entity>
        <entity name='taxpayer'  number='*'></entity>
        <entity name='dependent' number='*'></entity>
      </entities>

      <initialization>
        <initialentity entity='job' epush='true'></initialentity>
      </initialization>
    </XMLtoEDD>
  </mapping>

The same <setattribute> entries as seen from the xlsx MAP sheet:

  A1:  MAP: taxdata

  Row  Tag            RAttribute     Enclosure    Type
  ---- -------------- -------------- ------------ --------
   3   # Job (root entity) mappings
   4   tax_year       tax_year       job          integer
   5   filing_status  filing_status  job          string
   6   # Taxpayer mappings
   7   id             id             taxpayer     integer
   8   name           name           taxpayer     string
   9   w2_wages       w2_wages       taxpayer     double
  10   # Dependent mappings
  11   id             id             dependent    integer
  12   name           name           dependent    string
  13   age            age            dependent    integer


Common Patterns
---------------

Renaming input tags to entity field names:
  Input XML uses abbreviated tags; entities use descriptive names.

  <!-- input: <dob>1990-05-15</dob> -> taxpayer.date_of_birth -->
  <setattribute tag='dob' RAttribute='date_of_birth'
                enclosure='taxpayer' type='date'></setattribute>

  <!-- input: <wages> -> taxpayer.w2_wages -->
  <setattribute tag='wages' RAttribute='w2_wages'
                enclosure='taxpayer' type='double'></setattribute>

Shared tag names across entity types:
  Multiple entities can share the same XML tag name (e.g., 'id', 'name').
  Add one <setattribute> per enclosure; the engine picks the right one
  based on which entity was most recently created.

  <setattribute tag='id' RAttribute='id' enclosure='taxpayer'  type='integer'></setattribute>
  <setattribute tag='id' RAttribute='id' enclosure='dependent' type='integer'></setattribute>
  <setattribute tag='id' RAttribute='id' enclosure='income'    type='integer'></setattribute>

Entity creation with a list:
  Use list= to collect instances into an array field on the job entity.

  <createentity entity='income' tag='income' id='id' list='incomes'></createentity>

Entity creation without a list:
  Singleton-like entities that exist once per job but are created from XML
  (not pre-initialized) omit the list= attribute.

  <createentity entity='noncash_contribution' tag='noncash_contribution' id='id'></createentity>


File Naming
-----------
The mapping file must be in the same directory as the EDD and DT files.
Recognized naming patterns (same conventions as EDD/DT):

  *_map.xml       (e.g., TaxReturn_map.xml)
  map_*.xml       (e.g., map_TaxReturn.xml)
  mapping.xml


See Also
--------
  dtrules docs edd         Entity Data Dictionary — defining entities and fields
  dtrules docs xml-format  Full XML file format reference
  dtrules docs workflow    Development workflow (Excel <-> XML round-trip)
`
