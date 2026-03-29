; op_math.asm - Mathematical operators
; DTRules Assembly Implementation
; Arithmetic and mathematical functions

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"

extern state
extern stack_data_pop
extern stack_data_peek
extern stack_data_push_integer
extern stack_data_push_double
extern double_from_integer
extern double_to_integer
extern integer_abs
extern integer_min
extern integer_max
extern integer_pow

section .data
    align 16
    const_pi:       dq 3.14159265358979323846
    const_e:        dq 2.71828182845904523536
    const_ln2:      dq 0.693147180559945309417
    const_ln10:     dq 2.302585092994045684018

section .text

;-----------------------------------------------------------------------------
; Helper: Get two numeric operands
; Output: xmm0 = first (deeper), xmm1 = second (top)
;         rax = type (0 = both int, 1 = has double)
;-----------------------------------------------------------------------------
get_two_numerics:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    sub rsp, 16

    ; Pop second operand
    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, rax

    ; Pop first operand
    call stack_data_pop
    test rax, rax
    jz .error
    mov r12, rax

    ; Check types
    movzx ecx, byte [rbx + VALUE_TAG_OFF]
    movzx edx, byte [r12 + VALUE_TAG_OFF]

    ; Both integers?
    cmp cl, VTAG_INTEGER
    jne .check_first_double
    cmp dl, VTAG_INTEGER
    jne .first_int_second_other

    ; Both integers - return as integers
    mov rax, [r12 + VALUE_NUM_OFF]
    mov [rsp], rax
    mov rax, [rbx + VALUE_NUM_OFF]
    mov [rsp + 8], rax
    xor eax, eax            ; Type 0 = both int
    jmp .done

.first_int_second_other:
    cmp dl, VTAG_DOUBLE
    jne .type_error

    ; First is int, second is double - convert first
    mov rdi, [r12 + VALUE_NUM_OFF]
    call double_from_integer
    movsd [rsp], xmm0

    movsd xmm1, [rbx + VALUE_NUM_OFF]
    movsd [rsp + 8], xmm1
    mov eax, 1              ; Has double
    jmp .done

.check_first_double:
    cmp cl, VTAG_DOUBLE
    jne .type_error

    cmp dl, VTAG_INTEGER
    je .first_double_second_int
    cmp dl, VTAG_DOUBLE
    jne .type_error

    ; Both doubles
    movsd xmm0, [r12 + VALUE_NUM_OFF]
    movsd xmm1, [rbx + VALUE_NUM_OFF]
    movsd [rsp], xmm0
    movsd [rsp + 8], xmm1
    mov eax, 1
    jmp .done

.first_double_second_int:
    ; First is double, second is int
    movsd xmm0, [r12 + VALUE_NUM_OFF]
    movsd [rsp], xmm0

    mov rdi, [rbx + VALUE_NUM_OFF]
    call double_from_integer
    movsd [rsp + 8], xmm0
    mov eax, 1
    jmp .done

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
    mov eax, -1
    jmp .exit

.error:
    mov eax, -1

.done:
    ; Load results
    push rax
    movsd xmm0, [rsp + 8]
    movsd xmm1, [rsp + 16]
    pop rax

