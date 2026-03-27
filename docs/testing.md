# DTRules Testing Guide

This document describes the testing infrastructure for DTRules across all implementations.

## Overview

DTRules has three main implementations:

1. **Go** - Primary production implementation
2. **Java** - Legacy reference implementation
3. **ASM** - x86-64 assembly implementation (educational/performance research)

Each implementation has its own test suite, plus cross-platform comparison tests to ensure behavioral consistency.

## Quick Start

Run all tests across all platforms:

```bash
./test/run-all-tests.sh
```

Run tests for specific implementations:

```bash
# Go only
./test/run-all-tests.sh --go-only

# ASM only
./test/run-all-tests.sh --asm-only

# Java only
./test/run-all-tests.sh --java-only
```

## Test Locations

| Implementation | Test Location | Command |
|----------------|---------------|---------|
| Go | `go/pkg/dtrules/**/*_test.go` | `cd go && go test ./...` |
| ASM | `asm/test/unit/*.asm` | `cd asm && make test` |
| Java | `dtrules-engine/src/test/java/**/*Test.java` | `mvn test` |
| Cross-platform | `test/vectors/*.json` | `./test/run-all-tests.sh` |

## Go Tests

The Go implementation has comprehensive unit tests covering:

- **Operators** - All 100+ operators (`operators/operators_test.go`)
- **Value types** - Integer, Double, Boolean, String, etc.
- **Session management** - Session lifecycle, state
- **Entity operations** - Entity creation, attribute access
- **Runtime implementations** - GoRuntime, NativeASM, ASMRuntime
- **Bytecode** - Compilation and execution
- **Integration** - End-to-end rule execution

### Running Go Tests

```bash
cd go

# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package
go test -v ./pkg/dtrules/operators/...

# Run with race detection
go test -race ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Go Test Coverage

Current coverage targets:
- Core packages: >70%
- Operators: >80%
- Runtime: >60%

## ASM Tests

The ASM implementation uses a custom test framework written in assembly.

### Test Modules

| Module | Tests | Description |
|--------|-------|-------------|
| test_arithmetic.asm | 11 | +, -, *, /, mod, neg, abs, min, max |
| test_comparison.asm | 6 | ==, !=, <, <=, >, >= |
| test_boolean.asm | 4 | and, or, not, xor |
| test_stack.asm | 9 | push, pop, dup, swap, rot, over, pick, roll, clear |
| test_string.asm | 18 | concat, substring, trim, indexof, etc. |
| test_array.asm | 6 | newarray, addto, length, get, set |
| test_control.asm | 5 | if, ifelse, exec, while, for |
| test_math_ext.asm | 12 | floor, ceil, round, truncate, pow |
| test_value.asm | 5 | Value type creation and conversion |
| test_operators.asm | 4 | Operator dispatch mechanism |
| test_bytecode_load.asm | 2 | Bytecode loading |
| test_compiled_bytecode.asm | 3 | Bytecode compilation |

### Running ASM Tests

```bash
cd asm

# Run all unit tests
make test-unit

# Run all tests (unit + integration)
make test

# Run specific test module
./test/unit/build/test_arithmetic

# Run with verbose output
./test/run_tests.sh
```

### ASM Test Harness

The test harness (`test_harness.asm`) provides:

- `test_start(name)` - Begin a test
- `test_end_pass` - Mark test as passed
- `test_end_fail` - Mark test as failed
- `assert_eq(actual, expected)` - Assert equality
- `reset_state` - Reset VM state between tests
- `print_test_summary` - Print pass/fail counts

## Java Tests

The Java implementation uses JUnit for testing.

### Test Classes

| Class | Tests | Description |
|-------|-------|-------------|
| RIntegerTest | 15 | Integer value operations |
| RDoubleTest | 16 | Double value operations |
| RBooleanTest | 14 | Boolean value operations |
| RStringTest | 15 | String value operations |
| RNameTest | 12 | Name/symbol operations |
| RArrayTest | 12 | Array operations |

### Running Java Tests

```bash
# Run all tests
mvn test

# Run specific test class
mvn test -Dtest=RIntegerTest

