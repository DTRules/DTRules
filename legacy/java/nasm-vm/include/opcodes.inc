; DTRules NASM VM - Opcode Definitions
; Copyright 2024 Paul Snow
; Licensed under the Apache License, Version 2.0

; Opcode constants - must match go/pkg/dtrules/bytecode.go

; Stack operations
%define OP_NOP       0
%define OP_PUSH      1    ; Push constant from pool
%define OP_PUSH_INT  2    ; Push immediate integer (followed by varint)
%define OP_POP       3    ; Pop and discard
%define OP_DUP       4    ; Duplicate top
%define OP_SWAP      5    ; Swap top two
%define OP_ROT       6    ; Rotate top three
%define OP_ROLL      7    ; Roll n items
%define OP_INDEX     8    ; Copy nth item to top
%define OP_CLEAR     9    ; Clear stack to mark
%define OP_MARK      10   ; Push mark

; Arithmetic operations
%define OP_ADD       20   ; a + b
%define OP_SUB       21   ; a - b
%define OP_MUL       22   ; a * b
%define OP_DIV       23   ; a / b
%define OP_MOD       24   ; a % b
%define OP_NEG       25   ; -a
%define OP_ABS       26   ; |a|
%define OP_INC       27   ; a + 1
%define OP_DEC       28   ; a - 1

; Comparison operations
%define OP_EQ        30   ; a == b
%define OP_NE        31   ; a != b
%define OP_LT        32   ; a < b
%define OP_LE        33   ; a <= b
%define OP_GT        34   ; a > b
%define OP_GE        35   ; a >= b

; Boolean operations
%define OP_AND       40   ; a && b
%define OP_OR        41   ; a || b
%define OP_NOT       42   ; !a
%define OP_XOR       43   ; a ^ b

; Control flow
%define OP_EXEC      50   ; Execute top of stack
%define OP_IF        51   ; Conditional execution
%define OP_IFELSE    52   ; if-else
%define OP_WHILE     53   ; while loop
%define OP_FOR       54   ; for loop
%define OP_FORALL    55   ; iterate array
%define OP_RETURN    56   ; Return from procedure
%define OP_JUMP      57   ; Unconditional jump
%define OP_JUMP_IF   58   ; Conditional jump
%define OP_CALL      59   ; Call operator by index

; Entity operations
%define OP_ENTITY_PUSH  60
%define OP_ENTITY_POP   61
%define OP_DEF          62
%define OP_LOOKUP       63
%define OP_NEW_ENTITY   64

; Array operations
%define OP_NEW_ARRAY    70
%define OP_ADD_TO       71
%define OP_LENGTH       72
%define OP_GET          73
%define OP_PUT          74

; Table operations
%define OP_NEW_TABLE    80
%define OP_TABLE_GET    81
%define OP_TABLE_PUT    82

; String operations
%define OP_CONCAT       90
%define OP_SUBSTRING    91

; Constants
%define OP_PUSH_TRUE    100
%define OP_PUSH_FALSE   101
%define OP_PUSH_NULL    102
%define OP_PUSH_ZERO    103
%define OP_PUSH_ONE     104

; Extended opcodes
%define OP_OPERATOR     200
%define OP_CONSTANT     201
%define OP_NAME         202

; Halt (internal)
%define OP_HALT         255

; Maximum opcode value for jump table
%define MAX_OPCODE      255
