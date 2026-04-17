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
	"bigint":          docBigInt,
	"bytes":           docBytes,
	"el":              docEL,
	"xml-format":      docXMLFormat,
	"edd":             docEDD,
	"decision-tables": docDecisionTables,
	"operators":       docOperators,
	"examples":        docExamples,
	"mapping":         docMapping,
	"project-layout":  docProjectLayout,
	"database":        docDatabase,
	"architecture":    docArchitecture,
	"embedding":       docEmbedding,
	"workflow":        docWorkflow,
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
		"bytes":           "Immutable byte sequences with constant-time equality (blockchain / token use cases)",
		"el":              "Expression Language syntax (REQUIRED for all tables)",
		"xml-format":      "XML file format specification (EDD and DT)",
		"edd":             "Entity Data Dictionary - defining entities and fields",
		"decision-tables": "How to write decision tables",
		"operators":       "All available operators with examples",
		"examples":        "Complete working examples",
		"mapping":         "Mapping XML and xlsx schema for translating input data into EDD entities",
		"project-layout":  "Project folder conventions and the _dt/_edd/_map file naming rule",
		"database":        "KV database design driven by the EDD: key composition, arrays, references, mapping*key",
		"architecture":    "Dev-time vs deploy-time layouts (files on disk vs single embedded binary)",
		"embedding":       "Embed DTRules rules into a single Go binary via //go:embed (no xlsx or xml at runtime)",
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

===========================================================================
IMPORTANT: EL is the only language to author rules in.
Postfix and bytecode are internal compilation targets — do not write them
by hand.
===========================================================================

EL is the REQUIRED format for writing decision table conditions and actions.

EL is case-insensitive. Articles (a, an, the) are ignored by the parser.
Statements are separated by semicolons. Comments use // or /* ... */.

Always use EL in *_dsl tags
(e.g., condition_dsl, action_dsl, context_dsl, initial_action_dsl).


Overview
--------
EL expressions fit into four entry points depending on where they appear:

  Condition cell  -> boolean expression (bexpr)
  Action cell     -> statement list (statementList)
  Context cell    -> context statement (contextForTable)
  Policy cell     -> any scalar expression (string, number, boolean, date)

Grammar productions are described below, organized by type.


Literals
--------
Integer:       42     -7     0     1000000
Double:        3.14   0.044  100.0
String:        "hello"   'single quoted'   "John's tax"
Boolean:       true   false   default   otherwise   always
               perform when called
Date:          current date       (today's date)
               current timestamp  (returns the current timestamp as string)
Null:          (tested with "is null" / "is not null")


Identifiers and Field Access
-----------------------------
A bare identifier resolves to a typed field in the current context:
    income          -> field in current entity
    taxpayer.income -> field on specific entity

Possessive syntax (equivalent to dot notation):
    taxpayer's income          same as taxpayer.income
    customer's plan's balance  same as customer.plan.balance
    :Customer:plan's balance   colon-prefixed typed chain

Array element access (zero-based index):
    accounts[0]                first element of accounts array
    (long) accounts[i]         element cast to long
    (string) accounts[i]       element cast to string
    (date) accounts[i]         element cast to date
    (boolean) accounts[i]      element cast to boolean


Integer Expressions (iexpr)
----------------------------
Arithmetic:
    income * rate                  multiplication
    gross - deductions             subtraction
    a + b                          addition
    total / count                  division (integer)
    (a + b) * c                    grouping
    -amount                        negation

Built-in integer functions:
    number of accounts                    count of array
    number of accounts where active       count with filter
    length of myArray                     array length
    length of myString                    string length
    index of "sub" in "string"            position of substring (-1 if absent)
    sum of count in orders                sum of integer field across array
    absolute value of amount              absolute value
    get days in year of someDate          days in year containing date
    get days in months for someDate       days in month of date
    get days of months for someDate       day-of-month number
    get yearof someDate                   four-digit year
    days from d1 to d2                    days between two dates
    months from d1 to d2                  whole months between two dates
    years from d1 to d2                   whole years between two dates
    long value of myOperator(args)        integer result from a custom operator

Mutating integer operations (used in actions):
    add to myLong 5                add 5 to variable
    subtract from myLong 3         subtract 3 from variable
    multiply myLong by 2           multiply variable in place
    divide myLong by 4             divide variable in place
    increment myLong               add 1
    decrement myLong               subtract 1

Cast to integer:
    (long) "42"                    parse string to integer
    (long) 3.7                     truncate double to integer
    (long) someTable("key")        table lookup returning integer
    (long) accounts[i]             array element cast to integer


Double Expressions (fexpr)
---------------------------
Arithmetic:
    rate * income                  multiplication (double * double)
    income * 0.044                 double * integer
    gross - deductions             subtraction
    a + b                          addition
    total / count                  division
    -amount                        negation
    (a - b) * c                    grouping

Built-in double functions:
    sum of balance in accounts     sum of double field across array
    absolute value of rate         absolute value
    double value of myOp(args)     double result from custom operator

Rounding:
    amount rounded                         round to nearest integer
    amount rounded to 2 decimal places     round to N decimal places
    amount rounded to 2 decimal places with boundry 0.5
                                           round with custom boundary

Mutating double operations:
    add to myDouble 1.5            add to variable in place
    subtract from myDouble 0.5     subtract from variable in place
    multiply myDouble by 1.1       multiply in place
    divide myDouble by 2.0         divide in place

Cast to double:
    (double) "3.14"                parse string to double
    (double) 42                    promote integer to double
    (double) someTable("key")      table lookup returning double
    (double) accounts[i]           array element cast to double


BigInt Expressions (bigexpr)
-----------------------------
BigInt provides arbitrary-precision integer arithmetic.

Arithmetic:
    amount + other                 addition
    amount - other                 subtraction
    amount * multiplier            multiplication
    amount / divisor               integer division
    -amount                        negation
    absolute value of amount       absolute value

Cast to bigint:
    (bigint) "123456789012345678901234567890"   from string
    (biginteger) "..."                           (biginteger is synonym)
    (bigint) 42                                  from integer
    (bigint) 3.14                                from double (truncates)

See 'dtrules docs bigint' for full bigint documentation.


Bytes Expressions (bytesexpr)
------------------------------
Bytes is an immutable byte-sequence type for opaque binary data (hashes,
tokens, addresses). Equality uses constant-time comparison to prevent
timing side-channels.

Literal syntax:
    0x4a5b6c7d                      hex literal, lowercase or uppercase
    0x                              empty literal (zero-length)

Operations:
    prefix + suffix                 concatenation → bytes (when both are bytes type)
    data from 2 to 6                slice [2,6) → bytes
    length of data                  number of bytes → integer
    data[0]                         byte at index 0 → integer 0-255

Hashes:
    sha256 of data                  SHA-256 → bytes(32)
    keccak256 of data               Keccak-256 (Ethereum) → bytes(32)
    ripemd160 of data               RIPEMD-160 → bytes(20)
    sha3 of data                    SHA3-256 → bytes(32)

Encoding (bytes ↔ string / bigint):
    hex of data                     lowercase hex string, no 0x prefix → string
    bytes of hex "deadbeef"         decode hex string (accepts 0x prefix) → bytes
    base58check of data version 0   base58check encode with version byte → string
    bytes of base58check encoded    decode base58check → bytes (version on stack, pop if not needed)
    bech32 of data hrp "bc"         BIP-173 bech32 encode → string
    bytes of bech32 encoded         BIP-173 bech32 decode → bytes (hrp on stack, pop if not needed)
    bytes of bigint n size 32       big-endian, zero-padded to 32 bytes → bytes
    bigint of bytes data            unsigned big-endian interpretation → bigint

Equality (constant-time):
    hash is equal to expected       bytes == bytes → boolean
    hash is not equal to expected   bytes != bytes → boolean

Local variable:
    local bytes myHash
    local bytes myHash = 0xdeadbeef

EDD field declaration:
    <field name="commitment" type="bytes" default_value=""/>

See 'dtrules docs bytes' for more detail.


String Expressions (strexpr)
-----------------------------
Literals and variables:
    "hello world"                  string literal
    myString                       typed string variable
    taxpayer's name                possessive access

Concatenation (+ operator accepts mixed types):
    firstName + " " + lastName     string + string
    "Amount: " + amount            string + integer
    "Rate: " + rate                string + double
    "Date: " + reportDate          string + date
    "Name: " + personName          string + name value
    "Status: " + myEntity          string + entity
    "Items: " + myArray            string + array

String functions:
    substring of s from 0 to 5     extract characters 0..4
    trim(myString)                  strip leading/trailing whitespace
    change myString to upper case   convert to uppercase
    change myString to lower case   convert to lowercase
    get current timestamp           current date/time as string
    string value of myDouble        double to string
    string value of myInt           integer to string
    string value of myDate          date to string
    string value of boolean myBool  boolean to string
    relationship between e1 and e2  named relationship between entities
    attribute someAttr of myEntity  XML attribute value on entity
    mapping key                     current mapping key (in map context)
    table information               returns name of executing table
    index of "x" in myString        position of substring (-1 if absent)
    s starts with "prefix"          prefix test

Table lookup returning string:
    (string) someTable("key")       table lookup returning string

String from index/cast:
    (string) accounts[i]           array element as string


Boolean Expressions (bexpr)
-----------------------------
Boolean literals:
    true     false     default     otherwise     always
    perform when called

Typed boolean variable:
    eligible                       boolean field
    taxpayer's eligible            possessive
    eligible is true               explicit true check
    eligible is false              explicit false check
    eligible is not true           negated check
    eligible is not false          negated check

Logical operators:
    a AND b    (also: a && b)      both must be true
    a OR b     (also: a || b)      either must be true
    NOT a                          negation

Null tests (work on any type):
    myField is null
    myField is not null

Numeric comparisons (integer, double, or bigint):
    a == b     (also: a is equal to b,   equal to b)
    a != b     (also: a is not equal to b)
    a > b      (also: a is greater than b,   greater than b)
    a >= b     (also: a is greater than or equal to b,   at or above b)
    a < b      (also: a is less than b,   less than b)
    a <= b     (also: a is less than or equal to b,   at or below b)

    fexpr is within N percent of fexpr    percentage proximity test
    fexpr is plus or minus N of fexpr     absolute proximity test

String comparisons:
    s == "text"                    equal
    s != "text"                    not equal
    s is "text"                    same as ==
    s is not "text"                same as !=
    s is equal to ignore case "text"              case-insensitive equal
    s is not equal to ignore case "text"          case-insensitive not equal
    s == "a", "b", "c"             equal to any in list
    s is equal to ignore case "a", "b"            case-insensitive list match
    s starts with "prefix"         prefix test
    s at 3 starts with "x"         starts-with at character offset
    s > "b"    s < "b"    s >= "b"    s <= "b"   lexicographic comparison
    s matches "regex.*"            regular expression match
    s is one of myArray            member of array
    s is not one of myArray        not a member of array

Name comparisons:
    $myName == $otherName          name equality
    $myName != $otherName          name not equal

Date comparisons:
    d1 == d2
    d1 < d2    (also: d1 is before d2)
    d1 > d2    (also: d1 is after d2)
    d1 <= d2
    d1 >= d2
    d1 is between d2 and d3

Entity comparisons:
    e1 == e2                       same entity instance
    e1 != e2                       different instances
    myEntity entity is in context        entity type in context
    myEntity entity is not in context

Entity existence / relationship tests:
    myEntity has a "relationship"           entity has named relationship
    myEntity has a "rel" where bexpr        relationship with filter
    myEntity is "role" of otherEntity       entity plays role
    entity does not have "relationship"     negative has-a test

Array inclusion tests:
    myArray includes value N               array contains integer
    myArray includes string "x"            array contains string
    myArray includes date d                array contains date
    myArray includes entity e              array contains entity
    myArray does include value N           explicit form
    myArray does not include value N       negative test

Collection quantifiers:
    all myArray have bexpr                 every element satisfies condition
    one of myArray has a bexpr             at least one element satisfies

Existence quantifiers:
    there is entity in array where bexpr   any matching entity exists
    is there entity in array where bexpr   question form
    was there entity in array where bexpr  past-tense form
    there is no entity in array where bexpr  none match

Question-form tests:
    does bexpr?
    is bexpr?
    was bexpr?

Custom operator returning boolean:
    boolean value of myOp(a, b)

Cast from index or string:
    (boolean) accounts[i]
    (boolean) "true"


Date Expressions (dexpr)
--------------------------
Sources:
    current date                   today's date
    myDate                         typed date field
    taxpayer's birthDate           possessive

Cast to date:
    (date) "2024-01-15"            parse string to date (pure date, midnight UTC)
    (date) "2026-04-17T21:05:30Z"  parse RFC 3339 timestamp
    (date) "2026-04-17 21:05:30"   parse space-separated datetime
    (date) accounts[i]             array element as date
    (date) someTable("key")        table lookup returning date
    using myEntity(myDate)         delegate to entity's table

Accepted string formats (most-specific tried first):
    RFC 3339 with nanoseconds: "2026-04-17T21:05:30.123456789Z"
    RFC 3339:                  "2026-04-17T21:05:30Z"
    Space-separated datetime:  "2026-04-17 21:05:30"
    Pure date (midnight UTC):  "2026-04-17"

Pure dates serialize as "YYYY-MM-DD"; timestamps serialize as RFC 3339.

Date arithmetic (returns dexpr):
    (5 days)                       duration literal (integer days)

    add N years to myDate          mutating add
    add N months to myDate         mutating add
    add N days to myDate           mutating add
    subtract N years from myDate   mutating subtract
    subtract N months from myDate
    subtract N days from myDate

    d plus N years                 non-mutating date expression
    d plus N months
    d plus N days
    d minus N years
    d minus N months
    d minus N days

    subtract N years from dexpr    expression-level (returns new dexpr)
    subtract N months from dexpr
    subtract N days from dexpr
    add N years to dexpr
    add N months to dexpr
    add N days to dexpr

Date navigation:
    first of years of d            January 1 of d's year
    first of months of d           first day of d's month
    end of months of d             last day of d's month
    earliest of myArray after d    earliest date in array that is after d

Table lookup returning date:
    (date) someTable("key")


Name Expressions (nexpr)
--------------------------
A "name" is a symbolic identifier (prefixed with $ or bare 'name' keyword).

    $myName                        named variable
    name                           bare name keyword
    nameof myEntity                entity's type name
    the name "some.literal"        name from string literal
    name myArray[i]                name at array index
    (name) "some.string"           cast string to name
    using myEntity(myName)         delegate to entity's table


Array Expressions (arrayExpr)
------------------------------
Sources:
    myArray                        typed array field
    policy statements              special context array
    taxpayer's accounts            possessive array

Constructors:
    { e1, e2, e3 }                 array literal (entities, strings, numbers)
    array of values [ v1, v2 ]     array of scalar values
    tokenize "a,b,c" by ","        split string into array

Copies:
    get copy of myArray            shallow copy
    copy of myArray                shallow copy (shorthand)
    get deep copy of myArray       deep copy (entities cloned)
    deep copy of myArray           deep copy (shorthand)

Map through table:
    map myArray through someTable  apply table to each element, return results


Entity Expressions (eexpr)
---------------------------
    myEntity                       typed entity field
    taxpayer's plan                possessive
    new $myName entity             create new entity by name
    new myEntity entity            create new entity of same type
    clone of myEntity              shallow clone
    (entity) someTable(args)       table lookup returning entity
    first myEntity in myArray where bexpr    first matching entity in array
    first myEntity where bexpr     first matching entity in context
    "role" of otherEntity          entity via relationship
    using myEntity(eexpr)          delegate to entity's table


Table Expressions (texpr)
--------------------------
    someTable                      reference to a decision table
    new "tableName" table of "entityType"   create a new table


Statements (Actions)
---------------------
SET - assign a value:
    set result.tax = income * rate
    set result.status = "approved"
    set result.eligible = income > 0
    set myDate = current date
    set myArray = otherArray
    set myBigInt = (bigint) "12345678901234567890"

    set result.str = myDouble            (double to string)
    set result.str = myDate              (date to string)
    set result.str = $myName             (name to string)
    set result.str = someTable           (table to string)
    set result.bool = $myName            (name to boolean)

INCREMENT / DECREMENT:
    increment myLong
    increment myDouble
    decrement myLong
    decrement myDouble

PERFORM - call another decision table:
    perform Calculate_Deductions
    Calculate_Deductions              (implicit perform)
    perform "DynamicTableName"        (perform by name variable)
    perform myTable and on error add myEntity to context and perform ErrorHandler

ADD - add to arrays or numbers:
    add myEntity to myArray
    add myEntity to myArray and to otherArray
    add myEntity if not member to myArray      (no duplicates)
    add myStr to myArray
    add 5 to myArray
    add myDate to myArray
    add myArray to otherArray
    add myArray to otherArray if not member    (set union)

SUBTRACT from numbers:
    subtract 5 from myLong
    subtract 3.14 from myDouble

REMOVE from arrays:
    remove 2 element from myArray array        remove at index
    remove each myEntity from myArray where bexpr   remove matching
    remove $myName from myArray array          remove by name
    remove "string" from myArray array         remove by string value
    remove myEntity from myArray array         remove entity

CLEAR array:
    clear myArray

SORT array:
    sort myArray in ascending order by $nameField
    sort myArray in descending order by $nameField

RANDOMIZE array:
    randomize myArray

XML ATTRIBUTE operations:
    myXml: set attribute "attr" = value
    myXml: add attribute "attr" = value

DEBUG / PRINT (diagnostic output):
    debug "message"
    debug myInteger
    debug myDouble
    debug myBoolean
    debug myEntity
    debug myDate
    debug myArray
    print "message"      (same as debug; alias)

ERRORS AND WARNINGS:
    error <strexpr>      Halt rule-set execution; the string is surfaced to the
                         host as an ELStatementError (distinguishable from
                         runtime/VM errors via errors.As).
    warn  <strexpr>      Non-fatal diagnostic; logs the string and continues.

    Examples:
        error "Policy rejected: income below minimum";
        error "Invalid status: " + applicant.status;
        warn  "Tax bracket lookup returned null";
        if amount < 0 then error "amount must not be negative"; endif

LOCAL VARIABLE declarations (context statements only):
    local entity myVar
    local entity myVar = someEntity
    local long counter = 0
    local double total = 0.0
    local boolean found = false
    local string msg = "start"
    local date d = current date
    local array results
    local bigint amount = (bigint) "0"

IF / ELSE / ENDIF:
    if bexpr then
        set result.x = 1
    endif

    if bexpr then
        set result.x = 1
    else
        set result.x = 0
    endif

    if bexpr then
        set result.x = 1
    else if otherExpr then
        set result.x = 2
    else
        set result.x = 3
    endif

USING - execute sub-table in entity context:
    using myEntity(myDate)      returns date from entity's table
    using myEntity(myInt)       returns integer
    using myEntity(myStr)       returns string
    using myEntity(myBool)      returns boolean
    using myEntity(myFloat)     returns double
    using myEntity(myArray)     returns array
    using myArray 42            integer lookup in array


Context Statements
------------------
Context statements configure how a decision table executes. Place them in
the Contexts row (row 6) in Excel, or in <contexts> in XML.

FOR ALL - iterate over array (one execution per element):
    for all dependents
    for all job.taxpayers
    for all accounts where account.active is true
    for all items allowing array to be removed
    for all items in myEntity
    for all items in myEntity allowing array to be removed
    for all items in myEntity where bexpr
    for all items where bexpr allowing array to be removed

FOR FIRST - find first matching entity:
    for first of dependents where dependent.age < 18
    for first of items and its details where details.valid is true
    for first in myArray where bexpr

FOR LOOP - counter-based iteration:
    for i = 0; i < 10; increment i
    (same syntax as SET statement with SEMI bexpr SEMI statement)

LOCAL VARIABLES - declare table-scoped variables:
    local entity temp
    local entity current = someEntity
    local long counter = 0
    local double total = 0.0
    local boolean found = false
    local string msg = ""
    local date start = current date
    local array results
    local bigint amount

ADD TO CONTEXT - push entity for field resolution:
    add customer to context of this table
    add order.customer to context for this table

DEBUG before table execution:
    debug "Starting execution"
    debug myEntity


FOR FIRST Blocks (in actions)
------------------------------
FOR FIRST can also appear as a block in action context:
    for first of myArray where bexpr then
        set result.found = true
    elseifnonearefound
        set result.found = false
    endff

    for first of myArray and its relatedEntity where bexpr then
        set result.x = relatedEntity.value
    endff

FOR ALL Blocks (in actions)
----------------------------
FORALL iterates over an array in an action block:
    for all myArray { set item.processed = true }

FOREACH iterates with an entity variable:
    for each myEntity in myArray { set result.count = result.count + 1 }
    for each myEntity in myArray where bexpr { ... }
    for each myEntity and its related in myArray { ... }
    for each myEntity and its related in myArray where bexpr { ... }

USING block:
    using :TypeName:fieldName
        { set result.x = field.value }

Block with trailing FORALL:
    { set x = 1; set y = 2 } for all myArray


Best Practices
--------------
1. Use full entity paths: taxpayer.income (not just income)
2. Quote strings: "SINGLE" (not SINGLE)
3. Use semicolons between statements: set a = 1; set b = 2
4. No shorthand: use set X = X + 1 (not X += 1)
5. Use is true/is false for booleans: eligible is true
6. EL is case-insensitive, but use consistent casing for readability


Real-World Examples: Tax Calculation
-------------------------------------
The TaxReturn sample project (sampleprojects/TaxReturn/xml/) uses EL throughout.
These are actual condition_dsl and action_dsl values from those files.

Filing status dispatch (condition cell):
    job.filing_status is equal to "MFJ" or job.filing_status is equal to "QSS"
    job.filing_status is equal to "HOH"
    otherwise

Tax bracket conditions (condition cells, Apply_Tax_Brackets_Single):
    taxable_income at or below bracket_1_single
    taxable_income at or below bracket_2_single

Child Tax Credit conditions (condition cells, Calculate_Child_Tax_Credit):
    dependent.relationship is equal to "child"
    dependent.age < ctc_age_limit
    dependent.has_ssn == true

EITC conditions (condition cells, Calculate_EITC):
    result.total_earned_income > 0
    result.eitc_qualifying_children == 2
    result.eitc_qualifying_children == 0

Standard deduction actions (action cells, Calculate_Standard_Deduction):
    set result.standard_deduction = standard_deduction_mfj
    set result.standard_deduction = standard_deduction_single
    add senior_extra_deduction_married to result.standard_deduction

Taxable income actions (action cells, Calculate_Taxable_Income):
    set result.taxable_income = the maximum of (result.agi - result.total_deduction - result.qbi_deduction) and 0
    set result.qbi_deduction = 0

Tax bracket actions (action cells):
    set result.regular_tax = result.taxable_income * bracket_1_rate
    set result.deduction_used = result.total_itemized
    set result.deduction_used = result.standard_deduction

CTC accumulator actions:
    add constants.ctc_amount to result.total_ctc
    add constants.odc_amount to result.total_odc

Audit trail actions:
    add "Filing as MFJ per Form 1040 Line 1" to job.audit_trail
    add "Filing as " + job.filing_status + " per Form 1040" to job.audit_trail

Table orchestration actions (Compute_Tax_Return entry point):
    perform Calculate_Gross_Income
    perform Calculate_Deductions
    perform Calculate_Taxable_Income
    perform Calculate_Tax_Liability
    perform Calculate_Credits

For-all iteration (action cell, Calculate_Credits):
    for all dependents perform Calculate_Child_Tax_Credit

For-all iteration (action cell, Calculate_EITC):
    for all dependents perform Count_EITC_Qualifying_Child

Colorado state tax conditions (states/CO_dt.xml):
    taxpayer.age >= 55 and taxpayer.age <= 64
    taxpayer.age >= 65


Real-World Examples: Eligibility Rules
----------------------------------------
The KidAid sample project (sampleprojects/KidAid/xml/kidaid_dt.xml) uses EL
for program dispatch and income calculation.

Program dispatch conditions (condition cells):
    job.program == KidAid
    job.program == MEDICAID
    job.program == FOODSTAMPS

Iterating over clients (context cell):
    for all clients

Income filter conditions:
    income.earned == true
    ExcludedIncomeTypes includes the string income.type
    client.applying == true

Income accumulation action:
    add income.amount to the client.totalIncome

FPL percentage calculation action:
    set client_fpl = (100.0 * totalGroupIncome) / FPL

Eligibility note actions:
    Add "Client is eligible for Medicaid, so cannot enroll in KidAid" to client.notes
    Add "Must Provide Validation of Citizenship" to client.notes
    Add "Case must be within a KidAid County" to client.notes

Eligible/ineligible actions:
    set client.eligible = false

Result population actions:
    Set Result.eligible = client.eligible
    Set Result.program = job.program
    Set Result.client_fpl = client.client_fpl

Table orchestration actions (Determine_Eligibility entry point):
    Perform Calculate_Individual_Income
    Perform Calculate_Group_Size
    Perform Evaluate_KidAid_Eligibility
    Perform Evaluate_Results


See Also
--------
  dtrules docs xml-format       XML file structure
  dtrules docs decision-tables  Full decision table guide
  dtrules docs operators        All EL operators with syntax
  dtrules docs bigint           Arbitrary-precision integers
  dtrules docs bytes            Immutable byte sequences (blockchain / token use cases)
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
  - <expression>: EL condition (compiled automatically)
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
bytes     (empty)    Immutable byte sequence        0xdeadbeef
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

const docOperators = `DTRules EL Operators
====================

This document lists every EL operator by category with syntax and examples.
EL is case-insensitive. All examples use EL syntax as it appears in *_dsl tags.

For full grammar context see 'dtrules docs el'.


Arithmetic Operators
--------------------
Operator  EL Syntax                           Result Type   Example
--------  ----------------------------------  -----------   ------------------------------------
+         a + b                               int/double    income + bonus
-         a - b                               int/double    gross - deductions
*         a * b                               int/double    income * 0.044
/         a / b  (also: a div b)              int/double    total / count
-         -a                                  int/double    -amount   (unary negation)

Absolute value:
    absolute value of amount                  same type     absolute value of -5 -> 5

Rounding (double only):
    amount rounded                            double        3.7 rounded -> 4.0
    amount rounded to 2 decimal places        double
    amount rounded to 2 decimal places with boundry 0.5

Mutating shortcuts (action statements):
    increment myLong                          (adds 1 to long field)
    decrement myLong                          (subtracts 1)
    increment myDouble
    decrement myDouble
    add to myLong 5                           (myLong = myLong + 5)
    subtract from myLong 3                    (myLong = myLong - 3)
    multiply myLong by 2                      (myLong = myLong * 2)
    divide myLong by 4                        (myLong = myLong / 4)
    add to myDouble 1.5
    subtract from myDouble 0.5
    multiply myDouble by 1.1
    divide myDouble by 2.0


Comparison Operators
--------------------
Each operator has symbolic and natural-language forms. All are equivalent.

Equal:
    a == b
    a is equal to b
    equal to b

Not equal:
    a != b
    a is not equal to b
    not equal to b

Greater than:
    a > b
    a is greater than b
    greater than b

Greater than or equal:
    a >= b
    a is greater than or equal to b
    greater than or equal to b
    a at or above b

Less than:
    a < b
    a is less than b
    less than b

Less than or equal:
    a <= b
    a is less than or equal to b
    less than or equal to b
    a at or below b

Case-insensitive string equal:
    s is equal to ignore case "text"
    s is equal to ignore case "a", "b", "c"    (list form)

Case-insensitive string not equal:
    s is not equal to ignore case "text"

Numeric proximity tests:
    a is within 5 percent of b      true if |a-b|/b <= 0.05
    a is plus or minus 10 of b      true if |a-b| <= 10

Date ordering:
    d1 is before d2                 same as d1 < d2
    d1 is after d2                  same as d1 > d2
    d1 is between d2 and d3         d2 <= d1 <= d3


Boolean Operators
-----------------
Operator  EL Syntax            Example
--------  -------------------  ----------------------------------------
AND       a AND b              eligible AND income > 0
AND       a && b               eligible && income > 0
OR        a OR b               co_resident OR filed_jointly
OR        a || b               co_resident || filed_jointly
NOT       NOT a                NOT expired
IS NULL   a is null            myField is null
IS NOT NULL  a is not null     myField is not null

Boolean "is" checks:
    myBool is true
    myBool is false
    myBool is not true
    myBool is not false
    taxpayer's eligible is true

Question forms (evaluate to boolean):
    does bexpr?
    is bexpr?
    was bexpr?


String Operators
----------------
Operator      EL Syntax                                  Example
-----------   ----------------------------------------   ------------------------------------
+             s + t                                      firstName + " " + lastName
+             s + intExpr                                "Count: " + total
+             s + floatExpr                              "Rate: " + rate
+             s + dateExpr                               "Date: " + dueDate
+             s + nameExpr                               "Name: " + $person
+             s + entityExpr                             "Entity: " + myEntity
+             s + arrayExpr                              "Items: " + myList

Substring:    substring of s from start to end           substring of name from 0 to 3
Trim:         trim(s)                                    trim(input.value)
Upper case:   change s to upper case                     change input.state to upper case
Lower case:   change s to lower case                     change input.code to lower case
Timestamp:    get current timestamp                      current date/time as string
Index of:     index of sub in s                          index of "x" in myStr   (-> -1 if absent)
Starts with:  s starts with "prefix"
              s at N starts with "prefix"                starts at character offset N

String value of:
    string value of myDouble                             "3.14"
    string value of myInt                                "42"
    string value of myDate                               ISO date string
    string value of boolean myBool                       "true" or "false"

Relationship:
    relationship between e1 and e2                       named relationship string

Attribute:
    attribute "attrName" of myEntity                     XML attribute value

Table information:
    table information                                    name of currently-executing table

Mapping key:
    mapping key                                          current key in a MAP...THROUGH context


Collection Operators
--------------------
Array construction:
    { e1, e2, e3 }                       literal array
    array of values [ v1, v2, v3 ]       scalar array
    tokenize "a,b,c" by ","              split string into array

Array copy:
    get copy of myArray                  shallow copy
    copy of myArray                      shallow copy (shorthand)
    get deep copy of myArray             deep copy (clones entities)
    deep copy of myArray                 deep copy (shorthand)

Map through table:
    map myArray through someTable        apply table to each element

Number of elements:
    number of myArray                    count of all elements
    number of myArray where bexpr        count of matching elements

Length:
    length of myArray                    same as number of
    length of myString                   character count

Sum:
    sum of intField in myArray           sum integer field across array
    sum of doubleField in myArray        sum double field across array

Array inclusion:
    myArray includes value N             contains integer N
    myArray includes string "x"          contains string "x"
    myArray includes date d              contains date d
    myArray includes entity e            contains entity e
    myArray does include value N         (explicit form, same)
    myArray does not include value N     negation

Array membership:
    s is one of myArray                  string is in array
    s is not one of myArray              string is not in array

Quantifiers:
    all myArray have bexpr               every element satisfies
    one of myArray has a bexpr           at least one satisfies


Integer Built-in Functions
--------------------------
Function                                      Returns   Description
--------------------------------------------  --------  ------------------------------------
number of arrayExpr                           integer   count of elements
number of arrayExpr where bexpr               integer   count of matching elements
length of arrayExpr                           integer   array size
length of strexpr                             integer   string length
index of strexpr in strexpr                   integer   substring position (-1 if absent)
sum of iexpr in arrayExpr                     integer   sum of integer field
absolute value of iexpr                       integer   absolute value
get days in year of dexpr                     integer   days in date's year (365 or 366)
get days in months for dexpr                  integer   days in date's month
get days of months for dexpr                  integer   day-of-month (1-31)
get yearof dexpr                              integer   four-digit year
days from d1 to d2                            integer   days between dates
months from d1 to d2                          integer   whole months between dates
years from d1 to d2                           integer   whole years between dates
long value of typedOperator(args)             integer   custom operator result

Cast to integer:
    (long) "42"                               integer   parse string
    (int) "42"                                integer   (long and int are synonyms)
    (long) 3.7                                integer   truncate double
    (long) someTable("key")                   integer   table lookup
    (long) myArray[i]                         integer   array element


Double Built-in Functions
-------------------------
Function                                      Returns   Description
--------------------------------------------  --------  ------------------------------------
sum of fexpr in arrayExpr                     double    sum of double field across array
absolute value of fexpr                       double    absolute value
fexpr rounded                                 double    round to nearest integer
fexpr rounded to N decimal places             double    round to N places
fexpr rounded to N decimal places
    with boundry B                            double    round with custom boundary
double value of typedOperator(args)           double    custom operator result

Cast to double:
    (double) "3.14"                           double    parse string
    (double) 42                               double    promote integer
    (double) someTable("key")                 double    table lookup
    (double) myArray[i]                       double    array element


BigInt Operators
----------------
Operator   EL Syntax                       Example
---------  -----------------------------   ------------------------------------------
+          amount + other                  total + fee
-          amount - other                  balance - withdrawal
*          amount * multiplier             principal * rate
/          amount / divisor                total / shares  (integer division)
-          -amount                         unary negation

absolute value of bigexpr                  absolute value of deficit

Cast to bigint:
    (bigint) "12345678901234567890"         from string
    (biginteger) "..."                      (biginteger is synonym)
    (bigint) 42                             from integer
    (bigint) 3.14                           from double (truncates)

Cast bigint to other types:
    (string) myBigInt                       bigint -> string for display

Comparisons (same operators as integer/double):
    amount == other     amount != other
    amount > other      amount >= other
    amount < other      amount <= other

See 'dtrules docs bigint' for full bigint documentation.


Bytes Operators
---------------
Bytes is an immutable byte-sequence type for opaque binary data.

Literal:
    0x4a5b6c7d                      hex literal (case-insensitive, even length)
    0x                              empty literal

Operations:
    prefix + suffix                 concat two byte sequences → bytes
    data from 2 to 6                slice [from, to) inclusive/exclusive → bytes
    length of data                  number of bytes → integer
    data[0]                         byte at index → integer 0-255

Hashes:
    sha256 of data                  SHA-256 → bytes(32)
    keccak256 of data               Keccak-256 (Ethereum) → bytes(32)
    ripemd160 of data               RIPEMD-160 → bytes(20)
    sha3 of data                    SHA3-256 → bytes(32)

Encoding (bytes ↔ string):
    hex of data                     lowercase hex string without 0x prefix → string
    bytes of hex s                  decode hex string (optional 0x prefix) → bytes
    base58check of data version v   base58check encode → string
    bytes of base58check s          base58check decode → bytes (version on stack)
    bech32 of data hrp s            BIP-173 bech32 encode → string
    bytes of bech32 s               BIP-173 bech32 decode → bytes (hrp on stack)

Bigint ↔ bytes:
    bytes of bigint n size s        big-endian, zero-padded to s bytes → bytes
    bigint of bytes data            unsigned big-endian interpretation → bigint

Equality (constant-time via crypto/subtle.ConstantTimeCompare):
    hash is equal to expected       → boolean
    hash is not equal to expected   → boolean

See 'dtrules docs bytes' for full bytes documentation.


Date Operators
--------------
Source operators:
    current date                            today's date
    current timestamp                       current date/time as string

Arithmetic returning dexpr:
    d plus N years
    d plus N months
    d plus N days
    d minus N years
    d minus N months
    d minus N days
    (N days)                                duration literal

Arithmetic on expressions (non-mutating):
    add N years to dexpr                    returns new dexpr
    add N months to dexpr
    add N days to dexpr
    subtract N years from dexpr
    subtract N months from dexpr
    subtract N days from dexpr

Mutating date statements (action context, operate on typed date fields):
    add N years to typedDate
    add N months to typedDate
    add N days to typedDate
    subtract N years from typedDate
    subtract N months from typedDate
    subtract N days from typedDate

Navigation:
    first of years of d                     January 1 of d's year
    first of months of d                    first day of d's month
    end of months of d                      last day of d's month
    earliest of myArray after d             earliest date in array after d

Cast to date:
    (date) "2024-01-15"                     parse ISO string
    (time) "2024-01-15"                     (time is synonym for date)
    (date) myArray[i]                       array element as date
    (date) someTable("key")                 table lookup


Entity Operators
----------------
Creation:
    new $myName entity                      create entity by name variable
    new myEntity entity                     create entity of same type
    clone of myEntity                       shallow clone

Access:
    myEntity                                typed entity field
    taxpayer's plan                         possessive
    "roleName" of otherEntity               via relationship
    first myEntity in myArray where bexpr   first matching in array
    first myEntity where bexpr              first matching in context
    (entity) someTable(args)                table lookup

Delegation:
    using myEntity(iexpr)                   entity supplies integer
    using myEntity(fexpr)                   entity supplies double
    using myEntity(strexpr)                 entity supplies string
    using myEntity(bexpr)                   entity supplies boolean
    using myEntity(dexpr)                   entity supplies date
    using myEntity(nexpr)                   entity supplies name
    using myEntity(arrayExpr)               entity supplies array


Name Operators
--------------
    $myName                                 named identifier
    name                                    bare name keyword
    nameof myEntity                         entity's type name
    the name "some.literal"                 name from string literal
    name myArray[i]                         name at array index
    (name) "some.string"                    cast string to name
    using myEntity(nexpr)                   name from entity's table


Logical / Control Operators in Statements
-----------------------------------------
IF / ENDIF:
    if income > 50000 then
        set result.bracket = "high"
    endif

IF / ELSE / ENDIF:
    if income > 50000 then
        set result.bracket = "high"
    else
        set result.bracket = "low"
    endif

IF / ELSE IF / ENDIF:
    if income > 100000 then
        set result.bracket = "top"
    else if income > 50000 then
        set result.bracket = "high"
    else
        set result.bracket = "low"
    endif

PERFORM (call another decision table):
    perform Calculate_Deductions
    Calculate_Deductions                   (implicit perform)
    perform "DynamicName"                  (by name variable)
    perform myTable and on error add myEntity to context and perform ErrorHandler

FORALL block (action context):
    for all myArray { set item.processed = true }

FOREACH block:
    for each myEntity in myArray { set result.count = result.count + 1 }
    for each myEntity in myArray where bexpr { ... }
    for each myEntity and its related in myArray { ... }
    for each myEntity and its related in myArray where bexpr { ... }

FOR FIRST block:
    for first of myArray where bexpr then
        set result.found = true
    elseifnonearefound
        set result.found = false
    endff

USING block:
    using :TypeName:fieldName { set result.x = field.value }


Real-World Operator Usage: Tax Rules
--------------------------------------
These expressions come directly from TaxReturn decision table files
(sampleprojects/TaxReturn/xml/).

Comparison operators in condition cells:
    taxable_income at or below bracket_1_single        <=
    taxable_income at or below bracket_2_single        <=
    dependent.age < ctc_age_limit                      <
    result.eitc_qualifying_children == 2               ==
    result.total_earned_income > 0                     >
    dependent.has_ssn == true                          ==
    result.total_itemized > result.standard_deduction  >

String equality in condition cells:
    job.filing_status is equal to "MFJ" or job.filing_status is equal to "QSS"
    job.filing_status is equal to "HOH"
    dependent.relationship is equal to "child"

AND / OR in condition cells:
    job.filing_status is equal to "MFJ" or job.filing_status is equal to "QSS"
    taxpayer.age >= 55 and taxpayer.age <= 64

Arithmetic in action cells:
    set result.regular_tax = result.taxable_income * bracket_1_rate
    set result.standard_deduction = standard_deduction_mfj

Add operator in action cells:
    add constants.ctc_amount to result.total_ctc
    add senior_extra_deduction_married to result.standard_deduction
    add "Filing as MFJ per Form 1040 Line 1" to job.audit_trail

String concatenation in action cells:
    add "Filing as " + job.filing_status + " per Form 1040" to job.audit_trail

Maximum expression in action cells:
    set result.taxable_income = the maximum of (result.agi - result.total_deduction - result.qbi_deduction) and 0


Real-World Operator Usage: Eligibility Rules
---------------------------------------------
These expressions come directly from KidAid decision table files
(sampleprojects/KidAid/xml/kidaid_dt.xml).

String equality in condition cells:
    job.program == KidAid
    income.earned == true
    client.applying == true

Array inclusion test in condition cell:
    ExcludedIncomeTypes includes the string income.type

Add to array in action cell:
    add income.amount to the client.totalIncome
    Add "Must Provide Validation of Citizenship" to client.notes

Boolean assignment in action cell:
    set client.eligible = false

Division in action cell (FPL calculation):
    set client_fpl = (100.0 * totalGroupIncome) / FPL


See Also
--------
  dtrules docs el              Full EL grammar and syntax reference
  dtrules docs bigint          Arbitrary-precision integers
  dtrules docs decision-tables Decision table guide with examples
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


CLI Usage:
----------
# Run with test data
dtrules run ./rules Calculate_Tax --data input.xml

# Or via REST API
curl -X POST http://localhost:8080/execute \
  -H "Content-Type: application/json" \
  -d '{
    "project": "Tax",
    "table": "Calculate_Tax",
    "data": {
      "input": {
        "income": 85000.0,
        "state": "CO"
      }
    }
  }'


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
All rules MUST be written in EL (Expression Language). The EL compiler
generates internal bytecode automatically — never write bytecode by hand.


The One Command: dtrules build
-------------------------------
Run this after any edit — whether you changed Excel or XML:

  dtrules build [path]

dtrules build auto-detects which files changed, runs the correct pipeline,
and leaves canonical Excel files and compiled XML on disk. You cannot skip
steps; the pipeline is always complete.

Flags:
  --from-excel   Force Excel-authored path (Excel → XML)
  --from-xml     Force XML-authored path   (XML → Excel → XML)
  --dry-run      Show what would change without writing files
  -v, --verbose  Verbose output


Excel-Authored Path (default when .xlsx is newer)
--------------------------------------------------
Best for: Business analysts editing rules in spreadsheets.

1. Edit rules in Excel (EDD sheet + DT sheets with EL expressions).
2. Run the pipeline:
     dtrules build
3. Test the compiled rules:
     dtrules -rules ./xml -test ./testfiles -entry Main


XML-Authored Path (default when .xml is newer)
-----------------------------------------------
Best for: AI or developer edits directly in XML.

1. Edit XML files — use EL in *_dsl tags only (condition_dsl, action_dsl,
   context_dsl, initial_action_dsl). Never hand-code postfix.
2. Run the pipeline:
     dtrules build
   This exports your XML to Excel, then re-imports to normalize and compile.
3. Test the compiled rules.


Directory Structure
-------------------
project/
├── excel/                    # Source of truth (canonical after each build)
│   ├── .sync-manifest.json   # Tracks export timestamps (do not commit)
│   ├── Rules.xlsx
│   └── states/
│       ├── CO.xlsx
│       └── CA.xlsx
├── xml/                      # Compiled from Excel — edit only *_dsl tags
│   ├── Rules_edd.xml
│   ├── Rules_dt.xml
│   └── states/
│       ├── CO_edd.xml
│       ├── CO_dt.xml
│       ├── CA_edd.xml
│       └── CA_dt.xml
└── testfiles/                # Test scenarios
    └── TestScenarios/


EL Examples
-----------
EL (in Excel cells or XML *_dsl tags):
    taxpayer.income > 50000
    taxpayer.filing_status == "SINGLE"
    set result.tax = income * rate


Validate Command
----------------
dtrules validate           Check project structure and EL compliance
dtrules validate --strict  Fail if any legacy non-EL files exist

The validate command checks:
  1. Project structure (excel/, xml/, testfiles/)
  2. EL compliance (no hand-coded postfix)
  3. Sync status (no pending user edits)


CI/CD Integration
-----------------
# 1. Build (idempotent — safe to run even when nothing changed)
dtrules build

# 2. Validate
dtrules validate --strict

# 3. Run tests
dtrules -rules ./xml -test ./testfiles -entry Main


Best Practices
--------------
1. Always use 'dtrules build' — never invoke sync import/export manually.
2. Always write EL in *_dsl tags — never hand-code internal formats.
3. Run 'dtrules validate' before committing.
4. Use --strict in CI to enforce EL compliance.
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


Internal Operators (Reference)
------------------------------
These operators are generated by the EL compiler (do not write these directly):

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
  dtrules docs bytes       Immutable byte sequences
`

const docBytes = `Bytes - Immutable Byte Sequences
================================

Bytes is a first-class type for opaque binary data: hashes, addresses,
tokens, or any content that must be compared in constant time. It is the
right type for blockchain policy rules — e.g., "does the provided script
hash match the expected commitment?"

IMPORTANT: Equality uses crypto/subtle.ConstantTimeCompare, preventing
timing side-channel attacks when comparing secrets such as Bitcoin script
hashes or HMAC values.


Literal Syntax
--------------
Hex literals start with 0x (case-insensitive). Length must be even.

    0xdeadbeef              4-byte literal
    0xDEADBEEF              same value — case doesn't matter
    0x                      zero-length literal

Invalid literals produce a compile-time error:
    0x1                     ERROR: odd length
    0xgg                    ERROR: non-hex character


EDD Field Declaration
---------------------
    <field name="script_hash"   type="bytes" default_value=""/>
    <field name="expected_hash" type="bytes" default_value=""/>


Local Variable Declaration
--------------------------
    local bytes myHash                              uninitialized
    local bytes myHash = 0xdeadbeef0102030405      initialized


Operations
----------

Concatenation
    prefix + suffix                 → bytes
    When both operands are bytes-typed, + produces byte concatenation.
    If operands are of other types, + is string or integer concatenation
    as usual.

Slice
    data from 2 to 6                → bytes (indices 2, 3, 4, 5)
    from is inclusive, to is exclusive.
    Out-of-range index is a runtime error.

Length
    length of data                  → integer (number of bytes)

Indexed access
    data[0]                         → integer 0-255 (byte at index 0)
    data[length of data - 1]        last byte
    Out-of-range index is a runtime error.

Equality (constant-time)
    hash is equal to expected       → boolean
    hash is not equal to expected   → boolean
    Uses crypto/subtle.ConstantTimeCompare. Both values must be bytes type.


Encoding
--------
Convert bytes to/from string encodings and bigint:

    hex of data                     lowercase hex, no 0x prefix → string
    bytes of hex "deadbeef"         decode hex (0x prefix optional) → bytes
    base58check of data version 0   base58check encode → string
    bytes of base58check encoded    decode base58check → bytes (version on stack)
    bech32 of data hrp "bc"         BIP-173 bech32 encode → string
    bytes of bech32 encoded         bech32 decode → bytes (hrp on stack)
    bytes of bigint n size 32       big-endian, zero-padded → bytes
    bigint of bytes data            unsigned big-endian → bigint

Bitcoin address example (hash pubkey, base58check with version 0):
    base58check of (ripemd160 of (sha256 of pubkey)) version 0


Blockchain Policy Example
-------------------------
A policy rule that checks whether a received script hash matches the
expected commitment hash:

Context:
    local bytes expected = 0xa914748284390f9e263a4b766a75d99999999999987

Condition:
    received_hash is equal to expected

This is equivalent to:
    crypto/subtle.ConstantTimeCompare(received_hash, expected) == 1

Without constant-time comparison, a timing oracle could leak information
about secret values byte-by-byte.


See Also
--------
  dtrules docs el          EL syntax reference
  dtrules docs operators   All operators including bytes
  dtrules docs bigint      Arbitrary-precision integers
`
