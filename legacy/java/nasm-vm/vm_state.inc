; DTRules NASM VM State Structures
; Copyright 2024 Paul Snow
; Licensed under Apache License 2.0
;
; This file defines the state structures and macros

%ifndef VM_STATE_INC
%define VM_STATE_INC

%include "vm_constants.inc"

; ============================================================================
; Value Structure (24 bytes)
; ============================================================================
; Matches Go Value struct layout:
;   tag:  1 byte  - type discriminator
;   _:    7 bytes - padding
;   num:  8 bytes - numeric value (int64 or float64 bits)
;   ptr:  8 bytes - pointer to complex types

struc Value
    .tag:       resb 1
    .padding:   resb 7
    .num:       resq 1
    .ptr:       resq 1
endstruc

; ============================================================================
; VM Context Structure
; ============================================================================
; Contains all VM state needed for execution

struc VMContext
    ; Stack pointers (current position)
    .data_sp:       resq 1      ; Current data stack position
    .entity_sp:     resq 1      ; Current entity stack position
    .ctrl_sp:       resq 1      ; Current control stack position

    ; Stack bases (start of allocated memory)
    .data_base:     resq 1      ; Data stack base address
    .entity_base:   resq 1      ; Entity stack base address
    .ctrl_base:     resq 1      ; Control stack base address

    ; Stack limits (maximum positions)
    .data_limit:    resq 1      ; Data stack limit
    .entity_limit:  resq 1      ; Entity stack limit
    .ctrl_limit:    resq 1      ; Control stack limit

    ; Bytecode execution
    .bytecode:      resq 1      ; Bytecode buffer pointer
    .bytecode_len:  resq 1      ; Bytecode length
    .pc:            resq 1      ; Program counter (offset into bytecode)

    ; Constant pools
    .constants:     resq 1      ; Constants array pointer
    .const_count:   resq 1      ; Number of constants
    .names:         resq 1      ; Names array pointer
    .name_count:    resq 1      ; Number of names

    ; Jump table
    .jump_table:    resq 1      ; Jump table base address

    ; State flags
    .state_flags:   resd 1      ; DEBUG, TRACE, ECHO, VERBOSE
    .error_code:    resd 1      ; Last error code

    ; Frame management
    .current_frame: resq 1      ; Current frame index
    .frames:        resq 1      ; Frame stack pointer
    .frame_count:   resq 1      ; Number of frames

    ; Trace buffer (for recording execution)
    .trace_buf:     resq 1      ; Trace buffer pointer
    .trace_pos:     resq 1      ; Current trace position
    .trace_limit:   resq 1      ; Trace buffer limit

    ; Session reference (for callbacks to Go runtime)
    .session:       resq 1      ; Opaque session pointer

    ; Operator table (for OP_OPERATOR dispatch)
    .op_table:      resq 1      ; Operator handler table
    .op_count:      resq 1      ; Number of operators

    ; Statistics
    .insn_count:    resq 1      ; Instructions executed
    .cycle_count:   resq 1      ; Estimated cycles
endstruc

; ============================================================================
; Trace Entry Structure (for execution recording)
; ============================================================================

struc TraceEntry
    .opcode:        resb 1      ; Opcode executed
    .padding:       resb 7
    .pc:            resq 1      ; PC before execution
    .value:         resb VALUE_SIZE ; Top of stack after
endstruc

%endif ; VM_STATE_INC
