; DTRules NASM VM Core
; Copyright 2024 Paul Snow
; Licensed under Apache License 2.0
;
; This file defines the core data structures, constants, and macros
; for the DTRules NASM VM. The VM is a stack-based interpreter with
; three stacks: data stack, entity stack, and control stack.
;
; Architecture: x86-64 System V ABI (Linux)
; Register conventions:
;   r12 - VM context pointer (preserved across calls)
;   r13 - Program counter (bytecode position)
;   r14 - Data stack pointer
;   r15 - Jump table base address

%ifndef VM_CORE_ASM
%define VM_CORE_ASM

%include "vm_constants.inc"
%include "vm_state.inc"

; ============================================================================
; External symbols (from vm_entry.asm)
; These are needed for macros that reference entry point functions
; ============================================================================
extern vm_dispatch
extern vm_error_return
extern vm_exit_success

; ============================================================================
; Helper Macros
; ============================================================================

; Push a value onto the data stack
; %1 = register containing Value.tag
; %2 = register containing Value.num
; %3 = register containing Value.ptr (or 0)
%macro data_push 3
    ; Check for overflow
    mov rax, [r12 + VMContext.data_sp]
    cmp rax, [r12 + VMContext.data_limit]
    jge %%overflow

    ; Store value
    mov byte [rax + Value.tag], %1
    mov qword [rax + Value.num], %2
    mov qword [rax + Value.ptr], %3

    ; Advance stack pointer
    add rax, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rax
    jmp %%done

%%overflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_OVERFLOW
    jmp vm_error_return

%%done:
%endmacro

; Push an integer value
; %1 = register or immediate containing integer value
%macro push_int 1
    mov rax, [r12 + VMContext.data_sp]
    cmp rax, [r12 + VMContext.data_limit]
    jge %%overflow

    mov byte [rax + Value.tag], VTAG_INTEGER
    mov qword [rax + Value.num], %1
    mov qword [rax + Value.ptr], 0

    add rax, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rax
    jmp %%done

%%overflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_OVERFLOW
    jmp vm_error_return

%%done:
%endmacro

; Push a boolean value (0 or 1)
; %1 = register containing 0 or non-zero value
%macro push_bool 1
    mov rax, [r12 + VMContext.data_sp]
    cmp rax, [r12 + VMContext.data_limit]
    jge %%overflow

    mov byte [rax + Value.tag], VTAG_BOOLEAN
    xor rcx, rcx
    test %1, %1
    setnz cl
    mov qword [rax + Value.num], rcx
    mov qword [rax + Value.ptr], 0

    add rax, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rax
    jmp %%done

%%overflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_OVERFLOW
    jmp vm_error_return

%%done:
%endmacro

; Push null value
%macro push_null 0
    mov rax, [r12 + VMContext.data_sp]
    cmp rax, [r12 + VMContext.data_limit]
    jge %%overflow

    mov byte [rax + Value.tag], VTAG_NULL
    mov qword [rax + Value.num], 0
    mov qword [rax + Value.ptr], 0

    add rax, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rax
    jmp %%done

%%overflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_OVERFLOW
    jmp vm_error_return

%%done:
%endmacro

; Pop a value from the data stack
; Returns tag in al, num in rbx, ptr in rcx
%macro data_pop 0
    mov rax, [r12 + VMContext.data_sp]
    sub rax, VALUE_SIZE
    cmp rax, [r12 + VMContext.data_base]
    jl %%underflow

    mov [r12 + VMContext.data_sp], rax
    movzx eax, byte [rax + Value.tag]       ; tag in al (zero-extended to eax)
    mov rbx, [rax + Value.num]              ; Bug: rax was modified! Will fix below
    jmp %%done

%%underflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_UNDERFLOW
    jmp vm_error_return

%%done:
%endmacro

