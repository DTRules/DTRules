; test_value.asm - Unit tests for Value type
; DTRules Assembly Implementation

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"

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
extern pools_init
extern heap_init

section .data
    test_integer_msg:   db "Testing integer values...", 10, 0
    test_boolean_msg:   db "Testing boolean values...", 10, 0
    test_null_msg:      db "Testing null values...", 10, 0
    test_truthy_msg:    db "Testing truthy...", 10, 0
    test_equals_msg:    db "Testing equals...", 10, 0
    pass_msg:           db "  PASS", 10, 0
    fail_msg:           db "  FAIL", 10, 0
    all_pass_msg:       db "All value tests passed!", 10, 0

    test_count:         dq 0
    fail_count:         dq 0

section .text
    global test_value_main

;-----------------------------------------------------------------------------
; Test runner macros
;-----------------------------------------------------------------------------

%macro TEST_START 1
    lea rdi, [%1]
    call print_string
%endmacro

%macro ASSERT_EQ 2
    mov rdi, %1
    mov rsi, %2
    call assert_equal
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
; test_value_main - Run all value tests
;-----------------------------------------------------------------------------
test_value_main:
    push rbp
    mov rbp, rsp

    ; Initialize pools
    call pools_init
    call heap_init

    ; Run tests
    call test_integer_values
    call test_boolean_values
    call test_null_values
    call test_truthy_values
    call test_equality

    ; Report results
    mov rax, [fail_count]
    test rax, rax
    jnz .has_failures

    lea rdi, [all_pass_msg]
    call print_string
    xor eax, eax
    jmp .done

.has_failures:
    mov eax, 1

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_integer_values
;-----------------------------------------------------------------------------
test_integer_values:
    push rbp
    mov rbp, rsp
    push rbx

    TEST_START test_integer_msg

    ; Create integer value 42
    mov rdi, 42
    call value_new_integer
    test rax, rax
    jz .fail
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

    lea rdi, [pass_msg]
    call print_string
    jmp .done

.fail:
    lea rdi, [fail_msg]
    call print_string
    inc qword [fail_count]

.done:
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

    lea rdi, [pass_msg]
    call print_string
    jmp .done

.fail:
    lea rdi, [fail_msg]
    call print_string
    inc qword [fail_count]

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

    lea rdi, [pass_msg]
    call print_string
    jmp .done

.fail:
    lea rdi, [fail_msg]
    call print_string
    inc qword [fail_count]

.done:
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

    lea rdi, [pass_msg]
    call print_string
    jmp .done

.fail:
    lea rdi, [fail_msg]
    call print_string
    inc qword [fail_count]

.done:
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
    push r12

    TEST_START test_equals_msg

    ; Same integers are equal
    mov rdi, 42
    call value_new_integer
    mov rbx, rax

    mov rdi, 42
    call value_new_integer
    mov r12, rax

    mov rdi, rbx
    mov rsi, r12
    call value_equals
    ASSERT_TRUE rax

    ; Different integers are not equal
    mov rdi, 100
    call value_new_integer
    mov r12, rax

    mov rdi, rbx
    mov rsi, r12
    call value_equals
    ASSERT_FALSE rax

    ; null equals null
    call value_new_null
    mov rbx, rax

    call value_new_null
    mov r12, rax

    mov rdi, rbx
    mov rsi, r12
    call value_equals
    ASSERT_TRUE rax

    lea rdi, [pass_msg]
    call print_string
    jmp .done

.fail:
    lea rdi, [fail_msg]
    call print_string
    inc qword [fail_count]

.done:
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; Assertion helpers
;-----------------------------------------------------------------------------

assert_equal:
    inc qword [test_count]
    cmp rdi, rsi
    jne .fail
    ret
.fail:
    inc qword [fail_count]
    ret

assert_true:
    inc qword [test_count]
    test rdi, rdi
    jz .fail
    ret
.fail:
    inc qword [fail_count]
    ret

assert_false:
    inc qword [test_count]
    test rdi, rdi
    jnz .fail
    ret
.fail:
    inc qword [fail_count]
    ret
