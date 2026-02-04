; stack_data.asm - Data stack operations
; DTRules Assembly Implementation
; Main value stack for the VM

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state
extern value_copy

section .text

;-----------------------------------------------------------------------------
; stack_data_init - Initialize the data stack
; Notes: Memory already allocated in memory_init, this just resets pointers
;-----------------------------------------------------------------------------
global stack_data_init
stack_data_init:
    ; Reset stack pointer to base (empty stack)
    mov rax, [state + State.data_stack_base]
    mov [state + State.data_stack], rax
    mov r12, rax            ; Also update register
    ret

;-----------------------------------------------------------------------------
; stack_data_push - Push a Value onto the data stack
; Input: rdi = pointer to Value to push (copied onto stack)
; Returns: 0 on success, ERR_STACK_OVERFLOW on error
;-----------------------------------------------------------------------------
global stack_data_push
stack_data_push:
    push rbp
    mov rbp, rsp
    push rbx

    ; Check for overflow
    mov rax, r12
    sub rax, VALUE_SIZE
    cmp rax, [state + State.data_stack_end]
    jb .overflow

    ; Move stack pointer
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12

    ; Copy Value onto stack
    mov rbx, rdi            ; Source
    mov rdi, r12            ; Destination
    mov rsi, rbx
    mov ecx, VALUE_SIZE
    rep movsb

    xor eax, eax            ; Success
    jmp .done

.overflow:
    mov dword [state + State.error], ERR_STACK_OVERFLOW
    mov eax, ERR_STACK_OVERFLOW

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; stack_data_push_integer - Push an integer directly
; Input: rdi = integer value
; Returns: 0 on success
;-----------------------------------------------------------------------------
global stack_data_push_integer
stack_data_push_integer:
    push rbp
    mov rbp, rsp
    push rbx

    mov rbx, rdi            ; Save integer

    ; Check for overflow
    mov rax, r12
    sub rax, VALUE_SIZE
    cmp rax, [state + State.data_stack_end]
    jb .overflow

    ; Move stack pointer
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12

    ; Create integer value in place
    mov byte [r12 + VALUE_TAG_OFF], VTAG_INTEGER
    mov qword [r12 + VALUE_NUM_OFF], rbx
    mov qword [r12 + VALUE_PTR_OFF], 0

    xor eax, eax
    jmp .done

.overflow:
    mov dword [state + State.error], ERR_STACK_OVERFLOW
    mov eax, ERR_STACK_OVERFLOW

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; stack_data_push_boolean - Push a boolean directly
; Input: rdi = boolean (0 or non-zero)
; Returns: 0 on success
;-----------------------------------------------------------------------------
global stack_data_push_boolean
stack_data_push_boolean:
    push rbp
    mov rbp, rsp

    ; Normalize boolean
    test rdi, rdi
    setnz dil
    movzx edi, dil

    ; Check for overflow
    mov rax, r12
    sub rax, VALUE_SIZE
    cmp rax, [state + State.data_stack_end]
    jb .overflow

    ; Move stack pointer
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12

    ; Create boolean value in place
    mov byte [r12 + VALUE_TAG_OFF], VTAG_BOOLEAN
    mov [r12 + VALUE_NUM_OFF], rdi
    mov qword [r12 + VALUE_PTR_OFF], 0

    xor eax, eax
    jmp .done

.overflow:
    mov dword [state + State.error], ERR_STACK_OVERFLOW
    mov eax, ERR_STACK_OVERFLOW

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; stack_data_push_null - Push null value
; Returns: 0 on success
;-----------------------------------------------------------------------------
global stack_data_push_null
stack_data_push_null:
    ; Check for overflow
    mov rax, r12
    sub rax, VALUE_SIZE
    cmp rax, [state + State.data_stack_end]
    jb .overflow

    ; Move stack pointer
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12

    ; Create null value in place
    mov byte [r12 + VALUE_TAG_OFF], VTAG_NULL
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov qword [r12 + VALUE_PTR_OFF], 0

    xor eax, eax
    ret

