; DTRules NASM VM - Trace Recording Implementation
; Copyright 2024 Paul Snow - Apache License 2.0
;
; This file implements trace recording for the VM.
; Trace recording writes events to a pre-allocated buffer without
; any function calls during execution (lock-free, no allocations).
;
; Trace entry format (variable length):
;   +0: event_type (1 byte)  - TRACE_* constant
;   +1: timestamp (4 bytes)  - low 32 bits of rdtsc
;   +5: payload_len (2 bytes) - length of payload data
;   +7: payload (variable)   - event-specific data
;
; Design principles:
;   - No function calls from handlers (inline everything)
;   - No memory allocations (pre-allocated buffer)
;   - rdtsc for timestamping (fast, monotonic)
;   - Graceful overflow handling (stops recording, sets flag)

%include "dtrules.inc"

section .data

section .bss

section .text

; ===========================================================================
; trace_record_opcode - Record opcode execution to trace buffer
; ===========================================================================
; Input:
;   rdi - pointer to VMContext
;   esi - opcode value (8 bits used)
;
; Output:
;   None
;
; Clobbers: rax, rcx, rdx, r10, r11
;
; This is called from the dispatch loop when VM_FLAG_TRACE is set.
; It writes a minimal trace entry for the opcode.
;
; Entry format for TRACE_OPCODE (9 bytes total):
;   +0: TRACE_OPCODE (1 byte)
;   +1: timestamp (4 bytes)
;   +5: payload_len = 2 (2 bytes)
;   +7: opcode (1 byte)
;   +8: stack_depth_low (1 byte) - low 8 bits of stack depth
; ===========================================================================
global trace_record_opcode
trace_record_opcode:
    ; Quick check if tracing is enabled (redundant but defensive)
    test qword [rdi + VMContext.flags], VM_FLAG_TRACE
    jz .done

    ; Check buffer space (need 9 bytes for opcode entry)
    mov rax, [rdi + VMContext.trace_pos]
    mov rcx, [rdi + VMContext.trace_cap]
    lea r10, [rax + 9]          ; new position after write
    cmp r10, rcx
    ja .buffer_full

    ; Get trace buffer pointer
    mov r11, [rdi + VMContext.trace_buffer]
    add r11, rax                ; r11 = write position

    ; Write event type
    mov byte [r11 + TRACE_ENTRY_TYPE_OFFSET], TRACE_OPCODE

    ; Write timestamp (low 32 bits of rdtsc)
    rdtsc                       ; edx:eax = timestamp
    mov [r11 + TRACE_ENTRY_TIME_OFFSET], eax

    ; Write payload length (2 bytes)
    mov word [r11 + TRACE_ENTRY_LEN_OFFSET], 2

    ; Write opcode (1 byte)
    mov [r11 + TRACE_ENTRY_DATA_OFFSET], sil

    ; Write stack depth low byte
    mov rax, [rdi + VMContext.data_sp]
    mov [r11 + TRACE_ENTRY_DATA_OFFSET + 1], al

    ; Update trace position
    mov [rdi + VMContext.trace_pos], r10

    ; Increment trace entry count
    inc qword [rdi + VMContext.trace_count]

.done:
    ret

.buffer_full:
    ; Disable tracing on buffer full (graceful degradation)
    and qword [rdi + VMContext.flags], ~VM_FLAG_TRACE
    ret


; ===========================================================================
; trace_record_push - Record value push to trace buffer
; ===========================================================================
; Input:
;   rdi - pointer to VMContext
;   rsi - value tag
;   rdx - value num
;
; Output:
;   None
;
; Entry format for TRACE_PUSH (23 bytes total):
;   +0: TRACE_PUSH (1 byte)
;   +1: timestamp (4 bytes)
;   +5: payload_len = 16 (2 bytes)
;   +7: tag (8 bytes)
;   +15: num (8 bytes)
; ===========================================================================
global trace_record_push
trace_record_push:
    test qword [rdi + VMContext.flags], VM_FLAG_TRACE
    jz .done

    ; Check buffer space (need 23 bytes)
    mov rax, [rdi + VMContext.trace_pos]
    mov rcx, [rdi + VMContext.trace_cap]
    lea r10, [rax + 23]
    cmp r10, rcx
    ja .buffer_full

    mov r11, [rdi + VMContext.trace_buffer]
    add r11, rax

    ; Write header
    mov byte [r11 + TRACE_ENTRY_TYPE_OFFSET], TRACE_PUSH
    rdtsc
    mov [r11 + TRACE_ENTRY_TIME_OFFSET], eax
    mov word [r11 + TRACE_ENTRY_LEN_OFFSET], 16

    ; Write payload (tag and num)
    mov [r11 + TRACE_ENTRY_DATA_OFFSET], rsi
    mov [r11 + TRACE_ENTRY_DATA_OFFSET + 8], rdx

    ; Update position and count
    mov [rdi + VMContext.trace_pos], r10
    inc qword [rdi + VMContext.trace_count]

