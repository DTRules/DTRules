; test_control.asm - Unit tests for Control Flow operations
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

; Boolean/value creation
extern value_new_boolean
extern value_new_array
extern array_alloc

; Control flow operators to test
extern op_if
extern op_ifelse
extern op_while
extern op_for
extern op_forall
extern op_break
extern op_continue
extern op_return
extern op_exec
extern op_nop
extern op_halt

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
    test_header:        db "=== Control Flow Operation Tests ===", 10, 0

    ; Test names
    test_nop:           db "nop", 0
    test_if_true:       db "if (true)", 0
    test_if_false:      db "if (false)", 0
    test_ifelse_true:   db "ifelse (true)", 0
    test_ifelse_false:  db "ifelse (false)", 0
    test_exec:          db "exec", 0

section .text
    global test_main

;-----------------------------------------------------------------------------
; Helper: Push a boolean value onto the data stack
; Input: rdi = 0 (false) or non-zero (true)
;-----------------------------------------------------------------------------
push_boolean:
    push rbp
    mov rbp, rsp
    push rbx

    mov rbx, rdi

    ; Create boolean value
    call value_new_boolean
    test rax, rax
    jz .done

    ; Push to stack
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12

    mov rcx, [rax]
    mov [r12], rcx
    mov rcx, [rax + 8]
    mov [r12 + 8], rcx
    mov rcx, [rax + 16]
    mov [r12 + 16], rcx

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; Helper: Create and push an empty executable array (code block)
; This represents a bytecode block for if/ifelse/while etc.
;-----------------------------------------------------------------------------
push_code_block:
    push rbp
    mov rbp, rsp
    push rbx

    ; Create array with capacity 8
    mov rdi, 8
    call array_alloc
    test rax, rax
    jz .done

    mov rbx, rax

    ; Create value
    call value_new_array
    test rax, rax
    jz .done

    ; Push to stack
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12

    mov rcx, [rax]
    mov [r12], rcx
    mov rcx, [rax + 8]
    mov [r12 + 8], rcx
    mov rcx, [rax + 16]
    mov [r12 + 16], rcx

    mov rax, rbx

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_main - Run all control flow tests
;-----------------------------------------------------------------------------
test_main:
    push rbp
    mov rbp, rsp

    ; Print header
    lea rdi, [test_header]
    call print_string

    ; Run tests
    call test_nop_op
    call test_if_true_op
    call test_if_false_op
    call test_ifelse_true_op
    call test_ifelse_false_op
    call test_exec_op

    ; Print summary
    call print_test_summary

    ; Return fail count
    mov rax, [fail_count]

    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_nop_op - Test nop (does nothing)
;-----------------------------------------------------------------------------
test_nop_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_nop]
    call test_start

    call reset_state

    ; Push a value
    mov rdi, 42
    call stack_data_push_integer

    ; Get stack depth before
    call stack_data_depth
    push rax

    ; Execute nop
    call op_nop

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Stack depth should be unchanged
    call stack_data_depth
    pop rsi
    mov rdi, rax
    call assert_eq
    test eax, eax
    jz .fail

    call test_end_pass
    jmp .done

.fail:
    add rsp, 8                  ; Clean up pushed value
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_if_true_op - Test if with true condition
; Stack: body cond -- (executes body)
;-----------------------------------------------------------------------------
test_if_true_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_if_true]
    call test_start

    call reset_state

    ; Push code block (body)
    call push_code_block
    test rax, rax
    jz .fail

    ; Push true condition
    mov rdi, 1
    call push_boolean

    ; Execute if
    call op_if

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_if_false_op - Test if with false condition
; Stack: body cond -- (does not execute body)
;-----------------------------------------------------------------------------
test_if_false_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_if_false]
    call test_start

    call reset_state

    ; Push code block (body)
    call push_code_block
    test rax, rax
    jz .fail

    ; Push false condition
    mov rdi, 0
    call push_boolean

    ; Execute if
    call op_if

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_ifelse_true_op - Test ifelse with true condition
; Stack: true_body false_body cond -- (executes true_body)
;-----------------------------------------------------------------------------
test_ifelse_true_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_ifelse_true]
    call test_start

    call reset_state

    ; Push true body
    call push_code_block
    test rax, rax
    jz .fail

    ; Push false body
    call push_code_block
    test rax, rax
    jz .fail

    ; Push true condition
    mov rdi, 1
    call push_boolean

    ; Execute ifelse
    call op_ifelse

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_ifelse_false_op - Test ifelse with false condition
; Stack: true_body false_body cond -- (executes false_body)
;-----------------------------------------------------------------------------
test_ifelse_false_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_ifelse_false]
    call test_start

    call reset_state

    ; Push true body
    call push_code_block
    test rax, rax
    jz .fail

    ; Push false body
    call push_code_block
    test rax, rax
    jz .fail

    ; Push false condition
    mov rdi, 0
    call push_boolean

    ; Execute ifelse
    call op_ifelse

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_exec_op - Test exec (execute array as code)
;-----------------------------------------------------------------------------
test_exec_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_exec]
    call test_start

    call reset_state

    ; Push empty code block
    call push_code_block
    test rax, rax
    jz .fail

    ; Execute it
    call op_exec

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret
