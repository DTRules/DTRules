# DTRules Test Infrastructure

This directory contains the unified test infrastructure for DTRules, including shared test vectors and cross-platform test orchestration.

## Directory Structure

```
test/
├── vectors/           # Shared test vectors (JSON)
│   ├── arithmetic.json
│   ├── comparison.json
│   ├── boolean.json
│   ├── stack.json
│   ├── string.json
│   ├── array.json
│   ├── control.json
│   ├── table.json
│   ├── entity.json
│   └── datetime.json
├── run-all-tests.sh   # Cross-platform test runner
└── readme.md          # This file
```

## Quick Start

Run all tests across all implementations:

```bash
./test/run-all-tests.sh
```

Run specific implementations:

```bash
./test/run-all-tests.sh --go-only      # Only Go tests
./test/run-all-tests.sh --asm-only     # Only ASM tests
./test/run-all-tests.sh --java-only    # Only Java tests
```

Skip specific implementations:

```bash
./test/run-all-tests.sh --no-java      # Skip Java tests
./test/run-all-tests.sh --no-asm       # Skip ASM tests
```

Verbose output:

```bash
./test/run-all-tests.sh --verbose
```

Generate JUnit XML report:

```bash
./test/run-all-tests.sh --junit results.xml
```

## Test Vectors

Test vectors are JSON files that define expected behavior for operators across all implementations. Each implementation can read these vectors and verify its operators produce the same results.

### Vector Format

Each test vector file contains:

```json
{
  "name": "Category Name",
  "description": "Description of test category",
  "tests": [
    {
      "name": "test_name",
      "description": "What this test verifies",
      "operator": "operator_name",
      "inputs": [...],
      "expected": ...,
      "type": "result_type"
    }
  ]
}
```

### Types

- `integer` - Integer result
- `double` - Floating point result
- `boolean` - True/false result
- `string` - String result
- `array` - Array result
- `null` - Null result
- `stack` - Stack manipulation (uses `stack_before` and `stack_after`)
- `control` - Control flow tests
- `entity` - Entity operations
- `table` - Hash table operations
- `date` - Date/time operations

### Stack Tests

Stack operation tests use a different format:

```json
{
  "name": "swap_two",
  "description": "Swap top two elements",
  "operator": "swap",
  "stack_before": [1, 2],    // Stack state before (bottom to top)
  "stack_after": [2, 1],     // Expected stack state after
  "type": "stack"
}
```

## Implementation-Specific Tests

### Java Tests

Location: `dtrules-engine/src/test/java/`

Run with Maven:
```bash
mvn test
```

### Go Tests

Location: `go/pkg/dtrules/**/*_test.go`

Run with Go:
```bash
cd go && go test ./...
```

### ASM Tests

Location: `asm/test/`

Run with Make:
```bash
cd asm && make test
```

## Comparison Tests

Comparison tests verify that different implementations produce identical results for the same inputs.

Location: `asm/test/comparison/`

Run:
```bash
cd asm && make test-comparison
```

## Adding New Tests

1. Add test cases to the appropriate vector file in `test/vectors/`
2. Implement the operator in each platform
3. Update platform-specific tests to read from vectors (optional)
4. Run `./test/run-all-tests.sh` to verify

## CI/CD Integration

For CI/CD systems, use the JUnit output:

```bash
./test/run-all-tests.sh --junit test-results.xml
```

GitHub Actions example:

```yaml
- name: Run tests
  run: ./test/run-all-tests.sh --junit test-results.xml

- name: Upload test results
  uses: actions/upload-artifact@v3
  with:
    name: test-results
    path: test-results.xml
```

## Coverage

### Operator Categories

| Category | Operators | Tests |
|----------|-----------|-------|
| Arithmetic | +, -, *, /, abs, negate, f+, f-, f*, fdiv, fabs, fnegate | 28 |
| Comparison | ==, !=, <, >, <=, >= | 24 |
| Boolean | and, or, not, xor, beq | 17 |
| Stack | pop, dup, swap, rot, over, pick, roll, clear | 17 |
| String | concat, substring, trim, lowercase, uppercase, indexof, etc. | 34 |
| Array | newarray, addto, length, getat, removeat, memberof, etc. | 19 |
| Control | if, ifelse, for, while, forall | 12 |
| Table | newtable, tableget, tableput, tablecontains, tableremove | 16 |
| Entity | def, lookup, entitypush, entitypop, get, createentity | 15 |
| DateTime | newdate, getyear, adddays, addmonths, daysbetween, etc. | 24 |

### Platform Status

| Platform | Arithmetic | Comparison | Boolean | Stack | String | Array | Control | Table | Entity | DateTime |
|----------|------------|------------|---------|-------|--------|-------|---------|-------|--------|----------|
| Go       | Complete   | Complete   | Complete| Complete | Complete | Complete | Complete | Complete | Complete | Complete |
| ASM      | Partial    | Partial    | Partial | Complete | Pending | Pending | Pending | Pending | Pending | Pending |
| Java     | Complete   | Complete   | Complete| Complete | Complete | Complete | Complete | Complete | Complete | Complete |
