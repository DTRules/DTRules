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
