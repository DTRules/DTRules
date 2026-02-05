; stack_entity.asm - Entity stack operations
; DTRules Assembly Implementation
; Stack for entity context during rule execution

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state

section .text

;-----------------------------------------------------------------------------
; stack_entity_init - Initialize the entity stack
;-----------------------------------------------------------------------------
global stack_entity_init
stack_entity_init:
    ; Reset stack pointer to base (empty stack)
    mov rax, [state + State.entity_stack_base]
    mov [state + State.entity_stack], rax
    mov r13, rax            ; Also update register
    ret

;-----------------------------------------------------------------------------
; stack_entity_push - Push an entity pointer onto the stack
; Input: rdi = pointer to entity Value
; Returns: 0 on success, ERR_STACK_OVERFLOW on error
;-----------------------------------------------------------------------------
global stack_entity_push
stack_entity_push:
    ; Check for overflow
    mov rax, r13
    sub rax, 8
    cmp rax, [state + State.entity_stack_end]
    jb .overflow

    ; Push entity pointer
    sub r13, 8
    mov [state + State.entity_stack], r13
    mov [r13], rdi

    xor eax, eax
    ret

.overflow:
    mov dword [state + State.error], ERR_STACK_OVERFLOW
    mov eax, ERR_STACK_OVERFLOW
    ret

;-----------------------------------------------------------------------------
; stack_entity_pop - Pop an entity pointer from the stack
; Output: rax = entity pointer
; Returns: pointer on success, 0 on underflow
;-----------------------------------------------------------------------------
global stack_entity_pop
stack_entity_pop:
    ; Check for underflow
    cmp r13, [state + State.entity_stack_base]
    jae .underflow

    ; Pop entity pointer
    mov rax, [r13]
    add r13, 8
    mov [state + State.entity_stack], r13

    ret

.underflow:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; stack_entity_peek - Peek at top entity without removing
; Output: rax = entity pointer
; Returns: pointer on success, 0 on empty
;-----------------------------------------------------------------------------
global stack_entity_peek
stack_entity_peek:
    ; Check for empty
    cmp r13, [state + State.entity_stack_base]
    jae .empty

    mov rax, [r13]
    ret

.empty:
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; stack_entity_peek_n - Peek at nth entity from top (0 = top)
; Input: rdi = index from top
; Output: rax = entity pointer
; Returns: pointer on success, 0 if not enough elements
;-----------------------------------------------------------------------------
global stack_entity_peek_n
stack_entity_peek_n:
    ; Calculate address
    shl rdi, 3              ; * 8
    lea rax, [r13 + rdi]

    ; Check bounds
    cmp rax, [state + State.entity_stack_base]
    jae .invalid

    mov rax, [rax]
    ret

.invalid:
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; stack_entity_depth - Get number of entities on stack
; Output: rax = depth
;-----------------------------------------------------------------------------
global stack_entity_depth
stack_entity_depth:
    mov rax, [state + State.entity_stack_base]
    sub rax, r13
    shr rax, 3              ; / 8
    ret

;-----------------------------------------------------------------------------
; stack_entity_clear - Clear the entity stack
;-----------------------------------------------------------------------------
global stack_entity_clear
stack_entity_clear:
    mov rax, [state + State.entity_stack_base]
    mov [state + State.entity_stack], rax
    mov r13, rax
    ret

;-----------------------------------------------------------------------------
; stack_entity_lookup - Look up a name through entity stack
; Input: rdi = pointer to name (length-prefixed string)
; Output: rax = pointer to Value if found, 0 if not found
;
; Searches from top of entity stack downward for an attribute
; matching the given name
;-----------------------------------------------------------------------------
global stack_entity_lookup
stack_entity_lookup:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r14

    mov rbx, rdi            ; Save name pointer

    ; Get stack depth
    call stack_entity_depth
    test rax, rax
    jz .not_found

    mov r14, rax            ; Number of entities to check
    xor r12, r12            ; Current index

.check_entity:
    ; Get entity at index r12
    mov rdi, r12
    call stack_entity_peek_n
    test rax, rax
    jz .next_entity

    ; rax = pointer to entity Value
    ; Get entity structure pointer
    mov rax, [rax + VALUE_PTR_OFF]
    test rax, rax
    jz .next_entity

    ; Search for attribute in this entity
    ; Entity structure: [type_ptr:8][count:8][cap:8][attrs_ptr:8]
    mov rcx, [rax + 8]      ; Attribute count
    mov rdx, [rax + 24]     ; Attributes array pointer
    test rcx, rcx
    jz .next_entity

    ; Each attribute: [name_ptr:8][value:24]
.check_attr:
    ; Compare names
    mov rsi, [rdx]          ; Attribute name pointer
    test rsi, rsi
    jz .next_attr

    ; Compare length-prefixed strings
    push rcx
    push rdx

    mov rdi, rbx            ; Search name
    ; rsi already has attribute name
    call string_compare

    pop rdx
    pop rcx

    test eax, eax
    jz .found               ; Names match

.next_attr:
    add rdx, 32             ; Move to next attribute slot
    dec rcx
    jnz .check_attr

.next_entity:
    inc r12
    cmp r12, r14
    jb .check_entity

.not_found:
    xor eax, eax
    jmp .done

.found:
    ; rdx points to attribute entry, value is at offset 8
    lea rax, [rdx + 8]

