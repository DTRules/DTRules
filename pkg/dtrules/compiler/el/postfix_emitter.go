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

// LocalVar tracks a local variable's stack frame index and type.
type LocalVar struct {
	Index int
	Type  string
}

// PostfixEmitter walks the EL parse tree and emits postfix notation.
type PostfixEmitter struct {
	*BaseELVisitor
	output    strings.Builder
	errors    []error
	symbols   map[string]string   // symbol table for type resolution from EDD
	locals    map[string]LocalVar // local variable stack frame indices
	localCnt  int                 // next available local variable index
}

// NewPostfixEmitter creates a new postfix emitter.
func NewPostfixEmitter() *PostfixEmitter {
	return &PostfixEmitter{
		BaseELVisitor: &BaseELVisitor{},
		symbols:       make(map[string]string),
		locals:        make(map[string]LocalVar),
		localCnt:      0,
	}
}

// SetSymbols sets the symbol table for type resolution.
func (e *PostfixEmitter) SetSymbols(symbols map[string]string) {
	e.symbols = symbols
}

// declareLocal registers a local variable and returns its stack frame index.
func (e *PostfixEmitter) declareLocal(name string, varType string) int {
	name = strings.ToLower(name)
	idx := e.localCnt
	e.locals[name] = LocalVar{Index: idx, Type: varType}
	e.localCnt++
	return idx
}

// lookupLocal returns the local variable info if it exists.
func (e *PostfixEmitter) lookupLocal(name string) (LocalVar, bool) {
	name = strings.ToLower(name)
	v, ok := e.locals[name]
	return v, ok
}

// isLocal returns true if the identifier is a local variable.
func (e *PostfixEmitter) isLocal(name string) bool {
	_, ok := e.lookupLocal(name)
	return ok
}

// emitLocalRef emits a local variable reference: "<index> local@"
func (e *PostfixEmitter) emitLocalRef(name string) bool {
	if v, ok := e.lookupLocal(name); ok {
		e.emit(fmt.Sprintf("%d", v.Index))
		e.emit("local@")
		return true
	}
	return false
}

// emitLocalAssign emits a local variable assignment: "<index> local!"
func (e *PostfixEmitter) emitLocalAssign(name string) bool {
	if v, ok := e.lookupLocal(name); ok {
		e.emit(fmt.Sprintf("%d", v.Index))
		e.emit("local!")
		return true
	}
	return false
}

// lookupType returns the type of an identifier from the symbol table.
// It checks both "entity.field" and "field" forms.
func (e *PostfixEmitter) lookupType(ident string) string {
	if e.symbols == nil {
		return ""
	}
	ident = strings.ToLower(ident)
	if t, ok := e.symbols[ident]; ok {
		return t
	}
	// Try just the field name if entity.field didn't match
	if idx := strings.LastIndex(ident, "."); idx >= 0 {
		field := ident[idx+1:]
		if t, ok := e.symbols[field]; ok {
			return t
		}
	}
	return ""
}

// typeConverter returns the appropriate type conversion operator for a type.
// Returns empty string if no conversion needed (e.g., for arrays).
func (e *PostfixEmitter) typeConverter(fieldType string) string {
	switch fieldType {
	case TypeInteger, TypeLong:
		return "cvi"
	case TypeDouble:
		return "cvd"
	case TypeString:
		return "cvs"
	case TypeBoolean:
		return "cvb"
	case TypeEntity:
		return "cve"
	case TypeDate:
		return "cvd"
	case TypeBigInt:
		return "cvbi"
	case TypeBytes:
		return "cvbytes"
	case TypeArray, TypeName, TypeTable, TypeXmlValue:
		return "" // No conversion needed
	default:
		return "cvi" // Default to integer conversion for unknown types
	}
}

// getExprType determines the type of an integer expression by examining its structure.
// Returns TypeBigInt if the expression involves bigint variables, otherwise TypeInteger.
func (e *PostfixEmitter) getExprType(ctx antlr.ParseTree) string {
	if ctx == nil {
		return TypeInteger
	}

	// Check for typed integer context (variable reference)
	if typedCtx, ok := ctx.(*IntTypedContext); ok {
		if tl := typedCtx.TypedLong(); tl != nil {
			if ident := tl.IDENT(); ident != nil {
				name := ident.GetText()
				// Check local variables first
				if lv, ok := e.lookupLocal(name); ok {
					return lv.Type
				}
				// Check symbol table
				if t := e.lookupType(name); t != "" {
					return t
				}
			}
		}
	}

	// Check for colon reference context (entity.field)
	if colonCtx, ok := ctx.(*IntColonRefContext); ok {
		if tl := colonCtx.TypedLong(); tl != nil {
			if ident := tl.IDENT(); ident != nil {
				name := ident.GetText()
				// Check local variables first
				if lv, ok := e.lookupLocal(name); ok {
					return lv.Type
				}
				// Check symbol table - try full entity.field name
				if cr := colonCtx.ColonRef(); cr != nil {
					fullName := cr.GetText() + "." + name
					if t := e.lookupType(fullName); t != "" {
						return t
					}
				}
				// Try just the field name
				if t := e.lookupType(name); t != "" {
					return t
				}
			}
		}
	}

	// For compound expressions, check children recursively
	switch c := ctx.(type) {
	case *IntAddContext:
		if e.getExprType(c.Iexpr(0)) == TypeBigInt || e.getExprType(c.Iexpr(1)) == TypeBigInt {
			return TypeBigInt
		}
	case *IntSubContext:
		if e.getExprType(c.Iexpr(0)) == TypeBigInt || e.getExprType(c.Iexpr(1)) == TypeBigInt {
			return TypeBigInt
		}
	case *IntMulContext:
		if e.getExprType(c.Iexpr(0)) == TypeBigInt || e.getExprType(c.Iexpr(1)) == TypeBigInt {
			return TypeBigInt
		}
	case *IntDivContext:
		if e.getExprType(c.Iexpr(0)) == TypeBigInt || e.getExprType(c.Iexpr(1)) == TypeBigInt {
			return TypeBigInt
		}
	case *IntNegateContext:
		return e.getExprType(c.Iexpr())
	case *IntParenContext:
		return e.getExprType(c.Iexpr())
	}

	return TypeInteger
}

// isBigIntExpr returns true if the expression involves bigint types.
func (e *PostfixEmitter) isBigIntExpr(ctx antlr.ParseTree) bool {
	return e.getExprType(ctx) == TypeBigInt
}

// emitWithBigIntConversion emits an integer expression, converting to bigint if needed.
// If the expression is integer but we need bigint (for mixed-type operations),
// it emits a cvbi conversion after the expression.
func (e *PostfixEmitter) emitWithBigIntConversion(ctx antlr.ParseTree, needsBigInt bool) {
	exprType := e.getExprType(ctx)
	e.Visit(ctx)
	if needsBigInt && exprType != TypeBigInt {
		e.emit("cvbi")
	}
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
	left, right := ctx.Iexpr(0), ctx.Iexpr(1)

	// If both sides are bytes-typed, use constant-time equality.
	if e.iexprIsBytes(left) && e.iexprIsBytes(right) {
		e.Visit(left)
		e.Visit(right)
		e.emit("bytes==")
		return nil
	}

	leftIsBigInt := e.isBigIntExpr(left)
	rightIsBigInt := e.isBigIntExpr(right)
	needsBigInt := leftIsBigInt || rightIsBigInt

	e.emitWithBigIntConversion(left, needsBigInt)
	e.emitWithBigIntConversion(right, needsBigInt)

	if needsBigInt {
		e.emit("b==")
	} else {
		e.emit("==")
	}
	return nil
}

func (e *PostfixEmitter) VisitBoolIntNeq(ctx *BoolIntNeqContext) interface{} {
	left, right := ctx.Iexpr(0), ctx.Iexpr(1)

	// If both sides are bytes-typed, use constant-time inequality.
	if e.iexprIsBytes(left) && e.iexprIsBytes(right) {
		e.Visit(left)
		e.Visit(right)
		e.emit("bytes!=")
		return nil
	}

	leftIsBigInt := e.isBigIntExpr(left)
	rightIsBigInt := e.isBigIntExpr(right)
	needsBigInt := leftIsBigInt || rightIsBigInt

	e.emitWithBigIntConversion(left, needsBigInt)
	e.emitWithBigIntConversion(right, needsBigInt)

	if needsBigInt {
		e.emit("b!=")
	} else {
		e.emit("==")
		e.emit("not")
	}
	return nil
}

func (e *PostfixEmitter) VisitBoolIntGt(ctx *BoolIntGtContext) interface{} {
	left, right := ctx.Iexpr(0), ctx.Iexpr(1)
	leftIsBigInt := e.isBigIntExpr(left)
	rightIsBigInt := e.isBigIntExpr(right)
	needsBigInt := leftIsBigInt || rightIsBigInt

	e.emitWithBigIntConversion(left, needsBigInt)
	e.emitWithBigIntConversion(right, needsBigInt)

	if needsBigInt {
		e.emit("b>")
	} else {
		e.emit(">")
	}
	return nil
}

func (e *PostfixEmitter) VisitBoolIntGte(ctx *BoolIntGteContext) interface{} {
	left, right := ctx.Iexpr(0), ctx.Iexpr(1)
	leftIsBigInt := e.isBigIntExpr(left)
	rightIsBigInt := e.isBigIntExpr(right)
	needsBigInt := leftIsBigInt || rightIsBigInt

	e.emitWithBigIntConversion(left, needsBigInt)
	e.emitWithBigIntConversion(right, needsBigInt)

	if needsBigInt {
		e.emit("b>=")
	} else {
		e.emit(">=")
	}
	return nil
}

func (e *PostfixEmitter) VisitBoolIntLt(ctx *BoolIntLtContext) interface{} {
	left, right := ctx.Iexpr(0), ctx.Iexpr(1)
	leftIsBigInt := e.isBigIntExpr(left)
	rightIsBigInt := e.isBigIntExpr(right)
	needsBigInt := leftIsBigInt || rightIsBigInt

	e.emitWithBigIntConversion(left, needsBigInt)
	e.emitWithBigIntConversion(right, needsBigInt)

	if needsBigInt {
		e.emit("b<")
	} else {
		e.emit("<")
	}
	return nil
}

func (e *PostfixEmitter) VisitBoolIntLte(ctx *BoolIntLteContext) interface{} {
	left, right := ctx.Iexpr(0), ctx.Iexpr(1)
	leftIsBigInt := e.isBigIntExpr(left)
	rightIsBigInt := e.isBigIntExpr(right)
	needsBigInt := leftIsBigInt || rightIsBigInt

	e.emitWithBigIntConversion(left, needsBigInt)
	e.emitWithBigIntConversion(right, needsBigInt)

	if needsBigInt {
		e.emit("b<=")
	} else {
		e.emit("<=")
	}
	return nil
}

func (e *PostfixEmitter) VisitBoolFloatEq(ctx *BoolFloatEqContext) interface{} {
	e.Visit(ctx.Fexpr(0))
	e.Visit(ctx.Fexpr(1))
	e.emit("f==")
	return nil
}

func (e *PostfixEmitter) VisitBoolFloatNeq(ctx *BoolFloatNeqContext) interface{} {
	e.Visit(ctx.Fexpr(0))
	e.Visit(ctx.Fexpr(1))
	e.emit("f==")
	e.emit("not")
	return nil
}

func (e *PostfixEmitter) VisitBoolFloatGt(ctx *BoolFloatGtContext) interface{} {
	e.Visit(ctx.Fexpr(0))
	e.Visit(ctx.Fexpr(1))
	e.emit("f>")
	return nil
}

func (e *PostfixEmitter) VisitBoolFloatLt(ctx *BoolFloatLtContext) interface{} {
	e.Visit(ctx.Fexpr(0))
	e.Visit(ctx.Fexpr(1))
	e.emit("f<")
	return nil
}

func (e *PostfixEmitter) VisitBoolFloatGte(ctx *BoolFloatGteContext) interface{} {
	e.Visit(ctx.Fexpr(0))
	e.Visit(ctx.Fexpr(1))
	e.emit("f>=")
	return nil
}

func (e *PostfixEmitter) VisitBoolFloatLte(ctx *BoolFloatLteContext) interface{} {
	e.Visit(ctx.Fexpr(0))
	e.Visit(ctx.Fexpr(1))
	e.emit("f<=")
	return nil
}

// arrayExprIsBytes returns true if an arrayExpr resolves to a bytes-typed identifier.
func (e *PostfixEmitter) arrayExprIsBytes(ctx IArrayExprContext) bool {
	base, ok := ctx.(*ArrayBaseContext)
	if !ok {
		return false
	}
	typed, ok2 := base.ArrayExpr2().(*ArrayTypedContext)
	if !ok2 {
		return false
	}
	return e.identIsBytes(typed.TypedArray().GetText())
}

// identIsBytes returns true if an identifier name is typed as bytes.
func (e *PostfixEmitter) identIsBytes(name string) bool {
	if lv, ok := e.lookupLocal(name); ok {
		return lv.Type == TypeBytes
	}
	return e.lookupType(name) == TypeBytes
}

// iexprIsBytes returns true if an iexpr resolves to a bytes-typed identifier.
func (e *PostfixEmitter) iexprIsBytes(ctx IIexprContext) bool {
	if ctx == nil {
		return false
	}
	switch c := ctx.(type) {
	case *IntTypedContext:
		return e.identIsBytes(c.TypedLong().GetText())
	case *IntParenContext:
		return e.iexprIsBytes(c.Iexpr())
	case *IntAddContext:
		// Bytes concat masquerading as int add
		return e.iexprIsBytes(c.Iexpr(0)) && e.iexprIsBytes(c.Iexpr(1))
	}
	return false
}

// strexprIsBytes returns true if a strexpr resolves to an identifier typed as bytes.
func (e *PostfixEmitter) strexprIsBytes(ctx IStrexprContext) bool {
	typed, ok := ctx.(*StrTypedContext)
	if !ok {
		return false
	}
	ident := typed.TypedString().GetText()
	if lv, ok := e.lookupLocal(ident); ok {
		return lv.Type == TypeBytes
	}
	return e.lookupType(ident) == TypeBytes
}

func (e *PostfixEmitter) VisitBoolStrEq(ctx *BoolStrEqContext) interface{} {
	left, right := ctx.Strexpr(0), ctx.Strexpr(1)
	if e.strexprIsBytes(left) && e.strexprIsBytes(right) {
		e.Visit(left)
		e.Visit(right)
		e.emit("bytes==")
		return nil
	}
	e.Visit(left)
	e.Visit(right)
	e.emit("streq")
	return nil
}

func (e *PostfixEmitter) VisitBoolStrNeq(ctx *BoolStrNeqContext) interface{} {
	left, right := ctx.Strexpr(0), ctx.Strexpr(1)
	if e.strexprIsBytes(left) && e.strexprIsBytes(right) {
		e.Visit(left)
		e.Visit(right)
		e.emit("bytes!=")
		return nil
	}
	e.Visit(left)
	e.Visit(right)
	e.emit("strneq")
	return nil
}

func (e *PostfixEmitter) VisitBoolStrIs(ctx *BoolStrIsContext) interface{} {
	left, right := ctx.Strexpr(0), ctx.Strexpr(1)
	if e.strexprIsBytes(left) && e.strexprIsBytes(right) {
		e.Visit(left)
		e.Visit(right)
		e.emit("bytes==")
		return nil
	}
	e.Visit(left)
	e.Visit(right)
	e.emit("streq")
	return nil
}

func (e *PostfixEmitter) VisitBoolStrIsNot(ctx *BoolStrIsNotContext) interface{} {
	left, right := ctx.Strexpr(0), ctx.Strexpr(1)
	if e.strexprIsBytes(left) && e.strexprIsBytes(right) {
		e.Visit(left)
		e.Visit(right)
		e.emit("bytes!=")
		return nil
	}
	e.Visit(left)
	e.Visit(right)
	e.emit("strneq")
	e.emit("not")
	return nil
}

func (e *PostfixEmitter) VisitBoolStrEqIc(ctx *BoolStrEqIcContext) interface{} {
	e.Visit(ctx.Strexpr(0))
	e.Visit(ctx.Strexpr(1))
	e.emit("sic==")
	return nil
}

func (e *PostfixEmitter) VisitBoolStrNeqIc(ctx *BoolStrNeqIcContext) interface{} {
	e.Visit(ctx.Strexpr(0))
	e.Visit(ctx.Strexpr(1))
	e.emit("sic==")
	e.emit("not")
	return nil
}

func (e *PostfixEmitter) VisitBoolAnd(ctx *BoolAndContext) interface{} {
	// Lazy evaluation: e1 { pop e2 } over if
	e.Visit(ctx.Bexpr(0))
	e.emit("{")
	e.emit("pop")
	e.Visit(ctx.Bexpr(1))
	e.emit("}")
	e.emit("over")
	e.emit("if")
	return nil
}

func (e *PostfixEmitter) VisitBoolOr(ctx *BoolOrContext) interface{} {
	// Lazy evaluation: e1 { pop e2 } over not if
	e.Visit(ctx.Bexpr(0))
	e.emit("{")
	e.emit("pop")
	e.Visit(ctx.Bexpr(1))
	e.emit("}")
	e.emit("over")
	e.emit("not")
	e.emit("if")
	return nil
}

func (e *PostfixEmitter) VisitBoolNot(ctx *BoolNotContext) interface{} {
	e.Visit(ctx.Bexpr())
	e.emit("not")
	return nil
}

func (e *PostfixEmitter) VisitBoolBoolEq(ctx *BoolBoolEqContext) interface{} {
	e.Visit(ctx.Bexpr(0))
	e.Visit(ctx.Bexpr(1))
	e.emit("beq")
	return nil
}

func (e *PostfixEmitter) VisitBoolBoolNeq(ctx *BoolBoolNeqContext) interface{} {
	e.Visit(ctx.Bexpr(0))
	e.Visit(ctx.Bexpr(1))
	e.emit("beq")
	e.emit("not")
	return nil
}

func (e *PostfixEmitter) VisitBoolBexprIsNull(ctx *BoolBexprIsNullContext) interface{} {
	e.Visit(ctx.Bexpr())
	e.emit("isnull")
	return nil
}

func (e *PostfixEmitter) VisitBoolBexprIsNotNull(ctx *BoolBexprIsNotNullContext) interface{} {
	e.Visit(ctx.Bexpr())
	e.emit("isnull")
	e.emit("not")
	return nil
}

func (e *PostfixEmitter) VisitBoolTypedIsLiteral(ctx *BoolTypedIsLiteralContext) interface{} {
	// typedBoolean IS RBOOLEAN -> flag.a is true => flag.a true beq
	e.Visit(ctx.TypedBoolean())
	text := strings.ToLower(ctx.RBOOLEAN().GetText())
	e.emit(text)
	e.emit("beq")
	return nil
}

func (e *PostfixEmitter) VisitBoolTypedIsNotLiteral(ctx *BoolTypedIsNotLiteralContext) interface{} {
	// typedBoolean IS NOT RBOOLEAN -> flag.a is not true => flag.a true beq not
	e.Visit(ctx.TypedBoolean())
	text := strings.ToLower(ctx.RBOOLEAN().GetText())
	e.emit(text)
	e.emit("beq")
	e.emit("not")
	return nil
}

func (e *PostfixEmitter) VisitBoolColonIsLiteral(ctx *BoolColonIsLiteralContext) interface{} {
	// colonRef typedBoolean IS RBOOLEAN
	e.Visit(ctx.ColonRef())
	e.Visit(ctx.TypedBoolean())
	text := strings.ToLower(ctx.RBOOLEAN().GetText())
	e.emit(text)
	e.emit("beq")
	return nil
}

func (e *PostfixEmitter) VisitBoolColonIsNotLiteral(ctx *BoolColonIsNotLiteralContext) interface{} {
	// colonRef typedBoolean IS NOT RBOOLEAN
	e.Visit(ctx.ColonRef())
	e.Visit(ctx.TypedBoolean())
	text := strings.ToLower(ctx.RBOOLEAN().GetText())
	e.emit(text)
	e.emit("beq")
	e.emit("not")
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
	e.emit("d==")
	return nil
}

func (e *PostfixEmitter) VisitBoolDateLt(ctx *BoolDateLtContext) interface{} {
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(1))
	e.emit("d<")
	return nil
}

func (e *PostfixEmitter) VisitBoolDateGt(ctx *BoolDateGtContext) interface{} {
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(1))
	e.emit("d>")
	return nil
}

func (e *PostfixEmitter) VisitBoolDateBefore(ctx *BoolDateBeforeContext) interface{} {
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(1))
	e.emit("d<")
	return nil
}

func (e *PostfixEmitter) VisitBoolDateAfter(ctx *BoolDateAfterContext) interface{} {
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(1))
	e.emit("d>")
	return nil
}

func (e *PostfixEmitter) VisitBoolDateGte(ctx *BoolDateGteContext) interface{} {
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(1))
	e.emit("d<")
	e.emit("not")
	return nil
}

func (e *PostfixEmitter) VisitBoolDateLte(ctx *BoolDateLteContext) interface{} {
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(1))
	e.emit("d>")
	e.emit("not")
	return nil
}

func (e *PostfixEmitter) VisitBoolEntityEq(ctx *BoolEntityEqContext) interface{} {
	e.Visit(ctx.Eexpr(0))
	e.Visit(ctx.Eexpr(1))
	e.emit("req")
	return nil
}

func (e *PostfixEmitter) VisitBoolEntityNeq(ctx *BoolEntityNeqContext) interface{} {
	e.Visit(ctx.Eexpr(0))
	e.Visit(ctx.Eexpr(1))
	e.emit("req")
	e.emit("not")
	return nil
}

func (e *PostfixEmitter) VisitBoolNameEq(ctx *BoolNameEqContext) interface{} {
	// Check if either operand is an entity (local variable or from EDD)
	// If so, use req (reference equals) instead of streq (string equals)
	name0 := ctx.Nexpr(0).GetText()
	name1 := ctx.Nexpr(1).GetText()

	// bytes type: check before entity so bytes==bytes uses constant-time comparison
	isBytes0 := e.identIsBytes(name0)
	isBytes1 := e.identIsBytes(name1)
	if isBytes0 && isBytes1 {
		e.Visit(ctx.Nexpr(0))
		e.Visit(ctx.Nexpr(1))
		e.emit("bytes==")
		return nil
	}

	isEntity := false
	if lv, ok := e.lookupLocal(name0); ok && lv.Type == TypeEntity {
		isEntity = true
	} else if lv, ok := e.lookupLocal(name1); ok && lv.Type == TypeEntity {
		isEntity = true
	} else if t := e.lookupType(name0); t == TypeEntity {
		isEntity = true
	} else if t := e.lookupType(name1); t == TypeEntity {
		isEntity = true
	}

	e.Visit(ctx.Nexpr(0))
	e.Visit(ctx.Nexpr(1))
	if isEntity {
		e.emit("req") // Reference equals for entities
	} else {
		e.emit("streq") // String equals for names
	}
	return nil
}

func (e *PostfixEmitter) VisitBoolNameNeq(ctx *BoolNameNeqContext) interface{} {
	// Check if either operand is an entity
	name0 := ctx.Nexpr(0).GetText()
	name1 := ctx.Nexpr(1).GetText()

	// bytes: constant-time inequality
	if e.identIsBytes(name0) && e.identIsBytes(name1) {
		e.Visit(ctx.Nexpr(0))
		e.Visit(ctx.Nexpr(1))
		e.emit("bytes!=")
		return nil
	}

	isEntity := false
	if lv, ok := e.lookupLocal(name0); ok && lv.Type == TypeEntity {
		isEntity = true
	} else if lv, ok := e.lookupLocal(name1); ok && lv.Type == TypeEntity {
		isEntity = true
	} else if t := e.lookupType(name0); t == TypeEntity {
		isEntity = true
	} else if t := e.lookupType(name1); t == TypeEntity {
		isEntity = true
	}

	e.Visit(ctx.Nexpr(0))
	e.Visit(ctx.Nexpr(1))
	if isEntity {
		e.emit("req") // Reference equals for entities
	} else {
		e.emit("streq") // String equals for names
	}
	e.emit("not")
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
	e.emit("memberof")
	return nil
}

func (e *PostfixEmitter) VisitBoolArrayDoesInclude(ctx *BoolArrayDoesIncludeContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.Visit(ctx.IncludeSearch())
	e.emit("memberof")
	return nil
}

func (e *PostfixEmitter) VisitBoolArrayNotInclude(ctx *BoolArrayNotIncludeContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.Visit(ctx.IncludeSearch())
	e.emit("memberof")
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
	left, right := ctx.Iexpr(0), ctx.Iexpr(1)

	// Bytes concat: when both sides resolve to bytes, emit bytes+
	if e.iexprIsBytes(left) && e.iexprIsBytes(right) {
		e.Visit(left)
		e.Visit(right)
		e.emit("bytes+")
		return nil
	}

	leftIsBigInt := e.isBigIntExpr(left)
	rightIsBigInt := e.isBigIntExpr(right)
	needsBigInt := leftIsBigInt || rightIsBigInt

	e.emitWithBigIntConversion(left, needsBigInt)
	e.emitWithBigIntConversion(right, needsBigInt)

	if needsBigInt {
		e.emit("b+")
	} else {
		e.emit("+")
	}
	return nil
}

func (e *PostfixEmitter) VisitIntSub(ctx *IntSubContext) interface{} {
	left, right := ctx.Iexpr(0), ctx.Iexpr(1)
	leftIsBigInt := e.isBigIntExpr(left)
	rightIsBigInt := e.isBigIntExpr(right)
	needsBigInt := leftIsBigInt || rightIsBigInt

	e.emitWithBigIntConversion(left, needsBigInt)
	e.emitWithBigIntConversion(right, needsBigInt)

	if needsBigInt {
		e.emit("b-")
	} else {
		e.emit("-")
	}
	return nil
}

func (e *PostfixEmitter) VisitIntMul(ctx *IntMulContext) interface{} {
	left, right := ctx.Iexpr(0), ctx.Iexpr(1)
	leftIsBigInt := e.isBigIntExpr(left)
	rightIsBigInt := e.isBigIntExpr(right)
	needsBigInt := leftIsBigInt || rightIsBigInt

	e.emitWithBigIntConversion(left, needsBigInt)
	e.emitWithBigIntConversion(right, needsBigInt)

	if needsBigInt {
		e.emit("b*")
	} else {
		e.emit("*")
	}
	return nil
}

func (e *PostfixEmitter) VisitIntDiv(ctx *IntDivContext) interface{} {
	left, right := ctx.Iexpr(0), ctx.Iexpr(1)
	leftIsBigInt := e.isBigIntExpr(left)
	rightIsBigInt := e.isBigIntExpr(right)
	needsBigInt := leftIsBigInt || rightIsBigInt

	e.emitWithBigIntConversion(left, needsBigInt)
	e.emitWithBigIntConversion(right, needsBigInt)

	if needsBigInt {
		e.emit("b/")
	} else {
		e.emit("/")
	}
	return nil
}

func (e *PostfixEmitter) VisitIntNegate(ctx *IntNegateContext) interface{} {
	expr := ctx.Iexpr()
	isBigInt := e.isBigIntExpr(expr)
	e.Visit(expr)
	if isBigInt {
		e.emit("bnegate")
	} else {
		e.emit("neg")
	}
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
	// If the arrayExpr is actually a bytes-typed identifier, emit byteslen.
	if e.arrayExprIsBytes(ctx.ArrayExpr()) {
		e.Visit(ctx.ArrayExpr())
		e.emit("byteslen")
		return nil
	}
	e.Visit(ctx.ArrayExpr())
	e.emit("length")
	return nil
}

func (e *PostfixEmitter) VisitIntLengthStr(ctx *IntLengthStrContext) interface{} {
	// If the strexpr resolves to a bytes-typed identifier, emit byteslen instead.
	if e.strexprIsBytes(ctx.Strexpr()) {
		e.Visit(ctx.Strexpr())
		e.emit("byteslen")
		return nil
	}
	e.Visit(ctx.Strexpr())
	e.emit("length")
	return nil
}

func (e *PostfixEmitter) VisitIntLengthBytes(ctx *IntLengthBytesContext) interface{} {
	e.Visit(ctx.Bytesexpr())
	e.emit("byteslen")
	return nil
}

func (e *PostfixEmitter) VisitIntBytesIndex(ctx *IntBytesIndexContext) interface{} {
	e.Visit(ctx.Bytesexpr())
	e.Visit(ctx.Iexpr())
	e.emit("bytesidx")
	return nil
}

func (e *PostfixEmitter) VisitIntMinOf(ctx *IntMinOfContext) interface{} {
	e.Visit(ctx.Iexpr(0))
	e.Visit(ctx.Iexpr(1))
	e.emit("min")
	return nil
}

func (e *PostfixEmitter) VisitIntMinOfComma(ctx *IntMinOfCommaContext) interface{} {
	e.Visit(ctx.Iexpr(0))
	e.Visit(ctx.Iexpr(1))
	e.emit("min")
	return nil
}

func (e *PostfixEmitter) VisitIntMaxOf(ctx *IntMaxOfContext) interface{} {
	e.Visit(ctx.Iexpr(0))
	e.Visit(ctx.Iexpr(1))
	e.emit("max")
	return nil
}

func (e *PostfixEmitter) VisitIntMaxOfComma(ctx *IntMaxOfCommaContext) interface{} {
	e.Visit(ctx.Iexpr(0))
	e.Visit(ctx.Iexpr(1))
	e.emit("max")
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
	e.emit("fmul")
	return nil
}

func (e *PostfixEmitter) VisitFloatDivFloat(ctx *FloatDivFloatContext) interface{} {
	e.Visit(ctx.Fexpr(0))
	e.Visit(ctx.Fexpr(1))
	e.emit("fdiv")
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
	e.emit("fmul")
	return nil
}

func (e *PostfixEmitter) VisitFloatDivInt(ctx *FloatDivIntContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.Visit(ctx.Iexpr())
	e.emit("fdiv")
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
	e.emit("fmul")
	return nil
}

func (e *PostfixEmitter) VisitIntDivFloat(ctx *IntDivFloatContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.Visit(ctx.Fexpr())
	e.emit("fdiv")
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

func (e *PostfixEmitter) VisitFloatMinOfFloat(ctx *FloatMinOfFloatContext) interface{} {
	e.Visit(ctx.Fexpr(0))
	e.Visit(ctx.Fexpr(1))
	e.emit("fmin")
	return nil
}

func (e *PostfixEmitter) VisitFloatMinOfInt(ctx *FloatMinOfIntContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.Visit(ctx.Iexpr())
	e.emit("fmin")
	return nil
}

func (e *PostfixEmitter) VisitFloatMinIntOf(ctx *FloatMinIntOfContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.Visit(ctx.Fexpr())
	e.emit("fmin")
	return nil
}

func (e *PostfixEmitter) VisitFloatMinOfFloatComma(ctx *FloatMinOfFloatCommaContext) interface{} {
	e.Visit(ctx.Fexpr(0))
	e.Visit(ctx.Fexpr(1))
	e.emit("fmin")
	return nil
}

func (e *PostfixEmitter) VisitFloatMinOfIntComma(ctx *FloatMinOfIntCommaContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.Visit(ctx.Iexpr())
	e.emit("fmin")
	return nil
}

func (e *PostfixEmitter) VisitFloatMinIntOfComma(ctx *FloatMinIntOfCommaContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.Visit(ctx.Fexpr())
	e.emit("fmin")
	return nil
}

func (e *PostfixEmitter) VisitFloatMaxOfFloat(ctx *FloatMaxOfFloatContext) interface{} {
	e.Visit(ctx.Fexpr(0))
	e.Visit(ctx.Fexpr(1))
	e.emit("fmax")
	return nil
}

func (e *PostfixEmitter) VisitFloatMaxOfInt(ctx *FloatMaxOfIntContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.Visit(ctx.Iexpr())
	e.emit("fmax")
	return nil
}

func (e *PostfixEmitter) VisitFloatMaxIntOf(ctx *FloatMaxIntOfContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.Visit(ctx.Fexpr())
	e.emit("fmax")
	return nil
}

func (e *PostfixEmitter) VisitFloatMaxOfFloatComma(ctx *FloatMaxOfFloatCommaContext) interface{} {
	e.Visit(ctx.Fexpr(0))
	e.Visit(ctx.Fexpr(1))
	e.emit("fmax")
	return nil
}

func (e *PostfixEmitter) VisitFloatMaxOfIntComma(ctx *FloatMaxOfIntCommaContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.Visit(ctx.Iexpr())
	e.emit("fmax")
	return nil
}

func (e *PostfixEmitter) VisitFloatMaxIntOfComma(ctx *FloatMaxIntOfCommaContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.Visit(ctx.Fexpr())
	e.emit("fmax")
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

func (e *PostfixEmitter) VisitStrXmlValue(ctx *StrXmlValueContext) interface{} {
	e.Visit(ctx.TypedXmlValue())
	return nil
}

func (e *PostfixEmitter) VisitStrTyped(ctx *StrTypedContext) interface{} {
	e.Visit(ctx.TypedString())
	return nil
}

func (e *PostfixEmitter) VisitTypedXmlValue(ctx *TypedXmlValueContext) interface{} {
	e.emit(ctx.GetText())
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
	// Java pattern: /EntityName createentity
	entityName := ctx.TypedEntity().GetText()
	e.emit("/" + entityName)
	e.emit("createentity")
	return nil
}

func (e *PostfixEmitter) VisitEntityNewName(ctx *EntityNewNameContext) interface{} {
	// Java pattern: /EntityName createentity
	// The name expression contains the entity type name
	e.emit("/" + ctx.Nexpr().GetText())
	e.emit("createentity")
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
	name := ctx.GetText()
	// Check if this is a local variable - emit stack frame access
	if !e.emitLocalRef(name) {
		e.emit(name)
	}
	return nil
}

func (e *PostfixEmitter) VisitTypedLong(ctx *TypedLongContext) interface{} {
	name := ctx.GetText()
	// Check if this is a local variable - emit stack frame access
	if !e.emitLocalRef(name) {
		e.emit(name)
	}
	return nil
}

func (e *PostfixEmitter) VisitTypedDouble(ctx *TypedDoubleContext) interface{} {
	name := ctx.GetText()
	// Check if this is a local variable - emit stack frame access
	if !e.emitLocalRef(name) {
		e.emit(name)
	}
	return nil
}

func (e *PostfixEmitter) VisitTypedString(ctx *TypedStringContext) interface{} {
	name := ctx.GetText()
	// Check if this is a local variable - emit stack frame access
	if !e.emitLocalRef(name) {
		e.emit(name)
	}
	return nil
}

func (e *PostfixEmitter) VisitTypedBoolean(ctx *TypedBooleanContext) interface{} {
	name := ctx.GetText()
	// Check if this is a local variable - emit stack frame access
	if !e.emitLocalRef(name) {
		e.emit(name)
	}
	return nil
}

func (e *PostfixEmitter) VisitTypedDate(ctx *TypedDateContext) interface{} {
	name := ctx.GetText()
	// Check if this is a local variable - emit stack frame access
	if !e.emitLocalRef(name) {
		e.emit(name)
	}
	return nil
}

func (e *PostfixEmitter) VisitTypedArray(ctx *TypedArrayContext) interface{} {
	name := ctx.GetText()
	// Check if this is a local variable - emit stack frame access
	if !e.emitLocalRef(name) {
		e.emit(name)
	}
	return nil
}

func (e *PostfixEmitter) VisitTypedTable(ctx *TypedTableContext) interface{} {
	name := ctx.GetText()
	// Check if this is a local variable - emit stack frame access
	if !e.emitLocalRef(name) {
		e.emit(name)
	}
	return nil
}

func (e *PostfixEmitter) VisitTypedName(ctx *TypedNameContext) interface{} {
	name := ctx.GetText()
	// Check if this is a local variable - emit stack frame access
	if !e.emitLocalRef(name) {
		e.emit(name)
	}
	return nil
}

func (e *PostfixEmitter) VisitTypedDecisionTable(ctx *TypedDecisionTableContext) interface{} {
	e.emit(ctx.GetText())
	return nil
}

// ============================================================================
// Context Visitors
// ============================================================================

// VisitContextStatement is called for `context <contextForTable>` statements.
// We override this to use our own visitor dispatch (visitChildren uses e.Visit)
// instead of the base visitor which would pass the embedded BaseELVisitor to Accept.
func (e *PostfixEmitter) VisitContextStatement(ctx *ContextStatementContext) interface{} {
	// Visit the contextForTable child - this will dispatch to the appropriate
	// visitor method like VisitContextLocal based on which alternative matched
	return e.Visit(ctx.ContextForTable())
}

func (e *PostfixEmitter) VisitContextLocal(ctx *ContextLocalContext) interface{} {
	// Visit the localvariables child to register local variables
	return e.Visit(ctx.Localvariables())
}

func (e *PostfixEmitter) VisitContextDebug(ctx *ContextDebugContext) interface{} {
	return e.Visit(ctx.Debugstatement())
}

func (e *PostfixEmitter) VisitContextFor(ctx *ContextForContext) interface{} {
	return e.Visit(ctx.Forctl())
}

func (e *PostfixEmitter) VisitContextForallCtl(ctx *ContextForallCtlContext) interface{} {
	return e.Visit(ctx.Forallctl())
}

// Forall context emitters.
//
// compileContextsPostfix wraps each context around a table body block so that
// entering our emit the data stack holds [body], where body is the compiled
// { /TableName executetable }. forall has signature (body array --), so each
// emit must leave the stack clean after iterating.
//
// forallSimple: dup body so forall can consume it, iterate, drop the dup.
// forallWhere:  replace body with a filter { <bexpr> { dup execute } if }.
//               Entering forall the stack is [body, filter, array]; forall
//               consumes (filter, array) leaving [body]. During each iteration
//               the element is on the entity stack, so <bexpr> resolves
//               element attributes. When <bexpr> is true the filter runs
//               `dup execute`, which re-pushes and invokes body without
//               consuming the copy that survives to the next iteration.
//               Final pop drops the surviving body.
// forallInEntity: evaluate eexpr, entitypush to make its attributes reachable
//               while arrayExpr and the body run, iterate via the simple
//               form, then entitypop + pop to clean up.
// *AllowRemove: authoring-time affirmation that the body may mutate the array;
//               emits identically to its non-allowing counterpart.

// VisitForallSimple: `for all <array>`.
func (e *PostfixEmitter) VisitForallSimple(ctx *ForallSimpleContext) interface{} {
	e.emit("dup")
	e.Visit(ctx.ArrayExpr())
	e.emit("forall")
	e.emit("pop")
	return nil
}

// VisitForallAllowRemove: `for all <array> allowing array to be removed`.
func (e *PostfixEmitter) VisitForallAllowRemove(ctx *ForallAllowRemoveContext) interface{} {
	e.emit("dup")
	e.Visit(ctx.ArrayExpr())
	e.emit("forall")
	e.emit("pop")
	return nil
}

// VisitForallWhere: `for all <array> where <bexpr>`.
// `if` signature is (body test --), so push body block first then test.
func (e *PostfixEmitter) VisitForallWhere(ctx *ForallWhereContext) interface{} {
	e.emit("{")
	e.emit("{")
	e.emit("dup")
	e.emit("execute")
	e.emit("}")
	e.Visit(ctx.Bexpr())
	e.emit("if")
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forall")
	e.emit("pop")
	return nil
}

// VisitForallWhereAllowRemove: `for all <array> where <bexpr> allowing array to be removed`.
func (e *PostfixEmitter) VisitForallWhereAllowRemove(ctx *ForallWhereAllowRemoveContext) interface{} {
	e.emit("{")
	e.emit("{")
	e.emit("dup")
	e.emit("execute")
	e.emit("}")
	e.Visit(ctx.Bexpr())
	e.emit("if")
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forall")
	e.emit("pop")
	return nil
}

// VisitForallInEntity: `for all <array> in <entity>`.
func (e *PostfixEmitter) VisitForallInEntity(ctx *ForallInEntityContext) interface{} {
	e.Visit(ctx.Eexpr())
	e.emit("entitypush")
	e.emit("dup")
	e.Visit(ctx.ArrayExpr())
	e.emit("forall")
	e.emit("pop")
	e.emit("entitypop")
	e.emit("pop")
	return nil
}

// VisitForallInEntityAllowRemove: `for all <array> in <entity> allowing array to be removed`.
func (e *PostfixEmitter) VisitForallInEntityAllowRemove(ctx *ForallInEntityAllowRemoveContext) interface{} {
	e.Visit(ctx.Eexpr())
	e.emit("entitypush")
	e.emit("dup")
	e.Visit(ctx.ArrayExpr())
	e.emit("forall")
	e.emit("pop")
	e.emit("entitypop")
	e.emit("pop")
	return nil
}

// VisitForallInEntityWhere: `for all <array> in <entity> where <bexpr>`.
func (e *PostfixEmitter) VisitForallInEntityWhere(ctx *ForallInEntityWhereContext) interface{} {
	e.Visit(ctx.Eexpr())
	e.emit("entitypush")
	e.emit("{")
	e.emit("{")
	e.emit("dup")
	e.emit("execute")
	e.emit("}")
	e.Visit(ctx.Bexpr())
	e.emit("if")
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forall")
	e.emit("pop")
	e.emit("entitypop")
	e.emit("pop")
	return nil
}

func (e *PostfixEmitter) VisitContextForfirst(ctx *ContextForfirstContext) interface{} {
	return e.Visit(ctx.Forfirstctl())
}

// Forfirstctl context emitters. Same wrapping rules as forallctl: the
// compileContextsPostfix wrap leaves the table body block on the data stack.
// opForfirst signature is (body test array --); pop order is array, test,
// body. Emit pushes body-duplicate, test block, array — forfirst consumes
// three, leaving the original body to drop with a final pop.

// VisitForfirstOf: `for first of <array> where <bexpr>`.
func (e *PostfixEmitter) VisitForfirstOf(ctx *ForfirstOfContext) interface{} {
	e.emit("dup")
	e.emit("{")
	e.Visit(ctx.Bexpr())
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forfirst")
	e.emit("pop")
	return nil
}

// VisitForfirstIn: `for first in <array> where <bexpr>`. Same semantics as
// forfirstOf.
func (e *PostfixEmitter) VisitForfirstIn(ctx *ForfirstInContext) interface{} {
	e.emit("dup")
	e.emit("{")
	e.Visit(ctx.Bexpr())
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forfirst")
	e.emit("pop")
	return nil
}

// VisitForfirstOfIts: `for first of <array> and its <eexpr> where <bexpr>`.
// The related entity eexpr must be on the entity stack while both the test
// and the body run. We wrap both with entitypush/entitypop so the test still
// leaves a single bool on the data stack and the body remains stack-neutral
// via the `dup execute` trick that references the outer table body still
// sitting on the data stack below forfirst's operands.
func (e *PostfixEmitter) VisitForfirstOfIts(ctx *ForfirstOfItsContext) interface{} {
	// body-wrapper: push eexpr entity, execute outer body (via dup/execute),
	// then pop the entity. Net data-stack effect: neutral.
	e.emit("{")
	e.Visit(ctx.Eexpr())
	e.emit("entitypush")
	e.emit("dup")
	e.emit("execute")
	e.emit("entitypop")
	e.emit("pop")
	e.emit("}")
	// test-wrapper: push eexpr entity, evaluate bexpr, pop entity, leaving
	// just the bool on the data stack.
	e.emit("{")
	e.Visit(ctx.Eexpr())
	e.emit("entitypush")
	e.Visit(ctx.Bexpr())
	e.emit("entitypop")
	e.emit("pop")
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forfirst")
	e.emit("pop")
	return nil
}

func (e *PostfixEmitter) VisitContextCtx(ctx *ContextCtxContext) interface{} {
	return e.Visit(ctx.Contextstatement())
}

// ============================================================================
// Local Variable Declaration Visitors
// ============================================================================

func (e *PostfixEmitter) VisitLocalEntityUndef(ctx *LocalEntityUndefContext) interface{} {
	name := ctx.UndefinedIdent().GetText()
	e.declareLocal(name, TypeEntity)
	e.emit("null")
	e.emit("allocate")
	e.emit("execute")
	e.emit("deallocate")
	e.emit("pop")
	return nil
}

func (e *PostfixEmitter) VisitLocalEntityInit(ctx *LocalEntityInitContext) interface{} {
	name := ctx.UndefinedIdent().GetText()
	e.declareLocal(name, TypeEntity)
	e.Visit(ctx.Eexpr())
	e.emit("cve")
	e.emit("allocate")
	e.emit("execute")
	e.emit("deallocate")
	e.emit("pop")
	return nil
}

func (e *PostfixEmitter) VisitLocalEntityDefined(ctx *LocalEntityDefinedContext) interface{} {
	// Already defined entity - this is an error in Java, we'll just emit the name
	e.emit(ctx.TypedEntity().GetText())
	return nil
}

func (e *PostfixEmitter) VisitLocalLongUndef(ctx *LocalLongUndefContext) interface{} {
	name := ctx.UndefinedIdent().GetText()
	e.declareLocal(name, TypeInteger)
	e.emit("null")
	e.emit("allocate")
	e.emit("execute")
	e.emit("deallocate")
	e.emit("pop")
	return nil
}

func (e *PostfixEmitter) VisitLocalLongInit(ctx *LocalLongInitContext) interface{} {
	name := ctx.UndefinedIdent().GetText()
	e.declareLocal(name, TypeInteger)
	e.Visit(ctx.Number())
	e.emit("cvi")
	e.emit("allocate")
	e.emit("execute")
	e.emit("deallocate")
	e.emit("pop")
	return nil
}

func (e *PostfixEmitter) VisitLocalLongDefined(ctx *LocalLongDefinedContext) interface{} {
	e.emit(ctx.TypedLong().GetText())
	return nil
}

func (e *PostfixEmitter) VisitLocalDoubleUndef(ctx *LocalDoubleUndefContext) interface{} {
	name := ctx.UndefinedIdent().GetText()
	e.declareLocal(name, TypeDouble)
	e.emit("null")
	e.emit("allocate")
	e.emit("execute")
	e.emit("deallocate")
	e.emit("pop")
	return nil
}

func (e *PostfixEmitter) VisitLocalDoubleInit(ctx *LocalDoubleInitContext) interface{} {
	name := ctx.UndefinedIdent().GetText()
	e.declareLocal(name, TypeDouble)
	e.Visit(ctx.Number())
	e.emit("cvr")
	e.emit("allocate")
	e.emit("execute")
	e.emit("deallocate")
	e.emit("pop")
	return nil
}

func (e *PostfixEmitter) VisitLocalDoubleDefined(ctx *LocalDoubleDefinedContext) interface{} {
	e.emit(ctx.TypedDouble().GetText())
	return nil
}

func (e *PostfixEmitter) VisitLocalBoolUndef(ctx *LocalBoolUndefContext) interface{} {
	name := ctx.UndefinedIdent().GetText()
	e.declareLocal(name, TypeBoolean)
	e.emit("null")
	e.emit("allocate")
	e.emit("execute")
	e.emit("deallocate")
	e.emit("pop")
	return nil
}

func (e *PostfixEmitter) VisitLocalBoolInit(ctx *LocalBoolInitContext) interface{} {
	name := ctx.UndefinedIdent().GetText()
	e.declareLocal(name, TypeBoolean)
	e.Visit(ctx.Bexpr())
	e.emit("cvb")
	e.emit("allocate")
	e.emit("execute")
	e.emit("deallocate")
	e.emit("pop")
	return nil
}

func (e *PostfixEmitter) VisitLocalBoolDefined(ctx *LocalBoolDefinedContext) interface{} {
	e.emit(ctx.TypedBoolean().GetText())
	return nil
}

func (e *PostfixEmitter) VisitLocalDateUndef(ctx *LocalDateUndefContext) interface{} {
	name := ctx.UndefinedIdent().GetText()
	e.declareLocal(name, TypeDate)
	e.emit("null")
	e.emit("allocate")
	e.emit("execute")
	e.emit("deallocate")
	e.emit("pop")
	return nil
}

func (e *PostfixEmitter) VisitLocalDateInit(ctx *LocalDateInitContext) interface{} {
	name := ctx.UndefinedIdent().GetText()
	e.declareLocal(name, TypeDate)
	e.Visit(ctx.Dexpr())
	e.emit("cvd")
	e.emit("allocate")
	e.emit("execute")
	e.emit("deallocate")
	e.emit("pop")
	return nil
}

func (e *PostfixEmitter) VisitLocalDateDefined(ctx *LocalDateDefinedContext) interface{} {
	e.emit(ctx.TypedDate().GetText())
	return nil
}

func (e *PostfixEmitter) VisitLocalArrayUndef(ctx *LocalArrayUndefContext) interface{} {
	name := ctx.UndefinedIdent().GetText()
	e.declareLocal(name, TypeArray)
	e.emit("null")
	e.emit("allocate")
	e.emit("execute")
	e.emit("deallocate")
	e.emit("pop")
	return nil
}

func (e *PostfixEmitter) VisitLocalArrayInit(ctx *LocalArrayInitContext) interface{} {
	name := ctx.UndefinedIdent().GetText()
	e.declareLocal(name, TypeArray)
	e.Visit(ctx.ArrayExpr())
	e.emit("allocate")
	e.emit("execute")
	e.emit("deallocate")
	e.emit("pop")
	return nil
}

func (e *PostfixEmitter) VisitLocalArrayDefined(ctx *LocalArrayDefinedContext) interface{} {
	e.emit(ctx.TypedArray().GetText())
	return nil
}

func (e *PostfixEmitter) VisitLocalStringUndef(ctx *LocalStringUndefContext) interface{} {
	name := ctx.UndefinedIdent().GetText()
	e.declareLocal(name, TypeString)
	e.emit("null")
	e.emit("allocate")
	e.emit("execute")
	e.emit("deallocate")
	e.emit("pop")
	return nil
}

func (e *PostfixEmitter) VisitLocalStringInit(ctx *LocalStringInitContext) interface{} {
	name := ctx.UndefinedIdent().GetText()
	e.declareLocal(name, TypeString)
	e.Visit(ctx.Strexpr())
	e.emit("cvs")
	e.emit("allocate")
	e.emit("execute")
	e.emit("deallocate")
	e.emit("pop")
	return nil
}

func (e *PostfixEmitter) VisitLocalStringDefined(ctx *LocalStringDefinedContext) interface{} {
	e.emit(ctx.TypedString().GetText())
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
	// Just emit the table name - no executetable needed
	e.Visit(ctx.TypedDecisionTable())
	return nil
}

func (e *PostfixEmitter) VisitPerformDTExplicit(ctx *PerformDTExplicitContext) interface{} {
	// Just emit the table name - no executetable needed
	e.Visit(ctx.TypedDecisionTable())
	return nil
}

func (e *PostfixEmitter) VisitErrorStmt(ctx *ErrorStmtContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("elstmterror")
	return nil
}

func (e *PostfixEmitter) VisitWarnStmt(ctx *WarnStmtContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("elstmtwarn")
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
	// Look up the target field type, default to integer from grammar context
	leftField := ctx.LeftIexpr().GetText()
	fieldType := e.lookupType(leftField)
	if fieldType == "" {
		fieldType = TypeInteger
	}
	if conv := e.typeConverter(fieldType); conv != "" {
		e.emit(conv)
	}
	e.Visit(ctx.LeftIexpr())
	return nil
}

func (e *PostfixEmitter) VisitSetFloat(ctx *SetFloatContext) interface{} {
	e.Visit(ctx.Number())
	// Look up the target field type, default to double from grammar context
	leftField := ctx.LeftFexpr().GetText()
	fieldType := e.lookupType(leftField)
	if fieldType == "" {
		fieldType = TypeDouble
	}
	if conv := e.typeConverter(fieldType); conv != "" {
		e.emit(conv)
	}
	e.Visit(ctx.LeftFexpr())
	return nil
}

func (e *PostfixEmitter) VisitSetBool(ctx *SetBoolContext) interface{} {
	e.Visit(ctx.Bexpr())
	// Look up the target field type, default to boolean from grammar context
	leftField := ctx.LeftBexpr().GetText()
	fieldType := e.lookupType(leftField)
	if fieldType == "" {
		fieldType = TypeBoolean
	}
	if conv := e.typeConverter(fieldType); conv != "" {
		e.emit(conv)
	}
	e.Visit(ctx.LeftBexpr())
	return nil
}

func (e *PostfixEmitter) VisitSetEntity(ctx *SetEntityContext) interface{} {
	e.Visit(ctx.Eexpr())
	// Look up the target field type, default to entity from grammar context
	leftField := ctx.LeftEexpr().GetText()
	fieldType := e.lookupType(leftField)
	if fieldType == "" {
		fieldType = TypeEntity
	}
	if conv := e.typeConverter(fieldType); conv != "" {
		e.emit(conv)
	}
	e.Visit(ctx.LeftEexpr())
	return nil
}

func (e *PostfixEmitter) VisitSetString(ctx *SetStringContext) interface{} {
	e.Visit(ctx.Strexpr())
	// Look up the target field type, default to string from grammar context
	leftField := ctx.LeftStrexpr().GetText()
	fieldType := e.lookupType(leftField)
	if fieldType == "" {
		fieldType = TypeString
	}
	if conv := e.typeConverter(fieldType); conv != "" {
		e.emit(conv)
	}
	e.Visit(ctx.LeftStrexpr())
	return nil
}

func (e *PostfixEmitter) VisitSetDate(ctx *SetDateContext) interface{} {
	e.Visit(ctx.Dexpr())
	// Look up the target field type, default to date from grammar context
	leftField := ctx.LeftDexpr().GetText()
	fieldType := e.lookupType(leftField)
	if fieldType == "" {
		fieldType = TypeDate
	}
	if conv := e.typeConverter(fieldType); conv != "" {
		e.emit(conv)
	}
	e.Visit(ctx.LeftDexpr())
	return nil
}

func (e *PostfixEmitter) VisitLeftIexprSimple(ctx *LeftIexprSimpleContext) interface{} {
	// Emit left value format: /<fieldname> xdef
	e.emit("/" + ctx.TypedLong().GetText())
	e.emit("xdef")
	return nil
}

func (e *PostfixEmitter) VisitLeftFexprSimple(ctx *LeftFexprSimpleContext) interface{} {
	// Emit left value format: /<fieldname> xdef
	e.emit("/" + ctx.TypedDouble().GetText())
	e.emit("xdef")
	return nil
}

func (e *PostfixEmitter) VisitLeftBexprSimple(ctx *LeftBexprSimpleContext) interface{} {
	// Emit left value format: /<fieldname> xdef
	e.emit("/" + ctx.TypedBoolean().GetText())
	e.emit("xdef")
	return nil
}

func (e *PostfixEmitter) VisitLeftEexprSimple(ctx *LeftEexprSimpleContext) interface{} {
	// Emit left value format: /<fieldname> xdef
	e.emit("/" + ctx.TypedEntity().GetText())
	e.emit("xdef")
	return nil
}

func (e *PostfixEmitter) VisitLeftStrexprSimple(ctx *LeftStrexprSimpleContext) interface{} {
	// Emit left value format: /<fieldname> xdef
	e.emit("/" + ctx.TypedString().GetText())
	e.emit("xdef")
	return nil
}

func (e *PostfixEmitter) VisitLeftDexprSimple(ctx *LeftDexprSimpleContext) interface{} {
	// Emit left value format: /<fieldname> xdef
	e.emit("/" + ctx.TypedDate().GetText())
	e.emit("xdef")
	return nil
}

// ============================================================================
// If Statement Visitors
// ============================================================================

func (e *PostfixEmitter) VisitBlockIf(ctx *BlockIfContext) interface{} {
	e.Visit(ctx.Ifblock())
	return nil
}

// Block alternatives that delegate to sub-rules. Each sub-rule handles its
// own emit shape; blockCurly/blockUsing below inline.

func (e *PostfixEmitter) VisitBlockForall(ctx *BlockForallContext) interface{} {
	return e.Visit(ctx.Forallblock())
}

func (e *PostfixEmitter) VisitBlockForeach(ctx *BlockForeachContext) interface{} {
	return e.Visit(ctx.Foreachblock())
}

func (e *PostfixEmitter) VisitBlockFirst(ctx *BlockFirstContext) interface{} {
	return e.Visit(ctx.Firstblock())
}

func (e *PostfixEmitter) VisitBlockUsing(ctx *BlockUsingContext) interface{} {
	return e.Visit(ctx.Usingblock())
}

// VisitBlockGforall: `{ statementList } forallctl` — a body block followed by
// a forallctl that iterates it. Grammar puts the body BEFORE the forallctl,
// but the forallctl visitor already expects a body on the data stack and
// consumes it. Emit the body as a quoted block, then visit the forallctl.
func (e *PostfixEmitter) VisitBlockGforall(ctx *BlockGforallContext) interface{} {
	e.emit("{")
	e.Visit(ctx.StatementList())
	e.emit("}")
	return e.Visit(ctx.Forallctl())
}

// Foreachblock action emitters. opForall signature is (body array --).
// Each element entity is auto-pushed by forall before the body runs.
// `and its <eexpr>` variants push a second related entity; `where <bexpr>`
// variants filter with an if-guarded body.

// VisitForeachSimple: `<eexpr> in <arr> { block }`.
func (e *PostfixEmitter) VisitForeachSimple(ctx *ForeachSimpleContext) interface{} {
	e.emit("{")
	e.Visit(ctx.Block())
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forall")
	return nil
}

// VisitForeachWhere: `<eexpr> in <arr> where <bexpr> { block }`.
func (e *PostfixEmitter) VisitForeachWhere(ctx *ForeachWhereContext) interface{} {
	// forall body is `{ { block } <bexpr> if }` — if the predicate is true,
	// execute the block; otherwise do nothing.
	e.emit("{")
	e.emit("{")
	e.Visit(ctx.Block())
	e.emit("}")
	e.Visit(ctx.Bexpr())
	e.emit("if")
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forall")
	return nil
}

// VisitForeachIts: `<e1> and its <e2> in <arr> { block }`. e1 is pushed by
// forall, e2 is evaluated per-iteration and pushed via entitypush.
func (e *PostfixEmitter) VisitForeachIts(ctx *ForeachItsContext) interface{} {
	e.emit("{")
	e.Visit(ctx.Eexpr(1)) // <e2>
	e.emit("entitypush")
	e.Visit(ctx.Block())
	e.emit("entitypop")
	e.emit("pop")
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forall")
	return nil
}

// VisitForeachItsWhere: `<e1> and its <e2> in <arr> where <bexpr> { block }`.
// entitypop must run whether or not the predicate matched.
func (e *PostfixEmitter) VisitForeachItsWhere(ctx *ForeachItsWhereContext) interface{} {
	e.emit("{")
	e.Visit(ctx.Eexpr(1)) // <e2>
	e.emit("entitypush")
	e.emit("{")
	e.Visit(ctx.Block())
	e.emit("}")
	e.Visit(ctx.Bexpr())
	e.emit("if")
	e.emit("entitypop")
	e.emit("pop")
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forall")
	return nil
}

// Firstblock action emitters. opForfirst is (body test array --), which runs
// body on the first match; opForfirstelse is (body1 body2 test array --),
// with body2 running if no match is found.

// VisitFirstBlockSimple: `for first of <arr> where <bexpr> then <block> endff`.
func (e *PostfixEmitter) VisitFirstBlockSimple(ctx *FirstBlockSimpleContext) interface{} {
	e.emit("{")
	e.Visit(ctx.Block())
	e.emit("}")
	e.emit("{")
	e.Visit(ctx.Bexpr())
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forfirst")
	return nil
}

// VisitFirstBlockElse: `for first of <arr> where <bexpr> then <block1>
// else if none are found <block2> endff`.
func (e *PostfixEmitter) VisitFirstBlockElse(ctx *FirstBlockElseContext) interface{} {
	e.emit("{")
	e.Visit(ctx.Block(0))
	e.emit("}")
	e.emit("{")
	e.Visit(ctx.Block(1))
	e.emit("}")
	e.emit("{")
	e.Visit(ctx.Bexpr())
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forfirstelse")
	return nil
}

// Usingblock emitters. `using E1, E2 { block }` pushes each typed entity
// onto the entity stack, runs the block with them all in scope, then pops
// them in reverse grammar order (ANTLR's right-recursive match structure
// naturally produces reverse-order pops via the recursive visit).

// VisitUsingBlockEntity: `typedEntity usingblock`.
func (e *PostfixEmitter) VisitUsingBlockEntity(ctx *UsingBlockEntityContext) interface{} {
	e.Visit(ctx.TypedEntity())
	e.emit("entitypush")
	e.Visit(ctx.Usingblock())
	e.emit("entitypop")
	e.emit("pop")
	return nil
}

// VisitUsingBlockEntityComma: `typedEntity , usingblock`.
func (e *PostfixEmitter) VisitUsingBlockEntityComma(ctx *UsingBlockEntityCommaContext) interface{} {
	e.Visit(ctx.TypedEntity())
	e.emit("entitypush")
	e.Visit(ctx.Usingblock())
	e.emit("entitypop")
	e.emit("pop")
	return nil
}

// VisitUsingBlockBase: the leaf case — just emit the block inline.
func (e *PostfixEmitter) VisitUsingBlockBase(ctx *UsingBlockBaseContext) interface{} {
	return e.Visit(ctx.Block())
}

// Forallblock emitters (action position, explicit block body).

// VisitForallBlockSimple: `<arr> { block }`. Action form with no filter.
func (e *PostfixEmitter) VisitForallBlockSimple(ctx *ForallBlockSimpleContext) interface{} {
	e.emit("{")
	e.Visit(ctx.Block())
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forall")
	return nil
}

// VisitForallBlockWhere: `<arr> where <bexpr> { block }`. Filter before exec.
func (e *PostfixEmitter) VisitForallBlockWhere(ctx *ForallBlockWhereContext) interface{} {
	e.emit("{")
	e.emit("{")
	e.Visit(ctx.Block())
	e.emit("}")
	e.Visit(ctx.Bexpr())
	e.emit("if")
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forall")
	return nil
}

// VisitFirstBlockItsElse: same as FirstBlockElse but with `and its <e2>`.
// The matched body and the test both run with e2 on the entity stack; body2
// runs when no match is found, so no entity push is needed there.
func (e *PostfixEmitter) VisitFirstBlockItsElse(ctx *FirstBlockItsElseContext) interface{} {
	// body1 wrapper
	e.emit("{")
	e.Visit(ctx.Eexpr())
	e.emit("entitypush")
	e.Visit(ctx.Block(0))
	e.emit("entitypop")
	e.emit("pop")
	e.emit("}")
	// body2: no entity pushed
	e.emit("{")
	e.Visit(ctx.Block(1))
	e.emit("}")
	// test wrapper
	e.emit("{")
	e.Visit(ctx.Eexpr())
	e.emit("entitypush")
	e.Visit(ctx.Bexpr())
	e.emit("entitypop")
	e.emit("pop")
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forfirstelse")
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

// VisitIfElseIf: `else if <ifblock>`. The outer ifblock's ifelse expects a
// false-body; wrap the inner ifblock emit in `{ }` so it becomes a single
// body that ifelse can execute. The inner ifblock itself emits its own
// bexpr / { ... } / false-body / ifelse sequence recursively.
func (e *PostfixEmitter) VisitIfElseIf(ctx *IfElseIfContext) interface{} {
	e.emit("{")
	e.Visit(ctx.Ifblock())
	e.emit("}")
	return nil
}

// Increment/decrement statements. typedLong/typedDouble is a name reference;
// the emit fetches the current value, adjusts by 1, and stores back using the
// same /<name> xdef pattern leftIexprSimple uses.

func (e *PostfixEmitter) VisitIncrementLong(ctx *IncrementLongContext) interface{} {
	name := ctx.TypedLong().GetText()
	e.emit(name)
	e.emit("1")
	e.emit("+")
	e.emit("/" + name)
	e.emit("xdef")
	return nil
}

func (e *PostfixEmitter) VisitIncrementDouble(ctx *IncrementDoubleContext) interface{} {
	name := ctx.TypedDouble().GetText()
	e.emit(name)
	e.emit("1.0")
	e.emit("f+")
	e.emit("/" + name)
	e.emit("xdef")
	return nil
}

func (e *PostfixEmitter) VisitDecrementLong(ctx *DecrementLongContext) interface{} {
	name := ctx.TypedLong().GetText()
	e.emit(name)
	e.emit("1")
	e.emit("-")
	e.emit("/" + name)
	e.emit("xdef")
	return nil
}

func (e *PostfixEmitter) VisitDecrementDouble(ctx *DecrementDoubleContext) interface{} {
	name := ctx.TypedDouble().GetText()
	e.emit(name)
	e.emit("1.0")
	e.emit("f-")
	e.emit("/" + name)
	e.emit("xdef")
	return nil
}

// VisitSetStringFromName: `set <leftStrexpr> = <nexpr>`. Convert the resolved
// name reference to a string.
func (e *PostfixEmitter) VisitSetStringFromName(ctx *SetStringFromNameContext) interface{} {
	e.Visit(ctx.Nexpr())
	e.emit("cvs")
	e.Visit(ctx.LeftStrexpr())
	return nil
}

// VisitSetBoolFromName: `set <leftBexpr> = <nexpr>`.
func (e *PostfixEmitter) VisitSetBoolFromName(ctx *SetBoolFromNameContext) interface{} {
	e.Visit(ctx.Nexpr())
	e.emit("cvb")
	e.Visit(ctx.LeftBexpr())
	return nil
}

// Bexpr emitters — simple single-op translations.

// String comparisons: emit `<l> <r> s<op>` using the s<, s>, s<=, s>= ops.
func (e *PostfixEmitter) VisitBoolStrGt(ctx *BoolStrGtContext) interface{} {
	e.Visit(ctx.Strexpr(0))
	e.Visit(ctx.Strexpr(1))
	e.emit("s>")
	return nil
}
func (e *PostfixEmitter) VisitBoolStrLt(ctx *BoolStrLtContext) interface{} {
	e.Visit(ctx.Strexpr(0))
	e.Visit(ctx.Strexpr(1))
	e.emit("s<")
	return nil
}
func (e *PostfixEmitter) VisitBoolStrGte(ctx *BoolStrGteContext) interface{} {
	e.Visit(ctx.Strexpr(0))
	e.Visit(ctx.Strexpr(1))
	e.emit("s>=")
	return nil
}
func (e *PostfixEmitter) VisitBoolStrLte(ctx *BoolStrLteContext) interface{} {
	e.Visit(ctx.Strexpr(0))
	e.Visit(ctx.Strexpr(1))
	e.emit("s<=")
	return nil
}

// VisitBoolStartsWith: `<s1> starts with <s2>` → `<s1> <s2> startswith`.
func (e *PostfixEmitter) VisitBoolStartsWith(ctx *BoolStartsWithContext) interface{} {
	e.Visit(ctx.Strexpr(0))
	e.Visit(ctx.Strexpr(1))
	e.emit("startswith")
	return nil
}

// VisitBoolMatches: `<s1> matches <s2>` → `<s1> <s2> regexmatch`.
func (e *PostfixEmitter) VisitBoolMatches(ctx *BoolMatchesContext) interface{} {
	e.Visit(ctx.Strexpr(0))
	e.Visit(ctx.Strexpr(1))
	e.emit("regexmatch")
	return nil
}

// VisitBoolDateBetween: `<d1> is between <d2> and <d3>` →
// `<d1> <d2> d>= <d1> <d3> d<= and`. Inclusive range on both endpoints.
func (e *PostfixEmitter) VisitBoolDateBetween(ctx *BoolDateBetweenContext) interface{} {
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(1))
	e.emit("d>=")
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(2))
	e.emit("d<=")
	e.emit("and")
	return nil
}

// Entity in context: `<entity> is in context` → `/<entity> incontext`.
func (e *PostfixEmitter) VisitBoolEntityInContext(ctx *BoolEntityInContextContext) interface{} {
	e.emit("/" + ctx.TypedEntity().GetText())
	e.emit("incontext")
	return nil
}
func (e *PostfixEmitter) VisitBoolEntityNotInContext(ctx *BoolEntityNotInContextContext) interface{} {
	e.emit("/" + ctx.TypedEntity().GetText())
	e.emit("incontext")
	e.emit("not")
	return nil
}
func (e *PostfixEmitter) VisitBoolStrEntityInContext(ctx *BoolStrEntityInContextContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("incontext")
	return nil
}
func (e *PostfixEmitter) VisitBoolStrEntityNotInContext(ctx *BoolStrEntityNotInContextContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("incontext")
	e.emit("not")
	return nil
}

// Rhetorical question forms (`does X?` / `is X?` / `was X?`) just evaluate
// the wrapped bexpr.
func (e *PostfixEmitter) VisitBoolDoesQuestion(ctx *BoolDoesQuestionContext) interface{} {
	return e.Visit(ctx.Bexpr())
}
func (e *PostfixEmitter) VisitBoolIsQuestion(ctx *BoolIsQuestionContext) interface{} {
	return e.Visit(ctx.Bexpr())
}
func (e *PostfixEmitter) VisitBoolWasQuestion(ctx *BoolWasQuestionContext) interface{} {
	return e.Visit(ctx.Bexpr())
}

// (boolean) <indx-or-str> cast.
func (e *PostfixEmitter) VisitBoolFromIndex(ctx *BoolFromIndexContext) interface{} {
	e.Visit(ctx.IndxExpr())
	e.emit("cvb")
	return nil
}
func (e *PostfixEmitter) VisitBoolFromStr(ctx *BoolFromStrContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("cvb")
	return nil
}

// Boolean array-at: `boolean <arr>[<i>]` → `<arr> <i> getat`.
func (e *PostfixEmitter) VisitBoolArrayAt(ctx *BoolArrayAtContext) interface{} {
	e.emit(ctx.TypedArray().GetText())
	e.Visit(ctx.Iexpr())
	e.emit("getat")
	return nil
}

// VisitBoolStrIsOneOf: `<s> is one of <arr>` → `<arr> <s> memberof`.
func (e *PostfixEmitter) VisitBoolStrIsOneOf(ctx *BoolStrIsOneOfContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.Visit(ctx.Strexpr())
	e.emit("memberof")
	return nil
}

// VisitBoolStrIsNotOneOf: `<s> is not one of <arr>` → `<arr> <s> memberof not`.
func (e *PostfixEmitter) VisitBoolStrIsNotOneOf(ctx *BoolStrIsNotOneOfContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.Visit(ctx.Strexpr())
	e.emit("memberof")
	e.emit("not")
	return nil
}

// VisitForctl: `for <leftIexpr> = <number>; <bexpr>; <statement>`.
// Context-position only (reachable via contextForTable/contextFor). The
// outer table body is on the data stack when this runs. Emit:
//   init: <number> cvi leftIexpr-assign
//   loop: { dup execute <statement> } { <bexpr> } while pop
// The body block dups the table body and executes it, then runs the
// increment statement; the test block evaluates bexpr. After while consumes
// body and test, the surviving outer body is dropped with pop.
func (e *PostfixEmitter) VisitForctl(ctx *ForctlContext) interface{} {
	// init — coerce number to int and assign via leftIexpr.
	e.Visit(ctx.Number())
	e.emit("cvi")
	e.Visit(ctx.LeftIexpr())
	// body block
	e.emit("{")
	e.emit("dup")
	e.emit("execute")
	e.Visit(ctx.Statement())
	e.emit("}")
	// test block
	e.emit("{")
	e.Visit(ctx.Bexpr())
	e.emit("}")
	e.emit("while")
	e.emit("pop")
	return nil
}

// Nexpr emitters.

// VisitNameTheName: `the name <strexpr>` → `<strexpr> cvn`. Converts a string
// into a name reference.
func (e *PostfixEmitter) VisitNameTheName(ctx *NameTheNameContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("cvn")
	return nil
}

// VisitNameUsing: `using <eexpr> ( <nexpr> )` — push eexpr onto entity stack,
// resolve nexpr with it in scope, pop entity while preserving the name
// value. Parallels boolUsing.
func (e *PostfixEmitter) VisitNameUsing(ctx *NameUsingContext) interface{} {
	e.Visit(ctx.Eexpr())
	e.emit("entitypush")
	e.Visit(ctx.Nexpr())
	e.emit("entitypop")
	e.emit("swap")
	e.emit("pop")
	return nil
}

// Subtodest emitters. Mirror addtodest but subtract. `<value> <subtodest>`
// expects the rhs value on the stack, then runs field - value and stores.
// Because `-` is a -b operator, we swap before subtracting to get field-value
// rather than value-field.
func (e *PostfixEmitter) VisitSubDestLong(ctx *SubDestLongContext) interface{} {
	name := ctx.TypedLong().GetText()
	e.emit(name)
	e.emit("swap")
	e.emit("-")
	e.emit("/" + name)
	e.emit("xdef")
	return nil
}

func (e *PostfixEmitter) VisitSubDestDouble(ctx *SubDestDoubleContext) interface{} {
	name := ctx.TypedDouble().GetText()
	e.emit(name)
	e.emit("swap")
	e.emit("f-")
	e.emit("/" + name)
	e.emit("xdef")
	return nil
}

// VisitSubtractNum: `subtract <number> from <subtodest>` → push the number,
// then delegate to subtodest which computes field - value and stores.
func (e *PostfixEmitter) VisitSubtractNum(ctx *SubtractNumContext) interface{} {
	e.Visit(ctx.Number())
	e.Visit(ctx.Subtodest())
	return nil
}

// Datestatement emitters. Adjust a date field in place: fetch, compute new
// date, store back. Subtract negates the count before calling adddays.

func (e *PostfixEmitter) VisitDateAddDays(ctx *DateAddDaysContext) interface{} {
	name := ctx.TypedDate().GetText()
	e.emit(name)
	e.Visit(ctx.Number())
	e.emit("adddays")
	e.emit("/" + name)
	e.emit("xdef")
	return nil
}
func (e *PostfixEmitter) VisitDateAddMonths(ctx *DateAddMonthsContext) interface{} {
	name := ctx.TypedDate().GetText()
	e.emit(name)
	e.Visit(ctx.Number())
	e.emit("addmonths")
	e.emit("/" + name)
	e.emit("xdef")
	return nil
}
func (e *PostfixEmitter) VisitDateAddYears(ctx *DateAddYearsContext) interface{} {
	name := ctx.TypedDate().GetText()
	e.emit(name)
	e.Visit(ctx.Number())
	e.emit("addyears")
	e.emit("/" + name)
	e.emit("xdef")
	return nil
}
func (e *PostfixEmitter) VisitDateSubDays(ctx *DateSubDaysContext) interface{} {
	name := ctx.TypedDate().GetText()
	e.emit(name)
	e.Visit(ctx.Number())
	e.emit("negate")
	e.emit("adddays")
	e.emit("/" + name)
	e.emit("xdef")
	return nil
}
func (e *PostfixEmitter) VisitDateSubMonths(ctx *DateSubMonthsContext) interface{} {
	name := ctx.TypedDate().GetText()
	e.emit(name)
	e.Visit(ctx.Number())
	e.emit("negate")
	e.emit("addmonths")
	e.emit("/" + name)
	e.emit("xdef")
	return nil
}
func (e *PostfixEmitter) VisitDateSubYears(ctx *DateSubYearsContext) interface{} {
	name := ctx.TypedDate().GetText()
	e.emit(name)
	e.Visit(ctx.Number())
	e.emit("negate")
	e.emit("addyears")
	e.emit("/" + name)
	e.emit("xdef")
	return nil
}

// Iexpr emitters — date-derived integer forms.

func (e *PostfixEmitter) VisitIntDaysInYear(ctx *IntDaysInYearContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.emit("getdaysinyear")
	return nil
}
func (e *PostfixEmitter) VisitIntDaysInMonth(ctx *IntDaysInMonthContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.emit("getdaysinmonth")
	return nil
}
func (e *PostfixEmitter) VisitIntDayOfMonth(ctx *IntDayOfMonthContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.emit("getdayofmonth")
	return nil
}
func (e *PostfixEmitter) VisitIntDaysBetween(ctx *IntDaysBetweenContext) interface{} {
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(1))
	e.emit("daysbetween")
	return nil
}
func (e *PostfixEmitter) VisitIntMonthsBetween(ctx *IntMonthsBetweenContext) interface{} {
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(1))
	e.emit("monthsbetween")
	return nil
}
func (e *PostfixEmitter) VisitIntYearsBetween(ctx *IntYearsBetweenContext) interface{} {
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(1))
	e.emit("yearsbetween")
	return nil
}
func (e *PostfixEmitter) VisitIntYearOf(ctx *IntYearOfContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.emit("yearof")
	return nil
}

// Strexpr emitters.

// VisitStrConcatName: `<s> + <nexpr>` → `<s> <n> cvs concat`.
func (e *PostfixEmitter) VisitStrConcatName(ctx *StrConcatNameContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.Visit(ctx.Nexpr())
	e.emit("cvs")
	e.emit("concat")
	return nil
}

// VisitStrFromIndex: `(string) <indx>` → `<indx> cvs`.
func (e *PostfixEmitter) VisitStrFromIndex(ctx *StrFromIndexContext) interface{} {
	e.Visit(ctx.IndxExpr())
	e.emit("cvs")
	return nil
}

// VisitStrValueOfInt / Float / Date / Bool: emit value-to-string cast.
func (e *PostfixEmitter) VisitStrValueOfInt(ctx *StrValueOfIntContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.emit("cvs")
	return nil
}
func (e *PostfixEmitter) VisitStrValueOfFloat(ctx *StrValueOfFloatContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.emit("cvs")
	return nil
}
func (e *PostfixEmitter) VisitStrValueOfDate(ctx *StrValueOfDateContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.emit("cvs")
	return nil
}
func (e *PostfixEmitter) VisitStrValueOfBool(ctx *StrValueOfBoolContext) interface{} {
	e.Visit(ctx.Bexpr())
	e.emit("cvs")
	return nil
}

// Eexpr emitters.

// VisitEntityIndex: eexpr as an indxExpr (array/index form). Just delegate.
func (e *PostfixEmitter) VisitEntityIndex(ctx *EntityIndexContext) interface{} {
	return e.Visit(ctx.IndxExpr())
}

// VisitEntityColonRef: `<colonRef> <typedEntity>`. Emit colonRef postfix
// followed by the entity name token (which resolves to the entity object at
// runtime).
func (e *PostfixEmitter) VisitEntityColonRef(ctx *EntityColonRefContext) interface{} {
	e.Visit(ctx.ColonRef())
	e.emit(ctx.TypedEntity().GetText())
	return nil
}

// VisitBoolPlusOrMinus: `<f1> is plus or minus <n> of <f2>` →
// `|f1 - f2| <= n`. The `number` may be int or float; `cvr` coerces to
// double for the comparison.
func (e *PostfixEmitter) VisitBoolPlusOrMinus(ctx *BoolPlusOrMinusContext) interface{} {
	e.Visit(ctx.Fexpr(0))
	e.Visit(ctx.Fexpr(1))
	e.emit("f-")
	e.emit("fabs")
	e.Visit(ctx.Number())
	e.emit("cvr")
	e.emit("f<=")
	return nil
}

// VisitBoolWithinPercent: `<f1> is within <n> percent of <f2>` →
// `|(f1 - f2) / f2| * 100 <= n`.
func (e *PostfixEmitter) VisitBoolWithinPercent(ctx *BoolWithinPercentContext) interface{} {
	e.Visit(ctx.Fexpr(0))
	e.Visit(ctx.Fexpr(1))
	e.emit("f-")
	e.Visit(ctx.Fexpr(1))
	e.emit("fdiv")
	e.emit("fabs")
	e.emit("100.0")
	e.emit("f*")
	e.Visit(ctx.Number())
	e.emit("cvr")
	e.emit("f<=")
	return nil
}

// VisitBoolUsing: `using <eexpr> ( <bexpr> )` — push eexpr onto the entity
// stack, evaluate bexpr, pop the entity while preserving the bool result.
// entitypop pushes the popped entity onto the data stack, so we swap under
// the bool and discard the entity with pop.
func (e *PostfixEmitter) VisitBoolUsing(ctx *BoolUsingContext) interface{} {
	e.Visit(ctx.Eexpr())
	e.emit("entitypush")
	e.Visit(ctx.Bexpr())
	e.emit("entitypop")
	e.emit("swap")
	e.emit("pop")
	return nil
}

// Randomstatements emitters. Array mutation statements.

// VisitRemoveAtIndex: `remove <iexpr> element from <arrayExpr> array`.
// opRemoveAt signature: (array index --).
func (e *PostfixEmitter) VisitRemoveAtIndex(ctx *RemoveAtIndexContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.Visit(ctx.Iexpr())
	e.emit("removeat")
	return nil
}

// VisitRemoveName: `remove <nexpr> from <arrayExpr> array`.
// opRemove signature: (array element --).
func (e *PostfixEmitter) VisitRemoveName(ctx *RemoveNameContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.Visit(ctx.Nexpr())
	e.emit("remove")
	return nil
}

// VisitRemoveString: `remove <strexpr> from <arrayExpr> array`.
func (e *PostfixEmitter) VisitRemoveString(ctx *RemoveStringContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.Visit(ctx.Strexpr())
	e.emit("remove")
	return nil
}

// VisitRemoveEntity: `remove <eexpr> from <arrayExpr> array`.
func (e *PostfixEmitter) VisitRemoveEntity(ctx *RemoveEntityContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.Visit(ctx.Eexpr())
	e.emit("remove")
	return nil
}

// VisitRandomizeArray: `randomize <arrayExpr>`.
func (e *PostfixEmitter) VisitRandomizeArray(ctx *RandomizeArrayContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.emit("randomize")
	return nil
}

// VisitClearstatement: `clear <arrayExpr>`. The statement-level alternative
// takes precedence over randomstatements/clearArray in the parser, so we
// mirror the clearArray emit here.
func (e *PostfixEmitter) VisitClearstatement(ctx *ClearstatementContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.emit("cleararray")
	return nil
}

// VisitClearArray: `clear <arrayExpr>`.
func (e *PostfixEmitter) VisitClearArray(ctx *ClearArrayContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.emit("cleararray")
	return nil
}

// VisitSortAscending: `sort <arrayExpr> in ascending order by <nexpr>`.
// opSortEntities signature: (array name asc --).
func (e *PostfixEmitter) VisitSortAscending(ctx *SortAscendingContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.Visit(ctx.Nexpr())
	e.emit("true")
	e.emit("sortentities")
	return nil
}

// VisitSortDescending: `sort <arrayExpr> in descending order by <nexpr>`.
func (e *PostfixEmitter) VisitSortDescending(ctx *SortDescendingContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.Visit(ctx.Nexpr())
	e.emit("false")
	e.emit("sortentities")
	return nil
}

// Debugstatement emitters. `debug <expr>` and `print <expr>` each push one
// value and call the corresponding operator. Per-type labels exist for
// parser disambiguation; each has the same emit shape.

func (e *PostfixEmitter) VisitDebugStr(ctx *DebugStrContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("debug")
	return nil
}
func (e *PostfixEmitter) VisitDebugBool(ctx *DebugBoolContext) interface{} {
	e.Visit(ctx.Bexpr())
	e.emit("debug")
	return nil
}
func (e *PostfixEmitter) VisitDebugInt(ctx *DebugIntContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.emit("debug")
	return nil
}
func (e *PostfixEmitter) VisitDebugFloat(ctx *DebugFloatContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.emit("debug")
	return nil
}
func (e *PostfixEmitter) VisitDebugEntity(ctx *DebugEntityContext) interface{} {
	e.Visit(ctx.Eexpr())
	e.emit("debug")
	return nil
}
func (e *PostfixEmitter) VisitDebugDate(ctx *DebugDateContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.emit("debug")
	return nil
}
func (e *PostfixEmitter) VisitDebugArray(ctx *DebugArrayContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.emit("debug")
	return nil
}
func (e *PostfixEmitter) VisitPrintStr(ctx *PrintStrContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("print")
	return nil
}
func (e *PostfixEmitter) VisitPrintBool(ctx *PrintBoolContext) interface{} {
	e.Visit(ctx.Bexpr())
	e.emit("print")
	return nil
}
func (e *PostfixEmitter) VisitPrintInt(ctx *PrintIntContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.emit("print")
	return nil
}
func (e *PostfixEmitter) VisitPrintFloat(ctx *PrintFloatContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.emit("print")
	return nil
}
func (e *PostfixEmitter) VisitPrintEntity(ctx *PrintEntityContext) interface{} {
	e.Visit(ctx.Eexpr())
	e.emit("print")
	return nil
}
func (e *PostfixEmitter) VisitPrintDate(ctx *PrintDateContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.emit("print")
	return nil
}
func (e *PostfixEmitter) VisitPrintArray(ctx *PrintArrayContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.emit("print")
	return nil
}

// ============================================================================
// Relationship and Entity Tests
// ============================================================================

func (e *PostfixEmitter) VisitBoolEntityIsOf(ctx *BoolEntityIsOfContext) interface{} {
	// "eexpr IS strexpr OF eexpr" e.g., "client is the parent of ThisClient"
	// Java pattern: /source client /target 0 local@ /type parent relationships findmatch swap pop
	e.emit("/source")
	e.Visit(ctx.Eexpr(0)) // source entity (client)
	e.emit("/target")
	e.Visit(ctx.Eexpr(1)) // target entity (ThisClient)
	e.emit("/type")
	e.Visit(ctx.Strexpr()) // relationship type (parent)
	e.emit("relationships")
	e.emit("findmatch")
	e.emit("swap")
	e.emit("pop")
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

func (e *PostfixEmitter) VisitAddArrayToArray(ctx *AddArrayToArrayContext) interface{} {
	// Check if this is actually a numeric add to a possessive field
	// Pattern: add <value> to <entity>'s <field>
	// The parser matches this as arrayExpr TO arrayExpr, but if the destination
	// is a possessive with a numeric field, we should emit arithmetic

	destExpr := ctx.ArrayExpr(1)

	// Check if destination is colonRef (possessive pattern)
	if colonRefCtx, ok := destExpr.(*ArrayColonRefContext); ok {
		colonRef := colonRefCtx.ColonRef()
		possRef := colonRef.PossessiveRef()

		// Get the field name from typedArray
		fieldName := colonRefCtx.TypedArray().GetText()
		fieldType := e.lookupType(fieldName)

		// If field is numeric, use arithmetic pattern
		if fieldType == TypeInteger || fieldType == TypeLong || fieldType == TypeDouble {
			// Emit the value first
			e.Visit(ctx.ArrayExpr(0))

			// Handle possessive chain for entity reference
			if possChain, ok := possRef.(*PossessiveChainContext); ok {
				tokens := possChain.AllPOSSESSIVE()
				if len(tokens) > 0 {
					poss := tokens[0].GetText()
					entityName := poss[:len(poss)-2] // Remove 's suffix

					if e.emitLocalRef(entityName) {
						// emitLocalRef already emitted "<index> local@"
					} else {
						e.emit(entityName)
					}
				}
			}

			e.emit("entitypush")
			e.emit(fieldName)
			if fieldType == TypeDouble {
				e.emit("f+")
			} else {
				e.emit("+")
			}
			e.emit("/" + fieldName)
			e.emit("xdef")
			e.emit("entitypop")
			return nil
		}
	}

	// Check if destination is a simple field (arrayBase -> arrayExpr2 -> typedArray)
	// Pattern: add <value> to <entity.field>
	if baseCtx, ok := destExpr.(*ArrayBaseContext); ok {
		if arrayExpr2 := baseCtx.ArrayExpr2(); arrayExpr2 != nil {
			if typedCtx, ok := arrayExpr2.(*ArrayTypedContext); ok {
				fieldName := typedCtx.TypedArray().GetText()
				fieldType := e.lookupType(fieldName)

				// If field is numeric, use arithmetic pattern
				if fieldType == TypeInteger || fieldType == TypeLong || fieldType == TypeDouble {
					// Emit value and field, then arithmetic
					e.Visit(ctx.ArrayExpr(0))
					e.emit(fieldName)
					if fieldType == TypeDouble {
						e.emit("f+")
					} else {
						e.emit("+")
					}
					e.emit("/" + fieldName)
					e.emit("xdef")
					return nil
				}
			}
		}
	}

	// Determine if source is a single entity or an array
	// Check source type from EDD
	srcExpr := ctx.ArrayExpr(0)
	srcIsArray := true // Default to array

	if baseCtx, ok := srcExpr.(*ArrayBaseContext); ok {
		if arrayExpr2 := baseCtx.ArrayExpr2(); arrayExpr2 != nil {
			if typedCtx, ok := arrayExpr2.(*ArrayTypedContext); ok {
				srcName := typedCtx.TypedArray().GetText()
				srcType := e.lookupType(srcName)
				// If source is entity type, not array
				if srcType == TypeEntity {
					srcIsArray = false
				}
			}
		}
	}

	// If source is a single entity and dest is an array field, use swap addto
	if !srcIsArray {
		e.Visit(ctx.ArrayExpr(0))
		e.Visit(ctx.ArrayExpr(1))
		e.emit("swap")
		e.emit("addto")
		return nil
	}

	// Default: array-to-array addition
	e.Visit(ctx.ArrayExpr(0))
	e.Visit(ctx.ArrayExpr(1))
	e.emit("true")
	e.emit("addarray")
	return nil
}

func (e *PostfixEmitter) VisitAddArrayNoMember(ctx *AddArrayNoMemberContext) interface{} {
	e.Visit(ctx.ArrayExpr(0))
	e.Visit(ctx.ArrayExpr(1))
	e.emit("false")
	e.emit("addarray")
	return nil
}

func (e *PostfixEmitter) VisitAddEntityToDest(ctx *AddEntityToDestContext) interface{} {
	e.Visit(ctx.Eexpr())
	e.Visit(ctx.Addtodest())
	e.emit("swap") // Java pattern: value dest swap addto
	e.emit("addto")
	return nil
}

func (e *PostfixEmitter) VisitAddStrToDest(ctx *AddStrToDestContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.Visit(ctx.Addtodest())
	e.emit("swap") // Java pattern: value dest swap addto
	e.emit("addto")
	return nil
}

func (e *PostfixEmitter) VisitAddNumToDest(ctx *AddNumToDestContext) interface{} {
	// Pattern: value field + /field xdef
	// e.g., "add 5 to client.income" => "5 client.income + /client.income xdef"
	e.Visit(ctx.Number())
	e.Visit(ctx.Addtodest())
	return nil
}

func (e *PostfixEmitter) VisitAddDestArray(ctx *AddDestArrayContext) interface{} {
	e.Visit(ctx.ArrayExpr2())
	return nil
}

func (e *PostfixEmitter) VisitAddDestLong(ctx *AddDestLongContext) interface{} {
	// Pattern: field + /field xdef
	fieldName := ctx.TypedLong().GetText()
	e.emit(fieldName)
	e.emit("+")
	e.emit("/" + fieldName)
	e.emit("xdef")
	return nil
}

func (e *PostfixEmitter) VisitAddDestDouble(ctx *AddDestDoubleContext) interface{} {
	// Pattern: field f+ /field xdef
	fieldName := ctx.TypedDouble().GetText()
	e.emit(fieldName)
	e.emit("f+")
	e.emit("/" + fieldName)
	e.emit("xdef")
	return nil
}

func (e *PostfixEmitter) VisitAddDestColon(ctx *AddDestColonContext) interface{} {
	// Pattern: <entity-ref> entitypush <field> + /<field> xdef entitypop
	// The colonRef contains the possessive (e.g., "ThisClient's")
	// The addtodest2 contains the field (e.g., "IncomeGroupCount")

	// Get the possessive from colonRef
	colonRef := ctx.ColonRef()
	possRef := colonRef.PossessiveRef()

	// Handle possessive chain
	if possChain, ok := possRef.(*PossessiveChainContext); ok {
		// Get all POSSESSIVE tokens
		tokens := possChain.AllPOSSESSIVE()
		if len(tokens) > 0 {
			poss := tokens[0].GetText()
			// Remove 's suffix to get entity name
			entityName := poss[:len(poss)-2]

			// Check if entity is a local variable
			if e.emitLocalRef(entityName) {
				// emitLocalRef already emitted "<index> local@"
			} else {
				e.emit(entityName)
			}
		}
	}

	e.emit("entitypush")

	// Get the field from addtodest2
	addDest2 := ctx.Addtodest2()
	var fieldName string
	var isDouble bool

	if longCtx, ok := addDest2.(*AddDestLong2Context); ok {
		fieldName = longCtx.TypedLong().GetText()
	} else if doubleCtx, ok := addDest2.(*AddDestDouble2Context); ok {
		fieldName = doubleCtx.TypedDouble().GetText()
		isDouble = true
	} else if arrayCtx, ok := addDest2.(*AddDestArray2Context); ok {
		// Get the field name from the array context
		fieldName = arrayCtx.ArrayExpr2().GetText()

		// Check the type from EDD to determine if it's actually numeric or array
		fieldType := e.lookupType(fieldName)
		switch fieldType {
		case TypeInteger, TypeLong:
			// Integer field - use arithmetic
			e.emit(fieldName)
			e.emit("+")
			e.emit("/" + fieldName)
			e.emit("xdef")
			e.emit("entitypop")
			return nil
		case TypeDouble:
			// Double field - use float arithmetic
			e.emit(fieldName)
			e.emit("f+")
			e.emit("/" + fieldName)
			e.emit("xdef")
			e.emit("entitypop")
			return nil
		default:
			// Array field or unknown - use array operations
			e.Visit(arrayCtx.ArrayExpr2())
			e.emit("swap")
			e.emit("addto")
			e.emit("entitypop")
			return nil
		}
	}

	e.emit(fieldName)
	if isDouble {
		e.emit("f+")
	} else {
		e.emit("+")
	}
	e.emit("/" + fieldName)
	e.emit("xdef")
	e.emit("entitypop")
	return nil
}

func (e *PostfixEmitter) VisitAddDestPossessiveLong(ctx *AddDestPossessiveLongContext) interface{} {
	// Pattern: <entity-ref> entitypush <field> + /<field> xdef entitypop
	// e.g., "add 1 to ThisClient's IncomeGroupCount" =>
	//       "1 0 local@ entitypush IncomeGroupCount + /IncomeGroupCount xdef entitypop"
	poss := ctx.POSSESSIVE().GetText()
	// Remove 's suffix to get entity name
	entityName := poss[:len(poss)-2]
	fieldName := ctx.TypedLong().GetText()

	// Check if entity is a local variable
	if e.emitLocalRef(entityName) {
		// emitLocalRef already emitted "<index> local@"
	} else {
		e.emit(entityName)
	}
	e.emit("entitypush")
	e.emit(fieldName)
	e.emit("+")
	e.emit("/" + fieldName)
	e.emit("xdef")
	e.emit("entitypop")
	return nil
}

func (e *PostfixEmitter) VisitAddDestPossessiveDouble(ctx *AddDestPossessiveDoubleContext) interface{} {
	// Pattern: <entity-ref> entitypush <field> + /<field> xdef entitypop
	poss := ctx.POSSESSIVE().GetText()
	// Remove 's suffix to get entity name
	entityName := poss[:len(poss)-2]
	fieldName := ctx.TypedDouble().GetText()

	// Check if entity is a local variable
	if e.emitLocalRef(entityName) {
		// emitLocalRef already emitted "<index> local@"
	} else {
		e.emit(entityName)
	}
	e.emit("entitypush")
	e.emit(fieldName)
	e.emit("+")
	e.emit("/" + fieldName)
	e.emit("xdef")
	e.emit("entitypop")
	return nil
}

func (e *PostfixEmitter) VisitSubDestPossessiveLong(ctx *SubDestPossessiveLongContext) interface{} {
	// Pattern: <entity-ref> entitypush <field> - /<field> xdef entitypop
	poss := ctx.POSSESSIVE().GetText()
	// Remove 's suffix to get entity name
	entityName := poss[:len(poss)-2]
	fieldName := ctx.TypedLong().GetText()

	// Check if entity is a local variable
	if e.emitLocalRef(entityName) {
		// emitLocalRef already emitted "<index> local@"
	} else {
		e.emit(entityName)
	}
	e.emit("entitypush")
	e.emit(fieldName)
	e.emit("-")
	e.emit("/" + fieldName)
	e.emit("xdef")
	e.emit("entitypop")
	return nil
}

func (e *PostfixEmitter) VisitSubDestPossessiveDouble(ctx *SubDestPossessiveDoubleContext) interface{} {
	// Pattern: <entity-ref> entitypush <field> - /<field> xdef entitypop
	poss := ctx.POSSESSIVE().GetText()
	// Remove 's suffix to get entity name
	entityName := poss[:len(poss)-2]
	fieldName := ctx.TypedDouble().GetText()

	// Check if entity is a local variable
	if e.emitLocalRef(entityName) {
		// emitLocalRef already emitted "<index> local@"
	} else {
		e.emit(entityName)
	}
	e.emit("entitypush")
	e.emit(fieldName)
	e.emit("-")
	e.emit("/" + fieldName)
	e.emit("xdef")
	e.emit("entitypop")
	return nil
}

// ============================================================================
// Context Statement Visitors
// ============================================================================

func (e *PostfixEmitter) VisitAddToContextOf(ctx *AddToContextOfContext) interface{} {
	// Java pattern: <entity> entitypush
	e.Visit(ctx.Eexpr())
	e.emit("entitypush")
	return nil
}

func (e *PostfixEmitter) VisitAddToContextFor(ctx *AddToContextForContext) interface{} {
	// Java pattern: <entity> entitypush
	e.Visit(ctx.Eexpr())
	e.emit("entitypush")
	return nil
}

// ============================================================================
// Mixed Type Comparison Visitors
// ============================================================================

func (e *PostfixEmitter) VisitBoolFloatEqInt(ctx *BoolFloatEqIntContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.Visit(ctx.Iexpr())
	e.emit("f==")
	return nil
}

func (e *PostfixEmitter) VisitBoolIntEqFloat(ctx *BoolIntEqFloatContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.Visit(ctx.Fexpr())
	e.emit("f==")
	return nil
}

func (e *PostfixEmitter) VisitBoolFloatNeqInt(ctx *BoolFloatNeqIntContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.Visit(ctx.Iexpr())
	e.emit("f==")
	e.emit("not")
	return nil
}

func (e *PostfixEmitter) VisitBoolIntNeqFloat(ctx *BoolIntNeqFloatContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.Visit(ctx.Fexpr())
	e.emit("f==")
	e.emit("not")
	return nil
}

func (e *PostfixEmitter) VisitBoolFloatGtInt(ctx *BoolFloatGtIntContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.Visit(ctx.Iexpr())
	e.emit("f>")
	return nil
}

func (e *PostfixEmitter) VisitBoolIntGtFloat(ctx *BoolIntGtFloatContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.Visit(ctx.Fexpr())
	e.emit("f>")
	return nil
}

func (e *PostfixEmitter) VisitBoolFloatLtInt(ctx *BoolFloatLtIntContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.Visit(ctx.Iexpr())
	e.emit("f<")
	return nil
}

func (e *PostfixEmitter) VisitBoolIntLtFloat(ctx *BoolIntLtFloatContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.Visit(ctx.Fexpr())
	e.emit("f<")
	return nil
}

func (e *PostfixEmitter) VisitBoolFloatGteInt(ctx *BoolFloatGteIntContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.Visit(ctx.Iexpr())
	e.emit("f>=")
	return nil
}

func (e *PostfixEmitter) VisitBoolIntGteFloat(ctx *BoolIntGteFloatContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.Visit(ctx.Fexpr())
	e.emit("f>=")
	return nil
}

func (e *PostfixEmitter) VisitBoolFloatLteInt(ctx *BoolFloatLteIntContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.Visit(ctx.Iexpr())
	e.emit("f<=")
	return nil
}

func (e *PostfixEmitter) VisitBoolIntLteFloat(ctx *BoolIntLteFloatContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.Visit(ctx.Fexpr())
	e.emit("f<=")
	return nil
}

// ============================================================================
// Possessive Reference Visitors
// ============================================================================

func (e *PostfixEmitter) VisitPossessiveChain(ctx *PossessiveChainContext) interface{} {
	// Handle possessive chains like "client's plan's"
	// Each POSSESSIVE pushes an entity onto the context stack
	possessives := ctx.AllPOSSESSIVE()
	for i, poss := range possessives {
		text := poss.GetText()
		// Remove "'s" suffix
		entityName := text[:len(text)-2]
		e.emit(entityName)
		if i < len(possessives)-1 {
			e.emit("entitypush")
		}
	}
	// Handle optional nested possessiveRef
	if ctx.PossessiveRef() != nil {
		e.emit("entitypush")
		e.Visit(ctx.PossessiveRef())
		e.emit("entitypop")
	}
	return nil
}

func (e *PostfixEmitter) VisitColonChain(ctx *ColonChainContext) interface{} {
	// Handle colon-style entity references like ":Client:plan"
	e.Visit(ctx.TypedEntity())
	if ctx.PossessiveRef() != nil {
		e.emit("entitypush")
		e.Visit(ctx.PossessiveRef())
		e.emit("entitypop")
	}
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

// ============================================================================
// BigInt Expression Visitors
// ============================================================================

func (e *PostfixEmitter) VisitBigMul(ctx *BigMulContext) interface{} {
	e.Visit(ctx.Bigexpr(0))
	e.Visit(ctx.Bigexpr(1))
	e.emit("b*")
	return nil
}

func (e *PostfixEmitter) VisitBigDiv(ctx *BigDivContext) interface{} {
	e.Visit(ctx.Bigexpr(0))
	e.Visit(ctx.Bigexpr(1))
	e.emit("b/")
	return nil
}

func (e *PostfixEmitter) VisitBigAdd(ctx *BigAddContext) interface{} {
	e.Visit(ctx.Bigexpr(0))
	e.Visit(ctx.Bigexpr(1))
	e.emit("b+")
	return nil
}

func (e *PostfixEmitter) VisitBigSub(ctx *BigSubContext) interface{} {
	e.Visit(ctx.Bigexpr(0))
	e.Visit(ctx.Bigexpr(1))
	e.emit("b-")
	return nil
}

func (e *PostfixEmitter) VisitBigNegate(ctx *BigNegateContext) interface{} {
	e.Visit(ctx.Bigexpr())
	e.emit("bnegate")
	return nil
}

func (e *PostfixEmitter) VisitBigParen(ctx *BigParenContext) interface{} {
	e.Visit(ctx.Bigexpr())
	return nil
}

func (e *PostfixEmitter) VisitBigTyped(ctx *BigTypedContext) interface{} {
	e.Visit(ctx.TypedBigInt())
	return nil
}

func (e *PostfixEmitter) VisitBigColonRef(ctx *BigColonRefContext) interface{} {
	e.Visit(ctx.ColonRef())
	e.Visit(ctx.TypedBigInt())
	return nil
}

func (e *PostfixEmitter) VisitBigFromStr(ctx *BigFromStrContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("cvbi")
	return nil
}

func (e *PostfixEmitter) VisitBigFromInt(ctx *BigFromIntContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.emit("cvbi")
	return nil
}

func (e *PostfixEmitter) VisitBigFromFloat(ctx *BigFromFloatContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.emit("cvbi")
	return nil
}

func (e *PostfixEmitter) VisitBigUsing(ctx *BigUsingContext) interface{} {
	e.Visit(ctx.Eexpr())
	e.emit("entitypush")
	e.Visit(ctx.Bigexpr())
	e.emit("entitypop")
	return nil
}

func (e *PostfixEmitter) VisitBigAbs(ctx *BigAbsContext) interface{} {
	e.Visit(ctx.Bigexpr())
	e.emit("babs")
	return nil
}

func (e *PostfixEmitter) VisitTypedBigInt(ctx *TypedBigIntContext) interface{} {
	name := ctx.GetText()
	// Check if this is a local variable - emit stack frame access
	if !e.emitLocalRef(name) {
		e.emit(name)
	}
	return nil
}

// ============================================================================
// BigInt Local Variable Declaration Visitors
// ============================================================================

func (e *PostfixEmitter) VisitLocalBigIntUndef(ctx *LocalBigIntUndefContext) interface{} {
	name := ctx.UndefinedIdent().GetText()
	e.declareLocal(name, TypeBigInt)
	e.emit("null")
	e.emit("allocate")
	e.emit("execute")
	e.emit("deallocate")
	e.emit("pop")
	return nil
}

func (e *PostfixEmitter) VisitLocalBigIntInit(ctx *LocalBigIntInitContext) interface{} {
	name := ctx.UndefinedIdent().GetText()
	e.declareLocal(name, TypeBigInt)
	e.Visit(ctx.Bigexpr())
	e.emit("cvbi")
	e.emit("allocate")
	e.emit("execute")
	e.emit("deallocate")
	e.emit("pop")
	return nil
}

func (e *PostfixEmitter) VisitLocalBigIntDefined(ctx *LocalBigIntDefinedContext) interface{} {
	e.emit(ctx.TypedBigInt().GetText())
	return nil
}

// ============================================================================
// BigInt Assignment Visitors
// ============================================================================

func (e *PostfixEmitter) VisitSetBigInt(ctx *SetBigIntContext) interface{} {
	e.Visit(ctx.Bigexpr())
	// Look up the target field type, default to bigint from grammar context
	leftField := ctx.LeftBigexpr().GetText()
	fieldType := e.lookupType(leftField)
	if fieldType == "" {
		fieldType = TypeBigInt
	}
	if conv := e.typeConverter(fieldType); conv != "" {
		e.emit(conv)
	}
	e.Visit(ctx.LeftBigexpr())
	return nil
}

func (e *PostfixEmitter) VisitLeftBigexprSimple(ctx *LeftBigexprSimpleContext) interface{} {
	// Emit left value format: /<fieldname> xdef
	e.emit("/" + ctx.TypedBigInt().GetText())
	e.emit("xdef")
	return nil
}

func (e *PostfixEmitter) VisitLeftBigexprColon(ctx *LeftBigexprColonContext) interface{} {
	e.Visit(ctx.ColonRef())
	e.Visit(ctx.LeftBigexpr())
	return nil
}

// ============================================================================
// BigInt Comparison Visitors
// ============================================================================

func (e *PostfixEmitter) VisitBoolBigEq(ctx *BoolBigEqContext) interface{} {
	e.Visit(ctx.Bigexpr(0))
	e.Visit(ctx.Bigexpr(1))
	e.emit("b==")
	return nil
}

func (e *PostfixEmitter) VisitBoolBigNeq(ctx *BoolBigNeqContext) interface{} {
	e.Visit(ctx.Bigexpr(0))
	e.Visit(ctx.Bigexpr(1))
	e.emit("b==")
	e.emit("not")
	return nil
}

func (e *PostfixEmitter) VisitBoolBigGt(ctx *BoolBigGtContext) interface{} {
	e.Visit(ctx.Bigexpr(0))
	e.Visit(ctx.Bigexpr(1))
	e.emit("b>")
	return nil
}

func (e *PostfixEmitter) VisitBoolBigGte(ctx *BoolBigGteContext) interface{} {
	e.Visit(ctx.Bigexpr(0))
	e.Visit(ctx.Bigexpr(1))
	e.emit("b>=")
	return nil
}

func (e *PostfixEmitter) VisitBoolBigLt(ctx *BoolBigLtContext) interface{} {
	e.Visit(ctx.Bigexpr(0))
	e.Visit(ctx.Bigexpr(1))
	e.emit("b<")
	return nil
}

func (e *PostfixEmitter) VisitBoolBigLte(ctx *BoolBigLteContext) interface{} {
	e.Visit(ctx.Bigexpr(0))
	e.Visit(ctx.Bigexpr(1))
	e.emit("b<=")
	return nil
}

// ============================================================================
// Bytes Expression Visitors
// ============================================================================

func (e *PostfixEmitter) VisitBytesLiteral(ctx *BytesLiteralContext) interface{} {
	e.emit(ctx.HEX_BYTES_LITERAL().GetText())
	e.emit("cvbytes")
	return nil
}

func (e *PostfixEmitter) VisitBytesTyped(ctx *BytesTypedContext) interface{} {
	e.Visit(ctx.TypedBytes())
	return nil
}

func (e *PostfixEmitter) VisitBytesColonRef(ctx *BytesColonRefContext) interface{} {
	e.Visit(ctx.ColonRef())
	e.Visit(ctx.TypedBytes())
	return nil
}

func (e *PostfixEmitter) VisitBytesParen(ctx *BytesParenContext) interface{} {
	e.Visit(ctx.Bytesexpr())
	return nil
}

func (e *PostfixEmitter) VisitBytesConcat(ctx *BytesConcatContext) interface{} {
	e.Visit(ctx.Bytesexpr(0))
	e.Visit(ctx.Bytesexpr(1))
	e.emit("bytes+")
	return nil
}

func (e *PostfixEmitter) VisitBytesSlice(ctx *BytesSliceContext) interface{} {
	e.Visit(ctx.Bytesexpr())
	e.Visit(ctx.Iexpr(0))
	e.Visit(ctx.Iexpr(1))
	e.emit("bytesslice")
	return nil
}

func (e *PostfixEmitter) VisitBytesSha256(ctx *BytesSha256Context) interface{} {
	e.Visit(ctx.Bytesexpr())
	e.emit("sha256")
	return nil
}

func (e *PostfixEmitter) VisitBytesKeccak256(ctx *BytesKeccak256Context) interface{} {
	e.Visit(ctx.Bytesexpr())
	e.emit("keccak256")
	return nil
}

func (e *PostfixEmitter) VisitBytesRipemd160(ctx *BytesRipemd160Context) interface{} {
	e.Visit(ctx.Bytesexpr())
	e.emit("ripemd160")
	return nil
}

func (e *PostfixEmitter) VisitBytesSha3(ctx *BytesSha3Context) interface{} {
	e.Visit(ctx.Bytesexpr())
	e.emit("sha3")
	return nil
}

func (e *PostfixEmitter) VisitBytesCvHex(ctx *BytesCvHexContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("cvhex")
	return nil
}

func (e *PostfixEmitter) VisitBytesCvBase58Check(ctx *BytesCvBase58CheckContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("cvb58check")
	e.emit("pop")
	return nil
}

func (e *PostfixEmitter) VisitBytesCvBech32(ctx *BytesCvBech32Context) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("cvbech32")
	e.emit("pop")
	return nil
}

func (e *PostfixEmitter) VisitBytesCvBigInt(ctx *BytesCvBigIntContext) interface{} {
	e.Visit(ctx.Bigexpr())
	e.Visit(ctx.Iexpr())
	e.emit("bigintbytes")
	return nil
}

func (e *PostfixEmitter) VisitStrHexOfBytes(ctx *StrHexOfBytesContext) interface{} {
	e.Visit(ctx.Bytesexpr())
	e.emit("hex")
	return nil
}

func (e *PostfixEmitter) VisitStrBase58CheckOfBytes(ctx *StrBase58CheckOfBytesContext) interface{} {
	e.Visit(ctx.Bytesexpr())
	e.Visit(ctx.Iexpr())
	e.emit("b58check")
	return nil
}

func (e *PostfixEmitter) VisitStrBech32OfBytes(ctx *StrBech32OfBytesContext) interface{} {
	e.Visit(ctx.Bytesexpr())
	e.Visit(ctx.Strexpr())
	e.emit("bech32")
	return nil
}

func (e *PostfixEmitter) VisitBigFromBytes(ctx *BigFromBytesContext) interface{} {
	e.Visit(ctx.Bytesexpr())
	e.emit("bytesbigint")
	return nil
}

func (e *PostfixEmitter) VisitTypedBytes(ctx *TypedBytesContext) interface{} {
	name := ctx.GetText()
	if !e.emitLocalRef(name) {
		e.emit(name)
	}
	return nil
}

func (e *PostfixEmitter) VisitBoolBytesEq(ctx *BoolBytesEqContext) interface{} {
	e.Visit(ctx.Bytesexpr(0))
	e.Visit(ctx.Bytesexpr(1))
	e.emit("bytes==")
	return nil
}

func (e *PostfixEmitter) VisitBoolBytesNeq(ctx *BoolBytesNeqContext) interface{} {
	e.Visit(ctx.Bytesexpr(0))
	e.Visit(ctx.Bytesexpr(1))
	e.emit("bytes!=")
	return nil
}

// ============================================================================
// Bytes Local Variable Declaration Visitors
// ============================================================================

func (e *PostfixEmitter) VisitLocalBytesUndef(ctx *LocalBytesUndefContext) interface{} {
	name := ctx.UndefinedIdent().GetText()
	e.declareLocal(name, TypeBytes)
	e.emit("null")
	e.emit("allocate")
	e.emit("execute")
	e.emit("deallocate")
	e.emit("pop")
	return nil
}

func (e *PostfixEmitter) VisitLocalBytesInit(ctx *LocalBytesInitContext) interface{} {
	name := ctx.UndefinedIdent().GetText()
	e.declareLocal(name, TypeBytes)
	e.Visit(ctx.Bytesexpr())
	e.emit("cvbytes")
	e.emit("allocate")
	e.emit("execute")
	e.emit("deallocate")
	e.emit("pop")
	return nil
}

func (e *PostfixEmitter) VisitLocalBytesDefined(ctx *LocalBytesDefinedContext) interface{} {
	e.emit(ctx.TypedBytes().GetText())
	return nil
}
