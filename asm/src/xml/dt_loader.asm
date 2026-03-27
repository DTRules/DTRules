; dt_loader.asm - Decision table loader
; DTRules Assembly Implementation
; Loads decision tables from parsed XML

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"

extern state
extern heap_alloc
extern xml_node_find_child
extern xml_node_get_attr
extern xml_node_get_child
extern xml_node_get_text
extern table_alloc
extern table_set_name
extern table_set_root
extern table_register
extern cnode_alloc
extern cnode_set_condition
extern cnode_set_true_branch
extern cnode_set_false_branch
extern anode_alloc
extern anode_set_actions
extern anode_set_next
extern string_alloc

;-----------------------------------------------------------------------------
; Tag name constants (length-prefixed)
;-----------------------------------------------------------------------------
section .data
    align 8
    tag_decision_tables:
        dq 14
        db "decision_tables", 0

    tag_decision_table:
        dq 14
        db "decision_table", 0

    tag_conditions:
        dq 10
        db "conditions", 0

    tag_condition:
        dq 9
        db "condition", 0

    tag_actions:
        dq 7
        db "actions", 0

    tag_action:
        dq 6
        db "action", 0

    tag_columns:
        dq 7
        db "columns", 0

    tag_column:
        dq 6
        db "column", 0

    attr_name:
        dq 4
        db "name", 0

    attr_col:
        dq 3
        db "col", 0

section .text

;-----------------------------------------------------------------------------
; dt_load - Load decision tables from XML root
; Input: rdi = XML root node
; Output: rax = 0 on success, non-zero on error
;-----------------------------------------------------------------------------
global dt_load
dt_load:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; Root node

    ; Find decision_tables element
    mov rsi, tag_decision_tables
    call xml_node_find_child
    test rax, rax
    jz .no_tables           ; No tables defined - not an error

    mov r12, rax            ; decision_tables element

    ; Iterate through decision_table children
    mov r13, [r12 + 56]     ; child_count (XMLNode.child_count offset)
    mov r12, [r12 + 48]     ; children array (XMLNode.children offset)
    xor ebx, ebx            ; Index

.load_loop:
    cmp rbx, r13
    jae .done

    ; Get child
    mov rax, [r12 + rbx * 8]

    ; Check if it's a decision_table element
    cmp dword [rax], 1      ; XML_ELEMENT
    jne .next

    ; Load this decision table
    push rbx
    push r12
    push r13
    mov rdi, rax
    call load_decision_table
    pop r13
    pop r12
    pop rbx

    test eax, eax
    jnz .error

.next:
    inc rbx
    jmp .load_loop

.no_tables:
    xor eax, eax
    jmp .done

.error:
    ; Keep error from load_decision_table

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; load_decision_table - Load a single decision table
; Input: rdi = XML decision_table element
; Output: rax = 0 on success
;-----------------------------------------------------------------------------
load_decision_table:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; XML element

    ; Allocate table
    call table_alloc
    test rax, rax
    jz .alloc_error

    mov r12, rax            ; Table pointer

    ; Get name attribute
    mov rdi, rbx
    mov rsi, attr_name
    call xml_node_get_attr
    test rax, rax
    jz .no_name

    ; Set table name
    mov rdi, r12
    mov rsi, rax
    call table_set_name

.no_name:
    ; Build decision tree from conditions and actions
    ; For simplicity, create a single ANode with all actions

    call anode_alloc
    test rax, rax
    jz .alloc_error

    mov r13, rax            ; ANode

    ; Set as root (it's an ANode, not CNode)
    mov rdi, r12
    mov rsi, r13
    xor dl, dl              ; is_cnode = false
    call table_set_root

    ; TODO: Actually parse conditions into CNode tree
    ; For now, we just register the table structure

    ; Register table
    mov rdi, r12
    call table_register

    xor eax, eax
    jmp .done

.alloc_error:
    mov eax, ERR_OUT_OF_MEMORY

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; compile_condition - Compile a condition expression to bytecode
; Input: rdi = condition text string
; Output: rax = bytecode pointer, rdx = length
;
; TODO: Implement expression compiler
; For now, returns empty bytecode
;-----------------------------------------------------------------------------
compile_condition:
    xor eax, eax
    xor edx, edx
    ret

;-----------------------------------------------------------------------------
; compile_action - Compile an action expression to bytecode
; Input: rdi = action text string
; Output: rax = bytecode pointer, rdx = length
;
; TODO: Implement expression compiler
;-----------------------------------------------------------------------------
compile_action:
    xor eax, eax
    xor edx, edx
    ret
