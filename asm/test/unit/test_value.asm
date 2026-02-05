; test_value.asm - Unit tests for Value type
; DTRules Assembly Implementation

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state
extern value_new_integer
extern value_new_boolean
extern value_new_null
extern value_new_double
extern value_get_tag
extern value_get_integer
extern value_get_boolean
extern value_is_truthy
extern value_equals
extern value_compare
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
    test_integer_msg:   db "integer values", 0
    test_boolean_msg:   db "boolean values", 0
    test_null_msg:      db "null values", 0
    test_truthy_msg:    db "truthy evaluation", 0
    test_equals_msg:    db "value equality", 0
    test_header:        db "=== Value Type Tests ===", 10, 0

section .text
    global test_main

;-----------------------------------------------------------------------------
; Test helper macros
;-----------------------------------------------------------------------------

%macro TEST_START 1
    lea rdi, [%1]
    call test_start
%endmacro

%macro ASSERT_EQ 2
    mov rdi, %1
    mov rsi, %2
    call assert_eq
    test eax, eax
    jz %%fail
    jmp %%done
%%fail:
    ; Continue testing but mark failure
%%done:
%endmacro

%macro ASSERT_TRUE 1
    mov rdi, %1
    call assert_true
%endmacro

%macro ASSERT_FALSE 1
    mov rdi, %1
    call assert_false
%endmacro

;-----------------------------------------------------------------------------
; test_main - Run all value tests (called from harness)
;-----------------------------------------------------------------------------
test_main:
    push rbp
    mov rbp, rsp

    ; Print header
    lea rdi, [test_header]
    call print_string

    ; Run tests
    call test_integer_values
    call test_boolean_values
    call test_null_values
    call test_truthy_values
    call test_equality

    ; Print summary
    call print_test_summary

    ; Return fail count as exit status
    mov rax, [fail_count]

    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_integer_values
;-----------------------------------------------------------------------------
test_integer_values:
    push rbp
    mov rbp, rsp
    push rbx
    push r15                    ; Use r15 instead of r12 (r12 is stack pointer!)

    TEST_START test_integer_msg
    xor r15d, r15d              ; Track local failures

    ; Create integer value 42
    mov rdi, 42
    call value_new_integer
    test rax, rax
    jz .alloc_fail
    mov rbx, rax

    ; Check tag
    mov rdi, rbx
    call value_get_tag
    ASSERT_EQ rax, VTAG_INTEGER

    ; Check value
    mov rdi, rbx
    call value_get_integer
    ASSERT_EQ rax, 42

    ; Create negative integer
    mov rdi, -100
    call value_new_integer
    mov rbx, rax

    mov rdi, rbx
    call value_get_integer
    ASSERT_EQ rax, -100

    ; Create zero
    xor edi, edi
    call value_new_integer
    mov rbx, rax

    mov rdi, rbx
    call value_get_integer
    ASSERT_EQ rax, 0

    call test_end_pass
    jmp .done

.alloc_fail:
    call test_end_fail
    inc r15

.done:
    pop r15
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_boolean_values
;-----------------------------------------------------------------------------
test_boolean_values:
    push rbp
    mov rbp, rsp
    push rbx

    TEST_START test_boolean_msg

    ; Create true
    mov rdi, 1
    call value_new_boolean
    mov rbx, rax

    mov rdi, rbx
    call value_get_tag
    ASSERT_EQ rax, VTAG_BOOLEAN

    mov rdi, rbx
    call value_get_boolean
    ASSERT_TRUE rax

    ; Create false
    xor edi, edi
    call value_new_boolean
    mov rbx, rax

    mov rdi, rbx
    call value_get_boolean
    ASSERT_FALSE rax

    call test_end_pass
    jmp .done

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_null_values
;-----------------------------------------------------------------------------
test_null_values:
    push rbp
    mov rbp, rsp
    push rbx

    TEST_START test_null_msg

    call value_new_null
    mov rbx, rax

    mov rdi, rbx
    call value_get_tag
    ASSERT_EQ rax, VTAG_NULL

    call test_end_pass

    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_truthy_values
;-----------------------------------------------------------------------------
test_truthy_values:
    push rbp
    mov rbp, rsp
    push rbx

    TEST_START test_truthy_msg

    ; null is falsy
    call value_new_null
    mov rdi, rax
    call value_is_truthy
    ASSERT_FALSE rax

    ; 0 is falsy
    xor edi, edi
    call value_new_integer
    mov rdi, rax
    call value_is_truthy
    ASSERT_FALSE rax

    ; 1 is truthy
    mov rdi, 1
    call value_new_integer
    mov rdi, rax
    call value_is_truthy
    ASSERT_TRUE rax

    ; -1 is truthy
    mov rdi, -1
    call value_new_integer
    mov rdi, rax
    call value_is_truthy
    ASSERT_TRUE rax

    ; false is falsy
    xor edi, edi
    call value_new_boolean
    mov rdi, rax
    call value_is_truthy
    ASSERT_FALSE rax

    ; true is truthy
    mov rdi, 1
    call value_new_boolean
    mov rdi, rax
    call value_is_truthy
    ASSERT_TRUE rax

    call test_end_pass

    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_equality
;-----------------------------------------------------------------------------
test_equality:
    push rbp
    mov rbp, rsp
    push rbx
    push r15                    ; Use r15 instead of r12 (r12 is stack pointer!)

    TEST_START test_equals_msg

    ; Same integers are equal
    mov rdi, 42
    call value_new_integer
    mov rbx, rax

    mov rdi, 42
    call value_new_integer
    mov r15, rax

    mov rdi, rbx
    mov rsi, r15
    call value_equals
    ASSERT_TRUE rax

    ; Different integers are not equal
    mov rdi, 100
    call value_new_integer
    mov r15, rax

    mov rdi, rbx
    mov rsi, r15
    call value_equals
    ASSERT_FALSE rax

    ; null equals null
    call value_new_null
    mov rbx, rax

    call value_new_null
    mov r15, rax

    mov rdi, rbx
    mov rsi, r15
    call value_equals
    ASSERT_TRUE rax

    call test_end_pass

    pop r15
    pop rbx
    pop rbp
    ret
