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

import (
	"fmt"
	"sort"
	"strings"
)

// Documentation topics embedded in the executable
var docTopics = map[string]string{
	"bigint":           docBigInt,
	"el":               docEL,
	"xml-format":       docXMLFormat,
	"edd":              docEDD,
	"decision-tables":  docDecisionTables,
	"operators":        docOperators,
	"expressions":      docExpressions,
	"sdk":              docSDK,
	"examples":         docExamples,
	"workflow":         docWorkflow,
}

func runDocs(args []string) error {
	if len(args) == 0 {
		printDocIndex()
		return nil
	}

	topic := strings.ToLower(args[0])
	if doc, ok := docTopics[topic]; ok {
		fmt.Println(doc)
		return nil
	}

	fmt.Printf("Unknown topic: %s\n\n", topic)
	printDocIndex()
	return fmt.Errorf("unknown topic: %s", topic)
}

func printDocIndex() {
	fmt.Println("DTRules Documentation")
	fmt.Println("=====================")
	fmt.Println()
	fmt.Println("Available topics:")
	fmt.Println()

	// Sort topics for consistent output
	topics := make([]string, 0, len(docTopics))
	for t := range docTopics {
		topics = append(topics, t)
	}
	sort.Strings(topics)

	descriptions := map[string]string{
		"bigint":          "Arbitrary-precision integer support for financial calculations",
		"el":              "Expression Language syntax (REQUIRED for all tables)",
		"xml-format":      "XML file format specification (EDD and DT)",
		"edd":             "Entity Data Dictionary - defining entities and fields",
		"decision-tables": "How to write decision tables",
		"operators":       "All available operators with examples",
		"expressions":     "Postfix expression syntax (compiled from EL)",
		"sdk":             "Embedding DTRules in Go applications",
		"examples":        "Complete working examples",
		"workflow":        "Development workflow with Excel and XML",
	}

	for _, t := range topics {
		fmt.Printf("  %-18s  %s\n", t, descriptions[t])
	}

	fmt.Println()
	fmt.Println("Usage: dtrules docs <topic>")
	fmt.Println()
	fmt.Println("Example: dtrules docs decision-tables")
}

const docEL = `Expression Language (EL)
========================

EL is the REQUIRED format for writing decision table conditions and actions.
The EL compiler converts human-readable syntax to executable postfix notation.


Why EL is Required
------------------
- READABLE: Business analysts can understand and modify rules
- VALIDATED: Syntax errors caught at compile time, not runtime
- CONSISTENT: Eliminates hand-coded postfix errors

IMPORTANT: Do NOT hand-code postfix. Always use EL in *_dsl tags
(e.g., condition_dsl, action_dsl, context_dsl, initial_action_dsl).


Conditions
----------
Comparison:
    taxpayer.income > 50000
    taxpayer.filing_status == "SINGLE"
    result.has_nexus is true
    customer.age >= 18 AND customer.income > 0

Boolean:
    eligible AND has_documents
    status == "active" OR override is true
    NOT expired

String:
    state == "CO"
    name == "John Doe"


Actions
-------
Assignment:
    set result.tax = income * rate
    set taxpayer.exemptions = taxpayer.exemptions + 1
    set result.status = "approved"

Multiple statements (use semicolons):
    set result.a = 1; set result.b = 2; set result.c = 3

Execute another table:
    perform Calculate_Deductions
    perform Validate_Input

Tables call tables directly - no external orchestration needed:
    perform Calculate_Gross_Income;
    perform Calculate_Deductions;
    perform Calculate_Tax_Liability


Context Statements
------------------
Context statements configure how a decision table executes. They can set up
iteration, declare local variables, add entities to context, or output debug info.
Place in the Contexts row (row 6) of Excel, or <contexts> element in XML.

1. FOR ALL (iterate over arrays):
    for all dependents
    for all job.taxpayers
    for all accounts where account.active is true
    for all items allowing array to be removed

   The table executes once per entity, with that entity in context.

2. FOR FIRST (find first matching):
    for first of dependents where dependent.age < 18
    for first of accounts where account.balance > 0
    for first of items and its details where details.valid is true

   Finds the first matching entity and executes the table with it in context.

3. FOR LOOP (Java-style iteration):
    for i = 0; i < 10; increment i

   Traditional counter-based loop for numeric iteration.

4. LOCAL VARIABLES (declare table-scoped variables):
    local entity tempCustomer
    local entity current = input.customer
    local long counter = 0
    local double total = 0.0
    local boolean found = false
    local string message = "processing"
    local date startDate = current date
    local array items

   Declares variables scoped to the table execution. Supported types:
   entity, long, double, boolean, string, date, array.

5. ADD TO CONTEXT (push entity to context stack):
    add customer to context of this table
    add order.customer to context for this table

   Pushes an entity onto the context stack for field resolution.

6. DEBUG (output before processing):
    debug "Starting table execution"
    debug input.customer

   Outputs debug information before the table processes.


Arithmetic
----------
    income * 0.044                    Multiplication
    gross - deductions                Subtraction
    principal + interest              Addition
    total / count                     Division
    (income - deductions) * rate      Grouping with parentheses


Comparisons
-----------
    a == b        Equal (numeric)
    a == "text"   Equal (string - use quotes)
    a > b         Greater than
    a >= b        Greater than or equal
    a < b         Less than
    a <= b        Less than or equal
    a is true     Boolean check
    a is false    Boolean check


Boolean Logic
-------------
    a AND b       Both must be true
    a OR b        Either can be true
    NOT a         Negation


BigInt Operations
------------------
BigInt values support arbitrary-precision integer arithmetic:
    context local bigint amount = (bigint) input.amount_string
    set result.total = amount * 2
    amount > 1000000000000


XML Structure (EL Format)
-------------------------
<decision_table name="Calculate_Tax" number="1000">
  <conditions>
    <condition name="high_income">
      <expression>taxpayer.income > 100000</expression>
    </condition>
    <condition name="single_filer">
      <expression>taxpayer.filing_status == "SINGLE"</expression>
    </condition>
  </conditions>
  <rules>
    <rule number="1">
      <conditions>
        <high_income>Y</high_income>
        <single_filer>Y</single_filer>
      </conditions>
      <actions>
        <action>
          set result.tax = taxpayer.income * 0.24;
          set result.bracket = "high"
        </action>
      </actions>
    </rule>
  </rules>
</decision_table>


Best Practices
--------------
1. Use full entity paths: taxpayer.income (not just income)
2. Quote strings: "SINGLE" (not SINGLE)
3. Use semicolons between statements: set a = 1; set b = 2
4. No shorthand: use set X = X + 1 (not X += 1)
5. Use is true/is false for booleans: eligible is true


See Also
--------
  dtrules docs xml-format       XML file structure
  dtrules docs decision-tables  Full decision table guide
  dtrules docs operators        All available operators
`

