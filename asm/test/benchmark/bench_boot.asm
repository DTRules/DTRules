; bench_boot.asm - Minimal boot stub for benchmarks
; Provides state, memory_init, and sys_exit without _start

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern heap_alloc

;-----------------------------------------------------------------------------
; Global state
;-----------------------------------------------------------------------------
section .bss
    align 16
    global state
    state: resb State_size

    global temp_value
    temp_value: resb VALUE_SIZE

    global temp_value2
    temp_value2: resb VALUE_SIZE

section .text

;-----------------------------------------------------------------------------
; memory_init - Initialize memory via mmap
; Returns: 0 on success, -1 on failure
;-----------------------------------------------------------------------------
global memory_init
memory_init:
    push rbp
    mov rbp, rsp
    push rbx

    ; Allocate data stack
    mov rax, SYS_MMAP
    xor edi, edi
    mov esi, DATA_STACK_SIZE * VALUE_SIZE
    mov edx, PROT_READ | PROT_WRITE
    mov r10d, MAP_PRIVATE | MAP_ANONYMOUS
    mov r8d, -1
    xor r9d, r9d
    syscall

    cmp rax, -1
    je .fail

    mov rbx, rax
    add rbx, DATA_STACK_SIZE * VALUE_SIZE
    mov [state + State.data_stack_base], rbx
    mov [state + State.data_stack], rbx
    mov [state + State.data_stack_end], rax
    mov r12, rbx

    ; Allocate entity stack
    mov rax, SYS_MMAP
    xor edi, edi
    mov esi, ENTITY_STACK_SIZE * 8
    mov edx, PROT_READ | PROT_WRITE
    mov r10d, MAP_PRIVATE | MAP_ANONYMOUS
    mov r8d, -1
    xor r9d, r9d
    syscall

    cmp rax, -1
    je .fail

    mov rbx, rax
    add rbx, ENTITY_STACK_SIZE * 8
    mov [state + State.entity_stack_base], rbx
    mov [state + State.entity_stack], rbx
    mov [state + State.entity_stack_end], rax
    mov r13, rbx

    ; Allocate control stack
    mov rax, SYS_MMAP
    xor edi, edi
    mov esi, CONTROL_STACK_SIZE * 32
    mov edx, PROT_READ | PROT_WRITE
    mov r10d, MAP_PRIVATE | MAP_ANONYMOUS
    mov r8d, -1
    xor r9d, r9d
    syscall

    cmp rax, -1
    je .fail

    mov rbx, rax
    add rbx, CONTROL_STACK_SIZE * 32
    mov [state + State.control_stack_base], rbx
    mov [state + State.control_stack], rbx
    mov [state + State.control_stack_end], rax
    mov r14, rbx

    ; Allocate heap
    mov rax, SYS_MMAP
    xor edi, edi
    mov esi, HEAP_SIZE
    mov edx, PROT_READ | PROT_WRITE
    mov r10d, MAP_PRIVATE | MAP_ANONYMOUS
    mov r8d, -1
    xor r9d, r9d
    syscall

    cmp rax, -1
    je .fail

    mov [state + State.heap_base], rax
    mov [state + State.heap_ptr], rax
    add rax, HEAP_SIZE
    mov [state + State.heap_end], rax

    ; Clear error state
    mov dword [state + State.error], ERR_NONE
    mov dword [state + State.halted], 0

    ; Store state pointer in r15
    lea r15, [state]

    xor eax, eax
    jmp .done

.fail:
    mov eax, -1

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; sys_exit - Exit with code
; Input: edi = exit code
;-----------------------------------------------------------------------------
global sys_exit
sys_exit:
    mov eax, SYS_EXIT_GROUP
    syscall
