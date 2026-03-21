; macros.inc - Common assembly macros
; DTRules NASM VM Implementation

%ifndef MACROS_INC
%define MACROS_INC

%include "include/constants.inc"

;-----------------------------------------------------------------------------
; Syscall macros - Linux x86-64 calling convention
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
; Value copy macros (24-byte Values)
; Much faster than rep movsb
;-----------------------------------------------------------------------------

; Copy 24-byte value: %1 = destination, %2 = source
; Clobbers: rax, rcx, rdx
%macro COPY_VALUE 2
    mov rax, [%2]
    mov rcx, [%2 + 8]
    mov rdx, [%2 + 16]
    mov [%1], rax
    mov [%1 + 8], rcx
    mov [%1 + 16], rdx
%endmacro

; Alternative that preserves rax, rcx, rdx
%macro COPY_VALUE_ALT 2
    mov r8, [%2]
    mov r9, [%2 + 8]
    mov r10, [%2 + 16]
    mov [%1], r8
    mov [%1 + 8], r9
    mov [%1 + 16], r10
%endmacro

;-----------------------------------------------------------------------------
; Data stack macros (using r12 as stack pointer)
; Stack grows DOWNWARD
;-----------------------------------------------------------------------------

; Push a 24-byte Value onto data stack
; Assumes rax points to the Value to push
%macro DPUSH 0
    sub r12, VALUE_SIZE
    mov rcx, [rax]
    mov rdx, [rax + 8]
    mov r8, [rax + 16]
    mov [r12], rcx
    mov [r12 + 8], rdx
    mov [r12 + 16], r8
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
; VM-specific macros
;-----------------------------------------------------------------------------

; Save VM registers before calling external functions
%macro VM_SAVE_REGS 0
    push r12
    push r13
    push r14
    push r15
    push rbx
%endmacro

; Restore VM registers after calling external functions
%macro VM_RESTORE_REGS 0
    pop rbx
    pop r15
    pop r14
    pop r13
    pop r12
%endmacro

;-----------------------------------------------------------------------------
; Jump table dispatch
; Jump through table based on opcode in al
;-----------------------------------------------------------------------------
%macro DISPATCH 1       ; %1 = jump table label
    movzx eax, al
    lea rcx, [%1]
    jmp [rcx + rax * 8]
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
; Error handling macros
;-----------------------------------------------------------------------------

%macro SET_ERROR 1      ; %1 = error code
    mov dword [r15 + State.error], %1
%endmacro

%macro CHECK_ERROR 0
    cmp dword [r15 + State.error], ERR_NONE
    jne .error_exit
%endmacro

;-----------------------------------------------------------------------------
; Debug/assertion macros (only active when DEBUG is defined)
;-----------------------------------------------------------------------------

%ifdef DEBUG
%macro ASSERT 1
    %1
    jnz %%ok
    ; Assertion failed
    mov rdi, 1
    mov rax, SYS_EXIT_GROUP
    syscall
%%ok:
%endmacro

%macro ASSERT_EQ 2
    mov rdi, %1
    mov rsi, %2
    cmp rdi, rsi
    je %%ok
    mov rdi, 1
    mov rax, SYS_EXIT_GROUP
    syscall
%%ok:
%endmacro
%else
%macro ASSERT 1
%endmacro
%macro ASSERT_EQ 2
%endmacro
%endif

%endif ; MACROS_INC
