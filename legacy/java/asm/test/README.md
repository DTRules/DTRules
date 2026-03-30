# DTRules Assembly Test Suite

Comprehensive test suite for verifying the x86-64 assembly implementation of DTRules.

## Quick Start

```bash
# From the asm/ directory
make test          # Run unit tests
make test-all      # Run all tests
make test-comparison  # Compare with Go implementation
```

## Directory Structure

```
test/
├── run_tests.sh          # Main test orchestrator
├── README.md             # This file
│
├── unit/                 # Assembly unit tests
│   ├── Makefile          # Build system for unit tests
│   ├── test_harness.asm  # Common test infrastructure
│   ├── test_value.asm    # Value type tests
│   ├── test_stack.asm    # Stack operation tests
│   ├── test_arithmetic.asm  # Math operator tests
│   ├── test_comparison.asm  # Comparison operator tests
│   ├── test_boolean.asm  # Boolean operator tests
│   └── build/            # Compiled test executables
│
├── integration/          # Full system tests
│   └── (shell scripts)
│
└── comparison/           # Go vs ASM comparison
    ├── runner.sh         # Comparison test runner
    ├── normalize.py      # Output normalization
    ├── rules/            # Test XML files
    │   ├── basic_arithmetic.xml
    │   ├── stack_operations.xml
    │   └── boolean_logic.xml
    └── output/           # Test output (gitignored)
```

## Test Categories

### Unit Tests

Low-level tests that verify individual assembly functions:

| Test File | Coverage |
|-----------|----------|
| `test_value.asm` | Value creation, tags, truthy evaluation, equality |
| `test_stack.asm` | push, pop, dup, swap, rot, over, pick, roll, clear |
| `test_arithmetic.asm` | add, sub, mul, div, mod, neg, abs, min, max |
| `test_comparison.asm` | eq, ne, lt, le, gt, ge |
| `test_boolean.asm` | and, or, not, xor (all truth table combinations) |

Run with:
```bash
make test-unit
# or
cd test/unit && make run
```

### Comparison Tests

Verify that ASM produces same output as Go for identical rules:

```bash
# Requires dtrules-go in PATH
make test-comparison

# ASM-only smoke test
./test/comparison/runner.sh
```

Output normalization handles:
- Floating point precision differences
- Error message format differences
- Timestamp removal
- Whitespace normalization

### Integration Tests

End-to-end tests exercising the full pipeline:

```bash
make test-integration
```

## Writing Tests

### Unit Test Template

```asm
; test_example.asm
bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state
extern function_to_test
extern test_start, test_end_pass, test_end_fail
extern assert_eq, print_test_summary, reset_state
extern test_count, fail_count

section .data
    test_header: db "=== Example Tests ===", 10, 0
    test_name:   db "example case", 0

section .text
    global test_main

test_main:
    push rbp
    mov rbp, rsp

    lea rdi, [test_header]
    call print_string

    call test_example_case

    call print_test_summary
    mov rax, [fail_count]
    pop rbp
    ret

test_example_case:
    lea rdi, [test_name]
    call test_start
    call reset_state

    ; Test code here...
    mov rdi, actual
    mov rsi, expected
    call assert_eq
    test eax, eax
    jz .fail

    call test_end_pass
    ret
.fail:
    call test_end_fail
    ret
```

### Comparison Test Template

```xml
<?xml version="1.0" encoding="UTF-8"?>
<ruleset name="test_name">
    <decision_tables>
        <decision_table name="test">
            <actions>
                <action name="test_action">
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

## Test Harness API

### Assertion Functions

| Function | Purpose | Returns |
|----------|---------|---------|
| `assert_eq(rdi, rsi)` | Assert rdi == rsi | 1 if true |
| `assert_true(rdi)` | Assert rdi != 0 | 1 if true |
| `assert_false(rdi)` | Assert rdi == 0 | 1 if true |

### Utility Functions

| Function | Purpose |
|----------|---------|
| `test_start(rdi)` | Print test name (rdi = string ptr) |
| `test_end_pass()` | Mark test passed, print "PASS" |
| `test_end_fail()` | Mark test failed, print "FAIL" |
| `reset_state()` | Reset VM state for next test |
| `print_test_summary()` | Print pass/fail counts |

## Exit Codes

- **0**: All tests passed
- **Non-zero**: Number of failed tests

## CI Integration

```yaml
test:
  script:
    - make -C asm
    - make -C asm test
```

## Troubleshooting

### "dtrules-asm not found"
Run `make` in the asm/ directory first.

### Unit tests won't build
Check that all source files exist and NASM is installed:
```bash
nasm --version
make -C test/unit clean all
```

### Comparison tests all skip
The Go binary `dtrules-go` must be in your PATH. Tests will still run in ASM-only smoke test mode.

### Test output differs
Check `test/comparison/output/` for detailed diff files. Minor floating point differences are normalized.

## Coverage Status

| Component | Unit | Comparison |
|-----------|------|------------|
| Value types | ✅ | - |
| Stack ops | ✅ | ✅ |
| Arithmetic | ✅ | ✅ |
| Comparison | ✅ | - |
| Boolean | ✅ | ✅ |
| Strings | ❌ | ❌ |
| Arrays | ❌ | ❌ |
| Entities | ❌ | ❌ |
| Control flow | ❌ | ❌ |

✅ = Complete, ❌ = Pending implementation
