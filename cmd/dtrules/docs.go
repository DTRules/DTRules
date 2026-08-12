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
	"bigint":             docBigInt,
	"bytes":              docBytes,
	"cli":                docCLI,
	"authoring-contract": docAuthoringContract,
	"fixed":              docFixed,
	"el":                 docEL,
	"xml-format":         docXMLFormat,
	"edd":                docEDD,
	"decision-tables":    docDecisionTables,
	"debug":              docDebug,
	"operators":          docOperators,
	"examples":           docExamples,
	"mapping":            docMapping,
	"project-layout":     docProjectLayout,
	"database":           docDatabase,
	"architecture":       docArchitecture,
	"embedding":          docEmbedding,
	"warnings":           docWarnings,
	"workflow":           docWorkflow,
	"authoring":          docAuthoring,
	"entry-points":       docEntryPoints,
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
		"bigint":             "Arbitrary-precision integer support for financial calculations",
		"bytes":              "Immutable byte sequences with constant-time equality (blockchain / token use cases)",
		"cli":                "Getting started with the dtrules binary: install, init, build, validate, verify",
		"authoring-contract": "READ FIRST when changing rules: Excel is the system of record; the two ways to author; what verify enforces",
		"fixed":              "256-bit fixed-point type (10^-8 grid) for token/staking/blockchain decimal math",
		"el":                 "Expression Language syntax (REQUIRED for all tables)",
		"xml-format":         "XML file format specification (EDD and DT)",
		"edd":                "Entity Data Dictionary - defining entities and fields",
		"decision-tables":    "How to write decision tables",
		"debug":              "Traces, the trace debugger, Find/why, reports, and speculative reruns",
		"operators":          "All available operators with examples",
		"examples":           "Complete working examples",
		"mapping":            "Mapping XML and xlsx schema for translating input data into EDD entities",
		"project-layout":     "Project folder conventions and the _dt/_edd/_map file naming rule",
		"database":           "KV database design driven by the EDD: key composition, arrays, references, mapping*key",
		"architecture":       "Dev-time vs deploy-time layouts (files on disk vs single embedded binary)",
		"embedding":          "Embed DTRules rules into a single Go binary via //go:embed (no xlsx or xml at runtime)",
		"warnings":           "Every advisory warning kind, repro, and what to do about it",
		"workflow":           "Development workflow with Excel and XML",
		"authoring":          "Go authoring SDK: open, edit, execute, and test projects programmatically",
		"entry-points":       "Run multiple decision tables as separate entry points against one loaded session",
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

What case-insensitive means here:
    taxpayer.filing_status and Taxpayer.Filing_Status are the SAME name.
    Not two names that clash — one name, written two ways. There is no
    ambiguity to resolve and nothing to disambiguate.

    Case is for you, not the engine. Use it to make a name readable, or to
    follow your project's conventions: Compute_Tax_Return for a table,
    taxpayer.gross_income for a field. The engine ignores it entirely when
    resolving a name, and always will.

    DTRules preserves the spelling you write when it writes your rules back
    out to Excel or XML, so your conventions survive a build. It does not
    treat a different spelling as a different thing.

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
Fixed:         1.5fp   0fp   100.0FP   (8-decimal fixed-point, see 'docs fixed')
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
    sum of count in orders where active   sum with filter (parity with number of)
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
    add 5 to myLong                add 5 to variable
    subtract 3 from myLong         subtract 3 from variable
    increment myLong               add 1
    decrement myLong               subtract 1

    There is no in-place multiply or divide shortcut; write the assignment:
    set myLong = myLong * 2
    set myLong = myLong / 4

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
    amount rounded to 2 decimal places with boundary 0.5
                                           round with custom boundary
    the ceiling of amount                  round up   (postfix: ceiling)
    the floor of amount                    round down (postfix: floor)

Min / max (also 'smaller of' / 'larger of'; 'the' is optional):
    the minimum of a and b                 lesser of two values  (postfix: fmin)
    the maximum of a and b                 greater of two values (postfix: fmax)
    the maximum of (result.agi - deduction) and 0.0

Mutating double operations:
    add 1.5 to myDouble            add to variable in place
    subtract 0.5 from myDouble     subtract from variable in place
    set myDouble = myDouble * 1.1  multiply in place (no shortcut form)
    set myDouble = myDouble / 2.0  divide in place (no shortcut form)

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


Fixed Expressions (fpexpr)
---------------------------
Fixed is a signed 256-bit fixed-point decimal type on a 10^-8 grid.
Target use: token amounts, staking rewards, fee/rate math — any place
float64 drift is unacceptable.

Literal syntax:
    1.5fp                          mantissa 150_000_000 (case-insensitive fp)
    0fp                            zero
    100.0FP                        mantissa 10_000_000_000
    -0.00000001fp                  smallest negative value

At most 8 fractional digits in a literal (more is a compile-time error).

Arithmetic:
    a + b                          addition (exact)
    a - b                          subtraction (exact)
    a * b                          multiplication (truncate toward zero)
    a / b                          division (truncate toward zero)
    -a                             negation
    absolute value of a            absolute value

Cast to fixed:
    (fixed) 42                     from integer (exact, auto-scaled)
    (fixed) myBigInt               from bigint (exact, range-checked)
    (fixed) 3.14                   from double — explicit cast required
    (fixed) "1.25"                 parse string

Comparisons (exact mantissa compare — no epsilon needed):
    a == b    a != b
    a > b     a >= b
    a < b     a <= b

Fixed-specific operators:
    fpmin a b                      minimum of two fp values
    fpmax a b                      maximum of two fp values
    fptrunc a                      truncate fractional part toward zero

Mutating variants (action context):
    add to myFixed 1.5fp
    subtract from myFixed 0.25fp
    multiply myFixed by 2fp
    divide myFixed by 3fp
    increment myFixed              adds 1.00000000
    decrement myFixed              subtracts 1.00000000

Local variable:
    local fixed rate = 0.05fp
    local fixed x

EDD field declaration:
    <field name="reward_rate" type="fixed" default_value="0.05fp"/>

Interactions with other numerics:
    int + fixed    -> fixed (exact)
    bigint + fixed -> fixed (exact, range-checked)
    double + fixed -> ERROR — use explicit (fixed) cast

See 'dtrules docs fixed' for full fixed documentation.


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
    policy statements              the run's policy-statement report
    taxpayer's accounts            possessive array

Constructors:
    { e1, e2, e3 }                 array literal (entities, strings, numbers)
    array of values [ v1, v2 ]     array of scalar values
    tokenize "a,b,c" by ","        split string into array

Iteration (context cell):
    for all myArray                forward, the usual form
    for all myArray in reverse     last element to first
    for all myArray where bexpr    filtered
    for all myArray allowing array to be removed
                                   reverse, so removing inside the body is safe

  In an action cell the body is a block, and its statements need their own
  semicolons:
    for all myArray in reverse { set x = 1; }
    for all myArray in reverse where bexpr { set x = 1; }

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

    Create-and-push idiom — create an entity AND put it on the entity
    stack for the rest of the table run (postfix: createentity entitypush):
        add new result entity to context of this table

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
    for all payouts as p                 bind each element to alias p
    for all payouts as p where p.amount > 0   alias + filter

    The "as <alias>" form binds each iteration's element to a named local,
    referenced as <alias>.field. Use it to disambiguate nested loops over the
    same entity type (no shadowing):
        for all relatives as parent
            for all relatives as child where child.parent_id == parent.id

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
fixed     0.0fp      8-decimal fixed-point          1.5fp, 100.00000000fp
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


Access (input / output / internal)
----------------------------------
The access attribute declares how the rules use a field. It drives the
EDD-usage analysis in 'dtrules review' so genuine inputs and outputs
aren't mis-flagged:

  access="r"    Input. Set by the host / mapping, read by the rules.
                Flagged only if NEVER read by any rule.
  access="w"    Output. Set by the rules, consumed by the host (or emitted
                via the mapping). NOT read back in DSL by design, so it is
                NOT flagged "write-only". Flagged only if NEVER written
                ("declared output never written").
  access="rw"   Internal / read-write (the default when omitted). Flagged
                "unused" if never referenced, "write-only" if set but never
                read.

  <field name="agi"        type="double" access="r"/>   <!-- input  -->
  <field name="dose_mg"    type="integer" access="w"/>  <!-- output -->
  <field name="subtotal"   type="double" access="rw"/>  <!-- internal -->

