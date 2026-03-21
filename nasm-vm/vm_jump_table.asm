; DTRules NASM VM Jump Table and Opcode Handlers
; Copyright 2024 Paul Snow
; Licensed under Apache License 2.0
;
; This file implements:
;   1. The jump table for opcode dispatch
;   2. Handlers for each opcode
;
; Design principles:
;   - Jump table dispatch: jmp [table + opcode*8]
;   - Each handler ends with dispatch macro (jmp back to dispatcher)
;   - No ret in handlers - pure threaded interpretation
;   - Trace recording writes to buffer, not function calls

%include "vm_core.asm"

; Note: vm_dispatch, vm_error_return, vm_exit_success are declared
; as extern in vm_core.asm (needed for macros)

section .data

; ============================================================================
; Jump Table (256 entries, 8 bytes each = 2KB)
; ============================================================================

align 64
global vm_jump_table
vm_jump_table:
    ; Stack operations (0-19)
    dq op_nop               ; 0: OP_NOP
    dq op_push              ; 1: OP_PUSH
    dq op_push_int          ; 2: OP_PUSH_INT
    dq op_pop               ; 3: OP_POP
    dq op_dup               ; 4: OP_DUP
    dq op_swap              ; 5: OP_SWAP
    dq op_rot               ; 6: OP_ROT
    dq op_roll              ; 7: OP_ROLL
    dq op_index             ; 8: OP_INDEX
    dq op_clear             ; 9: OP_CLEAR
    dq op_mark              ; 10: OP_MARK
    dq op_invalid           ; 11: unused
    dq op_invalid           ; 12: unused
    dq op_invalid           ; 13: unused
    dq op_invalid           ; 14: unused
    dq op_invalid           ; 15: unused
    dq op_invalid           ; 16: unused
    dq op_invalid           ; 17: unused
    dq op_invalid           ; 18: unused
    dq op_invalid           ; 19: unused

    ; Arithmetic (20-29)
    dq op_add               ; 20: OP_ADD
    dq op_sub               ; 21: OP_SUB
    dq op_mul               ; 22: OP_MUL
    dq op_div               ; 23: OP_DIV
    dq op_mod               ; 24: OP_MOD
    dq op_neg               ; 25: OP_NEG
    dq op_abs               ; 26: OP_ABS
    dq op_inc               ; 27: OP_INC
    dq op_dec               ; 28: OP_DEC
    dq op_invalid           ; 29: unused

    ; Comparison (30-39)
    dq op_eq                ; 30: OP_EQ
    dq op_ne                ; 31: OP_NE
    dq op_lt                ; 32: OP_LT
    dq op_le                ; 33: OP_LE
    dq op_gt                ; 34: OP_GT
    dq op_ge                ; 35: OP_GE
    dq op_invalid           ; 36: unused
    dq op_invalid           ; 37: unused
    dq op_invalid           ; 38: unused
    dq op_invalid           ; 39: unused

    ; Boolean (40-49)
    dq op_and               ; 40: OP_AND
    dq op_or                ; 41: OP_OR
    dq op_not               ; 42: OP_NOT
    dq op_xor               ; 43: OP_XOR
    dq op_invalid           ; 44: unused
    dq op_invalid           ; 45: unused
    dq op_invalid           ; 46: unused
    dq op_invalid           ; 47: unused
    dq op_invalid           ; 48: unused
    dq op_invalid           ; 49: unused

    ; Control flow (50-59)
    dq op_exec              ; 50: OP_EXEC
    dq op_if                ; 51: OP_IF
    dq op_if_else           ; 52: OP_IF_ELSE
    dq op_while             ; 53: OP_WHILE
    dq op_for               ; 54: OP_FOR
    dq op_for_all           ; 55: OP_FOR_ALL
    dq op_return            ; 56: OP_RETURN
    dq op_jump              ; 57: OP_JUMP
    dq op_jump_if           ; 58: OP_JUMP_IF
    dq op_call              ; 59: OP_CALL

    ; Entity operations (60-69)
    dq op_entity_push       ; 60: OP_ENTITY_PUSH
    dq op_entity_pop        ; 61: OP_ENTITY_POP
    dq op_def               ; 62: OP_DEF
    dq op_lookup            ; 63: OP_LOOKUP
    dq op_new_entity        ; 64: OP_NEW_ENTITY
    dq op_invalid           ; 65: unused
    dq op_invalid           ; 66: unused
    dq op_invalid           ; 67: unused
    dq op_invalid           ; 68: unused
    dq op_invalid           ; 69: unused

    ; Array operations (70-79)
    dq op_new_array         ; 70: OP_NEW_ARRAY
    dq op_add_to            ; 71: OP_ADD_TO
    dq op_length            ; 72: OP_LENGTH
    dq op_get               ; 73: OP_GET
    dq op_put               ; 74: OP_PUT
    dq op_invalid           ; 75: unused
    dq op_invalid           ; 76: unused
    dq op_invalid           ; 77: unused
    dq op_invalid           ; 78: unused
    dq op_invalid           ; 79: unused

    ; Table operations (80-89)
    dq op_new_table         ; 80: OP_NEW_TABLE
    dq op_table_get         ; 81: OP_TABLE_GET
    dq op_table_put         ; 82: OP_TABLE_PUT
    dq op_invalid           ; 83: unused
    dq op_invalid           ; 84: unused
    dq op_invalid           ; 85: unused
    dq op_invalid           ; 86: unused
    dq op_invalid           ; 87: unused
    dq op_invalid           ; 88: unused
    dq op_invalid           ; 89: unused

    ; String operations (90-99)
    dq op_concat            ; 90: OP_CONCAT
    dq op_substring         ; 91: OP_SUBSTRING
    dq op_invalid           ; 92: unused
    dq op_invalid           ; 93: unused
    dq op_invalid           ; 94: unused
    dq op_invalid           ; 95: unused
    dq op_invalid           ; 96: unused
    dq op_invalid           ; 97: unused
    dq op_invalid           ; 98: unused
    dq op_invalid           ; 99: unused

    ; Constants (100-109)
    dq op_push_true         ; 100: OP_PUSH_TRUE
    dq op_push_false        ; 101: OP_PUSH_FALSE
    dq op_push_null         ; 102: OP_PUSH_NULL
    dq op_push_zero         ; 103: OP_PUSH_ZERO
    dq op_push_one          ; 104: OP_PUSH_ONE
    dq op_invalid           ; 105: unused
    dq op_invalid           ; 106: unused
    dq op_invalid           ; 107: unused
    dq op_invalid           ; 108: unused
    dq op_invalid           ; 109: unused

    ; Fill 110-199 with invalid
    %rep 90
    dq op_invalid
    %endrep

    ; Extended opcodes (200-202)
    dq op_operator          ; 200: OP_OPERATOR
    dq op_constant          ; 201: OP_CONSTANT
    dq op_name              ; 202: OP_NAME

    ; Fill 203-254 with invalid
    %rep 52
    dq op_invalid
    %endrep

    ; 255: OP_HALT
    dq op_halt

section .text

; ============================================================================
; Invalid Opcode Handler
; ============================================================================

op_invalid:
    mov dword [r12 + VMContext.error_code], ERR_INVALID_OPCODE
    jmp vm_error_return

; ============================================================================
; Stack Operations
; ============================================================================

; OP_NOP (0) - No operation
op_nop:
    dispatch

; OP_PUSH (1) - Push constant from pool (followed by varint index)
op_push:
    read_varint                     ; index in rax

    ; Get constant from pool
    mov rdi, [r12 + VMContext.constants]
    imul rax, VALUE_SIZE
    add rdi, rax                    ; rdi = &constants[index]

    ; Check stack limit
    mov rax, [r12 + VMContext.data_sp]
    cmp rax, [r12 + VMContext.data_limit]
    jge .overflow

    ; Copy value to stack
    mov rcx, VALUE_SIZE / 8
    mov rsi, rdi
    mov rdi, rax
    rep movsq

    add rax, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rax
    dispatch

.overflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_OVERFLOW
    jmp vm_error_return

; OP_PUSH_INT (2) - Push immediate signed integer (varint)
op_push_int:
    read_varint                     ; value in rax

    ; Check stack limit
    mov rdi, [r12 + VMContext.data_sp]
    cmp rdi, [r12 + VMContext.data_limit]
    jge .overflow

    ; Store integer value
    mov byte [rdi + Value.tag], VTAG_INTEGER
    mov qword [rdi + Value.num], rax
    mov qword [rdi + Value.ptr], 0

    add rdi, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rdi
    dispatch

.overflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_OVERFLOW
    jmp vm_error_return

; OP_POP (3) - Pop and discard top
op_pop:
    mov rax, [r12 + VMContext.data_sp]
    sub rax, VALUE_SIZE
    cmp rax, [r12 + VMContext.data_base]
    jl .underflow

    mov [r12 + VMContext.data_sp], rax
    dispatch

.underflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_UNDERFLOW
    jmp vm_error_return

; OP_DUP (4) - Duplicate top of stack
op_dup:
    mov rsi, [r12 + VMContext.data_sp]
    sub rsi, VALUE_SIZE
    cmp rsi, [r12 + VMContext.data_base]
    jl .underflow

    mov rdi, [r12 + VMContext.data_sp]
    cmp rdi, [r12 + VMContext.data_limit]
    jge .overflow

    ; Copy top value
    mov rcx, VALUE_SIZE / 8
    push rsi
    mov rax, rsi                    ; save source
    rep movsq
    pop rsi

    add qword [r12 + VMContext.data_sp], VALUE_SIZE
    dispatch

.underflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_UNDERFLOW
    jmp vm_error_return

.overflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_OVERFLOW
    jmp vm_error_return

; OP_SWAP (5) - Swap top two values
op_swap:
    mov rax, [r12 + VMContext.data_sp]
    sub rax, VALUE_SIZE * 2
    cmp rax, [r12 + VMContext.data_base]
    jl .underflow

    ; rax points to second from top
    ; rax + VALUE_SIZE points to top

    ; Swap the values (24 bytes each)
    ; We need to swap 3 qwords
    mov rdi, [rax + 0]              ; first qword of a
    mov rsi, [rax + VALUE_SIZE + 0] ; first qword of b
    mov [rax + 0], rsi
    mov [rax + VALUE_SIZE + 0], rdi

    mov rdi, [rax + 8]              ; second qword of a
    mov rsi, [rax + VALUE_SIZE + 8] ; second qword of b
    mov [rax + 8], rsi
    mov [rax + VALUE_SIZE + 8], rdi

    mov rdi, [rax + 16]             ; third qword of a
    mov rsi, [rax + VALUE_SIZE + 16] ; third qword of b
    mov [rax + 16], rsi
    mov [rax + VALUE_SIZE + 16], rdi

    dispatch

.underflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_UNDERFLOW
    jmp vm_error_return

; OP_ROT (6) - Rotate top three: (a b c -- b c a)
op_rot:
    mov rax, [r12 + VMContext.data_sp]
    sub rax, VALUE_SIZE * 3
    cmp rax, [r12 + VMContext.data_base]
    jl .underflow

    ; a = [rax], b = [rax + VALUE_SIZE], c = [rax + 2*VALUE_SIZE]
    ; Result: b c a

    ; Save a to temp (on stack)
    sub rsp, VALUE_SIZE
    mov rsi, rax
    mov rdi, rsp
    mov rcx, VALUE_SIZE / 8
    rep movsq

    ; Move b to a position
    lea rsi, [rax + VALUE_SIZE]
    mov rdi, rax
    mov rcx, VALUE_SIZE / 8
    rep movsq

    ; Move c to b position
    lea rsi, [rax + VALUE_SIZE * 2]
    lea rdi, [rax + VALUE_SIZE]
    mov rcx, VALUE_SIZE / 8
    rep movsq

    ; Move saved a to c position
    mov rsi, rsp
    lea rdi, [rax + VALUE_SIZE * 2]
    mov rcx, VALUE_SIZE / 8
    rep movsq

    add rsp, VALUE_SIZE
    dispatch

.underflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_UNDERFLOW
    jmp vm_error_return

; OP_ROLL (7) - Roll n items (TODO: implement)
op_roll:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

; OP_INDEX (8) - Copy nth item to top (TODO: implement)
op_index:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

; OP_CLEAR (9) - Clear stack to mark (TODO: implement)
op_clear:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

; OP_MARK (10) - Push mark (TODO: implement)
op_mark:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

; ============================================================================
; Arithmetic Operations
; ============================================================================

; Helper macro to pop two integers
; After this: rbx = first operand, rcx = second operand
%macro pop_two_integers 0
    ; Pop second operand
    mov rax, [r12 + VMContext.data_sp]
    sub rax, VALUE_SIZE
    cmp rax, [r12 + VMContext.data_base]
    jl %%underflow

    ; Check type
    cmp byte [rax + Value.tag], VTAG_INTEGER
    jne %%type_error

    mov rcx, [rax + Value.num]      ; second operand in rcx
    mov [r12 + VMContext.data_sp], rax

    ; Pop first operand
    sub rax, VALUE_SIZE
    cmp rax, [r12 + VMContext.data_base]
    jl %%underflow

    cmp byte [rax + Value.tag], VTAG_INTEGER
    jne %%type_error

    mov rbx, [rax + Value.num]      ; first operand in rbx
    mov [r12 + VMContext.data_sp], rax
    jmp %%done

%%underflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_UNDERFLOW
    jmp vm_error_return

%%type_error:
    mov dword [r12 + VMContext.error_code], ERR_TYPE_MISMATCH
    jmp vm_error_return

%%done:
%endmacro

; Helper macro to push integer result (in rax)
%macro push_integer_result 0
    mov rdi, [r12 + VMContext.data_sp]
    cmp rdi, [r12 + VMContext.data_limit]
    jge %%overflow

    mov byte [rdi + Value.tag], VTAG_INTEGER
    mov qword [rdi + Value.num], rax
    mov qword [rdi + Value.ptr], 0

    add rdi, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rdi
    jmp %%done

%%overflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_OVERFLOW
    jmp vm_error_return

%%done:
%endmacro

; OP_ADD (20) - Add two integers
op_add:
    pop_two_integers                ; rbx = a, rcx = b
    mov rax, rbx
    add rax, rcx
    push_integer_result
    dispatch

; OP_SUB (21) - Subtract: a - b
op_sub:
    pop_two_integers                ; rbx = a, rcx = b
    mov rax, rbx
    sub rax, rcx
    push_integer_result
    dispatch

; OP_MUL (22) - Multiply
op_mul:
    pop_two_integers                ; rbx = a, rcx = b
    mov rax, rbx
    imul rax, rcx
    push_integer_result
    dispatch

; OP_DIV (23) - Divide: a / b
op_div:
    pop_two_integers                ; rbx = a, rcx = b

    ; Check for division by zero
    test rcx, rcx
    jz .div_zero

    mov rax, rbx
    cqo                             ; Sign-extend rax to rdx:rax
    idiv rcx                        ; rax = quotient
    push_integer_result
    dispatch

.div_zero:
    mov dword [r12 + VMContext.error_code], ERR_DIV_ZERO
    jmp vm_error_return

; OP_MOD (24) - Modulo: a % b
op_mod:
    pop_two_integers                ; rbx = a, rcx = b

    test rcx, rcx
    jz .div_zero

    mov rax, rbx
    cqo
    idiv rcx                        ; rdx = remainder
    mov rax, rdx
    push_integer_result
    dispatch

.div_zero:
    mov dword [r12 + VMContext.error_code], ERR_DIV_ZERO
    jmp vm_error_return

; OP_NEG (25) - Negate: -a
op_neg:
    mov rax, [r12 + VMContext.data_sp]
    sub rax, VALUE_SIZE
    cmp rax, [r12 + VMContext.data_base]
    jl .underflow

    cmp byte [rax + Value.tag], VTAG_INTEGER
    jne .type_error

    neg qword [rax + Value.num]
    dispatch

.underflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_UNDERFLOW
    jmp vm_error_return

.type_error:
    mov dword [r12 + VMContext.error_code], ERR_TYPE_MISMATCH
    jmp vm_error_return

; OP_ABS (26) - Absolute value
op_abs:
    mov rax, [r12 + VMContext.data_sp]
    sub rax, VALUE_SIZE
    cmp rax, [r12 + VMContext.data_base]
    jl .underflow

    cmp byte [rax + Value.tag], VTAG_INTEGER
    jne .type_error

    mov rbx, [rax + Value.num]
    test rbx, rbx
    jns .done                       ; Already positive
    neg rbx
    mov [rax + Value.num], rbx

.done:
    dispatch

.underflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_UNDERFLOW
    jmp vm_error_return

.type_error:
    mov dword [r12 + VMContext.error_code], ERR_TYPE_MISMATCH
    jmp vm_error_return

; OP_INC (27) - Increment: a + 1
op_inc:
    mov rax, [r12 + VMContext.data_sp]
    sub rax, VALUE_SIZE
    cmp rax, [r12 + VMContext.data_base]
    jl .underflow

    cmp byte [rax + Value.tag], VTAG_INTEGER
    jne .type_error

    inc qword [rax + Value.num]
    dispatch

.underflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_UNDERFLOW
    jmp vm_error_return

.type_error:
    mov dword [r12 + VMContext.error_code], ERR_TYPE_MISMATCH
    jmp vm_error_return

; OP_DEC (28) - Decrement: a - 1
op_dec:
    mov rax, [r12 + VMContext.data_sp]
    sub rax, VALUE_SIZE
    cmp rax, [r12 + VMContext.data_base]
    jl .underflow

    cmp byte [rax + Value.tag], VTAG_INTEGER
    jne .type_error

    dec qword [rax + Value.num]
    dispatch

.underflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_UNDERFLOW
    jmp vm_error_return

.type_error:
    mov dword [r12 + VMContext.error_code], ERR_TYPE_MISMATCH
    jmp vm_error_return

; ============================================================================
; Comparison Operations
; ============================================================================

; Helper macro to push boolean result
%macro push_bool_result 1  ; %1 = 0 or 1
    mov rdi, [r12 + VMContext.data_sp]
    cmp rdi, [r12 + VMContext.data_limit]
    jge %%overflow

    mov byte [rdi + Value.tag], VTAG_BOOLEAN
    mov qword [rdi + Value.num], %1
    mov qword [rdi + Value.ptr], 0

    add rdi, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rdi
    jmp %%done

%%overflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_OVERFLOW
    jmp vm_error_return

%%done:
%endmacro

; OP_EQ (30) - Equal: a == b
op_eq:
    pop_two_integers                ; rbx = a, rcx = b
    xor eax, eax
    cmp rbx, rcx
    sete al
    push_integer_result             ; Using integer result for boolean
    ; Fix: set tag to boolean
    mov rdi, [r12 + VMContext.data_sp]
    mov byte [rdi - VALUE_SIZE + Value.tag], VTAG_BOOLEAN
    dispatch

; OP_NE (31) - Not equal: a != b
op_ne:
    pop_two_integers
    xor eax, eax
    cmp rbx, rcx
    setne al
    push_integer_result
    mov rdi, [r12 + VMContext.data_sp]
    mov byte [rdi - VALUE_SIZE + Value.tag], VTAG_BOOLEAN
    dispatch

; OP_LT (32) - Less than: a < b
op_lt:
    pop_two_integers
    xor eax, eax
    cmp rbx, rcx
    setl al
    push_integer_result
    mov rdi, [r12 + VMContext.data_sp]
    mov byte [rdi - VALUE_SIZE + Value.tag], VTAG_BOOLEAN
    dispatch

