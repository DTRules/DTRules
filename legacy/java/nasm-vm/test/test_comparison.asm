; test_comparison.asm - Tests for DTRules NASM VM comparison operations
; Tests eq, ne, lt, le, gt, ge operators

bits 64
default rel

%include "include/constants.inc"
%include "include/state.inc"

;-----------------------------------------------------------------------------
; External symbols
;-----------------------------------------------------------------------------
extern test_start
extern test_end_pass
extern test_end_fail
extern reset_state
extern print_test_summary
extern state
extern data_stack

; VM functions
extern vm_execute
extern vm_init
extern stack_data_push_integer
extern stack_data_push_boolean
extern stack_data_push_double
extern stack_data_pop
extern stack_data_peek

; Comparison operators (direct calls for unit testing)
extern op_eq
extern op_ne
extern op_lt
extern op_le
extern op_gt
extern op_ge
extern dispatch

;-----------------------------------------------------------------------------
; Exported symbols
;-----------------------------------------------------------------------------
global test_main

;-----------------------------------------------------------------------------
section .data
;-----------------------------------------------------------------------------

; Test names
test_eq_int_equal:      db "op_eq: integers equal (5 == 5)", 0
test_eq_int_notequal:   db "op_eq: integers not equal (5 == 3)", 0
test_ne_int_notequal:   db "op_ne: integers not equal (5 != 3)", 0
test_ne_int_equal:      db "op_ne: integers equal (5 != 5)", 0
test_lt_int_less:       db "op_lt: 3 < 5", 0
test_lt_int_equal:      db "op_lt: 5 < 5", 0
test_lt_int_greater:    db "op_lt: 7 < 5", 0
test_le_int_less:       db "op_le: 3 <= 5", 0
test_le_int_equal:      db "op_le: 5 <= 5", 0
test_le_int_greater:    db "op_le: 7 <= 5", 0
test_gt_int_greater:    db "op_gt: 7 > 5", 0
test_gt_int_equal:      db "op_gt: 5 > 5", 0
test_gt_int_less:       db "op_gt: 3 > 5", 0
test_ge_int_greater:    db "op_ge: 7 >= 5", 0
test_ge_int_equal:      db "op_ge: 5 >= 5", 0
test_ge_int_less:       db "op_ge: 3 >= 5", 0
test_eq_bool_true:      db "op_eq: booleans true == true", 0
test_eq_bool_false:     db "op_eq: booleans true == false", 0
test_lt_negative:       db "op_lt: -5 < 3 (signed)", 0
test_gt_negative:       db "op_gt: 3 > -5 (signed)", 0

; Bytecode sequences
; Format: opcode [operands] ... HALT
bc_eq_5_5:      db OP_PUSH_INT, 5, OP_PUSH_INT, 5, OP_EQ, OP_HALT
bc_eq_5_5_len   equ $ - bc_eq_5_5

bc_eq_5_3:      db OP_PUSH_INT, 5, OP_PUSH_INT, 3, OP_EQ, OP_HALT
bc_eq_5_3_len   equ $ - bc_eq_5_3

bc_ne_5_3:      db OP_PUSH_INT, 5, OP_PUSH_INT, 3, OP_NE, OP_HALT
bc_ne_5_3_len   equ $ - bc_ne_5_3

bc_ne_5_5:      db OP_PUSH_INT, 5, OP_PUSH_INT, 5, OP_NE, OP_HALT
bc_ne_5_5_len   equ $ - bc_ne_5_5

bc_lt_3_5:      db OP_PUSH_INT, 3, OP_PUSH_INT, 5, OP_LT, OP_HALT
bc_lt_3_5_len   equ $ - bc_lt_3_5

bc_lt_5_5:      db OP_PUSH_INT, 5, OP_PUSH_INT, 5, OP_LT, OP_HALT
bc_lt_5_5_len   equ $ - bc_lt_5_5

