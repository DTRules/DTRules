; stack_control.asm - Control stack and frames
; DTRules Assembly Implementation
; Manages control flow frames for loops, conditionals, and calls

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state

;-----------------------------------------------------------------------------
; Frame types
;-----------------------------------------------------------------------------
FRAME_NONE      equ 0
FRAME_CALL      equ 1       ; Function call frame
FRAME_LOOP      equ 2       ; Loop frame (while, for)
FRAME_FORALL    equ 3       ; Forall iteration frame
FRAME_IF        equ 4       ; Conditional frame
FRAME_BLOCK     equ 5       ; Generic block frame

;-----------------------------------------------------------------------------
; Control frame structure (32 bytes)
;-----------------------------------------------------------------------------
struc Frame
    .type:          resb 1  ; Frame type
    .flags:         resb 1  ; Flags
    .pad:           resb 6  ; Padding
    .return_addr:   resq 1  ; Return bytecode address
    .saved_data_sp: resq 1  ; Saved data stack pointer
    .extra:         resq 1  ; Extra data (loop counter, iterator, etc.)
endstruc

FRAME_SIZE      equ 32

;-----------------------------------------------------------------------------
; Frame flags
;-----------------------------------------------------------------------------
FRAME_FLAG_BREAK    equ 1   ; Break was triggered
FRAME_FLAG_CONTINUE equ 2   ; Continue was triggered
FRAME_FLAG_RETURN   equ 4   ; Return was triggered

section .text

;-----------------------------------------------------------------------------
; stack_control_init - Initialize the control stack
;-----------------------------------------------------------------------------
global stack_control_init
stack_control_init:
    ; Reset stack pointer to base (empty stack)
    mov rax, [state + State.control_stack_base]
    mov [state + State.control_stack], rax
    mov r14, rax            ; Also update register
    ret

;-----------------------------------------------------------------------------
; stack_control_push_frame - Push a new control frame
; Input:
;   rdi = frame type
;   rsi = return address (bytecode pointer)
;   rdx = extra data
; Returns: 0 on success, ERR_STACK_OVERFLOW on error
;-----------------------------------------------------------------------------
global stack_control_push_frame
stack_control_push_frame:
    push rbp
    mov rbp, rsp

    ; Check for overflow
    mov rax, r14
    sub rax, FRAME_SIZE
    cmp rax, [state + State.control_stack_end]
    jb .overflow

    ; Push frame
    sub r14, FRAME_SIZE
    mov [state + State.control_stack], r14

    ; Initialize frame
    mov byte [r14 + Frame.type], dil
    mov byte [r14 + Frame.flags], 0
    mov [r14 + Frame.return_addr], rsi
    mov [r14 + Frame.saved_data_sp], r12    ; Save current data stack
    mov [r14 + Frame.extra], rdx

    xor eax, eax
    jmp .done

.overflow:
    mov dword [state + State.error], ERR_STACK_OVERFLOW
    mov eax, ERR_STACK_OVERFLOW

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; stack_control_pop_frame - Pop a control frame
; Output: rax = pointer to popped frame (valid until next push)
; Returns: pointer on success, 0 on underflow
;-----------------------------------------------------------------------------
global stack_control_pop_frame
stack_control_pop_frame:
    ; Check for underflow
    cmp r14, [state + State.control_stack_base]
    jae .underflow

    ; Return pointer to frame
    mov rax, r14

    ; Pop frame
    add r14, FRAME_SIZE
    mov [state + State.control_stack], r14

    ret

.underflow:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; stack_control_peek_frame - Peek at top frame
; Output: rax = pointer to frame
; Returns: pointer on success, 0 if empty
;-----------------------------------------------------------------------------
global stack_control_peek_frame
stack_control_peek_frame:
    cmp r14, [state + State.control_stack_base]
    jae .empty

    mov rax, r14
    ret

.empty:
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; stack_control_depth - Get number of frames on stack
; Output: rax = depth
;-----------------------------------------------------------------------------
global stack_control_depth
stack_control_depth:
    mov rax, [state + State.control_stack_base]
    sub rax, r14
    mov rcx, FRAME_SIZE
    xor edx, edx
    div rcx
    ret

;-----------------------------------------------------------------------------
; stack_control_clear - Clear the control stack
;-----------------------------------------------------------------------------
global stack_control_clear
stack_control_clear:
    mov rax, [state + State.control_stack_base]
    mov [state + State.control_stack], rax
    mov r14, rax
    ret

;-----------------------------------------------------------------------------
; stack_control_get_frame_type - Get type of top frame
; Output: rax = frame type (FRAME_NONE if empty)
;-----------------------------------------------------------------------------
global stack_control_get_frame_type
stack_control_get_frame_type:
    cmp r14, [state + State.control_stack_base]
    jae .empty

    movzx eax, byte [r14 + Frame.type]
    ret

.empty:
    xor eax, eax            ; FRAME_NONE
    ret

;-----------------------------------------------------------------------------
; stack_control_set_flag - Set a flag on top frame
; Input: rdi = flag bit(s) to set
;-----------------------------------------------------------------------------
global stack_control_set_flag
stack_control_set_flag:
    cmp r14, [state + State.control_stack_base]
    jae .done

    or [r14 + Frame.flags], dil

.done:
    ret

;-----------------------------------------------------------------------------
; stack_control_clear_flag - Clear a flag on top frame
; Input: rdi = flag bit(s) to clear
;-----------------------------------------------------------------------------
global stack_control_clear_flag
stack_control_clear_flag:
    cmp r14, [state + State.control_stack_base]
    jae .done

    not dil
    and [r14 + Frame.flags], dil

.done:
    ret

