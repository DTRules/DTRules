; parser.asm - XML DOM builder
; DTRules Assembly Implementation
; Builds a tree of XML nodes from tokens

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

extern state
extern heap_alloc
extern lexer_init
extern lexer_next_token
extern lexer_scan_attribute
extern lexer_scan_attr_value
extern lexer_get_token_string
extern lexer_skip_whitespace
extern string_alloc

;-----------------------------------------------------------------------------
; XML Node structure
;-----------------------------------------------------------------------------
struc XMLNode
    .type:          resd 1  ; Node type
    .pad:           resd 1  ; Padding
    .name:          resq 1  ; Tag name (length-prefixed string)
    .text:          resq 1  ; Text content (for text nodes)
    .attrs:         resq 1  ; Attributes array pointer
    .attr_count:    resq 1  ; Number of attributes
    .children:      resq 1  ; Children array pointer
    .child_count:   resq 1  ; Number of children
    .child_cap:     resq 1  ; Children capacity
    .parent:        resq 1  ; Parent node pointer
endstruc

;-----------------------------------------------------------------------------
; XML Attribute structure
;-----------------------------------------------------------------------------
struc XMLAttr
    .name:          resq 1  ; Attribute name (length-prefixed string)
    .value:         resq 1  ; Attribute value (length-prefixed string)
endstruc

;-----------------------------------------------------------------------------
; Node types
;-----------------------------------------------------------------------------
XML_ELEMENT     equ 1
XML_TEXT        equ 2
XML_COMMENT     equ 3
XML_CDATA       equ 4

;-----------------------------------------------------------------------------
; Token types (from lexer)
;-----------------------------------------------------------------------------
TOK_EOF         equ 0
TOK_TEXT        equ 1
TOK_OPEN_TAG    equ 2
TOK_CLOSE_TAG   equ 3
TOK_SELF_CLOSE  equ 4
TOK_TAG_END     equ 5
TOK_ATTR_NAME   equ 6
TOK_ATTR_VALUE  equ 7
TOK_COMMENT     equ 8
TOK_CDATA       equ 9
TOK_PI          equ 10
TOK_DOCTYPE     equ 11
TOK_ERROR       equ 255

;-----------------------------------------------------------------------------
; Parser state
;-----------------------------------------------------------------------------
section .bss
    align 8
    parser_root:    resq 1  ; Root node
    parser_current: resq 1  ; Current node being built

section .text

;-----------------------------------------------------------------------------
; xml_node_alloc - Allocate a new XML node
; Input: rdi = node type
; Output: rax = node pointer
;-----------------------------------------------------------------------------
global xml_node_alloc
xml_node_alloc:
    push rbp
    mov rbp, rsp
    push rbx

    mov rbx, rdi            ; Save type

    ; Allocate node
    mov rdi, XMLNode_size
    call heap_alloc
    test rax, rax
    jz .done

    ; Initialize
    mov dword [rax + XMLNode.type], ebx
    mov qword [rax + XMLNode.name], 0
    mov qword [rax + XMLNode.text], 0
    mov qword [rax + XMLNode.attrs], 0
    mov qword [rax + XMLNode.attr_count], 0
    mov qword [rax + XMLNode.children], 0
    mov qword [rax + XMLNode.child_count], 0
    mov qword [rax + XMLNode.child_cap], 0
    mov qword [rax + XMLNode.parent], 0

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; xml_node_add_child - Add child to node
; Input: rdi = parent node, rsi = child node
; Output: rax = 0 on success
;-----------------------------------------------------------------------------
global xml_node_add_child
xml_node_add_child:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; Parent
    mov r12, rsi            ; Child

    ; Set parent pointer
    mov [r12 + XMLNode.parent], rbx

    ; Check capacity
    mov rax, [rbx + XMLNode.child_count]
    cmp rax, [rbx + XMLNode.child_cap]
    jb .have_space

    ; Need to grow
    mov r13, [rbx + XMLNode.child_cap]
    shl r13, 1
    test r13, r13
    jnz .have_new_cap
    mov r13, 8              ; Initial capacity
.have_new_cap:

    ; Allocate new array
    imul rdi, r13, 8
    call heap_alloc
    test rax, rax
    jz .fail

    ; Copy existing children
    mov rdi, rax
    mov rsi, [rbx + XMLNode.children]
    mov rcx, [rbx + XMLNode.child_count]
    shl rcx, 3
    push rax
    rep movsb
    pop rax

    mov [rbx + XMLNode.children], rax
    mov [rbx + XMLNode.child_cap], r13

.have_space:
    ; Add child
    mov rax, [rbx + XMLNode.child_count]
    mov rcx, [rbx + XMLNode.children]
    mov [rcx + rax * 8], r12
    inc qword [rbx + XMLNode.child_count]

    xor eax, eax
    jmp .done

