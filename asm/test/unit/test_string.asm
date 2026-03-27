; test_string.asm - Unit tests for String operations
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

; String creation
extern string_alloc
extern value_new_string

; String operators to test
extern op_str_concat
extern op_str_length
extern op_str_substring
extern op_str_indexof
extern op_str_lastindexof
extern op_str_startswith
extern op_str_endswith
extern op_str_contains
extern op_str_replace
extern op_str_replaceall
extern op_str_split
extern op_str_join
extern op_str_trim
extern op_str_trimleft
extern op_str_trimright
extern op_str_toupper
extern op_str_tolower
extern op_str_tostring
extern op_str_tointeger
extern op_str_todouble

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
    test_header:        db "=== String Operation Tests ===", 10, 0

    ; Test names
    test_concat:        db "str_concat", 0
    test_length:        db "str_length", 0
    test_substring:     db "str_substring", 0
    test_indexof:       db "str_indexof", 0
    test_lastindexof:   db "str_lastindexof", 0
    test_startswith:    db "str_startswith", 0
    test_endswith:      db "str_endswith", 0
    test_contains:      db "str_contains", 0
    test_replace:       db "str_replace", 0
    test_replaceall:    db "str_replaceall", 0
    test_split:         db "str_split", 0
    test_join:          db "str_join", 0
    test_trim:          db "str_trim", 0
    test_trimleft:      db "str_trimleft", 0
    test_trimright:     db "str_trimright", 0
    test_toupper:       db "str_toupper", 0
    test_tolower:       db "str_tolower", 0
    test_tostring:      db "str_tostring", 0
    test_tointeger:     db "str_tointeger", 0
    test_todouble:      db "str_todouble", 0

    ; Test strings (length-prefixed format: 8-byte length + data + null)
    align 8
    str_hello:
        dq 5
        db "hello", 0
    align 8
    str_world:
        dq 5
        db "world", 0
    align 8
    str_helloworld:
        dq 10
        db "helloworld", 0
    align 8
    str_space:
        dq 1
        db " ", 0
    align 8
    str_hello_space_world:
        dq 11
        db "hello world", 0
    align 8
    str_ell:
        dq 3
        db "ell", 0
    align 8
    str_empty:
        dq 0
        db 0
    align 8
    str_trimtest:
        dq 13
        db "  trimtest   ", 0
    align 8
    str_trimmed:
        dq 8
        db "trimtest", 0
    align 8
    str_upper:
        dq 5
        db "HELLO", 0
    align 8
    str_lower:
        dq 5
        db "hello", 0
    align 8
    str_number:
        dq 3
        db "123", 0
    align 8
    str_float:
        dq 4
        db "3.14", 0
    align 8
    str_comma:
        dq 1
        db ",", 0
    align 8
    str_abc:
        dq 3
        db "abc", 0
    align 8
    str_ll:
        dq 2
        db "ll", 0
    align 8
    str_xx:
        dq 2
        db "xx", 0
    align 8
    str_hexxo:
        dq 5
        db "hexxo", 0

section .text
    global test_main

;-----------------------------------------------------------------------------
; Helper: Push a string value onto the data stack
; Input: rdi = pointer to length-prefixed string (8-byte length + data)
;-----------------------------------------------------------------------------
push_string_value:
    push rbp
    mov rbp, rsp
    push rbx

    mov rbx, rdi                ; Save pointer to length-prefixed string
    mov rsi, [rbx]              ; Get length from prefix
    lea rdi, [rbx + 8]          ; Point to actual string data

    ; Create a Value with the string
    call value_new_string
    test rax, rax
    jz .done

    ; Push to data stack - value is already in correct format
    ; r12 is data stack pointer
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12

    ; Copy the Value (24 bytes)
    mov rcx, [rax]              ; tag
    mov [r12], rcx
    mov rcx, [rax + 8]          ; num
    mov [r12 + 8], rcx
    mov rcx, [rax + 16]         ; ptr
    mov [r12 + 16], rcx

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_main - Run all string tests
;-----------------------------------------------------------------------------
test_main:
    push rbp
    mov rbp, rsp

    ; Print header
    lea rdi, [test_header]
    call print_string

    ; Run tests
    call test_str_concat_op
    call test_str_length_op
    call test_str_substring_op
    call test_str_indexof_op
    call test_str_lastindexof_op
    call test_str_startswith_op
    call test_str_endswith_op
    call test_str_contains_op
    call test_str_replace_op
    call test_str_replaceall_op
    call test_str_trim_op
    call test_str_trimleft_op
    call test_str_trimright_op
    call test_str_toupper_op
    call test_str_tolower_op
    call test_str_tostring_op
    call test_str_tointeger_op
    call test_str_todouble_op

    ; Print summary
    call print_test_summary

    ; Return fail count as exit status
    mov rax, [fail_count]

    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_str_concat_op - Test string concatenation: "hello" + "world" = "helloworld"
;-----------------------------------------------------------------------------
test_str_concat_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_concat]
    call test_start

    call reset_state

    ; Push "hello"
    lea rdi, [str_hello]
    call push_string_value

    ; Push "world"
    lea rdi, [str_world]
    call push_string_value

    ; Concat
    call op_str_concat

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result - pop and verify length is 10
    call stack_data_pop
    test rax, rax
    jz .fail

    ; Get string pointer
    mov rdi, [rax + VALUE_PTR_OFF]
    test rdi, rdi
    jz .fail

    ; Check length
    mov rsi, [rdi]              ; Get length from string header
    cmp rsi, 10
    jne .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_str_length_op - Test string length: len("hello") = 5
;-----------------------------------------------------------------------------
test_str_length_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_length]
    call test_start

    call reset_state

    ; Push "hello"
    lea rdi, [str_hello]
    call push_string_value

    ; Length
    call op_str_length

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
; test_str_substring_op - Test substring: substr("hello", 1, 3) = "ell"
;-----------------------------------------------------------------------------
test_str_substring_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_substring]
    call test_start

    call reset_state

    ; Push "hello"
    lea rdi, [str_hello]
    call push_string_value

    ; Push start index 1
    mov rdi, 1
    call stack_data_push_integer

    ; Push length 3
    mov rdi, 3
    call stack_data_push_integer

    ; Substring
    call op_str_substring

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result length is 3
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_PTR_OFF]
    test rdi, rdi
    jz .fail

    mov rsi, [rdi]              ; length
    cmp rsi, 3
    jne .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_str_indexof_op - Test index of: indexOf("hello", "ll") = 2
;-----------------------------------------------------------------------------
test_str_indexof_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_indexof]
    call test_start

    call reset_state

    ; Push "hello"
    lea rdi, [str_hello]
    call push_string_value

    ; Push "ll"
    lea rdi, [str_ll]
    call push_string_value

    ; IndexOf
    call op_str_indexof

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is 2
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
; test_str_lastindexof_op - Test last index of: lastIndexOf("hello", "l") = 3
;-----------------------------------------------------------------------------
test_str_lastindexof_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_lastindexof]
    call test_start

    call reset_state

    ; Push "hello"
    lea rdi, [str_hello]
    call push_string_value

    ; Push "l" (single char - use first char of "ll")
    lea rdi, [str_ll]
    call push_string_value

    ; LastIndexOf - looking for "ll", last occurrence at index 2
    call op_str_lastindexof

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is 2 (last occurrence of "ll" in "hello")
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
; test_str_startswith_op - Test startsWith: startsWith("hello", "hel") = true
;-----------------------------------------------------------------------------
test_str_startswith_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_startswith]
    call test_start

    call reset_state

    ; Push "hello"
    lea rdi, [str_hello]
    call push_string_value

    ; Push "ell" to test - should return false
    lea rdi, [str_ell]
    call push_string_value

    ; StartsWith
    call op_str_startswith

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is false (0) - "hello" doesn't start with "ell"
    call stack_data_pop
    test rax, rax
    jz .fail

    ; Check tag is boolean
    movzx edi, byte [rax + VALUE_TAG_OFF]
    cmp edi, VTAG_BOOLEAN
    jne .fail

    ; Check value is false
    mov rdi, [rax + VALUE_NUM_OFF]
    call assert_false
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
; test_str_endswith_op - Test endsWith: endsWith("hello", "lo") = true
;-----------------------------------------------------------------------------
test_str_endswith_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_endswith]
    call test_start

    call reset_state

    ; Push "hello"
    lea rdi, [str_hello]
    call push_string_value

    ; Push "ll"
    lea rdi, [str_ll]
    call push_string_value

    ; EndsWith - "hello" doesn't end with "ll"
    call op_str_endswith

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is false
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    call assert_false
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
; test_str_contains_op - Test contains: contains("hello", "ell") = true
;-----------------------------------------------------------------------------
test_str_contains_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_contains]
    call test_start

    call reset_state

    ; Push "hello"
    lea rdi, [str_hello]
    call push_string_value

    ; Push "ell"
    lea rdi, [str_ell]
    call push_string_value

    ; Contains
    call op_str_contains

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is true
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    call assert_true
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
; test_str_replace_op - Test replace: replace("hello", "ll", "xx") = "hexxo"
;-----------------------------------------------------------------------------
test_str_replace_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_replace]
    call test_start

    call reset_state

    ; Push "hello"
    lea rdi, [str_hello]
    call push_string_value

    ; Push "ll" (pattern)
    lea rdi, [str_ll]
    call push_string_value

    ; Push "xx" (replacement)
    lea rdi, [str_xx]
    call push_string_value

    ; Replace
    call op_str_replace

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result length is 5 (same as "hexxo")
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_PTR_OFF]
    test rdi, rdi
    jz .fail

    mov rsi, [rdi]
    cmp rsi, 5
    jne .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_str_replaceall_op - Test replaceAll
;-----------------------------------------------------------------------------
test_str_replaceall_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_replaceall]
    call test_start

    call reset_state

    ; Push "hello"
    lea rdi, [str_hello]
    call push_string_value

    ; Push "l" (pattern - will match both l's)
    lea rdi, [str_ll]
    call push_string_value

    ; Push "xx" (replacement)
    lea rdi, [str_xx]
    call push_string_value

    ; ReplaceAll
    call op_str_replaceall

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Pop result - just verify it's a string
    call stack_data_pop
    test rax, rax
    jz .fail

    movzx edi, byte [rax + VALUE_TAG_OFF]
    cmp edi, VTAG_STRING
    jne .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_str_trim_op - Test trim: trim("  hello  ") removes leading/trailing
;-----------------------------------------------------------------------------
test_str_trim_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_trim]
    call test_start

    call reset_state

    ; Push "  trimtest   "
    lea rdi, [str_trimtest]
    call push_string_value

    ; Trim
    call op_str_trim

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result length is 8 ("trimtest")
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_PTR_OFF]
    test rdi, rdi
    jz .fail

    mov rsi, [rdi]
    cmp rsi, 8
    jne .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_str_trimleft_op - Test trimLeft
;-----------------------------------------------------------------------------
test_str_trimleft_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_trimleft]
    call test_start

    call reset_state

    ; Push "  trimtest   "
    lea rdi, [str_trimtest]
    call push_string_value

    ; TrimLeft
    call op_str_trimleft

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result length is 11 ("trimtest   ")
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_PTR_OFF]
    test rdi, rdi
    jz .fail

    mov rsi, [rdi]
    cmp rsi, 11                 ; "trimtest   " without leading spaces
    jne .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_str_trimright_op - Test trimRight
;-----------------------------------------------------------------------------
test_str_trimright_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_trimright]
    call test_start

    call reset_state

    ; Push "  trimtest   "
    lea rdi, [str_trimtest]
    call push_string_value

    ; TrimRight
    call op_str_trimright

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result length is 10 ("  trimtest")
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_PTR_OFF]
    test rdi, rdi
    jz .fail

    mov rsi, [rdi]
    cmp rsi, 10                 ; "  trimtest" without trailing spaces
    jne .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_str_toupper_op - Test toUpper: toUpper("hello") = "HELLO"
;-----------------------------------------------------------------------------
test_str_toupper_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_toupper]
    call test_start

    call reset_state

    ; Push "hello"
    lea rdi, [str_hello]
    call push_string_value

    ; ToUpper
    call op_str_toupper

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Verify result is a string of length 5
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_PTR_OFF]
    test rdi, rdi
    jz .fail

    mov rsi, [rdi]
    cmp rsi, 5
    jne .fail

    ; Check first char is 'H' (uppercase)
    movzx eax, byte [rdi + 8]   ; Skip length, get first char
    cmp al, 'H'
    jne .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_str_tolower_op - Test toLower: toLower("HELLO") = "hello"
;-----------------------------------------------------------------------------
test_str_tolower_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_tolower]
    call test_start

    call reset_state

    ; Push "HELLO"
    lea rdi, [str_upper]
    call push_string_value

    ; ToLower
    call op_str_tolower

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Verify result
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_PTR_OFF]
    test rdi, rdi
    jz .fail

    ; Check first char is 'h' (lowercase)
    movzx eax, byte [rdi + 8]
    cmp al, 'h'
    jne .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_str_tostring_op - Test toString: toString(123) = "123"
;-----------------------------------------------------------------------------
test_str_tostring_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_tostring]
    call test_start

    call reset_state

    ; Push integer 123
    mov rdi, 123
    call stack_data_push_integer

    ; ToString
    call op_str_tostring

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Verify result is a string
    call stack_data_pop
    test rax, rax
    jz .fail

    movzx edi, byte [rax + VALUE_TAG_OFF]
    cmp edi, VTAG_STRING
    jne .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_str_tointeger_op - Test toInteger: toInteger("123") = 123
;-----------------------------------------------------------------------------
test_str_tointeger_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_tointeger]
    call test_start

    call reset_state

    ; Push "123"
    lea rdi, [str_number]
    call push_string_value

    ; ToInteger
    call op_str_tointeger

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Verify result is 123
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, 123
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
; test_str_todouble_op - Test toDouble: toDouble("3.14") ~ 3.14
;-----------------------------------------------------------------------------
test_str_todouble_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_todouble]
    call test_start

    call reset_state

    ; Push "3.14"
    lea rdi, [str_float]
    call push_string_value

    ; ToDouble
    call op_str_todouble

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Verify result is a double
    call stack_data_pop
    test rax, rax
    jz .fail

    movzx edi, byte [rax + VALUE_TAG_OFF]
    cmp edi, VTAG_DOUBLE
    jne .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret
