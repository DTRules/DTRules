; file.asm - File I/O operations
; DTRules Assembly Implementation
; Direct Linux syscalls for file operations

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state
extern heap_alloc

section .data
    ; Initial read buffer size
    READ_BUF_SIZE   equ 4096

section .text

;-----------------------------------------------------------------------------
; file_open - Open a file for reading
; Input: rdi = pointer to null-terminated filename
; Output: rax = file descriptor, or negative on error
;-----------------------------------------------------------------------------
global file_open
file_open:
    mov rax, SYS_OPEN
    ; rdi already has filename
    mov rsi, O_RDONLY
    xor edx, edx            ; Mode (not used for read-only)
    syscall
    ret

;-----------------------------------------------------------------------------
; file_close - Close a file
; Input: rdi = file descriptor
; Output: rax = 0 on success, negative on error
;-----------------------------------------------------------------------------
global file_close
file_close:
    mov rax, SYS_CLOSE
    syscall
    ret

;-----------------------------------------------------------------------------
; file_read - Read from a file
; Input:
;   rdi = file descriptor
;   rsi = buffer pointer
;   rdx = buffer size
; Output: rax = bytes read, 0 on EOF, negative on error
;-----------------------------------------------------------------------------
global file_read
file_read:
    mov rax, SYS_READ
    syscall
    ret

;-----------------------------------------------------------------------------
; file_size - Get file size using lseek
; Input: rdi = file descriptor
; Output: rax = file size, or negative on error
;-----------------------------------------------------------------------------
global file_size
file_size:
    push rbp
    mov rbp, rsp
    push rbx

    mov rbx, rdi            ; Save fd

    ; Seek to end
    mov rax, SYS_LSEEK
    ; rdi = fd (already set)
    xor esi, esi            ; Offset 0
    mov edx, SEEK_END
    syscall

    test rax, rax
    js .error

    push rax                ; Save size

    ; Seek back to start
    mov rax, SYS_LSEEK
    mov rdi, rbx
    xor esi, esi
    mov edx, SEEK_SET
    syscall

    pop rax                 ; Restore size

.error:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; file_read_all - Read entire file into heap-allocated buffer
; Input: rdi = pointer to null-terminated filename
; Output:
;   rax = pointer to buffer (null-terminated), or 0 on error
;   rdx = file size (excluding null terminator)
;-----------------------------------------------------------------------------
global file_read_all
file_read_all:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    ; Open file
    call file_open
    test rax, rax
    js .open_error

    mov rbx, rax            ; Save fd

    ; Get file size
    mov rdi, rbx
    call file_size
    test rax, rax
    js .size_error

    mov r12, rax            ; Save size

    ; Allocate buffer (size + 1 for null terminator)
    lea rdi, [r12 + 1]
    call heap_alloc
    test rax, rax
    jz .alloc_error

    mov r13, rax            ; Save buffer pointer

    ; Read entire file
    mov rdi, rbx            ; fd
    mov rsi, r13            ; buffer
    mov rdx, r12            ; size
    call file_read

    cmp rax, r12            ; Verify we read everything
    jne .read_error

    ; Null terminate
    mov byte [r13 + r12], 0

    ; Close file
    mov rdi, rbx
    call file_close

    ; Return buffer and size
    mov rax, r13
    mov rdx, r12
    jmp .done

.open_error:
    mov dword [state + State.error], ERR_FILE_NOT_FOUND
    xor eax, eax
    jmp .done

.size_error:
.read_error:
    mov rdi, rbx
    call file_close
.alloc_error:
    mov dword [state + State.error], ERR_IO_ERROR
    xor eax, eax

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; file_exists - Check if file exists
; Input: rdi = pointer to null-terminated filename
; Output: rax = 1 if exists, 0 if not
;-----------------------------------------------------------------------------
global file_exists
file_exists:
    push rbp
    mov rbp, rsp

    call file_open
    test rax, rax
    js .not_exists

    ; Close the file
    mov rdi, rax
    call file_close

    mov eax, 1
    jmp .done

.not_exists:
    xor eax, eax

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; write_stdout - Write to standard output
; Input: rdi = buffer pointer, rsi = length
; Output: rax = bytes written
;-----------------------------------------------------------------------------
global write_stdout
write_stdout:
    mov rax, SYS_WRITE
    mov rdx, rsi            ; length
    mov rsi, rdi            ; buffer
    mov rdi, STDOUT
    syscall
    ret

;-----------------------------------------------------------------------------
; write_stderr - Write to standard error
; Input: rdi = buffer pointer, rsi = length
; Output: rax = bytes written
;-----------------------------------------------------------------------------
global write_stderr
write_stderr:
    mov rax, SYS_WRITE
    mov rdx, rsi            ; length
    mov rsi, rdi            ; buffer
    mov rdi, STDERR
    syscall
    ret
