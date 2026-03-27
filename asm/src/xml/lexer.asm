; lexer.asm - XML tokenizer
; DTRules Assembly Implementation
; Hand-written XML lexer for parsing rule files

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"

extern state
extern heap_alloc

;-----------------------------------------------------------------------------
; Token types
;-----------------------------------------------------------------------------
TOK_EOF         equ 0
TOK_TEXT        equ 1       ; Text content
TOK_OPEN_TAG    equ 2       ; <tag
TOK_CLOSE_TAG   equ 3       ; </tag
TOK_SELF_CLOSE  equ 4       ; />
TOK_TAG_END     equ 5       ; >
TOK_ATTR_NAME   equ 6       ; Attribute name
TOK_ATTR_VALUE  equ 7       ; Attribute value
TOK_COMMENT     equ 8       ; <!-- ... -->
TOK_CDATA       equ 9       ; <![CDATA[ ... ]]>
TOK_PI          equ 10      ; <?...?>
TOK_DOCTYPE     equ 11      ; <!DOCTYPE...>
TOK_ERROR       equ 255

;-----------------------------------------------------------------------------
; Lexer state structure
;-----------------------------------------------------------------------------
struc Lexer
    .input:         resq 1  ; Input string pointer
    .pos:           resq 1  ; Current position
    .end:           resq 1  ; End of input
    .line:          resq 1  ; Current line number
    .col:           resq 1  ; Current column
    .token_type:    resd 1  ; Current token type
    .token_start:   resq 1  ; Token start pointer
    .token_len:     resq 1  ; Token length
endstruc

section .bss
    align 8
    global xml_lexer
    xml_lexer: resb Lexer_size

section .text

;-----------------------------------------------------------------------------
; lexer_init - Initialize the lexer
; Input: rdi = input string, rsi = length
;-----------------------------------------------------------------------------
global lexer_init
lexer_init:
    lea rax, [xml_lexer]

    mov [rax + Lexer.input], rdi
    mov [rax + Lexer.pos], rdi
    add rsi, rdi
    mov [rax + Lexer.end], rsi
    mov qword [rax + Lexer.line], 1
    mov qword [rax + Lexer.col], 1
    mov dword [rax + Lexer.token_type], TOK_EOF
    mov qword [rax + Lexer.token_start], 0
    mov qword [rax + Lexer.token_len], 0

    ret

;-----------------------------------------------------------------------------
; lexer_peek - Peek at current character
; Output: rax = character, or -1 if EOF
;-----------------------------------------------------------------------------
global lexer_peek
lexer_peek:
    lea rcx, [xml_lexer]
    mov rax, [rcx + Lexer.pos]
    cmp rax, [rcx + Lexer.end]
    jae .eof
    movzx eax, byte [rax]
    ret
.eof:
    mov rax, -1
    ret

;-----------------------------------------------------------------------------
; lexer_peek_n - Peek n characters ahead
; Input: rdi = offset
; Output: rax = character, or -1 if EOF
;-----------------------------------------------------------------------------
global lexer_peek_n
lexer_peek_n:
    lea rcx, [xml_lexer]
    mov rax, [rcx + Lexer.pos]
    add rax, rdi
    cmp rax, [rcx + Lexer.end]
    jae .eof
    movzx eax, byte [rax]
    ret
.eof:
    mov rax, -1
    ret

;-----------------------------------------------------------------------------
; lexer_advance - Advance to next character
;-----------------------------------------------------------------------------
global lexer_advance
lexer_advance:
    lea rcx, [xml_lexer]
    mov rax, [rcx + Lexer.pos]
    cmp rax, [rcx + Lexer.end]
    jae .done

    ; Check for newline
    movzx edx, byte [rax]
    cmp dl, 10
    jne .not_newline

    inc qword [rcx + Lexer.line]
    mov qword [rcx + Lexer.col], 0

.not_newline:
    inc qword [rcx + Lexer.pos]
    inc qword [rcx + Lexer.col]

.done:
    ret

;-----------------------------------------------------------------------------
; lexer_skip_whitespace - Skip whitespace characters
;-----------------------------------------------------------------------------
global lexer_skip_whitespace
lexer_skip_whitespace:
.loop:
    call lexer_peek
    cmp rax, -1
    je .done
    cmp al, ' '
    je .skip
    cmp al, 9               ; Tab
    je .skip
    cmp al, 10              ; Newline
    je .skip
    cmp al, 13              ; CR
    je .skip
    jmp .done
