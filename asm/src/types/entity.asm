; entity.asm - Typed dictionary (entity) operations
; DTRules Assembly Implementation
; Entities are typed dictionaries mapping names to values

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"

extern state
extern heap_alloc
extern string_equals

;-----------------------------------------------------------------------------
; Entity structure (stored at VALUE_PTR_OFF)
; Offset 0:  type name pointer (8 bytes, or 0 if untyped)
; Offset 8:  attribute count (8 bytes)
; Offset 16: attribute capacity (8 bytes)
; Offset 24: attributes pointer (8 bytes)
;
; Each attribute entry (32 bytes):
; Offset 0:  name pointer (8 bytes, length-prefixed string)
; Offset 8:  value (24 bytes, inline Value)
;-----------------------------------------------------------------------------

ENTITY_TYPE_OFF     equ 0
ENTITY_COUNT_OFF    equ 8
ENTITY_CAP_OFF      equ 16
ENTITY_ATTRS_OFF    equ 24

ATTR_NAME_OFF       equ 0
ATTR_VALUE_OFF      equ 8
ATTR_SIZE           equ 32

section .text

;-----------------------------------------------------------------------------
; entity_alloc - Allocate a new entity
; Input: rdi = type name pointer (can be 0 for untyped)
; Output: rax = pointer to entity structure
;-----------------------------------------------------------------------------
global entity_alloc
entity_alloc:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    mov r12, rdi            ; Save type name

    ; Allocate entity header (32 bytes)
    mov rdi, 32
    call heap_alloc
    test rax, rax
    jz .done

    mov rbx, rax            ; Save header pointer

    ; Initialize header
    mov [rbx + ENTITY_TYPE_OFF], r12
    mov qword [rbx + ENTITY_COUNT_OFF], 0
    mov qword [rbx + ENTITY_CAP_OFF], ENTITY_INITIAL_CAP

    ; Allocate attribute array
    mov rdi, ENTITY_INITIAL_CAP * ATTR_SIZE
    call heap_alloc
    test rax, rax
    jz .fail

    mov [rbx + ENTITY_ATTRS_OFF], rax

    ; Zero attribute array
    mov rdi, rax
    mov rcx, ENTITY_INITIAL_CAP * ATTR_SIZE
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
; entity_get_type - Get entity type name
; Input: rdi = entity pointer
; Output: rax = type name pointer (or 0 if untyped)
;-----------------------------------------------------------------------------
global entity_get_type
entity_get_type:
    mov rax, [rdi + ENTITY_TYPE_OFF]
    ret

;-----------------------------------------------------------------------------
; entity_attr_count - Get number of attributes
; Input: rdi = entity pointer
; Output: rax = count
;-----------------------------------------------------------------------------
global entity_attr_count
entity_attr_count:
    mov rax, [rdi + ENTITY_COUNT_OFF]
    ret

;-----------------------------------------------------------------------------
; entity_find_attr - Find attribute by name
; Input: rdi = entity pointer, rsi = name pointer (length-prefixed)
; Output: rax = pointer to attribute entry, or 0 if not found
;-----------------------------------------------------------------------------
global entity_find_attr
entity_find_attr:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; Entity
    mov r12, rsi            ; Name to find

    mov r13, [rbx + ENTITY_COUNT_OFF]
    mov rdi, [rbx + ENTITY_ATTRS_OFF]

    xor ecx, ecx            ; Index

.search:
    cmp rcx, r13
    jae .not_found

    ; Get attribute name
    mov rsi, [rdi + ATTR_NAME_OFF]
    test rsi, rsi
    jz .next                ; Empty slot

    ; Compare names
    push rcx
    push rdi
    mov rdi, r12
    ; rsi already has attr name
    call string_equals
    pop rdi
    pop rcx

    test eax, eax
    jnz .found

.next:
    add rdi, ATTR_SIZE
    inc rcx
    jmp .search