bc_lt_7_5:      db OP_PUSH_INT, 7, OP_PUSH_INT, 5, OP_LT, OP_HALT
bc_lt_7_5_len   equ $ - bc_lt_7_5

bc_le_3_5:      db OP_PUSH_INT, 3, OP_PUSH_INT, 5, OP_LE, OP_HALT
bc_le_3_5_len   equ $ - bc_le_3_5

bc_le_5_5:      db OP_PUSH_INT, 5, OP_PUSH_INT, 5, OP_LE, OP_HALT
bc_le_5_5_len   equ $ - bc_le_5_5

bc_le_7_5:      db OP_PUSH_INT, 7, OP_PUSH_INT, 5, OP_LE, OP_HALT
bc_le_7_5_len   equ $ - bc_le_7_5

bc_gt_7_5:      db OP_PUSH_INT, 7, OP_PUSH_INT, 5, OP_GT, OP_HALT
bc_gt_7_5_len   equ $ - bc_gt_7_5

bc_gt_5_5:      db OP_PUSH_INT, 5, OP_PUSH_INT, 5, OP_GT, OP_HALT
bc_gt_5_5_len   equ $ - bc_gt_5_5

bc_gt_3_5:      db OP_PUSH_INT, 3, OP_PUSH_INT, 5, OP_GT, OP_HALT
bc_gt_3_5_len   equ $ - bc_gt_3_5

bc_ge_7_5:      db OP_PUSH_INT, 7, OP_PUSH_INT, 5, OP_GE, OP_HALT
bc_ge_7_5_len   equ $ - bc_ge_7_5

bc_ge_5_5:      db OP_PUSH_INT, 5, OP_PUSH_INT, 5, OP_GE, OP_HALT
bc_ge_5_5_len   equ $ - bc_ge_5_5

bc_ge_3_5:      db OP_PUSH_INT, 3, OP_PUSH_INT, 5, OP_GE, OP_HALT
bc_ge_3_5_len   equ $ - bc_ge_3_5

; Negative number tests (using signed byte: -5 = 0xFB)
bc_lt_neg5_3:   db OP_PUSH_INT, 0xFB, OP_PUSH_INT, 3, OP_LT, OP_HALT
bc_lt_neg5_3_len equ $ - bc_lt_neg5_3

bc_gt_3_neg5:   db OP_PUSH_INT, 3, OP_PUSH_INT, 0xFB, OP_GT, OP_HALT
bc_gt_3_neg5_len equ $ - bc_gt_3_neg5

;-----------------------------------------------------------------------------
section .text
;-----------------------------------------------------------------------------

;-----------------------------------------------------------------------------
; test_main - Main test entry point
; Runs all comparison tests and prints summary
;-----------------------------------------------------------------------------
test_main:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14
    push r15

    ;-------------------------------------------------------------------------
    ; Test: op_eq integers equal (5 == 5)
    ;-------------------------------------------------------------------------
    call reset_state
    lea rdi, [test_eq_int_equal]
    call test_start

    lea rdi, [bc_eq_5_5]
    mov rsi, bc_eq_5_5_len
    call vm_execute

    ; Check result: should be boolean true (1)
    call stack_data_peek
    test rax, rax
    jz .fail_eq_int_equal

    ; Verify it's a boolean with value 1
    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .fail_eq_int_equal
    cmp qword [rax + VALUE_NUM_OFF], 1
    jne .fail_eq_int_equal

    call test_end_pass
    jmp .test_eq_int_notequal

.fail_eq_int_equal:
    call test_end_fail

    ;-------------------------------------------------------------------------
    ; Test: op_eq integers not equal (5 == 3)
    ;-------------------------------------------------------------------------