.overflow:
    mov dword [state + State.error], ERR_STACK_OVERFLOW
    mov eax, ERR_STACK_OVERFLOW
    ret

;-----------------------------------------------------------------------------
; stack_data_push_double - Push a double directly
; Input: xmm0 = double value
; Returns: 0 on success
;-----------------------------------------------------------------------------
global stack_data_push_double
stack_data_push_double:
    ; Check for overflow
    mov rax, r12
    sub rax, VALUE_SIZE
    cmp rax, [state + State.data_stack_end]
    jb .overflow

    ; Move stack pointer
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12

    ; Create double value in place
    mov byte [r12 + VALUE_TAG_OFF], VTAG_DOUBLE
    movsd [r12 + VALUE_NUM_OFF], xmm0
    mov qword [r12 + VALUE_PTR_OFF], 0

    xor eax, eax
    ret

.overflow:
    mov dword [state + State.error], ERR_STACK_OVERFLOW
    mov eax, ERR_STACK_OVERFLOW
    ret

;-----------------------------------------------------------------------------
; stack_data_pop - Pop a Value from the data stack
; Output: rax = pointer to popped Value (still on stack, valid until next push)
; Returns: pointer on success, 0 on underflow
;-----------------------------------------------------------------------------
global stack_data_pop
stack_data_pop:
    ; Check for underflow
    cmp r12, [state + State.data_stack_base]
    jae .underflow

    ; Return pointer to current top
    mov rax, r12

    ; Move stack pointer up
    add r12, VALUE_SIZE
    mov [state + State.data_stack], r12

    ret

.underflow:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; stack_data_peek - Peek at top Value without removing
; Output: rax = pointer to top Value
; Returns: pointer on success, 0 on empty
;-----------------------------------------------------------------------------
global stack_data_peek
stack_data_peek:
    ; Check for empty
    cmp r12, [state + State.data_stack_base]
    jae .empty

    mov rax, r12
    ret

.empty:
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; stack_data_peek_n - Peek at nth Value from top (0 = top)
; Input: rdi = index from top
; Output: rax = pointer to Value
; Returns: pointer on success, 0 if not enough elements
;-----------------------------------------------------------------------------
global stack_data_peek_n
stack_data_peek_n:
    ; Calculate address
    imul rdi, VALUE_SIZE
    lea rax, [r12 + rdi]

    ; Check bounds
    cmp rax, [state + State.data_stack_base]
    jae .invalid

    ret

.invalid:
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; stack_data_depth - Get number of Values on stack
; Output: rax = depth
;-----------------------------------------------------------------------------
global stack_data_depth
stack_data_depth:
    mov rax, [state + State.data_stack_base]
    sub rax, r12
    mov rcx, VALUE_SIZE
    xor edx, edx
    div rcx                 ; rax = depth
    ret

;-----------------------------------------------------------------------------
; stack_data_dup - Duplicate top Value
; Returns: 0 on success
;-----------------------------------------------------------------------------
global stack_data_dup
stack_data_dup:
    push rbp
    mov rbp, rsp

    ; Check not empty
    cmp r12, [state + State.data_stack_base]
    jae .underflow

    ; Check overflow
    mov rax, r12
    sub rax, VALUE_SIZE
    cmp rax, [state + State.data_stack_end]
    jb .overflow

    ; Copy top value
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12

    mov rdi, r12            ; Destination
    lea rsi, [r12 + VALUE_SIZE]  ; Source (old top)
    mov ecx, VALUE_SIZE
    rep movsb

    xor eax, eax
    jmp .done

.underflow:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    mov eax, ERR_STACK_UNDERFLOW
    jmp .done

