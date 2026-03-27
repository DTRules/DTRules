// Copyright 2024 Paul Snow
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package asmruntime provides CGO bindings to the x86-64 assembly implementation
// of the DTRules bytecode virtual machine.
package asmruntime

/*
#cgo CFLAGS: -I${SRCDIR}/../../../../asm/include
#cgo LDFLAGS: -L${SRCDIR}/../../../../asm/build -ldtrules_vm

// ASM library function declarations
extern int lib_init(void);
extern int lib_set_bytecode(void* code, long length);
extern int lib_set_constant_pool(void* pool, long count);
extern int lib_set_name_pool(void* pool, long count);
extern int lib_get_error(void);
extern void lib_clear_error(void);
extern int lib_reset(void);
extern void* lib_get_state_ptr(void);
extern int lib_data_stack_push(void* value);
extern int lib_data_stack_pop(void* value);
extern long lib_data_stack_depth(void);
extern int lib_data_stack_peek(void* value);
extern int lib_vm_execute(void);  // CGO-safe wrapper for vm_execute

// Entity stack operations
extern int lib_entity_stack_push(void* entity);
extern void* lib_entity_stack_pop(void);
extern long lib_entity_stack_depth(void);
extern void* lib_entity_stack_peek(long index);

// Heap and entity operations
extern void* lib_heap_alloc(long size);
extern void* lib_entity_alloc(void* type_name);
extern int lib_entity_set_attr(void* entity, void* name, void* value);
extern void* lib_entity_get_attr(void* entity, void* name);
extern void* lib_create_string(void* str, long length);

// Decision table operations
extern void* lib_table_alloc(void);
extern void lib_table_set_name(void* table, void* name);
extern void lib_table_set_root(void* table, void* root, int is_cnode);
extern int lib_table_register(void* table);
extern int lib_table_execute_by_name(void* name);
extern void lib_table_clear_registry(void);

// CNode operations
extern void* lib_cnode_alloc(void);
extern void lib_cnode_set_condition(void* cnode, void* bytecode, long length);
extern void lib_cnode_set_true_branch(void* cnode, void* branch, int is_cnode);
extern void lib_cnode_set_false_branch(void* cnode, void* branch, int is_cnode);

// ANode operations
extern void* lib_anode_alloc(void);
extern int lib_anode_add_action(void* anode, void* bytecode, long length);

// Error codes (must match constants.inc)
#define ERR_NONE            0
#define ERR_STACK_OVERFLOW  1
#define ERR_STACK_UNDERFLOW 2
#define ERR_TYPE_MISMATCH   3
#define ERR_DIV_BY_ZERO     4
#define ERR_OUT_OF_MEMORY   5
#define ERR_INVALID_OPCODE  6
#define ERR_INDEX_BOUNDS    7
#define ERR_NAME_NOT_FOUND  8
#define ERR_ATTR_NOT_FOUND  9
#define ERR_PARSE_ERROR     10
#define ERR_FILE_NOT_FOUND  11
#define ERR_IO_ERROR        12

// Value type tags (must match value.go and constants.inc)
#define VTAG_NULL    0
#define VTAG_INTEGER 1
#define VTAG_DOUBLE  2
#define VTAG_BOOLEAN 3
#define VTAG_STRING  4
#define VTAG_NAME    5
#define VTAG_ARRAY   6
#define VTAG_ENTITY  7
#define VTAG_OBJECT  8

// Value structure (24 bytes, matching Go's Value and ASM's Value)
// Must be kept in sync with both implementations
typedef struct {
    unsigned char tag;      // Type discriminator
    unsigned char pad[7];   // Padding for alignment
    long num;               // Integer value or float64 bits
    void* ptr;              // Pointer for complex types
} ASMValue;
*/
import "C"
import (
	"errors"
	"runtime"
	"sync"
	"unsafe"

	"github.com/DTRules/DTRules/go/pkg/dtrules"
)

