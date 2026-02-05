; heap.asm - Arena allocator
; DTRules Assembly Implementation
; Simple bump pointer allocation with reset capability

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state

;-----------------------------------------------------------------------------
; Arena allocation statistics (optional tracking)
;-----------------------------------------------------------------------------
section .bss
    align 8
    total_allocated: resq 1
    allocation_count: resq 1

section .text

;-----------------------------------------------------------------------------
; heap_init - Initialize the heap (already done in memory_init)
; Provided for explicit initialization if needed
;-----------------------------------------------------------------------------
global heap_init
heap_init:
    ; Reset allocation pointer to base
    mov rax, [state + State.heap_base]
    mov [state + State.heap_ptr], rax

    ; Clear statistics
    xor eax, eax
    mov [total_allocated], rax
    mov [allocation_count], rax

    ret

;-----------------------------------------------------------------------------
; heap_alloc - Allocate memory from heap
; Input: rdi = size in bytes
; Output: rax = pointer to allocated memory, or 0 if out of memory
; Notes:
;   - Memory is 16-byte aligned
;   - No individual free - use heap_reset to free all
;-----------------------------------------------------------------------------
global heap_alloc
heap_alloc:
    push rbp
    mov rbp, rsp

    ; Align size up to 16 bytes
    add rdi, 15
    and rdi, ~15

    ; Get current pointer
    mov rax, [state + State.heap_ptr]

    ; Calculate new pointer
    lea rcx, [rax + rdi]

    ; Check if we have space
    cmp rcx, [state + State.heap_end]
    ja .out_of_memory

    ; Update heap pointer
    mov [state + State.heap_ptr], rcx

    ; Update statistics
    add [total_allocated], rdi
    inc qword [allocation_count]

    ; Return pointer (already in rax)
    pop rbp
    ret

.out_of_memory:
    ; Set error and return null
    mov dword [state + State.error], ERR_OUT_OF_MEMORY
    xor eax, eax
    pop rbp
    ret

;-----------------------------------------------------------------------------
; heap_alloc_zeroed - Allocate and zero memory
; Input: rdi = size in bytes
; Output: rax = pointer to zeroed memory, or 0 if out of memory
;-----------------------------------------------------------------------------
global heap_alloc_zeroed
heap_alloc_zeroed:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    mov r12, rdi            ; Save size

    ; Allocate
    call heap_alloc
    test rax, rax
    jz .done

    mov rbx, rax            ; Save pointer

    ; Zero the memory
    mov rdi, rax
    xor eax, eax
    mov rcx, r12
    rep stosb

    mov rax, rbx            ; Restore pointer

.done:
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; heap_reset - Reset heap to empty state
; Notes: All previously allocated memory becomes invalid
;-----------------------------------------------------------------------------
global heap_reset
heap_reset:
    ; Reset pointer to base
    mov rax, [state + State.heap_base]
    mov [state + State.heap_ptr], rax

    ; Clear statistics
    xor eax, eax
    mov [total_allocated], rax
    mov [allocation_count], rax

    ret

;-----------------------------------------------------------------------------
; heap_mark - Get current heap position for later restore
; Output: rax = current heap position marker
;-----------------------------------------------------------------------------
global heap_mark
heap_mark:
    mov rax, [state + State.heap_ptr]
    ret

;-----------------------------------------------------------------------------
; heap_restore - Restore heap to a previous position
; Input: rdi = position marker from heap_mark
; Notes: All allocations after the mark become invalid
;-----------------------------------------------------------------------------
global heap_restore
heap_restore:
    ; Validate marker is within heap bounds
    cmp rdi, [state + State.heap_base]
    jb .invalid
    cmp rdi, [state + State.heap_end]
    ja .invalid

    mov [state + State.heap_ptr], rdi
    ret

.invalid:
    ; Invalid marker - don't change anything
    ret

;-----------------------------------------------------------------------------
; heap_available - Get available heap space
; Output: rax = bytes available
;-----------------------------------------------------------------------------
global heap_available
heap_available:
    mov rax, [state + State.heap_end]
    sub rax, [state + State.heap_ptr]
    ret

;-----------------------------------------------------------------------------
; heap_used - Get used heap space
; Output: rax = bytes used
;-----------------------------------------------------------------------------
global heap_used
heap_used:
    mov rax, [state + State.heap_ptr]
    sub rax, [state + State.heap_base]
    ret

;-----------------------------------------------------------------------------
; heap_stats - Get allocation statistics
; Output: rax = total bytes allocated, rdx = number of allocations
;-----------------------------------------------------------------------------
global heap_stats
heap_stats:
    mov rax, [total_allocated]
    mov rdx, [allocation_count]
    ret