.done:
    ret

.buffer_full:
    and qword [rdi + VMContext.flags], ~VM_FLAG_TRACE
    ret


; ===========================================================================
; trace_record_pop - Record value pop from trace buffer
; ===========================================================================
; Input:
;   rdi - pointer to VMContext
;   rsi - value tag
;   rdx - value num
;
; Entry format same as TRACE_PUSH (23 bytes)
; ===========================================================================
global trace_record_pop
trace_record_pop:
    test qword [rdi + VMContext.flags], VM_FLAG_TRACE
    jz .done

    mov rax, [rdi + VMContext.trace_pos]
    mov rcx, [rdi + VMContext.trace_cap]
    lea r10, [rax + 23]
    cmp r10, rcx
    ja .buffer_full

    mov r11, [rdi + VMContext.trace_buffer]
    add r11, rax

    mov byte [r11 + TRACE_ENTRY_TYPE_OFFSET], TRACE_POP
    rdtsc
    mov [r11 + TRACE_ENTRY_TIME_OFFSET], eax
    mov word [r11 + TRACE_ENTRY_LEN_OFFSET], 16

    mov [r11 + TRACE_ENTRY_DATA_OFFSET], rsi
    mov [r11 + TRACE_ENTRY_DATA_OFFSET + 8], rdx

    mov [rdi + VMContext.trace_pos], r10
    inc qword [rdi + VMContext.trace_count]

.done:
    ret

.buffer_full:
    and qword [rdi + VMContext.flags], ~VM_FLAG_TRACE
    ret


; ===========================================================================
; trace_record_def - Record variable definition
; ===========================================================================
; Input:
;   rdi - pointer to VMContext
;   rsi - name pointer (RName*)
;   rdx - value tag
;   rcx - value num
;
; Entry format for TRACE_DEF (31 bytes total):
;   +0: TRACE_DEF (1 byte)
;   +1: timestamp (4 bytes)
;   +5: payload_len = 24 (2 bytes)
;   +7: name_ptr (8 bytes)
;   +15: tag (8 bytes)
;   +23: num (8 bytes)
; ===========================================================================
global trace_record_def
trace_record_def:
    test qword [rdi + VMContext.flags], VM_FLAG_TRACE
    jz .done

    mov rax, [rdi + VMContext.trace_pos]
    mov r8, [rdi + VMContext.trace_cap]
    lea r10, [rax + 31]
    cmp r10, r8
    ja .buffer_full

    mov r11, [rdi + VMContext.trace_buffer]
    add r11, rax

    mov byte [r11 + TRACE_ENTRY_TYPE_OFFSET], TRACE_DEF
    push rdx
    rdtsc
    pop rdx
    mov [r11 + TRACE_ENTRY_TIME_OFFSET], eax
    mov word [r11 + TRACE_ENTRY_LEN_OFFSET], 24

    mov [r11 + TRACE_ENTRY_DATA_OFFSET], rsi         ; name_ptr
    mov [r11 + TRACE_ENTRY_DATA_OFFSET + 8], rdx     ; tag
    mov [r11 + TRACE_ENTRY_DATA_OFFSET + 16], rcx    ; num

    mov [rdi + VMContext.trace_pos], r10
    inc qword [rdi + VMContext.trace_count]

.done:
    ret

.buffer_full:
    and qword [rdi + VMContext.flags], ~VM_FLAG_TRACE
    ret


; ===========================================================================
; trace_record_lookup - Record name lookup result
; ===========================================================================
; Input:
;   rdi - pointer to VMContext
;   rsi - name pointer (RName*)
;   rdx - result tag
;   rcx - result num
;
; Entry format same as TRACE_DEF (31 bytes)
; ===========================================================================
global trace_record_lookup
trace_record_lookup:
    test qword [rdi + VMContext.flags], VM_FLAG_TRACE
    jz .done

    mov rax, [rdi + VMContext.trace_pos]
    mov r8, [rdi + VMContext.trace_cap]
    lea r10, [rax + 31]
    cmp r10, r8
    ja .buffer_full

    mov r11, [rdi + VMContext.trace_buffer]
    add r11, rax

    mov byte [r11 + TRACE_ENTRY_TYPE_OFFSET], TRACE_LOOKUP
    push rdx
    rdtsc
    pop rdx
    mov [r11 + TRACE_ENTRY_TIME_OFFSET], eax
    mov word [r11 + TRACE_ENTRY_LEN_OFFSET], 24

    mov [r11 + TRACE_ENTRY_DATA_OFFSET], rsi
    mov [r11 + TRACE_ENTRY_DATA_OFFSET + 8], rdx
    mov [r11 + TRACE_ENTRY_DATA_OFFSET + 16], rcx

    mov [rdi + VMContext.trace_pos], r10
    inc qword [rdi + VMContext.trace_count]

.done:
    ret

.buffer_full:
    and qword [rdi + VMContext.flags], ~VM_FLAG_TRACE
    ret


; ===========================================================================
; trace_record_entity_push - Record entity stack push
; ===========================================================================
; Input:
;   rdi - pointer to VMContext
;   rsi - entity pointer
;
; Entry format for TRACE_ENTITY_PUSH (15 bytes):
;   +0: TRACE_ENTITY_PUSH (1 byte)
;   +1: timestamp (4 bytes)
;   +5: payload_len = 8 (2 bytes)
;   +7: entity_ptr (8 bytes)
; ===========================================================================
global trace_record_entity_push
trace_record_entity_push:
    test qword [rdi + VMContext.flags], VM_FLAG_TRACE
    jz .done

    mov rax, [rdi + VMContext.trace_pos]
    mov rcx, [rdi + VMContext.trace_cap]
    lea r10, [rax + 15]
    cmp r10, rcx
    ja .buffer_full

    mov r11, [rdi + VMContext.trace_buffer]
    add r11, rax

    mov byte [r11 + TRACE_ENTRY_TYPE_OFFSET], TRACE_ENTITY_PUSH
    rdtsc
    mov [r11 + TRACE_ENTRY_TIME_OFFSET], eax
    mov word [r11 + TRACE_ENTRY_LEN_OFFSET], 8
    mov [r11 + TRACE_ENTRY_DATA_OFFSET], rsi

    mov [rdi + VMContext.trace_pos], r10
    inc qword [rdi + VMContext.trace_count]

.done:
    ret

.buffer_full:
    and qword [rdi + VMContext.flags], ~VM_FLAG_TRACE
    ret


; ===========================================================================
; trace_record_entity_pop - Record entity stack pop
; ===========================================================================
; Input:
;   rdi - pointer to VMContext
;   rsi - entity pointer
;
; Entry format same as TRACE_ENTITY_PUSH (15 bytes)
; ===========================================================================
global trace_record_entity_pop
trace_record_entity_pop:
    test qword [rdi + VMContext.flags], VM_FLAG_TRACE
    jz .done

    mov rax, [rdi + VMContext.trace_pos]
    mov rcx, [rdi + VMContext.trace_cap]
    lea r10, [rax + 15]
    cmp r10, rcx
    ja .buffer_full

    mov r11, [rdi + VMContext.trace_buffer]
    add r11, rax

    mov byte [r11 + TRACE_ENTRY_TYPE_OFFSET], TRACE_ENTITY_POP
    rdtsc
    mov [r11 + TRACE_ENTRY_TIME_OFFSET], eax
    mov word [r11 + TRACE_ENTRY_LEN_OFFSET], 8
    mov [r11 + TRACE_ENTRY_DATA_OFFSET], rsi

    mov [rdi + VMContext.trace_pos], r10
    inc qword [rdi + VMContext.trace_count]

.done:
    ret

.buffer_full:
    and qword [rdi + VMContext.flags], ~VM_FLAG_TRACE
    ret


