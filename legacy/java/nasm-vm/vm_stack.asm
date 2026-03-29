; DTRules NASM VM Stack Operations
; Copyright 2024 Paul Snow
; Licensed under Apache License 2.0
;
; This file implements basic stack operations for the DTRules NASM VM:
;   - OP_NOP (0): No operation
;   - OP_POP (3): Pop and discard top value
;   - OP_DUP (4): Duplicate top value
;   - OP_SWAP (5): Swap top two values
;   - OP_PUSH_TRUE (100): Push boolean true
;   - OP_PUSH_FALSE (101): Push boolean false
;   - OP_PUSH_NULL (102): Push null value
;   - OP_PUSH_ZERO (103): Push integer 0
;   - OP_PUSH_ONE (104): Push integer 1
;   - OP_HALT (255): Stop execution

%include "vm_constants.inc"
%include "vm_state.inc"

; External symbols from vm_entry.asm
extern vm_dispatch
extern vm_error_return
extern vm_exit_success

section .text

; ============================================================================
; OP_NOP (0) - No Operation
; ============================================================================
global op_nop
op_nop:
    jmp vm_dispatch


; ============================================================================
; OP_POP (3) - Pop and discard
; ============================================================================
; Removes the top value from the stack
; Stack: a -> (empty)
;
global op_pop
op_pop:
    ; Get stack pointer
    mov rax, [r12 + VMContext.data_sp]

    ; Need at least 1 element
    sub rax, VALUE_SIZE
    cmp rax, [r12 + VMContext.data_base]
    jl .underflow

    ; Update stack pointer (discard top value)
    mov [r12 + VMContext.data_sp], rax

    jmp vm_dispatch

.underflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_UNDERFLOW
    jmp vm_error_return


; ============================================================================
; OP_DUP (4) - Duplicate top
; ============================================================================
; Duplicates the top value
; Stack: a -> a a
;
global op_dup
op_dup:
    ; Get stack pointer
    mov rax, [r12 + VMContext.data_sp]

    ; Need at least 1 element to duplicate
    mov rbx, rax
    sub rbx, VALUE_SIZE
    cmp rbx, [r12 + VMContext.data_base]
    jl .underflow

    ; Check for overflow (need room for one more)
    cmp rax, [r12 + VMContext.data_limit]
    jge .overflow

    ; Copy top value
    ; Source: rax - VALUE_SIZE
    ; Dest: rax
    sub rax, VALUE_SIZE

    ; Copy tag
    movzx ecx, byte [rax + Value.tag]
    mov byte [rax + VALUE_SIZE + Value.tag], cl

    ; Copy num
    mov rcx, [rax + Value.num]
    mov [rax + VALUE_SIZE + Value.num], rcx

    ; Copy ptr
    mov rcx, [rax + Value.ptr]
    mov [rax + VALUE_SIZE + Value.ptr], rcx

    ; Update stack pointer
    add rax, VALUE_SIZE * 2
    mov [r12 + VMContext.data_sp], rax

    jmp vm_dispatch

.underflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_UNDERFLOW
    jmp vm_error_return

.overflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_OVERFLOW
    jmp vm_error_return


; ============================================================================
; OP_SWAP (5) - Swap top two
; ============================================================================
; Swaps the top two values
; Stack: a b -> b a
;
global op_swap
op_swap:
    ; Get stack pointer
    mov rax, [r12 + VMContext.data_sp]

    ; Need at least 2 elements
    sub rax, VALUE_SIZE * 2
    cmp rax, [r12 + VMContext.data_base]
    jl .underflow

    ; rax points to 'a', rax + VALUE_SIZE points to 'b'
    ; Swap tags
    movzx ecx, byte [rax + Value.tag]
    movzx edx, byte [rax + VALUE_SIZE + Value.tag]
    mov byte [rax + Value.tag], dl
    mov byte [rax + VALUE_SIZE + Value.tag], cl

    ; Swap nums
    mov rcx, [rax + Value.num]
    mov rdx, [rax + VALUE_SIZE + Value.num]
    mov [rax + Value.num], rdx
    mov [rax + VALUE_SIZE + Value.num], rcx

    ; Swap ptrs
    mov rcx, [rax + Value.ptr]
    mov rdx, [rax + VALUE_SIZE + Value.ptr]
    mov [rax + Value.ptr], rdx
    mov [rax + VALUE_SIZE + Value.ptr], rcx

    jmp vm_dispatch

.underflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_UNDERFLOW
    jmp vm_error_return


; ============================================================================
; OP_PUSH_TRUE (100) - Push boolean true
; ============================================================================
global op_push_true
op_push_true:
    ; Get stack pointer
    mov rax, [r12 + VMContext.data_sp]

    ; Check for overflow
    cmp rax, [r12 + VMContext.data_limit]
    jge .overflow

    ; Push true value
    mov byte [rax + Value.tag], VTAG_BOOLEAN
    mov qword [rax + Value.num], 1
    mov qword [rax + Value.ptr], 0

    ; Update stack pointer
    add rax, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rax

    jmp vm_dispatch

.overflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_OVERFLOW
    jmp vm_error_return


; ============================================================================
; OP_PUSH_FALSE (101) - Push boolean false
; ============================================================================
global op_push_false
op_push_false:
    ; Get stack pointer
    mov rax, [r12 + VMContext.data_sp]

    ; Check for overflow
    cmp rax, [r12 + VMContext.data_limit]
    jge .overflow

    ; Push false value
    mov byte [rax + Value.tag], VTAG_BOOLEAN
    mov qword [rax + Value.num], 0
    mov qword [rax + Value.ptr], 0

    ; Update stack pointer
    add rax, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rax

    jmp vm_dispatch

.overflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_OVERFLOW
    jmp vm_error_return


; ============================================================================
; OP_PUSH_NULL (102) - Push null value
; ============================================================================
global op_push_null
op_push_null:
    ; Get stack pointer
    mov rax, [r12 + VMContext.data_sp]

    ; Check for overflow
    cmp rax, [r12 + VMContext.data_limit]
    jge .overflow

    ; Push null value
    mov byte [rax + Value.tag], VTAG_NULL
    mov qword [rax + Value.num], 0
    mov qword [rax + Value.ptr], 0

    ; Update stack pointer
    add rax, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rax

    jmp vm_dispatch

.overflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_OVERFLOW
    jmp vm_error_return


; ============================================================================
; OP_PUSH_ZERO (103) - Push integer 0
; ============================================================================
global op_push_zero
op_push_zero:
    ; Get stack pointer
    mov rax, [r12 + VMContext.data_sp]

    ; Check for overflow
    cmp rax, [r12 + VMContext.data_limit]
    jge .overflow

    ; Push integer 0
    mov byte [rax + Value.tag], VTAG_INTEGER
    mov qword [rax + Value.num], 0
    mov qword [rax + Value.ptr], 0

    ; Update stack pointer
    add rax, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rax

    jmp vm_dispatch

.overflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_OVERFLOW
    jmp vm_error_return


; ============================================================================
; OP_PUSH_ONE (104) - Push integer 1
; ============================================================================
global op_push_one
op_push_one:
    ; Get stack pointer
    mov rax, [r12 + VMContext.data_sp]

    ; Check for overflow
    cmp rax, [r12 + VMContext.data_limit]
    jge .overflow

    ; Push integer 1
    mov byte [rax + Value.tag], VTAG_INTEGER
    mov qword [rax + Value.num], 1
    mov qword [rax + Value.ptr], 0

    ; Update stack pointer
    add rax, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rax

    jmp vm_dispatch

.overflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_OVERFLOW
    jmp vm_error_return


; ============================================================================
; OP_HALT (255) - Stop execution
; ============================================================================
global op_halt
op_halt:
    jmp vm_exit_success


; ============================================================================
; Mark stack as non-executable
; ============================================================================
section .note.GNU-stack noalloc noexec nowrite progbits
