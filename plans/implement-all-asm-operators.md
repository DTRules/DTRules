# Plan: Implement All Go Operators in ASM

## Goal

Implement all 183 Go operators in the ASM runtime with **full working implementations** so bytecode compiled by Go executes identically in ASM.

---

## Current State

### Go Operator Registry (183 operators, indices 0-182)

```
  0: newarray       1: addto         2: addat         3: length        4: getat
  5: removeat       6: remove        7: memberof      8: copy          9: first
 10: last          11: copyelements 12: sortarray    13: sortentities 14: add_no_dups
 15: merge         16: randomize    17: intersection 18: intersects   19: addarray
 20: deepcopy      21: tokenize     22: findmatch    23: cleararray   24: and
 25: or            26: not          27: xor          28: >            29: <
 30: >=            31: <=           32: ==           33: !=           34: beq
 35: isnull        36: notnull      37: true         38: false        39: if
 40: ifelse        41: while        42: execute      43: for          44: forr
 45: forall        46: forallr      47: doloop       48: lookup       49: throwexception
 50: forfirst      51: forfirstelse 52: entityforall 53: allocate     54: deallocate
 55: local@        56: local!       57: ignore       58: executetable 59: now
 60: today         61: newdate      62: getyear      63: getmonth     64: getday
 65: adddays       66: addmonths    67: addyears     68: daysbetween  69: monthsbetween
 70: yearsbetween  71: datecmp      72: firstofmonth 73: firstofyear  74: endofmonth
 75: yearof        76: monthof      77: dayof        78: getdaysinyear 79: getdaysinmonth
 80: getdayofmonth 81: d<           82: d>           83: d==          84: d+
 85: d-            86: getdate      87: gettimestamp 88: entitypush   89: entitypop
 90: def           91: InContext    92: newentity    93: entityname   94: entityid
 95: req           96: +            97: -            98: *            99: /
100: abs          101: negate      102: f+          103: f-          104: f*
105: fdiv         106: fabs        107: fnegate     108: roundto     109: pop
110: dup          111: swap        112: over        113: rot         114: pick
115: roll         116: clear       117: null        118: mark        119: arraytomark
120: counttomark  121: cleartomark 122: debug       123: print       124: traceon
125: traceoff     126: setdebug    127: pstack      128: printtos    129: clone
130: >r           131: r>          132: i           133: j           134: k
135: entityfetch  136: get         137: find        138: xdef        139: createentity
140: findcreateentity 141: cvi    142: cvr         143: cvb         144: cve
145: cvn          146: cvd         147: error       148: policystatements
149: concat       150: s+          151: substring   152: trim        153: strtrim
154: lowercase    155: tolowercase 156: uppercase   157: touppercase 158: stringlength
159: strlength    160: indexof     161: startswith  162: endswith    163: contains
164: replace      165: split       166: tostring    167: cvs         168: regexmatch
169: s==          170: s==i        171: s<          172: s>          173: s<=
174: s>=          175: newtable    176: tableget    177: tableput    178: tablekeys
179: tablevalues  180: tablecontains 181: tableremove 182: tablesize
```

### Implemented vs Missing

**Already Implemented (~74):** Core arithmetic, comparisons, boolean ops, basic control flow, basic array/string/entity ops.

**Missing (~109):** See detailed list below.

---

## Assembly Optimization Strategy

### Register Usage
The ASM runtime uses dedicated registers for hot paths:
- `r12` - Data stack pointer (no memory lookup)
- `r13` - Entity stack pointer
- `r14` - Control stack pointer
- `rbx` - Bytecode pointer (preserved across calls)

### Optimization Techniques

1. **Inline Stack Operations**
   - Push/pop directly manipulate r12 without function calls
   - Value copying uses `rep movsq` for 24-byte Values (3 qwords)

2. **Branch Prediction Hints**
   - Use `likely`/`unlikely` macros for conditional branches
   - Common paths (success) fall through, error paths branch

3. **SIMD for Bulk Operations**
   - Array copy: `rep movsq` or AVX `vmovdqu` for aligned data
   - String compare: `repe cmpsb` or SSE4.2 `pcmpistri`
   - Date arithmetic: Scalar (timestamps are single int64)

4. **Zero-Copy Where Possible**
   - Values are pointers to heap; operators pass pointers
   - Array iteration doesn't copy elements
   - Entity lookup returns pointer to attribute value

5. **Inlined Common Operators**
   - Arithmetic (+, -, *, /) inline with no function call
   - Comparisons inline with conditional set instructions
   - Boolean ops inline with bit operations

6. **Cache-Friendly Patterns**
   - Stack grows down (spatial locality)
   - Sequential array access (prefetching works)
   - Entity attributes in contiguous hash table

