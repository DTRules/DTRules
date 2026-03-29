; test_array.asm - Unit tests for Array operations
; DTRules Assembly Implementation

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state

; Stack functions
extern stack_data_push_integer
extern stack_data_push_double
extern stack_data_pop
extern stack_data_peek
extern stack_data_depth
extern stack_data_clear

; Array creation
extern array_alloc
extern value_new_array

; Array operators to test
extern op_array_new
extern op_array_get
extern op_array_set
extern op_array_len
extern op_array_push
extern op_array_pop
extern op_array_concat
extern op_array_slice
extern op_array_copy
extern op_array_clear
extern op_array_remove
extern op_array_insert
extern op_array_reverse
extern op_array_sort
extern op_array_contains
extern op_array_indexof

; Print functions
extern print_string
extern print_integer
extern print_newline

; Test harness functions
extern test_start
extern test_end_pass
extern test_end_fail
extern test_pass
extern test_fail
extern assert_eq
extern assert_true
extern assert_false
extern print_test_summary
extern reset_state
extern test_count
extern fail_count
extern pass_count

section .data
    test_header:        db "=== Array Operation Tests ===", 10, 0

    ; Test names
    test_new:           db "array_new", 0
    test_get:           db "array_get", 0
    test_set:           db "array_set", 0
    test_len:           db "array_len", 0
    test_push:          db "array_push", 0
    test_pop:           db "array_pop", 0
    test_concat:        db "array_concat", 0
    test_slice:         db "array_slice", 0
    test_copy:          db "array_copy", 0
    test_clear:         db "array_clear", 0
    test_remove:        db "array_remove", 0
    test_insert:        db "array_insert", 0
    test_reverse:       db "array_reverse", 0
    test_sort:          db "array_sort", 0
    test_contains:      db "array_contains", 0
    test_indexof:       db "array_indexof", 0

section .bss
    ; Temporary storage for array pointers
    temp_array1:        resq 1
    temp_array2:        resq 1

section .text
    global test_main

;-----------------------------------------------------------------------------
; Helper: Create and push a new empty array onto the data stack
; Output: rax = array header pointer (also pushed to stack)
;-----------------------------------------------------------------------------
create_and_push_array:
    push rbp
    mov rbp, rsp
    push rbx

    ; Create array Value (this allocates array header and data)
    call value_new_array
    test rax, rax
    jz .fail

    mov rbx, rax                ; Save Value pointer

    ; Get the array header pointer for return value
    mov rax, [rbx + VALUE_PTR_OFF]
    push rax                    ; Save array header pointer

    ; Push Value to data stack
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12

    ; Copy the Value (24 bytes)
    mov rcx, [rbx]
    mov [r12], rcx
    mov rcx, [rbx + 8]
    mov [r12 + 8], rcx
    mov rcx, [rbx + 16]
    mov [r12 + 16], rcx

    pop rax                     ; Return array header pointer
    jmp .done

.fail:
    xor eax, eax

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_main - Run all array tests
;-----------------------------------------------------------------------------
test_main:
    push rbp
    mov rbp, rsp

    ; Print header
    lea rdi, [test_header]
    call print_string

    ; Run tests
    call test_array_new_op
    call test_array_len_op
    call test_array_push_op
    call test_array_pop_op
    call test_array_get_op
    call test_array_set_op
    call test_array_copy_op
    call test_array_clear_op
    call test_array_contains_op
    call test_array_indexof_op
    call test_array_concat_op
    call test_array_slice_op
    call test_array_remove_op
    call test_array_insert_op
    call test_array_reverse_op
    call test_array_sort_op

    ; Print summary
    call print_test_summary

    ; Return fail count as exit status
    mov rax, [fail_count]

    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_array_new_op - Test array creation
;-----------------------------------------------------------------------------
test_array_new_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_new]
    call test_start

    call reset_state

    ; Push initial capacity
    mov rdi, 10
    call stack_data_push_integer

    ; Create new array
    call op_array_new

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Verify result is an array
    call stack_data_pop
    test rax, rax
    jz .fail

    movzx edi, byte [rax + VALUE_TAG_OFF]
    cmp edi, VTAG_ARRAY
    jne .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_array_len_op - Test array length
;-----------------------------------------------------------------------------
test_array_len_op:
    push rbp
    mov rbp, rsp

    lea rdi, [test_len]
    call test_start

    call reset_state

    ; Create empty array
    call create_and_push_array
    test rax, rax
    jz .fail

    ; Get length
    call op_array_len

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Verify length is 0
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, 0
    call assert_eq
    test eax, eax
    jz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_array_push_op - Test array push: push 42 to array
;-----------------------------------------------------------------------------
test_array_push_op:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_push]
    call test_start

    call reset_state

    ; Create empty array
    call create_and_push_array
    test rax, rax
    jz .fail
    mov rbx, rax                ; Save array pointer

    ; Push value 42
    mov rdi, 42
    call stack_data_push_integer

    ; Array push
    call op_array_push

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Verify array length is now 1 by getting the length field
    mov rdi, [rbx]              ; Array length is first field
    mov rsi, 1
    call assert_eq
    test eax, eax
    jz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_array_pop_op - Test array pop
;-----------------------------------------------------------------------------
test_array_pop_op:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_pop]
    call test_start

    call reset_state

    ; Create array and push a value
    call create_and_push_array
    test rax, rax
    jz .fail
    mov rbx, rax

    mov rdi, 99
    call stack_data_push_integer

    call op_array_push

    ; Check no error from push
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Push array again for pop
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    ; Pop from array
    call op_array_pop

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Verify popped value is 99
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, 99
    call assert_eq
    test eax, eax
    jz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_array_get_op - Test array get
;-----------------------------------------------------------------------------
test_array_get_op:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_get]
    call test_start

    call reset_state

    ; Create array and push a value
    call create_and_push_array
    test rax, rax
    jz .fail
    mov rbx, rax

    mov rdi, 77
    call stack_data_push_integer

    call op_array_push

    ; Check no error from push
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Push array for get
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    ; Push index 0
    mov rdi, 0
    call stack_data_push_integer

    ; Get element at index 0
    call op_array_get

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Verify result is 77
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, 77
    call assert_eq
    test eax, eax
    jz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_array_set_op - Test array set
;-----------------------------------------------------------------------------
test_array_set_op:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_set]
    call test_start

    call reset_state

    ; Create array and push initial value
    call create_and_push_array
    test rax, rax
    jz .fail
    mov rbx, rax

    mov rdi, 10
    call stack_data_push_integer

    call op_array_push

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Push array for set
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    ; Push index 0
    mov rdi, 0
    call stack_data_push_integer

    ; Push new value 20
    mov rdi, 20
    call stack_data_push_integer

    ; Set element at index 0
    call op_array_set

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Verify by getting the element back
    ; Push array for get
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    mov rdi, 0
    call stack_data_push_integer

    call op_array_get

    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, 20
    call assert_eq
    test eax, eax
    jz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_array_copy_op - Test array copy
;-----------------------------------------------------------------------------
test_array_copy_op:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_copy]
    call test_start

    call reset_state

    ; Create array and push some values
    call create_and_push_array
    test rax, rax
    jz .fail
    mov rbx, rax

    mov rdi, 1
    call stack_data_push_integer
    call op_array_push

    ; Push array again
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    mov rdi, 2
    call stack_data_push_integer
    call op_array_push

    ; Push array again
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    mov rdi, 3
    call stack_data_push_integer
    call op_array_push

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Push array for copy
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    ; Copy
    call op_array_copy

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Verify copy is an array with same length
    call stack_data_pop
    test rax, rax
    jz .fail

    movzx edi, byte [rax + VALUE_TAG_OFF]
    cmp edi, VTAG_ARRAY
    jne .fail

    ; Get the copy's array pointer and check length
    mov rdi, [rax + VALUE_PTR_OFF]
    test rdi, rdi
    jz .fail

    mov rsi, [rdi]              ; Get length
    cmp rsi, 3
    jne .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_array_clear_op - Test array clear
;-----------------------------------------------------------------------------
test_array_clear_op:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_clear]
    call test_start

    call reset_state

    ; Create array and push a value
    call create_and_push_array
    test rax, rax
    jz .fail
    mov rbx, rax

    mov rdi, 42
    call stack_data_push_integer
    call op_array_push

    ; Push array for clear
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    ; Clear
    call op_array_clear

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Verify length is now 0
    mov rdi, [rbx]
    mov rsi, 0
    call assert_eq
    test eax, eax
    jz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_array_contains_op - Test array contains
;-----------------------------------------------------------------------------
test_array_contains_op:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_contains]
    call test_start

    call reset_state

    ; Create array and push values
    call create_and_push_array
    test rax, rax
    jz .fail
    mov rbx, rax

    mov rdi, 10
    call stack_data_push_integer
    call op_array_push

    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    mov rdi, 20
    call stack_data_push_integer
    call op_array_push

    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    mov rdi, 30
    call stack_data_push_integer
    call op_array_push

    ; Push array for contains
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    ; Push value to search for
    mov rdi, 20
    call stack_data_push_integer

    ; Contains
    call op_array_contains

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Verify result is true
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    call assert_true
    test eax, eax
    jz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_array_indexof_op - Test array indexOf
;-----------------------------------------------------------------------------
test_array_indexof_op:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_indexof]
    call test_start

    call reset_state

    ; Create array and push values [10, 20, 30]
    call create_and_push_array
    test rax, rax
    jz .fail
    mov rbx, rax

    mov rdi, 10
    call stack_data_push_integer
    call op_array_push

    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    mov rdi, 20
    call stack_data_push_integer
    call op_array_push

    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    mov rdi, 30
    call stack_data_push_integer
    call op_array_push

    ; Push array for indexOf
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    ; Push value to search for (20)
    mov rdi, 20
    call stack_data_push_integer

    ; IndexOf
    call op_array_indexof

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Verify result is 1
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, 1
    call assert_eq
    test eax, eax
    jz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_array_concat_op - Test array concatenation
;-----------------------------------------------------------------------------
test_array_concat_op:
    push rbp
    mov rbp, rsp
    push rbx
    push r13

    lea rdi, [test_concat]
    call test_start

    call reset_state

    ; Create first array [1, 2]
    call create_and_push_array
    test rax, rax
    jz .fail
    mov rbx, rax

    mov rdi, 1
    call stack_data_push_integer
    call op_array_push

    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    mov rdi, 2
    call stack_data_push_integer
    call op_array_push

    ; Create second array [3, 4]
    call create_and_push_array
    test rax, rax
    jz .fail
    mov r13, rax

    mov rdi, 3
    call stack_data_push_integer
    call op_array_push

    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], r13

    mov rdi, 4
    call stack_data_push_integer
    call op_array_push

    ; Push first array
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    ; Push second array
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], r13

    ; Concat
    call op_array_concat

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Verify result has length 4
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_PTR_OFF]
    test rdi, rdi
    jz .fail

    mov rsi, [rdi]
    cmp rsi, 4
    jne .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop r13
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_array_slice_op - Test array slice
;-----------------------------------------------------------------------------
test_array_slice_op:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_slice]
    call test_start

    call reset_state

    ; Create array [10, 20, 30, 40, 50]
    call create_and_push_array
    test rax, rax
    jz .fail
    mov rbx, rax

    mov rdi, 10
    call stack_data_push_integer
    call op_array_push

    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    mov rdi, 20
    call stack_data_push_integer
    call op_array_push

    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    mov rdi, 30
    call stack_data_push_integer
    call op_array_push

    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    mov rdi, 40
    call stack_data_push_integer
    call op_array_push

    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    mov rdi, 50
    call stack_data_push_integer
    call op_array_push

    ; Push array for slice
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    ; Push start index 1
    mov rdi, 1
    call stack_data_push_integer

    ; Push end index 4
    mov rdi, 4
    call stack_data_push_integer

    ; Slice [1:4] -> [20, 30, 40]
    call op_array_slice

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Verify result has length 3
    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_PTR_OFF]
    test rdi, rdi
    jz .fail

    mov rsi, [rdi]
    cmp rsi, 3
    jne .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_array_remove_op - Test array remove
;-----------------------------------------------------------------------------
test_array_remove_op:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_remove]
    call test_start

    call reset_state

    ; Create array [10, 20, 30]
    call create_and_push_array
    test rax, rax
    jz .fail
    mov rbx, rax

    mov rdi, 10
    call stack_data_push_integer
    call op_array_push

    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    mov rdi, 20
    call stack_data_push_integer
    call op_array_push

    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    mov rdi, 30
    call stack_data_push_integer
    call op_array_push

    ; Push array for remove
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    ; Push index 1 (remove 20)
    mov rdi, 1
    call stack_data_push_integer

    ; Remove
    call op_array_remove

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Verify length is now 2
    mov rdi, [rbx]
    mov rsi, 2
    call assert_eq
    test eax, eax
    jz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_array_insert_op - Test array insert
;-----------------------------------------------------------------------------
test_array_insert_op:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_insert]
    call test_start

    call reset_state

    ; Create array [10, 30]
    call create_and_push_array
    test rax, rax
    jz .fail
    mov rbx, rax

    mov rdi, 10
    call stack_data_push_integer
    call op_array_push

    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    mov rdi, 30
    call stack_data_push_integer
    call op_array_push

    ; Push array for insert
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    ; Push index 1
    mov rdi, 1
    call stack_data_push_integer

    ; Push value 20
    mov rdi, 20
    call stack_data_push_integer

    ; Insert -> [10, 20, 30]
    call op_array_insert

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Verify length is now 3
    mov rdi, [rbx]
    mov rsi, 3
    call assert_eq
    test eax, eax
    jz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_array_reverse_op - Test array reverse
;-----------------------------------------------------------------------------
test_array_reverse_op:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_reverse]
    call test_start

    call reset_state

    ; Create array [1, 2, 3]
    call create_and_push_array
    test rax, rax
    jz .fail
    mov rbx, rax

    mov rdi, 1
    call stack_data_push_integer
    call op_array_push

    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    mov rdi, 2
    call stack_data_push_integer
    call op_array_push

    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    mov rdi, 3
    call stack_data_push_integer
    call op_array_push

    ; Push array for reverse
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    ; Reverse -> [3, 2, 1]
    call op_array_reverse

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Verify first element is now 3
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    mov rdi, 0
    call stack_data_push_integer

    call op_array_get

    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, 3
    call assert_eq
    test eax, eax
    jz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; test_array_sort_op - Test array sort
;-----------------------------------------------------------------------------
test_array_sort_op:
    push rbp
    mov rbp, rsp
    push rbx

    lea rdi, [test_sort]
    call test_start

    call reset_state

    ; Create array [30, 10, 20]
    call create_and_push_array
    test rax, rax
    jz .fail
    mov rbx, rax

    mov rdi, 30
    call stack_data_push_integer
    call op_array_push

    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    mov rdi, 10
    call stack_data_push_integer
    call op_array_push

    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    mov rdi, 20
    call stack_data_push_integer
    call op_array_push

    ; Push array for sort
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    ; Sort -> [10, 20, 30]
    call op_array_sort

    ; Check no error
    mov eax, [state + State.error]
    test eax, eax
    jnz .fail

    ; Verify first element is now 10
    sub r12, VALUE_SIZE
    mov [state + State.data_stack], r12
    mov byte [r12 + VALUE_TAG_OFF], VTAG_ARRAY
    mov qword [r12 + VALUE_NUM_OFF], 0
    mov [r12 + VALUE_PTR_OFF], rbx

    mov rdi, 0
    call stack_data_push_integer

    call op_array_get

    call stack_data_pop
    test rax, rax
    jz .fail

    mov rdi, [rax + VALUE_NUM_OFF]
    mov rsi, 10
    call assert_eq
    test eax, eax
    jz .fail

    call test_end_pass
    jmp .done

.fail:
    call test_end_fail

.done:
    pop rbx
    pop rbp
    ret
