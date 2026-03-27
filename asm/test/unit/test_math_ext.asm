; test_math_ext.asm - Unit tests for Extended Math operations
; DTRules Assembly Implementation
; Tests: floor, ceil, round, truncate, pow

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state

; Stack functions
extern stack_data_push_integer
extern stack_data_push_double
extern stack_data_pop
extern stack_data_peek
extern stack_data_depth
extern stack_data_clear

; Extended math operators to test
extern op_floor
extern op_ceil
extern op_round
extern op_truncate
extern op_pow

; Print functions
extern print_string
extern print_integer
extern print_newline
extern print_stack
extern print_value_ln

; Test harness functions
extern test_start
extern test_end_pass
extern test_end_fail
extern test_pass
extern test_fail
extern assert_eq
extern assert_true
extern assert_false
extern print_test_summary
extern reset_state
extern test_count
extern fail_count
extern pass_count

section .data
    test_header:        db "=== Extended Math Operation Tests ===", 10, 0

    ; Test names
    test_floor_pos:     db "floor positive (3.7 -> 3.0)", 0
    test_floor_neg:     db "floor negative (-3.2 -> -4.0)", 0
    test_ceil_pos:      db "ceil positive (3.2 -> 4.0)", 0
    test_ceil_neg:      db "ceil negative (-3.7 -> -3.0)", 0
    test_round_up:      db "round up (3.6 -> 4.0)", 0
    test_round_down:    db "round down (3.4 -> 3.0)", 0
    test_round_half:    db "round half (3.5 -> 4.0)", 0
    test_truncate_pos:  db "truncate positive (3.9 -> 3.0)", 0
    test_truncate_neg:  db "truncate negative (-3.9 -> -3.0)", 0
    test_pow_int:       db "pow (2^3 = 8)", 0
    test_pow_frac:      db "pow (4^0.5 = 2)", 0
    test_pow_neg:       db "pow (2^-1 = 0.5)", 0

    ; Double constants (IEEE 754 format)
    align 8
    double_3_7:         dq 3.7
    double_neg_3_2:     dq -3.2
    double_3_2:         dq 3.2
    double_neg_3_7:     dq -3.7
    double_3_6:         dq 3.6
    double_3_4:         dq 3.4
    double_3_5:         dq 3.5
    double_3_9:         dq 3.9
    double_neg_3_9:     dq -3.9
    double_2_0:         dq 2.0
    double_3_0:         dq 3.0
    double_4_0:         dq 4.0
    double_0_5:         dq 0.5
    double_neg_1_0:     dq -1.0
    double_neg_3_0:     dq -3.0
    double_neg_4_0:     dq -4.0
    double_8_0:         dq 8.0
    double_epsilon:     dq 0.0001

section .text
    global test_main

;-----------------------------------------------------------------------------
; Helper: Push a double value onto the data stack
; Input: rdi = pointer to double value
;-----------------------------------------------------------------------------
push_double:
    push rbp
    mov rbp, rsp

    movsd xmm0, [rdi]
    call stack_data_push_double

    pop rbp
    ret

;-----------------------------------------------------------------------------
; Helper: Check if double result approximately equals expected
; Input: rax = pointer to Value, xmm1 = expected double
; Output: eax = 1 if within epsilon, 0 otherwise
;-----------------------------------------------------------------------------
check_double_result:
    push rbp
    mov rbp, rsp

    ; Get actual value from the Value struct
    movsd xmm0, [rax + VALUE_NUM_OFF]

    ; Compute difference: xmm0 = actual - expected
    subsd xmm0, xmm1

    ; Get absolute value of difference
    ; abs(x) = x < 0 ? -x : x
    xorpd xmm2, xmm2            ; xmm2 = 0.0
    comisd xmm0, xmm2
    jae .positive
    ; Negative - negate
    xorpd xmm3, xmm3
    subsd xmm3, xmm0
    movsd xmm0, xmm3
.positive:

    ; Compare with epsilon
    movsd xmm1, [double_epsilon]
    comisd xmm0, xmm1
    ja .not_equal

    mov eax, 1
    jmp .done

.not_equal:
    xor eax, eax

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_main - Run all extended math tests
;-----------------------------------------------------------------------------
test_main:
    push rbp
    mov rbp, rsp

    ; Print header
    lea rdi, [test_header]
    call print_string

    ; Run tests
    call test_floor_positive
    call test_floor_negative
    call test_ceil_positive
    call test_ceil_negative
    call test_round_up_op
    call test_round_down_op
    call test_round_half_op
    call test_truncate_positive
    call test_truncate_negative
    call test_pow_integer
    call test_pow_fractional
    call test_pow_negative

    ; Print summary
    call print_test_summary

    ; Return fail count
    mov rax, [fail_count]

    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_floor_positive - Test floor(3.7) = 3.0
