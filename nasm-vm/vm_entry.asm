; DTRules NASM VM Entry Point
; Copyright 2024 Paul Snow
; Licensed under Apache License 2.0
;
; This file provides the C-callable entry point for the VM.
; It follows System V AMD64 ABI conventions:
;   - Arguments: rdi, rsi, rdx, rcx, r8, r9
;   - Return: rax, rdx
;   - Callee-saved: rbx, rbp, r12-r15
;   - Caller-saved: rax, rcx, rdx, rsi, rdi, r8-r11
;
; Single entry/exit design:
;   - One call to vm_execute runs bytecode to completion
;   - No per-operation CGO-style calls
;   - Trace recording writes to buffer, not callbacks

; Include only constants and state, NOT vm_core.asm
; (vm_core.asm has extern declarations that conflict with our global defs)
%include "vm_constants.inc"
%include "vm_state.inc"

section .data

; Null value constant for initialization
align 8
null_value:
    db VTAG_NULL        ; tag
    times 7 db 0        ; padding
    dq 0                ; num
    dq 0                ; ptr

; True value constant
align 8
true_value:
    db VTAG_BOOLEAN     ; tag
    times 7 db 0        ; padding
    dq 1                ; num = 1 (true)
    dq 0                ; ptr

; False value constant
align 8
false_value:
    db VTAG_BOOLEAN     ; tag
    times 7 db 0        ; padding
    dq 0                ; num = 0 (false)
    dq 0                ; ptr

section .text

; ============================================================================
; Main Entry Point
; ============================================================================

; vm_execute - Execute bytecode
;
; C prototype:
;   int vm_execute(VMContext *ctx);
;
; Arguments:
;   rdi - Pointer to initialized VMContext
;
; Returns:
;   eax - Error code (0 = success)
;
; The VMContext must be initialized with:
;   - Valid stack bases and limits
;   - Bytecode buffer and length
;   - Jump table pointer
;   - Constant and name pools
;
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

    ; Set up VM registers
    mov r12, rdi                    ; r12 = VMContext pointer
    mov r13, [r12 + VMContext.pc]   ; r13 = program counter
    mov r14, [r12 + VMContext.data_sp] ; r14 = data stack pointer
    mov r15, [r12 + VMContext.jump_table] ; r15 = jump table base

    ; Clear error code
    mov dword [r12 + VMContext.error_code], ERR_NONE

    ; Reset instruction count
    mov qword [r12 + VMContext.insn_count], 0

    ; Start dispatch loop
    jmp vm_dispatch

; Global dispatch entry point - called from opcode handlers
global vm_dispatch
vm_dispatch:
    ; Check PC bounds
    mov rax, [r12 + VMContext.pc]
    cmp rax, [r12 + VMContext.bytecode_len]
    jge vm_exit_success             ; End of bytecode

    ; Fetch and decode opcode
    mov rdi, [r12 + VMContext.bytecode]
    movzx eax, byte [rdi + rax]     ; opcode in eax
    inc qword [r12 + VMContext.pc]

    ; Increment instruction count
    inc qword [r12 + VMContext.insn_count]

    ; Dispatch via jump table
    jmp [r15 + rax * 8]

; Global exit point - can be called from op_halt in vm_jump_table.asm
global vm_exit_success
vm_exit_success:
    ; NOTE: We don't save r14 here because handlers update VMContext.data_sp
    ; directly. Saving r14 would overwrite their changes.
    ; Success - return 0
    xor eax, eax
    jmp vm_epilogue

; ============================================================================
; Error Return Point
; ============================================================================

; vm_error_return - Return from VM with error
; Called by handlers when an error occurs
; Error code should already be set in VMContext.error_code
global vm_error_return
vm_error_return:
    ; NOTE: We don't save r14 here because handlers update VMContext.data_sp directly

    ; Get error code
    mov eax, [r12 + VMContext.error_code]
    ; Fall through to epilogue

