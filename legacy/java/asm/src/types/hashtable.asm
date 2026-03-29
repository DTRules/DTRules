; hashtable.asm - Hash table (map) implementation
; DTRules Assembly Implementation
; Key-value map with string keys

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state
extern heap_alloc
extern string_equals
extern string_length
extern array_alloc
extern array_push

;-----------------------------------------------------------------------------
; Hash table structure
; Offset 0:  bucket count (8 bytes)
; Offset 8:  entry count (8 bytes)
; Offset 16: buckets pointer (8 bytes) - array of bucket heads
;
; Each bucket is a linked list of entries:
; Entry:
; Offset 0:  key pointer (string) (8 bytes)
; Offset 8:  value (24 bytes - Value struct)
; Offset 32: next pointer (8 bytes)
;-----------------------------------------------------------------------------

HASHTABLE_BUCKETS_OFF   equ 0
HASHTABLE_COUNT_OFF     equ 8
HASHTABLE_DATA_OFF      equ 16

ENTRY_KEY_OFF           equ 0
ENTRY_VALUE_OFF         equ 8
ENTRY_NEXT_OFF          equ 32
ENTRY_SIZE              equ 40

DEFAULT_BUCKET_COUNT    equ 16

section .text

;-----------------------------------------------------------------------------
; hashtable_alloc - Allocate a new hash table
; Input: rdi = initial bucket count (0 for default)
; Output: rax = pointer to hash table, or 0 on error
;-----------------------------------------------------------------------------
global hashtable_alloc
hashtable_alloc:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    ; Use default if 0
    test rdi, rdi
    jnz .have_count
    mov rdi, DEFAULT_BUCKET_COUNT
.have_count:
    mov r12, rdi            ; Save bucket count

    ; Allocate header (24 bytes)
    mov rdi, 24
    call heap_alloc
    test rax, rax
    jz .fail
    mov rbx, rax

    ; Initialize header
    mov [rbx + HASHTABLE_BUCKETS_OFF], r12
    mov qword [rbx + HASHTABLE_COUNT_OFF], 0

    ; Allocate bucket array (8 bytes per bucket)
    imul rdi, r12, 8
    call heap_alloc
    test rax, rax
    jz .fail
    mov [rbx + HASHTABLE_DATA_OFF], rax

    ; Zero the bucket array
    mov rdi, rax
    imul rcx, r12, 8
    xor al, al
    rep stosb

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
; hashtable_hash - Compute hash of a string key
; Input: rdi = string pointer
; Output: rax = hash value
;-----------------------------------------------------------------------------
hashtable_hash:
    push rbx

    mov rcx, [rdi]          ; Length
    add rdi, 8              ; Data
    xor eax, eax            ; Hash accumulator

.hash_loop:
    test rcx, rcx
    jz .hash_done

    imul eax, 31
    movzx edx, byte [rdi]
    add eax, edx
    inc rdi
    dec rcx
    jmp .hash_loop

.hash_done:
    pop rbx
    ret

;-----------------------------------------------------------------------------
; hashtable_get - Get value for key
; Input: rdi = table pointer, rsi = key string
; Output: rax = pointer to Value, or 0 if not found
;-----------------------------------------------------------------------------
global hashtable_get
hashtable_get:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; Table
    mov r12, rsi            ; Key

    ; Hash the key
    mov rdi, r12
    call hashtable_hash
    mov r13, rax            ; Hash value

    ; Find bucket
    xor edx, edx
    mov rcx, [rbx + HASHTABLE_BUCKETS_OFF]
    div rcx                 ; rdx = hash % bucket_count

    ; Get bucket head
    mov rax, [rbx + HASHTABLE_DATA_OFF]
    mov rax, [rax + rdx * 8]

    ; Search linked list
.search:
    test rax, rax
    jz .not_found

    ; Compare keys
    push rax
    mov rdi, [rax + ENTRY_KEY_OFF]
    mov rsi, r12
    call string_equals
    pop rcx                 ; Entry pointer

    test eax, eax
    jnz .found

    ; Next entry
    mov rax, [rcx + ENTRY_NEXT_OFF]
    jmp .search

.found:
    ; Return pointer to value
    lea rax, [rcx + ENTRY_VALUE_OFF]
    jmp .done