; OP_LE (33) - Less than or equal: a <= b
op_le:
    pop_two_integers
    xor eax, eax
    cmp rbx, rcx
    setle al
    push_integer_result
    mov rdi, [r12 + VMContext.data_sp]
    mov byte [rdi - VALUE_SIZE + Value.tag], VTAG_BOOLEAN
    dispatch

; OP_GT (34) - Greater than: a > b
op_gt:
    pop_two_integers
    xor eax, eax
    cmp rbx, rcx
    setg al
    push_integer_result
    mov rdi, [r12 + VMContext.data_sp]
    mov byte [rdi - VALUE_SIZE + Value.tag], VTAG_BOOLEAN
    dispatch

; OP_GE (35) - Greater than or equal: a >= b
op_ge:
    pop_two_integers
    xor eax, eax
    cmp rbx, rcx
    setge al
    push_integer_result
    mov rdi, [r12 + VMContext.data_sp]
    mov byte [rdi - VALUE_SIZE + Value.tag], VTAG_BOOLEAN
    dispatch

; ============================================================================
; Boolean Operations
; ============================================================================

; Helper macro to pop two booleans
%macro pop_two_booleans 0
    ; Pop second operand
    mov rax, [r12 + VMContext.data_sp]
    sub rax, VALUE_SIZE
    cmp rax, [r12 + VMContext.data_base]
    jl %%underflow

    cmp byte [rax + Value.tag], VTAG_BOOLEAN
    jne %%type_error

    mov rcx, [rax + Value.num]      ; second operand (0 or 1)
    mov [r12 + VMContext.data_sp], rax

    ; Pop first operand
    sub rax, VALUE_SIZE
    cmp rax, [r12 + VMContext.data_base]
    jl %%underflow

    cmp byte [rax + Value.tag], VTAG_BOOLEAN
    jne %%type_error

    mov rbx, [rax + Value.num]      ; first operand
    mov [r12 + VMContext.data_sp], rax
    jmp %%done

%%underflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_UNDERFLOW
    jmp vm_error_return

%%type_error:
    mov dword [r12 + VMContext.error_code], ERR_TYPE_MISMATCH
    jmp vm_error_return

%%done:
%endmacro

; OP_AND (40) - Logical AND
op_and:
    pop_two_booleans                ; rbx = a, rcx = b
    and rbx, rcx
    mov rdi, [r12 + VMContext.data_sp]
    cmp rdi, [r12 + VMContext.data_limit]
    jge .overflow

    mov byte [rdi + Value.tag], VTAG_BOOLEAN
    mov qword [rdi + Value.num], rbx
    mov qword [rdi + Value.ptr], 0
    add rdi, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rdi
    dispatch

.overflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_OVERFLOW
    jmp vm_error_return

; OP_OR (41) - Logical OR
op_or:
    pop_two_booleans
    or rbx, rcx
    ; Normalize to 0 or 1
    test rbx, rbx
    setnz bl
    movzx rbx, bl

    mov rdi, [r12 + VMContext.data_sp]
    cmp rdi, [r12 + VMContext.data_limit]
    jge .overflow

    mov byte [rdi + Value.tag], VTAG_BOOLEAN
    mov qword [rdi + Value.num], rbx
    mov qword [rdi + Value.ptr], 0
    add rdi, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rdi
    dispatch

.overflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_OVERFLOW
    jmp vm_error_return

; OP_NOT (42) - Logical NOT
op_not:
    mov rax, [r12 + VMContext.data_sp]
    sub rax, VALUE_SIZE
    cmp rax, [r12 + VMContext.data_base]
    jl .underflow

    cmp byte [rax + Value.tag], VTAG_BOOLEAN
    jne .type_error

    xor qword [rax + Value.num], 1  ; Flip 0 <-> 1
    dispatch

.underflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_UNDERFLOW
    jmp vm_error_return

.type_error:
    mov dword [r12 + VMContext.error_code], ERR_TYPE_MISMATCH
    jmp vm_error_return

; OP_XOR (43) - Logical XOR
op_xor:
    pop_two_booleans
    xor rbx, rcx
    ; Normalize
    test rbx, rbx
    setnz bl
    movzx rbx, bl

    mov rdi, [r12 + VMContext.data_sp]
    cmp rdi, [r12 + VMContext.data_limit]
    jge .overflow

    mov byte [rdi + Value.tag], VTAG_BOOLEAN
    mov qword [rdi + Value.num], rbx
    mov qword [rdi + Value.ptr], 0
    add rdi, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rdi
    dispatch

.overflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_OVERFLOW
    jmp vm_error_return

; ============================================================================
; Control Flow Operations (stubs)
; ============================================================================

op_exec:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

op_if:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

op_if_else:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

op_while:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

op_for:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

op_for_all:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

op_return:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

; OP_JUMP (57) - Unconditional jump
op_jump:
    read_varint                     ; offset in rax
    add [r12 + VMContext.pc], rax
    dispatch

; OP_JUMP_IF (58) - Conditional jump (jump if top is true)
op_jump_if:
    read_varint                     ; offset in rax
    mov rbx, rax                    ; save offset

    ; Pop condition
    mov rax, [r12 + VMContext.data_sp]
    sub rax, VALUE_SIZE
    cmp rax, [r12 + VMContext.data_base]
    jl .underflow

    cmp byte [rax + Value.tag], VTAG_BOOLEAN
    jne .type_error

    mov rcx, [rax + Value.num]      ; condition
    mov [r12 + VMContext.data_sp], rax

    ; Jump if true
    test rcx, rcx
    jz .no_jump
    add [r12 + VMContext.pc], rbx

.no_jump:
    dispatch

.underflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_UNDERFLOW
    jmp vm_error_return

.type_error:
    mov dword [r12 + VMContext.error_code], ERR_TYPE_MISMATCH
    jmp vm_error_return

op_call:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

; ============================================================================
; Entity Operations (stubs)
; ============================================================================

op_entity_push:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

op_entity_pop:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

op_def:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

op_lookup:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

op_new_entity:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

; ============================================================================
; Array Operations (stubs)
; ============================================================================

op_new_array:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

op_add_to:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

op_length:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

op_get:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

op_put:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

; ============================================================================
; Table Operations (stubs)
; ============================================================================

op_new_table:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

op_table_get:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

op_table_put:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

; ============================================================================
; String Operations (stubs)
; ============================================================================

op_concat:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

op_substring:
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

; ============================================================================
; Constant Push Operations
; ============================================================================

; OP_PUSH_TRUE (100)
op_push_true:
    mov rdi, [r12 + VMContext.data_sp]
    cmp rdi, [r12 + VMContext.data_limit]
    jge .overflow

    mov byte [rdi + Value.tag], VTAG_BOOLEAN
    mov qword [rdi + Value.num], 1
    mov qword [rdi + Value.ptr], 0

    add rdi, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rdi
    dispatch

.overflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_OVERFLOW
    jmp vm_error_return

; OP_PUSH_FALSE (101)
op_push_false:
    mov rdi, [r12 + VMContext.data_sp]
    cmp rdi, [r12 + VMContext.data_limit]
    jge .overflow

    mov byte [rdi + Value.tag], VTAG_BOOLEAN
    mov qword [rdi + Value.num], 0
    mov qword [rdi + Value.ptr], 0

    add rdi, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rdi
    dispatch

.overflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_OVERFLOW
    jmp vm_error_return

; OP_PUSH_NULL (102)
op_push_null:
    mov rdi, [r12 + VMContext.data_sp]
    cmp rdi, [r12 + VMContext.data_limit]
    jge .overflow

    mov byte [rdi + Value.tag], VTAG_NULL
    mov qword [rdi + Value.num], 0
    mov qword [rdi + Value.ptr], 0

    add rdi, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rdi
    dispatch

.overflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_OVERFLOW
    jmp vm_error_return

; OP_PUSH_ZERO (103)
op_push_zero:
    mov rdi, [r12 + VMContext.data_sp]
    cmp rdi, [r12 + VMContext.data_limit]
    jge .overflow

    mov byte [rdi + Value.tag], VTAG_INTEGER
    mov qword [rdi + Value.num], 0
    mov qword [rdi + Value.ptr], 0

    add rdi, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rdi
    dispatch

.overflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_OVERFLOW
    jmp vm_error_return

; OP_PUSH_ONE (104)
op_push_one:
    mov rdi, [r12 + VMContext.data_sp]
    cmp rdi, [r12 + VMContext.data_limit]
    jge .overflow

    mov byte [rdi + Value.tag], VTAG_INTEGER
    mov qword [rdi + Value.num], 1
    mov qword [rdi + Value.ptr], 0

    add rdi, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rdi
    dispatch

.overflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_OVERFLOW
    jmp vm_error_return

; ============================================================================
; Extended Opcodes
; ============================================================================

; OP_OPERATOR (200) - Call operator by index
op_operator:
    read_varint                     ; operator index in rax

    ; Get operator handler from table
    mov rdi, [r12 + VMContext.op_table]
    test rdi, rdi
    jz .no_table

    cmp rax, [r12 + VMContext.op_count]
    jge .invalid

    ; Call operator (requires callback mechanism - stub for now)
    mov dword [r12 + VMContext.error_code], ERR_NOT_IMPLEMENTED
    jmp vm_error_return

.no_table:
.invalid:
    mov dword [r12 + VMContext.error_code], ERR_INVALID_OPCODE
    jmp vm_error_return

; OP_CONSTANT (201) - Push constant by index
op_constant:
    read_varint                     ; constant index in rax

    ; Bounds check
    cmp rax, [r12 + VMContext.const_count]
    jge .invalid

    ; Get constant address
    mov rdi, [r12 + VMContext.constants]
    imul rax, VALUE_SIZE
    add rdi, rax                    ; rdi = &constants[index]

    ; Push onto stack
    mov rax, [r12 + VMContext.data_sp]
    cmp rax, [r12 + VMContext.data_limit]
    jge .overflow

    ; Copy value
    mov rsi, rdi
    mov rdi, rax
    mov rcx, VALUE_SIZE / 8
    rep movsq

    add qword [r12 + VMContext.data_sp], VALUE_SIZE
    dispatch

.overflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_OVERFLOW
    jmp vm_error_return

.invalid:
    mov dword [r12 + VMContext.error_code], ERR_OUT_OF_BOUNDS
    jmp vm_error_return

; OP_NAME (202) - Push name by index
op_name:
    read_varint                     ; name index in rax

    ; Bounds check
    cmp rax, [r12 + VMContext.name_count]
    jge .invalid

    ; Get name pointer
    mov rdi, [r12 + VMContext.names]
    mov rdi, [rdi + rax * 8]        ; names[index] (array of pointers)

    ; Push as Value with name tag
    mov rax, [r12 + VMContext.data_sp]
    cmp rax, [r12 + VMContext.data_limit]
    jge .overflow

    mov byte [rax + Value.tag], VTAG_NAME
    mov qword [rax + Value.num], 0
    mov [rax + Value.ptr], rdi

    add rax, VALUE_SIZE
    mov [r12 + VMContext.data_sp], rax
    dispatch

.overflow:
    mov dword [r12 + VMContext.error_code], ERR_STACK_OVERFLOW
    jmp vm_error_return

.invalid:
    mov dword [r12 + VMContext.error_code], ERR_OUT_OF_BOUNDS
    jmp vm_error_return

; ============================================================================
; Halt
; ============================================================================

; OP_HALT (255) - Stop execution
op_halt:
    ; Normal termination
    ; NOTE: Don't save r14 - handlers update VMContext.data_sp directly
    xor eax, eax
    jmp vm_exit_success
