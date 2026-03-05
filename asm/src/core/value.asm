; value.asm - 24-byte tagged union Value type
; DTRules Assembly Implementation
; Matches Go's Value structure exactly

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state
extern heap_alloc
extern value_pool_alloc
extern value_pool_free

section .data
    ; Type names for printing
    type_names:
        dq type_null
        dq type_integer
        dq type_double
        dq type_boolean
        dq type_string
        dq type_name
        dq type_array
        dq type_entity

    type_null:    db "null", 0
    type_integer: db "integer", 0
    type_double:  db "double", 0
    type_boolean: db "boolean", 0
    type_string:  db "string", 0
    type_name:    db "name", 0
    type_array:   db "array", 0
    type_entity:  db "entity", 0

    str_true:     db "true", 0
    str_false:    db "false", 0
    str_null:     db "null", 0

section .text

;-----------------------------------------------------------------------------
; value_alloc - Allocate a new Value from pool
; Output: rax = pointer to uninitialized Value
;-----------------------------------------------------------------------------
global value_alloc
value_alloc:
    jmp value_pool_alloc

;-----------------------------------------------------------------------------
; value_free - Return Value to pool
; Input: rdi = pointer to Value
;-----------------------------------------------------------------------------
global value_free
value_free:
    jmp value_pool_free

;-----------------------------------------------------------------------------
; value_new_null - Create a null Value
; Output: rax = pointer to new null Value
;-----------------------------------------------------------------------------
global value_new_null
value_new_null:
    push rbp
    mov rbp, rsp

    call value_alloc
    test rax, rax
    jz .done

    mov byte [rax + VALUE_TAG_OFF], VTAG_NULL
    mov qword [rax + VALUE_NUM_OFF], 0
    mov qword [rax + VALUE_PTR_OFF], 0

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; value_new_integer - Create an integer Value
; Input: rdi = integer value
; Output: rax = pointer to new integer Value
;-----------------------------------------------------------------------------
global value_new_integer
value_new_integer:
    push rbp
    mov rbp, rsp
    push rbx

    mov rbx, rdi            ; Save integer value

    call value_alloc
    test rax, rax
    jz .done

    mov byte [rax + VALUE_TAG_OFF], VTAG_INTEGER
    mov [rax + VALUE_NUM_OFF], rbx
    mov qword [rax + VALUE_PTR_OFF], 0

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; value_new_double - Create a double Value
; Input: xmm0 = double value
; Output: rax = pointer to new double Value
;-----------------------------------------------------------------------------
global value_new_double
value_new_double:
    push rbp
    mov rbp, rsp
    sub rsp, 16

    ; Save double value
    movsd [rbp - 8], xmm0

    call value_alloc
    test rax, rax
    jz .done

    mov byte [rax + VALUE_TAG_OFF], VTAG_DOUBLE
    mov rcx, [rbp - 8]      ; Load double bits
    mov [rax + VALUE_NUM_OFF], rcx
    mov qword [rax + VALUE_PTR_OFF], 0

.done:
    add rsp, 16
    pop rbp
    ret

;-----------------------------------------------------------------------------
; value_new_boolean - Create a boolean Value
; Input: rdi = boolean (0 or non-zero)
; Output: rax = pointer to new boolean Value
;-----------------------------------------------------------------------------
global value_new_boolean
value_new_boolean:
    push rbp
    mov rbp, rsp
    push rbx

    ; Normalize to 0 or 1
    test rdi, rdi
    setnz bl
    movzx ebx, bl

    call value_alloc
    test rax, rax
    jz .done

    mov byte [rax + VALUE_TAG_OFF], VTAG_BOOLEAN
    mov [rax + VALUE_NUM_OFF], rbx
    mov qword [rax + VALUE_PTR_OFF], 0

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; value_new_string - Create a string Value
; Input: rdi = pointer to string data, rsi = length
; Output: rax = pointer to new string Value
; Notes: String data is copied to heap (length-prefixed format)
;-----------------------------------------------------------------------------
global value_new_string
value_new_string:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov r12, rdi            ; Source pointer
    mov r13, rsi            ; Length

    ; Allocate string storage: 8 bytes length + data + null terminator
    lea rdi, [r13 + 9]
    call heap_alloc
    test rax, rax
    jz .fail

    mov rbx, rax            ; Save string storage pointer

    ; Store length prefix
    mov [rbx], r13

    ; Copy string data
    lea rdi, [rbx + 8]
    mov rsi, r12
    mov rcx, r13
    rep movsb

    ; Null terminate
    mov byte [rdi], 0

    ; Allocate Value
    call value_alloc
    test rax, rax
    jz .fail

    mov byte [rax + VALUE_TAG_OFF], VTAG_STRING
    mov qword [rax + VALUE_NUM_OFF], 0
    mov [rax + VALUE_PTR_OFF], rbx

    jmp .done

.fail:
    xor eax, eax

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; value_new_string_cstr - Create a string Value from C string
; Input: rdi = pointer to null-terminated string
; Output: rax = pointer to new string Value
;-----------------------------------------------------------------------------
global value_new_string_cstr
value_new_string_cstr:
    push rbp
    mov rbp, rsp
    push rbx

    mov rbx, rdi            ; Save string pointer

    ; Calculate length
    xor ecx, ecx
.count:
    cmp byte [rbx + rcx], 0
    je .got_len
    inc ecx
    jmp .count

.got_len:
    mov rdi, rbx
    mov rsi, rcx
    call value_new_string

    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; value_new_name - Create a name Value (interned symbol)
; Input: rdi = pointer to name string, rsi = length
; Output: rax = pointer to new name Value
;-----------------------------------------------------------------------------
global value_new_name
value_new_name:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov r12, rdi            ; Source pointer
    mov r13, rsi            ; Length

    ; TODO: Intern the name in the name table
    ; For now, just store like a string

    ; Allocate name storage
    lea rdi, [r13 + 9]
    call heap_alloc
    test rax, rax
    jz .fail

    mov rbx, rax

    ; Store length prefix
    mov [rbx], r13

    ; Copy name data
    lea rdi, [rbx + 8]
    mov rsi, r12
    mov rcx, r13
    rep movsb
    mov byte [rdi], 0

    ; Allocate Value
    call value_alloc
    test rax, rax
    jz .fail

    mov byte [rax + VALUE_TAG_OFF], VTAG_NAME
    mov qword [rax + VALUE_NUM_OFF], 0
    mov [rax + VALUE_PTR_OFF], rbx

    jmp .done

.fail:
    xor eax, eax

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; value_new_array - Create an empty array Value
; Output: rax = pointer to new array Value
;-----------------------------------------------------------------------------
global value_new_array
value_new_array:
    push rbp
    mov rbp, rsp
    push rbx

    ; Allocate array header (24 bytes: len, cap, data pointer)
    mov rdi, 24
    call heap_alloc
    test rax, rax
    jz .fail

    mov rbx, rax

    ; Initialize header
    mov qword [rbx + ARRAY_LEN_OFF], 0
    mov qword [rbx + ARRAY_CAP_OFF], ARRAY_INITIAL_CAP

    ; Allocate initial data space
    mov rdi, ARRAY_INITIAL_CAP * VALUE_SIZE
    call heap_alloc
    test rax, rax
    jz .fail

    mov [rbx + ARRAY_DATA_OFF], rax

    ; Allocate Value
    call value_alloc
    test rax, rax
    jz .fail

    mov byte [rax + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [rax + VALUE_NUM_OFF], 0
    mov [rax + VALUE_PTR_OFF], rbx

    jmp .done

.fail:
    xor eax, eax

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; value_new_entity - Create a new entity Value
; Input: rdi = pointer to entity type name (can be NULL)
; Output: rax = pointer to new entity Value
;-----------------------------------------------------------------------------
global value_new_entity
value_new_entity:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    mov r12, rdi            ; Save type name

    ; Allocate entity structure (see entity.asm for layout)
    ; For now, minimal structure: type_ptr, attr_count, attr_cap, attrs_ptr
    mov rdi, 32
    call heap_alloc
    test rax, rax
    jz .fail

    mov rbx, rax

    ; Initialize entity
    mov [rbx], r12          ; Type name pointer (or NULL)
    mov qword [rbx + 8], 0  ; Attribute count
    mov qword [rbx + 16], ENTITY_INITIAL_CAP  ; Capacity

    ; Allocate attribute storage
    mov rdi, ENTITY_INITIAL_CAP * 32  ; Each attr: name_ptr(8) + value(24)
    call heap_alloc
    test rax, rax
    jz .fail

    mov [rbx + 24], rax     ; Attributes pointer

    ; Allocate Value
    call value_alloc
    test rax, rax
    jz .fail

    mov byte [rax + VALUE_TAG_OFF], VTAG_ENTITY
    mov qword [rax + VALUE_NUM_OFF], 0
    mov [rax + VALUE_PTR_OFF], rbx

    jmp .done

.fail:
    xor eax, eax

.done:
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; value_copy - Create a copy of a Value
; Input: rdi = pointer to source Value
; Output: rax = pointer to new Value (shallow copy)
;-----------------------------------------------------------------------------
global value_copy
value_copy:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    mov r12, rdi            ; Save source

    call value_alloc
    test rax, rax
    jz .done

    mov rbx, rax            ; Save destination

    ; Copy 24 bytes
    mov rdi, rbx
    mov rsi, r12
    mov ecx, VALUE_SIZE
    rep movsb

    mov rax, rbx

.done:
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; value_get_tag - Get Value type tag
; Input: rdi = pointer to Value
; Output: rax = type tag
;-----------------------------------------------------------------------------
global value_get_tag
value_get_tag:
    movzx eax, byte [rdi + VALUE_TAG_OFF]
    ret

;-----------------------------------------------------------------------------
; value_get_integer - Get integer from Value
; Input: rdi = pointer to Value
; Output: rax = integer value (undefined if not integer type)
;-----------------------------------------------------------------------------
global value_get_integer
value_get_integer:
    mov rax, [rdi + VALUE_NUM_OFF]
    ret

;-----------------------------------------------------------------------------
; value_get_double - Get double from Value
; Input: rdi = pointer to Value
; Output: xmm0 = double value (undefined if not double type)
;-----------------------------------------------------------------------------
global value_get_double
value_get_double:
    movsd xmm0, [rdi + VALUE_NUM_OFF]
    ret

;-----------------------------------------------------------------------------
; value_get_boolean - Get boolean from Value
; Input: rdi = pointer to Value
; Output: rax = 0 or 1 (undefined if not boolean type)
;-----------------------------------------------------------------------------
global value_get_boolean
value_get_boolean:
    mov rax, [rdi + VALUE_NUM_OFF]
    ret

;-----------------------------------------------------------------------------
; value_get_string - Get string pointer and length
; Input: rdi = pointer to Value
; Output: rax = pointer to string data, rdx = length
;-----------------------------------------------------------------------------
global value_get_string
value_get_string:
    mov rax, [rdi + VALUE_PTR_OFF]  ; String storage pointer
    mov rdx, [rax]                   ; Length prefix
    add rax, 8                       ; Data starts after length
    ret

;-----------------------------------------------------------------------------
; value_get_array - Get array pointer
; Input: rdi = pointer to Value
; Output: rax = pointer to array header
;-----------------------------------------------------------------------------
global value_get_array
value_get_array:
    mov rax, [rdi + VALUE_PTR_OFF]
    ret

;-----------------------------------------------------------------------------
; value_get_entity - Get entity pointer
; Input: rdi = pointer to Value
; Output: rax = pointer to entity structure
;-----------------------------------------------------------------------------
global value_get_entity
value_get_entity:
    mov rax, [rdi + VALUE_PTR_OFF]
    ret

;-----------------------------------------------------------------------------
; value_is_null - Check if Value is null
; Input: rdi = pointer to Value
; Output: rax = 1 if null, 0 otherwise
;-----------------------------------------------------------------------------
global value_is_null
value_is_null:
    xor eax, eax
    cmp byte [rdi + VALUE_TAG_OFF], VTAG_NULL
    sete al
    ret

;-----------------------------------------------------------------------------
; value_is_truthy - Check if Value is truthy
; Input: rdi = pointer to Value
; Output: rax = 1 if truthy, 0 if falsy
; Rules:
;   - null is falsy
;   - boolean false is falsy
;   - integer 0 is falsy
;   - empty string is falsy
;   - empty array is falsy
;   - everything else is truthy
;-----------------------------------------------------------------------------
global value_is_truthy
value_is_truthy:
    movzx eax, byte [rdi + VALUE_TAG_OFF]

    cmp al, VTAG_NULL
    je .falsy

    cmp al, VTAG_BOOLEAN
    jne .check_integer
    mov rax, [rdi + VALUE_NUM_OFF]
    ret

.check_integer:
    cmp al, VTAG_INTEGER
    jne .check_double
    mov rax, [rdi + VALUE_NUM_OFF]
    test rax, rax
    setnz al
    movzx eax, al
    ret

.check_double:
    cmp al, VTAG_DOUBLE
    jne .check_string
    movsd xmm0, [rdi + VALUE_NUM_OFF]
    xorpd xmm1, xmm1
    ucomisd xmm0, xmm1
    setne al
    movzx eax, al
    ret

.check_string:
    cmp al, VTAG_STRING
    jne .check_array
    mov rax, [rdi + VALUE_PTR_OFF]
    mov rax, [rax]          ; Length
    test rax, rax
    setnz al
    movzx eax, al
    ret

.check_array:
    cmp al, VTAG_ARRAY
    jne .truthy
    mov rax, [rdi + VALUE_PTR_OFF]
    mov rax, [rax + ARRAY_LEN_OFF]
    test rax, rax
    setnz al
    movzx eax, al
    ret

.truthy:
    mov eax, 1
    ret

.falsy:
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; value_equals - Check if two Values are equal
; Input: rdi = pointer to Value 1, rsi = pointer to Value 2
; Output: rax = 1 if equal, 0 otherwise
;-----------------------------------------------------------------------------
global value_equals
value_equals:
    push rbp
    mov rbp, rsp

    ; Check tags match
    movzx eax, byte [rdi + VALUE_TAG_OFF]
    movzx ecx, byte [rsi + VALUE_TAG_OFF]
    cmp al, cl
    jne .not_equal

    ; Both null
    cmp al, VTAG_NULL
    je .equal

    ; Integer comparison
    cmp al, VTAG_INTEGER
    jne .check_double
    mov rax, [rdi + VALUE_NUM_OFF]
    cmp rax, [rsi + VALUE_NUM_OFF]
    je .equal
    jmp .not_equal

.check_double:
    cmp al, VTAG_DOUBLE
    jne .check_boolean
    movsd xmm0, [rdi + VALUE_NUM_OFF]
    movsd xmm1, [rsi + VALUE_NUM_OFF]
    ucomisd xmm0, xmm1
    jp .not_equal           ; NaN
    jne .not_equal
    jmp .equal

.check_boolean:
    cmp al, VTAG_BOOLEAN
    jne .check_string
    mov rax, [rdi + VALUE_NUM_OFF]
    cmp rax, [rsi + VALUE_NUM_OFF]
    je .equal
    jmp .not_equal

.check_string:
    cmp al, VTAG_STRING
    jne .check_name
    ; Compare strings
    mov rax, [rdi + VALUE_PTR_OFF]
    mov rcx, [rsi + VALUE_PTR_OFF]
    ; Compare lengths
    mov rdx, [rax]
    cmp rdx, [rcx]
    jne .not_equal
    ; Compare data
    add rax, 8
    add rcx, 8
.cmp_loop:
    test rdx, rdx
    jz .equal
    mov r8b, [rax]
    cmp r8b, [rcx]
    jne .not_equal
    inc rax
    inc rcx
    dec rdx
    jmp .cmp_loop

.check_name:
    cmp al, VTAG_NAME
    jne .check_array
    ; Names are interned, so pointer comparison should work
    ; But for safety, compare like strings
    mov rax, [rdi + VALUE_PTR_OFF]
    cmp rax, [rsi + VALUE_PTR_OFF]
    je .equal
    jmp .not_equal

.check_array:
    cmp al, VTAG_ARRAY
    jne .check_entity
    ; For arrays, compare pointers (reference equality)
    mov rax, [rdi + VALUE_PTR_OFF]
    cmp rax, [rsi + VALUE_PTR_OFF]
    je .equal
    jmp .not_equal

.check_entity:
    ; For entities, compare pointers (reference equality)
    mov rax, [rdi + VALUE_PTR_OFF]
    cmp rax, [rsi + VALUE_PTR_OFF]
    je .equal
    jmp .not_equal

.equal:
    mov eax, 1
    pop rbp
    ret

.not_equal:
    xor eax, eax
    pop rbp
    ret

;-----------------------------------------------------------------------------
; value_compare - Compare two Values
; Input: rdi = pointer to Value 1, rsi = pointer to Value 2
; Output: rax = -1 if v1 < v2, 0 if equal, 1 if v1 > v2
;-----------------------------------------------------------------------------
global value_compare
value_compare:
    push rbp
    mov rbp, rsp

    movzx eax, byte [rdi + VALUE_TAG_OFF]
    movzx ecx, byte [rsi + VALUE_TAG_OFF]

    ; Must be same type for comparison
    cmp al, cl
    jne .type_mismatch

    ; Integer comparison
    cmp al, VTAG_INTEGER
    jne .check_double
    mov rax, [rdi + VALUE_NUM_OFF]
    mov rcx, [rsi + VALUE_NUM_OFF]
    cmp rax, rcx
    jl .less
    jg .greater
    jmp .equal

.check_double:
    cmp al, VTAG_DOUBLE
    jne .check_string
    movsd xmm0, [rdi + VALUE_NUM_OFF]
    movsd xmm1, [rsi + VALUE_NUM_OFF]
    ucomisd xmm0, xmm1
    jp .type_mismatch       ; NaN - can't compare
    jb .less
    ja .greater
    jmp .equal

.check_string:
    cmp al, VTAG_STRING
    jne .type_mismatch
    ; Lexicographic string comparison
    mov rax, [rdi + VALUE_PTR_OFF]
    mov rcx, [rsi + VALUE_PTR_OFF]
    mov rdx, [rax]          ; len1
    mov r8, [rcx]           ; len2
    add rax, 8
    add rcx, 8

    ; Compare min(len1, len2) bytes
    push rdx
    push r8
    cmp rdx, r8
    cmova rdx, r8           ; rdx = min

.str_cmp:
    test rdx, rdx
    jz .str_len_cmp
    movzx r9d, byte [rax]
    movzx r10d, byte [rcx]
    cmp r9d, r10d
    jl .less_pop
    jg .greater_pop
    inc rax
    inc rcx
    dec rdx
    jmp .str_cmp

.str_len_cmp:
    pop r8
    pop rdx
    cmp rdx, r8
    jl .less
    jg .greater
    jmp .equal

.less_pop:
    pop r8
    pop rdx
.less:
    mov eax, -1
    pop rbp
    ret

.greater_pop:
    pop r8
    pop rdx
.greater:
    mov eax, 1
    pop rbp
    ret

.equal:
    xor eax, eax
    pop rbp
    ret

.type_mismatch:
    ; Return 0 but set error flag
    mov dword [state + State.error], ERR_TYPE_MISMATCH
    xor eax, eax
    pop rbp
    ret

;-----------------------------------------------------------------------------
; value_type_name - Get type name string
; Input: rdi = pointer to Value
; Output: rax = pointer to null-terminated type name
;-----------------------------------------------------------------------------
global value_type_name
value_type_name:
    movzx eax, byte [rdi + VALUE_TAG_OFF]
    cmp al, VTAG_ENTITY
    ja .invalid
    lea rcx, [type_names]
    mov rax, [rcx + rax * 8]
    ret
.invalid:
    lea rax, [type_null]    ; Return "null" for unknown
    ret

;-----------------------------------------------------------------------------
; value_print - Print a Value to stdout
; Input: rdi = pointer to Value
;-----------------------------------------------------------------------------
extern print_string
extern print_integer
extern print_double
extern string_data

global value_print
value_print:
    push rbp
    mov rbp, rsp
    push rbx

    mov rbx, rdi

    movzx eax, byte [rbx + VALUE_TAG_OFF]

    cmp al, VTAG_NULL
    je .print_null
    cmp al, VTAG_INTEGER
    je .print_integer
    cmp al, VTAG_DOUBLE
    je .print_double
    cmp al, VTAG_BOOLEAN
    je .print_boolean
    cmp al, VTAG_STRING
    je .print_string
    cmp al, VTAG_NAME
    je .print_name
    cmp al, VTAG_ARRAY
    je .print_array
    cmp al, VTAG_ENTITY
    je .print_entity
    cmp al, VTAG_MARK
    je .print_mark
    jmp .print_unknown

.print_null:
    lea rdi, [str_null]
    call print_string
    jmp .done

.print_integer:
    mov rdi, [rbx + VALUE_NUM_OFF]
    call print_integer
    jmp .done

.print_double:
    movsd xmm0, [rbx + VALUE_NUM_OFF]
    call print_double
    jmp .done

.print_boolean:
    mov rax, [rbx + VALUE_NUM_OFF]
    test rax, rax
    jz .print_false
    lea rdi, [str_true]
    jmp .print_bool_str
.print_false:
    lea rdi, [str_false]
.print_bool_str:
    call print_string
    jmp .done

.print_string:
    mov rdi, [rbx + VALUE_PTR_OFF]
    call string_data
    mov rdi, rax
    call print_string
    jmp .done

.print_name:
    ; Names stored same as strings
    mov rdi, [rbx + VALUE_PTR_OFF]
    test rdi, rdi
    jz .print_null
    call string_data
    mov rdi, rax
    call print_string
    jmp .done

.print_array:
    lea rdi, [str_array_fmt]
    call print_string
    jmp .done

.print_entity:
    lea rdi, [str_entity_fmt]
    call print_string
    jmp .done

.print_mark:
    lea rdi, [str_mark_fmt]
    call print_string
    jmp .done

.print_unknown:
    lea rdi, [str_unknown_fmt]
    call print_string

.done:
    pop rbx
    pop rbp
    ret

section .data
    str_array_fmt: db "[array]", 0
    str_entity_fmt: db "[entity]", 0
    str_mark_fmt: db "[mark]", 0
    str_unknown_fmt: db "[?]", 0

section .text

;-----------------------------------------------------------------------------
; value_clone - Deep copy a Value
; Input: rdi = pointer to source Value
; Output: rax = pointer to new Value (clone), or 0 on error
;-----------------------------------------------------------------------------
extern array_copy
extern string_copy
extern entity_copy

global value_clone
value_clone:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    mov rbx, rdi            ; Source value

    ; Allocate new value
    call value_alloc
    test rax, rax
    jz .error
    mov r12, rax            ; New value

    ; Copy basic fields
    movzx eax, byte [rbx + VALUE_TAG_OFF]
    mov byte [r12 + VALUE_TAG_OFF], al
    mov rcx, [rbx + VALUE_NUM_OFF]
    mov [r12 + VALUE_NUM_OFF], rcx

    ; Handle pointer types specially
    cmp al, VTAG_STRING
    je .clone_string
    cmp al, VTAG_NAME
    je .clone_string        ; Names are like strings
    cmp al, VTAG_ARRAY
    je .clone_array
    cmp al, VTAG_ENTITY
    je .clone_entity

    ; Simple type, just copy pointer field
    mov rcx, [rbx + VALUE_PTR_OFF]
    mov [r12 + VALUE_PTR_OFF], rcx
    mov rax, r12
    jmp .done

.clone_string:
    mov rdi, [rbx + VALUE_PTR_OFF]
    test rdi, rdi
    jz .null_ptr
    call string_copy
    test rax, rax
    jz .error
    mov [r12 + VALUE_PTR_OFF], rax
    mov rax, r12
    jmp .done

.clone_array:
    mov rdi, [rbx + VALUE_PTR_OFF]
    test rdi, rdi
    jz .null_ptr
    call array_copy
    test rax, rax
    jz .error
    mov [r12 + VALUE_PTR_OFF], rax
    mov rax, r12
    jmp .done

.clone_entity:
    mov rdi, [rbx + VALUE_PTR_OFF]
    test rdi, rdi
    jz .null_ptr
    call entity_copy
    test rax, rax
    jz .error
    mov [r12 + VALUE_PTR_OFF], rax
    mov rax, r12
    jmp .done

.null_ptr:
    mov qword [r12 + VALUE_PTR_OFF], 0
    mov rax, r12
    jmp .done

.error:
    xor eax, eax

.done:
    pop r12
    pop rbx
    pop rbp
    ret
