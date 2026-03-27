; start.asm - Entry point and initialization
; DTRules Assembly Implementation
; No libc - direct syscalls only

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

;-----------------------------------------------------------------------------
; External references
;-----------------------------------------------------------------------------
extern heap_init
extern heap_alloc
extern value_new_integer
extern value_new_boolean
extern value_new_null
extern stack_data_init
extern stack_data_push
extern stack_data_pop
extern stack_entity_init
extern stack_control_init
extern vm_execute
extern print_string
extern print_newline
extern print_integer
extern file_read_all
extern xml_parse
extern dt_load

;-----------------------------------------------------------------------------
; Global state
;-----------------------------------------------------------------------------
section .bss
    align 16
    global state
    state: resb State_size

    ; Temporary value buffer for operations
    global temp_value
    temp_value: resb VALUE_SIZE

    global temp_value2
    temp_value2: resb VALUE_SIZE

;-----------------------------------------------------------------------------
; Read-only data
;-----------------------------------------------------------------------------
section .rodata
    banner: db "DTRules x86-64 Assembly Engine v0.1", 10, 0
    banner_len equ $ - banner - 1

    usage_msg: db "Usage: dtrules-asm <rules.xml> [input.json]", 10, 0
    usage_len equ $ - usage_msg - 1

    init_msg: db "Initializing...", 10, 0
    init_len equ $ - init_msg - 1

    ready_msg: db "Engine ready.", 10, 0
    ready_len equ $ - ready_msg - 1

    err_mmap: db "Error: mmap failed", 10, 0
    err_mmap_len equ $ - err_mmap - 1

    err_load: db "Error: failed to load rules", 10, 0
    err_load_len equ $ - err_load - 1

    newline: db 10

;-----------------------------------------------------------------------------
; Entry point
;-----------------------------------------------------------------------------
section .text
    global _start

_start:
    ; Stack at entry:
    ;   [rsp]     = argc
    ;   [rsp+8]   = argv[0]
    ;   [rsp+16]  = argv[1]
    ;   ...

    ; Save argc/argv
    mov rax, [rsp]          ; argc
    mov [state + State.argc], rax
    lea rax, [rsp + 8]      ; argv
    mov [state + State.argv], rax

    ; Print banner
    mov rax, SYS_WRITE
    mov rdi, STDOUT
    lea rsi, [banner]
    mov rdx, banner_len
    syscall

    ; Initialize memory subsystem
    call memory_init
    test rax, rax
    js .mmap_error

    ; Initialize stacks
    call stack_data_init
    call stack_entity_init
    call stack_control_init

    ; Initialize name table
    call name_table_init

    ; Check command line arguments
    mov rax, [state + State.argc]
    cmp rax, 2
    jl .show_usage

    ; Load rules file (argv[1])
    mov rax, [state + State.argv]
    mov rdi, [rax + 8]      ; argv[1]
    call load_rules
    test rax, rax
    jnz .load_error

    ; Execute the loaded rules
    call vm_execute

    ; Exit with success
    xor edi, edi
    jmp .exit

.show_usage:
    mov rax, SYS_WRITE
    mov rdi, STDOUT
    lea rsi, [usage_msg]
    mov rdx, usage_len
    syscall
    mov edi, 1
    jmp .exit

.mmap_error:
    mov rax, SYS_WRITE
    mov rdi, STDERR
    lea rsi, [err_mmap]
    mov rdx, err_mmap_len
    syscall
    mov edi, 1
    jmp .exit

.load_error:
    mov rax, SYS_WRITE
    mov rdi, STDERR
    lea rsi, [err_load]
    mov rdx, err_load_len
    syscall
    mov edi, 1
    jmp .exit

.exit:
    mov eax, SYS_EXIT_GROUP
    syscall

;-----------------------------------------------------------------------------
; memory_init - Initialize memory via mmap
; Returns: 0 on success, -1 on failure
;-----------------------------------------------------------------------------
memory_init:
    push rbp
    mov rbp, rsp
    push rbx

    ; Calculate total memory needed:
    ; - Data stack: DATA_STACK_SIZE * VALUE_SIZE
    ; - Entity stack: ENTITY_STACK_SIZE * 8
    ; - Control stack: CONTROL_STACK_SIZE * 32
    ; - Heap: HEAP_SIZE

    ; Allocate data stack
    mov rax, SYS_MMAP
    xor edi, edi            ; addr = NULL (let kernel choose)
    mov esi, DATA_STACK_SIZE * VALUE_SIZE
    mov edx, PROT_READ | PROT_WRITE
    mov r10d, MAP_PRIVATE | MAP_ANONYMOUS
    mov r8d, -1             ; fd = -1 (anonymous)
    xor r9d, r9d            ; offset = 0
    syscall

    cmp rax, -1
    je .fail

    ; Data stack grows downward, so base is at end
    mov rbx, rax
    add rbx, DATA_STACK_SIZE * VALUE_SIZE
    mov [state + State.data_stack_base], rbx
    mov [state + State.data_stack], rbx      ; Stack starts empty
    mov [state + State.data_stack_end], rax  ; Limit (can't go below this)

    ; Store in r12 for fast access
    mov r12, rbx

    ; Allocate entity stack
    mov rax, SYS_MMAP
    xor edi, edi
    mov esi, ENTITY_STACK_SIZE * 8
    mov edx, PROT_READ | PROT_WRITE
    mov r10d, MAP_PRIVATE | MAP_ANONYMOUS
    mov r8d, -1
    xor r9d, r9d
    syscall

    cmp rax, -1
    je .fail

    mov rbx, rax
    add rbx, ENTITY_STACK_SIZE * 8
    mov [state + State.entity_stack_base], rbx
    mov [state + State.entity_stack], rbx
    mov [state + State.entity_stack_end], rax
    mov r13, rbx

    ; Allocate control stack
    mov rax, SYS_MMAP
    xor edi, edi
    mov esi, CONTROL_STACK_SIZE * 32
    mov edx, PROT_READ | PROT_WRITE
    mov r10d, MAP_PRIVATE | MAP_ANONYMOUS
    mov r8d, -1
    xor r9d, r9d
    syscall

    cmp rax, -1
    je .fail

    mov rbx, rax
    add rbx, CONTROL_STACK_SIZE * 32
    mov [state + State.control_stack_base], rbx
    mov [state + State.control_stack], rbx
    mov [state + State.control_stack_end], rax
    mov r14, rbx

    ; Allocate heap
    mov rax, SYS_MMAP
    xor edi, edi
    mov esi, HEAP_SIZE
    mov edx, PROT_READ | PROT_WRITE
    mov r10d, MAP_PRIVATE | MAP_ANONYMOUS
    mov r8d, -1
    xor r9d, r9d
    syscall

    cmp rax, -1
    je .fail

    mov [state + State.heap_base], rax
    mov [state + State.heap_ptr], rax       ; Bump pointer starts at base
    add rax, HEAP_SIZE
    mov [state + State.heap_end], rax

    ; Clear error state
    mov dword [state + State.error], ERR_NONE
    mov dword [state + State.halted], 0

    ; Store state pointer in r15
    lea r15, [state]

    xor eax, eax            ; Return 0 (success)
    jmp .done

.fail:
    mov eax, -1

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; name_table_init - Initialize the name interning hash table
;-----------------------------------------------------------------------------
name_table_init:
    push rbp
    mov rbp, rsp

    ; Allocate hash table: NAME_TABLE_SIZE * 16 bytes per entry
    ; Each entry: [hash:8][ptr:8]
    mov rdi, NAME_TABLE_SIZE * 16
    call heap_alloc
    mov [state + State.name_table], rax

    ; Zero the table
    mov rdi, rax
    mov rcx, NAME_TABLE_SIZE * 16
    xor al, al
    rep stosb

    pop rbp
    ret

;-----------------------------------------------------------------------------
; load_rules - Load rules from XML file
; Input: rdi = filename
; Returns: 0 on success, non-zero on error
;-----------------------------------------------------------------------------
load_rules:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    mov rbx, rdi            ; Save filename

    ; Read entire file
    mov rdi, rbx
    call file_read_all
    test rax, rax
    jz .fail

    mov r12, rax            ; Save buffer pointer

    ; Parse XML
    mov rdi, r12
    mov rsi, rdx            ; Length returned in rdx
    call xml_parse
    test rax, rax
    jz .fail

    ; Load decision tables from parsed XML
    mov rdi, rax
    call dt_load
    test rax, rax
    jnz .fail

    xor eax, eax
    jmp .done

.fail:
    mov eax, 1

.done:
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; sys_exit - Exit with code
; Input: edi = exit code
;-----------------------------------------------------------------------------
global sys_exit
sys_exit:
    mov eax, SYS_EXIT_GROUP
    syscall
    ; Never returns
