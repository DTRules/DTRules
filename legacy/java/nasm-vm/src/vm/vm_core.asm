; vm_core.asm - VM execution core
; DTRules NASM VM Implementation
;
; This is the main dispatch loop for the bytecode VM.
; Design principles:
; - Single entry/exit (no CGO-style per-operation calls)
; - Jump table dispatch: jmp [table + opcode*8], not if-else chains
; - Each handler ends with jmp dispatch, not ret
; - Trace recording writes to buffer, not function calls

bits 64
default rel

%include "include/syscalls.inc"
%include "include/constants.inc"
%include "include/macros.inc"
%include "include/state.inc"

;-----------------------------------------------------------------------------
; External references
;-----------------------------------------------------------------------------
extern state
extern opcode_table
extern stack_data_push_integer
extern stack_data_push_boolean
extern stack_data_push_null
extern stack_data_pop
extern stack_data_peek

;-----------------------------------------------------------------------------
; Export symbols
;-----------------------------------------------------------------------------
global vm_execute
global vm_dispatch
global read_varint
global read_svarint

;-----------------------------------------------------------------------------
; Read-only data
;-----------------------------------------------------------------------------
section .rodata
    err_invalid_opcode: db "Error: invalid opcode", 10, 0
    err_invalid_opcode_len equ $ - err_invalid_opcode - 1

;-----------------------------------------------------------------------------
; Code section
;-----------------------------------------------------------------------------
section .text

;-----------------------------------------------------------------------------
; vm_execute - Main execution loop
;
; This is the main entry point for executing bytecode.
; The bytecode should already be loaded in state.bytecode/bytecode_end.
;
; Register usage during execution:
;   r12 = data stack pointer
;   r13 = entity stack pointer
;   r14 = control stack pointer
;   r15 = state pointer
;   rbx = bytecode instruction pointer (IP)
;
; Returns: 0 on success, error code on failure
;-----------------------------------------------------------------------------
vm_execute:
    push rbp
    mov rbp, rsp
    push rbx
    ; NOTE: Do NOT save r12-r15 here - they are VM registers that
    ; must persist across the execution session

    ; Load state pointer into r15
    lea r15, [state]

    ; Load bytecode IP into rbx
    mov rbx, [r15 + State.bytecode]

    ; Clear halted flag
    mov dword [r15 + State.halted], 0

    ; Fall through to dispatch loop
    jmp vm_dispatch

;-----------------------------------------------------------------------------
; vm_dispatch - Main dispatch loop (internal)
;
; Fetches next opcode and dispatches to handler.
; Handlers should jump back here (not call/ret) for efficiency.
;
; Entry: rbx = bytecode IP, r15 = state pointer
; Exit: returns when halted or error
;-----------------------------------------------------------------------------
vm_dispatch:
    ; Check if halted
    cmp dword [r15 + State.halted], 0
    jne .done

    ; Check bytecode bounds
    cmp rbx, [r15 + State.bytecode_end]
    jae .done

    ; Fetch opcode
    movzx eax, byte [rbx]
    inc rbx                     ; Advance IP

    ; Dispatch through jump table
    ; Each handler should: process opcode, then jmp vm_dispatch
    lea rcx, [opcode_table]
    jmp [rcx + rax * 8]

.error:
    mov eax, [r15 + State.error]
    jmp .exit

.done:
    xor eax, eax

.exit:
    ; Save bytecode pointer back to state
    mov [r15 + State.bytecode], rbx

    pop rbx
    pop rbp
    ret

;-----------------------------------------------------------------------------
; Bytecode reading helpers
;
; These are called by opcode handlers to read operands from bytecode.
; rbx points to current position, advanced after read.
;-----------------------------------------------------------------------------

;-----------------------------------------------------------------------------
; read_varint - Read unsigned varint from bytecode
;
; Input: rbx = pointer to bytecode
; Output: rax = value, rbx advanced past varint
; Clobbers: rcx, rdx, r8
;-----------------------------------------------------------------------------
read_varint:
    xor rax, rax            ; result = 0
    xor ecx, ecx            ; shift = 0

.loop:
    movzx edx, byte [rbx]   ; read byte
    inc rbx                 ; advance pointer

    mov r8d, edx            ; save byte
    and r8d, 0x7F           ; mask off continuation bit
    mov r9, r8
    shl r9, cl              ; shift by current amount
    or rax, r9              ; add to result

    test dl, 0x80           ; continuation bit set?
    jz .done                ; no, we're done

    add cl, 7               ; shift += 7
    cmp cl, 64              ; overflow check
    jb .loop

.done:
    ret

;-----------------------------------------------------------------------------
; read_svarint - Read signed varint (zigzag encoded)
;
; Input: rbx = pointer to bytecode
; Output: rax = value (signed), rbx advanced
;-----------------------------------------------------------------------------
read_svarint:
    call read_varint
    ; Zigzag decode: (n >> 1) ^ -(n & 1)
    mov rcx, rax
    shr rax, 1
    and ecx, 1
    neg rcx
    xor rax, rcx
    ret

;-----------------------------------------------------------------------------
; read_qword - Read 8-byte value from bytecode
;
; Input: rbx = pointer to bytecode
; Output: rax = value, rbx advanced by 8
;-----------------------------------------------------------------------------
global read_qword
read_qword:
    mov rax, [rbx]
    add rbx, 8
    ret

;-----------------------------------------------------------------------------
; trace_record - Record opcode execution to trace buffer
;
; This writes trace data directly to a buffer (no function calls).
; Format: [timestamp:8][opcode:1][stack_depth:4][...]
;
; Input: al = opcode
; Clobbers: rcx, rdx
;-----------------------------------------------------------------------------
global trace_record
trace_record:
    ; Check if tracing is enabled
    cmp byte [r15 + State.trace_enabled], 0
    je .skip

    ; Get trace buffer position
    mov rcx, [r15 + State.trace_pos]

    ; Check buffer space
    lea rdx, [rcx + 16]
    cmp rdx, [r15 + State.trace_end]
    jae .skip

    ; Write opcode
    mov [rcx], al

    ; Update position
    add qword [r15 + State.trace_pos], 1

.skip:
    ret

;-----------------------------------------------------------------------------
; op_invalid - Handler for invalid/unimplemented opcodes
;
; Sets error and halts the VM.
;-----------------------------------------------------------------------------
global op_invalid
op_invalid:
    mov dword [r15 + State.error], ERR_INVALID_OPCODE
    mov dword [r15 + State.halted], 1
    jmp vm_dispatch
