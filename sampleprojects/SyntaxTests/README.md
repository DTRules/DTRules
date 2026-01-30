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
├── DTRules.xml                        # Project configuration
├── DecisionTables/
│   ├── SyntaxExampleDTs.xls           # Main syntax examples
│   ├── RunTimeTests.xls               # Runtime feature tests
│   └── PSSyntaxExampleDTs.xls         # PowerScript syntax examples
├── edd/
│   └── SampleSyntaxEDD.xls            # Entity definitions for testing
├── xml/                               # Compiled XML (generated)
├── testfiles/                         # Test input files
├── EL Documentation-1.odt             # EL language reference document
└── src/main/java/
    └── com/dtrules/samples/sampleproject2/
        ├── CompileSyntaxExamples.java # Compiles Excel → XML
        └── TestSyntaxExamples.java    # Runs tests
```

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

### Prerequisites

Build DTRules from the repository root:
```bash
cd /path/to/DTRules
mvn clean install
```

### Compile Decision Tables

```bash
cd sampleprojects/SyntaxTests
mvn exec:java -Dexec.mainClass="com.dtrules.samples.sampleproject2.CompileSyntaxExamples"
```

### Run Tests

```bash
mvn exec:java -Dexec.mainClass="com.dtrules.samples.sampleproject2.TestSyntaxExamples"
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
