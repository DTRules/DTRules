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

package el

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
)

// PostfixEmitter walks the EL parse tree and emits postfix notation.
type PostfixEmitter struct {
	*BaseELVisitor
	output  strings.Builder
	errors  []error
	symbols map[string]string // symbol table for type resolution
}

// NewPostfixEmitter creates a new postfix emitter.
func NewPostfixEmitter() *PostfixEmitter {
	return &PostfixEmitter{
		BaseELVisitor: &BaseELVisitor{},
		symbols:       make(map[string]string),
	}
}

// SetSymbols sets the symbol table for type resolution.
func (e *PostfixEmitter) SetSymbols(symbols map[string]string) {
	e.symbols = symbols
}

// Emit returns the accumulated postfix output.
func (e *PostfixEmitter) Emit() string {
	return strings.TrimSpace(e.output.String())
}

// Errors returns any errors encountered during emission.
func (e *PostfixEmitter) Errors() []error {
	return e.errors
}

// Reset clears the emitter state.
func (e *PostfixEmitter) Reset() {
	e.output.Reset()
	e.errors = nil
}

// emit adds a token to the output.
func (e *PostfixEmitter) emit(token string) {
	if e.output.Len() > 0 {
		e.output.WriteString(" ")
	}
	e.output.WriteString(token)
}

// emitError records an error.
func (e *PostfixEmitter) emitError(format string, args ...interface{}) {
	e.errors = append(e.errors, fmt.Errorf(format, args...))
}

// visitChildren visits all children of a node.
func (e *PostfixEmitter) visitChildren(ctx antlr.RuleContext) interface{} {
	for i := 0; i < ctx.GetChildCount(); i++ {
		child := ctx.GetChild(i)
		if parseTree, ok := child.(antlr.ParseTree); ok {
			e.Visit(parseTree)
		}
	}
	return nil
}

// Visit dispatches to the appropriate visitor method.
func (e *PostfixEmitter) Visit(tree antlr.ParseTree) interface{} {
	return tree.Accept(e)
}

// VisitTerminal handles terminal nodes (tokens).
func (e *PostfixEmitter) VisitTerminal(node antlr.TerminalNode) interface{} {
	// Most terminals are handled by their parent rules
	return nil
}

// VisitErrorNode handles error nodes.
func (e *PostfixEmitter) VisitErrorNode(node antlr.ErrorNode) interface{} {
	e.emitError("parse error: %s", node.GetText())
	return nil
}

// ============================================================================
// Boolean Expression Visitors
// ============================================================================

func (e *PostfixEmitter) VisitBoolIntEq(ctx *BoolIntEqContext) interface{} {
	e.Visit(ctx.Iexpr(0))
	e.Visit(ctx.Iexpr(1))
	e.emit("eq")
	return nil
}

func (e *PostfixEmitter) VisitBoolIntNeq(ctx *BoolIntNeqContext) interface{} {
	e.Visit(ctx.Iexpr(0))
	e.Visit(ctx.Iexpr(1))
	e.emit("ne")
	return nil
}

func (e *PostfixEmitter) VisitBoolIntGt(ctx *BoolIntGtContext) interface{} {
	e.Visit(ctx.Iexpr(0))
	e.Visit(ctx.Iexpr(1))
	e.emit("gt")
	return nil
}

func (e *PostfixEmitter) VisitBoolIntGte(ctx *BoolIntGteContext) interface{} {
	e.Visit(ctx.Iexpr(0))
	e.Visit(ctx.Iexpr(1))
	e.emit("ge")
	return nil
}

func (e *PostfixEmitter) VisitBoolIntLt(ctx *BoolIntLtContext) interface{} {
	e.Visit(ctx.Iexpr(0))
	e.Visit(ctx.Iexpr(1))
	e.emit("lt")
	return nil
}

func (e *PostfixEmitter) VisitBoolIntLte(ctx *BoolIntLteContext) interface{} {
	e.Visit(ctx.Iexpr(0))
	e.Visit(ctx.Iexpr(1))
	e.emit("le")
	return nil
}

func (e *PostfixEmitter) VisitBoolFloatEq(ctx *BoolFloatEqContext) interface{} {
	e.Visit(ctx.Fexpr(0))
	e.Visit(ctx.Fexpr(1))
	e.emit("eq")
	return nil
}

func (e *PostfixEmitter) VisitBoolFloatNeq(ctx *BoolFloatNeqContext) interface{} {
	e.Visit(ctx.Fexpr(0))
	e.Visit(ctx.Fexpr(1))
	e.emit("ne")
	return nil
}

func (e *PostfixEmitter) VisitBoolFloatGt(ctx *BoolFloatGtContext) interface{} {
	e.Visit(ctx.Fexpr(0))
	e.Visit(ctx.Fexpr(1))
	e.emit("gt")
	return nil
}

func (e *PostfixEmitter) VisitBoolFloatLt(ctx *BoolFloatLtContext) interface{} {
	e.Visit(ctx.Fexpr(0))
	e.Visit(ctx.Fexpr(1))
	e.emit("lt")
	return nil
}

func (e *PostfixEmitter) VisitBoolFloatGte(ctx *BoolFloatGteContext) interface{} {
	e.Visit(ctx.Fexpr(0))
	e.Visit(ctx.Fexpr(1))
	e.emit("ge")
	return nil
}

func (e *PostfixEmitter) VisitBoolFloatLte(ctx *BoolFloatLteContext) interface{} {
	e.Visit(ctx.Fexpr(0))
	e.Visit(ctx.Fexpr(1))
	e.emit("le")
	return nil
}

func (e *PostfixEmitter) VisitBoolStrEq(ctx *BoolStrEqContext) interface{} {
	e.Visit(ctx.Strexpr(0))
	e.Visit(ctx.Strexpr(1))
	e.emit("streq")
	return nil
}

func (e *PostfixEmitter) VisitBoolStrNeq(ctx *BoolStrNeqContext) interface{} {
	e.Visit(ctx.Strexpr(0))
	e.Visit(ctx.Strexpr(1))
	e.emit("strneq")
	return nil
}

func (e *PostfixEmitter) VisitBoolStrIs(ctx *BoolStrIsContext) interface{} {
	e.Visit(ctx.Strexpr(0))
	e.Visit(ctx.Strexpr(1))
	e.emit("streq")
	return nil
}

func (e *PostfixEmitter) VisitBoolStrIsNot(ctx *BoolStrIsNotContext) interface{} {
	e.Visit(ctx.Strexpr(0))
	e.Visit(ctx.Strexpr(1))
	e.emit("strneq")
	e.emit("not")
	return nil
}

func (e *PostfixEmitter) VisitBoolStrEqIc(ctx *BoolStrEqIcContext) interface{} {
	e.Visit(ctx.Strexpr(0))
	e.Visit(ctx.Strexpr(1))
	e.emit("streqic")
	return nil
}

func (e *PostfixEmitter) VisitBoolStrNeqIc(ctx *BoolStrNeqIcContext) interface{} {
	e.Visit(ctx.Strexpr(0))
	e.Visit(ctx.Strexpr(1))
	e.emit("streqic")
	e.emit("not")
	return nil
}

func (e *PostfixEmitter) VisitBoolAnd(ctx *BoolAndContext) interface{} {
	e.Visit(ctx.Bexpr(0))
	e.Visit(ctx.Bexpr(1))
	e.emit("and")
	return nil
}

func (e *PostfixEmitter) VisitBoolOr(ctx *BoolOrContext) interface{} {
	e.Visit(ctx.Bexpr(0))
	e.Visit(ctx.Bexpr(1))
	e.emit("or")
	return nil
}

func (e *PostfixEmitter) VisitBoolNot(ctx *BoolNotContext) interface{} {
	e.Visit(ctx.Bexpr())
	e.emit("not")
	return nil
}

func (e *PostfixEmitter) VisitBoolTypedIsLiteral(ctx *BoolTypedIsLiteralContext) interface{} {
	// typedBoolean IS RBOOLEAN -> flag.a is true => flag.a true eq
	e.Visit(ctx.TypedBoolean())
	text := strings.ToLower(ctx.RBOOLEAN().GetText())
	e.emit(text)
	e.emit("eq")
	return nil
}

func (e *PostfixEmitter) VisitBoolTypedIsNotLiteral(ctx *BoolTypedIsNotLiteralContext) interface{} {
	// typedBoolean IS NOT RBOOLEAN -> flag.a is not true => flag.a true neq
	e.Visit(ctx.TypedBoolean())
	text := strings.ToLower(ctx.RBOOLEAN().GetText())
	e.emit(text)
	e.emit("neq")
	return nil
}

func (e *PostfixEmitter) VisitBoolColonIsLiteral(ctx *BoolColonIsLiteralContext) interface{} {
	// colonRef typedBoolean IS RBOOLEAN
	e.Visit(ctx.ColonRef())
	e.Visit(ctx.TypedBoolean())
	text := strings.ToLower(ctx.RBOOLEAN().GetText())
	e.emit(text)
	e.emit("eq")
	return nil
}

func (e *PostfixEmitter) VisitBoolColonIsNotLiteral(ctx *BoolColonIsNotLiteralContext) interface{} {
	// colonRef typedBoolean IS NOT RBOOLEAN
	e.Visit(ctx.ColonRef())
	e.Visit(ctx.TypedBoolean())
	text := strings.ToLower(ctx.RBOOLEAN().GetText())
	e.emit(text)
	e.emit("neq")
	return nil
}

func (e *PostfixEmitter) VisitBoolParen(ctx *BoolParenContext) interface{} {
	e.Visit(ctx.Bexpr())
	return nil
}

func (e *PostfixEmitter) VisitBoolLiteral(ctx *BoolLiteralContext) interface{} {
	text := strings.ToLower(ctx.GetText())
	switch text {
	case "true":
		e.emit("true")
	case "false":
		e.emit("false")
	case "otherwise", "default", "always":
		e.emit("otherwise")
	default:
		e.emit(text)
	}
	return nil
}

func (e *PostfixEmitter) VisitBoolTyped(ctx *BoolTypedContext) interface{} {
	e.Visit(ctx.TypedBoolean())
	return nil
}

func (e *PostfixEmitter) VisitBoolDateEq(ctx *BoolDateEqContext) interface{} {
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(1))
	e.emit("eq")
	return nil
}

func (e *PostfixEmitter) VisitBoolDateLt(ctx *BoolDateLtContext) interface{} {
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(1))
	e.emit("lt")
	return nil
}

func (e *PostfixEmitter) VisitBoolDateGt(ctx *BoolDateGtContext) interface{} {
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(1))
	e.emit("gt")
	return nil
}

func (e *PostfixEmitter) VisitBoolDateBefore(ctx *BoolDateBeforeContext) interface{} {
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(1))
	e.emit("lt")
	return nil
}

func (e *PostfixEmitter) VisitBoolDateAfter(ctx *BoolDateAfterContext) interface{} {
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(1))
	e.emit("gt")
	return nil
}

func (e *PostfixEmitter) VisitBoolDateGte(ctx *BoolDateGteContext) interface{} {
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(1))
	e.emit("ge")
	return nil
}

func (e *PostfixEmitter) VisitBoolDateLte(ctx *BoolDateLteContext) interface{} {
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(1))
	e.emit("le")
	return nil
}

func (e *PostfixEmitter) VisitBoolEntityEq(ctx *BoolEntityEqContext) interface{} {
	e.Visit(ctx.Eexpr(0))
	e.Visit(ctx.Eexpr(1))
	e.emit("entityeq")
	return nil
}

func (e *PostfixEmitter) VisitBoolEntityNeq(ctx *BoolEntityNeqContext) interface{} {
	e.Visit(ctx.Eexpr(0))
	e.Visit(ctx.Eexpr(1))
	e.emit("entityeq")
	e.emit("not")
	return nil
}

func (e *PostfixEmitter) VisitBoolNameEq(ctx *BoolNameEqContext) interface{} {
	e.Visit(ctx.Nexpr(0))
	e.Visit(ctx.Nexpr(1))
	e.emit("eq")
	return nil
}

func (e *PostfixEmitter) VisitBoolNameNeq(ctx *BoolNameNeqContext) interface{} {
	e.Visit(ctx.Nexpr(0))
	e.Visit(ctx.Nexpr(1))
	e.emit("ne")
	return nil
}

func (e *PostfixEmitter) VisitBoolNameEqStr(ctx *BoolNameEqStrContext) interface{} {
	e.Visit(ctx.Nexpr())
	e.Visit(ctx.Strexpr())
	e.emit("streq")
	return nil
}

func (e *PostfixEmitter) VisitBoolNameNeqStr(ctx *BoolNameNeqStrContext) interface{} {
	e.Visit(ctx.Nexpr())
	e.Visit(ctx.Strexpr())
	e.emit("streq")
	e.emit("not")
	return nil
}

func (e *PostfixEmitter) VisitBoolStrIsNull(ctx *BoolStrIsNullContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("isnull")
	return nil
}

func (e *PostfixEmitter) VisitBoolStrIsNotNull(ctx *BoolStrIsNotNullContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("isnull")
	e.emit("not")
	return nil
}

func (e *PostfixEmitter) VisitBoolEntityIsNull(ctx *BoolEntityIsNullContext) interface{} {
	e.Visit(ctx.Eexpr())
	e.emit("isnull")
	return nil
}

func (e *PostfixEmitter) VisitBoolEntityIsNotNull(ctx *BoolEntityIsNotNullContext) interface{} {
	e.Visit(ctx.Eexpr())
	e.emit("isnull")
	e.emit("not")
	return nil
}

func (e *PostfixEmitter) VisitBoolDateIsNull(ctx *BoolDateIsNullContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.emit("isnull")
	return nil
}

func (e *PostfixEmitter) VisitBoolDateIsNotNull(ctx *BoolDateIsNotNullContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.emit("isnull")
	e.emit("not")
	return nil
}

func (e *PostfixEmitter) VisitBoolArrayIsNull(ctx *BoolArrayIsNullContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.emit("isnull")
	return nil
}

func (e *PostfixEmitter) VisitBoolArrayIsNotNull(ctx *BoolArrayIsNotNullContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.emit("isnull")
	e.emit("not")
	return nil
}

func (e *PostfixEmitter) VisitBoolNumIsNull(ctx *BoolNumIsNullContext) interface{} {
	e.Visit(ctx.Number())
	e.emit("isnull")
	return nil
}

func (e *PostfixEmitter) VisitBoolNumIsNotNull(ctx *BoolNumIsNotNullContext) interface{} {
	e.Visit(ctx.Number())
	e.emit("isnull")
	e.emit("not")
	return nil
}

func (e *PostfixEmitter) VisitBoolArrayIncludes(ctx *BoolArrayIncludesContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.Visit(ctx.IncludeSearch())
	e.emit("member")
	return nil
}

func (e *PostfixEmitter) VisitBoolArrayDoesInclude(ctx *BoolArrayDoesIncludeContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.Visit(ctx.IncludeSearch())
	e.emit("member")
	return nil
}

func (e *PostfixEmitter) VisitBoolArrayNotInclude(ctx *BoolArrayNotIncludeContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.Visit(ctx.IncludeSearch())
	e.emit("member")
	e.emit("not")
	return nil
}

// ============================================================================
// Integer Expression Visitors
// ============================================================================

func (e *PostfixEmitter) VisitIntLiteral(ctx *IntLiteralContext) interface{} {
	e.emit(ctx.GetText())
	return nil
}

func (e *PostfixEmitter) VisitIntTyped(ctx *IntTypedContext) interface{} {
	e.Visit(ctx.TypedLong())
	return nil
}

func (e *PostfixEmitter) VisitIntAdd(ctx *IntAddContext) interface{} {
	e.Visit(ctx.Iexpr(0))
	e.Visit(ctx.Iexpr(1))
	e.emit("+")
	return nil
}

func (e *PostfixEmitter) VisitIntSub(ctx *IntSubContext) interface{} {
	e.Visit(ctx.Iexpr(0))
	e.Visit(ctx.Iexpr(1))
	e.emit("-")
	return nil
}

func (e *PostfixEmitter) VisitIntMul(ctx *IntMulContext) interface{} {
	e.Visit(ctx.Iexpr(0))
	e.Visit(ctx.Iexpr(1))
	e.emit("*")
	return nil
}

func (e *PostfixEmitter) VisitIntDiv(ctx *IntDivContext) interface{} {
	e.Visit(ctx.Iexpr(0))
	e.Visit(ctx.Iexpr(1))
	e.emit("/")
	return nil
}

func (e *PostfixEmitter) VisitIntNegate(ctx *IntNegateContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.emit("neg")
	return nil
}

func (e *PostfixEmitter) VisitIntParen(ctx *IntParenContext) interface{} {
	e.Visit(ctx.Iexpr())
	return nil
}

func (e *PostfixEmitter) VisitIntNumberOf(ctx *IntNumberOfContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.emit("numberof")
	return nil
}

func (e *PostfixEmitter) VisitIntLengthArray(ctx *IntLengthArrayContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.emit("length")
	return nil
}

func (e *PostfixEmitter) VisitIntLengthStr(ctx *IntLengthStrContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("length")
	return nil
}

// ============================================================================
// Float Expression Visitors
// ============================================================================

func (e *PostfixEmitter) VisitFloatLiteral(ctx *FloatLiteralContext) interface{} {
	e.emit(ctx.GetText())
	return nil
}

func (e *PostfixEmitter) VisitFloatTyped(ctx *FloatTypedContext) interface{} {
	e.Visit(ctx.TypedDouble())
	return nil
}

func (e *PostfixEmitter) VisitFloatAddFloat(ctx *FloatAddFloatContext) interface{} {
	e.Visit(ctx.Fexpr(0))
	e.Visit(ctx.Fexpr(1))
	e.emit("+")
	return nil
}

func (e *PostfixEmitter) VisitFloatSubFloat(ctx *FloatSubFloatContext) interface{} {
	e.Visit(ctx.Fexpr(0))
	e.Visit(ctx.Fexpr(1))
	e.emit("-")
	return nil
}

func (e *PostfixEmitter) VisitFloatMulFloat(ctx *FloatMulFloatContext) interface{} {
	e.Visit(ctx.Fexpr(0))
	e.Visit(ctx.Fexpr(1))
	e.emit("*")
	return nil
}

func (e *PostfixEmitter) VisitFloatDivFloat(ctx *FloatDivFloatContext) interface{} {
	e.Visit(ctx.Fexpr(0))
	e.Visit(ctx.Fexpr(1))
	e.emit("/")
	return nil
}

func (e *PostfixEmitter) VisitFloatAddInt(ctx *FloatAddIntContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.Visit(ctx.Iexpr())
	e.emit("+")
	return nil
}

func (e *PostfixEmitter) VisitFloatSubInt(ctx *FloatSubIntContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.Visit(ctx.Iexpr())
	e.emit("-")
	return nil
}

func (e *PostfixEmitter) VisitFloatMulInt(ctx *FloatMulIntContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.Visit(ctx.Iexpr())
	e.emit("*")
	return nil
}

func (e *PostfixEmitter) VisitFloatDivInt(ctx *FloatDivIntContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.Visit(ctx.Iexpr())
	e.emit("/")
	return nil
}

func (e *PostfixEmitter) VisitIntAddFloat(ctx *IntAddFloatContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.Visit(ctx.Fexpr())
	e.emit("+")
	return nil
}

func (e *PostfixEmitter) VisitIntSubFloat(ctx *IntSubFloatContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.Visit(ctx.Fexpr())
	e.emit("-")
	return nil
}

func (e *PostfixEmitter) VisitIntMulFloat(ctx *IntMulFloatContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.Visit(ctx.Fexpr())
	e.emit("*")
	return nil
}

func (e *PostfixEmitter) VisitIntDivFloat(ctx *IntDivFloatContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.Visit(ctx.Fexpr())
	e.emit("/")
	return nil
}

func (e *PostfixEmitter) VisitFloatNegate(ctx *FloatNegateContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.emit("neg")
	return nil
}

func (e *PostfixEmitter) VisitFloatParen(ctx *FloatParenContext) interface{} {
	e.Visit(ctx.Fexpr())
	return nil
}

func (e *PostfixEmitter) VisitFloatRounded(ctx *FloatRoundedContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.emit("round")
	return nil
}

// ============================================================================
// String Expression Visitors
// ============================================================================

func (e *PostfixEmitter) VisitStrLiteral(ctx *StrLiteralContext) interface{} {
	text := ctx.GetText()
	// Keep the quotes for string literals
	e.emit(text)
	return nil
}

func (e *PostfixEmitter) VisitStrTyped(ctx *StrTypedContext) interface{} {
	e.Visit(ctx.TypedString())
	return nil
}

func (e *PostfixEmitter) VisitStrConcat(ctx *StrConcatContext) interface{} {
	e.Visit(ctx.Strexpr(0))
	e.Visit(ctx.Strexpr(1))
	e.emit("strconcat")
	return nil
}

func (e *PostfixEmitter) VisitStrParen(ctx *StrParenContext) interface{} {
	e.Visit(ctx.Strexpr())
	return nil
}

func (e *PostfixEmitter) VisitStrTrim(ctx *StrTrimContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("trim")
	return nil
}

func (e *PostfixEmitter) VisitStrToLower(ctx *StrToLowerContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("tolower")
	return nil
}

func (e *PostfixEmitter) VisitStrToUpper(ctx *StrToUpperContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("toupper")
	return nil
}

// ============================================================================
// Entity Expression Visitors
// ============================================================================

func (e *PostfixEmitter) VisitEntityTyped(ctx *EntityTypedContext) interface{} {
	e.Visit(ctx.TypedEntity())
	return nil
}

func (e *PostfixEmitter) VisitEntityParen(ctx *EntityParenContext) interface{} {
	e.Visit(ctx.Eexpr())
	return nil
}

func (e *PostfixEmitter) VisitEntityNewTyped(ctx *EntityNewTypedContext) interface{} {
	e.Visit(ctx.TypedEntity())
	e.emit("newentity")
	return nil
}

func (e *PostfixEmitter) VisitEntityClone(ctx *EntityCloneContext) interface{} {
	e.Visit(ctx.Eexpr())
	e.emit("clone")
	return nil
}

// ============================================================================
// Date Expression Visitors
// ============================================================================

func (e *PostfixEmitter) VisitDateTyped(ctx *DateTypedContext) interface{} {
	e.Visit(ctx.TypedDate())
	return nil
}

func (e *PostfixEmitter) VisitDateParen(ctx *DateParenContext) interface{} {
	e.Visit(ctx.Dexpr())
	return nil
}

func (e *PostfixEmitter) VisitDateCurrentDate(ctx *DateCurrentDateContext) interface{} {
	e.emit("currentdate")
	return nil
}

func (e *PostfixEmitter) VisitDateAdd(ctx *DateAddContext) interface{} {
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(1))
	e.emit("+")
	return nil
}

func (e *PostfixEmitter) VisitDateSub(ctx *DateSubContext) interface{} {
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(1))
	e.emit("-")
	return nil
}

// ============================================================================
// Name Expression Visitors
// ============================================================================

func (e *PostfixEmitter) VisitNameTyped(ctx *NameTypedContext) interface{} {
	e.Visit(ctx.TypedName())
	return nil
}

func (e *PostfixEmitter) VisitNameLiteral(ctx *NameLiteralContext) interface{} {
	text := ctx.GetText()
	if strings.HasPrefix(text, "$") {
		e.emit(text)
	} else {
		e.emit("/" + text)
	}
	return nil
}

func (e *PostfixEmitter) VisitNameOf(ctx *NameOfContext) interface{} {
	e.Visit(ctx.Eexpr())
	e.emit("nameof")
	return nil
}

// ============================================================================
// Array Expression Visitors
// ============================================================================

func (e *PostfixEmitter) VisitArrayTyped(ctx *ArrayTypedContext) interface{} {
	e.Visit(ctx.TypedArray())
	return nil
}

func (e *PostfixEmitter) VisitArrayParen(ctx *ArrayParenContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	return nil
}

func (e *PostfixEmitter) VisitArrayBase(ctx *ArrayBaseContext) interface{} {
	e.Visit(ctx.ArrayExpr2())
	return nil
}

func (e *PostfixEmitter) VisitArrayCopy(ctx *ArrayCopyContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.emit("copy")
	return nil
}

func (e *PostfixEmitter) VisitArrayCopySimple(ctx *ArrayCopySimpleContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.emit("copy")
	return nil
}

// ============================================================================
// Typed Identifier Visitors
// ============================================================================

func (e *PostfixEmitter) VisitTypedEntity(ctx *TypedEntityContext) interface{} {
	e.emit(ctx.GetText())
	return nil
}

func (e *PostfixEmitter) VisitTypedLong(ctx *TypedLongContext) interface{} {
	e.emit(ctx.GetText())
	return nil
}

func (e *PostfixEmitter) VisitTypedDouble(ctx *TypedDoubleContext) interface{} {
	e.emit(ctx.GetText())
	return nil
}

func (e *PostfixEmitter) VisitTypedString(ctx *TypedStringContext) interface{} {
	e.emit(ctx.GetText())
	return nil
}

func (e *PostfixEmitter) VisitTypedBoolean(ctx *TypedBooleanContext) interface{} {
	e.emit(ctx.GetText())
	return nil
}

func (e *PostfixEmitter) VisitTypedDate(ctx *TypedDateContext) interface{} {
	e.emit(ctx.GetText())
	return nil
}

func (e *PostfixEmitter) VisitTypedArray(ctx *TypedArrayContext) interface{} {
	e.emit(ctx.GetText())
	return nil
}

func (e *PostfixEmitter) VisitTypedTable(ctx *TypedTableContext) interface{} {
	e.emit(ctx.GetText())
	return nil
}

func (e *PostfixEmitter) VisitTypedName(ctx *TypedNameContext) interface{} {
	e.emit("/" + ctx.GetText())
	return nil
}

func (e *PostfixEmitter) VisitTypedDecisionTable(ctx *TypedDecisionTableContext) interface{} {
	e.emit(ctx.GetText())
	return nil
}

// ============================================================================
// Number Visitors
// ============================================================================

func (e *PostfixEmitter) VisitNumber(ctx *NumberContext) interface{} {
	if ctx.Iexpr() != nil {
		e.Visit(ctx.Iexpr())
	} else if ctx.Fexpr() != nil {
		e.Visit(ctx.Fexpr())
	}
	return nil
}

// ============================================================================
// Include Search Visitors
// ============================================================================

func (e *PostfixEmitter) VisitIncludeNumber(ctx *IncludeNumberContext) interface{} {
	e.Visit(ctx.Number())
	return nil
}

func (e *PostfixEmitter) VisitIncludeEntity(ctx *IncludeEntityContext) interface{} {
	e.Visit(ctx.Eexpr())
	return nil
}

func (e *PostfixEmitter) VisitIncludeString(ctx *IncludeStringContext) interface{} {
	e.Visit(ctx.Strexpr())
	return nil
}

func (e *PostfixEmitter) VisitIncludeDate(ctx *IncludeDateContext) interface{} {
	e.Visit(ctx.Dexpr())
	return nil
}

// ============================================================================
// Statement Visitors
// ============================================================================

func (e *PostfixEmitter) VisitPerformDT(ctx *PerformDTContext) interface{} {
	e.Visit(ctx.TypedDecisionTable())
	e.emit("executetable")
	return nil
}

func (e *PostfixEmitter) VisitPerformDTExplicit(ctx *PerformDTExplicitContext) interface{} {
	e.Visit(ctx.TypedDecisionTable())
	e.emit("executetable")
	return nil
}

// ============================================================================
// Done (Entry Point) Visitors
// ============================================================================

func (e *PostfixEmitter) VisitConditionExpr(ctx *ConditionExprContext) interface{} {
	e.Visit(ctx.Bexpr())
	return nil
}

func (e *PostfixEmitter) VisitActionStatement(ctx *ActionStatementContext) interface{} {
	e.Visit(ctx.StatementList())
	return nil
}

func (e *PostfixEmitter) VisitEmptyAction(ctx *EmptyActionContext) interface{} {
	return nil
}

func (e *PostfixEmitter) VisitEmptyCondition(ctx *EmptyConditionContext) interface{} {
	return nil
}

// ============================================================================
// Statement List Visitors
// ============================================================================

func (e *PostfixEmitter) VisitStatementList(ctx *StatementListContext) interface{} {
	for _, block := range ctx.AllBlock() {
		e.Visit(block)
	}
	return nil
}

func (e *PostfixEmitter) VisitBlockStatement(ctx *BlockStatementContext) interface{} {
	e.Visit(ctx.Statement())
	return nil
}

func (e *PostfixEmitter) VisitBlockCurly(ctx *BlockCurlyContext) interface{} {
	e.Visit(ctx.StatementList())
	return nil
}

func (e *PostfixEmitter) VisitStatement(ctx *StatementContext) interface{} {
	// Visit all children - one will be the actual statement
	for i := 0; i < ctx.GetChildCount(); i++ {
		child := ctx.GetChild(i)
		if parseTree, ok := child.(antlr.ParseTree); ok {
			e.Visit(parseTree)
		}
	}
	return nil
}

// ============================================================================
// Set Statement Visitors
// ============================================================================

func (e *PostfixEmitter) VisitSetInt(ctx *SetIntContext) interface{} {
	e.Visit(ctx.Number())
	e.Visit(ctx.LeftIexpr())
	e.emit("=")
	return nil
}

func (e *PostfixEmitter) VisitSetFloat(ctx *SetFloatContext) interface{} {
	e.Visit(ctx.Number())
	e.Visit(ctx.LeftFexpr())
	e.emit("=")
	return nil
}

func (e *PostfixEmitter) VisitSetBool(ctx *SetBoolContext) interface{} {
	e.Visit(ctx.Bexpr())
	e.Visit(ctx.LeftBexpr())
	e.emit("=")
	return nil
}

func (e *PostfixEmitter) VisitSetEntity(ctx *SetEntityContext) interface{} {
	e.Visit(ctx.Eexpr())
	e.Visit(ctx.LeftEexpr())
	e.emit("=")
	return nil
}

func (e *PostfixEmitter) VisitSetString(ctx *SetStringContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.Visit(ctx.LeftStrexpr())
	e.emit("=")
	return nil
}

func (e *PostfixEmitter) VisitSetDate(ctx *SetDateContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.LeftDexpr())
	e.emit("=")
	return nil
}

func (e *PostfixEmitter) VisitLeftIexprSimple(ctx *LeftIexprSimpleContext) interface{} {
	e.Visit(ctx.TypedLong())
	return nil
}

func (e *PostfixEmitter) VisitLeftFexprSimple(ctx *LeftFexprSimpleContext) interface{} {
	e.Visit(ctx.TypedDouble())
	return nil
}

func (e *PostfixEmitter) VisitLeftBexprSimple(ctx *LeftBexprSimpleContext) interface{} {
	e.Visit(ctx.TypedBoolean())
	return nil
}

func (e *PostfixEmitter) VisitLeftEexprSimple(ctx *LeftEexprSimpleContext) interface{} {
	e.Visit(ctx.TypedEntity())
	return nil
}

func (e *PostfixEmitter) VisitLeftStrexprSimple(ctx *LeftStrexprSimpleContext) interface{} {
	e.Visit(ctx.TypedString())
	return nil
}

func (e *PostfixEmitter) VisitLeftDexprSimple(ctx *LeftDexprSimpleContext) interface{} {
	e.Visit(ctx.TypedDate())
	return nil
}

// ============================================================================
// If Statement Visitors
// ============================================================================

func (e *PostfixEmitter) VisitBlockIf(ctx *BlockIfContext) interface{} {
	e.Visit(ctx.Ifblock())
	return nil
}

func (e *PostfixEmitter) VisitIfblock(ctx *IfblockContext) interface{} {
	e.Visit(ctx.Bexpr())
	e.emit("{")
	e.Visit(ctx.StatementList())
	e.emit("}")
	e.Visit(ctx.Ifcontinue())
	e.emit("ifelse")
	return nil
}

func (e *PostfixEmitter) VisitIfEnd(ctx *IfEndContext) interface{} {
	e.emit("{}")
	return nil
}

func (e *PostfixEmitter) VisitIfElse(ctx *IfElseContext) interface{} {
	e.emit("{")
	e.Visit(ctx.StatementList())
	e.emit("}")
	return nil
}

// ============================================================================
// Relationship and Entity Tests
// ============================================================================

func (e *PostfixEmitter) VisitBoolEntityIsOf(ctx *BoolEntityIsOfContext) interface{} {
	// "eexpr IS strexpr OF eexpr" -> "e1 e2 relationship streq"
	e.Visit(ctx.Eexpr(0))
	e.Visit(ctx.Eexpr(1))
	e.emit("relationship")
	e.Visit(ctx.Strexpr())
	e.emit("streq")
	return nil
}

func (e *PostfixEmitter) VisitEntityRelationship(ctx *EntityRelationshipContext) interface{} {
	// "strexpr OF eexpr" -> get entity via relationship
	e.Visit(ctx.Eexpr())
	e.Visit(ctx.Strexpr())
	e.emit("getrelationship")
	return nil
}

func (e *PostfixEmitter) VisitBoolEntityHasa(ctx *BoolEntityHasaContext) interface{} {
	e.Visit(ctx.Eexpr())
	e.Visit(ctx.Strexpr())
	e.emit("hasrelationship")
	return nil
}

func (e *PostfixEmitter) VisitBoolEntityNotHas(ctx *BoolEntityNotHasContext) interface{} {
	e.Visit(ctx.Eexpr())
	e.Visit(ctx.Strexpr())
	e.emit("hasrelationship")
	e.emit("not")
	return nil
}

// ============================================================================
// Date Arithmetic Visitors
// ============================================================================

func (e *PostfixEmitter) VisitDateDays(ctx *DateDaysContext) interface{} {
	e.Visit(ctx.Number())
	e.emit("days")
	return nil
}

func (e *PostfixEmitter) VisitDatePlusDays(ctx *DatePlusDaysContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Number())
	e.emit("adddays")
	return nil
}

func (e *PostfixEmitter) VisitDateMinusDays(ctx *DateMinusDaysContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Number())
	e.emit("subdays")
	return nil
}

func (e *PostfixEmitter) VisitDatePlusMonths(ctx *DatePlusMonthsContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Number())
	e.emit("addmonths")
	return nil
}

func (e *PostfixEmitter) VisitDateMinusMonths(ctx *DateMinusMonthsContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Number())
	e.emit("submonths")
	return nil
}

func (e *PostfixEmitter) VisitDatePlusYears(ctx *DatePlusYearsContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Number())
	e.emit("addyears")
	return nil
}

func (e *PostfixEmitter) VisitDateMinusYears(ctx *DateMinusYearsContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Number())
	e.emit("subyears")
	return nil
}

// ============================================================================
// Add Statement Visitors
// ============================================================================

func (e *PostfixEmitter) VisitAddEntityToDest(ctx *AddEntityToDestContext) interface{} {
	e.Visit(ctx.Eexpr())
	e.Visit(ctx.Addtodest())
	e.emit("addto")
	return nil
}

func (e *PostfixEmitter) VisitAddStrToDest(ctx *AddStrToDestContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.Visit(ctx.Addtodest())
	e.emit("addto")
	return nil
}

func (e *PostfixEmitter) VisitAddNumToDest(ctx *AddNumToDestContext) interface{} {
	e.Visit(ctx.Number())
	e.Visit(ctx.Addtodest())
	e.emit("+")
	e.emit("=")
	return nil
}

func (e *PostfixEmitter) VisitAddDestArray(ctx *AddDestArrayContext) interface{} {
	e.Visit(ctx.ArrayExpr2())
	return nil
}

func (e *PostfixEmitter) VisitAddDestLong(ctx *AddDestLongContext) interface{} {
	e.Visit(ctx.TypedLong())
	return nil
}

func (e *PostfixEmitter) VisitAddDestDouble(ctx *AddDestDoubleContext) interface{} {
	e.Visit(ctx.TypedDouble())
	return nil
}

func (e *PostfixEmitter) VisitAddDestPossessiveLong(ctx *AddDestPossessiveLongContext) interface{} {
	// Emit the possessive entity reference and field
	poss := ctx.POSSESSIVE().GetText()
	// Remove 's suffix to get entity name
	entityName := poss[:len(poss)-2]
	e.emit(entityName)
	e.emit("entitypush")
	e.Visit(ctx.TypedLong())
	return nil
}

func (e *PostfixEmitter) VisitAddDestPossessiveDouble(ctx *AddDestPossessiveDoubleContext) interface{} {
	// Emit the possessive entity reference and field
	poss := ctx.POSSESSIVE().GetText()
	// Remove 's suffix to get entity name
	entityName := poss[:len(poss)-2]
	e.emit(entityName)
	e.emit("entitypush")
	e.Visit(ctx.TypedDouble())
	return nil
}

func (e *PostfixEmitter) VisitSubDestPossessiveLong(ctx *SubDestPossessiveLongContext) interface{} {
	// Emit the possessive entity reference and field
	poss := ctx.POSSESSIVE().GetText()
	// Remove 's suffix to get entity name
	entityName := poss[:len(poss)-2]
	e.emit(entityName)
	e.emit("entitypush")
	e.Visit(ctx.TypedLong())
	return nil
}

func (e *PostfixEmitter) VisitSubDestPossessiveDouble(ctx *SubDestPossessiveDoubleContext) interface{} {
	// Emit the possessive entity reference and field
	poss := ctx.POSSESSIVE().GetText()
	// Remove 's suffix to get entity name
	entityName := poss[:len(poss)-2]
	e.emit(entityName)
	e.emit("entitypush")
	e.Visit(ctx.TypedDouble())
	return nil
}

// ============================================================================
// Context Statement Visitors
// ============================================================================

func (e *PostfixEmitter) VisitAddToContextOf(ctx *AddToContextOfContext) interface{} {
	e.Visit(ctx.Eexpr())
	e.emit("addtocontext")
	return nil
}

func (e *PostfixEmitter) VisitAddToContextFor(ctx *AddToContextForContext) interface{} {
	e.Visit(ctx.Eexpr())
	e.emit("addtocontext")
	return nil
}

// ============================================================================
// Mixed Type Comparison Visitors
// ============================================================================

func (e *PostfixEmitter) VisitBoolFloatEqInt(ctx *BoolFloatEqIntContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.Visit(ctx.Iexpr())
	e.emit("eq")
	return nil
}

func (e *PostfixEmitter) VisitBoolIntEqFloat(ctx *BoolIntEqFloatContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.Visit(ctx.Fexpr())
	e.emit("eq")
	return nil
}

func (e *PostfixEmitter) VisitBoolFloatNeqInt(ctx *BoolFloatNeqIntContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.Visit(ctx.Iexpr())
	e.emit("ne")
	return nil
}

func (e *PostfixEmitter) VisitBoolIntNeqFloat(ctx *BoolIntNeqFloatContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.Visit(ctx.Fexpr())
	e.emit("ne")
	return nil
}

func (e *PostfixEmitter) VisitBoolFloatGtInt(ctx *BoolFloatGtIntContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.Visit(ctx.Iexpr())
	e.emit("gt")
	return nil
}

func (e *PostfixEmitter) VisitBoolIntGtFloat(ctx *BoolIntGtFloatContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.Visit(ctx.Fexpr())
	e.emit("gt")
	return nil
}

func (e *PostfixEmitter) VisitBoolFloatLtInt(ctx *BoolFloatLtIntContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.Visit(ctx.Iexpr())
	e.emit("lt")
	return nil
}

func (e *PostfixEmitter) VisitBoolIntLtFloat(ctx *BoolIntLtFloatContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.Visit(ctx.Fexpr())
	e.emit("lt")
	return nil
}

func (e *PostfixEmitter) VisitBoolFloatGteInt(ctx *BoolFloatGteIntContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.Visit(ctx.Iexpr())
	e.emit("ge")
	return nil
}

func (e *PostfixEmitter) VisitBoolIntGteFloat(ctx *BoolIntGteFloatContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.Visit(ctx.Fexpr())
	e.emit("ge")
	return nil
}

func (e *PostfixEmitter) VisitBoolFloatLteInt(ctx *BoolFloatLteIntContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.Visit(ctx.Iexpr())
	e.emit("le")
	return nil
}

func (e *PostfixEmitter) VisitBoolIntLteFloat(ctx *BoolIntLteFloatContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.Visit(ctx.Fexpr())
	e.emit("le")
	return nil
}

// ============================================================================
// Possessive Reference Visitors
// ============================================================================

func (e *PostfixEmitter) VisitPossessiveChain(ctx *PossessiveChainContext) interface{} {
	// Handle possessive chains like "ThisClient's, parent's"
	e.emit(ctx.POSSESSIVE().GetText())
	e.Visit(ctx.PossessiveRef())
	return nil
}

func (e *PostfixEmitter) VisitPossessiveSingle(ctx *PossessiveSingleContext) interface{} {
	e.emit(ctx.POSSESSIVE().GetText())
	return nil
}

func (e *PostfixEmitter) VisitLeftIexprColon(ctx *LeftIexprColonContext) interface{} {
	e.Visit(ctx.ColonRef())
	e.Visit(ctx.LeftIexpr())
	return nil
}

func (e *PostfixEmitter) VisitLeftFexprColon(ctx *LeftFexprColonContext) interface{} {
	e.Visit(ctx.ColonRef())
	e.Visit(ctx.LeftFexpr())
	return nil
}

func (e *PostfixEmitter) VisitLeftBexprColon(ctx *LeftBexprColonContext) interface{} {
	e.Visit(ctx.ColonRef())
	e.Visit(ctx.LeftBexpr())
	return nil
}

func (e *PostfixEmitter) VisitLeftEexprColon(ctx *LeftEexprColonContext) interface{} {
	e.Visit(ctx.ColonRef())
	e.Visit(ctx.LeftEexpr())
	return nil
}

func (e *PostfixEmitter) VisitLeftStrexprColon(ctx *LeftStrexprColonContext) interface{} {
	e.Visit(ctx.ColonRef())
	e.Visit(ctx.LeftStrexpr())
	return nil
}

func (e *PostfixEmitter) VisitLeftDexprColon(ctx *LeftDexprColonContext) interface{} {
	e.Visit(ctx.ColonRef())
	e.Visit(ctx.LeftDexpr())
	return nil
}

func (e *PostfixEmitter) VisitColonRef(ctx *ColonRefContext) interface{} {
	e.Visit(ctx.PossessiveRef())
	return nil
}

// ============================================================================
// Policy Statements Visitor
// ============================================================================

func (e *PostfixEmitter) VisitArrayPolicyStatements(ctx *ArrayPolicyStatementsContext) interface{} {
	e.emit("policystatements")
	return nil
}
