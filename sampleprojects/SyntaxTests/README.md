# SyntaxTests - Expression Language Feature Examples

A comprehensive sample project demonstrating all features of the EL (Expression Language).

## Overview

This project serves as both a test suite and reference implementation for the DTRules Expression Language. It contains decision tables that exercise every language feature, making it the best resource for learning EL syntax.

## What This Demonstrates

- All EL data types (string, integer, double, date, boolean, arrays, entities)
- Comparison and logical operators
- Arithmetic operations
- Array/collection manipulation
- Entity references and navigation
- Complex nested expressions
- Date handling
- String operations

## Project Structure

```
SyntaxTests/
├── DTRules.xml                        # Project marker; <entry> names the start table
├── DecisionTables/
│   ├── SyntaxExampleDTs.xls           # Main syntax examples
│   ├── RunTimeTests.xls               # Runtime feature tests
│   └── PSSyntaxExampleDTs.xls         # PowerScript syntax examples
├── edd/
│   └── SampleSyntaxEDD.xls            # Entity definitions for testing
├── xml/                               # The rules: EDD, decision tables, mapping
├── testfiles/test.xml                 # Test input: one job, two cases, two clients each
└── EL Documentation-1.odt             # EL language reference document
```

The `.xls` files are the historical authoring source. `xml/` is what the engine
reads and what the authoring API writes.

## Entities

The EDD defines entities specifically for testing all data types:

| Entity | Purpose |
|--------|---------|
| `case` | Container with various field types |
| `client` | Complex entity with dates, arrays, nested references |
| `address` | Demonstrates entity relationships |
| `income` | Numeric field testing |
| `job` | Array containment testing |
| `thing` | Simple entity for collection operations |

## Decision Tables

### SyntaxExampleDTs.xls
Main examples covering:
- Variable declarations and assignments
- Conditional expressions
- Arithmetic operations
- String manipulation
- Date comparisons

### RunTimeTests.xls
Tests for:
- Runtime type checking
- Error handling
- Edge cases

### PSSyntaxExampleDTs.xls
PowerScript syntax variations and extensions.

## Running This Sample

From the repository root:

```bash
make build
build/dtrules run ./sampleprojects/SyntaxTests \
  --input sampleprojects/SyntaxTests/testfiles/test.xml \
  --trace /tmp/syntax-trace.xml
```

The entry table is `Run_Syntax_Examples`, declared in `DTRules.xml`. It iterates
`job.cases` and performs the tables that execute cleanly.

## What executes, and what does not

This is a **syntax catalogue**, not an application. Every row compiles from EL —
there is no hand-written postfix anywhere in the project — but not every table
runs to completion against the test data, and that is expected.

Three tables execute clean and are performed by the entry table:

| Table | Demonstrates |
|---|---|
| `Syntax_Examples_5` | Date and string operators, comparison phrasings, the `does` / `is` / `was` prefixes |
| `Run_Test_15` | Context locals declared in one row and read by the rows beneath it |
| `Error_Handling_Table` | The `perform ... and on error ...` path |

The rest compile and load but do not execute, for three reasons:

- **Entity scope.** Most tables open `for all clients`, but `clients` is a field
  of `case`, and only `job` and `constants` are pushed at load. Several rows go
  further and reference an `address` field (`street`, `city`) while iterating
  cases or clients — broken against any data, not just this input. The entry
  table supplies the missing `case` scope for the tables it performs.
- **Forms the compiler does not implement.** `attribute <name> of <entity>`
  compiles to a deliberate error stub, so any table containing it stops there.
  Likewise `perform $<name>`: it parses but has no emitter, so those two rows
  are commented out and point at the supported `perform table named (...)`.
- **Uninitialised context locals.** `local boolean Test` with no initialiser,
  then `does test == true ?`, puts a null where a boolean is required.

`TestSyntaxTestsExecuteEachTable` pins the executable set;
`TestSampleProjectsProduceLoadableTraces` pins that the project runs and leaves
a trace with real fired columns.

## Editing the rules

Never hand-edit `xml/`. Use the authoring API, which writes the DSL, compiles
the postfix, and keeps the two in step:

```bash
build/dtrules table get Syntax_Examples_5 --project sampleprojects/SyntaxTests
build/dtrules table put Syntax_Examples_5 --project sampleprojects/SyntaxTests < table.json
```

## EL Quick Reference

### Data Types
```
string      "Hello World"
integer     42
double      3.14159
boolean     true, false
date        01/15/2024
array       [1, 2, 3]
entity      client, case.client
```

### Operators
```
Comparison: =, <>, <, >, <=, >=
Logical:    and, or, not
Arithmetic: +, -, *, /
Collection: is in, is not in, add to, remove from
```

### Conditions (Examples)
```
client.age < 18
income.amount >= poverty_level * 2
applicant.status is in ["active", "pending"]
client.name <> ""
```

### Actions (Examples)
```
result.eligible = true
result.reason = "Approved"
add client to approved_list
remove client from pending_list
counter = counter + 1
```

## Documentation

The file `EL Documentation-1.odt` contains comprehensive EL language documentation including:
- Complete syntax reference
- All operators with examples
- Data type details
- Best practices

## Use as Reference

When implementing your own decision tables, refer to this project for:
1. Correct syntax patterns
2. Complex expression examples
3. Entity relationship modeling
4. Test case structure

## DSL

Uses **EL (Expression Language)** - the standard DTRules DSL.
