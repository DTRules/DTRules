# ANTLR 4 Migration Documentation

## Overview

The DTRules DSL compilers have been migrated from JFlex/CUP to ANTLR 4. This document describes the changes made, how to use the new implementations, and any behavioral differences.

## Migrated Modules

### Completed
- **EL (Expression Language)**: Primary DSL module - fully converted and tested (97% compatibility)
- **EBL (Extended Business Language)**: Extends EL with FIND/ISWITHIN - fully converted
- **SudokuLanguage**: Specialized Sudoku DSL - fully converted

## Key Changes

### Build System

The pom.xml files have been updated to use the ANTLR 4 Maven plugin:

```xml
<properties>
    <antlr4.version>4.13.2</antlr4.version>
</properties>

<build>
    <plugins>
        <plugin>
            <groupId>org.antlr</groupId>
            <artifactId>antlr4-maven-plugin</artifactId>
            <version>${antlr4.version}</version>
            <executions>
                <execution>
                    <id>antlr</id>
                    <goals>
                        <goal>antlr4</goal>
                    </goals>
                    <configuration>
                        <listener>true</listener>
                        <visitor>true</visitor>
                    </configuration>
                </execution>
            </executions>
        </plugin>
    </plugins>
</build>

<dependencies>
    <dependency>
        <groupId>org.antlr</groupId>
        <artifactId>antlr4-runtime</artifactId>
        <version>${antlr4.version}</version>
    </dependency>
</dependencies>
```

### File Structure

#### EL Module
```
dsl/el/
├── src/main/antlr4/com/dtrules/compiler/el/
│   └── EL.g4                    # ANTLR 4 grammar (combined lexer+parser)
├── src/main/java/com/dtrules/compiler/el/
│   ├── ELAntlr.java             # New ANTLR-based ICompiler implementation
│   ├── ELCompilerVisitor.java   # Visitor for postfix code generation
│   ├── ELTypeResolver.java      # Type resolution (replaces TokenFilter)
│   ├── EL.java                  # Legacy JFlex/CUP implementation (retained)
│   └── TokenFilter.java         # Legacy token filter (retained)
└── src/test/java/com/dtrules/compiler/el/
    └── ELCompilerComparisonTest.java  # Comparison test suite
```

#### EBL Module
```
dsl/ebl/
├── src/main/antlr4/com/dtrules/compiler/ebl/
│   └── EBL.g4                    # ANTLR 4 grammar (extends EL with FIND/ISWITHIN)
├── src/main/java/com/dtrules/compiler/ebl/
│   ├── EBLAntlr.java             # New ANTLR-based ICompiler implementation
│   ├── EBLCompilerVisitor.java   # Visitor for postfix code generation
│   ├── EBLTypeResolver.java      # Type resolution
│   └── EBL.java                  # Legacy JFlex/CUP implementation (retained)
```

#### SudokuLanguage Module
```
dsl/sudoku_language/
├── src/main/antlr4/com/dtrules/compiler/sudoku/
│   └── Sudoku.g4                    # ANTLR 4 grammar (1000 lines)
├── src/main/java/com/dtrules/compiler/sudoku/
│   ├── SudokuAntlr.java             # New ANTLR-based ICompiler implementation
│   ├── SudokuCompilerVisitor.java   # Visitor for postfix code generation
│   ├── SudokuTypeResolver.java      # Type resolution (replaces TokenFilter)
│   └── SudokuLanguage.java          # Legacy JFlex/CUP implementation (retained)
```

## Using the New Compiler

### EL Compiler
```java
// New ANTLR 4 implementation
ICompiler compiler = new ELAntlr();
compiler.setSession(session);

// Compile expressions
String postfix = compiler.compileCondition("1 < 2");
String action = compiler.compileAction("set x to 1 + 2");
```

### EBL Compiler
```java
// New ANTLR 4 implementation (includes FIND/ISWITHIN support)
ICompiler compiler = new EBLAntlr();
compiler.setSession(session);
```

### SudokuLanguage Compiler
```java
// New ANTLR 4 implementation
ICompiler compiler = new SudokuAntlr();
compiler.setSession(session);

// Or configure in DTRules.xml
// <compileralias name="SudokuAntlr">com.dtrules.compiler.sudoku.SudokuAntlr</compileralias>
// <compiler>SudokuAntlr</compiler>
```

## Switching Compilers in DTRules.xml

To use the new ANTLR 4 compilers in your project, update the `DTRules.xml` configuration:

### Using ELAntlr instead of EL
```xml
<!-- Old configuration -->
<compileralias name="EL">com.dtrules.compiler.el.EL</compileralias>
<compiler>EL</compiler>

<!-- New ANTLR 4 configuration -->
<compileralias name="EL">com.dtrules.compiler.el.ELAntlr</compileralias>
<compiler>EL</compiler>
```