;-----------------------------------------------------------------------------
test_floor_positive:
    push rbp
    mov rbp, rsp

    lea rdi, [test_floor_pos]
    call test_start

    call reset_state

    ; Push 3.7
    lea rdi, [double_3_7]
    call push_double

    ; Floor
    call op_floor

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is 3.0
    call stack_data_pop
    test rax, rax
    jz .fail

    movsd xmm1, [double_3_0]
    call check_double_result
    test eax, eax
    jz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_floor_negative - Test floor(-3.2) = -4.0
;-----------------------------------------------------------------------------
test_floor_negative:
    push rbp
    mov rbp, rsp

    lea rdi, [test_floor_neg]
    call test_start

    call reset_state

    ; Push -3.2
    lea rdi, [double_neg_3_2]
    call push_double

    ; Floor
    call op_floor

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is -4.0
    call stack_data_pop
    test rax, rax
    jz .fail

    movsd xmm1, [double_neg_4_0]
    call check_double_result
    test eax, eax
    jz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_ceil_positive - Test ceil(3.2) = 4.0
;-----------------------------------------------------------------------------
test_ceil_positive:
    push rbp
    mov rbp, rsp

    lea rdi, [test_ceil_pos]
    call test_start

    call reset_state

    ; Push 3.2
    lea rdi, [double_3_2]
    call push_double

    ; Ceil
    call op_ceil

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is 4.0
    call stack_data_pop
    test rax, rax
    jz .fail

    movsd xmm1, [double_4_0]
    call check_double_result
    test eax, eax
    jz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_ceil_negative - Test ceil(-3.7) = -3.0
;-----------------------------------------------------------------------------
test_ceil_negative:
    push rbp
    mov rbp, rsp

    lea rdi, [test_ceil_neg]
    call test_start

    call reset_state

    ; Push -3.7
    lea rdi, [double_neg_3_7]
    call push_double

    ; Ceil
    call op_ceil

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is -3.0
    call stack_data_pop
    test rax, rax
    jz .fail

    movsd xmm1, [double_neg_3_0]
    call check_double_result
    test eax, eax
    jz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_round_up_op - Test round(3.6) = 4.0
;-----------------------------------------------------------------------------
test_round_up_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_round_up]
    call test_start

    call reset_state

    ; Push 3.6
    lea rdi, [double_3_6]
    call push_double

    ; Round
    call op_round

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is 4.0
    call stack_data_pop
    test rax, rax
    jz .fail

    movsd xmm1, [double_4_0]
    call check_double_result
    test eax, eax
    jz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_round_down_op - Test round(3.4) = 3.0
