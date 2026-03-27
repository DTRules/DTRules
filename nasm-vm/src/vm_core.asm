; vm_core.asm - DTRules NASM VM Core
; Jump table dispatch with single entry/exit pattern
;
; Architecture:
; - Single entry point (vm_execute)
; - Jump table dispatch for opcodes
; - Each handler ends with jmp dispatch
; - No CGO-style per-operation calls

bits 64
default rel

%include "include/constants.inc"
%include "include/state.inc"
%include "include/macros.inc"

;-----------------------------------------------------------------------------
; External symbols
;-----------------------------------------------------------------------------
extern state
extern data_stack

;-----------------------------------------------------------------------------
; Exported symbols
;-----------------------------------------------------------------------------
global vm_execute
global vm_init
global dispatch

; Stack operations
global stack_data_push
global stack_data_push_integer
global stack_data_push_boolean
global stack_data_push_double
global stack_data_pop
global stack_data_peek

; Comparison operators (exported for testing)
global op_eq
global op_ne
global op_lt
global op_le
global op_gt
global op_ge

;-----------------------------------------------------------------------------
section .data
;-----------------------------------------------------------------------------

; Jump table for opcode dispatch (256 entries, 8 bytes each)
align 8
jump_table:
    dq op_nop           ; 0: NOP
    dq op_push          ; 1: PUSH constant
    dq op_push_int      ; 2: PUSH_INT immediate
    dq op_pop           ; 3: POP
    dq op_dup           ; 4: DUP
    dq op_swap          ; 5: SWAP
    times 14 dq op_invalid  ; 6-19: reserved

    ; Arithmetic (20-29)
    dq op_add           ; 20: ADD
    dq op_sub           ; 21: SUB
    dq op_mul           ; 22: MUL
    dq op_div           ; 23: DIV
    times 6 dq op_invalid   ; 24-29: reserved

    ; Comparison (30-35)
    dq op_eq            ; 30: EQ
    dq op_ne            ; 31: NE
    dq op_lt            ; 32: LT
    dq op_le            ; 33: LE
    dq op_gt            ; 34: GT
    dq op_ge            ; 35: GE
    times 4 dq op_invalid   ; 36-39: reserved

    ; Boolean (40-42)
    dq op_and           ; 40: AND
    dq op_or            ; 41: OR
    dq op_not           ; 42: NOT
    times 212 dq op_invalid ; 43-254: reserved

    dq op_halt          ; 255: HALT

;-----------------------------------------------------------------------------
section .bss
;-----------------------------------------------------------------------------

;-----------------------------------------------------------------------------
section .text
;-----------------------------------------------------------------------------

;-----------------------------------------------------------------------------
; vm_init - Initialize VM state
; No arguments
; Clobbers: rax, rcx
;-----------------------------------------------------------------------------
vm_init:
    ; Initialize data stack pointer to base
    lea rax, [data_stack]
    mov [state + State.data_stack], rax
    mov [state + State.data_stack_base], rax

    ; Set stack end (base + size)
    add rax, STACK_SIZE_BYTES
    mov [state + State.data_stack_end], rax

    ; Clear error and halt flags
    mov dword [state + State.error], ERR_NONE
    mov dword [state + State.halted], 0

    ret

;-----------------------------------------------------------------------------
; vm_execute - Main VM execution loop
; Input: rdi = pointer to bytecode
;        rsi = bytecode length
; Output: rax = 0 on success, error code on failure
;
; Uses jump table dispatch pattern:
;   1. Fetch opcode byte
;   2. Use opcode as index into jump table
;   3. Jump to handler
;   4. Handler ends with jmp dispatch
;-----------------------------------------------------------------------------
vm_execute:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    ; Set up bytecode pointers
    mov [state + State.bytecode], rdi
    add rsi, rdi
    mov [state + State.bytecode_end], rsi

    ; r12 = jump table base
    lea r12, [jump_table]

    ; Clear halt flag
    mov dword [state + State.halted], 0

dispatch:
    ; Check if halted
    mov eax, [state + State.halted]
    test eax, eax
    jnz .done

    ; Check if we've reached end of bytecode
    mov rax, [state + State.bytecode]
    cmp rax, [state + State.bytecode_end]
    jae .done

    ; Fetch opcode and advance PC
    movzx eax, byte [rax]
    inc qword [state + State.bytecode]

    ; Jump via table: jmp [table + opcode*8]
    jmp [r12 + rax*8]

.done:
    ; Return error code (0 = success)
    mov eax, [state + State.error]

    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; Stack Operations
;-----------------------------------------------------------------------------

; stack_data_push - Push a value onto the data stack
; Input: rdi = pointer to 24-byte value to push
; Output: rax = 0 on success, -1 on overflow
stack_data_push:
    mov rax, [state + State.data_stack]
    mov rcx, [state + State.data_stack_end]

    ; Check for overflow
    lea rdx, [rax + VALUE_SIZE]
    cmp rdx, rcx
    ja .overflow

    ; Copy 24 bytes
    mov rcx, [rdi]
    mov [rax], rcx
    mov rcx, [rdi + 8]
    mov [rax + 8], rcx
    mov rcx, [rdi + 16]
    mov [rax + 16], rcx

    ; Update stack pointer
    mov [state + State.data_stack], rdx

    xor eax, eax
    ret

.overflow:
    mov dword [state + State.error], ERR_STACK_OVERFLOW
    mov eax, -1
    ret

; stack_data_push_integer - Push an integer value
; Input: rdi = integer value
; Output: rax = pointer to pushed value, 0 on overflow
stack_data_push_integer:
    mov rax, [state + State.data_stack]
    mov rcx, [state + State.data_stack_end]

    ; Check for overflow
    lea rdx, [rax + VALUE_SIZE]
    cmp rdx, rcx
    ja .overflow

    ; Write value: tag=INTEGER, num=value
    mov byte [rax + VALUE_TAG_OFF], VTAG_INTEGER
    mov qword [rax + VALUE_NUM_OFF], rdi
    mov qword [rax + VALUE_PTR_OFF], 0

    ; Update stack pointer
    mov [state + State.data_stack], rdx
    ret

.overflow:
    mov dword [state + State.error], ERR_STACK_OVERFLOW
    xor eax, eax
    ret

; stack_data_push_boolean - Push a boolean value
; Input: rdi = 0 for false, non-zero for true
; Output: rax = pointer to pushed value, 0 on overflow
stack_data_push_boolean:
    mov rax, [state + State.data_stack]
    mov rcx, [state + State.data_stack_end]

    ; Check for overflow
    lea rdx, [rax + VALUE_SIZE]
    cmp rdx, rcx
    ja .overflow

    ; Normalize boolean: anything non-zero becomes 1
    test rdi, rdi
    setnz dil
    movzx edi, dil

    ; Write value: tag=BOOLEAN, num=0 or 1
    mov byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    mov qword [rax + VALUE_NUM_OFF], rdi
    mov qword [rax + VALUE_PTR_OFF], 0

    ; Update stack pointer
    mov [state + State.data_stack], rdx
    ret

.overflow:
    mov dword [state + State.error], ERR_STACK_OVERFLOW
    xor eax, eax
    ret

; stack_data_push_double - Push a double value
; Input: xmm0 = double value
; Output: rax = pointer to pushed value, 0 on overflow
stack_data_push_double:
    mov rax, [state + State.data_stack]
    mov rcx, [state + State.data_stack_end]

    ; Check for overflow
    lea rdx, [rax + VALUE_SIZE]
    cmp rdx, rcx
    ja .overflow

    ; Write value: tag=DOUBLE, num=double bits
    mov byte [rax + VALUE_TAG_OFF], VTAG_DOUBLE
    movsd [rax + VALUE_NUM_OFF], xmm0
    mov qword [rax + VALUE_PTR_OFF], 0

    ; Update stack pointer
    mov [state + State.data_stack], rdx
    ret

.overflow:
    mov dword [state + State.error], ERR_STACK_OVERFLOW
    xor eax, eax
    ret

; stack_data_pop - Pop a value from the data stack
; Output: rax = pointer to value (now below stack top), 0 on underflow
stack_data_pop:
    mov rax, [state + State.data_stack]
    mov rcx, [state + State.data_stack_base]

    ; Check for underflow
    cmp rax, rcx
    jbe .underflow

    ; Move stack pointer down and return pointer to value
    sub rax, VALUE_SIZE
    mov [state + State.data_stack], rax
    ret

.underflow:
    mov dword [state + State.error], ERR_STACK_UNDERFLOW
    xor eax, eax
    ret

; stack_data_peek - Get pointer to top of stack without popping
; Output: rax = pointer to top value, 0 if stack empty
stack_data_peek:
    mov rax, [state + State.data_stack]
    mov rcx, [state + State.data_stack_base]

    cmp rax, rcx
    jbe .empty

    sub rax, VALUE_SIZE
    ret

.empty:
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; Opcode Handlers
; Each handler ends with: jmp dispatch
;-----------------------------------------------------------------------------

op_nop:
    jmp dispatch

op_halt:
    mov dword [state + State.halted], 1
    jmp dispatch

op_invalid:
    mov dword [state + State.error], ERR_INVALID_OPCODE
    mov dword [state + State.halted], 1
    jmp dispatch

op_push:
    ; TODO: Push from constant pool
    jmp dispatch

op_push_int:
    ; Read varint from bytecode (simplified: just read byte for now)
    mov rax, [state + State.bytecode]
    movsx rdi, byte [rax]
    inc qword [state + State.bytecode]
    call stack_data_push_integer
    jmp dispatch

op_pop:
    call stack_data_pop
    jmp dispatch

op_dup:
    call stack_data_peek
    test rax, rax
    jz .error
    mov rdi, rax
    call stack_data_push
    jmp dispatch
.error:
    jmp dispatch

op_swap:
    ; Pop two values, push in reverse order
    call stack_data_pop
    test rax, rax
    jz .error
    mov r13, rax            ; Save second value ptr

    call stack_data_pop
    test rax, rax
    jz .error

    ; Push in reverse order: second first, then first (currently in rax area)
    ; Copy first value to temp
    push qword [rax + 16]
    push qword [rax + 8]
    push qword [rax]

    ; Push second value
    mov rdi, r13
    call stack_data_push

    ; Push first value from temp
    mov rax, [state + State.data_stack]
    pop qword [rax]
    pop qword [rax + 8]
    pop qword [rax + 16]
    add qword [state + State.data_stack], VALUE_SIZE

.error:
    jmp dispatch

;-----------------------------------------------------------------------------
; Arithmetic Operations
; Uses r13 (callee-saved) to preserve pointer across pop calls
;-----------------------------------------------------------------------------

op_add:
    call stack_data_pop
    test rax, rax
    jz .error
    mov r13, rax            ; Save in callee-saved reg

    call stack_data_pop
    test rax, rax
    jz .error

    ; Both must be integers (simplified)
    cmp byte [r13 + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error

    mov rdi, [rax + VALUE_NUM_OFF]
    add rdi, [r13 + VALUE_NUM_OFF]
    call stack_data_push_integer
    jmp dispatch

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    jmp dispatch

op_sub:
    call stack_data_pop
    test rax, rax
    jz .error
    mov r13, rax            ; Save in callee-saved reg

    call stack_data_pop
    test rax, rax
    jz .error

    cmp byte [r13 + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error

    mov rdi, [rax + VALUE_NUM_OFF]
    sub rdi, [r13 + VALUE_NUM_OFF]
    call stack_data_push_integer
    jmp dispatch

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    jmp dispatch

op_mul:
    call stack_data_pop
    test rax, rax
    jz .error
    mov r13, rax            ; Save in callee-saved reg

    call stack_data_pop
    test rax, rax
    jz .error

    cmp byte [r13 + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error

    mov rdi, [rax + VALUE_NUM_OFF]
    imul rdi, [r13 + VALUE_NUM_OFF]
    call stack_data_push_integer
    jmp dispatch

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    jmp dispatch

op_div:
    call stack_data_pop
    test rax, rax
    jz .error
    mov r13, rax            ; Save in callee-saved reg

    call stack_data_pop
    test rax, rax
    jz .error

    cmp byte [r13 + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error

    mov r8, [r13 + VALUE_NUM_OFF]
    test r8, r8
    jz .div_zero

    mov rdi, [rax + VALUE_NUM_OFF]
    cqo
    idiv r8
    mov rdi, rax
    call stack_data_push_integer
    jmp dispatch

.div_zero:
    mov dword [state + State.error], ERR_DIV_BY_ZERO
    jmp dispatch

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    jmp dispatch

;-----------------------------------------------------------------------------
; Comparison Operations
; Stack: ( a b -- result )
; Compares a with b, pushes boolean result
;-----------------------------------------------------------------------------

;-----------------------------------------------------------------------------
; op_eq - Equality comparison: ( a b -- a==b )
; Supports integers, doubles, booleans, null
; Uses r13 (callee-saved) to preserve pointer across pop calls
;-----------------------------------------------------------------------------
op_eq:
    call stack_data_pop
    test rax, rax
    jz .error
    mov r13, rax            ; b (right operand) - saved in callee-saved reg

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax            ; a (left operand)

    ; Get types
    movzx esi, byte [r13 + VALUE_TAG_OFF]   ; b type
    movzx edi, byte [rdx + VALUE_TAG_OFF]   ; a type

    ; Types must match for equality
    cmp esi, edi
    jne .not_equal

    ; Same type - compare based on type
    cmp esi, VTAG_NULL
    je .equal               ; Both null are equal

    cmp esi, VTAG_INTEGER
    je .cmp_int
    cmp esi, VTAG_BOOLEAN
    je .cmp_int             ; Booleans compared like integers
    cmp esi, VTAG_DOUBLE
    je .cmp_double

    ; For complex types (strings, arrays, entities), compare pointers
    mov rax, [r13 + VALUE_PTR_OFF]
    cmp rax, [rdx + VALUE_PTR_OFF]
    je .equal
    jmp .not_equal

.cmp_int:
    mov rax, [r13 + VALUE_NUM_OFF]
    cmp rax, [rdx + VALUE_NUM_OFF]
    je .equal
    jmp .not_equal

.cmp_double:
    movsd xmm0, [r13 + VALUE_NUM_OFF]
    ucomisd xmm0, [rdx + VALUE_NUM_OFF]
    jp .not_equal           ; NaN is not equal to anything
    je .equal
    jmp .not_equal

.equal:
    mov rdi, 1
    call stack_data_push_boolean
    jmp dispatch

.not_equal:
    xor edi, edi
    call stack_data_push_boolean
    jmp dispatch

.error:
    jmp dispatch

;-----------------------------------------------------------------------------
; op_ne - Not equal comparison: ( a b -- a!=b )
; Implemented as NOT(EQ)
; Uses r13 (callee-saved) to preserve pointer across pop calls
;-----------------------------------------------------------------------------
op_ne:
    call stack_data_pop
    test rax, rax
    jz .error
    mov r13, rax            ; b - saved in callee-saved reg

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax            ; a

    movzx esi, byte [r13 + VALUE_TAG_OFF]
    movzx edi, byte [rdx + VALUE_TAG_OFF]

    cmp esi, edi
    jne .not_equal          ; Different types -> not equal -> NE is true

    cmp esi, VTAG_NULL
    je .equal               ; Both null -> equal -> NE is false

    cmp esi, VTAG_INTEGER
    je .cmp_int
    cmp esi, VTAG_BOOLEAN
    je .cmp_int
    cmp esi, VTAG_DOUBLE
    je .cmp_double

    mov rax, [r13 + VALUE_PTR_OFF]
    cmp rax, [rdx + VALUE_PTR_OFF]
    je .equal
    jmp .not_equal

.cmp_int:
    mov rax, [r13 + VALUE_NUM_OFF]
    cmp rax, [rdx + VALUE_NUM_OFF]
    je .equal
    jmp .not_equal

.cmp_double:
    movsd xmm0, [r13 + VALUE_NUM_OFF]
    ucomisd xmm0, [rdx + VALUE_NUM_OFF]
    jp .not_equal           ; NaN -> not equal -> NE is true
    je .equal
    jmp .not_equal

.equal:
    ; a == b, so a != b is FALSE
    xor edi, edi
    call stack_data_push_boolean
    jmp dispatch

.not_equal:
    ; a != b is TRUE
    mov rdi, 1
    call stack_data_push_boolean
    jmp dispatch

.error:
    jmp dispatch

;-----------------------------------------------------------------------------
; op_lt - Less than comparison: ( a b -- a<b )
; Supports integers and doubles (with type promotion)
; Uses r13 (callee-saved) to preserve pointer across pop calls
;-----------------------------------------------------------------------------
op_lt:
    call stack_data_pop
    test rax, rax
    jz .error
    mov r13, rax            ; b (right operand) - saved in callee-saved reg

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax            ; a (left operand)

    movzx esi, byte [r13 + VALUE_TAG_OFF]   ; b type
    movzx edi, byte [rdx + VALUE_TAG_OFF]   ; a type

    ; Check if both are integers
    cmp esi, VTAG_INTEGER
    jne .check_double_b
    cmp edi, VTAG_INTEGER
    jne .a_not_int

    ; Both integers: a < b
    mov rax, [rdx + VALUE_NUM_OFF]  ; a
    cmp rax, [r13 + VALUE_NUM_OFF]  ; compare with b
    setl dil                         ; set if a < b (signed)
    movzx edi, dil
    call stack_data_push_boolean
    jmp dispatch

.a_not_int:
    ; b is integer, a might be double
    cmp edi, VTAG_DOUBLE
    jne .type_error
    ; a is double, b is integer - convert b to double
    cvtsi2sd xmm1, qword [r13 + VALUE_NUM_OFF]  ; b as double
    movsd xmm0, [rdx + VALUE_NUM_OFF]           ; a as double
    ucomisd xmm0, xmm1
    setb dil                         ; set if a < b (unsigned for doubles)
    movzx edi, dil
    call stack_data_push_boolean
    jmp dispatch

.check_double_b:
    ; b is not integer - check if double
    cmp esi, VTAG_DOUBLE
    jne .type_error
    cmp edi, VTAG_DOUBLE
    jne .a_is_int_b_double

    ; Both doubles
    movsd xmm0, [rdx + VALUE_NUM_OFF]   ; a
    ucomisd xmm0, [r13 + VALUE_NUM_OFF] ; compare with b
    setb dil
    movzx edi, dil
    call stack_data_push_boolean
    jmp dispatch

.a_is_int_b_double:
    ; a might be integer, b is double
    cmp edi, VTAG_INTEGER
    jne .type_error
    ; a is integer, b is double - convert a to double
    cvtsi2sd xmm0, qword [rdx + VALUE_NUM_OFF]  ; a as double
    ucomisd xmm0, [r13 + VALUE_NUM_OFF]         ; compare with b
    setb dil
    movzx edi, dil
    call stack_data_push_boolean
    jmp dispatch

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    jmp dispatch

;-----------------------------------------------------------------------------
; op_le - Less than or equal comparison: ( a b -- a<=b )
; Uses r13 (callee-saved) to preserve pointer across pop calls
;-----------------------------------------------------------------------------
op_le:
    call stack_data_pop
    test rax, rax
    jz .error
    mov r13, rax            ; b - saved in callee-saved reg

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax            ; a

    movzx esi, byte [r13 + VALUE_TAG_OFF]
    movzx edi, byte [rdx + VALUE_TAG_OFF]

    cmp esi, VTAG_INTEGER
    jne .check_double_b
    cmp edi, VTAG_INTEGER
    jne .a_not_int

    ; Both integers
    mov rax, [rdx + VALUE_NUM_OFF]
    cmp rax, [r13 + VALUE_NUM_OFF]
    setle dil
    movzx edi, dil
    call stack_data_push_boolean
    jmp dispatch

.a_not_int:
    cmp edi, VTAG_DOUBLE
    jne .type_error
    cvtsi2sd xmm1, qword [r13 + VALUE_NUM_OFF]
    movsd xmm0, [rdx + VALUE_NUM_OFF]
    ucomisd xmm0, xmm1
    setbe dil
    movzx edi, dil
    call stack_data_push_boolean
    jmp dispatch

.check_double_b:
    cmp esi, VTAG_DOUBLE
    jne .type_error
    cmp edi, VTAG_DOUBLE
    jne .a_is_int_b_double

    ; Both doubles
    movsd xmm0, [rdx + VALUE_NUM_OFF]
    ucomisd xmm0, [r13 + VALUE_NUM_OFF]
    setbe dil
    movzx edi, dil
    call stack_data_push_boolean
    jmp dispatch

.a_is_int_b_double:
    cmp edi, VTAG_INTEGER
    jne .type_error
    cvtsi2sd xmm0, qword [rdx + VALUE_NUM_OFF]
    ucomisd xmm0, [r13 + VALUE_NUM_OFF]
    setbe dil
    movzx edi, dil
    call stack_data_push_boolean
    jmp dispatch

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    jmp dispatch

;-----------------------------------------------------------------------------
; op_gt - Greater than comparison: ( a b -- a>b )
; Uses r13 (callee-saved) to preserve pointer across pop calls
;-----------------------------------------------------------------------------
op_gt:
    call stack_data_pop
    test rax, rax
    jz .error
    mov r13, rax            ; b - saved in callee-saved reg

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax            ; a

    movzx esi, byte [r13 + VALUE_TAG_OFF]
    movzx edi, byte [rdx + VALUE_TAG_OFF]

    cmp esi, VTAG_INTEGER
    jne .check_double_b
    cmp edi, VTAG_INTEGER
    jne .a_not_int

    ; Both integers
    mov rax, [rdx + VALUE_NUM_OFF]
    cmp rax, [r13 + VALUE_NUM_OFF]
    setg dil
    movzx edi, dil
    call stack_data_push_boolean
    jmp dispatch

.a_not_int:
    cmp edi, VTAG_DOUBLE
    jne .type_error
    cvtsi2sd xmm1, qword [r13 + VALUE_NUM_OFF]
    movsd xmm0, [rdx + VALUE_NUM_OFF]
    ucomisd xmm0, xmm1
    seta dil
    movzx edi, dil
    call stack_data_push_boolean
    jmp dispatch

.check_double_b:
    cmp esi, VTAG_DOUBLE
    jne .type_error
    cmp edi, VTAG_DOUBLE
    jne .a_is_int_b_double

    ; Both doubles
    movsd xmm0, [rdx + VALUE_NUM_OFF]
    ucomisd xmm0, [r13 + VALUE_NUM_OFF]
    seta dil
    movzx edi, dil
    call stack_data_push_boolean
    jmp dispatch

.a_is_int_b_double:
    cmp edi, VTAG_INTEGER
    jne .type_error
    cvtsi2sd xmm0, qword [rdx + VALUE_NUM_OFF]
    ucomisd xmm0, [r13 + VALUE_NUM_OFF]
    seta dil
    movzx edi, dil
    call stack_data_push_boolean
    jmp dispatch

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    jmp dispatch

;-----------------------------------------------------------------------------
; op_ge - Greater than or equal comparison: ( a b -- a>=b )
; Uses r13 (callee-saved) to preserve pointer across pop calls
;-----------------------------------------------------------------------------
op_ge:
    call stack_data_pop
    test rax, rax
    jz .error
    mov r13, rax            ; b - saved in callee-saved reg

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax            ; a

    movzx esi, byte [r13 + VALUE_TAG_OFF]
    movzx edi, byte [rdx + VALUE_TAG_OFF]

    cmp esi, VTAG_INTEGER
    jne .check_double_b
    cmp edi, VTAG_INTEGER
    jne .a_not_int

    ; Both integers
    mov rax, [rdx + VALUE_NUM_OFF]
    cmp rax, [r13 + VALUE_NUM_OFF]
    setge dil
    movzx edi, dil
    call stack_data_push_boolean
    jmp dispatch

.a_not_int:
    cmp edi, VTAG_DOUBLE
    jne .type_error
    cvtsi2sd xmm1, qword [r13 + VALUE_NUM_OFF]
    movsd xmm0, [rdx + VALUE_NUM_OFF]
    ucomisd xmm0, xmm1
    setae dil
    movzx edi, dil
    call stack_data_push_boolean
    jmp dispatch

.check_double_b:
    cmp esi, VTAG_DOUBLE
    jne .type_error
    cmp edi, VTAG_DOUBLE
    jne .a_is_int_b_double

    ; Both doubles
    movsd xmm0, [rdx + VALUE_NUM_OFF]
    ucomisd xmm0, [r13 + VALUE_NUM_OFF]
    setae dil
    movzx edi, dil
    call stack_data_push_boolean
    jmp dispatch

.a_is_int_b_double:
    cmp edi, VTAG_INTEGER
    jne .type_error
    cvtsi2sd xmm0, qword [rdx + VALUE_NUM_OFF]
    ucomisd xmm0, [r13 + VALUE_NUM_OFF]
    setae dil
    movzx edi, dil
    call stack_data_push_boolean
    jmp dispatch

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    jmp dispatch

;-----------------------------------------------------------------------------
; Boolean Operations
; Uses r13 (callee-saved) to preserve pointer across pop calls
;-----------------------------------------------------------------------------

op_and:
    call stack_data_pop
    test rax, rax
    jz .error
    mov r13, rax            ; Save in callee-saved reg

    call stack_data_pop
    test rax, rax
    jz .error

    cmp byte [r13 + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .type_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .type_error

    mov rdi, [rax + VALUE_NUM_OFF]
    and rdi, [r13 + VALUE_NUM_OFF]
    call stack_data_push_boolean
    jmp dispatch

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    jmp dispatch

op_or:
    call stack_data_pop
    test rax, rax
    jz .error
    mov r13, rax            ; Save in callee-saved reg

    call stack_data_pop
    test rax, rax
    jz .error

    cmp byte [r13 + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .type_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .type_error

    mov rdi, [rax + VALUE_NUM_OFF]
    or rdi, [r13 + VALUE_NUM_OFF]
    call stack_data_push_boolean
    jmp dispatch

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    jmp dispatch

op_not:
    call stack_data_pop
    test rax, rax
    jz .error

    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .type_error

    mov rdi, [rax + VALUE_NUM_OFF]
    xor rdi, 1
    call stack_data_push_boolean
    jmp dispatch

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    jmp dispatch
