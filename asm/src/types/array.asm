; array.asm - Dynamic array operations
; DTRules Assembly Implementation
; Arrays of 24-byte Values with automatic growth

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state
extern heap_alloc
extern value_copy

;-----------------------------------------------------------------------------
; Array header structure (stored at VALUE_PTR_OFF)
; Offset 0:  length (8 bytes)
; Offset 8:  capacity (8 bytes)
; Offset 16: data pointer (8 bytes)
;-----------------------------------------------------------------------------

section .text

;-----------------------------------------------------------------------------
; array_alloc - Allocate a new array
; Input: rdi = initial capacity (0 for default)
; Output: rax = pointer to array header
;-----------------------------------------------------------------------------
global array_alloc
array_alloc:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    ; Use default capacity if 0
    test rdi, rdi
    jnz .have_cap
    mov rdi, ARRAY_INITIAL_CAP
.have_cap:
    mov r12, rdi            ; Save capacity

    ; Allocate header (24 bytes)
    mov rdi, 24
    call heap_alloc
    test rax, rax
    jz .done

    mov rbx, rax            ; Save header pointer

    ; Initialize header
    mov qword [rbx + ARRAY_LEN_OFF], 0
    mov [rbx + ARRAY_CAP_OFF], r12

    ; Allocate data array
    imul rdi, r12, VALUE_SIZE
    call heap_alloc
    test rax, rax
    jz .fail

    mov [rbx + ARRAY_DATA_OFF], rax

    mov rax, rbx
    jmp .done

.fail:
    xor eax, eax

.done:
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; array_length - Get array length
; Input: rdi = array header pointer
; Output: rax = length
;-----------------------------------------------------------------------------
global array_length
array_length:
    mov rax, [rdi + ARRAY_LEN_OFF]
    ret

;-----------------------------------------------------------------------------
; array_capacity - Get array capacity
; Input: rdi = array header pointer
; Output: rax = capacity
;-----------------------------------------------------------------------------
global array_capacity
array_capacity:
    mov rax, [rdi + ARRAY_CAP_OFF]
    ret

;-----------------------------------------------------------------------------
; array_get - Get element at index
; Input: rdi = array header, rsi = index
; Output: rax = pointer to Value at index, or 0 if out of bounds
;-----------------------------------------------------------------------------
global array_get
array_get:
    ; Check bounds
    cmp rsi, [rdi + ARRAY_LEN_OFF]
    jae .out_of_bounds

    ; Calculate offset
    imul rax, rsi, VALUE_SIZE
    add rax, [rdi + ARRAY_DATA_OFF]
    ret

.out_of_bounds:
    mov dword [state + State.error], ERR_INDEX_BOUNDS
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; array_set - Set element at index
; Input: rdi = array header, rsi = index, rdx = pointer to Value
; Output: rax = 0 on success, non-zero on error
;-----------------------------------------------------------------------------
global array_set
array_set:
    push rbp
    mov rbp, rsp
    push rbx

    ; Check bounds
    cmp rsi, [rdi + ARRAY_LEN_OFF]
    jae .out_of_bounds

    ; Calculate destination
    mov rbx, rdx            ; Save source value
    imul rax, rsi, VALUE_SIZE
    add rax, [rdi + ARRAY_DATA_OFF]

    ; Copy value
    mov rdi, rax
    mov rsi, rbx
    mov ecx, VALUE_SIZE
    rep movsb

    xor eax, eax
    jmp .done

.out_of_bounds:
    mov dword [state + State.error], ERR_INDEX_BOUNDS
    mov eax, ERR_INDEX_BOUNDS

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; array_grow - Grow array capacity
; Input: rdi = array header
; Output: rax = 0 on success, non-zero on error
;-----------------------------------------------------------------------------
global array_grow
array_grow:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; Array header

    ; Calculate new capacity
    mov r12, [rbx + ARRAY_CAP_OFF]
    shl r12, 1              ; Double capacity
    test r12, r12
    jnz .have_cap
    mov r12, ARRAY_INITIAL_CAP
.have_cap:

    ; Allocate new data array
    imul rdi, r12, VALUE_SIZE
    call heap_alloc
    test rax, rax
    jz .fail

    mov r13, rax            ; New data pointer

    ; Copy existing elements
    mov rdi, r13
    mov rsi, [rbx + ARRAY_DATA_OFF]
    mov rcx, [rbx + ARRAY_LEN_OFF]
    imul rcx, VALUE_SIZE
    rep movsb

    ; Update header
    mov [rbx + ARRAY_CAP_OFF], r12
    mov [rbx + ARRAY_DATA_OFF], r13

    xor eax, eax
    jmp .done

.fail:
    mov eax, ERR_OUT_OF_MEMORY

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; array_push - Append element to end
; Input: rdi = array header, rsi = pointer to Value
; Output: rax = 0 on success
;-----------------------------------------------------------------------------
global array_push
array_push:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    mov rbx, rdi            ; Array header
    mov r12, rsi            ; Value to push

    ; Check if we need to grow
    mov rax, [rbx + ARRAY_LEN_OFF]
    cmp rax, [rbx + ARRAY_CAP_OFF]
    jb .have_space

    mov rdi, rbx
    call array_grow
    test rax, rax
    jnz .done

