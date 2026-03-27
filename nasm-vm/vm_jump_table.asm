; DTRules NASM VM Jump Table
; Copyright 2024 Paul Snow
; Licensed under Apache License 2.0
;
; This file defines the jump table for opcode dispatch.
; Each entry is a 64-bit pointer to the handler for that opcode.
; Invalid/unimplemented opcodes point to op_invalid.

%include "vm_constants.inc"
%include "vm_state.inc"

; External symbols from vm_entry.asm
extern vm_dispatch
extern vm_error_return
extern vm_exit_success

; External opcode handlers from vm_boolean.asm
extern op_and
extern op_or
extern op_not
extern op_xor
extern op_isnull
extern op_notnull
extern op_beq

; External opcode handlers from vm_stack.asm (if implemented)
extern op_nop
extern op_push_true
extern op_push_false
extern op_push_null
extern op_push_zero
extern op_push_one
extern op_dup
extern op_pop
extern op_swap
extern op_halt

section .data

; ============================================================================
; Jump Table
; ============================================================================
; 256 entries, 8 bytes each = 2048 bytes
; Indexed by opcode value

global vm_jump_table
align 64
vm_jump_table:
    ; Stack operations (0-19)
    dq op_nop               ; 0:  OP_NOP
    dq op_invalid           ; 1:  OP_PUSH
    dq op_invalid           ; 2:  OP_PUSH_INT
    dq op_pop               ; 3:  OP_POP
    dq op_dup               ; 4:  OP_DUP
    dq op_swap              ; 5:  OP_SWAP
    dq op_invalid           ; 6:  OP_ROT
    dq op_invalid           ; 7:  OP_ROLL
    dq op_invalid           ; 8:  OP_INDEX
    dq op_invalid           ; 9:  OP_CLEAR
    dq op_invalid           ; 10: OP_MARK
    dq op_invalid           ; 11: (unused)
    dq op_invalid           ; 12: (unused)
    dq op_invalid           ; 13: (unused)
    dq op_invalid           ; 14: (unused)
    dq op_invalid           ; 15: (unused)
    dq op_invalid           ; 16: (unused)
    dq op_invalid           ; 17: (unused)
    dq op_invalid           ; 18: (unused)
    dq op_invalid           ; 19: (unused)

    ; Arithmetic (20-29)
    dq op_invalid           ; 20: OP_ADD
    dq op_invalid           ; 21: OP_SUB
    dq op_invalid           ; 22: OP_MUL
    dq op_invalid           ; 23: OP_DIV
    dq op_invalid           ; 24: OP_MOD
    dq op_invalid           ; 25: OP_NEG
    dq op_invalid           ; 26: OP_ABS
    dq op_invalid           ; 27: OP_INC
    dq op_invalid           ; 28: OP_DEC
    dq op_invalid           ; 29: (unused)

    ; Comparison (30-39)
    dq op_invalid           ; 30: OP_EQ
    dq op_invalid           ; 31: OP_NE
    dq op_invalid           ; 32: OP_LT
    dq op_invalid           ; 33: OP_LE
    dq op_invalid           ; 34: OP_GT
    dq op_invalid           ; 35: OP_GE
    dq op_invalid           ; 36: (unused)
    dq op_invalid           ; 37: (unused)
    dq op_invalid           ; 38: (unused)
    dq op_invalid           ; 39: (unused)

    ; Boolean (40-49) - PRIMARY FOCUS FOR ISSUE #166
    dq op_and               ; 40: OP_AND
    dq op_or                ; 41: OP_OR
    dq op_not               ; 42: OP_NOT
    dq op_xor               ; 43: OP_XOR
    dq op_isnull            ; 44: OP_ISNULL
    dq op_notnull           ; 45: OP_NOTNULL
    dq op_beq               ; 46: OP_BEQ
    dq op_invalid           ; 47: (unused)
    dq op_invalid           ; 48: (unused)
    dq op_invalid           ; 49: (unused)

    ; Control flow (50-59)
    dq op_invalid           ; 50: OP_EXEC
    dq op_invalid           ; 51: OP_IF
    dq op_invalid           ; 52: OP_IF_ELSE
    dq op_invalid           ; 53: OP_WHILE
    dq op_invalid           ; 54: OP_FOR
    dq op_invalid           ; 55: OP_FOR_ALL
    dq op_invalid           ; 56: OP_RETURN
    dq op_invalid           ; 57: OP_JUMP
    dq op_invalid           ; 58: OP_JUMP_IF
    dq op_invalid           ; 59: OP_CALL

    ; Entity operations (60-69)
    dq op_invalid           ; 60: OP_ENTITY_PUSH
    dq op_invalid           ; 61: OP_ENTITY_POP
    dq op_invalid           ; 62: OP_DEF
    dq op_invalid           ; 63: OP_LOOKUP
    dq op_invalid           ; 64: OP_NEW_ENTITY
    dq op_invalid           ; 65: (unused)
    dq op_invalid           ; 66: (unused)
    dq op_invalid           ; 67: (unused)
    dq op_invalid           ; 68: (unused)
    dq op_invalid           ; 69: (unused)

    ; Array operations (70-79)
    dq op_invalid           ; 70: OP_NEW_ARRAY
    dq op_invalid           ; 71: OP_ADD_TO
    dq op_invalid           ; 72: OP_LENGTH
    dq op_invalid           ; 73: OP_GET
    dq op_invalid           ; 74: OP_PUT
    dq op_invalid           ; 75: (unused)
    dq op_invalid           ; 76: (unused)
    dq op_invalid           ; 77: (unused)
    dq op_invalid           ; 78: (unused)
    dq op_invalid           ; 79: (unused)

    ; Table operations (80-89)
    dq op_invalid           ; 80: OP_NEW_TABLE
    dq op_invalid           ; 81: OP_TABLE_GET
    dq op_invalid           ; 82: OP_TABLE_PUT
    dq op_invalid           ; 83: (unused)
    dq op_invalid           ; 84: (unused)
    dq op_invalid           ; 85: (unused)
    dq op_invalid           ; 86: (unused)
    dq op_invalid           ; 87: (unused)
    dq op_invalid           ; 88: (unused)
    dq op_invalid           ; 89: (unused)

    ; String operations (90-99)
    dq op_invalid           ; 90: OP_CONCAT
    dq op_invalid           ; 91: OP_SUBSTRING
    dq op_invalid           ; 92: (unused)
    dq op_invalid           ; 93: (unused)
    dq op_invalid           ; 94: (unused)
    dq op_invalid           ; 95: (unused)
    dq op_invalid           ; 96: (unused)
    dq op_invalid           ; 97: (unused)
    dq op_invalid           ; 98: (unused)
    dq op_invalid           ; 99: (unused)

    ; Constants (100-109)
    dq op_push_true         ; 100: OP_PUSH_TRUE
    dq op_push_false        ; 101: OP_PUSH_FALSE
    dq op_push_null         ; 102: OP_PUSH_NULL
    dq op_push_zero         ; 103: OP_PUSH_ZERO
    dq op_push_one          ; 104: OP_PUSH_ONE
    dq op_invalid           ; 105: (unused)
    dq op_invalid           ; 106: (unused)
    dq op_invalid           ; 107: (unused)
    dq op_invalid           ; 108: (unused)
    dq op_invalid           ; 109: (unused)

    ; Fill remaining entries (110-199) with op_invalid
    %rep 90
    dq op_invalid
    %endrep

    ; Extended opcodes (200-254)
    dq op_invalid           ; 200: OP_OPERATOR
    dq op_invalid           ; 201: OP_CONSTANT
    dq op_invalid           ; 202: OP_NAME

    ; Fill (203-254)
    %rep 52
    dq op_invalid
    %endrep

    ; HALT (255)
    dq op_halt              ; 255: OP_HALT


section .text

; ============================================================================
; op_invalid - Handler for unimplemented opcodes
; ============================================================================
global op_invalid
op_invalid:
    mov dword [r12 + VMContext.error_code], ERR_INVALID_OPCODE
    jmp vm_error_return


; ============================================================================
; Mark stack as non-executable
; ============================================================================
section .note.GNU-stack noalloc noexec nowrite progbits
