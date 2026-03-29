; DTRules NASM VM - Entry Point and Main Dispatch Loop
; Copyright 2024 Paul Snow - Apache License 2.0
;
; This file implements:
; - Single entry point (vm_execute)
; - Main dispatch loop using jump table
; - Opcode handlers return via jmp dispatch
;
; Register conventions:
;   r12 - VMContext pointer (preserved across handlers)
;   r13 - bytecode base pointer
;   r14 - PC (program counter offset)
;   r15 - data stack base pointer
;
; All handlers preserve these registers and jump back to dispatch.

%include "dtrules.inc"

; External handler groups
extern jump_table               ; The opcode jump table
extern trace_record_opcode      ; Record opcode to trace buffer

; Core routines
extern vm_data_push
extern vm_data_pop
extern vm_data_peek
extern vm_read_varint
extern vm_get_constant
extern vm_get_name

; Go callbacks
extern vm_callback_lookup
extern vm_callback_def
extern vm_callback_operator

section .data

section .text

; ---------------------------------------------------------------------------
; vm_execute - Main entry point for bytecode execution
; ---------------------------------------------------------------------------
; Input:
;   rdi - pointer to VMContext (initialized)
;
; Output:
;   rax - 0 on success, error code on failure
;
; This is the ONLY entry point into the VM. All state is in VMContext.
; Execution continues until:
;   - End of bytecode reached
;   - VM_FLAG_HALT is set
;   - VM_FLAG_ERROR is set
;   - OP_RETURN at top level
; ---------------------------------------------------------------------------
global vm_execute
vm_execute:
    ; Prologue - save callee-saved registers
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14
    push r15
    sub rsp, 40             ; Local space + alignment

    ; Setup register conventions
    mov r12, rdi            ; r12 = VMContext pointer
    mov r13, [r12 + VMContext.bytecode]
    mov r14, [r12 + VMContext.pc]
    mov r15, [r12 + VMContext.data_stack]

    ; Check if already in error state
    test qword [r12 + VMContext.flags], VM_FLAG_ERROR
    jnz .error_exit

    ; Check if running
    test qword [r12 + VMContext.flags], VM_FLAG_RUNNING
    jz .not_running

; ---------------------------------------------------------------------------
; dispatch - Main dispatch loop
; ---------------------------------------------------------------------------
; Fetch opcode, lookup handler in jump table, jump to it.
; Each handler jumps back here when done.
; ---------------------------------------------------------------------------
.dispatch:
    ; Update context PC (for error reporting and callbacks)
    mov [r12 + VMContext.pc], r14

    ; Check for halt
    test qword [r12 + VMContext.flags], VM_FLAG_HALT
    jnz .halt_exit

    ; Check for error
    test qword [r12 + VMContext.flags], VM_FLAG_ERROR
    jnz .error_exit

    ; Check bounds
    cmp r14, [r12 + VMContext.bytecode_len]
    jge .end_of_code

    ; Fetch opcode
    movzx eax, byte [r13 + r14]
    inc r14                 ; Advance PC

    ; Increment instruction counter
    inc qword [r12 + VMContext.instructions_executed]

    ; Record trace if enabled
    test qword [r12 + VMContext.flags], VM_FLAG_TRACE
    jz .no_trace

    ; Save opcode and call trace_record_opcode
    push rax
    mov rdi, r12            ; context
    mov esi, eax            ; opcode
    call trace_record_opcode
    pop rax

.no_trace:
    ; Dispatch via jump table
    ; jump_table[opcode] contains handler address
    lea rcx, [rel jump_table]
    mov rcx, [rcx + rax * 8]
    jmp rcx

; ---------------------------------------------------------------------------
; Exit points
; ---------------------------------------------------------------------------

.end_of_code:
    ; Normal termination - end of bytecode
    xor eax, eax
    jmp .epilogue

.halt_exit:
    ; Explicit halt requested
    xor eax, eax
    jmp .epilogue