.not_found:
    xor eax, eax

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; hashtable_put - Put key-value pair
; Input: rdi = table pointer, rsi = key string, rdx = pointer to Value
; Output: rax = 0 on success
;-----------------------------------------------------------------------------
global hashtable_put
hashtable_put:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14
    push r15

    mov rbx, rdi            ; Table
    mov r12, rsi            ; Key
    mov r13, rdx            ; Value pointer

    ; Hash the key
    mov rdi, r12
    call hashtable_hash
    mov r14, rax            ; Hash value

    ; Find bucket index
    xor edx, edx
    mov rcx, [rbx + HASHTABLE_BUCKETS_OFF]
    mov rax, r14
    div rcx                 ; rdx = hash % bucket_count
    mov r15, rdx            ; Bucket index

    ; Get bucket head
    mov rax, [rbx + HASHTABLE_DATA_OFF]
    mov rcx, [rax + r15 * 8]

    ; Search for existing key
.put_search:
    test rcx, rcx
    jz .put_new

    push rcx
    mov rdi, [rcx + ENTRY_KEY_OFF]
    mov rsi, r12
    call string_equals
    pop rcx

    test eax, eax
    jnz .put_update

    mov rcx, [rcx + ENTRY_NEXT_OFF]
    jmp .put_search

.put_update:
    ; Update existing entry
    lea rdi, [rcx + ENTRY_VALUE_OFF]
    mov rsi, r13
    mov ecx, VALUE_SIZE
    rep movsb
    xor eax, eax
    jmp .put_done

.put_new:
    ; Allocate new entry
    mov rdi, ENTRY_SIZE
    call heap_alloc
    test rax, rax
    jz .put_fail
    mov rcx, rax            ; New entry

    ; Initialize entry
    mov [rcx + ENTRY_KEY_OFF], r12

    ; Copy value
    push rcx
    lea rdi, [rcx + ENTRY_VALUE_OFF]
    mov rsi, r13
    mov ecx, VALUE_SIZE
    rep movsb
    pop rcx

    ; Insert at head of bucket
    mov rax, [rbx + HASHTABLE_DATA_OFF]
    mov rdx, [rax + r15 * 8]        ; Old head
    mov [rcx + ENTRY_NEXT_OFF], rdx ; New->next = old head
    mov [rax + r15 * 8], rcx        ; Bucket head = new

    ; Increment count
    inc qword [rbx + HASHTABLE_COUNT_OFF]

    xor eax, eax
    jmp .put_done

.put_fail:
    mov eax, ERR_OUT_OF_MEMORY

.put_done:
    pop r15
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; hashtable_contains - Check if key exists
; Input: rdi = table pointer, rsi = key string
; Output: rax = 1 if exists, 0 otherwise
;-----------------------------------------------------------------------------
global hashtable_contains
hashtable_contains:
    call hashtable_get
    test rax, rax
    setnz al
    movzx eax, al
    ret

;-----------------------------------------------------------------------------
; hashtable_remove - Remove key from table
; Input: rdi = table pointer, rsi = key string
; Output: rax = pointer to removed Value, or 0 if not found
;-----------------------------------------------------------------------------
global hashtable_remove
hashtable_remove:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14
    push r15

    mov rbx, rdi            ; Table
    mov r12, rsi            ; Key

    ; Hash the key
    mov rdi, r12
    call hashtable_hash
    mov r13, rax

    ; Find bucket index
    xor edx, edx
    mov rcx, [rbx + HASHTABLE_BUCKETS_OFF]
    div rcx
    mov r14, rdx            ; Bucket index

    ; Get bucket pointer
    mov rax, [rbx + HASHTABLE_DATA_OFF]
    lea r15, [rax + r14 * 8]    ; Pointer to bucket head pointer

    mov rcx, [r15]          ; Current entry

.remove_search:
    test rcx, rcx
    jz .remove_not_found

    push rcx
    mov rdi, [rcx + ENTRY_KEY_OFF]
    mov rsi, r12
    call string_equals
    pop rcx

    test eax, eax
    jnz .remove_found

    ; Advance prev pointer and current
    lea r15, [rcx + ENTRY_NEXT_OFF]
    mov rcx, [r15]
    jmp .remove_search

