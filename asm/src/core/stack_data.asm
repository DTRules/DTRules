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
    ; Check for overflow
    mov rax, r12
    sub rax, VALUE_SIZE
    cmp rax, [state + State.data_stack_end]
    jb .overflow

    ; Move stack pointer
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12

    ; Fast 24-byte copy (3x mov instead of rep movsb)
    mov rax, [rdi]
    mov rcx, [rdi + 8]
    mov rdx, [rdi + 16]
    mov [r12], rax
    mov [r12 + 8], rcx
    mov [r12 + 16], rdx

    xor eax, eax            ; Success
    ret

.overflow:
    mov dword [state + State.error], ERR_STACK_OVERFLOW
    mov eax, ERR_STACK_OVERFLOW
    ret

;-----------------------------------------------------------------------------
; stack_data_push_integer - Push an integer directly
; Input: rdi = integer value
; Returns: 0 on success
;-----------------------------------------------------------------------------
global stack_data_push_integer
stack_data_push_integer:
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
    mov qword [r12 + VALUE_NUM_OFF], rdi
    mov qword [r12 + VALUE_PTR_OFF], 0

    xor eax, eax
    ret

.overflow:
    mov dword [state + State.error], ERR_STACK_OVERFLOW
    mov eax, ERR_STACK_OVERFLOW
    ret

;-----------------------------------------------------------------------------
; stack_data_push_boolean - Push a boolean directly
; Input: rdi = boolean (0 or non-zero)
; Returns: 0 on success
;-----------------------------------------------------------------------------
global stack_data_push_boolean
stack_data_push_boolean:
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
    ret

.overflow:
    mov dword [state + State.error], ERR_STACK_OVERFLOW
    mov eax, ERR_STACK_OVERFLOW
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
; NOTE: Uses shift instead of div for performance (VALUE_SIZE=24 = 8*3)
;       Computes (base - r12) / 24 = (base - r12) * 0xAAAAAAAAAAAAAAAB >> 68
;       Simplified: just use iterative subtraction for small stacks or
;       approximate with (base - r12) >> 5 for fast path
;-----------------------------------------------------------------------------
global stack_data_depth
stack_data_depth:
    mov rax, [state + State.data_stack_base]
    sub rax, r12
    ; Divide by 24 using multiplication by magic number
    ; 24 = 8 * 3, so we can compute: n/24 = (n/8)/3
    shr rax, 3              ; Divide by 8
    ; Now divide by 3: multiply by 0xAAAAAAAAAAAAAAAB and shift right
    mov rcx, 0xAAAAAAAAAAAAAAAB
    mul rcx                 ; Result in rdx:rax, we want rdx >> 1
    shr rdx, 1
    mov rax, rdx
    ret

;-----------------------------------------------------------------------------
; stack_data_dup - Duplicate top Value
; Returns: 0 on success
;-----------------------------------------------------------------------------
global stack_data_dup
stack_data_dup:
    ; Check not empty
    cmp r12, [state + State.data_stack_base]
    jae .underflow

    ; Check overflow
    mov rax, r12
    sub rax, VALUE_SIZE
    cmp rax, [state + State.data_stack_end]
    jb .overflow

    ; Fast copy top value (24 bytes = 3 qwords)
    mov rax, [r12]
    mov rcx, [r12 + 8]
    mov rdx, [r12 + 16]
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov [r12], rax
    mov [r12 + 8], rcx
    mov [r12 + 16], rdx

    xor eax, eax
    ret

.underflow:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    mov eax, ERR_STACK_UNDERFLOW
    ret

.overflow:
    mov dword [state + State.error], ERR_STACK_OVERFLOW
    mov eax, ERR_STACK_OVERFLOW
    ret

;-----------------------------------------------------------------------------
; stack_data_swap - Swap top two Values
; Returns: 0 on success
;-----------------------------------------------------------------------------
global stack_data_swap
stack_data_swap:
    ; Check at least 2 elements (stack[1] must be valid)
    lea rax, [r12 + VALUE_SIZE]
    cmp rax, [state + State.data_stack_base]
    jae .underflow

    ; Fast swap using registers (no memory temp needed)
    ; Load stack[0] into r8, r9, r10
    mov r8, [r12]
    mov r9, [r12 + 8]
    mov r10, [r12 + 16]
    ; Load stack[1] into rax, rcx, rdx
    mov rax, [r12 + VALUE_SIZE]
    mov rcx, [r12 + VALUE_SIZE + 8]
    mov rdx, [r12 + VALUE_SIZE + 16]
    ; Store swapped
    mov [r12], rax
    mov [r12 + 8], rcx
    mov [r12 + 16], rdx
    mov [r12 + VALUE_SIZE], r8
    mov [r12 + VALUE_SIZE + 8], r9
    mov [r12 + VALUE_SIZE + 16], r10

    xor eax, eax
    ret

.underflow:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    mov eax, ERR_STACK_UNDERFLOW
    ret