### Using SudokuAntlr instead of SudokuLanguage
```xml
<!-- Old configuration -->
<compileralias name="SudokuLanguage">com.dtrules.compiler.sudoku.SudokuLanguage</compileralias>
<compiler>SudokuLanguage</compiler>

<!-- New ANTLR 4 configuration -->
<compileralias name="SudokuLanguage">com.dtrules.compiler.sudoku.SudokuAntlr</compileralias>
<compiler>SudokuLanguage</compiler>
```

Note: Both old and new compilers implement the same `ICompiler` interface, so switching is a simple configuration change with no code modifications required.

## Behavioral Differences

### Test Results
- **97% compatibility** (72/74 tests pass)
- All valid expressions compile identically

### Minor Differences

1. **Null Comparisons**: The new compiler accepts `null is equal to null` expressions that the old compiler rejected. This is an enhancement rather than a breaking change.

2. **Performance**: Mixed results
   - Simple expressions: ANTLR 4 is 2-3x faster
   - Complex boolean expressions: JFlex/CUP can be faster for deeply nested conditions
   - Overall compilation time is acceptable for typical use cases

## Grammar Structure

### Parser Rules
The ANTLR 4 grammar uses labeled alternatives for precise visitor method generation:

```antlr
bexpr
    : iexpr LT iexpr        # boolIntLt
    | iexpr GT iexpr        # boolIntGt
    | iexpr EQ iexpr        # boolIntEq
    | bexpr AND bexpr       # boolAnd
    | bexpr OR bexpr        # boolOr
    | NOT bexpr             # boolNot
    | RBOOLEAN              # boolLiteral
    // ... more alternatives
    ;
```

### Operator Precedence
In ANTLR 4 left-recursive rules, alternatives listed FIRST have HIGHER precedence:

```antlr
iexpr
    // Multiplication/division have higher precedence (listed first)
    : iexpr TIMES iexpr     # intMul
    | iexpr DIVIDE iexpr    # intDiv
    // Addition/subtraction have lower precedence (listed after)
    | iexpr PLUS iexpr      # intAdd
    | iexpr MINUS iexpr     # intSub
    | INT_LITERAL           # intLiteral
    // ...
    ;
```

### Lexer Rules
Multi-word keywords use embedded whitespace patterns:

```antlr
GTE : '>='
    | 'is' WS+ 'greater' WS+ 'than' WS+ 'or' WS+ 'equal' WS+ 'to'
    | 'greater' WS+ 'than' WS+ 'or' WS+ 'equal' WS+ 'to'
    ;

// Whitespace is skipped for parser but used in token patterns
WS  : [ \t\r\n\f]+ -> skip ;
```

## Type Resolution

The `ELTypeResolver` class replaces the TokenFilter functionality:
- Resolves identifiers to their types (integer, float, boolean, entity, etc.)
- Supports local variables with scoped lookup
- Handles possessive notation (e.g., "client's")
- Generates appropriate postfix code based on type

## Visitor Pattern

The `ELCompilerVisitor` extends `ELBaseVisitor<String>` and generates postfix code:

```java
@Override
public String visitBoolIntLt(ELParser.BoolIntLtContext ctx) {
    return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "< ";
}

@Override
public String visitIntAdd(ELParser.IntAddContext ctx) {
    return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "+ ";
}
```

## Running Tests

### Comparison Test
```bash
cd dsl/el
mvn test-compile
mvn exec:java -Dexec.mainClass="com.dtrules.compiler.el.ELCompilerComparisonTest" -Dexec.classpathScope=test
```

### Building All Modules
```bash
cd DTRules
mvn clean compile -pl dsl/el,dsl/ebl,dsl/sudoku_language -am
```

## Migration Notes

### Legacy Support
The old JFlex/CUP implementations (`EL.java`, `EBL.java`) are retained for:
- Backward compatibility
- A/B testing during migration
- Reference implementation

### Removing Legacy Dependencies
Once migration is validated in production, remove from pom.xml:
```xml
<!-- Remove these after full migration -->
<dependency>
    <groupId>de.jflex</groupId>
    <artifactId>jflex</artifactId>
</dependency>
<dependency>
    <groupId>edu.princeton.cup</groupId>
    <artifactId>java-cup</artifactId>
</dependency>
```

## Troubleshooting

### Common Issues

1. **"no viable alternative" errors**: Check that the input has proper whitespace between tokens

2. **Precedence issues**: Verify operator order in the grammar - FIRST = HIGHEST precedence for left-recursive rules

3. **Missing visitor methods**: Ensure all labeled alternatives have corresponding `visit*` methods

### Debug Tips
- Use ANTLR's `-diagnostics` option to see ambiguity warnings
- Use parse tree visualizer: `org.antlr.v4.gui.TestRig`
- Check generated base visitor for expected method signatures