;-----------------------------------------------------------------------------
; stack_control_test_flag - Test if flag is set on top frame
; Input: rdi = flag bit to test
; Output: rax = 1 if set, 0 if not
;-----------------------------------------------------------------------------
global stack_control_test_flag
stack_control_test_flag:
    cmp r14, [state + State.control_stack_base]
    jae .not_set

    test [r14 + Frame.flags], dil
    jz .not_set

    mov eax, 1
    ret

.not_set:
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; stack_control_find_loop - Find nearest loop frame
; Output: rax = pointer to loop frame, 0 if not found
;
; Searches from top of stack for FRAME_LOOP or FRAME_FORALL
;-----------------------------------------------------------------------------
global stack_control_find_loop
stack_control_find_loop:
    push rbp
    mov rbp, rsp

    mov rax, r14            ; Start at top

.search:
    cmp rax, [state + State.control_stack_base]
    jae .not_found

    movzx ecx, byte [rax + Frame.type]
    cmp cl, FRAME_LOOP
    je .found
    cmp cl, FRAME_FORALL
    je .found

    add rax, FRAME_SIZE
    jmp .search

.found:
    pop rbp
    ret

.not_found:
    xor eax, eax
    pop rbp
    ret

;-----------------------------------------------------------------------------
; stack_control_unwind_to - Unwind stack to specific frame
; Input: rdi = target frame pointer
;
; Pops frames until reaching the target frame (inclusive)
;-----------------------------------------------------------------------------
global stack_control_unwind_to
stack_control_unwind_to:
    push rbp
    mov rbp, rsp

.unwind:
    cmp r14, [state + State.control_stack_base]
    jae .done               ; Stack is empty

    cmp r14, rdi
    ja .done                ; Past target (shouldn't happen)

    ; Pop this frame
    add r14, FRAME_SIZE

    cmp r14, rdi
    jbe .unwind             ; Keep going

.done:
    mov [state + State.control_stack], r14
    pop rbp
    ret

;-----------------------------------------------------------------------------
; stack_control_push_call - Push a call frame
; Input: rdi = return bytecode address
; Returns: 0 on success
;-----------------------------------------------------------------------------
global stack_control_push_call
stack_control_push_call:
    mov rsi, rdi            ; Return address
    mov rdi, FRAME_CALL     ; Frame type
    xor edx, edx            ; No extra data
    jmp stack_control_push_frame

;-----------------------------------------------------------------------------
; stack_control_push_loop - Push a loop frame
; Input: rdi = loop start bytecode address
; Returns: 0 on success
;-----------------------------------------------------------------------------
global stack_control_push_loop
stack_control_push_loop:
    mov rsi, rdi            ; Loop start address
    mov rdi, FRAME_LOOP
    xor edx, edx
    jmp stack_control_push_frame

;-----------------------------------------------------------------------------
; stack_control_push_forall - Push a forall iteration frame
; Input: rdi = loop start address, rsi = array/iterator pointer
; Returns: 0 on success
;-----------------------------------------------------------------------------
global stack_control_push_forall
stack_control_push_forall:
    mov rdx, rsi            ; Iterator in extra
    mov rsi, rdi            ; Loop start
    mov rdi, FRAME_FORALL
    jmp stack_control_push_frame

;-----------------------------------------------------------------------------
; stack_control_get_return_addr - Get return address from top frame
; Output: rax = return address
;-----------------------------------------------------------------------------
global stack_control_get_return_addr
stack_control_get_return_addr:
    cmp r14, [state + State.control_stack_base]
    jae .empty

    mov rax, [r14 + Frame.return_addr]
    ret

.empty:
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; stack_control_get_extra - Get extra data from top frame
; Output: rax = extra data
;-----------------------------------------------------------------------------
global stack_control_get_extra
stack_control_get_extra:
    cmp r14, [state + State.control_stack_base]
    jae .empty

    mov rax, [r14 + Frame.extra]
    ret

.empty:
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; stack_control_set_extra - Set extra data on top frame
; Input: rdi = extra data
;-----------------------------------------------------------------------------
global stack_control_set_extra
stack_control_set_extra:
    cmp r14, [state + State.control_stack_base]
    jae .done

    mov [r14 + Frame.extra], rdi

.done:
    ret

;-----------------------------------------------------------------------------
; stack_control_handle_break - Handle break statement
; Returns: bytecode address to jump to, or 0 if no enclosing loop
;
; Finds nearest loop frame, sets break flag, returns exit address
;-----------------------------------------------------------------------------
global stack_control_handle_break
stack_control_handle_break:
    push rbp
    mov rbp, rsp
    push rbx

    ; Find nearest loop
    call stack_control_find_loop
    test rax, rax
    jz .no_loop

    mov rbx, rax            ; Save frame pointer

    ; Set break flag
    mov byte [rbx + Frame.flags], FRAME_FLAG_BREAK

    ; Return address is after the loop body
    ; The bytecode interpreter will handle this
    mov rax, [rbx + Frame.return_addr]

    pop rbx
    pop rbp
    ret

.no_loop:
    xor eax, eax
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; stack_control_handle_continue - Handle continue statement
; Returns: bytecode address to jump to, or 0 if no enclosing loop
;
; Finds nearest loop frame, returns loop start address
;-----------------------------------------------------------------------------
global stack_control_handle_continue
stack_control_handle_continue:
    push rbp
    mov rbp, rsp

    ; Find nearest loop
    call stack_control_find_loop
    test rax, rax
    jz .no_loop

    ; Return loop start address
    mov rax, [rax + Frame.return_addr]

    pop rbp
    ret

.no_loop:
    xor eax, eax
    pop rbp
    ret