.found:
    mov rax, rdi
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
; entity_get_attr - Get attribute value
; Input: rdi = entity pointer, rsi = name pointer
; Output: rax = pointer to Value, or 0 if not found
;-----------------------------------------------------------------------------
global entity_get_attr
entity_get_attr:
    push rbp
    mov rbp, rsp

    call entity_find_attr
    test rax, rax
    jz .done

    ; Return pointer to value within attribute entry
    add rax, ATTR_VALUE_OFF

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; entity_has_attr - Check if entity has attribute
; Input: rdi = entity pointer, rsi = name pointer
; Output: rax = 1 if has, 0 otherwise
;-----------------------------------------------------------------------------
global entity_has_attr
entity_has_attr:
    call entity_find_attr
    test rax, rax
    setnz al
    movzx eax, al
    ret

;-----------------------------------------------------------------------------
; entity_grow - Grow entity capacity
; Input: rdi = entity pointer
; Output: rax = 0 on success
;-----------------------------------------------------------------------------
global entity_grow
entity_grow:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; Entity

    ; Calculate new capacity
    mov r12, [rbx + ENTITY_CAP_OFF]
    shl r12, 1              ; Double

    ; Allocate new attribute array
    imul rdi, r12, ATTR_SIZE
    call heap_alloc
    test rax, rax
    jz .fail

    mov r13, rax            ; New attrs pointer

    ; Zero new array
    mov rdi, r13
    imul rcx, r12, ATTR_SIZE
    push rax
    xor al, al
    rep stosb
    pop rax

    ; Copy existing attributes
    mov rdi, r13
    mov rsi, [rbx + ENTITY_ATTRS_OFF]
    mov rcx, [rbx + ENTITY_COUNT_OFF]
    imul rcx, ATTR_SIZE
    rep movsb

    ; Update entity
    mov [rbx + ENTITY_CAP_OFF], r12
    mov [rbx + ENTITY_ATTRS_OFF], r13

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
; entity_set_attr - Set or create attribute
; Input: rdi = entity pointer, rsi = name pointer, rdx = value pointer
; Output: rax = 0 on success
;-----------------------------------------------------------------------------
global entity_set_attr
entity_set_attr:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; Entity
    mov r12, rsi            ; Name
    mov r13, rdx            ; Value

    ; Try to find existing attribute
    call entity_find_attr
    test rax, rax
    jnz .update

    ; Need to add new attribute
    ; Check capacity
    mov rax, [rbx + ENTITY_COUNT_OFF]
    cmp rax, [rbx + ENTITY_CAP_OFF]
    jb .have_space

    mov rdi, rbx
    call entity_grow
    test rax, rax
    jnz .done

.have_space:
    ; Find empty slot or append
    mov rax, [rbx + ENTITY_COUNT_OFF]
    imul rax, ATTR_SIZE
    add rax, [rbx + ENTITY_ATTRS_OFF]

    ; Set name
    mov [rax + ATTR_NAME_OFF], r12

    ; Increment count
    inc qword [rbx + ENTITY_COUNT_OFF]

.update:
    ; rax points to attribute entry
    ; Copy value
    lea rdi, [rax + ATTR_VALUE_OFF]
    mov rsi, r13
    mov ecx, VALUE_SIZE
    rep movsb

    xor eax, eax

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; entity_delete_attr - Delete attribute
; Input: rdi = entity pointer, rsi = name pointer
; Output: rax = 0 on success, non-zero if not found
;-----------------------------------------------------------------------------
global entity_delete_attr
entity_delete_attr:
    push rbp
    mov rbp, rsp
    push rbx

    mov rbx, rdi            ; Save entity

    ; Find attribute
    call entity_find_attr
    test rax, rax
    jz .not_found

    ; Clear the slot (set name to 0)
    mov qword [rax + ATTR_NAME_OFF], 0

    ; Note: We don't compact the array for simplicity
    ; This means deleted slots remain until entity is recreated

    xor eax, eax
    jmp .done