;-----------------------------------------------------------------------------
; stack_data_rot - Rotate top 3 Values (a b c -- b c a)
; Returns: 0 on success
;-----------------------------------------------------------------------------
global stack_data_rot
stack_data_rot:
    push rbx
    push r11

    ; Check at least 3 elements (stack[2] must be valid)
    lea rax, [r12 + 2*VALUE_SIZE]
    cmp rax, [state + State.data_stack_base]
    jae .underflow

    ; Rotation: stack[2] -> top, others shift down
    ; Save stack[2] (deepest) - will become new top
    mov rax, [r12 + 2*VALUE_SIZE]
    mov rcx, [r12 + 2*VALUE_SIZE + 8]
    mov rdx, [r12 + 2*VALUE_SIZE + 16]

    ; stack[2] = stack[1]
    mov r8, [r12 + VALUE_SIZE]
    mov r9, [r12 + VALUE_SIZE + 8]
    mov r10, [r12 + VALUE_SIZE + 16]
    mov [r12 + 2*VALUE_SIZE], r8
    mov [r12 + 2*VALUE_SIZE + 8], r9
    mov [r12 + 2*VALUE_SIZE + 16], r10

    ; stack[1] = stack[0]
    mov r8, [r12]
    mov r9, [r12 + 8]
    mov r10, [r12 + 16]
    mov [r12 + VALUE_SIZE], r8
    mov [r12 + VALUE_SIZE + 8], r9
    mov [r12 + VALUE_SIZE + 16], r10

    ; stack[0] = saved (old stack[2])
    mov [r12], rax
    mov [r12 + 8], rcx
    mov [r12 + 16], rdx

    xor eax, eax
    jmp .done

.underflow:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    mov eax, ERR_STACK_UNDERFLOW

.done:
    pop r11
    pop rbx
    ret

;-----------------------------------------------------------------------------
; stack_data_over - Copy second Value to top (a b -- a b a)
; Returns: 0 on success
;-----------------------------------------------------------------------------
global stack_data_over
stack_data_over:
    ; Check at least 2 elements
    lea rax, [r12 + VALUE_SIZE]
    cmp rax, [state + State.data_stack_base]
    jae .underflow

    ; Check overflow
    mov rax, r12
    sub rax, VALUE_SIZE
    cmp rax, [state + State.data_stack_end]
    jb .overflow

    ; Fast copy stack[1] (second element) to new top
    ; stack[1] is at r12 + VALUE_SIZE, after push it will be at r12 + 2*VALUE_SIZE
    mov rax, [r12 + VALUE_SIZE]
    mov rcx, [r12 + VALUE_SIZE + 8]
    mov rdx, [r12 + VALUE_SIZE + 16]
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov [r12], rax
    mov [r12 + 8], rcx
    mov [r12 + 16], rdx

    xor eax, eax
    ret

.underflow:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    mov eax, ERR_STACK_UNDERFLOW
    ret

.overflow:
    mov dword [state + State.error], ERR_STACK_OVERFLOW
    mov eax, ERR_STACK_OVERFLOW
    ret

;-----------------------------------------------------------------------------
; stack_data_pick - Copy nth Value to top (0 = dup)
; Input: rdi = index from top
; Returns: 0 on success
;-----------------------------------------------------------------------------
global stack_data_pick
stack_data_pick:
    ; Calculate source address and check bounds
    imul rdi, VALUE_SIZE
    lea rax, [r12 + rdi]
    cmp rax, [state + State.data_stack_base]
    jae .underflow

    ; Check overflow
    mov rcx, r12
    sub rcx, VALUE_SIZE
    cmp rcx, [state + State.data_stack_end]
    jb .overflow

    ; Fast copy stack[n] to new top
    mov r8, [rax]
    mov r9, [rax + 8]
    mov r10, [rax + 16]
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov [r12], r8
    mov [r12 + 8], r9
    mov [r12 + 16], r10

    xor eax, eax
    ret

.underflow:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    mov eax, ERR_STACK_UNDERFLOW
    ret

.overflow:
    mov dword [state + State.error], ERR_STACK_OVERFLOW
    mov eax, ERR_STACK_OVERFLOW
    ret

;-----------------------------------------------------------------------------
; stack_data_roll - Roll nth Value to top, shifting others down
; Input: rdi = index from top (2 = rot)
; Returns: 0 on success
;-----------------------------------------------------------------------------
global stack_data_roll
stack_data_roll:
    push rbx
    push r13
    push r14
    push r15

    mov rbx, rdi            ; Save index

    ; Check bounds - stack[n] must be valid
    imul rax, rbx, VALUE_SIZE
    lea rcx, [r12 + rax]
    cmp rcx, [state + State.data_stack_base]
    jae .underflow

    test rbx, rbx
    jz .done_success        ; roll 0 is no-op

    ; Save stack[n] in registers
    mov r13, [rcx]
    mov r14, [rcx + 8]
    mov r15, [rcx + 16]

    ; Shift elements down: stack[n] = stack[n-1], ..., stack[1] = stack[0]
    ; Use fast 24-byte copies
.shift_loop:
    imul rax, rbx, VALUE_SIZE
    lea rdi, [r12 + rax]            ; Destination: stack[i]
    ; Source: stack[i-1]
    mov r8, [rdi - VALUE_SIZE]
    mov r9, [rdi - VALUE_SIZE + 8]
    mov r10, [rdi - VALUE_SIZE + 16]
    mov [rdi], r8
    mov [rdi + 8], r9
    mov [rdi + 16], r10
    dec rbx
    jnz .shift_loop

    ; stack[0] = saved value
    mov [r12], r13
    mov [r12 + 8], r14
    mov [r12 + 16], r15

.done_success:
    xor eax, eax
    jmp .done

.underflow:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    mov eax, ERR_STACK_UNDERFLOW

.done:
    pop r15
    pop r14
    pop r13
    pop rbx
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
    ; Calculate new stack pointer
    imul rdi, VALUE_SIZE
    lea rax, [r12 + rdi]

    ; Check bounds
    cmp rax, [state + State.data_stack_base]
    ja .underflow

    ; Move pointer
    mov r12, rax
    mov [state + State.data_stack], r12

    xor eax, eax
    ret

.underflow:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    mov eax, ERR_STACK_UNDERFLOW
    ret
