; double.asm - IEEE 754 double-precision floating point operations
; DTRules Assembly Implementation
; Uses SSE2 instructions

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"

extern state

section .data
    align 16
    ; Constants for floating point operations
    const_zero:     dq 0.0
    const_one:      dq 1.0
    const_neg_one:  dq -1.0
    const_half:     dq 0.5
    const_two:      dq 2.0
    const_ten:      dq 10.0

    ; Masks
    sign_mask:      dq 0x8000000000000000
    abs_mask:       dq 0x7FFFFFFFFFFFFFFF

    ; Special values
    const_nan:      dq 0x7FF8000000000000    ; Quiet NaN
    const_inf:      dq 0x7FF0000000000000    ; Positive infinity
    const_neg_inf:  dq 0xFFF0000000000000    ; Negative infinity

section .text

;-----------------------------------------------------------------------------
; double_add - Add two doubles
; Input: xmm0 = a, xmm1 = b
; Output: xmm0 = a + b
;-----------------------------------------------------------------------------
global double_add
double_add:
    addsd xmm0, xmm1
    ret

;-----------------------------------------------------------------------------
; double_sub - Subtract two doubles
; Input: xmm0 = a, xmm1 = b
; Output: xmm0 = a - b
;-----------------------------------------------------------------------------
global double_sub
double_sub:
    subsd xmm0, xmm1
    ret

;-----------------------------------------------------------------------------
; double_mul - Multiply two doubles
; Input: xmm0 = a, xmm1 = b
; Output: xmm0 = a * b
;-----------------------------------------------------------------------------
global double_mul
double_mul:
    mulsd xmm0, xmm1
    ret

;-----------------------------------------------------------------------------
; double_div - Divide two doubles
; Input: xmm0 = a, xmm1 = b
; Output: xmm0 = a / b
;-----------------------------------------------------------------------------
global double_div
double_div:
    divsd xmm0, xmm1
    ret

;-----------------------------------------------------------------------------
; double_neg - Negate a double
; Input: xmm0 = value
; Output: xmm0 = -value
;-----------------------------------------------------------------------------
global double_neg
double_neg:
    movsd xmm1, [sign_mask]
    xorpd xmm0, xmm1
    ret

;-----------------------------------------------------------------------------
; double_abs - Absolute value
; Input: xmm0 = value
; Output: xmm0 = |value|
;-----------------------------------------------------------------------------
global double_abs
double_abs:
    movsd xmm1, [abs_mask]
    andpd xmm0, xmm1
    ret

;-----------------------------------------------------------------------------
; double_min - Minimum of two doubles
; Input: xmm0 = a, xmm1 = b
; Output: xmm0 = min(a, b)
;-----------------------------------------------------------------------------
global double_min
double_min:
    minsd xmm0, xmm1
    ret

;-----------------------------------------------------------------------------
; double_max - Maximum of two doubles
; Input: xmm0 = a, xmm1 = b
; Output: xmm0 = max(a, b)
;-----------------------------------------------------------------------------
global double_max
double_max:
    maxsd xmm0, xmm1
    ret

;-----------------------------------------------------------------------------
; double_floor - Floor (round toward negative infinity)
; Input: xmm0 = value
; Output: xmm0 = floor(value)
;-----------------------------------------------------------------------------
global double_floor
double_floor:
    ; SSE4.1 roundsd with mode 1 (floor)
    ; For older CPUs, we'd need a different approach
    roundsd xmm0, xmm0, 1
    ret

;-----------------------------------------------------------------------------
; double_ceil - Ceiling (round toward positive infinity)
; Input: xmm0 = value
; Output: xmm0 = ceil(value)
;-----------------------------------------------------------------------------
global double_ceil
double_ceil:
    roundsd xmm0, xmm0, 2
    ret

;-----------------------------------------------------------------------------
; double_round - Round to nearest integer
; Input: xmm0 = value
; Output: xmm0 = round(value)
;-----------------------------------------------------------------------------
global double_round
double_round:
    roundsd xmm0, xmm0, 0   ; Round to nearest even
    ret

;-----------------------------------------------------------------------------
; double_truncate - Truncate toward zero
; Input: xmm0 = value
; Output: xmm0 = trunc(value)
;-----------------------------------------------------------------------------
global double_truncate
double_truncate:
    roundsd xmm0, xmm0, 3   ; Truncate
    ret

;-----------------------------------------------------------------------------
; double_sqrt - Square root
; Input: xmm0 = value
; Output: xmm0 = sqrt(value)
;-----------------------------------------------------------------------------
global double_sqrt
double_sqrt:
    sqrtsd xmm0, xmm0
    ret

;-----------------------------------------------------------------------------
; double_pow - Power function
; Input: xmm0 = base, xmm1 = exponent
; Output: xmm0 = base^exponent
; Note: Uses logarithm method, handles integer exponents specially
;-----------------------------------------------------------------------------
global double_pow
double_pow:
    push rbp
    mov rbp, rsp
    sub rsp, 32

    ; Save inputs
    movsd [rsp], xmm0       ; base
    movsd [rsp + 8], xmm1   ; exponent

    ; Check if exponent is small integer
    cvttsd2si rax, xmm1
    cvtsi2sd xmm2, rax
    ucomisd xmm1, xmm2
    jne .use_log            ; Not an integer

    ; Integer exponent - use repeated multiplication
    test rax, rax
    jz .exp_zero
    js .neg_exp

    ; Positive integer exponent
    movsd xmm0, [const_one]
    movsd xmm1, [rsp]       ; base