.error_exit:
    ; Error occurred
    mov rax, [r12 + VMContext.error_code]
    test rax, rax
    jnz .epilogue
    mov eax, 1              ; Generic error if no code set
    jmp .epilogue

.not_running:
    mov eax, 5              ; Not running error
    jmp .epilogue

.epilogue:
    ; Save final PC
    mov [r12 + VMContext.pc], r14

    ; Clear running flag
    and qword [r12 + VMContext.flags], ~VM_FLAG_RUNNING

    ; Restore callee-saved registers
    add rsp, 40
    pop r15
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret


; ---------------------------------------------------------------------------
; Handler implementations
; ---------------------------------------------------------------------------
; Each handler receives:
;   r12 = VMContext pointer
;   r13 = bytecode base
;   r14 = PC (already past opcode byte)
;   r15 = data stack base
;
; Each handler must end with: jmp vm_execute.dispatch
; Use the global label for dispatch re-entry
; ---------------------------------------------------------------------------

; Make dispatch accessible to other handler files
global vm_dispatch
vm_dispatch equ vm_execute.dispatch


; ---------------------------------------------------------------------------
; op_nop - No operation
; ---------------------------------------------------------------------------
global op_nop
op_nop:
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_push_true - Push boolean true
; ---------------------------------------------------------------------------
global op_push_true
op_push_true:
    mov rdi, r12
    mov rsi, VTAG_BOOLEAN
    mov rdx, 1              ; true = 1
    call vm_data_push
    test eax, eax
    jnz .error
    jmp vm_dispatch
.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 1  ; overflow
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_push_false - Push boolean false
; ---------------------------------------------------------------------------
global op_push_false
op_push_false:
    mov rdi, r12
    mov rsi, VTAG_BOOLEAN
    xor edx, edx            ; false = 0
    call vm_data_push
    test eax, eax
    jnz .error
    jmp vm_dispatch
.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 1
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_push_null - Push null value
; ---------------------------------------------------------------------------
global op_push_null
op_push_null:
    mov rdi, r12
    mov rsi, VTAG_NULL
    xor edx, edx
    call vm_data_push
    test eax, eax
    jnz .error
    jmp vm_dispatch
.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 1
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_push_zero - Push integer 0
; ---------------------------------------------------------------------------
global op_push_zero
op_push_zero:
    mov rdi, r12
    mov rsi, VTAG_INTEGER
    xor edx, edx
    call vm_data_push
    test eax, eax
    jnz .error
    jmp vm_dispatch
.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 1
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_push_one - Push integer 1
; ---------------------------------------------------------------------------
global op_push_one
op_push_one:
    mov rdi, r12
    mov rsi, VTAG_INTEGER
    mov rdx, 1
    call vm_data_push
    test eax, eax
    jnz .error
    jmp vm_dispatch
.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 1
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_push_int - Push immediate integer (varint follows)
; ---------------------------------------------------------------------------
global op_push_int
op_push_int:
    ; Read varint
    mov rdi, r12
    call vm_read_varint
    ; Check for error (already sets flag)
    test qword [r12 + VMContext.flags], VM_FLAG_ERROR
    jnz vm_dispatch

    ; Update r14 from context (varint reader updates PC)
    mov r14, [r12 + VMContext.pc]

    ; Push the value
    mov rdx, rax            ; value
    mov rdi, r12
    mov rsi, VTAG_INTEGER
    call vm_data_push
    test eax, eax
    jnz .error
    jmp vm_dispatch
.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 1
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_constant - Push constant from pool (varint index follows)
; ---------------------------------------------------------------------------
global op_constant
op_constant:
    ; Read index
    mov rdi, r12
    call vm_read_varint
    test qword [r12 + VMContext.flags], VM_FLAG_ERROR
    jnz vm_dispatch
    mov r14, [r12 + VMContext.pc]

    ; Get constant
    mov rdi, r12
    mov rsi, rax            ; index
    call vm_get_constant
    test eax, eax
    jnz .error

    ; Push value (rdx = tag, rcx = num)
    mov rdi, r12
    mov rsi, rdx
    mov rdx, rcx
    call vm_data_push
    test eax, eax
    jnz .error
    jmp vm_dispatch
