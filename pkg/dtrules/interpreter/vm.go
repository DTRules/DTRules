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
	"log"

	"github.com/DTRules/DTRules/pkg/dtrules"
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
			// Integer division for integers, float division otherwise
			if a.IsInteger() && b.IsInteger() {
				bVal := b.AsInteger()
				if bVal == 0 {
					return dtrules.NewRulesError("Division By Zero", "OpDiv", "cannot divide by zero")
				}
				if err := s.ValuePush(dtrules.NewValueInteger(a.AsInteger() / bVal)); err != nil {
					return err
				}
			} else {
				// Use TryDiv to check for division by zero (returns error instead of Inf/NaN)
				result, divErr := a.TryDiv(b)
				if divErr != nil {
					return dtrules.NewRulesError("Division By Zero", "OpDiv", "cannot divide by zero")
				}
				if err := s.ValuePush(result); err != nil {
					return err
				}
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
		case dtrules.OpMod:
			b, err := s.ValuePop()
			if err != nil {
				return err
			}
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			bVal := b.AsInteger()
			if bVal == 0 {
				return dtrules.NewRulesError("Division By Zero", "OpMod", "cannot mod by zero")
			}
			if err := s.ValuePush(dtrules.NewValueInteger(a.AsInteger() % bVal)); err != nil {
				return err
			}
		case dtrules.OpAbs:
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			if a.IsInteger() {
				v := a.AsInteger()
				if v < 0 {
					v = -v
				}
				if err := s.ValuePush(dtrules.NewValueInteger(v)); err != nil {
					return err
				}
			} else {
				v := a.AsDouble()
				if v < 0 {
					v = -v
				}
				if err := s.ValuePush(dtrules.NewValueDouble(v)); err != nil {
					return err
				}
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

		// Bytes operations
		case dtrules.OpBytesConcat:
			b, err := s.ValuePop()
			if err != nil {
				return err
			}
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			ab, err := a.AsObject().RBytesValue()
			if err != nil {
				return err
			}
			bb, err := b.AsObject().RBytesValue()
			if err != nil {
				return err
			}
			if err := s.ValuePush(dtrules.ValueFromObject(ab.Concat(bb))); err != nil {
				return err
			}
		case dtrules.OpBytesLen:
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			ab, err := a.AsObject().RBytesValue()
			if err != nil {
				return err
			}
			if err := s.ValuePush(dtrules.NewValueInteger(int64(ab.Len()))); err != nil {
				return err
			}
		case dtrules.OpBytesSlice:
			toVal, err := s.ValuePop()
			if err != nil {
				return err
			}
			fromVal, err := s.ValuePop()
			if err != nil {
				return err
			}
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			ab, err := a.AsObject().RBytesValue()
			if err != nil {
				return err
			}
			result, err := ab.Slice(int(fromVal.AsInteger()), int(toVal.AsInteger()))
			if err != nil {
				return err
			}
			if err := s.ValuePush(dtrules.ValueFromObject(result)); err != nil {
				return err
			}
		case dtrules.OpBytesIndex:
			idxVal, err := s.ValuePop()
			if err != nil {
				return err
			}
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			ab, err := a.AsObject().RBytesValue()
			if err != nil {
				return err
			}
			byteVal, err := ab.At(int(idxVal.AsInteger()))
			if err != nil {
				return err
			}
			if err := s.ValuePush(dtrules.NewValueInteger(byteVal)); err != nil {
				return err
			}
		case dtrules.OpBytesEq:
			b, err := s.ValuePop()
			if err != nil {
				return err
			}
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			ab, err := a.AsObject().RBytesValue()
			if err != nil {
				return err
			}
			bb, err := b.AsObject().RBytesValue()
			if err != nil {
				return err
			}
			if err := s.ValuePush(dtrules.NewValueBoolean(ab.ConstantTimeEqual(bb))); err != nil {
				return err
			}
		case dtrules.OpBytesNe:
			b, err := s.ValuePop()
			if err != nil {
				return err
			}
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			ab, err := a.AsObject().RBytesValue()
			if err != nil {
				return err
			}
			bb, err := b.AsObject().RBytesValue()
			if err != nil {
				return err
			}
			if err := s.ValuePush(dtrules.NewValueBoolean(!ab.ConstantTimeEqual(bb))); err != nil {
				return err
			}

		// Hash operations
		case dtrules.OpSha256:
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			ab, err := a.AsObject().RBytesValue()
			if err != nil {
				return err
			}
			result := ab.Sha256()
			if err := s.ValuePush(dtrules.ValueFromObject(result)); err != nil {
				return err
			}
		case dtrules.OpKeccak256:
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			ab, err := a.AsObject().RBytesValue()
			if err != nil {
				return err
			}
			result := ab.Keccak256()
			if err := s.ValuePush(dtrules.ValueFromObject(result)); err != nil {
				return err
			}
		case dtrules.OpRipemd160:
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			ab, err := a.AsObject().RBytesValue()
			if err != nil {
				return err
			}
			result := ab.Ripemd160()
			if err := s.ValuePush(dtrules.ValueFromObject(result)); err != nil {
				return err
			}
		case dtrules.OpSha3:
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			ab, err := a.AsObject().RBytesValue()
			if err != nil {
				return err
			}
			result := ab.Sha3()
			if err := s.ValuePush(dtrules.ValueFromObject(result)); err != nil {
				return err
			}

		// Encoding operations
		case dtrules.OpHex:
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			ab, err := a.AsObject().RBytesValue()
			if err != nil {
				return err
			}
			if err := s.ValuePush(dtrules.ValueFromObject(dtrules.NewRString(ab.HexString()))); err != nil {
				return err
			}

		case dtrules.OpCvHex:
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			result, err := dtrules.NewRBytesFromHex(a.AsObject().StringValue())
			if err != nil {
				return dtrules.ConversionError("OpCvHex", err.Error())
			}
			if err := s.ValuePush(dtrules.ValueFromObject(result)); err != nil {
				return err
			}

		case dtrules.OpB58Check:
			versionVal, err := s.ValuePop()
			if err != nil {
				return err
			}
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			ab, err := a.AsObject().RBytesValue()
			if err != nil {
				return err
			}
			version := int(versionVal.AsInteger())
			if version < 0 || version > 255 {
				return dtrules.ConversionError("OpB58Check", "version byte out of range [0,255]")
			}
			encoded := ab.Base58CheckEncode(byte(version))
			if err := s.ValuePush(dtrules.ValueFromObject(dtrules.NewRString(encoded))); err != nil {
				return err
			}

		case dtrules.OpCvB58Check:
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			payload, version, decErr := dtrules.RBytesBase58CheckDecode(a.AsObject().StringValue())
			if decErr != nil {
				return dtrules.ConversionError("OpCvB58Check", decErr.Error())
			}
			if err := s.ValuePush(dtrules.ValueFromObject(dtrules.NewRBytes(payload))); err != nil {
				return err
			}
			if err := s.ValuePush(dtrules.NewValueInteger(int64(version))); err != nil {
				return err
			}

		case dtrules.OpBech32:
			hrpVal, err := s.ValuePop()
			if err != nil {
				return err
			}
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			ab, err := a.AsObject().RBytesValue()
			if err != nil {
				return err
			}
			hrp := hrpVal.AsObject().StringValue()
			encoded, encErr := ab.Bech32Encode(hrp)
			if encErr != nil {
				return dtrules.ConversionError("OpBech32", encErr.Error())
			}
			if err := s.ValuePush(dtrules.ValueFromObject(dtrules.NewRString(encoded))); err != nil {
				return err
			}

		case dtrules.OpCvBech32:
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			payload, hrp, decErr := dtrules.RBytesBech32Decode(a.AsObject().StringValue())
			if decErr != nil {
				return dtrules.ConversionError("OpCvBech32", decErr.Error())
			}
			if err := s.ValuePush(dtrules.ValueFromObject(dtrules.NewRBytes(payload))); err != nil {
				return err
			}
			if err := s.ValuePush(dtrules.ValueFromObject(dtrules.NewRString(hrp))); err != nil {
				return err
			}

		case dtrules.OpBigIntBytes:
			sizeVal, err := s.ValuePop()
			if err != nil {
				return err
			}
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			bi, err := a.AsObject().RBigIntValue()
			if err != nil {
				return err
			}
			size := int(sizeVal.AsInteger())
			if size <= 0 {
				return dtrules.ConversionError("OpBigIntBytes", "size must be positive")
			}
			result, convErr := bi.ToBytes(size)
			if convErr != nil {
				return dtrules.ConversionError("OpBigIntBytes", convErr.Error())
			}
			if err := s.ValuePush(dtrules.ValueFromObject(dtrules.NewRBytes(result))); err != nil {
				return err
			}

		case dtrules.OpBytesBigInt:
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			ab, err := a.AsObject().RBytesValue()
			if err != nil {
				return err
			}
			result := ab.ToBigInt()
			if err := s.ValuePush(dtrules.ValueFromObject(dtrules.NewRBigInt(result))); err != nil {
				return err
			}

		case dtrules.OpError:
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			return dtrules.NewELStatementError(a.AsObject().StringValue())

		case dtrules.OpWarn:
			a, err := s.ValuePop()
			if err != nil {
				return err
			}
			// TODO: wire to a real logger-injection point (see future issue)
			log.Printf("[warn] %s", a.AsObject().StringValue())

		default:
			return dtrules.NewRulesError("Invalid Opcode", "ExecuteBytecode", "unknown opcode")
		}
	}

	return nil
}

// EvaluateBytecodeCondition executes bytecode and returns the boolean result.
// The bytecode should produce at least one boolean value on the stack.
// If more values are left, the top value is used and extras are cleaned up.
func (s *DTState) EvaluateBytecodeCondition(bc *dtrules.BytecodeChunk) (bool, error) {
	initialDepth := s.ValueStackDepth()
	if err := s.ExecuteBytecode(bc); err != nil {
		return false, err
	}

	// Must have at least one result
	if s.ValueStackDepth() <= initialDepth {
		return false, dtrules.NewRulesError("Stack Check Exception", "EvaluateBytecodeCondition", "No result on stack")
	}

	// Pop the result
	result, err := s.ValuePop()
	if err != nil {
		return false, err
	}

	// Clean up any extra values left on the stack (some expressions leave extras)
	for s.ValueStackDepth() > initialDepth {
		s.ValuePop()
	}

	return result.AsBoolean(), nil
}

// EvaluateBytecodeAction executes bytecode for an action.
// Actions should not leave extra values on the stack. If the stack is
// unbalanced after execution, the extra values are cleaned up and an
// error is returned to signal the problem.
func (s *DTState) EvaluateBytecodeAction(bc *dtrules.BytecodeChunk) error {
	initialDepth := s.ValueStackDepth()
	if err := s.ExecuteBytecode(bc); err != nil {
		return err
	}

	// Check for unbalanced stack - actions should not leave extra values
	if s.ValueStackDepth() != initialDepth {
		// Clean up extra values so the interpreter can continue
		for s.ValueStackDepth() > initialDepth {
			s.ValuePop()
		}
		return dtrules.NewRulesError("Stack Check Exception", "EvaluateBytecodeAction",
			"action left extra values on the stack")
	}

	return nil
}