.pos_loop:
    test rax, 1
    jz .even
    mulsd xmm0, xmm1

.even:
    mulsd xmm1, xmm1
    shr rax, 1
    jnz .pos_loop
    jmp .done

.exp_zero:
    movsd xmm0, [const_one]
    jmp .done

.neg_exp:
    ; base^(-n) = 1 / base^n
    neg rax
    movsd xmm0, [const_one]
    movsd xmm1, [rsp]

.neg_loop:
    test rax, 1
    jz .neg_even
    mulsd xmm0, xmm1

.neg_even:
    mulsd xmm1, xmm1
    shr rax, 1
    jnz .neg_loop

    movsd xmm1, xmm0
    movsd xmm0, [const_one]
    divsd xmm0, xmm1
    jmp .done

.use_log:
    ; General case: base^exp = exp(exp * ln(base))
    ; This requires math library functions
    ; For now, return NaN for non-integer exponents
    movsd xmm0, [const_nan]

.done:
    add rsp, 32
    pop rbp
    ret

;-----------------------------------------------------------------------------
; double_compare - Compare two doubles
; Input: xmm0 = a, xmm1 = b
; Output: rax = -1 if a < b, 0 if equal, 1 if a > b, 2 if NaN
;-----------------------------------------------------------------------------
global double_compare
double_compare:
    ucomisd xmm0, xmm1
    jp .nan                 ; Unordered (NaN)
    je .equal
    jb .less
    mov eax, 1
    ret
.less:
    mov eax, -1
    ret
.equal:
    xor eax, eax
    ret
.nan:
    mov eax, 2
    ret

;-----------------------------------------------------------------------------
; double_is_nan - Check if value is NaN
; Input: xmm0 = value
; Output: rax = 1 if NaN, 0 otherwise
;-----------------------------------------------------------------------------
global double_is_nan
double_is_nan:
    ucomisd xmm0, xmm0
    setp al                 ; Parity flag set if NaN
    movzx eax, al
    ret

;-----------------------------------------------------------------------------
; double_is_inf - Check if value is infinite
; Input: xmm0 = value
; Output: rax = 1 if infinite, 0 otherwise
;-----------------------------------------------------------------------------
global double_is_inf
double_is_inf:
    movq rax, xmm0
    and rax, [abs_mask]
    cmp rax, [const_inf]
    sete al
    movzx eax, al
    ret

;-----------------------------------------------------------------------------
; double_is_finite - Check if value is finite
; Input: xmm0 = value
; Output: rax = 1 if finite, 0 otherwise
;-----------------------------------------------------------------------------
global double_is_finite
double_is_finite:
    movq rax, xmm0
    and rax, [abs_mask]
    cmp rax, [const_inf]
    setb al
    movzx eax, al
    ret

;-----------------------------------------------------------------------------
; double_sign - Sign of double
; Input: xmm0 = value
; Output: rax = -1, 0, or 1
;-----------------------------------------------------------------------------
global double_sign
double_sign:
    xorpd xmm1, xmm1        ; Zero
    ucomisd xmm0, xmm1
    jp .nan
    je .zero
    ja .positive
    mov eax, -1
    ret
.positive:
    mov eax, 1
    ret
.zero:
.nan:
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; double_from_integer - Convert integer to double
; Input: rdi = integer
; Output: xmm0 = double
;-----------------------------------------------------------------------------
global double_from_integer
double_from_integer:
    cvtsi2sd xmm0, rdi
    ret

;-----------------------------------------------------------------------------
; double_to_integer - Convert double to integer (truncate)
; Input: xmm0 = double
; Output: rax = integer
;-----------------------------------------------------------------------------
global double_to_integer
double_to_integer:
    cvttsd2si rax, xmm0
    ret

;-----------------------------------------------------------------------------
; double_to_integer_round - Convert double to integer (round)
; Input: xmm0 = double
; Output: rax = integer
;-----------------------------------------------------------------------------
global double_to_integer_round
double_to_integer_round:
    roundsd xmm0, xmm0, 0
    cvtsd2si rax, xmm0
    ret

;-----------------------------------------------------------------------------
; double_fmod - Floating point modulo
; Input: xmm0 = a, xmm1 = b
; Output: xmm0 = a mod b
;-----------------------------------------------------------------------------
global double_fmod
double_fmod:
    ; a mod b = a - floor(a/b) * b
    movsd xmm2, xmm0        ; Save a
    divsd xmm0, xmm1        ; a / b
    roundsd xmm0, xmm0, 1   ; floor(a/b)
    mulsd xmm0, xmm1        ; floor(a/b) * b
    movsd xmm1, xmm2
    subsd xmm1, xmm0        ; a - floor(a/b) * b
    movsd xmm0, xmm1
    ret