vm_epilogue:
    ; Epilogue - restore callee-saved registers
    pop r15
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

; ============================================================================
; Context Initialization
; ============================================================================

; vm_init_context - Initialize a VMContext structure
;
; C prototype:
;   void vm_init_context(VMContext *ctx,
;                        void *data_stack, size_t data_size,
;                        void *entity_stack, size_t entity_size,
;                        void *ctrl_stack, size_t ctrl_size);
;
; Arguments:
;   rdi - VMContext pointer
;   rsi - Data stack buffer
;   rdx - Data stack size (bytes)
;   rcx - Entity stack buffer
;   r8  - Entity stack size (bytes)
;   r9  - Control stack buffer
;   [rbp+16] - Control stack size
;
global vm_init_context
vm_init_context:
    push rbp
    mov rbp, rsp

    ; Zero the context
    push rdi
    mov rcx, VMContext_size
    xor al, al
    rep stosb
    pop rdi

    ; Data stack
    mov [rdi + VMContext.data_base], rsi
    mov [rdi + VMContext.data_sp], rsi
    add rsi, rdx
    mov [rdi + VMContext.data_limit], rsi

    ; Entity stack
    mov [rdi + VMContext.entity_base], rcx
    mov [rdi + VMContext.entity_sp], rcx
    add rcx, r8
    mov [rdi + VMContext.entity_limit], rcx

    ; Control stack
    mov [rdi + VMContext.ctrl_base], r9
    mov [rdi + VMContext.ctrl_sp], r9
    mov rax, [rbp + 16]             ; ctrl_size from stack
    add r9, rax
    mov [rdi + VMContext.ctrl_limit], r9

    pop rbp
    ret

; ============================================================================
; Bytecode Loading
; ============================================================================

; vm_load_bytecode - Load bytecode into context
;
; C prototype:
;   void vm_load_bytecode(VMContext *ctx, void *bytecode, size_t len);
;
; Arguments:
;   rdi - VMContext pointer
;   rsi - Bytecode buffer
;   rdx - Bytecode length
;
global vm_load_bytecode
vm_load_bytecode:
    mov [rdi + VMContext.bytecode], rsi
    mov [rdi + VMContext.bytecode_len], rdx
    mov qword [rdi + VMContext.pc], 0
    ret

; ============================================================================
; Constant Pool Loading
; ============================================================================

; vm_load_constants - Load constant pool into context
;
; C prototype:
;   void vm_load_constants(VMContext *ctx, Value *constants, size_t count);
;
; Arguments:
;   rdi - VMContext pointer
;   rsi - Constants array
;   rdx - Number of constants
;
global vm_load_constants
vm_load_constants:
    mov [rdi + VMContext.constants], rsi
    mov [rdi + VMContext.const_count], rdx
    ret

; ============================================================================
; Name Pool Loading
; ============================================================================

; vm_load_names - Load name pool into context
;
; C prototype:
;   void vm_load_names(VMContext *ctx, RName **names, size_t count);
;
; Arguments:
;   rdi - VMContext pointer
;   rsi - Names array (array of pointers)
;   rdx - Number of names
;
global vm_load_names
vm_load_names:
    mov [rdi + VMContext.names], rsi
    mov [rdi + VMContext.name_count], rdx
    ret

; ============================================================================
; Jump Table Loading
; ============================================================================

; vm_set_jump_table - Set jump table for opcode dispatch
;
; C prototype:
;   void vm_set_jump_table(VMContext *ctx, void **jump_table);
;
; Arguments:
;   rdi - VMContext pointer
;   rsi - Jump table (array of 256 function pointers)
;
global vm_set_jump_table
vm_set_jump_table:
    mov [rdi + VMContext.jump_table], rsi
    ret

; ============================================================================
; Trace Buffer Setup
; ============================================================================

