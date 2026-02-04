; string.asm - Length-prefixed string operations
; DTRules Assembly Implementation
; Strings are stored as: [length:8 bytes][data:variable][null terminator]

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"

extern state
extern heap_alloc
extern value_new_string

section .text

;-----------------------------------------------------------------------------
; string_alloc - Allocate string storage
; Input: rdi = length (not including null terminator)
; Output: rax = pointer to string structure, or 0 on failure
;-----------------------------------------------------------------------------
global string_alloc
string_alloc:
    push rbp
    mov rbp, rsp
    push rbx

    mov rbx, rdi            ; Save length

    ; Allocate: 8 (length) + data + 1 (null)
    lea rdi, [rbx + 9]
    call heap_alloc
    test rax, rax
    jz .done

    ; Store length
    mov [rax], rbx

    ; Null terminate
    mov byte [rax + rbx + 8], 0

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; string_copy - Copy a string
; Input: rdi = source string pointer
; Output: rax = new string pointer
;-----------------------------------------------------------------------------
global string_copy
string_copy:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    mov rbx, rdi            ; Source

    ; Get length
    mov r12, [rbx]          ; Length

    ; Allocate new string
    mov rdi, r12
    call string_alloc
    test rax, rax
    jz .done

    push rax                ; Save new string pointer

    ; Copy data
    lea rdi, [rax + 8]      ; Destination data
    lea rsi, [rbx + 8]      ; Source data
    mov rcx, r12
    inc rcx                 ; Include null terminator
    rep movsb

    pop rax

.done:
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; string_length - Get string length
; Input: rdi = string pointer
; Output: rax = length
;-----------------------------------------------------------------------------
global string_length
string_length:
    mov rax, [rdi]
    ret

;-----------------------------------------------------------------------------
; string_data - Get pointer to string data
; Input: rdi = string pointer
; Output: rax = data pointer
;-----------------------------------------------------------------------------
global string_data
string_data:
    lea rax, [rdi + 8]
    ret

;-----------------------------------------------------------------------------
; string_equals - Compare two strings for equality
; Input: rdi = string 1, rsi = string 2
; Output: rax = 1 if equal, 0 otherwise
;-----------------------------------------------------------------------------
global string_equals
string_equals:
    push rbp
    mov rbp, rsp

    ; Compare lengths
    mov rax, [rdi]
    cmp rax, [rsi]
    jne .not_equal

    ; Compare data
    mov rcx, rax
    add rdi, 8
    add rsi, 8

    test rcx, rcx
    jz .equal

.cmp_loop:
    mov al, [rdi]
    cmp al, [rsi]
    jne .not_equal
    inc rdi
    inc rsi
    dec rcx
    jnz .cmp_loop

.equal:
    mov eax, 1
    pop rbp
    ret

.not_equal:
    xor eax, eax
    pop rbp
    ret

;-----------------------------------------------------------------------------
; string_compare - Lexicographic string comparison
; Input: rdi = string 1, rsi = string 2
; Output: rax = -1 if s1 < s2, 0 if equal, 1 if s1 > s2
;-----------------------------------------------------------------------------
global string_compare
string_compare:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    mov rbx, [rdi]          ; len1
    mov r12, [rsi]          ; len2
    add rdi, 8              ; data1
    add rsi, 8              ; data2

    ; Compare min(len1, len2) bytes
    mov rcx, rbx
    cmp rcx, r12
    cmova rcx, r12

.cmp_loop:
    test rcx, rcx
    jz .check_len

    movzx eax, byte [rdi]
    movzx edx, byte [rsi]
    cmp eax, edx
    jl .less
    jg .greater

    inc rdi
    inc rsi
    dec rcx
    jmp .cmp_loop

.check_len:
    ; Strings equal up to min length, compare by length
    cmp rbx, r12
    jl .less
    jg .greater
    xor eax, eax            ; Equal
    jmp .done

.less:
    mov eax, -1
    jmp .done

.greater:
    mov eax, 1