const docXMLFormat = `DTRules XML Format Specification
=================================

DTRules uses two XML file types:
1. EDD (Entity Data Dictionary) - defines entities and their fields
2. DT (Decision Tables) - defines the business rules

IMPORTANT: All decision tables MUST use EL (Expression Language) format.
See 'dtrules docs el' for EL syntax reference.


File Naming Convention
----------------------
  - EDD files: *_edd.xml, edd_*.xml, or edd.xml
  - DT files:  *_dt.xml, dt_*.xml, or decisiontables.xml

Files are loaded in order: EDD files first, then DT files.


EDD XML Structure
-----------------
<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
    <entity name="entity_name" readonly="false">
        <field name="field_name"
               type="string|integer|double|boolean|date|entity|array"
               subtype=""
               default_value="value"
               comment="Description"/>
    </entity>
</entity_data_dictionary>

Field Types:
  - string:   Text values (default_value="")
  - integer:  Whole numbers (default_value="0")
  - double:   Decimal numbers (default_value="0.0")
  - boolean:  true/false (default_value="false")
  - date:     Date values
  - entity:   Reference to another entity (subtype="entity_name")
  - array:    List of values (subtype="element_type")


Decision Table XML Structure (EL Format - REQUIRED)
---------------------------------------------------
<?xml version="1.0" encoding="UTF-8"?>
<decision_tables name="Project_Tables">
    <decision_table name="Table_Name" number="1000">
        <description>What this table does</description>

        <contexts>
            <context_details>
                <context_dsl>for all dependents</context_dsl>
            </context_details>
        </contexts>

        <conditions>
            <condition name="condition_name">
                <expression>taxpayer.income > 50000</expression>
                <comment>Description of condition</comment>
            </condition>
        </conditions>

        <rules>
            <rule number="1">
                <conditions>
                    <condition_name>Y</condition_name>
                </conditions>
                <actions>
                    <action>
                        set result.tax = income * rate;
                        set result.status = "calculated"
                    </action>
                </actions>
                <policy>Business rationale for this rule</policy>
            </rule>
        </rules>
    </decision_table>
</decision_tables>

Context Statement Types (in <contexts>):
  - for all array                   Iterate over entity array
  - for first of array where cond   Find first matching entity
  - for i = 0; i < n; increment i   Counter-based loop
  - local type varname = value      Declare local variable
  - add entity to context of this table   Push to context
  - debug "message"                 Debug output

Key Elements:
  - name attribute on <decision_table>: Table identifier
  - number attribute: Unique table number for ordering
  - <expression>: EL condition (compiled to postfix automatically)
  - <action>: EL statements separated by semicolons
  - Y/N/-: Condition values (Yes/No/Don't care)

DSL Tag Names:
  - <context_dsl>: Context statement in EL (inside <context_details>)
  - <condition_dsl>: Condition expression in EL (inside <condition_details>)
  - <action_dsl>: Action statement in EL (inside <action_details>)
  - <initial_action_dsl>: Initial action in EL (executed before conditions)

Note: For backward compatibility, the system also reads legacy tag names
(*_description instead of *_dsl). New code should use *_dsl tags.

Table Types (in <attribute_fields>):
  - FIRST:    Execute only the first matching row (most common)
  - ALL:      Execute all matching rows in order
  - BALANCED: All condition combinations must be defined

See 'dtrules docs el' for EL syntax.
See 'dtrules docs decision-tables' for detailed examples.
See 'dtrules docs edd' for entity definition details.
`

const docEDD = `Entity Data Dictionary (EDD)
============================

The EDD defines all entities and their fields used by decision tables.


Basic Structure
---------------
<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
    <entity name="customer" readonly="false">
        <field name="name" type="string" default_value=""/>
        <field name="age" type="integer" default_value="0"/>
        <field name="balance" type="double" default_value="0.0"/>
        <field name="active" type="boolean" default_value="false"/>
    </entity>
</entity_data_dictionary>


Field Types
-----------
Type      Default    Description                    Example Values
--------  --------   ----------------------------   --------------
string    ""         Text values                    "hello", "John Doe"
integer   0          Whole numbers                  42, -7, 1000000
double    0.0        Decimal numbers                3.14, -0.5, 100.0
boolean   false      True/false values              true, false
date      null       Date values                    2024-01-15
bigint    0          Arbitrary-precision integer    123456789012345678901234567890
entity    null       Reference to another entity    (see below)
array     []         List of values                 (see below)


Entity References
-----------------
Use type="entity" with subtype to reference another entity:

<entity name="order">
    <field name="customer" type="entity" subtype="customer"/>
    <field name="total" type="double" default_value="0"/>
</entity>


Arrays
------
Use type="array" with subtype for the element type:

<entity name="shopping_cart">
    <field name="items" type="array" subtype="order_item"/>
    <field name="quantities" type="array" subtype="integer"/>
</entity>


Read-Only Entities
------------------
Set readonly="true" for entities that should not be modified:

<entity name="tax_rates" readonly="true">
    <field name="federal_rate" type="double" default_value="0.22"/>
    <field name="state_rate" type="double" default_value="0.05"/>
</entity>


Comments
--------
Add documentation with the comment attribute:

<field name="credit_score"
       type="integer"
       default_value="0"
       comment="FICO score 300-850, 0 if unknown"/>


Best Practices
--------------
1. Use lowercase_with_underscores for names
2. Group related fields in the same entity
3. Use meaningful default values
4. Add comments for non-obvious fields
5. Create separate EDD files for different domains:
   - TaxReturn_edd.xml (main entities)
   - states/CO_edd.xml (Colorado-specific)
   - states/CA_edd.xml (California-specific)


Common Patterns
---------------

Input/Output separation:
    <entity name="input" readonly="true">
        <field name="income" type="double"/>
        <field name="filing_status" type="string"/>
    </entity>

    <entity name="result" readonly="false">
        <field name="tax_due" type="double"/>
        <field name="eligible" type="boolean"/>
        <field name="message" type="string"/>
    </entity>

Constants entity:
    <entity name="constants" readonly="true">
        <field name="min_age" type="integer" default_value="18"/>
        <field name="max_amount" type="double" default_value="10000"/>
    </entity>
`

const docDecisionTables = `Decision Tables
===============

Decision tables encode business rules as conditions and actions.

IMPORTANT: All conditions and actions MUST use EL (Expression Language).
See 'dtrules docs el' for complete EL syntax reference.


Anatomy of a Decision Table
---------------------------

    +---+------------------+------------------+------------------+
    |   | Condition 1      | Condition 2      | Action 1         |
    +---+------------------+------------------+------------------+
    | 1 | input.age >= 18  | input.income > 0 | result.eligible  |
    | 2 | true             | true             | true             |
    | 3 | true             | false            | false            |
    | 4 | false            | *                | false            |
    +---+------------------+------------------+------------------+

Row 1: Column headers (descriptions)
Row 2+: Rules (condition values → action values)


Table Types
-----------

FIRST (most common):
  - Executes the FIRST row where all conditions match
  - Stops after first match
  - Use for: if/else logic, lookup tables

ALL:
  - Executes ALL rows where conditions match
  - Continues through all rows
  - Use for: accumulating values, multiple effects

BALANCED:
  - Requires all condition combinations to be defined
  - Compile-time check for completeness
  - Use for: exhaustive rule sets


Condition Values
----------------

Value        Meaning                      Example
---------    --------------------------   ----------------------
true         Condition must be true       input.age >= 18 → true
false        Condition must be false      input.age >= 18 → false
*            Any value (don't care)       input.age >= 18 → *
-            Skip this row                Used for N/A cases


Action Column Types
-------------------

VALUE (assign to field):
    <column_header>result.status</column_header>
    <column_type>VALUE</column_type>

    Action values: "approved", "denied", 42, true, expression

EXECUTE (run expression):
    <column_header>Calculate Tax</column_header>
    <column_type>EXECUTE</column_type>

    Action values: result.tax = input.income * 0.25

CONTEXT (push entity):
    <column_header>customer</column_header>
    <column_type>CONTEXT</column_type>

    Pushes entity onto context stack for field resolution


XML Structure
-------------
<decision_table>
    <table_name>Determine_Eligibility</table_name>
    <table_number>100</table_number>
    <type>FIRST</type>

    <!-- Condition columns -->
    <condition_columns>
        <column_number>1</column_number>
        <column_header>input.age >= 18</column_header>
        <conditions>
            <condition_details>
                <row_number>1</row_number>
                <condition_value>true</condition_value>
            </condition_details>
            <condition_details>
                <row_number>2</row_number>
                <condition_value>false</condition_value>
            </condition_details>
        </conditions>
    </condition_columns>

    <!-- Action columns -->
    <action_columns>
        <column_number>1</column_number>
        <column_header>result.eligible</column_header>
        <column_type>VALUE</column_type>
        <actions>
            <action_details>
                <row_number>1</row_number>
                <action_value>true</action_value>
            </action_details>
            <action_details>
                <row_number>2</row_number>
                <action_value>false</action_value>
            </action_details>
        </actions>
    </action_columns>
</decision_table>


Calling Other Tables
--------------------
Use 'perform' in an action to call another decision table:

    perform Calculate_Tax_Details

In EL syntax (action column):
    perform Calculate_Deductions;
    perform Validate_Input;

Tables call tables - there is no external orchestration needed. The entry point
table calls other tables, which can call more tables, forming a call graph:

    Compute_Tax_Return (entry point)
    ├── perform Calculate_Gross_Income
    │   ├── perform Process_W2_Income
    │   └── perform Process_Self_Employment
    ├── perform Calculate_Deductions
    └── perform Calculate_Tax_Liability


Context Statements
------------------
Context statements control table execution behavior. The context row (row 6 in
Excel, or <contexts> element in XML) supports six statement types:

FOR ALL - Iteration:
    for all dependents              Iterate over array
    for all job.taxpayers           Iterate with entity path
    for all accounts where active   Iterate with filter condition
    for all items allowing array to be removed

FOR FIRST - Find First Match:
    for first of dependents where dependent.age < 18
    for first of accounts and its owner where owner.active is true

FOR LOOP - Counter-Based:
    for i = 0; i < count; increment i

LOCAL VARIABLES - Table-Scoped Variables:
    local entity temp
    local long counter = 0
    local double sum = 0.0
    local boolean found = false
    local string msg = ""
    local date start = current date
    local array results

ADD TO CONTEXT - Push Entity:
    add customer to context of this table

DEBUG - Output Before Processing:
    debug "Starting execution"


Best Practices
--------------
1. Use descriptive table names: Calculate_Tax, Validate_Input
2. Number tables uniquely (100, 101, 102...)
3. Put most specific conditions first (FIRST type)
4. Use * for "don't care" conditions
5. Keep tables focused on one decision
6. Use EXECUTE columns for side effects
7. Document complex conditions in column headers


Common Patterns
---------------

Lookup table:
    State | Tax Rate
    "CO"  | 0.044
    "CA"  | 0.0725
    *     | 0.0

Validation:
    input.age >= 0 | input.amount > 0 | result.valid | result.error
    false          | *                | false        | "Invalid age"
    *              | false            | false        | "Invalid amount"
    true           | true             | true         | ""

Tiered calculation:
    input.amount >= 100000 | result.rate
    true                   | 0.05
    input.amount >= 10000  | 0.08
    *                      | 0.10
`

