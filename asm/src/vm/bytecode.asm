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
extern print_string
extern print_hex
extern print_stack

; String operations
extern string_concat
extern string_length
extern string_substring
extern string_equals
extern string_index_of
extern string_last_index_of
extern string_starts_with
extern string_ends_with
extern string_contains
extern string_replace
extern string_replace_all
extern string_split
extern string_join
extern string_trim
extern string_trim_left
extern string_trim_right
extern string_to_upper
extern string_to_lower

; Array operations
extern array_alloc
extern array_get
extern array_set
extern array_length
extern array_push
extern array_pop
extern array_concat
extern array_slice
extern array_copy
extern array_clear
extern array_remove
extern array_insert
extern array_reverse
extern array_sort
extern array_contains
extern array_index_of

; Entity stack operations
extern stack_entity_push
extern stack_entity_pop
extern stack_entity_peek
extern stack_entity_peek_n
extern stack_entity_depth
extern stack_entity_lookup
extern stack_entity_define

; Entity type operations
extern entity_alloc
extern entity_get_attr
extern entity_set_attr
extern entity_find_attr

; Memory operations
extern heap_alloc

; Value creation
extern value_new_string
extern value_new_array
extern value_new_integer
extern value_new_name
extern value_new_double
extern value_clone
extern value_print

; Additional array operations
extern array_remove_element
extern array_copy_elements
extern array_sort_entities
extern array_merge
extern array_randomize
extern array_intersection
extern array_intersects
extern array_deep_copy
extern array_find_match

; Additional string operations
extern string_tokenize
extern string_compare
extern string_equals_ignore_case
extern string_regex_match

; Date operations
extern date_create
extern date_get_year
extern date_get_month
extern date_get_day
extern date_add_days
extern date_add_months
extern date_add_years
extern date_days_between
extern date_months_between
extern date_years_between
extern date_compare
extern date_first_of_month
extern date_first_of_year
extern date_end_of_month
extern date_days_in_year
extern date_days_in_month
extern date_parse
extern time_get_timestamp

; Additional entity operations
extern entity_get_type_name
extern entity_get_id
extern entity_get_all_of_type
extern stack_entity_has_type
extern stack_entity_find
extern stack_entity_fetch

; Table operations (decision tables)
extern table_execute_by_name

; Hash table (map) operations
extern hashtable_alloc
extern hashtable_get
extern hashtable_put
extern hashtable_contains
extern hashtable_remove
extern hashtable_size
extern hashtable_keys
extern hashtable_values

; Math operations
extern math_round_to

; Return stack operations
extern stack_return_push
extern stack_return_pop
extern stack_return_peek

; Stack utilities
extern stack_data_count_to_mark
extern stack_data_print

;-----------------------------------------------------------------------------
; Export all opcode handlers for testing
;-----------------------------------------------------------------------------
; Stack/Push ops
global op_nop, op_push_null, op_push_true, op_push_false
global op_push_int, op_push_double, op_push_string, op_push_name
global op_dup, op_swap, op_pop, op_rot, op_over, op_pick, op_roll, op_clear
global op_push_constant, op_mark, op_push_zero, op_push_one
global op_inc, op_dec, op_depth, op_operator, op_constant, op_name

; Entity ops
global op_entitypush, op_entitypop, op_def, op_lookup, op_newentity

; Table ops
global op_newtable, op_tableget, op_tableput

; Arithmetic ops
global op_add, op_sub, op_mul, op_div, op_mod
global op_neg, op_abs, op_min, op_max
global op_floor, op_ceil, op_round, op_truncate, op_pow

; Comparison ops
global op_eq, op_ne, op_lt, op_le, op_gt, op_ge

; Boolean ops
global op_and, op_or, op_not, op_xor

; Control flow ops
global op_if, op_ifelse, op_while, op_for, op_forall
global op_break, op_continue, op_return, op_exec
global op_call, op_jump, op_jumpif, op_jumpifnot, op_halt

; String ops
global op_str_concat, op_str_length, op_str_substring
global op_str_indexof, op_str_lastindexof
global op_str_startswith, op_str_endswith, op_str_contains
global op_str_replace, op_str_replaceall
global op_str_split, op_str_join
global op_str_trim, op_str_trimleft, op_str_trimright
global op_str_toupper, op_str_tolower
global op_str_tostring, op_str_tointeger, op_str_todouble

; Array ops
global op_array_new, op_array_get, op_array_set, op_array_len
global op_array_push, op_array_pop, op_array_concat, op_array_slice
global op_array_copy, op_array_clear, op_array_remove, op_array_insert
global op_array_reverse, op_array_sort, op_array_contains, op_array_indexof

; I/O ops
global op_print, op_trace, op_debug

