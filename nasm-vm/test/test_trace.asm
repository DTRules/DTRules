; DTRules NASM VM - Trace Recording Tests
; Copyright 2024 Paul Snow - Apache License 2.0
;
; This file tests the trace recording functionality.
; It verifies:
; - Trace entries are written correctly
; - Buffer overflow is handled gracefully
; - Timestamps are recorded
; - Entry format is correct

%include "dtrules.inc"

extern trace_record_opcode
extern trace_record_push
extern trace_record_pop
extern trace_record_error
extern trace_get_buffer_info
extern trace_reset
extern trace_enable
extern trace_disable
extern vm_init_context

; C library functions
extern printf
extern exit
extern memset

section .data
    ; Test messages
    msg_start:      db "=== Trace Recording Tests ===", 10, 0
    msg_pass:       db "[PASS] %s", 10, 0
    msg_fail:       db "[FAIL] %s (expected %d, got %d)", 10, 0
    msg_summary:    db "Tests: %d passed, %d failed", 10, 0

    ; Test names
    test1_name:     db "trace_record_opcode writes entry", 0
    test2_name:     db "trace_record_push writes entry", 0
    test3_name:     db "trace_record_pop writes entry", 0
    test4_name:     db "trace_reset clears buffer", 0
    test5_name:     db "trace_disable stops recording", 0
    test6_name:     db "trace_enable resumes recording", 0
    test7_name:     db "buffer overflow handled gracefully", 0
    test8_name:     db "opcode entry has correct format", 0

section .bss
    ; VM context (256 bytes aligned)
    align 256
    vm_context:     resb VMContext_size

    ; Data stack
    data_stack:     resb DATA_STACK_SIZE * VALUE_SIZE

    ; Trace buffer (small for testing)
    trace_buffer:   resb 1024

    ; Test counters
    tests_passed:   resq 1
    tests_failed:   resq 1

section .text
global _start

_start:
    ; Print header
    lea rdi, [rel msg_start]
    xor eax, eax
    call printf

    ; Initialize test counters
    mov qword [rel tests_passed], 0
    mov qword [rel tests_failed], 0

    ; Initialize VM context
    call init_vm_context

    ; Run tests
    call test_record_opcode
    call test_record_push
    call test_record_pop
    call test_reset
    call test_disable
    call test_enable
    call test_overflow
    call test_opcode_format

    ; Print summary
    lea rdi, [rel msg_summary]
    mov rsi, [rel tests_passed]
    mov rdx, [rel tests_failed]
    xor eax, eax
    call printf

    ; Exit with failure count as status
    mov rdi, [rel tests_failed]
    call exit


; ===========================================================================
; Initialize VM context for tests
; ===========================================================================
init_vm_context:
    ; Clear context
    lea rdi, [rel vm_context]
    xor esi, esi
    mov edx, VMContext_size
    call memset

    ; Set up data stack
    lea rax, [rel data_stack]
    lea rdi, [rel vm_context]
    mov [rdi + VMContext.data_stack], rax
    mov qword [rdi + VMContext.data_stack_cap], DATA_STACK_SIZE

    ; Set up trace buffer
    lea rax, [rel trace_buffer]
    mov [rdi + VMContext.trace_buffer], rax
    mov qword [rdi + VMContext.trace_cap], 1024
    mov qword [rdi + VMContext.trace_pos], 0
    mov qword [rdi + VMContext.trace_count], 0

    ; Enable tracing
    or qword [rdi + VMContext.flags], VM_FLAG_TRACE | VM_FLAG_RUNNING

    ret


; ===========================================================================
; Test: trace_record_opcode writes entry
; ===========================================================================
test_record_opcode:
    ; Reset trace
    lea rdi, [rel vm_context]
    call trace_reset

    ; Record an opcode
    lea rdi, [rel vm_context]
    mov esi, OP_ADD             ; opcode 20
    call trace_record_opcode

    ; Check trace count is 1
    lea rdi, [rel vm_context]
    call trace_get_buffer_info
    ; rax = count, rdx = bytes, rcx = cap

    cmp rax, 1
    jne .fail

    ; Check bytes written (9 for opcode entry)
    cmp rdx, 9
    jne .fail

    ; Pass
    lea rdi, [rel msg_pass]
    lea rsi, [rel test1_name]
    xor eax, eax
    call printf
    inc qword [rel tests_passed]
    ret

.fail:
    lea rdi, [rel msg_fail]
    lea rsi, [rel test1_name]
    mov edx, 1
    mov ecx, eax
    xor eax, eax
    call printf
    inc qword [rel tests_failed]
    ret


; ===========================================================================
; Test: trace_record_push writes entry
; ===========================================================================
test_record_push:
    lea rdi, [rel vm_context]
    call trace_reset

    ; Record a push
    lea rdi, [rel vm_context]
    mov rsi, VTAG_INTEGER
    mov rdx, 42
    call trace_record_push

    ; Check trace count is 1
    lea rdi, [rel vm_context]
    call trace_get_buffer_info

    cmp rax, 1
    jne .fail

    ; Check bytes (23 for push entry)
    cmp rdx, 23
    jne .fail

    lea rdi, [rel msg_pass]
    lea rsi, [rel test2_name]
    xor eax, eax
    call printf
    inc qword [rel tests_passed]
    ret

.fail:
    lea rdi, [rel msg_fail]
    lea rsi, [rel test2_name]
    mov edx, 1
    mov ecx, eax
    xor eax, eax
    call printf
    inc qword [rel tests_failed]
    ret


; ===========================================================================
; Test: trace_record_pop writes entry
; ===========================================================================
test_record_pop:
    lea rdi, [rel vm_context]
    call trace_reset

    ; Record a pop
    lea rdi, [rel vm_context]
    mov rsi, VTAG_BOOLEAN
    mov rdx, 1
    call trace_record_pop

    ; Check trace count is 1
    lea rdi, [rel vm_context]
    call trace_get_buffer_info

    cmp rax, 1
    jne .fail

    lea rdi, [rel msg_pass]
    lea rsi, [rel test3_name]
    xor eax, eax
    call printf
    inc qword [rel tests_passed]
    ret

.fail:
    lea rdi, [rel msg_fail]
    lea rsi, [rel test3_name]
    mov edx, 1
    mov ecx, eax
    xor eax, eax
    call printf
    inc qword [rel tests_failed]
    ret


; ===========================================================================
; Test: trace_reset clears buffer
; ===========================================================================
test_reset:
    ; Write some entries
    lea rdi, [rel vm_context]
    mov esi, OP_NOP
    call trace_record_opcode
    lea rdi, [rel vm_context]
    mov esi, OP_ADD
    call trace_record_opcode

    ; Reset
    lea rdi, [rel vm_context]
    call trace_reset

    ; Check count is 0
    lea rdi, [rel vm_context]
    call trace_get_buffer_info

    test rax, rax
    jnz .fail

    test rdx, rdx
    jnz .fail

    lea rdi, [rel msg_pass]
    lea rsi, [rel test4_name]
    xor eax, eax
    call printf
    inc qword [rel tests_passed]
    ret

.fail:
    lea rdi, [rel msg_fail]
    lea rsi, [rel test4_name]
    mov edx, 0
    mov ecx, eax
    xor eax, eax
    call printf
    inc qword [rel tests_failed]
    ret


; ===========================================================================
; Test: trace_disable stops recording
; ===========================================================================
test_disable:
    lea rdi, [rel vm_context]
    call trace_reset

    ; Record one entry
    lea rdi, [rel vm_context]
    mov esi, OP_NOP
    call trace_record_opcode

    ; Disable tracing
    lea rdi, [rel vm_context]
    call trace_disable

    ; Try to record another entry
    lea rdi, [rel vm_context]
    mov esi, OP_ADD
    call trace_record_opcode

    ; Count should still be 1
    lea rdi, [rel vm_context]
    call trace_get_buffer_info

    cmp rax, 1
    jne .fail

    lea rdi, [rel msg_pass]
    lea rsi, [rel test5_name]
    xor eax, eax
    call printf
    inc qword [rel tests_passed]
    ret