; ===========================================================================
; trace_record_error - Record error occurrence
; ===========================================================================
; Input:
;   rdi - pointer to VMContext
;   esi - error code
;   rdx - error PC
;
; Entry format for TRACE_ERROR (19 bytes):
;   +0: TRACE_ERROR (1 byte)
;   +1: timestamp (4 bytes)
;   +5: payload_len = 12 (2 bytes)
;   +7: error_code (4 bytes)
;   +11: error_pc (8 bytes)
; ===========================================================================
global trace_record_error
trace_record_error:
    test qword [rdi + VMContext.flags], VM_FLAG_TRACE
    jz .done

    mov rax, [rdi + VMContext.trace_pos]
    mov rcx, [rdi + VMContext.trace_cap]
    lea r10, [rax + 19]
    cmp r10, rcx
    ja .buffer_full

    mov r11, [rdi + VMContext.trace_buffer]
    add r11, rax

    mov byte [r11 + TRACE_ENTRY_TYPE_OFFSET], TRACE_ERROR
    mov r8d, esi                ; save error code
    rdtsc
    mov [r11 + TRACE_ENTRY_TIME_OFFSET], eax
    mov word [r11 + TRACE_ENTRY_LEN_OFFSET], 12
    mov [r11 + TRACE_ENTRY_DATA_OFFSET], r8d
    mov [r11 + TRACE_ENTRY_DATA_OFFSET + 4], rdx

    mov [rdi + VMContext.trace_pos], r10
    inc qword [rdi + VMContext.trace_count]

.done:
    ret

.buffer_full:
    and qword [rdi + VMContext.flags], ~VM_FLAG_TRACE
    ret


; ===========================================================================
; trace_record_dt_enter - Record decision table entry
; ===========================================================================
; Input:
;   rdi - pointer to VMContext
;   rsi - decision table name pointer
;
; Entry format for TRACE_DT_ENTER (15 bytes):
;   +0: TRACE_DT_ENTER (1 byte)
;   +1: timestamp (4 bytes)
;   +5: payload_len = 8 (2 bytes)
;   +7: dt_name_ptr (8 bytes)
; ===========================================================================
global trace_record_dt_enter
trace_record_dt_enter:
    test qword [rdi + VMContext.flags], VM_FLAG_TRACE
    jz .done

    mov rax, [rdi + VMContext.trace_pos]
    mov rcx, [rdi + VMContext.trace_cap]
    lea r10, [rax + 15]
    cmp r10, rcx
    ja .buffer_full

    mov r11, [rdi + VMContext.trace_buffer]
    add r11, rax

    mov byte [r11 + TRACE_ENTRY_TYPE_OFFSET], TRACE_DT_ENTER
    rdtsc
    mov [r11 + TRACE_ENTRY_TIME_OFFSET], eax
    mov word [r11 + TRACE_ENTRY_LEN_OFFSET], 8
    mov [r11 + TRACE_ENTRY_DATA_OFFSET], rsi

    mov [rdi + VMContext.trace_pos], r10
    inc qword [rdi + VMContext.trace_count]

.done:
    ret

.buffer_full:
    and qword [rdi + VMContext.flags], ~VM_FLAG_TRACE
    ret


; ===========================================================================
; trace_record_dt_exit - Record decision table exit
; ===========================================================================
; Input:
;   rdi - pointer to VMContext
;   rsi - decision table name pointer
;
; Entry format same as TRACE_DT_ENTER (15 bytes)
; ===========================================================================
global trace_record_dt_exit
trace_record_dt_exit:
    test qword [rdi + VMContext.flags], VM_FLAG_TRACE
    jz .done

    mov rax, [rdi + VMContext.trace_pos]
    mov rcx, [rdi + VMContext.trace_cap]
    lea r10, [rax + 15]
    cmp r10, rcx
    ja .buffer_full

    mov r11, [rdi + VMContext.trace_buffer]
    add r11, rax

    mov byte [r11 + TRACE_ENTRY_TYPE_OFFSET], TRACE_DT_EXIT
    rdtsc
    mov [r11 + TRACE_ENTRY_TIME_OFFSET], eax
    mov word [r11 + TRACE_ENTRY_LEN_OFFSET], 8
    mov [r11 + TRACE_ENTRY_DATA_OFFSET], rsi

    mov [rdi + VMContext.trace_pos], r10
    inc qword [rdi + VMContext.trace_count]

.done:
    ret

.buffer_full:
    and qword [rdi + VMContext.flags], ~VM_FLAG_TRACE
    ret


; ===========================================================================
; trace_record_condition - Record condition evaluation
; ===========================================================================
; Input:
;   rdi - pointer to VMContext
;   esi - column index
;   edx - result (0=false, 1=true)
;
; Entry format for TRACE_CONDITION (11 bytes):
;   +0: TRACE_CONDITION (1 byte)
;   +1: timestamp (4 bytes)
;   +5: payload_len = 4 (2 bytes)
;   +7: column (2 bytes)
;   +9: result (2 bytes)
; ===========================================================================
global trace_record_condition
trace_record_condition:
    test qword [rdi + VMContext.flags], VM_FLAG_TRACE
    jz .done

    mov rax, [rdi + VMContext.trace_pos]
    mov rcx, [rdi + VMContext.trace_cap]
    lea r10, [rax + 11]
    cmp r10, rcx
    ja .buffer_full

    mov r11, [rdi + VMContext.trace_buffer]
    add r11, rax

    mov byte [r11 + TRACE_ENTRY_TYPE_OFFSET], TRACE_CONDITION
    mov r8d, esi                ; save column
    mov r9d, edx                ; save result
    rdtsc
    mov [r11 + TRACE_ENTRY_TIME_OFFSET], eax
    mov word [r11 + TRACE_ENTRY_LEN_OFFSET], 4
    mov [r11 + TRACE_ENTRY_DATA_OFFSET], r8w
    mov [r11 + TRACE_ENTRY_DATA_OFFSET + 2], r9w

    mov [rdi + VMContext.trace_pos], r10
    inc qword [rdi + VMContext.trace_count]

.done:
    ret

.buffer_full:
    and qword [rdi + VMContext.flags], ~VM_FLAG_TRACE
    ret


; ===========================================================================
; trace_record_action - Record action execution
; ===========================================================================
; Input:
;   rdi - pointer to VMContext
;   esi - column index
;   edx - row index
;
; Entry format for TRACE_ACTION (11 bytes):
;   +0: TRACE_ACTION (1 byte)
;   +1: timestamp (4 bytes)
;   +5: payload_len = 4 (2 bytes)
;   +7: column (2 bytes)
;   +9: row (2 bytes)
; ===========================================================================
global trace_record_action
trace_record_action:
    test qword [rdi + VMContext.flags], VM_FLAG_TRACE
    jz .done

    mov rax, [rdi + VMContext.trace_pos]
    mov rcx, [rdi + VMContext.trace_cap]
    lea r10, [rax + 11]
    cmp r10, rcx
    ja .buffer_full

    mov r11, [rdi + VMContext.trace_buffer]
    add r11, rax

    mov byte [r11 + TRACE_ENTRY_TYPE_OFFSET], TRACE_ACTION
    mov r8d, esi                ; save column
    mov r9d, edx                ; save row
    rdtsc
    mov [r11 + TRACE_ENTRY_TIME_OFFSET], eax
    mov word [r11 + TRACE_ENTRY_LEN_OFFSET], 4
    mov [r11 + TRACE_ENTRY_DATA_OFFSET], r8w
    mov [r11 + TRACE_ENTRY_DATA_OFFSET + 2], r9w

    mov [rdi + VMContext.trace_pos], r10
    inc qword [rdi + VMContext.trace_count]

.done:
    ret

.buffer_full:
    and qword [rdi + VMContext.flags], ~VM_FLAG_TRACE
    ret


; ===========================================================================
; trace_record_create_entity - Record entity creation
; ===========================================================================
; Input:
;   rdi - pointer to VMContext
;   rsi - entity pointer
;   rdx - entity type name pointer
;
; Entry format for TRACE_CREATE_ENTITY (23 bytes):
;   +0: TRACE_CREATE_ENTITY (1 byte)
;   +1: timestamp (4 bytes)
;   +5: payload_len = 16 (2 bytes)
;   +7: entity_ptr (8 bytes)
;   +15: type_name_ptr (8 bytes)
; ===========================================================================
global trace_record_create_entity
trace_record_create_entity:
    test qword [rdi + VMContext.flags], VM_FLAG_TRACE
    jz .done

    mov rax, [rdi + VMContext.trace_pos]
    mov rcx, [rdi + VMContext.trace_cap]
    lea r10, [rax + 23]
    cmp r10, rcx
    ja .buffer_full

    mov r11, [rdi + VMContext.trace_buffer]
    add r11, rax

    mov byte [r11 + TRACE_ENTRY_TYPE_OFFSET], TRACE_CREATE_ENTITY
    mov r8, rsi                 ; save entity ptr
    mov r9, rdx                 ; save type name ptr
    rdtsc
    mov [r11 + TRACE_ENTRY_TIME_OFFSET], eax
    mov word [r11 + TRACE_ENTRY_LEN_OFFSET], 16
    mov [r11 + TRACE_ENTRY_DATA_OFFSET], r8
    mov [r11 + TRACE_ENTRY_DATA_OFFSET + 8], r9

    mov [rdi + VMContext.trace_pos], r10
    inc qword [rdi + VMContext.trace_count]

