; test_boolean.asm - Unit tests for Boolean operations
; DTRules Assembly Implementation

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state

; Stack functions
extern stack_data_push_boolean
extern stack_data_pop
extern stack_data_peek

; Boolean operators to test (from bytecode.asm)
extern op_and
extern op_or
extern op_not
extern op_xor

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
    test_header:        db "=== Boolean Operation Tests ===", 10, 0
    test_and_tt:        db "and (T && T)", 0
    test_and_tf:        db "and (T && F)", 0
    test_and_ft:        db "and (F && T)", 0
    test_and_ff:        db "and (F && F)", 0
    test_or_tt:         db "or (T || T)", 0
    test_or_tf:         db "or (T || F)", 0
    test_or_ft:         db "or (F || T)", 0
    test_or_ff:         db "or (F || F)", 0
    test_not_t:         db "not (T)", 0
    test_not_f:         db "not (F)", 0
    test_xor_tt:        db "xor (T ^ T)", 0
    test_xor_tf:        db "xor (T ^ F)", 0
    test_xor_ft:        db "xor (F ^ T)", 0
    test_xor_ff:        db "xor (F ^ F)", 0

section .text
    global test_main

;-----------------------------------------------------------------------------
; test_main - Run all boolean tests
;-----------------------------------------------------------------------------
test_main:
    push rbp
    mov rbp, rsp

    ; Print header
    lea rdi, [test_header]
    call print_string

    ; AND tests
    call test_and_true_true
    call test_and_true_false
    call test_and_false_true
    call test_and_false_false

    ; OR tests
    call test_or_true_true
    call test_or_true_false
    call test_or_false_true
    call test_or_false_false

    ; NOT tests
    call test_not_true
    call test_not_false

    ; XOR tests
    call test_xor_true_true
    call test_xor_true_false
    call test_xor_false_true
    call test_xor_false_false

    ; Print summary
    call print_test_summary

    ; Return fail count as exit status
    mov rax, [fail_count]

    pop rbp
    ret

;-----------------------------------------------------------------------------
; AND tests
;-----------------------------------------------------------------------------

test_and_true_true:
    push rbp
    mov rbp, rsp

    lea rdi, [test_and_tt]
    call test_start

    call reset_state

    ; Push true, true
    mov rdi, 1
    call stack_data_push_boolean
    mov rdi, 1
    call stack_data_push_boolean

    ; AND
    call op_and

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

test_and_true_false:
    push rbp
    mov rbp, rsp

    lea rdi, [test_and_tf]
    call test_start

    call reset_state

    ; Push true, false
    mov rdi, 1
    call stack_data_push_boolean
    xor edi, edi
    call stack_data_push_boolean

    ; AND
    call op_and

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

test_and_false_true:
    push rbp
    mov rbp, rsp

    lea rdi, [test_and_ft]
    call test_start

    call reset_state

    ; Push false, true
    xor edi, edi
    call stack_data_push_boolean
    mov rdi, 1
    call stack_data_push_boolean

    ; AND
    call op_and

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

test_and_false_false:
    push rbp
    mov rbp, rsp

    lea rdi, [test_and_ff]
    call test_start

    call reset_state

    ; Push false, false
    xor edi, edi
    call stack_data_push_boolean
    xor edi, edi
    call stack_data_push_boolean

    ; AND
    call op_and

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
; OR tests
;-----------------------------------------------------------------------------

test_or_true_true:
    push rbp
    mov rbp, rsp

    lea rdi, [test_or_tt]
    call test_start

    call reset_state

    ; Push true, true
    mov rdi, 1
    call stack_data_push_boolean
    mov rdi, 1
    call stack_data_push_boolean

    ; OR
    call op_or

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

test_or_true_false:
    push rbp
    mov rbp, rsp

    lea rdi, [test_or_tf]
    call test_start

    call reset_state

    ; Push true, false
    mov rdi, 1
    call stack_data_push_boolean
    xor edi, edi
    call stack_data_push_boolean

    ; OR
    call op_or

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

test_or_false_true:
    push rbp
    mov rbp, rsp

    lea rdi, [test_or_ft]
    call test_start

    call reset_state

    ; Push false, true
    xor edi, edi
    call stack_data_push_boolean
    mov rdi, 1
    call stack_data_push_boolean

    ; OR
    call op_or

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

test_or_false_false:
    push rbp
    mov rbp, rsp

    lea rdi, [test_or_ff]
    call test_start

    call reset_state

    ; Push false, false
    xor edi, edi
    call stack_data_push_boolean
    xor edi, edi
    call stack_data_push_boolean

    ; OR
    call op_or

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
; NOT tests
;-----------------------------------------------------------------------------

test_not_true:
    push rbp
    mov rbp, rsp

    lea rdi, [test_not_t]
    call test_start

    call reset_state

    ; Push true
    mov rdi, 1
    call stack_data_push_boolean

    ; NOT
    call op_not

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

test_not_false:
    push rbp
    mov rbp, rsp

    lea rdi, [test_not_f]
    call test_start

    call reset_state

    ; Push false
    xor edi, edi
    call stack_data_push_boolean

    ; NOT
    call op_not

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
; XOR tests
;-----------------------------------------------------------------------------

test_xor_true_true:
    push rbp
    mov rbp, rsp

    lea rdi, [test_xor_tt]
    call test_start

    call reset_state

    ; Push true, true
    mov rdi, 1
    call stack_data_push_boolean
    mov rdi, 1
    call stack_data_push_boolean

    ; XOR
    call op_xor

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is false (T xor T = F)
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

test_xor_true_false:
    push rbp
    mov rbp, rsp

    lea rdi, [test_xor_tf]
    call test_start

    call reset_state

    ; Push true, false
    mov rdi, 1
    call stack_data_push_boolean
    xor edi, edi
    call stack_data_push_boolean

    ; XOR
    call op_xor

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is true (T xor F = T)
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

test_xor_false_true:
    push rbp
    mov rbp, rsp

    lea rdi, [test_xor_ft]
    call test_start

    call reset_state

    ; Push false, true
    xor edi, edi
    call stack_data_push_boolean
    mov rdi, 1
    call stack_data_push_boolean

    ; XOR
    call op_xor

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is true (F xor T = T)
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

test_xor_false_false:
    push rbp
    mov rbp, rsp

    lea rdi, [test_xor_ff]
    call test_start

    call reset_state

    ; Push false, false
    xor edi, edi
    call stack_data_push_boolean
    xor edi, edi
    call stack_data_push_boolean

    ; XOR
    call op_xor

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Check result is false (F xor F = F)
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
