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
	"math/big"
	"strings"

	"github.com/antlr4-go/antlr/v4"
)

// LocalVar tracks a local variable's stack frame index and type.
//
// For TypeEntity locals declared via `local entity r = new T entity`,
// the EntityType field captures `T` so mutations like
// `set r.field = X` can look up `T.field` in the EDD symbol table for
// correct type-aware dispatch (#819). Empty EntityType means the type
// wasn't determinable at declaration time (e.g.
// `local entity r = some_dynamic_eexpr`) — callers fall back to
// default integer dispatch in that case.
type LocalVar struct {
	Index      int
	Type       string
	EntityType string // for Type==TypeEntity locals from `new T entity`
}

// PostfixEmitter walks the EL parse tree and emits postfix notation.
type PostfixEmitter struct {
	*BaseELVisitor
	output  strings.Builder
	errors  []error
	symbols map[string]string // symbol table for type resolution from EDD
	// operatorExists reports whether a statement-form operator name is
	// registered with the engine. Injected rather than imported: the runtime
	// registry lives under pkg/dtrules/operators, which imports pkg/dtrules,
	// and pkg/dtrules's own tests import this package — importing it here
	// closes that loop. nil means no check, which is what el's isolated unit
	// tests want when they exercise emission with a stand-in operator name.
	operatorExists func(string) bool
	// operatorArity reports how many arguments a statement-form call to this
	// operator must supply, or <= 0 when unrecorded.
	operatorArity     func(string) int
	locals            map[string]LocalVar // local variable stack frame indices
	localCnt          int                 // next available local variable index
	resolveCollection func(entityType string) (ownerEntity, fieldName string, err error)
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

// SetCollectionResolver registers the callback used by the
// `for all <type> entities` DSL form to turn a bare entity-type name into the
// owning array's `<owner>.<field>` path.
func (e *PostfixEmitter) SetCollectionResolver(fn func(entityType string) (string, string, error)) {
	e.resolveCollection = fn
}

// declareLocal registers a local variable and returns its stack frame index.
func (e *PostfixEmitter) declareLocal(name string, varType string) int {
	return e.declareLocalEntity(name, varType, "")
}

// declareLocalEntity registers a local variable that is specifically a
// TypeEntity local with a known entity type T. The third argument is
// the entity-type name (e.g. "token_recipient"); pass "" if the type
// isn't known. Stored on LocalVar so mutationType can resolve
// `<local>.<field>` against `<T>.<field>` in the symbol table — without
// this, SET dispatch falls back to default integer cv* (#819).
func (e *PostfixEmitter) declareLocalEntity(name, varType, entityType string) int {
	name = strings.ToLower(name)
	idx := e.localCnt
	e.locals[name] = LocalVar{Index: idx, Type: varType, EntityType: entityType}
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

// emitAliasFieldAccess tries to resolve `head.tail` where head is a local
// entity alias introduced by `for all <arr> as <alias>`. When it is, the
// emitter produces `<N> local@ /<tail> get` so the attribute is fetched from
// the aliased entity rather than the entity stack. Returns true on a hit.
//
// `for all <arr> as <alias>` deliberately does NOT push the iteration entity
// on the entity stack, so field references qualified by the alias are the
// only way to reach it. That breaks the bare-name lookup path used by
// non-aliased forall bodies, which this helper restores via the local slot.
func (e *PostfixEmitter) emitAliasFieldAccess(name string) bool {
	idx := strings.Index(name, ".")
	if idx <= 0 {
		return false
	}
	head := name[:idx]
	tail := name[idx+1:]
	if tail == "" || strings.Contains(tail, ".") {
		return false
	}
	lv, ok := e.lookupLocal(head)
	if !ok || lv.Type != TypeEntity {
		return false
	}
	e.emit(fmt.Sprintf("%d", lv.Index))
	e.emit("local@")
	e.emit("/" + tail)
	e.emit("get")
	return true
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
		return "cvdate"
	case TypeBigInt:
		return "cvbi"
	case TypeBytes:
		return "cvbytes"
	case TypeFixed:
		return "cvfp"
	case TypeArray, TypeName, TypeXmlValue:
		return "" // No conversion needed
	default:
		return "cvi" // Default to integer conversion for unknown types
	}
}

// getExprType determines the type of an integer expression by examining its
// structure. Returns TypeFixed for FP_LITERAL and (fixed)-cast nodes and for
// fixed-typed references; TypeBigInt for bigint references; otherwise
// TypeInteger.
func (e *PostfixEmitter) getExprType(ctx antlr.ParseTree) string {
	if ctx == nil {
		return TypeInteger
	}

	// FP_LITERAL (e.g. `1.5fp`) and (fixed) casts are fixed-typed directly
	// from the parse tree — no raw-text inspection needed.
	switch ctx.(type) {
	case *FixedLiteralContext, *FixedFromStrContext, *FixedFromNumberContext,
		*FixedFromFloatContext, *FixedFromIndexContext:
		return TypeFixed
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

	// Float-typed NAME contexts: a fixed/bigint/double field that matched the
	// grammar's TypedDouble alternative. Resolve to the DECLARED type so a
	// fixed/bigint field used in a float position is still seen as exact and
	// caught by the double/exact reject — not silently treated as double (#894).
	if typedCtx, ok := ctx.(*FloatTypedContext); ok {
		if td := typedCtx.TypedDouble(); td != nil {
			if ident := td.IDENT(); ident != nil {
				name := ident.GetText()
				if lv, ok := e.lookupLocal(name); ok {
					return lv.Type
				}
				if t := e.lookupType(name); t != "" {
					return t
				}
			}
		}
	}
	if colonCtx, ok := ctx.(*FloatColonRefContext); ok {
		if td := colonCtx.TypedDouble(); td != nil {
			if ident := td.IDENT(); ident != nil {
				name := ident.GetText()
				if lv, ok := e.lookupLocal(name); ok {
					return lv.Type
				}
				if cr := colonCtx.ColonRef(); cr != nil {
					if t := e.lookupType(cr.GetText() + "." + name); t != "" {
						return t
					}
				}
				if t := e.lookupType(name); t != "" {
					return t
				}
			}
		}
	}

	// For compound expressions, propagate the widest operand type via
	// promoteArithType (Fixed > BigInt > Double > Integer).
	switch c := ctx.(type) {
	case *IntAddContext:
		return promoteArithType(e.getExprType(c.Iexpr(0)), e.getExprType(c.Iexpr(1)))
	case *IntSubContext:
		return promoteArithType(e.getExprType(c.Iexpr(0)), e.getExprType(c.Iexpr(1)))
	case *IntMulContext:
		return promoteArithType(e.getExprType(c.Iexpr(0)), e.getExprType(c.Iexpr(1)))
	case *IntDivContext:
		return promoteArithType(e.getExprType(c.Iexpr(0)), e.getExprType(c.Iexpr(1)))
	case *IntNegateContext:
		return e.getExprType(c.Iexpr())
	case *IntParenContext:
		return e.getExprType(c.Iexpr())
	}

	// Float ARITHMETIC compounds promote through their operands, mirroring
	// what their emitters actually produce (emitMixedFloatArith): two fixed
	// operands that matched the grammar's fexpr alternative yield a fixed
	// result (fp-family op), not a double (#903). Without this, a nested
	// `a * b` dividend types as double and downstream dispatch degrades.
	switch c := ctx.(type) {
	case *FloatAddFloatContext:
		return promoteArithType(e.getExprType(c.Fexpr(0)), e.getExprType(c.Fexpr(1)))
	case *FloatSubFloatContext:
		return promoteArithType(e.getExprType(c.Fexpr(0)), e.getExprType(c.Fexpr(1)))
	case *FloatMulFloatContext:
		return promoteArithType(e.getExprType(c.Fexpr(0)), e.getExprType(c.Fexpr(1)))
	case *FloatDivFloatContext:
		return promoteArithType(e.getExprType(c.Fexpr(0)), e.getExprType(c.Fexpr(1)))
	case *FloatAddIntContext:
		return promoteArithType(e.getExprType(c.Fexpr()), e.getExprType(c.Iexpr()))
	case *FloatSubIntContext:
		return promoteArithType(e.getExprType(c.Fexpr()), e.getExprType(c.Iexpr()))
	case *FloatMulIntContext:
		return promoteArithType(e.getExprType(c.Fexpr()), e.getExprType(c.Iexpr()))
	case *FloatDivIntContext:
		return promoteArithType(e.getExprType(c.Fexpr()), e.getExprType(c.Iexpr()))
	case *IntAddFloatContext:
		return promoteArithType(e.getExprType(c.Iexpr()), e.getExprType(c.Fexpr()))
	case *IntSubFloatContext:
		return promoteArithType(e.getExprType(c.Iexpr()), e.getExprType(c.Fexpr()))
	case *IntMulFloatContext:
		return promoteArithType(e.getExprType(c.Iexpr()), e.getExprType(c.Fexpr()))
	case *IntDivFloatContext:
		return promoteArithType(e.getExprType(c.Iexpr()), e.getExprType(c.Fexpr()))
	}

	// Float-VALUED expressions (literals, explicit (double) casts, float
	// negation) produce a double. Typed-name fexprs were resolved above;
	// what reaches here is genuinely double — so a double literal mixed with a
	// fixed/bigint field is caught by the double/exact reject (#894).
	switch c := ctx.(type) {
	case *FloatLiteralContext, *FloatFromStrContext, *FloatFromIntContext,
		*FloatFromIndexContext, *FloatNegateContext:
		return TypeDouble
	case *DivideRoundingByContext:
		// The fp-family divide ops (fp/, fphalfup/, fpdivr/) always produce
		// a fixed result.
		return TypeFixed
	case *FloatParenContext:
		return e.getExprType(c.Fexpr())
	}

	// Fallback: a bare-identifier expression in any wrapper context (Number,
	// ArrayExpr, FloatTyped, …) whose specific case isn't enumerated above.
	// A compound expression's text always contains operators/parens, so an
	// identifier-only text is necessarily a simple name reference — resolve
	// its declared type instead of defaulting to integer, which let double/
	// fixed operands slip past the #876 reject on the mul-by and mutation
	// paths (#882).
	if name := ctx.GetText(); isIdentifier(name) {
		if lv, ok := e.lookupLocal(name); ok {
			return lv.Type
		}
		if t := e.lookupType(name); t != "" {
			return t
		}
	}

	return TypeInteger
}

// promoteArithType returns the widest type for a mixed-type integer
// arithmetic or comparison expression. Precedence (highest first):
//
//	Fixed > BigInt > Double > Integer
//
// Double sits below BigInt because BigInt is the lossless-large-integer
// type and silently snapping a bigint operand to a double's mantissa
// would discard precision. Double sits above Integer because mixing
// `int_field + 0.5` is the standard widening case (int → double).
//
// Two doubles produce a double — closes #790's sibling gap where
// double × double was returning Integer and the emitter then picked
// integer ops (`*`, `min`, `cvi`) on operands the runtime stored as
// `*RDouble`, crashing later with `IntValue: No Integer value exists
// for this type`.
func promoteArithType(a, b string) string {
	if a == TypeFixed || b == TypeFixed {
		return TypeFixed
	}
	if a == TypeBigInt || b == TypeBigInt {
		return TypeBigInt
	}
	if a == TypeDouble || b == TypeDouble {
		return TypeDouble
	}
	return TypeInteger
}

// isDoubleExactMix reports whether one operand is double and the other is an
// exact type (fixed or bigint) — a combination DTRules will not promote
// implicitly.
func isDoubleExactMix(a, b string) bool {
	isExact := func(t string) bool { return t == TypeFixed || t == TypeBigInt }
	return (a == TypeDouble && isExact(b)) || (b == TypeDouble && isExact(a))
}

// promote is the emission-site wrapper over promoteArithType. It records a
// compile error when a double is mixed with an exact type (fixed or bigint)
// instead of silently emitting a cast: the runtime deliberately refuses to
// promote double→fixed implicitly (see RFixed.promote), and truncating
// double→bigint would silently drop precision. Authors must opt in with an
// explicit cast, e.g. `(double) x` or `(fixed) x`. (#876)
func (e *PostfixEmitter) promote(a, b string) string {
	if isDoubleExactMix(a, b) {
		e.emitDoubleMixError(exactOf(a, b))
	}
	return promoteArithType(a, b)
}

// exactOf returns whichever of a/b is the exact type in a double/exact mix.
func exactOf(a, b string) string {
	if a == TypeDouble {
		return b
	}
	return a
}

// emitDoubleMixError records the standard #876 diagnostic for an implicit
// double/exact-type combination.
func (e *PostfixEmitter) emitDoubleMixError(exact string) {
	e.emitError("cannot combine double with %s implicitly; add an explicit cast "+
		"(e.g. \"(%s) x\" to keep exactness, or \"(double) x\" to compute in double)", exact, exact)
}

// arithOp picks the correct postfix opcode for an arithmetic or comparison
// operation based on the promoted expression type.
// arithOp picks the right operator name for the promoted target type.
// Caller passes the per-family op names; we route Fixed → fpOp,
// BigInt → bigOp, Double → dblOp, Integer (or unknown) → intOp.
//
// Adding the dblOp parameter closes the dispatch gap that made
// `double × double` (or any expression promoting to Double via
// `promoteArithType`) emit integer ops — runtime then crashed on
// `IntValue` because the operands were `*RDouble`.
func arithOp(target, intOp, bigOp, dblOp, fpOp string) string {
	switch target {
	case TypeFixed:
		return fpOp
	case TypeBigInt:
		return bigOp
	case TypeDouble:
		return dblOp
	default:
		return intOp
	}
}

// emitWithTypeConversion emits an integer-family expression and casts the
// result to targetType on the stack. Handles int→bigint, int→fixed, and
// bigint→fixed promotions. Same-type targets emit no cast.
func (e *PostfixEmitter) emitWithTypeConversion(ctx antlr.ParseTree, targetType string) {
	srcType := e.getExprType(ctx)
	e.Visit(ctx)
	e.emitTypeCast(srcType, targetType)
}

// emitTypeCast emits the postfix cast op needed to convert a stack value of
// srcType into targetType. Same-type is a no-op. Only the three implicit
// numeric promotions are handled (int→bigint, int→fixed, bigint→fixed);
// double→fixed requires explicit cvfp and lands here only as a pre-cast
// source, never as a target.
func (e *PostfixEmitter) emitTypeCast(srcType, targetType string) {
	if srcType == targetType {
		return
	}
	switch targetType {
	case TypeBigInt:
		if srcType == TypeInteger {
			e.emit("cvbi")
		}
	case TypeFixed:
		if srcType == TypeInteger || srcType == TypeBigInt {
			e.emit("cvfp")
		}
	}
}

// mutationType returns the declared numeric type for a mutation target,
// checking the local-variable table before the EDD symbol table. Locals
// shadow entity fields in the postfix emitter's scope, so a `local bigint
// x` must dispatch as bigint even if the EDD also has an `x` symbol.
// Returns "" when no declared type is found.
func (e *PostfixEmitter) mutationType(name string) string {
	if lv, ok := e.lookupLocal(name); ok {
		return lv.Type
	}
	// #819: `<local-entity>.<field>` resolution. If `name` has the
	// shape `head.tail` and `head` is a TypeEntity local with a known
	// entity type, look up `<EntityType>.<tail>` in the EDD symbol
	// table. Without this, SET dispatch on local-entity fields falls
	// back to the default integer cv* cast.
	if idx := strings.Index(name, "."); idx > 0 {
		head := name[:idx]
		tail := name[idx+1:]
		if tail != "" && !strings.Contains(tail, ".") {
			if lv, ok := e.lookupLocal(head); ok && lv.Type == TypeEntity && lv.EntityType != "" {
				if t := e.lookupType(lv.EntityType + "." + tail); t != "" {
					return t
				}
			}
		}
	}
	return e.lookupType(name)
}

// emitFieldPush pushes the current value of a mutation target onto the
// data stack. For declared locals it emits `<index> local@`; for entity
// fields it emits the bare name (which the postfix runtime resolves via
// the current entity frame).
func (e *PostfixEmitter) emitFieldPush(name string) {
	if !e.emitLocalRef(name) {
		e.emit(name)
	}
}

// emitFieldStore stores the top-of-stack value back into a mutation
// target. Tries each strategy in order until one matches:
//
//  1. The name is a declared local — emit `<index> local!`.
//  2. The name has the shape `local.field` where `local` is a TypeEntity
//     local — emit the entity-stack-mediated assignment sequence
//     `<slot> local@ entitypush /<field> xdef entitypop pop` (#819).
//     Symmetric to emitAliasFieldAccess on the read side.
//  3. Plain entity field — emit `/<name> xdef`.
func (e *PostfixEmitter) emitFieldStore(name string) {
	if e.emitLocalAssign(name) {
		return
	}
	if e.emitAliasFieldStore(name) {
		return
	}
	e.emit("/" + name)
	e.emit("xdef")
}

// emitAliasFieldStore handles `local.field = value` for a TypeEntity
// local. Stack effect with value already on top:
//
//	[..., value]                  on entry
//	 → <slot> local@              [..., value, entity]
//	 → entitypush                 [..., value]            entity-stack: [..., entity]
//	 → /<field>                   [..., value, /field]
//	 → xdef                       [...]                   (entity.Put via stack lookup)
//	 → entitypop                  [..., entity]
//	 → pop                        [...]
//
// Returns false (no emission) if the name isn't shaped like
// `head.tail` or the head isn't a TypeEntity local. Symmetric to
// `emitAliasFieldAccess`.
func (e *PostfixEmitter) emitAliasFieldStore(name string) bool {
	idx := strings.Index(name, ".")
	if idx <= 0 {
		return false
	}
	head := name[:idx]
	tail := name[idx+1:]
	if tail == "" || strings.Contains(tail, ".") {
		return false
	}
	lv, ok := e.lookupLocal(head)
	if !ok || lv.Type != TypeEntity {
		return false
	}
	e.emit(fmt.Sprintf("%d", lv.Index))
	e.emit("local@")
	e.emit("entitypush")
	e.emit("/" + tail)
	e.emit("xdef")
	e.emit("entitypop")
	e.emit("pop")
	return true
}

// emitTypeAwareAddSub emits the store-back sequence for a field mutation
// (`add value to field` or `subtract value from field`) where the value is
// already pushed onto the data stack by the caller. Looks up the target's
// declared type (local first, then EDD symbol table) and emits the correct
// typed op (fp+/fp-, b+/b-, f+/f-, +/-) after promoting the value to
// match. Subtract orders the operands so the result is `field - value`,
// matching DTRules' existing semantics. The read (push current value) and
// write (store-back) both route through emitFieldPush / emitFieldStore so
// locals dispatch to `<index> local@` / `<index> local!` while entity
// fields keep the bare-name / xdef forms.
//
// op is "+" for add, "-" for sub. Any other op panics — intentional; this
// helper is not meant to grow.
func (e *PostfixEmitter) emitTypeAwareAddSub(fieldName, op string) {
	if op != "+" && op != "-" {
		panic("emitTypeAwareAddSub: op must be + or -")
	}
	switch e.mutationType(fieldName) {
	case TypeFixed:
		e.emit("cvfp")
		e.emitFieldPush(fieldName)
		if op == "-" {
			e.emit("swap")
		}
		e.emit("fp" + op)
	case TypeBigInt:
		e.emit("cvbi")
		e.emitFieldPush(fieldName)
		if op == "-" {
			e.emit("swap")
		}
		e.emit("b" + op)
	case TypeDouble:
		e.emit("cvd")
		e.emitFieldPush(fieldName)
		if op == "-" {
			e.emit("swap")
		}
		e.emit("f" + op)
	default:
		e.emitFieldPush(fieldName)
		if op == "-" {
			e.emit("swap")
		}
		e.emit(op)
	}
	e.emitFieldStore(fieldName)
}

// identNumericType returns the EDD/local-declared type of a name reference
// if it resolves to an integer, bigint, or fixed field. Returns "" for
// anything else (string, boolean, entity, double, unknown). Double is
// intentionally excluded so fixed ↔ double comparisons stay on the legacy
// string-compare path rather than silently snapping onto the 10^-8 grid.
func (e *PostfixEmitter) identNumericType(name string) string {
	if lv, ok := e.lookupLocal(name); ok {
		switch lv.Type {
		case TypeInteger, TypeLong, TypeBigInt, TypeFixed:
			return lv.Type
		}
	}
	switch t := e.lookupType(name); t {
	case TypeInteger, TypeLong, TypeBigInt, TypeFixed:
		return t
	}
	return ""
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

// ResetLocals clears the local variable state (names + slot counter).
// Call this between tables to prevent slot indices from bleeding across
// independent compilation scopes. Within a single table — the context,
// conditions, actions — locals must persist so `<alias>.<field>` in a
// condition can see the slot declared in the context.
func (e *PostfixEmitter) ResetLocals() {
	e.locals = make(map[string]LocalVar)
	e.localCnt = 0
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

	target := e.promote(e.getExprType(left), e.getExprType(right))
	e.emitWithTypeConversion(left, target)
	e.emitWithTypeConversion(right, target)
	e.emit(arithOp(target, "==", "b==", "f==", "fp=="))
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

	target := e.promote(e.getExprType(left), e.getExprType(right))
	e.emitWithTypeConversion(left, target)
	e.emitWithTypeConversion(right, target)
	switch target {
	case TypeFixed:
		e.emit("fp!=")
	case TypeBigInt:
		e.emit("b!=")
	case TypeDouble:
		e.emit("f!=")
	default:
		e.emit("==")
		e.emit("not")
	}
	return nil
}

func (e *PostfixEmitter) VisitBoolIntGt(ctx *BoolIntGtContext) interface{} {
	left, right := ctx.Iexpr(0), ctx.Iexpr(1)
	target := e.promote(e.getExprType(left), e.getExprType(right))
	e.emitWithTypeConversion(left, target)
	e.emitWithTypeConversion(right, target)
	e.emit(arithOp(target, ">", "b>", "f>", "fp>"))
	return nil
}

func (e *PostfixEmitter) VisitBoolIntGte(ctx *BoolIntGteContext) interface{} {
	left, right := ctx.Iexpr(0), ctx.Iexpr(1)
	target := e.promote(e.getExprType(left), e.getExprType(right))
	e.emitWithTypeConversion(left, target)
	e.emitWithTypeConversion(right, target)
	e.emit(arithOp(target, ">=", "b>=", "f>=", "fp>="))
	return nil
}

func (e *PostfixEmitter) VisitBoolIntLt(ctx *BoolIntLtContext) interface{} {
	left, right := ctx.Iexpr(0), ctx.Iexpr(1)
	target := e.promote(e.getExprType(left), e.getExprType(right))
	e.emitWithTypeConversion(left, target)
	e.emitWithTypeConversion(right, target)
	e.emit(arithOp(target, "<", "b<", "f<", "fp<"))
	return nil
}

func (e *PostfixEmitter) VisitBoolIntLte(ctx *BoolIntLteContext) interface{} {
	left, right := ctx.Iexpr(0), ctx.Iexpr(1)
	target := e.promote(e.getExprType(left), e.getExprType(right))
	e.emitWithTypeConversion(left, target)
	e.emitWithTypeConversion(right, target)
	e.emit(arithOp(target, "<=", "b<=", "f<=", "fp<="))
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
	// `strneq` was never registered as a runtime op; the runtime has
	// only `s==` (alias `streq`). Compose inequality as equality then
	// `not` so the dispatch resolves (#835).
	e.Visit(left)
	e.Visit(right)
	e.emit("s==")
	e.emit("not")
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
	// `<a> is not <b>` is logical inequality. Pre-fix this emitted
	// `strneq not` which was doubly broken: `strneq` is unregistered,
	// and even if it weren't, the trailing `not` would re-invert the
	// result. The correct form is `s== not` (#835).
	e.Visit(left)
	e.Visit(right)
	e.emit("s==")
	e.emit("not")
	return nil
}

func (e *PostfixEmitter) VisitBoolStrEqIc(ctx *BoolStrEqIcContext) interface{} {
	e.Visit(ctx.Strexpr(0))
	e.Visit(ctx.Strexpr(1))
	// `sic==` was never registered; runtime has `s==i` (alias
	// `streqignorecase`) (#835).
	e.emit("s==i")
	return nil
}

// collectBlistStrexprs walks a blist / blistIc tree (left-leaning,
// terminating in an OR alt) and returns the flat list of comparison
// values in source order. blistMulti has shape `strexpr COMMA blist`
// and blistOr has shape `OR strexpr`. Same shape for blistIc.
//
// Pulling the walk out of the antlr Visit dispatch lets the parent
// visitor emit the membership-test postfix in one pass without
// relying on individual Blist* overrides (which never fire because
// VisitChildren is a no-op).
func collectBlistStrexprs(node antlr.Tree) []IStrexprContext {
	var out []IStrexprContext
	var walk func(antlr.Tree)
	walk = func(n antlr.Tree) {
		switch c := n.(type) {
		case *BlistMultiContext:
			out = append(out, c.Strexpr())
			if tail := c.Blist(); tail != nil {
				walk(tail)
			}
		case *BlistOrContext:
			out = append(out, c.Strexpr())
		case *BlistIcMultiContext:
			out = append(out, c.Strexpr())
			if tail := c.Blist(); tail != nil {
				walk(tail)
			}
		case *BlistIcOrContext:
			out = append(out, c.Strexpr())
		}
	}
	walk(node)
	return out
}

// emitStrEqList emits `<lhs> v0 <op>  <lhs> v1 <op> or  <lhs> v2 <op>
// or  ...` — the lhs is re-visited per element rather than dup'd so
// any side effects in <lhs> evaluation match the singleton boolStrEq
// path. Used by both boolStrEqList (op=s==) and boolStrEqIcList
// (op=s==i).
func (e *PostfixEmitter) emitStrEqList(lhs IStrexprContext, values []IStrexprContext, eqOp string) {
	if len(values) == 0 {
		// Defensive: empty list shouldn't be reachable through the grammar
		// (blist always terminates in an OR strexpr), but emit a literal
		// false so a malformed tree fails closed rather than silently.
		e.emit("false")
		return
	}
	for i, v := range values {
		e.Visit(lhs)
		e.Visit(v)
		e.emit(eqOp)
		if i > 0 {
			e.emit("or")
		}
	}
}

// VisitBoolStrEqList: `<strexpr> = "a", "b" or "c"` — membership
// against a comma-separated list ending in OR. Emit a chain of
// `s==` comparisons OR'd together. Pre-fix this rule silently
// emitted nothing.
func (e *PostfixEmitter) VisitBoolStrEqList(ctx *BoolStrEqListContext) interface{} {
	e.emitStrEqList(ctx.Strexpr(), collectBlistStrexprs(ctx.Blist()), "s==")
	return nil
}

// VisitBoolStrEqIcList: `<strexpr> equals "a", "b" or "c"` —
// case-insensitive variant. Same shape as boolStrEqList but using
// the s==i op. Pre-fix this rule silently emitted nothing.
func (e *PostfixEmitter) VisitBoolStrEqIcList(ctx *BoolStrEqIcListContext) interface{} {
	e.emitStrEqList(ctx.Strexpr(), collectBlistStrexprs(ctx.BlistIc()), "s==i")
	return nil
}

func (e *PostfixEmitter) VisitBoolStrNeqIc(ctx *BoolStrNeqIcContext) interface{} {
	e.Visit(ctx.Strexpr(0))
	e.Visit(ctx.Strexpr(1))
	// `sic==` was never registered; runtime has `s==i` (#835).
	e.emit("s==i")
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

// VisitBoolFirstPass: `first pass` (#764) — true on the first iteration
// of the innermost active loop in the table's context. Lowers to the
// `firstpass` postfix op, which queries the runtime's loop-iteration
// stack. With no active loop the op pushes false.
func (e *PostfixEmitter) VisitBoolFirstPass(ctx *BoolFirstPassContext) interface{} {
	e.emit("firstpass")
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

	// Numeric names: the grammar routes `field == field` to this nexpr
	// visitor regardless of type, so fixed/bigint/integer comparisons used
	// to fall through to `streq` — correct only for values whose decimal
	// string forms happen to match. Dispatch to the proper family with
	// Fixed > BigInt > Integer promotion when both sides are numeric.
	if t0, t1 := e.identNumericType(name0), e.identNumericType(name1); t0 != "" && t1 != "" {
		target := e.promote(t0, t1)
		e.Visit(ctx.Nexpr(0))
		e.emitTypeCast(t0, target)
		e.Visit(ctx.Nexpr(1))
		e.emitTypeCast(t1, target)
		e.emit(arithOp(target, "==", "b==", "f==", "fp=="))
		return nil
	}

	// The numeric block above excludes double, so fixed/bigint == double would
	// otherwise fall through to a meaningless string compare. Reject the
	// implicit mix the same way the arithmetic path does (#876).
	if t0, t1 := e.lookupType(name0), e.lookupType(name1); isDoubleExactMix(t0, t1) {
		e.emitDoubleMixError(exactOf(t0, t1))
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

	// Numeric names: dispatch to the proper inequality family. Same
	// Fixed > BigInt > Integer promotion as BoolNameEq; fp!= and b!=
	// are distinct ops, integer falls back to the historic `== not`.
	if t0, t1 := e.identNumericType(name0), e.identNumericType(name1); t0 != "" && t1 != "" {
		target := e.promote(t0, t1)
		e.Visit(ctx.Nexpr(0))
		e.emitTypeCast(t0, target)
		e.Visit(ctx.Nexpr(1))
		e.emitTypeCast(t1, target)
		switch target {
		case TypeFixed:
			e.emit("fp!=")
		case TypeBigInt:
			e.emit("b!=")
		default:
			e.emit("==")
			e.emit("not")
		}
		return nil
	}

	// Reject an implicit fixed/bigint != double the same way == does (#876).
	if t0, t1 := e.lookupType(name0), e.lookupType(name1); isDoubleExactMix(t0, t1) {
		e.emitDoubleMixError(exactOf(t0, t1))
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

	target := e.promote(e.getExprType(left), e.getExprType(right))
	e.emitWithTypeConversion(left, target)
	e.emitWithTypeConversion(right, target)
	e.emit(arithOp(target, "+", "b+", "f+", "fp+"))
	return nil
}

func (e *PostfixEmitter) VisitIntSub(ctx *IntSubContext) interface{} {
	left, right := ctx.Iexpr(0), ctx.Iexpr(1)
	target := e.promote(e.getExprType(left), e.getExprType(right))
	e.emitWithTypeConversion(left, target)
	e.emitWithTypeConversion(right, target)
	e.emit(arithOp(target, "-", "b-", "f-", "fp-"))
	return nil
}

func (e *PostfixEmitter) VisitIntMul(ctx *IntMulContext) interface{} {
	left, right := ctx.Iexpr(0), ctx.Iexpr(1)
	target := e.promote(e.getExprType(left), e.getExprType(right))
	e.emitWithTypeConversion(left, target)
	e.emitWithTypeConversion(right, target)
	e.emit(arithOp(target, "*", "b*", "fmul", "fp*"))
	return nil
}

func (e *PostfixEmitter) VisitIntDiv(ctx *IntDivContext) interface{} {
	left, right := ctx.Iexpr(0), ctx.Iexpr(1)
	target := e.promote(e.getExprType(left), e.getExprType(right))
	e.emitWithTypeConversion(left, target)
	e.emitWithTypeConversion(right, target)
	e.emit(arithOp(target, "/", "b/", "fdiv", "fp/"))
	return nil
}

func (e *PostfixEmitter) VisitIntNegate(ctx *IntNegateContext) interface{} {
	expr := ctx.Iexpr()
	exprType := e.getExprType(expr)
	e.Visit(expr)
	switch exprType {
	case TypeFixed:
		e.emit("fpnegate")
	case TypeBigInt:
		e.emit("bnegate")
	case TypeDouble:
		e.emit("fnegate") // negate would truncate the double via IntValue (#894)
	default:
		e.emit("negate")
	}
	return nil
}

func (e *PostfixEmitter) VisitIntParen(ctx *IntParenContext) interface{} {
	e.Visit(ctx.Iexpr())
	return nil
}

// VisitIntNumberOf: `number of <arrayExpr>`. Uses the same runtime op as
// `length of <arrayExpr>` — `length` — since there's no separate
// `numberof` operator registered. Previously emitted `numberof`, which
// resolved to an executable-name lookup and failed at runtime.
func (e *PostfixEmitter) VisitIntNumberOf(ctx *IntNumberOfContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.emit("length")
	return nil
}

// VisitIntNumberOfWhere: `number of <arrayExpr> where <bexpr>` — count
// elements matching bexpr. Emit a count-accumulator fold:
//
//	0 { { 1 + } <bexpr> if } <arrayExpr> forall
//
// Two stack-discipline rules govern the operand order (both verified by the
// predicated-fold execution test, and both matched by the working
// VisitForallWhere template):
//
//  1. forall is ( body array -- ): opForall pops the ARRAY off the TOP, then
//     the body block. So the array is emitted LAST, after the body — emitting
//     it before leaves the body block on top and forall iterates the block's
//     tokens ("non-Entity entry in array").
//  2. if is ( body boolean -- ): it pops the boolean off the top, then the
//     body. So the inner block goes BEFORE the predicate: `{ 1 + } bexpr if`.
//
// The seed 0 stays on the data stack below the body and array, untouched by
// forall's two pops, and the body accumulates into it per element.
func (e *PostfixEmitter) VisitIntNumberOfWhere(ctx *IntNumberOfWhereContext) interface{} {
	e.emit("0")
	e.emit("{")
	e.emit("{")
	e.emit("1")
	e.emit("+")
	e.emit("}")
	e.Visit(ctx.Bexpr())
	e.emit("if")
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forall")
	return nil
}

// VisitIntSumOf: `sum of <iexpr> in <arrayExpr>` — fold the array
// accumulating <iexpr> per element. Element entity is auto-pushed by
// forall, so <iexpr> may reference the element's fields directly. Emits
// `0 { <iexpr> + } <arrayExpr> forall` — body before array, since opForall
// pops the array off the top (see VisitIntNumberOfWhere for the full
// stack-discipline note).
func (e *PostfixEmitter) VisitIntSumOf(ctx *IntSumOfContext) interface{} {
	e.emit("0")
	e.emit("{")
	e.Visit(ctx.Iexpr())
	e.emit("+")
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forall")
	return nil
}

// VisitIntSumOfWhere: `sum of <iexpr> in <arrayExpr> where <bexpr>` —
// predicated sum, the parity fill for `number of … where`. Same fold as
// IntSumOf, but each element only contributes <iexpr> when <bexpr> holds.
// The element entity is auto-pushed by forall, so both <bexpr> and
// <iexpr> may reference the element's fields directly.
func (e *PostfixEmitter) VisitIntSumOfWhere(ctx *IntSumOfWhereContext) interface{} {
	// Emits `0 { { <iexpr> + } <bexpr> if } <arrayExpr> forall` — body before
	// array (opForall pops the array off the top) and inner block before the
	// predicate (`if` pops the boolean off the top). Same operand-order
	// discipline as the plain folds (#867) and the VisitForallWhere template.
	e.emit("0")
	e.emit("{")
	e.emit("{")
	e.Visit(ctx.Iexpr())
	e.emit("+")
	e.emit("}")
	e.Visit(ctx.Bexpr())
	e.emit("if")
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forall")
	return nil
}

// VisitIntIndexOf: `index of <needle> in <haystack>` — emit
// `<haystack> <needle> indexof`. opIndexOf pops substring then
// string. Pre-fix the rule silently emitted nothing.
func (e *PostfixEmitter) VisitIntIndexOf(ctx *IntIndexOfContext) interface{} {
	all := ctx.AllStrexpr()
	if len(all) != 2 {
		return nil
	}
	e.Visit(all[1]) // haystack
	e.Visit(all[0]) // needle
	e.emit("indexof")
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

// minMaxOp picks the correct op name for a numeric min/max dispatch.
// The fp path uses the dedicated fpmin/fpmax ops so precision is kept
// on the 10⁻⁸ grid. The double path uses fmin/fmax so RDouble operands
// don't hit IntValue at runtime — closes the same dispatch gap
// promoteArithType's Double arm opened up.
//
// The bigint path uses the dedicated bmin/bmax ops (compare via big.Int) so a
// bigint exceeding int64 range is not truncated by the integer min/max's
// IntValue() (#899).
func minMaxOp(target, intOp, bigOp, dblOp, fpOp string) string {
	switch target {
	case TypeFixed:
		return fpOp
	case TypeBigInt:
		return bigOp
	case TypeDouble:
		return dblOp
	default:
		return intOp
	}
}

func (e *PostfixEmitter) VisitIntMinOf(ctx *IntMinOfContext) interface{} {
	l, r := ctx.Iexpr(0), ctx.Iexpr(1)
	target := e.promote(e.getExprType(l), e.getExprType(r))
	e.emitWithTypeConversion(l, target)
	e.emitWithTypeConversion(r, target)
	e.emit(minMaxOp(target, "min", "bmin", "fmin", "fpmin"))
	return nil
}

func (e *PostfixEmitter) VisitIntMinOfComma(ctx *IntMinOfCommaContext) interface{} {
	l, r := ctx.Iexpr(0), ctx.Iexpr(1)
	target := e.promote(e.getExprType(l), e.getExprType(r))
	e.emitWithTypeConversion(l, target)
	e.emitWithTypeConversion(r, target)
	e.emit(minMaxOp(target, "min", "bmin", "fmin", "fpmin"))
	return nil
}

func (e *PostfixEmitter) VisitIntMaxOf(ctx *IntMaxOfContext) interface{} {
	l, r := ctx.Iexpr(0), ctx.Iexpr(1)
	target := e.promote(e.getExprType(l), e.getExprType(r))
	e.emitWithTypeConversion(l, target)
	e.emitWithTypeConversion(r, target)
	e.emit(minMaxOp(target, "max", "bmax", "fmax", "fpmax"))
	return nil
}

func (e *PostfixEmitter) VisitIntMaxOfComma(ctx *IntMaxOfCommaContext) interface{} {
	l, r := ctx.Iexpr(0), ctx.Iexpr(1)
	target := e.promote(e.getExprType(l), e.getExprType(r))
	e.emitWithTypeConversion(l, target)
	e.emitWithTypeConversion(r, target)
	e.emit(minMaxOp(target, "max", "bmax", "fmax", "fpmax"))
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

// emitMixedFloatArith handles +/- where at least one operand is a float
// expression (fexpr — always double-typed). It promotes through e.promote so
// the float forces a double result (f+/f-) with any integer operand promoted
// by the op, and a fixed/bigint operand mixed with the float is rejected
// (#876) rather than silently truncated. Without it these visitors emitted a
// bare integer `+`/`-` that truncated e.g. `db2 + 1.0` at runtime (#884).
// Pass TypeDouble for an fexpr operand (getExprType only types iexprs).
func (e *PostfixEmitter) emitMixedFloatArith(left, right antlr.ParseTree, leftType, rightType, intOp, bigOp, dblOp, fpOp string) {
	target := e.promote(leftType, rightType)
	e.emitWithTypeConversion(left, target)
	e.emitWithTypeConversion(right, target)
	e.emit(arithOp(target, intOp, bigOp, dblOp, fpOp))
}

func (e *PostfixEmitter) VisitFloatAddFloat(ctx *FloatAddFloatContext) interface{} {
	e.emitMixedFloatArith(ctx.Fexpr(0), ctx.Fexpr(1), e.getExprType(ctx.Fexpr(0)), e.getExprType(ctx.Fexpr(1)), "+", "b+", "f+", "fp+")
	return nil
}

func (e *PostfixEmitter) VisitFloatSubFloat(ctx *FloatSubFloatContext) interface{} {
	e.emitMixedFloatArith(ctx.Fexpr(0), ctx.Fexpr(1), e.getExprType(ctx.Fexpr(0)), e.getExprType(ctx.Fexpr(1)), "-", "b-", "f-", "fp-")
	return nil
}

// Mul/div route through emitMixedFloatArith exactly like the fexpr add/sub
// visitors: a fixed field that matched the grammar's fexpr alternative (e.g.
// the dividend of `divide … rounding by`) must dispatch to fp*/fp/, not the
// double ops — staking mantissas exceed a double's exact-integer range, so an
// unconditional fmul silently loses precision (#903, same class as #874/#884).
func (e *PostfixEmitter) VisitFloatMulFloat(ctx *FloatMulFloatContext) interface{} {
	e.emitMixedFloatArith(ctx.Fexpr(0), ctx.Fexpr(1), e.getExprType(ctx.Fexpr(0)), e.getExprType(ctx.Fexpr(1)), "*", "b*", "fmul", "fp*")
	return nil
}

func (e *PostfixEmitter) VisitFloatDivFloat(ctx *FloatDivFloatContext) interface{} {
	e.emitMixedFloatArith(ctx.Fexpr(0), ctx.Fexpr(1), e.getExprType(ctx.Fexpr(0)), e.getExprType(ctx.Fexpr(1)), "/", "b/", "fdiv", "fp/")
	return nil
}

func (e *PostfixEmitter) VisitFloatAddInt(ctx *FloatAddIntContext) interface{} {
	e.emitMixedFloatArith(ctx.Fexpr(), ctx.Iexpr(), e.getExprType(ctx.Fexpr()), e.getExprType(ctx.Iexpr()), "+", "b+", "f+", "fp+")
	return nil
}

func (e *PostfixEmitter) VisitFloatSubInt(ctx *FloatSubIntContext) interface{} {
	e.emitMixedFloatArith(ctx.Fexpr(), ctx.Iexpr(), e.getExprType(ctx.Fexpr()), e.getExprType(ctx.Iexpr()), "-", "b-", "f-", "fp-")
	return nil
}

func (e *PostfixEmitter) VisitFloatMulInt(ctx *FloatMulIntContext) interface{} {
	e.emitMixedFloatArith(ctx.Fexpr(), ctx.Iexpr(), e.getExprType(ctx.Fexpr()), e.getExprType(ctx.Iexpr()), "*", "b*", "fmul", "fp*")
	return nil
}

func (e *PostfixEmitter) VisitFloatDivInt(ctx *FloatDivIntContext) interface{} {
	e.emitMixedFloatArith(ctx.Fexpr(), ctx.Iexpr(), e.getExprType(ctx.Fexpr()), e.getExprType(ctx.Iexpr()), "/", "b/", "fdiv", "fp/")
	return nil
}

// VisitDivideRoundingBy: `divide <a> by <b> rounding by <fpLit>` (#801).
// The rounding fraction must be a literal `FP_LITERAL` token in [0, 1).
// We fold at compile time:
//
//	r == 0   → emit `<a> <b> fp/`
//	r == 0.5 → emit `<a> <b> fphalfup/`
//	else     → emit `<a> <b> <rLit> fpdivr/`
//
// r outside [0, 1) is a compile error — the grammar requires a literal so
// we can range-check here rather than at runtime. The non-literal-R case
// is currently impossible by grammar; if the rule is ever relaxed, the
// emitter falls through to the ternary path with the visited expression.
func (e *PostfixEmitter) VisitDivideRoundingBy(ctx *DivideRoundingByContext) interface{} {
	// The fp-family divide ops require fixed operands. Integer/bigint
	// operands are promoted via cvfp; a double operand is rejected per the
	// #876 policy (the runtime will not promote double→fixed implicitly, and
	// staking mantissas exceed a double's exact-integer range) — authors opt
	// in with an explicit `(fixed)` cast. (#903)
	for i := 0; i < 2; i++ {
		if t := e.getExprType(ctx.Fexpr(i)); t == TypeDouble {
			e.emitError("divide … rounding by requires fixed operands; " +
				"cast the double operand explicitly (e.g. \"(fixed) x\")")
			return nil
		}
	}
	// Emit both operands left-associated. Everywhere else in the language a
	// bare `x * y * z` groups left, but the two operands of this rule parse
	// right-nested: `iexpr TIMES fexpr` is not a left-recursive alternative of
	// fexpr, so ANTLR tries it as a primary here and its right operand
	// swallows the rest of the chain.
	//
	// That matters because fp* rounds. Regrouping a product moves the rounding
	// point, so the two forms are not interchangeable: on the Accumulate
	// staking rules the right-nested form shifts a payout by 1 nanoACME and
	// breaks their on-chain period reproduction. Their committed postfix is
	// left-associated, compiled before this regressed (#1015).
	//
	// Only unparenthesised chains are re-associated — a parenthesised
	// subexpression is its own node and is emitted as the author grouped it.
	e.emitFixedProductLeftAssoc(ctx.Fexpr(0))
	e.emitFixedProductLeftAssoc(ctx.Fexpr(1))

	rText := ctx.FP_LITERAL().GetText()
	rMantissa, err := parseFpLiteralToMantissa(rText)
	if err != nil {
		e.emitError("rounding fraction %q: %v", rText, err)
		return nil
	}
	// Range check: 0 <= r < 1.
	if rMantissa.Sign() < 0 {
		e.emitError("rounding fraction must be >= 0, got %q", rText)
		return nil
	}
	scale := new(big.Int).SetInt64(100_000_000) // 10^8
	if rMantissa.Cmp(scale) >= 0 {
		e.emitError("rounding fraction must be < 1.0, got %q", rText)
		return nil
	}

	// Fold table.
	switch {
	case rMantissa.Sign() == 0:
		e.emit("fp/")
	case rMantissa.Cmp(new(big.Int).SetInt64(50_000_000)) == 0:
		e.emit("fphalfup/")
	default:
		e.emit(rText)
		e.emit("fpdivr/")
	}
	return nil
}

// parseFpLiteralToMantissa parses an FP_LITERAL token (e.g. "0.5fp",
// "0fp", "0.99999999fp") to its mantissa scaled by 10^8. Returns an
// error if the text isn't a valid fp literal (the lexer already
// guarantees the shape; this is defense-in-depth).
//
// Sign is not part of FP_LITERAL grammar (it's handled by the unary
// minus rule), so the returned mantissa is always nonneg here. The
// fold caller treats sign separately if a future grammar extension
// allows it.
func parseFpLiteralToMantissa(text string) (*big.Int, error) {
	if len(text) < 3 {
		return nil, fmt.Errorf("too short to be an fp literal")
	}
	if !strings.EqualFold(text[len(text)-2:], "fp") {
		return nil, fmt.Errorf("missing fp suffix")
	}
	body := text[:len(text)-2]
	if body == "" {
		return nil, fmt.Errorf("empty fp literal body")
	}

	whole, frac := body, ""
	if dot := strings.IndexByte(body, '.'); dot >= 0 {
		whole = body[:dot]
		frac = body[dot+1:]
	}
	if whole == "" {
		whole = "0"
	}

	// Pad / truncate frac to 8 digits.
	const fixedDecimals = 8
	if len(frac) > fixedDecimals {
		frac = frac[:fixedDecimals]
	} else if len(frac) < fixedDecimals {
		frac += strings.Repeat("0", fixedDecimals-len(frac))
	}

	wholeBig, ok := new(big.Int).SetString(whole, 10)
	if !ok {
		return nil, fmt.Errorf("invalid whole part %q", whole)
	}
	fracBig, ok := new(big.Int).SetString(frac, 10)
	if !ok {
		return nil, fmt.Errorf("invalid fractional part %q", frac)
	}

	scale := new(big.Int).SetInt64(100_000_000)
	mantissa := new(big.Int).Mul(wholeBig, scale)
	mantissa.Add(mantissa, fracBig)
	return mantissa, nil
}

// =============================================================================
// #803 — silent-failure visitors that were inherited from BaseELVisitor and
// silently produced empty postfix because antlr's BaseParseTreeVisitor
// VisitChildren is a no-op. Each of the alternatives below has a
// reproducer that confirmed empty / op-dropping output.
// =============================================================================

// VisitIntMulBy: `multiply <ident> by <number>` — prefix multiplication.
// The grammar has both intMulBy (in iexpr) and floatMulBy (in fexpr),
// but typedLong and typedDouble both lex as IDENT so the parser can't
// disambiguate; intMulBy usually wins. Dispatch by declared field type
// at compile time so int/double/fixed/bigint fields all get the right
// op — same pattern VisitIncrementLong uses for the same reason.
func (e *PostfixEmitter) VisitIntMulBy(ctx *IntMulByContext) interface{} {
	emitMulDivBy(e, ctx.TypedLong(), ctx.Number(), "*", "b*", "fmul", "fp*")
	return nil
}

// VisitIntDivBy: `divide <ident> by <number>` — sister of IntMulBy.
func (e *PostfixEmitter) VisitIntDivBy(ctx *IntDivByContext) interface{} {
	emitMulDivBy(e, ctx.TypedLong(), ctx.Number(), "/", "b/", "fdiv", "fp/")
	return nil
}

// VisitFloatMulBy: rarely reached (intMulBy wins parser-side), but
// when it is — declared type still drives the op choice. We can't
// assume the IDENT is double-typed just because the alt's label says
// "Float".
func (e *PostfixEmitter) VisitFloatMulBy(ctx *FloatMulByContext) interface{} {
	emitMulDivBy(e, ctx.TypedDouble(), ctx.Number(), "*", "b*", "fmul", "fp*")
	return nil
}

// VisitFloatDivBy: same rare-reach + type-aware shape.
func (e *PostfixEmitter) VisitFloatDivBy(ctx *FloatDivByContext) interface{} {
	emitMulDivBy(e, ctx.TypedDouble(), ctx.Number(), "/", "b/", "fdiv", "fp/")
	return nil
}

// emitMulDivBy is the shared emission helper for `multiply <ident> by
// <number>` and `divide <ident> by <number>`. The field's declared type
// drives op choice; we promote only the RHS number to match the field's
// type (the field's visitor emits it correctly-typed already, so casting
// the LHS would just add a redundant cvfp/cvbi). This produces postfix
// identical to the canonical `<field> * <number>` form (verified by
// regression tests).
//
// Without this helper, both prefix forms silently dropped the op
// because antlr's BaseParseTreeVisitor.VisitChildren is a no-op (#803).
func emitMulDivBy(e *PostfixEmitter, lhs, rhs antlr.ParseTree, intOp, bigOp, dblOp, fpOp string) {
	name := lhs.(interface{ GetText() string }).GetText()
	target := e.lookupType(name)
	if target == "" {
		target = TypeInteger
	}
	// Reject `multiply/divide <fixed|bigint> by <double>` rather than feed the
	// exact-type op an un-cast double (a runtime promote error) or silently
	// snap it — same policy as the binary-op reject (#876/#882).
	if rt := e.getExprType(rhs); isDoubleExactMix(rt, target) {
		e.emitDoubleMixError(exactOf(rt, target))
		return
	}
	e.Visit(lhs)
	e.emitWithTypeConversion(rhs, target)
	e.emit(arithOp(target, intOp, bigOp, dblOp, fpOp))
}

// VisitStrSubstring: `substring of <strexpr> from <iexpr> to <iexpr>`.
// Lowers to the registered `substring` op with arity (str start end --
// result). Without this override the entire substring call silently
// disappeared from the postfix.
func (e *PostfixEmitter) VisitStrSubstring(ctx *StrSubstringContext) interface{} {
	// Grammar is `from <start> to <end>` (end exclusive, by character index),
	// but opSubstring is ( str start length -- ). Compute length = end - start
	// instead of passing the end index as the length — the latter was correct
	// only when start == 0 (where end == length) and silently wrong otherwise
	// (#889). Stack: str start end -> over (copy start back) -> `-` = end-start.
	e.Visit(ctx.Strexpr())
	e.Visit(ctx.Iexpr(0)) // start
	e.Visit(ctx.Iexpr(1)) // end
	e.emit("over")
	e.emit("-")
	e.emit("substring")
	return nil
}

// VisitFloatFromStr: `(double) <strexpr>` — explicit string→double
// cast. Without this, the entire RHS was dropped — the assignment
// trailer `cvd /x xdef` ran on an empty stack.
func (e *PostfixEmitter) VisitFloatFromStr(ctx *FloatFromStrContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("cvd")
	return nil
}

// VisitFloatFromInt: `(double) <iexpr>` — explicit int→double cast.
// Same silent-drop shape as FloatFromStr.
func (e *PostfixEmitter) VisitFloatFromInt(ctx *FloatFromIntContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.emit("cvd")
	return nil
}

// VisitFloatFromIndex: `(double) <indxExpr>` — explicit cast from an
// indexed expression (array element / dict lookup result).
func (e *PostfixEmitter) VisitFloatFromIndex(ctx *FloatFromIndexContext) interface{} {
	e.Visit(ctx.IndxExpr())
	e.emit("cvd")
	return nil
}

// VisitIntFromStr: `(int|long) <strexpr>` — explicit string→int cast.
func (e *PostfixEmitter) VisitIntFromStr(ctx *IntFromStrContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("cvi")
	return nil
}

// VisitIntFromNumber: `(int|long) <number>` — explicit cast from a
// numeric expression (which itself may be int or float).
func (e *PostfixEmitter) VisitIntFromNumber(ctx *IntFromNumberContext) interface{} {
	e.Visit(ctx.Number())
	e.emit("cvi")
	return nil
}

// VisitIntFromIndex: `(int|long) <indxExpr>` — sister of FloatFromIndex
// for the int target.
func (e *PostfixEmitter) VisitIntFromIndex(ctx *IntFromIndexContext) interface{} {
	e.Visit(ctx.IndxExpr())
	e.emit("cvi")
	return nil
}

// =============================================================================
// #803 batch 2: date arithmetic in dexpr position. The statement forms
// (e.g. `add 3 days to D` as a free-standing action) have visitors and
// work; the *expression* forms (used inside `set X = ...`) had no
// visitors and produced empty postfix.
//
// All registered date ops are unary-with-N: ( date number -- date' ).
// For SUBTRACT, no subdays/submonths/subyears ops are registered (the
// dateMinus* visitors emit them anyway but that's a separate bug); we
// emit `negate adddays` etc. to stay on the verified path.
// =============================================================================

// VisitDateExprAddYears: `add <number> years to <dexpr>` as an
// expression. Statement form via VisitDateAddYears already worked.
func (e *PostfixEmitter) VisitDateExprAddYears(ctx *DateExprAddYearsContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Number())
	e.emit("addyears")
	return nil
}

// VisitDateExprAddMonths: same shape for months.
func (e *PostfixEmitter) VisitDateExprAddMonths(ctx *DateExprAddMonthsContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Number())
	e.emit("addmonths")
	return nil
}

// VisitDateExprAddDays: same shape for days.
func (e *PostfixEmitter) VisitDateExprAddDays(ctx *DateExprAddDaysContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Number())
	e.emit("adddays")
	return nil
}

// VisitDateExprSubYears: `subtract <number> years from <dexpr>` as an
// expression. There's no subyears op; we mirror the statement form's
// `<num> negate addyears` pattern.
func (e *PostfixEmitter) VisitDateExprSubYears(ctx *DateExprSubYearsContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Number())
	e.emit("negate")
	e.emit("addyears")
	return nil
}

// VisitDateExprSubMonths: same shape for months.
func (e *PostfixEmitter) VisitDateExprSubMonths(ctx *DateExprSubMonthsContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Number())
	e.emit("negate")
	e.emit("addmonths")
	return nil
}

// VisitDateExprSubDays: same shape for days.
func (e *PostfixEmitter) VisitDateExprSubDays(ctx *DateExprSubDaysContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Number())
	e.emit("negate")
	e.emit("adddays")
	return nil
}

// VisitDateFirstOfMonth: `first of months of <dexpr>` — start-of-month
// date constructor. The in-zone variant has its own visitor that runs
// for the longer-match case; this is the plain form.
func (e *PostfixEmitter) VisitDateFirstOfMonth(ctx *DateFirstOfMonthContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.emit("firstofmonth")
	return nil
}

// VisitDateFirstOfYear: `first of years of <dexpr>` — start-of-year.
func (e *PostfixEmitter) VisitDateFirstOfYear(ctx *DateFirstOfYearContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.emit("firstofyear")
	return nil
}

// VisitDateEndOfMonth: `end of months of <dexpr>` — last day of month.
func (e *PostfixEmitter) VisitDateEndOfMonth(ctx *DateEndOfMonthContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.emit("endofmonth")
	return nil
}

func (e *PostfixEmitter) VisitIntAddFloat(ctx *IntAddFloatContext) interface{} {
	e.emitMixedFloatArith(ctx.Iexpr(), ctx.Fexpr(), e.getExprType(ctx.Iexpr()), e.getExprType(ctx.Fexpr()), "+", "b+", "f+", "fp+")
	return nil
}

func (e *PostfixEmitter) VisitIntSubFloat(ctx *IntSubFloatContext) interface{} {
	e.emitMixedFloatArith(ctx.Iexpr(), ctx.Fexpr(), e.getExprType(ctx.Iexpr()), e.getExprType(ctx.Fexpr()), "-", "b-", "f-", "fp-")
	return nil
}

func (e *PostfixEmitter) VisitIntMulFloat(ctx *IntMulFloatContext) interface{} {
	e.emitMixedFloatArith(ctx.Iexpr(), ctx.Fexpr(), e.getExprType(ctx.Iexpr()), e.getExprType(ctx.Fexpr()), "*", "b*", "fmul", "fp*")
	return nil
}

func (e *PostfixEmitter) VisitIntDivFloat(ctx *IntDivFloatContext) interface{} {
	e.emitMixedFloatArith(ctx.Iexpr(), ctx.Fexpr(), e.getExprType(ctx.Iexpr()), e.getExprType(ctx.Fexpr()), "/", "b/", "fdiv", "fp/")
	return nil
}

func (e *PostfixEmitter) VisitFloatNegate(ctx *FloatNegateContext) interface{} {
	e.Visit(ctx.Fexpr())
	// fnegate (opFNegate) negates via DoubleValue; the previously emitted
	// `neg` was never registered, so any float-expr negation crashed at
	// runtime with operator-not-found. `negate` would truncate the double
	// via IntValue, so it is not a substitute here. (#878)
	e.emit("fnegate")
	return nil
}

func (e *PostfixEmitter) VisitFloatParen(ctx *FloatParenContext) interface{} {
	e.Visit(ctx.Fexpr())
	return nil
}

// VisitFloatAbs / VisitIntAbs: `absolute value of <expr>`. The grammar has
// three separate labeled alternatives (floatAbs / intAbs / bigAbs); before
// this commit only bigAbs had a visitor, so intAbs and floatAbs fell
// through to the default child visit — which emitted nothing at all,
// leaving the enclosing `set` with an empty RHS.
//
// For intAbs with a declared-fp field we also want fpabs rather than
// the plain `abs` op, so the int path looks at the resolved type. For
// floatAbs we similarly look at the inner text to catch fp literals;
// typical fp fields route through intAbs (typedLong wins), not here.
func (e *PostfixEmitter) VisitIntAbs(ctx *IntAbsContext) interface{} {
	inner := ctx.Iexpr()
	e.Visit(inner)
	switch e.getExprType(inner) {
	case TypeFixed:
		e.emit("fpabs")
	case TypeBigInt:
		e.emit("babs")
	case TypeDouble:
		e.emit("fabs")
	default:
		e.emit("abs")
	}
	return nil
}

func (e *PostfixEmitter) VisitFloatAbs(ctx *FloatAbsContext) interface{} {
	inner := ctx.Fexpr()
	e.Visit(inner)
	// fp operands reach FloatAbs when a fp-declared field is referenced
	// inside a fexpr context (grammar picks typedDouble over typedLong
	// depending on surrounding rules). Emit fpabs so the value stays on
	// the 10⁻⁸ grid; otherwise the fabs op coerces to float64 first and
	// silently loses fractional precision.
	if fexprIsFixed(e, inner) {
		e.emit("fpabs")
		return nil
	}
	e.emit("fabs")
	return nil
}

// VisitFloatCeilingOf / VisitFloatFloorOf: `ceiling of <expr>` and
// `floor of <expr>`. The runtime ops coerce to double and return the
// rounded double, so both the fexpr and iexpr alternatives emit the
// same op after visiting the operand.
func (e *PostfixEmitter) VisitFloatCeilingOf(ctx *FloatCeilingOfContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.emit("ceiling")
	return nil
}

func (e *PostfixEmitter) VisitFloatCeilingOfInt(ctx *FloatCeilingOfIntContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.emit("ceiling")
	return nil
}

func (e *PostfixEmitter) VisitFloatFloorOf(ctx *FloatFloorOfContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.emit("floor")
	return nil
}

func (e *PostfixEmitter) VisitFloatFloorOfInt(ctx *FloatFloorOfIntContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.emit("floor")
	return nil
}

// fexprIsFixed reports whether a fexpr context resolves to an fp-typed
// value — either a literal with `fp` suffix or a name that maps to
// TypeFixed in the symbol / local table.
func fexprIsFixed(e *PostfixEmitter, ctx interface{ GetText() string }) bool {
	if ctx == nil {
		return false
	}
	text := ctx.GetText()
	if isFixedLiteralText(text) {
		return true
	}
	if lv, ok := e.lookupLocal(text); ok && lv.Type == TypeFixed {
		return true
	}
	return e.lookupType(text) == TypeFixed
}

// isFixedLiteralText reports whether text is a valid fp literal like
// "1.5fp", "0fp", or "100.0FP" — digit sequence (optional leading sign,
// optional single dot) followed by a case-insensitive "fp" suffix.
// Restored after #699 cleanup left a dangling caller in fexprIsFixed.
func isFixedLiteralText(text string) bool {
	if len(text) < 3 {
		return false
	}
	if !strings.EqualFold(text[len(text)-2:], "fp") {
		return false
	}
	body := text[:len(text)-2]
	if body == "" {
		return false
	}
	if body[0] == '-' || body[0] == '+' {
		body = body[1:]
	}
	if body == "" {
		return false
	}
	seenDot := false
	seenDigit := false
	for i := 0; i < len(body); i++ {
		switch c := body[i]; {
		case c >= '0' && c <= '9':
			seenDigit = true
		case c == '.':
			if seenDot {
				return false
			}
			seenDot = true
		default:
			return false
		}
	}
	return seenDigit
}

// VisitFloatRounded / VisitFloatRoundedTo / VisitFloatRoundedBoundry:
// the three `<fexpr> rounded [...]` grammar alternatives. The pre-fix
// state was:
//   - VisitFloatRounded emitted `fexpr round` — but `round` was never a
//     registered operator, so every `X rounded` rule failed at runtime.
//   - VisitFloatRoundedTo and VisitFloatRoundedBoundry had no visitors
//     at all, so the default child visit produced empty postfix.
//
// All three now route through the registered `roundto` operator which
// takes `(number places boundary -- result)`. Defaults: 0 decimal places
// and 0.5 boundary (half-up rounding) when the DSL doesn't specify.
// VisitFloatRounded: `<fexpr> rounded`. For a double operand, route
// through the registered `roundto` op with default places=0 and
// boundary=0.5 (half-up). For an fp operand, `rounded` without an
// explicit `to N decimal places` clause is "truncate the fractional
// part" — emit `fptrunc` so the result stays exactly on the 10⁻⁸
// grid instead of coercing through float64.
func (e *PostfixEmitter) VisitFloatRounded(ctx *FloatRoundedContext) interface{} {
	e.Visit(ctx.Fexpr())
	if fexprIsFixed(e, ctx.Fexpr()) {
		e.emit("fptrunc")
		return nil
	}
	e.emit("0")
	e.emit("0.5")
	e.emit("roundto")
	return nil
}

// VisitFloatRoundedTo: `<fexpr> rounded to N decimal places`. No fp-
// equivalent operator exists for rounding to N fractional places, so
// fp operands are rejected with a clear runtime error pointing to
// the explicit `(double)` cast. Rule authors who truly need fp
// rounding to N places should cast the fp value first (accepting the
// precision concession) or wait for a dedicated `fpround` op.
func (e *PostfixEmitter) VisitFloatRoundedTo(ctx *FloatRoundedToContext) interface{} {
	if fexprIsFixed(e, ctx.Fexpr()) {
		e.emit("\"rounded to N decimal places not supported on fixed-point values; use (double) cast if precision loss is acceptable\"")
		e.emit("elstmterror")
		return nil
	}
	e.Visit(ctx.Fexpr())
	e.Visit(ctx.Iexpr())
	e.emit("0.5")
	e.emit("roundto")
	return nil
}

// VisitFloatRoundedBoundry: `<fexpr> rounded to N decimal places with
// boundry B`. Same fp-rejection rationale as VisitFloatRoundedTo.
func (e *PostfixEmitter) VisitFloatRoundedBoundry(ctx *FloatRoundedBoundryContext) interface{} {
	if fexprIsFixed(e, ctx.Fexpr(0)) {
		e.emit("\"rounded to N decimal places with boundry not supported on fixed-point values; use (double) cast if precision loss is acceptable\"")
		e.emit("elstmterror")
		return nil
	}
	e.Visit(ctx.Fexpr(0))
	e.Visit(ctx.Iexpr())
	e.Visit(ctx.Fexpr(1))
	e.emit("roundto")
	return nil
}

// VisitFloatSumOf: `sum of <typedDouble> in <arrayExpr>` — fold the
// array summing the named double field. forall auto-pushes each
// element entity onto the entity stack, so the typedDouble resolves
// against the element via the standard field-access path.
//
// Currently unreachable due to a grammar prediction issue: ANTLR's
// LL(*) picks intSumOf over floatSumOf for `sum of <ident> in
// <array>` because typedLong/typedDouble both lex as bare IDENT, and
// `number : iexpr | fexpr` lists iexpr first. The override is kept
// defensively for when a future token-classification pass makes
// floatSumOf reachable. See TestIssue803_FloatSumOf_Unreachable.
func (e *PostfixEmitter) VisitFloatSumOf(ctx *FloatSumOfContext) interface{} {
	e.emit("0.0")
	e.emit("{")
	e.Visit(ctx.TypedDouble())
	e.emit("f+")
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forall")
	return nil
}

// VisitFloatSumOfWhere: `sum of <typedDouble> in <arrayExpr> where <bexpr>`
// — the float counterpart of VisitIntSumOfWhere. Each element contributes
// <typedDouble> to the running total only when <bexpr> holds.
func (e *PostfixEmitter) VisitFloatSumOfWhere(ctx *FloatSumOfWhereContext) interface{} {
	// Body before array, inner block before predicate — see
	// VisitIntSumOfWhere.
	e.emit("0.0")
	e.emit("{")
	e.emit("{")
	e.Visit(ctx.TypedDouble())
	e.emit("f+")
	e.emit("}")
	e.Visit(ctx.Bexpr())
	e.emit("if")
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forall")
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
	name := ctx.GetText()
	// Alias field access: `<alias>.<field>` where alias is a local entity
	// slot declared by `for all X as alias` (#712, #714). Must be checked
	// here because the grammar prefers typedXmlValue over typedString for
	// bare IDENTs in strexpr, so `outer_a.label + ";"` routes here.
	if e.emitAliasFieldAccess(name) {
		return nil
	}
	if !e.emitLocalRef(name) {
		e.emit(name)
	}
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
	e.emit("lowercase") // registered op name (#888)
	return nil
}

// VisitStrLowercaseOf / VisitStrUppercaseOf: `lowercase of <s>` /
// `uppercase of <s>` (#904). Before the dedicated tokens, `lowercase of url`
// parsed as relationship traversal (`url lowercase getrelationship`), which
// errors at runtime on string operands — the op existed with no surface.
// Equivalent to the `change <s> to lower/upper case` forms.
func (e *PostfixEmitter) VisitStrLowercaseOf(ctx *StrLowercaseOfContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("lowercase")
	return nil
}

func (e *PostfixEmitter) VisitStrUppercaseOf(ctx *StrUppercaseOfContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("uppercase")
	return nil
}

// VisitStrTimestamp: `get current_timestamp` → `gettimestamp`. The
// runtime op is niladic and pushes the current wall-clock timestamp
// (RFC 3339 string). Pre-fix this rule silently emitted nothing.
func (e *PostfixEmitter) VisitStrTimestamp(ctx *StrTimestampContext) interface{} {
	e.emit("gettimestamp")
	return nil
}

// VisitStrAttrOf: `attribute <strexpr> of <eexpr>` — generic
// attribute lookup by string name on an entity. No dedicated runtime
// op exists (the standard field-access path goes through
// possessive/colon refs with a known typed field). Emit an
// elstmterror placeholder so this fails loudly at runtime instead of
// silently dropping the access. Pre-fix the rule silently emitted
// nothing.
func (e *PostfixEmitter) VisitStrAttrOf(ctx *StrAttrOfContext) interface{} {
	e.emit(`"attribute <name> of <entity> not yet implemented"`)
	e.emit("elstmterror")
	return nil
}

// VisitStrMappingKey: `mappingkey` — niladic keyword that yields the
// current decision-table key inside a mapping operation. The runtime
// has no `mappingkey` op registered (the legacy Java engine had one
// tied to its mapping-table machinery; the Go runtime doesn't ship
// equivalent yet). Emit an elstmterror placeholder; pre-fix this
// rule silently emitted nothing.
func (e *PostfixEmitter) VisitStrMappingKey(ctx *StrMappingKeyContext) interface{} {
	e.emit(`"mappingkey not yet implemented"`)
	e.emit("elstmterror")
	return nil
}

// VisitStrRelationship: `relationship between <e1> and <e2>` — name
// of the relationship that links two entities, used by audit/policy
// rendering. No dedicated runtime op. Emit elstmterror; pre-fix this
// rule silently emitted nothing.
func (e *PostfixEmitter) VisitStrRelationship(ctx *StrRelationshipContext) interface{} {
	e.emit(`"relationship between ... not yet implemented"`)
	e.emit("elstmterror")
	return nil
}

// VisitStrXmlAttr: `<typedXmlValue> : get attribute <strexpr>` —
// read an attribute from an XML-valued field. No XML runtime ops
// registered. Emit elstmterror; pre-fix this rule silently emitted
// nothing.
func (e *PostfixEmitter) VisitStrXmlAttr(ctx *StrXmlAttrContext) interface{} {
	e.emit(`"xml get attribute not yet implemented"`)
	e.emit("elstmterror")
	return nil
}

// VisitXmlvalues: container rule whose body is one of
// strexpr/iexpr/fexpr/dexpr/nexpr. Dispatch to the present child so
// the parent xmlSetAttr/xmlAddAttr rule sees the value on the data
// stack.
//
// Currently unreachable: every xmlvaluestatements alt
// (xmlSetAttr/xmlSetAttrEntity/xmlAddAttr/xmlAddAttrEntity) emits an
// elstmterror placeholder for the whole mutation without visiting
// the RHS xmlvalues — the Go runtime has no XML mutation ops. This
// override is kept defensively for when the XML runtime is wired up.
func (e *PostfixEmitter) VisitXmlvalues(ctx *XmlvaluesContext) interface{} {
	switch {
	case ctx.Strexpr() != nil:
		e.Visit(ctx.Strexpr())
	case ctx.Iexpr() != nil:
		e.Visit(ctx.Iexpr())
	case ctx.Fexpr() != nil:
		e.Visit(ctx.Fexpr())
	case ctx.Dexpr() != nil:
		e.Visit(ctx.Dexpr())
	case ctx.Nexpr() != nil:
		e.Visit(ctx.Nexpr())
	}
	return nil
}

func (e *PostfixEmitter) VisitStrToUpper(ctx *StrToUpperContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("uppercase") // registered op name (#888)
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
	e.emit("today") // registered op name (#888)
	return nil
}

// Phase 2 of #743: explicit timezone visitors. Each pushes its arguments in
// the order the runtime op expects (date — if any — first, then the zone
// strexpr) and emits the *inzone op. Without these the labeled alternatives
// would fall through to VisitChildren, which silently drops the op.

func (e *PostfixEmitter) VisitDateCurrentDateInZone(ctx *DateCurrentDateInZoneContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("currentdateinzone")
	return nil
}

func (e *PostfixEmitter) VisitDateInZone(ctx *DateInZoneContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Strexpr())
	e.emit("dateinzone")
	return nil
}

func (e *PostfixEmitter) VisitDateFirstOfYearInZone(ctx *DateFirstOfYearInZoneContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Strexpr())
	e.emit("firstofyearinzone")
	return nil
}

