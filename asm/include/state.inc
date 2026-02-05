; state.inc - VM State structure definition
; DTRules Assembly Implementation
; Shared state structure used across all modules

%ifndef STATE_INC
%define STATE_INC

;-----------------------------------------------------------------------------
; State structure - VM execution state
;-----------------------------------------------------------------------------
struc State
    .data_stack:        resq 1  ; Pointer to data stack top
    .data_stack_base:   resq 1  ; Pointer to data stack bottom
    .data_stack_end:    resq 1  ; Pointer to data stack limit

    .entity_stack:      resq 1  ; Pointer to entity stack top
    .entity_stack_base: resq 1  ; Pointer to entity stack bottom
    .entity_stack_end:  resq 1  ; Pointer to entity stack limit

    .control_stack:     resq 1  ; Pointer to control stack top
    .control_stack_base: resq 1 ; Pointer to control stack bottom
    .control_stack_end: resq 1  ; Pointer to control stack limit

    .heap_ptr:          resq 1  ; Current heap allocation pointer
    .heap_base:         resq 1  ; Heap start
    .heap_end:          resq 1  ; Heap limit

    .bytecode:          resq 1  ; Current bytecode pointer
    .bytecode_end:      resq 1  ; End of bytecode

    .constant_pool:     resq 1  ; Pointer to constant pool (array of Values)
    .constant_count:    resq 1  ; Number of constants in pool
    .name_pool:         resq 1  ; Pointer to name pool (array of name pointers)
    .name_count:        resq 1  ; Number of names in pool

    .name_table:        resq 1  ; Name interning hash table

    .error:             resd 1  ; Last error code
    .halted:            resd 1  ; VM halted flag

    .argc:              resq 1  ; Command line argument count
    .argv:              resq 1  ; Command line arguments pointer

    .error_msg:         resq 1  ; Error message pointer
    .trace:             resb 1  ; Trace flag
    .debug_level:       resq 1  ; Debug level
endstruc

%endif ; STATE_INC