Note: an array output appended via 'add X to result.list' is seen as a
read of the array, so append-style outputs (e.g. a warnings/rationale
list) stay access="rw", not "w". Use "w" for scalar set-once outputs.


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


Policy Statements
-----------------

A policy statement documents the conclusion a column reached. Each rule
column can carry one, and the text is a template: {expr} substitutes a
runtime value.

    Column 4: thing.value is out of range, i.e.  {thing.value}

Statements collect on their own. When a column fires, its statement is
rendered against the data as of that decision and appended to the run's
policy-statement report — no rule has to ask, and the report spans every
table performed since the last clear. Two EL phrases use it:

    add the policy statements to <array>    copy the report into a field
    clear the policy statements             start the next report

That is what documents a run rather than a single table. Evaluate a
household member against every program, then read back everything those
program tables concluded for that member:

    for all household.members
      clear the policy statements;
      perform Evaluate_All_Programs;
      add new person_report entity to context of this table;
      set person_report.person = person.name;
      add the policy statements to person_report.findings;
      add person_report to household.report;

Without the clear, each member's findings would also carry every earlier
member's — the report accumulates until something empties it.

A table that reports per iteration puts the clear in an initial action,
which runs once per pass of its context. TestProject does exactly this: a
"for all things" context, "clear the policy statements" as the initial
action, and "add the policy statements to the job.notes" as the action —
one statement per thing, each naming that thing's value.


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
    for all payouts as p            Bind each element to alias p (p.field)
    for all payouts as p where p.amount > 0   Alias + filter

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


Shared Constants: Push Once, Reference Unqualified (RECOMMENDED)
---------------------------------------------------------------
A bare identifier resolves against the ENTITY STACK: DTRules searches the
stack from the most-recently-pushed entity downward and uses the first one
that declares the field. The stack PROPAGATES DOWN the perform call chain —
an entity pushed by a table is still on the stack while every table it
performs runs.

This makes a powerful pattern for the constant pools that regulations and
policies need by the dozen. Instead of qualifying every read:

    constants.reduced_dose            constants.standard_dose
    constants.adult_age               constants.renal_ccr_threshold

push the constants entity ONCE onto the context of the ENTRY (top) table:

    add constants to context of this table       (in Determine_Therapy)

Now every table reachable from that entry — Select_Medication,
Determine_Dose, Check_Drug_Interactions, ... — can write the fields bare:

    reduced_dose      standard_dose      adult_age      renal_ccr_threshold

Guidance for authors and LLMs generating rules:
- Put a project's shared constants/config in one entity and push it at the
  single entry table. Do NOT add the context to every leaf table — the
  stack already propagates down perform calls.
- Prefer unqualified field names once the entity is on the stack; reach for
  the entity.field form only to disambiguate when two stacked entities
  declare the same field name.
- "dtrules review" emits a context hint when a table references one entity's
  fields with a qualifier many times and that entity is not on its stack —
  that's the cue to push the entity at the entry table.


Best Practices
--------------
1. Use descriptive table names: Calculate_Tax, Validate_Input
2. Number tables uniquely (100, 101, 102...)
3. Put most specific conditions first (FIRST type)
4. Use * for "don't care" conditions
5. Keep tables focused on one decision
6. Use EXECUTE columns for side effects
7. Document complex conditions in column headers
8. Push shared constants onto the entry table's context and reference their
   fields unqualified (see "Shared Constants" above)


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
    amount rounded to 2 decimal places with boundary 0.5
    the ceiling of amount                     double        3.2 -> 4.0   (postfix: ceiling)
    the floor of amount                       double        3.7 -> 3.0   (postfix: floor)

Min / max ('minimum of'/'smaller of', 'maximum of'/'larger of'; 'and' or comma):
    the minimum of a and b                    double        min(a, b)    (postfix: fmin)
    the maximum of a and b                    double        max(a, b)    (postfix: fmax)

Mutating shortcuts (action statements):
    increment myLong                          (adds 1 to long field)
    decrement myLong                          (subtracts 1)
    increment myDouble
    decrement myDouble
    add 5 to myLong                           (myLong = myLong + 5)
    subtract 3 from myLong                    (myLong = myLong - 3)
    add 1.5 to myDouble
    subtract 0.5 from myDouble

    No in-place multiply or divide shortcut exists — use the assignment form:
    set myLong = myLong * 2                   (myLong = myLong * 2)
    set myLong = myLong / 4                   (myLong = myLong / 4)
    set myDouble = myDouble * 1.1
    set myDouble = myDouble / 2.0


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
    sum of amount in myArray where bexpr sum only matching elements

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
    with boundary B                            double    round with custom boundary
the ceiling of fexpr                          double    round up (postfix: ceiling)
the floor of fexpr                            double    round down (postfix: floor)
the minimum of fexpr and fexpr                double    lesser value (postfix: fmin)
the maximum of fexpr and fexpr                double    greater value (postfix: fmax)
double value of typedOperator(args)           double    custom operator result

Cast to double:
    (double) "3.14"                           double    parse string
    (double) 42                               double    promote integer
    (double) someTable("key")                 double    table lookup
    (double) myArray[i]                       double    array element


