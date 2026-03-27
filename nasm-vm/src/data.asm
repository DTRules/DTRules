; data.asm - DTRules NASM VM Data Segment
; Global state and stack storage

bits 64
default rel

%include "include/constants.inc"
%include "include/state.inc"

;-----------------------------------------------------------------------------
; Exported symbols
;-----------------------------------------------------------------------------
global state
global data_stack

;-----------------------------------------------------------------------------
section .bss
;-----------------------------------------------------------------------------

; VM State structure
align 16
state: resb State_size

; Data stack (1024 values * 24 bytes = 24KB)
align 16
data_stack: resb STACK_SIZE_BYTES
