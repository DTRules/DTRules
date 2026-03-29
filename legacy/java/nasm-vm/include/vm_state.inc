; DTRules NASM VM - VM State Structure
; Copyright 2024 Paul Snow
; Licensed under the Apache License, Version 2.0

%include "value.inc"

; VM State structure layout
; This structure is passed from C/Go and contains all VM execution context

struc VMState
    .bytecode       resq 1      ; Pointer to bytecode array
    .bytecode_len   resq 1      ; Length of bytecode
    .pc             resq 1      ; Program counter (current position in bytecode)

    .stack          resq 1      ; Pointer to data stack base
    .stack_size     resq 1      ; Maximum stack size (number of entries)
    .sp             resq 1      ; Stack pointer (current top, grows up)

    .constants      resq 1      ; Pointer to constants pool
    .const_count    resq 1      ; Number of constants

    .names          resq 1      ; Pointer to names pool
    .name_count     resq 1      ; Number of names

    .trace_buf      resq 1      ; Pointer to trace buffer (optional)
    .trace_pos      resq 1      ; Current position in trace buffer
    .trace_size     resq 1      ; Size of trace buffer

    .error_code     resq 1      ; Error code (0 = success)
    .error_pc       resq 1      ; PC where error occurred
endstruc

; Offsets for easier access
%define VMSTATE_BYTECODE    0
%define VMSTATE_BC_LEN      8
%define VMSTATE_PC          16
%define VMSTATE_STACK       24
%define VMSTATE_STACK_SIZE  32
%define VMSTATE_SP          40
%define VMSTATE_CONSTANTS   48
%define VMSTATE_CONST_COUNT 56
%define VMSTATE_NAMES       64
%define VMSTATE_NAME_COUNT  72
%define VMSTATE_TRACE_BUF   80
%define VMSTATE_TRACE_POS   88
%define VMSTATE_TRACE_SIZE  96
%define VMSTATE_ERROR_CODE  104
%define VMSTATE_ERROR_PC    112
%define VMSTATE_SIZE        120
