; init.asm - Library initialization for CGO integration
; DTRules Assembly Implementation
; Provides state and init functions without _start entry point

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
extern stack_data_init
extern stack_entity_init
extern stack_control_init

;-----------------------------------------------------------------------------
; Global state (exported for CGO)
;-----------------------------------------------------------------------------
section .bss
    align 16
    global state
    state: resb State_size

    ; Temporary value buffers for operations
    global temp_value
    temp_value: resb VALUE_SIZE

    global temp_value2
    temp_value2: resb VALUE_SIZE

    ; Flag to track if memory has been initialized
    memory_initialized: resb 1

;-----------------------------------------------------------------------------
; Library initialization
;-----------------------------------------------------------------------------
section .text

;-----------------------------------------------------------------------------
; lib_init - Initialize the ASM runtime for library use
; Called from Go via CGO before any bytecode execution
; Returns: 0 on success, error code on failure
;-----------------------------------------------------------------------------
global lib_init
lib_init:
    push rbp
    mov rbp, rsp
    push rbx
    push r12
    push r13
    push r14
    push r15

    ; Check if already initialized
    cmp byte [memory_initialized], 1
    je .already_init

    ; Initialize memory via mmap
    call memory_init_lib
    test eax, eax
    jnz .error

    ; Mark as initialized
    mov byte [memory_initialized], 1

.already_init:
    ; Initialize data stack (or reset it)
    call stack_data_init

    ; Initialize entity stack
    call stack_entity_init

    ; Initialize control stack
    call stack_control_init

    ; Clear error state
    mov dword [state + State.error], ERR_NONE
    mov dword [state + State.halted], 0

    ; Success
    xor eax, eax

.exit:
    pop r15
    pop r14
    pop r13
    pop r12
    pop rbx
    pop rbp
    ret

.error:
    mov eax, ERR_OUT_OF_MEMORY
    jmp .exit

;-----------------------------------------------------------------------------
; memory_init_lib - Initialize memory via mmap for library use
; Returns: 0 on success, non-zero on failure
;-----------------------------------------------------------------------------
memory_init_lib:
    push rbp
    mov rbp, rsp
    push rbx

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

    ; Data stack grows DOWNWARD, so:
    ; - base is at the TOP (high address)
    ; - end is the lower limit
    ; - stack pointer starts at base (empty stack)
    mov rbx, rax
    add rbx, DATA_STACK_SIZE * VALUE_SIZE
    mov [state + State.data_stack_base], rbx
    mov [state + State.data_stack], rbx      ; Stack starts empty
    mov [state + State.data_stack_end], rax  ; Lower limit

    ; Store in r12 for fast access
    mov r12, rbx

    ; Allocate entity stack (also grows downward)
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

    ; Allocate control stack (also grows downward)
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

    xor eax, eax            ; Return 0 (success)
    jmp .done

.fail:
    mov eax, -1

.done:
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; lib_vm_execute - CGO-safe wrapper for vm_execute
; Loads r12/r13/r14 from state before calling vm_execute
; Returns: 0 on success, error code on failure
;-----------------------------------------------------------------------------
extern vm_execute
global lib_vm_execute
lib_vm_execute:
    push rbp
    mov rbp, rsp
    push r12
    push r13
    push r14
    push r15

    ; Load stack pointers from state into registers
    ; (vm_execute and opcodes expect these to be set)
    mov r12, [state + State.data_stack]
    mov r13, [state + State.entity_stack]
    mov r14, [state + State.control_stack]

    ; Call the actual vm_execute
    call vm_execute

    ; Save stack pointers back to state
    ; (in case vm_execute modified them)
    mov [state + State.data_stack], r12
    mov [state + State.entity_stack], r13
    mov [state + State.control_stack], r14

    ; Result already in eax from vm_execute

    pop r15
    pop r14
    pop r13
    pop r12
    pop rbp
    ret

;-----------------------------------------------------------------------------
; lib_set_bytecode - Set bytecode for execution
; Input: rdi = pointer to bytecode start
;        rsi = bytecode length
; Returns: 0 on success
;-----------------------------------------------------------------------------
global lib_set_bytecode
lib_set_bytecode:
    mov [state + State.bytecode], rdi
    add rsi, rdi
    mov [state + State.bytecode_end], rsi
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; lib_set_constant_pool - Set constant pool
; Input: rdi = pointer to constant pool (array of Values)
;        rsi = number of constants
; Returns: 0 on success
;-----------------------------------------------------------------------------
global lib_set_constant_pool
lib_set_constant_pool:
    mov [state + State.constant_pool], rdi
    mov [state + State.constant_count], rsi
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; lib_set_name_pool - Set name pool
; Input: rdi = pointer to name pool (array of pointers)
;        rsi = number of names
; Returns: 0 on success
;-----------------------------------------------------------------------------
global lib_set_name_pool
lib_set_name_pool:
    mov [state + State.name_pool], rdi
    mov [state + State.name_count], rsi
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; lib_get_error - Get last error code
; Returns: error code in eax
;-----------------------------------------------------------------------------
global lib_get_error
lib_get_error:
    mov eax, [state + State.error]
    ret

;-----------------------------------------------------------------------------
; lib_clear_error - Clear error state
;-----------------------------------------------------------------------------
global lib_clear_error
lib_clear_error:
    mov dword [state + State.error], ERR_NONE
    ret

;-----------------------------------------------------------------------------
; lib_reset - Reset VM state for new execution
; Clears stacks but keeps heap
; NOTE: For CGO, we reset state directly without using r12/r13/r14 registers
;-----------------------------------------------------------------------------
global lib_reset
lib_reset:
    push rbp
    mov rbp, rsp

    ; Reset data stack pointer to base (direct state manipulation, no r12)
    mov rax, [state + State.data_stack_base]
    mov [state + State.data_stack], rax

    ; Reset entity stack pointer to base
    mov rax, [state + State.entity_stack_base]
    mov [state + State.entity_stack], rax

    ; Reset control stack pointer to base
    mov rax, [state + State.control_stack_base]
    mov [state + State.control_stack], rax

    ; Clear state
    mov dword [state + State.error], ERR_NONE
    mov dword [state + State.halted], 0

    pop rbp
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; lib_get_state_ptr - Get pointer to state structure
; Returns: pointer in rax (for CGO to access state fields directly)
;-----------------------------------------------------------------------------
global lib_get_state_ptr
lib_get_state_ptr:
    lea rax, [state]
    ret

;-----------------------------------------------------------------------------
; lib_data_stack_push - Push a Value onto the data stack
; Input: rdi = pointer to Value (24 bytes)
; Returns: 0 on success, ERR_STACK_OVERFLOW on failure
; Note: Stack grows DOWNWARD (decrementing addresses)
;-----------------------------------------------------------------------------
global lib_data_stack_push
lib_data_stack_push:
    push rbp
    mov rbp, rsp

    ; Check for overflow (stack pointer - VALUE_SIZE would go below end)
    mov rax, [state + State.data_stack]
    sub rax, VALUE_SIZE
    cmp rax, [state + State.data_stack_end]
    jb .overflow

    ; Move stack pointer down
    sub qword [state + State.data_stack], VALUE_SIZE

    ; Copy Value from rdi to stack (stack pointer now points to new slot)
    mov rsi, rdi                    ; source
    mov rcx, [state + State.data_stack]  ; dest

    ; Copy 24 bytes
    mov rax, [rsi]
    mov [rcx], rax
    mov rax, [rsi + 8]
    mov [rcx + 8], rax
    mov rax, [rsi + 16]
    mov [rcx + 16], rax

    xor eax, eax
    pop rbp
    ret

.overflow:
    mov eax, ERR_STACK_OVERFLOW
    pop rbp
    ret