.remove_found:
    ; Unlink entry: prev->next = entry->next
    mov rax, [rcx + ENTRY_NEXT_OFF]
    mov [r15], rax

    ; Decrement count
    dec qword [rbx + HASHTABLE_COUNT_OFF]

    ; Return pointer to value
    lea rax, [rcx + ENTRY_VALUE_OFF]
    jmp .remove_done

.remove_not_found:
    xor eax, eax

.remove_done:
    pop r15
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; hashtable_size - Get number of entries
; Input: rdi = table pointer
; Output: rax = entry count
;-----------------------------------------------------------------------------
global hashtable_size
hashtable_size:
    mov rax, [rdi + HASHTABLE_COUNT_OFF]
    ret

;-----------------------------------------------------------------------------
; hashtable_keys - Get array of all keys
; Input: rdi = table pointer
; Output: rax = array of key strings
;-----------------------------------------------------------------------------
global hashtable_keys
hashtable_keys:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14

    mov rbx, rdi            ; Table

    ; Allocate result array
    mov rdi, [rbx + HASHTABLE_COUNT_OFF]
    test rdi, rdi
    jz .keys_empty
    call array_alloc
    test rax, rax
    jz .keys_fail
    mov r12, rax            ; Result array

    ; Iterate all buckets
    mov r13, [rbx + HASHTABLE_BUCKETS_OFF]
    mov r14, [rbx + HASHTABLE_DATA_OFF]
    xor ecx, ecx            ; Bucket index

.keys_bucket_loop:
    cmp rcx, r13
    jae .keys_done

    push rcx
    mov rax, [r14 + rcx * 8]    ; Bucket head

.keys_entry_loop:
    test rax, rax
    jz .keys_next_bucket

    push rax
    ; Create string Value for key
    mov rdi, [rax + ENTRY_KEY_OFF]
    ; We need to create a Value wrapper
    ; For simplicity, just push the key string directly as the value
    ; The array_push expects a Value pointer
    sub rsp, 24             ; Temp Value on stack
    mov byte [rsp], VTAG_STRING
    mov qword [rsp + 8], 0
    mov [rsp + 16], rdi

    mov rdi, r12
    mov rsi, rsp
    call array_push

    add rsp, 24
    pop rax
    mov rax, [rax + ENTRY_NEXT_OFF]
    jmp .keys_entry_loop

.keys_next_bucket:
    pop rcx
    inc rcx
    jmp .keys_bucket_loop

.keys_done:
    mov rax, r12
    jmp .keys_exit

.keys_empty:
    xor edi, edi
    call array_alloc
    jmp .keys_exit

.keys_fail:
    xor eax, eax

.keys_exit:
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; hashtable_values - Get array of all values
; Input: rdi = table pointer
; Output: rax = array of values
;-----------------------------------------------------------------------------
global hashtable_values
hashtable_values:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14

    mov rbx, rdi            ; Table

    ; Allocate result array
    mov rdi, [rbx + HASHTABLE_COUNT_OFF]
    test rdi, rdi
    jz .values_empty
    call array_alloc
    test rax, rax
    jz .values_fail
    mov r12, rax            ; Result array

    ; Iterate all buckets
    mov r13, [rbx + HASHTABLE_BUCKETS_OFF]
    mov r14, [rbx + HASHTABLE_DATA_OFF]
    xor ecx, ecx

.values_bucket_loop:
    cmp rcx, r13
    jae .values_done

    push rcx
    mov rax, [r14 + rcx * 8]

.values_entry_loop:
    test rax, rax
    jz .values_next_bucket

    push rax
    lea rsi, [rax + ENTRY_VALUE_OFF]
    mov rdi, r12
    call array_push
    pop rax
    mov rax, [rax + ENTRY_NEXT_OFF]
    jmp .values_entry_loop

.values_next_bucket:
    pop rcx
    inc rcx
    jmp .values_bucket_loop

.values_done:
    mov rax, r12
    jmp .values_exit

.values_empty:
    xor edi, edi
    call array_alloc
    jmp .values_exit

.values_fail:
    xor eax, eax

.values_exit:
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret
