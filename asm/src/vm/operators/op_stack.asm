; op_stack.asm - Stack manipulation operators
; DTRules Assembly Implementation
; Additional stack operators beyond basic push/pop

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"

extern state
extern stack_data_push
extern stack_data_pop
extern stack_data_peek
extern stack_data_peek_n
extern stack_data_depth
extern stack_data_dup
extern stack_data_swap
extern stack_data_rot
extern stack_data_over
extern stack_data_pick
extern stack_data_roll
extern stack_data_drop
extern stack_data_drop_n
extern stack_data_clear
extern value_copy

section .text

;-----------------------------------------------------------------------------
; op_count - Push stack depth onto stack
;-----------------------------------------------------------------------------
global op_count
op_count:
    push rbp
    mov rbp, rsp

    call stack_data_depth

    ; Push as integer
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_INTEGER
    mov [r12 + VALUE_NUM_OFF], rax
    mov qword [r12 + VALUE_PTR_OFF], 0

    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_index - Get element at index (0 = top)
; Stack: index -- value
;-----------------------------------------------------------------------------
global op_index
op_index:
    push rbp
    mov rbp, rsp

    ; Pop index
    call stack_data_pop
    test rax, rax
    jz .error

    ; Check it's an integer
    cmp byte [rax + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error

    mov rdi, [rax + VALUE_NUM_OFF]

    ; Get element at index
    call stack_data_peek_n
    test rax, rax
    jz .error

    ; Push copy
    mov rdi, rax
    call stack_data_push

    xor eax, eax
    jmp .done

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    mov eax, [state + State.error]

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_copy - Copy n elements from stack
; Stack: n -- copies
;-----------------------------------------------------------------------------
global op_copy
op_copy:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    ; Pop count
    call stack_data_pop
    test rax, rax
    jz .error

    cmp byte [rax + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error

    mov r12, [rax + VALUE_NUM_OFF]  ; Count

    ; Check we have enough elements
    call stack_data_depth
    cmp r12, rax
    ja .underflow

    ; Copy elements (from bottom to top to preserve order)
    xor ebx, ebx

.copy_loop:
    cmp rbx, r12
    jae .done_success

    ; Get element at index (r12 - 1 - rbx) from where we started
    mov rdi, r12
    dec rdi
    sub rdi, rbx
    add rdi, rbx            ; Adjust for elements we've pushed
    call stack_data_peek_n
    test rax, rax
    jz .error

    mov rdi, rax
    call stack_data_push

    inc rbx
    jmp .copy_loop

.done_success:
    xor eax, eax
    jmp .done

.underflow:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    jmp .error

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH

.error:
    mov eax, [state + State.error]

.done:
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_exch - Exchange top two elements (alias for swap)
;-----------------------------------------------------------------------------
global op_exch
op_exch:
    jmp stack_data_swap

;-----------------------------------------------------------------------------
; op_dup2 - Duplicate top two elements
; Stack: a b -- a b a b
;-----------------------------------------------------------------------------
global op_dup2
op_dup2:
    push rbp
    mov rbp, rsp

    ; Need at least 2 elements
    call stack_data_depth
    cmp rax, 2
    jb .underflow

    ; Copy element at index 1 (second from top)
    mov rdi, 1
    call stack_data_peek_n
    mov rdi, rax
    call stack_data_push

    ; Copy element at index 1 (was 0, now moved)
    mov rdi, 1
    call stack_data_peek_n
    mov rdi, rax
    call stack_data_push

    xor eax, eax
    jmp .done

.underflow:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    mov eax, ERR_STACK_UNDERFLOW

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_nip - Remove second element
; Stack: a b -- b
;-----------------------------------------------------------------------------
global op_nip
op_nip:
    push rbp
    mov rbp, rsp

    call stack_data_depth
    cmp rax, 2
    jb .underflow

    ; Swap then drop
    call stack_data_swap
    call stack_data_drop

    xor eax, eax
    jmp .done

.underflow:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    mov eax, ERR_STACK_UNDERFLOW

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_tuck - Tuck top under second
; Stack: a b -- b a b
;-----------------------------------------------------------------------------
global op_tuck
op_tuck:
    push rbp
    mov rbp, rsp

    call stack_data_depth
    cmp rax, 2
    jb .underflow

    ; dup, then rot twice
    call stack_data_dup
    call stack_data_rot
    call stack_data_rot

    xor eax, eax
    jmp .done

.underflow:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    mov eax, ERR_STACK_UNDERFLOW

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_2swap - Swap top two pairs
; Stack: a b c d -- c d a b
;-----------------------------------------------------------------------------
global op_2swap
op_2swap:
    push rbp
    mov rbp, rsp

    call stack_data_depth
    cmp rax, 4
    jb .underflow

    ; Roll 3, roll 3
    mov rdi, 3
    call stack_data_roll
    mov rdi, 3
    call stack_data_roll

    xor eax, eax
    jmp .done

.underflow:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    mov eax, ERR_STACK_UNDERFLOW

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_2over - Copy second pair over top
; Stack: a b c d -- a b c d a b
;-----------------------------------------------------------------------------
global op_2over
op_2over:
    push rbp
    mov rbp, rsp

    call stack_data_depth
    cmp rax, 4
    jb .underflow

    ; pick 3, pick 3
    mov rdi, 3
    call stack_data_pick
    mov rdi, 3
    call stack_data_pick

    xor eax, eax
    jmp .done

.underflow:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    mov eax, ERR_STACK_UNDERFLOW

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_2drop - Drop top two elements
; Stack: a b --
;-----------------------------------------------------------------------------
global op_2drop
op_2drop:
    mov rdi, 2
    jmp stack_data_drop_n

;-----------------------------------------------------------------------------
; op_cleartomark - Clear stack down to a mark
; Not implemented - would need mark support
;-----------------------------------------------------------------------------
global op_cleartomark
op_cleartomark:
    ; For now, just clear all
    jmp stack_data_clear