;-----------------------------------------------------------------------------
; Opcode handler table (matches Go's bytecode.go numbering)
;-----------------------------------------------------------------------------
section .data
    align 8
    global opcode_table
    opcode_table:
        ; 0-10: Stack operations (Go compatible)
        dq op_nop           ; 0 - OP_NOP
        dq op_push_constant ; 1 - OP_PUSH (constant from pool)
        dq op_push_int      ; 2 - OP_PUSH_INT
        dq op_pop           ; 3 - OP_POP
        dq op_dup           ; 4 - OP_DUP
        dq op_swap          ; 5 - OP_SWAP
        dq op_rot           ; 6 - OP_ROT
        dq op_roll          ; 7 - OP_ROLL
        dq op_pick          ; 8 - OP_INDEX (pick)
        dq op_clear         ; 9 - OP_CLEAR
        dq op_mark          ; 10 - OP_MARK

        ; 11-19: Reserved
        times 9 dq op_invalid

        ; 20-28: Arithmetic (Go compatible)
        dq op_add           ; 20 - OP_ADD
        dq op_sub           ; 21 - OP_SUB
        dq op_mul           ; 22 - OP_MUL
        dq op_div           ; 23 - OP_DIV
        dq op_mod           ; 24 - OP_MOD
        dq op_neg           ; 25 - OP_NEG
        dq op_abs           ; 26 - OP_ABS
        dq op_inc           ; 27 - OP_INC
        dq op_dec           ; 28 - OP_DEC

        ; 29: Reserved
        dq op_invalid

        ; 30-35: Comparison (Go compatible)
        dq op_eq            ; 30 - OP_EQ
        dq op_ne            ; 31 - OP_NE
        dq op_lt            ; 32 - OP_LT
        dq op_le            ; 33 - OP_LE
        dq op_gt            ; 34 - OP_GT
        dq op_ge            ; 35 - OP_GE

        ; 36-39: Reserved
        times 4 dq op_invalid

        ; 40-43: Logical (Go compatible)
        dq op_and           ; 40 - OP_AND
        dq op_or            ; 41 - OP_OR
        dq op_not           ; 42 - OP_NOT
        dq op_xor           ; 43 - OP_XOR

        ; 44-49: Reserved
        times 6 dq op_invalid

        ; 50-59: Control flow (Go compatible)
        dq op_exec          ; 50 - OP_EXEC
        dq op_if            ; 51 - OP_IF
        dq op_ifelse        ; 52 - OP_IFELSE
        dq op_while         ; 53 - OP_WHILE
        dq op_for           ; 54 - OP_FOR
        dq op_forall        ; 55 - OP_FORALL
        dq op_return        ; 56 - OP_RETURN
        dq op_jump          ; 57 - OP_JUMP
        dq op_jumpif        ; 58 - OP_JUMPIF
        dq op_call          ; 59 - OP_CALL

        ; 60-64: Entity operations (Go compatible)
        dq op_entitypush    ; 60 - OP_ENTITYPUSH
        dq op_entitypop     ; 61 - OP_ENTITYPOP
        dq op_def           ; 62 - OP_DEF
        dq op_lookup        ; 63 - OP_LOOKUP
        dq op_newentity     ; 64 - OP_NEWENTITY

        ; 65-69: Reserved
        times 5 dq op_invalid

        ; 70-74: Array operations (Go compatible)
        dq op_array_new     ; 70 - OP_NEWARRAY
        dq op_array_push    ; 71 - OP_ADDTO
        dq op_array_len     ; 72 - OP_LENGTH
        dq op_array_get     ; 73 - OP_GET
        dq op_array_set     ; 74 - OP_PUT

        ; 75-79: Extended array ops
        dq op_array_pop     ; 75 - OP_ARRAYPOP
        dq op_array_concat  ; 76 - OP_ARRAYCONCAT
        dq op_array_slice   ; 77 - OP_ARRAYSLICE
        dq op_array_copy    ; 78 - OP_ARRAYCOPY
        dq op_array_clear   ; 79 - OP_ARRAYCLEAR

        ; 80-82: Table operations (Go compatible)
        dq op_newtable      ; 80 - OP_NEWTABLE
        dq op_tableget      ; 81 - OP_TABLEGET
        dq op_tableput      ; 82 - OP_TABLEPUT

        ; 83-84: Reserved
        times 2 dq op_invalid

        ; 85-89: Extended array ops
        dq op_array_remove  ; 85 - OP_ARRAYREMOVE
        dq op_array_insert  ; 86 - OP_ARRAYINSERT
        dq op_array_reverse ; 87 - OP_ARRAYREVERSE
        dq op_array_sort    ; 88 - OP_ARRAYSORT
        dq op_array_contains ; 89 - OP_ARRAYCONTAINS

        ; 90-91: String operations (Go compatible)
        dq op_str_concat    ; 90 - OP_CONCAT
        dq op_str_substring ; 91 - OP_SUBSTRING

        ; 92-99: Reserved
        times 8 dq op_invalid

        ; 100-104: Constant shortcuts (Go compatible)
        dq op_push_true     ; 100 - OP_PUSH_TRUE
        dq op_push_false    ; 101 - OP_PUSH_FALSE
        dq op_push_null     ; 102 - OP_PUSH_NULL
        dq op_push_zero     ; 103 - OP_PUSH_ZERO
        dq op_push_one      ; 104 - OP_PUSH_ONE

        ; 105-109: Reserved
        times 5 dq op_invalid

        ; 110: Extended control
        dq op_jumpifnot     ; 110 - OP_JUMPIFNOT

        ; 111-199: Reserved
        times 89 dq op_invalid

        ; 200-202: Extended opcodes (Go compatible)
        dq op_operator      ; 200 - OP_OPERATOR
        dq op_constant      ; 201 - OP_CONSTANT
        dq op_name          ; 202 - OP_NAME

        ; 203-209: Reserved
        times 7 dq op_invalid

        ; 210-216: Extended arithmetic
        dq op_min           ; 210 - OP_MIN
        dq op_max           ; 211 - OP_MAX
        dq op_floor         ; 212 - OP_FLOOR
        dq op_ceil          ; 213 - OP_CEIL
        dq op_round         ; 214 - OP_ROUND
        dq op_truncate      ; 215 - OP_TRUNCATE
        dq op_pow           ; 216 - OP_POW

        ; 217-219: Reserved
        times 3 dq op_invalid

        ; 220-222: Extended stack
        dq op_over          ; 220 - OP_OVER
        dq op_pick          ; 221 - OP_PICK (alias)
        dq op_depth         ; 222 - OP_DEPTH

        ; 223-229: Reserved
        times 7 dq op_invalid

        ; 230-248: Extended string operations
        dq op_str_indexof      ; 230 - OP_INDEXOF
        dq op_str_lastindexof  ; 231 - OP_LASTINDEXOF
        dq op_str_startswith   ; 232 - OP_STARTSWITH
        dq op_str_endswith     ; 233 - OP_ENDSWITH
        dq op_str_contains     ; 234 - OP_CONTAINS
        dq op_str_replace      ; 235 - OP_REPLACE
        dq op_str_replaceall   ; 236 - OP_REPLACEALL
        dq op_str_split        ; 237 - OP_SPLIT
        dq op_str_join         ; 238 - OP_JOIN
        dq op_str_trim         ; 239 - OP_TRIM
        dq op_str_trimleft     ; 240 - OP_TRIMLEFT
        dq op_str_trimright    ; 241 - OP_TRIMRIGHT
        dq op_str_toupper      ; 242 - OP_TOUPPER
        dq op_str_tolower      ; 243 - OP_TOLOWER
        dq op_str_tostring     ; 244 - OP_TOSTRING
        dq op_str_tointeger    ; 245 - OP_TOINTEGER
        dq op_str_todouble     ; 246 - OP_TODOUBLE
        dq op_invalid          ; 247 - OP_MATCH (reserved)
        dq op_invalid          ; 248 - OP_FORMAT (reserved)

        ; 249: Reserved
        dq op_invalid

        ; 250-252: Debug/utility
        dq op_print         ; 250 - OP_PRINT
        dq op_trace         ; 251 - OP_TRACE
        dq op_debug         ; 252 - OP_DEBUG

        ; 253-254: Reserved
        times 2 dq op_invalid

        ; 255: Halt
        dq op_halt          ; 255 - OP_HALT

    ; String constants for tostring conversion
    str_true_const:  db "true", 0
    str_false_const: db "false", 0
    str_null_const:  db "null", 0
    str_object_const: db "[object]", 0

    ; Debug/trace constants
    trace_prefix:    db "TRACE: @", 0
    debug_header:    db "=== DEBUG ===", 0
    debug_stack_label: db "Stack: ", 0

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
    ; NOTE: Do NOT save/restore r12, r13, r14!
    ; These are the VM stack pointers and must persist across execution.
    ; r12 = data stack pointer
    ; r13 = entity stack pointer
    ; r14 = control stack pointer

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

    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; vm_execute_value - Execute a bytecode Value
; Input: rdi = pointer to Value containing bytecode
; Output: rax = 0 on success
;
; This saves the current bytecode state, executes the bytecode in the Value,
; then restores the previous state.
;-----------------------------------------------------------------------------
global vm_execute_value
vm_execute_value:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    ; Check if value is bytecode type
    movzx eax, byte [rdi + VALUE_TAG_OFF]
    cmp al, VTAG_BYTECODE
    jne .not_bytecode

    ; Get bytecode structure from value
    ; Bytecode struct: [length:8][data:...]
    mov r12, [rdi + VALUE_PTR_OFF]
    test r12, r12
    jz .done_exec

    ; Save current bytecode state
    mov r13, rbx                    ; Save current IP
    push qword [state + State.bytecode]
    push qword [state + State.bytecode_end]

    ; Set up new bytecode
    mov rax, [r12]                  ; Length
    lea rcx, [r12 + 8]              ; Data pointer
    mov [state + State.bytecode], rcx
    add rax, rcx
    mov [state + State.bytecode_end], rax
    mov rbx, rcx                    ; Set IP

    ; Execute
    call vm_execute

    ; Restore bytecode state
    pop qword [state + State.bytecode_end]
    pop qword [state + State.bytecode]
    mov rbx, r13                    ; Restore IP

    xor eax, eax
    jmp .done_exec

.not_bytecode:
    ; If not bytecode, just ignore (or could push to stack and exec)
    xor eax, eax

.done_exec:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; Bytecode reading helpers
;-----------------------------------------------------------------------------

; Read unsigned varint from bytecode
; Input: rbx = pointer to bytecode
; Output: rax = value, rbx advanced past varint
; Uses: rax, rcx, rdx (rbx modified)
read_varint:
    xor rax, rax            ; result = 0
    xor rcx, rcx            ; shift = 0

.loop:
    movzx rdx, byte [rbx]   ; read byte
    inc rbx                 ; advance pointer

    mov r8, rdx             ; save byte
    and r8, 0x7F            ; mask off continuation bit
    shl r8, cl              ; shift by current amount
    or rax, r8              ; add to result

    test dl, 0x80           ; continuation bit set?
    jz .done                ; no, we're done

    add cl, 7               ; shift += 7
    cmp cl, 64              ; overflow check
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
    call read_varint
    mov rdi, rax
    call stack_data_push_integer
    ret

op_push_double:
    call read_qword
    movq xmm0, rax
    call stack_data_push_double
    ret

op_push_string:
    ; Read length (varint) then bytes from bytecode
    ; Bytecode format: [length:varint][data:bytes]
    push rbp
    mov rbp, rsp
    push r12

    ; Read string length
    call read_varint
    mov r12, rax            ; Save length

    ; Create string value from bytecode data
    ; rbx points to string data in bytecode
    mov rdi, rbx            ; Source data pointer
    mov rsi, r12            ; Length
    call value_new_string
    test rax, rax
    jz .push_str_error

    ; Advance bytecode pointer past string data
    add rbx, r12

    ; Push value onto stack
    mov rdi, rax
    call stack_data_push

.push_str_done:
    pop r12
    pop rbp
    ret

.push_str_error:
    mov dword [state + State.error], ERR_OUT_OF_MEMORY
    pop r12
    pop rbp
    ret

op_push_name:
    ; Read length (varint) then bytes from bytecode
    ; Names are interned symbols
    push rbp
    mov rbp, rsp
    push r12

    ; Read name length
    call read_varint
    mov r12, rax            ; Save length

    ; Create name value from bytecode data
    mov rdi, rbx            ; Source data pointer
    mov rsi, r12            ; Length
    call value_new_name
    test rax, rax
    jz .push_name_error

    ; Advance bytecode pointer past name data
    add rbx, r12

    ; Push value onto stack
    mov rdi, rax
    call stack_data_push

.push_name_done:
    pop r12
    pop rbp
    ret

.push_name_error:
    mov dword [state + State.error], ERR_OUT_OF_MEMORY
    pop r12
    pop rbp
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
    ; Mixed types: convert integer to double and add
    ; esi = first operand tag, edi = second operand tag
    ; rcx = first operand (top of stack), rdx = second operand
    cmp esi, VTAG_INTEGER
    je .first_is_int
    ; First is double, second must be integer (we already checked first is double above)
    cmp edi, VTAG_INTEGER
    jne .type_error
    ; First (rcx) is double, second (rdx) is integer - convert rdx to double
    movsd xmm0, [rcx + VALUE_NUM_OFF]       ; Load first double
    cvtsi2sd xmm1, qword [rdx + VALUE_NUM_OFF]  ; Convert second int to double
    addsd xmm0, xmm1
    call stack_data_push_double
    ret

.first_is_int:
    ; First (rcx) is integer, second (rdx) must be double (we checked first is int, second is not int)
    cmp edi, VTAG_DOUBLE
    jne .type_error
    cvtsi2sd xmm0, qword [rcx + VALUE_NUM_OFF]  ; Convert first int to double
    addsd xmm0, [rdx + VALUE_NUM_OFF]           ; Add second double
    call stack_data_push_double
    ret

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    ret

op_sub:
    call stack_data_pop
    test rax, rax
    jz .error
    mov rcx, rax              ; rcx = first operand (subtrahend, what we subtract)

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax              ; rdx = second operand (minuend, what we subtract from)

    movzx esi, byte [rcx + VALUE_TAG_OFF]
    movzx edi, byte [rdx + VALUE_TAG_OFF]

    cmp esi, VTAG_INTEGER
    jne .check_double
    cmp edi, VTAG_INTEGER
    jne .mixed

    mov rdi, [rdx + VALUE_NUM_OFF]
    sub rdi, [rcx + VALUE_NUM_OFF]
    call stack_data_push_integer
    ret

.check_double:
    cmp esi, VTAG_DOUBLE
    jne .type_error
    cmp edi, VTAG_DOUBLE
    jne .mixed

    movsd xmm0, [rdx + VALUE_NUM_OFF]
    subsd xmm0, [rcx + VALUE_NUM_OFF]
    call stack_data_push_double
    ret

.mixed:
    ; Mixed types: convert integer to double
    cmp esi, VTAG_INTEGER
    je .first_is_int
    ; First is double, second is integer
    cmp edi, VTAG_INTEGER
    jne .type_error
    movsd xmm0, [rcx + VALUE_NUM_OFF]           ; Load first (subtrahend) as double
    cvtsi2sd xmm1, qword [rdx + VALUE_NUM_OFF]  ; Convert second (minuend) to double
    subsd xmm1, xmm0                            ; minuend - subtrahend
    movsd xmm0, xmm1
    call stack_data_push_double
    ret

.first_is_int:
    ; First is integer, second is double
    cmp edi, VTAG_DOUBLE
    jne .type_error
    cvtsi2sd xmm0, qword [rcx + VALUE_NUM_OFF]  ; Convert first (subtrahend) to double
    movsd xmm1, [rdx + VALUE_NUM_OFF]           ; Load second (minuend) as double
    subsd xmm1, xmm0                            ; minuend - subtrahend
    movsd xmm0, xmm1
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
    jne .mixed

    mov rax, [rdx + VALUE_NUM_OFF]
    imul rax, [rcx + VALUE_NUM_OFF]
    mov rdi, rax
    call stack_data_push_integer
    ret

.check_double:
    cmp esi, VTAG_DOUBLE
    jne .type_error
    cmp edi, VTAG_DOUBLE
    jne .mixed

    movsd xmm0, [rdx + VALUE_NUM_OFF]
    mulsd xmm0, [rcx + VALUE_NUM_OFF]
    call stack_data_push_double
    ret

.mixed:
    ; Mixed types: convert integer to double
    cmp esi, VTAG_INTEGER
    je .first_is_int
    ; First is double, second is integer
    cmp edi, VTAG_INTEGER
    jne .type_error
    movsd xmm0, [rcx + VALUE_NUM_OFF]           ; Load first as double
    cvtsi2sd xmm1, qword [rdx + VALUE_NUM_OFF]  ; Convert second to double
    mulsd xmm0, xmm1
    call stack_data_push_double
    ret

.first_is_int:
    ; First is integer, second is double
    cmp edi, VTAG_DOUBLE
    jne .type_error
    cvtsi2sd xmm0, qword [rcx + VALUE_NUM_OFF]  ; Convert first to double
    mulsd xmm0, [rdx + VALUE_NUM_OFF]           ; Multiply with second
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
    mov rcx, rax              ; rcx = divisor

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax              ; rdx = dividend

    movzx esi, byte [rcx + VALUE_TAG_OFF]
    movzx edi, byte [rdx + VALUE_TAG_OFF]

    cmp esi, VTAG_INTEGER
    jne .check_double
    cmp edi, VTAG_INTEGER
    jne .mixed

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
    jne .mixed

    movsd xmm0, [rdx + VALUE_NUM_OFF]
    divsd xmm0, [rcx + VALUE_NUM_OFF]
    call stack_data_push_double
    ret

.mixed:
    ; Mixed types: convert integer to double
    cmp esi, VTAG_INTEGER
    je .first_is_int
    ; First (divisor) is double, second (dividend) is integer
    cmp edi, VTAG_INTEGER
    jne .type_error
    movsd xmm0, [rcx + VALUE_NUM_OFF]           ; Load divisor as double
    cvtsi2sd xmm1, qword [rdx + VALUE_NUM_OFF]  ; Convert dividend to double
    divsd xmm1, xmm0                            ; dividend / divisor
    movsd xmm0, xmm1
    call stack_data_push_double
    ret

.first_is_int:
    ; First (divisor) is integer, second (dividend) is double
    cmp edi, VTAG_DOUBLE
    jne .type_error
    cvtsi2sd xmm0, qword [rcx + VALUE_NUM_OFF]  ; Convert divisor to double
    movsd xmm1, [rdx + VALUE_NUM_OFF]           ; Load dividend as double
    divsd xmm1, xmm0                            ; dividend / divisor
    movsd xmm0, xmm1
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

    movzx esi, byte [rcx + VALUE_TAG_OFF]
    movzx edi, byte [rdx + VALUE_TAG_OFF]

    cmp esi, VTAG_INTEGER
    jne .check_double
    cmp edi, VTAG_INTEGER
    jne .mixed

    ; Both integers
    mov rdi, [rcx + VALUE_NUM_OFF]
    mov rax, [rdx + VALUE_NUM_OFF]
    cmp rax, rdi
    cmovl rdi, rax
    call stack_data_push_integer
    ret

.check_double:
    cmp esi, VTAG_DOUBLE
    jne .type_error
    cmp edi, VTAG_DOUBLE
    jne .mixed

    ; Both doubles - use minsd
    movsd xmm0, [rcx + VALUE_NUM_OFF]
    minsd xmm0, [rdx + VALUE_NUM_OFF]
    call stack_data_push_double
    ret

.mixed:
    ; Mixed types: convert integer to double
    cmp esi, VTAG_INTEGER
    je .first_is_int
    cmp esi, VTAG_DOUBLE
    jne .type_error
    ; First (rcx) is double, second (rdx) is integer
    cmp edi, VTAG_INTEGER
    jne .type_error
    movsd xmm0, [rcx + VALUE_NUM_OFF]
    cvtsi2sd xmm1, qword [rdx + VALUE_NUM_OFF]
    minsd xmm0, xmm1
    call stack_data_push_double
    ret

.first_is_int:
    ; First (rcx) is integer, second (rdx) is double
    cmp edi, VTAG_DOUBLE
    jne .type_error
    cvtsi2sd xmm0, qword [rcx + VALUE_NUM_OFF]
    minsd xmm0, [rdx + VALUE_NUM_OFF]
    call stack_data_push_double
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

    movzx esi, byte [rcx + VALUE_TAG_OFF]
    movzx edi, byte [rdx + VALUE_TAG_OFF]

    cmp esi, VTAG_INTEGER
    jne .check_double
    cmp edi, VTAG_INTEGER
    jne .mixed

    ; Both integers
    mov rdi, [rcx + VALUE_NUM_OFF]
    mov rax, [rdx + VALUE_NUM_OFF]
    cmp rax, rdi
    cmovg rdi, rax
    call stack_data_push_integer
    ret

.check_double:
    cmp esi, VTAG_DOUBLE
    jne .type_error
    cmp edi, VTAG_DOUBLE
    jne .mixed

    ; Both doubles - use maxsd
    movsd xmm0, [rcx + VALUE_NUM_OFF]
    maxsd xmm0, [rdx + VALUE_NUM_OFF]
    call stack_data_push_double
    ret

.mixed:
    ; Mixed types: convert integer to double
    cmp esi, VTAG_INTEGER
    je .first_is_int
    cmp esi, VTAG_DOUBLE
    jne .type_error
    ; First (rcx) is double, second (rdx) is integer
    cmp edi, VTAG_INTEGER
    jne .type_error
    movsd xmm0, [rcx + VALUE_NUM_OFF]
    cvtsi2sd xmm1, qword [rdx + VALUE_NUM_OFF]
    maxsd xmm0, xmm1
    call stack_data_push_double
    ret

.first_is_int:
    ; First (rcx) is integer, second (rdx) is double
    cmp edi, VTAG_DOUBLE
    jne .type_error
    cvtsi2sd xmm0, qword [rcx + VALUE_NUM_OFF]
    maxsd xmm0, [rdx + VALUE_NUM_OFF]
    call stack_data_push_double
    ret

.type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.error:
    ret

op_floor:
    ; Floor: round toward negative infinity
    call stack_data_peek
    test rax, rax
    jz .floor_error

    cmp byte [rax + VALUE_TAG_OFF], VTAG_DOUBLE
    jne .floor_int_check

    ; Double floor using SSE4.1 roundsd
    movsd xmm0, [rax + VALUE_NUM_OFF]
    roundsd xmm0, xmm0, 1       ; 1 = round toward -infinity
    movsd [rax + VALUE_NUM_OFF], xmm0
    ret

.floor_int_check:
    cmp byte [rax + VALUE_TAG_OFF], VTAG_INTEGER
    jne .floor_type_error
    ; Integer is already floored
    ret

.floor_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.floor_error:
    ret

op_ceil:
    ; Ceiling: round toward positive infinity
    call stack_data_peek
    test rax, rax
    jz .ceil_error

    cmp byte [rax + VALUE_TAG_OFF], VTAG_DOUBLE
    jne .ceil_int_check

    movsd xmm0, [rax + VALUE_NUM_OFF]
    roundsd xmm0, xmm0, 2       ; 2 = round toward +infinity
    movsd [rax + VALUE_NUM_OFF], xmm0
    ret

.ceil_int_check:
    cmp byte [rax + VALUE_TAG_OFF], VTAG_INTEGER
    jne .ceil_type_error
    ret

.ceil_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.ceil_error:
    ret

op_round:
    ; Round: round to nearest, ties to even
    call stack_data_peek
    test rax, rax
    jz .round_error

    cmp byte [rax + VALUE_TAG_OFF], VTAG_DOUBLE
    jne .round_int_check

    movsd xmm0, [rax + VALUE_NUM_OFF]
    roundsd xmm0, xmm0, 0       ; 0 = round to nearest
    movsd [rax + VALUE_NUM_OFF], xmm0
    ret

.round_int_check:
    cmp byte [rax + VALUE_TAG_OFF], VTAG_INTEGER
    jne .round_type_error
    ret

.round_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.round_error:
    ret

op_truncate:
    ; Truncate: round toward zero
    call stack_data_peek
    test rax, rax
    jz .trunc_error

    cmp byte [rax + VALUE_TAG_OFF], VTAG_DOUBLE
    jne .trunc_int_check

    movsd xmm0, [rax + VALUE_NUM_OFF]
    roundsd xmm0, xmm0, 3       ; 3 = round toward zero
    movsd [rax + VALUE_NUM_OFF], xmm0
    ret

.trunc_int_check:
    cmp byte [rax + VALUE_TAG_OFF], VTAG_INTEGER
    jne .trunc_type_error
    ret

.trunc_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.trunc_error:
    ret

op_pow:
    ; Power: base^exponent
    ; Stack: base exponent -- result
    ; Uses x87 FPU: x^y = 2^(y * log2(x))
    push rbp
    mov rbp, rsp
    sub rsp, 32             ; Space for FPU operations and saved values
    push rbx                ; Use rbx instead of r12 (r12 is stack pointer!)

    ; Pop exponent
    call stack_data_pop
    test rax, rax
    jz .pow_error
    mov [rbp - 16], rax     ; Save exponent value pointer

    ; Pop base
    call stack_data_pop
    test rax, rax
    jz .pow_error
    mov rbx, rax            ; Base value pointer

    mov rax, [rbp - 16]     ; Get exponent pointer back

    ; Get numeric values (convert int to double if needed)
    cmp byte [rbx + VALUE_TAG_OFF], VTAG_DOUBLE
    je .pow_base_double
    cmp byte [rbx + VALUE_TAG_OFF], VTAG_INTEGER
    jne .pow_type_error
    ; Convert base integer to double
    fild qword [rbx + VALUE_NUM_OFF]
    jmp .pow_check_exp

.pow_base_double:
    fld qword [rbx + VALUE_NUM_OFF]

.pow_check_exp:
    ; Now ST(0) = base, rax = exponent pointer
    cmp byte [rax + VALUE_TAG_OFF], VTAG_DOUBLE
    je .pow_exp_double
    cmp byte [rax + VALUE_TAG_OFF], VTAG_INTEGER
    jne .pow_type_error_pop
    ; Convert exponent integer to double
    fild qword [rax + VALUE_NUM_OFF]
    jmp .pow_compute

.pow_exp_double:
    fld qword [rax + VALUE_NUM_OFF]

.pow_compute:
    ; ST(0) = exponent, ST(1) = base
    ; Compute base^exponent = 2^(exponent * log2(base))

    ; Swap so ST(0) = base, ST(1) = exponent
    fxch st1

    ; Check for base <= 0 (can't take log of non-positive)
    ftst
    fstsw ax
    sahf
    jbe .pow_special_case

    ; ST(0) = base, ST(1) = exponent
    ; fyl2x: ST(1) = ST(1) * log2(ST(0)), pop
    fyl2x                   ; ST(0) = exponent * log2(base)

    ; Now compute 2^ST(0)
    ; Split into integer and fractional parts
    fld st0                 ; Duplicate
    frndint                 ; ST(0) = integer part
    fsub st1, st0           ; ST(1) = fractional part
    fxch st1                ; ST(0) = fractional, ST(1) = integer

    ; 2^fractional using f2xm1 (works for -1 <= x <= 1)
    f2xm1                   ; ST(0) = 2^frac - 1
    fld1
    faddp st1, st0          ; ST(0) = 2^frac

    ; Scale by 2^integer
    fscale                  ; ST(0) = 2^frac * 2^int = 2^(frac+int)
    fstp st1                ; Pop the integer part

    ; Store result
    fstp qword [rbp - 8]
    movsd xmm0, [rbp - 8]
    call stack_data_push_double
    jmp .pow_done

.pow_special_case:
    ; Handle base <= 0
    ; For simplicity, return 0 for base=0, NaN for negative
    fstp st0                ; Pop base
    fstp st0                ; Pop exponent
    xorpd xmm0, xmm0        ; Return 0
    call stack_data_push_double
    jmp .pow_done

.pow_type_error_pop:
    fstp st0                ; Clean up FPU stack
.pow_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
    jmp .pow_done

.pow_error:
.pow_done:
    pop rbx
    add rsp, 32
    pop rbp
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

    movzx esi, byte [rcx + VALUE_TAG_OFF]
    movzx edi, byte [rdx + VALUE_TAG_OFF]

    cmp esi, VTAG_INTEGER
    jne .check_double
    cmp edi, VTAG_INTEGER
    jne .mixed

    ; Both integers
    mov rax, [rdx + VALUE_NUM_OFF]
    cmp rax, [rcx + VALUE_NUM_OFF]
    setl dil
    movzx edi, dil
    call stack_data_push_boolean
    ret

.check_double:
    cmp esi, VTAG_DOUBLE
    jne .type_error
    cmp edi, VTAG_DOUBLE
    jne .mixed

    ; Both doubles
    movsd xmm0, [rdx + VALUE_NUM_OFF]  ; Left
    ucomisd xmm0, [rcx + VALUE_NUM_OFF] ; Compare with right
    setb dil                            ; Set if below (unsigned for floats)
    movzx edi, dil
    call stack_data_push_boolean
    ret

.mixed:
    ; Mixed types: convert integer to double
    cmp esi, VTAG_INTEGER
    je .right_is_int
    cmp esi, VTAG_DOUBLE
    jne .type_error
    ; Right (rcx) is double, left (rdx) must be integer
    cmp edi, VTAG_INTEGER
    jne .type_error
    cvtsi2sd xmm0, qword [rdx + VALUE_NUM_OFF]  ; Convert left to double
    ucomisd xmm0, [rcx + VALUE_NUM_OFF]         ; Compare
    setb dil
    movzx edi, dil
    call stack_data_push_boolean
    ret

.right_is_int:
    ; Right (rcx) is integer, left (rdx) must be double
    cmp edi, VTAG_DOUBLE
    jne .type_error
    movsd xmm0, [rdx + VALUE_NUM_OFF]           ; Load left as double
    cvtsi2sd xmm1, qword [rcx + VALUE_NUM_OFF]  ; Convert right to double
    ucomisd xmm0, xmm1                          ; Compare
    setb dil
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
    mov rcx, rax            ; Right operand

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax            ; Left operand

    movzx esi, byte [rcx + VALUE_TAG_OFF]
    movzx edi, byte [rdx + VALUE_TAG_OFF]

    cmp esi, VTAG_INTEGER
    jne .check_double
    cmp edi, VTAG_INTEGER
    jne .mixed

    ; Both integers
    mov rax, [rdx + VALUE_NUM_OFF]
    cmp rax, [rcx + VALUE_NUM_OFF]
    setle dil
    movzx edi, dil
    call stack_data_push_boolean
    ret

.check_double:
    cmp esi, VTAG_DOUBLE
    jne .type_error
    cmp edi, VTAG_DOUBLE
    jne .mixed

    ; Both doubles
    movsd xmm0, [rdx + VALUE_NUM_OFF]  ; Left
    ucomisd xmm0, [rcx + VALUE_NUM_OFF] ; Compare with right
    setbe dil                           ; Set if below or equal
    movzx edi, dil
    call stack_data_push_boolean
    ret

.mixed:
    ; Mixed types: convert integer to double
    cmp esi, VTAG_INTEGER
    je .right_is_int
    cmp esi, VTAG_DOUBLE
    jne .type_error
    ; Right (rcx) is double, left (rdx) must be integer
    cmp edi, VTAG_INTEGER
    jne .type_error
    cvtsi2sd xmm0, qword [rdx + VALUE_NUM_OFF]  ; Convert left to double
    ucomisd xmm0, [rcx + VALUE_NUM_OFF]         ; Compare
    setbe dil
    movzx edi, dil
    call stack_data_push_boolean
    ret

.right_is_int:
    ; Right (rcx) is integer, left (rdx) must be double
    cmp edi, VTAG_DOUBLE
    jne .type_error
    movsd xmm0, [rdx + VALUE_NUM_OFF]           ; Load left as double
    cvtsi2sd xmm1, qword [rcx + VALUE_NUM_OFF]  ; Convert right to double
    ucomisd xmm0, xmm1                          ; Compare
    setbe dil
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
    mov rcx, rax            ; Right operand

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax            ; Left operand

    movzx esi, byte [rcx + VALUE_TAG_OFF]
    movzx edi, byte [rdx + VALUE_TAG_OFF]

    cmp esi, VTAG_INTEGER
    jne .check_double
    cmp edi, VTAG_INTEGER
    jne .mixed

    ; Both integers
    mov rax, [rdx + VALUE_NUM_OFF]
    cmp rax, [rcx + VALUE_NUM_OFF]
    setg dil
    movzx edi, dil
    call stack_data_push_boolean
    ret

.check_double:
    cmp esi, VTAG_DOUBLE
    jne .type_error
    cmp edi, VTAG_DOUBLE
    jne .mixed

    ; Both doubles
    movsd xmm0, [rdx + VALUE_NUM_OFF]  ; Left
    ucomisd xmm0, [rcx + VALUE_NUM_OFF] ; Compare with right
    seta dil                            ; Set if above
    movzx edi, dil
    call stack_data_push_boolean
    ret

.mixed:
    ; Mixed types: convert integer to double
    cmp esi, VTAG_INTEGER
    je .right_is_int
    cmp esi, VTAG_DOUBLE
    jne .type_error
    ; Right (rcx) is double, left (rdx) must be integer
    cmp edi, VTAG_INTEGER
    jne .type_error
    cvtsi2sd xmm0, qword [rdx + VALUE_NUM_OFF]  ; Convert left to double
    ucomisd xmm0, [rcx + VALUE_NUM_OFF]         ; Compare
    seta dil
    movzx edi, dil
    call stack_data_push_boolean
    ret

.right_is_int:
    ; Right (rcx) is integer, left (rdx) must be double
    cmp edi, VTAG_DOUBLE
    jne .type_error
    movsd xmm0, [rdx + VALUE_NUM_OFF]           ; Load left as double
    cvtsi2sd xmm1, qword [rcx + VALUE_NUM_OFF]  ; Convert right to double
    ucomisd xmm0, xmm1                          ; Compare
    seta dil
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
    mov rcx, rax            ; Right operand

    call stack_data_pop
    test rax, rax
    jz .error
    mov rdx, rax            ; Left operand

    movzx esi, byte [rcx + VALUE_TAG_OFF]
    movzx edi, byte [rdx + VALUE_TAG_OFF]

    cmp esi, VTAG_INTEGER
    jne .check_double
    cmp edi, VTAG_INTEGER
    jne .mixed

    ; Both integers
    mov rax, [rdx + VALUE_NUM_OFF]
    cmp rax, [rcx + VALUE_NUM_OFF]
    setge dil
    movzx edi, dil
    call stack_data_push_boolean
    ret

.check_double:
    cmp esi, VTAG_DOUBLE
    jne .type_error
    cmp edi, VTAG_DOUBLE
    jne .mixed

    ; Both doubles
    movsd xmm0, [rdx + VALUE_NUM_OFF]  ; Left
    ucomisd xmm0, [rcx + VALUE_NUM_OFF] ; Compare with right
    setae dil                           ; Set if above or equal
    movzx edi, dil
    call stack_data_push_boolean
    ret

.mixed:
    ; Mixed types: convert integer to double
    cmp esi, VTAG_INTEGER
    je .right_is_int
    cmp esi, VTAG_DOUBLE
    jne .type_error
    ; Right (rcx) is double, left (rdx) must be integer
    cmp edi, VTAG_INTEGER
    jne .type_error
    cvtsi2sd xmm0, qword [rdx + VALUE_NUM_OFF]  ; Convert left to double
    ucomisd xmm0, [rcx + VALUE_NUM_OFF]         ; Compare
    setae dil
    movzx edi, dil
    call stack_data_push_boolean
    ret

.right_is_int:
    ; Right (rcx) is integer, left (rdx) must be double
    cmp edi, VTAG_DOUBLE
    jne .type_error
    movsd xmm0, [rdx + VALUE_NUM_OFF]           ; Load left as double
    cvtsi2sd xmm1, qword [rcx + VALUE_NUM_OFF]  ; Convert right to double
    ucomisd xmm0, xmm1                          ; Compare
    setae dil
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
; Control flow operations
;-----------------------------------------------------------------------------

extern stack_control_push_frame
extern stack_control_pop_frame
extern stack_control_find_loop
extern stack_control_handle_break
extern stack_control_handle_continue
extern stack_entity_push
extern stack_entity_pop
extern stack_entity_lookup
extern array_length
extern array_get
extern value_is_truthy

; Frame types (from stack_control.asm)
FRAME_LOOP      equ 2
FRAME_FORALL    equ 3
FRAME_FLAG_BREAK    equ 1
FRAME_FLAG_CONTINUE equ 2

;-----------------------------------------------------------------------------
; op_if - Execute body if condition is true
; Stack: body bool --
;-----------------------------------------------------------------------------
op_if:
    push rbp
    mov rbp, rsp
    push rbx                ; Use rbx instead of r12 (r12 is stack pointer!)

    ; Pop condition
    call stack_data_pop
    test rax, rax
    jz .if_error
    mov rbx, rax            ; Condition value

    ; Pop body (bytecode)
    call stack_data_pop
    test rax, rax
    jz .if_error
    push rax                ; Save body

    ; Check if condition is truthy
    mov rdi, rbx
    call value_is_truthy
    test eax, eax
    jz .if_skip

    ; Execute body
    pop rdi                 ; Body value
    call execute_bytecode_value
    jmp .if_done

.if_skip:
    pop rax                 ; Discard body

.if_done:
    pop rbx
    pop rbp
    ret

.if_error:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_ifelse - Execute appropriate body based on condition
; Stack: true_body false_body bool --
;-----------------------------------------------------------------------------
op_ifelse:
    push rbp
    mov rbp, rsp
    push rbx                ; Use rbx instead of r12 (r12 is stack pointer!)
    push r15                ; Use r15 instead of r13 (r13 is entity stack pointer!)

    ; Pop condition
    call stack_data_pop
    test rax, rax
    jz .ifelse_error
    mov rbx, rax            ; Condition

    ; Pop false body
    call stack_data_pop
    test rax, rax
    jz .ifelse_error
    mov r15, rax            ; False body

    ; Pop true body
    call stack_data_pop
    test rax, rax
    jz .ifelse_error
    push rax                ; True body

    ; Check condition
    mov rdi, rbx
    call value_is_truthy
    test eax, eax
    jz .ifelse_false

    ; Execute true body
    pop rdi
    call execute_bytecode_value
    jmp .ifelse_done

.ifelse_false:
    pop rax                 ; Discard true body
    mov rdi, r15
    call execute_bytecode_value

.ifelse_done:
    pop r15
    pop rbx
    pop rbp
    ret

.ifelse_error:
    pop r15
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_while - Execute body while condition is true
; Stack: body test --
;-----------------------------------------------------------------------------
op_while:
    push rbp
    mov rbp, rsp
    push r12
    push r13
    push r14

    ; Pop test bytecode
    call stack_data_pop
    test rax, rax
    jz .while_error
    mov r12, rax            ; Test

    ; Pop body bytecode
    call stack_data_pop
    test rax, rax
    jz .while_error
    mov r13, rax            ; Body

    ; Push loop frame (return address is for break)
    mov rdi, FRAME_LOOP
    mov rsi, rbx            ; Current bytecode position
    xor edx, edx
    call stack_control_push_frame
    test eax, eax
    jnz .while_error

.while_loop:
    ; Execute test
    mov rdi, r12
    call execute_bytecode_value

    ; Pop test result
    call stack_data_pop
    test rax, rax
    jz .while_exit

    ; Check if truthy
    mov rdi, rax
    call value_is_truthy
    test eax, eax
    jz .while_exit

    ; Execute body
    mov rdi, r13
    call execute_bytecode_value

    ; Check for break flag
    call stack_control_peek_frame
    test rax, rax
    jz .while_loop
    test byte [rax + 1], FRAME_FLAG_BREAK  ; Frame.flags offset
    jnz .while_exit

    ; Check for continue flag - clear it and continue
    test byte [rax + 1], FRAME_FLAG_CONTINUE
    jz .while_loop
    and byte [rax + 1], ~FRAME_FLAG_CONTINUE
    jmp .while_loop

.while_exit:
    ; Pop loop frame
    call stack_control_pop_frame

.while_done:
    pop r14
    pop r13
    pop r12
    pop rbp
    ret

.while_error:
    pop r14
    pop r13
    pop r12
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_for - Iterate over array, pushing each element
; Stack: body array --
;-----------------------------------------------------------------------------
op_for:
    push rbp
    mov rbp, rsp
    push r12
    push r13
    push r14
    push r15

    ; Pop array
    call stack_data_pop
    test rax, rax
    jz .for_error

    ; Check it's an array
    cmp byte [rax + VALUE_TAG_OFF], VTAG_ARRAY
    jne .for_type_error
    mov r12, [rax + VALUE_PTR_OFF]  ; Array header

    ; Pop body
    call stack_data_pop
    test rax, rax
    jz .for_error
    mov r13, rax            ; Body

    ; Get array length
    mov rdi, r12
    call array_length
    mov r14, rax            ; Length
    xor r15, r15            ; Index = 0

    ; Push loop frame
    mov rdi, FRAME_LOOP
    mov rsi, rbx
    xor edx, edx
    call stack_control_push_frame
    test eax, eax
    jnz .for_error

.for_loop:
    cmp r15, r14
    jae .for_exit

    ; Get element at index
    mov rdi, r12
    mov rsi, r15
    call array_get
    test rax, rax
    jz .for_next

    ; Push element to data stack
    mov rdi, rax
    call stack_data_push

    ; Execute body
    mov rdi, r13
    call execute_bytecode_value

    ; Check for break
    call stack_control_peek_frame
    test rax, rax
    jz .for_next
    test byte [rax + 1], FRAME_FLAG_BREAK
    jnz .for_exit

    ; Check for continue - clear and continue
    test byte [rax + 1], FRAME_FLAG_CONTINUE
    jz .for_next
    and byte [rax + 1], ~FRAME_FLAG_CONTINUE

.for_next:
    inc r15
    jmp .for_loop

.for_exit:
    call stack_control_pop_frame

.for_done:
    pop r15
    pop r14
    pop r13
    pop r12
    pop rbp
    ret

.for_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.for_error:
    pop r15
    pop r14
    pop r13
    pop r12
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_forall - Iterate over array, pushing each element to entity stack
; Stack: body array --
;-----------------------------------------------------------------------------
op_forall:
    push rbp
    mov rbp, rsp
    push r12
    push r13
    push r14
    push r15

    ; Pop array
    call stack_data_pop
    test rax, rax
    jz .forall_error

    cmp byte [rax + VALUE_TAG_OFF], VTAG_ARRAY
    jne .forall_type_error
    mov r12, [rax + VALUE_PTR_OFF]  ; Array header

    ; Pop body
    call stack_data_pop
    test rax, rax
    jz .forall_error
    mov r13, rax            ; Body

    ; Get array length
    mov rdi, r12
    call array_length
    mov r14, rax            ; Length
    xor r15, r15            ; Index = 0

    ; Push loop frame with FORALL type
    mov rdi, FRAME_FORALL
    mov rsi, rbx
    mov rdx, r12            ; Store array in extra
    call stack_control_push_frame
    test eax, eax
    jnz .forall_error

.forall_loop:
    cmp r15, r14
    jae .forall_exit

    ; Get element at index
    mov rdi, r12
    mov rsi, r15
    call array_get
    test rax, rax
    jz .forall_next

    ; Push to entity stack
    mov rdi, rax
    call stack_entity_push

    ; Execute body
    mov rdi, r13
    call execute_bytecode_value

    ; Pop from entity stack
    call stack_entity_pop

    ; Check for break
    call stack_control_peek_frame
    test rax, rax
    jz .forall_next
    test byte [rax + 1], FRAME_FLAG_BREAK
    jnz .forall_exit

    ; Check for continue
    test byte [rax + 1], FRAME_FLAG_CONTINUE
    jz .forall_next
    and byte [rax + 1], ~FRAME_FLAG_CONTINUE

.forall_next:
    inc r15
    jmp .forall_loop

.forall_exit:
    call stack_control_pop_frame

.forall_done:
    pop r15
    pop r14
    pop r13
    pop r12
    pop rbp
    ret

.forall_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.forall_error:
    pop r15
    pop r14
    pop r13
    pop r12
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_break - Exit innermost loop
; Stack: --
;-----------------------------------------------------------------------------
op_break:
    push rbp
    mov rbp, rsp

    ; Find nearest loop frame
    call stack_control_find_loop
    test rax, rax
    jz .break_error         ; No enclosing loop

    ; Set break flag
    or byte [rax + 1], FRAME_FLAG_BREAK

.break_done:
    pop rbp
    ret

.break_error:
    mov dword [state + State.error], ERR_INVALID_OPCODE  ; No loop to break from
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_continue - Jump to next loop iteration
; Stack: --
;-----------------------------------------------------------------------------
op_continue:
    push rbp
    mov rbp, rsp

    ; Find nearest loop frame
    call stack_control_find_loop
    test rax, rax
    jz .continue_error

    ; Set continue flag
    or byte [rax + 1], FRAME_FLAG_CONTINUE

.continue_done:
    pop rbp
    ret

.continue_error:
    mov dword [state + State.error], ERR_INVALID_OPCODE
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_return - Return from current procedure
; Stack: --
;-----------------------------------------------------------------------------
op_return:
    ; Set halted flag to exit current execution
    mov dword [state + State.halted], 1
    ret

;-----------------------------------------------------------------------------
; op_exec - Execute bytecode object
; Stack: bytecode --
;-----------------------------------------------------------------------------
op_exec:
    push rbp
    mov rbp, rsp

    ; Pop bytecode value
    call stack_data_pop
    test rax, rax
    jz .exec_error

    ; Execute it
    mov rdi, rax
    call execute_bytecode_value

.exec_done:
    pop rbp
    ret

.exec_error:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_call - Call named procedure
; Stack: name --
; Looks up name in entity stack and executes the associated bytecode
;-----------------------------------------------------------------------------
op_call:
    push rbp
    mov rbp, rsp
    push r12

    ; Pop name value
    call stack_data_pop
    test rax, rax
    jz .call_error

    ; Check it's a name or string
    movzx ecx, byte [rax + VALUE_TAG_OFF]
    cmp cl, VTAG_NAME
    je .call_lookup
    cmp cl, VTAG_STRING
    jne .call_type_error

.call_lookup:
    ; Get the name/string data pointer
    mov r12, [rax + VALUE_PTR_OFF]  ; Length-prefixed string

    ; Look up in entity stack
    mov rdi, r12
    call stack_entity_lookup
    test rax, rax
    jz .call_not_found

    ; rax points to the Value containing the procedure bytecode
    ; Execute it
    mov rdi, rax
    call execute_bytecode_value

    jmp .call_done

.call_not_found:
    mov dword [state + State.error], ERR_NAME_NOT_FOUND
    jmp .call_done

.call_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH

.call_error:
.call_done:
    pop r12
    pop rbp
    ret

;-----------------------------------------------------------------------------
; execute_bytecode_value - Execute bytecode stored in a Value
; Input: rdi = pointer to Value containing bytecode
; This is a helper for control flow operations
;-----------------------------------------------------------------------------
execute_bytecode_value:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    ; Check value type - should be array or string containing bytecode
    movzx eax, byte [rdi + VALUE_TAG_OFF]
    cmp al, VTAG_ARRAY
    je .exec_array
    cmp al, VTAG_STRING
    je .exec_string
    jmp .exec_type_error

.exec_array:
    ; Array of bytecode - get data pointer
    mov rax, [rdi + VALUE_PTR_OFF]  ; Array header
    ; For now, treat as raw bytecode if it's a string-like array
    ; This is a simplification - real implementation would iterate opcodes
    jmp .exec_done

.exec_string:
    ; String containing bytecode
    mov rax, [rdi + VALUE_PTR_OFF]  ; String storage
    mov r12, [rax]                   ; Length
    lea r13, [rax + 8]               ; Bytecode data

    ; Save current bytecode pointers
    push qword [state + State.bytecode]
    push qword [state + State.bytecode_end]
    push rbx                         ; Current bytecode IP

    ; Set up new bytecode pointers
    mov [state + State.bytecode], r13
    lea rax, [r13 + r12]
    mov [state + State.bytecode_end], rax
    mov rbx, r13

    ; Clear halted flag for nested execution
    mov dword [state + State.halted], 0

    ; Execute nested bytecode
.exec_nested_loop:
    cmp dword [state + State.halted], 0
    jne .exec_nested_done

    cmp rbx, [state + State.bytecode_end]
    jae .exec_nested_done

    ; Check for errors
    cmp dword [state + State.error], ERR_NONE
    jne .exec_nested_done

    ; Fetch and dispatch opcode
    movzx eax, byte [rbx]
    inc rbx

    lea rcx, [opcode_table]
    mov rcx, [rcx + rax * 8]
    call rcx

    jmp .exec_nested_loop

.exec_nested_done:
    ; Restore bytecode pointers
    pop rbx
    pop qword [state + State.bytecode_end]
    pop qword [state + State.bytecode]

    ; Clear halted flag (it was just for nested execution)
    mov dword [state + State.halted], 0

    jmp .exec_done

.exec_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH

.exec_done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; Helper: stack_control_peek_frame - Get top frame without popping
; Output: rax = frame pointer or 0
;-----------------------------------------------------------------------------
stack_control_peek_frame:
    cmp r14, [state + State.control_stack_base]
    jae .peek_empty
    mov rax, r14
    ret
.peek_empty:
    xor eax, eax
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
; String operations
;-----------------------------------------------------------------------------

;-----------------------------------------------------------------------------
; op_str_concat - Concatenate two strings
; Stack: str1 str2 -- result
;-----------------------------------------------------------------------------
op_str_concat:
    push rbp
    mov rbp, rsp
    push rbx                ; Use rbx instead of r12 (r12 is stack pointer!)

    ; Pop second string
    call stack_data_pop
    test rax, rax
    jz .concat_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .concat_type_error
    mov rbx, [rax + VALUE_PTR_OFF]  ; String 2

    ; Pop first string
    call stack_data_pop
    test rax, rax
    jz .concat_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .concat_type_error

    ; Concatenate
    mov rdi, [rax + VALUE_PTR_OFF]  ; String 1
    mov rsi, rbx                     ; String 2
    call string_concat
    test rax, rax
    jz .concat_error

    ; Create value and push
    push rax
    lea rdi, [rax + 8]      ; String data
    mov rsi, [rax]          ; Length
    call value_new_string
    add rsp, 8
    test rax, rax
    jz .concat_error

    mov rdi, rax
    call stack_data_push

.concat_done:
    pop rbx
    pop rbp
    ret

.concat_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.concat_error:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_str_length - Get string length
; Stack: str -- int
;-----------------------------------------------------------------------------
op_str_length:
    call stack_data_pop
    test rax, rax
    jz .strlen_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .strlen_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    call string_length
    mov rdi, rax
    call stack_data_push_integer
    ret

.strlen_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.strlen_error:
    ret

;-----------------------------------------------------------------------------
; op_str_substring - Extract substring
; Stack: str start len -- result
;-----------------------------------------------------------------------------
op_str_substring:
    push rbp
    mov rbp, rsp
    push rbx                ; Use rbx instead of r12 (r12 is stack pointer!)
    push r15                ; Use r15 instead of r13 (r13 is entity stack pointer!)

    ; Pop length
    call stack_data_pop
    test rax, rax
    jz .substr_error
    mov rbx, [rax + VALUE_NUM_OFF]  ; Length

    ; Pop start
    call stack_data_pop
    test rax, rax
    jz .substr_error
    mov r15, [rax + VALUE_NUM_OFF]  ; Start

    ; Pop string
    call stack_data_pop
    test rax, rax
    jz .substr_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .substr_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, r15
    mov rdx, rbx
    call string_substring
    test rax, rax
    jz .substr_error

    ; Create value and push
    push rax
    lea rdi, [rax + 8]
    mov rsi, [rax]
    call value_new_string
    add rsp, 8
    test rax, rax
    jz .substr_error

    mov rdi, rax
    call stack_data_push

.substr_done:
    pop r15
    pop rbx
    pop rbp
    ret

.substr_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.substr_error:
    pop r15
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_str_indexof - Find first occurrence of substring
; Stack: haystack needle -- index
;-----------------------------------------------------------------------------
op_str_indexof:
    push rbp
    mov rbp, rsp
    push rbx                ; Use rbx instead of r12 (r12 is stack pointer!)

    ; Pop needle
    call stack_data_pop
    test rax, rax
    jz .indexof_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .indexof_type_error
    mov rbx, [rax + VALUE_PTR_OFF]

    ; Pop haystack
    call stack_data_pop
    test rax, rax
    jz .indexof_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .indexof_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call string_index_of
    mov rdi, rax
    call stack_data_push_integer

    pop rbx
    pop rbp
    ret

.indexof_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.indexof_error:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_str_lastindexof - Find last occurrence of substring
; Stack: haystack needle -- index
;-----------------------------------------------------------------------------
op_str_lastindexof:
    push rbp
    mov rbp, rsp
    push rbx                ; Use rbx instead of r12 (r12 is stack pointer!)

    call stack_data_pop
    test rax, rax
    jz .lastidx_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .lastidx_type_error
    mov rbx, [rax + VALUE_PTR_OFF]

    call stack_data_pop
    test rax, rax
    jz .lastidx_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .lastidx_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call string_last_index_of
    mov rdi, rax
    call stack_data_push_integer

    pop rbx
    pop rbp
    ret

.lastidx_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.lastidx_error:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_str_startswith - Check if string starts with prefix
; Stack: str prefix -- bool
;-----------------------------------------------------------------------------
op_str_startswith:
    push rbp
    mov rbp, rsp
    push rbx                ; Use rbx instead of r12 (r12 is stack pointer!)

    call stack_data_pop
    test rax, rax
    jz .startswith_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .startswith_type_error
    mov rbx, [rax + VALUE_PTR_OFF]

    call stack_data_pop
    test rax, rax
    jz .startswith_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .startswith_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call string_starts_with
    mov rdi, rax
    call stack_data_push_boolean

    pop rbx
    pop rbp
    ret

.startswith_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.startswith_error:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_str_endswith - Check if string ends with suffix
; Stack: str suffix -- bool
;-----------------------------------------------------------------------------
op_str_endswith:
    push rbp
    mov rbp, rsp
    push rbx                ; Use rbx instead of r12 (r12 is stack pointer!)

    call stack_data_pop
    test rax, rax
    jz .endswith_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .endswith_type_error
    mov rbx, [rax + VALUE_PTR_OFF]

    call stack_data_pop
    test rax, rax
    jz .endswith_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .endswith_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call string_ends_with
    mov rdi, rax
    call stack_data_push_boolean

    pop rbx
    pop rbp
    ret

.endswith_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.endswith_error:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_str_contains - Check if string contains substring
; Stack: str substr -- bool
;-----------------------------------------------------------------------------
op_str_contains:
    push rbp
    mov rbp, rsp
    push rbx                ; Use rbx instead of r12 (r12 is stack pointer!)

    call stack_data_pop
    test rax, rax
    jz .contains_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .contains_type_error
    mov rbx, [rax + VALUE_PTR_OFF]

    call stack_data_pop
    test rax, rax
    jz .contains_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .contains_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call string_contains
    mov rdi, rax
    call stack_data_push_boolean

    pop rbx
    pop rbp
    ret

.contains_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.contains_error:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_str_replace - Replace first occurrence
; Stack: str pattern replacement -- result
;-----------------------------------------------------------------------------
op_str_replace:
    push rbp
    mov rbp, rsp
    push rbx                ; Use rbx instead of r12 (r12 is stack pointer!)
    push r15                ; Use r15 instead of r13 (r13 is entity stack pointer!)

    ; Pop replacement
    call stack_data_pop
    test rax, rax
    jz .replace_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .replace_type_error
    mov rbx, [rax + VALUE_PTR_OFF]

    ; Pop pattern
    call stack_data_pop
    test rax, rax
    jz .replace_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .replace_type_error
    mov r15, [rax + VALUE_PTR_OFF]

    ; Pop string
    call stack_data_pop
    test rax, rax
    jz .replace_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .replace_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, r15
    mov rdx, rbx
    call string_replace
    test rax, rax
    jz .replace_error

    ; Create value and push
    push rax
    lea rdi, [rax + 8]
    mov rsi, [rax]
    call value_new_string
    add rsp, 8
    test rax, rax
    jz .replace_error

    mov rdi, rax
    call stack_data_push

    pop r15
    pop rbx
    pop rbp
    ret

.replace_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.replace_error:
    pop r15
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_str_replaceall - Replace all occurrences
; Stack: str pattern replacement -- result
;-----------------------------------------------------------------------------
op_str_replaceall:
    push rbp
    mov rbp, rsp
    push rbx                ; Use rbx instead of r12 (r12 is stack pointer!)
    push r15                ; Use r15 instead of r13 (r13 is entity stack pointer!)

    call stack_data_pop
    test rax, rax
    jz .replall_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .replall_type_error
    mov rbx, [rax + VALUE_PTR_OFF]

    call stack_data_pop
    test rax, rax
    jz .replall_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .replall_type_error
    mov r15, [rax + VALUE_PTR_OFF]

    call stack_data_pop
    test rax, rax
    jz .replall_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .replall_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, r15
    mov rdx, rbx
    call string_replace_all
    test rax, rax
    jz .replall_error

    push rax
    lea rdi, [rax + 8]
    mov rsi, [rax]
    call value_new_string
    add rsp, 8
    test rax, rax
    jz .replall_error

    mov rdi, rax
    call stack_data_push

    pop r15
    pop rbx
    pop rbp
    ret

.replall_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.replall_error:
    pop r15
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_str_split - Split string by delimiter
; Stack: str delim -- array
;-----------------------------------------------------------------------------
op_str_split:
    push rbp
    mov rbp, rsp
    push rbx                ; Use rbx instead of r12 (r12 is stack pointer!)

    call stack_data_pop
    test rax, rax
    jz .split_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .split_type_error
    mov rbx, [rax + VALUE_PTR_OFF]

    call stack_data_pop
    test rax, rax
    jz .split_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .split_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call string_split
    test rax, rax
    jz .split_error

    ; Create array value and push
    push rax
    call value_new_array
    pop rcx                 ; Array header
    test rax, rax
    jz .split_error
    mov [rax + VALUE_PTR_OFF], rcx

    mov rdi, rax
    call stack_data_push

    pop rbx
    pop rbp
    ret

.split_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.split_error:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_str_join - Join array of strings with delimiter
; Stack: array delim -- str
;-----------------------------------------------------------------------------
op_str_join:
    push rbp
    mov rbp, rsp
    push rbx                ; Use rbx instead of r12 (r12 is stack pointer!)

    call stack_data_pop
    test rax, rax
    jz .join_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .join_type_error
    mov rbx, [rax + VALUE_PTR_OFF]

    call stack_data_pop
    test rax, rax
    jz .join_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_ARRAY
    jne .join_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call string_join
    test rax, rax
    jz .join_error

    push rax
    lea rdi, [rax + 8]
    mov rsi, [rax]
    call value_new_string
    add rsp, 8
    test rax, rax
    jz .join_error

    mov rdi, rax
    call stack_data_push

    pop rbx
    pop rbp
    ret

.join_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.join_error:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_str_trim - Trim whitespace from both ends
; Stack: str -- result
;-----------------------------------------------------------------------------
op_str_trim:
    call stack_data_pop
    test rax, rax
    jz .trim_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .trim_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    call string_trim
    test rax, rax
    jz .trim_error

    push rax
    lea rdi, [rax + 8]
    mov rsi, [rax]
    call value_new_string
    add rsp, 8
    test rax, rax
    jz .trim_error

    mov rdi, rax
    call stack_data_push
    ret

.trim_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.trim_error:
    ret

;-----------------------------------------------------------------------------
; op_str_trimleft - Trim whitespace from start
; Stack: str -- result
;-----------------------------------------------------------------------------
op_str_trimleft:
    call stack_data_pop
    test rax, rax
    jz .triml_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .triml_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    call string_trim_left
    test rax, rax
    jz .triml_error

    push rax
    lea rdi, [rax + 8]
    mov rsi, [rax]
    call value_new_string
    add rsp, 8
    test rax, rax
    jz .triml_error

    mov rdi, rax
    call stack_data_push
    ret

.triml_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.triml_error:
    ret

;-----------------------------------------------------------------------------
; op_str_trimright - Trim whitespace from end
; Stack: str -- result
;-----------------------------------------------------------------------------
op_str_trimright:
    call stack_data_pop
    test rax, rax
    jz .trimr_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .trimr_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    call string_trim_right
    test rax, rax
    jz .trimr_error

    push rax
    lea rdi, [rax + 8]
    mov rsi, [rax]
    call value_new_string
    add rsp, 8
    test rax, rax
    jz .trimr_error

    mov rdi, rax
    call stack_data_push
    ret

.trimr_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.trimr_error:
    ret

;-----------------------------------------------------------------------------
; op_str_toupper - Convert to uppercase
; Stack: str -- result
;-----------------------------------------------------------------------------
op_str_toupper:
    call stack_data_pop
    test rax, rax
    jz .toupper_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .toupper_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    call string_to_upper
    test rax, rax
    jz .toupper_error

    push rax
    lea rdi, [rax + 8]
    mov rsi, [rax]
    call value_new_string
    add rsp, 8
    test rax, rax
    jz .toupper_error

    mov rdi, rax
    call stack_data_push
    ret

.toupper_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.toupper_error:
    ret

;-----------------------------------------------------------------------------
; op_str_tolower - Convert to lowercase
; Stack: str -- result
;-----------------------------------------------------------------------------
op_str_tolower:
    call stack_data_pop
    test rax, rax
    jz .tolower_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .tolower_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    call string_to_lower
    test rax, rax
    jz .tolower_error

    push rax
    lea rdi, [rax + 8]
    mov rsi, [rax]
    call value_new_string
    add rsp, 8
    test rax, rax
    jz .tolower_error

    mov rdi, rax
    call stack_data_push
    ret

.tolower_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.tolower_error:
    ret

;-----------------------------------------------------------------------------
; op_str_tostring - Convert value to string representation
; Stack: value -- str
;-----------------------------------------------------------------------------
op_str_tostring:
    push rbp
    mov rbp, rsp
    sub rsp, 32             ; Buffer for number conversion
    push rbx                ; Use rbx instead of r12 (r12 is stack pointer!)

    call stack_data_pop
    test rax, rax
    jz .tostr_error
    mov rbx, rax

    movzx eax, byte [rbx + VALUE_TAG_OFF]

    cmp al, VTAG_STRING
    je .tostr_string
    cmp al, VTAG_INTEGER
    je .tostr_integer
    cmp al, VTAG_DOUBLE
    je .tostr_double
    cmp al, VTAG_BOOLEAN
    je .tostr_boolean
    cmp al, VTAG_NULL
    je .tostr_null
    jmp .tostr_other

.tostr_string:
    ; Already a string, just push it back
    mov rdi, rbx
    call stack_data_push
    jmp .tostr_done

.tostr_integer:
    ; Convert integer to string
    mov rax, [rbx + VALUE_NUM_OFF]
    lea rdi, [rbp - 32]     ; Buffer
    mov rsi, rdi            ; Save start
    add rdi, 20             ; Start from end
    mov byte [rdi], 0       ; Null terminator
    push rsi

    ; Handle negative
    mov r8, 0
    test rax, rax
    jns .tostr_int_pos
    neg rax
    mov r8, 1

.tostr_int_pos:
    test rax, rax
    jnz .tostr_int_loop
    dec rdi
    mov byte [rdi], '0'
    jmp .tostr_int_neg_check

.tostr_int_loop:
    test rax, rax
    jz .tostr_int_neg_check
    mov rcx, 10
    xor edx, edx
    div rcx
    add dl, '0'
    dec rdi
    mov [rdi], dl
    jmp .tostr_int_loop

.tostr_int_neg_check:
    test r8, r8
    jz .tostr_int_create
    dec rdi
    mov byte [rdi], '-'

.tostr_int_create:
    pop rsi                 ; Original buffer start (unused)
    ; rdi points to start of string
    ; Calculate length
    lea rax, [rbp - 32 + 20]
    sub rax, rdi            ; Length
    mov rsi, rax
    call value_new_string
    test rax, rax
    jz .tostr_error
    mov rdi, rax
    call stack_data_push
    jmp .tostr_done

.tostr_double:
    ; Simplified: convert double to string with fixed precision
    ; For now, just push a placeholder
    ; Full implementation would need proper float formatting
    movsd xmm0, [rbx + VALUE_NUM_OFF]

    ; Convert to integer part for simple representation
    cvttsd2si rax, xmm0
    lea rdi, [rbp - 32]
    add rdi, 20
    mov byte [rdi], 0

    mov r8, 0
    test rax, rax
    jns .tostr_dbl_pos
    neg rax
    mov r8, 1

.tostr_dbl_pos:
    test rax, rax
    jnz .tostr_dbl_loop
    dec rdi
    mov byte [rdi], '0'
    jmp .tostr_dbl_neg

.tostr_dbl_loop:
    test rax, rax
    jz .tostr_dbl_neg
    mov rcx, 10
    xor edx, edx
    div rcx
    add dl, '0'
    dec rdi
    mov [rdi], dl
    jmp .tostr_dbl_loop

.tostr_dbl_neg:
    test r8, r8
    jz .tostr_dbl_create
    dec rdi
    mov byte [rdi], '-'

.tostr_dbl_create:
    lea rax, [rbp - 32 + 20]
    sub rax, rdi
    mov rsi, rax
    call value_new_string
    test rax, rax
    jz .tostr_error
    mov rdi, rax
    call stack_data_push
    jmp .tostr_done

.tostr_boolean:
    mov rax, [rbx + VALUE_NUM_OFF]
    test rax, rax
    jz .tostr_false
    ; "true"
    lea rdi, [str_true_const]
    mov rsi, 4
    jmp .tostr_bool_create

.tostr_false:
    lea rdi, [str_false_const]
    mov rsi, 5

.tostr_bool_create:
    call value_new_string
    test rax, rax
    jz .tostr_error
    mov rdi, rax
    call stack_data_push
    jmp .tostr_done

.tostr_null:
    lea rdi, [str_null_const]
    mov rsi, 4
    call value_new_string
    test rax, rax
    jz .tostr_error
    mov rdi, rax
    call stack_data_push
    jmp .tostr_done

.tostr_other:
    ; For arrays/entities, return type name
    lea rdi, [str_object_const]
    mov rsi, 8
    call value_new_string
    test rax, rax
    jz .tostr_error
    mov rdi, rax
    call stack_data_push
    jmp .tostr_done

.tostr_error:
.tostr_done:
    pop rbx
    add rsp, 32
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_str_tointeger - Convert string to integer
; Stack: str -- int
;-----------------------------------------------------------------------------
op_str_tointeger:
    push rbp
    mov rbp, rsp
    push rbx                ; Use rbx instead of r12 (r12 is stack pointer!)
    push r15                ; Use r15 instead of r13 (r13 is entity stack pointer!)

    call stack_data_pop
    test rax, rax
    jz .toint_error

    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .toint_type_error

    mov rax, [rax + VALUE_PTR_OFF]  ; String storage
    mov rbx, [rax]                   ; Length
    lea r15, [rax + 8]               ; Data pointer

    ; Parse integer: optional sign followed by digits
    xor ecx, ecx            ; Result
    xor r8d, r8d            ; Sign flag (0=positive)
    xor r9d, r9d            ; Index

    ; Check for empty string
    test rbx, rbx
    jz .toint_zero

    ; Check for sign
    movzx eax, byte [r15]
    cmp al, '-'
    jne .toint_check_plus
    mov r8d, 1
    inc r9
    jmp .toint_parse

.toint_check_plus:
    cmp al, '+'
    jne .toint_parse
    inc r9

.toint_parse:
    cmp r9, rbx
    jae .toint_finish

    movzx eax, byte [r15 + r9]
    cmp al, '0'
    jb .toint_finish
    cmp al, '9'
    ja .toint_finish

    ; digit = al - '0'
    sub al, '0'
    ; result = result * 10 + digit
    imul rcx, 10
    movzx eax, al
    add rcx, rax
    inc r9
    jmp .toint_parse

.toint_finish:
    ; Apply sign
    test r8d, r8d
    jz .toint_push
    neg rcx

.toint_push:
    mov rdi, rcx
    call stack_data_push_integer
    jmp .toint_done

.toint_zero:
    xor edi, edi
    call stack_data_push_integer
    jmp .toint_done

.toint_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.toint_error:
.toint_done:
    pop r15
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_str_todouble - Convert string to double
; Stack: str -- double
;-----------------------------------------------------------------------------
op_str_todouble:
    push rbp
    mov rbp, rsp
    sub rsp, 32             ; Use stack frame for Length and Data instead of r12/r13
    push rbx                ; Use rbx for integer part (instead of r14)
    push r11                ; Use r11 for index (instead of r15)

    call stack_data_pop
    test rax, rax
    jz .todbl_error

    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .todbl_type_error

    mov rax, [rax + VALUE_PTR_OFF]
    mov rcx, [rax]                   ; Length
    mov [rbp - 8], rcx               ; Store length in stack frame
    lea rcx, [rax + 8]
    mov [rbp - 16], rcx              ; Store data ptr in stack frame

    ; Parse: [sign] digits [. digits] [e [sign] digits]
    xor rbx, rbx            ; Integer part
    xor r11, r11            ; Index
    xor r8d, r8d            ; Sign (0=pos)

    mov rcx, [rbp - 8]      ; Length
    test rcx, rcx
    jz .todbl_zero

    ; Check sign
    mov rax, [rbp - 16]     ; Data pointer
    movzx eax, byte [rax]
    cmp al, '-'
    jne .todbl_check_plus
    mov r8d, 1
    inc r11
    jmp .todbl_int_part

.todbl_check_plus:
    cmp al, '+'
    jne .todbl_int_part
    inc r11

.todbl_int_part:
    ; Parse integer part
    mov rcx, [rbp - 8]      ; Length
    cmp r11, rcx
    jae .todbl_convert

    mov rax, [rbp - 16]     ; Data pointer
    movzx eax, byte [rax + r11]
    cmp al, '0'
    jb .todbl_check_dot
    cmp al, '9'
    ja .todbl_check_dot

    sub al, '0'
    imul rbx, 10
    movzx eax, al
    add rbx, rax
    inc r11
    jmp .todbl_int_part

.todbl_check_dot:
    cmp al, '.'
    jne .todbl_convert
    inc r11

    ; Parse fractional part
    mov rcx, 1              ; Divisor (will be 10, 100, 1000, etc.)
    xor r9, r9              ; Fractional digits value

.todbl_frac_part:
    mov rax, [rbp - 8]      ; Length
    cmp r11, rax
    jae .todbl_convert

    mov rax, [rbp - 16]     ; Data pointer
    movzx eax, byte [rax + r11]
    cmp al, '0'
    jb .todbl_convert
    cmp al, '9'
    ja .todbl_convert

    sub al, '0'
    imul r9, 10
    movzx eax, al
    add r9, rax
    imul rcx, 10
    inc r11
    jmp .todbl_frac_part

.todbl_convert:
    ; Convert integer part to double
    cvtsi2sd xmm0, rbx

    ; Add fractional part if any
    test rcx, rcx
    jz .todbl_apply_sign
    cmp rcx, 1
    je .todbl_apply_sign

    cvtsi2sd xmm1, r9
    cvtsi2sd xmm2, rcx
    divsd xmm1, xmm2
    addsd xmm0, xmm1

.todbl_apply_sign:
    test r8d, r8d
    jz .todbl_push

    ; Negate
    mov rax, 0x8000000000000000
    movq xmm1, rax
    xorpd xmm0, xmm1

.todbl_push:
    call stack_data_push_double
    jmp .todbl_done

.todbl_zero:
    xorpd xmm0, xmm0
    call stack_data_push_double
    jmp .todbl_done

.todbl_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.todbl_error:
.todbl_done:
    pop r11
    pop rbx
    add rsp, 32
    pop rbp
    ret

;-----------------------------------------------------------------------------
; Array operations
;-----------------------------------------------------------------------------

;-----------------------------------------------------------------------------
; op_array_new - Create new empty array
; Stack: -- array
;-----------------------------------------------------------------------------
op_array_new:
    xor edi, edi
    call array_alloc
    test rax, rax
    jz .anew_error

    push rax
    call value_new_array
    pop rcx
    test rax, rax
    jz .anew_error
    mov [rax + VALUE_PTR_OFF], rcx

    mov rdi, rax
    call stack_data_push
    ret

.anew_error:
    ret

;-----------------------------------------------------------------------------
; op_array_get - Get array element
; Stack: array index -- value
;-----------------------------------------------------------------------------
op_array_get:
    push rbp
    mov rbp, rsp
    push rbx                ; Use rbx instead of r12 (r12 is stack pointer!)

    call stack_data_pop
    test rax, rax
    jz .aget_error
    mov rbx, [rax + VALUE_NUM_OFF]  ; Index

    call stack_data_pop
    test rax, rax
    jz .aget_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_ARRAY
    jne .aget_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call array_get
    test rax, rax
    jz .aget_error

    mov rdi, rax
    call stack_data_push

    pop rbx
    pop rbp
    ret

.aget_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.aget_error:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_array_set - Set array element
; Stack: array index value --
;-----------------------------------------------------------------------------
op_array_set:
    push rbp
    mov rbp, rsp
    push rbx                ; Use rbx instead of r12 (r12 is stack pointer!)
    push r15                ; Use r15 instead of r13 (r13 is entity stack pointer!)

    call stack_data_pop
    test rax, rax
    jz .aset_error
    mov rbx, rax            ; Value

    call stack_data_pop
    test rax, rax
    jz .aset_error
    mov r15, [rax + VALUE_NUM_OFF]  ; Index

    call stack_data_pop
    test rax, rax
    jz .aset_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_ARRAY
    jne .aset_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, r15
    mov rdx, rbx
    call array_set

    pop r15
    pop rbx
    pop rbp
    ret

.aset_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.aset_error:
    pop r15
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_array_len - Get array length
; Stack: array -- int
;-----------------------------------------------------------------------------
op_array_len:
    call stack_data_pop
    test rax, rax
    jz .alen_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_ARRAY
    jne .alen_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    call array_length
    mov rdi, rax
    call stack_data_push_integer
    ret

.alen_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.alen_error:
    ret

;-----------------------------------------------------------------------------
; op_array_push - Push element to end of array
; Stack: array value --
;-----------------------------------------------------------------------------
op_array_push:
    push rbp
    mov rbp, rsp
    push rbx                ; Use rbx instead of r12 (r12 is stack pointer!)

    call stack_data_pop
    test rax, rax
    jz .apush_error
    mov rbx, rax            ; Value

    call stack_data_pop
    test rax, rax
    jz .apush_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_ARRAY
    jne .apush_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call array_push

    pop rbx
    pop rbp
    ret

.apush_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.apush_error:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_array_pop - Pop element from end of array
; Stack: array -- value
;-----------------------------------------------------------------------------
op_array_pop:
    call stack_data_pop
    test rax, rax
    jz .apop_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_ARRAY
    jne .apop_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    call array_pop
    test rax, rax
    jz .apop_error

    mov rdi, rax
    call stack_data_push
    ret

.apop_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.apop_error:
    ret

;-----------------------------------------------------------------------------
; op_array_concat - Concatenate two arrays
; Stack: array1 array2 -- result
;-----------------------------------------------------------------------------
op_array_concat:
    push rbp
    mov rbp, rsp
    push rbx                ; Use rbx instead of r12 (r12 is stack pointer!)

    call stack_data_pop
    test rax, rax
    jz .aconcat_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_ARRAY
    jne .aconcat_type_error
    mov rbx, [rax + VALUE_PTR_OFF]

    call stack_data_pop
    test rax, rax
    jz .aconcat_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_ARRAY
    jne .aconcat_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call array_concat
    test rax, rax
    jz .aconcat_error

    push rax
    call value_new_array
    pop rcx
    test rax, rax
    jz .aconcat_error
    mov [rax + VALUE_PTR_OFF], rcx

    mov rdi, rax
    call stack_data_push

    pop rbx
    pop rbp
    ret

.aconcat_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.aconcat_error:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_array_slice - Extract slice of array
; Stack: array start end -- result
;-----------------------------------------------------------------------------
op_array_slice:
    push rbp
    mov rbp, rsp
    push rbx                ; Use rbx instead of r12 (r12 is stack pointer!)
    push r15                ; Use r15 instead of r13 (r13 is entity stack pointer!)

    call stack_data_pop
    test rax, rax
    jz .aslice_error
    mov rbx, [rax + VALUE_NUM_OFF]  ; End

    call stack_data_pop
    test rax, rax
    jz .aslice_error
    mov r15, [rax + VALUE_NUM_OFF]  ; Start

    call stack_data_pop
    test rax, rax
    jz .aslice_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_ARRAY
    jne .aslice_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, r15
    mov rdx, rbx
    call array_slice
    test rax, rax
    jz .aslice_error

    push rax
    call value_new_array
    pop rcx
    test rax, rax
    jz .aslice_error
    mov [rax + VALUE_PTR_OFF], rcx

    mov rdi, rax
    call stack_data_push

    pop r15
    pop rbx
    pop rbp
    ret

.aslice_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.aslice_error:
    pop r15
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_array_copy - Create shallow copy of array
; Stack: array -- result
;-----------------------------------------------------------------------------
op_array_copy:
    call stack_data_pop
    test rax, rax
    jz .acopy_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_ARRAY
    jne .acopy_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    call array_copy
    test rax, rax
    jz .acopy_error

    push rax
    call value_new_array
    pop rcx
    test rax, rax
    jz .acopy_error
    mov [rax + VALUE_PTR_OFF], rcx

    mov rdi, rax
    call stack_data_push
    ret

.acopy_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.acopy_error:
    ret

;-----------------------------------------------------------------------------
; op_array_clear - Clear all elements from array
; Stack: array --
;-----------------------------------------------------------------------------
op_array_clear:
    call stack_data_pop
    test rax, rax
    jz .aclear_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_ARRAY
    jne .aclear_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    call array_clear
    ret

.aclear_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.aclear_error:
    ret

;-----------------------------------------------------------------------------
; op_array_remove - Remove element at index
; Stack: array index --
;-----------------------------------------------------------------------------
op_array_remove:
    push rbp
    mov rbp, rsp
    push rbx                ; Use rbx instead of r12 (r12 is stack pointer!)

    call stack_data_pop
    test rax, rax
    jz .aremove_error
    mov rbx, [rax + VALUE_NUM_OFF]  ; Index

    call stack_data_pop
    test rax, rax
    jz .aremove_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_ARRAY
    jne .aremove_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call array_remove

    pop rbx
    pop rbp
    ret

.aremove_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.aremove_error:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_array_insert - Insert element at index
; Stack: array index value --
;-----------------------------------------------------------------------------
op_array_insert:
    push rbp
    mov rbp, rsp
    push rbx                ; Use rbx instead of r12 (r12 is stack pointer!)
    push r15                ; Use r15 instead of r13 (r13 is entity stack pointer!)

    call stack_data_pop
    test rax, rax
    jz .ainsert_error
    mov rbx, rax            ; Value

    call stack_data_pop
    test rax, rax
    jz .ainsert_error
    mov r15, [rax + VALUE_NUM_OFF]  ; Index

    call stack_data_pop
    test rax, rax
    jz .ainsert_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_ARRAY
    jne .ainsert_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, r15
    mov rdx, rbx
    call array_insert

    pop r15
    pop rbx
    pop rbp
    ret

.ainsert_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.ainsert_error:
    pop r15
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_array_reverse - Reverse array in place
; Stack: array --
;-----------------------------------------------------------------------------
op_array_reverse:
    call stack_data_pop
    test rax, rax
    jz .areverse_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_ARRAY
    jne .areverse_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    call array_reverse
    ret

.areverse_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.areverse_error:
    ret

;-----------------------------------------------------------------------------
; op_array_sort - Sort array in place
; Stack: array --
;-----------------------------------------------------------------------------
op_array_sort:
    call stack_data_pop
    test rax, rax
    jz .asort_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_ARRAY
    jne .asort_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    call array_sort
    ret

.asort_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.asort_error:
    ret

;-----------------------------------------------------------------------------
; op_array_contains - Check if array contains value
; Stack: array value -- bool
;-----------------------------------------------------------------------------
op_array_contains:
    push rbp
    mov rbp, rsp
    push rbx                ; Use rbx instead of r12 (r12 is stack pointer!)

    call stack_data_pop
    test rax, rax
    jz .acontains_error
    mov rbx, rax            ; Value

    call stack_data_pop
    test rax, rax
    jz .acontains_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_ARRAY
    jne .acontains_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call array_contains
    mov rdi, rax
    call stack_data_push_boolean

    pop rbx
    pop rbp
    ret

.acontains_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.acontains_error:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; op_array_indexof - Find index of value in array
; Stack: array value -- int
;-----------------------------------------------------------------------------
op_array_indexof:
    push rbp
    mov rbp, rsp
    push rbx                ; Use rbx instead of r12 (r12 is stack pointer!)

    call stack_data_pop
    test rax, rax
    jz .aidxof_error
    mov rbx, rax            ; Value

    call stack_data_pop
    test rax, rax
    jz .aidxof_error
    cmp byte [rax + VALUE_TAG_OFF], VTAG_ARRAY
    jne .aidxof_type_error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call array_index_of
    mov rdi, rax
    call stack_data_push_integer

    pop rbx
    pop rbp
    ret

.aidxof_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.aidxof_error:
    pop rbx
    pop rbp
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
    ; Trace: Print trace message with bytecode position
    ; Prints: "TRACE: @<offset>"
    push rbp
    mov rbp, rsp

    ; Print "TRACE: @"
    lea rdi, [trace_prefix]
    call print_string

    ; Print current bytecode offset
    mov rdi, rbx
    sub rdi, [state + State.bytecode]  ; Subtract base to get offset
    dec rdi                             ; Account for opcode we just read
    call print_hex

    call print_newline

    pop rbp
    ret

op_debug:
    ; Debug: Print stack state
    ; Shows data stack contents
    push rbp
    mov rbp, rsp

    ; Print header
    lea rdi, [debug_header]
    call print_string
    call print_newline

    ; Print data stack
    lea rdi, [debug_stack_label]
    call print_string
    call print_stack

    call print_newline

    pop rbp
    ret

op_halt:
    mov dword [state + State.halted], 1
    ret

op_invalid:
    mov dword [state + State.error], ERR_INVALID_OPCODE
    ret

;-----------------------------------------------------------------------------
; New opcodes for Go compatibility
;-----------------------------------------------------------------------------

; op_push_constant - Push constant from pool (opcode 1)
; Followed by varint index into constant pool
global op_push_constant
op_push_constant:
    ; Read constant index
    call read_varint
    ; Get constant from pool (TODO: implement constant pool)
    ; For now, just push null as placeholder
    call stack_data_push_null
    ret

; op_mark - Push a mark value onto stack (opcode 10)
global op_mark
op_mark:
    ; Push a mark/sentinel value (implemented as null for now)
    call stack_data_push_null
    ret

; op_inc - Increment top of stack (opcode 27)
global op_inc
op_inc:
    call stack_data_pop
    test rax, rax
    jz .inc_done
    cmp byte [rax + VALUE_TAG_OFF], VTAG_INTEGER
    jne .inc_type_error
    mov rdi, [rax + VALUE_NUM_OFF]
    inc rdi
    call stack_data_push_integer
    ret
.inc_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.inc_done:
    ret

; op_dec - Decrement top of stack (opcode 28)
global op_dec
op_dec:
    call stack_data_pop
    test rax, rax
    jz .dec_done
    cmp byte [rax + VALUE_TAG_OFF], VTAG_INTEGER
    jne .dec_type_error
    mov rdi, [rax + VALUE_NUM_OFF]
    dec rdi
    call stack_data_push_integer
    ret
.dec_type_error:
    mov dword [state + State.error], ERR_TYPE_MISMATCH
.dec_done:
    ret

; op_push_zero - Push integer 0 (opcode 103)
global op_push_zero
op_push_zero:
    xor edi, edi
    call stack_data_push_integer
    ret

; op_push_one - Push integer 1 (opcode 104)
global op_push_one
op_push_one:
    mov edi, 1
    call stack_data_push_integer
    ret

; op_entitypush - Push entity to entity stack (opcode 60)
; Stack: entity --
global op_entitypush
op_entitypush:
    push rbp
    mov rbp, rsp

    ; Pop entity Value from data stack
    call stack_data_pop
    test rax, rax
    jz .error

    ; Check it's an entity type
    movzx ecx, byte [rax]
    cmp cl, VTAG_ENTITY
    jne .type_error

    ; Push entity pointer to entity stack
    mov rdi, rax
    call stack_entity_push

    pop rbp
    ret

.type_error:
    ; Not an entity - just return without error for now
.error:
    pop rbp
    ret

; op_entitypop - Pop entity from entity stack (opcode 61)
; Stack: -- entity
global op_entitypop
op_entitypop:
    push rbp
    mov rbp, rsp

    ; Pop from entity stack
    call stack_entity_pop
    test rax, rax
    jz .empty

    ; Push the entity Value to data stack
    mov rdi, rax
    call stack_data_push

    pop rbp
    ret

.empty:
    ; Stack was empty, push null
    call stack_data_push_null
    pop rbp
    ret

; op_def - Define variable on current entity (opcode 62)
; Stack: value name --
; Note: Name should be a literal name (from /name syntax)
global op_def
op_def:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    ; Pop name from stack
    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, rax            ; Name Value

    ; Pop value from stack
    call stack_data_pop
    test rax, rax
    jz .error
    mov r12, rax            ; Value to define

    ; Get name pointer (for entity define, we need string pointer)
    ; Names are stored with ptr pointing to length-prefixed string
    mov rdi, [rbx + VALUE_PTR_OFF]
    test rdi, rdi
    jz .error

    ; Define on top entity
    mov rsi, r12            ; Value pointer
    call stack_entity_define

    pop r12
    pop rbx
    pop rbp
    ret

.error:
    pop r12
    pop rbx
    pop rbp
    ret

; op_lookup - Lookup name in context (opcode 63)
; Stack: name -- value
; Pops name from data stack, looks up through entity stack
; Handles dotted names (entity.attribute) by finding the entity first
; If found and executable, executes it; otherwise pushes the result
global op_lookup
op_lookup:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    ; Pop name from data stack
    call stack_data_pop
    test rax, rax
    jz .not_found

    ; Check if it's a name type
    mov cl, [rax]               ; Get tag
    cmp cl, VTAG_NAME
    jne .not_found              ; Not a name

    ; Get name string pointer from Value.ptr
    mov r12, [rax + VALUE_PTR_OFF]  ; r12 = name string pointer
    test r12, r12
    jz .not_found

    ; Check if name contains a dot (entity.attribute format)
    ; Format: [length:8][data...]
    mov rcx, [r12]              ; Get length
    test rcx, rcx
    jz .not_found

    ; Scan for dot
    lea rsi, [r12 + 8]          ; Point to string data
    xor rdx, rdx                ; Index = 0
.scan_dot:
    cmp rdx, rcx
    jae .no_dot                 ; No dot found
    cmp byte [rsi + rdx], '.'
    je .has_dot
    inc rdx
    jmp .scan_dot

.no_dot:
    ; Simple name - look up in entity stack as attribute
    mov rdi, r12
    call stack_entity_lookup
    test rax, rax
    jz .not_found
    jmp .found

.has_dot:
    ; rdx = position of dot
    ; Create temporary strings for entity name and attribute name
    ; Entity name: [rdx bytes from start]
    ; Attribute name: [rest after dot]
    mov r13, rdx                ; Save dot position

    ; Allocate space for entity name string
    lea rdi, [rdx + 9]          ; length (8) + data (rdx) + null
    call heap_alloc
    test rax, rax
    jz .not_found
    mov rbx, rax                ; rbx = entity name string

    ; Copy entity name
    mov qword [rbx], r13        ; Set length
    lea rdi, [rbx + 8]          ; Destination
    lea rsi, [r12 + 8]          ; Source (original string data)
    mov rcx, r13                ; Length
    rep movsb

    ; Now search entity stack for entity with this name
    call stack_entity_find_by_name
    test rax, rax
    jz .not_found

    ; rax = pointer to entity Value
    ; Get entity structure pointer
    mov rax, [rax + VALUE_PTR_OFF]
    test rax, rax
    jz .not_found

    ; Allocate attribute name string
    mov rcx, [r12]              ; Original length
    sub rcx, r13                ; Subtract entity part
    dec rcx                     ; Subtract the dot
    test rcx, rcx
    jz .not_found               ; Empty attribute name

    push rax                    ; Save entity pointer
    lea rdi, [rcx + 9]          ; length (8) + data (attr_len) + null
    call heap_alloc
    test rax, rax
    jz .not_found_pop
    mov rbx, rax                ; rbx = attribute name string

    mov rcx, [r12]              ; Original length
    sub rcx, r13                ; Subtract entity part
    dec rcx                     ; Subtract the dot
    mov qword [rbx], rcx        ; Set length

    ; Copy attribute name
    lea rdi, [rbx + 8]          ; Destination
    lea rsi, [r12 + 8 + r13 + 1] ; Source (after entity part + dot)
    rep movsb

    pop rax                     ; Restore entity pointer

    ; Look up attribute in this entity
    mov rdi, rax                ; Entity structure
    mov rsi, rbx                ; Attribute name
    call entity_get_attr
    test rax, rax
    jz .not_found
    ; rax now points to the Value
    jmp .found

.not_found_pop:
    pop rax                     ; Clean up stack
    jmp .not_found

.found:
    ; Push found value to data stack
    mov rdi, rax
    call stack_data_push
    jmp .done

.not_found:
    ; Push null for not found
    call stack_data_push_null

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

; stack_entity_find_by_name - Find entity by name
; Input: rbx = pointer to name string (length-prefixed)
; Output: rax = pointer to entity Value, or 0 if not found
stack_entity_find_by_name:
    push rbp
    mov rbp, rsp
    push r12
    push r13
    push r14

    mov r12, rbx                ; Save name pointer

    ; Get entity stack depth
    call stack_entity_depth
    test rax, rax
    jz .sef_not_found

    mov r13, rax                ; Number of entities
    xor r14, r14                ; Current index

.sef_check_entity:
    ; Get entity at index r14
    mov rdi, r14
    call stack_entity_peek_n
    test rax, rax
    jz .sef_next

    ; rax = pointer to entity Value
    push rax                    ; Save entity Value pointer
    ; Get entity structure pointer
    mov rax, [rax + VALUE_PTR_OFF]
    test rax, rax
    jz .sef_next_pop

    ; Get entity type name pointer
    ; Entity structure: [type_ptr:8][count:8][cap:8][attrs_ptr:8]
    mov rdi, [rax]              ; Type name pointer
    test rdi, rdi
    jz .sef_next_pop

    ; Compare type name with search name
    mov rsi, r12
    call string_compare
    test eax, eax
    jnz .sef_next_pop           ; Not equal

    ; Found matching entity
    pop rax                     ; Restore entity Value pointer
    jmp .sef_done

.sef_next_pop:
    pop rax                     ; Clean up stack

.sef_next:
    inc r14
    cmp r14, r13
    jb .sef_check_entity

.sef_not_found:
    xor eax, eax

.sef_done:
    pop r14
    pop r13
    pop r12
    pop rbp
    ret

; op_newentity - Create new entity (opcode 64)
; Stack: -- entity
global op_newentity
op_newentity:
    push rbp
    mov rbp, rsp
    push rbx

    ; Allocate new entity (untyped)
    xor edi, edi            ; No type name
    call entity_alloc
    test rax, rax
    jz .error

    mov rbx, rax            ; Save entity pointer

    ; Allocate Value for the entity
    mov edi, VALUE_SIZE
    call heap_alloc
    test rax, rax
    jz .error

    ; Initialize Value as entity
    mov byte [rax], VTAG_ENTITY
    mov qword [rax + VALUE_NUM_OFF], 0
    mov [rax + VALUE_PTR_OFF], rbx

    ; Push to data stack
    mov rdi, rax
    call stack_data_push

    pop rbx
    pop rbp
    ret

.error:
    call stack_data_push_null
    pop rbx
    pop rbp
    ret

; op_newtable - Create new table/map (opcode 80)
; Stack: ( -- table )
global op_newtable
op_newtable:
    push rbp
    mov rbp, rsp
    push rbx

    ; Allocate hash table
    xor edi, edi            ; Default bucket count
    call hashtable_alloc
    test rax, rax
    jz .newtable_error

    mov rbx, rax            ; Save table pointer

    ; Allocate Value wrapper
    mov edi, VALUE_SIZE
    call heap_alloc
    test rax, rax
    jz .newtable_error

    ; Initialize Value
    mov byte [rax], VTAG_TABLE
    mov qword [rax + VALUE_NUM_OFF], 0
    mov [rax + VALUE_PTR_OFF], rbx

    ; Push to stack
    mov rdi, rax
    call stack_data_push
    jmp .newtable_done

.newtable_error:
    call stack_data_push_null

.newtable_done:
    pop rbx
    pop rbp
    ret

; op_tableget - Get value from table (opcode 81)
; Stack: ( table key -- value )
global op_tableget
op_tableget:
    push rbp
    mov rbp, rsp
    push rbx

    ; Pop key
    call stack_data_pop
    test rax, rax
    jz .tableget_error
    mov rbx, [rax + VALUE_PTR_OFF]  ; Key string

    ; Pop table
    call stack_data_pop
    test rax, rax
    jz .tableget_error
    mov rdi, [rax + VALUE_PTR_OFF]  ; Hash table
    test rdi, rdi
    jz .tableget_error

    ; Get value
    mov rsi, rbx
    call hashtable_get
    test rax, rax
    jz .tableget_null

    ; Push found value
    mov rdi, rax
    call stack_data_push
    jmp .tableget_done

.tableget_null:
    call stack_data_push_null
    jmp .tableget_done

.tableget_error:
    call stack_data_push_null

.tableget_done:
    pop rbx
    pop rbp
    ret

; op_tableput - Put value in table (opcode 82)
; Stack: ( table key value -- )
global op_tableput
op_tableput:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    ; Pop value
    call stack_data_pop
    test rax, rax
    jz .tableput_done
    mov r12, rax            ; Value pointer

    ; Pop key
    call stack_data_pop
    test rax, rax
    jz .tableput_done
    mov r13, [rax + VALUE_PTR_OFF]  ; Key string

    ; Pop table
    call stack_data_pop
    test rax, rax
    jz .tableput_done
    mov rbx, [rax + VALUE_PTR_OFF]  ; Hash table
    test rbx, rbx
    jz .tableput_done

    ; Put value
    mov rdi, rbx
    mov rsi, r13
    mov rdx, r12
    call hashtable_put

.tableput_done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

; op_operator - Call operator by index (opcode 200)
; Reads varint index, dispatches to operator implementation
; NOTE: rbx is the bytecode pointer and must be preserved across this call
;       (read_varint advances it, which is correct)
global op_operator
op_operator:
    push rbp
    mov rbp, rsp
    push r15            ; Use r15 instead of rbx for the index

    ; Read operator index from bytecode
    ; rbx will be advanced past the varint - this is correct!
    call read_varint
    mov r15, rax        ; Save index in r15 (not rbx!)

    ; Bounds check
    cmp r15, OPERATOR_TABLE_SIZE
    jae .unknown

    ; Dispatch via jump table
    lea rax, [operator_table]
    mov rax, [rax + r15 * 8]
    test rax, rax
    jz .unknown

    ; Call the operator
    call rax

    pop r15
    pop rbp
    ret

.unknown:
    ; Unknown operator - just return for now
    pop r15
    pop rbp
    ret

;-----------------------------------------------------------------------------
; Operator dispatch table
; Indices match Go operator registry assignments
;-----------------------------------------------------------------------------
OPERATOR_TABLE_SIZE equ 200

section .data
align 8
operator_table:
    dq impl_newarray        ; 0: newarray
    dq impl_addto           ; 1: addto
    dq impl_addat           ; 2: addat
    dq impl_length          ; 3: length
    dq impl_getat           ; 4: getat
    dq impl_removeat        ; 5: removeat
    dq impl_remove          ; 6: remove
    dq impl_memberof        ; 7: memberof
    dq impl_copy            ; 8: copy
    dq impl_first           ; 9: first
    dq impl_last            ; 10: last
    dq impl_copyelements    ; 11: copyelements
    dq impl_sortarray       ; 12: sortarray
    dq impl_sortentities    ; 13: sortentities
    dq impl_add_no_dups     ; 14: add_no_dups
    dq impl_merge           ; 15: merge
    dq impl_randomize       ; 16: randomize
    dq impl_intersection    ; 17: intersection
    dq impl_intersects      ; 18: intersects
    dq impl_addarray        ; 19: addarray
    dq impl_deepcopy        ; 20: deepcopy
    dq impl_tokenize        ; 21: tokenize
    dq impl_findmatch       ; 22: findmatch
    dq impl_cleararray      ; 23: cleararray
    dq op_and               ; 24: and
    dq op_or                ; 25: or
    dq op_not               ; 26: not
    dq op_xor               ; 27: xor
    dq op_gt                ; 28: >
    dq op_lt                ; 29: <
    dq op_ge                ; 30: >=
    dq op_le                ; 31: <=
    dq op_eq                ; 32: ==
    dq op_ne                ; 33: !=
    dq impl_beq             ; 34: beq
    dq impl_isnull          ; 35: isnull
    dq impl_notnull         ; 36: notnull
    dq op_push_true         ; 37: true
    dq op_push_false        ; 38: false
    dq op_if                ; 39: if
    dq op_ifelse            ; 40: ifelse
    dq op_while             ; 41: while
    dq op_exec              ; 42: execute
    dq op_for               ; 43: for
    dq impl_forr            ; 44: forr
    dq op_forall            ; 45: forall
    dq impl_forallr         ; 46: forallr
    dq impl_doloop          ; 47: doloop
    dq impl_lookup          ; 48: lookup
    dq impl_throwexception  ; 49: throwexception
    dq impl_forfirst        ; 50: forfirst
    dq impl_forfirstelse    ; 51: forfirstelse
    dq impl_entityforall    ; 52: entityforall
    dq impl_allocate        ; 53: allocate
    dq impl_deallocate      ; 54: deallocate
    dq impl_local_fetch     ; 55: local@
    dq impl_local_store     ; 56: local!
    dq impl_ignore          ; 57: ignore
    dq impl_executetable    ; 58: executetable
    dq impl_now             ; 59: now
    dq impl_today           ; 60: today
    dq impl_newdate         ; 61: newdate
    dq impl_getyear         ; 62: getyear
    dq impl_getmonth        ; 63: getmonth
    dq impl_getday          ; 64: getday
    dq impl_adddays         ; 65: adddays
    dq impl_addmonths       ; 66: addmonths
    dq impl_addyears        ; 67: addyears
    dq impl_daysbetween     ; 68: daysbetween
    dq impl_monthsbetween   ; 69: monthsbetween
    dq impl_yearsbetween    ; 70: yearsbetween
    dq impl_datecmp         ; 71: datecmp
    dq impl_firstofmonth    ; 72: firstofmonth
    dq impl_firstofyear     ; 73: firstofyear
    dq impl_endofmonth      ; 74: endofmonth
    dq impl_yearof          ; 75: yearof
    dq impl_monthof         ; 76: monthof
    dq impl_dayof           ; 77: dayof
    dq impl_getdaysinyear   ; 78: getdaysinyear
    dq impl_getdaysinmonth  ; 79: getdaysinmonth
    dq impl_getdayofmonth   ; 80: getdayofmonth
    dq impl_datelt          ; 81: d<
    dq impl_dategt          ; 82: d>
    dq impl_dateeq          ; 83: d==
    dq impl_dateplus        ; 84: d+
    dq impl_dateminus       ; 85: d-
    dq impl_getdate         ; 86: getdate
    dq impl_gettimestamp    ; 87: gettimestamp
    dq op_entitypush        ; 88: entitypush
    dq op_entitypop         ; 89: entitypop
    dq op_def               ; 90: def
    dq impl_incontext       ; 91: InContext
    dq op_newentity         ; 92: newentity
    dq impl_entityname      ; 93: entityname
    dq impl_entityid        ; 94: entityid
    dq impl_req             ; 95: req
    dq op_add               ; 96: +
    dq op_sub               ; 97: -
    dq op_mul               ; 98: *
    dq op_div               ; 99: / (fdiv?)
    dq op_abs               ; 100: abs
    dq op_neg               ; 101: negate
    dq op_add               ; 102: f+
    dq op_sub               ; 103: f-
    dq op_mul               ; 104: f*
    dq op_div               ; 105: fdiv
    dq op_abs               ; 106: fabs
    dq op_neg               ; 107: fnegate
    dq impl_roundto         ; 108: roundto
    dq op_pop               ; 109: pop
    dq op_dup               ; 110: dup
    dq op_swap              ; 111: swap
    dq op_over              ; 112: over
    dq op_rot               ; 113: rot
    dq op_pick              ; 114: pick
    dq op_roll              ; 115: roll
    dq op_clear             ; 116: clear
    dq op_push_null         ; 117: null
    dq op_mark              ; 118: mark
    dq impl_arraytomark     ; 119: arraytomark
    dq impl_counttomark     ; 120: counttomark
    dq impl_cleartomark     ; 121: cleartomark
    dq op_debug             ; 122: debug
    dq op_print             ; 123: print
    dq impl_traceon         ; 124: traceon
    dq impl_traceoff        ; 125: traceoff
    dq impl_setdebug        ; 126: setdebug
    dq impl_pstack          ; 127: pstack
    dq impl_printtos        ; 128: printtos
    dq impl_clone           ; 129: clone
    dq impl_tor             ; 130: >r
    dq impl_fromr           ; 131: r>
    dq impl_i               ; 132: i
    dq impl_j               ; 133: j
    dq impl_k               ; 134: k
    dq impl_entityfetch     ; 135: entityfetch
    dq impl_get             ; 136: get
    dq impl_find            ; 137: find
    dq impl_xdef            ; 138: xdef
    dq impl_createentity    ; 139: createentity
    dq impl_findcreateentity ; 140: findcreateentity
    dq impl_cvi             ; 141: cvi
    dq impl_cvr             ; 142: cvr
    dq impl_cvb             ; 143: cvb
    dq impl_cve             ; 144: cve
    dq impl_cvn             ; 145: cvn
    dq impl_cvd             ; 146: cvd
    dq impl_error           ; 147: error
    dq impl_policystatements ; 148: policystatements
    dq op_str_concat        ; 149: concat
    dq op_str_concat        ; 150: s+
    dq op_str_substring     ; 151: substring
    dq op_str_trim          ; 152: trim
    dq op_str_trim          ; 153: strtrim
    dq op_str_tolower       ; 154: lowercase
    dq op_str_tolower       ; 155: tolowercase
    dq op_str_toupper       ; 156: uppercase
    dq op_str_toupper       ; 157: touppercase
    dq op_str_length        ; 158: stringlength
    dq op_str_length        ; 159: strlength
    dq op_str_indexof       ; 160: indexof
    dq op_str_startswith    ; 161: startswith
    dq op_str_endswith      ; 162: endswith
    dq op_str_contains      ; 163: contains
    dq op_str_replace       ; 164: replace
    dq op_str_split         ; 165: split
    dq op_str_tostring      ; 166: tostring
    dq op_str_tostring      ; 167: cvs
    dq impl_regexmatch      ; 168: regexmatch
    dq impl_streq           ; 169: s==
    dq impl_streqi          ; 170: s==i
    dq impl_strlt           ; 171: s<
    dq impl_strgt           ; 172: s>
    dq impl_strle           ; 173: s<=
    dq impl_strge           ; 174: s>=
    dq op_newtable          ; 175: newtable
    dq op_tableget          ; 176: tableget
    dq op_tableput          ; 177: tableput
    dq impl_tablekeys       ; 178: tablekeys
    dq impl_tablevalues     ; 179: tablevalues
    dq impl_tablecontains   ; 180: tablecontains
    dq impl_tableremove     ; 181: tableremove
    dq impl_tablesize       ; 182: tablesize
    ; Fill remainder with nop
    times (OPERATOR_TABLE_SIZE - 183) dq impl_nop

section .text

;-----------------------------------------------------------------------------
; Operator implementations
;-----------------------------------------------------------------------------

; impl_nop - Placeholder for unimplemented operators
impl_nop:
    ret

; impl_beq - Boolean equality
; Stack: a b -- bool
impl_beq:
    push rbp
    mov rbp, rsp
    push rbx

    ; Pop two values
    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, rax

    call stack_data_pop
    test rax, rax
    jz .error

    ; Get boolean values
    movzx ecx, byte [rax]
    cmp cl, VTAG_BOOLEAN
    jne .false
    movzx edx, byte [rbx]
    cmp dl, VTAG_BOOLEAN
    jne .false

    ; Compare boolean values (in num field)
    mov rcx, [rax + VALUE_NUM_OFF]
    mov rdx, [rbx + VALUE_NUM_OFF]
    cmp rcx, rdx
    jne .false

    mov edi, 1
    call stack_data_push_boolean
    jmp .done

.false:
    xor edi, edi
    call stack_data_push_boolean
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_streq - String equality
; Stack: a b -- bool
impl_streq:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, rax

    call stack_data_pop
    test rax, rax
    jz .error
    mov r12, rax

    ; Check both are strings
    movzx ecx, byte [r12]
    cmp cl, VTAG_STRING
    jne .false
    movzx edx, byte [rbx]
    cmp dl, VTAG_STRING
    jne .false

    ; Compare string pointers (length-prefixed strings)
    mov rdi, [r12 + VALUE_PTR_OFF]
    mov rsi, [rbx + VALUE_PTR_OFF]
    call string_equals
    test eax, eax
    jz .false

    mov edi, 1
    call stack_data_push_boolean
    jmp .done

.false:
    xor edi, edi
    call stack_data_push_boolean
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop r12
    pop rbx
    pop rbp
    ret

; impl_isnull - Check if value is null
; Stack: value -- bool
impl_isnull:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .is_null

    movzx ecx, byte [rax]
    cmp cl, VTAG_NULL
    je .is_null

    xor edi, edi
    call stack_data_push_boolean
    jmp .done

.is_null:
    mov edi, 1
    call stack_data_push_boolean

.done:
    pop rbp
    ret

; impl_notnull - Check if value is not null
; Stack: value -- bool
impl_notnull:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .is_null

    movzx ecx, byte [rax]
    cmp cl, VTAG_NULL
    je .is_null

    mov edi, 1
    call stack_data_push_boolean
    jmp .done

.is_null:
    xor edi, edi
    call stack_data_push_boolean

.done:
    pop rbp
    ret

; impl_lookup - Lookup name on entity stack
; Stack: name -- value
impl_lookup:
    push rbp
    mov rbp, rsp

    ; Pop name from stack
    call stack_data_pop
    test rax, rax
    jz .not_found

    ; Get name string pointer
    mov rdi, [rax + VALUE_PTR_OFF]
    test rdi, rdi
    jz .not_found

    ; Lookup on entity stack
    call stack_entity_lookup
    test rax, rax
    jz .not_found

    ; Push found value
    mov rdi, rax
    call stack_data_push
    jmp .done

.not_found:
    call stack_data_push_null

.done:
    pop rbp
    ret

; impl_get - Get attribute from entity
; Stack: name entity -- value
impl_get:
    push rbp
    mov rbp, rsp
    push rbx

    ; Pop entity
    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, rax

    ; Pop name
    call stack_data_pop
    test rax, rax
    jz .error

    ; Get entity pointer
    mov rdi, [rbx + VALUE_PTR_OFF]
    test rdi, rdi
    jz .error

    ; Get name string pointer
    mov rsi, [rax + VALUE_PTR_OFF]
    test rsi, rsi
    jz .error

    ; Get attribute
    call entity_get_attr
    test rax, rax
    jz .not_found

    ; Push attribute value
    mov rdi, rax
    call stack_data_push
    jmp .done

.not_found:
    call stack_data_push_null
    jmp .done

.error:
    call stack_data_push_null

.done:
    pop rbx
    pop rbp
    ret

; impl_xdef - Define on specific entity
; Stack: value name entity -- (defines name=value on entity)
impl_xdef:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    ; Pop entity
    call stack_data_pop
    test rax, rax
    jz .done
    mov rbx, rax            ; Entity Value

    ; Pop name
    call stack_data_pop
    test rax, rax
    jz .done
    mov r12, rax            ; Name Value

    ; Pop value
    call stack_data_pop
    test rax, rax
    jz .done
    mov r13, rax            ; Value to define

    ; Get entity structure pointer
    mov rdi, [rbx + VALUE_PTR_OFF]
    test rdi, rdi
    jz .done

    ; Get name string pointer
    mov rsi, [r12 + VALUE_PTR_OFF]
    test rsi, rsi
    jz .done

    ; Set attribute
    mov rdx, r13
    call entity_set_attr

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

; impl_req - Entity reference equality
; Stack: entity1 entity2 -- bool
impl_req:
    push rbp
    mov rbp, rsp
    push rbx

    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, rax

    call stack_data_pop
    test rax, rax
    jz .error

    ; Compare entity pointers
    mov rcx, [rax + VALUE_PTR_OFF]
    mov rdx, [rbx + VALUE_PTR_OFF]
    cmp rcx, rdx
    jne .false

    mov edi, 1
    call stack_data_push_boolean
    jmp .done

.false:
    xor edi, edi
    call stack_data_push_boolean
    jmp .done

.error:
    call stack_data_push_null

.done:
    pop rbx
    pop rbp
    ret

; impl_newarray - Create new array
; Stack: -- array
impl_newarray:
    push rbp
    mov rbp, rsp
    push rbx

    ; Allocate array
    xor edi, edi            ; Initial capacity = 0
    call array_alloc
    test rax, rax
    jz .error

    mov rbx, rax

    ; Create Value for array
    mov edi, VALUE_SIZE
    call heap_alloc
    test rax, rax
    jz .error

    mov byte [rax], VTAG_ARRAY
    mov qword [rax + VALUE_NUM_OFF], 0
    mov [rax + VALUE_PTR_OFF], rbx

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null

.done:
    pop rbx
    pop rbp
    ret

; impl_addto - Add element to array
; Stack: element array -- array
impl_addto:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    ; Pop array
    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, rax

    ; Pop element
    call stack_data_pop
    test rax, rax
    jz .error
    mov r12, rax

    ; Get array pointer
    mov rdi, [rbx + VALUE_PTR_OFF]
    test rdi, rdi
    jz .error

    ; Push element to array
    mov rsi, r12
    call array_push

    ; Push array back
    mov rdi, rbx
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null

.done:
    pop r12
    pop rbx
    pop rbp
    ret

; impl_length - Get array length
; Stack: array -- int
impl_length:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_PTR_OFF]
    test rdi, rdi
    jz .error

    call array_length
    mov rdi, rax
    call stack_data_push_integer
    jmp .done

.error:
    call stack_data_push_null

.done:
    pop rbp
    ret

; impl_getat - Get array element at index
; Stack: index array -- element
impl_getat:
    push rbp
    mov rbp, rsp
    push rbx

    ; Pop array
    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, rax

    ; Pop index
    call stack_data_pop
    test rax, rax
    jz .error
    mov rsi, [rax + VALUE_NUM_OFF]  ; Index

    ; Get array pointer
    mov rdi, [rbx + VALUE_PTR_OFF]
    test rdi, rdi
    jz .error

    call array_get
    test rax, rax
    jz .error

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null

.done:
    pop rbx
    pop rbp
    ret

; impl_first - Get first element
; Stack: array -- element
impl_first:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_PTR_OFF]
    test rdi, rdi
    jz .error

    xor esi, esi            ; Index 0
    call array_get
    test rax, rax
    jz .error

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null

.done:
    pop rbp
    ret

; impl_last - Get last element
; Stack: array -- element
impl_last:
    push rbp
    mov rbp, rsp
    push rbx

    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, rax

    mov rdi, [rbx + VALUE_PTR_OFF]
    test rdi, rdi
    jz .error

    push rdi
    call array_length
    pop rdi
    test rax, rax
    jz .error

    dec rax                 ; Last index
    mov rsi, rax
    call array_get
    test rax, rax
    jz .error

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null

.done:
    pop rbx
    pop rbp
    ret

; impl_memberof - Check if element is in array
; Stack: element array -- bool
impl_memberof:
    push rbp
    mov rbp, rsp
    push rbx

    ; Pop array
    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, rax

    ; Pop element
    call stack_data_pop
    test rax, rax
    jz .error

    ; Get array pointer
    mov rdi, [rbx + VALUE_PTR_OFF]
    test rdi, rdi
    jz .false

    ; Check if array contains element
    mov rsi, rax
    call array_contains
    test eax, eax
    jz .false

    mov edi, 1
    call stack_data_push_boolean
    jmp .done

.false:
    xor edi, edi
    call stack_data_push_boolean
    jmp .done

.error:
    call stack_data_push_null

.done:
    pop rbx
    pop rbp
    ret