// Error definitions
var (
	ErrNotInitialized  = errors.New("ASM runtime not initialized")
	ErrStackOverflow   = errors.New("stack overflow")
	ErrStackUnderflow  = errors.New("stack underflow")
	ErrTypeMismatch    = errors.New("type mismatch")
	ErrDivByZero       = errors.New("division by zero")
	ErrOutOfMemory     = errors.New("out of memory")
	ErrInvalidOpcode   = errors.New("invalid opcode")
	ErrIndexBounds     = errors.New("index out of bounds")
	ErrNameNotFound    = errors.New("name not found")
	ErrAttrNotFound    = errors.New("attribute not found")
	ErrParseError      = errors.New("parse error")
	ErrFileNotFound    = errors.New("file not found")
	ErrIOError         = errors.New("I/O error")
	ErrUnknown         = errors.New("unknown error")
)

// errFromCode converts ASM error code to Go error
func errFromCode(code C.int) error {
	switch code {
	case C.ERR_NONE:
		return nil
	case C.ERR_STACK_OVERFLOW:
		return ErrStackOverflow
	case C.ERR_STACK_UNDERFLOW:
		return ErrStackUnderflow
	case C.ERR_TYPE_MISMATCH:
		return ErrTypeMismatch
	case C.ERR_DIV_BY_ZERO:
		return ErrDivByZero
	case C.ERR_OUT_OF_MEMORY:
		return ErrOutOfMemory
	case C.ERR_INVALID_OPCODE:
		return ErrInvalidOpcode
	case C.ERR_INDEX_BOUNDS:
		return ErrIndexBounds
	case C.ERR_NAME_NOT_FOUND:
		return ErrNameNotFound
	case C.ERR_ATTR_NOT_FOUND:
		return ErrAttrNotFound
	case C.ERR_PARSE_ERROR:
		return ErrParseError
	case C.ERR_FILE_NOT_FOUND:
		return ErrFileNotFound
	case C.ERR_IO_ERROR:
		return ErrIOError
	default:
		return ErrUnknown
	}
}

// Runtime state
var (
	initialized bool
	initMu      sync.Mutex
)

// Init initializes the ASM runtime. Must be called before any other functions.
// Safe to call multiple times; subsequent calls are no-ops.
func Init() error {
	initMu.Lock()
	defer initMu.Unlock()

	if initialized {
		return nil
	}

	result := C.lib_init()
	if result != 0 {
		return errFromCode(result)
	}

	initialized = true
	return nil
}

// Reset resets the VM state for a new execution.
// Clears stacks but keeps heap allocations.
func Reset() error {
	if !initialized {
		return ErrNotInitialized
	}

	result := C.lib_reset()
	return errFromCode(result)
}

// ClearError clears the error state.
func ClearError() {
	if initialized {
		C.lib_clear_error()
	}
}

// GetError returns the current error code from the ASM runtime.
func GetError() error {
	if !initialized {
		return ErrNotInitialized
	}
	return errFromCode(C.lib_get_error())
}

// ExecuteBytecode executes a bytecode chunk through the ASM VM.
// The bytecode chunk must contain compiled bytecode instructions.
func ExecuteBytecode(chunk *dtrules.BytecodeChunk) error {
	if !initialized {
		return ErrNotInitialized
	}

	code := chunk.Code()
	if len(code) == 0 {
		return nil // Nothing to execute
	}

	// Use Pinner to pin Go memory before passing to CGO
	var pinner runtime.Pinner
	defer pinner.Unpin()

	// Set bytecode pointers
	pinner.Pin(&code[0])
	codePtr := unsafe.Pointer(&code[0])
	result := C.lib_set_bytecode(codePtr, C.long(len(code)))
	if result != 0 {
		return errFromCode(result)
	}

	// Set constant pool if present
	constants := chunk.Constants()
	if len(constants) > 0 {
		// Pin each constant's pointer field if it contains a pointer
		pinner.Pin(&constants[0])
		for i := range constants {
			_, _, ptr := constants[i].RawParts()
			if ptr != nil {
				pinner.Pin(ptr)
			}
		}
		constPtr := unsafe.Pointer(&constants[0])
		result = C.lib_set_constant_pool(constPtr, C.long(len(constants)))
		if result != 0 {
			return errFromCode(result)
		}
	} else {
		C.lib_set_constant_pool(nil, 0)
	}

	// Set name pool if present
	// IMPORTANT: Go names are *RName structs, but ASM expects length-prefixed strings.
	// We must convert each name to an ASM heap string.
	names := chunk.Names()
	if len(names) > 0 {
		// Create an array of ASM string pointers
		asmNames := make([]unsafe.Pointer, len(names))
		for i, name := range names {
			if name != nil {
				// Convert RName to ASM length-prefixed string
				asmNames[i] = CreateString(name.StringValue())
			}
		}
		// Pin the array
		pinner.Pin(&asmNames[0])
		namePtr := unsafe.Pointer(&asmNames[0])
		result = C.lib_set_name_pool(namePtr, C.long(len(names)))
		if result != 0 {
			return errFromCode(result)
		}
	} else {
		C.lib_set_name_pool(nil, 0)
	}

	// Execute via CGO-safe wrapper
	result = C.lib_vm_execute()
	return errFromCode(result)
}

// PushValue pushes a Value onto the ASM data stack.
func PushValue(v dtrules.Value) error {
	if !initialized {
		return ErrNotInitialized
	}

	// Value has the same memory layout in Go and ASM (24 bytes)
	result := C.lib_data_stack_push(unsafe.Pointer(&v))
	return errFromCode(result)
}

// PopValue pops a Value from the ASM data stack.
func PopValue() (dtrules.Value, error) {
	if !initialized {
		return dtrules.ValueNull, ErrNotInitialized
	}

	var v dtrules.Value
	result := C.lib_data_stack_pop(unsafe.Pointer(&v))
	if result != 0 {
		return dtrules.ValueNull, errFromCode(result)
	}
	return v, nil
}

// PeekValue peeks at the top Value without popping.
func PeekValue() (dtrules.Value, error) {
	if !initialized {
		return dtrules.ValueNull, ErrNotInitialized
	}

	var v dtrules.Value
	result := C.lib_data_stack_peek(unsafe.Pointer(&v))
	if result != 0 {
		return dtrules.ValueNull, errFromCode(result)
	}
	return v, nil
}

// StackDepth returns the number of values on the data stack.
func StackDepth() int {
	if !initialized {
		return 0
	}
	return int(C.lib_data_stack_depth())
}

// StatePtr returns a pointer to the ASM state structure.
// This is for advanced use cases that need direct state access.
func StatePtr() unsafe.Pointer {
	if !initialized {
		return nil
	}
	return C.lib_get_state_ptr()
}

// IsInitialized returns true if the ASM runtime has been initialized.
func IsInitialized() bool {
	initMu.Lock()
	defer initMu.Unlock()
	return initialized
}

// Entity stack operations

// EntityStackPush pushes an entity pointer onto the ASM entity stack.
func EntityStackPush(entityPtr unsafe.Pointer) error {
	if !initialized {
		return ErrNotInitialized
	}
	result := C.lib_entity_stack_push(entityPtr)
	return errFromCode(result)
}

// EntityStackPop pops an entity pointer from the ASM entity stack.
func EntityStackPop() unsafe.Pointer {
	if !initialized {
		return nil
	}
	return C.lib_entity_stack_pop()
}

// EntityStackDepth returns the number of entities on the entity stack.
func EntityStackDepth() int {
	if !initialized {
		return 0
	}
	return int(C.lib_entity_stack_depth())
}

// EntityStackPeek returns the entity at the given index (0 = top).
func EntityStackPeek(index int) unsafe.Pointer {
	if !initialized {
		return nil
	}
	return C.lib_entity_stack_peek(C.long(index))
}

// Heap operations

// HeapAlloc allocates memory from the ASM heap.
func HeapAlloc(size int) unsafe.Pointer {
	if !initialized {
		return nil
	}
	return C.lib_heap_alloc(C.long(size))
}

// CreateString creates a length-prefixed string in the ASM heap.
// Returns pointer to the string (length:8 bytes + data).
func CreateString(s string) unsafe.Pointer {
	if !initialized || len(s) == 0 {
		return nil
	}
	return C.lib_create_string(unsafe.Pointer(&[]byte(s)[0]), C.long(len(s)))
}

// Entity operations

// EntityAlloc allocates a new entity in ASM memory.
// typeName can be nil for untyped entities.
func EntityAlloc(typeName unsafe.Pointer) unsafe.Pointer {
	if !initialized {
		return nil
	}
	return C.lib_entity_alloc(typeName)
}

// EntitySetAttr sets an attribute on an ASM entity.
// name must be a pointer to a length-prefixed string.
// value must be a pointer to a 24-byte Value.
func EntitySetAttr(entity, name, value unsafe.Pointer) error {
	if !initialized {
		return ErrNotInitialized
	}
	result := C.lib_entity_set_attr(entity, name, value)
	return errFromCode(result)
}

// EntityGetAttr gets an attribute from an ASM entity.
// Returns pointer to the Value, or nil if not found.
func EntityGetAttr(entity, name unsafe.Pointer) unsafe.Pointer {
	if !initialized {
		return nil
	}
	return C.lib_entity_get_attr(entity, name)
}

// MarshalEntity converts a Go entity to ASM entity format.
// Returns pointer to the ASM entity, or nil on error.
func MarshalEntity(entity dtrules.Entity) (unsafe.Pointer, error) {
	if !initialized {
		return nil, ErrNotInitialized
	}

	// Create type name string
	var typeNamePtr unsafe.Pointer
	if name := entity.GetName(); name != nil {
		typeNamePtr = CreateString(name.StringValue())
	}

	// Allocate entity
	asmEntity := EntityAlloc(typeNamePtr)
	if asmEntity == nil {
		return nil, ErrOutOfMemory
	}

	// Copy attributes
	attrNames := entity.GetAttributeNames()
	for _, attrName := range attrNames {
		// Get attribute value from Go entity
		val, err := entity.Get(attrName)
		if err != nil {
			continue
		}

		// Create name string in ASM heap
		namePtr := CreateString(attrName.StringValue())
		if namePtr == nil {
			continue
		}

		// Marshal the value
		asmValue, err := MarshalValue(val)
		if err != nil {
			continue
		}

		// Set attribute
		EntitySetAttr(asmEntity, namePtr, asmValue)
	}

	return asmEntity, nil
}

// MarshalValue converts a Go dtrules.Object to ASM Value format.
// Returns pointer to 24-byte Value in ASM heap.
func MarshalValue(obj dtrules.Object) (unsafe.Pointer, error) {
	if !initialized {
		return nil, ErrNotInitialized
	}

	// Allocate Value in heap
	valuePtr := HeapAlloc(24)
	if valuePtr == nil {
		return nil, ErrOutOfMemory
	}

	// Convert Object to Value using ValueFromObject which properly sets
	// the type tag based on the actual object type (integer, double, boolean, etc.)
	// NOT NewValueObject which always wraps as VTagObject
	v := dtrules.ValueFromObject(obj)

	// Copy the 24-byte Value directly (same memory layout)
	*(*dtrules.Value)(valuePtr) = v

	return valuePtr, nil
}

// SetupEntityStack sets up the ASM entity stack from a Go session.
// This copies the entity stack from the Go interpreter to ASM.
func SetupEntityStack(state dtrules.State) error {
	if !initialized {
		return ErrNotInitialized
	}

	// Reset entity stack first
	// Note: lib_reset already does this, but we want to be explicit

	depth := state.EntityDepth()
	if depth == 0 {
		return nil
	}

	// Push entities in reverse order (bottom to top)
	for i := depth - 1; i >= 0; i-- {
		goEntity, err := state.EntityFetch(i)
		if err != nil {
			continue
		}

		// Marshal entity to ASM format
		asmEntity, err := MarshalEntity(goEntity)
		if err != nil {
			continue
		}

		// Create a Value wrapping the entity pointer
		entityValue := HeapAlloc(24)
		if entityValue == nil {
			return ErrOutOfMemory
		}

		// Set tag to VTAG_ENTITY (7) and ptr to entity
		*(*byte)(entityValue) = 7                                                           // tag
		*(*unsafe.Pointer)(unsafe.Pointer(uintptr(entityValue) + 16)) = asmEntity // ptr

		// Push onto entity stack
		if err := EntityStackPush(entityValue); err != nil {
			return err
		}
	}

	return nil
}

// =============================================================================
// Decision Table Operations for CGO
// =============================================================================

// TableAlloc allocates a new decision table in ASM memory.
func TableAlloc() unsafe.Pointer {
	if !initialized {
		return nil
	}
	return C.lib_table_alloc()
}

// TableSetName sets the name of an ASM decision table.
func TableSetName(table unsafe.Pointer, name unsafe.Pointer) {
	if initialized && table != nil {
		C.lib_table_set_name(table, name)
	}
}

// TableSetRoot sets the root node of an ASM decision table.
func TableSetRoot(table unsafe.Pointer, root unsafe.Pointer, isCNode bool) {
	if initialized && table != nil {
		var flag C.int
		if isCNode {
			flag = 1
		}
		C.lib_table_set_root(table, root, flag)
	}
}

// TableRegister registers a decision table in the ASM registry.
func TableRegister(table unsafe.Pointer) error {
	if !initialized {
		return ErrNotInitialized
	}
	result := C.lib_table_register(table)
	return errFromCode(result)
}

// TableExecuteByName executes a decision table by name.
func TableExecuteByName(name unsafe.Pointer) error {
	if !initialized {
		return ErrNotInitialized
	}
	result := C.lib_table_execute_by_name(name)
	return errFromCode(result)
}

// TableClearRegistry clears all registered tables.
func TableClearRegistry() {
	if initialized {
		C.lib_table_clear_registry()
	}
}

// CNodeAlloc allocates a new condition node in ASM memory.
func CNodeAlloc() unsafe.Pointer {
	if !initialized {
		return nil
	}
	return C.lib_cnode_alloc()
}

// CNodeSetCondition sets the condition bytecode for a CNode.
func CNodeSetCondition(cnode unsafe.Pointer, bytecode []byte) {
	if initialized && cnode != nil && len(bytecode) > 0 {
		C.lib_cnode_set_condition(cnode, unsafe.Pointer(&bytecode[0]), C.long(len(bytecode)))
	}
}

// CNodeSetTrueBranch sets the true branch of a CNode.
func CNodeSetTrueBranch(cnode unsafe.Pointer, branch unsafe.Pointer, isCNode bool) {
	if initialized && cnode != nil {
		var flag C.int
		if isCNode {
			flag = 1
		}
		C.lib_cnode_set_true_branch(cnode, branch, flag)
	}
}

// CNodeSetFalseBranch sets the false branch of a CNode.
func CNodeSetFalseBranch(cnode unsafe.Pointer, branch unsafe.Pointer, isCNode bool) {
	if initialized && cnode != nil {
		var flag C.int
		if isCNode {
			flag = 1
		}
		C.lib_cnode_set_false_branch(cnode, branch, flag)
	}
}

// ANodeAlloc allocates a new action node in ASM memory.
func ANodeAlloc() unsafe.Pointer {
	if !initialized {
		return nil
	}
	return C.lib_anode_alloc()
}

// ANodeAddAction adds an action bytecode to an ANode.
func ANodeAddAction(anode unsafe.Pointer, bytecode []byte) error {
	if !initialized {
		return ErrNotInitialized
	}
	if anode == nil || len(bytecode) == 0 {
		return nil
	}
	result := C.lib_anode_add_action(anode, unsafe.Pointer(&bytecode[0]), C.long(len(bytecode)))
	return errFromCode(result)
}

// ExecuteTableByName executes a decision table by its string name.
// This is a convenience wrapper that creates the name string and executes.
func ExecuteTableByName(tableName string) error {
	if !initialized {
		return ErrNotInitialized
	}
	if len(tableName) == 0 {
		return ErrNameNotFound
	}

	// Create name string in ASM heap
	namePtr := CreateString(tableName)
	if namePtr == nil {
		return ErrOutOfMemory
	}

	return TableExecuteByName(namePtr)
}
