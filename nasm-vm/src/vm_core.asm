; DTRules NASM VM - Core Execution Engine
; Copyright 2024 Paul Snow
; Licensed under the Apache License, Version 2.0
;
; This implements a stack-based VM with jump table dispatch.
; Architecture: Single entry/exit, each handler ends with jmp dispatch.

%include "opcodes.inc"
%include "value.inc"
%include "vm_state.inc"

section .data
    align 8

section .bss
    align 8

section .text
    global vm_execute
    global vm_init

; External handlers (defined in separate files)

; Stack operations
extern op_push_int_handler
extern op_pop_handler
extern op_dup_handler
extern op_swap_handler
extern op_push_true_handler
extern op_push_false_handler
extern op_push_null_handler
extern op_push_zero_handler
extern op_push_one_handler

; Arithmetic operations
extern op_add_handler
extern op_sub_handler
extern op_mul_handler
extern op_div_handler
extern op_mod_handler
extern op_neg_handler
extern op_abs_handler
extern op_inc_handler
extern op_dec_handler

;-----------------------------------------------------------------------------
; vm_execute - Main VM entry point
;
; Input:
;   rdi = pointer to VMState structure
;
; Output:
;   rax = error code (0 = success)
;
; Register usage (preserved across handlers):
;   r12 = VMState pointer
;   r13 = bytecode pointer
;   r14 = stack base pointer
;   r15 = program counter (byte offset)
;   rbx = stack pointer (entry offset from base)
;
; Handlers can use rax, rcx, rdx, rsi, rdi, r8-r11 freely.
;-----------------------------------------------------------------------------
vm_execute:
    ; Save callee-saved registers
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14
    push r15

    ; Set up VM state in preserved registers
    mov r12, rdi                        ; r12 = VMState*

    ; Load bytecode pointer
    mov r13, [r12 + VMSTATE_BYTECODE]   ; r13 = bytecode

    ; Load stack base
    mov r14, [r12 + VMSTATE_STACK]      ; r14 = stack base

    ; Load program counter
    mov r15, [r12 + VMSTATE_PC]         ; r15 = pc

    ; Load stack pointer
    mov rbx, [r12 + VMSTATE_SP]         ; rbx = sp (entry count)

    ; Clear error code
    mov qword [r12 + VMSTATE_ERROR_CODE], ERR_NONE

    ; Fall through to dispatch loop
    jmp vm_dispatch

;-----------------------------------------------------------------------------
; Dispatch loop - fetch opcode and jump to handler
;-----------------------------------------------------------------------------
global vm_dispatch
vm_dispatch:
    ; Check if we've reached end of bytecode
    mov rax, [r12 + VMSTATE_BC_LEN]
    cmp r15, rax
    jge vm_halt_success

    ; Fetch opcode
    movzx eax, byte [r13 + r15]         ; eax = opcode
    inc r15                             ; advance PC

    ; Bounds check opcode
    cmp eax, MAX_OPCODE
    ja vm_error_invalid_opcode

    ; Jump table dispatch
    lea rcx, [rel jump_table]
    jmp [rcx + rax*8]

;-----------------------------------------------------------------------------
; Success exit
;-----------------------------------------------------------------------------
global vm_halt_success
vm_halt_success:
    ; Save final state back to VMState
    mov [r12 + VMSTATE_PC], r15
    mov [r12 + VMSTATE_SP], rbx

    xor eax, eax                        ; return 0 (success)
    jmp vm_exit

;-----------------------------------------------------------------------------
; Error handlers
;-----------------------------------------------------------------------------
global vm_error_invalid_opcode
vm_error_invalid_opcode:
    mov qword [r12 + VMSTATE_ERROR_CODE], ERR_INVALID_OPCODE
    dec r15                             ; point back to invalid opcode
    mov [r12 + VMSTATE_ERROR_PC], r15
    mov eax, ERR_INVALID_OPCODE
    jmp vm_exit

global vm_error_stack_underflow
vm_error_stack_underflow:
    mov qword [r12 + VMSTATE_ERROR_CODE], ERR_STACK_UNDERFLOW
    mov [r12 + VMSTATE_ERROR_PC], r15
    mov eax, ERR_STACK_UNDERFLOW
    jmp vm_exit

global vm_error_stack_overflow
vm_error_stack_overflow:
    mov qword [r12 + VMSTATE_ERROR_CODE], ERR_STACK_OVERFLOW
    mov [r12 + VMSTATE_ERROR_PC], r15
    mov eax, ERR_STACK_OVERFLOW
    jmp vm_exit

global vm_error_type_mismatch
vm_error_type_mismatch:
    mov qword [r12 + VMSTATE_ERROR_CODE], ERR_TYPE_MISMATCH
    mov [r12 + VMSTATE_ERROR_PC], r15
    mov eax, ERR_TYPE_MISMATCH
    jmp vm_exit

global vm_error_div_by_zero
vm_error_div_by_zero:
    mov qword [r12 + VMSTATE_ERROR_CODE], ERR_DIV_BY_ZERO
    mov [r12 + VMSTATE_ERROR_PC], r15
    mov eax, ERR_DIV_BY_ZERO
    jmp vm_exit

;-----------------------------------------------------------------------------
; Exit point (global for external error handlers)
;-----------------------------------------------------------------------------
global vm_exit
vm_exit:
    ; Save final state
    mov [r12 + VMSTATE_PC], r15
    mov [r12 + VMSTATE_SP], rbx

    ; Restore callee-saved registers
    pop r15
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; NOP handler
;-----------------------------------------------------------------------------
global op_nop
op_nop:
    jmp vm_dispatch

;-----------------------------------------------------------------------------
; HALT handler (for explicit halt opcode)
;-----------------------------------------------------------------------------
global op_halt
op_halt:
    jmp vm_halt_success

;-----------------------------------------------------------------------------
; Unimplemented opcode handler
;-----------------------------------------------------------------------------
global op_unimplemented
op_unimplemented:
    jmp vm_error_invalid_opcode

;-----------------------------------------------------------------------------
; Jump table - 256 entries for all possible opcodes
;-----------------------------------------------------------------------------
section .rodata
    align 8