.done:
    ret

.buffer_full:
    and qword [rdi + VMContext.flags], ~VM_FLAG_TRACE
    ret


; ===========================================================================
; trace_record_array_op - Record array operation (NEW_ARRAY, ADD_TO, REMOVE)
; ===========================================================================
; Input:
;   rdi - pointer to VMContext
;   esi - operation type (TRACE_NEW_ARRAY, TRACE_ADD_TO, TRACE_REMOVE)
;   rdx - array pointer
;   rcx - index or value (depends on operation)
;
; Entry format (23 bytes):
;   +0: event_type (1 byte)
;   +1: timestamp (4 bytes)
;   +5: payload_len = 16 (2 bytes)
;   +7: array_ptr (8 bytes)
;   +15: index_or_value (8 bytes)
; ===========================================================================
global trace_record_array_op
trace_record_array_op:
    test qword [rdi + VMContext.flags], VM_FLAG_TRACE
    jz .done

    mov rax, [rdi + VMContext.trace_pos]
    mov r8, [rdi + VMContext.trace_cap]
    lea r10, [rax + 23]
    cmp r10, r8
    ja .buffer_full

    mov r11, [rdi + VMContext.trace_buffer]
    add r11, rax

    mov [r11 + TRACE_ENTRY_TYPE_OFFSET], sil    ; operation type
    mov r8, rdx                 ; save array ptr
    mov r9, rcx                 ; save index/value
    rdtsc
    mov [r11 + TRACE_ENTRY_TIME_OFFSET], eax
    mov word [r11 + TRACE_ENTRY_LEN_OFFSET], 16
    mov [r11 + TRACE_ENTRY_DATA_OFFSET], r8
    mov [r11 + TRACE_ENTRY_DATA_OFFSET + 8], r9

    mov [rdi + VMContext.trace_pos], r10
    inc qword [rdi + VMContext.trace_count]

.done:
    ret

.buffer_full:
    and qword [rdi + VMContext.flags], ~VM_FLAG_TRACE
    ret


; ===========================================================================
; trace_get_buffer_info - Get trace buffer information
; ===========================================================================
; Input:
;   rdi - pointer to VMContext
;
; Output:
;   rax - number of entries written
;   rdx - bytes used in buffer
;   rcx - buffer capacity
;
; This is a helper for Go to retrieve trace data.
; ===========================================================================
global trace_get_buffer_info
trace_get_buffer_info:
    mov rax, [rdi + VMContext.trace_count]
    mov rdx, [rdi + VMContext.trace_pos]
    mov rcx, [rdi + VMContext.trace_cap]
    ret


; ===========================================================================
; trace_reset - Reset trace buffer (clear all entries)
; ===========================================================================
; Input:
;   rdi - pointer to VMContext
;
; This clears the trace buffer position and count but does not
; zero the memory (for performance).
; ===========================================================================
global trace_reset
trace_reset:
    mov qword [rdi + VMContext.trace_pos], 0
    mov qword [rdi + VMContext.trace_count], 0
    mov qword [rdi + VMContext.trace_bytes_written], 0
    ; Re-enable tracing if buffer exists
    mov rax, [rdi + VMContext.trace_buffer]
    test rax, rax
    jz .done
    or qword [rdi + VMContext.flags], VM_FLAG_TRACE
.done:
    ret


; ===========================================================================
; trace_enable - Enable tracing
; ===========================================================================
; Input:
;   rdi - pointer to VMContext
;
; Returns:
;   rax - 0 if enabled, 1 if no buffer configured
; ===========================================================================
global trace_enable
trace_enable:
    mov rax, [rdi + VMContext.trace_buffer]
    test rax, rax
    jz .no_buffer
    or qword [rdi + VMContext.flags], VM_FLAG_TRACE
    xor eax, eax
    ret
.no_buffer:
    mov eax, 1
    ret


; ===========================================================================
; trace_disable - Disable tracing
; ===========================================================================
; Input:
;   rdi - pointer to VMContext
; ===========================================================================
global trace_disable
trace_disable:
    and qword [rdi + VMContext.flags], ~VM_FLAG_TRACE
    ret