const docOperators = `DTRules Operators
=================

DTRules uses postfix (RPN) notation: operands come before operators.

Example: "income 50000 >=" means "income >= 50000"


Arithmetic Operators
--------------------
Operator   Stack Effect              Example
--------   ---------------------     -----------------------
+          (a b -- sum)              10 5 +        → 15
-          (a b -- difference)       10 5 -        → 5
*          (a b -- product)          10 5 *        → 50
/          (a b -- quotient)         10 5 /        → 2
%          (a b -- remainder)        10 3 %        → 1
abs        (a -- |a|)                -5 abs        → 5
neg        (a -- -a)                 5 neg         → -5
round      (a -- rounded)            3.7 round     → 4
floor      (a -- floor)              3.7 floor     → 3
ceil       (a -- ceiling)            3.2 ceil      → 4
min        (a b -- min)              10 5 min      → 5
max        (a b -- max)              10 5 max      → 10


Comparison Operators
--------------------
Operator   Stack Effect              Example
--------   ---------------------     -----------------------
=          (a b -- bool)             10 10 =       → true
!=         (a b -- bool)             10 5 !=       → true
<          (a b -- bool)             5 10 <        → true
>          (a b -- bool)             10 5 >        → true
<=         (a b -- bool)             5 10 <=       → true
>=         (a b -- bool)             10 5 >=       → true


Boolean Operators
-----------------
Operator   Stack Effect              Example
--------   ---------------------     -----------------------
and        (a b -- bool)             true false and  → false
or         (a b -- bool)             true false or   → true
xor        (a b -- bool)             true false xor  → true
not        (a -- bool)               true not        → false


String Operators
----------------
Operator        Stack Effect              Example
--------------  ---------------------     -----------------------
+               (s1 s2 -- concat)         "Hello" " World" + → "Hello World"
length          (s -- len)                "Hello" length      → 5
substring       (s start end -- sub)      "Hello" 0 3 substring → "Hel"
upper           (s -- upper)              "hello" upper       → "HELLO"
lower           (s -- lower)              "HELLO" lower       → "hello"
trim            (s -- trimmed)            "  hi  " trim       → "hi"
contains        (s sub -- bool)           "Hello" "ell" contains → true
startswith      (s prefix -- bool)        "Hello" "He" startswith → true
endswith        (s suffix -- bool)        "Hello" "lo" endswith → true
split           (s delim -- array)        "a,b,c" "," split   → ["a","b","c"]
join            (array delim -- s)        ["a","b"] "," join  → "a,b"


Stack Operators
---------------
Operator   Stack Effect              Description
--------   ---------------------     -----------------------
dup        (a -- a a)                Duplicate top
drop       (a -- )                   Remove top
swap       (a b -- b a)              Swap top two
over       (a b -- a b a)            Copy second to top
rot        (a b c -- b c a)          Rotate top three


Array Operators
---------------
Operator        Stack Effect              Example
--------------  ---------------------     -----------------------
newarray        ( -- [])                  Create empty array
addto           (array val -- array)      Add value to array
length          (array -- len)            Get array length
get             (array idx -- val)        Get element at index
first           (array -- val)            Get first element
last            (array -- val)            Get last element
contains        (array val -- bool)       Check if contains value
sum             (array -- sum)            Sum numeric array
average         (array -- avg)            Average of array


Entity Operators
----------------
Operator        Stack Effect              Example
--------------  ---------------------     -----------------------
new             (name -- entity)          "customer" new
entitypush      (entity -- )              Push onto context
entitypop       ( -- entity)              Pop from context
get             (entity name -- val)      Get field value
put             (entity name val -- )     Set field value


Control Flow
------------
Operator        Stack Effect              Description
--------------  ---------------------     -----------------------
if              (bool {t} {f} -- ?)       Conditional execution
perform         (name -- )                Execute decision table
for             (array {body} -- )        Iterate over array
while           ({cond} {body} -- )       Loop while condition true
break           ( -- )                    Exit loop
return          ( -- )                    Return from table


Special Values
--------------
Operator        Stack Effect              Description
--------------  ---------------------     -----------------------
null            ( -- null)                Push null value
true            ( -- true)                Push true
false           ( -- false)               Push false


Type Conversion
---------------
Operator        Stack Effect              Description
--------------  ---------------------     -----------------------
tostring        (a -- string)             Convert to string
tointeger       (a -- int)                Convert to integer
todouble        (a -- double)             Convert to double
toboolean       (a -- bool)               Convert to boolean
cvbi            (a -- bigint)             Convert to bigint


BigInt Operators
----------------
Operator        Stack Effect              Description
--------------  ---------------------     -----------------------
b+              (a b -- sum)              Add two bigint values
b-              (a b -- difference)       Subtract two bigint values
b*              (a b -- product)          Multiply two bigint values
b/              (a b -- quotient)         Integer division of two bigint values
bmod            (a b -- remainder)        Modulo of two bigint values
babs            (a -- |a|)                Absolute value
bnegate         (a -- -a)                 Negate a bigint value
b=              (a b -- bool)             Equal comparison
b!=, b<>        (a b -- bool)             Not equal comparison
b>              (a b -- bool)             Greater than comparison
b>=             (a b -- bool)             Greater than or equal comparison
b<              (a b -- bool)             Less than comparison
b<=             (a b -- bool)             Less than or equal comparison


Date Operators
--------------
Operator        Stack Effect              Example
--------------  ---------------------     -----------------------
now             ( -- date)                Current date/time
today           ( -- date)                Current date
year            (date -- int)             Get year
month           (date -- int)             Get month (1-12)
day             (date -- int)             Get day of month
adddays         (date n -- date)          Add n days
addmonths       (date n -- date)          Add n months
daysbetween     (d1 d2 -- int)            Days between dates


Common Expression Patterns
--------------------------

Field comparison:
    input.age 18 >=                    → input.age >= 18

Field assignment:
    input.income 0.25 * result.tax =   → result.tax = input.income * 0.25

Conditional:
    input.age 18 >= { "adult" } { "minor" } if

Null check:
    input.value null =                 → input.value == null

String matching:
    input.state "CO" =                 → input.state == "CO"

Range check:
    input.age 18 >= input.age 65 < and → 18 <= age < 65

Calculation with assignment:
    input.principal input.rate * input.years * result.interest =
`