func (e *PostfixEmitter) VisitDateFirstOfMonthInZone(ctx *DateFirstOfMonthInZoneContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Strexpr())
	e.emit("firstofmonthinzone")
	return nil
}

func (e *PostfixEmitter) VisitDateEndOfMonthInZone(ctx *DateEndOfMonthInZoneContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Strexpr())
	e.emit("endofmonthinzone")
	return nil
}

// Phase 3 of #743: week/quarter/year bucket visitors. The starting-day form
// pushes the day-name string before the zone, so the runtime op pops zone,
// then start-day, then date.

func (e *PostfixEmitter) VisitDateFirstOfWeek(ctx *DateFirstOfWeekContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.emit("firstofweek")
	return nil
}

func (e *PostfixEmitter) VisitDateFirstOfWeekInZone(ctx *DateFirstOfWeekInZoneContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Strexpr())
	e.emit("firstofweekinzone")
	return nil
}

func (e *PostfixEmitter) VisitDateFirstOfWeekStarting(ctx *DateFirstOfWeekStartingContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Strexpr())
	e.emit("firstofweekstarting")
	return nil
}

func (e *PostfixEmitter) VisitDateFirstOfWeekStartingInZone(ctx *DateFirstOfWeekStartingInZoneContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Strexpr(0))
	e.Visit(ctx.Strexpr(1))
	e.emit("firstofweekstartinginzone")
	return nil
}