; vm_set_trace_buffer - Configure trace buffer
;
; C prototype:
;   void vm_set_trace_buffer(VMContext *ctx, TraceEntry *buf, size_t count);
;
; Arguments:
;   rdi - VMContext pointer
;   rsi - Trace buffer
;   rdx - Maximum number of trace entries
;
global vm_set_trace_buffer
vm_set_trace_buffer:
    mov [rdi + VMContext.trace_buf], rsi
    mov [rdi + VMContext.trace_pos], rsi
    ; Calculate limit
    imul rdx, TraceEntry_size
    add rdx, rsi
    mov [rdi + VMContext.trace_limit], rdx
    ret

; ============================================================================
; State Flag Control
; ============================================================================

; vm_enable_trace - Enable execution tracing
;
; C prototype:
;   void vm_enable_trace(VMContext *ctx);
;
global vm_enable_trace
vm_enable_trace:
    or dword [rdi + VMContext.state_flags], STATE_TRACE
    ret

; vm_disable_trace - Disable execution tracing
;
; C prototype:
;   void vm_disable_trace(VMContext *ctx);
;
global vm_disable_trace
vm_disable_trace:
    and dword [rdi + VMContext.state_flags], ~STATE_TRACE
    ret

; vm_enable_debug - Enable debug mode
;
; C prototype:
;   void vm_enable_debug(VMContext *ctx);
;
global vm_enable_debug
vm_enable_debug:
    or dword [rdi + VMContext.state_flags], STATE_DEBUG
    ret

; vm_disable_debug - Disable debug mode
;
; C prototype:
;   void vm_disable_debug(VMContext *ctx);
;
global vm_disable_debug
vm_disable_debug:
    and dword [rdi + VMContext.state_flags], ~STATE_DEBUG
    ret

; ============================================================================
; Stack Inspection (for testing and debugging)
; ============================================================================

; vm_get_stack_depth - Get current data stack depth
;
; C prototype:
;   size_t vm_get_stack_depth(VMContext *ctx);
;
global vm_get_stack_depth
vm_get_stack_depth:
    mov rax, [rdi + VMContext.data_sp]
    sub rax, [rdi + VMContext.data_base]
    xor rdx, rdx
    mov rcx, VALUE_SIZE
    div rcx
    ret

; vm_peek_stack - Peek at stack element
;
; C prototype:
;   int vm_peek_stack(VMContext *ctx, size_t index, Value *out);
;
; Arguments:
;   rdi - VMContext pointer
;   rsi - Index from top (0 = top)
;   rdx - Output Value pointer
;
; Returns:
;   0 on success, ERR_STACK_UNDERFLOW if index out of range
;
global vm_peek_stack
vm_peek_stack:
    ; Calculate address
    mov rax, [rdi + VMContext.data_sp]
    inc rsi                         ; index + 1
    imul rsi, VALUE_SIZE
    sub rax, rsi                    ; address = sp - (index+1) * VALUE_SIZE

    ; Check bounds
    cmp rax, [rdi + VMContext.data_base]
    jl .underflow

    ; Copy value to output
    mov rcx, VALUE_SIZE / 8
    mov rsi, rax
    mov rdi, rdx
    rep movsq

    xor eax, eax                    ; return 0
    ret

.underflow:
    mov eax, ERR_STACK_UNDERFLOW
    ret

; vm_get_insn_count - Get instruction count
;
; C prototype:
;   uint64_t vm_get_insn_count(VMContext *ctx);
;
global vm_get_insn_count
vm_get_insn_count:
    mov rax, [rdi + VMContext.insn_count]
    ret

; vm_get_error - Get last error code
;
; C prototype:
;   int vm_get_error(VMContext *ctx);
;
global vm_get_error
vm_get_error:
    mov eax, [rdi + VMContext.error_code]
    ret

; ============================================================================
; Mark stack as non-executable (modern security requirement)
; ============================================================================
section .note.GNU-stack noalloc noexec nowrite progbits