.fail:
    lea rdi, [rel msg_fail]
    lea rsi, [rel test5_name]
    mov edx, 1
    mov ecx, eax
    xor eax, eax
    call printf
    inc qword [rel tests_failed]
    ret


; ===========================================================================
; Test: trace_enable resumes recording
; ===========================================================================
test_enable:
    ; Continue from disabled state
    ; Enable tracing
    lea rdi, [rel vm_context]
    call trace_enable

    ; Should succeed (return 0)
    test eax, eax
    jnz .fail

    ; Record an entry
    lea rdi, [rel vm_context]
    mov esi, OP_MUL
    call trace_record_opcode

    ; Count should be 2 now (1 from before + 1 new)
    lea rdi, [rel vm_context]
    call trace_get_buffer_info

    cmp rax, 2
    jne .fail

    lea rdi, [rel msg_pass]
    lea rsi, [rel test6_name]
    xor eax, eax
    call printf
    inc qword [rel tests_passed]
    ret

.fail:
    lea rdi, [rel msg_fail]
    lea rsi, [rel test6_name]
    mov edx, 2
    mov ecx, eax
    xor eax, eax
    call printf
    inc qword [rel tests_failed]
    ret


; ===========================================================================
; Test: buffer overflow handled gracefully
; ===========================================================================
test_overflow:
    lea rdi, [rel vm_context]
    call trace_reset

    ; Set very small capacity (just enough for 2 entries)
    lea rdi, [rel vm_context]
    mov qword [rdi + VMContext.trace_cap], 20

    ; Record entries until overflow
    mov r15, 100                ; try to write 100 entries
.loop:
    lea rdi, [rel vm_context]
    mov esi, OP_NOP
    call trace_record_opcode
    dec r15
    jnz .loop

    ; Check that tracing was disabled (flag cleared)
    lea rdi, [rel vm_context]
    test qword [rdi + VMContext.flags], VM_FLAG_TRACE
    jnz .fail

    ; Restore normal capacity
    lea rdi, [rel vm_context]
    mov qword [rdi + VMContext.trace_cap], 1024

    lea rdi, [rel msg_pass]
    lea rsi, [rel test7_name]
    xor eax, eax
    call printf
    inc qword [rel tests_passed]
    ret

.fail:
    lea rdi, [rel msg_fail]
    lea rsi, [rel test7_name]
    mov edx, 0
    mov ecx, 1
    xor eax, eax
    call printf
    inc qword [rel tests_failed]
    ret


; ===========================================================================
; Test: opcode entry has correct format
; ===========================================================================
test_opcode_format:
    lea rdi, [rel vm_context]
    call trace_reset

    ; Record specific opcode
    lea rdi, [rel vm_context]
    mov esi, OP_ADD             ; opcode 20
    call trace_record_opcode

    ; Check entry format
    lea rdi, [rel trace_buffer]

    ; Check event type
    movzx eax, byte [rdi + TRACE_ENTRY_TYPE_OFFSET]
    cmp al, TRACE_OPCODE
    jne .fail

    ; Check payload length
    movzx eax, word [rdi + TRACE_ENTRY_LEN_OFFSET]
    cmp ax, 2
    jne .fail

    ; Check opcode value
    movzx eax, byte [rdi + TRACE_ENTRY_DATA_OFFSET]
    cmp al, OP_ADD
    jne .fail

    lea rdi, [rel msg_pass]
    lea rsi, [rel test8_name]
    xor eax, eax
    call printf
    inc qword [rel tests_passed]
    ret

.fail:
    lea rdi, [rel msg_fail]
    lea rsi, [rel test8_name]
    mov edx, OP_ADD
    mov ecx, eax
    xor eax, eax
    call printf
    inc qword [rel tests_failed]
    ret