.have_space:
    ; Calculate destination
    mov rax, [rbx + ARRAY_LEN_OFF]
    imul rax, VALUE_SIZE
    add rax, [rbx + ARRAY_DATA_OFF]

    ; Copy value
    mov rdi, rax
    mov rsi, r12
    mov ecx, VALUE_SIZE
    rep movsb

    ; Increment length
    inc qword [rbx + ARRAY_LEN_OFF]

    xor eax, eax

.done:
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; array_pop - Remove and return last element
; Input: rdi = array header
; Output: rax = pointer to popped Value, or 0 if empty
;-----------------------------------------------------------------------------
global array_pop
array_pop:
    ; Check if empty
    mov rax, [rdi + ARRAY_LEN_OFF]
    test rax, rax
    jz .empty

    ; Decrement length
    dec rax
    mov [rdi + ARRAY_LEN_OFF], rax

    ; Return pointer to element
    imul rax, VALUE_SIZE
    add rax, [rdi + ARRAY_DATA_OFF]
    ret

.empty:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; array_insert - Insert element at index
; Input: rdi = array header, rsi = index, rdx = pointer to Value
; Output: rax = 0 on success
;-----------------------------------------------------------------------------
global array_insert
array_insert:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14

    mov rbx, rdi            ; Array header
    mov r12, rsi            ; Index
    mov r13, rdx            ; Value

    ; Validate index (can insert at end)
    mov rax, [rbx + ARRAY_LEN_OFF]
    cmp r12, rax
    ja .out_of_bounds

    ; Check capacity
    cmp rax, [rbx + ARRAY_CAP_OFF]
    jb .have_space

    mov rdi, rbx
    call array_grow
    test rax, rax
    jnz .done

.have_space:
    ; Calculate number of elements to shift
    mov r14, [rbx + ARRAY_LEN_OFF]
    sub r14, r12            ; Elements to move

    ; Move elements if needed
    test r14, r14
    jz .insert

    ; Calculate source and destination
    mov rax, [rbx + ARRAY_DATA_OFF]
    imul rcx, r12, VALUE_SIZE
    lea rsi, [rax + rcx]    ; Source: index position
    lea rdi, [rsi + VALUE_SIZE]  ; Dest: index + 1

    ; Move backwards to avoid overlap issues
    imul rcx, r14, VALUE_SIZE
    add rsi, rcx
    add rdi, rcx
    sub rsi, VALUE_SIZE
    sub rdi, VALUE_SIZE

    std                     ; Reverse direction
    rep movsb
    cld

.insert:
    ; Insert new element
    mov rax, [rbx + ARRAY_DATA_OFF]
    imul rcx, r12, VALUE_SIZE
    lea rdi, [rax + rcx]
    mov rsi, r13
    mov ecx, VALUE_SIZE
    rep movsb

    ; Increment length
    inc qword [rbx + ARRAY_LEN_OFF]

    xor eax, eax
    jmp .done

.out_of_bounds:
    mov dword [state + State.error], ERR_INDEX_BOUNDS
    mov eax, ERR_INDEX_BOUNDS

.done:
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; array_remove - Remove element at index
; Input: rdi = array header, rsi = index
; Output: rax = 0 on success
;-----------------------------------------------------------------------------
global array_remove
array_remove:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    mov rbx, rdi            ; Array header
    mov r12, rsi            ; Index

    ; Validate index
    mov rax, [rbx + ARRAY_LEN_OFF]
    cmp r12, rax
    jae .out_of_bounds

    ; Calculate elements to shift
    dec rax
    sub rax, r12            ; Elements after removed one

    test rax, rax
    jz .no_shift

    ; Shift elements left
    mov rcx, [rbx + ARRAY_DATA_OFF]
    imul rdx, r12, VALUE_SIZE
    lea rdi, [rcx + rdx]    ; Destination: removed position
    lea rsi, [rdi + VALUE_SIZE]  ; Source: next position
    imul rcx, rax, VALUE_SIZE
    rep movsb

.no_shift:
    ; Decrement length
    dec qword [rbx + ARRAY_LEN_OFF]

    xor eax, eax
    jmp .done

.out_of_bounds:
    mov dword [state + State.error], ERR_INDEX_BOUNDS
    mov eax, ERR_INDEX_BOUNDS

.done:
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; array_clear - Clear all elements
; Input: rdi = array header
;-----------------------------------------------------------------------------
global array_clear
array_clear:
    mov qword [rdi + ARRAY_LEN_OFF], 0
    ret