### Example: Optimized `adddays` vs Naive

**Naive:**
```asm
impl_adddays:
    call stack_data_pop     ; days
    call stack_data_pop     ; date
    call date_add_days      ; compute
    call create_date_value  ; allocate
    call stack_data_push    ; push result
    ret
```

**Optimized:**
```asm
impl_adddays:
    ; Inline pop days
    mov rax, [r12]          ; peek top
    sub r12, VALUE_SIZE     ; pop
    mov rcx, [rax + 8]      ; days (num field)

    ; Inline pop date
    mov rax, [r12]
    sub r12, VALUE_SIZE
    mov rdx, [rax + 8]      ; timestamp

    ; Inline date arithmetic (days * 86400 + timestamp)
    imul rcx, 86400
    add rdx, rcx

    ; Reuse popped Value (avoid allocation)
    mov [rax + 8], rdx

    ; Inline push
    add r12, VALUE_SIZE
    mov [r12], rax
    ret
```

### Performance Targets

| Operation | Go Interpreter | ASM Target | Speedup |
|-----------|---------------|------------|---------|
| Arithmetic | ~30 ns | ~5 ns | 6x |
| Array access | ~50 ns | ~10 ns | 5x |
| Entity lookup | ~100 ns | ~20 ns | 5x |
| Date arithmetic | ~80 ns | ~15 ns | 5x |

---

## Implementation Plan

### Phase 1: Infrastructure (constants, state, helper functions)

#### 1.1 Add Missing Constants
**File: `asm/include/constants.inc`**
```asm
VTAG_DATE       equ 9
VTAG_TABLE      equ 10
VTAG_MARK       equ 11
```

#### 1.2 Add State Fields
**File: `asm/include/state.inc`**
```asm
.error_msg:     resq 1
.trace:         resb 1
.debug_level:   resq 1
```

#### 1.3 Create Return Stack
**File: `asm/src/core/stack_return.asm`** (NEW)

For loop index variables (i, j, k operators):
```asm
section .bss
return_stack:      resq 256
return_stack_ptr:  resq 1

section .text
global stack_return_push, stack_return_pop, stack_return_peek
```

### Phase 2: Array Operations (17 missing)

**File: `asm/src/types/array.asm`** - Add implementations:

| Operator | Stack Effect | Implementation |
|----------|--------------|----------------|
| addat | array elem idx -- | Insert at index, shift elements right |
| removeat | array idx -- elem | Remove at index, shift elements left, return removed |
| remove | array elem -- | Find element, remove first occurrence |
| copy | array -- newarray | Allocate new array, copy all elements |
| copyelements | dest src -- dest | Copy all elements from src to dest |
| sortarray | array -- array | Quicksort by value comparison |
| sortentities | array attr -- array | Quicksort entities by attribute value |
| add_no_dups | array elem -- array | Add only if not already present |
| merge | arr1 arr2 -- arr1 | Append all arr2 elements to arr1 |
| randomize | array -- array | Fisher-Yates shuffle |
| intersection | arr1 arr2 -- arr3 | New array with common elements |
| intersects | arr1 arr2 -- bool | True if any common elements |
| addarray | dest src -- dest | Same as merge |
| deepcopy | array -- newarray | Recursive clone of nested arrays |
| tokenize | str delim -- array | Split string, return array of tokens |
| findmatch | array table -- entity | Find entity matching key/value pairs |
| cleararray | array -- array | Remove all elements, return empty array |

### Phase 3: Control Flow (9 missing)

**File: `asm/src/vm/bytecode.asm`**

| Operator | Stack Effect | Implementation |
|----------|--------------|----------------|
| forr | limit init body -- | Loop from init down to limit |
| forallr | array body -- | Iterate array in reverse order |
| doloop | body -- | Execute body, pop bool, repeat while true |
| throwexception | msg -- | Set error flag, store message |
| forfirst | array cond -- elem/null | Find first element where condition is true |
| forfirstelse | array cond else -- result | forfirst with else clause |
| entityforall | type body -- | Iterate all entities of given type |
| ignore | value -- | Pop and discard (alias for pop) |
| executetable | name -- | Call table_execute_by_name |

### Phase 4: Date Operations (27 missing)

**File: `asm/src/types/date.asm`** (NEW)

Date stored as Unix timestamp (int64 seconds since epoch).

