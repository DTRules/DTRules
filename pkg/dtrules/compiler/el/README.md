# EL (Expression Language) Compiler

The EL compiler converts human-readable condition and action descriptions into postfix notation that the DTRules runtime can execute.

> **IMPORTANT**: EL is the required format for all new decision tables. See [Decision Table XML Format](../../../../docs/decision-table-xml-format.md) for the required XML structure. Legacy postfix format is deprecated and should not be used for new projects.

## Why EL is Required

### The Problem: Hand-Coded Postfix

Previously, decision table conditions and actions required hand-coded postfix notation:

```xml
<condition_postfix>
taxpayer.filing_status "SINGLE" streq
taxpayer.income 50000.0 f>
and
</condition_postfix>
```

This approach had several problems:
- **Error-prone**: Stack-based notation is difficult to write correctly
- **Hard to maintain**: Business users cannot read or modify postfix
- **No validation**: Syntax errors only discovered at runtime
- **Inconsistent**: Different authors used different patterns

### The Solution: EL Descriptions

EL allows conditions and actions to be written in a natural, readable syntax:

```xml
<condition_description>taxpayer.filing_status == "SINGLE" AND taxpayer.income > 50000.0</condition_description>
```

The EL compiler automatically generates the correct postfix notation:

```xml
<condition_postfix>
taxpayer.filing_status "SINGLE" streq taxpayer.income 50000.0 f> and
</condition_postfix>
```

## Workflow Integration

### 1. Writing Decision Tables

Authors write condition and action descriptions using EL syntax:

**Conditions:**
```
taxpayer.filing_status == "SINGLE"
taxpayer.income > 50000.0
result.has_nexus is true
client is the parent of ThisClient
```

**Actions:**
```
set result.tax_liability = income * rate
set taxpayer.exemptions = taxpayer.exemptions + 1
perform Calculate_Deductions
```

### 2. Compiling EL to Postfix

The EL compiler processes description fields and generates postfix:

```go
import "github.com/DTRules/DTRules/pkg/dtrules/compiler/el"

compiler := el.NewCompiler()
postfix, err := compiler.CompileCondition("taxpayer.income > 50000.0")
// postfix: "taxpayer.income 50000.0 f>"
```

### 3. Validation Testing

The test suite validates that EL descriptions compile to matching postfix:

```bash
go test ./pkg/dtrules/compiler/el/... -run TestCompileTaxProjects -v
```

Expected output:
```
=== Tax Project Compilation Results ===
Conditions: 56 total, 56 compiled (100.0%), 56 match existing (100.0%)
Actions: 77 total, 77 compiled (100.0%), 77 match existing (100.0%)
```

## EL Syntax Reference

### Comparisons
| EL Syntax | Postfix | Description |
|-----------|---------|-------------|
| `a == b` | `a b eq` | Equality (numeric) |
| `a == "text"` | `a "text" streq` | String equality |
| `a > b` | `a b f>` | Greater than (float) |
| `a >= b` | `a b f>=` | Greater than or equal |
| `a < b` | `a b f<` | Less than |
| `a <= b` | `a b f<=` | Less than or equal |

### Boolean Logic
| EL Syntax | Postfix | Description |
|-----------|---------|-------------|
| `a AND b` | `a b and` | Logical AND |
| `a OR b` | `a b or` | Logical OR |
| `NOT a` | `a not` | Logical NOT |
| `a is true` | `a true beq` | Boolean equality |

### Arithmetic
| EL Syntax | Postfix | Description |
|-----------|---------|-------------|
| `a + b` | `a b add` | Addition |
| `a - b` | `a b sub` | Subtraction |
| `a * b` | `a b fmul` | Multiplication (float) |
| `a / b` | `a b fdiv` | Division (float) |

### Assignments
| EL Syntax | Postfix | Description |
|-----------|---------|-------------|
| `set entity.field = value` | `value /entity.field xdef` | Assignment |
| `perform TableName` | `TableName performaliased` | Execute table |

### Relationships (Advanced)
| EL Syntax | Postfix | Description |
|-----------|---------|-------------|
| `a is the parent of b` | `/source a /target b /type parent relationships findmatch swap pop` | Relationship check |

## Best Practices

1. **Always use EL descriptions** - Never hand-code postfix
2. **Use proper quotes** - Strings require double quotes: `"SINGLE"`
3. **Full entity paths** - Use `taxpayer.income` not just `income`
4. **No shorthand operators** - Use `set X = X + 1` not `X += 1`
5. **Separate statements with semicolons** - `set a = 1; set b = 2`

## File Structure

```
pkg/dtrules/compiler/el/
├── EL.g4                    # ANTLR grammar definition
├── compiler.go              # Main compiler implementation
├── compiler_test.go         # Unit tests
├── edd_loader.go           # Entity Data Dictionary loader
├── el_*.go                 # ANTLR-generated parser files
├── postfix_emitter.go      # Postfix code generation
├── validate_samples_test.go # Sample project validation tests
└── README.md               # This file
```

## Running Tests

```bash
# Run all EL compiler tests
go test ./pkg/dtrules/compiler/el/... -v > /tmp/el-tests.log 2>&1
tail -50 /tmp/el-tests.log

# Run specific validation test
go test ./pkg/dtrules/compiler/el/... -run TestCompileTaxProjects -v

# Run postfix comparison test
go test ./pkg/dtrules/compiler/el/... -run TestPostfixMatchesJava -v
```