.done:
    pop r14
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; string_compare - Compare two length-prefixed strings
; Input: rdi = string 1, rsi = string 2
; Output: rax = 0 if equal, non-zero otherwise
;-----------------------------------------------------------------------------
string_compare:
    push rbp
    mov rbp, rsp

    ; Compare lengths
    mov rax, [rdi]          ; len1
    mov rcx, [rsi]          ; len2
    cmp rax, rcx
    jne .not_equal

    ; Compare data
    add rdi, 8
    add rsi, 8
    mov rcx, rax            ; Length to compare

    test rcx, rcx
    jz .equal

.cmp_loop:
    mov al, [rdi]
    cmp al, [rsi]
    jne .not_equal
    inc rdi
    inc rsi
    dec rcx
    jnz .cmp_loop

.equal:
    xor eax, eax
    pop rbp
    ret

.not_equal:
    mov eax, 1
    pop rbp
    ret

;-----------------------------------------------------------------------------
; stack_entity_define - Define a variable in current entity context
; Input: rdi = name pointer (length-prefixed), rsi = pointer to Value
; Returns: 0 on success, non-zero on error
;
; Defines the variable in the topmost entity on the stack
;-----------------------------------------------------------------------------
global stack_entity_define
stack_entity_define:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; Save name
    mov r12, rsi            ; Save value

    ; Get top entity
    call stack_entity_peek
    test rax, rax
    jz .no_entity

    ; rax = entity Value pointer
    mov r13, [rax + VALUE_PTR_OFF]  ; Entity structure
    test r13, r13
    jz .no_entity

    ; Check if attribute already exists
    mov rcx, [r13 + 8]      ; Attribute count
    mov rdx, [r13 + 24]     ; Attributes array

.check_existing:
    test rcx, rcx
    jz .add_new

    mov rsi, [rdx]          ; Attribute name
    test rsi, rsi
    jz .next_slot

    ; Compare names
    push rcx
    push rdx
    mov rdi, rbx
    call string_compare
    pop rdx
    pop rcx

    test eax, eax
    jz .update_existing

.next_slot:
    add rdx, 32
    dec rcx
    jmp .check_existing

.update_existing:
    ; rdx points to attribute entry, update value at offset 8
    lea rdi, [rdx + 8]
    mov rsi, r12
    mov ecx, VALUE_SIZE
    rep movsb
    xor eax, eax
    jmp .done

.add_new:
    ; Check if we have capacity
    mov rcx, [r13 + 8]      ; Count
    cmp rcx, [r13 + 16]     ; Capacity
    jae .need_resize

    ; Add new attribute at end
    mov rdx, [r13 + 24]     ; Attrs pointer
    imul rax, rcx, 32
    add rdx, rax            ; Point to new slot

    ; Store name pointer
    mov [rdx], rbx

    ; Copy value
    lea rdi, [rdx + 8]
    mov rsi, r12
    mov ecx, VALUE_SIZE
    rep movsb

    ; Increment count
    inc qword [r13 + 8]

    xor eax, eax
    jmp .done

.need_resize:
    ; TODO: Implement entity resize
    mov eax, ERR_OUT_OF_MEMORY
    jmp .done

.no_entity:
    mov eax, ERR_STACK_UNDERFLOW

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; stack_entity_xdefine - Define variable at stack depth
; Input: rdi = name pointer, rsi = value pointer, rdx = depth (0 = top)
; Returns: 0 on success
;
; Like define but can target any entity on the stack
;-----------------------------------------------------------------------------
global stack_entity_xdefine
stack_entity_xdefine:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; Save name
    mov r12, rsi            ; Save value
    mov r13, rdx            ; Save depth

    ; Get entity at depth
    mov rdi, r13
    call stack_entity_peek_n
    test rax, rax
    jz .no_entity

    ; Now define in this entity
    ; For simplicity, push it to top, define, then restore
    ; Actually, let's just do the same logic as define but with this entity

    mov rdi, [rax + VALUE_PTR_OFF]  ; Entity structure
    test rdi, rdi
    jz .no_entity

    ; Store in r13 now
    mov r13, rdi

    ; Same logic as stack_entity_define from here
    mov rcx, [r13 + 8]      ; Attribute count
    mov rdx, [r13 + 24]     ; Attributes array

.check_existing:
    test rcx, rcx
    jz .add_new

    mov rsi, [rdx]          ; Attribute name
    test rsi, rsi
    jz .next_slot

    push rcx
    push rdx
    mov rdi, rbx
    call string_compare
    pop rdx
    pop rcx

    test eax, eax
    jz .update_existing

.next_slot:
    add rdx, 32
    dec rcx
    jmp .check_existing

.update_existing:
    lea rdi, [rdx + 8]
    mov rsi, r12
    mov ecx, VALUE_SIZE
    rep movsb
    xor eax, eax
    jmp .done

.add_new:
    mov rcx, [r13 + 8]
    cmp rcx, [r13 + 16]
    jae .need_resize

    mov rdx, [r13 + 24]
    imul rax, rcx, 32
    add rdx, rax

    mov [rdx], rbx
    lea rdi, [rdx + 8]
    mov rsi, r12
    mov ecx, VALUE_SIZE
    rep movsb

    inc qword [r13 + 8]
    xor eax, eax
    jmp .done

.need_resize:
    mov eax, ERR_OUT_OF_MEMORY
    jmp .done

.no_entity:
    mov eax, ERR_STACK_UNDERFLOW

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret
