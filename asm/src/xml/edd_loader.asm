; edd_loader.asm - Entity definition loader
; DTRules Assembly Implementation
; Loads entity definitions from parsed XML

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"

extern state
extern heap_alloc
extern xml_node_find_child
extern xml_node_get_attr
extern xml_node_get_child
extern xml_node_get_text
extern entity_alloc
extern string_alloc

;-----------------------------------------------------------------------------
; Entity Definition structure
; Defines the schema for an entity type
;-----------------------------------------------------------------------------
struc EntityDef
    .name:          resq 1  ; Type name (length-prefixed string)
    .attrs:         resq 1  ; Array of attribute definitions
    .attr_count:    resq 1  ; Number of attributes
    .attr_cap:      resq 1  ; Attribute capacity
endstruc

;-----------------------------------------------------------------------------
; Attribute Definition structure
;-----------------------------------------------------------------------------
struc AttrDef
    .name:          resq 1  ; Attribute name
    .type:          resd 1  ; Value type tag
    .required:      resd 1  ; 1 if required, 0 if optional
    .default_val:   resq 1  ; Default value (pointer to Value, or 0)
endstruc

;-----------------------------------------------------------------------------
; Entity definition registry
;-----------------------------------------------------------------------------
section .bss
    align 8
    entity_defs:    resq 128    ; Up to 128 entity definitions
    entity_def_count: resq 1

;-----------------------------------------------------------------------------
; Tag name constants
;-----------------------------------------------------------------------------
section .data
    align 8
    tag_entities:
        dq 8
        db "entities", 0

    tag_entity:
        dq 6
        db "entity", 0

    tag_attribute:
        dq 9
        db "attribute", 0

    attr_name:
        dq 4
        db "name", 0

    attr_type:
        dq 4
        db "type", 0

    attr_required:
        dq 8
        db "required", 0

    attr_default:
        dq 7
        db "default", 0

    ; Type name strings
    type_str_integer:
        dq 7
        db "integer", 0

    type_str_double:
        dq 6
        db "double", 0

    type_str_string:
        dq 6
        db "string", 0

    type_str_boolean:
        dq 7
        db "boolean", 0

    type_str_array:
        dq 5
        db "array", 0

    type_str_entity:
        dq 6
        db "entity", 0

section .text

;-----------------------------------------------------------------------------
; edd_load - Load entity definitions from XML root
; Input: rdi = XML root node
; Output: rax = 0 on success, non-zero on error
;-----------------------------------------------------------------------------
global edd_load
edd_load:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; Root node

    ; Find entities element
    mov rsi, tag_entities
    call xml_node_find_child
    test rax, rax
    jz .no_entities

    mov r12, rax            ; entities element

    ; Iterate through entity children
    mov r13, [r12 + 56]     ; child_count
    mov r12, [r12 + 48]     ; children array
    xor ebx, ebx            ; Index

.load_loop:
    cmp rbx, r13
    jae .done

    mov rax, [r12 + rbx * 8]

    ; Check if it's an entity element
    cmp dword [rax], 1      ; XML_ELEMENT
    jne .next

    ; Load this entity definition
    push rbx
    push r12
    push r13
    mov rdi, rax
    call load_entity_def
    pop r13
    pop r12
    pop rbx

    test eax, eax
    jnz .error

.next:
    inc rbx
    jmp .load_loop

.no_entities:
    xor eax, eax
    jmp .done

.error:
    ; Keep error

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; load_entity_def - Load a single entity definition
; Input: rdi = XML entity element
; Output: rax = 0 on success
;-----------------------------------------------------------------------------
load_entity_def:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14

    mov rbx, rdi            ; XML element

    ; Allocate entity definition
    mov rdi, EntityDef_size
    call heap_alloc
    test rax, rax
    jz .alloc_error

    mov r12, rax            ; EntityDef pointer

    ; Initialize
    mov qword [r12 + EntityDef.name], 0
    mov qword [r12 + EntityDef.attrs], 0
    mov qword [r12 + EntityDef.attr_count], 0
    mov qword [r12 + EntityDef.attr_cap], 0

    ; Get name attribute
    mov rdi, rbx
    mov rsi, attr_name
    call xml_node_get_attr
    test rax, rax
    jz .no_name

    mov [r12 + EntityDef.name], rax

.no_name:
    ; Load attributes
    mov r13, [rbx + 56]     ; child_count
    mov r14, [rbx + 48]     ; children
    xor ecx, ecx            ; Index

.attr_loop:
    cmp rcx, r13
    jae .register

    push rcx
    mov rax, [r14 + rcx * 8]

    ; Check if attribute element
    cmp dword [rax], 1
    jne .next_attr

    ; Load attribute definition
    push r12
    push r13
    push r14
    mov rdi, rax
    mov rsi, r12
    call load_attr_def
    pop r14
    pop r13
    pop r12

.next_attr:
    pop rcx
    inc rcx
    jmp .attr_loop

.register:
    ; Register entity definition
    mov rax, [entity_def_count]
    cmp rax, 128
    jae .alloc_error

    lea rcx, [entity_defs]
    mov [rcx + rax * 8], r12
    inc qword [entity_def_count]

    xor eax, eax
    jmp .done

.alloc_error:
    mov eax, ERR_OUT_OF_MEMORY

.done:
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; load_attr_def - Load an attribute definition
; Input: rdi = XML attribute element, rsi = EntityDef pointer
; Output: rax = 0 on success
;-----------------------------------------------------------------------------
load_attr_def:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13

    mov rbx, rdi            ; XML element
    mov r12, rsi            ; EntityDef

    ; Allocate AttrDef if needed
    mov rax, [r12 + EntityDef.attr_count]
    cmp rax, [r12 + EntityDef.attr_cap]
    jb .have_space

    ; Grow attrs array
    mov r13, [r12 + EntityDef.attr_cap]
    shl r13, 1
    test r13, r13
    jnz .have_new_cap
    mov r13, 8
.have_new_cap:

    imul rdi, r13, AttrDef_size
    call heap_alloc
    test rax, rax
    jz .alloc_error

    ; Copy existing
    mov rdi, rax
    mov rsi, [r12 + EntityDef.attrs]
    mov rcx, [r12 + EntityDef.attr_count]
    imul rcx, AttrDef_size
    push rax
    rep movsb
    pop rax

    mov [r12 + EntityDef.attrs], rax
    mov [r12 + EntityDef.attr_cap], r13

.have_space:
    ; Get pointer to new AttrDef
    mov rax, [r12 + EntityDef.attr_count]
    imul rax, AttrDef_size
    add rax, [r12 + EntityDef.attrs]
    mov r13, rax            ; AttrDef pointer

    ; Get name
    mov rdi, rbx
    mov rsi, attr_name
    call xml_node_get_attr
    mov [r13 + AttrDef.name], rax

    ; Get type
    mov rdi, rbx
    mov rsi, attr_type
    call xml_node_get_attr
    test rax, rax
    jz .default_type

    ; Parse type string
    mov rdi, rax
    call parse_type_name
    mov [r13 + AttrDef.type], eax
    jmp .check_required

.default_type:
    mov dword [r13 + AttrDef.type], VTAG_STRING

.check_required:
    ; Get required attribute
    mov dword [r13 + AttrDef.required], 0
    mov rdi, rbx
    mov rsi, attr_required
    call xml_node_get_attr
    test rax, rax
    jz .no_default

    ; Check if "true"
    mov rcx, [rax]
    cmp rcx, 4
    jne .no_default
    mov edx, [rax + 8]
    cmp edx, 'true'
    jne .no_default
    mov dword [r13 + AttrDef.required], 1

.no_default:
    ; Default value - not implemented yet
    mov qword [r13 + AttrDef.default_val], 0

    ; Increment count
    inc qword [r12 + EntityDef.attr_count]

    xor eax, eax
    jmp .done

.alloc_error:
    mov eax, ERR_OUT_OF_MEMORY

.done:
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; parse_type_name - Parse type name string to tag
; Input: rdi = type name string (length-prefixed)
; Output: eax = type tag
;-----------------------------------------------------------------------------
parse_type_name:
    push rbp
    mov rbp, rsp

    ; Compare against known type names
    mov rsi, type_str_integer
    call string_equal
    test eax, eax
    jnz .is_integer

    mov rdi, [rsp - 8]      ; Reload rdi (was clobbered)
    mov rsi, type_str_double
    call string_equal
    test eax, eax
    jnz .is_double

    mov rdi, [rsp - 8]
    mov rsi, type_str_string
    call string_equal
    test eax, eax
    jnz .is_string

    mov rdi, [rsp - 8]
    mov rsi, type_str_boolean
    call string_equal
    test eax, eax
    jnz .is_boolean

    mov rdi, [rsp - 8]
    mov rsi, type_str_array
    call string_equal
    test eax, eax
    jnz .is_array

    mov rdi, [rsp - 8]
    mov rsi, type_str_entity
    call string_equal
    test eax, eax
    jnz .is_entity

    ; Default to string
.is_string:
    mov eax, VTAG_STRING
    jmp .done

.is_integer:
    mov eax, VTAG_INTEGER
    jmp .done

.is_double:
    mov eax, VTAG_DOUBLE
    jmp .done

.is_boolean:
    mov eax, VTAG_BOOLEAN
    jmp .done

.is_array:
    mov eax, VTAG_ARRAY
    jmp .done

.is_entity:
    mov eax, VTAG_ENTITY

.done:
    pop rbp
    ret

;-----------------------------------------------------------------------------
; string_equal - Compare two length-prefixed strings
; Input: rdi = string 1, rsi = string 2
; Output: eax = 1 if equal, 0 otherwise
;-----------------------------------------------------------------------------
string_equal:
    mov rax, [rdi]
    cmp rax, [rsi]
    jne .not_equal

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
    ret

.not_equal:
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; edd_find - Find entity definition by name
; Input: rdi = name string
; Output: rax = EntityDef pointer, or 0 if not found
;-----------------------------------------------------------------------------
global edd_find
edd_find:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    mov r12, rdi            ; Name to find

    mov rcx, [entity_def_count]
    lea rbx, [entity_defs]
    xor edx, edx

.search:
    cmp rdx, rcx
    jae .not_found

    mov rax, [rbx + rdx * 8]
    test rax, rax
    jz .next

    ; Compare names
    push rcx
    push rdx
    push rax

    mov rdi, [rax + EntityDef.name]
    mov rsi, r12
    call string_equal

    test eax, eax
    jz .next_pop

    ; Found
    pop rax
    add rsp, 16
    jmp .done

.next_pop:
    pop rax
    pop rdx
    pop rcx

.next:
    inc rdx
    jmp .search

.not_found:
    xor eax, eax

.done:
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; edd_get_count - Get number of entity definitions
; Output: rax = count
;-----------------------------------------------------------------------------
global edd_get_count
edd_get_count:
    mov rax, [entity_def_count]
    ret
