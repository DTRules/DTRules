; DTRules NASM VM - Stack Operations
; Copyright 2024 Paul Snow
; Licensed under the Apache License, Version 2.0
;
; Implements: PUSH_INT, PUSH_ZERO, PUSH_ONE, PUSH_TRUE, PUSH_FALSE, PUSH_NULL, POP, DUP, SWAP
;
; Register conventions (preserved across handlers):
;   r12 = VMState pointer
;   r13 = bytecode pointer
;   r14 = stack base pointer
;   r15 = program counter (byte offset)
;   rbx = stack pointer (byte offset from base)
;
; Each handler:
;   - Can freely use rax, rcx, rdx, rsi, rdi, r8-r11
;   - Must end with: jmp vm_dispatch

%include "opcodes.inc"
%include "value.inc"
%include "vm_state.inc"

section .text

; External symbols from vm_core.asm
extern vm_dispatch
extern vm_error_stack_underflow
extern vm_error_stack_overflow

;-----------------------------------------------------------------------------
; Helper macro: Check stack has room for N more entries
;-----------------------------------------------------------------------------
%macro CHECK_STACK_ROOM 1
    mov rax, [r12 + VMSTATE_STACK_SIZE]
    imul rax, VALUE_SIZE
    mov rcx, rbx
    add rcx, %1 * VALUE_SIZE
    cmp rcx, rax
    jg vm_error_stack_overflow
%endmacro

;-----------------------------------------------------------------------------
; Helper macro: Check stack has at least N entries
;-----------------------------------------------------------------------------
%macro CHECK_STACK_MIN 1
    cmp rbx, %1 * VALUE_SIZE
    jl vm_error_stack_underflow
%endmacro

;-----------------------------------------------------------------------------
; Helper macro: Get stack entry at offset from top
; %1 = register to store pointer, %2 = offset (0 = top, 1 = second, etc.)
;-----------------------------------------------------------------------------
%macro STACK_ENTRY 2
    lea %1, [r14 + rbx - (%2 + 1) * VALUE_SIZE]
%endmacro

;-----------------------------------------------------------------------------
; op_push_int_handler - Push immediate integer from bytecode
; Reads a varint from bytecode and pushes as integer
;-----------------------------------------------------------------------------
global op_push_int_handler
op_push_int_handler:
    CHECK_STACK_ROOM 1

    ; Read varint from bytecode
    xor eax, eax                    ; result = 0
    xor ecx, ecx                    ; shift = 0

.read_varint_loop:
    ; Check bounds
    mov rdx, [r12 + VMSTATE_BC_LEN]
    cmp r15, rdx
    jge .varint_truncated

    ; Read byte
    movzx edx, byte [r13 + r15]
    inc r15

    ; Extract 7 bits and add to result
    mov r8d, edx
    and r8d, 0x7f
    shlx r8, r8, rcx                ; r8 = (byte & 0x7f) << shift
    or rax, r8

    ; Check continuation bit
    test dl, 0x80
    jz .varint_done

    ; Continue reading
    add ecx, 7
    cmp ecx, 63                     ; prevent overflow
    jle .read_varint_loop

.varint_done:
    ; Sign extend if needed (varints are signed)
    ; For simplicity, we treat small varints as signed
    ; Check if the value should be negative (bit 6 of last byte was set in original encoding)
    ; Actually, the Go implementation uses unsigned varints. Let's keep it simple.

    ; Push integer onto stack
    lea rdi, [r14 + rbx]            ; rdi -> new top entry
    mov byte [rdi + VALUE_TAG_OFF], VTAG_INTEGER
    mov [rdi + VALUE_NUM_OFF], rax
    mov qword [rdi + VALUE_PTR_OFF], 0

    ; Advance stack pointer
    add rbx, VALUE_SIZE

    jmp vm_dispatch

.varint_truncated:
    ; Bytecode ended mid-varint - treat as end of bytecode
    jmp vm_dispatch

;-----------------------------------------------------------------------------
; op_push_zero_handler - Push integer 0
;-----------------------------------------------------------------------------
global op_push_zero_handler
op_push_zero_handler:
    CHECK_STACK_ROOM 1

    ; Push 0 onto stack
    lea rdi, [r14 + rbx]
    mov byte [rdi + VALUE_TAG_OFF], VTAG_INTEGER
    mov qword [rdi + VALUE_NUM_OFF], 0
    mov qword [rdi + VALUE_PTR_OFF], 0

    add rbx, VALUE_SIZE
    jmp vm_dispatch

;-----------------------------------------------------------------------------
; op_push_one_handler - Push integer 1
;-----------------------------------------------------------------------------
global op_push_one_handler
op_push_one_handler:
    CHECK_STACK_ROOM 1

    ; Push 1 onto stack
    lea rdi, [r14 + rbx]
    mov byte [rdi + VALUE_TAG_OFF], VTAG_INTEGER
    mov qword [rdi + VALUE_NUM_OFF], 1
    mov qword [rdi + VALUE_PTR_OFF], 0

    add rbx, VALUE_SIZE
    jmp vm_dispatch

;-----------------------------------------------------------------------------
; op_push_true_handler - Push boolean true
;-----------------------------------------------------------------------------
global op_push_true_handler
op_push_true_handler:
    CHECK_STACK_ROOM 1

    lea rdi, [r14 + rbx]
    mov byte [rdi + VALUE_TAG_OFF], VTAG_BOOLEAN
    mov qword [rdi + VALUE_NUM_OFF], 1
    mov qword [rdi + VALUE_PTR_OFF], 0

    add rbx, VALUE_SIZE
    jmp vm_dispatch

;-----------------------------------------------------------------------------
; op_push_false_handler - Push boolean false
;-----------------------------------------------------------------------------
global op_push_false_handler
op_push_false_handler:
    CHECK_STACK_ROOM 1

    lea rdi, [r14 + rbx]
    mov byte [rdi + VALUE_TAG_OFF], VTAG_BOOLEAN
    mov qword [rdi + VALUE_NUM_OFF], 0
    mov qword [rdi + VALUE_PTR_OFF], 0

    add rbx, VALUE_SIZE
    jmp vm_dispatch

;-----------------------------------------------------------------------------
; op_push_null_handler - Push null
;-----------------------------------------------------------------------------
global op_push_null_handler
op_push_null_handler:
    CHECK_STACK_ROOM 1

    lea rdi, [r14 + rbx]
    mov byte [rdi + VALUE_TAG_OFF], VTAG_NULL
    mov qword [rdi + VALUE_NUM_OFF], 0
    mov qword [rdi + VALUE_PTR_OFF], 0

    add rbx, VALUE_SIZE
    jmp vm_dispatch

;-----------------------------------------------------------------------------
; op_pop_handler - Pop and discard top of stack
;-----------------------------------------------------------------------------
global op_pop_handler
op_pop_handler:
    CHECK_STACK_MIN 1

    sub rbx, VALUE_SIZE
    jmp vm_dispatch

;-----------------------------------------------------------------------------
; op_dup_handler - Duplicate top of stack
;-----------------------------------------------------------------------------
global op_dup_handler
op_dup_handler:
    CHECK_STACK_MIN 1
    CHECK_STACK_ROOM 1

    ; Get source (current top)
    STACK_ENTRY rsi, 0

    ; Get destination (new top)
    lea rdi, [r14 + rbx]

    ; Copy 24 bytes (VALUE_SIZE)
    mov rax, [rsi]
    mov [rdi], rax
    mov rax, [rsi + 8]
    mov [rdi + 8], rax
    mov rax, [rsi + 16]
    mov [rdi + 16], rax

    add rbx, VALUE_SIZE
    jmp vm_dispatch

;-----------------------------------------------------------------------------
; op_swap_handler - Swap top two stack entries
;-----------------------------------------------------------------------------
global op_swap_handler
op_swap_handler:
    CHECK_STACK_MIN 2

    STACK_ENTRY rsi, 0              ; top
    STACK_ENTRY rdi, 1              ; second

    ; Swap using r8-r10 as temp
    mov rax, [rsi]
    mov rcx, [rdi]
    mov [rsi], rcx
    mov [rdi], rax

    mov rax, [rsi + 8]
    mov rcx, [rdi + 8]
    mov [rsi + 8], rcx
    mov [rdi + 8], rax

    mov rax, [rsi + 16]
    mov rcx, [rdi + 16]
    mov [rsi + 16], rcx
    mov [rdi + 16], rax

    jmp vm_dispatch

; Mark stack as non-executable (security best practice)
section .note.GNU-stack noalloc noexec nowrite progbits
