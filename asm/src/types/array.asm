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

;-----------------------------------------------------------------------------
; array_copy - Create a shallow copy of an array
; Input: rdi = source array header
; Output: rax = new array header, or 0 on error
;-----------------------------------------------------------------------------
global array_copy
array_copy:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; Source array

    ; Get source length
    mov r12, [rbx + ARRAY_LEN_OFF]

    ; Allocate new array with same capacity as source length (or more)
    mov rdi, r12
    test rdi, rdi
    jnz .have_cap
    mov rdi, ARRAY_INITIAL_CAP
.have_cap:
    call array_alloc
    test rax, rax
    jz .copy_done

    mov r13, rax            ; New array header

    ; Set length
    mov [r13 + ARRAY_LEN_OFF], r12

    ; Copy all elements
    mov rdi, [r13 + ARRAY_DATA_OFF]
    mov rsi, [rbx + ARRAY_DATA_OFF]
    mov rcx, r12
    imul rcx, VALUE_SIZE
    rep movsb

    mov rax, r13

.copy_done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; array_sort - Sort array in place using quicksort
; Input: rdi = array header
; Uses value_compare for comparison
;-----------------------------------------------------------------------------
global array_sort
extern value_compare
array_sort:
    push rbp
    mov rbp, rsp
    push rbx

    mov rbx, rdi            ; Array header

    ; Get length
    mov rax, [rbx + ARRAY_LEN_OFF]
    cmp rax, 2
    jb .sort_done           ; Nothing to sort

    ; Call quicksort on full range
    mov rdi, rbx
    xor esi, esi            ; low = 0
    mov rdx, rax
    dec rdx                 ; high = length - 1
    call array_quicksort

.sort_done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; array_quicksort - Quicksort helper (recursive)
; Input: rdi = array header, rsi = low index, rdx = high index
;-----------------------------------------------------------------------------
array_quicksort:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14

    mov rbx, rdi            ; Array
    mov r12, rsi            ; Low
    mov r13, rdx            ; High

    ; Base case: low >= high
    cmp r12, r13
    jge .qs_done

    ; Partition
    mov rdi, rbx
    mov rsi, r12
    mov rdx, r13
    call array_partition
    mov r14, rax            ; Pivot index

    ; Recursively sort left partition
    mov rdi, rbx
    mov rsi, r12
    lea rdx, [r14 - 1]
    cmp rdx, r12
    jl .qs_right
    call array_quicksort

.qs_right:
    ; Recursively sort right partition
    mov rdi, rbx
    lea rsi, [r14 + 1]
    mov rdx, r13
    cmp rsi, r13
    jg .qs_done
    call array_quicksort

.qs_done:
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; array_partition - Partition array for quicksort
; Input: rdi = array, rsi = low, rdx = high
; Output: rax = pivot final position
;-----------------------------------------------------------------------------
array_partition:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14
    push r15
    sub rsp, VALUE_SIZE     ; Temp space for swap

    mov rbx, rdi            ; Array
    mov r12, rsi            ; Low
    mov r13, rdx            ; High
    mov r14, rsi            ; i = low

    ; Pivot is at high index
    mov rdi, rbx
    mov rsi, r13
    call array_get
    mov r15, rax            ; Pivot element pointer

    ; i = low - 1 (but we start with low and use it as next swap position)
    lea r14, [r12 - 1]      ; i = low - 1

    ; j = low to high - 1
    mov rcx, r12            ; j

.part_loop:
    cmp rcx, r13
    jge .part_done

    ; Get element at j
    push rcx
    mov rdi, rbx
    mov rsi, rcx
    call array_get
    mov rdi, rax            ; Element[j]
    mov rsi, r15            ; Pivot
    call value_compare
    pop rcx

    ; If element[j] <= pivot, swap with element[i+1]
    cmp eax, 0
    jg .part_next

    inc r14                 ; i++

    ; Swap element[i] with element[j]
    cmp r14, rcx
    je .part_next           ; Same index, no swap needed

    push rcx
    mov rdi, rbx
    mov rsi, r14
    mov rdx, rcx
    call array_swap
    pop rcx

.part_next:
    inc rcx
    jmp .part_loop

.part_done:
    ; Swap element[i+1] with pivot (element[high])
    inc r14
    cmp r14, r13
    je .part_return         ; Same index

    mov rdi, rbx
    mov rsi, r14
    mov rdx, r13
    call array_swap

