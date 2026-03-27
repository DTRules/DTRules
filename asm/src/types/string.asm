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
    mov [rdi], dl           ; Store the modified character

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

;-----------------------------------------------------------------------------
; string_trim_left - Trim whitespace from start only
; Input: rdi = string
; Output: rax = new trimmed string
;-----------------------------------------------------------------------------
global string_trim_left
string_trim_left:
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

.ltrim_skip_start:
    cmp rcx, r12
    jae .ltrim_empty

    movzx eax, byte [r13 + rcx]
    cmp al, ' '
    je .ltrim_next_start
    cmp al, 9               ; Tab
    je .ltrim_next_start
    cmp al, 10              ; Newline
    je .ltrim_next_start
    cmp al, 13              ; CR
    je .ltrim_next_start
    jmp .ltrim_extract

.ltrim_next_start:
    inc rcx
    jmp .ltrim_skip_start

.ltrim_extract:
    ; Extract substring from rcx to end
    mov rdx, r12
    sub rdx, rcx            ; Length = total - start
    mov rdi, rbx
    mov rsi, rcx
    ; rdx already has length
    call string_substring
    jmp .ltrim_done

.ltrim_empty:
    xor edi, edi
    call string_alloc

.ltrim_done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; string_trim_right - Trim whitespace from end only
; Input: rdi = string
; Output: rax = new trimmed string
;-----------------------------------------------------------------------------
global string_trim_right
string_trim_right:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi
    mov r12, [rbx]          ; Length
    lea r13, [rbx + 8]      ; Data

    ; Empty string check
    test r12, r12
    jz .rtrim_empty

    ; Find end (skip trailing whitespace)
    mov rdx, r12            ; End index (exclusive)

.rtrim_skip_end:
    test rdx, rdx
    jz .rtrim_empty

    movzx eax, byte [r13 + rdx - 1]
    cmp al, ' '
    je .rtrim_next_end
    cmp al, 9
    je .rtrim_next_end
    cmp al, 10
    je .rtrim_next_end
    cmp al, 13
    je .rtrim_next_end
    jmp .rtrim_extract

.rtrim_next_end:
    dec rdx
    jmp .rtrim_skip_end

.rtrim_extract:
    ; Extract substring from 0 to rdx
    mov rdi, rbx
    xor esi, esi            ; Start at 0
    ; rdx already has length
    call string_substring
    jmp .rtrim_done

.rtrim_empty:
    xor edi, edi
    call string_alloc

.rtrim_done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; string_replace - Replace first occurrence of pattern with replacement
; Input: rdi = string, rsi = pattern, rdx = replacement
; Output: rax = new string with replacement
;-----------------------------------------------------------------------------
global string_replace
string_replace:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14
    push r15

    mov rbx, rdi            ; Original string
    mov r12, rsi            ; Pattern
    mov r13, rdx            ; Replacement

    ; Find pattern in string
    mov rdi, rbx
    mov rsi, r12
    call string_index_of
    cmp rax, -1
    je .repl_not_found

    mov r14, rax            ; Found position

    ; Calculate new string length
    ; new_len = orig_len - pattern_len + replacement_len
    mov r15, [rbx]          ; Original length
    sub r15, [r12]          ; - pattern length
    add r15, [r13]          ; + replacement length

    ; Allocate new string
    mov rdi, r15
    call string_alloc
    test rax, rax
    jz .repl_done

    push rax                ; Save result

    ; Copy: prefix + replacement + suffix
    ; 1. Copy prefix (0 to found_pos)
    lea rdi, [rax + 8]      ; Dest data
    lea rsi, [rbx + 8]      ; Source data
    mov rcx, r14            ; Prefix length
    rep movsb

    ; 2. Copy replacement
    lea rsi, [r13 + 8]      ; Replacement data
    mov rcx, [r13]          ; Replacement length
    rep movsb

    ; 3. Copy suffix (after pattern)
    lea rsi, [rbx + 8]
    add rsi, r14            ; Skip prefix
    add rsi, [r12]          ; Skip pattern
    mov rcx, [rbx]          ; Original length
    sub rcx, r14            ; - prefix length
    sub rcx, [r12]          ; - pattern length
    rep movsb

    ; Null terminate
    mov byte [rdi], 0

    pop rax
    jmp .repl_done

.repl_not_found:
    ; Return copy of original string
    mov rdi, rbx
    call string_copy

