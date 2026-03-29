; print.asm - Output formatting
; DTRules Assembly Implementation
; Functions for printing values and formatted output

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state
extern write_stdout
extern write_stderr
extern value_get_tag
extern value_get_integer
extern value_get_double
extern value_get_boolean
extern value_get_string
extern value_get_array
extern value_get_entity

section .data
    str_true:   db "true"
    str_true_len equ 4

    str_false:  db "false"
    str_false_len equ 5

    str_null:   db "null"
    str_null_len equ 4

    str_array_open:  db "["
    str_array_close: db "]"
    str_comma:       db ", "
    str_entity_open: db "{"
    str_entity_close: db "}"
    str_colon:       db ": "
    str_quote:       db '"'

    newline:    db 10

section .bss
    ; Buffer for number conversion
    num_buffer: resb 32

section .text

;-----------------------------------------------------------------------------
; print_string - Print a null-terminated string to stdout
; Input: rdi = pointer to null-terminated string
;-----------------------------------------------------------------------------
global print_string
print_string:
    push rbp
    mov rbp, rsp
    push rbx

    mov rbx, rdi            ; Save string pointer

    ; Calculate length
    xor ecx, ecx
.count:
    cmp byte [rbx + rcx], 0
    je .print
    inc ecx
    jmp .count

.print:
    mov rdi, rbx
    mov rsi, rcx
    call write_stdout

    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; print_string_len - Print string with explicit length
; Input: rdi = string pointer, rsi = length
;-----------------------------------------------------------------------------
global print_string_len
print_string_len:
    jmp write_stdout

;-----------------------------------------------------------------------------
; print_newline - Print a newline
;-----------------------------------------------------------------------------
global print_newline
print_newline:
    lea rdi, [newline]
    mov rsi, 1
    jmp write_stdout

;-----------------------------------------------------------------------------
; print_char - Print a single character
; Input: dil = character
;-----------------------------------------------------------------------------
global print_char
print_char:
    push rbp
    mov rbp, rsp
    sub rsp, 8

    mov [rsp], dil
    lea rdi, [rsp]
    mov rsi, 1
    call write_stdout

    add rsp, 8
    pop rbp
    ret

;-----------------------------------------------------------------------------
; print_integer - Print a signed 64-bit integer
; Input: rdi = integer value
;-----------------------------------------------------------------------------
global print_integer
print_integer:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    mov rax, rdi
    lea rbx, [num_buffer + 31]  ; Start from end
    mov byte [rbx], 0           ; Null terminator
    mov r12, 0                  ; Negative flag

    ; Handle negative
    test rax, rax
    jns .positive
    neg rax
    mov r12, 1

.positive:
    ; Handle zero
    test rax, rax
    jnz .convert
    dec rbx
    mov byte [rbx], '0'
    jmp .print

.convert:
    mov rcx, 10
.digit_loop:
    test rax, rax
    jz .check_neg
    xor edx, edx
    div rcx
    add dl, '0'
    dec rbx
    mov [rbx], dl
    jmp .digit_loop

.check_neg:
    test r12, r12
    jz .print
    dec rbx
    mov byte [rbx], '-'

.print:
    mov rdi, rbx
    call print_string

    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; print_unsigned - Print an unsigned 64-bit integer
; Input: rdi = unsigned integer value
;-----------------------------------------------------------------------------
global print_unsigned
print_unsigned:
    push rbp
    mov rbp, rsp
    push rbx

    mov rax, rdi
    lea rbx, [num_buffer + 31]
    mov byte [rbx], 0

    test rax, rax
    jnz .convert
    dec rbx
    mov byte [rbx], '0'
    jmp .print

.convert:
    mov rcx, 10
.digit_loop:
    test rax, rax
    jz .print
    xor edx, edx
    div rcx
    add dl, '0'
    dec rbx
    mov [rbx], dl
    jmp .digit_loop

.print:
    mov rdi, rbx
    call print_string

    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; print_hex - Print unsigned integer in hexadecimal
; Input: rdi = value
;-----------------------------------------------------------------------------
global print_hex
print_hex:
    push rbp
    mov rbp, rsp
    push rbx

    mov rax, rdi
    lea rbx, [num_buffer + 31]
    mov byte [rbx], 0

    test rax, rax
    jnz .convert
    dec rbx
    mov byte [rbx], '0'
    jmp .prefix

.convert:
.hex_loop:
    test rax, rax
    jz .prefix
    mov rcx, rax
    and cl, 0x0F
    cmp cl, 10
    jb .is_digit
    add cl, 'a' - 10
    jmp .store
.is_digit:
    add cl, '0'
.store:
    dec rbx
    mov [rbx], cl
    shr rax, 4
    jmp .hex_loop

.prefix:
    dec rbx
    mov byte [rbx], 'x'
    dec rbx
    mov byte [rbx], '0'

    mov rdi, rbx
    call print_string

    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; print_double - Print a double-precision float
; Input: xmm0 = double value
; Note: Simplified implementation - prints 6 decimal places
;-----------------------------------------------------------------------------
global print_double
print_double:
    push rbp
    mov rbp, rsp
    sub rsp, 16

    ; Store double
    movsd [rbp - 8], xmm0

    ; Check for negative
    movq rax, xmm0
    test rax, rax
    jns .positive

    ; Print minus sign
    mov dil, '-'
    call print_char

    ; Make positive
    movsd xmm0, [rbp - 8]
    mov rax, 0x7FFFFFFFFFFFFFFF
    movq xmm1, rax
    andpd xmm0, xmm1
    movsd [rbp - 8], xmm0

.positive:
    ; Get integer part
    movsd xmm0, [rbp - 8]
    cvttsd2si rdi, xmm0     ; Truncate to integer
    push rdi                ; Save integer part

    call print_integer

    ; Print decimal point
    mov dil, '.'
    call print_char

    ; Get fractional part
    pop rdi                 ; Restore integer part
    cvtsi2sd xmm1, rdi      ; Convert back to double
    movsd xmm0, [rbp - 8]
    subsd xmm0, xmm1        ; Fractional part

    ; Multiply by 1000000 for 6 decimal places
    mov rax, 1000000
    cvtsi2sd xmm1, rax
    mulsd xmm0, xmm1
    cvttsd2si rdi, xmm0

    ; Handle negative fractional (shouldn't happen with abs)
    test rdi, rdi
    jns .frac_positive
    neg rdi
.frac_positive:

    ; Print with leading zeros
    mov rax, rdi
    mov rcx, 100000         ; Start with 100000

.frac_loop:
    test rcx, rcx
    jz .frac_done
    xor edx, edx
    div rcx
    add al, '0'
    push rcx
    push rax
    movzx edi, al
    call print_char
    pop rax
    pop rcx
    mov rax, rdx            ; Remainder
    ; Divide divisor by 10
    push rax
    mov rax, rcx
    mov rcx, 10
    xor edx, edx
    div rcx
    mov rcx, rax
    pop rax
    jmp .frac_loop

.frac_done:
    add rsp, 16
    pop rbp
    ret

;-----------------------------------------------------------------------------
; print_value - Print a Value
; Input: rdi = pointer to Value
;-----------------------------------------------------------------------------
global print_value
print_value:
    push rbp
    mov rbp, rsp
    push rbx

    mov rbx, rdi            ; Save value pointer

    ; Get type tag
    movzx eax, byte [rbx + VALUE_TAG_OFF]

    cmp al, VTAG_NULL
    je .print_null
    cmp al, VTAG_INTEGER
    je .print_int
    cmp al, VTAG_DOUBLE
    je .print_dbl
    cmp al, VTAG_BOOLEAN
    je .print_bool
    cmp al, VTAG_STRING
    je .print_str
    cmp al, VTAG_NAME
    je .print_name
    cmp al, VTAG_ARRAY
    je .print_arr
    cmp al, VTAG_ENTITY
    je .print_ent

    ; Unknown type
    jmp .done

.print_null:
    lea rdi, [str_null]
    mov rsi, str_null_len
    call write_stdout
    jmp .done

.print_int:
    mov rdi, [rbx + VALUE_NUM_OFF]
    call print_integer
    jmp .done

.print_dbl:
    movsd xmm0, [rbx + VALUE_NUM_OFF]
    call print_double
    jmp .done

.print_bool:
    mov rax, [rbx + VALUE_NUM_OFF]
    test rax, rax
    jz .print_false
    lea rdi, [str_true]
    mov rsi, str_true_len
    jmp .print_bool_out
.print_false:
    lea rdi, [str_false]
    mov rsi, str_false_len
.print_bool_out:
    call write_stdout
    jmp .done

.print_str:
    ; Print quoted string
    mov dil, '"'
    call print_char

    mov rax, [rbx + VALUE_PTR_OFF]
    mov rsi, [rax]          ; Length
    add rax, 8              ; Data
    mov rdi, rax
    call write_stdout

    mov dil, '"'
    call print_char
    jmp .done

.print_name:
    ; Print name without quotes
    mov rax, [rbx + VALUE_PTR_OFF]
    mov rsi, [rax]          ; Length
    add rax, 8              ; Data
    mov rdi, rax
    call write_stdout
    jmp .done

.print_arr:
    ; Print array [elem, elem, ...]
    lea rdi, [str_array_open]
    mov rsi, 1
    call write_stdout

    mov rax, [rbx + VALUE_PTR_OFF]  ; Array header
    mov rcx, [rax + ARRAY_LEN_OFF]  ; Length
    test rcx, rcx
    jz .arr_close

    mov rdx, [rax + ARRAY_DATA_OFF] ; Data pointer
    push rcx
    push rdx

.arr_loop:
    pop rdx
    pop rcx

    push rcx
    push rdx

    ; Print element
    mov rdi, rdx
    call print_value

    pop rdx
    pop rcx

    dec rcx
    jz .arr_close

    ; Print comma
    push rcx
    lea rdi, [str_comma]
    mov rsi, 2
    call write_stdout
    pop rcx

    push rcx
    add rdx, VALUE_SIZE
    push rdx
    jmp .arr_loop

.arr_close:
    lea rdi, [str_array_close]
    mov rsi, 1
    call write_stdout
    jmp .done

.print_ent:
    ; Simplified entity printing
    lea rdi, [str_entity_open]
    mov rsi, 1
    call write_stdout

    ; TODO: Print entity attributes

    lea rdi, [str_entity_close]
    mov rsi, 1
    call write_stdout
    jmp .done

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; print_value_ln - Print a Value followed by newline
; Input: rdi = pointer to Value
;-----------------------------------------------------------------------------
global print_value_ln
print_value_ln:
    push rbp
    mov rbp, rsp

    call print_value
    call print_newline

    pop rbp
    ret

;-----------------------------------------------------------------------------
; print_stack - Print entire data stack (for debugging)
;-----------------------------------------------------------------------------
global print_stack
print_stack:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    ; Get stack depth
    mov rax, [state + State.data_stack_base]
    sub rax, r12
    mov rcx, VALUE_SIZE
    xor edx, edx
    div rcx
    mov r12, rax            ; Depth

    ; Print header
    lea rdi, [.header]
    call print_string
    mov rdi, r12
    call print_integer
    call print_newline

    ; Print each value
    xor ebx, ebx            ; Index
.print_loop:
    cmp rbx, r12
    jae .done

    ; Print index
    mov dil, '['
    call print_char
    mov rdi, rbx
    call print_integer
    mov dil, ']'
    call print_char
    mov dil, ' '
    call print_char

    ; Print value
    mov rax, [state + State.data_stack]
    imul rcx, rbx, VALUE_SIZE
    add rax, rcx
    mov rdi, rax
    call print_value_ln

    inc rbx
    jmp .print_loop

.done:
    pop r12
    pop rbx
    pop rbp
    ret

section .rodata
.header: db "Stack (depth: ", 0