;-----------------------------------------------------------------------------
; array_concat - Concatenate two arrays
; Input: rdi = array 1, rsi = array 2
; Output: rax = new array, or 0 on error
;-----------------------------------------------------------------------------
global array_concat
array_concat:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; Array 1
    mov r12, rsi            ; Array 2

    ; Calculate total length
    mov r13, [rbx + ARRAY_LEN_OFF]
    add r13, [r12 + ARRAY_LEN_OFF]

    ; Allocate new array
    mov rdi, r13
    call array_alloc
    test rax, rax
    jz .done

    push rax                ; Save new array

    ; Set length
    mov [rax + ARRAY_LEN_OFF], r13

    ; Copy first array
    mov rdi, [rax + ARRAY_DATA_OFF]
    mov rsi, [rbx + ARRAY_DATA_OFF]
    mov rcx, [rbx + ARRAY_LEN_OFF]
    imul rcx, VALUE_SIZE
    rep movsb

    ; Copy second array (rdi already advanced)
    mov rsi, [r12 + ARRAY_DATA_OFF]
    mov rcx, [r12 + ARRAY_LEN_OFF]
    imul rcx, VALUE_SIZE
    rep movsb

    pop rax

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; array_slice - Extract a slice of the array
; Input: rdi = array, rsi = start, rdx = end (exclusive)
; Output: rax = new array
;-----------------------------------------------------------------------------
global array_slice
array_slice:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14

    mov rbx, rdi            ; Array
    mov r12, rsi            ; Start
    mov r13, rdx            ; End

    ; Validate and clamp indices
    mov rax, [rbx + ARRAY_LEN_OFF]

    cmp r12, rax
    cmova r12, rax

    cmp r13, rax
    cmova r13, rax

    cmp r13, r12
    jb .empty

    ; Calculate slice length
    mov r14, r13
    sub r14, r12

    ; Allocate new array
    mov rdi, r14
    call array_alloc
    test rax, rax
    jz .done

    push rax

    ; Set length
    mov [rax + ARRAY_LEN_OFF], r14

    ; Copy elements
    mov rdi, [rax + ARRAY_DATA_OFF]
    mov rsi, [rbx + ARRAY_DATA_OFF]
    imul rcx, r12, VALUE_SIZE
    add rsi, rcx            ; Start offset
    imul rcx, r14, VALUE_SIZE
    rep movsb

    pop rax
    jmp .done

.empty:
    xor edi, edi
    call array_alloc

.done:
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; array_reverse - Reverse array in place
; Input: rdi = array header
;-----------------------------------------------------------------------------
global array_reverse
array_reverse:
    push rbp
    mov rbp, rsp
    sub rsp, VALUE_SIZE     ; Temp space
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; Array header

    mov r12, [rbx + ARRAY_LEN_OFF]
    test r12, r12
    jz .done

    mov r13, [rbx + ARRAY_DATA_OFF]

    ; Two pointers: left and right
    xor ecx, ecx            ; Left index
    dec r12                 ; Right index

.swap_loop:
    cmp rcx, r12
    jge .done

    ; Swap elements at rcx and r12
    ; temp = arr[left]
    imul rax, rcx, VALUE_SIZE
    lea rsi, [r13 + rax]
    lea rdi, [rbp - VALUE_SIZE]
    push rcx
    mov ecx, VALUE_SIZE
    rep movsb
    pop rcx

    ; arr[left] = arr[right]
    imul rax, rcx, VALUE_SIZE
    lea rdi, [r13 + rax]
    imul rax, r12, VALUE_SIZE
    lea rsi, [r13 + rax]
    push rcx
    mov ecx, VALUE_SIZE
    rep movsb
    pop rcx

    ; arr[right] = temp
    imul rax, r12, VALUE_SIZE
    lea rdi, [r13 + rax]
    lea rsi, [rbp - VALUE_SIZE]
    push rcx
    mov ecx, VALUE_SIZE
    rep movsb
    pop rcx

    inc rcx
    dec r12
    jmp .swap_loop

.done:
    pop r13
    pop r12
    pop rbx
    add rsp, VALUE_SIZE
    pop rbp
    ret

;-----------------------------------------------------------------------------
; array_index_of - Find first index of value
; Input: rdi = array header, rsi = pointer to Value to find
; Output: rax = index, or -1 if not found
;-----------------------------------------------------------------------------
global array_index_of
array_index_of:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; Array
    mov r12, rsi            ; Value to find

    mov r13, [rbx + ARRAY_LEN_OFF]
    mov rdi, [rbx + ARRAY_DATA_OFF]

    xor ecx, ecx            ; Index

.search:
    cmp rcx, r13
    jae .not_found

    ; Compare values
    push rcx
    push rdi

    mov rsi, r12
    ; Simple comparison: compare tag and num fields
    mov al, [rdi + VALUE_TAG_OFF]
    cmp al, [rsi + VALUE_TAG_OFF]
    jne .next

    mov rax, [rdi + VALUE_NUM_OFF]
    cmp rax, [rsi + VALUE_NUM_OFF]
    jne .next

    ; Found
    pop rdi
    pop rax                 ; Index
    jmp .done

.next:
    pop rdi
    pop rcx
    add rdi, VALUE_SIZE
    inc rcx
    jmp .search

.not_found:
    mov rax, -1

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; array_contains - Check if array contains value
; Input: rdi = array header, rsi = pointer to Value
; Output: rax = 1 if contains, 0 otherwise
;-----------------------------------------------------------------------------
global array_contains
array_contains:
    call array_index_of
    cmp rax, -1
    setne al
    movzx eax, al
    ret