# Run with coverage (requires Jacoco plugin)
mvn test jacoco:report
```

## Shared Test Vectors

Test vectors in `test/vectors/` define expected behavior that all implementations should match:

| File | Operators | Tests |
|------|-----------|-------|
| arithmetic.json | +, -, *, /, abs, negate, f+, f-, f*, fdiv, fabs, fnegate | 28 |
| comparison.json | ==, !=, <, >, <=, >= | 24 |
| boolean.json | and, or, not, xor, beq | 17 |
| stack.json | pop, dup, swap, rot, over, pick, roll, clear | 17 |
| string.json | concat, substring, trim, indexof, etc. | 34 |
| array.json | newarray, addto, length, getat, memberof | 19 |
| control.json | if, ifelse, for, while, forall | 12 |
| table.json | newtable, tableget, tableput, tablecontains | 16 |
| entity.json | def, lookup, entitypush, get, createentity | 15 |
| datetime.json | newdate, getyear, adddays, daysbetween | 24 |

### Vector Format

```json
{
  "name": "Test Category",
  "tests": [
    {
      "name": "test_name",
      "operator": "operator_name",
      "inputs": [arg1, arg2, ...],
      "expected": result,
      "type": "result_type"
    }
  ]
}
```

## Comparison Tests

Comparison tests verify that different implementations produce identical results:

```bash
cd asm
make test-comparison
```

These tests:
1. Execute the same bytecode in Go and ASM
2. Compare the resulting stack states
3. Report any differences

## CI/CD Integration

GitHub Actions workflow (`.github/workflows/tests.yml`) runs:

- Go tests on Linux, macOS, Windows with Go 1.21 and 1.22
- Java tests on Linux, macOS, Windows with JDK 11, 17, 21
- ASM tests on Linux (requires NASM)
- Comparison tests (ASM vs Go)
- NativeASM tests
- Performance benchmarks (on main branch only)

### CI Status Checks

All PRs require:
- Go tests pass
- Java tests pass
- Linting passes

Optional (may fail without blocking):
- ASM tests
- Comparison tests

## Writing New Tests

### Go

```go
func TestMyOperator(t *testing.T) {
    state := newTestState(t)

    // Setup
    state.DataPush(value1)
    state.DataPush(value2)

    // Execute
    err := operators.Get(dtrules.GetRName("myop")).Execute(state)
    require.NoError(t, err)

    // Verify
    result, err := state.DataPop()
    require.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

### ASM

```asm
test_my_operator:
    push rbp
    mov rbp, rsp

    lea rdi, [test_name]
    call test_start

    call reset_state

    ; Setup
    mov rdi, value1
    call stack_data_push_integer
    mov rdi, value2
    call stack_data_push_integer

    ; Execute
    call op_myop

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Verify
    call stack_data_pop
    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, expected
    call assert_eq
    test eax, eax
    jz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret
```

### Java

```java
@Test
public void testMyOperator() throws RulesException {
    // Setup
    MyValue v1 = MyValue.create(value1);
    MyValue v2 = MyValue.create(value2);

    // Execute (operator typically done through state)
    MyValue result = v1.operate(v2);

    // Verify
    assertEquals(expected, result.getValue());
}
```

## Debugging Failed Tests

### Go

```bash
# Run specific test with verbose output
go test -v -run TestMyOperator ./pkg/dtrules/...

# Run with debug logging
DEBUG=1 go test -v ./...
```

### ASM

```bash
# Build with debug symbols
make debug

# Run single test with GDB
gdb ./test/unit/build/test_arithmetic
(gdb) break test_add_op
(gdb) run
```

### Java

```bash
# Run with debug output
mvn test -Dtest=MyTest -X
```

## Performance Testing

### Go Benchmarks

```bash
cd go
go test -bench=. -benchmem ./pkg/dtrules/...
```

### ASM Benchmarks

```bash
cd asm
make bench
```

Benchmarks compare:
- Pure Go runtime
- NativeASM runtime (Go with assembly hot paths)
- Full ASM runtime (standalone assembly)

## References

- [Go Implementation Design](go-design-review.md)
- [ASM Guide](asm-guide.md)
- [Bytecode Specification](bytecode-spec.md)
- [NativeASM Runtime](nativeasm-runtime.md)
