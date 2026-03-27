; DTRules NASM VM - Value Type Definitions
; Copyright 2024 Paul Snow
; Licensed under the Apache License, Version 2.0

; Value type tags - must match go/pkg/dtrules/value.go
%define VTAG_NULL       0
%define VTAG_INTEGER    1
%define VTAG_DOUBLE     2
%define VTAG_BOOLEAN    3
%define VTAG_STRING     4
%define VTAG_NAME       5
%define VTAG_ARRAY      6
%define VTAG_ENTITY     7
%define VTAG_OBJECT     8

; Value structure layout (24 bytes, matching Go's Value struct)
; Offset 0:  tag    (1 byte)  - type discriminator
; Offset 1:  padding (7 bytes) - alignment padding
; Offset 8:  num    (8 bytes) - numeric value (int64 or float64 bits)
; Offset 16: ptr    (8 bytes) - pointer to complex types

%define VALUE_SIZE      24
%define VALUE_TAG_OFF   0
%define VALUE_NUM_OFF   8
%define VALUE_PTR_OFF   16

; Stack entry size (same as value size)
%define STACK_ENTRY_SIZE    VALUE_SIZE

; Error codes
%define ERR_NONE            0
%define ERR_STACK_UNDERFLOW 1
%define ERR_STACK_OVERFLOW  2
%define ERR_TYPE_MISMATCH   3
%define ERR_DIV_BY_ZERO     4
%define ERR_INVALID_OPCODE  5
%define ERR_OUT_OF_BOUNDS   6

; Boolean values
%define BOOL_FALSE      0
%define BOOL_TRUE       1
