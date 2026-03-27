; stack_return.asm - Return stack operations
; DTRules Assembly Implementation
; Return stack for loop indices (i, j, k operators)

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state
extern heap_alloc

;-----------------------------------------------------------------------------
; Return stack layout
; Each entry is a pointer to a Value (8 bytes)
; Stack grows downward like data stack
;-----------------------------------------------------------------------------

RETURN_STACK_SIZE equ 256    ; Max depth for nested loops

section .bss
    align 8
    return_stack:       resq RETURN_STACK_SIZE  ; Array of Value pointers
    return_stack_base:  resq 1                  ; Base (bottom) of stack
    return_stack_ptr:   resq 1                  ; Current top of stack
    return_stack_end:   resq 1                  ; End (limit) of stack

section .text

;-----------------------------------------------------------------------------
; stack_return_init - Initialize the return stack
; Call this during VM initialization
;-----------------------------------------------------------------------------
global stack_return_init
stack_return_init:
    ; Set up pointers
    lea rax, [return_stack + RETURN_STACK_SIZE * 8]
    mov [return_stack_base], rax
    mov [return_stack_ptr], rax     ; Empty stack

    lea rax, [return_stack]
    mov [return_stack_end], rax

    ret

;-----------------------------------------------------------------------------
; stack_return_push - Push a Value pointer onto return stack
; Input: rdi = pointer to Value
; Returns: 0 on success, error code on failure
;-----------------------------------------------------------------------------
global stack_return_push
stack_return_push:
    ; Check for overflow
    mov rax, [return_stack_ptr]
    sub rax, 8
    cmp rax, [return_stack_end]
    jb .overflow

    ; Push the pointer
    mov [return_stack_ptr], rax
    mov [rax], rdi

    xor eax, eax
    ret

.overflow:
    mov dword [state + State.error], ERR_STACK_OVERFLOW
    mov eax, ERR_STACK_OVERFLOW
    ret

;-----------------------------------------------------------------------------
; stack_return_pop - Pop a Value pointer from return stack
; Returns: rax = Value pointer, or 0 if empty
;-----------------------------------------------------------------------------
global stack_return_pop
stack_return_pop:
    mov rax, [return_stack_ptr]
    cmp rax, [return_stack_base]
    jae .empty

    ; Pop the pointer
    mov rax, [rax]
    mov rcx, [return_stack_ptr]
    add rcx, 8
    mov [return_stack_ptr], rcx

    ret

.empty:
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; stack_return_peek - Peek at Value pointer at given depth
; Input: rdi = depth (0 = top, 1 = second, etc.)
; Returns: rax = Value pointer, or 0 if depth exceeds stack
;-----------------------------------------------------------------------------
global stack_return_peek
stack_return_peek:
    ; Calculate offset from current top
    mov rax, [return_stack_ptr]
    shl rdi, 3              ; depth * 8
    add rax, rdi

    ; Check bounds
    cmp rax, [return_stack_base]
    jae .out_of_bounds

    ; Return the value at that depth
    mov rax, [rax]
    ret

.out_of_bounds:
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; stack_return_depth - Get current return stack depth
; Returns: rax = number of items on stack
;-----------------------------------------------------------------------------
global stack_return_depth
stack_return_depth:
    mov rax, [return_stack_base]
    sub rax, [return_stack_ptr]
    shr rax, 3              ; Divide by 8 (pointer size)
    ret

;-----------------------------------------------------------------------------
; stack_return_clear - Clear the return stack
;-----------------------------------------------------------------------------
global stack_return_clear
stack_return_clear:
    mov rax, [return_stack_base]
    mov [return_stack_ptr], rax
    ret
