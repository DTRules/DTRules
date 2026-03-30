; test_stack.asm - Unit tests for Data Stack operations
; DTRules Assembly Implementation

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state

; Stack functions to test
extern stack_data_init
extern stack_data_push
extern stack_data_push_integer
extern stack_data_push_boolean
extern stack_data_push_null
extern stack_data_pop
extern stack_data_peek
extern stack_data_peek_n
extern stack_data_depth
extern stack_data_dup
extern stack_data_swap
extern stack_data_rot
extern stack_data_over
extern stack_data_pick
extern stack_data_roll
extern stack_data_clear
extern stack_data_drop

; Value functions
extern value_new_integer
extern value_get_integer

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
    test_header:        db "=== Stack Operation Tests ===", 10, 0
    test_push_pop:      db "push/pop", 0
    test_depth:         db "depth", 0
    test_dup:           db "dup", 0
    test_swap:          db "swap", 0
    test_rot:           db "rot", 0
    test_over:          db "over", 0
    test_pick:          db "pick", 0
    test_roll:          db "roll", 0
    test_clear:         db "clear", 0

section .text
    global test_main

;-----------------------------------------------------------------------------
; test_main - Run all stack tests
;-----------------------------------------------------------------------------
test_main:
    push rbp
    mov rbp, rsp

    ; Print header
    lea rdi, [test_header]
    call print_string

    ; Run tests
    call test_push_pop_ops
    call test_depth_ops
    call test_dup_op
    call test_swap_op
    call test_rot_op
    call test_over_op
    call test_pick_op
    call test_roll_op
    call test_clear_op

    ; Print summary
    call print_test_summary

    ; Return fail count as exit status
    mov rax, [fail_count]

    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_push_pop_ops - Test push and pop
;-----------------------------------------------------------------------------
test_push_pop_ops:
    push rbp
    mov rbp, rsp
    push rbx
    ; NOTE: Do NOT save/restore r12 - it's the VM stack pointer!

    lea rdi, [test_push_pop]
    call test_start

    call reset_state

    ; Push 42
    mov rdi, 42
    call stack_data_push_integer
    test eax, eax
    jnz .fail

    ; Check depth is 1
    call stack_data_depth
    mov rdi, rax
    mov rsi, 1
    call assert_eq
    test eax, eax
    jz .fail

    ; Pop and check value
    call stack_data_pop
    test rax, rax
    jz .fail
    mov rbx, rax

    mov rax, [rbx + VALUE_NUM_OFF]
    mov rdi, rax
    mov rsi, 42
    call assert_eq
    test eax, eax
    jz .fail

    ; Check depth is 0
    call stack_data_depth
    mov rdi, rax
    xor esi, esi
    call assert_eq
    test eax, eax
    jz .fail

    ; Push multiple values and verify order
    mov rdi, 1
    call stack_data_push_integer
    mov rdi, 2
    call stack_data_push_integer
    mov rdi, 3
    call stack_data_push_integer

    ; Pop should return 3 first
    call stack_data_pop
    mov rax, [rax + VALUE_NUM_OFF]
    mov rdi, rax
    mov rsi, 3
    call assert_eq
    test eax, eax
    jz .fail

    ; Pop should return 2
    call stack_data_pop
    mov rax, [rax + VALUE_NUM_OFF]
    mov rdi, rax
    mov rsi, 2
    call assert_eq
    test eax, eax
    jz .fail

    ; Pop should return 1
    call stack_data_pop
    mov rax, [rax + VALUE_NUM_OFF]
    mov rdi, rax
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

;-----------------------------------------------------------------------------
; test_depth_ops - Test stack depth
;-----------------------------------------------------------------------------
test_depth_ops:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_depth]
    call test_start

    call reset_state

    ; Empty stack has depth 0
    call stack_data_depth
    mov rdi, rax
    xor esi, esi
    call assert_eq
    test eax, eax
    jz .fail

    ; Push 5 values
    mov rbx, 5
.push_loop:
    test rbx, rbx
    jz .check_depth
    mov rdi, rbx
    call stack_data_push_integer
    dec rbx
    jmp .push_loop

.check_depth:
    call stack_data_depth
    mov rdi, rax
    mov rsi, 5
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

;-----------------------------------------------------------------------------
; test_dup_op - Test dup operation
;-----------------------------------------------------------------------------
test_dup_op:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_dup]
    call test_start

    call reset_state

    ; Push 5, dup -> [5, 5]
    mov rdi, 5
    call stack_data_push_integer

    call stack_data_dup
    test eax, eax
    jnz .fail

    ; Check depth is 2
    call stack_data_depth
    mov rdi, rax
    mov rsi, 2
    call assert_eq
    test eax, eax
    jz .fail

    ; Both values should be 5
    call stack_data_pop
    mov rax, [rax + VALUE_NUM_OFF]
    mov rdi, rax
    mov rsi, 5
    call assert_eq
    test eax, eax
    jz .fail

    call stack_data_pop
    mov rax, [rax + VALUE_NUM_OFF]
    mov rdi, rax
    mov rsi, 5
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

;-----------------------------------------------------------------------------
; test_swap_op - Test swap operation
;-----------------------------------------------------------------------------
test_swap_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_swap]
    call test_start

    call reset_state

    ; Push 1, 2 -> [1, 2], swap -> [2, 1]
    mov rdi, 1
    call stack_data_push_integer
    mov rdi, 2
    call stack_data_push_integer

    call stack_data_swap
    test eax, eax
    jnz .fail

    ; Top should be 1
    call stack_data_pop
    mov rax, [rax + VALUE_NUM_OFF]
    mov rdi, rax
    mov rsi, 1
    call assert_eq
    test eax, eax
    jz .fail

    ; Next should be 2
    call stack_data_pop
    mov rax, [rax + VALUE_NUM_OFF]
    mov rdi, rax
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
; test_rot_op - Test rot operation (a b c -- b c a)
;-----------------------------------------------------------------------------
test_rot_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_rot]
    call test_start

    call reset_state

    ; Push 1, 2, 3 -> [1, 2, 3], rot -> [2, 3, 1]
    mov rdi, 1
    call stack_data_push_integer
    mov rdi, 2
    call stack_data_push_integer
    mov rdi, 3
    call stack_data_push_integer

    call stack_data_rot
    test eax, eax
    jnz .fail

    ; Top should be 1 (was deepest)
    call stack_data_pop
    mov rax, [rax + VALUE_NUM_OFF]
    mov rdi, rax
    mov rsi, 1
    call assert_eq
    test eax, eax
    jz .fail

    ; Next should be 3
    call stack_data_pop
    mov rax, [rax + VALUE_NUM_OFF]
    mov rdi, rax
    mov rsi, 3
    call assert_eq
    test eax, eax
    jz .fail

    ; Last should be 2
    call stack_data_pop
    mov rax, [rax + VALUE_NUM_OFF]
    mov rdi, rax
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
; test_over_op - Test over operation (a b -- a b a)
;-----------------------------------------------------------------------------
test_over_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_over]
    call test_start

    call reset_state

    ; Push 1, 2 -> [1, 2], over -> [1, 2, 1]
    mov rdi, 1
    call stack_data_push_integer
    mov rdi, 2
    call stack_data_push_integer

    call stack_data_over
    test eax, eax
    jnz .fail

    ; Check depth is 3
    call stack_data_depth
    mov rdi, rax
    mov rsi, 3
    call assert_eq
    test eax, eax
    jz .fail

    ; Top should be 1 (copy of second)
    call stack_data_pop
    mov rax, [rax + VALUE_NUM_OFF]
    mov rdi, rax
    mov rsi, 1
    call assert_eq
    test eax, eax
    jz .fail

    ; Next should be 2
    call stack_data_pop
    mov rax, [rax + VALUE_NUM_OFF]
    mov rdi, rax
    mov rsi, 2
    call assert_eq
    test eax, eax
    jz .fail

    ; Last should be 1
    call stack_data_pop
    mov rax, [rax + VALUE_NUM_OFF]
    mov rdi, rax
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
; test_pick_op - Test pick operation (copies nth element to top)
;-----------------------------------------------------------------------------
test_pick_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_pick]
    call test_start

    call reset_state

    ; Push 1, 2, 3 -> [1, 2, 3]
    mov rdi, 1
    call stack_data_push_integer
    mov rdi, 2
    call stack_data_push_integer
    mov rdi, 3
    call stack_data_push_integer

    ; pick 2 -> copies element at index 2 (the 1) to top
    ; Stack becomes [1, 2, 3, 1]
    mov rdi, 2
    call stack_data_pick
    test eax, eax
    jnz .fail

    ; Check depth is 4
    call stack_data_depth
    mov rdi, rax
    mov rsi, 4
    call assert_eq
    test eax, eax
    jz .fail

    ; Top should be 1
    call stack_data_pop
    mov rax, [rax + VALUE_NUM_OFF]
    mov rdi, rax
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
; test_roll_op - Test roll operation (moves nth element to top)
;-----------------------------------------------------------------------------
test_roll_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_roll]
    call test_start

    call reset_state

    ; Push 1, 2, 3, 4 -> [1, 2, 3, 4]
    mov rdi, 1
    call stack_data_push_integer
    mov rdi, 2
    call stack_data_push_integer
    mov rdi, 3
    call stack_data_push_integer
    mov rdi, 4
    call stack_data_push_integer

    ; roll 3 -> moves element at index 3 (the 1) to top
    ; Stack becomes [2, 3, 4, 1]
    mov rdi, 3
    call stack_data_roll
    test eax, eax
    jnz .fail

    ; Check depth is still 4
    call stack_data_depth
    mov rdi, rax
    mov rsi, 4
    call assert_eq
    test eax, eax
    jz .fail

    ; Top should be 1
    call stack_data_pop
    mov rax, [rax + VALUE_NUM_OFF]
    mov rdi, rax
    mov rsi, 1
    call assert_eq
    test eax, eax
    jz .fail

    ; Next should be 4
    call stack_data_pop
    mov rax, [rax + VALUE_NUM_OFF]
    mov rdi, rax
    mov rsi, 4
    call assert_eq
    test eax, eax
    jz .fail

    ; Next should be 3
    call stack_data_pop
    mov rax, [rax + VALUE_NUM_OFF]
    mov rdi, rax
    mov rsi, 3
    call assert_eq
    test eax, eax
    jz .fail

    ; Last should be 2
    call stack_data_pop
    mov rax, [rax + VALUE_NUM_OFF]
    mov rdi, rax
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
; test_clear_op - Test clear operation
;-----------------------------------------------------------------------------
test_clear_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_clear]
    call test_start

    call reset_state

    ; Push several values
    mov rdi, 1
    call stack_data_push_integer
    mov rdi, 2
    call stack_data_push_integer
    mov rdi, 3
    call stack_data_push_integer

    ; Verify non-empty
    call stack_data_depth
    mov rdi, rax
    mov rsi, 3
    call assert_eq
    test eax, eax
    jz .fail

    ; Clear
    call stack_data_clear

    ; Verify empty
    call stack_data_depth
    mov rdi, rax
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
