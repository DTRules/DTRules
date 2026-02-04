; benchmark.asm - Performance benchmarks for DTRules assembly implementation
; Tests stack operations, arithmetic, and VM dispatch

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state
extern memory_init
extern stack_data_init
extern stack_data_push_integer
extern stack_data_push_boolean
extern stack_data_pop
extern stack_data_peek
extern stack_data_dup
extern stack_data_swap
extern stack_data_rot
extern stack_data_drop
extern stack_data_clear
extern print_string
extern print_integer
extern print_newline

section .data
    align 8
    ; Benchmark iteration counts
    BENCH_ITERATIONS    equ 10000000    ; 10 million iterations per test

    ; Messages
    msg_header:     db "DTRules Assembly Benchmark Suite", 10
                    db "==================================", 10, 0
    msg_push_pop:   db "Push/Pop integers:      ", 0
    msg_dup:        db "Dup operation:          ", 0
    msg_swap:       db "Swap operation:         ", 0
    msg_rot:        db "Rot operation:          ", 0
    msg_arithmetic: db "Integer add (stack):    ", 0
    msg_ns:         db " ns/op", 10, 0
    msg_ops:        db " ops", 10, 0
    msg_cycles:     db " cycles/op", 10, 0
    msg_mops:       db " M ops/sec", 10, 0
    msg_error:      db "Error during benchmark", 10, 0
    msg_init:       db "Initializing...", 10, 0
    msg_running:    db "Running benchmarks...", 10, 10, 0

section .bss
    align 8
    start_cycles:   resq 1
    end_cycles:     resq 1
    bench_count:    resq 1

section .text

;-----------------------------------------------------------------------------
; get_cycles - Read CPU cycle counter (RDTSC)
; Output: rax = cycle count
;-----------------------------------------------------------------------------
get_cycles:
    rdtsc
    shl rdx, 32
    or rax, rdx
    ret

;-----------------------------------------------------------------------------
; bench_push_pop - Benchmark push/pop operations
;-----------------------------------------------------------------------------
bench_push_pop:
    push rbp
    mov rbp, rsp
    push rbx
    push r13
    push r14

    ; Clear stack
    call stack_data_clear

    ; Start timing
    call get_cycles
    mov r13, rax

    ; Run iterations
    mov r14, BENCH_ITERATIONS
.loop:
    ; Push integer
    mov rdi, r14
    call stack_data_push_integer
    ; Pop it
    call stack_data_pop

    dec r14
    jnz .loop

    ; End timing
    call get_cycles
    sub rax, r13            ; Cycles elapsed

    ; Calculate cycles per operation (2 ops per iteration: push + pop)
    mov rcx, BENCH_ITERATIONS * 2
    xor edx, edx
    div rcx                 ; rax = cycles per op

    pop r14
    pop r13
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; bench_dup - Benchmark dup operation
;-----------------------------------------------------------------------------
bench_dup:
    push rbp
    mov rbp, rsp
    push rbx
    push r13
    push r14

    ; Clear stack and push one value
    call stack_data_clear
    mov rdi, 42
    call stack_data_push_integer

    ; Start timing
    call get_cycles
    mov r13, rax

    ; Run iterations (dup then drop to keep stack size constant)
    mov r14, BENCH_ITERATIONS
.loop:
    call stack_data_dup
    call stack_data_drop

    dec r14
    jnz .loop

    ; End timing
    call get_cycles
    sub rax, r13

    ; Cycles per dup operation
    mov rcx, BENCH_ITERATIONS
    xor edx, edx
    div rcx

    pop r14
    pop r13
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; bench_swap - Benchmark swap operation
;-----------------------------------------------------------------------------
bench_swap:
    push rbp
    mov rbp, rsp
    push rbx
    push r13
    push r14

    ; Clear stack and push two values
    call stack_data_clear
    mov rdi, 1
    call stack_data_push_integer
    mov rdi, 2
    call stack_data_push_integer

    ; Start timing
    call get_cycles
    mov r13, rax

    ; Run iterations
    mov r14, BENCH_ITERATIONS
