; DTRules NASM VM - Jump Table and Arithmetic/Comparison Handlers
; Copyright 2024 Paul Snow - Apache License 2.0
;
; This file implements:
; - The opcode dispatch jump table (256 entries)
; - Arithmetic operators: add, sub, mul, div, mod, neg, abs, inc, dec
; - Comparison operators: eq, ne, lt, le, gt, ge
; - Boolean operators: and, or, not, xor

%include "dtrules.inc"

; External handlers from vm_entry.asm
extern op_nop
extern op_push_true
extern op_push_false
extern op_push_null
extern op_push_zero
extern op_push_one
extern op_push_int
extern op_constant
extern op_name
extern op_pop
extern op_dup
extern op_swap
extern op_rot
extern op_invalid
extern vm_dispatch

; Core routines
extern vm_data_push
extern vm_data_pop
extern vm_data_peek

section .data

; ---------------------------------------------------------------------------
; jump_table - Opcode dispatch table (256 entries, 8 bytes each)
; ---------------------------------------------------------------------------
; Each entry is a pointer to the handler function for that opcode.
; Invalid/unimplemented opcodes point to op_invalid.
; ---------------------------------------------------------------------------
align 16
global jump_table
jump_table:
    ; 0-9: Stack operations
    dq op_nop           ; 0: NOP
    dq op_invalid       ; 1: PUSH (constant from pool - use OpConstant instead)
    dq op_push_int      ; 2: PUSH_INT
    dq op_pop           ; 3: POP
    dq op_dup           ; 4: DUP
    dq op_swap          ; 5: SWAP
    dq op_rot           ; 6: ROT
    dq op_invalid       ; 7: ROLL (not yet implemented)
    dq op_invalid       ; 8: INDEX (not yet implemented)
    dq op_invalid       ; 9: CLEAR (not yet implemented)
    dq op_invalid       ; 10: MARK

    ; 11-19: Reserved
    dq op_invalid, op_invalid, op_invalid, op_invalid, op_invalid
    dq op_invalid, op_invalid, op_invalid, op_invalid

    ; 20-28: Arithmetic
    dq op_add           ; 20: ADD
    dq op_sub           ; 21: SUB
    dq op_mul           ; 22: MUL
    dq op_div           ; 23: DIV
    dq op_mod           ; 24: MOD
    dq op_neg           ; 25: NEG
    dq op_abs           ; 26: ABS
    dq op_inc           ; 27: INC
    dq op_dec           ; 28: DEC

    ; 29: Reserved
    dq op_invalid

    ; 30-35: Comparison
    dq op_eq            ; 30: EQ
    dq op_ne            ; 31: NE
    dq op_lt            ; 32: LT
    dq op_le            ; 33: LE
    dq op_gt            ; 34: GT
    dq op_ge            ; 35: GE

    ; 36-39: Reserved
    dq op_invalid, op_invalid, op_invalid, op_invalid

    ; 40-43: Boolean
    dq op_and           ; 40: AND
    dq op_or            ; 41: OR
    dq op_not           ; 42: NOT
    dq op_xor           ; 43: XOR

    ; 44-49: Reserved
    dq op_invalid, op_invalid, op_invalid, op_invalid, op_invalid, op_invalid

    ; 50-59: Control flow
    dq op_invalid       ; 50: EXEC
    dq op_invalid       ; 51: IF
    dq op_invalid       ; 52: IFELSE
    dq op_invalid       ; 53: WHILE
    dq op_invalid       ; 54: FOR
    dq op_invalid       ; 55: FORALL
    dq op_invalid       ; 56: RETURN
    dq op_invalid       ; 57: JUMP
    dq op_invalid       ; 58: JUMP_IF
    dq op_invalid       ; 59: CALL

    ; 60-64: Entity operations
    dq op_invalid       ; 60: ENTITY_PUSH
    dq op_invalid       ; 61: ENTITY_POP
    dq op_def           ; 62: DEF
    dq op_lookup        ; 63: LOOKUP
    dq op_invalid       ; 64: NEW_ENTITY

    ; 65-69: Reserved
    dq op_invalid, op_invalid, op_invalid, op_invalid, op_invalid

    ; 70-74: Array operations
    dq op_invalid       ; 70: NEW_ARRAY
    dq op_invalid       ; 71: ADD_TO
    dq op_invalid       ; 72: LENGTH
    dq op_invalid       ; 73: GET
    dq op_invalid       ; 74: PUT

    ; 75-79: Reserved
    dq op_invalid, op_invalid, op_invalid, op_invalid, op_invalid

    ; 80-82: Table operations
    dq op_invalid       ; 80: NEW_TABLE
    dq op_invalid       ; 81: TABLE_GET
    dq op_invalid       ; 82: TABLE_PUT

    ; 83-89: Reserved
    dq op_invalid, op_invalid, op_invalid, op_invalid
    dq op_invalid, op_invalid, op_invalid

    ; 90-91: String operations
    dq op_invalid       ; 90: CONCAT
    dq op_invalid       ; 91: SUBSTRING

    ; 92-99: Reserved
    dq op_invalid, op_invalid, op_invalid, op_invalid
    dq op_invalid, op_invalid, op_invalid, op_invalid

    ; 100-104: Constants
    dq op_push_true     ; 100: PUSH_TRUE
    dq op_push_false    ; 101: PUSH_FALSE
    dq op_push_null     ; 102: PUSH_NULL
    dq op_push_zero     ; 103: PUSH_ZERO
    dq op_push_one      ; 104: PUSH_ONE

    ; 105-199: Reserved (fill with invalid)
    times 95 dq op_invalid

    ; 200-202: Extended opcodes
    dq op_operator      ; 200: OPERATOR
    dq op_constant      ; 201: CONSTANT
    dq op_name          ; 202: NAME

    ; 203-255: Reserved
    times 53 dq op_invalid

section .text

; ===========================================================================
; Arithmetic Operators
; ===========================================================================

; ---------------------------------------------------------------------------
; op_add - Add top two values: ( a b -- a+b )
; ---------------------------------------------------------------------------
global op_add
op_add:
    ; Pop b
    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error
    mov r8, rsi             ; b tag
    mov r9, rdx             ; b num

    ; Pop a
    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error
    ; rsi = a tag, rdx = a num

    ; Type check - both must be integers (for now)
    cmp rsi, VTAG_INTEGER
    jne .type_error
    cmp r8, VTAG_INTEGER
    jne .type_error

    ; Add
    add rdx, r9

    ; Push result
    mov rdi, r12
    mov rsi, VTAG_INTEGER
    call vm_data_push
    jmp vm_dispatch

.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 2
    jmp vm_dispatch
.type_error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 5  ; type error
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_sub - Subtract: ( a b -- a-b )
; ---------------------------------------------------------------------------
global op_sub
op_sub:
    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error
    mov r8, rsi
    mov r9, rdx

    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error

    cmp rsi, VTAG_INTEGER
    jne .type_error
    cmp r8, VTAG_INTEGER
    jne .type_error

    sub rdx, r9

    mov rdi, r12
    mov rsi, VTAG_INTEGER
    call vm_data_push
    jmp vm_dispatch

.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 2
    jmp vm_dispatch
.type_error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 5
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_mul - Multiply: ( a b -- a*b )
; ---------------------------------------------------------------------------
global op_mul
op_mul:
    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error
    mov r8, rsi
    mov r9, rdx

    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error

    cmp rsi, VTAG_INTEGER
    jne .type_error
    cmp r8, VTAG_INTEGER
    jne .type_error

    imul rdx, r9

    mov rdi, r12
    mov rsi, VTAG_INTEGER
    call vm_data_push
    jmp vm_dispatch

.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 2
    jmp vm_dispatch
.type_error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 5
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_div - Divide: ( a b -- a/b )
; ---------------------------------------------------------------------------
global op_div
op_div:
    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error
    mov r8, rsi
    mov r9, rdx             ; b (divisor)

    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error
    ; rdx = a (dividend)

    cmp rsi, VTAG_INTEGER
    jne .type_error
    cmp r8, VTAG_INTEGER
    jne .type_error

    ; Check division by zero
    test r9, r9
    jz .div_zero

    ; Perform signed division
    mov rax, rdx
    cqo                     ; Sign-extend rax into rdx:rax
    idiv r9                 ; rax = quotient, rdx = remainder
    mov rdx, rax

    mov rdi, r12
    mov rsi, VTAG_INTEGER
    call vm_data_push
    jmp vm_dispatch

.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 2
    jmp vm_dispatch
.type_error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 5
    jmp vm_dispatch
.div_zero:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 7  ; division by zero
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_mod - Modulo: ( a b -- a%b )
; ---------------------------------------------------------------------------
global op_mod
op_mod:
    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error
    mov r8, rsi
    mov r9, rdx

    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error

    cmp rsi, VTAG_INTEGER
    jne .type_error
    cmp r8, VTAG_INTEGER
    jne .type_error

    test r9, r9
    jz .div_zero

    mov rax, rdx
    cqo
    idiv r9
    ; rdx now contains remainder

    mov rdi, r12
    mov rsi, VTAG_INTEGER
    call vm_data_push
    jmp vm_dispatch

.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 2
    jmp vm_dispatch
.type_error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 5
    jmp vm_dispatch
.div_zero:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 7
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_neg - Negate: ( a -- -a )
; ---------------------------------------------------------------------------
global op_neg
op_neg:
    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error

    cmp rsi, VTAG_INTEGER
    jne .type_error

    neg rdx

    mov rdi, r12
    mov rsi, VTAG_INTEGER
    call vm_data_push
    jmp vm_dispatch

.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 2
    jmp vm_dispatch
.type_error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 5
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_abs - Absolute value: ( a -- |a| )
; ---------------------------------------------------------------------------
global op_abs
op_abs:
    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error

    cmp rsi, VTAG_INTEGER
    jne .type_error

    ; Compute absolute value without branching
    mov rax, rdx
    sar rax, 63             ; Fill with sign bit
    xor rdx, rax            ; Flip bits if negative
    sub rdx, rax            ; Add 1 if was negative

    mov rdi, r12
    mov rsi, VTAG_INTEGER
    call vm_data_push
    jmp vm_dispatch

.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 2
    jmp vm_dispatch
.type_error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 5
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_inc - Increment: ( a -- a+1 )
; ---------------------------------------------------------------------------
global op_inc
op_inc:
    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error

    cmp rsi, VTAG_INTEGER
    jne .type_error

    inc rdx

    mov rdi, r12
    mov rsi, VTAG_INTEGER
    call vm_data_push
    jmp vm_dispatch

.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 2
    jmp vm_dispatch
.type_error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 5
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_dec - Decrement: ( a -- a-1 )
; ---------------------------------------------------------------------------
global op_dec
op_dec:
    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error

    cmp rsi, VTAG_INTEGER
    jne .type_error

    dec rdx

    mov rdi, r12
    mov rsi, VTAG_INTEGER
    call vm_data_push
    jmp vm_dispatch

.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 2
    jmp vm_dispatch
.type_error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 5
    jmp vm_dispatch


; ===========================================================================
; Comparison Operators
; ===========================================================================

; ---------------------------------------------------------------------------
; op_eq - Equal: ( a b -- a==b )
; ---------------------------------------------------------------------------
global op_eq
op_eq:
    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error
    mov r8, rsi             ; b tag
    mov r9, rdx             ; b num

    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error

    ; Compare types first
    xor edx, edx            ; result = false
    cmp rsi, r8
    jne .push_result

    ; Same type, compare values
    cmp rcx, r9             ; Note: rcx was rdx before reassignment
    ; Actually we need the original rdx from second pop
    ; Let me fix this

    ; Redo comparison properly
    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error
    mov r8, rsi
    mov r9, rdx

    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error
    mov r10, rsi            ; a tag
    mov r11, rdx            ; a num

    ; Compare
    xor edx, edx
    cmp r10, r8             ; tags equal?
    jne .push_result
    cmp r11, r9             ; values equal?
    jne .push_result
    mov edx, 1              ; true

.push_result:
    mov rdi, r12
    mov rsi, VTAG_BOOLEAN
    call vm_data_push
    jmp vm_dispatch

.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 2
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_ne - Not equal: ( a b -- a!=b )
; ---------------------------------------------------------------------------
global op_ne
op_ne:
    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error
    mov r8, rsi
    mov r9, rdx

    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error
    mov r10, rsi
    mov r11, rdx

    ; Compare - result is opposite of eq
    mov edx, 1              ; assume not equal
    cmp r10, r8
    jne .push_result
    cmp r11, r9
    jne .push_result
    xor edx, edx            ; equal, so ne = false

.push_result:
    mov rdi, r12
    mov rsi, VTAG_BOOLEAN
    call vm_data_push
    jmp vm_dispatch

.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 2
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_lt - Less than: ( a b -- a<b )
; ---------------------------------------------------------------------------
global op_lt
op_lt:
    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error
    mov r8, rsi
    mov r9, rdx             ; b

    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error
    ; rsi = a tag, rdx = a

    cmp rsi, VTAG_INTEGER
    jne .type_error
    cmp r8, VTAG_INTEGER
    jne .type_error

    xor eax, eax
    cmp rdx, r9             ; a < b ?
    setl al
    mov rdx, rax

    mov rdi, r12
    mov rsi, VTAG_BOOLEAN
    call vm_data_push
    jmp vm_dispatch

.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 2
    jmp vm_dispatch
.type_error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 5
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_le - Less than or equal: ( a b -- a<=b )
; ---------------------------------------------------------------------------
global op_le
op_le:
    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error
    mov r8, rsi
    mov r9, rdx

    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error

    cmp rsi, VTAG_INTEGER
    jne .type_error
    cmp r8, VTAG_INTEGER
    jne .type_error

    xor eax, eax
    cmp rdx, r9
    setle al
    mov rdx, rax

    mov rdi, r12
    mov rsi, VTAG_BOOLEAN
    call vm_data_push
    jmp vm_dispatch

.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 2
    jmp vm_dispatch
.type_error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 5
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_gt - Greater than: ( a b -- a>b )
; ---------------------------------------------------------------------------
global op_gt
op_gt:
    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error
    mov r8, rsi
    mov r9, rdx

    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error

    cmp rsi, VTAG_INTEGER
    jne .type_error
    cmp r8, VTAG_INTEGER
    jne .type_error

    xor eax, eax
    cmp rdx, r9
    setg al
    mov rdx, rax

    mov rdi, r12
    mov rsi, VTAG_BOOLEAN
    call vm_data_push
    jmp vm_dispatch

.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 2
    jmp vm_dispatch
.type_error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 5
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_ge - Greater than or equal: ( a b -- a>=b )
; ---------------------------------------------------------------------------
global op_ge
op_ge:
    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error
    mov r8, rsi
    mov r9, rdx

    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error

    cmp rsi, VTAG_INTEGER
    jne .type_error
    cmp r8, VTAG_INTEGER
    jne .type_error

    xor eax, eax
    cmp rdx, r9
    setge al
    mov rdx, rax

    mov rdi, r12
    mov rsi, VTAG_BOOLEAN
    call vm_data_push
    jmp vm_dispatch

.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 2
    jmp vm_dispatch
.type_error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 5
    jmp vm_dispatch


; ===========================================================================
; Boolean Operators
; ===========================================================================

; ---------------------------------------------------------------------------
; op_and - Logical AND: ( a b -- a&&b )
; ---------------------------------------------------------------------------
global op_and
op_and:
    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error
    mov r8, rsi
    mov r9, rdx             ; b

    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error
    ; rsi = a tag, rdx = a

    ; Both must be boolean (or we coerce to boolean)
    ; For now, treat nonzero as true
    test rdx, rdx
    jz .false
    test r9, r9
    jz .false
    mov edx, 1
    jmp .push
.false:
    xor edx, edx
.push:
    mov rdi, r12
    mov rsi, VTAG_BOOLEAN
    call vm_data_push
    jmp vm_dispatch

.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 2
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_or - Logical OR: ( a b -- a||b )
; ---------------------------------------------------------------------------
global op_or
op_or:
    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error
    mov r8, rsi
    mov r9, rdx

    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error

    xor eax, eax
    test rdx, rdx
    jnz .true
    test r9, r9
    jnz .true
    jmp .push
.true:
    mov eax, 1
.push:
    mov rdx, rax
    mov rdi, r12
    mov rsi, VTAG_BOOLEAN
    call vm_data_push
    jmp vm_dispatch

.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 2
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_not - Logical NOT: ( a -- !a )
; ---------------------------------------------------------------------------
global op_not
op_not:
    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error

    ; Invert truthiness
    test rdx, rdx
    setz al
    movzx edx, al

    mov rdi, r12
    mov rsi, VTAG_BOOLEAN
    call vm_data_push
    jmp vm_dispatch

.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 2
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_xor - Logical XOR: ( a b -- a^b )
; ---------------------------------------------------------------------------
global op_xor
op_xor:
    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error
    mov r8, rsi
    mov r9, rdx

    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error

    ; XOR of truthiness: (a != 0) ^ (b != 0)
    xor eax, eax
    test rdx, rdx
    setnz al
    xor ecx, ecx
    test r9, r9
    setnz cl
    xor eax, ecx
    mov rdx, rax

    mov rdi, r12
    mov rsi, VTAG_BOOLEAN
    call vm_data_push
    jmp vm_dispatch

.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 2
    jmp vm_dispatch


; ===========================================================================
; Entity Operations (stubs that call Go)
; ===========================================================================

; ---------------------------------------------------------------------------
; op_lookup - Look up name in entity context
; ---------------------------------------------------------------------------
extern vm_callback_lookup
global op_lookup
op_lookup:
    ; Pop name value
    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error

    ; Verify it's a name
    cmp rsi, VTAG_NAME
    jne .type_error

    ; Call Go callback: vm_callback_lookup(context, name_ptr)
    ; Returns value in Go calling convention
    mov rdi, r12
    mov rsi, rdx            ; name pointer
    call vm_callback_lookup

    ; Result is pushed by Go callback
    jmp vm_dispatch

.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 2
    jmp vm_dispatch
.type_error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 5
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_def - Define variable in entity context
; ---------------------------------------------------------------------------
extern vm_callback_def
global op_def
op_def:
    ; Pop name
    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error
    cmp rsi, VTAG_NAME
    jne .type_error
    mov r8, rdx             ; name pointer

    ; Pop value
    mov rdi, r12
    call vm_data_pop
    test eax, eax
    jnz .error

    ; Call Go callback: vm_callback_def(context, name_ptr, tag, num)
    mov rdi, r12
    ; rsi = tag (already set)
    ; rdx = num (already set)
    mov rcx, r8             ; name
    call vm_callback_def

    jmp vm_dispatch

.error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 2
    jmp vm_dispatch
.type_error:
    or qword [r12 + VMContext.flags], VM_FLAG_ERROR
    mov qword [r12 + VMContext.error_code], 5
    jmp vm_dispatch


; ---------------------------------------------------------------------------
; op_operator - Call operator by index (varint follows)
; ---------------------------------------------------------------------------
extern vm_callback_operator
global op_operator
op_operator:
    ; Read operator index
    mov rdi, r12
    call vm_read_varint
    test qword [r12 + VMContext.flags], VM_FLAG_ERROR
    jnz vm_dispatch
    mov r14, [r12 + VMContext.pc]

    ; Call Go callback: vm_callback_operator(context, op_index)
    mov rdi, r12
    mov rsi, rax            ; operator index
    call vm_callback_operator

    jmp vm_dispatch

; Need to declare vm_read_varint as external
extern vm_read_varint
