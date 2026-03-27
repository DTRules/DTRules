; DTRules NASM VM - Core Implementation
; Copyright 2024 Paul Snow - Apache License 2.0
;
; This file implements:
; - VM context management
; - Stack operations
; - Core utility routines
;
; Architecture pattern: Fifth/AISynth
; - Single entry/exit point (vm_execute)
; - Jump table dispatch
; - Handlers end with jmp dispatch, not ret
; - Trace writes to buffer, no function calls

%include "dtrules.inc"

; External symbols (provided by Go runtime)
extern vm_callback_lookup      ; Lookup name in entity context
extern vm_callback_def         ; Define variable in entity context
extern vm_callback_operator    ; Execute operator by index
extern vm_callback_error       ; Report error to Go

; ===========================================================================
; Data Section
; ===========================================================================

section .data

; Error messages
err_stack_overflow:     db "Stack overflow", 0
err_stack_underflow:    db "Stack underflow", 0
err_invalid_opcode:     db "Invalid opcode", 0
err_bounds:             db "Index out of bounds", 0
err_type:               db "Type mismatch", 0
err_div_zero:           db "Division by zero", 0

; ===========================================================================
; BSS Section (uninitialized data)
; ===========================================================================

section .bss

; Reserved for future use

; ===========================================================================
; Text Section (code)
; ===========================================================================

section .text

; ---------------------------------------------------------------------------
; vm_init_context - Initialize a VM context structure
; ---------------------------------------------------------------------------
; Input:
;   rdi - pointer to VMContext to initialize
;   rsi - pointer to bytecode
;   rdx - bytecode length
;   rcx - pointer to data stack
;   r8  - data stack capacity
;   r9  - pointer to trace buffer (0 to disable tracing)
;   [rsp+8] - trace buffer capacity
;
; Output:
;   None (context is initialized)
;
; Clobbers: rax, r10
; ---------------------------------------------------------------------------
global vm_init_context
vm_init_context:
    ; Clear the entire context structure first
    push rdi
    push rcx
    mov rcx, VMContext_size / 8
    xor eax, eax
.clear_loop:
    mov [rdi], rax
    add rdi, 8
    dec rcx
    jnz .clear_loop
    pop rcx
    pop rdi

    ; Set bytecode
    mov [rdi + VMContext.bytecode], rsi
    mov [rdi + VMContext.bytecode_len], rdx
    mov qword [rdi + VMContext.pc], 0

    ; Set data stack
    mov [rdi + VMContext.data_stack], rcx
    mov qword [rdi + VMContext.data_sp], 0
    mov [rdi + VMContext.data_stack_cap], r8

    ; Set trace buffer if provided
    test r9, r9
    jz .no_trace
    mov [rdi + VMContext.trace_buffer], r9
    mov qword [rdi + VMContext.trace_pos], 0
    mov r10, [rsp + 8]      ; trace capacity from stack
    mov [rdi + VMContext.trace_cap], r10
    or qword [rdi + VMContext.flags], VM_FLAG_TRACE
.no_trace:

    ; Set running flag
    or qword [rdi + VMContext.flags], VM_FLAG_RUNNING

    ret


; ---------------------------------------------------------------------------
; vm_data_push - Push a value onto the data stack
; ---------------------------------------------------------------------------
; Input:
;   rdi - pointer to VMContext
;   rsi - value tag
;   rdx - value num (integer, double bits, or pointer)
;
; Output:
;   rax - 0 on success, error code on failure
;
; Clobbers: rcx, r10, r11
; ---------------------------------------------------------------------------
global vm_data_push
vm_data_push:
    ; Check for overflow
    mov rcx, [rdi + VMContext.data_sp]
    cmp rcx, [rdi + VMContext.data_stack_cap]
    jge .overflow

    ; Calculate slot address: base + sp * VALUE_SIZE
    mov r10, [rdi + VMContext.data_stack]
    mov r11, rcx
    shl r11, 4              ; * 16 (VALUE_SIZE)
    add r10, r11

    ; Store value
    mov [r10 + VALUE_TAG_OFFSET], rsi
    mov [r10 + VALUE_NUM_OFFSET], rdx

    ; Increment stack pointer
    inc qword [rdi + VMContext.data_sp]

    xor eax, eax            ; success
    ret

.overflow:
    mov eax, 1              ; error: overflow
    ret


; ---------------------------------------------------------------------------
; vm_data_pop - Pop a value from the data stack
; ---------------------------------------------------------------------------
; Input:
;   rdi - pointer to VMContext
;
; Output:
;   rax - 0 on success, error code on failure
;   rsi - value tag (if success)
;   rdx - value num (if success)
;
; Clobbers: rcx, r10
; ---------------------------------------------------------------------------
global vm_data_pop
vm_data_pop:
    ; Check for underflow
    mov rcx, [rdi + VMContext.data_sp]
    test rcx, rcx
    jz .underflow

    ; Decrement stack pointer
    dec rcx
    mov [rdi + VMContext.data_sp], rcx

    ; Calculate slot address: base + sp * VALUE_SIZE
    mov r10, [rdi + VMContext.data_stack]
    shl rcx, 4              ; * 16 (VALUE_SIZE)
    add r10, rcx

    ; Load value
    mov rsi, [r10 + VALUE_TAG_OFFSET]
    mov rdx, [r10 + VALUE_NUM_OFFSET]

    xor eax, eax            ; success
    ret

