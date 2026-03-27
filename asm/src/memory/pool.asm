; pool.asm - Fixed-size object pools
; DTRules Assembly Implementation
; Free-list based allocation for frequently allocated sizes

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"

extern state
extern heap_alloc

;-----------------------------------------------------------------------------
; Pool structure
;-----------------------------------------------------------------------------
struc Pool
    .free_list:     resq 1      ; Head of free list
    .block_list:    resq 1      ; List of allocated blocks
    .object_size:   resq 1      ; Size of each object
    .objects_per_block: resq 1  ; Objects per allocation block
    .allocated:     resq 1      ; Total objects allocated
    .in_use:        resq 1      ; Objects currently in use
endstruc

;-----------------------------------------------------------------------------
; Pre-defined pools
;-----------------------------------------------------------------------------
section .bss
    align 16
    ; Pool for 24-byte Values
    global value_pool
    value_pool: resb Pool_size

    ; Pool for 32-byte Names
    global name_pool
    name_pool: resb Pool_size

    ; Pool for 64-byte small arrays
    global small_array_pool
    small_array_pool: resb Pool_size

section .text

;-----------------------------------------------------------------------------
; pool_init - Initialize a pool
; Input:
;   rdi = pointer to Pool structure
;   rsi = object size (will be rounded up to 16 bytes, minimum 16)
;   rdx = objects per block
;-----------------------------------------------------------------------------
global pool_init
pool_init:
    push rbp
    mov rbp, rsp

    ; Ensure minimum object size of 16 (to hold free list pointer)
    cmp rsi, 16
    jge .size_ok
    mov rsi, 16
.size_ok:
    ; Align size to 16 bytes
    add rsi, 15
    and rsi, ~15

    ; Initialize pool fields
    mov qword [rdi + Pool.free_list], 0
    mov qword [rdi + Pool.block_list], 0
    mov [rdi + Pool.object_size], rsi
    mov [rdi + Pool.objects_per_block], rdx
    mov qword [rdi + Pool.allocated], 0
    mov qword [rdi + Pool.in_use], 0

    pop rbp
    ret

;-----------------------------------------------------------------------------
; pools_init - Initialize all pre-defined pools
;-----------------------------------------------------------------------------
global pools_init
pools_init:
    push rbp
    mov rbp, rsp

    ; Initialize value pool (24-byte objects)
    lea rdi, [value_pool]
    mov rsi, VALUE_SIZE
    mov rdx, 256            ; 256 values per block
    call pool_init

    ; Initialize name pool (32-byte objects)
    lea rdi, [name_pool]
    mov rsi, 32
    mov rdx, 256
    call pool_init

    ; Initialize small array pool (64-byte objects)
    lea rdi, [small_array_pool]
    mov rsi, 64
    mov rdx, 128
    call pool_init

    pop rbp
    ret

;-----------------------------------------------------------------------------
; pool_alloc - Allocate object from pool
; Input: rdi = pointer to Pool
; Output: rax = pointer to object, or 0 if out of memory
;-----------------------------------------------------------------------------
global pool_alloc
pool_alloc:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    mov rbx, rdi            ; Save pool pointer

    ; Check free list
    mov rax, [rbx + Pool.free_list]
    test rax, rax
    jz .need_block

    ; Pop from free list
    mov rcx, [rax]          ; Next free object
    mov [rbx + Pool.free_list], rcx
    inc qword [rbx + Pool.in_use]
    jmp .done