.repl_done:
    pop r15
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; string_replace_all - Replace all occurrences of pattern with replacement
; Input: rdi = string, rsi = pattern, rdx = replacement
; Output: rax = new string with all replacements
;-----------------------------------------------------------------------------
global string_replace_all
string_replace_all:
    push rbp
    mov rbp, rsp
    sub rsp, 16             ; Local storage FIRST: [rbp-8] and [rbp-16]
    push rbx
    push r14                ; Only save registers we actually use

    mov rbx, rdi            ; Current string (will be updated)

    ; Store pattern and replacement in local storage
    mov [rbp - 8], rsi      ; Pattern
    mov [rbp - 16], rdx     ; Replacement

    ; If pattern is empty, return original
    cmp qword [rsi], 0
    je .replall_copy_orig

.replall_loop:
    ; Find pattern in current string
    mov rdi, rbx
    mov rsi, [rbp - 8]      ; Pattern
    call string_index_of
    cmp rax, -1
    je .replall_done        ; No more occurrences

    ; Replace one occurrence
    mov rdi, rbx
    mov rsi, [rbp - 8]      ; Pattern
    mov rdx, [rbp - 16]     ; Replacement
    call string_replace
    test rax, rax
    jz .replall_done        ; Allocation failed

    ; Use new string for next iteration
    mov rbx, rax
    jmp .replall_loop

.replall_copy_orig:
    mov rdi, rbx
    call string_copy
    mov rbx, rax

.replall_done:
    mov rax, rbx

    pop r14
    pop rbx
    add rsp, 16
    pop rbp
    ret

;-----------------------------------------------------------------------------
; string_split - Split string by delimiter into array
; Input: rdi = string, rsi = delimiter
; Output: rax = array header pointer containing string Values
;-----------------------------------------------------------------------------
global string_split
extern array_alloc
extern array_push
extern value_new_string
string_split:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14
    push r15
    sub rsp, 48             ; Local storage

    mov rbx, rdi            ; String
    mov r12, rsi            ; Delimiter
    xor r13, r13            ; Current position in string
    mov r14, [rbx]          ; String length
    mov r15, [r12]          ; Delimiter length

    ; Create result array
    xor edi, edi            ; Default capacity
    call array_alloc
    test rax, rax
    jz .split_done

    mov [rbp - 48], rax     ; Save array header

    ; Handle empty delimiter - split each character
    test r15, r15
    jz .split_by_char

.split_loop:
    ; Find next delimiter starting at current position
    mov rdi, rbx
    mov rsi, r13
    mov rdx, r14
    sub rdx, r13
    call string_substring
    test rax, rax
    jz .split_final

    push rax                ; Save temp substring
    mov rdi, rax
    mov rsi, r12            ; Delimiter
    call string_index_of
    mov rcx, rax            ; Found position (relative to current pos)
    pop rax                 ; Restore temp substring

    cmp rcx, -1
    je .split_last_segment

    ; Extract segment before delimiter
    push rcx
    mov rdi, rbx
    mov rsi, r13            ; Start
    mov rdx, rcx            ; Length
    call string_substring
    test rax, rax
    jz .split_pop_exit

    ; Create string Value
    push rax
    lea rdi, [rax + 8]      ; String data
    mov rsi, [rax]          ; Length
    call value_new_string
    add rsp, 8              ; Discard substring ptr
    test rax, rax
    jz .split_pop_exit

    ; Push to array
    mov rdi, [rbp - 48]     ; Array header
    mov rsi, rax            ; Value
    call array_push

    pop rcx                 ; Restore found position

    ; Move past delimiter
    add r13, rcx
    add r13, r15            ; Skip delimiter
    cmp r13, r14
    jb .split_loop
    jmp .split_final

.split_last_segment:
    ; Extract remaining string
    mov rdi, rbx
    mov rsi, r13
    mov rdx, r14
    sub rdx, r13
    call string_substring
    test rax, rax
    jz .split_final

    ; Create string Value
    push rax
    lea rdi, [rax + 8]
    mov rsi, [rax]
    call value_new_string
    add rsp, 8
    test rax, rax
    jz .split_final

    ; Push to array
    mov rdi, [rbp - 48]
    mov rsi, rax
    call array_push

.split_final:
    mov rax, [rbp - 48]
    jmp .split_done

.split_pop_exit:
    pop rcx
    mov rax, [rbp - 48]
    jmp .split_done

.split_by_char:
    ; Split into individual characters
    xor r13, r13
.split_char_loop:
    cmp r13, r14
    jae .split_final

    ; Create single-char string
    mov rdi, rbx
    mov rsi, r13
    mov rdx, 1
    call string_substring
    test rax, rax
    jz .split_final

    ; Create Value
    push rax
    lea rdi, [rax + 8]
    mov rsi, 1
    call value_new_string
    add rsp, 8
    test rax, rax
    jz .split_final

    ; Push to array
    mov rdi, [rbp - 48]
    mov rsi, rax
    call array_push

    inc r13
    jmp .split_char_loop

.split_done:
    add rsp, 48
    pop r15
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; string_join - Join array of strings with delimiter
; Input: rdi = array header, rsi = delimiter string
; Output: rax = joined string
;-----------------------------------------------------------------------------
global string_join
extern array_length
extern array_get
string_join:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14
    push r15
    sub rsp, 8

    mov rbx, rdi            ; Array
    mov r12, rsi            ; Delimiter

    ; Get array length
    mov rdi, rbx
    call array_length
    test rax, rax
    jz .join_empty

    mov r13, rax            ; Array length
    mov r14, [r12]          ; Delimiter length

    ; Calculate total length
    xor r15, r15            ; Total length
    xor ecx, ecx            ; Index

.join_calc_loop:
    cmp rcx, r13
    jae .join_calc_done

    push rcx
    mov rdi, rbx
    mov rsi, rcx
    call array_get
    pop rcx
    test rax, rax
    jz .join_calc_next

    ; Get string from Value
    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .join_calc_next

    mov rax, [rax + VALUE_PTR_OFF]
    add r15, [rax]          ; Add string length

.join_calc_next:
    inc rcx
    jmp .join_calc_loop

.join_calc_done:
    ; Add delimiter lengths: (count - 1) * delimiter_len
    mov rax, r13
    dec rax
    imul rax, r14
    add r15, rax

    ; Allocate result string
    mov rdi, r15
    call string_alloc
    test rax, rax
    jz .join_done

    mov [rbp - 8], rax      ; Save result
    lea rdi, [rax + 8]      ; Dest pointer

    ; Build joined string
    xor ecx, ecx            ; Index

.join_build_loop:
    cmp rcx, r13
    jae .join_finish

    ; Add delimiter if not first
    test rcx, rcx
    jz .join_no_delim

    push rcx
    lea rsi, [r12 + 8]      ; Delimiter data
    mov rcx, r14            ; Delimiter length
    rep movsb               ; Advances rdi by rcx
    pop rcx

.join_no_delim:
    push rcx
    push rdi
    mov rdi, rbx
    mov rsi, rcx
    call array_get
    pop rdi
    pop rcx
    test rax, rax
    jz .join_build_next

    cmp byte [rax + VALUE_TAG_OFF], VTAG_STRING
    jne .join_build_next

    ; Copy string data
    ; Note: rep movsb advances rdi automatically
    push rcx
    mov rax, [rax + VALUE_PTR_OFF]
    mov rcx, [rax]          ; Length
    lea rsi, [rax + 8]      ; Data
    rep movsb               ; Advances rdi by rcx
    pop rcx                 ; Restore loop counter

.join_build_next:
    inc rcx
    jmp .join_build_loop

.join_finish:
    mov byte [rdi], 0       ; Null terminate
    mov rax, [rbp - 8]
    jmp .join_done

.join_empty:
    xor edi, edi
    call string_alloc

.join_done:
    add rsp, 8
    pop r15
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; string_tokenize - Split string into tokens (like split but handles whitespace)
; Input: rdi = source string, rsi = delimiter string
; Output: rax = array of token strings
;-----------------------------------------------------------------------------
global string_tokenize
string_tokenize:
    ; Use string_split implementation
    jmp string_split

;-----------------------------------------------------------------------------
; string_equals_ignore_case - Case-insensitive string comparison
; Input: rdi = string 1, rsi = string 2
; Output: rax = 1 if equal (ignoring case), 0 otherwise
;-----------------------------------------------------------------------------
global string_equals_ignore_case
string_equals_ignore_case:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    ; Compare lengths first
    mov rax, [rdi]
    cmp rax, [rsi]
    jne .not_equal_case

    mov r12, rax            ; Length
    add rdi, 8              ; String 1 data
    add rsi, 8              ; String 2 data

    test r12, r12
    jz .equal_case

.compare_loop:
    movzx eax, byte [rdi]
    movzx ecx, byte [rsi]

    ; Convert both to lowercase for comparison
    cmp al, 'A'
    jb .check_c1
    cmp al, 'Z'
    ja .check_c1
    add al, 32              ; Convert to lowercase

.check_c1:
    cmp cl, 'A'
    jb .do_compare
    cmp cl, 'Z'
    ja .do_compare
    add cl, 32              ; Convert to lowercase

.do_compare:
    cmp al, cl
    jne .not_equal_case

    inc rdi
    inc rsi
    dec r12
    jnz .compare_loop

.equal_case:
    mov eax, 1
    jmp .done_case

.not_equal_case:
    xor eax, eax

.done_case:
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; string_regex_match - Basic regex pattern matching
; Input: rdi = string to match, rsi = pattern
; Output: rax = 1 if match, 0 otherwise
;
; Supports only simple patterns:
; - Literal characters
; - . matches any character
; - * matches zero or more of previous
; - ^ anchors to start
; - $ anchors to end
;-----------------------------------------------------------------------------
global string_regex_match
string_regex_match:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14

    ; Get string data pointers
    mov r12, [rdi]          ; String length
    add rdi, 8              ; String data
    mov rbx, rdi            ; String start

    mov r13, [rsi]          ; Pattern length
    add rsi, 8              ; Pattern data
    mov r14, rsi            ; Pattern start

    ; Check for ^ anchor
    cmp byte [r14], '^'
    jne .try_all_positions

    ; Must match from start
    inc r14                 ; Skip ^
    dec r13
    jmp .match_here

.try_all_positions:
    ; Try matching at each position
.try_pos:
    push rbx
    push r12
    push r14
    push r13

    mov rdi, rbx
    mov rsi, r14
    mov rcx, r13
    call .match_pattern_internal

    pop r13
    pop r14
    pop r12
    pop rbx

    test eax, eax
    jnz .regex_match_found

    ; Try next position
    inc rbx
    dec r12
    test r12, r12
    jnz .try_pos

    ; No match found
    xor eax, eax
    jmp .regex_done

.match_here:
    mov rdi, rbx
    mov rsi, r14
    mov rcx, r13
    call .match_pattern_internal
    jmp .regex_done

.regex_match_found:
    mov eax, 1

.regex_done:
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

; Internal helper: match pattern at current position
; rdi = string pointer, rsi = pattern pointer, rcx = pattern length
; Returns: rax = 1 if match, 0 otherwise
.match_pattern_internal:
    push rbx
    push r12

    mov rbx, rdi            ; String ptr
    mov r12, rsi            ; Pattern ptr

.pattern_loop:
    test rcx, rcx
    jz .pattern_match       ; End of pattern = match

    ; Check for $ at end
    cmp byte [r12], '$'
    jne .not_end_anchor
    cmp rcx, 1
    jne .not_end_anchor
    ; $ must match end of string
    cmp byte [rbx], 0
    je .pattern_match
    jmp .pattern_no_match

.not_end_anchor:
    ; Get pattern char
    movzx eax, byte [r12]

    ; Check for * (zero or more)
    cmp rcx, 1
    jbe .no_star
    cmp byte [r12 + 1], '*'
    jne .no_star

    ; Handle X* pattern
    mov dl, al              ; Char to match (or . for any)

.star_loop:
    ; Try matching rest of pattern
    push rbx
    push r12
    push rcx
    push rdx

    add r12, 2              ; Skip X*
    sub rcx, 2
    mov rdi, rbx
    mov rsi, r12
    call .match_pattern_internal

    pop rdx
    pop rcx
    pop r12
    pop rbx

    test eax, eax
    jnz .pattern_match

    ; Try consuming one more char
    cmp byte [rbx], 0
    je .pattern_no_match

    ; Check if current char matches pattern char
    cmp dl, '.'
    je .star_consume
    cmp dl, [rbx]
    jne .pattern_no_match

.star_consume:
    inc rbx
    jmp .star_loop

.no_star:
    ; Regular char match
    cmp byte [rbx], 0
    je .pattern_no_match    ; String ended but pattern remains

    cmp al, '.'
    je .any_match

    cmp al, [rbx]
    jne .pattern_no_match

.any_match:
    inc rbx
    inc r12
    dec rcx
    jmp .pattern_loop

.pattern_match:
    mov eax, 1
    jmp .pattern_done

.pattern_no_match:
    xor eax, eax

.pattern_done:
    pop r12
    pop rbx
    ret