.skip:
    call lexer_advance
    jmp .loop
.done:
    ret

;-----------------------------------------------------------------------------
; lexer_match - Check if current position matches string
; Input: rdi = string to match, rsi = length
; Output: rax = 1 if match, 0 otherwise
;-----------------------------------------------------------------------------
global lexer_match
lexer_match:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    mov rbx, rdi            ; String to match
    mov r12, rsi            ; Length

    lea rcx, [xml_lexer]
    mov rax, [rcx + Lexer.pos]
    mov rdx, [rcx + Lexer.end]

    ; Check if enough characters remain
    add rax, r12
    cmp rax, rdx
    ja .no_match

    mov rax, [rcx + Lexer.pos]
    xor ecx, ecx

.cmp_loop:
    cmp rcx, r12
    jae .match

    movzx edx, byte [rax + rcx]
    cmp dl, [rbx + rcx]
    jne .no_match

    inc rcx
    jmp .cmp_loop

.match:
    mov eax, 1
    jmp .done

.no_match:
    xor eax, eax

.done:
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; lexer_next_token - Get next token
; Output: rax = token type
;-----------------------------------------------------------------------------
global lexer_next_token
lexer_next_token:
    push rbp
    mov rbp, rsp
    push rbx

    lea rbx, [xml_lexer]

    ; Check for EOF
    call lexer_peek
    cmp rax, -1
    je .eof

    ; Check for tag start
    cmp al, '<'
    je .tag_start

    ; Otherwise it's text content
    jmp .text_content

.eof:
    mov dword [rbx + Lexer.token_type], TOK_EOF
    mov eax, TOK_EOF
    jmp .done

.tag_start:
    call lexer_advance      ; Skip <

    call lexer_peek
    cmp al, '/'
    je .close_tag
    cmp al, '!'
    je .special_tag
    cmp al, '?'
    je .processing_instruction

    ; Open tag
    jmp .open_tag

.close_tag:
    call lexer_advance      ; Skip /
    mov dword [rbx + Lexer.token_type], TOK_CLOSE_TAG
    jmp .scan_name

.special_tag:
    ; Check for comment or CDATA
    mov rdi, 1
    call lexer_peek_n
    cmp al, '-'
    je .comment

    ; Check for CDATA
    lea rdi, [.cdata_start]
    mov rsi, 7
    call lexer_match
    test eax, eax
    jnz .cdata

    ; Check for DOCTYPE
    lea rdi, [.doctype_start]
    mov rsi, 7
    call lexer_match
    test eax, eax
    jnz .doctype

    ; Unknown, treat as text
    jmp .text_content

.comment:
    ; Skip !--
    call lexer_advance      ; !
    call lexer_advance      ; -
    call lexer_advance      ; -

    mov rax, [rbx + Lexer.pos]
    mov [rbx + Lexer.token_start], rax

    ; Scan until -->
.comment_scan:
    call lexer_peek
    cmp rax, -1
    je .error

    cmp al, '-'
    jne .comment_next

    lea rdi, [.comment_end]
    mov rsi, 3
    call lexer_match
    test eax, eax
    jnz .comment_done

.comment_next:
    call lexer_advance
    jmp .comment_scan

.comment_done:
    ; Calculate length
    mov rax, [rbx + Lexer.pos]
    sub rax, [rbx + Lexer.token_start]
    mov [rbx + Lexer.token_len], rax

    ; Skip -->
    call lexer_advance
    call lexer_advance
    call lexer_advance

    mov dword [rbx + Lexer.token_type], TOK_COMMENT
    mov eax, TOK_COMMENT
    jmp .done

.cdata:
    ; Skip ![CDATA[
    mov rcx, 8
.cdata_skip:
    call lexer_advance
    dec rcx
    jnz .cdata_skip

    mov rax, [rbx + Lexer.pos]
    mov [rbx + Lexer.token_start], rax

    ; Scan until ]]>
.cdata_scan:
    call lexer_peek
    cmp rax, -1
    je .error

    cmp al, ']'
    jne .cdata_next

    lea rdi, [.cdata_end]
    mov rsi, 3
    call lexer_match
    test eax, eax
    jnz .cdata_done

.cdata_next:
    call lexer_advance
    jmp .cdata_scan

.cdata_done:
    mov rax, [rbx + Lexer.pos]
    sub rax, [rbx + Lexer.token_start]
    mov [rbx + Lexer.token_len], rax

    call lexer_advance
    call lexer_advance
    call lexer_advance

    mov dword [rbx + Lexer.token_type], TOK_CDATA
    mov eax, TOK_CDATA
    jmp .done

.doctype:
    ; Skip !DOCTYPE and scan until >
    mov rcx, 8
.doctype_skip:
    call lexer_advance
    dec rcx
    jnz .doctype_skip

    mov rax, [rbx + Lexer.pos]
    mov [rbx + Lexer.token_start], rax

.doctype_scan:
    call lexer_peek
    cmp rax, -1
    je .error
    cmp al, '>'
    je .doctype_done
    call lexer_advance
    jmp .doctype_scan

.doctype_done:
    mov rax, [rbx + Lexer.pos]
    sub rax, [rbx + Lexer.token_start]
    mov [rbx + Lexer.token_len], rax

    call lexer_advance      ; Skip >

    mov dword [rbx + Lexer.token_type], TOK_DOCTYPE
    mov eax, TOK_DOCTYPE
    jmp .done

.processing_instruction:
    call lexer_advance      ; Skip ?

    mov rax, [rbx + Lexer.pos]
    mov [rbx + Lexer.token_start], rax

.pi_scan:
    call lexer_peek
    cmp rax, -1
    je .error
    cmp al, '?'
    jne .pi_next

    mov rdi, 1
    call lexer_peek_n
    cmp al, '>'
    je .pi_done

.pi_next:
    call lexer_advance
    jmp .pi_scan

.pi_done:
    mov rax, [rbx + Lexer.pos]
    sub rax, [rbx + Lexer.token_start]
    mov [rbx + Lexer.token_len], rax

    call lexer_advance      ; ?
    call lexer_advance      ; >

    mov dword [rbx + Lexer.token_type], TOK_PI
    mov eax, TOK_PI
    jmp .done

.open_tag:
    mov dword [rbx + Lexer.token_type], TOK_OPEN_TAG

.scan_name:
    mov rax, [rbx + Lexer.pos]
    mov [rbx + Lexer.token_start], rax

.name_loop:
    call lexer_peek
    cmp rax, -1
    je .name_done

    ; Name characters: alphanumeric, _, -, :, .
    cmp al, '>'
    je .name_done
    cmp al, '/'
    je .name_done
    cmp al, ' '
    je .name_done
    cmp al, 9
    je .name_done
    cmp al, 10
    je .name_done
    cmp al, 13
    je .name_done
    cmp al, '='
    je .name_done

    call lexer_advance
    jmp .name_loop

.name_done:
    mov rax, [rbx + Lexer.pos]
    sub rax, [rbx + Lexer.token_start]
    mov [rbx + Lexer.token_len], rax

    mov eax, [rbx + Lexer.token_type]
    jmp .done

.text_content:
    mov rax, [rbx + Lexer.pos]
    mov [rbx + Lexer.token_start], rax

.text_loop:
    call lexer_peek
    cmp rax, -1
    je .text_done
    cmp al, '<'
    je .text_done
    call lexer_advance
    jmp .text_loop

.text_done:
    mov rax, [rbx + Lexer.pos]
    sub rax, [rbx + Lexer.token_start]
    mov [rbx + Lexer.token_len], rax

    mov dword [rbx + Lexer.token_type], TOK_TEXT
    mov eax, TOK_TEXT
    jmp .done

.error:
    mov dword [rbx + Lexer.token_type], TOK_ERROR
    mov eax, TOK_ERROR

.done:
    pop rbx
    pop rbp
    ret

section .rodata
.comment_end:   db "-->"
.cdata_start:   db "[CDATA["
.cdata_end:     db "]]>"
.doctype_start: db "DOCTYPE"

section .text

;-----------------------------------------------------------------------------
; lexer_scan_attribute - Scan attribute name=value
; Output: rax = TOK_ATTR_NAME or TOK_TAG_END or TOK_SELF_CLOSE
;-----------------------------------------------------------------------------
global lexer_scan_attribute
lexer_scan_attribute:
    push rbp
    mov rbp, rsp
    push rbx

    lea rbx, [xml_lexer]

    call lexer_skip_whitespace

    call lexer_peek
    cmp rax, -1
    je .error
    cmp al, '>'
    je .tag_end
    cmp al, '/'
    je .self_close

    ; Scan attribute name
    mov rax, [rbx + Lexer.pos]
    mov [rbx + Lexer.token_start], rax

.attr_name_loop:
    call lexer_peek
    cmp rax, -1
    je .attr_name_done
    cmp al, '='
    je .attr_name_done
    cmp al, ' '
    je .attr_name_done
    cmp al, 9
    je .attr_name_done
    cmp al, 10
    je .attr_name_done
    cmp al, '>'
    je .attr_name_done
    cmp al, '/'
    je .attr_name_done

    call lexer_advance
    jmp .attr_name_loop

.attr_name_done:
    mov rax, [rbx + Lexer.pos]
    sub rax, [rbx + Lexer.token_start]
    mov [rbx + Lexer.token_len], rax

    mov dword [rbx + Lexer.token_type], TOK_ATTR_NAME
    mov eax, TOK_ATTR_NAME
    jmp .done

.tag_end:
    call lexer_advance
    mov dword [rbx + Lexer.token_type], TOK_TAG_END
    mov eax, TOK_TAG_END
    jmp .done

.self_close:
    call lexer_advance      ; /
    call lexer_peek
    cmp al, '>'
    jne .error
    call lexer_advance      ; >
    mov dword [rbx + Lexer.token_type], TOK_SELF_CLOSE
    mov eax, TOK_SELF_CLOSE
    jmp .done

.error:
    mov dword [rbx + Lexer.token_type], TOK_ERROR
    mov eax, TOK_ERROR

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; lexer_scan_attr_value - Scan attribute value after =
; Output: rax = TOK_ATTR_VALUE or TOK_ERROR
;-----------------------------------------------------------------------------
global lexer_scan_attr_value
lexer_scan_attr_value:
    push rbp
    mov rbp, rsp
    push rbx

    lea rbx, [xml_lexer]

    call lexer_skip_whitespace

    ; Expect =
    call lexer_peek
    cmp al, '='
    jne .error

    call lexer_advance
    call lexer_skip_whitespace

    ; Expect quote
    call lexer_peek
    cmp al, '"'
    je .double_quote
    cmp al, "'"
    je .single_quote
    jmp .error

.double_quote:
    call lexer_advance
    mov rax, [rbx + Lexer.pos]
    mov [rbx + Lexer.token_start], rax

.dq_loop:
    call lexer_peek
    cmp rax, -1
    je .error
    cmp al, '"'
    je .dq_done
    call lexer_advance
    jmp .dq_loop

.dq_done:
    mov rax, [rbx + Lexer.pos]
    sub rax, [rbx + Lexer.token_start]
    mov [rbx + Lexer.token_len], rax
    call lexer_advance      ; Skip closing quote
    jmp .success

.single_quote:
    call lexer_advance
    mov rax, [rbx + Lexer.pos]
    mov [rbx + Lexer.token_start], rax

.sq_loop:
    call lexer_peek
    cmp rax, -1
    je .error
    cmp al, "'"
    je .sq_done
    call lexer_advance
    jmp .sq_loop

.sq_done:
    mov rax, [rbx + Lexer.pos]
    sub rax, [rbx + Lexer.token_start]
    mov [rbx + Lexer.token_len], rax
    call lexer_advance

.success:
    mov dword [rbx + Lexer.token_type], TOK_ATTR_VALUE
    mov eax, TOK_ATTR_VALUE
    jmp .done

.error:
    mov dword [rbx + Lexer.token_type], TOK_ERROR
    mov eax, TOK_ERROR

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; lexer_get_token_string - Get current token as string
; Output: rax = pointer, rdx = length
;-----------------------------------------------------------------------------
global lexer_get_token_string
lexer_get_token_string:
    lea rcx, [xml_lexer]
    mov rax, [rcx + Lexer.token_start]
    mov rdx, [rcx + Lexer.token_len]
    ret

;-----------------------------------------------------------------------------
; lexer_get_line - Get current line number
; Output: rax = line number
;-----------------------------------------------------------------------------
global lexer_get_line
lexer_get_line:
    lea rcx, [xml_lexer]
    mov rax, [rcx + Lexer.line]
    ret

;-----------------------------------------------------------------------------
; lexer_get_col - Get current column
; Output: rax = column
;-----------------------------------------------------------------------------
global lexer_get_col
lexer_get_col:
    lea rcx, [xml_lexer]
    mov rax, [rcx + Lexer.col]
    ret
