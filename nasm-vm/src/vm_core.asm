; vm_core.asm - DTRules NASM VM Core
; Copyright 2024 Paul Snow
; Licensed under Apache License 2.0
;
; Stack-based bytecode interpreter using jump table dispatch.
; Architecture: x86-64 System V ABI (Linux)
;
; Register conventions:
;   rdi - bytecode pointer (instruction pointer)
;   rsi - bytecode end pointer
;   r12 - data stack pointer (grows downward)
;   r13 - data stack base (limit for underflow check)
;   r14 - data stack end (limit for overflow check)
;   r15 - trace buffer pointer (optional)
;   rbp - preserved frame pointer
;   rsp - C stack pointer (preserved)
;
; Value representation (16 bytes per value):
;   +0: tag (8 bytes, holds type tag)
;   +8: value (8 bytes, int64/double/pointer)
;
; This allows simple 64-bit operations for most numeric work.

%include "src/opcodes.inc"

; Mark stack as non-executable for security
section .note.GNU-stack noalloc noexec nowrite progbits

section .data
    align 8
    ; Jump table for opcode dispatch
    ; Each entry is 8 bytes (pointer to handler)
jump_table:
    dq op_nop           ; 0 - NOP
    dq op_push          ; 1 - PUSH (constant from pool)
    dq op_push_int      ; 2 - PUSH_INT (immediate varint)
    dq op_pop           ; 3 - POP
    dq op_dup           ; 4 - DUP
    dq op_swap          ; 5 - SWAP
    dq op_rot           ; 6 - ROT
    dq op_roll          ; 7 - ROLL
    dq op_index         ; 8 - INDEX (pick)
    dq op_clear         ; 9 - CLEAR
    dq op_mark          ; 10 - MARK

    ; Fill gap 11-19
    dq op_invalid, op_invalid, op_invalid, op_invalid
    dq op_invalid, op_invalid, op_invalid, op_invalid
    dq op_invalid

    ; Arithmetic 20-28
    dq op_add           ; 20 - ADD
    dq op_sub           ; 21 - SUB
    dq op_mul           ; 22 - MUL
    dq op_div           ; 23 - DIV
    dq op_mod           ; 24 - MOD
    dq op_neg           ; 25 - NEG
    dq op_abs           ; 26 - ABS
    dq op_inc           ; 27 - INC
    dq op_dec           ; 28 - DEC

    ; Fill gap 29
    dq op_invalid

    ; Comparison 30-35
    dq op_eq            ; 30 - EQ
    dq op_ne            ; 31 - NE
    dq op_lt            ; 32 - LT
    dq op_le            ; 33 - LE
    dq op_gt            ; 34 - GT
    dq op_ge            ; 35 - GE

    ; Fill gap 36-39
    dq op_invalid, op_invalid, op_invalid, op_invalid

    ; Boolean 40-43
    dq op_and           ; 40 - AND
    dq op_or            ; 41 - OR
    dq op_not           ; 42 - NOT
    dq op_xor           ; 43 - XOR

    ; Fill gap 44-49
    dq op_invalid, op_invalid, op_invalid
    dq op_invalid, op_invalid, op_invalid

    ; Control flow 50-59 (stubs for now)
    dq op_invalid       ; 50 - EXEC
    dq op_invalid       ; 51 - IF
    dq op_invalid       ; 52 - IFELSE
    dq op_invalid       ; 53 - WHILE
    dq op_invalid       ; 54 - FOR
    dq op_invalid       ; 55 - FORALL
    dq op_invalid       ; 56 - RETURN
    dq op_invalid       ; 57 - JUMP
    dq op_invalid       ; 58 - JUMPIF
    dq op_invalid       ; 59 - CALL

    ; Entity operations 60-64 (stubs for now)
    dq op_invalid, op_invalid, op_invalid, op_invalid, op_invalid

    ; Fill gap 65-69
    dq op_invalid, op_invalid, op_invalid, op_invalid, op_invalid

    ; Array operations 70-74 (stubs for now)
    dq op_invalid, op_invalid, op_invalid, op_invalid, op_invalid

    ; Fill gap 75-79
    dq op_invalid, op_invalid, op_invalid, op_invalid, op_invalid

    ; Table operations 80-82 (stubs for now)
    dq op_invalid, op_invalid, op_invalid

    ; Fill gap 83-89
    dq op_invalid, op_invalid, op_invalid, op_invalid
    dq op_invalid, op_invalid, op_invalid

    ; String operations 90-91 (stubs for now)
    dq op_invalid, op_invalid

    ; Fill gap 92-99
    dq op_invalid, op_invalid, op_invalid, op_invalid
    dq op_invalid, op_invalid, op_invalid, op_invalid

    ; Constants 100-104
    dq op_push_true     ; 100 - PUSH_TRUE
    dq op_push_false    ; 101 - PUSH_FALSE
    dq op_push_null     ; 102 - PUSH_NULL
    dq op_push_zero     ; 103 - PUSH_ZERO
    dq op_push_one      ; 104 - PUSH_ONE

    ; Extended stack operations 105-107
    dq op_over          ; 105 - OVER
    dq op_pick          ; 106 - PICK
    dq op_pop           ; 107 - DROP (alias for POP)

jump_table_end:

    ; Error messages
err_underflow_msg: db "Stack underflow", 0
err_overflow_msg:  db "Stack overflow", 0
err_invalid_msg:   db "Invalid opcode", 0
err_divzero_msg:   db "Division by zero", 0

section .bss
    ; VM state structure
    alignb 16
vm_error_code: resq 1       ; Last error code
vm_error_pc:   resq 1       ; PC at error

section .text

; =============================================================================
; VM Entry Point
; =============================================================================
; int vm_execute(VMContext* ctx)
;
; VMContext structure (passed in rdi):
;   +0:  bytecode pointer (uint8_t*)
;   +8:  bytecode length (size_t)
;   +16: constants pool (Value*)
;   +24: constants count (size_t)
;   +32: names pool (RName**)
;   +40: names count (size_t)
;   +48: data stack (Value*)
;   +56: data stack size (size_t, current count)
;   +64: data stack capacity (size_t)
;   +72: trace buffer (uint8_t*, optional, NULL if no trace)
;   +80: trace buffer size (size_t)
; Returns: 0 on success, error code on failure
;
global vm_execute
vm_execute:
    ; Prologue - save callee-saved registers
    push rbp
    mov rbp, rsp
    push r12
    push r13
    push r14
    push r15
    push rbx

    ; Load context into registers
    mov rax, rdi                ; ctx in rax

    ; Load bytecode pointers
    mov rdi, [rax]              ; bytecode pointer
    mov rcx, [rax + 8]          ; bytecode length
    lea rsi, [rdi + rcx]        ; bytecode end = start + length

    ; Load data stack
    mov r12, [rax + 48]         ; data stack base
    mov rcx, [rax + 56]         ; current stack size
    shl rcx, 4                  ; rcx = size * 16 (sizeof Value)
    add r12, rcx                ; current stack top = base + size*16
    mov r13, [rax + 48]         ; stack base (underflow limit)
    mov rcx, [rax + 64]         ; stack capacity
    shl rcx, 4                  ; rcx = capacity * 16
    mov r14, [rax + 48]
    add r14, rcx                ; stack end = base + capacity*16 (overflow limit)

    ; Load trace buffer (can be NULL)
    mov r15, [rax + 72]         ; trace buffer pointer

    ; Save context pointer for later
    mov rbx, rax

    ; Clear error state
    xor eax, eax
    mov [rel vm_error_code], rax

    ; Begin dispatch loop
    jmp dispatch

; =============================================================================
; Dispatch Loop
; =============================================================================
dispatch:
    ; Check if we've reached the end of bytecode
    cmp rdi, rsi
    jae .done

    ; Fetch next opcode
    movzx eax, byte [rdi]
    inc rdi

    ; Bounds check opcode
    cmp eax, 108                ; max valid opcode + 1
    jae op_invalid

    ; Jump table dispatch
    lea rcx, [rel jump_table]
    jmp [rcx + rax*8]

.done:
    ; Success - calculate final stack size and store back
    mov rax, r12
    sub rax, [rbx + 48]         ; current top - base
    shr rax, 4                  ; divide by 16 (sizeof Value)
    mov [rbx + 56], rax         ; store new stack size

    xor eax, eax                ; return 0 (success)
    jmp vm_exit

; =============================================================================
; VM Exit
; =============================================================================
vm_exit:
    ; Epilogue - restore callee-saved registers
    pop rbx
    pop r15
    pop r14
    pop r13
    pop r12
    pop rbp
    ret

; =============================================================================
; Error Handlers
; =============================================================================
error_underflow:
    mov qword [rel vm_error_code], ERR_STACK_UNDERFLOW
    mov rax, rdi
    sub rax, 1
    mov [rel vm_error_pc], rax
    mov eax, ERR_STACK_UNDERFLOW
    jmp vm_exit

error_overflow:
    mov qword [rel vm_error_code], ERR_STACK_OVERFLOW
    mov rax, rdi
    sub rax, 1
    mov [rel vm_error_pc], rax
    mov eax, ERR_STACK_OVERFLOW
    jmp vm_exit

error_divzero:
    mov qword [rel vm_error_code], ERR_DIV_BY_ZERO
    mov rax, rdi
    sub rax, 1
    mov [rel vm_error_pc], rax
    mov eax, ERR_DIV_BY_ZERO
    jmp vm_exit

op_invalid:
    mov qword [rel vm_error_code], ERR_INVALID_OPCODE
    mov rax, rdi
    sub rax, 1
    mov [rel vm_error_pc], rax
    mov eax, ERR_INVALID_OPCODE
    jmp vm_exit

; =============================================================================
; Stack Operations
; =============================================================================

; op_nop: ( -- ) no operation
op_nop:
    jmp dispatch

; op_pop: ( a -- ) removes top element
op_pop:
    ; Check underflow
    cmp r12, r13
    je error_underflow

    ; Pop (just decrement stack pointer)
    sub r12, 16
    jmp dispatch

; op_dup: ( a -- a a ) duplicate top element
op_dup:
    ; Check underflow
    cmp r12, r13
    je error_underflow

    ; Check overflow
    lea rax, [r12 + 16]
    cmp rax, r14
    ja error_overflow

    ; Get top element
    mov rax, [r12 - 16]         ; tag
    mov rcx, [r12 - 8]          ; value

    ; Push copy
    mov [r12], rax
    mov [r12 + 8], rcx
    add r12, 16
    jmp dispatch

; op_swap: ( a b -- b a ) swap top two elements
op_swap:
    ; Check underflow (need 2 elements)
    lea rax, [r13 + 32]
    cmp r12, rax
    jb error_underflow

    ; Load both elements
    mov rax, [r12 - 16]         ; b.tag
    mov rcx, [r12 - 8]          ; b.value
    mov r8, [r12 - 32]          ; a.tag
    mov r9, [r12 - 24]          ; a.value

    ; Swap
    mov [r12 - 16], r8          ; a.tag -> TOS
    mov [r12 - 8], r9           ; a.value -> TOS
    mov [r12 - 32], rax         ; b.tag -> TOS-1
    mov [r12 - 24], rcx         ; b.value -> TOS-1
    jmp dispatch

; op_over: ( a b -- a b a ) copy second element to top
op_over:
    ; Check underflow (need 2 elements)
    lea rax, [r13 + 32]
    cmp r12, rax
    jb error_underflow

    ; Check overflow
    lea rax, [r12 + 16]
    cmp rax, r14
    ja error_overflow

    ; Get second element (a)
    mov rax, [r12 - 32]         ; a.tag
    mov rcx, [r12 - 24]         ; a.value

    ; Push copy
    mov [r12], rax
    mov [r12 + 8], rcx
    add r12, 16
    jmp dispatch

; op_rot: ( a b c -- b c a ) rotate third element to top
op_rot:
    ; Check underflow (need 3 elements)
    lea rax, [r13 + 48]
    cmp r12, rax
    jb error_underflow

    ; Load all three: a at -48, b at -32, c at -16
    mov rax, [r12 - 48]         ; a.tag
    mov rcx, [r12 - 40]         ; a.value
    mov r8, [r12 - 32]          ; b.tag
    mov r9, [r12 - 24]          ; b.value
    mov r10, [r12 - 16]         ; c.tag
    mov r11, [r12 - 8]          ; c.value

    ; Rotate: b->-48, c->-32, a->-16
    mov [r12 - 48], r8          ; b.tag
    mov [r12 - 40], r9          ; b.value
    mov [r12 - 32], r10         ; c.tag
    mov [r12 - 24], r11         ; c.value
    mov [r12 - 16], rax         ; a.tag
    mov [r12 - 8], rcx          ; a.value
    jmp dispatch

; op_roll: ( ... n -- ... ) roll n elements
; TODO: Implement full roll with variable n
op_roll:
    ; For now, just read n and do nothing (placeholder)
    cmp r12, r13
    je error_underflow
    sub r12, 16
    jmp dispatch

; op_index: ( ... n -- ... nth ) copy nth element to top (pick)
; op_pick is alias
op_index:
op_pick:
    ; Check underflow for n
    cmp r12, r13
    je error_underflow

    ; Get n from stack
    mov rax, [r12 - 8]          ; n value (assume integer)
    sub r12, 16                 ; pop n

    ; Calculate offset: (n+1)*16 bytes from new TOS
    ; TOS is at r12-16 (n=0), second is at r12-32 (n=1), etc.
    add rax, 1
    shl rax, 4                  ; multiply by 16

    ; Check bounds
    mov rcx, r12
    sub rcx, rax
    cmp rcx, r13
    jb error_underflow

    ; Check overflow for push
    lea rcx, [r12 + 16]
    cmp rcx, r14
    ja error_overflow

    ; Get nth element
    neg rax
    mov rcx, [r12 + rax]        ; tag
    mov r8, [r12 + rax + 8]     ; value

    ; Push copy
    mov [r12], rcx
    mov [r12 + 8], r8
    add r12, 16
    jmp dispatch

; op_clear: ( ... -- ) clear stack
op_clear:
    mov r12, r13                ; reset stack pointer to base
    jmp dispatch

; op_mark: ( -- mark ) push mark onto stack
op_mark:
    ; Check overflow
    lea rax, [r12 + 16]
    cmp rax, r14
    ja error_overflow

    ; Push mark value (tag=VTAG_MARK, value=0)
    mov qword [r12], VTAG_MARK
    mov qword [r12 + 8], 0
    add r12, 16
    jmp dispatch

; =============================================================================
; Push Operations
; =============================================================================

; op_push_true: ( -- true )
op_push_true:
    lea rax, [r12 + 16]
    cmp rax, r14
    ja error_overflow

    mov qword [r12], VTAG_BOOLEAN
    mov qword [r12 + 8], 1
    add r12, 16
    jmp dispatch

; op_push_false: ( -- false )
op_push_false:
    lea rax, [r12 + 16]
    cmp rax, r14
    ja error_overflow

    mov qword [r12], VTAG_BOOLEAN
    mov qword [r12 + 8], 0
    add r12, 16
    jmp dispatch

; op_push_null: ( -- null )
op_push_null:
    lea rax, [r12 + 16]
    cmp rax, r14
    ja error_overflow

    mov qword [r12], VTAG_NULL
    mov qword [r12 + 8], 0
    add r12, 16
    jmp dispatch

; op_push_zero: ( -- 0 )
op_push_zero:
    lea rax, [r12 + 16]
    cmp rax, r14
    ja error_overflow

    mov qword [r12], VTAG_INTEGER
    mov qword [r12 + 8], 0
    add r12, 16
    jmp dispatch

; op_push_one: ( -- 1 )
op_push_one:
    lea rax, [r12 + 16]
    cmp rax, r14
    ja error_overflow

    mov qword [r12], VTAG_INTEGER
    mov qword [r12 + 8], 1
    add r12, 16
    jmp dispatch

; op_push_int: ( -- n ) push immediate integer (read varint)
op_push_int:
    ; Check overflow
    lea rax, [r12 + 16]
    cmp rax, r14
    ja error_overflow

    ; Read varint from bytecode
    xor eax, eax                ; accumulator
    xor ecx, ecx                ; shift counter
.read_loop:
    cmp rdi, rsi
    jae .done_read
    movzx r8d, byte [rdi]
    inc rdi

    mov r9, r8
    and r9, 0x7f
    shl r9, cl
    or rax, r9

    add cl, 7
    test r8d, 0x80
    jnz .read_loop

.done_read:
    ; Sign extend if needed (values -64 to 63 in 7 bits)
    ; For simplicity, treat as unsigned for now

    ; Push integer
    mov qword [r12], VTAG_INTEGER
    mov [r12 + 8], rax
    add r12, 16
    jmp dispatch

; op_push: ( -- constant ) push constant from pool
op_push:
    ; Read constant index (varint)
    xor eax, eax
    xor ecx, ecx
.read_loop:
    cmp rdi, rsi
    jae .done_read
    movzx r8d, byte [rdi]
    inc rdi

    mov r9, r8
    and r9, 0x7f
    shl r9, cl
    or rax, r9

    add cl, 7
    test r8d, 0x80
    jnz .read_loop

.done_read:
    ; TODO: Look up constant from pool using rbx (context)
    ; For now, just push 0
    lea rcx, [r12 + 16]
    cmp rcx, r14
    ja error_overflow

    mov qword [r12], VTAG_INTEGER
    mov qword [r12 + 8], 0
    add r12, 16
    jmp dispatch

; =============================================================================
; Arithmetic Operations
; =============================================================================

; op_add: ( a b -- a+b )
op_add:
    lea rax, [r13 + 32]
    cmp r12, rax
    jb error_underflow

    ; Get both values (assume integers for now)
    mov rax, [r12 - 8]          ; b
    mov rcx, [r12 - 24]         ; a
    add rcx, rax

    ; Store result, pop one element
    sub r12, 16
    mov qword [r12 - 16], VTAG_INTEGER
    mov [r12 - 8], rcx
    jmp dispatch

; op_sub: ( a b -- a-b )
op_sub:
    lea rax, [r13 + 32]
    cmp r12, rax
    jb error_underflow

    mov rax, [r12 - 8]          ; b
    mov rcx, [r12 - 24]         ; a
    sub rcx, rax

    sub r12, 16
    mov qword [r12 - 16], VTAG_INTEGER
    mov [r12 - 8], rcx
    jmp dispatch

; op_mul: ( a b -- a*b )
op_mul:
    lea rax, [r13 + 32]
    cmp r12, rax
    jb error_underflow

    mov rax, [r12 - 8]          ; b
    mov rcx, [r12 - 24]         ; a
    imul rcx, rax

    sub r12, 16
    mov qword [r12 - 16], VTAG_INTEGER
    mov [r12 - 8], rcx
    jmp dispatch

; op_div: ( a b -- a/b )
op_div:
    lea rax, [r13 + 32]
    cmp r12, rax
    jb error_underflow

    mov rcx, [r12 - 8]          ; b (divisor)
    test rcx, rcx
    jz error_divzero

    mov rax, [r12 - 24]         ; a (dividend)
    cqo                         ; sign extend rax to rdx:rax
    idiv rcx                    ; rax = quotient

    sub r12, 16
    mov qword [r12 - 16], VTAG_INTEGER
    mov [r12 - 8], rax
    jmp dispatch

; op_mod: ( a b -- a%b )
op_mod:
    lea rax, [r13 + 32]
    cmp r12, rax
    jb error_underflow

    mov rcx, [r12 - 8]          ; b (divisor)
    test rcx, rcx
    jz error_divzero

    mov rax, [r12 - 24]         ; a (dividend)
    cqo                         ; sign extend
    idiv rcx                    ; rdx = remainder

    sub r12, 16
    mov qword [r12 - 16], VTAG_INTEGER
    mov [r12 - 8], rdx
    jmp dispatch

; op_neg: ( a -- -a )
op_neg:
    cmp r12, r13
    je error_underflow

    mov rax, [r12 - 8]
    neg rax
    mov [r12 - 8], rax
    jmp dispatch

; op_abs: ( a -- |a| )
op_abs:
    cmp r12, r13
    je error_underflow

    mov rax, [r12 - 8]
    mov rcx, rax
    neg rcx
    cmovs rcx, rax              ; if original was positive, keep it
    mov [r12 - 8], rcx
    jmp dispatch

; op_inc: ( a -- a+1 )
op_inc:
    cmp r12, r13
    je error_underflow

    inc qword [r12 - 8]
    jmp dispatch

; op_dec: ( a -- a-1 )
op_dec:
    cmp r12, r13
    je error_underflow

    dec qword [r12 - 8]
    jmp dispatch

; =============================================================================
; Comparison Operations
; =============================================================================

; op_eq: ( a b -- a==b )
op_eq:
    lea rax, [r13 + 32]
    cmp r12, rax
    jb error_underflow

    mov rax, [r12 - 8]          ; b
    cmp [r12 - 24], rax         ; compare with a
    sete al
    movzx eax, al

    sub r12, 16
    mov qword [r12 - 16], VTAG_BOOLEAN
    mov [r12 - 8], rax
    jmp dispatch

; op_ne: ( a b -- a!=b )
op_ne:
    lea rax, [r13 + 32]
    cmp r12, rax
    jb error_underflow

    mov rax, [r12 - 8]
    cmp [r12 - 24], rax
    setne al
    movzx eax, al

    sub r12, 16
    mov qword [r12 - 16], VTAG_BOOLEAN
    mov [r12 - 8], rax
    jmp dispatch

; op_lt: ( a b -- a<b )
op_lt:
    lea rax, [r13 + 32]
    cmp r12, rax
    jb error_underflow

    mov rax, [r12 - 8]          ; b
    mov rcx, [r12 - 24]         ; a
    cmp rcx, rax
    setl al
    movzx eax, al

    sub r12, 16
    mov qword [r12 - 16], VTAG_BOOLEAN
    mov [r12 - 8], rax
    jmp dispatch

; op_le: ( a b -- a<=b )
op_le:
    lea rax, [r13 + 32]
    cmp r12, rax
    jb error_underflow

    mov rax, [r12 - 8]
    mov rcx, [r12 - 24]
    cmp rcx, rax
    setle al
    movzx eax, al

    sub r12, 16
    mov qword [r12 - 16], VTAG_BOOLEAN
    mov [r12 - 8], rax
    jmp dispatch

; op_gt: ( a b -- a>b )
op_gt:
    lea rax, [r13 + 32]
    cmp r12, rax
    jb error_underflow

    mov rax, [r12 - 8]
    mov rcx, [r12 - 24]
    cmp rcx, rax
    setg al
    movzx eax, al

    sub r12, 16
    mov qword [r12 - 16], VTAG_BOOLEAN
    mov [r12 - 8], rax
    jmp dispatch

; op_ge: ( a b -- a>=b )
op_ge:
    lea rax, [r13 + 32]
    cmp r12, rax
    jb error_underflow

    mov rax, [r12 - 8]
    mov rcx, [r12 - 24]
    cmp rcx, rax
    setge al
    movzx eax, al

    sub r12, 16
    mov qword [r12 - 16], VTAG_BOOLEAN
    mov [r12 - 8], rax
    jmp dispatch

; =============================================================================
; Boolean Operations
; =============================================================================

; op_and: ( a b -- a&&b )
op_and:
    lea rax, [r13 + 32]
    cmp r12, rax
    jb error_underflow

    mov rax, [r12 - 8]          ; b
    mov rcx, [r12 - 24]         ; a
    test rax, rax
    setnz al
    test rcx, rcx
    setnz cl
    and al, cl
    movzx eax, al

    sub r12, 16
    mov qword [r12 - 16], VTAG_BOOLEAN
    mov [r12 - 8], rax
    jmp dispatch

; op_or: ( a b -- a||b )
op_or:
    lea rax, [r13 + 32]
    cmp r12, rax
    jb error_underflow

    mov rax, [r12 - 8]
    mov rcx, [r12 - 24]
    or rax, rcx
    test rax, rax
    setnz al
    movzx eax, al

    sub r12, 16
    mov qword [r12 - 16], VTAG_BOOLEAN
    mov [r12 - 8], rax
    jmp dispatch

; op_not: ( a -- !a )
op_not:
    cmp r12, r13
    je error_underflow

    mov rax, [r12 - 8]
    test rax, rax
    setz al
    movzx eax, al

    mov qword [r12 - 16], VTAG_BOOLEAN
    mov [r12 - 8], rax
    jmp dispatch

; op_xor: ( a b -- a^b )
op_xor:
    lea rax, [r13 + 32]
    cmp r12, rax
    jb error_underflow

    mov rax, [r12 - 8]          ; b
    mov rcx, [r12 - 24]         ; a
    test rax, rax
    setnz al
    test rcx, rcx
    setnz cl
    xor al, cl
    movzx eax, al

    sub r12, 16
    mov qword [r12 - 16], VTAG_BOOLEAN
    mov [r12 - 8], rax
    jmp dispatch

; =============================================================================
; Utility Functions (exported for C interop)
; =============================================================================

; Get last error code
global vm_get_error
vm_get_error:
    mov rax, [rel vm_error_code]
    ret

; Get PC at error
global vm_get_error_pc
vm_get_error_pc:
    mov rax, [rel vm_error_pc]
    ret
