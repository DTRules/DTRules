> **NOT FOR RULE AUTHORS.** Rule authors MUST use `dtrules docs el`.
> This file is a reference for engineers debugging the compiler or runtime.
> Never copy postfix from this file into a rule artifact — the EL compiler
> generates postfix automatically.

# EL Reference

Expression Language (EL) is the human-readable syntax used in DTRules condition, action, context, and policy-statement cells. The EL compiler (`pkg/dtrules/compiler/el`) parses EL via ANTLR4 and emits postfix notation. The DTRules runtime (`pkg/dtrules/compiler`) executes that postfix on a stack-based virtual machine.

## Table of Contents

1. [Overview](#overview)
2. [Literals](#literals)
3. [Types](#types)
4. [Operators](#operators)
   - [Arithmetic](#arithmetic-operators)
   - [Comparison](#comparison-operators)
   - [Logical](#logical-operators)
   - [String](#string-operators)
   - [Date](#date-operators)
   - [Array](#array-operators)
   - [Entity](#entity-operators)
5. [Built-in Functions](#built-in-functions)
6. [Control Flow](#control-flow)
7. [Statements](#statements)
   - [set](#set)
   - [increment / decrement](#increment--decrement)
   - [add to / subtract from](#add-to--subtract-from)
   - [perform](#perform)
   - [xml set attribute](#xml-set-attribute)
8. [Map / Filter / Sum / There-is](#map--filter--sum--there-is)
9. [Local Variables](#local-variables)
10. [Grammar Appendix](#grammar-appendix)

---

## Overview

```
EL source text
      │
      ▼ ANTLR4 parse (EL.g4)
 Parse tree
      │
      ▼ PostfixEmitter visitor (postfix_emitter.go)
 Postfix string
      │
      ▼ Runtime compiler (compiler/compiler.go)
 Executable array
      │
      ▼ DTRules stack VM
 Side effects / return value
```

Each decision-table cell holds EL that is compiled once and stored as postfix in the XML. The postfix is then compiled into an executable array by the runtime and executed on the entity stack. Entity field accesses are resolved at runtime against the current entity context.

**Entry points in the grammar (the `done` rule):**

| Cell type          | Grammar alternative      | Postfix result           |
|--------------------|--------------------------|--------------------------|
| `condition`        | `conditionExpr`          | boolean expression        |
| `action`           | `actionStatement`        | statement sequence        |
| `context`          | `contextStatement`       | forall / local var decl  |
| `policystatement`  | `policyStrExpr` etc.     | value expression          |

---

## Literals

### Integer literals

**Syntax**: `INT_LITERAL` — one or more digits.
**Semantics**: Pushes an integer value onto the stack.

**Example (EL)**: `taxpayer.age == 18`
**Compiled postfix**: `taxpayer.age 18 ==`

**Tax example**: `result.total_deduction == 0` → `result.total_deduction 0 ==`
**Eligibility example**: `person.age >= 18` → `person.age 18 >=`

---

### Float literals

**Syntax**: `FLOAT_LITERAL` — digits with a decimal point, e.g., `3.14`, `0.22`, `50000.0`.
**Semantics**: Pushes a double-precision float onto the stack.

**Example (EL)**: `result.agi >= 50000.0`
**Compiled postfix**: `result.agi 50000.0 f>=`

**Tax example**: `bracket.rate >= 0.22` → `bracket.rate 0.22 f>=`
**Eligibility example**: `household.income >= 10000.0` → `household.income 10000.0 f>=`

---

### Boolean literals

**Syntax**: `RBOOLEAN` — one of `true`, `false`, `default`, `otherwise`, `always`, `perform when called`.
**Semantics**: `true` and `false` are boolean constants. `otherwise`/`default`/`always` are synonyms meaning "this row always fires" — used in the last column of a FIRST-type decision table.

**Example (EL)**: `true`
**Compiled postfix**: `true`

**Example (EL)**: `otherwise`
**Compiled postfix**: `otherwise`

**Tax example**: `perform when called` → `otherwise` (the table runs whenever invoked)
**Eligibility example**: `otherwise` in the final catch-all row of a FIRST table.

---

### String literals

**Syntax**: `STRING_LITERAL` — characters enclosed in double or single quotes.
**Semantics**: Pushes a string value. Compared with `streq`.

**Example (EL)**: `taxpayer.filing_status == "MFJ"`
**Compiled postfix**: `taxpayer.filing_status "MFJ" streq`

**Tax example**: `taxpayer.filing_status == "SINGLE"` → `taxpayer.filing_status "SINGLE" streq`
**Eligibility example**: `person.relationship_to_head == "SPOUSE"` → `person.relationship_to_head "SPOUSE" streq`

---

## Types

DTRules supports the following primitive and composite types. Each type keyword is used in cast expressions and local variable declarations.

| EL keyword          | Notes                                    |
|---------------------|------------------------------------------|
| `int` / `long`      | 64-bit signed integer                    |
| `double`            | IEEE 754 double                          |
| `boolean`           | `true` / `false`                         |
| `string`            | UTF-8 string                             |
| `date` / `time`     | Represented as milliseconds since epoch  |
| `bigint` / `biginteger` | Arbitrary-precision integer          |
| `bytes`             | Immutable byte sequence (hex, constant-time equality) |
| `name`              | Symbol/name value (e.g., `$foo`)        |
| `entity`            | Reference to a DTRules entity            |
| `array`             | Ordered list of DTRules objects          |
| `table`             | Decision-table reference                 |

The type of each identifier is resolved from the Entity Definition Document (EDD) loaded at compile time. The compiler calls `SetSymbols(map[string]string)` with a flattened type map before emitting postfix.

---

## Operators

### Arithmetic Operators

EL supports standard arithmetic on integers, doubles, and bigints. The emitter maps to different postfix operators depending on the operand types.

#### Integer addition

**Syntax**: `iexpr + iexpr`
**Semantics**: Integer addition. Postfix operator: `+`.
**Example (EL)**: `taxpayer.age + 1 > 0`
**Compiled postfix**: `taxpayer.age 1 + 0 >`

**Tax example**: `taxpayer.age + 1 >= 18` → `taxpayer.age 1 + 18 >=`

#### Integer subtraction

**Syntax**: `iexpr - iexpr`
**Semantics**: Integer subtraction. Postfix operator: `-`.
**Example (EL)**: `count - 1 >= 0`
**Compiled postfix**: `count 1 - 0 >=`

#### Integer multiplication

**Syntax**: `iexpr * iexpr`
**Semantics**: Integer multiplication. Postfix operator: `*`.
**Example (EL)**: `count * 2 > 0`
**Compiled postfix**: `count 2 * 0 >`

#### Integer division

**Syntax**: `iexpr / iexpr` or `iexpr div iexpr`
**Semantics**: Integer division (truncates). Postfix operator: `/`.
**Example (EL)**: `total / count >= 0`
**Compiled postfix**: `total count / 0 >=`

#### Float addition

**Syntax**: `fexpr + fexpr` or `fexpr + iexpr`
**Semantics**: Float addition. Postfix operator: `+` (VM dispatches on stack types).
**Example (EL)**: `income.amount + result.agi >= 100.0`
**Compiled postfix**: `income.amount result.agi + 100.0 f>=`

**Tax example**: `result.agi - result.total_deduction > 0.0` → `result.agi result.total_deduction - 0.0 f>`

#### Float multiplication

**Syntax**: `fexpr * fexpr`
**Semantics**: Float multiplication. Postfix operator: `fmul`.
**Example (EL)**: `bracket.rate * result.taxable_income >= 0.0`
**Compiled postfix**: `bracket.rate result.taxable_income * 0.0 f>=`

#### Float division

**Syntax**: `fexpr / fexpr`
**Semantics**: Float division. Postfix operator: `fdiv`.
**Example (EL)**: `result.total_tax / result.agi >= 0.0`
**Compiled postfix**: `result.total_tax result.agi / 0.0 f>=`

#### Negation

**Syntax**: `-iexpr` or `-fexpr`
**Semantics**: Unary minus. Postfix operator: `neg`.
**Example (EL)**: `-count == 0`
**Compiled postfix**: `count neg 0 ==`

#### BigInt arithmetic

**Syntax**: `bigexpr OP bigexpr` where OP is `+`, `-`, `*`, `/`
**Semantics**: Arbitrary-precision operations. Postfix operators: `b+`, `b-`, `b*`, `b/`.
**Example (EL)**: `result.large_amount >= constants.limit` (both fields declared bigint in EDD)
**Compiled postfix**: `result.large_amount constants.limit b>=`

**BigInt equality**: uses `streq` (values compared as strings by default) or `b==` if both are bigint.

#### Rounding

**Syntax**: `fexpr rounded` or `fexpr rounded to N decimal places`
**Semantics**: Round float to nearest integer or to N decimal places. Postfix operator: `round`.
**Example (EL)**: `result.agi rounded == 50000`
**Compiled postfix**: `result.agi round 50000 ==`

#### Sum of

**Syntax**: `sum of fexpr IN arrayExpr` or `sum of iexpr IN arrayExpr`
**Semantics**: Sums a numeric field across all elements of an array. Postfix operator: `sumof`.
**Example (EL)**: `sum of income.amount in person.incomes >= 10000.0`
**Compiled postfix**: `person.incomes income.amount sumof 10000.0 f>=`

**Tax example**: `sum of w2.wages in w2s >= 0.0`
**Eligibility example (real postfix from DTEligibility)**:
```
0 person.incomes { income entitypush income.is_earned { income.amount + } if entitypop } forall
person /earned_income exch def
```

---

### Comparison Operators

EL comparison operators produce boolean values. The postfix operator depends on the operand types.

| EL operator                                             | Integer postfix | Float postfix | String postfix | Date postfix | BigInt postfix |
|---------------------------------------------------------|-----------------|---------------|----------------|--------------|----------------|
| `==` / `is equal to`                                    | `==`            | `f==`         | `streq`        | `d==`        | `b==`          |
| `!=` / `is not equal to`                                | `== not`        | `f== not`     | `streq not`    | `d== not`    | `b!=`          |
| `>` / `is greater than`                                 | `>`             | `f>`          | `>`            | `d>`         | `b>`           |
| `>=` / `is greater than or equal to` / `at or above`   | `>=`            | `f>=`         | `>=`           | `d< not`     | `b>=`          |
| `<` / `is less than`                                    | `<`             | `f<`          | `<`            | `d<`         | `b<`           |
| `<=` / `is less than or equal to` / `at or below`      | `<=`            | `f<=`         | `<=`           | `d> not`     | `b<=`          |

Natural language forms (`is greater than`, `at or above`, etc.) compile identically to the symbolic form.

**Example (EL)**: `taxpayer.age >= 18`
**Compiled postfix**: `taxpayer.age 18 >=`

**Example (EL)**: `result.agi >= 50000.0`
**Compiled postfix**: `result.agi 50000.0 f>=`

**Example (EL)**: `taxpayer.filing_status == "MFJ"`
**Compiled postfix**: `taxpayer.filing_status "MFJ" streq`

**Example (EL)**: `taxpayer.birth_date is before current date`
**Compiled postfix**: `taxpayer.birth_date currentdate d<`

**Tax example (natural language)**: `taxpayer.age is greater than or equal to 65` → `taxpayer.age 65 >=`
**Eligibility example (real postfix)**: `person.age constants.adult_age ge`

#### String "is one of"

**Syntax**: `strexpr IS ONE OF arrayExpr`
**Semantics**: True if the string is a member of the array. Postfix: `memberof`.
**Example (EL)**: `taxpayer.filing_status is one of valid_statuses`
**Compiled postfix**: `taxpayer.filing_status valid_statuses memberof`

#### Within percent

**Syntax**: `fexpr IS WITHIN number PERCENT OF fexpr`
**Semantics**: True if the two values differ by no more than N%.
**Example (EL)**: `result.agi is within 10 percent of previous.agi`
**Compiled postfix**: `result.agi 10 previous.agi withinpct`

---

### Logical Operators

#### AND (lazy evaluation)

**Syntax**: `bexpr AND bexpr` or `bexpr && bexpr`
**Semantics**: Short-circuits — if the left side is false, the right side is not evaluated.
**Example (EL)**: `x == 1 and y == 2`
**Compiled postfix**: `x 1 == { pop y 2 == } over if`

**Tax example**: `taxpayer.is_blind and taxpayer.age >= 65`
→ `taxpayer.is_blind { pop taxpayer.age 65 >= } over if`

**Eligibility example**: `person.is_adult and person.is_eligible`
→ `person.is_adult { pop person.is_eligible } over if`

#### OR (lazy evaluation)

**Syntax**: `bexpr OR bexpr` or `bexpr || bexpr`
**Semantics**: Short-circuits — if the left side is true, the right side is not evaluated.
**Example (EL)**: `x == 1 or y == 2`
**Compiled postfix**: `x 1 == { pop y 2 == } over not if`

**Tax example**: `taxpayer.filing_status == "MFJ" or taxpayer.filing_status == "MFS"`
→ `taxpayer.filing_status "MFJ" streq { pop taxpayer.filing_status "MFS" streq } over not if`

#### NOT

**Syntax**: `NOT bexpr`
**Semantics**: Boolean negation. Postfix operator: `not`.
**Example (EL)**: `not x == 1`
**Compiled postfix**: `x 1 == not`

**Tax example**: `not taxpayer.is_blind` → `taxpayer.is_blind not`
**Eligibility example (real postfix)**: `income.is_earned not`

#### Boolean equality

**Syntax**: `bexpr == bexpr`
**Semantics**: Tests two booleans for equality. Postfix operator: `beq`.
**Example (EL)**: `person.is_adult == person.is_eligible`
**Compiled postfix**: `person.is_adult person.is_eligible beq`

#### Boolean "is" test

**Syntax**: `typedBoolean IS RBOOLEAN` or `typedBoolean IS NOT RBOOLEAN`
**Semantics**: Compare a boolean field against a literal. Postfix: `beq` or `beq not`.
**Example (EL)**: `person.is_adult is true`
**Compiled postfix**: `person.is_adult true beq`

**Example (EL)**: `person.is_adult is not true`
**Compiled postfix**: `person.is_adult true beq not`

---

### String Operators

#### Concatenation

**Syntax**: `strexpr + strexpr`
**Semantics**: Concatenate two strings. Postfix operator: `strconcat`.
**Example (EL)**: `first_name + " " + last_name == "John Smith"`
**Compiled postfix**: `first_name " " strconcat last_name strconcat "John Smith" streq`

**Tax example**: `taxpayer.filing_status + taxpayer.age == "MFJ30"`
→ `taxpayer.filing_status taxpayer.age strconcat "MFJ30" streq`

#### Change to lower case / upper case

**Syntax**: `CHANGE strexpr TO LOWER_CASE` / `CHANGE strexpr TO UPPER_CASE`
**Semantics**: Convert string case. Postfix operators: `tolower`, `toupper`.
**Example (EL)**: `change status to lower case == "active"`
**Compiled postfix**: `status tolower "active" streq`

**Example (EL)**: `change taxpayer.filing_status to upper case == "MFJ"`
**Compiled postfix**: `taxpayer.filing_status toupper "MFJ" streq`

#### Trim

**Syntax**: `trim(strexpr)`
**Semantics**: Trim whitespace from both ends. Postfix operator: `trim`.
**Example (EL)**: `trim(taxpayer.filing_status) == "MFJ"`
**Compiled postfix**: `taxpayer.filing_status trim "MFJ" streq`

#### Starts with

**Syntax**: `strexpr STARTS_WITH strexpr`
**Semantics**: True if the string starts with the given prefix. Postfix operator: `startswith`.
**Example (EL)**: `taxpayer.filing_status starts with "M"`
**Compiled postfix**: `taxpayer.filing_status "M" startswith`

#### Matches (regex)

**Syntax**: `strexpr MATCHES strexpr`
**Semantics**: True if the string matches the regular expression. Postfix operator: `matches`.
**Example (EL)**: `taxpayer.filing_status matches "MF.*"`
**Compiled postfix**: `taxpayer.filing_status "MF.*" matches`

#### Equals ignore case

**Syntax**: `strexpr is equal to ignore case strexpr`
**Semantics**: Case-insensitive string equality. Postfix operator: `sic==`.
**Example (EL)**: `taxpayer.filing_status is equal to ignore case "mfj"`
**Compiled postfix**: `taxpayer.filing_status "mfj" sic==`

#### Substring

**Syntax**: `SUBSTRING OF strexpr FROM iexpr TO iexpr`
**Semantics**: Extract a substring by character index. Postfix operator: `substring`.
**Example (EL)**: `substring of taxpayer.filing_status from 0 to 1 == "M"`
**Compiled postfix**: `taxpayer.filing_status 0 1 substring "M" streq`

#### Index of

**Syntax**: `INDEX_OF strexpr IN strexpr`
**Semantics**: Returns the index of the first occurrence, or -1. Postfix operator: `indexof`.
**Example (EL)**: `index of "M" in taxpayer.filing_status >= 0`
**Compiled postfix**: `"M" taxpayer.filing_status indexof 0 >=`

#### String value of

**Syntax**: `STRING VALUE OF iexpr|fexpr|dexpr|BOOLEAN bexpr`
**Semantics**: Convert a value to its string representation.
**Example (EL)**: `string value of taxpayer.age == "65"`
**Compiled postfix**: `taxpayer.age cvs "65" streq`

**Example (EL)**: `string value of boolean taxpayer.is_blind == "true"`
**Compiled postfix**: `taxpayer.is_blind cvb cvs "true" streq`

#### Length of

**Syntax**: `LENGTH OF strexpr` or `LENGTH OF arrayExpr`
**Semantics**: Returns the length of a string or array. Postfix operator: `length`.
**Example (EL)**: `length of taxpayer.filing_status > 0`
**Compiled postfix**: `taxpayer.filing_status length 0 >`

---

### Date Operators

#### Current date

**Syntax**: `current date` or `current time`
**Semantics**: Returns the current date/time. Postfix: `currentdate`.
**Example (EL)**: `taxpayer.birth_date is before current date`
**Compiled postfix**: `taxpayer.birth_date currentdate d<`

#### Date from string

**Syntax**: `(date) strexpr` or `date(strexpr)`
**Semantics**: Parse a string into a date value. Postfix: `cvd`.
**Example (EL)**: `(date)"2024-01-01" is before current date`
**Compiled postfix**: `"2024-01-01" cvd currentdate d<`

#### Date arithmetic (date plus/minus days/months/years)

**Syntax**: `dexpr plus N days|months|years` / `dexpr minus N days|months|years`
**Semantics**: Add or subtract a date interval. Postfix operators: `adddays`, `subdays`, `addmonths`, `submonths`, `addyears`, `subyears`.

**Example (EL)**: `taxpayer.birth_date plus 18 years is before current date`
**Compiled postfix**: `taxpayer.birth_date 18 addyears currentdate d<`

**Example (EL)**: `taxpayer.birth_date minus 1 months == current date`
**Compiled postfix**: `taxpayer.birth_date 1 submonths currentdate d==`

#### Date statement arithmetic (modifies field in place)

**Syntax**: `ADD N DAYS|MONTHS|YEARS TO typedDate` / `SUBTRACT N DAYS|MONTHS|YEARS FROM typedDate`
**Semantics**: Modify a date field by adding or subtracting an interval. Used as an action statement.
**Example (EL)**: `add 18 years to taxpayer.birth_date`
**Compiled postfix**: `taxpayer.birth_date 18 addyears /taxpayer.birth_date xdef`

#### Days / months / years between

**Syntax**: `DAYS FROM dexpr TO dexpr` / `MONTHS FROM dexpr TO dexpr` / `YEARS FROM dexpr TO dexpr`
**Semantics**: Returns integer difference. Postfix operators: `daysbetween`, `monthsbetween`, `yearsbetween`.
**Example (EL)**: `years from taxpayer.birth_date to current date >= 18`
**Compiled postfix**: `taxpayer.birth_date currentdate yearsbetween 18 >=`

**Tax example**: `years from taxpayer.birth_date to current date >= 65` → age 65+ check
**Eligibility example**: `years from taxpayer.birth_date to current date >= constants.adult_age`

#### Is before / is after / is between

**Syntax**: `dexpr IS BEFORE dexpr` / `dexpr IS AFTER dexpr` / `dexpr IS BETWEEN dexpr AND dexpr`
**Semantics**: Date range comparisons.

| EL                         | Postfix               |
|----------------------------|-----------------------|
| `d1 is before d2`          | `d1 d2 d<`            |
| `d1 is after d2`           | `d1 d2 d>`            |
| `d1 < d2`                  | `d1 d2 d<`            |
| `d1 > d2`                  | `d1 d2 d>`            |
| `d1 >= d2`                 | `d1 d2 d< not`        |
| `d1 <= d2`                 | `d1 d2 d> not`        |

**Example (EL)**: `taxpayer.birth_date is after current date`
**Compiled postfix**: `taxpayer.birth_date currentdate d>`

#### First of year/month, end of month

**Syntax**: `FIRST OF YEARS OF dexpr` / `FIRST OF MONTHS OF dexpr` / `END OF MONTHS OF dexpr`
**Semantics**: Returns the first or last day of the year or month containing the date.
**Example (EL)**: `first of years of taxpayer.birth_date == current date`
**Compiled postfix**: `taxpayer.birth_date firstofyear currentdate d==`

**Example (EL)**: `end of months of taxpayer.birth_date == current date`
**Compiled postfix**: `taxpayer.birth_date endofmonth currentdate d==`

#### Get year of / days in year / days in month

**Syntax**: `GET YEAROF dexpr` / `GET DAYS IN YEAROF dexpr` / `GET DAYS IN MONTHS FOR dexpr`
**Semantics**: Extracts integer date components.
**Example (EL)**: `get yearof taxpayer.birth_date >= 1980`
**Compiled postfix**: `taxpayer.birth_date yearof 1980 >=`

**Example (EL)**: `get days in yearof taxpayer.birth_date == 365`
**Compiled postfix**: `taxpayer.birth_date daysinyr 365 ==`

---

### Array Operators

#### Number of

**Syntax**: `NUMBEROF arrayExpr` or `NUMBEROF arrayExpr WHERE bexpr`
**Semantics**: Count of elements in an array, optionally filtered. Postfix: `numberof`.
**Example (EL)**: `number of household.members == 4`
**Compiled postfix**: `household.members numberof 4 ==`

**Example (EL)**: `number of household.members where person.is_adult >= 1`
**Compiled postfix**: `household.members { person entitypush person.is_adult entitypop } filter numberof 1 >=`

**Tax example**: `number of w2s > 0`
**Eligibility example**: `number of household.members where person.is_adult >= 1`

#### Length of

**Syntax**: `LENGTH OF arrayExpr`
**Semantics**: Returns the count of items in an array. Postfix: `length`.
**Example (EL)**: `length of household.members >= 1`
**Compiled postfix**: `household.members length 1 >=`

#### Includes / does not include

**Syntax**: `arrayExpr INCLUDES includeSearch` / `arrayExpr DOES NOT INCLUDE includeSearch`
**Semantics**: Tests array membership. Postfix: `memberof`, or `memberof not`.
**Example (EL)**: `household.members includes person`
**Compiled postfix**: `household.members person memberof`

**Example (EL)**: `household.members does not include value 0`
**Compiled postfix**: `household.members 0 memberof not`

#### Add to array

**Syntax**: `ADD eexpr TO arrayExpr`
**Semantics**: Append an element to an array.
**Example (EL)**: `add person to household.members`
**Compiled postfix**: `person household.members swap addto`

#### Clear array

**Syntax**: `CLEAR arrayExpr`
**Semantics**: Remove all elements from an array. Postfix: `clear`.
**Example (EL)**: `clear household.members`
**Compiled postfix**: `household.members clear`

#### Sort array

**Syntax**: `SORT arrayExpr IN ASCENDINGORDER BY nexpr` / `SORT arrayExpr IN DESCENDINGORDER BY nexpr`
**Semantics**: Sort an array by a named field.
**Example (EL)**: `sort household.members in ascending order by name`
**Compiled postfix**: `household.members /name sortasc`

**Example (EL)**: `sort household.members in descending order by name`
**Compiled postfix**: `household.members /name sortdesc`

#### Remove from array

**Syntax**: `REMOVE eexpr FROM arrayExpr ARRAY`
**Semantics**: Remove a specific element from an array. Postfix: `removefrom`.
**Example (EL)**: `remove person from household.members array`
**Compiled postfix**: `person household.members removefrom`

#### Remove each where

**Syntax**: `REMOVE EACH eexpr FROM arrayExpr WHERE bexpr`
**Semantics**: Remove all elements matching a condition.
**Example (EL)**: `remove each person from household.members where not person.is_adult`
**Compiled postfix**: `household.members { person entitypush person.is_adult not entitypop } filter removefromall`

#### Array literal

**Syntax**: `{ item, item, ... }`
**Semantics**: Construct an inline array of values.
**Example (EL)**: `{ 1, 2, 3 }`
**Compiled postfix**: `1 2 3 3 array`

#### Array at index

**Syntax**: `(string) arrayExpr[iexpr]` / `(long) arrayExpr[iexpr]`
**Semantics**: Access an array element by index with a type cast.
**Example (EL)**: `(string) myarray[idx] == "foo"`
**Compiled postfix**: `myarray idx getat cvs "foo" streq`

#### Copy / deep copy

**Syntax**: `copy of arrayExpr` / `deep copy of arrayExpr`
**Semantics**: Shallow or deep copy of an array. Postfix: `copy` or `deepcopy`.
**Example (EL)**: `copy of household.members`
**Compiled postfix**: `household.members copy`

#### Map through

**Syntax**: `MAP arrayExpr THROUGH texpr`
**Semantics**: Apply a decision table to each element and collect results. Postfix: `mapthrough`.
**Example (EL)**: `map brackets through Apply_Bracket`
**Compiled postfix**: `brackets Apply_Bracket mapthrough`

#### Tokenize

**Syntax**: `TOKENIZE strexpr BY strexpr`
**Semantics**: Split a string by delimiter into an array. Postfix: `tokenize`.
**Example (EL)**: `tokenize csv_line by ","`
**Compiled postfix**: `csv_line "," tokenize`

---

### Entity Operators

#### Entity reference

**Syntax**: `typedEntity` — an identifier resolved to an entity type via the EDD.
**Semantics**: Pushes the entity reference onto the stack.
**Example (EL)**: `person` (identifier declared entity in EDD)
**Compiled postfix**: `person`

#### New entity

**Syntax**: `NEW typedEntity ENTITY` or `NEW nexpr ENTITY`
**Semantics**: Creates a new entity instance of the named type. Postfix: `createentity`.
**Example (EL)**: `set person = new Person entity`
**Compiled postfix**: `/Person createentity cve /person xdef`

#### Clone

**Syntax**: `CLONE OF eexpr`
**Semantics**: Creates a shallow clone of an entity. Postfix: `clone`.
**Example (EL)**: `clone of person`
**Compiled postfix**: `person clone`

#### Entity equality

**Syntax**: `eexpr == eexpr` / `eexpr != eexpr`
**Semantics**: Reference equality. Postfix: `req`, or `req not`.
**Example (EL)**: `person == primary_applicant`
**Compiled postfix**: `person primary_applicant req`

#### Entity null test

**Syntax**: `eexpr is null` / `eexpr is not null`
**Semantics**: Tests if an entity reference is null. Postfix: `isnull`, `isnull not`.
**Example (EL)**: `household.primary_earner is null`
**Compiled postfix**: `household.primary_earner isnull`

**Eligibility example (real postfix from DTEligibility)**:
`household.primary_earner null ne { true household.primary_earner /is_primary_earner exch def } if`

#### Has a (relationship)

**Syntax**: `eexpr HASA strexpr` / `eexpr DOES NOT HAVE strexpr`
**Semantics**: Tests if an entity has a named relationship. Postfix: `hasrelationship`.
**Example (EL)**: `person hasa "child"`
**Compiled postfix**: `person "child" hasrelationship`

#### Entity is of (relationship)

**Syntax**: `eexpr IS strexpr OF eexpr`
**Semantics**: Tests a named relationship between two entities.
**Example (EL)**: `client is "parent" of applicant`
**Compiled postfix**: `/source client /target applicant /type "parent" relationships findmatch swap pop`

#### Is in context

**Syntax**: `typedEntity ENTITY IS IN CONTEXT` / `typedEntity ENTITY IS NOT IN CONTEXT`
**Semantics**: Tests if an entity type is in the current rule context.
**Example (EL)**: `person entity is in context`
**Compiled postfix**: `person isincontext`

---

## Built-in Functions

### Absolute value

**Syntax**: `absolute value of iexpr|fexpr|bigexpr`
**Semantics**: Returns the absolute value. Postfix: `abs`.
**Example (EL)**: `absolute value of result.agi > 0.0`
**Compiled postfix**: `result.agi abs 0.0 f>`

### Nameof

**Syntax**: `nameof eexpr`
**Semantics**: Returns the name (type identifier) of an entity. Postfix: `nameof`.
**Example (EL)**: `nameof person == "Person"`
**Compiled postfix**: `person nameof "Person" streq`

### Earliest of after

**Syntax**: `EARLIEST OF arrayExpr AFTER dexpr`
**Semantics**: Returns the earliest date in an array that is after a given date.
**Example (EL)**: `earliest of pending_dates after current date`
**Compiled postfix**: `pending_dates currentdate earliestafter`

### Randomize

**Syntax**: `RANDOMIZE arrayExpr`
**Semantics**: Randomly reorders array elements in place.
**Example (EL)**: `randomize household.members`
**Compiled postfix**: `household.members randomize`

### Remove at index

**Syntax**: `REMOVE iexpr ELEMENT FROM arrayExpr ARRAY`
**Semantics**: Remove the element at a specific index.
**Example (EL)**: `remove 0 element from household.members array`
**Compiled postfix**: `0 household.members removeat`

### Table lookup

**Syntax**: `(long) typedTable(key)` / `(double) typedTable(key)` / `(string) typedTable(key)`
**Semantics**: Look up a value in a named decision table using a key string.
**Example (EL)**: `(double) bracket_table("0.22")`
**Compiled postfix**: `"0.22" bracket_table tablelookup cvd`

### Using (delegation)

**Syntax**: `USING eexpr (expr)`
**Semantics**: Evaluate an expression in the context of a different entity.
**Example (EL)**: `using person (person.age >= 18)`
**Compiled postfix**: `person entitypush person.age 18 >= entitypop`

### Get current timestamp

**Syntax**: `get current timestamp`
**Semantics**: Returns the current date/time as a formatted string.
**Example (EL)**: `get current timestamp == ""`
**Compiled postfix**: `currenttimestamp ""`

### Mapping key

**Syntax**: `mapping key`
**Semantics**: Returns the current key when iterating over a mapped structure.
**Example (EL)**: `mapping key == "CHIP"`
**Compiled postfix**: `mappingkey "CHIP" streq`

### Relationship between

**Syntax**: `RELATIONSHIP_BETWEEN eexpr AND eexpr`
**Semantics**: Returns the relationship string between two entities.
**Example (EL)**: `relationship between person and household.head == "SPOUSE"`
**Compiled postfix**: `person household.head relationshipbetween "SPOUSE" streq`

---

## Control Flow

### if / then / else / endif

**Syntax**:
```
if bexpr then
  { statements }
endif

if bexpr then
  { statements }
else
  { statements }
endif
```

**Semantics**: Conditional execution. Compiles to the pattern `condition { then-block } { else-block } ifelse`. An `if...endif` with no else emits `{}` for the empty else block.

**Example (EL)**: `if x > 0 then { set y = 1; } else { set y = 0; } endif`
**Compiled postfix**: `x 0 > { 1 cvi /y xdef } { 0 cvi /y xdef } ifelse`

**Tax example**:
```
if result.taxable_income >= 523601.0 then
  { set result.marginal_rate = 0.37; }
else
  { set result.marginal_rate = 0.35; }
endif
```
Compiles to:
`result.taxable_income 523601.0 f>= { 0.37 cvd /result.marginal_rate xdef } { 0.35 cvd /result.marginal_rate xdef } ifelse`

**Eligibility example (real postfix from DTEligibility)**:
`true person /is_adult exch def false person /is_child exch def`

---

### else if (chained)

**Syntax**:
```
if bexpr then
  { statements }
else if bexpr then
  { statements }
else
  { statements }
endif
```

**Semantics**: Compiles to nested `ifelse` calls.

**Example (EL)**:
```
if x == 1 then { set y = 1; }
else if x == 2 then { set y = 2; }
else { set y = 0; }
endif
```
**Compiled postfix**:
`x 1 == { 1 cvi /y xdef } { x 2 == { 2 cvi /y xdef } { 0 cvi /y xdef } ifelse } ifelse`

---

### for each (foreach)

**Syntax**:
```
for each eexpr in arrayExpr { statements }
for each eexpr in arrayExpr where bexpr { statements }
for each eexpr and its eexpr in arrayExpr { statements }
```

**Semantics**: Iterate over an array, binding each element. Compiles to `entitypush`/`entitypop` loops using `forall`.

**Example (EL)**: `for each person in household.members { Calculate_Individual_Income; }`
**Real postfix (from DTEligibility)**: `household.members { person entitypush Calculate_Individual_Income entitypop } forall`

**Tax example**: `for each w2 in w2s { Process_W2_Income; }`
**Real postfix**: `w2s { w2 entitypush Process_W2_Income entitypop } forall`

---

### for all (forall)

**Syntax**:
```
for all arrayExpr { statements }
for all arrayExpr where bexpr { statements }
for all arrayExpr in eexpr { statements }
for all arrayExpr allowing array to be removed { statements }
```

**Semantics**: Variant of foreach for cases where the entity binding is implicit. Also compiles to `forall` postfix loops.

**Real postfix (from DTEligibility)**:
```
household.members { person entitypush Determine_Adult_Status entitypop } forall
household.members { person entitypush Evaluate_Non_EDG_Programs entitypop } forall
```

---

### for first (first block)

**Syntax**:
```
for first of arrayExpr where bexpr then
  { statements }
endff

for first of arrayExpr where bexpr then
  { statements }
else if none are found
  { statements }
endff
```

**Semantics**: Find the first element matching a condition. If found, execute the first block; if none found, execute the else block.

**Example (EL)**:
```
for first of household.members where person.is_adult then
  { set result.head_of_household = person; }
else if none are found
  { set result.eligible = false; }
endff
```

**Eligibility example (real postfix pattern)**:
```
null 0 household.members
{ person entitypush person.is_adult
  { pop pop person person.earned_income } if entitypop } forall
pop household /primary_earner exch def
```

---

### for (C-style loop)

**Syntax**: `for leftIexpr = number ; bexpr ; statement`
**Semantics**: Classic for-loop used in context cells to iterate with an index.
**Example (EL)**: `for i = 0; i < 10; increment i;`
**Compiled postfix**: `0 cvi /i xdef { i 10 < } { i 1 + /i xdef } while`

---

## Statements

### set

**Syntax**: `SET leftExpr = expr`
**Semantics**: Assign a value to a field. The compiler emits the value, a type-conversion operator, then the assignment (`/fieldname xdef`).

**Example (EL)**: `set count = 0`
**Compiled postfix**: `0 cvi /count xdef`

**Example (EL)**: `set result.eligible = true`
**Compiled postfix**: `true cvb /result.eligible xdef`

**Example (EL)**: `set result.total_tax = 0.0`
**Compiled postfix**: `0.0 cvd /result.total_tax xdef`

**Example (EL)**: `set taxpayer.filing_status = "SINGLE"`
**Compiled postfix**: `"SINGLE" cvs /taxpayer.filing_status xdef`

**Example (EL)**: `set result.taxable_income = result.agi - result.total_deduction`
**Compiled postfix**: `result.agi result.total_deduction - cvd /result.taxable_income xdef`

**Tax example (real postfix from TaxReturn)**:
`0 cvi /result.qbi_deduction xdef`

**Eligibility example (real postfix from DTEligibility)**:
`true person /is_adult exch def false person /is_child exch def`
`person.earned_income person.unearned_income + person /total_income exch def`

The type-conversion operators used in set statements:

| Target type | Conversion op |
|-------------|--------------|
| `long`      | `cvi`        |
| `double`    | `cvd`        |
| `boolean`   | `cvb`        |
| `string`    | `cvs`        |
| `entity`    | `cve`        |
| `date`      | `cvd`        |
| `bigint`    | `cvbi`       |

---

### increment / decrement

**Syntax**: `INCREMENT typedLong` / `DECREMENT typedLong`
**Semantics**: Add or subtract 1 from an integer field in place.
**Example (EL)**: `increment taxpayer.age`
**Compiled postfix**: `taxpayer.age 1 + /taxpayer.age xdef`

**Example (EL)**: `decrement count`
**Compiled postfix**: `count 1 - /count xdef`

---

### add to / subtract from

**Syntax**: `ADD number TO typedDouble|typedLong` / `SUBTRACT number FROM typedDouble|typedLong`
**Semantics**: Add or subtract a constant from a numeric field in place.
**Example (EL)**: `add 500.0 to result.total_deduction`
**Compiled postfix**: `result.total_deduction 500.0 + /result.total_deduction xdef`

**Example (EL)**: `subtract 1000 from taxpayer.age`
**Compiled postfix**: `taxpayer.age 1000 - /taxpayer.age xdef`

---

### perform

**Syntax**: `PERFORM typedDecisionTable` or bare `typedDecisionTable`
**Semantics**: Execute a named decision table. Emits just the table name token; the runtime executes it.
**Example (EL)**: `perform Calculate_Tax`
**Compiled postfix**: `Calculate_Tax`

**Tax example (real from TaxReturn)**:
`Apply_Tax_Brackets_Single` → `Apply_Tax_Brackets_Single`

**Eligibility example (real from DTEligibility)**:
`Calculate_Household_Totals` → `Calculate_Household_Totals`

#### Perform with error handling

**Syntax**: `PERFORM typedDecisionTable AND ON ERROR ADD eexpr TO CONTEXT AND PERFORM typedDecisionTable`
**Semantics**: Execute a table; on error, add an entity to context and run the error handler.
**Example (EL)**: `perform Calculate_State_Tax and on error add error to context and perform Handle_Tax_Error`
**Compiled postfix**: `Calculate_State_Tax error context addto Handle_Tax_Error`

---

### xml set attribute

**Syntax**:
```
typedXmlValue : set attribute strexpr = xmlvalues
eexpr : set attribute strexpr = xmlvalues
typedXmlValue : add attribute strexpr = xmlvalues
```

**Semantics**: Set or add an XML attribute on an entity's backing XML element.
**Example (EL)**: `w2 : set attribute "wages" = w2.wages`
**Compiled postfix**: `w2 "wages" w2.wages setattr`

**Example (EL)**: `w2 : add attribute "employer" = employer_name`
**Compiled postfix**: `w2 "employer" employer_name addattr`

---

### debug / print

**Syntax**: `DEBUG expr` / `PRINT expr`
**Semantics**: Emit a debug or print statement at runtime. Postfix: `debug` or `print`.
**Example (EL)**: `debug taxpayer.age`
**Compiled postfix**: `taxpayer.age debug`

---

### add to context

**Syntax**: `ADD eexpr TO CONTEXT OF THIS TABLE` / `ADD eexpr TO CONTEXT FOR THIS TABLE`
**Semantics**: Add an entity to the decision table's context stack.
**Example (EL)**: `add person to context of this table`
**Compiled postfix**: `person context addto`

---

### local variables (context cell)

See [Local Variables](#local-variables) section.

---

## Map / Filter / Sum / There-is

### Map

**Syntax**: `MAP arrayExpr THROUGH texpr`
**Semantics**: Apply a decision table to each element in an array and collect the results into a new array.
**Example (EL)**: `map brackets through Apply_Bracket`
**Compiled postfix**: `brackets Apply_Bracket mapthrough`

**Tax example**: Map each W2 through a processing table:
`map w2s through Process_W2_Income`

---

### There is / Is there

**Syntax**:
```
there is eexpr in arrayExpr where bexpr
is there eexpr in arrayExpr where bexpr
there is no eexpr in arrayExpr where bexpr
there is no eexpr in eexpr where bexpr
```

**Semantics**: Existential quantifier over an array. Returns boolean.

**Example (EL)**: `there is person in household.members where person.is_adult`
**Compiled postfix**: `household.members { person entitypush person.is_adult entitypop } filter length 0 >`

**Example (EL)**: `there is no person in household.members where person.is_adult`
**Compiled postfix**: `household.members { person entitypush person.is_adult entitypop } filter length 0 ==`

**Tax example**: `there is person in household.members where person.has_income`
**Eligibility example (real pattern from DTEligibility)**:
`household.members { person entitypush person.is_adult entitypop } filter`

---

### All have / One of has a

**Syntax**: `ALL arrayExpr HAVE bexpr` / `ONE OF arrayExpr HASA bexpr`
**Semantics**: Universal and existential quantifiers.
**Example (EL)**: `all household.members have person.is_adult`
**Compiled postfix**: `household.members { person entitypush person.is_adult entitypop } filter length household.members length ==`

---

### Match forall

**Syntax**: `there is MATCH FORALL arrayExpr TO nexpr IN arrayExpr`
**Semantics**: Tests that every element in one array has a matching entry in another.
**Example (EL)**: `there is match forall required_docs to doc.name in submitted_docs`
**Compiled postfix**: `required_docs submitted_docs doc.name matchforall`

---

### First entity in array where

**Syntax**: `FIRST eexpr IN arrayExpr WHERE bexpr`
**Semantics**: Returns the first element in an array matching a condition, or null.
**Example (EL)**: `first person in household.members where person.is_adult`
**Compiled postfix**: `household.members { person entitypush person.is_adult entitypop } first`

**Tax example**: `first bracket in brackets where bracket.min <= result.taxable_income`
**Eligibility example (real pattern from DTEligibility)**:
```
null 0 household.members { person entitypush person.is_adult
  { person.earned_income 2 index gt { pop pop person person.earned_income } if } if
  entitypop } forall pop household /primary_earner exch def
```

---

## Local Variables

Local variables are declared in context cells and scoped to the current table execution. They are stored on a dedicated stack frame indexed by position (not by name at runtime).

**Syntax**:
```
local long myvar
local long myvar = 5
local double myvar = 3.14
local boolean myvar = true
local string myvar = "hello"
local date myvar = current date
local entity myvar = person
local array myvar
local bigint myvar = 1000000
local bytes myvar
local bytes myvar = 0xdeadbeef
```

**Semantics**: The compiler registers the variable in a local slot table. The declaration emits a `null allocate execute deallocate pop` (or initializer + conversion) sequence. Subsequent references emit `N local@` (read) and `N local!` (write) where N is the slot index (0-based).

| Declaration                     | Compiled postfix                                         |
|---------------------------------|----------------------------------------------------------|
| `local long myvar`              | `null allocate execute deallocate pop` (slot 0)          |
| `local long myvar = 5`          | `5 cvi allocate execute deallocate pop` (slot 0)         |
| `local double myvar = 3.14`     | `3.14 cvr allocate execute deallocate pop` (slot 0)      |
| `local boolean myvar = true`    | `true cvb allocate execute deallocate pop` (slot 0)      |
| `local entity myvar = person`   | `person cve allocate execute deallocate pop` (slot 0)    |
| `local bigint myvar = 1000000`  | `1000000 cvbi allocate execute deallocate pop` (slot 0)  |
| `local bytes myvar`             | `null allocate execute deallocate pop` (slot 0)          |
| `local bytes myvar = 0xdeadbeef` | `0xdeadbeef cvbytes allocate execute deallocate pop` (slot 0) |

After declaration, reads emit `0 local@` and writes emit `0 local!`.

**Tax example (context cell)**:
```
local double running_total = 0.0
```
Postfix: `0.0 cvr allocate execute deallocate pop`

Action referencing it: `running_total + bracket.rate * result.taxable_income`
→ `0 local@ bracket.rate result.taxable_income * +`

---

## Bytes Type

Bytes is an immutable byte-sequence type for opaque binary data.

### Literal

**Syntax**: `HEX_BYTES_LITERAL` — `0x` followed by an even number of hex digits (case-insensitive).  
**Semantics**: Creates an `RBytes` value. The empty literal `0x` is a zero-length byte sequence.

**Example (EL)**: `received_hash is equal to 0xa914748284390f9e263a4b766a75d999`  
**Compiled postfix**: `received_hash 0xa914748284390f9e263a4b766a75d999 cvbytes bytes==`

**Blockchain-policy example**: `received_hash is equal to expected_commitment` where both fields are `type="bytes"` in the EDD.  
**Tax example**: N/A — bytes is not a tax-domain type.

---

### Bytes operations

#### Concatenation

**Syntax**: `bytesexpr + bytesexpr` (both operands must be bytes-typed)  
**Semantics**: Returns the concatenation of the two byte sequences.  
**Compiled postfix**: `... bytes+`

#### Slice

**Syntax**: `bytesexpr from iexpr to iexpr`  
**Semantics**: Returns bytes `[from, to)`. `from` is inclusive, `to` is exclusive. Out-of-range is a runtime error.  
**Compiled postfix**: `<bytes> <from> <to> bytesslice`

#### Length

**Syntax**: `length of bytesexpr`  
**Semantics**: Returns the number of bytes as an integer.  
**Compiled postfix**: `<bytes> byteslen`

#### Indexed access

**Syntax**: `bytesexpr[iexpr]`  
**Semantics**: Returns the byte at index `i` as an integer 0–255. Out-of-range is a runtime error.  
**Compiled postfix**: `<bytes> <i> bytesidx`

#### Equality (constant-time)

**Syntax**: `bytesexpr is equal to bytesexpr`  
**Semantics**: Returns `true` iff both sequences have the same length and content, using `crypto/subtle.ConstantTimeCompare`.  
**Compiled postfix**: `<a> <b> bytes==`

**Syntax**: `bytesexpr is not equal to bytesexpr`  
**Compiled postfix**: `<a> <b> bytes!=`

---

## Grammar Appendix

The complete EL grammar is defined at:
`pkg/dtrules/compiler/el/EL.g4`

The top-level production is `done`, which dispatches based on the cell type prefix:

```antlr
done
    : ACTION statementList optSemi         # actionStatement
    | CONDITION bexpr optSemi              # conditionExpr
    | CONTEXT contextForTable optSemi      # contextStatement
    | POLICYSTATEMENT strexpr optSemi      # policyStrExpr
    | ...
    ;
```

Key production names referenced in this document:

| Production            | Description                          |
|-----------------------|--------------------------------------|
| `bexpr`               | Boolean expression                    |
| `iexpr`               | Integer expression                    |
| `fexpr`               | Float (double) expression             |
| `bigexpr`             | BigInt expression                     |
| `strexpr`             | String expression                     |
| `dexpr`               | Date expression                       |
| `eexpr`               | Entity expression                     |
| `nexpr`               | Name expression                       |
| `arrayExpr`           | Array expression                      |
| `texpr`               | Table expression                      |
| `setstatement`        | Assignment statement                  |
| `ifstatement`         | If/then/else block                    |
| `ifblock`             | If condition + body + continuation    |
| `ifcontinue`          | endif / else / else if continuation   |
| `forallctl`           | For-all loop control                  |
| `forallblock`         | For-all loop body                     |
| `foreachblock`        | For-each loop body                    |
| `firstblock`          | For-first block with else             |
| `forfirstctl`         | For-first loop control                |
| `localvariables`      | Local variable declarations           |
| `addtostatement`      | Array or numeric add                  |
| `subtodest`           | Subtract destination                  |
| `xmlvaluestatements`  | XML attribute statements              |
| `performstatement`    | Decision table invocation             |
| `debugstatement`      | Debug/print output                    |
| `contextForTable`     | Context cell entry point              |
| `contextstatement`    | Add entity to context                 |
| `statementList`       | Sequence of blocks                    |
| `block`               | Single statement or structured block  |
| `statement`           | One statement (with separator)        |
| `separator`           | `;` or `,`                            |
| `usingblock`          | Entity delegation block               |
| `usingstatement`      | Using statement                       |
| `possessiveRef`       | Possessive chain (`entity's field`)   |
| `colonRef`            | Colon-prefixed entity chain           |
| `done`                | Top-level grammar entry point         |
| `optSemi`             | Optional trailing semicolon           |
| `number`              | Integer or float literal              |
| `addtodest`           | Add-to destination                    |
| `includeSearch`       | Array include test target             |
| `blist`               | String list for `== "a", "b" or "c"` |
| `inthe`               | `in` / `for` / `on` preposition       |
| `thereis`             | `there is` / `is there` quantifier    |
| `forctl`              | C-style for loop                      |
| `datestatement`       | Date field arithmetic statement       |
| `randomstatements`    | Array manipulation statements         |
| `operatorstatements`  | Operator invocation                   |
| `operatorlist`        | Operator argument list                |
| `arrayLit`            | Array literal `{ ... }`               |
| `arrayList`           | Array literal element list            |
| `indxExpr`            | Array index expression                |
| `tablelist`           | Table lookup key list                 |
| `leftIexpr`           | Integer left-value (assignment target)|
| `leftFexpr`           | Float left-value                      |
| `leftBexpr`           | Boolean left-value                    |
| `leftEexpr`           | Entity left-value                     |
| `leftStrexpr`         | String left-value                     |
| `leftDexpr`           | Date left-value                       |
| `leftArrayRef`        | Array left-value                      |
| `leftBigexpr`         | BigInt left-value                     |

See `EL.g4` for the complete grammar with all alternatives and lexer rules.
