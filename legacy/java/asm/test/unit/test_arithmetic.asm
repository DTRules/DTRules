; test_arithmetic.asm - Unit tests for Arithmetic operations
; DTRules Assembly Implementation

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

; Operators to test (from bytecode.asm)
extern op_add
extern op_sub
extern op_mul
extern op_div
extern op_mod
extern op_neg
extern op_abs
extern op_min
extern op_max

; Print functions
extern print_string
extern print_integer
extern print_newline

; Test harness functions
extern test_start
extern test_end_pass
extern test_end_fail
extern test_pass
extern test_fail
extern assert_eq
extern assert_true
extern assert_false
extern assert_double_eq
extern print_test_summary
extern reset_state
extern test_count
extern fail_count
extern pass_count

section .data
    test_header:        db "=== Arithmetic Operation Tests ===", 10, 0
    test_add:           db "add", 0
    test_add_neg:       db "add negative", 0
    test_sub:           db "sub", 0
    test_mul:           db "mul", 0
    test_div:           db "div", 0
    test_mod:           db "mod", 0
    test_neg:           db "neg", 0
    test_abs_pos:       db "abs positive", 0
    test_abs_neg:       db "abs negative", 0
    test_min:           db "min", 0
    test_max:           db "max", 0
    ; Mixed-type tests
    test_add_mixed:     db "add int+double", 0
    test_sub_mixed:     db "sub int-double", 0
    test_mul_mixed:     db "mul int*double", 0
    test_div_mixed:     db "div int/double", 0
    test_min_mixed:     db "min int,double", 0
    test_max_mixed:     db "max int,double", 0
    test_double_add:    db "add double+double", 0
    test_double_cmp:    db "compare doubles", 0

section .text
    global test_main

;-----------------------------------------------------------------------------
; test_main - Run all arithmetic tests
;-----------------------------------------------------------------------------
test_main:
    push rbp
    mov rbp, rsp

    ; Print header
    lea rdi, [test_header]
    call print_string

    ; Run tests
    call test_add_op
    call test_add_negative_op
    call test_sub_op
    call test_mul_op
    call test_div_op
    call test_mod_op
    call test_neg_op
    call test_abs_positive_op
    call test_abs_negative_op
    call test_min_op
    call test_max_op

    ; Mixed-type tests
    call test_add_mixed_op
    call test_sub_mixed_op
    call test_mul_mixed_op
    call test_div_mixed_op
    call test_min_mixed_op
    call test_max_mixed_op
    call test_double_add_op
    call test_double_cmp_op

    ; Print summary
    call print_test_summary

    ; Return fail count as exit status
    mov rax, [fail_count]

    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_add_op - Test addition: 3 + 4 = 7
;-----------------------------------------------------------------------------
test_add_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_add]
    call test_start

    call reset_state

    ; Push 3, then 4
    mov rdi, 3
    call stack_data_push_integer
    mov rdi, 4
    call stack_data_push_integer

    ; Add
    call op_add

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, 7
    call assert_eq
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
; test_add_negative_op - Test addition with negatives: -5 + 3 = -2
;-----------------------------------------------------------------------------
test_add_negative_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_add_neg]
    call test_start

    call reset_state

    ; Push -5, then 3
    mov rdi, -5
    call stack_data_push_integer
    mov rdi, 3
    call stack_data_push_integer

    ; Add
    call op_add

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, -2
    call assert_eq
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
; test_sub_op - Test subtraction: 10 - 3 = 7
;-----------------------------------------------------------------------------
test_sub_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_sub]
    call test_start

    call reset_state

    ; Push 10, then 3
    mov rdi, 10
    call stack_data_push_integer
    mov rdi, 3
    call stack_data_push_integer

    ; Subtract
    call op_sub

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, 7
    call assert_eq
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
; test_mul_op - Test multiplication: 6 * 7 = 42
;-----------------------------------------------------------------------------
test_mul_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_mul]
    call test_start

    call reset_state

    ; Push 6, then 7
    mov rdi, 6
    call stack_data_push_integer
    mov rdi, 7
    call stack_data_push_integer

    ; Multiply
    call op_mul

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, 42
    call assert_eq
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
; test_div_op - Test division: 20 / 4 = 5
;-----------------------------------------------------------------------------
test_div_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_div]
    call test_start

    call reset_state

    ; Push 20, then 4
    mov rdi, 20
    call stack_data_push_integer
    mov rdi, 4
    call stack_data_push_integer

    ; Divide
    call op_div

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, 5
    call assert_eq
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
; test_mod_op - Test modulo: 17 % 5 = 2
;-----------------------------------------------------------------------------
test_mod_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_mod]
    call test_start

    call reset_state

    ; Push 17, then 5
    mov rdi, 17
    call stack_data_push_integer
    mov rdi, 5
    call stack_data_push_integer

    ; Mod
    call op_mod

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, 2
    call assert_eq
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
; test_neg_op - Test negation: neg(42) = -42
;-----------------------------------------------------------------------------
test_neg_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_neg]
    call test_start

    call reset_state

    ; Push 42
    mov rdi, 42
    call stack_data_push_integer

    ; Negate
    call op_neg

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result (peek since neg modifies in place)
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, -42
    call assert_eq
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
; test_abs_positive_op - Test abs of positive: abs(42) = 42
;-----------------------------------------------------------------------------
test_abs_positive_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_abs_pos]
    call test_start

    call reset_state

    ; Push 42
    mov rdi, 42
    call stack_data_push_integer

    ; Abs
    call op_abs

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, 42
    call assert_eq
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
; test_abs_negative_op - Test abs of negative: abs(-42) = 42
;-----------------------------------------------------------------------------
test_abs_negative_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_abs_neg]
    call test_start

    call reset_state

    ; Push -42
    mov rdi, -42
    call stack_data_push_integer

    ; Abs
    call op_abs

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, 42
    call assert_eq
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
; test_min_op - Test min: min(5, 3) = 3
;-----------------------------------------------------------------------------
test_min_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_min]
    call test_start

    call reset_state

    ; Push 5, then 3
    mov rdi, 5
    call stack_data_push_integer
    mov rdi, 3
    call stack_data_push_integer

    ; Min
    call op_min

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, 3
    call assert_eq
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
; test_max_op - Test max: max(5, 3) = 5
;-----------------------------------------------------------------------------
test_max_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_max]
    call test_start

    call reset_state

    ; Push 5, then 3
    mov rdi, 5
    call stack_data_push_integer
    mov rdi, 3
    call stack_data_push_integer

    ; Max
    call op_max

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, 5
    call assert_eq
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
; test_add_mixed_op - Test addition of integer + double: 5 + 3.14 = 8.14
;-----------------------------------------------------------------------------
test_add_mixed_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_add_mixed]
    call test_start

    call reset_state

    ; Push integer 5
    mov rdi, 5
    call stack_data_push_integer

    ; Push double 3.14
    mov rax, 0x40091EB851EB851F   ; 3.14 in IEEE 754
    movq xmm0, rax
    call stack_data_push_double

    ; Add
    call op_add

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is 8.14
    call stack_data_pop
    test rax, rax
    jz .fail

    ; Verify it's a double
    cmp byte [rax + VALUE_TAG_OFF], VTAG_DOUBLE
    jne .fail

    ; Compare result with expected 8.14
    movsd xmm0, [rax + VALUE_NUM_OFF]
    mov rax, 0x402047AE147AE148   ; 8.14 in IEEE 754
    movq xmm1, rax
    call assert_double_eq
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
; test_sub_mixed_op - Test subtraction of integer - double: 10 - 2.5 = 7.5
;-----------------------------------------------------------------------------
test_sub_mixed_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_sub_mixed]
    call test_start

    call reset_state

    ; Push integer 10
    mov rdi, 10
    call stack_data_push_integer

    ; Push double 2.5
    mov rax, 0x4004000000000000   ; 2.5 in IEEE 754
    movq xmm0, rax
    call stack_data_push_double

    ; Subtract
    call op_sub

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result
    call stack_data_pop
    test rax, rax
    jz .fail

    ; Verify it's a double
    cmp byte [rax + VALUE_TAG_OFF], VTAG_DOUBLE
    jne .fail

    ; Compare result with expected 7.5
    movsd xmm0, [rax + VALUE_NUM_OFF]
    mov rax, 0x401E000000000000   ; 7.5 in IEEE 754
    movq xmm1, rax
    call assert_double_eq
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
; test_mul_mixed_op - Test multiplication of integer * double: 4 * 2.5 = 10.0
;-----------------------------------------------------------------------------
test_mul_mixed_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_mul_mixed]
    call test_start

    call reset_state

    ; Push integer 4
    mov rdi, 4
    call stack_data_push_integer

    ; Push double 2.5
    mov rax, 0x4004000000000000   ; 2.5 in IEEE 754
    movq xmm0, rax
    call stack_data_push_double

    ; Multiply
    call op_mul

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result
    call stack_data_pop
    test rax, rax
    jz .fail

    ; Verify it's a double
    cmp byte [rax + VALUE_TAG_OFF], VTAG_DOUBLE
    jne .fail

    ; Compare result with expected 10.0
    movsd xmm0, [rax + VALUE_NUM_OFF]
    mov rax, 0x4024000000000000   ; 10.0 in IEEE 754
    movq xmm1, rax
    call assert_double_eq
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
; test_div_mixed_op - Test division of integer / double: 15 / 2.0 = 7.5
;-----------------------------------------------------------------------------
test_div_mixed_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_div_mixed]
    call test_start

    call reset_state

    ; Push integer 15
    mov rdi, 15
    call stack_data_push_integer

    ; Push double 2.0
    mov rax, 0x4000000000000000   ; 2.0 in IEEE 754
    movq xmm0, rax
    call stack_data_push_double

    ; Divide
    call op_div

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result
    call stack_data_pop
    test rax, rax
    jz .fail

    ; Verify it's a double
    cmp byte [rax + VALUE_TAG_OFF], VTAG_DOUBLE
    jne .fail

    ; Compare result with expected 7.5
    movsd xmm0, [rax + VALUE_NUM_OFF]
    mov rax, 0x401E000000000000   ; 7.5 in IEEE 754
    movq xmm1, rax
    call assert_double_eq
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
; test_min_mixed_op - Test min of integer and double: min(5, 3.14) = 3.14
;-----------------------------------------------------------------------------
test_min_mixed_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_min_mixed]
    call test_start

    call reset_state

    ; Push integer 5
    mov rdi, 5
    call stack_data_push_integer

    ; Push double 3.14
    mov rax, 0x40091EB851EB851F   ; 3.14 in IEEE 754
    movq xmm0, rax
    call stack_data_push_double

    ; Min
    call op_min

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result
    call stack_data_pop
    test rax, rax
    jz .fail

    ; Verify it's a double
    cmp byte [rax + VALUE_TAG_OFF], VTAG_DOUBLE
    jne .fail

    ; Compare result with expected 3.14
    movsd xmm0, [rax + VALUE_NUM_OFF]
    mov rax, 0x40091EB851EB851F   ; 3.14 in IEEE 754
    movq xmm1, rax
    call assert_double_eq
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
; test_max_mixed_op - Test max of integer and double: max(5, 3.14) = 5.0
;-----------------------------------------------------------------------------
test_max_mixed_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_max_mixed]
    call test_start

    call reset_state

    ; Push integer 5
    mov rdi, 5
    call stack_data_push_integer

    ; Push double 3.14
    mov rax, 0x40091EB851EB851F   ; 3.14 in IEEE 754
    movq xmm0, rax
    call stack_data_push_double

    ; Max
    call op_max

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result
    call stack_data_pop
    test rax, rax
    jz .fail

    ; Verify it's a double
    cmp byte [rax + VALUE_TAG_OFF], VTAG_DOUBLE
    jne .fail

    ; Compare result with expected 5.0
    movsd xmm0, [rax + VALUE_NUM_OFF]
    mov rax, 0x4014000000000000   ; 5.0 in IEEE 754
    movq xmm1, rax
    call assert_double_eq
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
; test_double_add_op - Test addition of double + double: 3.14 + 2.86 = 6.0
;-----------------------------------------------------------------------------
test_double_add_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_double_add]
    call test_start

    call reset_state

    ; Push double 3.14
    mov rax, 0x40091EB851EB851F   ; 3.14 in IEEE 754
    movq xmm0, rax
    call stack_data_push_double

    ; Push double 2.86
    mov rax, 0x4006E147AE147AE1   ; 2.86 in IEEE 754
    movq xmm0, rax
    call stack_data_push_double

    ; Add
    call op_add

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result
    call stack_data_pop
    test rax, rax
    jz .fail

    ; Verify it's a double
    cmp byte [rax + VALUE_TAG_OFF], VTAG_DOUBLE
    jne .fail

    ; Compare result with expected 6.0
    movsd xmm0, [rax + VALUE_NUM_OFF]
    mov rax, 0x4018000000000000   ; 6.0 in IEEE 754
    movq xmm1, rax
    call assert_double_eq
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
; test_double_cmp_op - Test double comparison: 3.14 < 5.0 is true
;-----------------------------------------------------------------------------
test_double_cmp_op:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_double_cmp]
    call test_start

    call reset_state

    ; Push double 3.14
    mov rax, 0x40091EB851EB851F   ; 3.14 in IEEE 754
    movq xmm0, rax
    call stack_data_push_double

    ; Push double 5.0
    mov rax, 0x4014000000000000   ; 5.0 in IEEE 754
    movq xmm0, rax
    call stack_data_push_double

    ; Compare 3.14 < 5.0
    extern op_lt
    call op_lt

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is true (1)
    call stack_data_pop
    test rax, rax
    jz .fail

    ; Verify it's a boolean
    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .fail

    ; Check value is 1 (true)
    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, 1
    call assert_eq
    test eax, eax
    jz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbx
    pop rbp
    ret
