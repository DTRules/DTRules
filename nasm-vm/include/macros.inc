; macros.inc - Assembly macros for DTRules NASM VM
; Helper macros for common patterns

%ifndef MACROS_INC
%define MACROS_INC

;-----------------------------------------------------------------------------
; System calls (Linux x86-64)
;-----------------------------------------------------------------------------
%define SYS_WRITE   1
%define SYS_EXIT    60
%define STDOUT      1

;-----------------------------------------------------------------------------
; Function prologue/epilogue
;-----------------------------------------------------------------------------
%macro PROLOGUE 0
    push rbp
    mov rbp, rsp
%endmacro

%macro EPILOGUE 0
    pop rbp
    ret
%endmacro

;-----------------------------------------------------------------------------
; Print string (null-terminated)
; Usage: PRINT_STRING label
;-----------------------------------------------------------------------------
%macro PRINT_STRING 1
    lea rdi, [%1]
    call print_string
%endmacro

%endif ; MACROS_INC
