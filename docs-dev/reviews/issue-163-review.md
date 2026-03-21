# Review Report: NASM VM Stack Operations

**Issue**: #163
**Date**: 2026-03-21
**Reviewer**: AI Review Agent

## Decision: APPROVED

The specification is comprehensive and well-structured. The implementation is solid with 45/45 tests passing. Critical comment inconsistencies have been fixed during this review.

## Fresh Eyes Test

### Points of Confusion

1. **Stack growth direction inconsistency**: The vm_core.asm header comment (line 11) states "grows downward" but the actual implementation grows upward (higher addresses). The specification and research documents correctly state "grows upward", but the code comment contradicts this. An AI implementing from scratch might be confused by this contradiction.

2. **OP_POP vs OP_DROP**: Both operations are documented (POP at opcode 3, DROP at opcode 107) but they do the exact same thing. The spec doesn't clearly state this is intentional aliasing - an AI might wonder if there's a subtle difference.

3. **OP_INDEX input requirements**: The spec states "n+2 elements on stack minimum" but the worked examples show:
   - Before: `[100, 200, 300, 1]` (4 elements, picking index 1)
   - The formula n+2=3 would require 3 elements, but we have 4 total (3 + n)

   This is actually correct behavior but the wording is confusing. After popping n, you need n+1 elements remaining.

4. **ROLL placeholder status**: The spec marks ROLL as "NOT FULLY IMPLEMENTED" on line 224, but this is easy to miss. An AI might attempt to use it.

### Unstated Assumptions

1. **Varint encoding is unsigned-only**: The spec mentions "Sign extension not implemented (unsigned only)" for PUSH_INT, but doesn't explain how to encode negative integers. This is a known limitation documented in research but not prominent in spec.

2. **All operations assume integer type**: Arithmetic and comparison operations work on the value field without checking the tag. This works because tests use integers, but mixing types (boolean + integer) has undefined behavior.

3. **CLEAR ignores marks**: The spec notes "Does not clear to mark (clears entire stack)" but an AI familiar with PostScript might expect mark-based clearing.

4. **Context pointer (rbx) usage**: The spec mentions rbx holds context pointer but doesn't explain it's needed for constant pool access (which is not implemented).

## Alternative Interpretations

| Step | Could Be Misread As | Clarification Needed |
|------|---------------------|---------------------|
| ROT `(a b c -- b c a)` | "c goes to the bottom" (wrong: a goes to top) | Add: "Third element moves to TOS, others shift down" |
| PICK n=0 | "Pick index 0 copies TOS" - but TOS was just popped (it's n) | Clarify: "n=0 copies what was second element before n was popped" |
| SWAP stores at -16 and -32 | Could confuse which is "first" vs "second" | The spec is clear, just complex to visualize |
| Stack "grows upward" | Might think r12 points to TOS value | Clarify: r12 points to next free slot (one past TOS) |
| Underflow check `r12 == r13` | Might think this is "equals base" | Correct, but add "empty stack means r12 at base" |

## Known Pitfalls Coverage

### Addressed in Specification
- **Underflow/overflow errors**: Clearly documented with error codes
- **Division by zero**: Error handling documented
- **Value representation (16 bytes)**: Well documented
- **Jump table dispatch**: Documented

### Not Addressed / Missing
- **No error log file exists**: `docs-dev/errors/error-log.md` referenced in review process but doesn't exist
- **ROLL not implemented**: Noted but not prominently
- **Signed integers not supported**: Mentioned but implications not clear
- **Type checking**: No validation that operands match expected types
- **Mixed-type operations**: Undefined behavior not documented

## Code Consistency Issues

| Issue | Spec Says | Code Does | Impact |
|-------|-----------|-----------|--------|
| Stack direction comment | "grows upward" | Comment says "downward", code grows upward | **HIGH** - comment is wrong |
| OP_CLEAR behavior | "Clear stack to mark" (opcodes.inc:17) | Clears entire stack | **MEDIUM** - code comment misleading |
| Register r12 in vm_core.asm | Line 11: "grows downward" | Incremented on push (upward) | **HIGH** - must fix comment |

### Code Comment Errors Found

**vm_core.asm:11** - Currently reads:
```nasm
;   r12 - data stack pointer (grows downward)
```
Should read:
```nasm
;   r12 - data stack pointer (grows upward)
```

**opcodes.inc:17** - Currently reads:
```nasm
%define OP_CLEAR    9       ; Clear stack to mark
```
Should read:
```nasm
%define OP_CLEAR    9       ; Clear entire stack
```

## Final Checklist

- [x] All examples verified (45/45 tests pass)
- [x] Implementation follows Fifth/AISynth architecture (jump table, single entry/exit)
- [x] Register conventions documented and correct
- [x] Error handling complete for implemented operations
- [x] Self-contained (comments fixed during review)
- [x] No high-risk ambiguities (stack direction comment fixed)
- [x] Ready for human review

## Required Changes Before Approval

### Critical (Fixed During Review)

1. **Fixed vm_core.asm line 11**: Changed "grows downward" to "grows upward" to match actual implementation. ✓

2. **Fixed opcodes.inc line 17**: Changed "Clear stack to mark" to "Clear entire stack" to match actual behavior (mark-based clearing not implemented). ✓

### Recommended (Should Fix)

3. **Add ROLL warning to spec**: Add a prominent note that ROLL (opcode 7) is a placeholder that only pops n without rolling.

4. **Clarify PICK semantics**: Add note that after popping n, the operation copies the nth element from the new TOS (n=0 copies what was the second element before n was popped).

5. **Document signed integer limitation**: Add a note to PUSH_INT that only non-negative integers work correctly, and explain varint encoding limits.

### Optional (Nice to Have)

6. **Add error log**: Create `docs-dev/errors/error-log.md` to track known issues and their resolutions.

7. **Document undefined behaviors**: Add a section on what happens with type mismatches (e.g., AND on integers vs booleans).

## Build Verification

```
$ cd nasm-vm && make clean && make
rm -rf build
mkdir -p build
nasm -f elf64 -g -F dwarf -o build/vm_core.o src/vm_core.asm
gcc -Wall -Wextra -g -O2 -Isrc -c -o build/vm_test.o src/vm_test.c
gcc -no-pie -o build/vm_test build/vm_core.o build/vm_test.o

$ make test
./build/vm_test
DTRules NASM VM Tests
=====================
[45 tests...]
Results: 45/45 tests passed
```

## Conclusion

The implementation is functionally correct with all 45 tests passing. However, the documentation contains critical inconsistencies that would confuse another AI attempting to implement or modify this code. The stack growth direction comment in vm_core.asm is particularly dangerous as it directly contradicts the actual implementation.

The two critical comment fixes have been applied during this review. The specification and implementation are now ready for human review and approval.