.underflow:
    mov eax, 2              ; error: underflow
    ret


; ---------------------------------------------------------------------------
; vm_data_peek - Peek at the top value without removing it
; ---------------------------------------------------------------------------
; Input:
;   rdi - pointer to VMContext
;   rsi - offset from top (0 = top, 1 = second from top, etc.)
;
; Output:
;   rax - 0 on success, error code on failure
;   rdx - value tag (if success)
;   rcx - value num (if success)
;
; Clobbers: r10, r11
; ---------------------------------------------------------------------------
global vm_data_peek
vm_data_peek:
    ; Check bounds
    mov r10, [rdi + VMContext.data_sp]
    cmp rsi, r10
    jge .out_of_bounds

    ; Calculate index: sp - 1 - offset
    mov r11, r10
    dec r11
    sub r11, rsi

    ; Calculate slot address
    mov r10, [rdi + VMContext.data_stack]
    shl r11, 4              ; * 16 (VALUE_SIZE)
    add r10, r11

    ; Load value
    mov rdx, [r10 + VALUE_TAG_OFFSET]
    mov rcx, [r10 + VALUE_NUM_OFFSET]

    xor eax, eax            ; success
    ret

.out_of_bounds:
    mov eax, 3              ; error: out of bounds
    ret


; ---------------------------------------------------------------------------
; vm_data_depth - Get the current stack depth
; ---------------------------------------------------------------------------
; Input:
;   rdi - pointer to VMContext
;
; Output:
;   rax - stack depth
; ---------------------------------------------------------------------------
global vm_data_depth
vm_data_depth:
    mov rax, [rdi + VMContext.data_sp]
    ret


; ---------------------------------------------------------------------------
; vm_read_varint - Read a variable-length integer from bytecode
; ---------------------------------------------------------------------------
; Input:
;   rdi - pointer to VMContext
;
; Output:
;   rax - decoded integer value
;   Sets VM_FLAG_ERROR on bounds error
;
; Clobbers: rcx, rdx, r10, r11
; ---------------------------------------------------------------------------
global vm_read_varint
vm_read_varint:
    xor eax, eax            ; result = 0
    xor ecx, ecx            ; shift = 0

    mov r10, [rdi + VMContext.bytecode]
    mov r11, [rdi + VMContext.pc]

.loop:
    ; Bounds check
    cmp r11, [rdi + VMContext.bytecode_len]
    jge .bounds_error

    ; Read byte
    movzx edx, byte [r10 + r11]
    inc r11

    ; Decode: result |= (byte & 0x7f) << shift
    mov r8d, edx
    and r8d, 0x7f
    push rcx
    shl r8, cl
    pop rcx
    or rax, r8

    ; Check continuation bit
    test dl, 0x80
    jz .done

    ; shift += 7
    add cl, 7
    cmp cl, 63              ; overflow check
    jle .loop

    ; Varint overflow - treat as error
    or qword [rdi + VMContext.flags], VM_FLAG_ERROR
    mov qword [rdi + VMContext.error_code], 4
    xor eax, eax
    ret

.done:
    ; Update PC
    mov [rdi + VMContext.pc], r11
    ret

.bounds_error:
    or qword [rdi + VMContext.flags], VM_FLAG_ERROR
    mov qword [rdi + VMContext.error_code], 3
    xor eax, eax
    ret


; ---------------------------------------------------------------------------
; vm_get_constant - Get a constant from the constant pool
; ---------------------------------------------------------------------------
; Input:
;   rdi - pointer to VMContext
;   rsi - constant index
;
; Output:
;   rax - 0 on success, error code on failure
;   rdx - value tag
;   rcx - value num
;
; Clobbers: r10
; ---------------------------------------------------------------------------
global vm_get_constant
vm_get_constant:
    ; Bounds check
    cmp rsi, [rdi + VMContext.constants_count]
    jge .out_of_bounds

    ; Calculate address: constants + index * VALUE_SIZE
    mov r10, [rdi + VMContext.constants]
    shl rsi, 4              ; * 16
    add r10, rsi

    ; Load value
    mov rdx, [r10 + VALUE_TAG_OFFSET]
    mov rcx, [r10 + VALUE_NUM_OFFSET]

    xor eax, eax
    ret

.out_of_bounds:
    mov eax, 3
    ret


; ---------------------------------------------------------------------------
; vm_get_name - Get a name from the name pool
; ---------------------------------------------------------------------------
; Input:
;   rdi - pointer to VMContext
;   rsi - name index
;
; Output:
;   rax - pointer to RName (or 0 on error)
;
; Clobbers: r10
; ---------------------------------------------------------------------------
global vm_get_name
vm_get_name:
    ; Bounds check
    cmp rsi, [rdi + VMContext.names_count]
    jge .out_of_bounds

    ; Calculate address: names + index * 8 (pointer size)
    mov r10, [rdi + VMContext.names]
    shl rsi, 3              ; * 8
    mov rax, [r10 + rsi]
    ret

.out_of_bounds:
    xor eax, eax
    ret