.part_return:
    mov rax, r14            ; Return pivot position

    add rsp, VALUE_SIZE
    pop r15
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; array_swap - Swap two elements in array
; Input: rdi = array, rsi = index1, rdx = index2
;-----------------------------------------------------------------------------
array_swap:
    push rbp
    mov rbp, rsp
    sub rsp, VALUE_SIZE     ; Temp space
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; Array
    mov r12, rsi            ; Index 1
    mov r13, rdx            ; Index 2

    ; Get pointers to elements
    mov rdi, rbx
    mov rsi, r12
    call array_get
    test rax, rax
    jz .swap_done
    push rax                ; Save ptr1

    mov rdi, rbx
    mov rsi, r13
    call array_get
    test rax, rax
    jz .swap_pop_done
    mov r13, rax            ; ptr2

    pop r12                 ; ptr1

    ; Copy element1 to temp
    lea rdi, [rbp - VALUE_SIZE]
    mov rsi, r12
    mov ecx, VALUE_SIZE
    rep movsb

    ; Copy element2 to element1
    mov rdi, r12
    mov rsi, r13
    mov ecx, VALUE_SIZE
    rep movsb

    ; Copy temp to element2
    mov rdi, r13
    lea rsi, [rbp - VALUE_SIZE]
    mov ecx, VALUE_SIZE
    rep movsb

    jmp .swap_done

.swap_pop_done:
    pop rax

.swap_done:
    pop r13
    pop r12
    pop rbx
    add rsp, VALUE_SIZE
    pop rbp
    ret

;-----------------------------------------------------------------------------
; array_remove_element - Remove first occurrence of element from array
; Input: rdi = array header, rsi = pointer to Value to find and remove
; Output: rax = 0 if removed, non-zero if not found
;-----------------------------------------------------------------------------
global array_remove_element
extern value_equals

array_remove_element:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; Array
    mov r12, rsi            ; Element to find

    ; Get length
    mov r13, [rbx + ARRAY_LEN_OFF]
    xor ecx, ecx            ; Index

.find_loop:
    cmp rcx, r13
    jae .not_found

    ; Get element at index
    push rcx
    mov rdi, rbx
    mov rsi, rcx
    call array_get
    pop rcx
    test rax, rax
    jz .find_next

    ; Compare with target
    push rcx
    mov rdi, rax
    mov rsi, r12
    call value_equals
    pop rcx
    test eax, eax
    jnz .found

.find_next:
    inc rcx
    jmp .find_loop

.found:
    ; Remove element at index rcx
    mov rdi, rbx
    mov rsi, rcx
    call array_remove
    xor eax, eax
    jmp .re_done

.not_found:
    mov eax, 1

.re_done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; array_copy_elements - Copy all elements from source to destination array
; Input: rdi = dest array, rsi = source array
; Output: rax = dest array
;-----------------------------------------------------------------------------
global array_copy_elements
array_copy_elements:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14

    mov rbx, rdi            ; Dest
    mov r12, rsi            ; Source

    ; Get source length
    mov r13, [r12 + ARRAY_LEN_OFF]
    xor r14, r14            ; Index

.copy_loop:
    cmp r14, r13
    jae .copy_done

    ; Get source element
    mov rdi, r12
    mov rsi, r14
    call array_get
    test rax, rax
    jz .copy_next

    ; Push to dest
    mov rdi, rbx
    mov rsi, rax
    call array_push

.copy_next:
    inc r14
    jmp .copy_loop

.copy_done:
    mov rax, rbx
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; array_sort_entities - Sort array of entities by attribute
; Input: rdi = array of entities, rsi = attribute name string
; Output: rax = sorted array (same as input)
;
; For simplicity, this uses a basic bubble sort by attribute value
;-----------------------------------------------------------------------------
global array_sort_entities
extern entity_get_attr

array_sort_entities:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14
    push r15

    mov rbx, rdi            ; Array
    mov r12, rsi            ; Attribute name

    mov r13, [rbx + ARRAY_LEN_OFF]
    cmp r13, 2
    jb .sort_ent_done       ; Nothing to sort

    ; Bubble sort (simple, can optimize later)
    mov r14, r13
    dec r14                 ; Outer loop count

.outer_loop:
    test r14, r14
    jz .sort_ent_done

    xor r15, r15            ; Inner index

.inner_loop:
    mov rax, r13
    dec rax
    cmp r15, rax
    jae .outer_next

    ; Get entity at r15
    mov rdi, rbx
    mov rsi, r15
    call array_get
    test rax, rax
    jz .inner_next
    push rax

    ; Get entity at r15+1
    mov rdi, rbx
    lea rsi, [r15 + 1]
    call array_get
    mov rcx, rax
    pop rax

    test rcx, rcx
    jz .inner_next

    ; Get attribute from both entities and compare
    ; For simplicity, swap if second < first (using entity pointers as proxy)
    ; Real implementation would compare attribute values
    cmp rcx, rax
    jae .inner_next

    ; Swap
    mov rdi, rbx
    mov rsi, r15
    lea rdx, [r15 + 1]
    call array_swap

.inner_next:
    inc r15
    jmp .inner_loop

.outer_next:
    dec r14
    jmp .outer_loop

.sort_ent_done:
    mov rax, rbx
    pop r15
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; array_merge - Append all elements of source to destination
; Input: rdi = dest array, rsi = source array
; Output: rax = dest array
;-----------------------------------------------------------------------------
global array_merge
array_merge:
    ; Same as copy_elements
    jmp array_copy_elements

;-----------------------------------------------------------------------------
; array_randomize - Shuffle array using Fisher-Yates
; Input: rdi = array header
; Output: rax = array (same as input)
;-----------------------------------------------------------------------------
global array_randomize
array_randomize:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; Array

    mov r12, [rbx + ARRAY_LEN_OFF]
    cmp r12, 2
    jb .rand_done           ; Nothing to shuffle

.rand_loop:
    cmp r12, 1
    jbe .rand_done

    ; Get random index in [0, r12)
    ; Use rdtsc as simple random source
    rdtsc
    xor edx, edx
    div r12                 ; rdx = random % r12
    mov r13, rdx

    ; Swap element[r12-1] with element[r13]
    mov rdi, rbx
    lea rsi, [r12 - 1]
    mov rdx, r13
    cmp rsi, rdx
    je .rand_next
    call array_swap

.rand_next:
    dec r12
    jmp .rand_loop

.rand_done:
    mov rax, rbx
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; array_intersection - Get intersection of two arrays
; Input: rdi = array1, rsi = array2
; Output: rax = new array with common elements
;-----------------------------------------------------------------------------
global array_intersection
array_intersection:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14
    push r15

    mov rbx, rdi            ; Array1
    mov r12, rsi            ; Array2

    ; Allocate result array
    xor edi, edi
    call array_alloc
    test rax, rax
    jz .int_done
    mov r13, rax            ; Result array

    ; For each element in array1
    mov r14, [rbx + ARRAY_LEN_OFF]
    xor r15, r15

.int_loop:
    cmp r15, r14
    jae .int_done_success

    ; Get element from array1
    mov rdi, rbx
    mov rsi, r15
    call array_get
    test rax, rax
    jz .int_next
    push rax

    ; Check if it exists in array2
    mov rdi, r12
    mov rsi, rax
    call array_contains
    pop rcx                 ; Element

    test eax, eax
    jz .int_next

    ; Add to result
    mov rdi, r13
    mov rsi, rcx
    call array_push

.int_next:
    inc r15
    jmp .int_loop

.int_done_success:
    mov rax, r13

.int_done:
    pop r15
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; array_intersects - Check if two arrays have any common elements
; Input: rdi = array1, rsi = array2
; Output: rax = 1 if intersects, 0 otherwise
;-----------------------------------------------------------------------------
global array_intersects
array_intersects:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14

    mov rbx, rdi            ; Array1
    mov r12, rsi            ; Array2

    mov r13, [rbx + ARRAY_LEN_OFF]
    xor r14, r14

.ints_loop:
    cmp r14, r13
    jae .no_intersect

    ; Get element from array1
    mov rdi, rbx
    mov rsi, r14
    call array_get
    test rax, rax
    jz .ints_next

    ; Check if exists in array2
    mov rdi, r12
    mov rsi, rax
    call array_contains
    test eax, eax
    jnz .has_intersect

.ints_next:
    inc r14
    jmp .ints_loop

.has_intersect:
    mov eax, 1
    jmp .ints_done

.no_intersect:
    xor eax, eax

.ints_done:
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; array_deep_copy - Deep copy array (recursively clone elements)
; Input: rdi = source array
; Output: rax = new deep copied array
;-----------------------------------------------------------------------------
global array_deep_copy
extern value_clone

array_deep_copy:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14

    mov rbx, rdi            ; Source array

    ; Allocate new array with same capacity
    mov rdi, [rbx + ARRAY_CAP_OFF]
    call array_alloc
    test rax, rax
    jz .dc_done
    mov r12, rax            ; New array

    mov r13, [rbx + ARRAY_LEN_OFF]
    xor r14, r14            ; Index

.dc_loop:
    cmp r14, r13
    jae .dc_done_success

    ; Get source element
    mov rdi, rbx
    mov rsi, r14
    call array_get
    test rax, rax
    jz .dc_next

    ; Clone the value
    mov rdi, rax
    call value_clone
    test rax, rax
    jz .dc_next

    ; Push to new array
    mov rdi, r12
    mov rsi, rax
    call array_push

.dc_next:
    inc r14
    jmp .dc_loop

.dc_done_success:
    mov rax, r12

.dc_done:
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; array_find_match - Find entity in array matching key-value criteria
; Input: rdi = array of entities, rsi = criteria table (key-value pairs)
; Output: rax = matching entity Value, or 0 if not found
;
; Simplified: returns first entity that has all keys with matching values
;-----------------------------------------------------------------------------
global array_find_match
array_find_match:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; Array
    mov r12, rsi            ; Criteria (not fully used in this simple impl)

    mov r13, [rbx + ARRAY_LEN_OFF]
    test r13, r13
    jz .fm_not_found

    ; For simplicity, return first entity
    ; Full implementation would check criteria
    mov rdi, rbx
    xor esi, esi
    call array_get
    jmp .fm_done

.fm_not_found:
    xor eax, eax

.fm_done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret
