# Testing Guide

This document explains how to run tests, write new tests, and interpret results for the DTRules Assembly implementation.

## Quick Start

```bash
# Run all tests
make test

# Run only unit tests
make test-unit

# Run comparison tests (requires dtrules-go in PATH)
make test-comparison
```

## Test Categories

### 1. Unit Tests

Location: `test/unit/`

Unit tests verify individual functions and operations at the assembly level. Each test file focuses on a specific component:

| File | Tests |
|------|-------|
| `test_value.asm` | Value type creation, tags, truthy, equality |
| `test_stack.asm` | Stack operations (push, pop, dup, swap, etc.) |
| `test_arithmetic.asm` | Math operators (add, sub, mul, div, etc.) |
| `test_comparison.asm` | Comparison operators (eq, ne, lt, gt, etc.) |
| `test_boolean.asm` | Boolean operators (and, or, not, xor) |

#### Running Unit Tests

```bash
# From asm/ directory
make test-unit

# Or run directly
cd test/unit && make run
```

#### Sample Output

```
=== Value Type Tests ===
Testing integer values... PASS
Testing boolean values... PASS
Testing null values... PASS
Testing truthy evaluation... PASS
Testing value equality... PASS

5 passed, 0 failed, 5 total
```

### 2. Comparison Tests

Location: `test/comparison/`

Comparison tests verify that the Assembly implementation produces the same output as the Go implementation for the same input rules.

| File | Purpose |
|------|---------|
| `runner.sh` | Test execution script |
| `normalize.py` | Output normalization for comparison |
| `rules/*.xml` | Test rule files |
| `output/` | Generated output files |

#### Running Comparison Tests

```bash
# Requires dtrules-go in PATH
make test-comparison

# ASM-only mode (if Go not available)
./test/comparison/runner.sh
```

#### Sample Output

```
DTRules Go vs Assembly Comparison Tests
========================================

Go binary: /usr/local/bin/dtrules-go
ASM binary: ./dtrules-asm

Testing basic_arithmetic... PASS
Testing stack_operations... PASS
Testing boolean_logic... PASS

========================================
Results: 3 passed, 0 failed, 0 skipped

Output files: test/comparison/output/
```

### 3. Integration Tests

Location: `test/integration/`

Integration tests verify complete workflows including XML parsing, decision table loading, and execution. These are shell scripts that exercise the full system.

```bash
make test-integration
```

## Writing New Tests

### Adding a Unit Test

1. Create `test/unit/test_<component>.asm`
2. Follow the existing pattern:

```asm
; test_example.asm - Unit tests for Example
bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state

; Functions to test
extern function_to_test

; Test harness
extern test_start
extern test_end_pass
extern test_end_fail
extern assert_eq
extern print_test_summary
extern reset_state
extern test_count
extern fail_count

section .data
    test_header:    db "=== Example Tests ===", 10, 0
    test_case_1:    db "first test case", 0

section .text
    global test_main

test_main:
    push rbp
    mov rbp, rsp

    lea rdi, [test_header]
    call print_string

    call test_case_1_impl

    call print_test_summary
    mov rax, [fail_count]

    pop rbp
    ret

test_case_1_impl:
    push rbp
    mov rbp, rsp

    lea rdi, [test_case_1]
    call test_start

    call reset_state

    ; Your test code here
    ; ...

    ; Check result
    mov rdi, actual_value
    mov rsi, expected_value
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

3. Add to `test/unit/Makefile`:

```makefile
$(TEST_BUILD_DIR)/test_example.o: test_example.asm | dirs
    $(ASM) $(ASMFLAGS) -o $@ $<

$(TEST_BUILD_DIR)/test_example: $(TEST_BUILD_DIR)/test_example.o $(TEST_BUILD_DIR)/test_harness.o $(COMMON_OBJ)
    $(LD) $(LDFLAGS) -o $@ $^
```

### Adding a Comparison Test

1. Create `test/comparison/rules/test_name.xml`

```xml
<?xml version="1.0" encoding="UTF-8"?>
<ruleset name="test_name">
    <decision_tables>
        <decision_table name="test">
            <actions>
                <action name="test_action">
                    <!-- Your test code -->
                    5 3 add print
                </action>
            </actions>
            <columns>
                <column number="1">
                    <action name="test_action"/>
                </column>
            </columns>
        </decision_table>
    </decision_tables>
</ruleset>
```

2. Run `make test-comparison` - the new file is automatically picked up

## Test Harness API

### Functions

| Function | Purpose | Arguments | Returns |
|----------|---------|-----------|---------|
| `test_start` | Print test name | rdi = string ptr | - |
| `test_end_pass` | Mark test passed | - | - |
| `test_end_fail` | Mark test failed | - | - |
| `test_pass` | Increment pass count | - | - |
| `test_fail` | Increment fail count | - | - |
| `assert_eq` | Assert rdi == rsi | rdi = actual, rsi = expected | 1 if equal, 0 if not |
| `assert_true` | Assert rdi != 0 | rdi = value | 1 if true, 0 if false |
| `assert_false` | Assert rdi == 0 | rdi = value | 1 if false, 0 if true |
| `print_test_summary` | Print final counts | - | - |
| `reset_state` | Reset VM state | - | - |

### Global Variables

| Variable | Type | Purpose |
|----------|------|---------|
| `test_count` | qword | Total assertions |
| `pass_count` | qword | Passed assertions |
| `fail_count` | qword | Failed assertions |

## Interpreting Results

### Unit Test Results

- **PASS**: All assertions in the test succeeded
- **FAIL**: One or more assertions failed
- Exit code equals number of failed tests

### Comparison Test Results

- **PASS**: Normalized ASM output matches normalized Go output
- **PASS (ASM only)**: Go binary not available, ASM ran without error
- **FAIL (output differs)**: Outputs don't match after normalization
- **FAIL (ASM error)**: ASM binary crashed or returned error
- **FAIL (Go error)**: Go binary crashed or returned error

### Debugging Failures

1. **Unit test failure**: Add print statements to trace values
2. **Comparison failure**: Check `test/comparison/output/` for diff files
3. **Crash**: Run with `gdb ./dtrules-asm` and use `bt` for backtrace

## Continuous Integration

The test suite can be integrated into CI:

```yaml
# .github/workflows/test.yml
test:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v2
    - name: Install NASM
      run: sudo apt-get install nasm
    - name: Build
      run: make -C asm
    - name: Test
      run: make -C asm test
```

## Test Coverage

Current coverage by component:

| Component | Unit Tests | Comparison Tests |
|-----------|------------|------------------|
| Value types | ✅ | - |
| Stack operations | ✅ | ✅ |
| Arithmetic | ✅ | ✅ |
| Comparison | ✅ | - |
| Boolean | ✅ | ✅ |
| Strings | ❌ (stubbed) | ❌ |
| Arrays | ❌ (stubbed) | ❌ |
| Entities | ❌ (stubbed) | ❌ |
| Control flow | ❌ (stubbed) | ❌ |
| Decision tables | ❌ (stubbed) | ❌ |

Legend: ✅ = Implemented, ❌ = Not yet implemented