const docExpressions = `Postfix Expression Syntax
=========================

NOTE: You should NOT write postfix directly. Use EL (Expression Language)
instead - the compiler generates postfix automatically.
See 'dtrules docs el' for the required EL syntax.

This document explains the compiled postfix format for debugging purposes.

DTRules uses postfix (Reverse Polish Notation) for all expressions.
In postfix, operands come before operators.


Basic Syntax
------------
Infix:     a + b
Postfix:   a b +

Infix:     (a + b) * c
Postfix:   a b + c *

Infix:     a >= 18 && b > 0
Postfix:   a 18 >= b 0 > and


Field References
----------------
Fields are referenced as: entity.field

Examples:
    input.age              → value of input.age
    result.eligible        → value of result.eligible
    customer.orders.total  → nested field access


Literals
--------
Type       Examples
--------   ----------------------
Integer    42, -7, 0, 1000000
Double     3.14, -0.5, 100.0
String     "hello", "John Doe", ""
Boolean    true, false
Null       null


Common Patterns
---------------

Comparison:
    Infix:   input.age >= 18
    Postfix: input.age 18 >=

Assignment:
    Infix:   result.tax = income * 0.25
    Postfix: income 0.25 * result.tax =

Conditional:
    Infix:   if (age >= 18) "adult" else "minor"
    Postfix: age 18 >= { "adult" } { "minor" } if

Boolean logic:
    Infix:   age >= 18 && income > 0
    Postfix: age 18 >= income 0 > and

    Infix:   status == "active" || override
    Postfix: status "active" = override or

Negation:
    Infix:   !eligible
    Postfix: eligible not

Calculation chain:
    Infix:   (principal * rate * years) / 12
    Postfix: principal rate * years * 12 /


Blocks
------
Code blocks are enclosed in braces: { ... }

Used for:
  - Conditionals: condition { true-branch } { false-branch } if
  - Loops: array { element process } for
  - Grouping multiple operations

Example - conditional assignment:
    input.age 18 >=
    { "adult" result.category = }
    { "minor" result.category = }
    if


Calling Decision Tables
-----------------------
Use 'perform' to call another decision table:

    Calculate_Tax perform

With context setup:
    customer entitypush Calculate_Discount perform entitypop


Reading the Stack
-----------------
Postfix is read left-to-right. Stack grows with values, shrinks with operators.

Expression: input.income 50000 >=

Step 1: input.income    Stack: [75000]
Step 2: 50000           Stack: [75000, 50000]
Step 3: >=              Stack: [true]

Expression: input.base input.rate * 12 /

Step 1: input.base      Stack: [1200]
Step 2: input.rate      Stack: [1200, 0.05]
Step 3: *               Stack: [60]
Step 4: 12              Stack: [60, 12]
Step 5: /               Stack: [5]


Complex Example
---------------
Infix:
    result.tax = max(0, (income - deduction) * rate)

Postfix:
    0 income deduction - rate * max result.tax =

Step by step:
    0                    Stack: [0]
    income               Stack: [0, 100000]
    deduction            Stack: [0, 100000, 15000]
    -                    Stack: [0, 85000]
    rate                 Stack: [0, 85000, 0.044]
    *                    Stack: [0, 3740]
    max                  Stack: [3740]
    result.tax           Stack: [3740, <field ref>]
    =                    Stack: []  (assigns 3740 to result.tax)


Tips
----
1. Read left-to-right, tracking the stack mentally
2. Each operator consumes its operands and pushes result
3. Assignments (=) consume the value and field reference
4. Use spaces to separate tokens
5. String literals use double quotes
6. Entity.field for all field access
`

const docSDK = `Embedding DTRules in Go Applications
=====================================

The SDK package allows embedding decision tables directly into Go binaries.


Installation
------------
go get github.com/DTRules/DTRules/pkg/dtrules/sdk


Basic Usage
-----------
package main

import (
    "log"
    "github.com/DTRules/DTRules/pkg/dtrules/sdk"
)

func main() {
    // Load from directory
    engine, err := sdk.NewEngine("MyRules", sdk.WithDirectory("./rules"))
    if err != nil {
        log.Fatal(err)
    }

    // Create execution context
    ctx := engine.NewContext()
    ctx.SetEntity("input", "income", 75000)
    ctx.SetEntity("input", "filing_status", "single")

    // Execute decision table
    result, err := engine.Execute("Calculate_Tax", ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Get results
    taxDue := result.GetFloat("tax_due")
    eligible := result.GetBool("eligible")
}


Embedding Rules in Binary
-------------------------
Use Go's embed package to compile rules into the binary:

package main

import (
    "embed"
    "log"
    "github.com/DTRules/DTRules/pkg/dtrules/sdk"
)

//go:embed rules/xml/*.xml
var rulesFS embed.FS

func main() {
    // Load from embedded filesystem
    engine, err := sdk.NewEngine("MyRules", sdk.WithFS(rulesFS, "rules/xml"))
    if err != nil {
        log.Fatal(err)
    }
    // Use engine...
}


Loading Options
---------------
// From filesystem directory
sdk.NewEngine("Name", sdk.WithDirectory("./rules"))

// From embedded filesystem
sdk.NewEngine("Name", sdk.WithFS(rulesFS, "rules/xml"))

// From explicit readers
eddFile, _ := os.Open("rules/edd.xml")
dtFile, _ := os.Open("rules/dt.xml")
sdk.NewEngine("Name", sdk.WithEDD(eddFile), sdk.WithDT(dtFile))


Setting Input Values
--------------------
ctx := engine.NewContext()

// Set on specific entity
ctx.SetEntity("customer", "name", "John")
ctx.SetEntity("customer", "age", 35)
ctx.SetEntity("customer", "premium", true)

// Simple key-value (session attributes)
ctx.Set("debug_mode", true)

// Method chaining
ctx.SetEntity("input", "a", 1).
    SetEntity("input", "b", 2).
    SetEntity("input", "c", 3)


Getting Results
---------------
result, _ := engine.Execute("TableName", ctx)

// Typed getters (return zero value if missing/wrong type)
s := result.GetString("status")      // string
n := result.GetInt("count")          // int64
f := result.GetFloat("amount")       // float64
b := result.GetBool("approved")      // bool

// Generic getter
v, ok := result.Get("field_name")    // interface{}, bool


Concurrent Execution
--------------------
Engine is safe for concurrent use. Each context is independent:

var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func(n int) {
        defer wg.Done()
        ctx := engine.NewContext()
        ctx.SetEntity("input", "value", n)
        result, _ := engine.Execute("Process", ctx)
        _ = result.GetInt("output")
    }(i)
}
wg.Wait()


Listing Available Tables
------------------------
tables := engine.ListTables()
for _, name := range tables {
    fmt.Println(name)
}


Listing Entities
----------------
entities := engine.ListEntities()
for _, name := range entities {
    fmt.Println(name)
}


File Naming Convention
----------------------
The SDK auto-detects file types:

EDD files: *_edd.xml, edd_*.xml, edd.xml
DT files:  *_dt.xml, dt_*.xml, decisiontables.xml

Files are loaded: EDD first, then DT.


Project Structure
-----------------
myapp/
├── main.go
├── rules/
│   └── xml/
│       ├── myapp_edd.xml
│       └── myapp_dt.xml
└── go.mod

With embed directive in main.go:
    //go:embed rules/xml/*.xml
    var rulesFS embed.FS


Error Handling
--------------
engine, err := sdk.NewEngine("Rules", sdk.WithDirectory("./rules"))
if err != nil {
    // Failed to load rules (file not found, parse error, etc.)
    log.Fatalf("Failed to initialize: %v", err)
}

result, err := engine.Execute("TableName", ctx)
if err != nil {
    // Execution failed (unknown table, runtime error, etc.)
    log.Printf("Execution error: %v", err)
}
`