.overflow:
    mov dword [state + State.error], ERR_STACK_OVERFLOW
    mov eax, ERR_STACK_OVERFLOW

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; stack_data_swap - Swap top two Values
; Returns: 0 on success
;-----------------------------------------------------------------------------
global stack_data_swap
stack_data_swap:
    push rbp
    mov rbp, rsp
    sub rsp, VALUE_SIZE     ; Temp space

    ; Need at least 2 elements
    call stack_data_depth
    cmp rax, 2
    jb .underflow

    ; Swap using temp space
    ; temp = stack[0]
    mov rdi, rsp
    mov rsi, r12
    mov ecx, VALUE_SIZE
    rep movsb

    ; stack[0] = stack[1]
    mov rdi, r12
    lea rsi, [r12 + VALUE_SIZE]
    mov ecx, VALUE_SIZE
    rep movsb

    ; stack[1] = temp
    lea rdi, [r12 + VALUE_SIZE]
    mov rsi, rsp
    mov ecx, VALUE_SIZE
    rep movsb

    xor eax, eax
    jmp .done

.underflow:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    mov eax, ERR_STACK_UNDERFLOW

.done:
    add rsp, VALUE_SIZE
    pop rbp
    ret

;-----------------------------------------------------------------------------
; stack_data_rot - Rotate top 3 Values (a b c -- b c a)
; Returns: 0 on success
;-----------------------------------------------------------------------------
global stack_data_rot
stack_data_rot:
    push rbp
    mov rbp, rsp
    sub rsp, VALUE_SIZE     ; Temp space

    ; Need at least 3 elements
    call stack_data_depth
    cmp rax, 3
    jb .underflow

    ; temp = stack[2] (deepest of 3)
    mov rdi, rsp
    lea rsi, [r12 + 2*VALUE_SIZE]
    mov ecx, VALUE_SIZE
    rep movsb

    ; stack[2] = stack[1]
    lea rdi, [r12 + 2*VALUE_SIZE]
    lea rsi, [r12 + VALUE_SIZE]
    mov ecx, VALUE_SIZE
    rep movsb

    ; stack[1] = stack[0]
    lea rdi, [r12 + VALUE_SIZE]
    mov rsi, r12
    mov ecx, VALUE_SIZE
    rep movsb

    ; stack[0] = temp
    mov rdi, r12
    mov rsi, rsp
    mov ecx, VALUE_SIZE
    rep movsb

    xor eax, eax
    jmp .done

.underflow:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    mov eax, ERR_STACK_UNDERFLOW

.done:
    add rsp, VALUE_SIZE
    pop rbp
    ret

;-----------------------------------------------------------------------------
; stack_data_over - Copy second Value to top (a b -- a b a)
; Returns: 0 on success
;-----------------------------------------------------------------------------
global stack_data_over
stack_data_over:
    push rbp
    mov rbp, rsp

    ; Need at least 2 elements
    call stack_data_depth
    cmp rax, 2
    jb .underflow

    ; Check overflow
    mov rax, r12
    sub rax, VALUE_SIZE
    cmp rax, [state + State.data_stack_end]
    jb .overflow

    ; Copy stack[1] to new top
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12

    mov rdi, r12
    lea rsi, [r12 + 2*VALUE_SIZE]  ; Old stack[1]
    mov ecx, VALUE_SIZE
    rep movsb

    xor eax, eax
    jmp .done

.underflow:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    mov eax, ERR_STACK_UNDERFLOW
    jmp .done

.overflow:
    mov dword [state + State.error], ERR_STACK_OVERFLOW
    mov eax, ERR_STACK_OVERFLOW

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; stack_data_pick - Copy nth Value to top (0 = dup)
; Input: rdi = index from top
; Returns: 0 on success
;-----------------------------------------------------------------------------
global stack_data_pick
stack_data_pick:
    push rbp
    mov rbp, rsp
    push rbx

    mov rbx, rdi            ; Save index

    ; Check depth
    call stack_data_depth
    cmp rbx, rax
    jae .underflow

    ; Check overflow
    mov rax, r12
    sub rax, VALUE_SIZE
    cmp rax, [state + State.data_stack_end]
    jb .overflow

    ; Copy stack[n] to new top
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12

    mov rdi, r12
    imul rbx, VALUE_SIZE
    lea rsi, [r12 + VALUE_SIZE + rbx]
    mov ecx, VALUE_SIZE
    rep movsb

    xor eax, eax
    jmp .done