.fail:
    mov eax, ERR_OUT_OF_MEMORY

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; xml_node_add_attr - Add attribute to node
; Input: rdi = node, rsi = name string, rdx = value string
; Output: rax = 0 on success
;-----------------------------------------------------------------------------
global xml_node_add_attr
xml_node_add_attr:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14

    mov rbx, rdi            ; Node
    mov r12, rsi            ; Name
    mov r13, rdx            ; Value

    ; Current attribute count
    mov r14, [rbx + XMLNode.attr_count]

    ; Check if we need to allocate/grow
    test r14, r14
    jnz .have_attrs

    ; First attribute - allocate array
    mov rdi, 8 * XMLAttr_size  ; Start with 8 slots
    call heap_alloc
    test rax, rax
    jz .fail
    mov [rbx + XMLNode.attrs], rax

.have_attrs:
    ; Add attribute at end
    mov rax, [rbx + XMLNode.attrs]
    imul rcx, r14, XMLAttr_size
    add rax, rcx

    mov [rax + XMLAttr.name], r12
    mov [rax + XMLAttr.value], r13

    inc qword [rbx + XMLNode.attr_count]

    xor eax, eax
    jmp .done

.fail:
    mov eax, ERR_OUT_OF_MEMORY

.done:
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; copy_token_string - Copy current token to heap string
; Output: rax = length-prefixed string pointer
;-----------------------------------------------------------------------------
copy_token_string:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    call lexer_get_token_string
    mov rbx, rax            ; Source
    mov r12, rdx            ; Length

    ; Allocate string
    mov rdi, r12
    call string_alloc
    test rax, rax
    jz .done

    push rax

    ; Copy data
    lea rdi, [rax + 8]
    mov rsi, rbx
    mov rcx, r12
    rep movsb

    pop rax

.done:
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; xml_parse - Parse XML string into DOM tree
; Input: rdi = input string, rsi = length
; Output: rax = root node, or 0 on error
;-----------------------------------------------------------------------------
global xml_parse
xml_parse:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    ; Initialize lexer
    call lexer_init

    ; Create root node (document)
    mov rdi, XML_ELEMENT
    call xml_node_alloc
    test rax, rax
    jz .error

    mov [parser_root], rax
    mov [parser_current], rax

.parse_loop:
    call lexer_next_token

    cmp eax, TOK_EOF
    je .done
    cmp eax, TOK_ERROR
    je .error

    cmp eax, TOK_OPEN_TAG
    je .open_tag
    cmp eax, TOK_CLOSE_TAG
    je .close_tag
    cmp eax, TOK_TEXT
    je .text_node
    cmp eax, TOK_COMMENT
    je .comment_node
    cmp eax, TOK_CDATA
    je .cdata_node
    cmp eax, TOK_PI
    je .parse_loop          ; Skip processing instructions
    cmp eax, TOK_DOCTYPE
    je .parse_loop          ; Skip DOCTYPE

    jmp .parse_loop

.open_tag:
    ; Create element node
    mov rdi, XML_ELEMENT
    call xml_node_alloc
    test rax, rax
    jz .error

    mov rbx, rax            ; New node

    ; Copy tag name
    call copy_token_string
    mov [rbx + XMLNode.name], rax

    ; Add as child of current
    mov rdi, [parser_current]
    mov rsi, rbx
    call xml_node_add_child

    ; Parse attributes
.attr_loop:
    call lexer_scan_attribute

    cmp eax, TOK_TAG_END
    je .tag_opened
    cmp eax, TOK_SELF_CLOSE
    je .self_closed
    cmp eax, TOK_ATTR_NAME
    jne .error

    ; Save attribute name
    call copy_token_string
    mov r12, rax

    ; Get attribute value
    call lexer_scan_attr_value
    cmp eax, TOK_ATTR_VALUE
    jne .error

    call copy_token_string
    mov r13, rax

    ; Add attribute
    mov rdi, rbx
    mov rsi, r12
    mov rdx, r13
    call xml_node_add_attr

    jmp .attr_loop

.tag_opened:
    ; Move into this node
    mov [parser_current], rbx
    jmp .parse_loop

.self_closed:
    ; Node is complete, don't move into it
    jmp .parse_loop

.close_tag:
    ; Move back to parent
    mov rax, [parser_current]
    mov rax, [rax + XMLNode.parent]
    test rax, rax
    jz .error               ; Unmatched close tag
    mov [parser_current], rax

    ; Skip to tag end
.skip_to_end:
    call lexer_scan_attribute
    cmp eax, TOK_TAG_END
    jne .skip_to_end

    jmp .parse_loop

.text_node:
    ; Skip whitespace-only text
    call lexer_get_token_string
    ; Quick check - if all whitespace, skip
    mov rcx, rdx
    mov rsi, rax
    xor edi, edi            ; Non-whitespace flag

.check_ws:
    test rcx, rcx
    jz .check_ws_done
    movzx eax, byte [rsi]
    cmp al, ' '
    je .next_ws
    cmp al, 9
    je .next_ws
    cmp al, 10
    je .next_ws
    cmp al, 13
    je .next_ws
    mov edi, 1              ; Found non-whitespace
    jmp .check_ws_done
.next_ws:
    inc rsi
    dec rcx
    jmp .check_ws
.check_ws_done:

    test edi, edi
    jz .parse_loop          ; Skip whitespace-only

    ; Create text node
    mov rdi, XML_TEXT
    call xml_node_alloc
    test rax, rax
    jz .error

    mov rbx, rax

    ; Copy text
    call copy_token_string
    mov [rbx + XMLNode.text], rax

    ; Add as child
    mov rdi, [parser_current]
    mov rsi, rbx
    call xml_node_add_child

    jmp .parse_loop

.comment_node:
    ; Create comment node
    mov rdi, XML_COMMENT
    call xml_node_alloc
    test rax, rax
    jz .error

    mov rbx, rax

    call copy_token_string
    mov [rbx + XMLNode.text], rax

    mov rdi, [parser_current]
    mov rsi, rbx
    call xml_node_add_child

    jmp .parse_loop

.cdata_node:
    ; Create CDATA node
    mov rdi, XML_CDATA
    call xml_node_alloc
    test rax, rax
    jz .error

    mov rbx, rax

    call copy_token_string
    mov [rbx + XMLNode.text], rax

    mov rdi, [parser_current]
    mov rsi, rbx
    call xml_node_add_child

    jmp .parse_loop

.done:
    mov rax, [parser_root]
    jmp .exit

.error:
    mov dword [state + State.error], ERR_PARSE_ERROR
    xor eax, eax

.exit:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; xml_node_get_child - Get child by index
; Input: rdi = node, rsi = index
; Output: rax = child node, or 0 if invalid
;-----------------------------------------------------------------------------
global xml_node_get_child
xml_node_get_child:
    cmp rsi, [rdi + XMLNode.child_count]
    jae .invalid

    mov rax, [rdi + XMLNode.children]
    mov rax, [rax + rsi * 8]
    ret

.invalid:
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; xml_node_find_child - Find child by tag name
; Input: rdi = node, rsi = name string
; Output: rax = child node, or 0 if not found
;-----------------------------------------------------------------------------
global xml_node_find_child
xml_node_find_child:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; Parent
    mov r12, rsi            ; Name to find

    mov r13, [rbx + XMLNode.child_count]
    mov rbx, [rbx + XMLNode.children]
    xor ecx, ecx

.search:
    cmp rcx, r13
    jae .not_found

    mov rax, [rbx + rcx * 8]

    ; Check if element
    cmp dword [rax + XMLNode.type], XML_ELEMENT
    jne .next

    ; Compare names
    push rcx
    push rax
    mov rdi, [rax + XMLNode.name]
    mov rsi, r12
    ; Simple comparison of length-prefixed strings
    mov rdx, [rdi]
    cmp rdx, [rsi]
    jne .next_pop

    add rdi, 8
    add rsi, 8
    mov rcx, rdx
    repe cmpsb
    jne .next_pop

    ; Found
    pop rax
    add rsp, 8
    jmp .done

.next_pop:
    pop rax
    pop rcx

.next:
    inc rcx
    jmp .search

.not_found:
    xor eax, eax

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; xml_node_get_attr - Get attribute value by name
; Input: rdi = node, rsi = attribute name
; Output: rax = value string, or 0 if not found
;-----------------------------------------------------------------------------
global xml_node_get_attr
xml_node_get_attr:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; Node
    mov r12, rsi            ; Name to find

    mov r13, [rbx + XMLNode.attr_count]
    mov rbx, [rbx + XMLNode.attrs]
    xor ecx, ecx

.search:
    cmp rcx, r13
    jae .not_found

    ; Get attribute at index
    imul rax, rcx, XMLAttr_size
    add rax, rbx

    ; Compare name
    push rcx
    push rax
    mov rdi, [rax + XMLAttr.name]
    mov rsi, r12

    mov rdx, [rdi]
    cmp rdx, [rsi]
    jne .next_pop

    add rdi, 8
    add rsi, 8
    mov rcx, rdx
    repe cmpsb
    jne .next_pop

    ; Found - return value
    pop rax
    mov rax, [rax + XMLAttr.value]
    add rsp, 8
    jmp .done

.next_pop:
    pop rax
    pop rcx

    inc rcx
    jmp .search

.not_found:
    xor eax, eax

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; xml_node_get_text - Get text content of node
; Input: rdi = node
; Output: rax = text string (concatenated from text children)
;-----------------------------------------------------------------------------
global xml_node_get_text
xml_node_get_text:
    ; For simplicity, return text of first text child
    push rbp
    mov rbp, rsp
    push rbx

    mov rbx, rdi

    ; If this is a text node, return its text
    cmp dword [rbx + XMLNode.type], XML_TEXT
    jne .search_children

    mov rax, [rbx + XMLNode.text]
    jmp .done

.search_children:
    mov rcx, [rbx + XMLNode.child_count]
    mov rbx, [rbx + XMLNode.children]
    xor edx, edx

.search:
    cmp rdx, rcx
    jae .not_found

    mov rax, [rbx + rdx * 8]
    cmp dword [rax + XMLNode.type], XML_TEXT
    je .found_text

    inc rdx
    jmp .search

.found_text:
    mov rax, [rax + XMLNode.text]
    jmp .done

.not_found:
    xor eax, eax

.done:
    pop rbx
    pop rbp
    ret
