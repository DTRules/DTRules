; constants.inc - DTRules VM constants
; Opcodes, type tags, and system limits
; Matches Go's bytecode.go opcode definitions

%ifndef CONSTANTS_INC
%define CONSTANTS_INC

;-----------------------------------------------------------------------------
; Value Type Tags (matches Go's value.go)
;-----------------------------------------------------------------------------
VTAG_NULL       equ 0
VTAG_INTEGER    equ 1
VTAG_DOUBLE     equ 2
VTAG_BOOLEAN    equ 3
VTAG_STRING     equ 4
VTAG_NAME       equ 5
VTAG_ARRAY      equ 6
VTAG_ENTITY     equ 7
VTAG_BYTECODE   equ 8
VTAG_DATE       equ 9
VTAG_TABLE      equ 10
VTAG_MARK       equ 11
VTAG_OPERATOR   equ 12

;-----------------------------------------------------------------------------
; Value structure (24 bytes, matching Go)
;-----------------------------------------------------------------------------
VALUE_SIZE      equ 24
VALUE_TAG_OFF   equ 0       ; Tag byte offset
VALUE_PAD_OFF   equ 1       ; Padding (7 bytes)
VALUE_NUM_OFF   equ 8       ; Number (int64 or float64 bits)
VALUE_PTR_OFF   equ 16      ; Pointer for complex types

;-----------------------------------------------------------------------------
; Boolean values
;-----------------------------------------------------------------------------
BOOL_FALSE      equ 0
BOOL_TRUE       equ 1

;-----------------------------------------------------------------------------
; Opcodes (matches Go's bytecode.go exactly)
;-----------------------------------------------------------------------------

; Stack operations (0-10)
OP_NOP          equ 0
OP_PUSH         equ 1       ; Push constant from pool (followed by varint index)
OP_PUSH_INT     equ 2       ; Push immediate integer (followed by varint)
OP_POP          equ 3       ; Pop and discard
OP_DUP          equ 4       ; Duplicate top
OP_SWAP         equ 5       ; Swap top two
OP_ROT          equ 6       ; Rotate top three
OP_ROLL         equ 7       ; Roll n items
OP_INDEX        equ 8       ; Copy nth item to top (pick)
OP_CLEAR        equ 9       ; Clear stack to mark
OP_MARK         equ 10      ; Push mark

; Arithmetic operations (20-28)
OP_ADD          equ 20
OP_SUB          equ 21
OP_MUL          equ 22
OP_DIV          equ 23
OP_MOD          equ 24
OP_NEG          equ 25
OP_ABS          equ 26
OP_INC          equ 27
OP_DEC          equ 28

; Comparison operations (30-35)
OP_EQ           equ 30
OP_NE           equ 31
OP_LT           equ 32
OP_LE           equ 33
OP_GT           equ 34
OP_GE           equ 35

; Logical operations (40-43)
OP_AND          equ 40
OP_OR           equ 41
OP_NOT          equ 42
OP_XOR          equ 43

; Control flow (50-59)
OP_EXEC         equ 50      ; Execute top of stack
OP_IF           equ 51      ; Conditional execution
OP_IFELSE       equ 52      ; if-else
OP_WHILE        equ 53      ; while loop
OP_FOR          equ 54      ; for loop
OP_FORALL       equ 55      ; iterate array
OP_RETURN       equ 56      ; Return from procedure
OP_JUMP         equ 57      ; Unconditional jump (followed by offset)
OP_JUMPIF       equ 58      ; Conditional jump if true
OP_CALL         equ 59      ; Call operator by index

; Entity operations (60-64)
OP_ENTITYPUSH   equ 60      ; Push entity onto entity stack
OP_ENTITYPOP    equ 61      ; Pop entity from entity stack
OP_DEF          equ 62      ; Define variable
OP_LOOKUP       equ 63      ; Lookup name in context
OP_NEWENTITY    equ 64      ; Create new entity

; Array operations (70-74)
OP_NEWARRAY     equ 70      ; Create new array
OP_ADDTO        equ 71      ; Add to array (push)
OP_LENGTH       equ 72      ; Get array/string length
OP_GET          equ 73      ; Get array element
OP_PUT          equ 74      ; Put array element

; Table operations (80-82)
OP_NEWTABLE     equ 80      ; Create new table/map
OP_TABLEGET     equ 81      ; Get table value
OP_TABLEPUT     equ 82      ; Put table value

; String operations (90-91)
OP_CONCAT       equ 90      ; String concatenation
OP_SUBSTRING    equ 91      ; Substring

; Constant shortcuts (100-104)
OP_PUSH_TRUE    equ 100
OP_PUSH_FALSE   equ 101
OP_PUSH_NULL    equ 102
OP_PUSH_ZERO    equ 103
OP_PUSH_ONE     equ 104

; Extended opcodes (200-202)
OP_OPERATOR     equ 200     ; Call operator by index (followed by varint)
OP_CONSTANT     equ 201     ; Push constant by index (followed by varint)
OP_NAME         equ 202     ; Push name by index (followed by varint)

; Debug/utility (250-255)
OP_PRINT        equ 250
OP_TRACE        equ 251
OP_DEBUG        equ 252
OP_HALT         equ 255

;-----------------------------------------------------------------------------
; Stack limits
;-----------------------------------------------------------------------------
DATA_STACK_SIZE     equ 4096    ; Max values on data stack
ENTITY_STACK_SIZE   equ 256     ; Max entities on entity stack
CONTROL_STACK_SIZE  equ 1024    ; Max frames on control stack

;-----------------------------------------------------------------------------
; Memory sizes
;-----------------------------------------------------------------------------
HEAP_SIZE           equ 16*1024*1024    ; 16 MB heap
VALUE_POOL_SIZE     equ 65536           ; Pre-allocated values
STRING_POOL_SIZE    equ 1024*1024       ; 1 MB for strings
NAME_TABLE_SIZE     equ 4096            ; Hash table buckets

;-----------------------------------------------------------------------------
; String limits
;-----------------------------------------------------------------------------
MAX_STRING_LEN      equ 65536   ; Maximum string length
MAX_NAME_LEN        equ 256     ; Maximum name length

;-----------------------------------------------------------------------------
; Array constants
;-----------------------------------------------------------------------------
ARRAY_INITIAL_CAP   equ 8       ; Initial array capacity
ARRAY_GROWTH_FACTOR equ 2       ; Capacity doubles on resize

;-----------------------------------------------------------------------------
; Entity constants
;-----------------------------------------------------------------------------
ENTITY_INITIAL_CAP  equ 16      ; Initial attribute slots
ENTITY_LOAD_FACTOR  equ 70      ; Resize at 70% full (as percentage)

;-----------------------------------------------------------------------------
; Error codes
;-----------------------------------------------------------------------------
ERR_NONE            equ 0
ERR_STACK_OVERFLOW  equ 1
ERR_STACK_UNDERFLOW equ 2
ERR_TYPE_MISMATCH   equ 3
ERR_DIV_BY_ZERO     equ 4
ERR_OUT_OF_MEMORY   equ 5
ERR_INVALID_OPCODE  equ 6
ERR_INDEX_BOUNDS    equ 7
ERR_NAME_NOT_FOUND  equ 8
ERR_ATTR_NOT_FOUND  equ 9
ERR_PARSE_ERROR     equ 10
ERR_FILE_NOT_FOUND  equ 11
ERR_IO_ERROR        equ 12

%endif ; CONSTANTS_INC