.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 3  ; bounds
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_name - Push name from pool (varint index follows)
; ---------------------------------------------------------------------------
global op_name
op_name:
    ; Read index
    mov rdi, r12
    call vm_read_varint
    test qword [r12 + VMContext.flags], VM_FLAG_ERROR
    jnz vm_dispatch
    mov r14, [r12 + VMContext.pc]

    ; Get name pointer
    mov rdi, r12
    mov rsi, rax
    call vm_get_name
    test rax, rax
    jz .error

    ; Push as name value
    mov rdi, r12
    mov rsi, VTAG_NAME
    mov rdx, rax            ; pointer to RName
    call vm_data_push
    test eax, eax
    jnz .error
    jmp vm_dispatch
.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 3
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_pop - Pop and discard top value
; ---------------------------------------------------------------------------
global op_pop
op_pop:
    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error
    jmp vm_dispatch
.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 2  ; underflow
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_dup - Duplicate top value
; ---------------------------------------------------------------------------
global op_dup
op_dup:
    ; Peek top value
    mov rdi, r12
    xor esi, esi            ; offset 0 = top
    call vm_data_peek
    test eax, eax
    jnz .error

    ; Push copy (rdx = tag, rcx = num)
    mov rdi, r12
    mov rsi, rdx
    mov rdx, rcx
    call vm_data_push
    test eax, eax
    jnz .error
    jmp vm_dispatch
.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 2
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_swap - Swap top two values
; ---------------------------------------------------------------------------
global op_swap
op_swap:
    ; Need at least 2 values
    mov rax, [r12 + VMContext.data_sp]
    cmp rax, 2
    jl .error

    ; Calculate addresses
    mov rcx, [r12 + VMContext.data_stack]
    mov rdx, rax
    dec rdx
    shl rdx, 4              ; top slot offset
    lea r8, [rcx + rdx]     ; r8 = top slot

    mov rdx, rax
    sub rdx, 2
    shl rdx, 4
    lea r9, [rcx + rdx]     ; r9 = second slot

    ; Swap using xmm registers (16 bytes each)
    movdqu xmm0, [r8]
    movdqu xmm1, [r9]
    movdqu [r8], xmm1
    movdqu [r9], xmm0

    jmp vm_dispatch
.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 2
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_rot - Rotate top three: ( a b c -- b c a )
; ---------------------------------------------------------------------------
global op_rot
op_rot:
    ; Need at least 3 values
    mov rax, [r12 + VMContext.data_sp]
    cmp rax, 3
    jl .error

    mov rcx, [r12 + VMContext.data_stack]

    ; Calculate addresses
    mov rdx, rax
    dec rdx
    shl rdx, 4
    lea r8, [rcx + rdx]     ; top (c)

    mov rdx, rax
    sub rdx, 2
    shl rdx, 4
    lea r9, [rcx + rdx]     ; second (b)

    mov rdx, rax
    sub rdx, 3
    shl rdx, 4
    lea r10, [rcx + rdx]    ; third (a)

    ; Rotate: save a, b->a, c->b, a->c
    movdqu xmm0, [r10]      ; save a
    movdqu xmm1, [r9]       ; load b
    movdqu [r10], xmm1      ; a = b
    movdqu xmm1, [r8]       ; load c
    movdqu [r9], xmm1       ; b = c
    movdqu [r8], xmm0       ; c = a

    jmp vm_dispatch
.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 2
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_invalid - Handler for invalid/unimplemented opcodes
; ---------------------------------------------------------------------------
global op_invalid
op_invalid:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 6  ; invalid opcode
    mov rax, r14
    dec rax                 ; PC of the bad opcode
    mov [r12 + VMContext.error_pc], rax
    jmp vm_dispatch
