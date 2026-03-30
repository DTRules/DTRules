; test_harness.asm - Common test infrastructure
; DTRules Assembly Unit Tests
; Provides _start, memory initialization, and test utilities

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

; External test main - each test file defines this
extern test_main

; External functions from main project
extern memory_init
extern pools_init
extern heap_init
extern stack_data_init
extern print_string
extern print_integer
extern print_newline

;-----------------------------------------------------------------------------
; Global state
;-----------------------------------------------------------------------------
section .bss
    global state
    state: resb State_size

    ; Memory regions
    alignb 16
    data_stack_mem:   resb DATA_STACK_SIZE * VALUE_SIZE
    entity_stack_mem: resb ENTITY_STACK_SIZE * VALUE_SIZE
    control_stack_mem: resb CONTROL_STACK_SIZE * 64
    heap_mem:         resb HEAP_SIZE

section .data
    test_pass_msg:   db "PASS", 10, 0
    test_fail_msg:   db "FAIL", 10, 0
    test_error_msg:  db "ERROR: ", 0

    global test_count
    global fail_count
    global pass_count
    test_count:      dq 0
    fail_count:      dq 0
    pass_count:      dq 0

section .text

;-----------------------------------------------------------------------------
; _start - Program entry point
;-----------------------------------------------------------------------------
global _start
_start:
    ; Initialize memory subsystem
    call init_state

    ; Run tests
    call test_main

    ; Exit with status based on fail count
    mov rdi, [fail_count]
    mov rax, SYS_EXIT
    syscall

;-----------------------------------------------------------------------------
; init_state - Initialize VM state for testing
;-----------------------------------------------------------------------------
init_state:
    push rbp
    mov rbp, rsp

    ; Initialize data stack
    lea rax, [data_stack_mem + DATA_STACK_SIZE * VALUE_SIZE]
    mov [state + State.data_stack], rax
    mov [state + State.data_stack_base], rax
    mov r12, rax                    ; r12 = data stack pointer
    lea rax, [data_stack_mem]
    mov [state + State.data_stack_end], rax

    ; Initialize entity stack
    lea rax, [entity_stack_mem + ENTITY_STACK_SIZE * VALUE_SIZE]
    mov [state + State.entity_stack], rax
    mov [state + State.entity_stack_base], rax
    mov r13, rax                    ; r13 = entity stack pointer
    lea rax, [entity_stack_mem]
    mov [state + State.entity_stack_end], rax

    ; Initialize control stack
    lea rax, [control_stack_mem + CONTROL_STACK_SIZE * 64]
    mov [state + State.control_stack], rax
    mov [state + State.control_stack_base], rax
    mov r14, rax                    ; r14 = control stack pointer
    lea rax, [control_stack_mem]
    mov [state + State.control_stack_end], rax

    ; Initialize heap
    lea rax, [heap_mem]
    mov [state + State.heap_ptr], rax
    mov [state + State.heap_base], rax
    lea rax, [heap_mem + HEAP_SIZE]
    mov [state + State.heap_end], rax

    ; Initialize memory pools (required for value allocation!)
    call pools_init

    ; Clear error state
    mov dword [state + State.error], ERR_NONE
    mov dword [state + State.halted], 0

    pop rbp
    ret

;-----------------------------------------------------------------------------
; Test helper functions - exported for use by test files
;-----------------------------------------------------------------------------

; test_start - Print test name
; Input: rdi = pointer to test name string
global test_start
test_start:
    push rbp
    mov rbp, rsp
    push rbx

    mov rbx, rdi

    ; Print "Testing "
    lea rdi, [.prefix]
    call print_string

    ; Print test name
    mov rdi, rbx
    call print_string

    ; Print "... "
    lea rdi, [.suffix]
    call print_string

    pop rbx
    pop rbp
    ret

section .rodata
.prefix: db "Testing ", 0
.suffix: db "... ", 0

section .text

