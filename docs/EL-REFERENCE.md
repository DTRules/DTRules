# Expression Language (EL) Reference

Complete reference for the DTRules Expression Language used in decision tables.

## Table of Contents

- [Data Types](#data-types)
- [Operators](#operators)
- [Control Flow](#control-flow)
- [Local Variables](#local-variables)
- [Entity Operations](#entity-operations)
- [Array Operations](#array-operations)
- [String Functions](#string-functions)
- [Date Functions](#date-functions)
- [Math Functions](#math-functions)
- [Type Conversions](#type-conversions)
- [Special Constructs](#special-constructs)

---

## Data Types

EL supports the following data types:

| Type | Description | Example |
|------|-------------|---------|
| `string` | Text values | `"Hello World"` |
| `integer` | Whole numbers (long) | `42`, `-100` |
| `double` | Floating-point numbers | `3.14159`, `-0.5` |
| `boolean` | True/false values | `true`, `false` |
| `date` | Date values | `01/15/2024` |
| `array` | Collections of any type | `[1, 2, 3]` |
| `entity` | Object instances | `client`, `case.applicant` |
| `null` | Undefined/missing value | `null` |

---

## Operators

### Comparison Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `=` or `==` | Equal to | `age = 18` |
| `<>` or `!=` | Not equal to | `status <> "inactive"` |
| `<` | Less than | `income < 50000` |
| `>` | Greater than | `age > 21` |
| `<=` | Less than or equal | `count <= 10` |
| `>=` | Greater than or equal | `score >= 70` |

**Verbose Forms:**
```
value is equal to 100
value is not equal to 100
value is equal to ignore case "TEST"
value is not equal to ignore case "test"
```

### Logical Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `and` | Logical AND | `age > 18 and income > 0` |
| `or` | Logical OR | `status = "A" or status = "B"` |
| `not` | Logical NOT | `not eligible` |

**Alternative syntax:** `&&` for `and`, `||` for `or`

### Arithmetic Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `+` | Addition | `total + tax` |
| `-` | Subtraction | `gross - deductions` |
| `*` | Multiplication | `rate * hours` |
| `/` or `div` | Division | `total / count` |

### String Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `+` | Concatenation | `first_name + " " + last_name` |
| `starts with` | Prefix match | `name starts with "Dr"` |
| `at N starts with` | Position prefix | `code at 0 starts with "A"` |
| `matches` | Regex match | `phone matches "[0-9]{3}-[0-9]{4}"` |

### Collection Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `is in` | Membership test | `status is in ["A", "B", "C"]` |
| `is not in` | Non-membership | `code is not in invalid_codes` |
| `is one of` | Set membership | `type is one of valid_types` |
| `includes` | Array contains | `list includes value` |
| `does not include` | Array excludes | `list does not include value` |

### Entity Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `has a` | Attribute exists | `client has a address` |
| `does not have` | Attribute missing | `client does not have phone` |
| `is type of` | Relationship test | `person is parent of child` |

---

## Control Flow

### If/Else

```
if condition then
    // actions when true
endif

if condition then
    // actions when true
else
    // actions when false
endif

if condition1 then
    // first case
elseif condition2 then
    // second case
else
    // default case
endif
```

### For Loop

```
for i = 0; i < 10; i = i + 1
    // loop body
```

### ForAll (Iterate Array)

Iterate over all elements:
```
forall items
    // process each item
```

With condition:
```
forall items where item.status = "active"
    // process active items only
```

Allow removal during iteration:
```
forall items allowing items to be removed
    if item.expired then
        remove item from items
    endif
```

### ForEach (Named Iterator)

```
foreach client in clients
    // client is bound to each element
    set client.processed = true
```

With nested entity:
```
foreach client and its address in clients
    // access client.address directly as address
```

With condition:
```
foreach client in clients where client.age > 18
    // process adult clients
```

### For First (Find First Match)

```
for first of clients where client.primary = true then
    set primary_client = client
elseifnonearefound
    set primary_client = null
endff
```

---

## Local Variables

Declare local variables within a decision table action:

```
// Entity variable
local entity undefined my_entity
local entity undefined my_entity = some_entity

// Integer variable
local long undefined counter
local long undefined counter = 0

// Double variable
local double undefined rate
local double undefined rate = 0.05

// Boolean variable
local boolean undefined found
local boolean undefined found = false

// Date variable
local date undefined start_date
local date undefined start_date = current_date

// Array variable
local array undefined items
local array undefined items = []

// String variable
local string undefined message
local string undefined message = ""
```

---

## Entity Operations

### Creating Entities

```
// Create new entity
new client entity

// Create with context
set new_client = new client entity
```

### Cloning Entities

```
// Shallow clone
set copy = clone of original_entity

// Deep copy (for arrays)
set copied_array = deep copy of original_array
```

### Entity Context

Switch entity context for attribute access:
```
set value = using client (income.amount * 12)
```

### Attribute Access

```
// Direct access
client.name
client.address.city

// Possessive form
client's name
client's address's city
```

---

## Array Operations

### Creating Arrays

```
// Literal array
[1, 2, 3, 4, 5]
["red", "green", "blue"]

// Empty array
[]

// Array of values
array_of_values {item1, item2, item3}
```

### Modifying Arrays

```
// Add element
add client to approved_list
add value to items

// Add if not duplicate
add if not member client to unique_list

// Remove element
remove client from pending_list
remove 0 element from items    // by index

// Remove with condition
remove each client from clients where client.inactive = true

// Clear array
clear items
```

### Array Functions

```
// Get length
length of items
number of items

// Count with condition
number of clients where client.age > 18

// Copy
copy of items           // shallow copy
deep copy of items      // deep copy

// Access by index
items[0]               // first element
items[length of items - 1]  // last element

// Find first match
first client where client.primary = true

// Sort
sort items in ascending order by name
sort items in descending order by date

// Randomize
randomize items

// Tokenize string to array
tokenize "a,b,c" by ","    // returns ["a", "b", "c"]
```

### Sum and Aggregation

```
// Sum values
sum_of amount in transactions

// Sum with condition
sum_of amount in transactions where transaction.type = "credit"
```

---

## String Functions

| Function | Description | Example |
|----------|-------------|---------|
| `length of str` | String length | `length of name` |
| `trim(str)` | Remove whitespace | `trim(input)` |
| `substring of str from N to M` | Extract substring | `substring of name from 0 to 5` |
| `index_of str1 in str2` | Find position | `index_of "@" in email` |
| `change str to upper_case` | Uppercase | `change name to upper_case` |
| `change str to lower_case` | Lowercase | `change code to lower_case` |
| `string value of X` | Convert to string | `string value of amount` |

---

## Date Functions

### Current Date/Time

```
current_date              // Today's date
get current_timestamp     // Timestamp string
```

### Date Arithmetic

```
// Add time
date + 30 days
date + 6 months
date + 1 years
add 30 days to date
add 6 months to date

// Subtract time
date - 7 days
date minus 1 months
subtract 1 years from date
```

### Date Comparisons

```
date1 is before date2
date1 is after date2
date is between start_date and end_date
```

### Date Parts

```
// Extract components
get yearof date           // Year number
get dayof date            // Day of month

// Period calculations
days from start_date to end_date
months from start_date to end_date
years from start_date to end_date

// Month/year boundaries
first of months of date    // First day of month
end of months of date      // Last day of month
first of years of date     // January 1st

// Days in period
get days in yearof date    // 365 or 366
get days in months for date // 28-31
```

### Parsing Dates

```
date("2024-01-15")
(date) date_string
```

---

## Math Functions

| Function | Description | Example |
|----------|-------------|---------|
| `absolutevalue of N` | Absolute value | `absolutevalue of balance` |
| `N rounded` | Round to integer | `amount rounded` |
| `N rounded to D decimal_places` | Round to precision | `rate rounded to 2 decimal_places` |
| `sum_of field in array` | Sum all values | `sum_of amount in payments` |

### Rounding with Boundary

```
// Round with tie-breaking boundary
amount rounded to 2 decimal_places with_boundary 0.5
```

---

## Type Conversions

### Explicit Conversions

```
// To integer
(long) string_value
(long) double_value

// To double
(double) string_value
(double) integer_value

// To string
string value of integer_value
string value of double_value
string value of boolean_value
string value of date_value

// To date
(date) string_value
date(string_value)

// To boolean
string value of boolean boolean_value
```

### Conversion Operators (Internal)

| Operator | Converts To |
|----------|-------------|
| `cvi` | Integer |
| `cvr` | Double (real) |
| `cvs` | String |
| `cvb` | Boolean |
| `cvd` | Date |
| `cve` | Entity |
| `cvn` | Name |

---

## Special Constructs

### Decision Table Execution

```
// Execute another decision table
perform Validate_Input

// With error handling
perform Main_Logic and onerror add error to context and perform Error_Handler
```

### Using Clause

Execute expression in context of an entity:
```
set total = using order (quantity * unit_price)
```

### Debug Output

```
debug "Processing client: " + client.name
print items
```

### Set Statements

```
// Simple assignment
set client.status = "active"

// Increment/decrement
increment counter
decrement remaining

// Conditional set
if condition then
    set result = value1
else
    set result = value2
endif
```

### Null Handling

```
// Check for null
client.phone is null
client.phone is not null

// Use with condition
if client.address is not null then
    set has_address = true
endif
```

---

## Decision Table Structure

In Excel decision tables, conditions and actions use EL syntax:

### Condition Rows (prefix: `c`)
```
c | client.age >= 18
c | client.income > poverty_level
c | client.citizenship = "US"
```

### Action Rows (prefix: `a`)
```
a | set result.eligible = true
a | set result.reason = "Qualified"
a | add client to approved_list
```

### Column Values

| Value | Meaning |
|-------|---------|
| `Y` | Condition must be true |
| `N` | Condition must be false |
| `-` | Condition doesn't matter (don't care) |
| `X` | Execute this action |
| (blank) | Skip this action |

---

## Best Practices

1. **Use meaningful names** - Entity and attribute names should be self-documenting

2. **Keep conditions simple** - Break complex logic into multiple rows

3. **Use local variables** - For intermediate calculations

4. **Handle nulls explicitly** - Check for null before accessing nested attributes

5. **Use foreach for clarity** - When you need to reference the iterator by name

6. **Document complex expressions** - Use comments in the Excel cells

7. **Test edge cases** - Empty arrays, null values, boundary conditions