global jump_table
jump_table:
    ; 0-9: Stack operations
    dq op_nop               ; 0  - NOP
    dq op_unimplemented     ; 1  - PUSH
    dq op_push_int_handler  ; 2  - PUSH_INT
    dq op_pop_handler       ; 3  - POP
    dq op_dup_handler       ; 4  - DUP
    dq op_swap_handler      ; 5  - SWAP
    dq op_unimplemented     ; 6  - ROT
    dq op_unimplemented     ; 7  - ROLL
    dq op_unimplemented     ; 8  - INDEX
    dq op_unimplemented     ; 9  - CLEAR
    dq op_unimplemented     ; 10 - MARK

    ; 11-19: Reserved
    dq op_unimplemented     ; 11
    dq op_unimplemented     ; 12
    dq op_unimplemented     ; 13
    dq op_unimplemented     ; 14
    dq op_unimplemented     ; 15
    dq op_unimplemented     ; 16
    dq op_unimplemented     ; 17
    dq op_unimplemented     ; 18
    dq op_unimplemented     ; 19

    ; 20-28: Arithmetic operations
    dq op_add_handler       ; 20 - ADD
    dq op_sub_handler       ; 21 - SUB
    dq op_mul_handler       ; 22 - MUL
    dq op_div_handler       ; 23 - DIV
    dq op_mod_handler       ; 24 - MOD
    dq op_neg_handler       ; 25 - NEG
    dq op_abs_handler       ; 26 - ABS
    dq op_inc_handler       ; 27 - INC
    dq op_dec_handler       ; 28 - DEC

    ; 29: Reserved
    dq op_unimplemented     ; 29

    ; 30-35: Comparison operations
    dq op_unimplemented     ; 30 - EQ
    dq op_unimplemented     ; 31 - NE
    dq op_unimplemented     ; 32 - LT
    dq op_unimplemented     ; 33 - LE
    dq op_unimplemented     ; 34 - GT
    dq op_unimplemented     ; 35 - GE

    ; 36-39: Reserved
    dq op_unimplemented     ; 36
    dq op_unimplemented     ; 37
    dq op_unimplemented     ; 38
    dq op_unimplemented     ; 39

    ; 40-43: Boolean operations
    dq op_unimplemented     ; 40 - AND
    dq op_unimplemented     ; 41 - OR
    dq op_unimplemented     ; 42 - NOT
    dq op_unimplemented     ; 43 - XOR

    ; 44-49: Reserved
    dq op_unimplemented     ; 44
    dq op_unimplemented     ; 45
    dq op_unimplemented     ; 46
    dq op_unimplemented     ; 47
    dq op_unimplemented     ; 48
    dq op_unimplemented     ; 49

    ; 50-59: Control flow
    dq op_unimplemented     ; 50 - EXEC
    dq op_unimplemented     ; 51 - IF
    dq op_unimplemented     ; 52 - IFELSE
    dq op_unimplemented     ; 53 - WHILE
    dq op_unimplemented     ; 54 - FOR
    dq op_unimplemented     ; 55 - FORALL
    dq op_unimplemented     ; 56 - RETURN
    dq op_unimplemented     ; 57 - JUMP
    dq op_unimplemented     ; 58 - JUMP_IF
    dq op_unimplemented     ; 59 - CALL

    ; 60-64: Entity operations
    dq op_unimplemented     ; 60 - ENTITY_PUSH
    dq op_unimplemented     ; 61 - ENTITY_POP
    dq op_unimplemented     ; 62 - DEF
    dq op_unimplemented     ; 63 - LOOKUP
    dq op_unimplemented     ; 64 - NEW_ENTITY

    ; 65-69: Reserved
    dq op_unimplemented     ; 65
    dq op_unimplemented     ; 66
    dq op_unimplemented     ; 67
    dq op_unimplemented     ; 68
    dq op_unimplemented     ; 69

    ; 70-74: Array operations
    dq op_unimplemented     ; 70 - NEW_ARRAY
    dq op_unimplemented     ; 71 - ADD_TO
    dq op_unimplemented     ; 72 - LENGTH
    dq op_unimplemented     ; 73 - GET
    dq op_unimplemented     ; 74 - PUT

    ; 75-79: Reserved
    dq op_unimplemented     ; 75
    dq op_unimplemented     ; 76
    dq op_unimplemented     ; 77
    dq op_unimplemented     ; 78
    dq op_unimplemented     ; 79

    ; 80-82: Table operations
    dq op_unimplemented     ; 80 - NEW_TABLE
    dq op_unimplemented     ; 81 - TABLE_GET
    dq op_unimplemented     ; 82 - TABLE_PUT

    ; 83-89: Reserved
    dq op_unimplemented     ; 83
    dq op_unimplemented     ; 84
    dq op_unimplemented     ; 85
    dq op_unimplemented     ; 86
    dq op_unimplemented     ; 87
    dq op_unimplemented     ; 88
    dq op_unimplemented     ; 89

    ; 90-91: String operations
    dq op_unimplemented     ; 90 - CONCAT
    dq op_unimplemented     ; 91 - SUBSTRING

    ; 92-99: Reserved
    dq op_unimplemented     ; 92
    dq op_unimplemented     ; 93
    dq op_unimplemented     ; 94
    dq op_unimplemented     ; 95
    dq op_unimplemented     ; 96
    dq op_unimplemented     ; 97
    dq op_unimplemented     ; 98
    dq op_unimplemented     ; 99

    ; 100-104: Constant operations
    dq op_push_true_handler  ; 100 - PUSH_TRUE
    dq op_push_false_handler ; 101 - PUSH_FALSE
    dq op_push_null_handler  ; 102 - PUSH_NULL
    dq op_push_zero_handler  ; 103 - PUSH_ZERO
    dq op_push_one_handler   ; 104 - PUSH_ONE

    ; 105-199: Reserved (fill with unimplemented)
    %rep 95
    dq op_unimplemented
    %endrep

    ; 200-202: Extended opcodes
    dq op_unimplemented     ; 200 - OPERATOR
    dq op_unimplemented     ; 201 - CONSTANT
    dq op_unimplemented     ; 202 - NAME

    ; 203-254: Reserved
    %rep 52
    dq op_unimplemented
    %endrep

    ; 255: HALT
    dq op_halt              ; 255 - HALT

; Mark stack as non-executable (security best practice)
section .note.GNU-stack noalloc noexec nowrite progbits
