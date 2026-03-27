; test_bytecode_load.asm - Test bytecode loading and execution
; Verifies Go-compiled bytecode runs correctly on ASM runtime

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state

; Stack functions
extern stack_data_pop
extern stack_data_push_integer
extern stack_data_depth

; VM execution
extern vm_execute

; Print functions
extern print_string
extern print_integer
extern print_newline
extern print_hex

; Test harness functions
extern test_start
extern test_end_pass
extern test_end_fail
extern assert_eq
extern print_test_summary
extern reset_state
extern test_count
extern fail_count
extern pass_count
extern pools_init

section .data
    test_header:        db "=== Bytecode Load Tests ===", 10, 0
    test_simple_add:    db "simple_add_bytecode", 0
    test_two_push_name: db "two_pushes", 0
    test_pushint_name:  db "push_int", 0
    err_msg:            db "  vm_execute returned: ", 0
    depth_msg:          db "  stack depth: ", 0
    result_msg:         db "  result: ", 0
    bc_addr_msg:        db "  bytecode addr: ", 0
    first_byte_msg:     db "  first byte: ", 0
    state_bc_msg:       db "  state.bytecode: ", 0

    ; Simple bytecode: push 1, push 2, add -> should result in 3
    ; OpPushOne (104), OpPushInt (2) + varint(2), OpAdd (20)
    simple_add_bc:
        db 104          ; OpPushOne (push 1)
        db 2, 2         ; OpPushInt + varint(2) (push 2)
        db 20           ; OpAdd
    simple_add_bc_end:

    ; Even simpler test: just push two integers
    ; OpPushOne (104), OpPushOne (104) -> stack should have [1, 1]
    simple_two_bc:
        db 104          ; OpPushOne (push 1)
        db 104          ; OpPushOne (push 1)
    simple_two_bc_end:

    ; Test OpPushInt: push 42 -> should have value 42
    pushint_bc:
        db 2            ; OpPushInt
        db 42           ; varint(42)
    pushint_bc_end:

section .text
    global test_main

;-----------------------------------------------------------------------------
; test_main - Run bytecode load tests
;-----------------------------------------------------------------------------
test_main:
    push rbp
    mov rbp, rsp

    ; Print header
    lea rdi, [test_header]
    call print_string

    ; Run tests
    call test_two_pushes
    call test_push_int
    call test_simple_add_execution

    ; Print summary
    call print_test_summary

    ; Return fail count as exit status
    mov rax, [fail_count]

    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_two_pushes - Test: push 1, push 1 -> depth 2
;-----------------------------------------------------------------------------
test_two_pushes:
    push rbp
    mov rbp, rsp

    lea rdi, [test_two_push_name]
    call test_start

    call reset_state

    ; Set up bytecode pointers
    lea rax, [simple_two_bc]
    mov [state + State.bytecode], rax
    lea rax, [simple_two_bc_end]
    mov [state + State.bytecode_end], rax

    ; Execute bytecode
    call vm_execute

    ; Check for execution error
    test eax, eax
    jnz .fail

    ; Check stack depth is 2
    call stack_data_depth
    cmp rax, 2
    jne .fail

    call test_end_pass
    jmp .done

.fail:
    ; Print actual depth
    push rax
    lea rdi, [depth_msg]
    call print_string
    pop rdi
    call print_integer
    call print_newline
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_push_int - Test: push 42 via OpPushInt
;-----------------------------------------------------------------------------
test_push_int:
    push rbp
    mov rbp, rsp

    lea rdi, [test_pushint_name]
    call test_start

    call reset_state

    ; Set up bytecode pointers
    lea rax, [pushint_bc]
    mov [state + State.bytecode], rax
    lea rax, [pushint_bc_end]
    mov [state + State.bytecode_end], rax

    ; Execute bytecode
    call vm_execute

    ; Check for execution error
    test eax, eax
    jnz .fail

    ; Check stack depth is 1
    call stack_data_depth
    cmp rax, 1
    jne .depth_fail

    ; Pop and verify value is 42
    call stack_data_pop
    test rax, rax
    jz .fail
    push rax

    mov rax, [rsp]
    mov rdi, [rax + VALUE_NUM_OFF]
    pop rax
    cmp rdi, 42
    jne .value_fail

    call test_end_pass
    jmp .done

.depth_fail:
    lea rdi, [depth_msg]
    call print_string
    call stack_data_depth
    mov rdi, rax
    call print_integer
    call print_newline
    call test_end_fail
    jmp .done

.value_fail:
    lea rdi, [result_msg]
    call print_string
    push rdi
    call print_integer
    pop rdi
    call print_newline
    call test_end_fail
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_simple_add_execution - Test: 1 2 + = 3
;-----------------------------------------------------------------------------
test_simple_add_execution:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_simple_add]
    call test_start

    call reset_state

    ; Debug: print bytecode location
    lea rdi, [bc_addr_msg]
    call print_string
    lea rdi, [simple_add_bc]
    call print_hex
    call print_newline

    ; Debug: print first byte of bytecode
    lea rdi, [first_byte_msg]
    call print_string
    lea rax, [simple_add_bc]
    movzx rdi, byte [rax]
    call print_integer
    call print_newline

    ; Set up bytecode pointers
    lea rax, [simple_add_bc]
    mov [state + State.bytecode], rax
    lea rax, [simple_add_bc_end]
    mov [state + State.bytecode_end], rax

    ; Debug: verify bytecode pointer in state
    lea rdi, [state_bc_msg]
    call print_string
    mov rdi, [state + State.bytecode]
    call print_hex
    call print_newline

    ; Execute bytecode
    call vm_execute
    mov rbx, rax        ; Save return value

    ; Print error code if any
    lea rdi, [err_msg]
    call print_string
    mov rdi, rbx
    call print_integer
    call print_newline

    ; Check for execution error
    test ebx, ebx
    jnz .fail

    ; Check stack depth is 1
    call stack_data_depth
    mov rbx, rax        ; Save depth

    ; Print depth
    lea rdi, [depth_msg]
    call print_string
    mov rdi, rbx
    call print_integer
    call print_newline

    cmp rbx, 1
    jne .fail

    ; Pop and verify result is 3
    call stack_data_pop
    test rax, rax
    jz .fail
    push rax                ; Save Value pointer on stack

    ; Debug: print the result
    lea rdi, [result_msg]
    call print_string
    mov rax, [rsp]          ; Get Value pointer
    mov rdi, [rax + VALUE_NUM_OFF]
    call print_integer
    call print_newline

    pop rax                 ; Restore Value pointer
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
    pop rbx
    pop rbp
    ret
