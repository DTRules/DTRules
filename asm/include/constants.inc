; constants.inc - DTRules constants
; Opcodes, type tags, and system limits

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
; Opcodes (matches Go's bytecode.go)
;-----------------------------------------------------------------------------
; Stack operations (0x00-0x0F)
OP_NOP          equ 0x00
OP_PUSH_NULL    equ 0x01
OP_PUSH_TRUE    equ 0x02
OP_PUSH_FALSE   equ 0x03
OP_PUSH_INT     equ 0x04    ; Followed by varint
OP_PUSH_DOUBLE  equ 0x05    ; Followed by 8 bytes
OP_PUSH_STRING  equ 0x06    ; Followed by length + bytes
OP_PUSH_NAME    equ 0x07    ; Followed by name reference
OP_DUP          equ 0x08
OP_SWAP         equ 0x09
OP_POP          equ 0x0A
OP_ROT          equ 0x0B
OP_OVER         equ 0x0C
OP_PICK         equ 0x0D    ; Followed by index
OP_ROLL         equ 0x0E    ; Followed by index
OP_CLEAR        equ 0x0F

; Arithmetic operations (0x10-0x1F)
OP_ADD          equ 0x10
OP_SUB          equ 0x11
OP_MUL          equ 0x12
OP_DIV          equ 0x13
OP_MOD          equ 0x14
OP_NEG          equ 0x15
OP_ABS          equ 0x16
OP_MIN          equ 0x17
OP_MAX          equ 0x18
OP_FLOOR        equ 0x19
OP_CEIL         equ 0x1A
OP_ROUND        equ 0x1B
OP_TRUNCATE     equ 0x1C
OP_POW          equ 0x1D

; Comparison operations (0x20-0x2F)
OP_EQ           equ 0x20
OP_NE           equ 0x21
OP_LT           equ 0x22
OP_LE           equ 0x23
OP_GT           equ 0x24
OP_GE           equ 0x25
OP_AND          equ 0x26
OP_OR           equ 0x27
OP_NOT          equ 0x28
OP_XOR          equ 0x29

; String operations (0x30-0x4F)
OP_CONCAT       equ 0x30
OP_LENGTH       equ 0x31
OP_SUBSTRING    equ 0x32
OP_INDEXOF      equ 0x33
OP_LASTINDEXOF  equ 0x34
OP_STARTSWITH   equ 0x35
OP_ENDSWITH     equ 0x36
OP_CONTAINS     equ 0x37
OP_REPLACE      equ 0x38
OP_REPLACEALL   equ 0x39
OP_SPLIT        equ 0x3A
OP_JOIN         equ 0x3B
OP_TRIM         equ 0x3C
OP_TRIMLEFT     equ 0x3D
OP_TRIMRIGHT    equ 0x3E
OP_TOUPPER      equ 0x3F
OP_TOLOWER      equ 0x40
OP_TOSTRING     equ 0x41
OP_TOINTEGER    equ 0x42
OP_TODOUBLE     equ 0x43
OP_MATCH        equ 0x44
OP_FORMAT       equ 0x45

; Array operations (0x50-0x5F)
OP_NEWARRAY     equ 0x50
OP_ARRAYGET     equ 0x51
OP_ARRAYSET     equ 0x52
OP_ARRAYLEN     equ 0x53
OP_ARRAYPUSH    equ 0x54
OP_ARRAYPOP     equ 0x55
OP_ARRAYCONCAT  equ 0x56
OP_ARRAYSLICE   equ 0x57
OP_ARRAYCOPY    equ 0x58
OP_ARRAYCLEAR   equ 0x59
OP_ARRAYREMOVE  equ 0x5A
OP_ARRAYINSERT  equ 0x5B
OP_ARRAYREVERSE equ 0x5C
OP_ARRAYSORT    equ 0x5D
OP_ARRAYCONTAINS equ 0x5E
OP_ARRAYINDEXOF equ 0x5F

; Control flow (0x60-0x6F)
OP_IF           equ 0x60
OP_IFELSE       equ 0x61
OP_WHILE        equ 0x62
OP_FOR          equ 0x63
OP_FORALL       equ 0x64
OP_BREAK        equ 0x65
OP_CONTINUE     equ 0x66
OP_RETURN       equ 0x67
OP_EXEC         equ 0x68
OP_CALL         equ 0x69
OP_JUMP         equ 0x6A    ; Followed by offset
OP_JUMPIF       equ 0x6B    ; Followed by offset
OP_JUMPIFNOT    equ 0x6C    ; Followed by offset

; Entity operations (0x70-0x7F)
OP_NEWENTITY    equ 0x70
OP_GETATTR      equ 0x71
OP_SETATTR      equ 0x72
OP_HASATTR      equ 0x73
OP_DELATTR      equ 0x74
OP_GETATTRS     equ 0x75
OP_ENTITYTYPE   equ 0x76
OP_PUSHENTITY   equ 0x77
OP_POPENTITY    equ 0x78
OP_PEEKENTITY   equ 0x79
OP_DEF          equ 0x7A
OP_XDEF         equ 0x7B
OP_LOOKUP       equ 0x7C
OP_GETENTITY    equ 0x7D
OP_BIND         equ 0x7E

; Decision table operations (0x80-0x8F)
OP_EXECUTETABLE equ 0x80
OP_CNODE        equ 0x81
OP_ANODE        equ 0x82

; Debug/utility (0xF0-0xFF)
OP_PRINT        equ 0xF0
OP_TRACE        equ 0xF1
OP_DEBUG        equ 0xF2
OP_HALT         equ 0xFF

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