const docExamples = `Complete Working Examples
=========================


Example 1: Tax Calculation
--------------------------

EDD (tax_edd.xml):
------------------
<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
    <entity name="input" readonly="true">
        <field name="income" type="double" default_value="0"/>
        <field name="filing_status" type="string" default_value="single"/>
        <field name="state" type="string" default_value=""/>
    </entity>

    <entity name="result" readonly="false">
        <field name="federal_tax" type="double" default_value="0"/>
        <field name="state_tax" type="double" default_value="0"/>
        <field name="total_tax" type="double" default_value="0"/>
        <field name="effective_rate" type="double" default_value="0"/>
    </entity>
</entity_data_dictionary>


Decision Table (tax_dt.xml):
----------------------------
<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
    <decision_table>
        <table_name>Calculate_Tax</table_name>
        <table_number>100</table_number>
        <type>FIRST</type>

        <condition_columns>
            <column_number>1</column_number>
            <column_header>input.income > 0</column_header>
            <conditions>
                <condition_details>
                    <row_number>1</row_number>
                    <condition_value>true</condition_value>
                </condition_details>
                <condition_details>
                    <row_number>2</row_number>
                    <condition_value>false</condition_value>
                </condition_details>
            </conditions>
        </condition_columns>

        <action_columns>
            <column_number>1</column_number>
            <column_header>Calculate Federal</column_header>
            <column_type>EXECUTE</column_type>
            <actions>
                <action_details>
                    <row_number>1</row_number>
                    <action_value>Calculate_Federal_Tax perform</action_value>
                </action_details>
                <action_details>
                    <row_number>2</row_number>
                    <action_value></action_value>
                </action_details>
            </actions>
        </action_columns>

        <action_columns>
            <column_number>2</column_number>
            <column_header>Calculate State</column_header>
            <column_type>EXECUTE</column_type>
            <actions>
                <action_details>
                    <row_number>1</row_number>
                    <action_value>Calculate_State_Tax perform</action_value>
                </action_details>
                <action_details>
                    <row_number>2</row_number>
                    <action_value></action_value>
                </action_details>
            </actions>
        </action_columns>
    </decision_table>

    <decision_table>
        <table_name>Calculate_Federal_Tax</table_name>
        <table_number>101</table_number>
        <type>FIRST</type>

        <condition_columns>
            <column_number>1</column_number>
            <column_header>input.income >= 100000</column_header>
            <conditions>
                <condition_details>
                    <row_number>1</row_number>
                    <condition_value>true</condition_value>
                </condition_details>
                <condition_details>
                    <row_number>2</row_number>
                    <condition_value>false</condition_value>
                </condition_details>
            </conditions>
        </condition_columns>

        <action_columns>
            <column_number>1</column_number>
            <column_header>result.federal_tax</column_header>
            <column_type>VALUE</column_type>
            <actions>
                <action_details>
                    <row_number>1</row_number>
                    <action_value>input.income 0.24 *</action_value>
                </action_details>
                <action_details>
                    <row_number>2</row_number>
                    <action_value>input.income 0.12 *</action_value>
                </action_details>
            </actions>
        </action_columns>
    </decision_table>

    <decision_table>
        <table_name>Calculate_State_Tax</table_name>
        <table_number>102</table_number>
        <type>FIRST</type>

        <condition_columns>
            <column_number>1</column_number>
            <column_header>input.state</column_header>
            <conditions>
                <condition_details>
                    <row_number>1</row_number>
                    <condition_value>input.state "CO" =</condition_value>
                </condition_details>
                <condition_details>
                    <row_number>2</row_number>
                    <condition_value>input.state "CA" =</condition_value>
                </condition_details>
                <condition_details>
                    <row_number>3</row_number>
                    <condition_value>true</condition_value>
                </condition_details>
            </conditions>
        </condition_columns>

        <action_columns>
            <column_number>1</column_number>
            <column_header>result.state_tax</column_header>
            <column_type>VALUE</column_type>
            <actions>
                <action_details>
                    <row_number>1</row_number>
                    <action_value>input.income 0.044 *</action_value>
                </action_details>
                <action_details>
                    <row_number>2</row_number>
                    <action_value>input.income 0.0725 *</action_value>
                </action_details>
                <action_details>
                    <row_number>3</row_number>
                    <action_value>0</action_value>
                </action_details>
            </actions>
        </action_columns>
    </decision_table>
</decision_tables>


Go Code:
--------
package main

import (
    "embed"
    "fmt"
    "log"
    "github.com/DTRules/DTRules/pkg/dtrules/sdk"
)

//go:embed rules/*.xml
var rulesFS embed.FS

func main() {
    engine, err := sdk.NewEngine("Tax", sdk.WithFS(rulesFS, "rules"))
    if err != nil {
        log.Fatal(err)
    }

    ctx := engine.NewContext()
    ctx.SetEntity("input", "income", 85000.0)
    ctx.SetEntity("input", "state", "CO")

    result, err := engine.Execute("Calculate_Tax", ctx)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Federal Tax: $%.2f\n", result.GetFloat("federal_tax"))
    fmt.Printf("State Tax: $%.2f\n", result.GetFloat("state_tax"))
}


Example 2: Eligibility Check
----------------------------

EDD (eligibility_edd.xml):
--------------------------
<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
    <entity name="applicant" readonly="true">
        <field name="age" type="integer" default_value="0"/>
        <field name="income" type="double" default_value="0"/>
        <field name="citizen" type="boolean" default_value="false"/>
        <field name="years_employed" type="integer" default_value="0"/>
    </entity>

    <entity name="decision" readonly="false">
        <field name="eligible" type="boolean" default_value="false"/>
        <field name="reason" type="string" default_value=""/>
        <field name="max_amount" type="double" default_value="0"/>
    </entity>
</entity_data_dictionary>


Decision Table (eligibility_dt.xml):
------------------------------------
<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
    <decision_table>
        <table_name>Check_Eligibility</table_name>
        <table_number>200</table_number>
        <type>FIRST</type>

        <condition_columns>
            <column_number>1</column_number>
            <column_header>applicant.age >= 18</column_header>
            <conditions>
                <condition_details><row_number>1</row_number><condition_value>false</condition_value></condition_details>
                <condition_details><row_number>2</row_number><condition_value>true</condition_value></condition_details>
                <condition_details><row_number>3</row_number><condition_value>true</condition_value></condition_details>
                <condition_details><row_number>4</row_number><condition_value>true</condition_value></condition_details>
            </conditions>
        </condition_columns>

        <condition_columns>
            <column_number>2</column_number>
            <column_header>applicant.citizen</column_header>
            <conditions>
                <condition_details><row_number>1</row_number><condition_value>*</condition_value></condition_details>
                <condition_details><row_number>2</row_number><condition_value>false</condition_value></condition_details>
                <condition_details><row_number>3</row_number><condition_value>true</condition_value></condition_details>
                <condition_details><row_number>4</row_number><condition_value>true</condition_value></condition_details>
            </conditions>
        </condition_columns>

        <condition_columns>
            <column_number>3</column_number>
            <column_header>applicant.income >= 25000</column_header>
            <conditions>
                <condition_details><row_number>1</row_number><condition_value>*</condition_value></condition_details>
                <condition_details><row_number>2</row_number><condition_value>*</condition_value></condition_details>
                <condition_details><row_number>3</row_number><condition_value>false</condition_value></condition_details>
                <condition_details><row_number>4</row_number><condition_value>true</condition_value></condition_details>
            </conditions>
        </condition_columns>

        <action_columns>
            <column_number>1</column_number>
            <column_header>decision.eligible</column_header>
            <column_type>VALUE</column_type>
            <actions>
                <action_details><row_number>1</row_number><action_value>false</action_value></action_details>
                <action_details><row_number>2</row_number><action_value>false</action_value></action_details>
                <action_details><row_number>3</row_number><action_value>false</action_value></action_details>
                <action_details><row_number>4</row_number><action_value>true</action_value></action_details>
            </actions>
        </action_columns>

        <action_columns>
            <column_number>2</column_number>
            <column_header>decision.reason</column_header>
            <column_type>VALUE</column_type>
            <actions>
                <action_details><row_number>1</row_number><action_value>"Must be 18 or older"</action_value></action_details>
                <action_details><row_number>2</row_number><action_value>"Citizenship required"</action_value></action_details>
                <action_details><row_number>3</row_number><action_value>"Minimum income not met"</action_value></action_details>
                <action_details><row_number>4</row_number><action_value>"Approved"</action_value></action_details>
            </actions>
        </action_columns>

        <action_columns>
            <column_number>3</column_number>
            <column_header>decision.max_amount</column_header>
            <column_type>VALUE</column_type>
            <actions>
                <action_details><row_number>1</row_number><action_value>0</action_value></action_details>
                <action_details><row_number>2</row_number><action_value>0</action_value></action_details>
                <action_details><row_number>3</row_number><action_value>0</action_value></action_details>
                <action_details><row_number>4</row_number><action_value>applicant.income 0.3 *</action_value></action_details>
            </actions>
        </action_columns>
    </decision_table>
</decision_tables>


Go Code:
--------
ctx := engine.NewContext()
ctx.SetEntity("applicant", "age", 25)
ctx.SetEntity("applicant", "income", 50000.0)
ctx.SetEntity("applicant", "citizen", true)

result, _ := engine.Execute("Check_Eligibility", ctx)

if result.GetBool("eligible") {
    fmt.Printf("Approved! Max amount: $%.2f\n", result.GetFloat("max_amount"))
} else {
    fmt.Printf("Denied: %s\n", result.GetString("reason"))
}
`

const docWorkflow = `DTRules Development Workflow
============================

CRITICAL: Excel is the System of Record
----------------------------------------
ALL postfix expressions MUST come from EL (Expression Language) compilation.
Hand-coded postfix is NOT allowed. The EL compiler generates postfix from
human-readable expressions during import.

Required workflow:
  1. User edits Excel      <- Source of truth (EL expressions)
  2. dtrules sync import   <- Compiles EL to postfix, generates XML
  3. dtrules -rules xml    <- Use compiled rules

If AI/developers modify XML directly:
  1. Edit XML (add/modify EL expressions in *_dsl tags)
  2. dtrules sync export   <- Update Excel with standard formatting
  3. dtrules sync import   <- Re-import to compile EL to postfix


Excel-First Workflow
--------------------
Best for: Business analysts editing rules in spreadsheets

1. Create/edit rules in Excel
   - EDD sheet: entity definitions
   - DT sheets: decision tables with EL expressions

2. Import to XML (compiles EL to postfix)
   dtrules sync import

3. Test
   dtrules -rules ./xml -test ./testfiles -entry Main

4. Validate
   dtrules validate


AI/Developer Workflow
---------------------
Best for: Programmatic rule editing

1. Check for pending user edits FIRST
   dtrules sync check

2. Edit XML files (EL in *_dsl tags only, NOT postfix)
   - *_edd.xml: entity definitions
   - *_dt.xml: decision tables (use condition_dsl, action_dsl, context_dsl)

3. Export to Excel (records changes)
   dtrules sync export

4. Re-import to compile EL
   dtrules sync import

5. Test
   dtrules -rules ./xml -test ./testfiles

IMPORTANT: Never hand-code postfix. Always use EL expressions in
the *_dsl tags. The import process compiles them automatically.


Validate Command
----------------
dtrules validate           Check project structure and EL compliance
dtrules validate --strict  Fail if any legacy postfix files exist
dtrules validate --allow-legacy  Allow legacy postfix (migration mode)

The validate command checks:
  1. Project structure (excel/, xml/, testfiles/)
  2. EL compliance (no hand-coded postfix)
  3. Sync status (no pending user edits)


Sync Commands
-------------
dtrules sync status    Show sync state
dtrules sync check     Check for pending user edits (exits non-zero if any)
dtrules sync import    Import Excel → XML (compiles EL to postfix)
dtrules sync export    Export XML → Excel (fails if user edits pending)
dtrules sync auto      Auto-detect direction and sync


Directory Structure
-------------------
project/
├── excel/                    # Source of truth
│   ├── .sync-manifest.json   # Tracks export timestamps
│   ├── Rules.xlsx
│   └── states/
│       ├── CO.xlsx
│       └── CA.xlsx
├── xml/                      # Generated from Excel - DO NOT EDIT POSTFIX
│   ├── Rules_edd.xml
│   ├── Rules_dt.xml
│   └── states/
│       ├── CO_edd.xml
│       ├── CO_dt.xml
│       ├── CA_edd.xml
│       └── CA_dt.xml
└── testfiles/                # Test scenarios
    └── TestScenarios/


EL vs Postfix
-------------
EL (in Excel cells or XML *_dsl tags):
    taxpayer.income > 50000
    taxpayer.filing_status == "SINGLE"
    set result.tax = income * rate

Postfix (generated by compiler - never hand-code):
    taxpayer.income 50000 >
    taxpayer.filing_status "SINGLE" =
    income rate * result.tax =


Resolving Conflicts
-------------------
If 'dtrules sync export' fails because Excel was modified:

1. Import their changes first:
   dtrules sync import

2. Re-apply your XML changes (edit *_dsl tags, not postfix)

3. Export the merged result:
   dtrules sync export

4. Re-import to compile:
   dtrules sync import


CI/CD Integration
-----------------
# In your CI pipeline:

# 1. Validate project
dtrules validate --strict
if [ $? -ne 0 ]; then
    echo "Validation failed!"
    exit 1
fi

# 2. Run tests
dtrules -rules ./xml -test ./testfiles -entry Main


Migrating Legacy Postfix
------------------------
If you have existing XML with hand-coded postfix:

1. Run validate to identify legacy files:
   dtrules validate

2. For each legacy file:
   a. Open the corresponding Excel file
   b. Add EL expressions to the DSL column (or description column for legacy)
   c. Run: dtrules sync import

3. Verify migration:
   dtrules validate --strict


Best Practices
--------------
1. NEVER edit postfix directly - always use EL expressions
2. Always run 'dtrules sync check' before editing XML
3. Always run 'dtrules sync import' after editing Excel
4. Run 'dtrules validate' before committing
5. Use --strict in CI to catch any hand-coded postfix
`

