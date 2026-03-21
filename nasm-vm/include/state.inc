; state.inc - VM State structure definition
; DTRules NASM VM Implementation
; Shared state structure used across all modules

%ifndef STATE_INC
%define STATE_INC

;-----------------------------------------------------------------------------
; State structure - VM execution state
;
; This structure holds all VM state including:
; - Three stacks (data, entity, control)
; - Heap allocation pointers
; - Bytecode execution state
; - Constant and name pools
;-----------------------------------------------------------------------------
struc State
    ; Data stack (holds 24-byte Values)
    .data_stack:        resq 1  ; Pointer to data stack top
    .data_stack_base:   resq 1  ; Pointer to data stack bottom
    .data_stack_end:    resq 1  ; Pointer to data stack limit

    ; Entity stack (holds entity pointers for context)
    .entity_stack:      resq 1  ; Pointer to entity stack top
    .entity_stack_base: resq 1  ; Pointer to entity stack bottom
    .entity_stack_end:  resq 1  ; Pointer to entity stack limit

    ; Control stack (holds call frames)
    .control_stack:     resq 1  ; Pointer to control stack top
    .control_stack_base: resq 1 ; Pointer to control stack bottom
    .control_stack_end: resq 1  ; Pointer to control stack limit

    ; Heap memory management
    .heap_ptr:          resq 1  ; Current heap allocation pointer
    .heap_base:         resq 1  ; Heap start
    .heap_end:          resq 1  ; Heap limit

    ; Bytecode execution state
    .bytecode:          resq 1  ; Current bytecode pointer (IP)
    .bytecode_end:      resq 1  ; End of bytecode

    ; Constant pools (for bytecode)
    .constant_pool:     resq 1  ; Pointer to constant pool (array of Values)
    .constant_count:    resq 1  ; Number of constants in pool
    .name_pool:         resq 1  ; Pointer to name pool (array of name pointers)
    .name_count:        resq 1  ; Number of names in pool

    ; Name interning
    .name_table:        resq 1  ; Name interning hash table

    ; Error and status
    .error:             resd 1  ; Last error code
    .halted:            resd 1  ; VM halted flag

    ; Trace buffer (for recording execution)
    .trace_buffer:      resq 1  ; Pointer to trace buffer
    .trace_pos:         resq 1  ; Current position in trace buffer
    .trace_end:         resq 1  ; End of trace buffer
    .trace_enabled:     resb 1  ; Trace recording enabled

    ; Debug settings
    .debug_level:       resq 1  ; Debug level (0=off)
endstruc

;-----------------------------------------------------------------------------
; Register Conventions
;
; The VM uses callee-saved registers for stack pointers:
; | Register | Purpose                              |
; |----------|--------------------------------------|
; | r12      | Data stack pointer (preserved)       |
; | r13      | Entity stack pointer (preserved)     |
; | r14      | Control stack pointer (preserved)    |
; | r15      | State pointer (preserved)            |
; | rbx      | Bytecode instruction pointer         |
; | rdi/rsi  | Function arguments                   |
; | rax      | Return value                         |
;
; Stack pointers grow DOWNWARD. Empty stack has pointer at base.
; Push: decrement pointer, then write
; Pop: read, then increment pointer
;-----------------------------------------------------------------------------

%endif ; STATE_INC