.done:
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; string_concat - Concatenate two strings
; Input: rdi = string 1, rsi = string 2
; Output: rax = new concatenated string
;-----------------------------------------------------------------------------
global string_concat
string_concat:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; String 1
    mov r12, rsi            ; String 2

    ; Calculate total length
    mov r13, [rbx]          ; len1
    add r13, [r12]          ; + len2

    ; Allocate new string
    mov rdi, r13
    call string_alloc
    test rax, rax
    jz .done

    push rax                ; Save result

    ; Copy first string
    lea rdi, [rax + 8]
    lea rsi, [rbx + 8]
    mov rcx, [rbx]
    rep movsb

    ; Copy second string (rdi already advanced)
    lea rsi, [r12 + 8]
    mov rcx, [r12]
    rep movsb

    ; Null terminate
    mov byte [rdi], 0

    pop rax

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; string_substring - Extract substring
; Input: rdi = string, rsi = start index, rdx = length
; Output: rax = new substring
;-----------------------------------------------------------------------------
global string_substring
string_substring:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; String
    mov r12, rsi            ; Start
    mov r13, rdx            ; Length

    ; Validate bounds
    mov rax, [rbx]          ; Original length
    cmp r12, rax
    jae .empty              ; Start beyond end

    ; Adjust length if needed
    mov rcx, rax
    sub rcx, r12            ; Available length
    cmp r13, rcx
    cmova r13, rcx          ; Clamp length

    ; Allocate new string
    mov rdi, r13
    call string_alloc
    test rax, rax
    jz .done

    push rax

    ; Copy substring
    lea rdi, [rax + 8]
    lea rsi, [rbx + 8 + r12]
    mov rcx, r13
    rep movsb

    pop rax
    jmp .done

.empty:
    xor edi, edi
    call string_alloc

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; string_index_of - Find first occurrence of substring
; Input: rdi = haystack, rsi = needle
; Output: rax = index, or -1 if not found
;-----------------------------------------------------------------------------
global string_index_of
string_index_of:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14

    mov rbx, rdi            ; Haystack
    mov r12, rsi            ; Needle

    mov r13, [rbx]          ; Haystack length
    mov r14, [r12]          ; Needle length

    ; Empty needle found at 0
    test r14, r14
    jz .found_zero

    ; Needle longer than haystack
    cmp r14, r13
    ja .not_found

    ; Search
    xor ecx, ecx            ; Current position

.search:
    mov rax, r13
    sub rax, rcx
    cmp rax, r14
    jb .not_found           ; Not enough chars left

    ; Compare at current position
    push rcx
    lea rdi, [rbx + 8 + rcx]
    lea rsi, [r12 + 8]
    mov rdx, r14

.cmp:
    test rdx, rdx
    jz .match

    mov al, [rdi]
    cmp al, [rsi]
    jne .next

    inc rdi
    inc rsi
    dec rdx
    jmp .cmp

.match:
    pop rax                 ; Return position
    jmp .done

.next:
    pop rcx
    inc rcx
    jmp .search

.found_zero:
    xor eax, eax
    jmp .done

.not_found:
    mov rax, -1

.done:
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; string_last_index_of - Find last occurrence of substring
; Input: rdi = haystack, rsi = needle
; Output: rax = index, or -1 if not found
;-----------------------------------------------------------------------------
global string_last_index_of
string_last_index_of:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14
    push r15

    mov rbx, rdi            ; Haystack
    mov r12, rsi            ; Needle

    mov r13, [rbx]          ; Haystack length
    mov r14, [r12]          ; Needle length

    ; Empty needle found at end
    test r14, r14
    jz .found_end

    ; Needle longer than haystack
    cmp r14, r13
    ja .not_found

    mov r15, -1             ; Last found position

    ; Search from start
    xor ecx, ecx

.search:
    mov rax, r13
    sub rax, rcx
    cmp rax, r14
    jb .check_found

    ; Compare at current position
    push rcx
    lea rdi, [rbx + 8 + rcx]
    lea rsi, [r12 + 8]
    mov rdx, r14

.cmp:
    test rdx, rdx
    jz .match

    mov al, [rdi]
    cmp al, [rsi]
    jne .next

    inc rdi
    inc rsi
    dec rdx
    jmp .cmp

.match:
    pop rcx
    mov r15, rcx            ; Update last found
    inc rcx
    jmp .search

.next:
    pop rcx
    inc rcx
    jmp .search

.check_found:
    mov rax, r15
    jmp .done

.found_end:
    mov rax, r13
    jmp .done

.not_found:
    mov rax, -1

.done:
    pop r15
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; string_starts_with - Check if string starts with prefix
; Input: rdi = string, rsi = prefix
; Output: rax = 1 if true, 0 if false
;-----------------------------------------------------------------------------
global string_starts_with
string_starts_with:
    push rbp
    mov rbp, rsp

    mov rax, [rsi]          ; Prefix length
    cmp rax, [rdi]          ; String length
    ja .false

    mov rcx, rax
    add rdi, 8
    add rsi, 8

.cmp:
    test rcx, rcx
    jz .true

    mov al, [rdi]
    cmp al, [rsi]
    jne .false

    inc rdi
    inc rsi
    dec rcx
    jmp .cmp

.true:
    mov eax, 1
    pop rbp
    ret

.false:
    xor eax, eax
    pop rbp
    ret

;-----------------------------------------------------------------------------
; string_ends_with - Check if string ends with suffix
; Input: rdi = string, rsi = suffix
; Output: rax = 1 if true, 0 if false
;-----------------------------------------------------------------------------
global string_ends_with
string_ends_with:
    push rbp
    mov rbp, rsp

    mov rax, [rsi]          ; Suffix length
    mov rcx, [rdi]          ; String length
    cmp rax, rcx
    ja .false

    ; Start comparison at end - suffix_len
    sub rcx, rax
    add rdi, 8
    add rdi, rcx            ; rdi points to start of potential suffix
    add rsi, 8
    mov rcx, rax            ; Compare suffix_len bytes

.cmp:
    test rcx, rcx
    jz .true

    mov al, [rdi]
    cmp al, [rsi]
    jne .false

    inc rdi
    inc rsi
    dec rcx
    jmp .cmp

.true:
    mov eax, 1
    pop rbp
    ret

.false:
    xor eax, eax
    pop rbp
    ret

;-----------------------------------------------------------------------------
; string_contains - Check if string contains substring
; Input: rdi = haystack, rsi = needle
; Output: rax = 1 if contains, 0 otherwise
;-----------------------------------------------------------------------------
global string_contains
string_contains:
    call string_index_of
    cmp rax, -1
    setne al
    movzx eax, al
    ret

;-----------------------------------------------------------------------------
; string_to_upper - Convert string to uppercase
; Input: rdi = string
; Output: rax = new uppercase string
;-----------------------------------------------------------------------------
global string_to_upper
string_to_upper:
    push rbp
    mov rbp, rsp
    push rbx

    mov rbx, rdi

    ; Copy string
    call string_copy
    test rax, rax
    jz .done

    ; Convert to upper
    mov rcx, [rax]          ; Length
    lea rdi, [rax + 8]      ; Data

.loop:
    test rcx, rcx
    jz .done

    movzx edx, byte [rdi]
    cmp dl, 'a'
    jb .next
    cmp dl, 'z'
    ja .next
    sub dl, 32              ; 'a' - 'A'
    mov [rdi], dl

.next:
    inc rdi
    dec rcx
    jmp .loop

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; string_to_lower - Convert string to lowercase
; Input: rdi = string
; Output: rax = new lowercase string
;-----------------------------------------------------------------------------
global string_to_lower
string_to_lower:
    push rbp
    mov rbp, rsp
    push rbx

    mov rbx, rdi

    call string_copy
    test rax, rax
    jz .done

    mov rcx, [rax]
    lea rdi, [rax + 8]

.loop:
    test rcx, rcx
    jz .done

    movzx edx, byte [rdi]
    cmp dl, 'A'
    jb .next
    cmp dl, 'Z'
    ja .next
    add dl, 32

.next:
    inc rdi
    dec rcx
    jmp .loop

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; string_trim - Trim whitespace from both ends
; Input: rdi = string
; Output: rax = new trimmed string
;-----------------------------------------------------------------------------
global string_trim
string_trim:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi
    mov r12, [rbx]          ; Length
    lea r13, [rbx + 8]      ; Data

    ; Find start (skip leading whitespace)
    xor ecx, ecx            ; Start index

.skip_start:
    cmp rcx, r12
    jae .empty

    movzx eax, byte [r13 + rcx]
    cmp al, ' '
    je .next_start
    cmp al, 9               ; Tab
    je .next_start
    cmp al, 10              ; Newline
    je .next_start
    cmp al, 13              ; CR
    je .next_start
    jmp .find_end

.next_start:
    inc rcx
    jmp .skip_start

.find_end:
    ; Find end (skip trailing whitespace)
    mov rdx, r12            ; End index (exclusive)

.skip_end:
    cmp rdx, rcx
    jle .empty

    movzx eax, byte [r13 + rdx - 1]
    cmp al, ' '
    je .next_end
    cmp al, 9
    je .next_end
    cmp al, 10
    je .next_end
    cmp al, 13
    je .next_end
    jmp .extract

.next_end:
    dec rdx
    jmp .skip_end

.extract:
    ; Extract substring from rcx to rdx
    sub rdx, rcx            ; Length
    mov rdi, rbx
    mov rsi, rcx
    ; rdx already has length
    call string_substring
    jmp .done

.empty:
    xor edi, edi
    call string_alloc

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret
