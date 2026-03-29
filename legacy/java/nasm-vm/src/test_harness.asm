; test_harness.asm - Test harness for DTRules NASM VM
; Provides test utilities and main entry point

bits 64
default rel

%include "include/constants.inc"
%include "include/state.inc"

;-----------------------------------------------------------------------------
; External symbols
;-----------------------------------------------------------------------------
extern state
extern vm_init

;-----------------------------------------------------------------------------
; Exported symbols
;-----------------------------------------------------------------------------
global _start
global test_start
global test_end_pass
global test_end_fail
global print_test_summary
global reset_state
global test_count
global fail_count
global pass_count
global print_string
global print_integer
global print_newline
global assert_eq
global assert_true
global assert_false

;-----------------------------------------------------------------------------
; System call numbers (Linux x86-64)
;-----------------------------------------------------------------------------
SYS_WRITE   equ 1
SYS_EXIT    equ 60
STDOUT      equ 1

;-----------------------------------------------------------------------------
section .data
;-----------------------------------------------------------------------------

test_count: dq 0
pass_count: dq 0
fail_count: dq 0

current_test: dq 0      ; Pointer to current test name

msg_test_start:     db "Test: ", 0
msg_pass:           db " [PASS]", 10, 0
msg_fail:           db " [FAIL]", 10, 0
msg_summary_pre:    db 10, "Results: ", 0
msg_passed:         db " passed, ", 0
msg_failed:         db " failed of ", 0
msg_total:          db " tests", 10, 0
msg_all_pass:       db "All tests passed!", 10, 0
msg_some_fail:      db "Some tests FAILED!", 10, 0

; Number buffer for printing integers
num_buffer: times 24 db 0

;-----------------------------------------------------------------------------
section .text
;-----------------------------------------------------------------------------

;-----------------------------------------------------------------------------
; _start - Entry point
; Calls test_main and exits with fail count
;-----------------------------------------------------------------------------
_start:
    ; Initialize VM
    call vm_init

    ; Call test_main (defined in test file)
    extern test_main
    call test_main

    ; Exit with fail count
    mov rdi, [fail_count]
    mov rax, SYS_EXIT
    syscall

;-----------------------------------------------------------------------------
; print_string - Print null-terminated string
; Input: rdi = pointer to string
;-----------------------------------------------------------------------------
print_string:
    push rbx
    mov rbx, rdi

    ; Find string length
    xor rcx, rcx
.find_len:
    cmp byte [rbx + rcx], 0
    je .found_len
    inc rcx
    jmp .find_len

.found_len:
    ; sys_write(stdout, string, length)
    mov rax, SYS_WRITE
    mov rdi, STDOUT
    mov rsi, rbx
    mov rdx, rcx
    syscall

    pop rbx
    ret

;-----------------------------------------------------------------------------
; print_integer - Print integer value
; Input: rdi = integer value
;-----------------------------------------------------------------------------
print_integer:
    push rbx
    push r12

    mov rax, rdi
    lea rbx, [num_buffer + 23]
    mov byte [rbx], 0           ; Null terminator

    ; Handle negative numbers
    xor r12d, r12d              ; Sign flag
    test rax, rax
    jns .positive
    neg rax
    mov r12d, 1

.positive:
    ; Handle zero
    test rax, rax
    jnz .convert
    dec rbx
    mov byte [rbx], '0'
    jmp .print

.convert:
    ; Convert to decimal digits (reverse order)
    mov rcx, 10
.digit_loop:
    test rax, rax
    jz .add_sign
    xor edx, edx
    div rcx
    add dl, '0'
    dec rbx
    mov [rbx], dl
    jmp .digit_loop

.add_sign:
    test r12d, r12d
    jz .print
    dec rbx
    mov byte [rbx], '-'

.print:
    mov rdi, rbx
    call print_string

    pop r12
    pop rbx
    ret

;-----------------------------------------------------------------------------
; print_newline - Print newline character
;-----------------------------------------------------------------------------
print_newline:
    push rax
    mov rax, SYS_WRITE
    mov rdi, STDOUT
    lea rsi, [msg_pass + 7]     ; Just the newline
    mov rdx, 1
    syscall
    pop rax
    ret

;-----------------------------------------------------------------------------
; test_start - Start a test
; Input: rdi = pointer to test name
;-----------------------------------------------------------------------------
test_start:
    mov [current_test], rdi
    inc qword [test_count]

    push rdi
    lea rdi, [msg_test_start]
    call print_string
    pop rdi
    call print_string
    ret

;-----------------------------------------------------------------------------
; test_end_pass - Mark current test as passed
;-----------------------------------------------------------------------------
test_end_pass:
    inc qword [pass_count]
    lea rdi, [msg_pass]
    call print_string
    ret

;-----------------------------------------------------------------------------
; test_end_fail - Mark current test as failed
;-----------------------------------------------------------------------------
test_end_fail:
    inc qword [fail_count]
    lea rdi, [msg_fail]
    call print_string
    ret

;-----------------------------------------------------------------------------
; assert_eq - Assert two values are equal
; Input: rdi = actual, rsi = expected
; Output: eax = 1 if equal, 0 if not
;-----------------------------------------------------------------------------
assert_eq:
    cmp rdi, rsi
    sete al
    movzx eax, al
    ret

;-----------------------------------------------------------------------------
; assert_true - Assert value is true (non-zero)
; Input: rdi = value
; Output: eax = 1 if true, 0 if false
;-----------------------------------------------------------------------------
assert_true:
    test rdi, rdi
    setnz al
    movzx eax, al
    ret

;-----------------------------------------------------------------------------
; assert_false - Assert value is false (zero)
; Input: rdi = value
; Output: eax = 1 if false, 0 if true
;-----------------------------------------------------------------------------
assert_false:
    test rdi, rdi
    setz al
    movzx eax, al
    ret

;-----------------------------------------------------------------------------
; reset_state - Reset VM state for next test
;-----------------------------------------------------------------------------
reset_state:
    call vm_init
    ret

;-----------------------------------------------------------------------------
; print_test_summary - Print test results summary
;-----------------------------------------------------------------------------
print_test_summary:
    ; Print summary header
    lea rdi, [msg_summary_pre]
    call print_string

    ; Print pass count
    mov rdi, [pass_count]
    call print_integer

    lea rdi, [msg_passed]
    call print_string

    ; Print fail count
    mov rdi, [fail_count]
    call print_integer

    lea rdi, [msg_failed]
    call print_string

    ; Print total count
    mov rdi, [test_count]
    call print_integer

    lea rdi, [msg_total]
    call print_string

    ; Print pass/fail message
    mov rax, [fail_count]
    test rax, rax
    jnz .some_failed

    lea rdi, [msg_all_pass]
    call print_string
    ret

.some_failed:
    lea rdi, [msg_some_fail]
    call print_string
    ret
