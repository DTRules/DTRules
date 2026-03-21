# Validation Report: NASM VM: Scaffolding

## Overall Status: NEEDS_REVISION

The NASM VM scaffolding is substantially complete with good architecture, but has critical bugs that cause all 24 tests to fail. The bugs are localized and fixable.

## Algorithm Verification

| Example | Spec Result | Calculated | Match? |
|---------|-------------|------------|--------|
| 10 + 32 = 42 | 42 | Test fails (stack depth 0) | NO - data_sp not saved |
| Compare and Branch | [true, true] | Cannot verify | NO - tests fail |
| 47 % 5 = 2 | 2 | Test gets garbage value | NO - same root cause |

### Root Cause Analysis

The fundamental issue is in `vm_entry.asm:122-149`:

```asm
; Global exit point - can be called from op_halt in vm_jump_table.asm
global vm_exit_success
vm_exit_success:
.exit:
    ; Epilogue - restore callee-saved registers
    pop r15
    pop r14
    ...
```

The code that saves `data_sp` back to the context (at line 115-116) is at label `.done` which is **before** `vm_exit_success`. When the dispatcher or any handler jumps to `vm_exit_success`, it skips the save of r14 to `VMContext.data_sp`.

**Bug Location**: `vm_entry.asm:113-131`

**Current (Broken)**:
```asm
.done:
    ; Save current stack pointer back to context
    mov [r12 + VMContext.data_sp], r14

    ; Success - return 0
    xor eax, eax

; Global exit point - can be called from op_halt in vm_jump_table.asm
global vm_exit_success
vm_exit_success:
.exit:
    ; Epilogue...
```

**Fix Required**:
Move the data_sp save into `vm_exit_success` before the epilogue:
```asm
global vm_exit_success
vm_exit_success:
    ; Save current stack pointer back to context
    mov [r12 + VMContext.data_sp], r14
    xor eax, eax
.exit:
    ; Epilogue...
```

### Secondary Bug: data_pop Macro

In `vm_core.asm:123-139`, the `data_pop` macro has a bug:

```asm
%macro data_pop 0
    mov rax, [r12 + VMContext.data_sp]
    sub rax, VALUE_SIZE
    cmp rax, [r12 + VMContext.data_base]
    jl %%underflow

    mov [r12 + VMContext.data_sp], rax
    movzx eax, byte [rax + Value.tag]       ; tag in al (zero-extended to eax)
    mov rbx, [rax + Value.num]              ; BUG: rax was modified!
    jmp %%done
```

Line 131 uses `rax` but `rax` was modified by line 130 to contain the tag value, not the address.

**Note**: The codebase includes a corrected `data_pop_safe` macro that doesn't have this bug. The handlers should use `data_pop_safe` or the buggy `data_pop` macro should be fixed.

## Code Reference Verification

| Reference | Valid? | Notes |
|-----------|--------|-------|
| vm_core.asm | Yes | Macros exist, data_pop has bug |
| vm_state.inc | Yes | Structures correctly defined |
| vm_constants.inc | Yes | All constants present |
| vm_entry.asm | Yes | Entry point exists, has exit bug |
| vm_jump_table.asm | Yes | All handlers present |
| test_harness.c | Yes | 24 tests defined, all failing |
| Makefile | Yes | Builds successfully |

### Build Verification

```
cd nasm-vm && make clean && make
```
**Result**: SUCCESS - builds without errors

### Test Verification

```
cd nasm-vm && make test
```
**Result**: FAIL - 0/24 tests pass

Full test output:
```
DTRules NASM VM Test Suite
==========================

Stack Operations:
  Test: push_int... FAIL: stack depth (expected 1, got 0)
  Test: push_zero/one... FAIL: stack depth (expected 2, got 0)
  Test: push_true/false... FAIL: stack depth (expected 2, got 0)
  Test: push_null... FAIL: stack depth (expected 1, got 0)
  Test: pop... FAIL: stack depth (expected 1, got 0)
  Test: dup... FAIL: stack depth (expected 2, got 0)
  Test: swap... FAIL: stack depth (expected 2, got 0)

Arithmetic Operations:
  Test: add... FAIL: stack depth (expected 1, got 0)
  Test: sub... FAIL: 100 - 58 = 42 (expected 42, got 0)
  Test: mul... FAIL: 6 * 7 = 42 (expected 42, got 0)
  Test: div... FAIL: 84 / 2 = 42 (expected 42, got 0)
  Test: div by zero... FAIL: should error on div by zero (expected 5, got 0)
  Test: mod... FAIL: 47 % 5 = 2 (expected 2, got 4204434)
  Test: neg... FAIL: -42 (expected -42, got 4204434)
  Test: abs... FAIL: abs(-42) = 42 (expected 42, got 4204434)
  Test: inc/dec... FAIL: 41 + 1 = 42 (expected 42, got 4204434)

Comparison Operations:
  Test: eq... FAIL: result should be boolean (expected 3, got 42)
  Test: lt... FAIL: result should be boolean (expected 3, got 216)
  Test: gt... FAIL: 20 > 10 should be true (expected 1, got 140732893771672)

Boolean Operations:
  Test: and... FAIL: result should be boolean (expected 3, got 1)
  Test: or... FAIL: false || true = true (expected 1, got 140732893771672)
  Test: not... FAIL: result should be boolean (expected 3, got 216)

Error Handling:
  Test: stack underflow... FAIL: should error on underflow (expected 2, got 0)
  Test: invalid opcode... FAIL: should error on invalid opcode (expected 3, got 0)

==========================
Results: 0/24 tests passed
```

## Completeness Score: 4/6

| Criteria | Status | Notes |
|----------|--------|-------|
| All steps have INPUT section | PASS | All opcodes specify inputs |
| All steps have OPERATION section | PASS | All opcodes describe operation |
| All steps have OUTPUT section | PASS | All opcodes specify outputs |
| All steps have precision rules | PARTIAL | Arithmetic has precision, others missing |
| At least 2 worked examples | PASS | Two examples provided |
| Edge cases documented | PARTIAL | Main cases covered, some gaps |

### Missing Precision Rules

- String operations: encoding, length limits
- Array operations: maximum size, element types
- Entity operations: reference semantics

### Missing Edge Cases

- Integer MIN_INT64 negation (undefined behavior?)
- Array index negative values
- Name lookup failure behavior

## Ambiguity Issues

### Words Found in Spec That Need Clarification

1. **"usually"** - Not found
2. **"typically"** - Not found
3. **"should"** - Found in error descriptions (acceptable)
4. **"may"** - Not found

### Undefined Terms

1. **"complex types"** - In Value.ptr description, needs enumeration
2. **"opaque session pointer"** - Needs clarification of usage

### Ambiguous Behavior

1. **OP_ROLL / OP_INDEX / OP_CLEAR / OP_MARK**: Marked as "not implemented" but no specification of intended behavior
2. **Control flow operations**: Most marked as not implemented, no specification
3. **vm_error_return stack state**: "Stack state is undefined after error" - should be more specific

## Required Changes

### Critical (Must Fix Before Use)

1. **Fix vm_exit_success to save data_sp**
   - Location: `vm_entry.asm:122`
   - Change: Add `mov [r12 + VMContext.data_sp], r14` before epilogue
   - Impact: All 24 tests will be unblocked

2. **Fix data_pop macro or remove it**
   - Location: `vm_core.asm:123-139`
   - Options:
     - Fix the macro to preserve address in a different register
     - Remove it since `data_pop_safe` exists and is correct
   - Impact: Any handler using `data_pop` is broken

### Important (Should Fix)

3. **Add vm_error_return data_sp save**
   - Location: `vm_entry.asm:141-149`
   - The error path also doesn't save r14 properly
   - Uses `mov [r12 + VMContext.data_sp], r14` but then jumps to exit which pops r14

4. **Verify register allocation in handlers**
   - Several handlers modify r14 via operations that use `mov [r12 + VMContext.data_sp], rdi`
   - Should consistently use r14 for data_sp

### Documentation (Should Add)

5. Add specification for unimplemented operations
6. Add precision rules for all value types
7. Document behavior on integer overflow
8. Document trace buffer format

## Summary

The NASM VM scaffolding demonstrates solid architecture following the Fifth/AISynth pattern:
- Jump table dispatch is correctly implemented
- System V ABI is properly followed
- Value structure and VMContext layout are well-designed
- Comprehensive opcode coverage (even if some not implemented)
- Good test harness with 24 tests

However, a critical bug in the exit path prevents the data stack pointer from being saved back to the context, causing all tests to fail with "stack depth 0" or garbage values.

**Estimated effort to fix**: 1-2 lines of code change in `vm_entry.asm`.

After fixing, the implementation should pass most/all of the 24 tests.