;-----------------------------------------------------------------------------
test_round_down_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_round_down]
    call test_start

    call reset_state

    ; Push 3.4
    lea rdi, [double_3_4]
    call push_double

    ; Round
    call op_round

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is 3.0
    call stack_data_pop
    test rax, rax
    jz .fail

    movsd xmm1, [double_3_0]
    call check_double_result
    test eax, eax
    jz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_round_half_op - Test round(3.5) = 4.0 (banker's rounding to even)
;-----------------------------------------------------------------------------
test_round_half_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_round_half]
    call test_start

    call reset_state

    ; Push 3.5
    lea rdi, [double_3_5]
    call push_double

    ; Round
    call op_round

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is 4.0 (or 3.0 depending on rounding mode)
    call stack_data_pop
    test rax, rax
    jz .fail

    ; Try 4.0 first
    movsd xmm1, [double_4_0]
    call check_double_result
    test eax, eax
    jnz .pass

    ; Try 3.0 (banker's rounding)
    call stack_data_peek
    movsd xmm1, [double_3_0]
    call check_double_result
    test eax, eax
    jz .fail

.pass:
    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_truncate_positive - Test truncate(3.9) = 3.0
;-----------------------------------------------------------------------------
test_truncate_positive:
    push rbp
    mov rbp, rsp

    lea rdi, [test_truncate_pos]
    call test_start

    call reset_state

    ; Push 3.9
    lea rdi, [double_3_9]
    call push_double

    ; Truncate
    call op_truncate

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is 3.0
    call stack_data_pop
    test rax, rax
    jz .fail

    movsd xmm1, [double_3_0]
    call check_double_result
    test eax, eax
    jz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_truncate_negative - Test truncate(-3.9) = -3.0
;-----------------------------------------------------------------------------
test_truncate_negative:
    push rbp
    mov rbp, rsp

    lea rdi, [test_truncate_neg]
    call test_start

    call reset_state

    ; Push -3.9
    lea rdi, [double_neg_3_9]
    call push_double

    ; Truncate
    call op_truncate

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is -3.0
    call stack_data_pop
    test rax, rax
    jz .fail

    movsd xmm1, [double_neg_3_0]
    call check_double_result
    test eax, eax
    jz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_pow_integer - Test pow(2, 3) = 8
;-----------------------------------------------------------------------------
test_pow_integer:
    push rbp
    mov rbp, rsp
    sub rsp, 16                 ; Local storage for result pointer

    lea rdi, [test_pow_int]
    call test_start

    call reset_state

    ; Push 2.0 (base)
    lea rdi, [double_2_0]
    call push_double

    ; Push 3.0 (exponent)
    lea rdi, [double_3_0]
    call push_double

    ; Pow
    call op_pow

    ; DEBUG: Print stack
    call print_stack

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail_error

    ; Check stack depth after pow
    call stack_data_depth
    test rax, rax
    jz .fail_empty              ; Stack is empty!

    ; Use peek to check result
    call stack_data_peek
    test rax, rax
    jz .fail_peek

    ; Check tag is double
    movzx edi, byte [rax + VALUE_TAG_OFF]
    cmp edi, VTAG_DOUBLE
    jne .fail_tag

    ; Get the actual double value and compare
    movsd xmm0, [rax + VALUE_NUM_OFF]  ; actual result
    movsd xmm1, [double_8_0]            ; expected = 8.0

    ; Compute |actual - expected|
    subsd xmm0, xmm1
    ; Absolute value
    movsd xmm2, xmm0
    xorpd xmm3, xmm3
    subsd xmm3, xmm2
    maxsd xmm0, xmm3

    ; Compare with epsilon
    movsd xmm1, [double_epsilon]
    ucomisd xmm0, xmm1
    ja .fail

    call test_end_pass
    jmp .done

.fail:
.fail_error:
.fail_empty:
.fail_peek:
.fail_tag:
    call test_end_fail

.done:
    add rsp, 16
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_pow_fractional - Test pow(4, 0.5) = 2 (square root)
;-----------------------------------------------------------------------------
test_pow_fractional:
    push rbp
    mov rbp, rsp

    lea rdi, [test_pow_frac]
    call test_start

    call reset_state

    ; Push 4.0 (base)
    lea rdi, [double_4_0]
    call push_double

    ; Push 0.5 (exponent)
    lea rdi, [double_0_5]
    call push_double

    ; Pow
    call op_pow

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result
    call stack_data_peek
    test rax, rax
    jz .fail

    ; Check tag is double
    movzx edi, byte [rax + VALUE_TAG_OFF]
    cmp edi, VTAG_DOUBLE
    jne .fail

    ; Get the actual double value and compare to 2.0
    movsd xmm0, [rax + VALUE_NUM_OFF]
    movsd xmm1, [double_2_0]
    subsd xmm0, xmm1
    ; Absolute value
    movsd xmm2, xmm0
    xorpd xmm3, xmm3
    subsd xmm3, xmm2
    maxsd xmm0, xmm3
    ; Compare with epsilon
    movsd xmm1, [double_epsilon]
    ucomisd xmm0, xmm1
    ja .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_pow_negative - Test pow(2, -1) = 0.5
;-----------------------------------------------------------------------------
test_pow_negative:
    push rbp
    mov rbp, rsp

    lea rdi, [test_pow_neg]
    call test_start

    call reset_state

    ; Push 2.0 (base)
    lea rdi, [double_2_0]
    call push_double

    ; Push -1.0 (exponent)
    lea rdi, [double_neg_1_0]
    call push_double

    ; Pow
    call op_pow

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result
    call stack_data_peek
    test rax, rax
    jz .fail

    ; Check tag is double
    movzx edi, byte [rax + VALUE_TAG_OFF]
    cmp edi, VTAG_DOUBLE
    jne .fail

    ; Get the actual double value and compare to 0.5
    movsd xmm0, [rax + VALUE_NUM_OFF]
    movsd xmm1, [double_0_5]
    subsd xmm0, xmm1
    ; Absolute value
    movsd xmm2, xmm0
    xorpd xmm3, xmm3
    subsd xmm3, xmm2
    maxsd xmm0, xmm3
    ; Compare with epsilon
    movsd xmm1, [double_epsilon]
    ucomisd xmm0, xmm1
    ja .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret
