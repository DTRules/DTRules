; state.inc - VM State structure definition
; DTRules NASM VM Implementation

%ifndef STATE_INC
%define STATE_INC

;-----------------------------------------------------------------------------
; State structure - VM execution state
;-----------------------------------------------------------------------------
struc State
    .data_stack:        resq 1  ; Pointer to data stack top
    .data_stack_base:   resq 1  ; Pointer to data stack bottom
    .data_stack_end:    resq 1  ; Pointer to data stack limit

    .bytecode:          resq 1  ; Current bytecode pointer (PC)
    .bytecode_end:      resq 1  ; End of bytecode

    .error:             resd 1  ; Last error code
    .halted:            resd 1  ; VM halted flag
endstruc

%endif ; STATE_INC