.need_block:
    ; Allocate new block
    mov rax, [rbx + Pool.object_size]
    imul rax, [rbx + Pool.objects_per_block]
    add rax, 8              ; Space for block list pointer

    mov rdi, rax
    call heap_alloc
    test rax, rax
    jz .done                ; Out of memory

    mov r12, rax            ; Save block pointer

    ; Link into block list
    mov rcx, [rbx + Pool.block_list]
    mov [rax], rcx          ; block->next = old head
    mov [rbx + Pool.block_list], rax

    ; Build free list from new block
    add rax, 8              ; Skip block header
    mov rcx, [rbx + Pool.objects_per_block]
    mov rdx, [rbx + Pool.object_size]

    ; First object will be returned, rest go to free list
    push rax                ; Save first object
    dec rcx                 ; One less for free list

    test rcx, rcx
    jz .no_free_list

    add rax, rdx            ; Move to second object
.build_list:
    lea rdi, [rax + rdx]    ; Next object
    cmp rcx, 1
    je .last_in_list
    mov [rax], rdi          ; object->next = next object
    mov rax, rdi
    dec rcx
    jmp .build_list

.last_in_list:
    mov qword [rax], 0      ; Last object->next = NULL

    ; Link new free list
    pop rax                 ; Restore first object
    push rax
    add rax, rdx            ; Point to second object (head of new free list)
    mov rcx, [rbx + Pool.free_list]
    ; Find end of new free list to link old
    ; Actually, we built it pointing to null, and old free list was empty
    mov [rbx + Pool.free_list], rax

.no_free_list:
    pop rax                 ; Return first object

    ; Update stats
    mov rcx, [rbx + Pool.objects_per_block]
    add [rbx + Pool.allocated], rcx
    inc qword [rbx + Pool.in_use]

.done:
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; pool_free - Return object to pool
; Input:
;   rdi = pointer to Pool
;   rsi = pointer to object
;-----------------------------------------------------------------------------
global pool_free
pool_free:
    ; Push onto free list
    mov rax, [rdi + Pool.free_list]
    mov [rsi], rax          ; object->next = old head
    mov [rdi + Pool.free_list], rsi
    dec qword [rdi + Pool.in_use]
    ret

;-----------------------------------------------------------------------------
; pool_reset - Reset pool (free all objects but keep blocks)
; Input: rdi = pointer to Pool
;-----------------------------------------------------------------------------
global pool_reset
pool_reset:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; Save pool pointer

    ; Clear free list
    mov qword [rbx + Pool.free_list], 0
    mov qword [rbx + Pool.in_use], 0

    ; Rebuild free list from all blocks
    mov r12, [rbx + Pool.block_list]
    test r12, r12
    jz .done

.next_block:
    ; Process this block
    lea rax, [r12 + 8]      ; Skip block header
    mov rcx, [rbx + Pool.objects_per_block]
    mov rdx, [rbx + Pool.object_size]

.add_objects:
    ; Add object to free list
    mov rdi, [rbx + Pool.free_list]
    mov [rax], rdi
    mov [rbx + Pool.free_list], rax
    add rax, rdx
    dec rcx
    jnz .add_objects

    ; Move to next block
    mov r12, [r12]          ; block->next
    test r12, r12
    jnz .next_block

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; pool_stats - Get pool statistics
; Input: rdi = pointer to Pool
; Output: rax = allocated count, rdx = in-use count
;-----------------------------------------------------------------------------
global pool_stats
pool_stats:
    mov rax, [rdi + Pool.allocated]
    mov rdx, [rdi + Pool.in_use]
    ret

;-----------------------------------------------------------------------------
; Convenience functions for pre-defined pools
;-----------------------------------------------------------------------------

global value_pool_alloc
value_pool_alloc:
    lea rdi, [value_pool]
    jmp pool_alloc

global value_pool_free
value_pool_free:
    lea rdi, [value_pool]
    mov rsi, rdi
    mov rdi, value_pool     ; Fix: load address properly
    lea rdi, [value_pool]
    jmp pool_free

global name_pool_alloc
name_pool_alloc:
    lea rdi, [name_pool]
    jmp pool_alloc

global name_pool_free
name_pool_free:
    mov rsi, rdi            ; Object pointer from first arg
    lea rdi, [name_pool]
    jmp pool_free
