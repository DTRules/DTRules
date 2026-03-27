; integer.asm - 64-bit integer operations
; DTRules Assembly Implementation

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"

extern state
extern value_new_integer
extern heap_alloc

section .text

;-----------------------------------------------------------------------------
; integer_add - Add two integers
; Input: rdi = a, rsi = b
; Output: rax = a + b
;-----------------------------------------------------------------------------
global integer_add
integer_add:
    lea rax, [rdi + rsi]
    ret

;-----------------------------------------------------------------------------
; integer_sub - Subtract two integers
; Input: rdi = a, rsi = b
; Output: rax = a - b
;-----------------------------------------------------------------------------
global integer_sub
integer_sub:
    mov rax, rdi
    sub rax, rsi
    ret

;-----------------------------------------------------------------------------
; integer_mul - Multiply two integers
; Input: rdi = a, rsi = b
; Output: rax = a * b
;-----------------------------------------------------------------------------
global integer_mul
integer_mul:
    mov rax, rdi
    imul rax, rsi
    ret

;-----------------------------------------------------------------------------
; integer_div - Divide two integers
; Input: rdi = a, rsi = b
; Output: rax = a / b (truncated toward zero)
; Note: Caller must check for division by zero
;-----------------------------------------------------------------------------
global integer_div
integer_div:
    mov rax, rdi
    cqo                     ; Sign-extend rax into rdx:rax
    idiv rsi
    ret

;-----------------------------------------------------------------------------
; integer_mod - Modulo of two integers
; Input: rdi = a, rsi = b
; Output: rax = a % b
; Note: Caller must check for division by zero
;-----------------------------------------------------------------------------
global integer_mod
integer_mod:
    mov rax, rdi
    cqo
    idiv rsi
    mov rax, rdx            ; Remainder
    ret

;-----------------------------------------------------------------------------
; integer_neg - Negate an integer
; Input: rdi = value
; Output: rax = -value
;-----------------------------------------------------------------------------
global integer_neg
integer_neg:
    mov rax, rdi
    neg rax
    ret

;-----------------------------------------------------------------------------
; integer_abs - Absolute value
; Input: rdi = value
; Output: rax = |value|
;-----------------------------------------------------------------------------
global integer_abs
integer_abs:
    mov rax, rdi
    test rax, rax
    jns .done
    neg rax
.done:
    ret

;-----------------------------------------------------------------------------
; integer_min - Minimum of two integers
; Input: rdi = a, rsi = b
; Output: rax = min(a, b)
;-----------------------------------------------------------------------------
global integer_min
integer_min:
    mov rax, rdi
    cmp rax, rsi
    cmovg rax, rsi
    ret

;-----------------------------------------------------------------------------
; integer_max - Maximum of two integers
; Input: rdi = a, rsi = b
; Output: rax = max(a, b)
;-----------------------------------------------------------------------------
global integer_max
integer_max:
    mov rax, rdi
    cmp rax, rsi
    cmovl rax, rsi
    ret

;-----------------------------------------------------------------------------
; integer_sign - Sign of integer
; Input: rdi = value
; Output: rax = -1, 0, or 1
;-----------------------------------------------------------------------------
global integer_sign
integer_sign:
    xor eax, eax
    test rdi, rdi
    je .done
    sets al                 ; Set al to 1 if negative
    neg eax                 ; -1 if negative
    jns .done
    mov eax, 1              ; 1 if positive (sign flag not set)
.done:
    ret

;-----------------------------------------------------------------------------
; integer_pow - Power (a^b)
; Input: rdi = base, rsi = exponent (non-negative)
; Output: rax = base^exponent
;-----------------------------------------------------------------------------
global integer_pow
integer_pow:
    push rbx

    ; Handle special cases
    test rsi, rsi
    jz .exp_zero            ; x^0 = 1

    mov rax, 1              ; Result
    mov rbx, rdi            ; Base

.loop:
    test rsi, 1
    jz .even
    imul rax, rbx           ; result *= base

.even:
    imul rbx, rbx           ; base *= base
    shr rsi, 1              ; exponent /= 2
    jnz .loop

    pop rbx
    ret