func (e *PostfixEmitter) VisitDateEndOfWeek(ctx *DateEndOfWeekContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.emit("endofweek")
	return nil
}

func (e *PostfixEmitter) VisitDateEndOfWeekInZone(ctx *DateEndOfWeekInZoneContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Strexpr())
	e.emit("endofweekinzone")
	return nil
}

func (e *PostfixEmitter) VisitDateEndOfWeekStarting(ctx *DateEndOfWeekStartingContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Strexpr())
	e.emit("endofweekstarting")
	return nil
}

func (e *PostfixEmitter) VisitDateEndOfWeekStartingInZone(ctx *DateEndOfWeekStartingInZoneContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Strexpr(0))
	e.Visit(ctx.Strexpr(1))
	e.emit("endofweekstartinginzone")
	return nil
}

func (e *PostfixEmitter) VisitDateFirstOfQuarter(ctx *DateFirstOfQuarterContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.emit("firstofquarter")
	return nil
}

func (e *PostfixEmitter) VisitDateFirstOfQuarterInZone(ctx *DateFirstOfQuarterInZoneContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Strexpr())
	e.emit("firstofquarterinzone")
	return nil
}

func (e *PostfixEmitter) VisitDateEndOfQuarter(ctx *DateEndOfQuarterContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.emit("endofquarter")
	return nil
}

func (e *PostfixEmitter) VisitDateEndOfQuarterInZone(ctx *DateEndOfQuarterInZoneContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Strexpr())
	e.emit("endofquarterinzone")
	return nil
}

func (e *PostfixEmitter) VisitDateEndOfYear(ctx *DateEndOfYearContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.emit("endofyear")
	return nil
}

func (e *PostfixEmitter) VisitDateEndOfYearInZone(ctx *DateEndOfYearInZoneContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Strexpr())
	e.emit("endofyearinzone")
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
	e.emit("entityname") // registered op name (#888)
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

// VisitArrayDeepCopy: `get deepcopy of <array>` → `<array> deepcopy`.
// opDeepCopy clones an array and all its elements (vs. opCopy which
// is shallow). Pre-fix this rule silently emitted nothing.
func (e *PostfixEmitter) VisitArrayDeepCopy(ctx *ArrayDeepCopyContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.emit("deepcopy")
	return nil
}

// VisitArrayDeepCopySimple: `deepcopy of <array>` → `<array> deepcopy`.
// Same semantics as `get deepcopy of`, just the shorter form.
func (e *PostfixEmitter) VisitArrayDeepCopySimple(ctx *ArrayDeepCopySimpleContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.emit("deepcopy")
	return nil
}

// VisitArrayName: `(array) NAME` is a cast of a NAME literal to its
// resolved array value. The NAME token (a `$ident` or `name`-keyword
// literal) names a global/local array variable. Mirror
// VisitNameLiteral's prefix logic and emit a `rlookup` to fetch the
// value bound to the name in the current context.
//
// No `rlookup`-style op is registered, so emit an elstmterror
// placeholder so this fails loudly at runtime until the
// name-resolution semantics are nailed down. Pre-fix the rule
// silently emitted nothing.
func (e *PostfixEmitter) VisitArrayName(ctx *ArrayNameContext) interface{} {
	e.emit(`"(array) NAME not yet implemented"`)
	e.emit("elstmterror")
	return nil
}

// VisitArrayTokenize: `tokenize <strexpr> by <strexpr>` splits a
// string on a delimiter and returns the resulting array of substrings.
// The runtime already has a `split` op with the right shape.
func (e *PostfixEmitter) VisitArrayTokenize(ctx *ArrayTokenizeContext) interface{} {
	all := ctx.AllStrexpr()
	if len(all) != 2 {
		return nil
	}
	e.Visit(all[0]) // source string
	e.Visit(all[1]) // delimiter
	e.emit("split")
	return nil
}

// VisitArrayMap: `map <arrayExpr> through <texpr>` applies a decision
// table to each element, collecting the results into a new array. No
// dedicated map runtime op exists. Emit an elstmterror placeholder so
// this fails loudly at runtime instead of silently. Pre-fix the rule
// silently emitted nothing.
func (e *PostfixEmitter) VisitArrayMap(ctx *ArrayMapContext) interface{} {
	e.emit(`"map ... through ... not yet implemented"`)
	e.emit("elstmterror")
	return nil
}

// =============================================================================
// #803 batch 3: array literal construction.
//
// Pre-fix: `set a.intlist = [1, 2, 3]` produced "" (empty postfix). The
// arrayLit / arrayLiteral / arrayList<Type>(Single)? / setArrayArray
// alts all inherited from BaseELVisitor, whose VisitChildren is a no-op.
//
// Construction pattern: leave the array reference on top of stack and
// addto-append each element. `addto` mutates in place (the array stays
// on the stack via `dup`):
//
//   newarray         // [array]
//   dup 1 addto      // [array]  array → [1]
//   dup 2 addto      // [array]  array → [1, 2]
//   dup 3 addto      // [array]  array → [1, 2, 3]
//
// Then the SET trailer takes the array on top and writes it to the LHS.
// =============================================================================

// emitArrayListAdd pushes the array element onto the stack (after a
// duplicate of the array so the next addto can reuse it) and appends
// via the `addto` op (stack effect: array element -- ).
func (e *PostfixEmitter) emitArrayListAdd(elem antlr.ParseTree) {
	e.emit("dup")
	e.Visit(elem)
	e.emit("addto")
}

// VisitArrayLit: `[ <arrayList> ]` — push a fresh array, then walk the
// element list which recursively addto's each element. The array stays
// on the stack after the last addto so the parent (SET trailer, etc.)
// can consume it.
func (e *PostfixEmitter) VisitArrayLit(ctx *ArrayLitContext) interface{} {
	e.emit("newarray")
	e.Visit(ctx.ArrayList())
	return nil
}

// VisitArrayLiteral: the arrayExpr2 wrapper `arrayLit # arrayLiteral`.
func (e *PostfixEmitter) VisitArrayLiteral(ctx *ArrayLiteralContext) interface{} {
	e.Visit(ctx.ArrayLit())
	return nil
}

// VisitArrayOfValues: `array of values { <arrayList> }` — same emission
// shape as arrayLit. Two surface forms, one runtime sequence.
func (e *PostfixEmitter) VisitArrayOfValues(ctx *ArrayOfValuesContext) interface{} {
	e.emit("newarray")
	e.Visit(ctx.ArrayList())
	return nil
}

// arrayList — recursive multi-element alts. Each emits the head (which
// recurses into a smaller arrayList) then appends this tail element.

func (e *PostfixEmitter) VisitArrayListInt(ctx *ArrayListIntContext) interface{} {
	e.Visit(ctx.ArrayList())
	e.emitArrayListAdd(ctx.Iexpr())
	return nil
}

func (e *PostfixEmitter) VisitArrayListStr(ctx *ArrayListStrContext) interface{} {
	e.Visit(ctx.ArrayList())
	e.emitArrayListAdd(ctx.Strexpr())
	return nil
}

func (e *PostfixEmitter) VisitArrayListFloat(ctx *ArrayListFloatContext) interface{} {
	e.Visit(ctx.ArrayList())
	e.emitArrayListAdd(ctx.Fexpr())
	return nil
}

func (e *PostfixEmitter) VisitArrayListBool(ctx *ArrayListBoolContext) interface{} {
	e.Visit(ctx.ArrayList())
	e.emitArrayListAdd(ctx.Bexpr())
	return nil
}

func (e *PostfixEmitter) VisitArrayListName(ctx *ArrayListNameContext) interface{} {
	e.Visit(ctx.ArrayList())
	e.emitArrayListAdd(ctx.Nexpr())
	return nil
}

func (e *PostfixEmitter) VisitArrayListEntity(ctx *ArrayListEntityContext) interface{} {
	e.Visit(ctx.ArrayList())
	e.emitArrayListAdd(ctx.Eexpr())
	return nil
}

func (e *PostfixEmitter) VisitArrayListArray(ctx *ArrayListArrayContext) interface{} {
	e.Visit(ctx.ArrayList())
	e.emitArrayListAdd(ctx.ArrayExpr())
	return nil
}

// arrayList — single-element base cases (leftmost element).

func (e *PostfixEmitter) VisitArrayListIntSingle(ctx *ArrayListIntSingleContext) interface{} {
	e.emitArrayListAdd(ctx.Iexpr())
	return nil
}

func (e *PostfixEmitter) VisitArrayListStrSingle(ctx *ArrayListStrSingleContext) interface{} {
	e.emitArrayListAdd(ctx.Strexpr())
	return nil
}

func (e *PostfixEmitter) VisitArrayListFloatSingle(ctx *ArrayListFloatSingleContext) interface{} {
	e.emitArrayListAdd(ctx.Fexpr())
	return nil
}

func (e *PostfixEmitter) VisitArrayListBoolSingle(ctx *ArrayListBoolSingleContext) interface{} {
	e.emitArrayListAdd(ctx.Bexpr())
	return nil
}

func (e *PostfixEmitter) VisitArrayListNameSingle(ctx *ArrayListNameSingleContext) interface{} {
	e.emitArrayListAdd(ctx.Nexpr())
	return nil
}

func (e *PostfixEmitter) VisitArrayListEntitySingle(ctx *ArrayListEntitySingleContext) interface{} {
	e.emitArrayListAdd(ctx.Eexpr())
	return nil
}

func (e *PostfixEmitter) VisitArrayListArraySingle(ctx *ArrayListArraySingleContext) interface{} {
	e.emitArrayListAdd(ctx.ArrayExpr())
	return nil
}

// ============================================================================
// Typed Identifier Visitors
// ============================================================================

func (e *PostfixEmitter) VisitTypedEntity(ctx *TypedEntityContext) interface{} {
	name := ctx.GetText()
	if e.emitAliasFieldAccess(name) {
		return nil
	}
	// Check if this is a local variable - emit stack frame access
	if !e.emitLocalRef(name) {
		e.emit(name)
	}
	return nil
}

func (e *PostfixEmitter) VisitTypedLong(ctx *TypedLongContext) interface{} {
	name := ctx.GetText()
	if e.emitAliasFieldAccess(name) {
		return nil
	}
	// Check if this is a local variable - emit stack frame access
	if !e.emitLocalRef(name) {
		e.emit(name)
	}
	return nil
}

func (e *PostfixEmitter) VisitTypedDouble(ctx *TypedDoubleContext) interface{} {
	name := ctx.GetText()
	if e.emitAliasFieldAccess(name) {
		return nil
	}
	// Check if this is a local variable - emit stack frame access
	if !e.emitLocalRef(name) {
		e.emit(name)
	}
	return nil
}

func (e *PostfixEmitter) VisitTypedString(ctx *TypedStringContext) interface{} {
	name := ctx.GetText()
	if e.emitAliasFieldAccess(name) {
		return nil
	}
	// Check if this is a local variable - emit stack frame access
	if !e.emitLocalRef(name) {
		e.emit(name)
	}
	return nil
}

func (e *PostfixEmitter) VisitTypedBoolean(ctx *TypedBooleanContext) interface{} {
	name := ctx.GetText()
	if e.emitAliasFieldAccess(name) {
		return nil
	}
	// Check if this is a local variable - emit stack frame access
	if !e.emitLocalRef(name) {
		e.emit(name)
	}
	return nil
}

func (e *PostfixEmitter) VisitTypedDate(ctx *TypedDateContext) interface{} {
	name := ctx.GetText()
	if e.emitAliasFieldAccess(name) {
		return nil
	}
	// Check if this is a local variable - emit stack frame access
	if !e.emitLocalRef(name) {
		e.emit(name)
	}
	return nil
}

func (e *PostfixEmitter) VisitTypedArray(ctx *TypedArrayContext) interface{} {
	name := ctx.GetText()
	if e.emitAliasFieldAccess(name) {
		return nil
	}
	// Check if this is a local variable - emit stack frame access
	if !e.emitLocalRef(name) {
		e.emit(name)
	}
	return nil
}

func (e *PostfixEmitter) VisitTypedTable(ctx *TypedTableContext) interface{} {
	name := ctx.GetText()
	if e.emitAliasFieldAccess(name) {
		return nil
	}
	// Check if this is a local variable - emit stack frame access
	if !e.emitLocalRef(name) {
		e.emit(name)
	}
	return nil
}

func (e *PostfixEmitter) VisitTypedName(ctx *TypedNameContext) interface{} {
	name := ctx.GetText()
	if e.emitAliasFieldAccess(name) {
		return nil
	}
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
// VisitForallReverse: `for all <array> in reverse` — iterate the array from
// the last element to the first.
//
// The runtime has always had forallr; until now the only EL that reached it
// was `remove each ... where ...` and the `allowing array to be removed`
// variants, both of which iterate backwards for removal safety rather than
// because the caller wanted that order. Rules that simply need reverse order
// had to be written as hand-coded postfix — SyntaxTests has 48 such rows
// (#975).
func (e *PostfixEmitter) VisitForallReverse(ctx *ForallReverseContext) interface{} {
	e.emit("dup")
	e.Visit(ctx.ArrayExpr())
	e.emit("forallr")
	e.emit("pop")
	return nil
}

// VisitForallReverseWhere: `for all <array> in reverse where <bexpr>`.
func (e *PostfixEmitter) VisitForallReverseWhere(ctx *ForallReverseWhereContext) interface{} {
	e.emit("{")
	e.emit("{")
	e.emit("dup")
	e.emit("execute")
	e.emit("}")
	e.Visit(ctx.Bexpr())
	e.emit("if")
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forallr")
	e.emit("pop")
	return nil
}

// The `allowing array to be removed` variants iterate in REVERSE. That is
// the whole point of the phrase: walking forward while the body removes
// elements skips entries, because the iterator's index and the array's
// contents move against each other. VisitRemoveEachWhere has always used
// forallr for exactly this reason. These three emitted plain `forall`, so a
// rule that said it was going to remove elements got the iteration order
// that makes removal unsafe.
func (e *PostfixEmitter) VisitForallAllowRemove(ctx *ForallAllowRemoveContext) interface{} {
	e.emit("dup")
	e.Visit(ctx.ArrayExpr())
	e.emit("forallr")
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
	e.emit("forallr")
	e.emit("pop")
	return nil
}

// resolveEntitiesCollection looks up the `<owner>.<field>` path for a bare
// entity-type name via the registered collection resolver. On any error it
// records the emission error and returns the empty string.
func (e *PostfixEmitter) resolveEntitiesCollection(entityType string) string {
	if e.resolveCollection == nil {
		e.emitError("for all %s entities: no collection resolver registered", entityType)
		return ""
	}
	owner, field, err := e.resolveCollection(entityType)
	if err != nil {
		e.emitError("%v", err)
		return ""
	}
	return owner + "." + field
}

// VisitForallTypeEntities: `for all <type> entities` — rewrites to the
// EDD-declared owning collection as `dup <owner>.<field> forall pop`.
func (e *PostfixEmitter) VisitForallTypeEntities(ctx *ForallTypeEntitiesContext) interface{} {
	path := e.resolveEntitiesCollection(ctx.TypedEntity().GetText())
	if path == "" {
		return nil
	}
	e.emit("dup")
	e.emit(path)
	e.emit("forall")
	e.emit("pop")
	return nil
}

// VisitForallTypeEntitiesWhere: `for all <type> entities where <bexpr>` —
// mirrors the `forallWhere` wrapped-block shape, substituting the resolved
// `<owner>.<field>` for the array expression.
func (e *PostfixEmitter) VisitForallTypeEntitiesWhere(ctx *ForallTypeEntitiesWhereContext) interface{} {
	path := e.resolveEntitiesCollection(ctx.TypedEntity().GetText())
	if path == "" {
		return nil
	}
	e.emit("{")
	e.emit("{")
	e.emit("dup")
	e.emit("execute")
	e.emit("}")
	e.Visit(ctx.Bexpr())
	e.emit("if")
	e.emit("}")
	e.emit(path)
	e.emit("forall")
	e.emit("pop")
	return nil
}

// VisitForallAs: `for all <array> as <alias>` (#712, #714).
//
// Binds each iteration entity to a local-entity slot named `alias` instead of
// pushing it on the entity stack. This makes nested same-list iterations
// non-shadowing: `for all taxpayers as parent { for all taxpayers as child
// where child.parent_id == parent.id ... }` — both `parent.*` and `child.*`
// resolve through distinct local slots.
//
// The context wraps the table call so the data stack holds the body block on
// entry: `{ /T executetable } <ctx-postfix>`. The shape below keeps the slot
// reserved across the whole iteration (so `local!`/`local@` are valid every
// element) and runs the body once per element, never once before the loop.
//
// Emit shape (body is on top of data stack at entry):
//
//	null allocate                          # reserve slot N (ctrl push null)
//	{ <N> local! dup execute }             # wrapper: stash elem, dup body, run
//	<arr>                                  # iteration array
//	for                                    # ( body array -- ), per iter:
//	                                       #   pushes elem, runs wrapper
//	deallocate pop                         # release slot, drop the null
//	pop                                    # drop the body that survived `for`
//
// `for` is used instead of `forall` because `for` pushes the element on the
// data stack, which the wrapper needs in order to store it into the alias
// local with `local!`. `forall` would push on the entity stack, which the
// issue explicitly forbids when `as` is used.
func (e *PostfixEmitter) VisitForallAs(ctx *ForallAsContext) interface{} {
	alias := ctx.UndefinedIdent().GetText()
	if err := e.checkAliasName(alias); err != nil {
		e.emitError("%v", err)
		return nil
	}
	idx := e.declareLocal(alias, TypeEntity)
	e.emit("null")
	e.emit("allocate")
	e.emit("{")
	e.emit(fmt.Sprintf("%d", idx))
	e.emit("local!")
	e.emit("dup")
	e.emit("execute")
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("for")
	e.emit("deallocate")
	e.emit("pop")
	e.emit("pop")
	return nil
}

// VisitForallAsWhere: `for all <array> as <alias> where <bexpr>` (#712, #714).
// Same binding shape as forallAs, but the body only runs when the predicate
// holds. The predicate is evaluated AFTER the alias slot is populated so
// `<alias>.<field>` references inside the where-clause resolve correctly.
func (e *PostfixEmitter) VisitForallAsWhere(ctx *ForallAsWhereContext) interface{} {
	alias := ctx.UndefinedIdent().GetText()
	if err := e.checkAliasName(alias); err != nil {
		e.emitError("%v", err)
		return nil
	}
	idx := e.declareLocal(alias, TypeEntity)
	e.emit("null")
	e.emit("allocate")
	e.emit("{")
	e.emit(fmt.Sprintf("%d", idx))
	e.emit("local!")
	e.emit("{")
	e.emit("dup")
	e.emit("execute")
	e.emit("}")
	e.Visit(ctx.Bexpr())
	e.emit("if")
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("for")
	e.emit("deallocate")
	e.emit("pop")
	e.emit("pop")
	return nil
}

// checkAliasName rejects aliases that collide with an existing EDD symbol
// (typically an entity type name). A collision would break reference
// resolution because `<alias>.<field>` would be ambiguous with the existing
// `<entity>.<field>` path registered in the symbol table.
func (e *PostfixEmitter) checkAliasName(name string) error {
	if t := e.lookupType(name); t != "" {
		return fmt.Errorf("alias %q collides with existing symbol (type %s)", name, t)
	}
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
	e.emit("forallr")
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
	// Capture the entity type when the RHS is `new <typedEntity> entity`
	// (#819) so mutationType can resolve `<local>.<field>` against the
	// EDD's `<typedEntity>.<field>` symbol — without this, SET on a
	// local-entity field gets the default integer cv* cast.
	e.declareLocalEntity(name, TypeEntity, entityTypeFromEexpr(ctx.Eexpr()))
	e.Visit(ctx.Eexpr())
	e.emit("cve")
	e.emit("allocate")
	e.emit("execute")
	e.emit("deallocate")
	e.emit("pop")
	return nil
}

// entityTypeFromEexpr returns the entity-type name when an eexpr is a
// `new T entity` constructor. Two grammar alts can match — ANTLR picks
// `entityNewName` first because IDENT also matches nexpr, but
// `entityNewTyped` is reachable in theory; handle both. Other eexpr
// shapes — bare typedEntity references, function calls, table lookups
// — don't expose a compile-time entity type, so we return "" and the
// local declaration is type-unaware.
func entityTypeFromEexpr(ee IEexprContext) string {
	if ee == nil {
		return ""
	}
	switch n := ee.(type) {
	case *EntityNewNameContext:
		if ne := n.Nexpr(); ne != nil {
			return ne.GetText()
		}
	case *EntityNewTypedContext:
		if te := n.TypedEntity(); te != nil {
			return te.GetText()
		}
	}
	return ""
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
	e.emit("cvd")
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
	e.emit("/" + ctx.TypedDecisionTable().GetText())
	e.emit("performtable")
	return nil
}

func (e *PostfixEmitter) VisitPerformDTExplicit(ctx *PerformDTExplicitContext) interface{} {
	e.emit("/" + ctx.TypedDecisionTable().GetText())
	e.emit("performtable")
	return nil
}

// VisitCreateEntityAs: `create <type-name> as <local-name>` — construct a
// fresh entity of the named type and bind it to a local name. Lowers to
// `/typeName createentity /localName xdef`, which matches the postfix
// idiom used to build state_tax_result instances inside dispatch actions.
// The type name is emitted as a name literal so the runtime can look up
// the entity type in the EDD at execute time.
func (e *PostfixEmitter) VisitCreateEntityAs(ctx *CreateEntityAsContext) interface{} {
	typeName := ctx.TypedEntity().GetText()
	localName := ctx.UndefinedIdent().GetText()
	e.emit("/" + typeName)
	e.emit("createentity")
	e.emit("/" + localName)
	e.emit("xdef")
	return nil
}

// VisitPerformDynamicTable: `perform table named (<string-expression>)` —
// evaluate the expression at runtime, coerce the resulting string to a
// decision-table name, and execute that table. Lowers to the same postfix
// that hand-coded dispatches used to write by binding the concatenated
// name into a local and then calling `execute`, except we skip the local
// since `performtable` takes the name directly off the data stack.
func (e *PostfixEmitter) VisitPerformDynamicTable(ctx *PerformDynamicTableContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("performtable")
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
	e.visitBlocksScoped(ctx.AllBlock())
	return nil
}

// visitBlocksScoped walks a statement list, giving local-variable and
// create-as declarations a real scope (#904). The runtime local machinery is
// `<init> allocate <body> execute deallocate pop` — the declared slot is only
// live while <body> runs. At context level the table body block is already on
// the data stack when the declaration postfix runs, so the flat shape works;
// inside an action body there is no block on the stack, and the flat shape
// emitted by the Local* visitors underflowed at `execute` while the
// statements that used the local ran AFTER `deallocate`. Here the remaining
// statements of the list become the executed block:
//
//	<init> allocate { <rest of statements> } execute deallocate pop
//
// nesting recursively for each declaration, mirroring how
// compileContextsPostfix nests per-cell context declarations.
func (e *PostfixEmitter) visitBlocksScoped(blocks []IBlockContext) {
	for i, b := range blocks {
		if decl := scopedDeclOf(b); decl != nil && e.emitScopedLocalPrefix(decl) {
			e.emit("allocate")
			e.emit("{")
			e.visitBlocksScoped(blocks[i+1:])
			e.emit("}")
			e.emit("execute")
			e.emit("deallocate")
			e.emit("pop")
			return
		}
		e.Visit(b)
	}
}

// scopedDeclOf returns the localvariables or createstatement context when a
// block consists of a single declaration statement, nil otherwise.
func scopedDeclOf(b IBlockContext) antlr.ParseTree {
	bs, ok := b.(*BlockStatementContext)
	if !ok {
		return nil
	}
	st, ok := bs.Statement().(*StatementContext)
	if !ok || st == nil {
		return nil
	}
	if lv := st.Localvariables(); lv != nil {
		// The concrete alternative context is the localvariables child.
		if pt, ok := lv.(antlr.ParseTree); ok {
			return pt
		}
	}
	if cs := st.Createstatement(); cs != nil {
		if pt, ok := cs.(antlr.ParseTree); ok {
			return pt
		}
	}
	return nil
}

// emitScopedLocalPrefix declares the local and emits its initial value for
// the scoped shape (everything up to, but not including, `allocate`).
// Returns false for declaration forms that don't bind a new local (the
// *Defined reference alts) and for create-as with a name that resolves in
// the EDD — that form keeps the legacy attribute-binding lowering for
// back-compat (see VisitCreateEntityAs).
func (e *PostfixEmitter) emitScopedLocalPrefix(decl antlr.ParseTree) bool {
	switch c := decl.(type) {
	case *CreateEntityAsContext:
		name := c.UndefinedIdent().GetText()
		if e.lookupType(name) != "" {
			return false // declared-attribute binding — legacy lowering
		}
		typeName := c.TypedEntity().GetText()
		e.declareLocalEntity(name, TypeEntity, typeName)
		e.emit("/" + typeName)
		e.emit("createentity")
	case *LocalEntityInitContext:
		e.declareLocalEntity(c.UndefinedIdent().GetText(), TypeEntity, entityTypeFromEexpr(c.Eexpr()))
		e.Visit(c.Eexpr())
		e.emit("cve")
	case *LocalEntityUndefContext:
		e.declareLocal(c.UndefinedIdent().GetText(), TypeEntity)
		e.emit("null")
	case *LocalLongInitContext:
		e.declareLocal(c.UndefinedIdent().GetText(), TypeInteger)
		e.Visit(c.Number())
		e.emit("cvi")
	case *LocalLongUndefContext:
		e.declareLocal(c.UndefinedIdent().GetText(), TypeInteger)
		e.emit("null")
	case *LocalDoubleInitContext:
		e.declareLocal(c.UndefinedIdent().GetText(), TypeDouble)
		e.Visit(c.Number())
		e.emit("cvd")
	case *LocalDoubleUndefContext:
		e.declareLocal(c.UndefinedIdent().GetText(), TypeDouble)
		e.emit("null")
	case *LocalBoolInitContext:
		e.declareLocal(c.UndefinedIdent().GetText(), TypeBoolean)
		e.Visit(c.Bexpr())
		e.emit("cvb")
	case *LocalBoolUndefContext:
		e.declareLocal(c.UndefinedIdent().GetText(), TypeBoolean)
		e.emit("null")
	case *LocalDateInitContext:
		e.declareLocal(c.UndefinedIdent().GetText(), TypeDate)
		e.Visit(c.Dexpr())
		e.emit("cvd")
	case *LocalDateUndefContext:
		e.declareLocal(c.UndefinedIdent().GetText(), TypeDate)
		e.emit("null")
	case *LocalArrayInitContext:
		e.declareLocal(c.UndefinedIdent().GetText(), TypeArray)
		e.Visit(c.ArrayExpr())
	case *LocalArrayUndefContext:
		e.declareLocal(c.UndefinedIdent().GetText(), TypeArray)
		e.emit("null")
	case *LocalStringInitContext:
		e.declareLocal(c.UndefinedIdent().GetText(), TypeString)
		e.Visit(c.Strexpr())
		e.emit("cvs")
	case *LocalStringUndefContext:
		e.declareLocal(c.UndefinedIdent().GetText(), TypeString)
		e.emit("null")
	case *LocalBigIntInitContext:
		e.declareLocal(c.UndefinedIdent().GetText(), TypeBigInt)
		e.Visit(c.Bigexpr())
		e.emit("cvbi")
	case *LocalBigIntUndefContext:
		e.declareLocal(c.UndefinedIdent().GetText(), TypeBigInt)
		e.emit("null")
	case *LocalFixedInitContext:
		e.declareLocal(c.UndefinedIdent().GetText(), TypeFixed)
		e.emitWithTypeConversion(c.Iexpr(), TypeFixed)
	case *LocalFixedUndefContext:
		e.declareLocal(c.UndefinedIdent().GetText(), TypeFixed)
		e.emit("null")
	case *LocalBytesInitContext:
		e.declareLocal(c.UndefinedIdent().GetText(), TypeBytes)
		e.Visit(c.Bytesexpr())
		e.emit("cvbytes")
	case *LocalBytesUndefContext:
		e.declareLocal(c.UndefinedIdent().GetText(), TypeBytes)
		e.emit("null")
	default:
		return false // *Defined reference alts keep their existing lowering
	}
	return true
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

// resolveSetTarget returns the declared type for a set-statement target. It
// checks locals first (via mutationType) so `local fixed amount` dispatches
// as fixed even if the EDD has an `amount` entity field, then falls back to
// the grammar-inferred default when neither source has a declared type.
func (e *PostfixEmitter) resolveSetTarget(name, defaultType string) string {
	if t := e.mutationType(name); t != "" {
		return t
	}
	return defaultType
}

func (e *PostfixEmitter) VisitSetInt(ctx *SetIntContext) interface{} {
	e.Visit(ctx.Number())
	fieldType := e.resolveSetTarget(ctx.LeftIexpr().GetText(), TypeInteger)
	if conv := e.typeConverter(fieldType); conv != "" {
		e.emit(conv)
	}
	e.Visit(ctx.LeftIexpr())
	return nil
}

func (e *PostfixEmitter) VisitSetFloat(ctx *SetFloatContext) interface{} {
	e.Visit(ctx.Number())
	fieldType := e.resolveSetTarget(ctx.LeftFexpr().GetText(), TypeDouble)
	if conv := e.typeConverter(fieldType); conv != "" {
		e.emit(conv)
	}
	e.Visit(ctx.LeftFexpr())
	return nil
}

func (e *PostfixEmitter) VisitSetBool(ctx *SetBoolContext) interface{} {
	e.Visit(ctx.Bexpr())
	fieldType := e.resolveSetTarget(ctx.LeftBexpr().GetText(), TypeBoolean)
	if conv := e.typeConverter(fieldType); conv != "" {
		e.emit(conv)
	}
	e.Visit(ctx.LeftBexpr())
	return nil
}

func (e *PostfixEmitter) VisitSetEntity(ctx *SetEntityContext) interface{} {
	e.Visit(ctx.Eexpr())
	fieldType := e.resolveSetTarget(ctx.LeftEexpr().GetText(), TypeEntity)
	if conv := e.typeConverter(fieldType); conv != "" {
		e.emit(conv)
	}
	e.Visit(ctx.LeftEexpr())
	return nil
}

func (e *PostfixEmitter) VisitSetString(ctx *SetStringContext) interface{} {
	e.Visit(ctx.Strexpr())
	fieldType := e.resolveSetTarget(ctx.LeftStrexpr().GetText(), TypeString)
	if conv := e.typeConverter(fieldType); conv != "" {
		e.emit(conv)
	}
	e.Visit(ctx.LeftStrexpr())
	return nil
}

func (e *PostfixEmitter) VisitSetDate(ctx *SetDateContext) interface{} {
	e.Visit(ctx.Dexpr())
	fieldType := e.resolveSetTarget(ctx.LeftDexpr().GetText(), TypeDate)
	if conv := e.typeConverter(fieldType); conv != "" {
		e.emit(conv)
	}
	e.Visit(ctx.LeftDexpr())
	return nil
}

// VisitSetStringFromDate handles `set <ident> = <dexpr>`. The setDate
// alt above (line 258 of EL.g4) IS the semantic intent, but ANTLR
// adaptive prediction matches `SET leftStrexpr ASSIGN dexpr`
// (setStringFromDate, line 254) first because leftStrexpr also accepts
// the IDENT — so this is the actually-reachable entry point. Without
// this visitor `set <date-field> = <complex-dexpr>` produced empty
// postfix (#803). resolveSetTarget recovers the true field type so the
// correct cv* trailer fires (cvdate for date fields; cvs for actual
// string fields receiving a stringified date).
func (e *PostfixEmitter) VisitSetStringFromDate(ctx *SetStringFromDateContext) interface{} {
	e.Visit(ctx.Dexpr())
	name := ctx.LeftStrexpr().GetText()
	fieldType := e.resolveSetTarget(name, TypeString)
	if conv := e.typeConverter(fieldType); conv != "" {
		e.emit(conv)
	}
	e.Visit(ctx.LeftStrexpr())
	return nil
}

// Left*Simple visitors route through emitFieldStore so declared locals get
// `<index> local!` while entity fields keep `/<name> xdef`. Without this,
// `set <local> = ...` would error at runtime because xdef can't resolve a
// name that isn't on the entity stack.

func (e *PostfixEmitter) VisitLeftIexprSimple(ctx *LeftIexprSimpleContext) interface{} {
	e.emitFieldStore(ctx.TypedLong().GetText())
	return nil
}

func (e *PostfixEmitter) VisitLeftFexprSimple(ctx *LeftFexprSimpleContext) interface{} {
	e.emitFieldStore(ctx.TypedDouble().GetText())
	return nil
}

func (e *PostfixEmitter) VisitLeftBexprSimple(ctx *LeftBexprSimpleContext) interface{} {
	e.emitFieldStore(ctx.TypedBoolean().GetText())
	return nil
}

func (e *PostfixEmitter) VisitLeftEexprSimple(ctx *LeftEexprSimpleContext) interface{} {
	e.emitFieldStore(ctx.TypedEntity().GetText())
	return nil
}

func (e *PostfixEmitter) VisitLeftStrexprSimple(ctx *LeftStrexprSimpleContext) interface{} {
	e.emitFieldStore(ctx.TypedString().GetText())
	return nil
}

func (e *PostfixEmitter) VisitLeftDexprSimple(ctx *LeftDexprSimpleContext) interface{} {
	e.emitFieldStore(ctx.TypedDate().GetText())
	return nil
}

// VisitLeftArraySimple: `<typedArray>` as the LHS of `set <array> = ...`.
// Matches the pattern of the other LeftXxxSimple visitors — emit the
// field-store trailer so `set my_list = [1,2,3]` lands correctly.
// Required by VisitSetArrayArray (#803 batch 3).
func (e *PostfixEmitter) VisitLeftArraySimple(ctx *LeftArraySimpleContext) interface{} {
	e.emitFieldStore(ctx.TypedArray().GetText())
	return nil
}

// VisitSetArrayArray: `set <array-field> = <arrayExpr>`. Matches the
// pattern of VisitSetEntity et al. — visit RHS (which leaves the array
// on the stack), then visit LHS to emit the field-store trailer. No
// type conversion needed for array→array.
func (e *PostfixEmitter) VisitSetArrayArray(ctx *SetArrayArrayContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.Visit(ctx.LeftArrayRef())
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
// VisitForallBlockReverse: `for all <array> in reverse { ... }` — the
// action-cell form of reverse iteration (#975). SyntaxTests has 48 rows of
// exactly this shape written as hand-coded postfix, because until now the
// only EL that reached forallr was the removal-safety phrasing.
func (e *PostfixEmitter) VisitForallBlockReverse(ctx *ForallBlockReverseContext) interface{} {
	e.emit("{")
	e.Visit(ctx.Block())
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forallr")
	return nil
}

// VisitForallBlockReverseWhere: the same with a filter.
func (e *PostfixEmitter) VisitForallBlockReverseWhere(ctx *ForallBlockReverseWhereContext) interface{} {
	e.emit("{")
	e.emit("{")
	e.Visit(ctx.Block())
	e.emit("}")
	e.Visit(ctx.Bexpr())
	e.emit("if")
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forallr")
	return nil
}

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

// VisitIfblock: the runtime's ifelse pops the TEST from the top of the
// stack (matching the lazy and/or `over if` emission and hasrelationship
// forms), so the bodies are pushed first and the bexpr last:
// `{ then } { else } <bexpr> ifelse`. Blocks are literals, so deferring
// the bexpr evaluation is side-effect free.
func (e *PostfixEmitter) VisitIfblock(ctx *IfblockContext) interface{} {
	e.emit("{")
	e.Visit(ctx.StatementList())
	e.emit("}")
	e.Visit(ctx.Ifcontinue())
	e.Visit(ctx.Bexpr())
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

// emitOneForField pushes the literal "1" (or "1.0" for double-typed fields)
// as the RHS value for increment / decrement statements. Double needs the
// decimal form so the subsequent f+ / f- op can consume a double rather
// than integer-truncating through cvd.
func (e *PostfixEmitter) emitOneForField(fieldName string) {
	if e.mutationType(fieldName) == TypeDouble {
		e.emit("1.0")
		return
	}
	e.emit("1")
}

func (e *PostfixEmitter) VisitIncrementLong(ctx *IncrementLongContext) interface{} {
	// typedLong matches any IDENT; the declared field type tells us which
	// numeric family to use. Without this check, incrementing a fixed,
	// bigint, or double field would emit plain `+` and truncate via
	// LongValue() at runtime.
	name := ctx.TypedLong().GetText()
	e.emitOneForField(name)
	e.emitTypeAwareAddSub(name, "+")
	return nil
}

func (e *PostfixEmitter) VisitIncrementDouble(ctx *IncrementDoubleContext) interface{} {
	// typedDouble is also just IDENT — this alternative is rarely reached
	// (typedLong wins first), but when it is we honor the declared field
	// type the same way IncrementLong does.
	name := ctx.TypedDouble().GetText()
	e.emitOneForField(name)
	e.emitTypeAwareAddSub(name, "+")
	return nil
}

func (e *PostfixEmitter) VisitDecrementLong(ctx *DecrementLongContext) interface{} {
	name := ctx.TypedLong().GetText()
	e.emitOneForField(name)
	e.emitTypeAwareAddSub(name, "-")
	return nil
}

func (e *PostfixEmitter) VisitDecrementDouble(ctx *DecrementDoubleContext) interface{} {
	name := ctx.TypedDouble().GetText()
	e.emitOneForField(name)
	e.emitTypeAwareAddSub(name, "-")
	return nil
}

// VisitSetStringFromName: `set <left> = <nexpr>`. The parser commits to
// this alternative for any IDENT target (setBoolFromName is structurally
// indistinguishable at parse time), so disambiguate here: look up the
// target field's type in the symbol table and emit the matching coercion
// (cvb for bool, cvs otherwise). `leftStrexpr` / `leftBexpr` both resolve
// to `/<name> xdef` so the left emit is identical regardless.
func (e *PostfixEmitter) VisitSetStringFromName(ctx *SetStringFromNameContext) interface{} {
	e.Visit(ctx.Nexpr())
	fieldName := ctx.LeftStrexpr().GetText()
	if e.mutationType(fieldName) == TypeBoolean {
		e.emit("cvb")
	} else {
		e.emit("cvs")
	}
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

// Phase 3 of #743: calendar comparison visitors. Each pushes the two dates,
// then the zone (and the start-day for week comparisons), and emits the
// samecalendar* op. INZONE is mandatory at the grammar level so there is no
// non-zone variant.

func (e *PostfixEmitter) VisitBoolSameCalendarDay(ctx *BoolSameCalendarDayContext) interface{} {
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(1))
	e.Visit(ctx.Strexpr())
	e.emit("samecalendardayinzone")
	return nil
}

func (e *PostfixEmitter) VisitBoolSameCalendarWeek(ctx *BoolSameCalendarWeekContext) interface{} {
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(1))
	e.Visit(ctx.Strexpr())
	e.emit("samecalendarweekinzone")
	return nil
}

func (e *PostfixEmitter) VisitBoolSameCalendarWeekStarting(ctx *BoolSameCalendarWeekStartingContext) interface{} {
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(1))
	e.Visit(ctx.Strexpr(0))
	e.Visit(ctx.Strexpr(1))
	e.emit("samecalendarweekstartinginzone")
	return nil
}

func (e *PostfixEmitter) VisitBoolSameCalendarMonth(ctx *BoolSameCalendarMonthContext) interface{} {
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(1))
	e.Visit(ctx.Strexpr())
	e.emit("samecalendarmonthinzone")
	return nil
}

func (e *PostfixEmitter) VisitBoolSameCalendarQuarter(ctx *BoolSameCalendarQuarterContext) interface{} {
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(1))
	e.Visit(ctx.Strexpr())
	e.emit("samecalendarquarterinzone")
	return nil
}

func (e *PostfixEmitter) VisitBoolSameCalendarYear(ctx *BoolSameCalendarYearContext) interface{} {
	e.Visit(ctx.Dexpr(0))
	e.Visit(ctx.Dexpr(1))
	e.Visit(ctx.Strexpr())
	e.emit("samecalendaryearinzone")
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

// Addtostatement dup-destination + no-dups family. opAddTo signature is
// (array element --); opAddArray signature is (array1 array2 bool --) where
// bool=false means skip duplicates; the Contains-then-Add pattern for a
// single element is the built-in add_no_dups op.
//
// Pattern for value-to-two-destinations: push value, dup, add to dest1, then
// add the remaining copy to dest2. addtodest's emit for typed-array dests is
// `<field> swap addto` — i.e. expects value already on stack, swaps so addto
// sees (array, value), consumes. The dup-form sequence becomes:
//     <value> dup <dest1> swap addto <dest2> swap addto
// Same shape works for entity / string / number / date values.

// VisitAddDateToDest: `add <dexpr> to <addtodest>`. The destination visitor
// owns the store (`<arr> swap addto` for arrays) — appending another
// `swap addto` here double-emitted the trailer for non-IDENT dexprs (#904;
// bare-IDENT date values match the arrayExpr alternatives and never reach
// this visitor). Mirrors VisitAddStrToDest (#781).
func (e *PostfixEmitter) VisitAddDateToDest(ctx *AddDateToDestContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Addtodest())
	return nil
}

// The dup-destination family relies on each destination visitor emitting its
// own store (`<arr> swap addto` for array dests, `<field> + /<field> xdef`
// for numeric dests) — appending an explicit `swap addto` here after the
// dest visit double-emitted the trailer and corrupted the stack (#904, same
// class as the #781 single-dest string fix). Shape:
//
//	<value> dup <dest0-with-store> <dest1-with-store>
//
// The first store consumes the dup'd copy, the second consumes the original.
func (e *PostfixEmitter) VisitAddEntityToDestDup(ctx *AddEntityToDestDupContext) interface{} {
	e.Visit(ctx.Eexpr())
	e.emit("dup")
	e.Visit(ctx.Addtodest(0))
	e.Visit(ctx.Addtodest(1))
	return nil
}

func (e *PostfixEmitter) VisitAddStrToDestDup(ctx *AddStrToDestDupContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("dup")
	e.Visit(ctx.Addtodest(0))
	e.Visit(ctx.Addtodest(1))
	return nil
}

func (e *PostfixEmitter) VisitAddDateToDestDup(ctx *AddDateToDestDupContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.emit("dup")
	e.Visit(ctx.Addtodest(0))
	e.Visit(ctx.Addtodest(1))
	return nil
}

// VisitAddNumToDestDup — number variant uses the same pattern. addtodest's
// emit for numeric targets does `<field> + /<field> xdef`, which expects the
// value below the field on the stack. For dup-form we can't cleanly reuse
// addtodest because each dest mutation is stateful. Emit two explicit
// accumulations on the named targets.
func (e *PostfixEmitter) VisitAddNumToDestDup(ctx *AddNumToDestDupContext) interface{} {
	e.Visit(ctx.Number())
	e.emit("dup")
	e.Visit(ctx.Addtodest(0))
	e.Visit(ctx.Addtodest(1))
	return nil
}

// VisitAddEntityNoDups: `add <e> if not member to <arr>` → `<arr> <e> add_no_dups`.
func (e *PostfixEmitter) VisitAddEntityNoDups(ctx *AddEntityNoDupsContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.Visit(ctx.Eexpr())
	e.emit("add_no_dups")
	return nil
}

func (e *PostfixEmitter) VisitAddStrNoDups(ctx *AddStrNoDupsContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.Visit(ctx.Strexpr())
	e.emit("add_no_dups")
	return nil
}

// Dup-destination no-dups variants: add element to each of two destinations
// if not already member. Dup the element and call add_no_dups on each.
func (e *PostfixEmitter) VisitAddEntityNoDupsDup(ctx *AddEntityNoDupsDupContext) interface{} {
	e.Visit(ctx.Eexpr())
	e.emit("dup")
	e.Visit(ctx.ArrayExpr(0))
	e.emit("swap")
	e.emit("add_no_dups")
	e.Visit(ctx.ArrayExpr(1))
	e.emit("swap")
	e.emit("add_no_dups")
	return nil
}

func (e *PostfixEmitter) VisitAddStrNoDupsDup(ctx *AddStrNoDupsDupContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("dup")
	e.Visit(ctx.ArrayExpr(0))
	e.emit("swap")
	e.emit("add_no_dups")
	e.Visit(ctx.ArrayExpr(1))
	e.emit("swap")
	e.emit("add_no_dups")
	return nil
}

// VisitForctl: `for <leftIexpr> = <number>; <bexpr>; <statement>`.
// Context-position only (reachable via contextForTable/contextFor). The
// outer table body is on the data stack when this runs. Emit:
//
//	init: <number> cvi leftIexpr-assign
//	loop: { dup execute <statement> } { <bexpr> } while pop
//
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

// VisitNameFromStr: `(name) <strexpr>` → `<strexpr> cvn`. Same
// coercion as `the name <strexpr>`; just a parenthesized cast form.
// Pre-fix the rule silently emitted nothing.
func (e *PostfixEmitter) VisitNameFromStr(ctx *NameFromStrContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("cvn")
	return nil
}

// VisitNameArrayAt: `name <typedArray>[<iexpr>]` → `<typedArray>
// <iexpr> getat`. Index into a name-typed array. Parallels
// VisitBoolArrayAt / VisitDateFromArrayAt. Pre-fix the rule silently
// emitted nothing.
func (e *PostfixEmitter) VisitNameArrayAt(ctx *NameArrayAtContext) interface{} {
	e.emit(ctx.TypedArray().GetText())
	e.Visit(ctx.Iexpr())
	e.emit("getat")
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
// Dispatch through emitTypeAwareAddSub so fixed / bigint / double targets
// don't silently truncate via plain `-`.
func (e *PostfixEmitter) VisitSubDestLong(ctx *SubDestLongContext) interface{} {
	e.emitTypeAwareAddSub(ctx.TypedLong().GetText(), "-")
	return nil
}

func (e *PostfixEmitter) VisitSubDestDouble(ctx *SubDestDoubleContext) interface{} {
	e.emitTypeAwareAddSub(ctx.TypedDouble().GetText(), "-")
	return nil
}

// VisitSubtractNum: `subtract <number> from <subtodest>` → push the number,
// then delegate to subtodest which computes field - value and stores.
func (e *PostfixEmitter) VisitSubtractNum(ctx *SubtractNumContext) interface{} {
	if e.rejectMutationDoubleMix(ctx.Number(), ctx.Subtodest()) {
		return nil
	}
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

// Phase 2 of #743: in-zone integer extractors. Mirror the layout of their
// plain counterparts above and emit the *inzone op so the runtime can read
// components in the requested zone.

func (e *PostfixEmitter) VisitIntDaysInYearInZone(ctx *IntDaysInYearInZoneContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Strexpr())
	e.emit("getdaysinyearinzone")
	return nil
}

func (e *PostfixEmitter) VisitIntDaysInMonthInZone(ctx *IntDaysInMonthInZoneContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Strexpr())
	e.emit("getdaysinmonthinzone")
	return nil
}

func (e *PostfixEmitter) VisitIntDayOfMonthInZone(ctx *IntDayOfMonthInZoneContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Strexpr())
	e.emit("getdayofmonthinzone")
	return nil
}

func (e *PostfixEmitter) VisitIntYearOfInZone(ctx *IntYearOfInZoneContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Strexpr())
	e.emit("yearofinzone")
	return nil
}

// Phase 3 of #743: time-component / dayofweek / weekofyear extractors.

func (e *PostfixEmitter) VisitIntHourOf(ctx *IntHourOfContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.emit("gethour")
	return nil
}

func (e *PostfixEmitter) VisitIntHourOfInZone(ctx *IntHourOfInZoneContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Strexpr())
	e.emit("gethourinzone")
	return nil
}

func (e *PostfixEmitter) VisitIntMinuteOf(ctx *IntMinuteOfContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.emit("getminute")
	return nil
}

func (e *PostfixEmitter) VisitIntMinuteOfInZone(ctx *IntMinuteOfInZoneContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Strexpr())
	e.emit("getminuteinzone")
	return nil
}

func (e *PostfixEmitter) VisitIntSecondOf(ctx *IntSecondOfContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.emit("getsecond")
	return nil
}

func (e *PostfixEmitter) VisitIntSecondOfInZone(ctx *IntSecondOfInZoneContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Strexpr())
	e.emit("getsecondinzone")
	return nil
}

func (e *PostfixEmitter) VisitIntDayOfWeek(ctx *IntDayOfWeekContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.emit("getdayofweek")
	return nil
}

func (e *PostfixEmitter) VisitIntDayOfWeekInZone(ctx *IntDayOfWeekInZoneContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Strexpr())
	e.emit("getdayofweekinzone")
	return nil
}

func (e *PostfixEmitter) VisitIntWeekOfYear(ctx *IntWeekOfYearContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.emit("getweekofyear")
	return nil
}

func (e *PostfixEmitter) VisitIntWeekOfYearInZone(ctx *IntWeekOfYearInZoneContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Strexpr())
	e.emit("getweekofyearinzone")
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

// VisitIndxExpr: `<arrayExpr> [ <iexpr> ]` — array element access. The
// rule had no override and antlr's BaseParseTreeVisitor.VisitChildren is
// a no-op, so every visitor that delegated via Visit(IndxExpr())
// silently produced empty output for the indexed expression. Affected
// callers include VisitEntityIndex, VisitDateFromIndex,
// VisitStrFromIndex, VisitIntFromIndex, VisitFloatFromIndex,
// VisitFixedFromIndex, and VisitBoolFromIndex (#803).
//
// Emits the standard array-index sequence the runtime expects:
// `<array> <index> bytesidx` — matching the postfix produced by the
// alternative iexpr path (`a.alist[0]` in an iexpr context) which has
// always worked through a different rule.
func (e *PostfixEmitter) VisitIndxExpr(ctx *IndxExprContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.Visit(ctx.Iexpr())
	e.emit("bytesidx")
	return nil
}

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

// Bexpr quantifier + existence family.
//
// Semantics and emits:
//
//   ALL arr HAVE b        # boolAllHave     →  true <arr> { <b> and } forall
//   ONE OF arr HASA b     # boolOneOfHasa   →  false <arr> { <b> or } forall
//
// Each seeds a bool accumulator on the data stack and folds bexpr per
// element. forall auto-pushes the element entity so bexpr resolves against
// each. The accumulator survives the loop as the final bool.
//
//   there is e where b              # boolThereIsWhere          →  <b>
//   there is e inthe e2 where b     # boolThereIsInEntityWhere  →  entitypush/eval/entitypop
//   there is e inthe arr where b    # boolThereIsInArrayWhere   →  false <arr> { <b> or } forall
//   there is no …                   # boolThereIsNo…Where       →  <corresponding positive> not
//
// The simple `there is e where b` form emits only the bexpr: the grammar
// position implies e is on the entity stack already; if not, runtime
// attribute lookup errors. More sophisticated scope-entry forms use the
// entitypush/entitypop or forall folding patterns.
//
//   eexpr HASA str WHERE bexpr      # boolEntityHasaWhere       →  ifelse on hasrelationship

// VisitWhereBody is the full-precedence predicate of a bexpr-hosted fold.
// The sub-rule exists so the where-clause takes the whole boolean expression
// instead of being cut at AND by the left-recursion rewrite (#1121); emission
// is a pass-through and the postfix is unchanged.
func (e *PostfixEmitter) VisitWhereBody(ctx *WhereBodyContext) interface{} {
	return e.Visit(ctx.Bexpr())
}

func (e *PostfixEmitter) VisitBoolAllHave(ctx *BoolAllHaveContext) interface{} {
	// opForall is ( body array -- ): the block must be emitted BEFORE the
	// array, else forall iterates the block as the array (#867 / #877).
	e.emit("true")
	e.emit("{")
	e.Visit(ctx.WhereBody())
	e.emit("and")
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forall")
	return nil
}

func (e *PostfixEmitter) VisitBoolOneOfHasa(ctx *BoolOneOfHasaContext) interface{} {
	// Block before array — opForall is ( body array -- ). See #877.
	e.emit("false")
	e.emit("{")
	e.Visit(ctx.WhereBody())
	e.emit("or")
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forall")
	return nil
}

func (e *PostfixEmitter) VisitBoolThereIsWhere(ctx *BoolThereIsWhereContext) interface{} {
	return e.Visit(ctx.WhereBody())
}

func (e *PostfixEmitter) VisitBoolThereIsNoWhere(ctx *BoolThereIsNoWhereContext) interface{} {
	e.Visit(ctx.WhereBody())
	e.emit("not")
	return nil
}

// eexprIsArray reports whether an eexpr that matched an entity-scope grammar
// alternative actually names an array (local or declared). The parser cannot
// tell `there is x in <entity> where …` from the array form — typedEntity and
// typedArray both come from IDENT — so the entity alternatives shadow
// boolThereIsInArrayWhere/NoInArrayWhere entirely and an array operand used
// to compile to `<arr> entitypush …`, crashing at runtime when entitypush
// calls REntityValue on the array (#869). Route by declared type instead.
func (e *PostfixEmitter) eexprIsArray(ctx antlr.ParseTree) bool {
	name := ctx.GetText()
	if lv, ok := e.lookupLocal(name); ok {
		return lv.Type == TypeArray
	}
	if e.lookupType(name) == TypeArray {
		return true
	}
	// Dotted reference: fall back to the field segment, mirroring the
	// colon-ref resolution in getExprType.
	if idx := strings.LastIndex(name, "."); idx > 0 {
		return e.lookupType(name[idx+1:]) == TypeArray
	}
	return false
}

// emitThereIsInArrayFold lowers `there is <x> in <array> where <p>` to the
// OR-accumulator fold, same shape as boolOneOfHasa: forall pushes each
// element onto the entity stack while running the body, so bare attribute
// names in the predicate resolve against the current element. The bound name
// <x> carries no scope of its own (exactly like `all … have`).
func (e *PostfixEmitter) emitThereIsInArrayFold(arr, pred antlr.ParseTree) {
	// Block before array — opForall is ( body array -- ). See #877.
	e.emit("false")
	e.emit("{")
	e.Visit(pred)
	e.emit("or")
	e.emit("}")
	e.Visit(arr)
	e.emit("forall")
}

func (e *PostfixEmitter) VisitBoolThereIsInEntityWhere(ctx *BoolThereIsInEntityWhereContext) interface{} {
	if e.eexprIsArray(ctx.Eexpr(1)) {
		e.emitThereIsInArrayFold(ctx.Eexpr(1), ctx.WhereBody())
		return nil
	}
	e.Visit(ctx.Eexpr(1))
	e.emit("entitypush")
	e.Visit(ctx.WhereBody())
	e.emit("entitypop")
	e.emit("swap")
	e.emit("pop")
	return nil
}

func (e *PostfixEmitter) VisitBoolThereIsNoInEntityWhere(ctx *BoolThereIsNoInEntityWhereContext) interface{} {
	if e.eexprIsArray(ctx.Eexpr(1)) {
		e.emitThereIsInArrayFold(ctx.Eexpr(1), ctx.WhereBody())
		e.emit("not")
		return nil
	}
	e.Visit(ctx.Eexpr(1))
	e.emit("entitypush")
	e.Visit(ctx.WhereBody())
	e.emit("entitypop")
	e.emit("swap")
	e.emit("pop")
	e.emit("not")
	return nil
}

func (e *PostfixEmitter) VisitBoolThereIsInArrayWhere(ctx *BoolThereIsInArrayWhereContext) interface{} {
	// Block before array — opForall is ( body array -- ). See #877.
	e.emit("false")
	e.emit("{")
	e.Visit(ctx.WhereBody())
	e.emit("or")
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forall")
	return nil
}

func (e *PostfixEmitter) VisitBoolThereIsNoInArrayWhere(ctx *BoolThereIsNoInArrayWhereContext) interface{} {
	// Block before array — opForall is ( body array -- ). See #877.
	e.emit("false")
	e.emit("{")
	e.Visit(ctx.WhereBody())
	e.emit("or")
	e.emit("}")
	e.Visit(ctx.ArrayExpr())
	e.emit("forall")
	e.emit("not")
	return nil
}

// VisitBoolStartsWithAt: `<s1> at <i> starts with <s2>` → extract the
// substring of s1 from position i to end, then startswith s2. opSubstring
// takes (str start length); we compute length as stringlength(s1) - i.
// To avoid evaluating s1 twice, dup it.
func (e *PostfixEmitter) VisitBoolStartsWithAt(ctx *BoolStartsWithAtContext) interface{} {
	e.Visit(ctx.Strexpr(0))
	e.emit("dup")
	e.emit("stringlength")
	e.Visit(ctx.Iexpr())
	e.emit("-")
	e.Visit(ctx.Iexpr())
	e.emit("swap")
	e.emit("substring")
	e.Visit(ctx.Strexpr(1))
	e.emit("startswith")
	return nil
}

func (e *PostfixEmitter) VisitBoolEntityHasaWhere(ctx *BoolEntityHasaWhereContext) interface{} {
	e.emit("{")
	e.Visit(ctx.WhereBody())
	e.emit("}")
	e.emit("{")
	e.emit("false")
	e.emit("}")
	e.Visit(ctx.Eexpr())
	e.Visit(ctx.Strexpr())
	e.emit("hasrelationship")
	e.emit("ifelse")
	return nil
}