.loop:
    call stack_data_swap

    dec r14
    jnz .loop

    ; End timing
    call get_cycles
    sub rax, r13

    mov rcx, BENCH_ITERATIONS
    xor edx, edx
    div rcx

    pop r14
    pop r13
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; bench_rot - Benchmark rot operation
;-----------------------------------------------------------------------------
bench_rot:
    push rbp
    mov rbp, rsp
    push rbx
    push r13
    push r14

    ; Clear stack and push three values
    call stack_data_clear
    mov rdi, 1
    call stack_data_push_integer
    mov rdi, 2
    call stack_data_push_integer
    mov rdi, 3
    call stack_data_push_integer

    ; Start timing
    call get_cycles
    mov r13, rax

    ; Run iterations
    mov r14, BENCH_ITERATIONS
.loop:
    call stack_data_rot

    dec r14
    jnz .loop

    ; End timing
    call get_cycles
    sub rax, r13

    mov rcx, BENCH_ITERATIONS
    xor edx, edx
    div rcx

    pop r14
    pop r13
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; bench_arithmetic - Benchmark integer addition via stack
;-----------------------------------------------------------------------------
bench_arithmetic:
    push rbp
    mov rbp, rsp
    push rbx
    push r13
    push r14

    ; Start timing
    call get_cycles
    mov r13, rax

    ; Run iterations
    mov r14, BENCH_ITERATIONS
.loop:
    ; Push two integers
    call stack_data_clear
    mov rdi, 100
    call stack_data_push_integer
    mov rdi, 200
    call stack_data_push_integer

    ; Pop and add manually (simulating op_add hot path)
    call stack_data_pop
    mov rbx, [rax + VALUE_NUM_OFF]  ; Second value
    call stack_data_pop
    add rbx, [rax + VALUE_NUM_OFF]  ; First value + second

    ; Push result
    mov rdi, rbx
    call stack_data_push_integer

    dec r14
    jnz .loop

    ; End timing
    call get_cycles
    sub rax, r13

    mov rcx, BENCH_ITERATIONS
    xor edx, edx
    div rcx

    pop r14
    pop r13
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; print_bench_result - Print benchmark result
; Input: rdi = message pointer, rsi = cycles per op
;-----------------------------------------------------------------------------
print_bench_result:
    push rbp
    mov rbp, rsp
    push rbx

    mov rbx, rsi            ; Save result

    ; Print message
    call print_string

    ; Print cycles
    mov rdi, rbx
    call print_integer

    ; Print units
    lea rdi, [msg_cycles]
    call print_string

    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; _start - Main entry point for benchmark
;-----------------------------------------------------------------------------
global _start
_start:
    ; Print header
    lea rdi, [msg_header]
    call print_string

    ; Initialize
    lea rdi, [msg_init]
    call print_string
    call memory_init
    call stack_data_init

    ; Running message
    lea rdi, [msg_running]
    call print_string

    ; Run benchmarks and print results

    ; Push/Pop
    call bench_push_pop
    mov rsi, rax
    lea rdi, [msg_push_pop]
    call print_bench_result

    ; Dup
    call bench_dup
    mov rsi, rax
    lea rdi, [msg_dup]
    call print_bench_result

    ; Swap
    call bench_swap
    mov rsi, rax
    lea rdi, [msg_swap]
    call print_bench_result

    ; Rot
    call bench_rot
    mov rsi, rax
    lea rdi, [msg_rot]
    call print_bench_result

    ; Arithmetic
    call bench_arithmetic
    mov rsi, rax
    lea rdi, [msg_arithmetic]
    call print_bench_result

    ; Exit
    mov rax, SYS_EXIT
    xor edi, edi
    syscall
