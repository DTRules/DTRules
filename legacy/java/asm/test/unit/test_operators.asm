; test_operators.asm - Test operator dispatch via op_operator
; Verifies that operator table dispatch works correctly

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state

; Stack functions
extern stack_data_pop
extern stack_data_push
extern stack_data_push_integer
extern stack_data_push_boolean
extern stack_data_depth

; VM execution
extern vm_execute
extern op_operator

; Print functions
extern print_string
extern print_integer
extern print_newline

; Test harness functions
extern test_start
extern test_end_pass
extern test_end_fail
extern print_test_summary
extern reset_state
extern test_count
extern fail_count
extern pass_count
extern pools_init

section .data
    test_header:        db "=== Operator Dispatch Tests ===", 10, 0
    test_add_op:        db "op_add_via_operator", 0
    test_beq_op:        db "op_beq", 0
    test_isnull_op:     db "op_isnull", 0
    test_dup_op:        db "op_dup_via_operator", 0
    depth_msg:          db "  depth: ", 0
    result_msg:         db "  result: ", 0

    ; Test bytecodes
    ; Test: 1 2 + via operator (OpPushOne, OpPushInt 2, OpOperator 96)
    ; Operator 96 is '+'
    add_op_bc:
        db 104          ; OpPushOne (push 1)
        db 2, 2         ; OpPushInt + varint(2) (push 2)
        db 200, 96      ; OpOperator + varint(96) (add)
    add_op_bc_end:

    ; Test: true true beq (should be true)
    ; OpPushTrue, OpPushTrue, OpOperator 34 (beq)
    beq_bc:
        db 100          ; OpPushTrue
        db 100          ; OpPushTrue
        db 200, 34      ; OpOperator 34 (beq)
    beq_bc_end:

    ; Test: null isnull (should be true)
    ; OpPushNull, OpOperator 35 (isnull)
    isnull_bc:
        db 102          ; OpPushNull
        db 200, 35      ; OpOperator 35 (isnull)
    isnull_bc_end:

    ; Test: 42 dup via operator
    ; OpPushInt 42, OpOperator 110 (dup)
    dup_op_bc:
        db 2, 42        ; OpPushInt 42
        db 200, 110     ; OpOperator 110 (dup)
    dup_op_bc_end:

section .text
    global test_main

;-----------------------------------------------------------------------------
; test_main - Run operator dispatch tests
;-----------------------------------------------------------------------------
test_main:
    push rbp
    mov rbp, rsp

    ; Print header
    lea rdi, [test_header]
    call print_string

    ; Run tests
    call test_add_via_operator
    call test_beq_operator
    call test_isnull_operator
    call test_dup_via_operator

    ; Print summary
    call print_test_summary

    ; Return fail count as exit status
    mov rax, [fail_count]

    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_add_via_operator - Test: 1 2 + via OpOperator
;-----------------------------------------------------------------------------
test_add_via_operator:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_add_op]
    call test_start

    call reset_state

    ; Set up bytecode pointers
    lea rax, [add_op_bc]
    mov [state + State.bytecode], rax
    lea rax, [add_op_bc_end]
    mov [state + State.bytecode_end], rax

    ; Execute bytecode
    call vm_execute
    mov rbx, rax

    ; Check for error
    test ebx, ebx
    jnz .fail

    ; Check depth is 1
    call stack_data_depth
    cmp rax, 1
    jne .fail

    ; Pop and verify result is 3
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    cmp rdi, 3
    jne .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_beq_operator - Test: true true beq
;-----------------------------------------------------------------------------
test_beq_operator:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_beq_op]
    call test_start

    call reset_state

    ; Set up bytecode
    lea rax, [beq_bc]
    mov [state + State.bytecode], rax
    lea rax, [beq_bc_end]
    mov [state + State.bytecode_end], rax

    ; Execute
    call vm_execute
    mov rbx, rax

    test ebx, ebx
    jnz .fail

    ; Check depth is 1
    call stack_data_depth
    cmp rax, 1
    jne .fail

    ; Pop and verify result is true (1)
    call stack_data_pop
    test rax, rax
    jz .fail

    ; Check it's a boolean with value 1
    movzx ecx, byte [rax]
    cmp cl, VTAG_BOOLEAN
    jne .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    cmp rdi, 1
    jne .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_isnull_operator - Test: null isnull
;-----------------------------------------------------------------------------
test_isnull_operator:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_isnull_op]
    call test_start

    call reset_state

    ; Set up bytecode
    lea rax, [isnull_bc]
    mov [state + State.bytecode], rax
    lea rax, [isnull_bc_end]
    mov [state + State.bytecode_end], rax

    ; Execute
    call vm_execute
    mov rbx, rax

    test ebx, ebx
    jnz .fail

    ; Check depth is 1
    call stack_data_depth
    cmp rax, 1
    jne .fail

    ; Pop and verify result is true
    call stack_data_pop
    test rax, rax
    jz .fail

    movzx ecx, byte [rax]
    cmp cl, VTAG_BOOLEAN
    jne .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    cmp rdi, 1
    jne .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_dup_via_operator - Test: 42 dup via operator
;-----------------------------------------------------------------------------
test_dup_via_operator:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_dup_op]
    call test_start

    call reset_state

    ; Set up bytecode
    lea rax, [dup_op_bc]
    mov [state + State.bytecode], rax
    lea rax, [dup_op_bc_end]
    mov [state + State.bytecode_end], rax

    ; Execute
    call vm_execute
    mov rbx, rax

    test ebx, ebx
    jnz .fail

    ; Check depth is 2 (42 was duplicated)
    call stack_data_depth
    cmp rax, 2
    jne .fail

    ; Pop and verify both are 42
    call stack_data_pop
    test rax, rax
    jz .fail
    mov rdi, [rax + VALUE_NUM_OFF]
    cmp rdi, 42
    jne .fail

    call stack_data_pop
    test rax, rax
    jz .fail
    mov rdi, [rax + VALUE_NUM_OFF]
    cmp rdi, 42
    jne .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbx
    pop rbp
    ret
