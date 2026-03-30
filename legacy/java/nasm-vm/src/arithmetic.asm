; DTRules NASM VM - Arithmetic Operations
; Copyright 2024 Paul Snow
; Licensed under the Apache License, Version 2.0
;
; Implements: ADD, SUB, MUL, DIV, MOD, NEG, ABS, INC, DEC
;
; Register conventions (preserved across handlers):
;   r12 = VMState pointer
;   r13 = bytecode pointer
;   r14 = stack base pointer
;   r15 = program counter (byte offset)
;   rbx = stack pointer (entry count * VALUE_SIZE)
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
extern vm_error_type_mismatch
extern vm_error_div_by_zero

;-----------------------------------------------------------------------------
; Helper macro: Check stack has at least N entries
; Jumps to error handler if underflow
;-----------------------------------------------------------------------------
%macro CHECK_STACK_MIN 1
    cmp rbx, %1 * VALUE_SIZE
    jl vm_error_stack_underflow
%endmacro

;-----------------------------------------------------------------------------
; Helper macro: Check stack has room for N more entries
; Jumps to error handler if overflow
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
; Helper macro: Get stack entry at offset from top
; %1 = register to store pointer, %2 = offset (0 = top, 1 = second, etc.)
;-----------------------------------------------------------------------------
%macro STACK_ENTRY 2
    lea %1, [r14 + rbx - (%2 + 1) * VALUE_SIZE]
%endmacro

;-----------------------------------------------------------------------------
; op_add_handler - Integer addition: ( a b -- a+b )
; Pops two integers, pushes their sum
;-----------------------------------------------------------------------------
global op_add_handler
op_add_handler:
    ; Need at least 2 entries
    CHECK_STACK_MIN 2

    ; Get pointers to top two entries
    STACK_ENTRY rsi, 0              ; rsi -> b (top)
    STACK_ENTRY rdi, 1              ; rdi -> a (second)

    ; Check both are integers
    movzx eax, byte [rsi + VALUE_TAG_OFF]
    cmp al, VTAG_INTEGER
    jne .add_type_error
    movzx eax, byte [rdi + VALUE_TAG_OFF]
    cmp al, VTAG_INTEGER
    jne .add_type_error

    ; Load values and add
    mov rax, [rdi + VALUE_NUM_OFF]  ; a
    add rax, [rsi + VALUE_NUM_OFF]  ; a + b

    ; Pop one entry (result goes in second position, now top)
    sub rbx, VALUE_SIZE

    ; Store result in new top
    STACK_ENTRY rdi, 0
    mov byte [rdi + VALUE_TAG_OFF], VTAG_INTEGER
    mov [rdi + VALUE_NUM_OFF], rax
    mov qword [rdi + VALUE_PTR_OFF], 0

    jmp vm_dispatch

.add_type_error:
    jmp vm_error_type_mismatch

;-----------------------------------------------------------------------------
; op_sub_handler - Integer subtraction: ( a b -- a-b )
; Pops two integers, pushes their difference
;-----------------------------------------------------------------------------
global op_sub_handler
op_sub_handler:
    CHECK_STACK_MIN 2

    STACK_ENTRY rsi, 0              ; rsi -> b (top)
    STACK_ENTRY rdi, 1              ; rdi -> a (second)

    ; Check both are integers
    movzx eax, byte [rsi + VALUE_TAG_OFF]
    cmp al, VTAG_INTEGER
    jne .sub_type_error
    movzx eax, byte [rdi + VALUE_TAG_OFF]
    cmp al, VTAG_INTEGER
    jne .sub_type_error

    ; Load values and subtract
    mov rax, [rdi + VALUE_NUM_OFF]  ; a
    sub rax, [rsi + VALUE_NUM_OFF]  ; a - b

    ; Pop one entry
    sub rbx, VALUE_SIZE

    ; Store result
    STACK_ENTRY rdi, 0
    mov byte [rdi + VALUE_TAG_OFF], VTAG_INTEGER
    mov [rdi + VALUE_NUM_OFF], rax
    mov qword [rdi + VALUE_PTR_OFF], 0

    jmp vm_dispatch

.sub_type_error:
    jmp vm_error_type_mismatch

;-----------------------------------------------------------------------------
; op_mul_handler - Integer multiplication: ( a b -- a*b )
; Pops two integers, pushes their product
;-----------------------------------------------------------------------------
global op_mul_handler
op_mul_handler:
    CHECK_STACK_MIN 2

    STACK_ENTRY rsi, 0              ; rsi -> b (top)
    STACK_ENTRY rdi, 1              ; rdi -> a (second)

    ; Check both are integers
    movzx eax, byte [rsi + VALUE_TAG_OFF]
    cmp al, VTAG_INTEGER
    jne .mul_type_error
    movzx eax, byte [rdi + VALUE_TAG_OFF]
    cmp al, VTAG_INTEGER
    jne .mul_type_error

    ; Load values and multiply
    mov rax, [rdi + VALUE_NUM_OFF]  ; a
    imul rax, [rsi + VALUE_NUM_OFF] ; a * b

    ; Pop one entry
    sub rbx, VALUE_SIZE

    ; Store result
    STACK_ENTRY rdi, 0
    mov byte [rdi + VALUE_TAG_OFF], VTAG_INTEGER
    mov [rdi + VALUE_NUM_OFF], rax
    mov qword [rdi + VALUE_PTR_OFF], 0

    jmp vm_dispatch

.mul_type_error:
    jmp vm_error_type_mismatch

;-----------------------------------------------------------------------------
; op_div_handler - Integer division: ( a b -- a/b )
; Pops two integers, pushes their quotient
; Returns error on division by zero
;-----------------------------------------------------------------------------
global op_div_handler
op_div_handler:
    CHECK_STACK_MIN 2

    STACK_ENTRY rsi, 0              ; rsi -> b (top, divisor)
    STACK_ENTRY rdi, 1              ; rdi -> a (second, dividend)

    ; Check both are integers
    movzx eax, byte [rsi + VALUE_TAG_OFF]
    cmp al, VTAG_INTEGER
    jne .div_type_error
    movzx eax, byte [rdi + VALUE_TAG_OFF]
    cmp al, VTAG_INTEGER
    jne .div_type_error

    ; Check for division by zero
    mov rcx, [rsi + VALUE_NUM_OFF]  ; b (divisor)
    test rcx, rcx
    jz vm_error_div_by_zero

    ; Load dividend and divide
    mov rax, [rdi + VALUE_NUM_OFF]  ; a (dividend)
    cqo                             ; sign-extend rax into rdx:rax
    idiv rcx                        ; rax = rdx:rax / rcx

    ; Pop one entry
    sub rbx, VALUE_SIZE

    ; Store result (quotient in rax)
    STACK_ENTRY rdi, 0
    mov byte [rdi + VALUE_TAG_OFF], VTAG_INTEGER
    mov [rdi + VALUE_NUM_OFF], rax
    mov qword [rdi + VALUE_PTR_OFF], 0

    jmp vm_dispatch

.div_type_error:
    jmp vm_error_type_mismatch

;-----------------------------------------------------------------------------
; op_mod_handler - Integer modulo: ( a b -- a%b )
; Pops two integers, pushes the remainder
; Returns error on division by zero
;-----------------------------------------------------------------------------
global op_mod_handler
op_mod_handler:
    CHECK_STACK_MIN 2

    STACK_ENTRY rsi, 0              ; rsi -> b (top, divisor)
    STACK_ENTRY rdi, 1              ; rdi -> a (second, dividend)

    ; Check both are integers
    movzx eax, byte [rsi + VALUE_TAG_OFF]
    cmp al, VTAG_INTEGER
    jne .mod_type_error
    movzx eax, byte [rdi + VALUE_TAG_OFF]
    cmp al, VTAG_INTEGER
    jne .mod_type_error

    ; Check for division by zero
    mov rcx, [rsi + VALUE_NUM_OFF]  ; b (divisor)
    test rcx, rcx
    jz vm_error_div_by_zero

    ; Load dividend and divide
    mov rax, [rdi + VALUE_NUM_OFF]  ; a (dividend)
    cqo                             ; sign-extend rax into rdx:rax
    idiv rcx                        ; rdx = rdx:rax % rcx

    ; Pop one entry
    sub rbx, VALUE_SIZE

    ; Store result (remainder in rdx)
    STACK_ENTRY rdi, 0
    mov byte [rdi + VALUE_TAG_OFF], VTAG_INTEGER
    mov [rdi + VALUE_NUM_OFF], rdx
    mov qword [rdi + VALUE_PTR_OFF], 0

    jmp vm_dispatch

