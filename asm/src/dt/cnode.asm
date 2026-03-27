; cnode.asm - Condition node execution
; DTRules Assembly Implementation
; CNode represents a condition in a decision tree

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state
extern heap_alloc
extern vm_execute
extern stack_data_pop
extern value_is_truthy

;-----------------------------------------------------------------------------
; CNode structure
; Represents a condition node in the decision tree
;-----------------------------------------------------------------------------
struc CNode
    .condition:     resq 1  ; Bytecode for condition expression
    .cond_len:      resq 1  ; Length of condition bytecode
    .true_branch:   resq 1  ; Pointer to node for true result
    .false_branch:  resq 1  ; Pointer to node for false result
    .true_is_cnode: resb 1  ; 1 if true_branch is CNode, 0 if ANode
    .false_is_cnode: resb 1 ; 1 if false_branch is CNode, 0 if ANode
    .pad:           resb 6  ; Padding
endstruc

section .text

;-----------------------------------------------------------------------------
; cnode_alloc - Allocate a new condition node
; Output: rax = pointer to CNode
;-----------------------------------------------------------------------------
global cnode_alloc
cnode_alloc:
    push rbp
    mov rbp, rsp

    mov rdi, CNode_size
    call heap_alloc
    test rax, rax
    jz .done

    ; Initialize to zeros
    mov qword [rax + CNode.condition], 0
    mov qword [rax + CNode.cond_len], 0
    mov qword [rax + CNode.true_branch], 0
    mov qword [rax + CNode.false_branch], 0
    mov byte [rax + CNode.true_is_cnode], 0
    mov byte [rax + CNode.false_is_cnode], 0

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; cnode_set_condition - Set condition bytecode
; Input: rdi = CNode pointer, rsi = bytecode pointer, rdx = length
;-----------------------------------------------------------------------------
global cnode_set_condition
cnode_set_condition:
    mov [rdi + CNode.condition], rsi
    mov [rdi + CNode.cond_len], rdx
    ret

;-----------------------------------------------------------------------------
; cnode_set_true_branch - Set true branch
; Input: rdi = CNode pointer, rsi = branch node pointer, dl = is_cnode
;-----------------------------------------------------------------------------
global cnode_set_true_branch
cnode_set_true_branch:
    mov [rdi + CNode.true_branch], rsi
    mov [rdi + CNode.true_is_cnode], dl
    ret

;-----------------------------------------------------------------------------
; cnode_set_false_branch - Set false branch
; Input: rdi = CNode pointer, rsi = branch node pointer, dl = is_cnode
;-----------------------------------------------------------------------------
global cnode_set_false_branch
cnode_set_false_branch:
    mov [rdi + CNode.false_branch], rsi
    mov [rdi + CNode.false_is_cnode], dl
    ret

;-----------------------------------------------------------------------------
; cnode_execute - Execute a condition node
; Input: rdi = CNode pointer
; Output: rax = 0 on success, non-zero on error
;
; Evaluates the condition, then follows the appropriate branch.
; If branch is a CNode, recursively executes it.
; If branch is an ANode, executes the actions.
;-----------------------------------------------------------------------------
global cnode_execute
cnode_execute:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    mov rbx, rdi            ; Save CNode pointer

    ; Set up bytecode for execution
    mov rax, [rbx + CNode.condition]
    test rax, rax
    jz .no_condition        ; No condition = always true

    mov [state + State.bytecode], rax
    add rax, [rbx + CNode.cond_len]
    mov [state + State.bytecode_end], rax

    ; Execute condition
    call vm_execute
    test eax, eax
    jnz .error

    ; Pop result from stack
    call stack_data_pop
    test rax, rax
    jz .error

    ; Check if truthy
    mov rdi, rax
    call value_is_truthy

    test eax, eax
    jz .false_branch

.true_branch:
    mov r12, [rbx + CNode.true_branch]
    test r12, r12
    jz .done                ; No branch = done

    ; Check if CNode or ANode
    cmp byte [rbx + CNode.true_is_cnode], 0
    je .exec_true_anode

    ; Execute CNode
    mov rdi, r12
    call cnode_execute
    jmp .done

.exec_true_anode:
    ; Execute ANode
    mov rdi, r12
    call anode_execute
    jmp .done

.false_branch:
    mov r12, [rbx + CNode.false_branch]
    test r12, r12
    jz .done                ; No branch = done

    cmp byte [rbx + CNode.false_is_cnode], 0
    je .exec_false_anode

    mov rdi, r12
    call cnode_execute
    jmp .done

.exec_false_anode:
    mov rdi, r12
    call anode_execute
    jmp .done

.no_condition:
    ; No condition means always take true branch
    jmp .true_branch

.error:
    mov eax, [state + State.error]
    test eax, eax
    jnz .done
    mov eax, ERR_TYPE_MISMATCH

.done:
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; External reference to anode_execute
;-----------------------------------------------------------------------------
extern anode_execute