.exit:
    add rsp, 16
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_idiv - Integer division
; Stack: a b -- a/b
;-----------------------------------------------------------------------------
global op_idiv
op_idiv:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error
    mov rcx, rax            ; Divisor

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax            ; Dividend

    ; Must be integers
    cmp byte [rcx + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error
    cmp byte [rdx + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error

    ; Check division by zero
    mov rsi, [rcx + VALUE_NUM_OFF]
    test rsi, rsi
    jz .div_zero

    ; Divide
    mov rax, [rdx + VALUE_NUM_OFF]
    cqo
    idiv rsi
    mov rdi, rax
    call stack_data_push_integer

    xor eax, eax
    jmp .done

.div_zero:
    mov dword [state + State.error], ERR_DIV_BY_ZERO
    jmp .error

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH

.error:
    mov eax, [state + State.error]

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_imod - Integer modulo
; Stack: a b -- a%b
;-----------------------------------------------------------------------------
global op_imod
op_imod:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error
    mov rcx, rax

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax

    cmp byte [rcx + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error
    cmp byte [rdx + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error

    mov rsi, [rcx + VALUE_NUM_OFF]
    test rsi, rsi
    jz .div_zero

    mov rax, [rdx + VALUE_NUM_OFF]
    cqo
    idiv rsi
    mov rdi, rdx            ; Remainder
    call stack_data_push_integer

    xor eax, eax
    jmp .done

.div_zero:
    mov dword [state + State.error], ERR_DIV_BY_ZERO
    jmp .error

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH

.error:
    mov eax, [state + State.error]

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_inc - Increment top of stack
; Stack: n -- n+1
;-----------------------------------------------------------------------------
global op_inc
op_inc:
    call stack_data_peek
    test rax, rax
    jz .error

    cmp byte [rax + VALUE_TAG_OFF], VTAG_INTEGER
    jne .try_double

    inc qword [rax + VALUE_NUM_OFF]
    xor eax, eax
    ret

.try_double:
    cmp byte [rax + VALUE_TAG_OFF], VTAG_DOUBLE
    jne .type_error

    movsd xmm0, [rax + VALUE_NUM_OFF]
    mov rcx, 1
    cvtsi2sd xmm1, rcx
    addsd xmm0, xmm1
    movsd [rax + VALUE_NUM_OFF], xmm0
    xor eax, eax
    ret

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    mov eax, [state + State.error]
    ret

;-----------------------------------------------------------------------------
; op_dec - Decrement top of stack
; Stack: n -- n-1
;-----------------------------------------------------------------------------
global op_dec
op_dec:
    call stack_data_peek
    test rax, rax
    jz .error

    cmp byte [rax + VALUE_TAG_OFF], VTAG_INTEGER
    jne .try_double

    dec qword [rax + VALUE_NUM_OFF]
    xor eax, eax
    ret

.try_double:
    cmp byte [rax + VALUE_TAG_OFF], VTAG_DOUBLE
    jne .type_error

    movsd xmm0, [rax + VALUE_NUM_OFF]
    mov rcx, 1
    cvtsi2sd xmm1, rcx
    subsd xmm0, xmm1
    movsd [rax + VALUE_NUM_OFF], xmm0
    xor eax, eax
    ret

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    mov eax, [state + State.error]
    ret

;-----------------------------------------------------------------------------
; op_sqrt_impl - Square root
; Stack: n -- sqrt(n)
;-----------------------------------------------------------------------------
global op_sqrt_impl
op_sqrt_impl:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error

    movzx ecx, byte [rax + VALUE_TAG_OFF]

    cmp cl, VTAG_INTEGER
    jne .try_double

    ; Convert to double for sqrt
    mov rdi, [rax + VALUE_NUM_OFF]
    cvtsi2sd xmm0, rdi
    sqrtsd xmm0, xmm0
    call stack_data_push_double
    xor eax, eax
    jmp .done

.try_double:
    cmp cl, VTAG_DOUBLE
    jne .type_error

    movsd xmm0, [rax + VALUE_NUM_OFF]
    sqrtsd xmm0, xmm0
    call stack_data_push_double
    xor eax, eax
    jmp .done

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH

.error:
    mov eax, [state + State.error]

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_sin_impl - Sine (would need FPU or lookup table)
;-----------------------------------------------------------------------------
global op_sin_impl
op_sin_impl:
    ; TODO: Implement using Taylor series or lookup table
    ; For now, just return the input
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; op_cos_impl - Cosine
;-----------------------------------------------------------------------------
global op_cos_impl
op_cos_impl:
    ; TODO: Implement
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; op_exp_impl - e^x
;-----------------------------------------------------------------------------
global op_exp_impl
op_exp_impl:
    ; TODO: Implement
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; op_log_impl - Natural logarithm
;-----------------------------------------------------------------------------
global op_log_impl
op_log_impl:
    ; TODO: Implement
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; op_log10_impl - Base-10 logarithm
;-----------------------------------------------------------------------------
global op_log10_impl
op_log10_impl:
    ; TODO: Implement
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; op_random - Push random number between 0 and 1
;-----------------------------------------------------------------------------
global op_random
op_random:
    push rbp
    mov rbp, rsp

    ; Simple LCG random - not cryptographically secure
    ; Uses rdtsc for seed if not initialized
    rdtsc
    shl rdx, 32
    or rax, rdx

    ; LCG: next = (a * current + c) mod m
    ; Using parameters from Numerical Recipes
    mov rcx, 6364136223846793005
    imul rax, rcx
    add rax, 1442695040888963407

    ; Convert to double in [0, 1)
    shr rax, 12             ; Keep 52 bits
    mov rcx, 0x3FF0000000000000  ; Exponent for 1.0
    or rax, rcx
    movq xmm0, rax
    mov rcx, 0x3FF0000000000000
    movq xmm1, rcx
    subsd xmm0, xmm1        ; Now in [0, 1)

    call stack_data_push_double

    xor eax, eax
    pop rbp
    ret