| Operator | Implementation |
|----------|----------------|
| newdate | year month day -- date: Convert to timestamp |
| getyear | date -- year: Extract year from timestamp |
| getmonth | date -- month: Extract month (1-12) |
| getday | date -- day: Extract day of month |
| adddays | date n -- date: Add n*86400 seconds |
| addmonths | date n -- date: Calendar math for months |
| addyears | date n -- date: Calendar math for years |
| daysbetween | d1 d2 -- n: (d2-d1)/86400 |
| monthsbetween | d1 d2 -- n: Calendar month difference |
| yearsbetween | d1 d2 -- n: Calendar year difference |
| datecmp | d1 d2 -- n: -1/0/1 comparison |
| firstofmonth | date -- date: Set day to 1 |
| firstofyear | date -- date: Set month=1, day=1 |
| endofmonth | date -- date: Last day of month |
| yearof | Same as getyear |
| monthof | Same as getmonth |
| dayof | Same as getday |
| getdaysinyear | date -- n: 365 or 366 |
| getdaysinmonth | date -- n: 28-31 |
| getdayofmonth | Same as getday |
| d< | d1 d2 -- bool: timestamp comparison |
| d> | d1 d2 -- bool |
| d== | d1 d2 -- bool |
| d+ | date days -- date: Same as adddays |
| d- | date days/date -- date/int |
| getdate | date -- date: Identity (strip time) |
| gettimestamp | -- int: Current Unix timestamp |

### Phase 5: Entity Operations (6 missing)

**File: `asm/src/types/entity.asm`**

| Operator | Implementation |
|----------|----------------|
| InContext | type -- bool: Check if entity of type is on stack |
| entityname | entity -- name: Get type name |
| entityid | entity -- int: Get unique ID |
| entityfetch | idx -- entity: Get entity at stack index |
| find | table -- entity: Find by attribute criteria |
| findcreateentity | type table -- entity: Find or create |

### Phase 6: Stack/Debug Operations (14 missing)

**File: `asm/src/vm/bytecode.asm`**

| Operator | Implementation |
|----------|----------------|
| arraytomark | mark e1..en -- array: Collect to mark into array |
| counttomark | mark e1..en -- mark e1..en n: Count elements |
| cleartomark | mark e1..en -- : Pop until mark |
| traceon | Set state.trace = 1 |
| traceoff | Set state.trace = 0 |
| setdebug | level -- : Set debug level |
| pstack | Print entire data stack |
| printtos | value -- : Print top of stack |
| clone | value -- value copy: Deep copy |
| >r | value -- R: value: Move to return stack |
| r> | R: value -- value: Move from return stack |
| i | -- n: Return stack[0] (innermost loop index) |
| j | -- n: Return stack[1] (outer loop index) |
| k | -- n: Return stack[2] (outermost loop index) |

### Phase 7: Type Conversion (5 missing)

| Operator | Implementation |
|----------|----------------|
| cvr | value -- double: Convert to real |
| cvn | value -- name: Convert to name |
| cvd | value -- date: Parse string or convert int |
| error | msg -- : Create error condition |
| policystatements | -- array: Get policy statements |

### Phase 8: String Operations (6 missing)

**File: `asm/src/types/string.asm`**

| Operator | Implementation |
|----------|----------------|
| regexmatch | str pattern -- bool: Regex match (basic patterns) |
| s==i | s1 s2 -- bool: Case-insensitive equals |
| s< | s1 s2 -- bool: Lexicographic less than |
| s> | s1 s2 -- bool: Lexicographic greater than |
| s<= | s1 s2 -- bool |
| s>= | s1 s2 -- bool |

### Phase 9: Table Operations (5 missing)

**File: `asm/src/types/table.asm`**

| Operator | Implementation |
|----------|----------------|
| tablekeys | table -- array: Get all keys |
| tablevalues | table -- array: Get all values |
| tablecontains | table key -- bool: Check key exists |
| tableremove | table key -- value: Remove and return |
| tablesize | table -- int: Number of entries |

### Phase 10: Math (1 missing)

| Operator | Implementation |
|----------|----------------|
| roundto | num places -- num: Round to decimal places |

---

## File Changes Summary

| File | Action | Description |
|------|--------|-------------|
| `asm/include/constants.inc` | MODIFY | Add VTAG_DATE, VTAG_TABLE, VTAG_MARK |
| `asm/include/state.inc` | MODIFY | Add error_msg, trace, debug_level |
| `asm/src/types/date.asm` | NEW | Full date arithmetic implementation |
| `asm/src/core/stack_return.asm` | NEW | Return stack for loop indices |
| `asm/src/types/array.asm` | MODIFY | Add 17 array operations |
| `asm/src/types/string.asm` | MODIFY | Add 6 string operations |
| `asm/src/types/table.asm` | MODIFY | Add 5 table operations |
| `asm/src/types/entity.asm` | MODIFY | Add 6 entity operations |
| `asm/src/vm/bytecode.asm` | MODIFY | Add all operator implementations, update dispatch table |

---

## Decision Table Execution

**Yes, this plan enables full decision table execution in ASM.**

The `executetable` operator (index 58) calls `table_execute_by_name()` which uses the existing decision table infrastructure:

- `asm/src/dt/table.asm` - Table registry, `table_execute_by_name()`, `table_execute()`
- `asm/src/dt/cnode.asm` - CNode execution with condition bytecode
- `asm/src/dt/anode.asm` - ANode execution with action bytecode

Flow:
```
executetable "Compute_Eligibility"
    └→ table_execute_by_name()
         └→ table_execute(table_ptr)
              └→ cnode_execute() / anode_execute()
                   └→ vm_execute() for condition/action bytecode
```

---

## Unit Tests

### Phase 11: Unit Tests for Each Category

**File: `asm/test/unit/test_array_ops.asm`** (NEW)
```asm
; Test addat, removeat, remove, copy, copyelements
; Test sortarray, sortentities, add_no_dups, merge
; Test randomize, intersection, intersects, addarray
; Test deepcopy, tokenize, findmatch, cleararray
```

**File: `asm/test/unit/test_control_ops.asm`** (NEW)
```asm
; Test forr, forallr, doloop
; Test forfirst, forfirstelse, entityforall
; Test executetable (with simple decision table)
```

**File: `asm/test/unit/test_date_ops.asm`** (NEW)
```asm
; Test newdate, getyear, getmonth, getday
; Test adddays, addmonths, addyears
; Test daysbetween, monthsbetween, yearsbetween
; Test date comparisons (d<, d>, d==)
; Test firstofmonth, endofmonth, firstofyear
```

**File: `asm/test/unit/test_entity_ops.asm`** (NEW)
```asm
; Test InContext, entityname, entityid
; Test entityfetch, find, findcreateentity
```

**File: `asm/test/unit/test_stack_ops.asm`** (MODIFY)
```asm
; Add tests for: arraytomark, counttomark, cleartomark
; Add tests for: >r, r>, i, j, k
; Add tests for: clone
```

**File: `asm/test/unit/test_string_ops.asm`** (MODIFY)
```asm
; Add tests for: s==i, s<, s>, s<=, s>=
; Add tests for: regexmatch (basic patterns)
```

**File: `asm/test/unit/test_table_ops.asm`** (NEW)
```asm
; Test tablekeys, tablevalues, tablecontains
; Test tableremove, tablesize
```

**File: `asm/test/unit/test_conversion_ops.asm`** (NEW)
```asm
; Test cvr, cvn, cvd
; Test roundto
```

### Test Harness Updates

**File: `asm/test/unit/Makefile`** - Add new test files to build

---

## File Changes Summary (Updated)

| File | Action | Description |
|------|--------|-------------|
| `asm/include/constants.inc` | MODIFY | Add VTAG_DATE, VTAG_TABLE, VTAG_MARK |
| `asm/include/state.inc` | MODIFY | Add error_msg, trace, debug_level |
| `asm/src/types/date.asm` | NEW | Full date arithmetic implementation |
| `asm/src/core/stack_return.asm` | NEW | Return stack for loop indices |
| `asm/src/types/array.asm` | MODIFY | Add 17 array operations |
| `asm/src/types/string.asm` | MODIFY | Add 6 string operations |
| `asm/src/types/table.asm` | MODIFY | Add 5 table operations |
| `asm/src/types/entity.asm` | MODIFY | Add 6 entity operations |
| `asm/src/vm/bytecode.asm` | MODIFY | Add all operator implementations |
| `asm/test/unit/test_array_ops.asm` | NEW | Array operator tests |
| `asm/test/unit/test_control_ops.asm` | NEW | Control flow operator tests |
| `asm/test/unit/test_date_ops.asm` | NEW | Date operator tests |
| `asm/test/unit/test_entity_ops.asm` | NEW | Entity operator tests |
| `asm/test/unit/test_table_ops.asm` | NEW | Table operator tests |
| `asm/test/unit/test_conversion_ops.asm` | NEW | Type conversion tests |
| `asm/test/unit/test_stack_ops.asm` | MODIFY | Add new stack op tests |
| `asm/test/unit/test_string_ops.asm` | MODIFY | Add new string op tests |
| `asm/test/unit/Makefile` | MODIFY | Add new test files |

---

## Verification

```bash
# 1. Build
cd asm && make clean && make

# 2. Run ALL unit tests
cd asm/test/unit && make test

# 3. Run Go integration tests
cd go && go test ./pkg/dtrules/asmruntime/... -v

# 4. Run CHIP tests (full decision table execution)
cd go && go test -run TestCHIP ./internal/integration/
```

---

## Success Criteria

1. `make` builds with no linker errors
2. All 183 operator_table entries point to real implementations (no impl_nop)
3. All unit tests pass (existing + new)
4. CHIP eligibility tests produce identical results via ASM decision table execution