// VisitBoolPlusOrMinus: `<f1> is plus or minus <n> of <f2>` →
// `|f1 - f2| <= n`. The `number` may be int or float; `cvd` coerces to
// double for the comparison.
func (e *PostfixEmitter) VisitBoolPlusOrMinus(ctx *BoolPlusOrMinusContext) interface{} {
	e.Visit(ctx.Fexpr(0))
	e.Visit(ctx.Fexpr(1))
	e.emit("f-")
	e.emit("fabs")
	e.Visit(ctx.Number())
	e.emit("cvd")
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
	e.emit("cvd")
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

// VisitOperatorstatements: `<typedOperator> ( <operatorlist> )` — push all
// args (operatorlist is a comma-separated list of exprs, emitted in order),
// then the operator name as an executable token. The runtime's executable-
// name dispatch finds the op and invokes it.
func (e *PostfixEmitter) VisitOperatorstatements(ctx *OperatorstatementsContext) interface{} {
	name := ctx.TypedOperator().GetText()

	// Reject a name the engine does not implement, here rather than at
	// execution.
	//
	// Nothing on this path used to check the registry, so `subests(...)` for
	// `subsets(...)` compiled clean, wrote postfix, passed build, and failed
	// only when that row ran — as "The Name 'subests' was not defined by any
	// Entity on the Entity Stack", which does not read like a typo (#1020).
	// For a rule set computing money that is a defect discovered mid-period.
	//
	// Checked against the registry directly: operators does not import el, so
	// there is no cycle, and a compiler that emits operator names should know
	// which ones exist. An injected checker would be forgettable, which is how
	// the EL compiler came to be missing from `sync import` in #929.
	if e.operatorExists != nil && !e.operatorExists(name) {
		e.emitError("unknown operator %q — it is not registered with the engine; "+
			"check the spelling, or see `dtrules docs operators` for the list", name)
		return nil
	}

	// And the count. The name check catches typos; a short call is the quiet
	// failure. Every argument after the source is a bare string or an array,
	// so `subsets(hand.cards)` with one of four compiles clean, then pops
	// whatever three values sit beneath it and reads them as typename,
	// sumfield and destination -- a runtime error, a write into the wrong
	// array, or a plausible-looking wrong answer, depending on what the row
	// did before (#1105).
	if e.operatorArity != nil {
		if want := e.operatorArity(name); want > 0 {
			if got := countOperatorArgs(ctx.Operatorlist()); got != want {
				e.emitError("operator %q takes %d arguments, got %d — "+
					"see `dtrules docs operators` for the argument order", name, want, got)
				return nil
			}
		}
	}

	e.Visit(ctx.Operatorlist())
	e.emit(name)
	return nil
}

// countOperatorArgs counts the arguments in a statement-form call.
//
// The list is right-recursive: each element node carries one expression and
// the tail, and the `...Single` variants are the last element. Counting the
// chain rather than visiting it means no postfix is emitted for a call that is
// about to be rejected.
func countOperatorArgs(ctx IOperatorlistContext) int {
	n := 0
	for ctx != nil {
		switch c := ctx.(type) {
		case *OpListFloatSingleContext, *OpListIntSingleContext, *OpListStrSingleContext:
			return n + 1
		case *OpListFloatContext:
			n++
			ctx = c.Operatorlist()
		case *OpListIntContext:
			n++
			ctx = c.Operatorlist()
		case *OpListStrContext:
			n++
			ctx = c.Operatorlist()
		default:
			// An alternative this does not know: count it as one argument and
			// stop, so an unrecognised shape cannot manufacture a mismatch.
			return n + 1
		}
	}
	return n
}

// VisitRemoveEachWhere: `remove each <eexpr> from <arrayExpr> where <bexpr>`
// — iterate in reverse (forallr is safer under removal), and for each
// element whose bexpr holds, call `remove(arr, element)`. We keep a second
// copy of the array on the data stack below forallr's operands so the body
// can reach it with `dup`; the element comes from the entity stack via
// `0 entityfetch`.
func (e *PostfixEmitter) VisitRemoveEachWhere(ctx *RemoveEachWhereContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.emit("dup")
	e.emit("{")
	e.Visit(ctx.Bexpr())
	e.emit("{")
	e.emit("dup")
	e.emit("0")
	e.emit("entityfetch")
	e.emit("remove")
	e.emit("}")
	e.emit("if")
	e.emit("}")
	e.emit("swap")
	e.emit("forallr")
	e.emit("pop")
	return nil
}

// VisitBoolMatchForall: `there is match for all <arr1> to <nexpr> in <arr2>`
// — every element of arr1 has a match in arr2 via the nexpr attribute.
// Emitted as nested folds: outer AND-accumulator over arr1, inner OR-
// accumulator over arr2 comparing y.<nexpr> to x (fetched from entity
// stack at depth 1). Semantic approximation; runtime correctness depends
// on the specific types of arr1/arr2/nexpr.
func (e *PostfixEmitter) VisitBoolMatchForall(ctx *BoolMatchForallContext) interface{} {
	// Both folds emit the block before the array — opForall is
	// ( body array -- ). Reordering tokens does not change the runtime
	// entity-stack depth (forall pushes each element while running the
	// body), so `entityfetch 1` still resolves the outer element. See #877.
	e.emit("true")
	e.emit("{")
	// Inner existence check over arr2
	e.emit("false")
	e.emit("{")
	e.Visit(ctx.Nexpr()) // y.<nexpr>
	e.emit("1")          // depth 1 = outer element x
	e.emit("entityfetch")
	e.emit("==")
	e.emit("or")
	e.emit("}")
	e.Visit(ctx.ArrayExpr(1))
	e.emit("forall")
	// AND with outer accumulator
	e.emit("and")
	e.emit("}")
	e.Visit(ctx.ArrayExpr(0))
	e.emit("forall")
	return nil
}

// VisitPerformCatchError: `perform <T1> and onerror add <e> to context
// and perform <T2>`. opPerformCatchError signature is (table errtable
// errentity --), so push /<T1>, /<T2>, /<e> as literal names then call the op.
func (e *PostfixEmitter) VisitPerformCatchError(ctx *PerformCatchErrorContext) interface{} {
	e.emit("/" + ctx.TypedDecisionTable(0).GetText())
	e.emit("/" + ctx.TypedDecisionTable(1).GetText())
	e.emit("/" + ctx.Eexpr().GetText())
	e.emit("performcatcherror")
	return nil
}

// XML DOM mutation statements. No runtime support — emit elstmterror so the
// forms parse but fail loudly at runtime.
func (e *PostfixEmitter) VisitXmlSetAttr(ctx *XmlSetAttrContext) interface{} {
	e.emit("\"xml mutation unsupported — xmlvaluestatements have no runtime\"")
	e.emit("elstmterror")
	return nil
}
func (e *PostfixEmitter) VisitXmlSetAttrEntity(ctx *XmlSetAttrEntityContext) interface{} {
	e.emit("\"xml mutation unsupported — xmlvaluestatements have no runtime\"")
	e.emit("elstmterror")
	return nil
}
func (e *PostfixEmitter) VisitXmlAddAttr(ctx *XmlAddAttrContext) interface{} {
	e.emit("\"xml mutation unsupported — xmlvaluestatements have no runtime\"")
	e.emit("elstmterror")
	return nil
}
func (e *PostfixEmitter) VisitXmlAddAttrEntity(ctx *XmlAddAttrEntityContext) interface{} {
	e.emit("\"xml mutation unsupported — xmlvaluestatements have no runtime\"")
	e.emit("elstmterror")
	return nil
}

// Policy statement dispatch. The runtime's `opPolicyStatements` was removed
// (#669) — authoring-side `policystatement <expr>;` now compiles to evaluate
// the expression and discard the result. Policy-statement collection is
// handled cmd-side from the XML, not at runtime.
func (e *PostfixEmitter) VisitPolicyBExpr(ctx *PolicyBExprContext) interface{} {
	e.Visit(ctx.Bexpr())
	e.emit("pop")
	return nil
}
func (e *PostfixEmitter) VisitPolicyIExpr(ctx *PolicyIExprContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.emit("pop")
	return nil
}
func (e *PostfixEmitter) VisitPolicyFExpr(ctx *PolicyFExprContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.emit("pop")
	return nil
}
func (e *PostfixEmitter) VisitPolicyDExpr(ctx *PolicyDExprContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.emit("pop")
	return nil
}
func (e *PostfixEmitter) VisitPolicyNExpr(ctx *PolicyNExprContext) interface{} {
	e.Visit(ctx.Nexpr())
	e.emit("pop")
	return nil
}
func (e *PostfixEmitter) VisitPolicyStrExpr(ctx *PolicyStrExprContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("pop")
	return nil
}

// Debug-statement wiring inside done-level alternatives. The grammar allows
// `condition debug "msg"; <cond>;` and `condition <cond>; debug "msg";` and
// the context analog. Each emits the debug statement first (or after) then
// the main body, separated by the explicit semicolon which the statement
// list handles.
func (e *PostfixEmitter) VisitConditionDebugBefore(ctx *ConditionDebugBeforeContext) interface{} {
	e.Visit(ctx.Debugstatement())
	e.Visit(ctx.Bexpr())
	return nil
}
func (e *PostfixEmitter) VisitConditionDebugAfter(ctx *ConditionDebugAfterContext) interface{} {
	e.Visit(ctx.Bexpr())
	e.Visit(ctx.Debugstatement())
	return nil
}
func (e *PostfixEmitter) VisitContextDebugBefore(ctx *ContextDebugBeforeContext) interface{} {
	e.Visit(ctx.Debugstatement())
	e.Visit(ctx.ContextForTable())
	return nil
}

// Operatorlist helpers — labeled alternatives need explicit Visit methods
// to cascade properly through ANTLR's override dispatch (same bug class as
// #626 / forallctl). Each variant emits its expression(s) in order.
func (e *PostfixEmitter) VisitOpListFloatSingle(ctx *OpListFloatSingleContext) interface{} {
	return e.Visit(ctx.Fexpr())
}
func (e *PostfixEmitter) VisitOpListIntSingle(ctx *OpListIntSingleContext) interface{} {
	return e.Visit(ctx.Iexpr())
}
func (e *PostfixEmitter) VisitOpListStrSingle(ctx *OpListStrSingleContext) interface{} {
	return e.Visit(ctx.Strexpr())
}
func (e *PostfixEmitter) VisitOpListFloat(ctx *OpListFloatContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.Visit(ctx.Operatorlist())
	return nil
}
func (e *PostfixEmitter) VisitOpListInt(ctx *OpListIntContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.Visit(ctx.Operatorlist())
	return nil
}
func (e *PostfixEmitter) VisitOpListStr(ctx *OpListStrContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.Visit(ctx.Operatorlist())
	return nil
}

// `<type> VALUE OF <operatorstatements>` — evaluate the operator call and
// coerce the result to the named type.
func (e *PostfixEmitter) VisitBoolValueOfOp(ctx *BoolValueOfOpContext) interface{} {
	e.Visit(ctx.Operatorstatements())
	e.emit("cvb")
	return nil
}
func (e *PostfixEmitter) VisitIntValueOfOp(ctx *IntValueOfOpContext) interface{} {
	e.Visit(ctx.Operatorstatements())
	e.emit("cvi")
	return nil
}
func (e *PostfixEmitter) VisitFloatValueOfOp(ctx *FloatValueOfOpContext) interface{} {
	e.Visit(ctx.Operatorstatements())
	e.emit("cvd")
	return nil
}
func (e *PostfixEmitter) VisitStrValueOfOp(ctx *StrValueOfOpContext) interface{} {
	e.Visit(ctx.Operatorstatements())
	e.emit("cvs")
	return nil
}

// VisitStrTableLookup / VisitTableNew — the hash-table ops were removed;
// the grammar still parses these forms. Emit an elstmterror so the compile
// succeeds (non-empty postfix) but the runtime raises a clear error.

func (e *PostfixEmitter) VisitStrTableLookup(ctx *StrTableLookupContext) interface{} {
	e.emit("\"hash tables removed — (string) table-lookup unsupported\"")
	e.emit("elstmterror")
	e.emit("\"\"")
	return nil
}

func (e *PostfixEmitter) VisitTableNew(ctx *TableNewContext) interface{} {
	e.emit("\"hash tables removed — `new X table of Y` unsupported\"")
	e.emit("elstmterror")
	e.emit("newarray")
	return nil
}

// =============================================================================
// #803 batch 6: rest of the table-lookup family. Hash tables were
// removed; the grammar still parses these forms so emit elstmterror
// placeholders symmetric to VisitStrTableLookup / VisitTableNew above.
// Each leaves a type-appropriate sentinel value on the stack so the
// surrounding expression compiles even though the runtime won't run.
// =============================================================================

func (e *PostfixEmitter) VisitFloatTableLookup(ctx *FloatTableLookupContext) interface{} {
	e.emit("\"hash tables removed — (double) table-lookup unsupported\"")
	e.emit("elstmterror")
	e.emit("0.0")
	return nil
}

func (e *PostfixEmitter) VisitIntTableLookup(ctx *IntTableLookupContext) interface{} {
	e.emit("\"hash tables removed — (long) table-lookup unsupported\"")
	e.emit("elstmterror")
	e.emit("0")
	return nil
}

func (e *PostfixEmitter) VisitEntityTableLookup(ctx *EntityTableLookupContext) interface{} {
	e.emit("\"hash tables removed — (entity) table-lookup unsupported\"")
	e.emit("elstmterror")
	e.emit("null")
	return nil
}

func (e *PostfixEmitter) VisitSetTable(ctx *SetTableContext) interface{} {
	e.emit("\"hash tables removed — `set <table> = <table>` unsupported\"")
	e.emit("elstmterror")
	return nil
}

// VisitStrTableInfo: the `tableinformation` keyword. Hash tables were
// removed; this leaves a sentinel string on the stack for the surrounding
// expression and errors loudly at runtime.
func (e *PostfixEmitter) VisitStrTableInfo(ctx *StrTableInfoContext) interface{} {
	e.emit("\"hash tables removed — `tableinformation` unsupported\"")
	e.emit("elstmterror")
	e.emit("\"\"")
	return nil
}

// VisitIntUsingArray: `USING arrayExpr number` — ANTLR adaptive
// prediction matches this for the `using <ident>(<expr>)` shape because
// arrayExpr accepts any IDENT and `(<expr>)` matches as `number`. The
// semantic intent is the entity-stack delegation pattern (see
// VisitBigUsing): push the entity, evaluate the inner expression in
// that context, pop.
func (e *PostfixEmitter) VisitIntUsingArray(ctx *IntUsingArrayContext) interface{} {
	e.Visit(ctx.ArrayExpr())
	e.emit("entitypush")
	e.Visit(ctx.Number())
	e.emit("entitypop")
	return nil
}

// VisitBoolFunction: `<typedBoolFunction>` as a boolean expression —
// a niladic boolean function call. Mirrors VisitOperatorstatements'
// shape (emit the name as executable) but with no argument list.
// Without this override, `bexpr: ... | typedBoolFunction # boolFunction`
// silently emitted nothing.
func (e *PostfixEmitter) VisitBoolFunction(ctx *BoolFunctionContext) interface{} {
	e.emit(ctx.TypedBoolFunction().GetText())
	return nil
}

// colonRefEntityName extracts the entity name from a colonRef AST node,
// handling both the possessive-chain (`Client's`) and the colon-chain
// (`:Client:`) forms. Returns "" if neither shape matches.
func colonRefEntityName(colonRef IColonRefContext) string {
	if colonRef == nil {
		return ""
	}
	possRef := colonRef.PossessiveRef()
	if possRef == nil {
		return ""
	}
	switch n := possRef.(type) {
	case *PossessiveChainContext:
		tokens := n.AllPOSSESSIVE()
		if len(tokens) > 0 {
			text := tokens[0].GetText()
			if strings.HasSuffix(text, "'s") {
				return text[:len(text)-2]
			}
			return text
		}
	case *ColonChainContext:
		if te := n.TypedEntity(); te != nil {
			return te.GetText()
		}
	}
	return ""
}

// VisitSubDestColon: `subtract <number> from <colonRef> <field>` —
// mirrors VisitAddDestColon for subtraction. Pre-fix the entire
// destination part was dropped (parent VisitSubtractNum emitted the
// number but Visit(subtodest) hit BaseELVisitor's no-op). Matches
// VisitSubDestPossessiveLong's `<field> - /<field> xdef` emission so
// the runtime computes `value - field`.
func (e *PostfixEmitter) VisitSubDestColon(ctx *SubDestColonContext) interface{} {
	entityName := colonRefEntityName(ctx.ColonRef())
	if entityName != "" {
		if !e.emitLocalRef(entityName) {
			e.emit(entityName)
		}
	}
	e.emit("entitypush")

	addDest2 := ctx.Addtodest2()
	var fieldName string
	switch d := addDest2.(type) {
	case *AddDestLong2Context:
		fieldName = d.TypedLong().GetText()
	case *AddDestDouble2Context:
		fieldName = d.TypedDouble().GetText()
	case *AddDestArray2Context:
		fieldName = d.ArrayExpr2().GetText()
	}
	if fieldName != "" {
		e.emit(fieldName)
		e.emit("-")
		e.emit("/" + fieldName)
		e.emit("xdef")
	}
	e.emit("entitypop")
	return nil
}

// VisitBoolEntityIsOf: `<e1> is the <R> of <e2>` — true when e2's R field
// holds e1.
//
//	the client is the parent of ApplyingClient   ->   ApplyingClient.parent == client
//
// That is all the relationship means: the entity is held by the named field
// of the other entity. The form used to call a findmatch-era lookup that went
// away with the hash-table ops, and since then it emitted an elstmterror so
// the row parsed and died at runtime (#927).
//
// It compiles to the same `getrelationship` the `"role" of entity` form uses
// — that operator reads the named field off an entity, which is the whole of
// the semantics:
//
//	<e2> "<R>" getrelationship <e1> req
func (e *PostfixEmitter) VisitBoolEntityIsOf(ctx *BoolEntityIsOfContext) interface{} {
	field := relationshipFieldName(ctx.Strexpr().GetText())
	if field == "" {
		e.emitError("relationship name is empty in `is ... of`")
		e.emit("false")
		return nil
	}

	e.Visit(ctx.Eexpr(1))
	e.emit("\"" + field + "\"")
	e.emit("getrelationship")
	e.Visit(ctx.Eexpr(0))
	e.emit("req")
	return nil
}

// relationshipFieldName strips the article and any quoting from the
// relationship in `is the <R> of`, leaving the field name to read.
func relationshipFieldName(text string) string {
	name := strings.TrimSpace(text)
	name = strings.Trim(name, "\"'")
	for _, article := range []string{"the ", "The ", "a ", "an "} {
		name = strings.TrimPrefix(strings.TrimSpace(name), article)
	}
	return strings.TrimSpace(name)
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
	// No `subdays` op exists; negate the count and reuse adddays (#888).
	e.emit("negate")
	e.emit("adddays")
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
	e.emit("negate")
	e.emit("addmonths")
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
	e.emit("negate")
	e.emit("addyears")
	return nil
}

// ============================================================================
// Add Statement Visitors
// ============================================================================

func (e *PostfixEmitter) VisitAddArrayToArray(ctx *AddArrayToArrayContext) interface{} {
	// The parser steers `add <value> to <field>` through this visitor when
	// both sides look like arrayExpr (the first matching alternative in the
	// addtostatement rule). If the destination actually resolves to a
	// declared numeric field, emit the typed store-back via the shared
	// helper so fixed and bigint targets don't get plain `+` (which would
	// truncate via LongValue at runtime).

	destExpr := ctx.ArrayExpr(1)

	// Possessive pattern: `add X to <entity>'s <field>` →
	//   <value> <entity> entitypush <addSubSequence> entitypop
	if colonRefCtx, ok := destExpr.(*ArrayColonRefContext); ok {
		fieldName := colonRefCtx.TypedArray().GetText()
		if isNumericType(e.lookupType(fieldName)) {
			if vt := e.getExprType(ctx.ArrayExpr(0)); isDoubleExactMix(vt, e.lookupType(fieldName)) {
				e.emitDoubleMixError(exactOf(vt, e.lookupType(fieldName)))
				return nil
			}
			e.Visit(ctx.ArrayExpr(0))
			if possChain, ok := colonRefCtx.ColonRef().PossessiveRef().(*PossessiveChainContext); ok {
				tokens := possChain.AllPOSSESSIVE()
				if len(tokens) > 0 {
					poss := tokens[0].GetText()
					entityName := poss[:len(poss)-2] // strip 's
					if !e.emitLocalRef(entityName) {
						e.emit(entityName)
					}
				}
			}
			e.emit("entitypush")
			e.emitTypeAwareAddSub(fieldName, "+")
			e.emit("entitypop")
			return nil
		}
	}

	// Bare-IDENT numeric destination: `add X to <field>` →
	//   <value> <addSubSequence>
	if baseCtx, ok := destExpr.(*ArrayBaseContext); ok {
		if arrayExpr2 := baseCtx.ArrayExpr2(); arrayExpr2 != nil {
			if typedCtx, ok := arrayExpr2.(*ArrayTypedContext); ok {
				fieldName := typedCtx.TypedArray().GetText()
				if isNumericType(e.mutationType(fieldName)) {
					if vt := e.getExprType(ctx.ArrayExpr(0)); isDoubleExactMix(vt, e.mutationType(fieldName)) {
						e.emitDoubleMixError(exactOf(vt, e.mutationType(fieldName)))
						return nil
					}
					e.Visit(ctx.ArrayExpr(0))
					e.emitTypeAwareAddSub(fieldName, "+")
					return nil
				}
			}
		}
	}

	// Determine if source is a single entity or an array. The parser sees
	// `ADD <IDENT> TO <IDENT>` as arrayExpr TO arrayExpr because
	// typedEntity and typedArray both come from IDENT — so this visitor is
	// the default landing spot for both `add entity X to array Y` and
	// `add array A to array B`. Use the symbol table to distinguish when
	// present. If unknown, default to the single-element-to-array pattern
	// (addto) since that's the overwhelmingly common case in practice;
	// array-to-array merge requires explicitly typed arrays in the EDD.
	srcExpr := ctx.ArrayExpr(0)
	srcIsArray := false // Default to entity-to-array (safer)

	// `the policy statements` is an array with no name to look up, so the
	// symbol-table test below can never see it. Without this it fell to the
	// single-element branch and a report got the whole accumulator as one
	// blob instead of its statements (#956).
	if _, ok := srcExpr.(*ArrayPolicyStatementsContext); ok {
		srcIsArray = true
	}

	if baseCtx, ok := srcExpr.(*ArrayBaseContext); ok {
		if arrayExpr2 := baseCtx.ArrayExpr2(); arrayExpr2 != nil {
			if typedCtx, ok := arrayExpr2.(*ArrayTypedContext); ok {
				srcName := typedCtx.TypedArray().GetText()
				srcType := e.lookupType(srcName)
				// If source is explicitly an array, use array merge.
				if srcType == TypeArray {
					srcIsArray = true
				}
			}
		}
	}

	// Single-element → array: emit `swap addto`.
	if !srcIsArray {
		e.Visit(ctx.ArrayExpr(0))
		e.Visit(ctx.ArrayExpr(1))
		e.emit("swap")
		e.emit("addto")
		return nil
	}

	// Array-to-array merge (source is declared array in the EDD).
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
	// The destination visitor (VisitAddDestArray for the common array case)
	// emits its own `swap addto` after pushing the array — emitting it here
	// too produced a duplicate trailer that corrupted the stack on
	// `add new T entity to <array>` (#904, same class as the #781 string
	// fix). Bare-IDENT entity values match the arrayExpr alternatives
	// instead, so this visitor is only reached for constructor-shaped
	// eexprs.
	e.Visit(ctx.Eexpr())
	e.Visit(ctx.Addtodest())
	return nil
}

func (e *PostfixEmitter) VisitAddStrToDest(ctx *AddStrToDestContext) interface{} {
	// The destination visitor (VisitAddDestArray for the common array
	// case) emits its own `swap addto` after pushing the array — emitting
	// it here too produced a duplicate trailer that crashed the runtime
	// on `add "x" to <array>` (#781 fallout). Mirrors VisitAddNumToDest's
	// shape, which has always relied on the dest visitor for the store.
	e.Visit(ctx.Strexpr())
	e.Visit(ctx.Addtodest())
	return nil
}

func (e *PostfixEmitter) VisitAddNumToDest(ctx *AddNumToDestContext) interface{} {
	// Pattern: value field + /field xdef
	// e.g., "add 5 to client.income" => "5 client.income + /client.income xdef"
	if e.rejectMutationDoubleMix(ctx.Number(), ctx.Addtodest()) {
		return nil
	}
	e.Visit(ctx.Number())
	e.Visit(ctx.Addtodest())
	return nil
}

// rejectMutationDoubleMix records the #876 error and returns true when a
// field mutation (`add/subtract <value> to/from <field>`) would fold a double
// value into a fixed or bigint field — the mutation analogue of the binary-op
// reject (#882). The dest's text is the bare field name for the common simple
// target; a complex dest (possessive/colon) that doesn't resolve is left to
// the existing snap rather than risk a false positive.
func (e *PostfixEmitter) rejectMutationDoubleMix(value, dest antlr.ParseTree) bool {
	if dest == nil || value == nil {
		return false
	}
	vt := e.getExprType(value)
	dt := e.mutationType(dest.GetText())
	if isDoubleExactMix(vt, dt) {
		e.emitDoubleMixError(exactOf(vt, dt))
		return true
	}
	return false
}

// isNumericType reports whether a declared EDD type string is one of the
// four numeric types the field-mutation visitors know how to dispatch on.
func isNumericType(t string) bool {
	switch t {
	case TypeInteger, TypeLong, TypeBigInt, TypeDouble, TypeFixed:
		return true
	}
	return false
}

// VisitAddDestArray is the default landing spot for bare-IDENT add-to
// targets: `arrayExpr2` wins the grammar alternative ordering over
// typedLong/typedDouble. Dispatch here on the declared field type:
//
//   - numeric (integer / bigint / double / fixed): emit the type-aware
//     store-back sequence via emitTypeAwareAddSub.
//   - array or unknown: fall through to the single-element-into-array
//     pattern (`value array swap addto`).
//
// Before this fix, all bare-IDENT numeric targets compiled to just
// `value field` with no op or xdef — the statement was essentially a no-op
// at runtime and the field was never updated.
func (e *PostfixEmitter) VisitAddDestArray(ctx *AddDestArrayContext) interface{} {
	if arrTyped, ok := ctx.ArrayExpr2().(*ArrayTypedContext); ok {
		name := arrTyped.TypedArray().GetText()
		switch e.mutationType(name) {
		case TypeInteger, TypeLong, TypeBigInt, TypeDouble, TypeFixed:
			e.emitTypeAwareAddSub(name, "+")
			return nil
		}
	}
	// Real array target (or unknown; default to single-element append for
	// consistency with VisitAddArrayToArray's overwhelmingly-common case).
	e.Visit(ctx.ArrayExpr2())
	e.emit("swap")
	e.emit("addto")
	return nil
}

func (e *PostfixEmitter) VisitAddDestLong(ctx *AddDestLongContext) interface{} {
	e.emitTypeAwareAddSub(ctx.TypedLong().GetText(), "+")
	return nil
}

func (e *PostfixEmitter) VisitAddDestDouble(ctx *AddDestDoubleContext) interface{} {
	e.emitTypeAwareAddSub(ctx.TypedDouble().GetText(), "+")
	return nil
}

func (e *PostfixEmitter) VisitAddDestColon(ctx *AddDestColonContext) interface{} {
	// Pattern: <entity-ref> entitypush <field> + /<field> xdef entitypop
	// The colonRef contains either a possessive (`ThisClient's`) or a
	// colon chain (`:ThisClient:`); both extract to an entity name.
	// The addtodest2 contains the field (e.g., "IncomeGroupCount").
	entityName := colonRefEntityName(ctx.ColonRef())
	if entityName != "" {
		if !e.emitLocalRef(entityName) {
			e.emit(entityName)
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

// =============================================================================
// #803 batch 4: colon-ref field access. Mirrors the existing
// VisitBigColonRef / VisitBytesColonRef pattern across all typed
// variants. Pre-fix, `:Client:fee > 0` and `:Client:active is true`
// produced empty LHS in postfix because the matching XxxColonRef alt
// had no override and antlr's VisitChildren is a no-op.
//
// Confirmed reachable by parse-tree inspection:
//   - intColonRef:  `:Client:<numeric-field>` (the most common case;
//                   typedLong wins because IDENT matches across types)
//   - boolColonRef: `:Client:<bool-field> is true`
//
// Likely dead grammar for the others (Float/Date/Str/Name/Array/Name
// don't appear as the parser's first-match for IDENT-prefixed forms).
// Adding all 7 anyway as defensive overrides; the visitors share the
// same simple emission shape so the cost is small.
// =============================================================================

func (e *PostfixEmitter) VisitIntColonRef(ctx *IntColonRefContext) interface{} {
	e.Visit(ctx.ColonRef())
	e.Visit(ctx.TypedLong())
	return nil
}

func (e *PostfixEmitter) VisitFloatColonRef(ctx *FloatColonRefContext) interface{} {
	e.Visit(ctx.ColonRef())
	e.Visit(ctx.TypedDouble())
	return nil
}

func (e *PostfixEmitter) VisitBoolColonRef(ctx *BoolColonRefContext) interface{} {
	e.Visit(ctx.ColonRef())
	e.Visit(ctx.TypedBoolean())
	return nil
}

func (e *PostfixEmitter) VisitDateColonRef(ctx *DateColonRefContext) interface{} {
	e.Visit(ctx.ColonRef())
	e.Visit(ctx.TypedDate())
	return nil
}

// VisitStrColonRef: `colonRef strexpr` — note this takes a full strexpr,
// not a typed-IDENT like the others. In practice the parser doesn't
// reach this alt because the simpler typed forms win first; included
// for grammar completeness and defensive depth.
func (e *PostfixEmitter) VisitStrColonRef(ctx *StrColonRefContext) interface{} {
	e.Visit(ctx.ColonRef())
	e.Visit(ctx.Strexpr())
	return nil
}

func (e *PostfixEmitter) VisitNameColonRef(ctx *NameColonRefContext) interface{} {
	e.Visit(ctx.ColonRef())
	e.Visit(ctx.TypedName())
	return nil
}

func (e *PostfixEmitter) VisitArrayColonRef(ctx *ArrayColonRefContext) interface{} {
	e.Visit(ctx.ColonRef())
	e.Visit(ctx.TypedArray())
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

// =============================================================================
// #803 batch 5: date constructor/cast family. Each pre-fix produced empty
// RHS — `set a.d = (date) "2026-01-15"` and friends emitted only the
// assignment trailer `cvdate /a.d xdef`. The inner expression was dropped
// because antlr's BaseParseTreeVisitor.VisitChildren is a no-op.
// =============================================================================

// VisitDateFromStrCast: `(date) <strexpr>` — explicit string→date cast.
func (e *PostfixEmitter) VisitDateFromStrCast(ctx *DateFromStrCastContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("cvdate")
	return nil
}

// VisitDateFromStrFunc: `date(<strexpr>)` — function-call cast form.
// Emission identical to the prefix-cast form.
func (e *PostfixEmitter) VisitDateFromStrFunc(ctx *DateFromStrFuncContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("cvdate")
	return nil
}

// VisitDateFromIndex: `(date) <arrayExpr>[<iexpr>]` — cast an indexed
// expression. The IndxExpr visitor (batch 1) emits the
// `<array> <index> bytesidx` sequence; we append cvdate.
func (e *PostfixEmitter) VisitDateFromIndex(ctx *DateFromIndexContext) interface{} {
	e.Visit(ctx.IndxExpr())
	e.emit("cvdate")
	return nil
}

// VisitDateFromArrayAt: `(date) <typedArray>[<iexpr>]` — same shape as
// FromIndex but the grammar inlines the array+index instead of going
// through indxExpr, so we emit the bytesidx ourselves.
func (e *PostfixEmitter) VisitDateFromArrayAt(ctx *DateFromArrayAtContext) interface{} {
	e.Visit(ctx.TypedArray())
	e.Visit(ctx.Iexpr())
	e.emit("bytesidx")
	e.emit("cvdate")
	return nil
}

// VisitDateUsing: `using <eexpr> (<dexpr>)` — evaluate the date
// expression with eexpr pushed onto the entity stack. Mirrors the
// existing VisitBigUsing pattern: entitypush before the inner
// expression, entitypop after.
func (e *PostfixEmitter) VisitDateUsing(ctx *DateUsingContext) interface{} {
	e.Visit(ctx.Eexpr())
	e.emit("entitypush")
	e.Visit(ctx.Dexpr())
	e.emit("entitypop")
	return nil
}

// VisitDateTableLookup: `(date) <typedTable>(<tablelist>)` — hash-table
// lookup. The hash-table ops were removed (see VisitStrTableLookup /
// VisitTableNew); the grammar still parses these forms. Emit an
// elstmterror so the compile produces non-empty postfix but the runtime
// raises a clear error — same pattern as the StrTableLookup placeholder.
func (e *PostfixEmitter) VisitDateTableLookup(ctx *DateTableLookupContext) interface{} {
	e.emit("\"hash tables removed — (date) table-lookup unsupported\"")
	e.emit("elstmterror")
	e.emit("today")
	return nil
}

func (e *PostfixEmitter) VisitBigAbs(ctx *BigAbsContext) interface{} {
	e.Visit(ctx.Bigexpr())
	e.emit("babs")
	return nil
}

func (e *PostfixEmitter) VisitTypedBigInt(ctx *TypedBigIntContext) interface{} {
	name := ctx.GetText()
	if e.emitAliasFieldAccess(name) {
		return nil
	}
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
	fieldType := e.resolveSetTarget(ctx.LeftBigexpr().GetText(), TypeBigInt)
	if conv := e.typeConverter(fieldType); conv != "" {
		e.emit(conv)
	}
	e.Visit(ctx.LeftBigexpr())
	return nil
}

func (e *PostfixEmitter) VisitLeftBigexprSimple(ctx *LeftBigexprSimpleContext) interface{} {
	e.emitFieldStore(ctx.TypedBigInt().GetText())
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
	if e.emitAliasFieldAccess(name) {
		return nil
	}
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

// ============================================================================
// Fixed-Point Expression Visitors
// ============================================================================

// VisitFixedLiteral emits an FP_LITERAL token verbatim (e.g. `1.5fp`). The
// runtime compiler recognizes the `fp` suffix and constructs an RFixed.
func (e *PostfixEmitter) VisitFixedLiteral(ctx *FixedLiteralContext) interface{} {
	e.emit(ctx.GetText())
	return nil
}

func (e *PostfixEmitter) VisitFixedFromStr(ctx *FixedFromStrContext) interface{} {
	e.Visit(ctx.Strexpr())
	e.emit("cvfp")
	return nil
}

func (e *PostfixEmitter) VisitFixedFromNumber(ctx *FixedFromNumberContext) interface{} {
	e.Visit(ctx.Iexpr())
	e.emit("cvfp")
	return nil
}

func (e *PostfixEmitter) VisitFixedFromFloat(ctx *FixedFromFloatContext) interface{} {
	e.Visit(ctx.Fexpr())
	e.emit("cvfp")
	return nil
}

func (e *PostfixEmitter) VisitFixedFromIndex(ctx *FixedFromIndexContext) interface{} {
	e.Visit(ctx.IndxExpr())
	e.emit("cvfp")
	return nil
}

// ============================================================================
// Fixed-Point Local Variable Declaration Visitors
// ============================================================================

func (e *PostfixEmitter) VisitLocalFixedUndef(ctx *LocalFixedUndefContext) interface{} {
	name := ctx.UndefinedIdent().GetText()
	e.declareLocal(name, TypeFixed)
	e.emit("null")
	e.emit("allocate")
	e.emit("execute")
	e.emit("deallocate")
	e.emit("pop")
	return nil
}

func (e *PostfixEmitter) VisitLocalFixedInit(ctx *LocalFixedInitContext) interface{} {
	name := ctx.UndefinedIdent().GetText()
	e.declareLocal(name, TypeFixed)
	e.emitWithTypeConversion(ctx.Iexpr(), TypeFixed)
	e.emit("allocate")
	e.emit("execute")
	e.emit("deallocate")
	e.emit("pop")
	return nil
}

func (e *PostfixEmitter) VisitLocalFixedDefined(ctx *LocalFixedDefinedContext) interface{} {
	e.emit(ctx.TypedLong().GetText())
	return nil
}

// Phase 4 of #743: format(date, layout) [in zone Z] visitors. Stack order
// matches the runtime ops: date layout [zone] -- string.

func (e *PostfixEmitter) VisitStrFormatDate(ctx *StrFormatDateContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Strexpr())
	e.emit("dateformat")
	return nil
}

func (e *PostfixEmitter) VisitStrFormatDateInZone(ctx *StrFormatDateInZoneContext) interface{} {
	e.Visit(ctx.Dexpr())
	e.Visit(ctx.Strexpr(0))
	e.Visit(ctx.Strexpr(1))
	e.emit("dateformatinzone")
	return nil
}

// Phase 4 of #743: `new date Y, M, D[, h, m, s] in zone Z [with dst_rule R]`
// constructor visitors. Hour/minute/second default to 0 in the YMD-only
// forms; the runtime op signature is the same so the emitter pushes
// literal zeros to fill the slots.
//
// Stack signature for newdateinzone:           ( y mo d h mi s zone )
// Stack signature for newdateinzonewithdst:    ( y mo d h mi s zone rule )

func (e *PostfixEmitter) VisitDateNewYMDInZone(ctx *DateNewYMDInZoneContext) interface{} {
	e.Visit(ctx.Iexpr(0))
	e.Visit(ctx.Iexpr(1))
	e.Visit(ctx.Iexpr(2))
	e.emit("0")
	e.emit("0")
	e.emit("0")
	e.Visit(ctx.Strexpr())
	e.emit("newdateinzone")
	return nil
}

func (e *PostfixEmitter) VisitDateNewYMDInZoneWithDST(ctx *DateNewYMDInZoneWithDSTContext) interface{} {
	e.Visit(ctx.Iexpr(0))
	e.Visit(ctx.Iexpr(1))
	e.Visit(ctx.Iexpr(2))
	e.emit("0")
	e.emit("0")
	e.emit("0")
	e.Visit(ctx.Strexpr(0))
	e.Visit(ctx.Strexpr(1))
	e.emit("newdateinzonewithdst")
	return nil
}

func (e *PostfixEmitter) VisitDateNewYMDhmsInZone(ctx *DateNewYMDhmsInZoneContext) interface{} {
	for i := 0; i < 6; i++ {
		e.Visit(ctx.Iexpr(i))
	}
	e.Visit(ctx.Strexpr())
	e.emit("newdateinzone")
	return nil
}

func (e *PostfixEmitter) VisitDateNewYMDhmsInZoneWithDST(ctx *DateNewYMDhmsInZoneWithDSTContext) interface{} {
	for i := 0; i < 6; i++ {
		e.Visit(ctx.Iexpr(i))
	}
	e.Visit(ctx.Strexpr(0))
	e.Visit(ctx.Strexpr(1))
	e.emit("newdateinzonewithdst")
	return nil
}

// ============================================================================
// #803 batch 12: leftovers (if/then, operatorlist tails, left-array-colon,
// thereis/entity-first/date-earliest).
// ============================================================================

// VisitIfThen: `if <bexpr> then <block> endif` → `<bexpr> { <block> } if`.
// Pre-fix the whole if-statement silently emitted nothing; conditional
// action blocks were entirely lost.
func (e *PostfixEmitter) VisitIfThen(ctx *IfThenContext) interface{} {
	// Runtime `if` pops the test from the top: `{ body } <bexpr> if`.
	e.emit("{")
	e.Visit(ctx.Block())
	e.emit("}")
	e.Visit(ctx.Bexpr())
	e.emit("if")
	return nil
}

// VisitIfThenElse: `if <bexpr> then <t> else <e> endif`
// → `{ <t> } { <e> } <bexpr> ifelse` — runtime ifelse pops the test
// from the top of the stack. Pre-fix this silently emitted nothing.
func (e *PostfixEmitter) VisitIfThenElse(ctx *IfThenElseContext) interface{} {
	e.emit("{")
	e.Visit(ctx.Block(0))
	e.emit("}")
	e.emit("{")
	e.Visit(ctx.Block(1))
	e.emit("}")
	e.Visit(ctx.Bexpr())
	e.emit("ifelse")
	return nil
}

// VisitOpListEntity: `<eexpr> , <operatorlist>` — non-terminal entity
// argument in an operator call. Mirror the existing VisitOpListInt /
// VisitOpListStr / VisitOpListFloat visitors: emit the eexpr first,
// then recurse into the tail. Pre-fix this rule silently dropped any
// entity-typed argument that wasn't last in the list.
func (e *PostfixEmitter) VisitOpListEntity(ctx *OpListEntityContext) interface{} {
	e.Visit(ctx.Eexpr())
	e.Visit(ctx.Operatorlist())
	return nil
}

// VisitOpListEntitySingle: terminal `<eexpr>` in an operator call.
// Mirrors VisitOpListIntSingle / VisitOpListStrSingle. Pre-fix this
// silently dropped a single entity argument.
func (e *PostfixEmitter) VisitOpListEntitySingle(ctx *OpListEntitySingleContext) interface{} {
	e.Visit(ctx.Eexpr())
	return nil
}

// VisitLeftArrayColon: `:Entity:<array-field>` on the LHS of `set X =
// Y`. Mirrors the existing VisitLeftIexprColon / VisitLeftFexprColon
// / VisitLeftEexprColon family — visit the colonRef (which pushes
// the entity and walks possessive-or-colon chain), then the inner
// leftArrayRef which emits the field-store trailer. Pre-fix the rule
// silently emitted nothing.
func (e *PostfixEmitter) VisitLeftArrayColon(ctx *LeftArrayColonContext) interface{} {
	e.Visit(ctx.ColonRef())
	e.Visit(ctx.LeftArrayRef())
	return nil
}

// VisitEntityFirst: `first <eexpr> where <bexpr>` — return the first
// entity in the iterated set whose bexpr holds. No `firstwhere`-style
// entity-returning runtime op exists yet. Emit an elstmterror
// placeholder so this fails loudly at runtime instead of silently
// emitting nothing. Tracked as a follow-up: needs a new runtime op
// or a fold-with-capture postfix pattern via local vars.
func (e *PostfixEmitter) VisitEntityFirst(ctx *EntityFirstContext) interface{} {
	e.emit(`"first <e> where ... not yet implemented (needs firstwhere runtime op)"`)
	e.emit("elstmterror")
	return nil
}

// VisitEntityFirstIn: `first <eexpr> in <array> where <bexpr>` — same
// as EntityFirst but with an explicit source array. Same
// elstmterror-placeholder treatment until the runtime op lands.
func (e *PostfixEmitter) VisitEntityFirstIn(ctx *EntityFirstInContext) interface{} {
	e.emit(`"first <e> in <arr> where ... not yet implemented (needs firstwhere runtime op)"`)
	e.emit("elstmterror")
	return nil
}

// VisitDateEarliestAfter: `earliest of <array> after <date>` — return
// the earliest date in the array that's strictly after the given
// date. No `earliestafter` runtime op exists. Emit elstmterror;
// tracked as a follow-up.
func (e *PostfixEmitter) VisitDateEarliestAfter(ctx *DateEarliestAfterContext) interface{} {
	e.emit(`"earliest of <arr> after <d> not yet implemented (needs earliestafter runtime op)"`)
	e.emit("elstmterror")
	return nil
}

// emitFixedProductLeftAssoc emits an fexpr as fixed, forcing a bare
// multiplication chain to group left-to-right.
//
// See VisitDivideRoundingBy for why: fp* rounds, so `(x*y)*z` and `x*(y*z)`
// are different numbers, and every other context in the language groups left.
// A chain of fewer than three operands has nothing to re-associate and takes
// the ordinary path.
func (e *PostfixEmitter) emitFixedProductLeftAssoc(ctx antlr.ParseTree) {
	operands := flattenMulChain(ctx)
	if len(operands) < 3 {
		e.emitWithTypeConversion(ctx, TypeFixed)
		return
	}
	e.emitWithTypeConversion(operands[0], TypeFixed)
	for _, operand := range operands[1:] {
		e.emitWithTypeConversion(operand, TypeFixed)
		e.emit("fp*")
	}
}

// flattenMulChain returns the operands of a multiplication chain in source
// order, whatever shape the parse gave it.
//
// It descends only through the three multiply alternatives. A parenthesised
// subexpression is a FloatParen node, not a multiply node, so it is returned
// whole — explicit grouping by the author survives, which is what makes
// `divide n by x * (y * z) rounding by 0.5fp` still mean what it says.
func flattenMulChain(t antlr.ParseTree) []antlr.ParseTree {
	switch c := t.(type) {
	case *FloatMulFloatContext:
		return append(flattenMulChain(c.Fexpr(0)), flattenMulChain(c.Fexpr(1))...)
	case *FloatMulIntContext:
		return append(flattenMulChain(c.Fexpr()), c.Iexpr())
	case *IntMulFloatContext:
		return append([]antlr.ParseTree{c.Iexpr()}, flattenMulChain(c.Fexpr())...)
	}
	return []antlr.ParseTree{t}
}

// max of / min of over a collection — the fold family stopped at `sum of`, so
// "the best of a set of options" had to be a host-side loop even when the
// criterion was pure policy: the table could score the options but not say
// which one won (#1024).
//
// Lowered exactly like `sum of`, swapping the accumulating operator. The
// accumulator starts at zero, which is the documented behaviour for an empty
// array or a `where` that matches nothing — the same answer `sum of` gives,
// and chosen for consistency within the family rather than mathematical
// purity. The consequence is explicit: over values that are all negative,
// `max of` returns 0 rather than the largest negative. Rules over money and
// counts do not meet that case; rules that might should guard the array.
//
// This is the value, not the element attaining it. `the <entity> in <array>
// with the max <field>` is the part that keeps a choice rule inside the
// tables, and is still open on #1024.

func (e *PostfixEmitter) VisitIntMaxOfArray(ctx *IntMaxOfArrayContext) interface{} {
	return e.emitIntFold(ctx.Iexpr(), ctx.ArrayExpr(), "max")
}

func (e *PostfixEmitter) VisitIntMinOfArray(ctx *IntMinOfArrayContext) interface{} {
	return e.emitIntFold(ctx.Iexpr(), ctx.ArrayExpr(), "min")
}

func (e *PostfixEmitter) VisitIntMaxOfArrayWhere(ctx *IntMaxOfArrayWhereContext) interface{} {
	return e.emitIntFoldWhere(ctx.Iexpr(), ctx.ArrayExpr(), ctx.Bexpr(), "max")
}

func (e *PostfixEmitter) VisitIntMinOfArrayWhere(ctx *IntMinOfArrayWhereContext) interface{} {
	return e.emitIntFoldWhere(ctx.Iexpr(), ctx.ArrayExpr(), ctx.Bexpr(), "min")
}

func (e *PostfixEmitter) VisitFloatMaxOfArray(ctx *FloatMaxOfArrayContext) interface{} {
	return e.emitFloatFold(ctx.TypedDouble(), ctx.ArrayExpr(), "fmax")
}

func (e *PostfixEmitter) VisitFloatMinOfArray(ctx *FloatMinOfArrayContext) interface{} {
	return e.emitFloatFold(ctx.TypedDouble(), ctx.ArrayExpr(), "fmin")
}

func (e *PostfixEmitter) VisitFloatMaxOfArrayWhere(ctx *FloatMaxOfArrayWhereContext) interface{} {
	return e.emitFloatFoldWhere(ctx.TypedDouble(), ctx.ArrayExpr(), ctx.Bexpr(), "fmax")
}

func (e *PostfixEmitter) VisitFloatMinOfArrayWhere(ctx *FloatMinOfArrayWhereContext) interface{} {
	return e.emitFloatFoldWhere(ctx.TypedDouble(), ctx.ArrayExpr(), ctx.Bexpr(), "fmin")
}

// emitIntFold emits `0 { <expr> <op> } <array> forall`. Body before array,
// because opForall pops the array off the top (#867).
func (e *PostfixEmitter) emitIntFold(expr, array antlr.ParserRuleContext, op string) interface{} {
	e.emit("0")
	e.emit("{")
	e.Visit(expr)
	e.emit(op)
	e.emit("}")
	e.Visit(array)
	e.emit("forall")
	return nil
}

// emitIntFoldWhere emits `0 { { <expr> <op> } <pred> if } <array> forall`.
// Inner block before the predicate, because `if` pops the boolean off the top.
func (e *PostfixEmitter) emitIntFoldWhere(expr, array, pred antlr.ParserRuleContext, op string) interface{} {
	e.emit("0")
	e.emit("{")
	e.emit("{")
	e.Visit(expr)
	e.emit(op)
	e.emit("}")
	e.Visit(pred)
	e.emit("if")
	e.emit("}")
	e.Visit(array)
	e.emit("forall")
	return nil
}

func (e *PostfixEmitter) emitFloatFold(expr, array antlr.ParserRuleContext, op string) interface{} {
	e.emit("0.0")
	e.emit("{")
	e.Visit(expr)
	e.emit(op)
	e.emit("}")
	e.Visit(array)
	e.emit("forall")
	return nil
}

func (e *PostfixEmitter) emitFloatFoldWhere(expr, array, pred antlr.ParserRuleContext, op string) interface{} {
	e.emit("0.0")
	e.emit("{")
	e.emit("{")
	e.Visit(expr)
	e.emit(op)
	e.emit("}")
	e.Visit(pred)
	e.emit("if")
	e.emit("}")
	e.Visit(array)
	e.emit("forall")
	return nil
}