.mod_type_error:
    jmp vm_error_type_mismatch

;-----------------------------------------------------------------------------
; op_neg_handler - Integer negation: ( a -- -a )
; Pops one integer, pushes its negation
;-----------------------------------------------------------------------------
global op_neg_handler
op_neg_handler:
    CHECK_STACK_MIN 1

    STACK_ENTRY rdi, 0              ; rdi -> a (top)

    ; Check it's an integer
    movzx eax, byte [rdi + VALUE_TAG_OFF]
    cmp al, VTAG_INTEGER
    jne .neg_type_error

    ; Negate in place
    mov rax, [rdi + VALUE_NUM_OFF]
    neg rax
    mov [rdi + VALUE_NUM_OFF], rax

    jmp vm_dispatch

.neg_type_error:
    jmp vm_error_type_mismatch

;-----------------------------------------------------------------------------
; op_abs_handler - Integer absolute value: ( a -- |a| )
; Pops one integer, pushes its absolute value
;-----------------------------------------------------------------------------
global op_abs_handler
op_abs_handler:
    CHECK_STACK_MIN 1

    STACK_ENTRY rdi, 0              ; rdi -> a (top)

    ; Check it's an integer
    movzx eax, byte [rdi + VALUE_TAG_OFF]
    cmp al, VTAG_INTEGER
    jne .abs_type_error

    ; Get absolute value
    mov rax, [rdi + VALUE_NUM_OFF]
    mov rcx, rax
    neg rcx                         ; rcx = -rax
    cmovs rcx, rax                  ; if original was negative, use negated
    ; Actually we want: if rax >= 0, keep rax; else use -rax
    ; cmovs sets rcx = rax if SF=1 (rcx was negative after neg, meaning rax was positive)
    ; Let's do it more clearly:
    mov rax, [rdi + VALUE_NUM_OFF]
    test rax, rax
    jns .abs_positive
    neg rax
.abs_positive:
    mov [rdi + VALUE_NUM_OFF], rax

    jmp vm_dispatch

.abs_type_error:
    jmp vm_error_type_mismatch

;-----------------------------------------------------------------------------
; op_inc_handler - Integer increment: ( a -- a+1 )
; Pops one integer, pushes a+1
;-----------------------------------------------------------------------------
global op_inc_handler
op_inc_handler:
    CHECK_STACK_MIN 1

    STACK_ENTRY rdi, 0              ; rdi -> a (top)

    ; Check it's an integer
    movzx eax, byte [rdi + VALUE_TAG_OFF]
    cmp al, VTAG_INTEGER
    jne .inc_type_error

    ; Increment in place
    inc qword [rdi + VALUE_NUM_OFF]

    jmp vm_dispatch

.inc_type_error:
    jmp vm_error_type_mismatch

;-----------------------------------------------------------------------------
; op_dec_handler - Integer decrement: ( a -- a-1 )
; Pops one integer, pushes a-1
;-----------------------------------------------------------------------------
global op_dec_handler
op_dec_handler:
    CHECK_STACK_MIN 1

    STACK_ENTRY rdi, 0              ; rdi -> a (top)

    ; Check it's an integer
    movzx eax, byte [rdi + VALUE_TAG_OFF]
    cmp al, VTAG_INTEGER
    jne .dec_type_error

    ; Decrement in place
    dec qword [rdi + VALUE_NUM_OFF]

    jmp vm_dispatch

.dec_type_error:
    jmp vm_error_type_mismatch

; Mark stack as non-executable (security best practice)
section .note.GNU-stack noalloc noexec nowrite progbits