.exp_zero:
    mov eax, 1
    pop rbx
    ret

;-----------------------------------------------------------------------------
; integer_gcd - Greatest common divisor
; Input: rdi = a, rsi = b
; Output: rax = gcd(a, b)
;-----------------------------------------------------------------------------
global integer_gcd
integer_gcd:
    ; Make both positive
    mov rax, rdi
    test rax, rax
    jns .a_pos
    neg rax
.a_pos:
    mov rcx, rsi
    test rcx, rcx
    jns .b_pos
    neg rcx
.b_pos:
    ; Euclidean algorithm
.loop:
    test rcx, rcx
    jz .done
    xor edx, edx
    div rcx
    mov rax, rcx
    mov rcx, rdx
    jmp .loop

.done:
    ret

;-----------------------------------------------------------------------------
; integer_to_string - Convert integer to string
; Input: rdi = integer value
; Output: rax = pointer to string Value
;-----------------------------------------------------------------------------
global integer_to_string
integer_to_string:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    sub rsp, 32             ; Buffer space

    mov r12, rdi            ; Save integer
    lea rbx, [rsp + 31]     ; End of buffer
    mov byte [rbx], 0       ; Null terminator

    ; Handle negative
    xor ecx, ecx            ; Negative flag
    test r12, r12
    jns .positive
    neg r12
    mov ecx, 1

.positive:
    ; Handle zero
    test r12, r12
    jnz .convert
    dec rbx
    mov byte [rbx], '0'
    jmp .check_neg

.convert:
    mov rax, r12
    mov r8, 10

.digit_loop:
    test rax, rax
    jz .check_neg
    xor edx, edx
    div r8
    add dl, '0'
    dec rbx
    mov [rbx], dl
    jmp .digit_loop

.check_neg:
    test ecx, ecx
    jz .make_string
    dec rbx
    mov byte [rbx], '-'

.make_string:
    ; Calculate length
    lea rax, [rsp + 31]
    sub rax, rbx

    ; Create string value
    mov rdi, rbx
    mov rsi, rax
    ; TODO: Call value_new_string

    add rsp, 32
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; integer_from_string - Parse integer from string
; Input: rdi = string pointer, rsi = length
; Output: rax = parsed integer, rdx = 0 on success, non-zero on error
;-----------------------------------------------------------------------------
global integer_from_string
integer_from_string:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    mov rbx, rdi            ; String pointer
    mov r12, rsi            ; Length

    xor eax, eax            ; Result
    xor ecx, ecx            ; Index
    xor r8d, r8d            ; Negative flag

    ; Skip leading whitespace
.skip_space:
    cmp rcx, r12
    jae .done
    movzx edx, byte [rbx + rcx]
    cmp dl, ' '
    je .next_space
    cmp dl, 9               ; Tab
    je .next_space
    jmp .check_sign

.next_space:
    inc rcx
    jmp .skip_space

.check_sign:
    cmp dl, '-'
    jne .check_plus
    mov r8d, 1
    inc rcx
    jmp .parse

.check_plus:
    cmp dl, '+'
    jne .parse
    inc rcx

.parse:
    cmp rcx, r12
    jae .check_empty

.digit_loop:
    cmp rcx, r12
    jae .apply_sign

    movzx edx, byte [rbx + rcx]
    sub dl, '0'
    cmp dl, 9
    ja .error               ; Not a digit

    imul rax, 10
    movzx edx, dl
    add rax, rdx
    inc rcx
    jmp .digit_loop

.apply_sign:
    test r8d, r8d
    jz .done
    neg rax

.done:
    xor edx, edx            ; Success
    jmp .exit

.check_empty:
    ; No digits found
.error:
    xor eax, eax
    mov edx, 1              ; Error

.exit:
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; integer_compare - Compare two integers
; Input: rdi = a, rsi = b
; Output: rax = -1 if a < b, 0 if equal, 1 if a > b
;-----------------------------------------------------------------------------
global integer_compare
integer_compare:
    xor eax, eax
    cmp rdi, rsi
    je .done
    setl al
    neg eax
    jns .done
    mov eax, 1
.done:
    ret
