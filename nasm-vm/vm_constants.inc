; DTRules NASM VM Constants
; Copyright 2024 Paul Snow
; Licensed under Apache License 2.0
;
; This file defines constants used across the VM

%ifndef VM_CONSTANTS_INC
%define VM_CONSTANTS_INC

; ============================================================================
; Configuration Constants
; ============================================================================

%define STACK_LIMIT         1000        ; Maximum stack depth
%define VALUE_SIZE          24          ; Size of Value struct in bytes
%define BYTECODE_MAX_SIZE   65536       ; Maximum bytecode size

; ============================================================================
; Value Type Tags (matching Go VTag* constants)
; ============================================================================

%define VTAG_NULL           0
%define VTAG_INTEGER        1
%define VTAG_DOUBLE         2
%define VTAG_BOOLEAN        3
%define VTAG_STRING         4
%define VTAG_NAME           5
%define VTAG_ARRAY          6
%define VTAG_ENTITY         7
%define VTAG_OBJECT         8

; ============================================================================
; State Flags (matching Go state flags)
; ============================================================================

%define STATE_DEBUG         0x00000001
%define STATE_TRACE         0x00000002
%define STATE_ECHO          0x00000004
%define STATE_VERBOSE       0x00000008

; ============================================================================
; Bytecode Opcodes (matching Go Opcode constants)
; ============================================================================

; Stack operations (0-19)
%define OP_NOP              0
%define OP_PUSH             1
%define OP_PUSH_INT         2
%define OP_POP              3
%define OP_DUP              4
%define OP_SWAP             5
%define OP_ROT              6
%define OP_ROLL             7
%define OP_INDEX            8
%define OP_CLEAR            9
%define OP_MARK             10

; Arithmetic (20-29)
%define OP_ADD              20
%define OP_SUB              21
%define OP_MUL              22
%define OP_DIV              23
%define OP_MOD              24
%define OP_NEG              25
%define OP_ABS              26
%define OP_INC              27
%define OP_DEC              28

; Comparison (30-39)
%define OP_EQ               30
%define OP_NE               31
%define OP_LT               32
%define OP_LE               33
%define OP_GT               34
%define OP_GE               35

; Boolean (40-49) - PRIMARY FOCUS FOR ISSUE #166
%define OP_AND              40
%define OP_OR               41
%define OP_NOT              42
%define OP_XOR              43
%define OP_ISNULL           44
%define OP_NOTNULL          45
%define OP_BEQ              46  ; Boolean equals

; Control flow (50-59)
%define OP_EXEC             50
%define OP_IF               51
%define OP_IF_ELSE          52
%define OP_WHILE            53
%define OP_FOR              54
%define OP_FOR_ALL          55
%define OP_RETURN           56
%define OP_JUMP             57
%define OP_JUMP_IF          58
%define OP_CALL             59

; Entity operations (60-69)
%define OP_ENTITY_PUSH      60
%define OP_ENTITY_POP       61
%define OP_DEF              62
%define OP_LOOKUP           63
%define OP_NEW_ENTITY       64

; Array operations (70-79)
%define OP_NEW_ARRAY        70
%define OP_ADD_TO           71
%define OP_LENGTH           72
%define OP_GET              73
%define OP_PUT              74

; Table operations (80-89)
%define OP_NEW_TABLE        80
%define OP_TABLE_GET        81
%define OP_TABLE_PUT        82

; String operations (90-99)
%define OP_CONCAT           90
%define OP_SUBSTRING        91

; Constants (100-109)
%define OP_PUSH_TRUE        100
%define OP_PUSH_FALSE       101
%define OP_PUSH_NULL        102
%define OP_PUSH_ZERO        103
%define OP_PUSH_ONE         104

; Extended opcodes (200+)
%define OP_OPERATOR         200
%define OP_CONSTANT         201
%define OP_NAME             202

; Halt/End execution
%define OP_HALT             255

; ============================================================================
; Error Codes
; ============================================================================

%define ERR_NONE            0
%define ERR_STACK_OVERFLOW  1
%define ERR_STACK_UNDERFLOW 2
%define ERR_INVALID_OPCODE  3
%define ERR_TYPE_MISMATCH   4
%define ERR_DIV_ZERO        5
%define ERR_OUT_OF_BOUNDS   6
%define ERR_NULL_PTR        7
%define ERR_NOT_IMPLEMENTED 8

%endif ; VM_CONSTANTS_INC
