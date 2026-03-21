; vm_jump_table.asm - Opcode dispatch table
; DTRules NASM VM Implementation
;
; This defines the jump table for opcode dispatch.
; Each entry is an 8-byte pointer to the handler function.
; Opcodes match Go's bytecode.go numbering exactly.

bits 64
default rel

%include "include/constants.inc"

;-----------------------------------------------------------------------------
; External references to opcode handlers
;-----------------------------------------------------------------------------
extern op_invalid
extern vm_dispatch

; Stack operations
extern op_nop
extern op_push_constant
extern op_push_int
extern op_pop
extern op_dup
extern op_swap
extern op_rot
extern op_roll
extern op_pick
extern op_clear
extern op_mark

; Arithmetic operations
extern op_add
extern op_sub
extern op_mul
extern op_div
extern op_mod
extern op_neg
extern op_abs
extern op_inc
extern op_dec

; Comparison operations
extern op_eq
extern op_ne
extern op_lt
extern op_le
extern op_gt
extern op_ge

; Logical operations
extern op_and
extern op_or
extern op_not
extern op_xor

; Control flow
extern op_exec
extern op_if
extern op_ifelse
extern op_while
extern op_for
extern op_forall
extern op_return
extern op_jump
extern op_jumpif
extern op_call

; Entity operations
extern op_entitypush
extern op_entitypop
extern op_def
extern op_lookup
extern op_newentity

; Array operations
extern op_array_new
extern op_array_push
extern op_array_len
extern op_array_get
extern op_array_set

; Table operations
extern op_newtable
extern op_tableget
extern op_tableput

; String operations
extern op_str_concat
extern op_str_substring

; Constant shortcuts
extern op_push_true
extern op_push_false
extern op_push_null
extern op_push_zero
extern op_push_one

; Extended opcodes
extern op_operator
extern op_constant
extern op_name

; Debug/utility
extern op_print
extern op_trace
extern op_debug
extern op_halt

;-----------------------------------------------------------------------------
; Opcode handler table
;
; This table must have exactly 256 entries (one per possible opcode byte).
; Unused entries point to op_invalid.
;
; Handler calling convention:
; - rbx = bytecode IP (handler should advance if reading operands)
; - r12 = data stack pointer
; - r13 = entity stack pointer
; - r14 = control stack pointer
; - r15 = state pointer
; - Handler should jmp vm_dispatch when done (not ret)
;-----------------------------------------------------------------------------
section .data
    align 8
    global opcode_table
    opcode_table:
        ; 0-10: Stack operations (Go compatible)
        dq op_nop           ; 0 - OP_NOP
        dq op_push_constant ; 1 - OP_PUSH (constant from pool)
        dq op_push_int      ; 2 - OP_PUSH_INT
        dq op_pop           ; 3 - OP_POP
        dq op_dup           ; 4 - OP_DUP
        dq op_swap          ; 5 - OP_SWAP
        dq op_rot           ; 6 - OP_ROT
        dq op_roll          ; 7 - OP_ROLL
        dq op_pick          ; 8 - OP_INDEX (pick)
        dq op_clear         ; 9 - OP_CLEAR
        dq op_mark          ; 10 - OP_MARK

        ; 11-19: Reserved
        times 9 dq op_invalid

        ; 20-28: Arithmetic (Go compatible)
        dq op_add           ; 20 - OP_ADD
        dq op_sub           ; 21 - OP_SUB
        dq op_mul           ; 22 - OP_MUL
        dq op_div           ; 23 - OP_DIV
        dq op_mod           ; 24 - OP_MOD
        dq op_neg           ; 25 - OP_NEG
        dq op_abs           ; 26 - OP_ABS
        dq op_inc           ; 27 - OP_INC
        dq op_dec           ; 28 - OP_DEC

        ; 29: Reserved
        dq op_invalid

        ; 30-35: Comparison (Go compatible)
        dq op_eq            ; 30 - OP_EQ
        dq op_ne            ; 31 - OP_NE
        dq op_lt            ; 32 - OP_LT
        dq op_le            ; 33 - OP_LE
        dq op_gt            ; 34 - OP_GT
        dq op_ge            ; 35 - OP_GE

        ; 36-39: Reserved
        times 4 dq op_invalid

        ; 40-43: Logical (Go compatible)
        dq op_and           ; 40 - OP_AND
        dq op_or            ; 41 - OP_OR
        dq op_not           ; 42 - OP_NOT
        dq op_xor           ; 43 - OP_XOR

        ; 44-49: Reserved
        times 6 dq op_invalid

        ; 50-59: Control flow (Go compatible)
        dq op_exec          ; 50 - OP_EXEC
        dq op_if            ; 51 - OP_IF
        dq op_ifelse        ; 52 - OP_IFELSE
        dq op_while         ; 53 - OP_WHILE
        dq op_for           ; 54 - OP_FOR
        dq op_forall        ; 55 - OP_FORALL
        dq op_return        ; 56 - OP_RETURN
        dq op_jump          ; 57 - OP_JUMP
        dq op_jumpif        ; 58 - OP_JUMPIF
        dq op_call          ; 59 - OP_CALL

        ; 60-64: Entity operations (Go compatible)
        dq op_entitypush    ; 60 - OP_ENTITYPUSH
        dq op_entitypop     ; 61 - OP_ENTITYPOP
        dq op_def           ; 62 - OP_DEF
        dq op_lookup        ; 63 - OP_LOOKUP
        dq op_newentity     ; 64 - OP_NEWENTITY

        ; 65-69: Reserved
        times 5 dq op_invalid

        ; 70-74: Array operations (Go compatible)
        dq op_array_new     ; 70 - OP_NEWARRAY
        dq op_array_push    ; 71 - OP_ADDTO
        dq op_array_len     ; 72 - OP_LENGTH
        dq op_array_get     ; 73 - OP_GET
        dq op_array_set     ; 74 - OP_PUT

        ; 75-79: Reserved
        times 5 dq op_invalid

        ; 80-82: Table operations (Go compatible)
        dq op_newtable      ; 80 - OP_NEWTABLE
        dq op_tableget      ; 81 - OP_TABLEGET
        dq op_tableput      ; 82 - OP_TABLEPUT

        ; 83-89: Reserved
        times 7 dq op_invalid

        ; 90-91: String operations (Go compatible)
        dq op_str_concat    ; 90 - OP_CONCAT
        dq op_str_substring ; 91 - OP_SUBSTRING

        ; 92-99: Reserved
        times 8 dq op_invalid

        ; 100-104: Constant shortcuts (Go compatible)
        dq op_push_true     ; 100 - OP_PUSH_TRUE
        dq op_push_false    ; 101 - OP_PUSH_FALSE
        dq op_push_null     ; 102 - OP_PUSH_NULL
        dq op_push_zero     ; 103 - OP_PUSH_ZERO
        dq op_push_one      ; 104 - OP_PUSH_ONE

        ; 105-199: Reserved
        times 95 dq op_invalid

        ; 200-202: Extended opcodes (Go compatible)
        dq op_operator      ; 200 - OP_OPERATOR
        dq op_constant      ; 201 - OP_CONSTANT
        dq op_name          ; 202 - OP_NAME

        ; 203-249: Reserved
        times 47 dq op_invalid

        ; 250-252: Debug/utility
        dq op_print         ; 250 - OP_PRINT
        dq op_trace         ; 251 - OP_TRACE
        dq op_debug         ; 252 - OP_DEBUG

        ; 253-254: Reserved
        times 2 dq op_invalid

        ; 255: Halt
        dq op_halt          ; 255 - OP_HALT