const docBigInt = `BigInt - Arbitrary-Precision Integers
=====================================

BigInt provides arbitrary-precision integer arithmetic for financial calculations
where standard 64-bit integers are insufficient.


Use Cases
---------
- Token amounts in cryptocurrency (nanoACME: 10^8 precision, values > 10^18)
- Financial calculations requiring exact precision
- Large integer math without overflow concerns


Declaring BigInt Fields (EDD)
-----------------------------
<entity name="budget" readonly="false">
    <field name="supply_limit" type="bigint" default_value="0"/>
    <field name="amount_issued" type="bigint" default_value="0"/>
    <field name="weekly_budget" type="bigint" default_value="0"/>
</entity>


EL Syntax
---------
BigInt values are created from strings or by casting:

    context local bigint amount = "12345678901234567890"
    context local bigint total = (bigint) input.amount_string
    set result.balance = amount + total


Arithmetic Operators
--------------------
All standard arithmetic operators work with bigint:

    amount + other           Addition
    amount - other           Subtraction
    amount * multiplier      Multiplication
    amount / divisor         Integer division
    amount % divisor         Modulo (remainder)


Comparison Operators
--------------------
Standard comparison operators:

    amount > threshold       Greater than
    amount >= minimum        Greater than or equal
    amount < maximum         Less than
    amount <= limit          Less than or equal
    amount == expected       Equal
    amount != forbidden      Not equal


Type Conversions
----------------
Convert between types:

    (bigint) "123456789"     String to bigint
    (bigint) 42              Integer to bigint
    (string) amount          BigInt to string (for display/storage)


Postfix Operators (Reference)
-----------------------------
These are the compiled postfix operators (generated from EL):

    b+        Add two bigint values
    b-        Subtract two bigint values
    b*        Multiply two bigint values
    b/        Integer division
    bmod      Modulo (remainder)
    babs      Absolute value
    bnegate   Negate a value
    b=        Equal comparison
    b!=, b<>  Not equal comparison
    b>        Greater than
    b>=       Greater than or equal
    b<        Less than
    b<=       Less than or equal
    cvbi      Convert to bigint


Example: Staking Rewards
------------------------
<entity name="staking" readonly="false">
    <field name="supply_limit" type="bigint" default_value="50000000000000000000"/>
    <field name="acme_issued" type="bigint" default_value="0"/>
    <field name="weekly_budget" type="bigint" default_value="0"/>
</entity>

Decision table condition:
    staking.acme_issued < staking.supply_limit

Decision table action:
    set staking.weekly_budget = (staking.supply_limit - staking.acme_issued) * 16 / 100 * 7 / 365


JSON Input/Output
-----------------
BigInt values are represented as strings in JSON to preserve precision:

{
    "staking": {
        "supply_limit": "50000000000000000000",
        "acme_issued": "12345678901234567890"
    }
}


Best Practices
--------------
1. Use bigint for any value that might exceed 2^63 (9.2 quintillion)
2. Store amounts as strings in JSON/databases to preserve precision
3. Use integer division (/) - there is no floating-point bigint
4. Convert to string for display: (string) amount


See Also
--------
  dtrules docs el          EL syntax reference
  dtrules docs edd         Entity field definitions
  dtrules docs operators   All operators including bigint
`