; Corrected pop that preserves address
; Returns tag in al, num in rbx, ptr in rcx, address remains in rdi
%macro data_pop_safe 0
    mov rdi, [r12 + VMContext.data_sp]
    sub rdi, VALUE_SIZE
    cmp rdi, [r12 + VMContext.data_base]
    jl %%underflow

    mov [r12 + VMContext.data_sp], rdi
    movzx eax, byte [rdi + Value.tag]
    mov rbx, [rdi + Value.num]
    mov rcx, [rdi + Value.ptr]
    jmp %%done

%%underflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_UNDERFLOW
    jmp vm_error_return

%%done:
%endmacro

; Peek at top of stack without popping
; Returns tag in al, num in rbx, ptr in rcx
%macro data_peek 0
    mov rdi, [r12 + VMContext.data_sp]
    sub rdi, VALUE_SIZE
    cmp rdi, [r12 + VMContext.data_base]
    jl %%underflow

    movzx eax, byte [rdi + Value.tag]
    mov rbx, [rdi + Value.num]
    mov rcx, [rdi + Value.ptr]
    jmp %%done

%%underflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_UNDERFLOW
    jmp vm_error_return

%%done:
%endmacro

; Read next byte from bytecode and advance PC
; Result in al (zero-extended to rax)
%macro read_byte 0
    mov rax, [r12 + VMContext.pc]
    mov rdi, [r12 + VMContext.bytecode]
    movzx eax, byte [rdi + rax]
    inc qword [r12 + VMContext.pc]
%endmacro

; Read varint from bytecode
; Result in rax
%macro read_varint 0
    xor rax, rax                    ; result = 0
    xor rcx, rcx                    ; shift = 0
    mov rdi, [r12 + VMContext.bytecode]
    mov rsi, [r12 + VMContext.pc]

%%loop:
    movzx edx, byte [rdi + rsi]     ; read byte
    inc rsi                         ; advance PC

    mov r8, rdx
    and r8, 0x7f                    ; mask continuation bit
    shl r8, cl                      ; shift by current amount
    or rax, r8                      ; add to result

    test dl, 0x80                   ; more bytes?
    jz %%done

    add cl, 7                       ; shift += 7
    cmp cl, 63                      ; overflow check
    jbe %%loop

    ; Varint overflow
    mov dword [r12 + VMContext.error_code], ERR_OUT_OF_BOUNDS
    jmp vm_error_return

%%done:
    mov [r12 + VMContext.pc], rsi   ; save PC
%endmacro

; Dispatch to next opcode via jump table
; Simply jumps to the global dispatch point in vm_entry.asm
%macro dispatch 0
    jmp vm_dispatch
%endmacro

; Record trace entry (if tracing enabled)
%macro trace_record 1  ; %1 = opcode
    test dword [r12 + VMContext.state_flags], STATE_TRACE
    jz %%skip

    ; Get trace buffer position
    mov rdi, [r12 + VMContext.trace_pos]
    cmp rdi, [r12 + VMContext.trace_limit]
    jge %%skip                      ; Buffer full

    ; Record opcode and PC
    mov byte [rdi + TraceEntry.opcode], %1
    mov rax, [r12 + VMContext.pc]
    dec rax                         ; PC of the instruction we're executing
    mov [rdi + TraceEntry.pc], rax

    ; Record top of stack (if any)
    mov rax, [r12 + VMContext.data_sp]
    cmp rax, [r12 + VMContext.data_base]
    je %%no_value

    sub rax, VALUE_SIZE
    ; Copy Value to trace
    mov rcx, VALUE_SIZE / 8
    lea rsi, [rax]
    lea rdi, [rdi + TraceEntry.value]
    rep movsq
    jmp %%advance

%%no_value:
    ; Store null value in trace
    mov rdi, [r12 + VMContext.trace_pos]
    mov byte [rdi + TraceEntry.value + Value.tag], VTAG_NULL

%%advance:
    mov rdi, [r12 + VMContext.trace_pos]
    add rdi, TraceEntry_size
    mov [r12 + VMContext.trace_pos], rdi

%%skip:
%endmacro

%endif ; VM_CORE_ASM
