; test_comparison.asm - Unit tests for Comparison operations
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
extern stack_data_push_boolean
extern stack_data_pop
extern stack_data_peek

; Comparison operators to test (from bytecode.asm)
extern op_eq
extern op_ne
extern op_lt
extern op_le
extern op_gt
extern op_ge

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
    test_header:        db "=== Comparison Operation Tests ===", 10, 0
    test_eq_true:       db "eq (true case)", 0
    test_eq_false:      db "eq (false case)", 0
    test_ne_true:       db "ne (true case)", 0
    test_ne_false:      db "ne (false case)", 0
    test_lt_true:       db "lt (true case)", 0
    test_lt_false:      db "lt (false case)", 0
    test_le_equal:      db "le (equal case)", 0
    test_le_less:       db "le (less case)", 0
    test_gt_true:       db "gt (true case)", 0
    test_gt_false:      db "gt (false case)", 0
    test_ge_equal:      db "ge (equal case)", 0
    test_ge_greater:    db "ge (greater case)", 0

section .text
    global test_main

;-----------------------------------------------------------------------------
; test_main - Run all comparison tests
;-----------------------------------------------------------------------------
test_main:
    push rbp
    mov rbp, rsp

    ; Print header
    lea rdi, [test_header]
    call print_string

    ; Run tests
    call test_eq_true_op
    call test_eq_false_op
    call test_ne_true_op
    call test_ne_false_op
    call test_lt_true_op
    call test_lt_false_op
    call test_le_equal_op
    call test_le_less_op
    call test_gt_true_op
    call test_gt_false_op
    call test_ge_equal_op
    call test_ge_greater_op

    ; Print summary
    call print_test_summary

    ; Return fail count as exit status
    mov rax, [fail_count]

    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_eq_true_op - Test equality true: 5 == 5 -> true
;-----------------------------------------------------------------------------
test_eq_true_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_eq_true]
    call test_start

    call reset_state

    ; Push 5, 5
    mov rdi, 5
    call stack_data_push_integer
    mov rdi, 5
    call stack_data_push_integer

    ; Equal
    call op_eq

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
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_eq_false_op - Test equality false: 5 == 6 -> false
;-----------------------------------------------------------------------------
test_eq_false_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_eq_false]
    call test_start

    call reset_state

    ; Push 5, 6
    mov rdi, 5
    call stack_data_push_integer
    mov rdi, 6
    call stack_data_push_integer

    ; Equal
    call op_eq

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is false (0)
    call stack_data_pop
    test rax, rax
    jz .fail

    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    xor esi, esi
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
; test_ne_true_op - Test not-equal true: 5 != 6 -> true
;-----------------------------------------------------------------------------
test_ne_true_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_ne_true]
    call test_start

    call reset_state

    ; Push 5, 6
    mov rdi, 5
    call stack_data_push_integer
    mov rdi, 6
    call stack_data_push_integer

    ; Not equal
    call op_ne

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is true
    call stack_data_pop
    test rax, rax
    jz .fail

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
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_ne_false_op - Test not-equal false: 5 != 5 -> false
;-----------------------------------------------------------------------------
test_ne_false_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_ne_false]
    call test_start

    call reset_state

    ; Push 5, 5
    mov rdi, 5
    call stack_data_push_integer
    mov rdi, 5
    call stack_data_push_integer

    ; Not equal
    call op_ne

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is false
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    xor esi, esi
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
; test_lt_true_op - Test less-than true: 3 < 5 -> true
;-----------------------------------------------------------------------------
test_lt_true_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_lt_true]
    call test_start

    call reset_state

    ; Push 3, 5
    mov rdi, 3
    call stack_data_push_integer
    mov rdi, 5
    call stack_data_push_integer

    ; Less than
    call op_lt

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is true
    call stack_data_pop
    test rax, rax
    jz .fail

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
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_lt_false_op - Test less-than false: 5 < 3 -> false
;-----------------------------------------------------------------------------
test_lt_false_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_lt_false]
    call test_start

    call reset_state

    ; Push 5, 3
    mov rdi, 5
    call stack_data_push_integer
    mov rdi, 3
    call stack_data_push_integer

    ; Less than
    call op_lt

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is false
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    xor esi, esi
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
; test_le_equal_op - Test less-or-equal (equal): 5 <= 5 -> true
;-----------------------------------------------------------------------------
test_le_equal_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_le_equal]
    call test_start

    call reset_state

    ; Push 5, 5
    mov rdi, 5
    call stack_data_push_integer
    mov rdi, 5
    call stack_data_push_integer

    ; Less or equal
    call op_le

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is true
    call stack_data_pop
    test rax, rax
    jz .fail

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
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_le_less_op - Test less-or-equal (less): 3 <= 5 -> true
;-----------------------------------------------------------------------------
test_le_less_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_le_less]
    call test_start

    call reset_state

    ; Push 3, 5
    mov rdi, 3
    call stack_data_push_integer
    mov rdi, 5
    call stack_data_push_integer

    ; Less or equal
    call op_le

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is true
    call stack_data_pop
    test rax, rax
    jz .fail

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
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_gt_true_op - Test greater-than true: 5 > 3 -> true
;-----------------------------------------------------------------------------
test_gt_true_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_gt_true]
    call test_start

    call reset_state

    ; Push 5, 3
    mov rdi, 5
    call stack_data_push_integer
    mov rdi, 3
    call stack_data_push_integer

    ; Greater than
    call op_gt

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is true
    call stack_data_pop
    test rax, rax
    jz .fail

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
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_gt_false_op - Test greater-than false: 3 > 5 -> false
;-----------------------------------------------------------------------------
test_gt_false_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_gt_false]
    call test_start

    call reset_state

    ; Push 3, 5
    mov rdi, 3
    call stack_data_push_integer
    mov rdi, 5
    call stack_data_push_integer

    ; Greater than
    call op_gt

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is false
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    xor esi, esi
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
; test_ge_equal_op - Test greater-or-equal (equal): 5 >= 5 -> true
;-----------------------------------------------------------------------------
test_ge_equal_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_ge_equal]
    call test_start

    call reset_state

    ; Push 5, 5
    mov rdi, 5
    call stack_data_push_integer
    mov rdi, 5
    call stack_data_push_integer

    ; Greater or equal
    call op_ge

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is true
    call stack_data_pop
    test rax, rax
    jz .fail

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
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_ge_greater_op - Test greater-or-equal (greater): 5 >= 3 -> true
;-----------------------------------------------------------------------------
test_ge_greater_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_ge_greater]
    call test_start

    call reset_state

    ; Push 5, 3
    mov rdi, 5
    call stack_data_push_integer
    mov rdi, 3
    call stack_data_push_integer

    ; Greater or equal
    call op_ge

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is true
    call stack_data_pop
    test rax, rax
    jz .fail

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
    pop rbp
    ret
