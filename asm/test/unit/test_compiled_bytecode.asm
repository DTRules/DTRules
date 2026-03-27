; test_compiled_bytecode.asm - Test bytecode compiled by Go
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
extern stack_data_push
extern stack_data_push_integer
extern stack_data_push_boolean
extern stack_data_depth

; VM execution
extern vm_execute

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
    test_header:        db "=== Compiled Bytecode Tests ===", 10, 0
    test_add_name:      db "go_simple_add", 0
    test_beq_name:      db "go_boolean_eq", 0
    test_compare_name:  db "go_comparison", 0
    test_isnull_name:   db "go_isnull", 0
    test_notnull_name:  db "go_notnull", 0

    ; Go-compiled bytecodes (from chip_compile.go output)

    ; simple_add = '1 2 +' => 104 2 2 20
    simple_add_bc:
        db 104, 2, 2, 20
    simple_add_bc_end:

    ; boolean_eq = 'true true beq' => 100 100 200 34
    boolean_eq_bc:
        db 100, 100, 200, 34
    boolean_eq_bc_end:

    ; comparison = '10 5 >' => 2 10 2 5 34
    comparison_bc:
        db 2, 10, 2, 5, 34
    comparison_bc_end:

    ; isnull = 'null isnull' => 102 200 35
    isnull_bc:
        db 102, 200, 35
    isnull_bc_end:

    ; notnull = '42 notnull' => 2 42 200 36
    notnull_bc:
        db 2, 42, 200, 36
    notnull_bc_end:

section .text
    global test_main

;-----------------------------------------------------------------------------
; test_main - Run compiled bytecode tests
;-----------------------------------------------------------------------------
test_main:
    push rbp
    mov rbp, rsp

    ; Print header
    lea rdi, [test_header]
    call print_string

    ; Run tests
    call test_simple_add
    call test_boolean_eq
    call test_comparison
    call test_isnull
    call test_notnull

    ; Print summary
    call print_test_summary

    ; Return fail count as exit status
    mov rax, [fail_count]

    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_simple_add - Test: 1 2 + = 3
;-----------------------------------------------------------------------------
test_simple_add:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_add_name]
    call test_start

    call reset_state

    ; Set up bytecode
    lea rax, [simple_add_bc]
    mov [state + State.bytecode], rax
    lea rax, [simple_add_bc_end]
    mov [state + State.bytecode_end], rax

    ; Execute
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
; test_boolean_eq - Test: true true beq = true
;-----------------------------------------------------------------------------
test_boolean_eq:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_beq_name]
    call test_start

    call reset_state

    ; Set up bytecode
    lea rax, [boolean_eq_bc]
    mov [state + State.bytecode], rax
    lea rax, [boolean_eq_bc_end]
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

    ; Check it's boolean true
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
; test_comparison - Test: 10 5 > = true
;-----------------------------------------------------------------------------
test_comparison:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_compare_name]
    call test_start

    call reset_state

    ; Set up bytecode
    lea rax, [comparison_bc]
    mov [state + State.bytecode], rax
    lea rax, [comparison_bc_end]
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

    ; Pop and verify result is true (10 > 5)
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
; test_isnull - Test: null isnull = true
;-----------------------------------------------------------------------------
test_isnull:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_isnull_name]
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
; test_notnull - Test: 42 notnull = true
;-----------------------------------------------------------------------------
test_notnull:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_notnull_name]
    call test_start

    call reset_state

    ; Set up bytecode
    lea rax, [notnull_bc]
    mov [state + State.bytecode], rax
    lea rax, [notnull_bc_end]
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

    ; Pop and verify result is true (42 is not null)
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
