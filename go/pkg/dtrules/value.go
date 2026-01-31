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

package dtrules

import (
	"fmt"
	"math"
	"unsafe"
)

// Value is an optimized tagged union type for the rules engine.
// It avoids interface boxing and heap allocation for primitive types.
// This is used internally for high-performance execution paths.
//
// Memory layout (24 bytes on 64-bit):
//   - tag:  1 byte  - type discriminator
//   - _:    7 bytes - padding
//   - num:  8 bytes - numeric value (int64 or float64 bits)
//   - ptr:  8 bytes - pointer to complex types (string, array, entity, etc.)
type Value struct {
	tag uint8
	_   [7]byte        // padding for alignment
	num int64          // For integers, booleans (0/1), or float64 bits
	ptr unsafe.Pointer // For strings, arrays, entities, objects
}

// Value type tags
const (
	VTagNull    uint8 = 0
	VTagInteger uint8 = 1
	VTagDouble  uint8 = 2
	VTagBoolean uint8 = 3
	VTagString  uint8 = 4
	VTagName    uint8 = 5
	VTagArray   uint8 = 6
	VTagEntity  uint8 = 7
	VTagObject  uint8 = 8 // Fallback to Object interface
)

// Null value singleton
var ValueNull = Value{tag: VTagNull}

// ValueTrue and ValueFalse are boolean singletons
var (
	ValueTrue  = Value{tag: VTagBoolean, num: 1}
	ValueFalse = Value{tag: VTagBoolean, num: 0}
)

// NewValueInteger creates an integer value.
func NewValueInteger(v int64) Value {
	return Value{tag: VTagInteger, num: v}
}

// NewValueDouble creates a double value.
func NewValueDouble(v float64) Value {
	return Value{tag: VTagDouble, num: int64(math.Float64bits(v))}
}

// NewValueBoolean creates a boolean value.
func NewValueBoolean(v bool) Value {
	if v {
		return ValueTrue
	}
	return ValueFalse
}

// NewValueString creates a string value.
func NewValueString(s string) Value {
	// Store pointer to the string header
	return Value{tag: VTagString, ptr: unsafe.Pointer(&s)}
}

// NewValueName creates a name value.
func NewValueName(n *RName) Value {
	return Value{tag: VTagName, ptr: unsafe.Pointer(n)}
}

// NewValueArray creates an array value.
func NewValueArray(a *RArray) Value {
	return Value{tag: VTagArray, ptr: unsafe.Pointer(a)}
}

// NewValueEntity creates an entity value.
func NewValueEntity(e Entity) Value {
	// Store as interface (uses Object fallback)
	return NewValueObject(e.(Object))
}

// NewValueObject wraps any Object in a Value.
func NewValueObject(o Object) Value {
	if o == nil {
		return ValueNull
	}
	// Store the interface as two words (type + data pointer)
	iface := *(*[2]unsafe.Pointer)(unsafe.Pointer(&o))
	return Value{
		tag: VTagObject,
		num: int64(uintptr(iface[0])), // type pointer
		ptr: iface[1],                 // data pointer
	}
}

// Tag returns the value's type tag.
func (v Value) Tag() uint8 {
	return v.tag
}

// IsNull returns true if this is a null value.
func (v Value) IsNull() bool {
	return v.tag == VTagNull
}

// IsInteger returns true if this is an integer value.
func (v Value) IsInteger() bool {
	return v.tag == VTagInteger
}

// IsDouble returns true if this is a double value.
func (v Value) IsDouble() bool {
	return v.tag == VTagDouble
}

// IsBoolean returns true if this is a boolean value.
func (v Value) IsBoolean() bool {
	return v.tag == VTagBoolean
}

// IsString returns true if this is a string value.
func (v Value) IsString() bool {
	return v.tag == VTagString
}

// IsName returns true if this is a name value.
func (v Value) IsName() bool {
	return v.tag == VTagName
}

// IsNumeric returns true if this is an integer or double.
func (v Value) IsNumeric() bool {
	return v.tag == VTagInteger || v.tag == VTagDouble
}

// AsInteger returns the integer value.
func (v Value) AsInteger() int64 {
	return v.num
}

// AsDouble returns the double value.
func (v Value) AsDouble() float64 {
	if v.tag == VTagInteger {
		return float64(v.num)
	}
	return math.Float64frombits(uint64(v.num))
}

// AsBoolean returns the boolean value.
func (v Value) AsBoolean() bool {
	return v.num != 0
}

// AsString returns the string value.
func (v Value) AsString() string {
	if v.tag == VTagString {
		return *(*string)(v.ptr)
	}
	return v.String()
}

// AsName returns the name value.
func (v Value) AsName() *RName {
	return (*RName)(v.ptr)
}

// AsArray returns the array value.
func (v Value) AsArray() *RArray {
	return (*RArray)(v.ptr)
}

// AsObject converts the value to an Object interface.
func (v Value) AsObject() Object {
	switch v.tag {
	case VTagNull:
		return GetRNull()
	case VTagInteger:
		return GetRIntegerValue(v.num)
	case VTagDouble:
		return GetRDoubleValue(math.Float64frombits(uint64(v.num)))
	case VTagBoolean:
		return GetRBoolean(v.num != 0)
	case VTagString:
		return GetRString(*(*string)(v.ptr))
	case VTagName:
		return (*RName)(v.ptr)
	case VTagArray:
		return (*RArray)(v.ptr)
	case VTagObject:
		// Reconstruct the interface from its two-word representation.
		// This is the inverse of NewValueObject which stores the interface
		// as (type pointer, data pointer). Go interfaces are represented
		// internally as two words: iface[0] is the type/itab pointer and
		// iface[1] is the data pointer. This technique is documented in
		// Go's reflect package and is safe because:
		// 1. The pointers were originally extracted from a valid interface
		// 2. The Value type preserves both pointers without modification
		// 3. The reconstructed interface matches the original type exactly
		//
		// Note: go vet may warn about "possible misuse of unsafe.Pointer"
		// because it cannot verify the invariants above at compile time.
		// This usage is intentional and tested.
		var o Object
		iface := (*[2]unsafe.Pointer)(unsafe.Pointer(&o))
		iface[0] = unsafe.Pointer(uintptr(v.num)) //nolint:unsafeptr // intentional interface reconstruction
		iface[1] = v.ptr
		return o
	default:
		return GetRNull()
	}
}

// String returns a string representation.
func (v Value) String() string {
	switch v.tag {
	case VTagNull:
		return "null"
	case VTagInteger:
		return fmt.Sprintf("%d", v.num)
	case VTagDouble:
		return fmt.Sprintf("%g", math.Float64frombits(uint64(v.num)))
	case VTagBoolean:
		if v.num != 0 {
			return "true"
		}
		return "false"
	case VTagString:
		return *(*string)(v.ptr)
	case VTagName:
		return (*RName)(v.ptr).StringValue()
	case VTagArray:
		return (*RArray)(v.ptr).String()
	case VTagObject:
		return v.AsObject().StringValue()
	default:
		return "unknown"
	}
}

// Add performs addition on two values.
func (v Value) Add(other Value) Value {
	if v.tag == VTagInteger && other.tag == VTagInteger {
		return NewValueInteger(v.num + other.num)
	}
	// Fall back to double arithmetic
	return NewValueDouble(v.AsDouble() + other.AsDouble())
}

// Sub performs subtraction on two values.
func (v Value) Sub(other Value) Value {
	if v.tag == VTagInteger && other.tag == VTagInteger {
		return NewValueInteger(v.num - other.num)
	}
	return NewValueDouble(v.AsDouble() - other.AsDouble())
}

// Mul performs multiplication on two values.
func (v Value) Mul(other Value) Value {
	if v.tag == VTagInteger && other.tag == VTagInteger {
		return NewValueInteger(v.num * other.num)
	}
	return NewValueDouble(v.AsDouble() * other.AsDouble())
}

// Div performs division on two values.
func (v Value) Div(other Value) Value {
	// Division always returns double to handle fractions
	return NewValueDouble(v.AsDouble() / other.AsDouble())
}

// Less returns true if v < other.
func (v Value) Less(other Value) bool {
	if v.tag == VTagInteger && other.tag == VTagInteger {
		return v.num < other.num
	}
	return v.AsDouble() < other.AsDouble()
}

// Equal returns true if v == other.
func (v Value) Equal(other Value) bool {
	if v.tag != other.tag {
		// Cross-type numeric comparison
		if v.IsNumeric() && other.IsNumeric() {
			return v.AsDouble() == other.AsDouble()
		}
		return false
	}
	switch v.tag {
	case VTagNull:
		return true
	case VTagInteger, VTagBoolean:
		return v.num == other.num
	case VTagDouble:
		return v.AsDouble() == other.AsDouble()
	case VTagString:
		return v.AsString() == other.AsString()
	case VTagName:
		return v.ptr == other.ptr // Names are interned
	default:
		return v.ptr == other.ptr
	}
}

// ValueFromObject creates a Value from an Object.
func ValueFromObject(o Object) Value {
	if o == nil {
		return ValueNull
	}
	switch t := o.Type(); t {
	case TypeNull:
		return ValueNull
	case TypeInteger:
		v, _ := o.LongValue()
		return NewValueInteger(v)
	case TypeDouble:
		v, _ := o.DoubleValue()
		return NewValueDouble(v)
	case TypeBoolean:
		v, _ := o.BooleanValue()
		return NewValueBoolean(v)
	case TypeString:
		return NewValueString(o.StringValue())
	case TypeName:
		n, _ := o.RNameValue()
		return NewValueName(n)
	case TypeArray:
		a, _ := o.RArrayValue()
		return NewValueArray(a)
	default:
		return NewValueObject(o)
	}
}
