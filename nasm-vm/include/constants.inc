; constants.inc - DTRules NASM VM constants
; Opcodes, type tags, and system limits

%ifndef CONSTANTS_INC
%define CONSTANTS_INC

;-----------------------------------------------------------------------------
; Value Type Tags (matches Go's value.go)
;-----------------------------------------------------------------------------
VTAG_NULL       equ 0
VTAG_INTEGER    equ 1
VTAG_DOUBLE     equ 2
VTAG_BOOLEAN    equ 3
VTAG_STRING     equ 4
VTAG_NAME       equ 5
VTAG_ARRAY      equ 6
VTAG_ENTITY     equ 7
VTAG_BYTECODE   equ 8
VTAG_DATE       equ 9
VTAG_TABLE      equ 10
VTAG_MARK       equ 11
VTAG_OPERATOR   equ 12

;-----------------------------------------------------------------------------
; Value structure (24 bytes, matching Go)
;-----------------------------------------------------------------------------
VALUE_SIZE      equ 24
VALUE_TAG_OFF   equ 0       ; Tag byte offset
VALUE_PAD_OFF   equ 1       ; Padding (7 bytes)
VALUE_NUM_OFF   equ 8       ; Number (int64 or float64 bits)
VALUE_PTR_OFF   equ 16      ; Pointer for complex types

;-----------------------------------------------------------------------------
; Boolean values
;-----------------------------------------------------------------------------
BOOL_FALSE      equ 0
BOOL_TRUE       equ 1

;-----------------------------------------------------------------------------
; Opcodes (matches Go's bytecode.go)
;-----------------------------------------------------------------------------

; Stack operations (0-10)
OP_NOP          equ 0
OP_PUSH         equ 1       ; Push constant from pool
OP_PUSH_INT     equ 2       ; Push immediate integer
OP_POP          equ 3       ; Pop and discard
OP_DUP          equ 4       ; Duplicate top
OP_SWAP         equ 5       ; Swap top two

; Arithmetic operations (20-28)
OP_ADD          equ 20
OP_SUB          equ 21
OP_MUL          equ 22
OP_DIV          equ 23

; Comparison operations (30-35)
OP_EQ           equ 30
OP_NE           equ 31
OP_LT           equ 32
OP_LE           equ 33
OP_GT           equ 34
OP_GE           equ 35

; Logical operations (40-43)
OP_AND          equ 40
OP_OR           equ 41
OP_NOT          equ 42

; Control flow
OP_HALT         equ 255     ; Stop execution

;-----------------------------------------------------------------------------
; Error codes
;-----------------------------------------------------------------------------
ERR_NONE            equ 0
ERR_STACK_UNDERFLOW equ 1
ERR_STACK_OVERFLOW  equ 2
ERR_TYPE_MISMATCH   equ 3
ERR_INVALID_OPCODE  equ 4
ERR_DIV_BY_ZERO     equ 5

;-----------------------------------------------------------------------------
; Stack limits
;-----------------------------------------------------------------------------
DATA_STACK_SIZE     equ 1024    ; Number of values
STACK_SIZE_BYTES    equ (DATA_STACK_SIZE * VALUE_SIZE)

%endif ; CONSTANTS_INC
