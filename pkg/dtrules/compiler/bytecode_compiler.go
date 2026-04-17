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

package compiler

import (
	"fmt"
	"strings"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/operators"
)

// CompileToBytecode compiles a postfix expression string into bytecode.
// This is the optimized compilation path that produces compact bytecode
// instead of Object arrays.
func (c *Compiler) CompileToBytecode(expr string) (*dtrules.BytecodeChunk, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return dtrules.NewBytecodeChunk(), nil
	}

	// Tokenize the expression
	tokens := c.tokenize(expr)
	if len(tokens) == 0 {
		return dtrules.NewBytecodeChunk(), nil
	}

	bc := dtrules.NewBytecodeChunk()

	for _, token := range tokens {
		if err := c.compileTokenToBytecode(bc, token); err != nil {
			return nil, fmt.Errorf("failed to compile token '%s': %w", token, err)
		}
	}

	return bc, nil
}

// compileTokenToBytecode compiles a single token into bytecode.
func (c *Compiler) compileTokenToBytecode(bc *dtrules.BytecodeChunk, token string) error {
	// Handle empty token
	if token == "" {
		return nil
	}

	// Handle quoted strings
	if len(token) >= 2 && (token[0] == '"' || token[0] == '\'') {
		str := token[1 : len(token)-1]
		bc.EmitPushConstant(dtrules.NewValueString(str))
		return nil
	}

	// Handle code blocks (braces) - these need special handling
	// For now, emit as a constant (the Object-based code block)
	if len(token) >= 2 && token[0] == '{' && token[len(token)-1] == '}' {
		// Compile the block to bytecode recursively
		inner := token[1 : len(token)-1]
		innerBC, err := c.CompileToBytecode(inner)
		if err != nil {
			return err
		}
		// For now, store the bytecode chunk as a constant
		// This would require extending Value to support bytecode chunks
		// For simplicity, fall back to Object-based execution for blocks
		obj, err := c.Compile(inner)
		if err != nil {
			return err
		}
		bc.EmitPushConstant(dtrules.NewValueObject(obj))
		_ = innerBC // We compiled it but use Object for now
		return nil
	}

	// Handle special literals
	switch strings.ToLower(token) {
	case "true":
		bc.Emit(dtrules.OpPushTrue)
		return nil
	case "false":
		bc.Emit(dtrules.OpPushFalse)
		return nil
	case "null":
		bc.Emit(dtrules.OpPushNull)
		return nil
	}

	// Try as integer
	if i, err := dtrules.GetRIntegerValueFromString(token); err == nil {
		v, _ := i.LongValue()
		bc.EmitPushConstant(dtrules.NewValueInteger(v))
		return nil
	}

	// Try as double
	if d, err := dtrules.GetRDoubleValueFromString(token); err == nil {
		v, _ := d.DoubleValue()
		bc.EmitPushConstant(dtrules.NewValueDouble(v))
		return nil
	}

	// Handle literal (non-executable) names starting with /
	if strings.HasPrefix(token, "/") {
		literalName := token[1:]
		name := dtrules.GetRName(literalName)
		if name == nil {
			return fmt.Errorf("invalid name syntax: %s", literalName)
		}
		bc.EmitPushName(name)
		return nil
	}

	// Get the name for this token - used for operator, table, and name lookups
	name := dtrules.GetRName(token)
	if name == nil {
		return fmt.Errorf("invalid name syntax: %s", token)
	}

	// Check if it's an operator - use indexed lookup
	if idx, ok := operators.GetIndex(name); ok {
		// Map common operators to dedicated opcodes for efficiency
		switch token {
		case "+":
			bc.Emit(dtrules.OpAdd)
		case "-":
			bc.Emit(dtrules.OpSub)
		case "*":
			bc.Emit(dtrules.OpMul)
		case "/":
			bc.Emit(dtrules.OpDiv)
		case "=", "eq":
			bc.Emit(dtrules.OpEq)
		case "!=", "ne":
			bc.Emit(dtrules.OpNe)
		case "<", "lt":
			bc.Emit(dtrules.OpLt)
		case "<=", "le":
			bc.Emit(dtrules.OpLe)
		case ">", "gt":
			bc.Emit(dtrules.OpGt)
		case ">=", "ge":
			bc.Emit(dtrules.OpGe)
		case "and":
			bc.Emit(dtrules.OpAnd)
		case "or":
			bc.Emit(dtrules.OpOr)
		case "not":
			bc.Emit(dtrules.OpNot)
		case "xor":
			bc.Emit(dtrules.OpXor)
		case "dup":
			bc.Emit(dtrules.OpDup)
		case "swap":
			bc.Emit(dtrules.OpSwap)
		case "pop":
			bc.Emit(dtrules.OpPop)
		case "def":
			bc.Emit(dtrules.OpDef)
		case "negate":
			bc.Emit(dtrules.OpNeg)
		case "elstmterror":
			bc.Emit(dtrules.OpError)
		case "elstmtwarn":
			bc.Emit(dtrules.OpWarn)
		default:
			// Use indexed operator call for other operators
			bc.EmitOperator(idx)
		}
		return nil
	}

	// Check if it's a decision table name
	if dt, err := c.factory.GetDecisionTable(name); err == nil && dt != nil {
		// Decision tables are executed via Object interface
		bc.EmitPushConstant(dtrules.NewValueObject(dt))
		bc.Emit(dtrules.OpExec)
		return nil
	}

	// Otherwise, treat it as a name to be looked up at runtime
	bc.EmitPushName(name)
	bc.Emit(dtrules.OpLookup)
	return nil
}

// CompileConditionToBytecode compiles a condition expression to bytecode.
// Conditions are expected to produce a boolean result.
func (c *Compiler) CompileConditionToBytecode(expr string) (*dtrules.BytecodeChunk, error) {
	return c.CompileToBytecode(expr)
}

// CompileActionToBytecode compiles an action expression to bytecode.
// Actions should leave the stack balanced (consume all produced values).
func (c *Compiler) CompileActionToBytecode(expr string) (*dtrules.BytecodeChunk, error) {
	return c.CompileToBytecode(expr)
}
