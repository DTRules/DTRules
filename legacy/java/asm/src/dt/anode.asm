; anode.asm - Action node execution
; DTRules Assembly Implementation
; ANode represents actions to execute when a condition branch is taken

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state
extern heap_alloc
extern vm_execute

;-----------------------------------------------------------------------------
; ANode structure
; Represents an action node in the decision tree
;-----------------------------------------------------------------------------
struc ANode
    .actions:       resq 1  ; Bytecode for actions
    .actions_len:   resq 1  ; Length of actions bytecode
    .column:        resq 1  ; Column number (for debugging)
    .next:          resq 1  ; Next ANode in chain (for multiple columns)
endstruc

section .text

;-----------------------------------------------------------------------------
; anode_alloc - Allocate a new action node
; Output: rax = pointer to ANode
;-----------------------------------------------------------------------------
global anode_alloc
anode_alloc:
    push rbp
    mov rbp, rsp

    mov rdi, ANode_size
    call heap_alloc
    test rax, rax
    jz .done

    ; Initialize
    mov qword [rax + ANode.actions], 0
    mov qword [rax + ANode.actions_len], 0
    mov qword [rax + ANode.column], 0
    mov qword [rax + ANode.next], 0

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; anode_set_actions - Set action bytecode
; Input: rdi = ANode pointer, rsi = bytecode pointer, rdx = length
;-----------------------------------------------------------------------------
global anode_set_actions
anode_set_actions:
    mov [rdi + ANode.actions], rsi
    mov [rdi + ANode.actions_len], rdx
    ret

;-----------------------------------------------------------------------------
; anode_set_column - Set column number
; Input: rdi = ANode pointer, rsi = column number
;-----------------------------------------------------------------------------
global anode_set_column
anode_set_column:
    mov [rdi + ANode.column], rsi
    ret

;-----------------------------------------------------------------------------
; anode_set_next - Set next ANode in chain
; Input: rdi = ANode pointer, rsi = next ANode pointer
;-----------------------------------------------------------------------------
global anode_set_next
anode_set_next:
    mov [rdi + ANode.next], rsi
    ret

;-----------------------------------------------------------------------------
; anode_execute - Execute an action node
; Input: rdi = ANode pointer
; Output: rax = 0 on success, non-zero on error
;
; Executes the actions, then follows to next ANode if present
;-----------------------------------------------------------------------------
global anode_execute
anode_execute:
    push rbp
    mov rbp, rsp
    push rbx

    mov rbx, rdi            ; Save ANode pointer

.execute_current:
    ; Check if we have actions
    mov rax, [rbx + ANode.actions]
    test rax, rax
    jz .check_next

    ; Set up bytecode for execution
    mov [state + State.bytecode], rax
    add rax, [rbx + ANode.actions_len]
    mov [state + State.bytecode_end], rax

    ; Execute actions
    call vm_execute
    test eax, eax
    jnz .error

.check_next:
    ; Check for next ANode
    mov rbx, [rbx + ANode.next]
    test rbx, rbx
    jnz .execute_current

    xor eax, eax
    jmp .done

.error:
    ; Error already set in state

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; anode_add_action - Add action bytecode to ANode
; Input: rdi = ANode pointer, rsi = bytecode pointer, rdx = length
; Output: rax = 0 on success, non-zero on error
;
; Note: This is a simple implementation that just sets the actions.
; For multiple actions, use chained ANodes.
;-----------------------------------------------------------------------------
global anode_add_action
anode_add_action:
    ; For now, just set the actions (same as anode_set_actions)
    ; A more complete implementation would chain multiple ANodes
    mov [rdi + ANode.actions], rsi
    mov [rdi + ANode.actions_len], rdx
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; anode_chain_length - Get length of ANode chain
; Input: rdi = ANode pointer
; Output: rax = number of nodes in chain
;-----------------------------------------------------------------------------
global anode_chain_length
anode_chain_length:
    xor eax, eax

.loop:
    test rdi, rdi
    jz .done
    inc rax
    mov rdi, [rdi + ANode.next]
    jmp .loop

.done:
    ret