.test_eq_int_notequal:
    call reset_state
    lea rdi, [test_eq_int_notequal]
    call test_start

    lea rdi, [bc_eq_5_3]
    mov rsi, bc_eq_5_3_len
    call vm_execute

    call stack_data_peek
    test rax, rax
    jz .fail_eq_int_notequal

    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .fail_eq_int_notequal
    cmp qword [rax + VALUE_NUM_OFF], 0
    jne .fail_eq_int_notequal

    call test_end_pass
    jmp .test_ne_int_notequal

.fail_eq_int_notequal:
    call test_end_fail

    ;-------------------------------------------------------------------------
    ; Test: op_ne integers not equal (5 != 3)
    ;-------------------------------------------------------------------------
.test_ne_int_notequal:
    call reset_state
    lea rdi, [test_ne_int_notequal]
    call test_start

    lea rdi, [bc_ne_5_3]
    mov rsi, bc_ne_5_3_len
    call vm_execute

    call stack_data_peek
    test rax, rax
    jz .fail_ne_int_notequal

    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .fail_ne_int_notequal
    cmp qword [rax + VALUE_NUM_OFF], 1
    jne .fail_ne_int_notequal

    call test_end_pass
    jmp .test_ne_int_equal

.fail_ne_int_notequal:
    call test_end_fail

    ;-------------------------------------------------------------------------
    ; Test: op_ne integers equal (5 != 5)
    ;-------------------------------------------------------------------------
.test_ne_int_equal:
    call reset_state
    lea rdi, [test_ne_int_equal]
    call test_start

    lea rdi, [bc_ne_5_5]
    mov rsi, bc_ne_5_5_len
    call vm_execute

    call stack_data_peek
    test rax, rax
    jz .fail_ne_int_equal

    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .fail_ne_int_equal
    cmp qword [rax + VALUE_NUM_OFF], 0
    jne .fail_ne_int_equal

    call test_end_pass
    jmp .test_lt_int_less

.fail_ne_int_equal:
    call test_end_fail

    ;-------------------------------------------------------------------------
    ; Test: op_lt 3 < 5 (should be true)
    ;-------------------------------------------------------------------------
.test_lt_int_less:
    call reset_state
    lea rdi, [test_lt_int_less]
    call test_start

    lea rdi, [bc_lt_3_5]
    mov rsi, bc_lt_3_5_len
    call vm_execute

    call stack_data_peek
    test rax, rax
    jz .fail_lt_int_less

    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .fail_lt_int_less
    cmp qword [rax + VALUE_NUM_OFF], 1
    jne .fail_lt_int_less

    call test_end_pass
    jmp .test_lt_int_equal

.fail_lt_int_less:
    call test_end_fail

    ;-------------------------------------------------------------------------
    ; Test: op_lt 5 < 5 (should be false)
    ;-------------------------------------------------------------------------
.test_lt_int_equal:
    call reset_state
    lea rdi, [test_lt_int_equal]
    call test_start

    lea rdi, [bc_lt_5_5]
    mov rsi, bc_lt_5_5_len
    call vm_execute

    call stack_data_peek
    test rax, rax
    jz .fail_lt_int_equal

    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .fail_lt_int_equal
    cmp qword [rax + VALUE_NUM_OFF], 0
    jne .fail_lt_int_equal

    call test_end_pass
    jmp .test_lt_int_greater

.fail_lt_int_equal:
    call test_end_fail

    ;-------------------------------------------------------------------------
    ; Test: op_lt 7 < 5 (should be false)
    ;-------------------------------------------------------------------------
.test_lt_int_greater:
    call reset_state
    lea rdi, [test_lt_int_greater]
    call test_start

    lea rdi, [bc_lt_7_5]
    mov rsi, bc_lt_7_5_len
    call vm_execute

    call stack_data_peek
    test rax, rax
    jz .fail_lt_int_greater

    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .fail_lt_int_greater
    cmp qword [rax + VALUE_NUM_OFF], 0
    jne .fail_lt_int_greater

    call test_end_pass
    jmp .test_le_int_less

.fail_lt_int_greater:
    call test_end_fail

    ;-------------------------------------------------------------------------
    ; Test: op_le 3 <= 5 (should be true)
    ;-------------------------------------------------------------------------
.test_le_int_less:
    call reset_state
    lea rdi, [test_le_int_less]
    call test_start

    lea rdi, [bc_le_3_5]
    mov rsi, bc_le_3_5_len
    call vm_execute

    call stack_data_peek
    test rax, rax
    jz .fail_le_int_less

    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .fail_le_int_less
    cmp qword [rax + VALUE_NUM_OFF], 1
    jne .fail_le_int_less

    call test_end_pass
    jmp .test_le_int_equal

.fail_le_int_less:
    call test_end_fail

    ;-------------------------------------------------------------------------
    ; Test: op_le 5 <= 5 (should be true)
    ;-------------------------------------------------------------------------
.test_le_int_equal:
    call reset_state
    lea rdi, [test_le_int_equal]
    call test_start

    lea rdi, [bc_le_5_5]
    mov rsi, bc_le_5_5_len
    call vm_execute

    call stack_data_peek
    test rax, rax
    jz .fail_le_int_equal

    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .fail_le_int_equal
    cmp qword [rax + VALUE_NUM_OFF], 1
    jne .fail_le_int_equal

    call test_end_pass
    jmp .test_le_int_greater

.fail_le_int_equal:
    call test_end_fail

    ;-------------------------------------------------------------------------
    ; Test: op_le 7 <= 5 (should be false)
    ;-------------------------------------------------------------------------
.test_le_int_greater:
    call reset_state
    lea rdi, [test_le_int_greater]
    call test_start

    lea rdi, [bc_le_7_5]
    mov rsi, bc_le_7_5_len
    call vm_execute

    call stack_data_peek
    test rax, rax
    jz .fail_le_int_greater

    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .fail_le_int_greater
    cmp qword [rax + VALUE_NUM_OFF], 0
    jne .fail_le_int_greater

    call test_end_pass
    jmp .test_gt_int_greater

.fail_le_int_greater:
    call test_end_fail

    ;-------------------------------------------------------------------------
    ; Test: op_gt 7 > 5 (should be true)
    ;-------------------------------------------------------------------------
.test_gt_int_greater:
    call reset_state
    lea rdi, [test_gt_int_greater]
    call test_start

    lea rdi, [bc_gt_7_5]
    mov rsi, bc_gt_7_5_len
    call vm_execute

    call stack_data_peek
    test rax, rax
    jz .fail_gt_int_greater

    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .fail_gt_int_greater
    cmp qword [rax + VALUE_NUM_OFF], 1
    jne .fail_gt_int_greater

    call test_end_pass
    jmp .test_gt_int_equal

.fail_gt_int_greater:
    call test_end_fail

    ;-------------------------------------------------------------------------
    ; Test: op_gt 5 > 5 (should be false)
    ;-------------------------------------------------------------------------
.test_gt_int_equal:
    call reset_state
    lea rdi, [test_gt_int_equal]
    call test_start

    lea rdi, [bc_gt_5_5]
    mov rsi, bc_gt_5_5_len
    call vm_execute

    call stack_data_peek
    test rax, rax
    jz .fail_gt_int_equal

    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .fail_gt_int_equal
    cmp qword [rax + VALUE_NUM_OFF], 0
    jne .fail_gt_int_equal

    call test_end_pass
    jmp .test_gt_int_less

.fail_gt_int_equal:
    call test_end_fail

    ;-------------------------------------------------------------------------
    ; Test: op_gt 3 > 5 (should be false)
    ;-------------------------------------------------------------------------
.test_gt_int_less:
    call reset_state
    lea rdi, [test_gt_int_less]
    call test_start

    lea rdi, [bc_gt_3_5]
    mov rsi, bc_gt_3_5_len
    call vm_execute

    call stack_data_peek
    test rax, rax
    jz .fail_gt_int_less

    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .fail_gt_int_less
    cmp qword [rax + VALUE_NUM_OFF], 0
    jne .fail_gt_int_less

    call test_end_pass
    jmp .test_ge_int_greater

.fail_gt_int_less:
    call test_end_fail

    ;-------------------------------------------------------------------------
    ; Test: op_ge 7 >= 5 (should be true)
    ;-------------------------------------------------------------------------
.test_ge_int_greater:
    call reset_state
    lea rdi, [test_ge_int_greater]
    call test_start

    lea rdi, [bc_ge_7_5]
    mov rsi, bc_ge_7_5_len
    call vm_execute

    call stack_data_peek
    test rax, rax
    jz .fail_ge_int_greater

    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .fail_ge_int_greater
    cmp qword [rax + VALUE_NUM_OFF], 1
    jne .fail_ge_int_greater

    call test_end_pass
    jmp .test_ge_int_equal

.fail_ge_int_greater:
    call test_end_fail

    ;-------------------------------------------------------------------------
    ; Test: op_ge 5 >= 5 (should be true)
    ;-------------------------------------------------------------------------
.test_ge_int_equal:
    call reset_state
    lea rdi, [test_ge_int_equal]
    call test_start

    lea rdi, [bc_ge_5_5]
    mov rsi, bc_ge_5_5_len
    call vm_execute

    call stack_data_peek
    test rax, rax
    jz .fail_ge_int_equal

    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .fail_ge_int_equal
    cmp qword [rax + VALUE_NUM_OFF], 1
    jne .fail_ge_int_equal

    call test_end_pass
    jmp .test_ge_int_less

.fail_ge_int_equal:
    call test_end_fail

    ;-------------------------------------------------------------------------
    ; Test: op_ge 3 >= 5 (should be false)
    ;-------------------------------------------------------------------------
.test_ge_int_less:
    call reset_state
    lea rdi, [test_ge_int_less]
    call test_start

    lea rdi, [bc_ge_3_5]
    mov rsi, bc_ge_3_5_len
    call vm_execute

    call stack_data_peek
    test rax, rax
    jz .fail_ge_int_less

    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .fail_ge_int_less
    cmp qword [rax + VALUE_NUM_OFF], 0
    jne .fail_ge_int_less

    call test_end_pass
    jmp .test_lt_negative

.fail_ge_int_less:
    call test_end_fail

    ;-------------------------------------------------------------------------
    ; Test: op_lt -5 < 3 (signed comparison, should be true)
    ;-------------------------------------------------------------------------
.test_lt_negative:
    call reset_state
    lea rdi, [test_lt_negative]
    call test_start

    lea rdi, [bc_lt_neg5_3]
    mov rsi, bc_lt_neg5_3_len
    call vm_execute

    call stack_data_peek
    test rax, rax
    jz .fail_lt_negative

    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .fail_lt_negative
    cmp qword [rax + VALUE_NUM_OFF], 1
    jne .fail_lt_negative

    call test_end_pass
    jmp .test_gt_negative

.fail_lt_negative:
    call test_end_fail

    ;-------------------------------------------------------------------------
    ; Test: op_gt 3 > -5 (signed comparison, should be true)
    ;-------------------------------------------------------------------------
.test_gt_negative:
    call reset_state
    lea rdi, [test_gt_negative]
    call test_start

    lea rdi, [bc_gt_3_neg5]
    mov rsi, bc_gt_3_neg5_len
    call vm_execute

    call stack_data_peek
    test rax, rax
    jz .fail_gt_negative

    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .fail_gt_negative
    cmp qword [rax + VALUE_NUM_OFF], 1
    jne .fail_gt_negative

    call test_end_pass
    jmp .done

.fail_gt_negative:
    call test_end_fail

    ;-------------------------------------------------------------------------
    ; Done - print summary
    ;-------------------------------------------------------------------------
.done:
    call print_test_summary

    pop r15
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret
