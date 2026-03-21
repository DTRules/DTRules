; DTRules NASM VM - Constants and Structures
; Copyright 2024 Paul Snow - Apache License 2.0
;
; This file defines:
; - Bytecode opcodes (matching Go implementation)
; - VM state structure
; - Trace buffer format
; - Value type tags

; ===========================================================================
; Bytecode Opcodes (matching go/pkg/dtrules/bytecode.go)
; ===========================================================================

; Stack operations
%define OP_NOP          0
%define OP_PUSH         1       ; Push constant from pool
%define OP_PUSH_INT     2       ; Push immediate integer (followed by varint)
%define OP_POP          3       ; Pop and discard
%define OP_DUP          4       ; Duplicate top
%define OP_SWAP         5       ; Swap top two
%define OP_ROT          6       ; Rotate top three
%define OP_ROLL         7       ; Roll n items
%define OP_INDEX        8       ; Copy nth item to top
%define OP_CLEAR        9       ; Clear stack to mark
%define OP_MARK         10      ; Push mark

; Arithmetic
%define OP_ADD          20      ; a + b
%define OP_SUB          21      ; a - b
%define OP_MUL          22      ; a * b
%define OP_DIV          23      ; a / b
%define OP_MOD          24      ; a % b
%define OP_NEG          25      ; -a
%define OP_ABS          26      ; |a|
%define OP_INC          27      ; a + 1
%define OP_DEC          28      ; a - 1

; Comparison
%define OP_EQ           30      ; a == b
%define OP_NE           31      ; a != b
%define OP_LT           32      ; a < b
%define OP_LE           33      ; a <= b
%define OP_GT           34      ; a > b
%define OP_GE           35      ; a >= b

; Boolean
%define OP_AND          40      ; a && b
%define OP_OR           41      ; a || b
%define OP_NOT          42      ; !a
%define OP_XOR          43      ; a ^ b

; Control flow
%define OP_EXEC         50      ; Execute top of stack
%define OP_IF           51      ; Conditional execution
%define OP_IFELSE       52      ; if-else
%define OP_WHILE        53      ; while loop
%define OP_FOR          54      ; for loop
%define OP_FORALL       55      ; iterate array
%define OP_RETURN       56      ; Return from procedure
%define OP_JUMP         57      ; Unconditional jump (followed by offset)
%define OP_JUMP_IF      58      ; Conditional jump
%define OP_CALL         59      ; Call operator by index

; Entity operations
%define OP_ENTITY_PUSH  60      ; Push entity onto entity stack
%define OP_ENTITY_POP   61      ; Pop entity from entity stack
%define OP_DEF          62      ; Define variable
%define OP_LOOKUP       63      ; Lookup name in context
%define OP_NEW_ENTITY   64      ; Create new entity

; Array operations
%define OP_NEW_ARRAY    70      ; Create new array
%define OP_ADD_TO       71      ; Add to array
%define OP_LENGTH       72      ; Get array length
%define OP_GET          73      ; Get array element
%define OP_PUT          74      ; Put array element

; Table operations
%define OP_NEW_TABLE    80      ; Create new table
%define OP_TABLE_GET    81      ; Get table value
%define OP_TABLE_PUT    82      ; Put table value

; String operations
%define OP_CONCAT       90      ; String concatenation
%define OP_SUBSTRING    91      ; Substring

; Constants
%define OP_PUSH_TRUE    100     ; Push true
%define OP_PUSH_FALSE   101     ; Push false
%define OP_PUSH_NULL    102     ; Push null
%define OP_PUSH_ZERO    103     ; Push 0
%define OP_PUSH_ONE     104     ; Push 1

; Extended opcodes
%define OP_OPERATOR     200     ; Call operator by index (followed by varint)
%define OP_CONSTANT     201     ; Push constant by index (followed by varint)
%define OP_NAME         202     ; Push name by index (followed by varint)

; Maximum opcode value (for jump table sizing)
%define OP_MAX          255

; ===========================================================================
; Value Type Tags (matching Go Value.tag)
; ===========================================================================

%define VTAG_NULL       0
%define VTAG_BOOLEAN    1
%define VTAG_INTEGER    2
%define VTAG_DOUBLE     3
%define VTAG_STRING     4
%define VTAG_NAME       5
%define VTAG_ARRAY      6
%define VTAG_ENTITY     7
%define VTAG_TABLE      8
%define VTAG_MARK       9
%define VTAG_OBJECT     10      ; Opaque Go object reference

; ===========================================================================
; Trace Event Types
; ===========================================================================

%define TRACE_OPCODE        0x01    ; Opcode execution
%define TRACE_PUSH          0x02    ; Value pushed to stack
%define TRACE_POP           0x03    ; Value popped from stack
%define TRACE_ENTITY_PUSH   0x04    ; Entity pushed to entity stack
%define TRACE_ENTITY_POP    0x05    ; Entity popped from entity stack
%define TRACE_DEF           0x06    ; Variable definition
%define TRACE_LOOKUP        0x07    ; Name lookup result
%define TRACE_CREATE_ENTITY 0x08    ; Entity created
%define TRACE_NEW_ARRAY     0x09    ; Array created
%define TRACE_ADD_TO        0x0A    ; Added to array
%define TRACE_REMOVE        0x0B    ; Removed from array
%define TRACE_DT_ENTER      0x10    ; Enter decision table
%define TRACE_DT_EXIT       0x11    ; Exit decision table
%define TRACE_COLUMN        0x12    ; Decision table column
%define TRACE_CONDITION     0x13    ; Condition evaluated
%define TRACE_ACTION        0x14    ; Action executed
%define TRACE_ERROR         0xFF    ; Error occurred

; ===========================================================================
; VM State Flags
; ===========================================================================

%define VM_FLAG_TRACE       0x01    ; Trace recording enabled
%define VM_FLAG_RUNNING     0x02    ; VM is currently running
%define VM_FLAG_ERROR       0x04    ; Error state
%define VM_FLAG_HALT        0x08    ; Halt requested

; ===========================================================================
; Stack Limits
; ===========================================================================

%define DATA_STACK_SIZE     4096    ; Maximum data stack entries
%define ENTITY_STACK_SIZE   256     ; Maximum entity stack entries
%define CTRL_STACK_SIZE     256     ; Maximum control stack entries
%define TRACE_BUFFER_SIZE   1048576 ; 1MB trace buffer

; ===========================================================================
; Value Structure (16 bytes per value, cache-friendly)
; ===========================================================================
; Layout:
;   +0: tag (8 bytes, aligned)
;   +8: num (8 bytes, integer/double value or pointer)

%define VALUE_SIZE          16
%define VALUE_TAG_OFFSET    0
%define VALUE_NUM_OFFSET    8

; ===========================================================================
; Trace Entry Structure (variable length)
; ===========================================================================
; Each trace entry starts with:
;   +0: event_type (1 byte)
;   +1: timestamp_low (4 bytes) - lower 32 bits of rdtsc
;   +5: payload_len (2 bytes)
;   +7: payload (variable)
;
; Total header: 7 bytes + payload

%define TRACE_ENTRY_TYPE_OFFSET     0
%define TRACE_ENTRY_TIME_OFFSET     1
%define TRACE_ENTRY_LEN_OFFSET      5
%define TRACE_ENTRY_DATA_OFFSET     7
%define TRACE_ENTRY_HEADER_SIZE     7

; ===========================================================================
; VM Context Structure
; ===========================================================================
; This structure holds all VM state and is passed to vm_execute

struc VMContext
    ; Bytecode execution state
    .bytecode:          resq 1      ; Pointer to bytecode array
    .bytecode_len:      resq 1      ; Length of bytecode
    .pc:                resq 1      ; Program counter (offset into bytecode)

    ; Constant pools (from BytecodeChunk)
    .constants:         resq 1      ; Pointer to constant values array
    .constants_count:   resq 1      ; Number of constants
    .names:             resq 1      ; Pointer to names array
    .names_count:       resq 1      ; Number of names

    ; Data stack (values)
    .data_stack:        resq 1      ; Pointer to data stack base
    .data_sp:           resq 1      ; Data stack pointer (index)
    .data_stack_cap:    resq 1      ; Data stack capacity

    ; Entity stack
    .entity_stack:      resq 1      ; Pointer to entity stack base
    .entity_sp:         resq 1      ; Entity stack pointer (index)
    .entity_stack_cap:  resq 1      ; Entity stack capacity

    ; Control stack
    .ctrl_stack:        resq 1      ; Pointer to control stack base
    .ctrl_sp:           resq 1      ; Control stack pointer (index)
    .ctrl_stack_cap:    resq 1      ; Control stack capacity

    ; Trace buffer
    .trace_buffer:      resq 1      ; Pointer to trace buffer
    .trace_pos:         resq 1      ; Current write position in trace buffer
    .trace_cap:         resq 1      ; Trace buffer capacity
    .trace_count:       resq 1      ; Number of trace entries written

    ; Operator table (for OpOperator)
    .operator_table:    resq 1      ; Pointer to operator function pointers
    .operator_count:    resq 1      ; Number of operators

    ; VM state
    .flags:             resq 1      ; VM_FLAG_* bits
    .error_code:        resq 1      ; Error code if VM_FLAG_ERROR set
    .error_pc:          resq 1      ; PC where error occurred

    ; Statistics
    .instructions_executed: resq 1  ; Total opcodes executed
    .trace_bytes_written:   resq 1  ; Total bytes written to trace

    ; Session reference (opaque pointer for Go interop)
    .session:           resq 1      ; Pointer to Go Session object

    ; Alignment padding to 256 bytes for cache efficiency
    .padding:           resb 32
endstruc

; Verify structure size
%if VMContext_size != 256
    %error "VMContext structure size must be 256 bytes for alignment"
%endif
