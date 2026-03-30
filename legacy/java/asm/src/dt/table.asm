; table.asm - Decision table execution
; DTRules Assembly Implementation
; Orchestrates decision table evaluation using CNode/ANode trees

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state
extern heap_alloc
extern cnode_execute
extern anode_execute
extern stack_entity_push
extern stack_entity_pop

;-----------------------------------------------------------------------------
; DecisionTable structure
;-----------------------------------------------------------------------------
struc DecisionTable
    .name:          resq 1  ; Table name (length-prefixed string)
    .root:          resq 1  ; Root CNode of decision tree
    .root_is_cnode: resb 1  ; 1 if root is CNode, 0 if ANode
    .pad:           resb 7  ; Padding
    .contexts:      resq 1  ; Array of context entity type names
    .context_count: resq 1  ; Number of contexts
    .columns:       resq 1  ; Number of columns (action sets)
endstruc

;-----------------------------------------------------------------------------
; Table registry
;-----------------------------------------------------------------------------
section .bss
    align 8
    table_registry: resq 256    ; Up to 256 tables
    table_count:    resq 1

section .text

;-----------------------------------------------------------------------------
; table_alloc - Allocate a new decision table
; Output: rax = pointer to DecisionTable
;-----------------------------------------------------------------------------
global table_alloc
table_alloc:
    push rbp
    mov rbp, rsp

    mov rdi, DecisionTable_size
    call heap_alloc
    test rax, rax
    jz .done

    ; Initialize
    mov qword [rax + DecisionTable.name], 0
    mov qword [rax + DecisionTable.root], 0
    mov byte [rax + DecisionTable.root_is_cnode], 1
    mov qword [rax + DecisionTable.contexts], 0
    mov qword [rax + DecisionTable.context_count], 0
    mov qword [rax + DecisionTable.columns], 0

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; table_set_name - Set table name
; Input: rdi = table pointer, rsi = name string
;-----------------------------------------------------------------------------
global table_set_name
table_set_name:
    mov [rdi + DecisionTable.name], rsi
    ret

;-----------------------------------------------------------------------------
; table_set_root - Set root node
; Input: rdi = table pointer, rsi = root node, dl = is_cnode
;-----------------------------------------------------------------------------
global table_set_root
table_set_root:
    mov [rdi + DecisionTable.root], rsi
    mov [rdi + DecisionTable.root_is_cnode], dl
    ret

;-----------------------------------------------------------------------------
; table_register - Register table in global registry
; Input: rdi = table pointer
; Output: rax = 0 on success
;-----------------------------------------------------------------------------
global table_register
table_register:
    mov rax, [table_count]
    cmp rax, 256
    jae .full

    lea rcx, [table_registry]
    mov [rcx + rax * 8], rdi
    inc qword [table_count]
    xor eax, eax
    ret

.full:
    mov eax, ERR_OUT_OF_MEMORY
    ret

;-----------------------------------------------------------------------------
; table_find - Find table by name
; Input: rdi = name string
; Output: rax = table pointer, or 0 if not found
;-----------------------------------------------------------------------------
global table_find
table_find:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    mov r12, rdi            ; Name to find

    mov rcx, [table_count]
    lea rbx, [table_registry]
    xor edx, edx

.search:
    cmp rdx, rcx
    jae .not_found

    mov rax, [rbx + rdx * 8]
    test rax, rax
    jz .next

    ; Compare names
    push rcx
    push rdx
    push rax

    mov rdi, [rax + DecisionTable.name]
    mov rsi, r12

    ; Length comparison
    mov rax, [rdi]
    cmp rax, [rsi]
    jne .next_pop

    ; Data comparison
    mov rcx, rax
    add rdi, 8
    add rsi, 8
    repe cmpsb
    jne .next_pop

    ; Found
    pop rax
    add rsp, 16
    jmp .done

.next_pop:
    pop rax
    pop rdx
    pop rcx

.next:
    inc rdx
    jmp .search

.not_found:
    xor eax, eax

.done:
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; table_execute - Execute a decision table
; Input: rdi = table pointer
; Output: rax = 0 on success, non-zero on error
;
; Executes the decision tree starting from the root node.
;-----------------------------------------------------------------------------
global table_execute
table_execute:
    push rbp
    mov rbp, rsp
    push rbx

    mov rbx, rdi            ; Save table pointer

    ; Get root node
    mov rdi, [rbx + DecisionTable.root]
    test rdi, rdi
    jz .no_root

    ; Check if CNode or ANode
    cmp byte [rbx + DecisionTable.root_is_cnode], 0
    je .exec_anode

    ; Execute CNode tree
    call cnode_execute
    jmp .done

.exec_anode:
    ; Execute ANode directly
    call anode_execute
    jmp .done

.no_root:
    ; No root = nothing to do
    xor eax, eax

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; table_execute_by_name - Execute table by name
; Input: rdi = table name string
; Output: rax = 0 on success, non-zero on error
;-----------------------------------------------------------------------------
global table_execute_by_name
table_execute_by_name:
    push rbp
    mov rbp, rsp

    ; Find table
    call table_find
    test rax, rax
    jz .not_found

    ; Execute it
    mov rdi, rax
    call table_execute
    jmp .done

.not_found:
    mov dword [state + State.error], ERR_NAME_NOT_FOUND
    mov eax, ERR_NAME_NOT_FOUND

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; table_get_count - Get number of registered tables
; Output: rax = count
;-----------------------------------------------------------------------------
global table_get_count
table_get_count:
    mov rax, [table_count]
    ret

;-----------------------------------------------------------------------------
; table_get_by_index - Get table by registry index
; Input: rdi = index
; Output: rax = table pointer, or 0 if invalid
;-----------------------------------------------------------------------------
global table_get_by_index
table_get_by_index:
    cmp rdi, [table_count]
    jae .invalid

    lea rax, [table_registry]
    mov rax, [rax + rdi * 8]
    ret

.invalid:
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; table_clear_registry - Clear all registered tables
;-----------------------------------------------------------------------------
global table_clear_registry
table_clear_registry:
    mov qword [table_count], 0

    ; Zero the registry
    lea rdi, [table_registry]
    mov rcx, 256 * 8
    xor al, al
    rep stosb

    ret
