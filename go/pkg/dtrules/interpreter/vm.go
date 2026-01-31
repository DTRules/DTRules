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

package interpreter

import (
	"github.com/PaulSnow/DTRules/go/pkg/dtrules"
)

// ExecuteBytecode executes a bytecode chunk on the state's value stack.
// This is the optimized execution path for compiled expressions.
// Returns an error if bytecode is malformed (out of bounds, overflow, etc.)
// rather than panicking, allowing graceful error handling.
func (s *DTState) ExecuteBytecode(bc *dtrules.BytecodeChunk) error {
	code := bc.Code()
	constants := bc.Constants()
	names := bc.Names()

	reader := dtrules.NewBytecodeReader(code)

	for reader.HasMore() {
		op, err := reader.TryReadOpcode()
		if err != nil {
			return err
		}

		switch op {
		case dtrules.OpNop:
			// Do nothing

		// Push operations
		case dtrules.OpPushTrue:
			if err := s.ValuePush(dtrules.ValueTrue); err != nil {
				return err
			}
		case dtrules.OpPushFalse:
			if err := s.ValuePush(dtrules.ValueFalse); err != nil {
				return err
			}
		case dtrules.OpPushNull:
			if err := s.ValuePush(dtrules.ValueNull); err != nil {
				return err
			}
		case dtrules.OpPushZero:
			if err := s.ValuePush(dtrules.NewValueInteger(0)); err != nil {
				return err
			}
		case dtrules.OpPushOne:
			if err := s.ValuePush(dtrules.NewValueInteger(1)); err != nil {
				return err
			}
		case dtrules.OpPushInt:
			v, err := reader.TryReadVarint()
			if err != nil {
				return err
			}
			if err := s.ValuePush(dtrules.NewValueInteger(int64(v))); err != nil {
				return err
			}
		case dtrules.OpConstant:
			idx, err := reader.TryReadVarint()
			if err != nil {
				return err
			}
			if idx < 0 || idx >= len(constants) {
				return dtrules.OutOfBoundsError("ExecuteBytecode", "constant index out of range")
			}
			if err := s.ValuePush(constants[idx]); err != nil {
				return err
			}
		case dtrules.OpName:
			idx, err := reader.TryReadVarint()
			if err != nil {
				return err
			}
			if idx < 0 || idx >= len(names) {
				return dtrules.OutOfBoundsError("ExecuteBytecode", "name index out of range")
			}
			if err := s.ValuePush(dtrules.NewValueName(names[idx])); err != nil {
				return err
			}

		// Stack operations
		case dtrules.OpPop:
			if _, err := s.ValuePop(); err != nil {
				return err
			}
		case dtrules.OpDup:
			if err := s.ValueDup(); err != nil {
				return err
			}
		case dtrules.OpSwap:
			if err := s.ValueSwap(); err != nil {
				return err
			}
		case dtrules.OpRot:
			if err := s.ValueRot(); err != nil {
				return err
			}

		// Arithmetic operations
		case dtrules.OpAdd:
			b, err := s.ValuePop()
			if err != nil {
				return err
			}
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			if err := s.ValuePush(a.Add(b)); err != nil {
				return err
			}
		case dtrules.OpSub:
			b, err := s.ValuePop()
			if err != nil {
				return err
			}
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			if err := s.ValuePush(a.Sub(b)); err != nil {
				return err
			}
		case dtrules.OpMul:
			b, err := s.ValuePop()
			if err != nil {
				return err
			}
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			if err := s.ValuePush(a.Mul(b)); err != nil {
				return err
			}
		case dtrules.OpDiv:
			b, err := s.ValuePop()
			if err != nil {
				return err
			}
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			if err := s.ValuePush(a.Div(b)); err != nil {
				return err
			}
		case dtrules.OpNeg:
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			if a.IsInteger() {
				if err := s.ValuePush(dtrules.NewValueInteger(-a.AsInteger())); err != nil {
					return err
				}
			} else {
				if err := s.ValuePush(dtrules.NewValueDouble(-a.AsDouble())); err != nil {
					return err
				}
			}
		case dtrules.OpInc:
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			if err := s.ValuePush(dtrules.NewValueInteger(a.AsInteger() + 1)); err != nil {
				return err
			}
		case dtrules.OpDec:
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			if err := s.ValuePush(dtrules.NewValueInteger(a.AsInteger() - 1)); err != nil {
				return err
			}

		// Comparison operations
		case dtrules.OpEq:
			b, err := s.ValuePop()
			if err != nil {
				return err
			}
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			if err := s.ValuePush(dtrules.NewValueBoolean(a.Equal(b))); err != nil {
				return err
			}
		case dtrules.OpNe:
			b, err := s.ValuePop()
			if err != nil {
				return err
			}
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			if err := s.ValuePush(dtrules.NewValueBoolean(!a.Equal(b))); err != nil {
				return err
			}
		case dtrules.OpLt:
			b, err := s.ValuePop()
			if err != nil {
				return err
			}
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			if err := s.ValuePush(dtrules.NewValueBoolean(a.Less(b))); err != nil {
				return err
			}
		case dtrules.OpLe:
			b, err := s.ValuePop()
			if err != nil {
				return err
			}
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			if err := s.ValuePush(dtrules.NewValueBoolean(a.Less(b) || a.Equal(b))); err != nil {
				return err
			}
		case dtrules.OpGt:
			b, err := s.ValuePop()
			if err != nil {
				return err
			}
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			if err := s.ValuePush(dtrules.NewValueBoolean(b.Less(a))); err != nil {
				return err
			}
		case dtrules.OpGe:
			b, err := s.ValuePop()
			if err != nil {
				return err
			}
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			if err := s.ValuePush(dtrules.NewValueBoolean(b.Less(a) || a.Equal(b))); err != nil {
				return err
			}

		// Boolean operations
		case dtrules.OpAnd:
			b, err := s.ValuePop()
			if err != nil {
				return err
			}
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			if err := s.ValuePush(dtrules.NewValueBoolean(a.AsBoolean() && b.AsBoolean())); err != nil {
				return err
			}
		case dtrules.OpOr:
			b, err := s.ValuePop()
			if err != nil {
				return err
			}
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			if err := s.ValuePush(dtrules.NewValueBoolean(a.AsBoolean() || b.AsBoolean())); err != nil {
				return err
			}
		case dtrules.OpNot:
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			if err := s.ValuePush(dtrules.NewValueBoolean(!a.AsBoolean())); err != nil {
				return err
			}
		case dtrules.OpXor:
			b, err := s.ValuePop()
			if err != nil {
				return err
			}
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			if err := s.ValuePush(dtrules.NewValueBoolean(a.AsBoolean() != b.AsBoolean())); err != nil {
				return err
			}

		// Name lookup (falls back to Object-based entity stack)
		case dtrules.OpLookup:
			nameVal, err := s.ValuePop()
			if err != nil {
				return err
			}
			name := nameVal.AsName()
			if name == nil {
				return dtrules.ConversionError("OpLookup", "expected name")
			}
			obj, err := s.Find(name)
			if err != nil {
				return err
			}
			if obj == nil {
				if err := s.ValuePush(dtrules.ValueNull); err != nil {
					return err
				}
			} else {
				if err := s.ValuePush(dtrules.ValueFromObject(obj)); err != nil {
					return err
				}
			}

		// Def (assign value to name in entity context)
		case dtrules.OpDef:
			nameVal, err := s.ValuePop()
			if err != nil {
				return err
			}
			value, err := s.ValuePop()
			if err != nil {
				return err
			}
			name := nameVal.AsName()
			if name == nil {
				return dtrules.ConversionError("OpDef", "expected name")
			}
			// Convert value to Object for entity storage
			obj := value.AsObject()
			ok, err := s.Def(name, obj, s.TestState(TRACE))
			if err != nil {
				return err
			}
			if !ok {
				return dtrules.UndefinedError("OpDef", name.StringValue()+" not found in context")
			}

		// Call operator by index
		// Note: This requires the OperatorTable to be set on the state
		case dtrules.OpOperator:
			idx, err := reader.TryReadVarint()
			if err != nil {
				return err
			}
			if s.operatorTable == nil || idx >= len(s.operatorTable) {
				return dtrules.UndefinedError("OpOperator", "operator not found")
			}
			op := s.operatorTable[idx]
			if op == nil {
				return dtrules.UndefinedError("OpOperator", "operator not found")
			}
			// Execute the operator using Object-based execution
			if err := op.Execute(s); err != nil {
				return err
			}

		default:
			return dtrules.NewRulesError("Invalid Opcode", "ExecuteBytecode", "unknown opcode")
		}
	}

	return nil
}

// EvaluateBytecodeCondition executes bytecode and returns the boolean result.
func (s *DTState) EvaluateBytecodeCondition(bc *dtrules.BytecodeChunk) (bool, error) {
	initialDepth := s.ValueStackDepth()
	if err := s.ExecuteBytecode(bc); err != nil {
		return false, err
	}
	if s.ValueStackDepth() != initialDepth+1 {
		return false, dtrules.NewRulesError("Stack Check Exception", "EvaluateBytecodeCondition", "expected one result")
	}
	result, err := s.ValuePop()
	if err != nil {
		return false, err
	}
	return result.AsBoolean(), nil
}

// EvaluateBytecodeAction executes bytecode that should leave the stack balanced.
func (s *DTState) EvaluateBytecodeAction(bc *dtrules.BytecodeChunk) error {
	initialDepth := s.ValueStackDepth()
	if err := s.ExecuteBytecode(bc); err != nil {
		return err
	}
	if s.ValueStackDepth() != initialDepth {
		return dtrules.NewRulesError("Stack Check Exception", "EvaluateBytecodeAction", "stack not balanced")
	}
	return nil
}
