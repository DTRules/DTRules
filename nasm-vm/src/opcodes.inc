; opcodes.inc - DTRules bytecode opcode definitions
; Copyright 2024 Paul Snow
; Licensed under Apache License 2.0
;
; Mirrors the Go bytecode.go definitions

; Stack operations
%define OP_NOP      0
%define OP_PUSH     1       ; Push constant from pool
%define OP_PUSH_INT 2       ; Push immediate integer (followed by varint)
%define OP_POP      3       ; Pop and discard
%define OP_DUP      4       ; Duplicate top
%define OP_SWAP     5       ; Swap top two
%define OP_ROT      6       ; Rotate top three (a b c -- b c a)
%define OP_ROLL     7       ; Roll n items
%define OP_INDEX    8       ; Copy nth item to top (pick)
%define OP_CLEAR    9       ; Clear stack to mark
%define OP_MARK     10      ; Push mark

; Arithmetic
%define OP_ADD      20      ; a + b
%define OP_SUB      21      ; a - b
%define OP_MUL      22      ; a * b
%define OP_DIV      23      ; a / b
%define OP_MOD      24      ; a % b
%define OP_NEG      25      ; -a
%define OP_ABS      26      ; |a|
%define OP_INC      27      ; a + 1
%define OP_DEC      28      ; a - 1

; Comparison
%define OP_EQ       30      ; a == b
%define OP_NE       31      ; a != b
%define OP_LT       32      ; a < b
%define OP_LE       33      ; a <= b
%define OP_GT       34      ; a > b
%define OP_GE       35      ; a >= b

; Boolean
%define OP_AND      40      ; a && b
%define OP_OR       41      ; a || b
%define OP_NOT      42      ; !a
%define OP_XOR      43      ; a ^ b

; Control flow
%define OP_EXEC     50      ; Execute top of stack
%define OP_IF       51      ; Conditional execution
%define OP_IFELSE   52      ; if-else
%define OP_WHILE    53      ; while loop
%define OP_FOR      54      ; for loop
%define OP_FORALL   55      ; iterate array
%define OP_RETURN   56      ; Return from procedure
%define OP_JUMP     57      ; Unconditional jump
%define OP_JUMPIF   58      ; Conditional jump
%define OP_CALL     59      ; Call operator by index

; Entity operations
%define OP_ENTITY_PUSH  60  ; Push entity onto entity stack
%define OP_ENTITY_POP   61  ; Pop entity from entity stack
%define OP_DEF          62  ; Define variable
%define OP_LOOKUP       63  ; Lookup name in context
%define OP_NEW_ENTITY   64  ; Create new entity

; Array operations
%define OP_NEW_ARRAY 70     ; Create new array
%define OP_ADDTO     71     ; Add to array
%define OP_LENGTH    72     ; Get array length
%define OP_GET       73     ; Get array element
%define OP_PUT       74     ; Put array element

; Table operations
%define OP_NEW_TABLE  80    ; Create new table
%define OP_TABLE_GET  81    ; Get table value
%define OP_TABLE_PUT  82    ; Put table value

; String operations
%define OP_CONCAT     90    ; String concatenation
%define OP_SUBSTRING  91    ; Substring

; Constants
%define OP_PUSH_TRUE  100   ; Push true
%define OP_PUSH_FALSE 101   ; Push false
%define OP_PUSH_NULL  102   ; Push null
%define OP_PUSH_ZERO  103   ; Push 0
%define OP_PUSH_ONE   104   ; Push 1

; Extended opcodes
%define OP_OPERATOR   200   ; Call operator by index
%define OP_CONSTANT   201   ; Push constant by index
%define OP_NAME       202   ; Push name by index

; Additional stack operations for Forth compatibility
%define OP_OVER       105   ; Copy second to top (a b -- a b a)
%define OP_PICK       106   ; Copy nth to top
%define OP_DROP       107   ; Alias for POP

; Error codes
%define ERR_NONE            0
%define ERR_STACK_UNDERFLOW 1
%define ERR_STACK_OVERFLOW  2
%define ERR_INVALID_OPCODE  3
%define ERR_DIV_BY_ZERO     4
%define ERR_OUT_OF_BOUNDS   5

; Value types (for tagged values)
%define VTAG_NULL     0
%define VTAG_BOOLEAN  1
%define VTAG_INTEGER  2
%define VTAG_DOUBLE   3
%define VTAG_STRING   4
%define VTAG_NAME     5
%define VTAG_ARRAY    6
%define VTAG_ENTITY   7
%define VTAG_MARK     8

; Stack limits
%define STACK_LIMIT   1000