.underflow:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    mov eax, ERR_STACK_UNDERFLOW
    jmp .done

.overflow:
    mov dword [state + State.error], ERR_STACK_OVERFLOW
    mov eax, ERR_STACK_OVERFLOW

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; stack_data_roll - Roll nth Value to top, shifting others down
; Input: rdi = index from top (2 = rot)
; Returns: 0 on success
;-----------------------------------------------------------------------------
global stack_data_roll
stack_data_roll:
    push rbp
    mov rbp, rsp
    sub rsp, VALUE_SIZE
    push rbx

    mov rbx, rdi            ; Save index

    ; Check depth
    call stack_data_depth
    cmp rbx, rax
    jae .underflow

    test rbx, rbx
    jz .done_success        ; roll 0 is no-op

    ; Save stack[n]
    lea rdi, [rbp - VALUE_SIZE]
    imul rax, rbx, VALUE_SIZE
    lea rsi, [r12 + rax]
    mov ecx, VALUE_SIZE
    rep movsb

    ; Shift elements down: stack[n] = stack[n-1], ..., stack[1] = stack[0]
    mov rcx, rbx
.shift_loop:
    imul rax, rcx, VALUE_SIZE
    lea rdi, [r12 + rax]            ; Destination: stack[i]
    lea rsi, [r12 + rax - VALUE_SIZE]  ; Source: stack[i-1]
    push rcx
    mov ecx, VALUE_SIZE
    rep movsb
    pop rcx
    dec rcx
    jnz .shift_loop

    ; stack[0] = saved value
    mov rdi, r12
    lea rsi, [rbp - VALUE_SIZE]
    mov ecx, VALUE_SIZE
    rep movsb

.done_success:
    xor eax, eax
    jmp .done

.underflow:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    mov eax, ERR_STACK_UNDERFLOW

.done:
    pop rbx
    add rsp, VALUE_SIZE
    pop rbp
    ret

;-----------------------------------------------------------------------------
; stack_data_clear - Clear the data stack
;-----------------------------------------------------------------------------
global stack_data_clear
stack_data_clear:
    mov rax, [state + State.data_stack_base]
    mov [state + State.data_stack], rax
    mov r12, rax
    ret

;-----------------------------------------------------------------------------
; stack_data_drop - Pop and discard top Value
; Returns: 0 on success
;-----------------------------------------------------------------------------
global stack_data_drop
stack_data_drop:
    ; Check for underflow
    cmp r12, [state + State.data_stack_base]
    jae .underflow

    ; Just move pointer
    add r12, VALUE_SIZE
    mov [state + State.data_stack], r12

    xor eax, eax
    ret

.underflow:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    mov eax, ERR_STACK_UNDERFLOW
    ret

;-----------------------------------------------------------------------------
; stack_data_drop_n - Pop and discard top n Values
; Input: rdi = number of values to drop
; Returns: 0 on success
;-----------------------------------------------------------------------------
global stack_data_drop_n
stack_data_drop_n:
    push rbp
    mov rbp, rsp
    push rbx

    mov rbx, rdi

    ; Check depth
    call stack_data_depth
    cmp rbx, rax
    ja .underflow

    ; Move pointer
    imul rbx, VALUE_SIZE
    add r12, rbx
    mov [state + State.data_stack], r12

    xor eax, eax
    jmp .done

.underflow:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    mov eax, ERR_STACK_UNDERFLOW

.done:
    pop rbx
    pop rbp
    ret
