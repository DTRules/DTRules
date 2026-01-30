# Sudoku - Puzzle Solver

A sample DTRules project demonstrating a custom Domain Specific Language for solving Sudoku puzzles.

## Overview

This project implements a Sudoku puzzle solver using DTRules. Unlike other sample projects that use the standard EL (Expression Language), this project uses a **custom DSL** specifically designed for Sudoku logic, demonstrating how DTRules can be extended with domain-specific languages.

## What This Demonstrates

- **Custom DSL creation** - The SudokuLanguage compiler
- Domain-specific entity modeling (cells, positions, possible values)
- Complex constraint propagation logic
- Iterative rule execution until puzzle is solved
- How to extend DTRules with specialized languages

## Project Structure

```
Sudoku/
├── DTRules.xml                    # Project configuration (uses SudokuLanguage)
├── DecisionTables/
│   └── Sudoku_dt.xls              # Decision tables with solving logic
├── edd/
│   └── Sudoku.xls                 # Entity Definition Document
├── xml/                           # Compiled XML (generated)
├── repository/                    # Packaged rules
├── testfiles/                     # Test puzzles
└── src/main/java/
    └── com/dtrules/samples/sudoku/
        ├── CompileSudoku.java     # Compiles Excel → XML
        ├── TestSudoku.java        # Runs puzzle tests
        └── nativesolution/
            └── Sudoku.java        # Native Java solver for comparison
```

## Entities

| Entity | Description |
|--------|-------------|
| `Cell` | A 3x3 block containing positions |
| `position` | Individual cell (row, column) with value and possible values |
| `possiblevalue` | A candidate value for a position |
| `Puzzle` | The complete puzzle state with cells and solving flags |
| `constants` | Array of possible values (1-9) |

## The SudokuLanguage DSL

This project uses a custom compiler (`com.dtrules.compiler.sudoku.SudokuLanguage`) instead of the standard EL. This demonstrates DTRules' extensibility for domain-specific problems.

The custom DSL provides syntax optimized for:
- Referencing cells by row/column
- Expressing Sudoku constraints (row uniqueness, column uniqueness, block uniqueness)
- Elimination of impossible values
- Detection of solved positions

## Running This Sample

### Prerequisites

Build DTRules from the repository root:
```bash
cd /path/to/DTRules
mvn clean install
```

### Compile Decision Tables

```bash
cd sampleprojects/Sudoku
mvn exec:java -Dexec.mainClass="com.dtrules.samples.sudoku.CompileSudoku"
```

### Run Tests

```bash
mvn exec:java -Dexec.mainClass="com.dtrules.samples.sudoku.TestSudoku"
```

## Configuration

`DTRules.xml`:
```xml
<DTRules>
  <!-- Custom compiler for Sudoku DSL -->
  <compileralias name="SudokuLanguage">
    com.dtrules.compiler.sudoku.SudokuLanguage
  </compileralias>
  <compiler>SudokuLanguage</compiler>

  <RuleSet name="Sudoku" source="file">
    <RuleSetFilePath>/xml</RuleSetFilePath>
    <DTExcelFolder>/DecisionTables/</DTExcelFolder>
    <EDDExcelFolder>/edd/</EDDExcelFolder>
    <Entities        name="Sudoku_edd.xml" />
    <Decisiontables  name="Sudoku_dt.xml"  />
    <Map             name="Sudoku_map.xml" />
  </RuleSet>
</DTRules>
```

## Solving Algorithm

The decision tables implement constraint propagation:

1. **Initialize** - Set up the puzzle with given values
2. **Eliminate** - Remove impossible values from each position based on:
   - Row constraints (no duplicate in row)
   - Column constraints (no duplicate in column)
   - Block constraints (no duplicate in 3x3 block)
3. **Solve** - When only one possible value remains, set it
4. **Iterate** - Repeat until puzzle is solved or no progress

## Native Comparison

The `nativesolution/Sudoku.java` provides a pure Java implementation of the same solving logic, allowing comparison between:
- Rules-based approach (decision tables)
- Imperative approach (Java code)

## Related Projects

- The custom DSL implementation is in `/dsl/sudoku_language/`

## DSL

Uses **SudokuLanguage** (custom DSL) - not the standard EL.