; test_pass - Record passed test
global test_pass
test_pass:
    inc qword [test_count]
    inc qword [pass_count]
    ret

; test_fail - Record failed test
global test_fail
test_fail:
    inc qword [test_count]
    inc qword [fail_count]
    ret

; test_end_pass - Print PASS and record it
global test_end_pass
test_end_pass:
    push rbp
    mov rbp, rsp

    lea rdi, [test_pass_msg]
    call print_string
    call test_pass

    pop rbp
    ret

; test_end_fail - Print FAIL and record it
global test_end_fail
test_end_fail:
    push rbp
    mov rbp, rsp

    lea rdi, [test_fail_msg]
    call print_string
    call test_fail

    pop rbp
    ret

; assert_eq - Assert two values are equal
; Input: rdi = actual, rsi = expected
; Returns: 1 if equal, 0 if not
global assert_eq
assert_eq:
    cmp rdi, rsi
    jne .fail
    call test_pass
    mov eax, 1
    ret
.fail:
    call test_fail
    xor eax, eax
    ret

; assert_true - Assert value is non-zero
; Input: rdi = value
; Returns: 1 if true, 0 if not
global assert_true
assert_true:
    test rdi, rdi
    jz .fail
    call test_pass
    mov eax, 1
    ret
.fail:
    call test_fail
    xor eax, eax
    ret

; assert_false - Assert value is zero
; Input: rdi = value
; Returns: 1 if false, 0 if not
global assert_false
assert_false:
    test rdi, rdi
    jnz .fail
    call test_pass
    mov eax, 1
    ret
.fail:
    call test_fail
    xor eax, eax
    ret

; assert_double_eq - Assert two doubles are equal (within epsilon)
; Input: xmm0 = actual, xmm1 = expected
; Returns: 1 if equal, 0 if not
; Uses absolute error tolerance of 1e-10
global assert_double_eq
assert_double_eq:
    push rbp
    mov rbp, rsp

    ; Calculate |actual - expected|
    movsd xmm2, xmm0
    subsd xmm2, xmm1

    ; Get absolute value
    movq rax, xmm2
    mov rdx, 0x7FFFFFFFFFFFFFFF
    and rax, rdx
    movq xmm2, rax

    ; Compare with epsilon (1e-10)
    mov rax, 0x3DDB7CDFD9D7BDBB   ; 1e-10 in IEEE 754
    movq xmm3, rax
    ucomisd xmm2, xmm3

    ja .fail                      ; If diff > epsilon, fail
    call test_pass
    mov eax, 1
    pop rbp
    ret

.fail:
    call test_fail
    xor eax, eax
    pop rbp
    ret

; print_test_summary - Print final test results
global print_test_summary
print_test_summary:
    push rbp
    mov rbp, rsp

    call print_newline

    ; Print passed count
    lea rdi, [.passed_msg]
    call print_string
    mov rdi, [pass_count]
    call print_integer

    ; Print failed count
    lea rdi, [.failed_msg]
    call print_string
    mov rdi, [fail_count]
    call print_integer

    ; Print total count
    lea rdi, [.total_msg]
    call print_string
    mov rdi, [test_count]
    call print_integer
    call print_newline

    pop rbp
    ret

section .rodata
.passed_msg: db " passed, ", 0
.failed_msg: db " failed, ", 0
.total_msg: db " total", 10, 0

section .text

; reset_state - Reset state for next test
global reset_state
reset_state:
    ; Reset data stack
    mov rax, [state + State.data_stack_base]
    mov [state + State.data_stack], rax
    mov r12, rax

    ; Reset entity stack
    mov rax, [state + State.entity_stack_base]
    mov [state + State.entity_stack], rax
    mov r13, rax

    ; Reset control stack
    mov rax, [state + State.control_stack_base]
    mov [state + State.control_stack], rax
    mov r14, rax

    ; Clear error
    mov dword [state + State.error], ERR_NONE

    ret

; write_stdout and write_stderr are provided by file.o
