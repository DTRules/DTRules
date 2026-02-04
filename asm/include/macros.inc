; macros.inc - Common assembly macros
; DTRules Assembly Implementation

;-----------------------------------------------------------------------------
; Syscall macros
;-----------------------------------------------------------------------------

; syscall with 1 argument
%macro SYSCALL1 2
    mov rax, %1
    mov rdi, %2
    syscall
%endmacro

; syscall with 2 arguments
%macro SYSCALL2 3
    mov rax, %1
    mov rdi, %2
    mov rsi, %3
    syscall
%endmacro

; syscall with 3 arguments
%macro SYSCALL3 4
    mov rax, %1
    mov rdi, %2
    mov rsi, %3
    mov rdx, %4
    syscall
%endmacro

; syscall with 6 arguments (for mmap)
%macro SYSCALL6 7
    mov rax, %1
    mov rdi, %2
    mov rsi, %3
    mov rdx, %4
    mov r10, %5
    mov r8, %6
    mov r9, %7
    syscall
%endmacro

;-----------------------------------------------------------------------------
; Stack manipulation macros (for data stack using r12)
;-----------------------------------------------------------------------------

; Push a 24-byte Value onto data stack
; Assumes rax points to the Value to push
%macro DPUSH 0
    sub r12, VALUE_SIZE
    mov rdi, r12
    mov rsi, rax
    mov rcx, VALUE_SIZE
    rep movsb
%endmacro

; Pop a 24-byte Value from data stack
; Result pointer in rax
%macro DPOP 0
    mov rax, r12
    add r12, VALUE_SIZE
%endmacro

; Peek at top of data stack without popping
; Result pointer in rax
%macro DPEEK 0
    mov rax, r12
%endmacro

; Check if data stack is empty
; Sets ZF if empty
%macro DEMPTY 0
    cmp r12, [state + State.data_stack_base]
%endmacro

;-----------------------------------------------------------------------------
; Value creation macros
;-----------------------------------------------------------------------------

; Initialize a Value at address with tag
%macro VALUE_INIT 2     ; %1 = address, %2 = tag
    mov byte [%1 + VALUE_TAG_OFF], %2
    xor eax, eax
    mov [%1 + VALUE_PAD_OFF], al
    mov qword [%1 + VALUE_NUM_OFF], 0
    mov qword [%1 + VALUE_PTR_OFF], 0
%endmacro

; Create integer value at address
%macro VALUE_INT 2      ; %1 = address, %2 = integer value
    mov byte [%1 + VALUE_TAG_OFF], VTAG_INTEGER
    mov qword [%1 + VALUE_NUM_OFF], %2
    mov qword [%1 + VALUE_PTR_OFF], 0
%endmacro

; Create boolean value at address
%macro VALUE_BOOL 2     ; %1 = address, %2 = 0 or 1
    mov byte [%1 + VALUE_TAG_OFF], VTAG_BOOLEAN
    mov qword [%1 + VALUE_NUM_OFF], %2
    mov qword [%1 + VALUE_PTR_OFF], 0
%endmacro

; Create null value at address
%macro VALUE_NULL 1     ; %1 = address
    mov byte [%1 + VALUE_TAG_OFF], VTAG_NULL
    mov qword [%1 + VALUE_NUM_OFF], 0
    mov qword [%1 + VALUE_PTR_OFF], 0
%endmacro

;-----------------------------------------------------------------------------
; Type checking macros
;-----------------------------------------------------------------------------

; Check if Value at address has specific tag
; Sets ZF if match
%macro CHECK_TAG 2      ; %1 = address, %2 = expected tag
    cmp byte [%1 + VALUE_TAG_OFF], %2
%endmacro

; Check if Value is numeric (integer or double)
; Sets CF if numeric
%macro CHECK_NUMERIC 1  ; %1 = address
    movzx eax, byte [%1 + VALUE_TAG_OFF]
    cmp al, VTAG_INTEGER
    je %%numeric
    cmp al, VTAG_DOUBLE
    je %%numeric
    clc
    jmp %%done
%%numeric:
    stc
%%done:
%endmacro

;-----------------------------------------------------------------------------
; Function prologue/epilogue (callee-saved: rbx, rbp, r12-r15)
;-----------------------------------------------------------------------------

%macro FUNC_PROLOG 0
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14
    push r15
%endmacro

%macro FUNC_EPILOG 0
    pop r15
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret
%endmacro

; Simpler prolog for leaf functions (no calls)
%macro LEAF_PROLOG 0
    push rbp
    mov rbp, rsp
%endmacro

%macro LEAF_EPILOG 0
    pop rbp
    ret
%endmacro

;-----------------------------------------------------------------------------
; Alignment macros
;-----------------------------------------------------------------------------

; Align value up to boundary
%macro ALIGN_UP 2       ; %1 = register, %2 = alignment (power of 2)
    add %1, %2 - 1
    and %1, ~(%2 - 1)
%endmacro

;-----------------------------------------------------------------------------
; Debug/assertion macros
;-----------------------------------------------------------------------------

%ifdef DEBUG
%macro ASSERT 1
    %1
    jnz %%ok
    ; Assertion failed - print message and halt
    mov rdi, assert_msg
    call print_string
    mov rdi, 1
    call sys_exit
%%ok:
%endmacro

%macro ASSERT_EQ 2
    mov rdi, %1
    mov rsi, %2
    cmp rdi, rsi
    je %%ok
    mov rdi, assert_eq_msg
    call print_string
    mov rdi, 1
    call sys_exit
%%ok:
%endmacro
%else
%macro ASSERT 1
%endmacro
%macro ASSERT_EQ 2
%endmacro
%endif

;-----------------------------------------------------------------------------
; Error handling macros
;-----------------------------------------------------------------------------

%macro SET_ERROR 1      ; %1 = error code
    mov dword [state + State.error], %1
%endmacro

%macro CHECK_ERROR 0
    cmp dword [state + State.error], ERR_NONE
    jne .error_exit
%endmacro

;-----------------------------------------------------------------------------
; String macros
;-----------------------------------------------------------------------------

; Get string length (assumes length-prefixed string)
; String pointer in %1, length returned in rax
%macro STRLEN 1
    mov rax, [%1]       ; First 8 bytes = length
%endmacro

; Get string data pointer (after length prefix)
%macro STRDATA 1
    lea rax, [%1 + 8]   ; Data starts after 8-byte length
%endmacro

;-----------------------------------------------------------------------------
; Array macros
;-----------------------------------------------------------------------------

; Array header structure offsets
ARRAY_LEN_OFF   equ 0   ; Current length (8 bytes)
ARRAY_CAP_OFF   equ 8   ; Capacity (8 bytes)
ARRAY_DATA_OFF  equ 16  ; Pointer to data (8 bytes)

%macro ARRAY_LEN 1
    mov rax, [%1 + ARRAY_LEN_OFF]
%endmacro

%macro ARRAY_CAP 1
    mov rax, [%1 + ARRAY_CAP_OFF]
%endmacro

;-----------------------------------------------------------------------------
; Jump table helper
;-----------------------------------------------------------------------------

; Jump through table based on opcode in al
%macro DISPATCH 1       ; %1 = jump table label
    movzx eax, al
    lea rcx, [%1]
    jmp [rcx + rax * 8]
%endmacro