.not_found:
    mov eax, ERR_ATTR_NOT_FOUND

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; entity_get_attrs - Get list of attribute names
; Input: rdi = entity pointer
; Output: rax = array of name pointers (needs to be freed)
;         rdx = count
;-----------------------------------------------------------------------------
global entity_get_attrs
entity_get_attrs:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; Entity

    mov r12, [rbx + ENTITY_COUNT_OFF]

    ; Allocate array of pointers
    imul rdi, r12, 8
    add rdi, 8              ; Extra space
    call heap_alloc
    test rax, rax
    jz .done

    mov r13, rax            ; Result array

    ; Copy non-null name pointers
    mov rsi, [rbx + ENTITY_ATTRS_OFF]
    xor ecx, ecx            ; Source index
    xor edx, edx            ; Dest index

.copy_loop:
    cmp rcx, r12
    jae .copy_done

    mov rax, [rsi + ATTR_NAME_OFF]
    test rax, rax
    jz .skip

    mov [r13 + rdx * 8], rax
    inc rdx

.skip:
    add rsi, ATTR_SIZE
    inc rcx
    jmp .copy_loop

.copy_done:
    mov rax, r13
    ; rdx has count

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; entity_copy - Create a shallow copy of an entity
; Input: rdi = source entity pointer
; Output: rax = new entity pointer
;-----------------------------------------------------------------------------
global entity_copy
entity_copy:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    mov rbx, rdi            ; Source entity

    ; Allocate new entity with same type
    mov rdi, [rbx + ENTITY_TYPE_OFF]
    call entity_alloc
    test rax, rax
    jz .done

    mov r12, rax            ; New entity

    ; Copy attributes
    mov rsi, [rbx + ENTITY_ATTRS_OFF]
    mov rcx, [rbx + ENTITY_COUNT_OFF]

.copy_loop:
    test rcx, rcx
    jz .copy_done

    ; Get name
    mov rax, [rsi + ATTR_NAME_OFF]
    test rax, rax
    jz .next

    ; Set attribute in new entity
    push rcx
    push rsi
    mov rdi, r12
    mov rsi, rax            ; Name
    lea rdx, [rsi + ATTR_VALUE_OFF - ATTR_NAME_OFF]  ; Actually need original rsi
    pop rsi
    push rsi
    lea rdx, [rsi + ATTR_VALUE_OFF]
    mov rsi, [rsi + ATTR_NAME_OFF]
    call entity_set_attr
    pop rsi
    pop rcx

.next:
    add rsi, ATTR_SIZE
    dec rcx
    jmp .copy_loop

.copy_done:
    mov rax, r12

.done:
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; entity_merge - Merge attributes from one entity into another
; Input: rdi = destination entity, rsi = source entity
; Output: rax = 0 on success
;
; Attributes from source override existing attributes in destination
;-----------------------------------------------------------------------------
global entity_merge
entity_merge:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; Destination
    mov r12, rsi            ; Source

    mov r13, [r12 + ENTITY_ATTRS_OFF]
    mov rcx, [r12 + ENTITY_COUNT_OFF]

.merge_loop:
    test rcx, rcx
    jz .done

    mov rax, [r13 + ATTR_NAME_OFF]
    test rax, rax
    jz .next

    ; Set attribute in destination
    push rcx
    mov rdi, rbx
    mov rsi, rax            ; Name
    lea rdx, [r13 + ATTR_VALUE_OFF]
    call entity_set_attr
    pop rcx

.next:
    add r13, ATTR_SIZE
    dec rcx
    jmp .merge_loop

.done:
    xor eax, eax
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; entity_clear - Remove all attributes
; Input: rdi = entity pointer
;-----------------------------------------------------------------------------
global entity_clear
entity_clear:
    push rbp
    mov rbp, rsp

    ; Zero the count
    mov qword [rdi + ENTITY_COUNT_OFF], 0

    ; Zero the attribute array
    push rdi
    mov rax, [rdi + ENTITY_CAP_OFF]
    imul rcx, rax, ATTR_SIZE
    mov rdi, [rdi + ENTITY_ATTRS_OFF]
    xor al, al
    rep stosb
    pop rdi

    pop rbp
    ret