; impl_allocate - Allocate local variable
; Stack: value -- (pushes entity with value as local 0)
impl_allocate:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    ; Pop value to store
    call stack_data_pop
    test rax, rax
    jz .error
    mov r12, rax

    ; Create entity for local scope
    xor edi, edi
    call entity_alloc
    test rax, rax
    jz .error
    mov rbx, rax

    ; Create Value for entity
    mov edi, VALUE_SIZE
    call heap_alloc
    test rax, rax
    jz .error

    mov byte [rax], VTAG_ENTITY
    mov qword [rax + VALUE_NUM_OFF], 0
    mov [rax + VALUE_PTR_OFF], rbx
    push rax                ; Save entity Value

    ; Set local variable 0 = value
    ; Create name "0" for the local
    mov rdi, local_name_0
    mov rsi, r12
    mov rdi, rbx
    call entity_set_attr

    ; Push entity to entity stack
    pop rdi
    call stack_entity_push
    jmp .done

.error:
.done:
    pop r12
    pop rbx
    pop rbp
    ret

; impl_deallocate - Deallocate local scope
; Stack: -- (pops entity from entity stack)
impl_deallocate:
    call stack_entity_pop
    ret

; impl_local_fetch - Fetch local variable
; Stack: index -- value
impl_local_fetch:
    push rbp
    mov rbp, rsp

    ; Pop index
    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_NUM_OFF]  ; Get integer index

    ; Get entity at top of entity stack (local scope)
    call stack_entity_peek
    test rax, rax
    jz .error

    ; Lookup local variable by index
    ; For simplicity, we look up "0", "1", etc.
    ; TODO: proper local variable implementation

    ; Push null for now
    call stack_data_push_null
    jmp .done

.error:
    call stack_data_push_null

.done:
    pop rbp
    ret

; impl_local_store - Store local variable
; Stack: value index --
impl_local_store:
    push rbp
    mov rbp, rsp

    ; Pop index
    call stack_data_pop
    test rax, rax
    jz .done

    ; Pop value
    call stack_data_pop
    ; TODO: store in local scope

.done:
    pop rbp
    ret

; impl_now - Get current timestamp
; Stack: -- timestamp
impl_now:
    ; TODO: implement using syscall
    call stack_data_push_null
    ret

; impl_today - Get current date
; Stack: -- date
impl_today:
    ; TODO: implement using syscall
    call stack_data_push_null
    ret

; impl_createentity - Create entity from type name
; Stack: typename -- entity
impl_createentity:
    push rbp
    mov rbp, rsp
    push rbx

    ; Pop type name
    call stack_data_pop
    test rax, rax
    jz .error

    ; Get name string
    mov rdi, [rax + VALUE_PTR_OFF]

    ; Allocate entity with type
    call entity_alloc
    test rax, rax
    jz .error
    mov rbx, rax

    ; Create Value for entity
    mov edi, VALUE_SIZE
    call heap_alloc
    test rax, rax
    jz .error

    mov byte [rax], VTAG_ENTITY
    mov qword [rax + VALUE_NUM_OFF], 0
    mov [rax + VALUE_PTR_OFF], rbx

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null

.done:
    pop rbx
    pop rbp
    ret

; impl_cvi - Convert to integer
; Stack: value -- int
impl_cvi:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error

    movzx ecx, byte [rax]
    cmp cl, VTAG_INTEGER
    je .push_it
    cmp cl, VTAG_DOUBLE
    je .from_double
    cmp cl, VTAG_BOOLEAN
    je .from_bool
    jmp .error

.from_double:
    ; Convert double to integer (truncate)
    mov rdi, [rax + VALUE_NUM_OFF]
    ; TODO: proper double conversion
    call stack_data_push_integer
    jmp .done

.from_bool:
    mov rdi, [rax + VALUE_NUM_OFF]
    call stack_data_push_integer
    jmp .done

.push_it:
    mov rdi, [rax + VALUE_NUM_OFF]
    call stack_data_push_integer
    jmp .done

.error:
    call stack_data_push_null

.done:
    pop rbp
    ret

; impl_cvb - Convert to boolean
; Stack: value -- bool
impl_cvb:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error

    ; Check type and convert
    movzx ecx, byte [rax]
    cmp cl, VTAG_BOOLEAN
    je .push_it
    cmp cl, VTAG_INTEGER
    je .from_int
    cmp cl, VTAG_NULL
    je .push_false
    jmp .push_true          ; Non-null values are truthy

.from_int:
    mov rdi, [rax + VALUE_NUM_OFF]
    test rdi, rdi
    jz .push_false
    jmp .push_true

.push_it:
    mov rdi, [rax + VALUE_NUM_OFF]
    call stack_data_push_boolean
    jmp .done

.push_false:
    xor edi, edi
    call stack_data_push_boolean
    jmp .done

.push_true:
    mov edi, 1
    call stack_data_push_boolean
    jmp .done

.error:
    call stack_data_push_null

.done:
    pop rbp
    ret

; impl_cve - Convert to entity (push to entity stack)
; Stack: entity --
impl_cve:
    ; This is essentially entitypush
    jmp op_entitypush

;-----------------------------------------------------------------------------
; Additional operator implementations
;-----------------------------------------------------------------------------

; impl_addat - Add element at index: ( array element index -- )
impl_addat:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    ; Pop index
    call stack_data_pop
    test rax, rax
    jz .done
    mov r13, [rax + VALUE_NUM_OFF]  ; index

    ; Pop element
    call stack_data_pop
    test rax, rax
    jz .done
    mov r12, rax                    ; element

    ; Pop array
    call stack_data_pop
    test rax, rax
    jz .done
    mov rbx, [rax + VALUE_PTR_OFF]  ; array ptr

    ; Insert element at index
    mov rdi, rbx
    mov rsi, r12
    mov rdx, r13
    call array_insert

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

; impl_removeat - Remove element at index: ( array index -- element )
impl_removeat:
    push rbp
    mov rbp, rsp
    push rbx

    ; Pop index
    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_NUM_OFF]  ; index

    ; Pop array
    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_PTR_OFF]
    test rdi, rdi
    jz .error

    mov rsi, rbx
    call array_remove
    test rax, rax
    jz .error

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_remove - Remove element from array: ( array element -- )
impl_remove:
    push rbp
    mov rbp, rsp
    push rbx

    ; Pop element
    call stack_data_pop
    test rax, rax
    jz .done
    mov rbx, rax

    ; Pop array
    call stack_data_pop
    test rax, rax
    jz .done

    mov rdi, [rax + VALUE_PTR_OFF]
    test rdi, rdi
    jz .done

    mov rsi, rbx
    call array_remove_element

.done:
    pop rbx
    pop rbp
    ret

; impl_copy - Copy array: ( array -- newarray )
impl_copy:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_PTR_OFF]
    test rdi, rdi
    jz .error

    call array_copy
    test rax, rax
    jz .error

    ; Create Value for new array
    push rax
    mov edi, VALUE_SIZE
    call heap_alloc
    pop rbx
    test rax, rax
    jz .error

    mov byte [rax], VTAG_ARRAY
    mov qword [rax + VALUE_NUM_OFF], 0
    mov [rax + VALUE_PTR_OFF], rbx

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_copyelements - Copy elements from one array to another
; Stack: destarray srcarray -- destarray
impl_copyelements:
    push rbp
    mov rbp, rsp
    push rbx

    ; Pop source array
    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_PTR_OFF]

    ; Pop destination array
    call stack_data_pop
    test rax, rax
    jz .error
    push rax                        ; Save dest Value

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call array_copy_elements

    ; Push destination back
    pop rdi
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_sortarray - Sort array: ( array -- array )
impl_sortarray:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error
    push rax                        ; Save Value

    mov rdi, [rax + VALUE_PTR_OFF]
    test rdi, rdi
    jz .error_pop

    call array_sort

    pop rdi
    call stack_data_push
    jmp .done

.error_pop:
    pop rax
.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_sortentities - Sort entities by attribute
; Stack: array attrname -- array
impl_sortentities:
    push rbp
    mov rbp, rsp
    push rbx

    ; Pop attribute name
    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_PTR_OFF]

    ; Pop array
    call stack_data_pop
    test rax, rax
    jz .error
    push rax

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call array_sort_entities

    pop rdi
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_add_no_dups - Add element if not already in array
; Stack: array element -- array
impl_add_no_dups:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    ; Pop element
    call stack_data_pop
    test rax, rax
    jz .error
    mov r12, rax

    ; Pop array
    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, rax

    ; Check if element exists
    mov rdi, [rbx + VALUE_PTR_OFF]
    mov rsi, r12
    call array_contains
    test eax, eax
    jnz .push_array                 ; Already exists, don't add

    ; Add element
    mov rdi, [rbx + VALUE_PTR_OFF]
    mov rsi, r12
    call array_push

.push_array:
    mov rdi, rbx
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop r12
    pop rbx
    pop rbp
    ret

; impl_merge - Merge two arrays: ( array1 array2 -- array1 )
impl_merge:
    push rbp
    mov rbp, rsp
    push rbx

    ; Pop array2
    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_PTR_OFF]

    ; Pop array1
    call stack_data_pop
    test rax, rax
    jz .error
    push rax

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call array_merge

    pop rdi
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_randomize - Randomize array order: ( array -- array )
impl_randomize:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error
    push rax

    mov rdi, [rax + VALUE_PTR_OFF]
    call array_randomize

    pop rdi
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_intersection - Get common elements: ( array1 array2 -- array )
impl_intersection:
    push rbp
    mov rbp, rsp
    push rbx

    ; Pop array2
    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_PTR_OFF]

    ; Pop array1
    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call array_intersection
    test rax, rax
    jz .error

    ; Create Value for result
    push rax
    mov edi, VALUE_SIZE
    call heap_alloc
    pop rbx
    test rax, rax
    jz .error

    mov byte [rax], VTAG_ARRAY
    mov qword [rax + VALUE_NUM_OFF], 0
    mov [rax + VALUE_PTR_OFF], rbx

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_intersects - Check if arrays have common elements: ( array1 array2 -- bool )
impl_intersects:
    push rbp
    mov rbp, rsp
    push rbx

    ; Pop array2
    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_PTR_OFF]

    ; Pop array1
    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call array_intersects

    mov edi, eax
    call stack_data_push_boolean
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_addarray - Add all elements from one array to another
; Stack: destarray srcarray -- destarray
impl_addarray:
    ; Same as merge
    jmp impl_merge

; impl_deepcopy - Deep copy array: ( array -- newarray )
impl_deepcopy:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_PTR_OFF]
    call array_deep_copy
    test rax, rax
    jz .error

    push rax
    mov edi, VALUE_SIZE
    call heap_alloc
    pop rbx
    test rax, rax
    jz .error

    mov byte [rax], VTAG_ARRAY
    mov qword [rax + VALUE_NUM_OFF], 0
    mov [rax + VALUE_PTR_OFF], rbx

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_tokenize - Split string into array of tokens
; Stack: string delimiter -- array
impl_tokenize:
    push rbp
    mov rbp, rsp
    push rbx

    ; Pop delimiter
    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_PTR_OFF]

    ; Pop string
    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call string_tokenize
    test rax, rax
    jz .error

    push rax
    mov edi, VALUE_SIZE
    call heap_alloc
    pop rbx
    test rax, rax
    jz .error

    mov byte [rax], VTAG_ARRAY
    mov qword [rax + VALUE_NUM_OFF], 0
    mov [rax + VALUE_PTR_OFF], rbx

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_findmatch - Find entity matching criteria in array
; Stack: array criteria -- entity
impl_findmatch:
    push rbp
    mov rbp, rsp
    push rbx

    ; Pop criteria (table)
    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_PTR_OFF]

    ; Pop array
    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call array_find_match
    test rax, rax
    jz .not_found

    mov rdi, rax
    call stack_data_push
    jmp .done

.not_found:
    call stack_data_push_null
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_cleararray - Clear all elements: ( array -- array )
impl_cleararray:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error
    push rax

    mov rdi, [rax + VALUE_PTR_OFF]
    call array_clear

    pop rdi
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_forr - Reverse for loop: ( limit init body -- )
impl_forr:
    ; Similar to op_for but counts down
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    ; Pop body
    call stack_data_pop
    test rax, rax
    jz .done
    mov r13, rax

    ; Pop init
    call stack_data_pop
    test rax, rax
    jz .done
    mov r12, [rax + VALUE_NUM_OFF]  ; counter

    ; Pop limit
    call stack_data_pop
    test rax, rax
    jz .done
    mov rbx, [rax + VALUE_NUM_OFF]  ; limit

.loop:
    cmp r12, rbx
    jl .done

    ; Push counter
    mov rdi, r12
    call stack_data_push_integer

    ; Execute body
    mov rdi, r13
    call vm_execute_value

    dec r12
    jmp .loop

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

; impl_forallr - Reverse forall: iterate array in reverse
impl_forallr:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    ; Pop body
    call stack_data_pop
    test rax, rax
    jz .done
    mov r13, rax

    ; Pop array
    call stack_data_pop
    test rax, rax
    jz .done
    mov rbx, [rax + VALUE_PTR_OFF]

    ; Get array length
    mov rdi, rbx
    call array_length
    mov r12, rax
    dec r12                         ; Start at last index

.loop:
    test r12, r12
    js .done                        ; If negative, we're done

    ; Get element at index
    mov rdi, rbx
    mov rsi, r12
    call array_get
    test rax, rax
    jz .next

    ; Push element
    mov rdi, rax
    call stack_data_push

    ; Execute body
    mov rdi, r13
    call vm_execute_value

.next:
    dec r12
    jmp .loop

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

; impl_doloop - Do-while loop: ( body -- )
impl_doloop:
    push rbp
    mov rbp, rsp
    push rbx

    ; Pop body
    call stack_data_pop
    test rax, rax
    jz .done
    mov rbx, rax

.loop:
    ; Execute body
    mov rdi, rbx
    call vm_execute_value

    ; Pop condition result
    call stack_data_pop
    test rax, rax
    jz .done

    ; Check if true
    movzx ecx, byte [rax]
    cmp cl, VTAG_BOOLEAN
    jne .done
    cmp qword [rax + VALUE_NUM_OFF], 0
    jne .loop

.done:
    pop rbx
    pop rbp
    ret

; impl_throwexception - Throw exception: ( message -- )
impl_throwexception:
    push rbp
    mov rbp, rsp

    ; Pop message
    call stack_data_pop
    ; Set error state
    mov byte [state + State.error], 1
    ; Store message pointer if needed
    test rax, rax
    jz .done
    mov rdi, [rax + VALUE_PTR_OFF]
    mov [state + State.error_msg], rdi

.done:
    pop rbp
    ret

; impl_forfirst - Execute for first matching element
; Stack: array condition -- element (or null)
impl_forfirst:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14

    ; Pop condition body
    call stack_data_pop
    test rax, rax
    jz .not_found
    mov r13, rax

    ; Pop array
    call stack_data_pop
    test rax, rax
    jz .not_found
    mov rbx, [rax + VALUE_PTR_OFF]

    ; Get array length
    mov rdi, rbx
    call array_length
    mov r14, rax
    xor r12, r12                    ; index = 0

.loop:
    cmp r12, r14
    jge .not_found

    ; Get element
    mov rdi, rbx
    mov rsi, r12
    call array_get
    test rax, rax
    jz .next
    push rax                        ; Save element

    ; Push element for condition
    mov rdi, rax
    call stack_data_push

    ; Execute condition
    mov rdi, r13
    call vm_execute_value

    ; Pop result
    call stack_data_pop
    pop rcx                         ; Restore element
    test rax, rax
    jz .next

    ; Check if true
    movzx edx, byte [rax]
    cmp dl, VTAG_BOOLEAN
    jne .next
    cmp qword [rax + VALUE_NUM_OFF], 0
    je .next

    ; Found! Push element
    mov rdi, rcx
    call stack_data_push
    jmp .done

.next:
    inc r12
    jmp .loop

.not_found:
    call stack_data_push_null
.done:
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

; impl_forfirstelse - Execute for first or else
; Stack: array condition elsebody -- result
impl_forfirstelse:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14
    push r15

    ; Pop else body
    call stack_data_pop
    test rax, rax
    jz .error
    mov r15, rax

    ; Pop condition body
    call stack_data_pop
    test rax, rax
    jz .error
    mov r13, rax

    ; Pop array
    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_PTR_OFF]

    ; Get array length
    mov rdi, rbx
    call array_length
    mov r14, rax
    xor r12, r12

.loop:
    cmp r12, r14
    jge .else_case

    mov rdi, rbx
    mov rsi, r12
    call array_get
    test rax, rax
    jz .next
    push rax

    mov rdi, rax
    call stack_data_push

    mov rdi, r13
    call vm_execute_value

    call stack_data_pop
    pop rcx
    test rax, rax
    jz .next

    movzx edx, byte [rax]
    cmp dl, VTAG_BOOLEAN
    jne .next
    cmp qword [rax + VALUE_NUM_OFF], 0
    je .next

    mov rdi, rcx
    call stack_data_push
    jmp .done

.next:
    inc r12
    jmp .loop

.else_case:
    ; Execute else body
    mov rdi, r15
    call vm_execute_value
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop r15
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

; impl_entityforall - Iterate over all entities of a type
; Stack: typename body --
impl_entityforall:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    ; Pop body
    call stack_data_pop
    test rax, rax
    jz .done
    mov r12, rax

    ; Pop typename
    call stack_data_pop
    test rax, rax
    jz .done
    mov rbx, [rax + VALUE_PTR_OFF]

    ; Get entity list for type
    mov rdi, rbx
    call entity_get_all_of_type
    test rax, rax
    jz .done

    push rax                        ; Save array
    mov rdi, rax
    call array_length
    mov rbx, rax
    pop rdi
    push rdi                        ; Array back on stack

    xor ecx, ecx                    ; index = 0
.loop:
    cmp rcx, rbx
    jge .cleanup

    push rcx
    pop rsi
    mov rdi, [rsp]                  ; Get array
    call array_get
    test rax, rax
    jz .next_iter

    mov rdi, rax
    call stack_data_push

    mov rdi, r12
    call vm_execute_value

.next_iter:
    pop rcx
    inc rcx
    push rcx
    jmp .loop

.cleanup:
    add rsp, 8                      ; Clean array from stack
.done:
    pop r12
    pop rbx
    pop rbp
    ret

; impl_ignore - Pop and discard value: ( value -- )
impl_ignore:
    call stack_data_pop
    ret

; impl_executetable - Execute decision table by name
; Stack: tablename --
impl_executetable:
    push rbp
    mov rbp, rsp

    ; Pop table name
    call stack_data_pop
    test rax, rax
    jz .done

    mov rdi, [rax + VALUE_PTR_OFF]
    call table_execute_by_name

.done:
    pop rbp
    ret

; impl_newdate - Create date from components
; Stack: year month day -- date
impl_newdate:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    ; Pop day
    call stack_data_pop
    test rax, rax
    jz .error
    mov r13, [rax + VALUE_NUM_OFF]

    ; Pop month
    call stack_data_pop
    test rax, rax
    jz .error
    mov r12, [rax + VALUE_NUM_OFF]

    ; Pop year
    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_NUM_OFF]

    ; Create date value
    mov rdi, rbx
    mov rsi, r12
    mov rdx, r13
    call date_create

    ; Create Value for date
    push rax
    mov edi, VALUE_SIZE
    call heap_alloc
    pop rbx
    test rax, rax
    jz .error

    mov byte [rax], VTAG_DATE
    mov [rax + VALUE_NUM_OFF], rbx
    mov qword [rax + VALUE_PTR_OFF], 0

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

; impl_getyear - Get year from date: ( date -- year )
impl_getyear:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_NUM_OFF]
    call date_get_year

    mov rdi, rax
    call stack_data_push_integer
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_getmonth - Get month from date: ( date -- month )
impl_getmonth:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_NUM_OFF]
    call date_get_month

    mov rdi, rax
    call stack_data_push_integer
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_getday - Get day from date: ( date -- day )
impl_getday:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_NUM_OFF]
    call date_get_day

    mov rdi, rax
    call stack_data_push_integer
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_adddays - Add days to date: ( date days -- date )
impl_adddays:
    push rbp
    mov rbp, rsp
    push rbx

    ; Pop days
    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_NUM_OFF]

    ; Pop date
    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, rbx
    call date_add_days

    push rax
    mov edi, VALUE_SIZE
    call heap_alloc
    pop rbx
    test rax, rax
    jz .error

    mov byte [rax], VTAG_DATE
    mov [rax + VALUE_NUM_OFF], rbx
    mov qword [rax + VALUE_PTR_OFF], 0

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_addmonths - Add months to date: ( date months -- date )
impl_addmonths:
    push rbp
    mov rbp, rsp
    push rbx

    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_NUM_OFF]

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, rbx
    call date_add_months

    push rax
    mov edi, VALUE_SIZE
    call heap_alloc
    pop rbx
    test rax, rax
    jz .error

    mov byte [rax], VTAG_DATE
    mov [rax + VALUE_NUM_OFF], rbx
    mov qword [rax + VALUE_PTR_OFF], 0

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_addyears - Add years to date: ( date years -- date )
impl_addyears:
    push rbp
    mov rbp, rsp
    push rbx

    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_NUM_OFF]

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, rbx
    call date_add_years

    push rax
    mov edi, VALUE_SIZE
    call heap_alloc
    pop rbx
    test rax, rax
    jz .error

    mov byte [rax], VTAG_DATE
    mov [rax + VALUE_NUM_OFF], rbx
    mov qword [rax + VALUE_PTR_OFF], 0

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_daysbetween - Days between two dates: ( date1 date2 -- days )
impl_daysbetween:
    push rbp
    mov rbp, rsp
    push rbx

    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_NUM_OFF]

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, rbx
    call date_days_between

    mov rdi, rax
    call stack_data_push_integer
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_monthsbetween - Months between two dates: ( date1 date2 -- months )
impl_monthsbetween:
    push rbp
    mov rbp, rsp
    push rbx

    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_NUM_OFF]

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, rbx
    call date_months_between

    mov rdi, rax
    call stack_data_push_integer
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_yearsbetween - Years between two dates: ( date1 date2 -- years )
impl_yearsbetween:
    push rbp
    mov rbp, rsp
    push rbx

    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_NUM_OFF]

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, rbx
    call date_years_between

    mov rdi, rax
    call stack_data_push_integer
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_datecmp - Compare two dates: ( date1 date2 -- int )
impl_datecmp:
    push rbp
    mov rbp, rsp
    push rbx

    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_NUM_OFF]

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, rbx
    call date_compare

    mov rdi, rax
    call stack_data_push_integer
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_firstofmonth - Get first day of month: ( date -- date )
impl_firstofmonth:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_NUM_OFF]
    call date_first_of_month

    push rax
    mov edi, VALUE_SIZE
    call heap_alloc
    pop rbx
    test rax, rax
    jz .error

    mov byte [rax], VTAG_DATE
    mov [rax + VALUE_NUM_OFF], rbx
    mov qword [rax + VALUE_PTR_OFF], 0

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_firstofyear - Get first day of year: ( date -- date )
impl_firstofyear:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_NUM_OFF]
    call date_first_of_year

    push rax
    mov edi, VALUE_SIZE
    call heap_alloc
    pop rbx
    test rax, rax
    jz .error

    mov byte [rax], VTAG_DATE
    mov [rax + VALUE_NUM_OFF], rbx
    mov qword [rax + VALUE_PTR_OFF], 0

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_endofmonth - Get last day of month: ( date -- date )
impl_endofmonth:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_NUM_OFF]
    call date_end_of_month

    push rax
    mov edi, VALUE_SIZE
    call heap_alloc
    pop rbx
    test rax, rax
    jz .error

    mov byte [rax], VTAG_DATE
    mov [rax + VALUE_NUM_OFF], rbx
    mov qword [rax + VALUE_PTR_OFF], 0

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_yearof - Get year from date: ( date -- year )
impl_yearof:
    jmp impl_getyear

; impl_monthof - Get month from date: ( date -- month )
impl_monthof:
    jmp impl_getmonth

; impl_dayof - Get day from date: ( date -- day )
impl_dayof:
    jmp impl_getday

; impl_getdaysinyear - Get days in year: ( date -- days )
impl_getdaysinyear:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_NUM_OFF]
    call date_days_in_year

    mov rdi, rax
    call stack_data_push_integer
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_getdaysinmonth - Get days in month: ( date -- days )
impl_getdaysinmonth:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_NUM_OFF]
    call date_days_in_month

    mov rdi, rax
    call stack_data_push_integer
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_getdayofmonth - Get day of month: ( date -- day )
impl_getdayofmonth:
    jmp impl_getday

; impl_datelt - Date less than: ( date1 date2 -- bool )
impl_datelt:
    push rbp
    mov rbp, rsp
    push rbx

    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_NUM_OFF]

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, rbx
    call date_compare

    cmp rax, 0
    setl al
    movzx edi, al
    call stack_data_push_boolean
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_dategt - Date greater than: ( date1 date2 -- bool )
impl_dategt:
    push rbp
    mov rbp, rsp
    push rbx

    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_NUM_OFF]

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, rbx
    call date_compare

    cmp rax, 0
    setg al
    movzx edi, al
    call stack_data_push_boolean
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_dateeq - Date equals: ( date1 date2 -- bool )
impl_dateeq:
    push rbp
    mov rbp, rsp
    push rbx

    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_NUM_OFF]

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, rbx
    call date_compare

    test rax, rax
    setz al
    movzx edi, al
    call stack_data_push_boolean
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_dateplus - Add days to date: ( date days -- date )
impl_dateplus:
    jmp impl_adddays

; impl_dateminus - Subtract days or dates: ( date days/date -- date/int )
impl_dateminus:
    push rbp
    mov rbp, rsp
    push rbx

    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, rax

    call stack_data_pop
    test rax, rax
    jz .error

    ; Check if second operand is date or integer
    movzx ecx, byte [rbx]
    cmp cl, VTAG_DATE
    je .date_diff

    ; Subtract days
    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, [rbx + VALUE_NUM_OFF]
    neg rsi
    call date_add_days

    push rax
    mov edi, VALUE_SIZE
    call heap_alloc
    pop rbx
    test rax, rax
    jz .error

    mov byte [rax], VTAG_DATE
    mov [rax + VALUE_NUM_OFF], rbx
    mov qword [rax + VALUE_PTR_OFF], 0
    mov rdi, rax
    call stack_data_push
    jmp .done

.date_diff:
    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, [rbx + VALUE_NUM_OFF]
    call date_days_between
    mov rdi, rax
    call stack_data_push_integer
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_getdate - Get date component: ( date -- date )
impl_getdate:
    ; For dates, just return the date
    ret

; impl_gettimestamp - Get timestamp: ( -- timestamp )
impl_gettimestamp:
    push rbp
    mov rbp, rsp

    call time_get_timestamp

    mov rdi, rax
    call stack_data_push_integer

    pop rbp
    ret

; impl_incontext - Check if in entity context
; Stack: entitytype -- bool
impl_incontext:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .false

    mov rdi, [rax + VALUE_PTR_OFF]
    call stack_entity_has_type

    mov edi, eax
    call stack_data_push_boolean
    jmp .done

.false:
    xor edi, edi
    call stack_data_push_boolean
.done:
    pop rbp
    ret

; impl_entityname - Get entity type name: ( entity -- name )
impl_entityname:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_PTR_OFF]
    call entity_get_type_name
    test rax, rax
    jz .error

    ; Create name Value
    push rax
    mov edi, VALUE_SIZE
    call heap_alloc
    pop rbx
    test rax, rax
    jz .error

    mov byte [rax], VTAG_NAME
    mov qword [rax + VALUE_NUM_OFF], 0
    mov [rax + VALUE_PTR_OFF], rbx

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_entityid - Get entity ID: ( entity -- id )
impl_entityid:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_PTR_OFF]
    call entity_get_id

    mov rdi, rax
    call stack_data_push_integer
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_roundto - Round to decimal places: ( num places -- num )
impl_roundto:
    push rbp
    mov rbp, rsp
    push rbx

    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_NUM_OFF]  ; places

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, rbx
    call math_round_to

    push rax
    mov edi, VALUE_SIZE
    call heap_alloc
    pop rbx
    test rax, rax
    jz .error

    mov byte [rax], VTAG_DOUBLE
    mov [rax + VALUE_NUM_OFF], rbx
    mov qword [rax + VALUE_PTR_OFF], 0

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_arraytomark - Collect elements to array: ( mark e1 e2 ... en -- array )
impl_arraytomark:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    ; Create new array
    xor edi, edi
    call array_alloc
    test rax, rax
    jz .error
    mov rbx, rax

    ; Count and collect elements
.loop:
    call stack_data_pop
    test rax, rax
    jz .done_collecting

    ; Check for mark
    movzx ecx, byte [rax]
    cmp cl, VTAG_MARK
    je .done_collecting

    ; Add to array (prepend since we're popping in reverse)
    mov rdi, rbx
    mov rsi, rax
    xor edx, edx                    ; index 0
    call array_insert
    jmp .loop

.done_collecting:
    ; Create Value for array
    mov edi, VALUE_SIZE
    call heap_alloc
    test rax, rax
    jz .error

    mov byte [rax], VTAG_ARRAY
    mov qword [rax + VALUE_NUM_OFF], 0
    mov [rax + VALUE_PTR_OFF], rbx

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop r12
    pop rbx
    pop rbp
    ret

; impl_counttomark - Count elements to mark: ( mark e1 ... en -- mark e1 ... en n )
impl_counttomark:
    push rbp
    mov rbp, rsp

    call stack_data_count_to_mark
    mov rdi, rax
    call stack_data_push_integer

    pop rbp
    ret

; impl_cleartomark - Clear stack to mark: ( mark e1 ... en -- )
impl_cleartomark:
    push rbp
    mov rbp, rsp

.loop:
    call stack_data_pop
    test rax, rax
    jz .done

    movzx ecx, byte [rax]
    cmp cl, VTAG_MARK
    jne .loop

.done:
    pop rbp
    ret

; impl_traceon - Enable tracing
impl_traceon:
    mov byte [state + State.trace], 1
    ret

; impl_traceoff - Disable tracing
impl_traceoff:
    mov byte [state + State.trace], 0
    ret

; impl_setdebug - Set debug level: ( level -- )
impl_setdebug:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .done

    mov rcx, [rax + VALUE_NUM_OFF]
    mov [state + State.debug_level], rcx

.done:
    pop rbp
    ret

; impl_pstack - Print stack
impl_pstack:
    call stack_data_print
    ret

; impl_printtos - Print top of stack: ( value -- )
impl_printtos:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .done

    mov rdi, rax
    call value_print

.done:
    pop rbp
    ret

; impl_clone - Clone value: ( value -- value copy )
impl_clone:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error

    push rax                        ; Save original
    mov rdi, rax
    call value_clone
    pop rdi                         ; Original
    test rax, rax
    jz .push_original

    push rax                        ; Save clone
    call stack_data_push            ; Push original
    pop rdi
    call stack_data_push            ; Push clone
    jmp .done

.push_original:
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_tor - Move to return stack: ( value -- ) R: ( -- value )
impl_tor:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .done

    mov rdi, rax
    call stack_return_push

.done:
    pop rbp
    ret

; impl_fromr - Move from return stack: ( -- value ) R: ( value -- )
impl_fromr:
    push rbp
    mov rbp, rsp

    call stack_return_pop
    test rax, rax
    jz .error

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_i - Get loop index: ( -- index )
impl_i:
    push rbp
    mov rbp, rsp

    xor edi, edi                    ; depth 0
    call stack_return_peek
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_NUM_OFF]
    call stack_data_push_integer
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_j - Get outer loop index: ( -- index )
impl_j:
    push rbp
    mov rbp, rsp

    mov edi, 1                      ; depth 1
    call stack_return_peek
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_NUM_OFF]
    call stack_data_push_integer
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_k - Get outermost loop index: ( -- index )
impl_k:
    push rbp
    mov rbp, rsp

    mov edi, 2                      ; depth 2
    call stack_return_peek
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_NUM_OFF]
    call stack_data_push_integer
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_entityfetch - Fetch entity by index: ( index -- entity )
impl_entityfetch:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_NUM_OFF]
    call stack_entity_fetch
    test rax, rax
    jz .error

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_find - Find entity by criteria: ( table -- entity )
impl_find:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_PTR_OFF]
    call stack_entity_find
    test rax, rax
    jz .not_found

    mov rdi, rax
    call stack_data_push
    jmp .done

.not_found:
    call stack_data_push_null
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_findcreateentity - Find or create entity
; Stack: typename criteria -- entity
impl_findcreateentity:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    ; Pop criteria
    call stack_data_pop
    test rax, rax
    jz .error
    mov r12, [rax + VALUE_PTR_OFF]

    ; Pop typename
    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_PTR_OFF]

    ; Try to find
    mov rdi, r12
    call stack_entity_find
    test rax, rax
    jnz .found

    ; Create new entity
    mov rdi, rbx
    call entity_alloc
    test rax, rax
    jz .error

    ; Create Value for entity
    push rax
    mov edi, VALUE_SIZE
    call heap_alloc
    pop rbx
    test rax, rax
    jz .error

    mov byte [rax], VTAG_ENTITY
    mov qword [rax + VALUE_NUM_OFF], 0
    mov [rax + VALUE_PTR_OFF], rbx

.found:
    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop r12
    pop rbx
    pop rbp
    ret

; impl_cvr - Convert to real/double: ( value -- double )
impl_cvr:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error

    movzx ecx, byte [rax]
    cmp cl, VTAG_DOUBLE
    je .push_it
    cmp cl, VTAG_INTEGER
    je .from_int
    jmp .error

.from_int:
    ; Convert integer to double
    mov rdi, [rax + VALUE_NUM_OFF]
    cvtsi2sd xmm0, rdi
    movq rdi, xmm0
    jmp .push_double

.push_it:
    mov rdi, [rax + VALUE_NUM_OFF]
.push_double:
    push rdi
    mov edi, VALUE_SIZE
    call heap_alloc
    pop rbx
    test rax, rax
    jz .error

    mov byte [rax], VTAG_DOUBLE
    mov [rax + VALUE_NUM_OFF], rbx
    mov qword [rax + VALUE_PTR_OFF], 0

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_cvn - Convert to name: ( value -- name )
impl_cvn:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error

    movzx ecx, byte [rax]
    cmp cl, VTAG_NAME
    je .push_it
    cmp cl, VTAG_STRING
    je .from_string
    jmp .error

.from_string:
    ; Convert string to name
    mov rdi, [rax + VALUE_PTR_OFF]
    ; String is already suitable for name
    jmp .create_name

.push_it:
    mov rdi, rax
    call stack_data_push
    jmp .done

.create_name:
    push rdi
    mov edi, VALUE_SIZE
    call heap_alloc
    pop rbx
    test rax, rax
    jz .error

    mov byte [rax], VTAG_NAME
    mov qword [rax + VALUE_NUM_OFF], 0
    mov [rax + VALUE_PTR_OFF], rbx

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_cvd - Convert to date: ( value -- date )
impl_cvd:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .error

    movzx ecx, byte [rax]
    cmp cl, VTAG_DATE
    je .push_it
    cmp cl, VTAG_STRING
    je .from_string
    cmp cl, VTAG_INTEGER
    je .from_int
    jmp .error

.from_string:
    mov rdi, [rax + VALUE_PTR_OFF]
    call date_parse
    jmp .create_date

.from_int:
    mov rdi, [rax + VALUE_NUM_OFF]
    jmp .create_date

.push_it:
    mov rdi, rax
    call stack_data_push
    jmp .done

.create_date:
    push rax
    mov edi, VALUE_SIZE
    call heap_alloc
    pop rbx
    test rax, rax
    jz .error

    mov byte [rax], VTAG_DATE
    mov [rax + VALUE_NUM_OFF], rbx
    mov qword [rax + VALUE_PTR_OFF], 0

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_error - Create error: ( message -- error )
impl_error:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .done

    ; Set error state
    mov byte [state + State.error], 1
    mov rdi, [rax + VALUE_PTR_OFF]
    mov [state + State.error_msg], rdi

.done:
    pop rbp
    ret

; policy_get_statements - Stub for policy statements
; Returns: rax = array pointer (or 0 for empty)
; Currently returns 0 (empty) - full implementation would track policy execution
global policy_get_statements
policy_get_statements:
    xor eax, eax
    ret

; impl_policystatements - Get policy statements
; Stack: -- array
impl_policystatements:
    push rbp
    mov rbp, rsp

    call policy_get_statements
    test rax, rax
    jz .empty

    mov rdi, rax
    call stack_data_push
    jmp .done

.empty:
    ; Return empty array
    xor edi, edi
    call array_alloc
    test rax, rax
    jz .error

    push rax
    mov edi, VALUE_SIZE
    call heap_alloc
    pop rbx
    test rax, rax
    jz .error

    mov byte [rax], VTAG_ARRAY
    mov qword [rax + VALUE_NUM_OFF], 0
    mov [rax + VALUE_PTR_OFF], rbx

    mov rdi, rax
    call stack_data_push
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbp
    ret

; impl_regexmatch - Regex match: ( string pattern -- bool )
impl_regexmatch:
    push rbp
    mov rbp, rsp
    push rbx

    ; Pop pattern
    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_PTR_OFF]

    ; Pop string
    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call string_regex_match

    mov edi, eax
    call stack_data_push_boolean
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_streqi - Case-insensitive string equality
; Stack: str1 str2 -- bool
impl_streqi:
    push rbp
    mov rbp, rsp
    push rbx

    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_PTR_OFF]

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call string_equals_ignore_case

    mov edi, eax
    call stack_data_push_boolean
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_strlt - String less than: ( str1 str2 -- bool )
impl_strlt:
    push rbp
    mov rbp, rsp
    push rbx

    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_PTR_OFF]

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call string_compare

    cmp rax, 0
    setl al
    movzx edi, al
    call stack_data_push_boolean
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_strgt - String greater than: ( str1 str2 -- bool )
impl_strgt:
    push rbp
    mov rbp, rsp
    push rbx

    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_PTR_OFF]

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call string_compare

    cmp rax, 0
    setg al
    movzx edi, al
    call stack_data_push_boolean
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_strle - String less than or equal: ( str1 str2 -- bool )
impl_strle:
    push rbp
    mov rbp, rsp
    push rbx

    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_PTR_OFF]

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call string_compare

    cmp rax, 0
    setle al
    movzx edi, al
    call stack_data_push_boolean
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_strge - String greater than or equal: ( str1 str2 -- bool )
impl_strge:
    push rbp
    mov rbp, rsp
    push rbx

    call stack_data_pop
    test rax, rax
    jz .error
    mov rbx, [rax + VALUE_PTR_OFF]

    call stack_data_pop
    test rax, rax
    jz .error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call string_compare

    cmp rax, 0
    setge al
    movzx edi, al
    call stack_data_push_boolean
    jmp .done

.error:
    call stack_data_push_null
.done:
    pop rbx
    pop rbp
    ret

; impl_tablekeys - Get table keys: ( table -- array )
impl_tablekeys:
    push rbp
    mov rbp, rsp
    push rbx

    call stack_data_pop
    test rax, rax
    jz .tablekeys_error

    mov rdi, [rax + VALUE_PTR_OFF]
    call hashtable_keys
    test rax, rax
    jz .tablekeys_error

    mov rbx, rax            ; Save array pointer

    mov edi, VALUE_SIZE
    call heap_alloc
    test rax, rax
    jz .tablekeys_error

    mov byte [rax], VTAG_ARRAY
    mov qword [rax + VALUE_NUM_OFF], 0
    mov [rax + VALUE_PTR_OFF], rbx

    mov rdi, rax
    call stack_data_push
    jmp .tablekeys_done

.tablekeys_error:
    call stack_data_push_null
.tablekeys_done:
    pop rbx
    pop rbp
    ret

; impl_tablevalues - Get table values: ( table -- array )
impl_tablevalues:
    push rbp
    mov rbp, rsp
    push rbx

    call stack_data_pop
    test rax, rax
    jz .tablevalues_error

    mov rdi, [rax + VALUE_PTR_OFF]
    call hashtable_values
    test rax, rax
    jz .tablevalues_error

    mov rbx, rax            ; Save array pointer

    mov edi, VALUE_SIZE
    call heap_alloc
    test rax, rax
    jz .tablevalues_error

    mov byte [rax], VTAG_ARRAY
    mov qword [rax + VALUE_NUM_OFF], 0
    mov [rax + VALUE_PTR_OFF], rbx

    mov rdi, rax
    call stack_data_push
    jmp .tablevalues_done

.tablevalues_error:
    call stack_data_push_null
.tablevalues_done:
    pop rbx
    pop rbp
    ret

; impl_tablecontains - Check if table contains key: ( table key -- bool )
impl_tablecontains:
    push rbp
    mov rbp, rsp
    push rbx

    ; Pop key
    call stack_data_pop
    test rax, rax
    jz .tablecontains_error
    mov rbx, [rax + VALUE_PTR_OFF]  ; Key string

    ; Pop table
    call stack_data_pop
    test rax, rax
    jz .tablecontains_error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call hashtable_contains

    mov edi, eax
    call stack_data_push_boolean
    jmp .tablecontains_done

.tablecontains_error:
    call stack_data_push_null
.tablecontains_done:
    pop rbx
    pop rbp
    ret

; impl_tableremove - Remove key from table: ( table key -- value )
impl_tableremove:
    push rbp
    mov rbp, rsp
    push rbx

    ; Pop key
    call stack_data_pop
    test rax, rax
    jz .tableremove_error
    mov rbx, [rax + VALUE_PTR_OFF]  ; Key string

    ; Pop table
    call stack_data_pop
    test rax, rax
    jz .tableremove_error

    mov rdi, [rax + VALUE_PTR_OFF]
    mov rsi, rbx
    call hashtable_remove
    test rax, rax
    jz .tableremove_null

    mov rdi, rax
    call stack_data_push
    jmp .tableremove_done

.tableremove_null:
    call stack_data_push_null
    jmp .tableremove_done

.tableremove_error:
    call stack_data_push_null
.tableremove_done:
    pop rbx
    pop rbp
    ret

; impl_tablesize - Get table size: ( table -- int )
impl_tablesize:
    push rbp
    mov rbp, rsp

    call stack_data_pop
    test rax, rax
    jz .tablesize_error

    mov rdi, [rax + VALUE_PTR_OFF]
    call hashtable_size

    mov rdi, rax
    call stack_data_push_integer
    jmp .tablesize_done

.tablesize_error:
    call stack_data_push_null
.tablesize_done:
    pop rbp
    ret

section .rodata
local_name_0: db 1, '0', 0  ; Length-prefixed "0"
local_name_1: db 1, '1', 0
local_name_2: db 1, '2', 0

section .text

; op_constant - Push constant by index (opcode 201)
; Reads varint index, pushes constant from pool
global op_constant
op_constant:
    push rbp
    mov rbp, rsp

    ; Read constant index from bytecode (rbx is updated by read_varint)
    call read_varint
    mov rcx, rax            ; Save index in rcx (not rbx!)

    ; Check bounds
    cmp rcx, [state + State.constant_count]
    jae .error

    ; Get constant pool pointer
    mov rax, [state + State.constant_pool]
    test rax, rax
    jz .error

    ; Calculate address: base + index * VALUE_SIZE
    imul rcx, VALUE_SIZE
    add rax, rcx

    ; Push the Value to data stack
    mov rdi, rax
    call stack_data_push

    pop rbp
    ret

.error:
    call stack_data_push_null
    pop rbp
    ret

; op_name - Push name by index (opcode 202)
; Reads varint index, creates and pushes name Value
global op_name
op_name:
    push rbp
    mov rbp, rsp
    push r15                ; Save r15 for index

    ; Read name index from bytecode (rbx is updated by read_varint)
    call read_varint
    mov r15, rax            ; Save index in r15 (not rbx!)

    ; Check bounds
    cmp r15, [state + State.name_count]
    jae .error

    ; Get name pool pointer
    mov rax, [state + State.name_pool]
    test rax, rax
    jz .error

    ; Get name pointer at index (use r15 for index)
    mov rcx, [rax + r15 * 8]
    test rcx, rcx
    jz .error

    ; Save name pointer
    mov r15, rcx

    ; Allocate Value for the name
    mov edi, VALUE_SIZE
    call heap_alloc
    test rax, rax
    jz .error

    ; Initialize Value as name
    mov byte [rax], VTAG_NAME
    mov qword [rax + VALUE_NUM_OFF], 0
    mov [rax + VALUE_PTR_OFF], r15

    ; Push to data stack
    mov rdi, rax
    call stack_data_push

    pop r15
    pop rbp
    ret

.error:
    call stack_data_push_null
    pop r15
    pop rbp
    ret

; op_depth - Push stack depth (opcode 222)
global op_depth
op_depth:
    ; Calculate depth = (stack_base - stack_ptr) / VALUE_SIZE
    mov rax, [state + State.data_stack_base]
    sub rax, r12
    mov rcx, VALUE_SIZE
    xor edx, edx
    div rcx
    mov rdi, rax
    call stack_data_push_integer
    ret