;-----------------------------------------------------------------------------
; lib_data_stack_pop - Pop a Value from the data stack
; Input: rdi = pointer to destination Value (24 bytes)
; Returns: 0 on success, ERR_STACK_UNDERFLOW on failure
; Note: Stack grows DOWNWARD, so pop moves pointer UP
;-----------------------------------------------------------------------------
global lib_data_stack_pop
lib_data_stack_pop:
    push rbp
    mov rbp, rsp

    ; Check for underflow (stack at base means empty)
    mov rax, [state + State.data_stack]
    cmp rax, [state + State.data_stack_base]
    jae .underflow

    ; Copy Value from stack to rdi (stack pointer points to top value)
    mov rsi, [state + State.data_stack]  ; source

    ; Copy 24 bytes
    mov rax, [rsi]
    mov [rdi], rax
    mov rax, [rsi + 8]
    mov [rdi + 8], rax
    mov rax, [rsi + 16]
    mov [rdi + 16], rax

    ; Move stack pointer up (pop)
    add qword [state + State.data_stack], VALUE_SIZE

    xor eax, eax
    pop rbp
    ret

.underflow:
    mov eax, ERR_STACK_UNDERFLOW
    pop rbp
    ret

;-----------------------------------------------------------------------------
; lib_data_stack_depth - Get depth of data stack
; Returns: number of values on stack
; Note: depth = (base - stack) / VALUE_SIZE (since stack grows down)
;-----------------------------------------------------------------------------
global lib_data_stack_depth
lib_data_stack_depth:
    mov rax, [state + State.data_stack_base]
    sub rax, [state + State.data_stack]
    mov rcx, VALUE_SIZE
    xor rdx, rdx
    div rcx
    ret

;-----------------------------------------------------------------------------
; lib_data_stack_peek - Peek at top of data stack without popping
; Input: rdi = pointer to destination Value (24 bytes)
; Returns: 0 on success, ERR_STACK_UNDERFLOW if empty
; Note: Stack pointer points to the top element
;-----------------------------------------------------------------------------
global lib_data_stack_peek
lib_data_stack_peek:
    push rbp
    mov rbp, rsp

    ; Check for empty stack (stack at base means empty)
    mov rax, [state + State.data_stack]
    cmp rax, [state + State.data_stack_base]
    jae .underflow

    ; Stack pointer points to top value
    mov rsi, [state + State.data_stack]

    ; Copy 24 bytes
    mov rax, [rsi]
    mov [rdi], rax
    mov rax, [rsi + 8]
    mov [rdi + 8], rax
    mov rax, [rsi + 16]
    mov [rdi + 16], rax

    xor eax, eax
    pop rbp
    ret

.underflow:
    mov eax, ERR_STACK_UNDERFLOW
    pop rbp
    ret

;-----------------------------------------------------------------------------
; Entity stack operations for CGO
;-----------------------------------------------------------------------------

;-----------------------------------------------------------------------------
; lib_entity_stack_push - Push entity pointer onto entity stack
; Input: rdi = pointer to entity Value (24 bytes)
; Returns: 0 on success, ERR_STACK_OVERFLOW on failure
;-----------------------------------------------------------------------------
global lib_entity_stack_push
lib_entity_stack_push:
    push rbp
    mov rbp, rsp

    ; Check for overflow
    mov rax, [state + State.entity_stack]
    sub rax, 8
    cmp rax, [state + State.entity_stack_end]
    jb .overflow

    ; Push entity pointer (entity is Value at rdi, push pointer to it)
    sub qword [state + State.entity_stack], 8
    mov rax, [state + State.entity_stack]
    mov [rax], rdi

    xor eax, eax
    pop rbp
    ret

.overflow:
    mov eax, ERR_STACK_OVERFLOW
    pop rbp
    ret

;-----------------------------------------------------------------------------
; lib_entity_stack_pop - Pop entity pointer from entity stack
; Returns: pointer in rax, or 0 on underflow
;-----------------------------------------------------------------------------
global lib_entity_stack_pop
lib_entity_stack_pop:
    mov rax, [state + State.entity_stack]
    cmp rax, [state + State.entity_stack_base]
    jae .underflow

    mov rax, [rax]
    add qword [state + State.entity_stack], 8
    ret

.underflow:
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; lib_entity_stack_depth - Get entity stack depth
; Returns: number of entities on stack
;-----------------------------------------------------------------------------
global lib_entity_stack_depth
lib_entity_stack_depth:
    mov rax, [state + State.entity_stack_base]
    sub rax, [state + State.entity_stack]
    shr rax, 3              ; Divide by 8
    ret

;-----------------------------------------------------------------------------
; lib_entity_stack_peek - Peek at entity at index
; Input: rdi = index from top (0 = top)
; Returns: pointer in rax, or 0 if invalid
;-----------------------------------------------------------------------------
global lib_entity_stack_peek
lib_entity_stack_peek:
    shl rdi, 3              ; * 8
    add rdi, [state + State.entity_stack]
    cmp rdi, [state + State.entity_stack_base]
    jae .invalid
    mov rax, [rdi]
    ret
.invalid:
    xor eax, eax
    ret

;-----------------------------------------------------------------------------
; Heap operations for CGO
;-----------------------------------------------------------------------------

;-----------------------------------------------------------------------------
; lib_heap_alloc - Allocate memory from heap
; Input: rdi = size in bytes
; Returns: pointer in rax, or 0 on failure
;-----------------------------------------------------------------------------
global lib_heap_alloc
lib_heap_alloc:
    push rbp
    mov rbp, rsp

    ; Align size to 8 bytes
    add rdi, 7
    and rdi, ~7

    ; Check if enough space
    mov rax, [state + State.heap_ptr]
    add rax, rdi
    cmp rax, [state + State.heap_end]
    ja .fail

    ; Bump allocate
    mov rax, [state + State.heap_ptr]
    add [state + State.heap_ptr], rdi

    pop rbp
    ret

.fail:
    xor eax, eax
    pop rbp
    ret

;-----------------------------------------------------------------------------
; Entity operations for CGO
;-----------------------------------------------------------------------------
extern entity_alloc
extern entity_set_attr
extern entity_get_attr

;-----------------------------------------------------------------------------
; lib_entity_alloc - Allocate new entity
; Input: rdi = type name pointer (can be 0)
; Returns: pointer to entity structure
;-----------------------------------------------------------------------------
global lib_entity_alloc
lib_entity_alloc:
    jmp entity_alloc

;-----------------------------------------------------------------------------
; lib_entity_set_attr - Set entity attribute
; Input: rdi = entity ptr, rsi = name ptr, rdx = value ptr
; Returns: 0 on success
;-----------------------------------------------------------------------------
global lib_entity_set_attr
lib_entity_set_attr:
    jmp entity_set_attr

;-----------------------------------------------------------------------------
; lib_entity_get_attr - Get entity attribute
; Input: rdi = entity ptr, rsi = name ptr
; Returns: pointer to Value or 0
;-----------------------------------------------------------------------------
global lib_entity_get_attr
lib_entity_get_attr:
    jmp entity_get_attr

;-----------------------------------------------------------------------------
; lib_create_string - Create length-prefixed string in heap
; Input: rdi = source string ptr, rsi = length
; Returns: pointer to heap string (length:8 + data)
;-----------------------------------------------------------------------------
global lib_create_string
lib_create_string:
    push rbp
    mov rbp, rsp
    push rbx
    push r12

    mov rbx, rdi            ; Source
    mov r12, rsi            ; Length

    ; Allocate space for length (8) + data + null terminator
    lea rdi, [r12 + 9]
    call lib_heap_alloc
    test rax, rax
    jz .done

    ; Store length
    mov [rax], r12

    ; Copy string data
    push rax
    lea rdi, [rax + 8]
    mov rsi, rbx
    mov rcx, r12
    rep movsb
    mov byte [rdi], 0       ; Null terminate
    pop rax

.done:
    pop r12
    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; Decision Table CGO Bridge Functions
;-----------------------------------------------------------------------------
extern table_alloc
extern table_set_name
extern table_set_root
extern table_register
extern table_find
extern table_execute
extern table_execute_by_name
extern table_clear_registry
extern cnode_alloc
extern cnode_set_condition
extern cnode_set_true_branch
extern cnode_set_false_branch
extern anode_alloc
extern anode_set_actions
extern anode_add_action

;-----------------------------------------------------------------------------
; lib_table_alloc - Allocate a new decision table
; Returns: pointer to DecisionTable
;-----------------------------------------------------------------------------
global lib_table_alloc
lib_table_alloc:
    jmp table_alloc

;-----------------------------------------------------------------------------
; lib_table_set_name - Set table name
; Input: rdi = table ptr, rsi = name string
;-----------------------------------------------------------------------------
global lib_table_set_name
lib_table_set_name:
    jmp table_set_name

;-----------------------------------------------------------------------------
; lib_table_set_root - Set root node
; Input: rdi = table ptr, rsi = root node, rdx = is_cnode (0=anode, 1=cnode)
;-----------------------------------------------------------------------------
global lib_table_set_root
lib_table_set_root:
    jmp table_set_root

;-----------------------------------------------------------------------------
; lib_table_register - Register table in global registry
; Input: rdi = table pointer
; Returns: 0 on success
;-----------------------------------------------------------------------------
global lib_table_register
lib_table_register:
    jmp table_register

;-----------------------------------------------------------------------------
; lib_table_execute_by_name - Execute table by name
; Input: rdi = table name string
; Returns: 0 on success
;-----------------------------------------------------------------------------
global lib_table_execute_by_name
lib_table_execute_by_name:
    push rbp
    mov rbp, rsp
    push r12
    push r13
    push r14
    push r15

    ; Load stack pointers from state into registers
    mov r12, [state + State.data_stack]
    mov r13, [state + State.entity_stack]
    mov r14, [state + State.control_stack]

    call table_execute_by_name

    ; Save stack pointers back to state
    mov [state + State.data_stack], r12
    mov [state + State.entity_stack], r13
    mov [state + State.control_stack], r14

    pop r15
    pop r14
    pop r13
    pop r12
    pop rbp
    ret

;-----------------------------------------------------------------------------
; lib_table_clear_registry - Clear all registered tables
;-----------------------------------------------------------------------------
global lib_table_clear_registry
lib_table_clear_registry:
    jmp table_clear_registry

;-----------------------------------------------------------------------------
; lib_cnode_alloc - Allocate a condition node
; Returns: pointer to CNode
;-----------------------------------------------------------------------------
global lib_cnode_alloc
lib_cnode_alloc:
    jmp cnode_alloc

;-----------------------------------------------------------------------------
; lib_cnode_set_condition - Set condition bytecode
; Input: rdi = CNode ptr, rsi = bytecode ptr, rdx = length
;-----------------------------------------------------------------------------
global lib_cnode_set_condition
lib_cnode_set_condition:
    jmp cnode_set_condition

;-----------------------------------------------------------------------------
; lib_cnode_set_true_branch - Set true branch
; Input: rdi = CNode ptr, rsi = branch node, rdx = is_cnode
;-----------------------------------------------------------------------------
global lib_cnode_set_true_branch
lib_cnode_set_true_branch:
    jmp cnode_set_true_branch

;-----------------------------------------------------------------------------
; lib_cnode_set_false_branch - Set false branch
; Input: rdi = CNode ptr, rsi = branch node, rdx = is_cnode
;-----------------------------------------------------------------------------
global lib_cnode_set_false_branch
lib_cnode_set_false_branch:
    jmp cnode_set_false_branch

;-----------------------------------------------------------------------------
; lib_anode_alloc - Allocate an action node
; Returns: pointer to ANode
;-----------------------------------------------------------------------------
global lib_anode_alloc
lib_anode_alloc:
    jmp anode_alloc

;-----------------------------------------------------------------------------
; lib_anode_add_action - Add action bytecode to ANode
; Input: rdi = ANode ptr, rsi = bytecode ptr, rdx = length
; Returns: 0 on success
;-----------------------------------------------------------------------------
global lib_anode_add_action
lib_anode_add_action:
    jmp anode_add_action