Combinatorial Operators
-----------------------
Generators that discover structures in an entity array — subsets, key
groups, consecutive runs — and materialize each structure as an entity of
a caller-named EDD type appended to a destination array. Tables then
iterate the results with ordinary 'for all' contexts and score them with
ordinary conditions: the loop stays in the operator, the policy stays in
the table. All four are statement-form operator calls (#980).

    combinations(src, k, "combo", "value", dest)
        Every k-card combination of src as a "combo" entity with fields
        members (the k entities, by reference), count, and sum of the
        named integer field ("" skips the sum). Source cap: 20.

    subsets(src, "combo", "value", dest)
        Every non-empty subset of src: 2^n - 1 combo entities.
        Source cap: 12 (4095 subsets).

    groupby(src, "rank", "group", dest)
        Partition src by an integer field; one "group" entity per
        distinct value, in first-seen order, with fields key, count,
        and members.

    maximalruns(src, "rank", 3, "run", dest)
        Every maximal interval of consecutive field values of length >=
        minlen as a "run" entity with fields start, length, and
        multiplicity (product of value counts: 2 = double run). The
        fields are count and span, not size and length — those are EL
        keywords.

    suffixes(src, 2, "rank", "combo", dest)
        Every trailing window of src with length >= minlen, LONGEST
        FIRST, as a "combo" entity with fields members (in source
        order), count, sum, distinct, and spread (max - min) over the
        named field. For order-dependent structure: a window is a run
        iff distinct == count and spread == count - 1 (any lay order),
        and a trailing pair block iff distinct == 1. The longest-first
        order is part of the contract — "longest run only" policies
        read as "the first qualifying window" behind a zero-guard.
        Source cap: 64.

Example — cribbage fifteens, pairs, and runs from decision tables:

    subsets(hand.cards, "combo", "value", hand.combos);
    groupby(hand.cards, "rank", "group", hand.rank_groups);
    maximalruns(hand.cards, "rank", 3, "run", hand.runs)

then score with conditions like 'combo.sum == 15' (add 2),
'group.count == 2/3/4' (add 2/6/12), and actions adding
'run.span * run.multiplicity'.


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


Fixed-Point Operators
---------------------
Fixed (` + "`fixed`" + `) is a signed 256-bit decimal type on a 10^-8 grid.
Add/subtract are exact; multiply/divide truncate toward zero onto the
grid. Mixed int/bigint operands auto-promote; double requires an
explicit ` + "`(fixed)`" + ` cast.

Arithmetic:
    a + b                           addition (exact) → fp
    a - b                           subtraction (exact) → fp
    a * b                           multiplication (truncate toward zero) → fp
    a / b                           division (truncate toward zero) → fp
    -a                              unary negation → fp
    absolute value of a             absolute value → fp

Comparisons (exact mantissa compare):
    a == b   a is equal to b                       → boolean
    a != b   a is not equal to b                   → boolean
    a > b    a is greater than b                   → boolean
    a >= b   a is greater than or equal to b       → boolean
    a < b    a is less than b                      → boolean
    a <= b   a is less than or equal to b          → boolean

Fixed-specific operators:
    fpmin a b                       minimum of two fp values → fp
    fpmax a b                       maximum of two fp values → fp
    fptrunc a                       truncate fractional part toward zero → fp
    fpabs a                         absolute value → fp
    fpnegate a                      unary negation → fp

Mutating shortcuts (action statements):
    increment myFixed               myFixed += 1.00000000
    decrement myFixed               myFixed -= 1.00000000
    add to myFixed 1.5fp
    subtract from myFixed 0.25fp
    multiply myFixed by 2fp
    divide myFixed by 3fp

Cast to fixed:
    (fixed) 42                      from integer (exact)
    (fixed) myBigInt                from bigint (exact, range-checked)
    (fixed) 3.14                    from double (truncate, explicit only)
    (fixed) "1.25"                  parse decimal string

Cast fixed to other types:
    (long)   myFixed                fp → integer (truncate toward zero)
    (double) myFixed                fp → double (may lose precision)
    (bigint) myFixed                fp → bigint (truncate toward zero)
    (string) myFixed                fp → string, always 8 fractional digits

Internal operator reference (emitted by the EL compiler):

    fp+, fpadd          Add two fp values (exact)
    fp-, fpsub          Subtract two fp values (exact)
    fp*, fpmul          Multiply two fp values (truncate toward zero)
    fp/, fpdiv          Divide two fp values (truncate toward zero)
    fphalfup/,
      fpdivhalfup       Divide with round-half-away-from-zero at the
                        mantissa grid (v1.14.7)
    fpdivr/, fpdivround Ternary divide with configurable rounding
                        fraction r in [0, 1) — r=0 truncates,
                        r=0.5 half-up, r→1 ceiling (v1.14.8)
    fpabs               Absolute value
    fpnegate            Unary negation
    fptrunc             Truncate fractional part toward zero
    fpmin               Minimum of two fp values
    fpmax               Maximum of two fp values
    fp==                Equality
    fp!=, fp<>          Inequality
    fp>                 Greater than
    fp>=                Greater than or equal
    fp<                 Less than
    fp<=                Less than or equal
    cvfp                Cast top-of-stack to fp (int/bigint/double/string)

See 'dtrules docs fixed' for the full fixed-point documentation.


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

Timezone-aware variants (#743): every date op above has an ` + "`in zone <tz>`" + `
counterpart that interprets the date in the given IANA timezone (e.g.
"America/Chicago", "UTC"). The runtime op names suffix ` + "`inzone`" + `.

    current date in zone "America/Chicago"      today in Chicago
    today in zone "UTC"                          today in UTC
    new date "2024-03-10" in zone "America/Chicago"
    new date "2024-03-10 02:30" in zone "America/Chicago" with dst rule "fall back"
    "2024-01-15" in zone "UTC"                   parse ISO string in tz
    format dexpr "Mon Jan 2" in zone "UTC"       format in tz
    get year of d in zone "UTC"                  year in tz
    get month of d in zone "UTC"                 month in tz
    get day of month of d in zone "UTC"
    get day of week of d in zone "UTC"
    get week of year of d in zone "UTC"
    get hour of d in zone "UTC"
    get minute of d in zone "UTC"
    get second of d in zone "UTC"
    get days in month of d in zone "UTC"
    get days in year of d in zone "UTC"
    first of years of d in zone "UTC"            Jan 1 in tz
    first of months of d in zone "UTC"
    first of quarters of d in zone "UTC"
    first of weeks of d in zone "UTC"
    first of weeks starting "Sunday" of d in zone "UTC"
    end of years of d in zone "UTC"
    end of months of d in zone "UTC"
    end of quarters of d in zone "UTC"
    end of weeks of d in zone "UTC"
    end of weeks starting "Sunday" of d in zone "UTC"
    d1 same calendar day as d2 in zone "UTC"
    d1 same calendar month as d2 in zone "UTC"
    d1 same calendar quarter as d2 in zone "UTC"
    d1 same calendar week as d2 in zone "UTC"
    d1 same calendar week starting "Sunday" as d2 in zone "UTC"
    d1 same calendar year as d2 in zone "UTC"

Runtime op names (emitted by the compiler):
    currentdateinzone, todayinzone, newdateinzone,
    newdateinzonewithdst, dateinzone, dateformatinzone,
    getyearinzone, getmonthinzone, getdayofmonthinzone,
    getdayofweekinzone, getweekofyearinzone, gethourinzone,
    getminuteinzone, getsecondinzone, getdaysinmonthinzone,
    getdaysinyearinzone, firstofmonthinzone, firstofquarterinzone,
    firstofyearinzone, firstofweekinzone, firstofweekstartinginzone,
    endofmonthinzone, endofquarterinzone, endofyearinzone,
    endofweekinzone, endofweekstartinginzone, samecalendardayinzone,
    samecalendarmonthinzone, samecalendarquarterinzone,
    samecalendarweekinzone, samecalendarweekstartinginzone,
    samecalendaryearinzone.


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

FIRST PASS predicate (#764):
    first pass                              true on the first iteration of
                                            the innermost active loop in
                                            this table's context; false on
                                            subsequent iterations; false
                                            when no loop is active.
                                            Runtime op: firstpass

CREATE typed entity in action context (#712 area):
    create state_tax_result as result       allocate a fresh entity and bind
                                            to a local name; subsequent
                                            ` + "`set result.field = ...`" + ` and
                                            ` + "`add result to <array>`" + ` target it.
                                            Lowers to ` + "`/state_tax_result createentity /result xdef`" + `

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


Reverse Index: Postfix Op -> EL Phrase
--------------------------------------
When reading stored postfix (traces, legacy tables, the debug console) and
working backward to the EL that produces it, use this table. Every op here
HAS an EL surface form — do not conclude a form is missing without checking
this list and 'dtrules docs el'.

Postfix op        EL phrase that emits it
----------------  ------------------------------------------------------------
createentity      new <type> entity            (eexpr; also: create <type> as x)
entitypush        add <entity> to context of this table
memberof          <value> is one of <array>    (negated: is not one of)
addto             add <value> to <array>       (emits: <value> <array> swap addto)
addarray          add <array> to <array>       (element-wise; also what
                  "add the policy statements to X" emits)
policystatements  the policy statements        (the run's report; see
                  dtrules docs decision-tables)
xdef              set <entity.field> = <expr>  (emits: <expr> /<field> xdef)
performtable      perform <TableName>          (runs the table's contexts too)
executetable      execute <TableName>          (skips contexts; use inside own context)
forall            for all <collection> [where <bexpr>]        (context row)
                  also: sum of <field> in <array> [where ...]
                  also: number of <array> where <bexpr>
fmin / fmax       the minimum/maximum of <a> and <b>
ceiling / floor   the ceiling/floor of <fexpr>
fabs / abs        absolute value of <expr>
streq             <s> is equal to "..."        (case-sensitive string compare)
isnull            <field> is null              (negated: is not null)
length            length of <array>
if / ifelse       if <bexpr> then { ... } [else { ... }] endif
cvb/cvi/cvd/cvs   implicit conversions inserted by the compiler (bool/int/
                  double/string) — never authored directly

Operand-order convention (matters when hand-reading postfix):
  'if' and 'ifelse' pop the TEST from the TOP of the data stack. Compiled
  form is '{ then } { else } <bexpr> ifelse'. Legacy hand postfix that put
  the test FIRST ('<bexpr> [then] [else] ifelse') fails at runtime with a
  BooleanValue conversion error — recompile the row from its DSL.

Never hand-write postfix. If a stored-postfix idiom seems to have no EL
equivalent, recheck the phrases above, then 'dtrules docs el' — and only
then file a grammar-gap issue quoting both the postfix and the phrases you
ruled out.

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


Go Code (preview — pkg/dtrules/sdk is being extracted, issue #757):
-------------------------------------------------------------------
import "github.com/DTRules/DTRules/pkg/dtrules/sdk/engine"

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

Until the SDK lands, the supported embedding path is to construct
the engine directly from cmd/dtrules (see cli.go) — both binaries
in this repo (cmd/dtrules, cmd/api) follow that pattern.
`

const docWorkflow = `DTRules Development Workflow
============================

CRITICAL: Excel is the System of Record
----------------------------------------
All rules MUST be written in EL (Expression Language). The EL compiler
generates internal bytecode automatically — never write bytecode by hand.

Since v1.14.0: the runtime loader does NOT compile DSL on load. It
consumes whatever postfix is stored in the XML. So 'dtrules build' (the
human path) or the authoring API ('dtrules table'/'dtrules edd', which
compile on write) MUST run between authoring a DSL change and embedding
the XML — otherwise the loader sees DSL with no postfix and refuses.

Since v1.14.1: every authoring/build surface surfaces advisory warnings
(decisiontable.Analyze) — no-op columns, subsumed columns, FIRST-policy
redundant conditions (#762), assignment-only tables (#763), unreachable
columns via DSL negation. The full warning catalogue with repros and
recommended actions lives at 'dtrules docs warnings'.


Two ways to change a rule
-------------------------

  dtrules build [path]              (the human path)
    Edit Excel, then build. Full Excel → XML extraction plus EL
    compile. Requires the canonical project layout (<project>/xml/
    and <project>/excel/). Excel is the input; XML is generated.

  dtrules table / dtrules edd       (the programmatic path, for AI agents)
    A write through the authoring API updates the XML DSL, compiles
    postfix, AND updates Excel in the same atomic operation. If the
    project has no Excel yet, the API bootstraps it from the XML.
    See 'dtrules docs authoring'.

Both paths keep Excel as the system of record. There is no command that
writes rule content into XML alone — the bypass writers ('dtrules compile',
'dtrules build --from-xml') were removed in v1.16.0.

dtrules build flags:
  --from-excel   Force Excel-authored path (Excel → XML)
  --dry-run      Show what would change without writing files
  -v, --verbose  Verbose output
  -q, --quiet    Suppress build summary unless there are drops


Migrating from v1.12.0 / v1.13.0 to v1.14.x
--------------------------------------------

v1.14.0 made the loader strict — no more silent recompile of DSL at
load time. The 'loader: context N (...) — recompiled postfix differs
from stored; using fresh compile' log lines are gone, but so is the
safety net for stale XML.

One-time migration per project:

  go get github.com/DTRules/DTRules@latest
  dtrules build <project>                # re-extract from Excel + compile postfix
  git diff                               # review the change
  git commit -am 'adopt v1.14.x — backfill postfix'

After this, the loader is silent on load (no more recompile chatter),
the runtime binary no longer pulls in compiler/el as a dependency,
and 'dtrules review' is the canonical "did I get the warnings right?"
check.


Build Summary
-------------
After every build, dtrules prints a structured summary proving round-trip
preservation. The summary shows artifact counts and any drops:

  Build Summary
  =============

  Import step (Excel → XML):
    tables=3  actions=12  conditions=8  entities=5  mappings=0
    compiled=20
    files-written=4
    drops: none

  Result: OK — no drops

If any EL expression fails to compile, the drop is named precisely:

  drops: 1
    table="CO_IncomeTax"  col=0  item="action 3"  reason=<compile error>

  Result: FAIL — 1 drop(s) detected

The build exits non-zero when drops are present so CI pipelines fail fast.
Use -q (--quiet) to suppress the summary on clean builds.


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
  dtrules docs fixed       Fixed-point decimal type
`

const docFixed = `Fixed - 256-bit Fixed-Point Decimals
====================================

Fixed (type keyword ` + "`fixed`" + `) is a signed 256-bit fixed-point numeric type
backed by an integer mantissa on the 10^-8 grid. It exists for
token amounts, staking rewards, fee/rate math, and any blockchain-adjacent
calculation where float64 drift is unacceptable and bigint (integer-only)
is insufficient.

Precision: exactly 8 fractional digits (8-decimal fixed-point). The
internal representation is an integer mantissa M with the implicit
meaning M × 10^-8.

Mantissa range: |M| < 2^255 — symmetric about zero so that negating any
representable value is always representable.


When to Use
-----------
- Token amounts with sub-integer precision (e.g. 1.50000000 ACME).
- Staking reward math where each intermediate must stay on the
  blockchain's fixed grid.
- Rates, fees, percentages that need predictable decimal rounding.
- Anywhere a double would silently snap (e.g. "0.1" stored as
  0.09999999...) and corrupt an invariant.

Use double (` + "`fexpr`" + `) for statistics, approximate scoring, or anything
where rounding to 8 places is acceptable. Use bigint for exact integer
arithmetic without a fractional part.


Literal Syntax
--------------
A fp literal is a decimal number followed by a case-insensitive ` + "`fp`" + `
suffix:

    1.5fp                   mantissa 150_000_000
    0fp                     zero, mantissa 0
    100.0FP                 mantissa 10_000_000_000
    -0.00000001fp           smallest negative value (mantissa -1)
    12345.67890000fp        exact — trailing zeros preserved

Rules:
- The decimal point is optional (` + "`0fp`" + ` and ` + "`42fp`" + ` are valid).
- At most 8 fractional digits — more is a compile-time error (no silent
  truncation of a user-written literal).
- The ` + "`fp`" + ` suffix is case-insensitive (` + "`FP`" + `, ` + "`Fp`" + `, ` + "`fP`" + ` all work).
- No underscores or thousands separators.


Local Variable Declarations
---------------------------
Declare a fp local with ` + "`local fixed`" + `:

    local fixed x                    uninitialized (mantissa 0)
    local fixed rate = 0.05fp
    local fixed principal = (fixed) input.amount_string
    local fixed earned = (fixed) 42

Reads and writes go through the same slot machinery as other typed
locals.


Casts
-----
Explicit cast with ` + "`(fixed)`" + ` — the value on top of the stack becomes fp:

    (fixed) 42                  integer to fp (exact, auto-scaled)
    (fixed) myBigInt            bigint to fp (exact, range-checked)
    (fixed) "1.25"              parse decimal string to fp
    (fixed) 3.14                DOUBLE to fp — explicit cast required

Double -> fixed REQUIRES the explicit ` + "`(fixed)`" + ` cast. There is no implicit
promotion because most finite decimals (0.1, 0.2, 0.05, ...) have no exact
float64 representation — silently snapping them to the fp grid would bake
float error into a "precise" calculation. The cast truncates toward zero
onto the 10^-8 grid and range-checks the mantissa.

Integer -> fixed and bigint -> fixed DO auto-promote in mixed arithmetic,
because both conversions are exact.

Reverse casts:

    (long) myFixed              fp to integer (truncate toward zero)
    (double) myFixed            fp to double (may lose precision)
    (bigint) myFixed            fp to bigint (truncate toward zero)
    (string) myFixed            fp to string (always 8 fractional digits)


Arithmetic Operators
--------------------
Standard arithmetic works on fp operands (mixed int/bigint are promoted
via ` + "`cvfp`" + `):

    a + b                       addition (exact)
    a - b                       subtraction (exact)
    a * b                       multiplication (truncate toward zero)
    a / b                       division (truncate toward zero)
    -a                          unary negation
    absolute value of a         |a|

Add and subtract cannot lose precision — mantissas are added/subtracted
directly and only overflow is checked. Multiply and divide must rescale
by 10^8; when the exact result has a ninth decimal digit, it is
truncated toward zero (never rounded away from zero, never banker's-
rounded). That rule is deliberate — it matches how on-chain fixed-point
math typically settles.

Mutating variants in action cells:

    add to myFixed 1.5fp
    subtract from myFixed 0.25fp
    multiply myFixed by 2fp
    divide myFixed by 3fp
    increment myFixed           adds 1.00000000
    decrement myFixed           subtracts 1.00000000

Extra fp-specific operators:

    fpabs                       absolute value
    fpnegate                    unary negation
    fptrunc                     truncate fractional part toward zero
                                (result is fp with .00000000)
    fpmin                       minimum of two fp values
    fpmax                       maximum of two fp values

All fp operators are registered with both symbolic (` + "`fp+`" + `) and word-
form (` + "`fpadd`" + `) names — use whichever reads better in context.


Comparison Operators
--------------------
Standard comparisons between fp operands:

    a == b      fp equality
    a != b      fp inequality
    a > b
    a >= b
    a < b
    a <= b

Equality is exact mantissa comparison, so ` + "`0.10000000fp == 0.1fp`" + ` is
true. Unlike double, there is no need for an epsilon.


Conversion Helpers
------------------
The ` + "`cvfp`" + ` operator is the underlying cast operator — the EL
compiler emits it whenever a ` + "`(fixed)`" + ` cast or auto-promotion is needed.
Rarely needed at the EL level, but listed here for completeness:

    cvfp                        top-of-stack value -> fp
                                - fixed  : identity
                                - int    : exact
                                - bigint : exact (range-checked)
                                - double : truncate toward zero on grid
                                - string : parse decimal literal


Interaction with Other Numeric Types
------------------------------------
Promotion rules in mixed arithmetic (fp + int, fp + bigint, etc.):

    fixed + integer       -> fixed  (integer auto-promoted)
    fixed + bigint        -> fixed  (bigint auto-promoted, range-checked)
    fixed + double        -> ERROR: explicit (fixed) cast required

When a ` + "`set`" + ` target is a fp field and the right-hand side is int or
bigint, the compiler inserts ` + "`cvfp`" + ` automatically so the assignment is
exact. When the RHS is double, the compiler rejects the assignment and
asks for an explicit cast.


Overflow and Error Semantics
----------------------------
- Add/subtract: overflow when the resulting mantissa crosses 2^255.
  Returns a Math Exception; the value is NOT silently wrapped.
- Multiply: the 256-bit product is scaled down by 10^8 with truncate-
  toward-zero; if the final mantissa exceeds 2^255, overflow error.
- Divide: division by zero raises a Math Exception. The default ` + "`a / b`" + `
  truncates toward zero. For configurable rounding, use the EL surface
  ` + "`divide a by b rounding by R`" + ` (v1.14.8, #801) — R is a literal
  fp constant in [0, 1) and authors must write it explicitly at each
  call site. The compiler folds R=0 to ` + "`fp/`" + `, R=0.5 to
  ` + "`fphalfup/`" + `, and any other in-range R to the ternary
  ` + "`fpdivr/`" + `. R outside [0, 1) is a compile-time error.
- Cast: a bigint or string whose value exceeds the 256-bit range raises
  a Math Exception at cast time.
- Cast from double: NaN or Inf raises a Math Exception.

Rendering: ` + "`(string) myFixed`" + ` always emits the sign, the integer
part, a decimal point, and exactly eight fractional digits. That means
` + "`0.5fp`" + ` round-trips to ` + "`\"0.50000000\"`" + ` — bit-exact for XML / JSON
storage.


EDD Field Declaration
---------------------
<entity name="staking" readonly="false">
    <field name="reward_rate"   type="fixed" default_value="0.05fp"/>
    <field name="principal"     type="fixed" default_value="0fp"/>
    <field name="total_reward"  type="fixed" default_value="0fp"/>
</entity>

Default values use the same literal syntax as EL source. The loader
parses them once at startup; a malformed default is a load-time error.


JSON / XML Round-Trip
---------------------
Fixed values are represented as decimal strings in JSON (never as numbers —
native JSON numbers go through float64 and would lose precision):

{
    "staking": {
        "reward_rate":  "0.05000000",
        "principal":    "100.00000000",
        "total_reward": "5.00000000"
    }
}

The runtime parses strings directly into the mantissa without any float
intermediate step. Output always renders 8 fractional digits.


Example: Staking Reward Step
----------------------------
Context:
    local fixed rate       = 0.05fp
    local fixed principal  = (fixed) input.amount_string

Action:
    set result.reward = rate * principal

For principal = 1000.00000000 and rate = 0.05000000, the result is
50.00000000 — exact, grid-aligned, and bit-identical whether you
compute it in EL, in Go, or on-chain.


Best Practices
--------------
1. Use ` + "`fixed`" + ` for anything where "decimal precision" is a
   correctness requirement, not just presentation.
2. Store fp values as decimal strings in JSON and databases. Do NOT
   serialize as a native number.
3. Prefer ` + "`fpmin`" + ` / ` + "`fpmax`" + ` over manual ` + "`if`" + ` chains — the operators
   are typed and won't accidentally fall back to integer comparison.
4. If you must bring in a double (e.g. from legacy input), make the
   ` + "`(fixed)`" + ` cast explicit and document the precision loss at that
   boundary.
5. Avoid chaining divides when you can factor them: ` + "`a * b / c`" + `
   truncates once; ` + "`(a / c) * b`" + ` truncates twice.


See Also
--------
  dtrules docs el          EL syntax reference
  dtrules docs edd         Entity field definitions (fixed is a valid type)
  dtrules docs operators   All operators including the fp* family
  dtrules docs bigint      Arbitrary-precision integers
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

const docAuthoring = `Authoring SDK
=============

Overview
--------

The authoring SDK is a typed Go package at
github.com/DTRules/DTRules/pkg/dtrules/authoring
for opening, editing, executing, and testing DTRules projects. It is the
recommended way to interact with rules programmatically. Editing raw XML
is discouraged: the round-trip serialiser may reformat or reorder
elements, and the authoring API validates every EL expression at the API
boundary before any file is touched.

Reach for this SDK when you want to script bulk edits, run regression
scenarios, measure coverage, or drive the debugger from Go. If you are
coming from another language, use the JSON CLI or MCP server described
in "Programmatic access from outside Go" below — they expose the same
surface without requiring Go.

Import:

  import "github.com/DTRules/DTRules/pkg/dtrules/authoring"


Worked Example
--------------

This example opens a project, mutates a table, loads test data, and runs
a scenario end to end. Every mutation validates EL before writing.

  package main

  import (
      "fmt"
      "log"

      "github.com/DTRules/DTRules/pkg/dtrules/authoring"
  )

  func main() {
      // Open a project directory (must contain xml/ with *_dt.xml, *_edd.xml).
      p, err := authoring.OpenProject("/path/to/MyProject")
      if err != nil {
          log.Fatal(err)
      }

      // Inspect a table.
      tbl := p.Table("EligibilityCheck")
      if tbl == nil {
          log.Fatal("table not found")
      }

      // Mutate an action — EL is validated before the change is committed.
      err = tbl.UpdateAction(1, authoring.Action{
          Comment: "set approved flag",
          DSL:     "applicant.approved = true",
          Columns: map[int]bool{1: true, 2: false},
      })
      if err != nil {
          log.Fatal("invalid EL:", err)
      }

      // Option A — set state directly and execute.
      p.ResetState()
      p.SetAttribute("applicant", "age", 30)
      p.SetAttribute("applicant", "income", 55000)

      trace, err := p.ExecuteEntry("EligibilityCheck")
      if err != nil {
          log.Fatal(err)
      }
      fmt.Println("approved:", trace.FinalState["applicant.approved"])

      // Option B — load test data from the project's _map.xml + XML file.
      if err := p.LoadTestData("testfiles/TestScenarios/basic.xml"); err != nil {
          log.Fatal(err)
      }
      _, _ = p.ExecuteEntry("EligibilityCheck")

      // Option C — run every scenario JSON under a directory and aggregate.
      batch, err := p.RunAllScenarios("testfiles/scenarios")
      if err != nil {
          log.Fatal(err)
      }
      fmt.Println(batch.Summary())

      // Persist the edits back to disk.
      if err := p.Save(); err != nil {
          log.Fatal(err)
      }
  }


Opening and Saving
------------------

  OpenProject(path string) (*Project, error)
      Load a project from a directory containing an xml/ subdirectory.
      Reads every *_dt.xml and lazily prepares the EDD and Mapping.

  (*Project).Save() error
      Write all in-memory decision-table mutations back to disk as
      canonical XML. The EDD is saved separately via SaveEDD.

  (*Project).SaveEDD() error
      Persist EDD mutations back to *_edd.xml.

  (*Project).Tables() []string
      Names of every loaded decision table.

  (*Project).Table(name string) *Table
      Typed view of a table, or nil if not found.

  (*Project).AddTable(name, file, reason string) (*Table, error)
      Create a new empty decision table in an existing file (auto-numbered
      within the file's range). A file is required.

  (*Project).DeleteTable(name, reason string) error
      Remove a decision table by name (reason recorded; empty files auto-deleted).

  (*Project).CreateFile(file string, lo, hi int, reason string) error
      Register a new DT file with a non-overlapping number range.

  (*Project).MoveTable(name, target, reason string) error
      Move a table to another file, renumbering into the target's range.

  (*Project).SetFileRange(file string, lo, hi int, reason string) error
      Change a file's number range (rejects overlap / shrink-below-used).

  (*Project).Files() []FileInfo / (*Project).AppendNote(text string)
      File map (ranges/purposes/tables) and a dated authoring-notes entry.


Editing Tables
--------------

Every table mutation compiles its EL text to postfix first (via
CheckCondition / CheckAction / CheckContext). A bad expression returns
an error and the table is not touched — callers can safely retry.

  Conditions:
    (*Table).AddCondition(c Condition) error
    (*Table).UpdateCondition(num int, c Condition) error
    (*Table).DeleteCondition(num int) error

  Actions (numbered, appear in columns):
    (*Table).AddAction(a Action) error
    (*Table).UpdateAction(num int, a Action) error
    (*Table).DeleteAction(num int) error

  Initial actions (run once before the column grid):
    (*Table).AddInitialAction(a InitialAction) error
    (*Table).UpdateInitialAction(idx int, a InitialAction) error
    (*Table).DeleteInitialAction(idx int) error

  Context statements:
    (*Table).AddContext(c Context) error
    (*Table).UpdateContext(idx int, c Context) error
    (*Table).DeleteContext(idx int) error

  Columns (rule cells — condition cell is "Y"/"N"/" ", actions fire by number):
    (*Table).AddColumn(conditions map[int]string, actions []int) error
    (*Table).UpdateColumn(col int, conditions map[int]string, actions []int) error
    (*Table).DeleteColumn(col int) error
    (*Table).Columns() int

Typical edit — add a new rule to an existing table:

  tbl := p.Table("ApproveLoan")
  err := tbl.AddColumn(
      map[int]string{1: "Y", 2: "N"}, // c1=Y, c2=N, others blank
      []int{3},                        // fire action 3
  )
  if err != nil {
      return fmt.Errorf("add column: %w", err)
  }


Editing the EDD
---------------

  (*Project).EDD() *EDD
      Lazy view of the entity data dictionary.

  (*EDD).Entities() []*Entity
  (*EDD).Entity(name string) *Entity
  (*EDD).AddEntity(name string) (*Entity, error)
  (*EDD).DeleteEntity(name string) error
      Checked deletion: rejected if any DT references the entity.

  (*Entity).AddAttribute(a Attribute) error
  (*Entity).UpdateAttribute(name string, a Attribute) error
  (*Entity).DeleteAttribute(name string) error

The Attribute struct carries all field metadata:

  type Attribute struct {
      Name, Type, Subtype, Default, Access, Input, Comment string
  }

After editing, call (*Project).SaveEDD() to persist.


Editing the Mapping
-------------------

  (*Project).Mapping() (*Mapping, error)
      Load the first *_map.xml in the project. Returns an error if no
      map file exists.

  (*Mapping).Entries() []SetAttribute
  (*Mapping).AddEntry(e SetAttribute) error
  (*Mapping).UpdateEntry(tag string, e SetAttribute) error
  (*Mapping).DeleteEntry(tag string) error

Mapping entries connect XML-encoded test data to EDD attributes:

  type SetAttribute struct {
      Tag, RAttribute, Enclosure, Type string
  }


Executing Rules
---------------

Set state, then run. State is retained across calls until you reset it.

  (*Project).SetAttribute(entityName, attribute string, value any) error
      Value may be bool, int, int64, float64, or string.
  (*Project).ResetState()
      Clear entity state; EDD/DT stay loaded.
  (*Project).EntityStackNames() []string
      Names of writable entities currently on the stack.

  (*Project).EvalCondition(el string) (bool, error)
      Compile and evaluate an EL boolean expression against current state.
  (*Project).EvalAction(el string) ([]AttributeChange, error)
      Compile and evaluate an EL action; returns the diff vs prior state.

  (*Table).Execute(p *Project) (*ExecutionTrace, error)
      Run a single table against current state and return a step-level trace.
  (*Table).NewStepper(p *Project) *Stepper
      Cursor-style execution: call Next() to advance, Current() to inspect.

  (*Project).ExecuteEntry(tableName string) (*RunTrace, error)
      Run the entry table and every descendant it calls, returning a
      flattened pre-order list of TableInvocations with before/after
      entity snapshots. This is the trace used by coverage and debug.


Test Data
---------

  (*Project).LoadTestData(path string) error
      Populate entity state from an XML test-data file via the project's
      _map.xml. The mapping defines how XML tags become entity fields.
  (*Project).LoadTestDataReader(mapReader, dataReader io.Reader) error
      Same, but from in-memory io.Reader pairs (useful in tests).


Batch testing
-------------

Scenarios are JSON files with inputs, an entry table, and expected
final state. Use them for regression suites.

  type ScenarioFile struct {
      Name       string
      EntryTable string
      Inputs     map[string]any    // "entity.attribute" -> value
      Expected   map[string]any
  }

  (*Project).RunAllScenarios(dir string) (*BatchResult, error)
      Walk dir non-recursively, load every *.json as a ScenarioFile, run
      each against p, and return an aggregate BatchResult.

  (*BatchResult).AllPassed() bool
  (*BatchResult).Summary() string

  (*Scenario).Run(p *Project) *ScenarioResult
      Run a programmatically constructed Scenario. ResetState is called
      first so runs are isolated.

  AssertState(actual, expected map[string]any) []AssertionFailure
      Compare two state maps with normalising string/number/bool equality.


Coverage
--------

  Cover(p *Project, results []ScenarioResult) *CoverageReport
      Walks the RunTrace of every result and records which tables and
      columns were exercised.

  type CoverageReport struct {
      ExercisedTables  map[string]bool
      ExercisedColumns map[string]map[int]int
      UntouchedTables  []string
      UntouchedColumns map[string][]int
  }
  (*CoverageReport).Summary() string


Diff
----

Run a scenario suite against two project versions and report the
attributes whose final values diverge — useful for pre/post refactor
regression checks.

  Diff(p1, p2 *Project, scenarios []*Scenario) *DiffReport
  (*DiffReport).Summary() string

  type DiffReport struct {
      Total    int
      Matching int
      Diverged []ScenarioDivergence
  }


Dependency Graph
----------------

  (*Table).Dependencies() []string
      Names of tables this table invokes (statically scanned from every
      condition, action, initial-action and context DSL text).
  (*Table).Callers() []string
      Names of tables that invoke this one.


Debug Stepping
--------------

ExecuteEntry records a RunTrace of every invocation. ResumeAt replays
that trace up to a chosen invocation and hands you a live DebugSession
paused there.

  (*Project).ResumeAt(trace *RunTrace, idx int) (*DebugSession, error)

  (*DebugSession).EntityStack() []EntityView
  (*DebugSession).Resolve(name string) (any, string, error)
      Look up "entity.attribute" or bare attribute against current stack.
  (*DebugSession).NextInvocation() *TableInvocation
      Peek at the next invocation without advancing.
  (*DebugSession).Step() (*TableInvocation, error)
      Advance one invocation, return the one just executed.
  (*DebugSession).Continue() (*RunTrace, error)
      Run to completion and return the final trace.
  (*DebugSession).SetAttribute(entityName, attribute string, value any) error
      Rewrite state mid-session, e.g. to explore a what-if branch.
  (*DebugSession).Close()


EL Validation
-------------

Pre-check an EL expression without mutating anything:

  CheckCondition(el string, symbols map[string]string) (postfix string, err error)
  CheckAction(el string, symbols map[string]string)    (postfix string, err error)
  CheckContext(el string, symbols map[string]string)   (postfix string, err error)

All three compile EL to postfix using the same pipeline as
Add/UpdateCondition etc. Call them directly to validate user-supplied
EL before queueing a mutation.


Trace Assertions
----------------

For scenario tests you usually want to assert on the shape of execution
rather than just the final state.

  (*RunTrace).AssertVisited(table string, column int) error
      Pass column=0 to match any column.
  (*RunTrace).AssertNotVisited(table string) error
  (*RunTrace).AssertSequence(tables []string) error
      Tables must appear in the given order (non-contiguous is fine).


Anti-Patterns
-------------

Do NOT edit *_dt.xml or *_edd.xml files with a text editor or XML library
directly. The authoring API:
  - validates EL before any file is written
  - maintains consistent attribute ordering
  - preserves sync-manifest state

Direct XML edits bypass all of these guarantees and may corrupt round-trip
semantics enforced by "dtrules build".


Programmatic access from outside Go
-----------------------------------

The authoring SDK is Go-only, but v1.9.1 ships two wrappers that expose
the same table/EDD read/write surface over stable interfaces so agents
and tools in any language can drive DTRules.

1. JSON CLI — "dtrules table" and "dtrules edd"

   Every subcommand takes --project <path> and reads or writes one
   table or the EDD as a JSON document. Schemas are stable and
   published via the schema subcommand.

     dtrules table list     --project P
     dtrules table get      --project P --name T
     dtrules table put      --project P --name T < table.json
     dtrules table patch    --project P --name T --op set-condition-cell ...
     dtrules table schema
     dtrules edd  get       --project P
     dtrules edd  put       --project P < edd.json
     dtrules edd  patch     --project P --op add-entity ...
     dtrules edd  schema

   Errors surface as a structured {error, hint, detail} payload on
   stderr and a non-zero exit code. See "dtrules docs cli" for the
   full reference.

2. MCP server — "dtrules mcp"

   A stdio JSON-RPC 2.0 server (MCP protocol version 2024-11-05) that
   exposes the same operations as MCP tools. Wire it into Claude Code
   or any other MCP client with:

     dtrules mcp --project /path/to/MyProject

   The ten tools exposed match the JSON CLI one-for-one:

     Reads:  table_list, table_get, table_schema,
             edd_get, edd_schema, project_validate
     Writes: table_put, table_patch, edd_put, edd_patch

   Every tool call is stateless: the server opens the project, applies
   the op, saves, and discards the project handle. Concurrent clients
   and external XML edits are safe by construction. Tool errors carry
   the same {error, hint, detail} payload as the CLI.


Full Reference
--------------

  go doc github.com/DTRules/DTRules/pkg/dtrules/authoring


See Also
--------
  dtrules docs cli             JSON CLI reference (table / edd subcommands)
  dtrules docs el              EL expression syntax
  dtrules docs decision-tables Decision table structure
  dtrules docs edd             Entity Data Dictionary reference
  dtrules docs mapping         Mapping file reference
  dtrules docs embedding       Deploy rules in a single Go binary
`

const docCLI = `DTRules CLI — Getting Started with the Binary
==============================================

The ` + "`dtrules`" + ` binary is a single self-contained CLI covering the full
authoring-to-deployment workflow. This topic walks through every
subcommand in the order a typical user hits them.


Install
-------

Option 1 — go install (requires Go 1.21+):

    go install github.com/DTRules/DTRules/cmd/dtrules@latest

Verify:

    dtrules version

Option 2 — build from source (useful when hacking on the binary):

    git clone https://github.com/DTRules/DTRules.git
    cd DTRules
    make build        # binary lands at ./build/dtrules
    make install      # installs to $GOPATH/bin


Top-level command map
---------------------

    dtrules init       Scaffold a new project directory
    dtrules build      Extract DSL from Excel + compile postfix (the human path)
    dtrules run        Run a decision table; --interactive collects missing inputs;
                       --trace records a debugger-ready execution trace
    dtrules debug      Run + trace + open the editor's trace debugger (one command)
    dtrules report     Generate an EDD-driven report from a trace (see docs debug)
    dtrules edit       Serve the visual editor/debugger in the browser
    dtrules table      JSON-first per-table read/write (the programmatic path)
    dtrules edd        JSON-first EDD read/write (the programmatic path)
    dtrules sync       Fine-grained Excel/XML sync (status/check/import/export/auto)
    dtrules validate   Check project structure + EL compliance
    dtrules verify     CI gate: Excel↔XML consistency + self-contained references
    dtrules review     Project-wide Full Review (errors + advisory warnings;
                       deployment gate when used with 'build --require-review')
    dtrules mcp        MCP server over stdio (for AI agents)
    dtrules docs       This documentation
    dtrules version    Version, commit, build date

Run any command with no arguments to see usage for that command.


1. Start a new project — dtrules init
--------------------------------------

    mkdir MyRules && cd MyRules
    dtrules init

Creates:

    MyRules/
    ├── excel/        # your Excel authoring sources (system of record)
    ├── xml/          # compiled XML artifacts (generated from Excel)
    └── testfiles/    # scenario JSONs for tests

An empty project is ready. You can now open 'excel/' in Excel and add
a ` + "`<name>.xlsx`" + ` for your decision tables and a ` + "`<name>_map.xlsx`" + `
for input mapping, following the conventions in:

    dtrules docs project-layout
    dtrules docs decision-tables
    dtrules docs edd


2. Author rules — two paths (Excel is the system of record)
-----------------------------------------------------------

  >> Read 'dtrules docs authoring-contract' before changing any rule. <<

There are exactly two ways to change a rule, and BOTH keep Excel current:

  (a) Edit Excel, then 'dtrules build' (the human path) — analysts edit
      .xlsx; build extracts the DSL to .xml and compiles postfix. Excel
      is the input; XML is generated.

  (b) The authoring API — 'dtrules table' / 'dtrules edd' (the
      programmatic path, for AI agents and tools). Each write updates the
      XML DSL, compiles postfix, AND updates Excel in one atomic
      operation. If the project has no Excel yet, the API bootstraps it.
      See section 8 and 'dtrules docs authoring'.

Write conditions and actions in EL (Expression Language):

    dtrules docs el

NEVER hand-edit XML, and never hand-author postfix or bytecode — XML is a
generated artifact and postfix is a compiled one. An agent that writes XML
directly produces a state the next build overwrites and that 'dtrules
verify' rejects. The bypass writers ('dtrules compile', 'build --from-xml')
were removed in v1.16.0.


3. Compile — dtrules build
---------------------------

Run this after any edit:

    dtrules build

The build auto-detects which side changed and runs the correct
pipeline. Every build ends with canonical .xlsx on disk and compiled
execution .xml on disk — the two stay in sync bit-for-bit.

Useful flags:

    --from-excel      Force Excel → XML (for Excel-authored workflow)
    --dry-run         Show what would change without writing files
    --verbose, -v     Show each intermediate step
    --quiet, -q       Only show output on drops or errors

After a successful build, dtrules prints a Build Summary like:

    Build Summary
    =============
    Import step (Excel → XML):
      tables=3  actions=12  conditions=8  entities=5  mappings=0
      compiled=20
      files-written=4
      drops: none

    Export step (XML → Excel):
      tables=3  files-written=2
      drops: none

Watch the "drops:" line — any non-"none" value means something in your
Excel didn't survive the round-trip. Fix before committing.

See:

    dtrules docs workflow        # build pipeline deep-dive


4. Check structure and EL — dtrules validate
---------------------------------------------

    dtrules validate

Runs structural checks:

  • project directory layout matches conventions
  • every decision table's condition_dsl / action_dsl / context_dsl
    parses as valid EL
  • referenced entity types and fields exist in the EDD

Flags:

    --xml-dir <path>     Override XML directory
    --excel-dir <path>   Override Excel directory

Use --xml-dir when the XML lives outside the conventional layout; the
tool will still require excel/ because Excel is the system of record.

If your project has no excel/ yet (rare — most sampleprojects do), the
authoring API bootstraps it: the first 'dtrules table'/'dtrules edd'
write to an Excel-less project generates Excel from the XML and writes a
sync manifest. After that the project is in normal steady state.


5. Gate CI — dtrules verify
----------------------------

    dtrules verify [path]

Designed for CI / pre-commit hooks. Fails (exit non-zero) on any of:

    build     'dtrules build' on the committed Excel would change the
              committed Excel or XML (drift).
    source    An XML artifact has no valid <source>/<xls_file> reference.
    order     NNN_ filename ordering disagrees with workbook sheet order.
    excel     A project has decision-table/EDD XML but NO Excel workbook —
              i.e. rules were authored straight into XML without building
              the Excel system of record. (Excel-presence gate.)
    external  A table depends on something the project doesn't define: a
              'perform' of an undefined table, an EDD field its entity
              doesn't declare, or an operator absent from the registry.
              (External-reference gate — rules must be self-contained.)

The 'excel' and 'external' gates are why an LLM cannot quietly skip Excel
or build a table on logic that doesn't exist: verify rejects both.

Flags:

    --strict   Also fail on warnings (e.g., missing <source> headers)
    --diff     On failure, print the diff between committed and built

Typical pre-commit hook:

    #!/bin/sh
    dtrules verify . --strict || exit 1

verify is strictly more pedantic than validate — validate catches EL
errors; verify catches round-trip drift, a missing Excel record, and
references to undefined tables/fields/operators.


6. Fine-grained sync — dtrules sync
------------------------------------

    dtrules sync <subcommand>

Subcommands:

    status   Show the sync state of every Excel / XML pair
    check    Exit non-zero if any file has pending user edits
    import   Excel → XML (same as ` + "`build --from-excel`" + `)
    export   XML → Excel (fails if Excel has unsaved edits)
    auto     Pick direction by newer mtime and sync

Most users just run ` + "`dtrules build`" + `; ` + "`dtrules sync`" + ` is for
scripting edge cases (e.g., staged merges, forensic auditing).


7. Version info — dtrules version
----------------------------------

    dtrules version

Prints version, commit SHA, and build date. CI pipelines can grep for
a version string to pin tool expectations.


Typical workflows
-----------------

  New analyst, starting fresh:

      dtrules init
      # open excel/, fill in decision tables
      dtrules build
      dtrules validate
      git commit -am 'initial rules'

  Iterative rule edit:

      # edit excel/<your>.xlsx
      dtrules build
      dtrules verify        # sanity check before commit
      git commit -am 'adjust tax bracket'

  CI integration:

      - run: go install github.com/DTRules/DTRules/cmd/dtrules@v1.8.1
      - run: dtrules verify --strict .
      - run: go test ./...

  Run a table (batch or interactive interview):

      dtrules run . --entry Determine_Therapy --input case.xml   # batch
      dtrules run . --entry Determine_Therapy --interactive      # prompt for
                                                                 # reached collect fields

  Programmatic editing (AI agent / tooling):

      # write through the authoring API — updates XML, postfix, AND Excel
      echo "$table_json" | dtrules table put My_Table --file rules_dt.xml ...
      dtrules verify        # confirms Excel stayed consistent + refs resolve
      # both sides already in sync — commit them


Common errors
-------------

  "excel/ directory not found"
      Project lacks Excel authoring files. The authoring API bootstraps
      Excel on the first ` + "`dtrules table`" + ` / ` + "`dtrules edd`" + ` write,
      or pass --excel-dir to point at a non-default location.

  "parse errors: mismatched input ..."
      An EL expression in a decision table didn't compile.
      ` + "`dtrules validate`" + ` shows the file and line.
      See: ` + "`dtrules docs el`" + `.

  "round-trip drift: build summary shows drops"
      Something in Excel didn't survive compilation. Usually a
      malformed condition column or a type the loader rejects.
      Fix the Excel cell, rebuild.


Where to go next
----------------

    dtrules docs authoring-contract # READ FIRST: the rules-authoring contract
    dtrules docs workflow        # dtrules build pipeline deep-dive
    dtrules docs el              # EL grammar
    dtrules docs edd             # Entity Data Dictionary
    dtrules docs decision-tables # condition/action table shape
    dtrules docs operators       # runtime operator reference
    dtrules docs embedding       # ship a Go binary with rules baked in
    dtrules docs authoring       # programmatic SDK
    dtrules docs entry-points    # run multiple tables against one loaded session
`

const docEntryPoints = `Multiple Entry Points Against a Single Session
==============================================

Overview
--------

A DTRules project usually has more than one decision table. To run
several of them as separate entry points against the same loaded
session — without re-loading rules or re-creating the session for each
table — call ` + "`RSession.Execute(tableName)`" + ` once per table.
The session's entity stack and field values persist across calls, so
mutations made by the first table are visible to the second.

There is no special "entry point" registration: every loaded decision
table is callable by name. ` + "`Execute(tableName)`" + ` is the entry-
point selector.


When to use it
--------------

Three common shapes:

  * Multi-pass evaluation — a table validates inputs and tags errors;
    a second computes a result only on records that passed; a third
    produces a summary.

  * Read-then-write workflows — Compute_Eligibility reads inputs and
    decides; Generate_Audit_Trail then reads the decision and
    populates an audit log entity.

  * Per-request branching — one bound session reused to answer
    different questions against the same loaded data:
    Check_Eligibility for one request, Compute_Risk for the next.

If your tables are wired so that one calls the next via ` + "`perform`" + `,
you don't need this pattern — the call graph handles it. This topic is
for the case where the caller (your Go code, service, or CLI) decides
which entry point to invoke.


The Pattern
-----------

  package main

  import (
      "strings"

      "github.com/DTRules/DTRules/pkg/dtrules"
      "github.com/DTRules/DTRules/pkg/dtrules/session"
  )

  func main() {
      rs := session.NewRuleSet("MyProject")

      // Load EDD + DT once.
      _ = rs.LoadEDD(strings.NewReader(eddXML))
      _ = rs.LoadDecisionTables(strings.NewReader(dtXML))

      // One session, one data load.
      sess, _ := rs.NewSession()
      rsess := sess.(*session.RSession)

      entity, _ := rsess.CreateEntity(dtrules.GetRName("client"))
      entity.Put(dtrules.GetRName("age"), dtrules.GetRIntegerValueFromInt(20))
      rsess.GetState().EntityPush(entity)

      // Entry point #1.
      _ = rsess.Execute("Check_Eligibility")

      // Entry point #2 — same session, same entity, same data.
      // Field values written by #1 are visible to #2.
      _ = rsess.Execute("Compute_Risk")

      rsess.GetState().EntityPop()
  }


What persists across Execute calls
----------------------------------

  Loaded entity definitions (EDD)        yes — owned by the RuleSet
  Loaded decision tables                 yes — same
  Created entities and their field values yes — live in RDTState
  Mutations made by previous tables      yes — visible to next call
  Entity stack pushes from your Go code  yes
  Entity stack pushes inside a table's <contexts>
                                         pushed at table entry,
                                         popped at table exit; do
                                         NOT leak across calls
  Data stack                             empty between calls (runtime
                                         enforces balance at table
                                         boundaries)
  Trace events                           accumulate if a tracker is
                                         attached


ExecuteAt — push entity, run, pop
---------------------------------

For the common case "run table T with entity E at the top of the
context stack," ExecuteAt does the push/pop boilerplate:

  err := rsess.ExecuteAt("Compute_Risk", "client")

Equivalent to:

  entity, _ := rsess.GetState().FindEntity(dtrules.GetRName("client"))
  rsess.GetState().EntityPush(entity)
  err := rsess.Execute("Compute_Risk")
  rsess.GetState().EntityPop()


Errors and partial state
------------------------

If the first Execute errors mid-way, mutations already applied stay
applied. The session is NOT rolled back automatically. Two patterns:

  Snapshot, run, discard on error: copy the entity values you care
                                   about before the first Execute;
                                   restore them on error.

  Idempotent tables:              author so re-running produces the
                                   same result. This is the dominant
                                   shape in tax / eligibility domains
                                   where actions are
                                   set <output> = <pure_function_of_inputs>.


Pitfalls
--------

  Don't reuse a session across unrelated tenants. A session carries
  every entity ever created in it. If you serve multiple tenants from
  one server, give each request its own session via rs.NewSession();
  the RuleSet is the cheap-to-share piece.

  Watch entity stack depth. Tables that push and forget to pop are
  bugs. The runtime checks balance at table boundaries so this
  surfaces as an error, not silent corruption — but the next Execute
  after a broken table sees a deeper stack than expected.

  Execute("UnknownTable") returns an error, not a panic. Treat
  undefined-table as a configuration bug surfaced to operators.


Related
-------

  dtrules docs cli         CLI surface (build / verify / review)
  dtrules docs authoring   programmatic SDK for editing projects
  dtrules docs sdk         embedding the engine in a Go application
  docs/multi-entry-points.md
                           long-form companion with extended examples
  pkg/dtrules/multi_entry_test.go
                           the regression test that pins this contract
`
