; DTRules NASM VM Boolean Operations
; Copyright 2024 Paul Snow
; Licensed under Apache License 2.0
;
; This file implements boolean operations for the DTRules NASM VM:
;   - OP_AND (40): Logical AND of top two booleans
;   - OP_OR  (41): Logical OR of top two booleans
;   - OP_NOT (42): Logical NOT of top boolean
;   - OP_XOR (43): Logical XOR of top two booleans
;   - OP_ISNULL (44): Push true if top is null, false otherwise
;   - OP_NOTNULL (45): Push true if top is not null, false otherwise
;   - OP_BEQ (46): Boolean equals - push true if both booleans are equal
;
; All operations follow the jump table dispatch pattern:
;   - Each handler ends with jmp vm_dispatch
;   - Handlers update VMContext.data_sp directly
;   - Error conditions jump to vm_error_return

%include "vm_constants.inc"
%include "vm_state.inc"

; External symbols from vm_entry.asm
extern vm_dispatch
extern vm_error_return

section .text

; ============================================================================
; OP_AND (40) - Logical AND
; ============================================================================
; Pops two values, pushes (a && b) as boolean
; Stack: a b -> (a && b)
;
global op_and
op_and:
    ; Get stack pointer
    mov rax, [r12 + VMContext.data_sp]

    ; Need at least 2 elements
    sub rax, VALUE_SIZE * 2
    cmp rax, [r12 + VMContext.data_base]
    jl .underflow

    ; Get both values - addresses are now in rax (a) and rax+VALUE_SIZE (b)
    ; a is at rax, b is at rax + VALUE_SIZE
    ; Read a.num (truthy if non-zero)
    mov rbx, [rax + Value.num]

    ; Read b.num
    mov rcx, [rax + VALUE_SIZE + Value.num]

    ; Compute AND: both must be non-zero
    test rbx, rbx
    jz .result_false
    test rcx, rcx
    jz .result_false

    ; Result is true
    mov qword [rax + Value.num], 1
    jmp .store_result

.result_false:
    mov qword [rax + Value.num], 0

.store_result:
    ; Set tag to boolean, clear ptr
    mov byte [rax + Value.tag], VTAG_BOOLEAN
    mov qword [rax + Value.ptr], 0

    ; Update stack pointer (pop one, result in place of first)
    add rax, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rax

    jmp vm_dispatch

.underflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_UNDERFLOW
    jmp vm_error_return


; ============================================================================
; OP_OR (41) - Logical OR
; ============================================================================
; Pops two values, pushes (a || b) as boolean
; Stack: a b -> (a || b)
;
global op_or
op_or:
    ; Get stack pointer
    mov rax, [r12 + VMContext.data_sp]

    ; Need at least 2 elements
    sub rax, VALUE_SIZE * 2
    cmp rax, [r12 + VMContext.data_base]
    jl .underflow

    ; Read a.num and b.num
    mov rbx, [rax + Value.num]
    mov rcx, [rax + VALUE_SIZE + Value.num]

    ; Compute OR: either non-zero
    test rbx, rbx
    jnz .result_true
    test rcx, rcx
    jnz .result_true

    ; Result is false
    mov qword [rax + Value.num], 0
    jmp .store_result

.result_true:
    mov qword [rax + Value.num], 1

.store_result:
    ; Set tag to boolean, clear ptr
    mov byte [rax + Value.tag], VTAG_BOOLEAN
    mov qword [rax + Value.ptr], 0

    ; Update stack pointer
    add rax, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rax

    jmp vm_dispatch

.underflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_UNDERFLOW
    jmp vm_error_return


; ============================================================================
; OP_NOT (42) - Logical NOT
; ============================================================================
; Pops one value, pushes !a as boolean
; Stack: a -> (!a)
;
global op_not
op_not:
    ; Get stack pointer
    mov rax, [r12 + VMContext.data_sp]

    ; Need at least 1 element
    sub rax, VALUE_SIZE
    cmp rax, [r12 + VMContext.data_base]
    jl .underflow

    ; Read a.num
    mov rbx, [rax + Value.num]

    ; Compute NOT: result is 1 if input is 0, else 0
    test rbx, rbx
    setz cl                          ; cl = 1 if rbx == 0
    movzx rcx, cl                    ; zero-extend to rcx

    ; Store result in place
    mov qword [rax + Value.num], rcx
    mov byte [rax + Value.tag], VTAG_BOOLEAN
    mov qword [rax + Value.ptr], 0

    ; Stack pointer unchanged (in-place operation)
    jmp vm_dispatch

.underflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_UNDERFLOW
    jmp vm_error_return


; ============================================================================
; OP_XOR (43) - Logical XOR
; ============================================================================
; Pops two values, pushes (a ^ b) as boolean
; Stack: a b -> (a XOR b) - true if exactly one is true
;
global op_xor
op_xor:
    ; Get stack pointer
    mov rax, [r12 + VMContext.data_sp]

    ; Need at least 2 elements
    sub rax, VALUE_SIZE * 2
    cmp rax, [r12 + VMContext.data_base]
    jl .underflow

    ; Read a.num and b.num
    mov rbx, [rax + Value.num]
    mov rcx, [rax + VALUE_SIZE + Value.num]

    ; Convert to boolean (0 or 1)
    test rbx, rbx
    setnz bl
    movzx rbx, bl

    test rcx, rcx
    setnz cl
    movzx rcx, cl

    ; XOR them
    xor rbx, rcx

    ; Store result
    mov qword [rax + Value.num], rbx
    mov byte [rax + Value.tag], VTAG_BOOLEAN
    mov qword [rax + Value.ptr], 0

    ; Update stack pointer
    add rax, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rax

    jmp vm_dispatch

.underflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_UNDERFLOW
    jmp vm_error_return


; ============================================================================
; OP_ISNULL (44) - Test for null
; ============================================================================
; Pops one value, pushes true if it was null, false otherwise
; Stack: a -> (a == null)
;
global op_isnull
op_isnull:
    ; Get stack pointer
    mov rax, [r12 + VMContext.data_sp]

    ; Need at least 1 element
    sub rax, VALUE_SIZE
    cmp rax, [r12 + VMContext.data_base]
    jl .underflow

    ; Read tag
    movzx ebx, byte [rax + Value.tag]

    ; Check if tag == VTAG_NULL
    cmp bl, VTAG_NULL
    sete cl                          ; cl = 1 if null
    movzx rcx, cl

    ; Store result in place
    mov qword [rax + Value.num], rcx
    mov byte [rax + Value.tag], VTAG_BOOLEAN
    mov qword [rax + Value.ptr], 0

    jmp vm_dispatch

.underflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_UNDERFLOW
    jmp vm_error_return


; ============================================================================
; OP_NOTNULL (45) - Test for not null
; ============================================================================
; Pops one value, pushes true if it was not null, false otherwise
; Stack: a -> (a != null)
;
global op_notnull
op_notnull:
    ; Get stack pointer
    mov rax, [r12 + VMContext.data_sp]

    ; Need at least 1 element
    sub rax, VALUE_SIZE
    cmp rax, [r12 + VMContext.data_base]
    jl .underflow

    ; Read tag
    movzx ebx, byte [rax + Value.tag]

    ; Check if tag != VTAG_NULL
    cmp bl, VTAG_NULL
    setne cl                         ; cl = 1 if not null
    movzx rcx, cl

    ; Store result in place
    mov qword [rax + Value.num], rcx
    mov byte [rax + Value.tag], VTAG_BOOLEAN
    mov qword [rax + Value.ptr], 0

    jmp vm_dispatch

.underflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_UNDERFLOW
    jmp vm_error_return


; ============================================================================
; OP_BEQ (46) - Boolean Equals
; ============================================================================
; Pops two boolean values, pushes true if they are equal
; Stack: a b -> (a == b)
;
global op_beq
op_beq:
    ; Get stack pointer
    mov rax, [r12 + VMContext.data_sp]

    ; Need at least 2 elements
    sub rax, VALUE_SIZE * 2
    cmp rax, [r12 + VMContext.data_base]
    jl .underflow

    ; Read a.num and b.num (as booleans, both are 0 or non-zero)
    mov rbx, [rax + Value.num]
    mov rcx, [rax + VALUE_SIZE + Value.num]

    ; Normalize to 0 or 1
    test rbx, rbx
    setnz bl
    movzx rbx, bl

    test rcx, rcx
    setnz cl
    movzx rcx, cl

    ; Compare
    cmp rbx, rcx
    sete dl                          ; dl = 1 if equal
    movzx rdx, dl

    ; Store result
    mov qword [rax + Value.num], rdx
    mov byte [rax + Value.tag], VTAG_BOOLEAN
    mov qword [rax + Value.ptr], 0

    ; Update stack pointer
    add rax, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rax

    jmp vm_dispatch

.underflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_UNDERFLOW
    jmp vm_error_return


; ============================================================================
; Mark stack as non-executable
; ============================================================================
section .note.GNU-stack noalloc noexec nowrite progbits
