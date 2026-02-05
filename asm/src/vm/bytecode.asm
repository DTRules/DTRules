; bytecode.asm - VM execution loop
; DTRules Assembly Implementation
; Main bytecode interpreter with opcode dispatch

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state
extern stack_data_push
extern stack_data_pop
extern stack_data_peek
extern stack_data_push_integer
extern stack_data_push_boolean
extern stack_data_push_null
extern stack_data_push_double
extern stack_data_dup
extern stack_data_swap
extern stack_data_rot
extern stack_data_over
extern stack_data_pick
extern stack_data_roll
extern stack_data_drop
extern stack_data_clear
extern print_value_ln
extern print_integer
extern print_newline

;-----------------------------------------------------------------------------
; Opcode handler table
;-----------------------------------------------------------------------------
section .data
    align 8
    global opcode_table
    opcode_table:
        ; 0x00-0x0F: Stack operations
        dq op_nop           ; 0x00
        dq op_push_null     ; 0x01
        dq op_push_true     ; 0x02
        dq op_push_false    ; 0x03
        dq op_push_int      ; 0x04
        dq op_push_double   ; 0x05
        dq op_push_string   ; 0x06
        dq op_push_name     ; 0x07
        dq op_dup           ; 0x08
        dq op_swap          ; 0x09
        dq op_pop           ; 0x0A
        dq op_rot           ; 0x0B
        dq op_over          ; 0x0C
        dq op_pick          ; 0x0D
        dq op_roll          ; 0x0E
        dq op_clear         ; 0x0F

        ; 0x10-0x1F: Arithmetic
        dq op_add           ; 0x10
        dq op_sub           ; 0x11
        dq op_mul           ; 0x12
        dq op_div           ; 0x13
        dq op_mod           ; 0x14
        dq op_neg           ; 0x15
        dq op_abs           ; 0x16
        dq op_min           ; 0x17
        dq op_max           ; 0x18
        dq op_floor         ; 0x19
        dq op_ceil          ; 0x1A
        dq op_round         ; 0x1B
        dq op_truncate      ; 0x1C
        dq op_pow           ; 0x1D
        dq op_invalid       ; 0x1E
        dq op_invalid       ; 0x1F

        ; 0x20-0x2F: Comparison/Logic
        dq op_eq            ; 0x20
        dq op_ne            ; 0x21
        dq op_lt            ; 0x22
        dq op_le            ; 0x23
        dq op_gt            ; 0x24
        dq op_ge            ; 0x25
        dq op_and           ; 0x26
        dq op_or            ; 0x27
        dq op_not           ; 0x28
        dq op_xor           ; 0x29
        dq op_invalid       ; 0x2A
        dq op_invalid       ; 0x2B
        dq op_invalid       ; 0x2C
        dq op_invalid       ; 0x2D
        dq op_invalid       ; 0x2E
        dq op_invalid       ; 0x2F

        ; 0x30-0x4F: String operations (placeholders for now)
        times 32 dq op_invalid

        ; 0x50-0x5F: Array operations (placeholders)
        times 16 dq op_invalid

        ; 0x60-0x6F: Control flow
        dq op_if            ; 0x60
        dq op_ifelse        ; 0x61
        dq op_while         ; 0x62
        dq op_for           ; 0x63
        dq op_forall        ; 0x64
        dq op_break         ; 0x65
        dq op_continue      ; 0x66
        dq op_return        ; 0x67
        dq op_exec          ; 0x68
        dq op_call          ; 0x69
        dq op_jump          ; 0x6A
        dq op_jumpif        ; 0x6B
        dq op_jumpifnot     ; 0x6C
        dq op_invalid       ; 0x6D
        dq op_invalid       ; 0x6E
        dq op_invalid       ; 0x6F

        ; 0x70-0x7F: Entity operations (placeholders)
        times 16 dq op_invalid

        ; 0x80-0x8F: Decision table operations (placeholders)
        times 16 dq op_invalid

        ; 0x90-0xEF: Reserved
        times 96 dq op_invalid

        ; 0xF0-0xFF: Debug/utility
        dq op_print         ; 0xF0
        dq op_trace         ; 0xF1
        dq op_debug         ; 0xF2
        times 12 dq op_invalid
        dq op_halt          ; 0xFF

section .text

;-----------------------------------------------------------------------------
; vm_execute - Main execution loop
; Input: Bytecode should be loaded in state.bytecode
; Returns: 0 on success, error code on failure
;-----------------------------------------------------------------------------
global vm_execute
vm_execute:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14
    push r15

    ; r15 = state pointer (already set)
    ; r12 = data stack pointer (already set)
    ; r13 = entity stack pointer (already set)
    ; r14 = control stack pointer (already set)

    ; rbx = bytecode instruction pointer
    mov rbx, [state + State.bytecode]

    ; Clear halted flag
    mov dword [state + State.halted], 0

.fetch:
    ; Check if halted
    cmp dword [state + State.halted], 0
    jne .done

    ; Check bytecode bounds
    cmp rbx, [state + State.bytecode_end]
    jae .done

    ; Fetch opcode
    movzx eax, byte [rbx]
    inc rbx                 ; Advance IP

    ; Dispatch
    lea rcx, [opcode_table]
    mov rcx, [rcx + rax * 8]
    call rcx

    ; Check for errors
    cmp dword [state + State.error], ERR_NONE
    jne .error

    jmp .fetch

.error:
    mov eax, [state + State.error]
    jmp .exit

.done:
    xor eax, eax

.exit:
    ; Save bytecode pointer back
    mov [state + State.bytecode], rbx

    pop r15
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; Bytecode reading helpers
;-----------------------------------------------------------------------------

; Read unsigned varint from bytecode
; Output: rax = value, rbx advanced past varint
read_varint:
    xor eax, eax
    xor ecx, ecx            ; Shift amount

.loop:
    movzx edx, byte [rbx]
    inc rbx

    push rcx
    mov cl, dl
    and cl, 0x7F
    movzx ecx, cl
    pop r8                  ; Restore shift to r8
    push r8
    mov cl, r8b
    shl rcx, cl
    pop r8
    or rax, rcx

    test dl, 0x80
    jz .done

    add r8, 7
    mov ecx, r8d
    cmp ecx, 64
    jb .loop

.done:
    ret

; Read signed varint (zigzag encoded)
; Output: rax = value
read_svarint:
    call read_varint
    ; Zigzag decode: (n >> 1) ^ -(n & 1)
    mov rcx, rax
    shr rax, 1
    and ecx, 1
    neg rcx
    xor rax, rcx
    ret

; Read 8-byte value from bytecode
; Output: rax = value
read_qword:
    mov rax, [rbx]
    add rbx, 8
    ret

;-----------------------------------------------------------------------------
; Opcode implementations
;-----------------------------------------------------------------------------

op_nop:
    ret

op_push_null:
    call stack_data_push_null
    ret

op_push_true:
    mov rdi, 1
    call stack_data_push_boolean
    ret

op_push_false:
    xor edi, edi
    call stack_data_push_boolean
    ret

op_push_int:
    call read_svarint
    mov rdi, rax
    call stack_data_push_integer
    ret

op_push_double:
    call read_qword
    movq xmm0, rax
    call stack_data_push_double
    ret

op_push_string:
    ; TODO: Implement string push
    ret

op_push_name:
    ; TODO: Implement name push
    ret

op_dup:
    call stack_data_dup
    ret

op_swap:
    call stack_data_swap
    ret

op_pop:
    call stack_data_drop
    ret

op_rot:
    call stack_data_rot
    ret

op_over:
    call stack_data_over
    ret

op_pick:
    call read_varint
    mov rdi, rax
    call stack_data_pick
    ret

op_roll:
    call read_varint
    mov rdi, rax
    call stack_data_roll
    ret

op_clear:
    call stack_data_clear
    ret

;-----------------------------------------------------------------------------
; Arithmetic operations
;-----------------------------------------------------------------------------

op_add:
    ; Pop two values, add, push result
    call stack_data_pop
    test rax, rax
    jz .error
    mov rcx, rax            ; First operand

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax            ; Second operand

    ; Check types - both must be numeric
    movzx esi, byte [rcx + VALUE_TAG_OFF]
    movzx edi, byte [rdx + VALUE_TAG_OFF]

    ; Both integers?
    cmp esi, VTAG_INTEGER
    jne .check_double
    cmp edi, VTAG_INTEGER
    jne .mixed

    ; Integer addition
    mov rdi, [rdx + VALUE_NUM_OFF]
    add rdi, [rcx + VALUE_NUM_OFF]
    call stack_data_push_integer
    ret

.check_double:
    cmp esi, VTAG_DOUBLE
    jne .type_error
    cmp edi, VTAG_DOUBLE
    jne .mixed

    ; Double addition
    movsd xmm0, [rdx + VALUE_NUM_OFF]
    addsd xmm0, [rcx + VALUE_NUM_OFF]
    call stack_data_push_double
    ret

.mixed:
    ; Convert to double and add
    ; For now, report type error
.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    ret

op_sub:
    call stack_data_pop
    test rax, rax
    jz .error
    mov rcx, rax

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax

    movzx esi, byte [rcx + VALUE_TAG_OFF]
    movzx edi, byte [rdx + VALUE_TAG_OFF]

    cmp esi, VTAG_INTEGER
    jne .check_double
    cmp edi, VTAG_INTEGER
    jne .type_error

    mov rdi, [rdx + VALUE_NUM_OFF]
    sub rdi, [rcx + VALUE_NUM_OFF]
    call stack_data_push_integer
    ret

.check_double:
    cmp esi, VTAG_DOUBLE
    jne .type_error
    cmp edi, VTAG_DOUBLE
    jne .type_error

    movsd xmm0, [rdx + VALUE_NUM_OFF]
    subsd xmm0, [rcx + VALUE_NUM_OFF]
    call stack_data_push_double
    ret

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    ret

op_mul:
    call stack_data_pop
    test rax, rax
    jz .error
    mov rcx, rax

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax

    movzx esi, byte [rcx + VALUE_TAG_OFF]
    movzx edi, byte [rdx + VALUE_TAG_OFF]

    cmp esi, VTAG_INTEGER
    jne .check_double
    cmp edi, VTAG_INTEGER
    jne .type_error

    mov rax, [rdx + VALUE_NUM_OFF]
    imul rax, [rcx + VALUE_NUM_OFF]
    mov rdi, rax
    call stack_data_push_integer
    ret

.check_double:
    cmp esi, VTAG_DOUBLE
    jne .type_error
    cmp edi, VTAG_DOUBLE
    jne .type_error

    movsd xmm0, [rdx + VALUE_NUM_OFF]
    mulsd xmm0, [rcx + VALUE_NUM_OFF]
    call stack_data_push_double
    ret

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    ret

op_div:
    call stack_data_pop
    test rax, rax
    jz .error
    mov rcx, rax

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax

    movzx esi, byte [rcx + VALUE_TAG_OFF]
    movzx edi, byte [rdx + VALUE_TAG_OFF]

    cmp esi, VTAG_INTEGER
    jne .check_double
    cmp edi, VTAG_INTEGER
    jne .type_error

    ; Check for division by zero
    mov rax, [rcx + VALUE_NUM_OFF]
    test rax, rax
    jz .div_zero

    mov rax, [rdx + VALUE_NUM_OFF]
    cqo
    idiv qword [rcx + VALUE_NUM_OFF]
    mov rdi, rax
    call stack_data_push_integer
    ret

.check_double:
    cmp esi, VTAG_DOUBLE
    jne .type_error
    cmp edi, VTAG_DOUBLE
    jne .type_error

    movsd xmm0, [rdx + VALUE_NUM_OFF]
    divsd xmm0, [rcx + VALUE_NUM_OFF]
    call stack_data_push_double
    ret

.div_zero:
    mov dword [state + State.error], ERR_DIV_BY_ZERO
    ret

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    ret

op_mod:
    call stack_data_pop
    test rax, rax
    jz .error
    mov rcx, rax

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax

    ; Only integer mod
    cmp byte [rcx + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error
    cmp byte [rdx + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error

    mov rax, [rcx + VALUE_NUM_OFF]
    test rax, rax
    jz .div_zero

    mov rax, [rdx + VALUE_NUM_OFF]
    cqo
    idiv qword [rcx + VALUE_NUM_OFF]
    mov rdi, rdx            ; Remainder in rdx
    call stack_data_push_integer
    ret

.div_zero:
    mov dword [state + State.error], ERR_DIV_BY_ZERO
    ret

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    ret

op_neg:
    call stack_data_peek
    test rax, rax
    jz .error

    movzx ecx, byte [rax + VALUE_TAG_OFF]

    cmp cl, VTAG_INTEGER
    jne .check_double

    neg qword [rax + VALUE_NUM_OFF]
    ret

.check_double:
    cmp cl, VTAG_DOUBLE
    jne .type_error

    ; Negate double by XOR with sign bit
    mov rcx, [rax + VALUE_NUM_OFF]
    mov rdx, 0x8000000000000000
    xor rcx, rdx
    mov [rax + VALUE_NUM_OFF], rcx
    ret

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    ret

op_abs:
    call stack_data_peek
    test rax, rax
    jz .error

    movzx ecx, byte [rax + VALUE_TAG_OFF]

    cmp cl, VTAG_INTEGER
    jne .check_double

    mov rdx, [rax + VALUE_NUM_OFF]
    test rdx, rdx
    jns .done
    neg rdx
    mov [rax + VALUE_NUM_OFF], rdx
    ret

.check_double:
    cmp cl, VTAG_DOUBLE
    jne .type_error

    mov rcx, [rax + VALUE_NUM_OFF]
    mov rdx, 0x7FFFFFFFFFFFFFFF
    and rcx, rdx
    mov [rax + VALUE_NUM_OFF], rcx
    ret

.done:
    ret

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    ret

op_min:
    call stack_data_pop
    test rax, rax
    jz .error
    mov rcx, rax

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax

    cmp byte [rcx + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error
    cmp byte [rdx + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error

    mov rdi, [rcx + VALUE_NUM_OFF]
    mov rax, [rdx + VALUE_NUM_OFF]
    cmp rax, rdi
    cmovl rdi, rax
    call stack_data_push_integer
    ret

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    ret

op_max:
    call stack_data_pop
    test rax, rax
    jz .error
    mov rcx, rax

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax

    cmp byte [rcx + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error
    cmp byte [rdx + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error

    mov rdi, [rcx + VALUE_NUM_OFF]
    mov rax, [rdx + VALUE_NUM_OFF]
    cmp rax, rdi
    cmovg rdi, rax
    call stack_data_push_integer
    ret

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    ret

op_floor:
    ; TODO: Implement floor for doubles
    ret

op_ceil:
    ; TODO: Implement ceil for doubles
    ret

op_round:
    ; TODO: Implement round for doubles
    ret

op_truncate:
    ; TODO: Implement truncate for doubles
    ret

op_pow:
    ; TODO: Implement power
    ret

;-----------------------------------------------------------------------------
; Comparison operations
;-----------------------------------------------------------------------------

op_eq:
    call stack_data_pop
    test rax, rax
    jz .error
    mov rcx, rax

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax

    ; Compare based on types
    movzx esi, byte [rcx + VALUE_TAG_OFF]
    movzx edi, byte [rdx + VALUE_TAG_OFF]
    cmp esi, edi
    jne .not_equal

    ; Same type - compare values
    cmp esi, VTAG_NULL
    je .equal               ; Both null

    cmp esi, VTAG_INTEGER
    je .cmp_int
    cmp esi, VTAG_BOOLEAN
    je .cmp_int
    cmp esi, VTAG_DOUBLE
    je .cmp_double

    ; For complex types, compare pointers
    mov rax, [rcx + VALUE_PTR_OFF]
    cmp rax, [rdx + VALUE_PTR_OFF]
    je .equal
    jmp .not_equal

.cmp_int:
    mov rax, [rcx + VALUE_NUM_OFF]
    cmp rax, [rdx + VALUE_NUM_OFF]
    je .equal
    jmp .not_equal

.cmp_double:
    movsd xmm0, [rcx + VALUE_NUM_OFF]
    ucomisd xmm0, [rdx + VALUE_NUM_OFF]
    jp .not_equal           ; NaN
    je .equal
    jmp .not_equal

.equal:
    mov rdi, 1
    call stack_data_push_boolean
    ret

.not_equal:
    xor edi, edi
    call stack_data_push_boolean
    ret

.error:
    ret

op_ne:
    call op_eq
    ; Invert result
    call stack_data_peek
    test rax, rax
    jz .done
    xor qword [rax + VALUE_NUM_OFF], 1
.done:
    ret

op_lt:
    call stack_data_pop
    test rax, rax
    jz .error
    mov rcx, rax            ; Right operand

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax            ; Left operand

    cmp byte [rcx + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error
    cmp byte [rdx + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error

    mov rax, [rdx + VALUE_NUM_OFF]
    cmp rax, [rcx + VALUE_NUM_OFF]
    setl dil
    movzx edi, dil
    call stack_data_push_boolean
    ret

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    ret

op_le:
    call stack_data_pop
    test rax, rax
    jz .error
    mov rcx, rax

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax

    cmp byte [rcx + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error
    cmp byte [rdx + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error

    mov rax, [rdx + VALUE_NUM_OFF]
    cmp rax, [rcx + VALUE_NUM_OFF]
    setle dil
    movzx edi, dil
    call stack_data_push_boolean
    ret

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    ret

op_gt:
    call stack_data_pop
    test rax, rax
    jz .error
    mov rcx, rax

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax

    cmp byte [rcx + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error
    cmp byte [rdx + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error

    mov rax, [rdx + VALUE_NUM_OFF]
    cmp rax, [rcx + VALUE_NUM_OFF]
    setg dil
    movzx edi, dil
    call stack_data_push_boolean
    ret

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    ret

op_ge:
    call stack_data_pop
    test rax, rax
    jz .error
    mov rcx, rax

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax

    cmp byte [rcx + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error
    cmp byte [rdx + VALUE_TAG_OFF], VTAG_INTEGER
    jne .type_error

    mov rax, [rdx + VALUE_NUM_OFF]
    cmp rax, [rcx + VALUE_NUM_OFF]
    setge dil
    movzx edi, dil
    call stack_data_push_boolean
    ret

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    ret

op_and:
    call stack_data_pop
    test rax, rax
    jz .error
    mov rcx, rax

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax

    cmp byte [rcx + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .type_error
    cmp byte [rdx + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .type_error

    mov rax, [rcx + VALUE_NUM_OFF]
    and rax, [rdx + VALUE_NUM_OFF]
    mov rdi, rax
    call stack_data_push_boolean
    ret

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    ret

op_or:
    call stack_data_pop
    test rax, rax
    jz .error
    mov rcx, rax

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax

    cmp byte [rcx + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .type_error
    cmp byte [rdx + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .type_error

    mov rax, [rcx + VALUE_NUM_OFF]
    or rax, [rdx + VALUE_NUM_OFF]
    mov rdi, rax
    call stack_data_push_boolean
    ret

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    ret

op_not:
    call stack_data_peek
    test rax, rax
    jz .error

    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .type_error

    xor qword [rax + VALUE_NUM_OFF], 1
    ret

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    ret

op_xor:
    call stack_data_pop
    test rax, rax
    jz .error
    mov rcx, rax

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax

    cmp byte [rcx + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .type_error
    cmp byte [rdx + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .type_error

    mov rax, [rcx + VALUE_NUM_OFF]
    xor rax, [rdx + VALUE_NUM_OFF]
    mov rdi, rax
    call stack_data_push_boolean
    ret

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    ret

;-----------------------------------------------------------------------------
; Control flow operations (stubs for now)
;-----------------------------------------------------------------------------

op_if:
    ; TODO: Implement conditional
    ret

op_ifelse:
    ; TODO: Implement if-else
    ret

op_while:
    ; TODO: Implement while loop
    ret

op_for:
    ; TODO: Implement for loop
    ret

op_forall:
    ; TODO: Implement forall
    ret

op_break:
    ; TODO: Implement break
    ret

op_continue:
    ; TODO: Implement continue
    ret

op_return:
    ; TODO: Implement return
    ret

op_exec:
    ; TODO: Implement exec
    ret

op_call:
    ; TODO: Implement call
    ret

op_jump:
    ; Read offset and jump
    call read_svarint
    add rbx, rax
    ret

op_jumpif:
    ; Pop condition, jump if true
    call stack_data_pop
    test rax, rax
    jz .no_jump

    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .no_jump

    cmp qword [rax + VALUE_NUM_OFF], 0
    je .no_jump

    call read_svarint
    add rbx, rax
    ret

.no_jump:
    call read_svarint       ; Still need to skip the offset
    ret

op_jumpifnot:
    ; Pop condition, jump if false
    call stack_data_pop
    test rax, rax
    jz .jump                ; Null is falsy

    cmp byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    jne .jump               ; Non-boolean is truthy

    cmp qword [rax + VALUE_NUM_OFF], 0
    jne .no_jump

.jump:
    call read_svarint
    add rbx, rax
    ret

.no_jump:
    call read_svarint
    ret

;-----------------------------------------------------------------------------
; Debug operations
;-----------------------------------------------------------------------------

op_print:
    call stack_data_pop
    test rax, rax
    jz .done
    mov rdi, rax
    call print_value_ln
.done:
    ret

op_trace:
    ; TODO: Implement trace
    ret

op_debug:
    ; TODO: Implement debug
    ret

op_halt:
    mov dword [state + State.halted], 1
    ret

op_invalid:
    mov dword [state + State.error], ERR_INVALID_OPCODE
    ret
