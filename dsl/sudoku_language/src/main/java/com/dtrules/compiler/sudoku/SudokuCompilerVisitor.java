/*
 * Copyright 2004-2009 DTRules.com, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package com.dtrules.compiler.sudoku;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;

/**
 * ANTLR 4 Visitor that generates postfix code from the Sudoku parse tree.
 * This replaces the semantic actions from the CUP parser.
 */
public class SudokuCompilerVisitor extends SudokuBaseVisitor<String> {

    private final SudokuTypeResolver typeResolver;
    private int localCnt = 0;

    public SudokuCompilerVisitor(SudokuTypeResolver typeResolver) {
        this.typeResolver = typeResolver;
    }

    public int getLocalCnt() {
        return localCnt;
    }

    public void setLocalCnt(int localCnt) {
        this.localCnt = localCnt;
    }

    @Override
    protected String defaultResult() {
        return "";
    }

    @Override
    protected String aggregateResult(String aggregate, String nextResult) {
        if (aggregate == null) aggregate = "";
        if (nextResult == null) nextResult = "";
        return aggregate + nextResult;
    }

    // ========================================================================
    // Top-level rules
    // ========================================================================

    @Override
    public String visitEmptyAction(SudokuParser.EmptyActionContext ctx) {
        return "";
    }

    @Override
    public String visitEmptyCondition(SudokuParser.EmptyConditionContext ctx) {
        return "";
    }

    @Override
    public String visitEmptyContext(SudokuParser.EmptyContextContext ctx) {
        return "execute ";
    }

    @Override
    public String visitEmptyPolicyStatement(SudokuParser.EmptyPolicyStatementContext ctx) {
        return "";
    }

    @Override
    public String visitActionStatement(SudokuParser.ActionStatementContext ctx) {
        return "\n" + visit(ctx.statementList()) + "\n";
    }

    @Override
    public String visitConditionExpr(SudokuParser.ConditionExprContext ctx) {
        return "\n" + visit(ctx.bexpr()) + "\n";
    }

    @Override
    public String visitConditionDebugBefore(SudokuParser.ConditionDebugBeforeContext ctx) {
        return "\n" + visit(ctx.debugstatement()) + visit(ctx.bexpr()) + "\n";
    }

    @Override
    public String visitConditionDebugAfter(SudokuParser.ConditionDebugAfterContext ctx) {
        return "\n" + visit(ctx.bexpr()) + visit(ctx.debugstatement()) + "\n";
    }

    @Override
    public String visitContextStatement(SudokuParser.ContextStatementContext ctx) {
        return "\n" + visit(ctx.contextForTable()) + "\n";
    }

    @Override
    public String visitPolicyStrExpr(SudokuParser.PolicyStrExprContext ctx) {
        return visit(ctx.strexpr());
    }

    @Override
    public String visitPolicyNExpr(SudokuParser.PolicyNExprContext ctx) {
        return visit(ctx.nexpr());
    }

    @Override
    public String visitPolicyIExpr(SudokuParser.PolicyIExprContext ctx) {
        return visit(ctx.iexpr());
    }

    @Override
    public String visitPolicyFExpr(SudokuParser.PolicyFExprContext ctx) {
        return visit(ctx.fexpr());
    }

    @Override
    public String visitPolicyBExpr(SudokuParser.PolicyBExprContext ctx) {
        return visit(ctx.bexpr());
    }

    @Override
    public String visitPolicyDExpr(SudokuParser.PolicyDExprContext ctx) {
        return visit(ctx.dexpr());
    }

    // ========================================================================
    // Statement list and blocks
    // ========================================================================

    @Override
    public String visitStatementList(SudokuParser.StatementListContext ctx) {
        StringBuilder sb = new StringBuilder();
        for (SudokuParser.BlockContext block : ctx.block()) {
            sb.append(visit(block));
        }
        return sb.toString();
    }

    @Override
    public String visitBlockCurly(SudokuParser.BlockCurlyContext ctx) {
        return visit(ctx.statementList()) + "\n";
    }

    @Override
    public String visitBlockUsing(SudokuParser.BlockUsingContext ctx) {
        return visit(ctx.usingblock());
    }

    @Override
    public String visitBlockGforall(SudokuParser.BlockGforallContext ctx) {
        String blk = visit(ctx.statementList());
        String ctl = visit(ctx.forallctl());
        return "{ " + blk + "} " + ctl + "pop ";
    }

    @Override
    public String visitBlockForall(SudokuParser.BlockForallContext ctx) {
        return visit(ctx.forblock());
    }

    @Override
    public String visitBlockFirst(SudokuParser.BlockFirstContext ctx) {
        return visit(ctx.firstblock());
    }

    @Override
    public String visitBlockIf(SudokuParser.BlockIfContext ctx) {
        return visit(ctx.ifblock());
    }

    @Override
    public String visitBlockStatement(SudokuParser.BlockStatementContext ctx) {
        return visit(ctx.statement());
    }

    @Override
    public String visitBlockSep(SudokuParser.BlockSepContext ctx) {
        return visit(ctx.block());
    }

    // ========================================================================
    // Using blocks
    // ========================================================================

    @Override
    public String visitUsingBlockEntity(SudokuParser.UsingBlockEntityContext ctx) {
        String ee = resolveEntity(ctx.typedEntity());
        String e = visit(ctx.usingblock());
        return ee + " entitypush " + e + "entitypop ";
    }

    @Override
    public String visitUsingBlockEntityComma(SudokuParser.UsingBlockEntityCommaContext ctx) {
        String ee = resolveEntity(ctx.typedEntity());
        String e = visit(ctx.usingblock());
        return ee + " entitypush " + e + "entitypop ";
    }

    @Override
    public String visitUsingBlockBase(SudokuParser.UsingBlockBaseContext ctx) {
        return visit(ctx.block());
    }

    // ========================================================================
    // Context for table
    // ========================================================================

    @Override
    public String visitContextDebug(SudokuParser.ContextDebugContext ctx) {
        return visit(ctx.debugstatement()) + "execute ";
    }

    @Override
    public String visitContextFor(SudokuParser.ContextForContext ctx) {
        return visit(ctx.forctl()) + "pop ";
    }

    @Override
    public String visitContextForallCtl(SudokuParser.ContextForallCtlContext ctx) {
        return visit(ctx.forallctl()) + "pop ";
    }

    @Override
    public String visitContextForfirst(SudokuParser.ContextForfirstContext ctx) {
        return visit(ctx.forfirstctl()) + "pop ";
    }

    @Override
    public String visitContextCtx(SudokuParser.ContextCtxContext ctx) {
        return visit(ctx.contextstatement()) + "execute entitypop ";
    }

    @Override
    public String visitContextLocal(SudokuParser.ContextLocalContext ctx) {
        return visit(ctx.localvariables());
    }

    // ========================================================================
    // Forall control
    // ========================================================================

    @Override
    public String visitForallSimple(SudokuParser.ForallSimpleContext ctx) {
        return "dup " + visit(ctx.arrayExpr()) + "forall ";
    }

    @Override
    public String visitForallAllowRemove(SudokuParser.ForallAllowRemoveContext ctx) {
        return "dup " + visit(ctx.arrayExpr()) + "forallr ";
    }

    @Override
    public String visitForallInEntity(SudokuParser.ForallInEntityContext ctx) {
        String a = visit(ctx.arrayExpr());
        String ev = visit(ctx.eexpr());
        return "dup " + ev + " entitypush " + a + "forall entitypop ";
    }

    @Override
    public String visitForallInEntityAllowRemove(SudokuParser.ForallInEntityAllowRemoveContext ctx) {
        String a = visit(ctx.arrayExpr());
        String ev = visit(ctx.eexpr());
        return "dup " + ev + " entitypush " + a + "forallr entitypop ";
    }

    @Override
    public String visitForallInEntityWhere(SudokuParser.ForallInEntityWhereContext ctx) {
        String a = visit(ctx.arrayExpr());
        String e2 = visit(ctx.eexpr());
        String b = visit(ctx.bexpr());
        return "{ dup " + b + "if } " + e2 + " entitypush " + a + "forall entitypop ";
    }

    @Override
    public String visitForallWhere(SudokuParser.ForallWhereContext ctx) {
        String a = visit(ctx.arrayExpr());
        String b = visit(ctx.bexpr());
        return "{ dup " + b + "if } " + a + "forall ";
    }

    @Override
    public String visitForallWhereAllowRemove(SudokuParser.ForallWhereAllowRemoveContext ctx) {
        String a = visit(ctx.arrayExpr());
        String b = visit(ctx.bexpr());
        return "{ dup " + b + "if } " + a + "forallr ";
    }

    @Override
    public String visitForallInThis(SudokuParser.ForallInThisContext ctx) {
        String a = visit(ctx.arrayExpr());
        String i = visit(ctx.iexpr());
        return i + "{ dup " + i + " == over swap if }" + a + "forall ";
    }

    @Override
    public String visitForallInAllInThis(SudokuParser.ForallInAllInThisContext ctx) {
        String a1 = visit(ctx.arrayExpr(0));
        String a2 = visit(ctx.arrayExpr(1));
        String i = visit(ctx.iexpr());
        return i + "{ dup " + i + "== { dup " + i + "== over swap if } " + a1 + "forall } " + a2 + " forall ";
    }

    // ========================================================================
    // Forblock (for forall loops)
    // ========================================================================

    @Override
    public String visitForblockSimple(SudokuParser.ForblockSimpleContext ctx) {
        String a = visit(ctx.arrayExpr());
        String blk = visit(ctx.block());
        return "{ " + blk + "} " + a + "forallr ";
    }

    @Override
    public String visitForblockInArray(SudokuParser.ForblockInArrayContext ctx) {
        String a = visit(ctx.arrayExpr());
        String blk = visit(ctx.block());
        return "{ " + blk + "} " + a + " forallr ";
    }

    @Override
    public String visitForblockInArrayWhere(SudokuParser.ForblockInArrayWhereContext ctx) {
        String a = visit(ctx.arrayExpr());
        String b = visit(ctx.bexpr());
        String blk = visit(ctx.block());
        return "{ { " + blk + " } " + b + "if } " + a + "forallr ";
    }

    @Override
    public String visitForblockArrayWhere(SudokuParser.ForblockArrayWhereContext ctx) {
        String a = visit(ctx.arrayExpr());
        String b = visit(ctx.bexpr());
        String blk = visit(ctx.block());
        return "{ { " + blk + " } " + b + "if } " + a + "forallr ";
    }

    @Override
    public String visitForblockIts(SudokuParser.ForblockItsContext ctx) {
        String e2 = visit(ctx.eexpr(1));
        String a = visit(ctx.arrayExpr());
        String blk = visit(ctx.block());
        return "{ " + e2 + "entitypush " + blk + " entitypop } " + a + " forallr ";
    }

    @Override
    public String visitForblockItsWhere(SudokuParser.ForblockItsWhereContext ctx) {
        String e2 = visit(ctx.eexpr(1));
        String a = visit(ctx.arrayExpr());
        String b = visit(ctx.bexpr());
        String blk = visit(ctx.block());
        return "{ " + e2 + "entitypush { " + blk + " } " + b + "if entitypop } " + a + "forallr ";
    }

    // ========================================================================
    // Forfirst control
    // ========================================================================

    @Override
    public String visitForfirstOf(SudokuParser.ForfirstOfContext ctx) {
        String a = visit(ctx.arrayExpr());
        String test = visit(ctx.bexpr());
        return "dup { " + test + " } " + a + " forfirst ";
    }

    @Override
    public String visitForfirstOfIts(SudokuParser.ForfirstOfItsContext ctx) {
        String a = visit(ctx.arrayExpr());
        String e1 = visit(ctx.eexpr());
        String test = visit(ctx.bexpr());
        return "{ " + e1 + " entitypush dup execute entitypop } { " + e1 + " entitypush " + test + " entitypop } " + a + " forfirst ";
    }

    @Override
    public String visitForfirstIn(SudokuParser.ForfirstInContext ctx) {
        String a = visit(ctx.arrayExpr());
        String test = visit(ctx.bexpr());
        return "dup { " + test + " } " + a + " forfirst ";
    }

    // ========================================================================
    // First block
    // ========================================================================

    @Override
    public String visitFirstBlockElse(SudokuParser.FirstBlockElseContext ctx) {
        String a = visit(ctx.arrayExpr());
        String test = visit(ctx.bexpr());
        String body1 = visit(ctx.block(0));
        String body2 = visit(ctx.block(1));
        return "{ " + body1 + " } { " + body2 + " } { " + test + " } " + a + " forfirstelse ";
    }

    @Override
    public String visitFirstBlockItsElse(SudokuParser.FirstBlockItsElseContext ctx) {
        String a = visit(ctx.arrayExpr());
        String e1 = visit(ctx.eexpr());
        String test = visit(ctx.bexpr());
        String body1 = visit(ctx.block(0));
        String body2 = visit(ctx.block(1));
        return "{ " + e1 + " entitypush " + body1 + "entitypop } { " + body2 + " } { " + e1 + " entitypush " + test + "entitypop } " + a + " forfirstelse ";
    }

    @Override
    public String visitFirstBlockCtl(SudokuParser.FirstBlockCtlContext ctx) {
        String ctl = visit(ctx.forfirstctl());
        String body = visit(ctx.block());
        return "{ " + body + " } " + ctl + " pop ";
    }

    // ========================================================================
    // Helper methods for type resolution
    // ========================================================================

    private String resolveEntity(SudokuParser.TypedEntityContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveLong(SudokuParser.TypedLongContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveDouble(SudokuParser.TypedDoubleContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveString(SudokuParser.TypedStringContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveBoolean(SudokuParser.TypedBooleanContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveDate(SudokuParser.TypedDateContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveArray(SudokuParser.TypedArrayContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveTable(SudokuParser.TypedTableContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveName(SudokuParser.TypedNameContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveDecisionTable(SudokuParser.TypedDecisionTableContext ctx) {
        return "/" + ctx.IDENT().getText() + " execute ";
    }

    private String resolveIdent(String ident) {
        try {
            SudokuTypeResolver.ResolvedIdentifier resolved = typeResolver.resolve(ident);
            return resolved.value;
        } catch (RulesException e) {
            throw new RuntimeException("Failed to resolve identifier: " + ident, e);
        }
    }

    private String resolveIdentLeft(String ident) {
        try {
            SudokuTypeResolver.ResolvedIdentifier resolved = typeResolver.resolve(ident);
            return resolved.leftValue;
        } catch (RulesException e) {
            throw new RuntimeException("Failed to resolve identifier: " + ident, e);
        }
    }

    // ========================================================================
    // Integer expressions
    // ========================================================================

    @Override
    public String visitIntAdd(SudokuParser.IntAddContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "+ ";
    }

    @Override
    public String visitIntSub(SudokuParser.IntSubContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "- ";
    }

    @Override
    public String visitIntMul(SudokuParser.IntMulContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "* ";
    }

    @Override
    public String visitIntDiv(SudokuParser.IntDivContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "div ";
    }

    @Override
    public String visitIntLiteral(SudokuParser.IntLiteralContext ctx) {
        return ctx.INT_LITERAL().getText() + " ";
    }

    @Override
    public String visitIntNegate(SudokuParser.IntNegateContext ctx) {
        return visit(ctx.iexpr()) + "negate ";
    }

    @Override
    public String visitIntParen(SudokuParser.IntParenContext ctx) {
        return visit(ctx.iexpr()) + " ";
    }

    @Override
    public String visitIntTyped(SudokuParser.IntTypedContext ctx) {
        return resolveLong(ctx.typedLong()) + " ";
    }

    @Override
    public String visitIntColonRef(SudokuParser.IntColonRefContext ctx) {
        String ref = visit(ctx.colonRef());
        String n = resolveLong(ctx.typedLong());
        return ref + "entitypush " + n + " entitypop ";
    }

    @Override
    public String visitIntNumberOf(SudokuParser.IntNumberOfContext ctx) {
        return visit(ctx.arrayExpr()) + "length ";
    }

    @Override
    public String visitIntNumberOfWhere(SudokuParser.IntNumberOfWhereContext ctx) {
        String a = visit(ctx.arrayExpr());
        String b = visit(ctx.bexpr());
        return "0 { { 1 ladd } " + b + "if }" + a + "forall ";
    }

    @Override
    public String visitIntLengthArray(SudokuParser.IntLengthArrayContext ctx) {
        return visit(ctx.arrayExpr()) + "length ";
    }

    @Override
    public String visitIntLengthStr(SudokuParser.IntLengthStrContext ctx) {
        return visit(ctx.strexpr()) + "strlength ";
    }

    @Override
    public String visitIntIndexOf(SudokuParser.IntIndexOfContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "indexof ";
    }

    @Override
    public String visitIntDaysBetween(SudokuParser.IntDaysBetweenContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "daysbetween ";
    }

    @Override
    public String visitIntMonthsBetween(SudokuParser.IntMonthsBetweenContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "monthsbetween ";
    }

    @Override
    public String visitIntYearsBetween(SudokuParser.IntYearsBetweenContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "yearsbetween ";
    }

    @Override
    public String visitIntYearOf(SudokuParser.IntYearOfContext ctx) {
        return visit(ctx.dexpr()) + "yearof ";
    }

    @Override
    public String visitIntAbs(SudokuParser.IntAbsContext ctx) {
        return visit(ctx.iexpr()) + "labs ";
    }

    @Override
    public String visitIntUsing(SudokuParser.IntUsingContext ctx) {
        String ee = visit(ctx.eexpr());
        String e = visit(ctx.iexpr());
        return ee + "entitypush " + e + "entitypop ";
    }

    @Override
    public String visitIntDaysInYear(SudokuParser.IntDaysInYearContext ctx) {
        return visit(ctx.dexpr()) + "getdaysinyear ";
    }

    @Override
    public String visitIntDaysInMonth(SudokuParser.IntDaysInMonthContext ctx) {
        return visit(ctx.dexpr()) + "getdaysinmonth ";
    }

    @Override
    public String visitIntDayOfMonth(SudokuParser.IntDayOfMonthContext ctx) {
        return visit(ctx.dexpr()) + "getdayofmonth ";
    }

    // ========================================================================
    // Float expressions
    // ========================================================================

    @Override
    public String visitFloatLiteral(SudokuParser.FloatLiteralContext ctx) {
        return ctx.FLOAT_LITERAL().getText() + " ";
    }

    @Override
    public String visitFloatTyped(SudokuParser.FloatTypedContext ctx) {
        return resolveDouble(ctx.typedDouble()) + " ";
    }

    @Override
    public String visitFloatColonRef(SudokuParser.FloatColonRefContext ctx) {
        String ref = visit(ctx.colonRef());
        String f = resolveDouble(ctx.typedDouble());
        return ref + "entitypush " + f + " entitypop ";
    }

    @Override
    public String visitFloatAddInt(SudokuParser.FloatAddIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "fadd ";
    }

    @Override
    public String visitFloatAddFloat(SudokuParser.FloatAddFloatContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "fadd ";
    }

    @Override
    public String visitIntAddFloat(SudokuParser.IntAddFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "fadd ";
    }

    @Override
    public String visitFloatSubInt(SudokuParser.FloatSubIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "fsub ";
    }

    @Override
    public String visitIntSubFloat(SudokuParser.IntSubFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "fsub ";
    }

    @Override
    public String visitFloatSubFloat(SudokuParser.FloatSubFloatContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "fsub ";
    }

    @Override
    public String visitFloatMulInt(SudokuParser.FloatMulIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "fmul ";
    }

    @Override
    public String visitIntMulFloat(SudokuParser.IntMulFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "fmul ";
    }

    @Override
    public String visitFloatMulFloat(SudokuParser.FloatMulFloatContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "fmul ";
    }

    @Override
    public String visitFloatDivInt(SudokuParser.FloatDivIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "fdiv ";
    }

    @Override
    public String visitIntDivFloat(SudokuParser.IntDivFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "fdiv ";
    }

    @Override
    public String visitFloatDivFloat(SudokuParser.FloatDivFloatContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "fdiv ";
    }

    @Override
    public String visitFloatNegate(SudokuParser.FloatNegateContext ctx) {
        return visit(ctx.fexpr()) + "fnegate ";
    }

    @Override
    public String visitFloatParen(SudokuParser.FloatParenContext ctx) {
        return visit(ctx.fexpr());
    }

    @Override
    public String visitFloatAbs(SudokuParser.FloatAbsContext ctx) {
        return visit(ctx.fexpr()) + "fabs ";
    }

    @Override
    public String visitFloatUsing(SudokuParser.FloatUsingContext ctx) {
        String ee = visit(ctx.eexpr());
        String e = visit(ctx.fexpr());
        return ee + "entitypush " + e + "entitypop ";
    }

    @Override
    public String visitFloatRounded(SudokuParser.FloatRoundedContext ctx) {
        return visit(ctx.fexpr()) + "0 0.5 roundto ";
    }

    @Override
    public String visitFloatRoundedTo(SudokuParser.FloatRoundedToContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "0.5 roundto ";
    }

    @Override
    public String visitFloatRoundedBoundry(SudokuParser.FloatRoundedBoundryContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.iexpr()) + visit(ctx.fexpr(1)) + " roundto ";
    }

    @Override
    public String visitFloatSumOf(SudokuParser.FloatSumOfContext ctx) {
        String f = visit(ctx.fexpr());
        String a = visit(ctx.arrayExpr());
        return "0 { " + f + "ladd } " + a + "forall ";
    }

    @Override
    public String visitIntSumOf(SudokuParser.IntSumOfContext ctx) {
        String i = visit(ctx.iexpr());
        String a = visit(ctx.arrayExpr());
        return "0 { " + i + "ladd } " + a + "forall ";
    }

    // ========================================================================
    // String expressions
    // ========================================================================

    @Override
    public String visitStrLiteral(SudokuParser.StrLiteralContext ctx) {
        return ctx.STRING_LITERAL().getText() + " ";
    }

    @Override
    public String visitStrTyped(SudokuParser.StrTypedContext ctx) {
        return resolveString(ctx.typedString()) + " ";
    }

    @Override
    public String visitStrColonRef(SudokuParser.StrColonRefContext ctx) {
        String ref = visit(ctx.colonRef());
        String s = visit(ctx.strexpr());
        return ref + "entitypush " + s + "entitypop ";
    }

    @Override
    public String visitStrConcat(SudokuParser.StrConcatContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "strconcat ";
    }

    @Override
    public String visitStrConcatInt(SudokuParser.StrConcatIntContext ctx) {
        return visit(ctx.strexpr()) + visit(ctx.iexpr()) + "strconcat ";
    }

    @Override
    public String visitStrConcatFloat(SudokuParser.StrConcatFloatContext ctx) {
        return visit(ctx.strexpr()) + visit(ctx.fexpr()) + "strconcat ";
    }

    @Override
    public String visitStrConcatName(SudokuParser.StrConcatNameContext ctx) {
        return visit(ctx.strexpr()) + visit(ctx.nexpr()) + "strconcat ";
    }

    @Override
    public String visitStrConcatEntity(SudokuParser.StrConcatEntityContext ctx) {
        return visit(ctx.strexpr()) + visit(ctx.eexpr()) + "strconcat ";
    }

    @Override
    public String visitStrConcatDate(SudokuParser.StrConcatDateContext ctx) {
        return visit(ctx.strexpr()) + visit(ctx.dexpr()) + "strconcat ";
    }

    @Override
    public String visitStrConcatArray(SudokuParser.StrConcatArrayContext ctx) {
        return visit(ctx.strexpr()) + visit(ctx.arrayExpr()) + "strconcat ";
    }

    @Override
    public String visitStrTrim(SudokuParser.StrTrimContext ctx) {
        return visit(ctx.strexpr()) + "strtrim ";
    }

    @Override
    public String visitStrToLower(SudokuParser.StrToLowerContext ctx) {
        return visit(ctx.strexpr()) + "tolowercase ";
    }

    @Override
    public String visitStrToUpper(SudokuParser.StrToUpperContext ctx) {
        return visit(ctx.strexpr()) + "touppercase ";
    }

    @Override
    public String visitStrCurrentDate(SudokuParser.StrCurrentDateContext ctx) {
        return "getdate ";
    }

    @Override
    public String visitStrTimestamp(SudokuParser.StrTimestampContext ctx) {
        return "gettimestamp ";
    }

    @Override
    public String visitStrUsing(SudokuParser.StrUsingContext ctx) {
        String ee = visit(ctx.eexpr());
        String e = visit(ctx.strexpr());
        return ee + "entitypush " + e + "entitypop ";
    }

    @Override
    public String visitStrParen(SudokuParser.StrParenContext ctx) {
        return visit(ctx.strexpr());
    }

    @Override
    public String visitStrSubstring(SudokuParser.StrSubstringContext ctx) {
        String s = visit(ctx.strexpr());
        String start = visit(ctx.iexpr(0));
        String end = visit(ctx.iexpr(1));
        return end + start + s + "substring ";
    }

    @Override
    public String visitStrValueOfFloat(SudokuParser.StrValueOfFloatContext ctx) {
        return visit(ctx.fexpr()) + "tostring ";
    }

    @Override
    public String visitStrValueOfInt(SudokuParser.StrValueOfIntContext ctx) {
        return visit(ctx.iexpr()) + "tostring ";
    }

    @Override
    public String visitStrValueOfDate(SudokuParser.StrValueOfDateContext ctx) {
        return visit(ctx.dexpr()) + "tostring ";
    }

    @Override
    public String visitStrCvsOfBool(SudokuParser.StrCvsOfBoolContext ctx) {
        return visit(ctx.bexpr()) + "tostring ";
    }

    @Override
    public String visitStrTableInfo(SudokuParser.StrTableInfoContext ctx) {
        return "actionstring ";
    }

    @Override
    public String visitStrMappingKey(SudokuParser.StrMappingKeyContext ctx) {
        return "\"mapping*key\" cvn execute ";
    }

    // ========================================================================
    // Boolean expressions
    // ========================================================================

    @Override
    public String visitBoolTyped(SudokuParser.BoolTypedContext ctx) {
        return resolveBoolean(ctx.typedBoolean()) + " ";
    }

    @Override
    public String visitBoolLiteral(SudokuParser.BoolLiteralContext ctx) {
        return ctx.RBOOLEAN().getText().toLowerCase() + " ";
    }

    @Override
    public String visitBoolColonRef(SudokuParser.BoolColonRefContext ctx) {
        String ref = visit(ctx.colonRef());
        String b = resolveBoolean(ctx.typedBoolean());
        return ref + "entitypush " + b + " entitypop ";
    }

    @Override
    public String visitBoolAnd(SudokuParser.BoolAndContext ctx) {
        String e1 = visit(ctx.bexpr(0));
        String e2 = visit(ctx.bexpr(1));
        return e1 + "{ pop " + e2 + "} over if\n";
    }

    @Override
    public String visitBoolOr(SudokuParser.BoolOrContext ctx) {
        String e1 = visit(ctx.bexpr(0));
        String e2 = visit(ctx.bexpr(1));
        return e1 + "{ pop " + e2 + "} over not if\n";
    }

    @Override
    public String visitBoolNot(SudokuParser.BoolNotContext ctx) {
        return visit(ctx.bexpr()) + "not ";
    }

    @Override
    public String visitBoolParen(SudokuParser.BoolParenContext ctx) {
        return visit(ctx.bexpr());
    }

    @Override
    public String visitBoolUsing(SudokuParser.BoolUsingContext ctx) {
        String ee = visit(ctx.eexpr());
        String e = visit(ctx.bexpr());
        return ee + "entitypush " + e + "entitypop ";
    }

    // Integer comparisons
    @Override
    public String visitBoolIntEq(SudokuParser.BoolIntEqContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "== ";
    }

    @Override
    public String visitBoolIntNeq(SudokuParser.BoolIntNeqContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "== not ";
    }

    @Override
    public String visitBoolIntGt(SudokuParser.BoolIntGtContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "> ";
    }

    @Override
    public String visitBoolIntGte(SudokuParser.BoolIntGteContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + ">= ";
    }

    @Override
    public String visitBoolIntLt(SudokuParser.BoolIntLtContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "< ";
    }

    @Override
    public String visitBoolIntLte(SudokuParser.BoolIntLteContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "<= ";
    }

    // Float comparisons
    @Override
    public String visitBoolFloatEq(SudokuParser.BoolFloatEqContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "f== ";
    }

    @Override
    public String visitBoolFloatNeq(SudokuParser.BoolFloatNeqContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "f== not ";
    }

    @Override
    public String visitBoolFloatGt(SudokuParser.BoolFloatGtContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "f> ";
    }

    @Override
    public String visitBoolFloatGte(SudokuParser.BoolFloatGteContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "f>= ";
    }

    @Override
    public String visitBoolFloatLt(SudokuParser.BoolFloatLtContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "f< ";
    }

    @Override
    public String visitBoolFloatLte(SudokuParser.BoolFloatLteContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "f<= ";
    }

    // Mixed comparisons
    @Override
    public String visitBoolFloatEqInt(SudokuParser.BoolFloatEqIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "f== ";
    }

    @Override
    public String visitBoolIntEqFloat(SudokuParser.BoolIntEqFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "f== ";
    }

    @Override
    public String visitBoolFloatNeqInt(SudokuParser.BoolFloatNeqIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "f== not ";
    }

    @Override
    public String visitBoolIntNeqFloat(SudokuParser.BoolIntNeqFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "f== not ";
    }

    @Override
    public String visitBoolFloatGtInt(SudokuParser.BoolFloatGtIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "f> ";
    }

    @Override
    public String visitBoolIntGtFloat(SudokuParser.BoolIntGtFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "f> ";
    }

    @Override
    public String visitBoolFloatGteInt(SudokuParser.BoolFloatGteIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "f>= ";
    }

    @Override
    public String visitBoolIntGteFloat(SudokuParser.BoolIntGteFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "f>= ";
    }

    @Override
    public String visitBoolFloatLtInt(SudokuParser.BoolFloatLtIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "f< ";
    }

    @Override
    public String visitBoolIntLtFloat(SudokuParser.BoolIntLtFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "f< ";
    }

    @Override
    public String visitBoolFloatLteInt(SudokuParser.BoolFloatLteIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "f<= ";
    }

    @Override
    public String visitBoolIntLteFloat(SudokuParser.BoolIntLteFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "f<= ";
    }

    // String comparisons
    @Override
    public String visitBoolStrEq(SudokuParser.BoolStrEqContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "streq ";
    }

    @Override
    public String visitBoolStrNeq(SudokuParser.BoolStrNeqContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "streq not ";
    }

    @Override
    public String visitBoolStrEqIc(SudokuParser.BoolStrEqIcContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "sic== ";
    }

    @Override
    public String visitBoolStrNeqIc(SudokuParser.BoolStrNeqIcContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "sic== not ";
    }

    @Override
    public String visitBoolStrGt(SudokuParser.BoolStrGtContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "s> ";
    }

    @Override
    public String visitBoolStrLt(SudokuParser.BoolStrLtContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "s< ";
    }

    @Override
    public String visitBoolStrGte(SudokuParser.BoolStrGteContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "s>= ";
    }

    @Override
    public String visitBoolStrLte(SudokuParser.BoolStrLteContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "s<= ";
    }

    @Override
    public String visitBoolStartsWith(SudokuParser.BoolStartsWithContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "0 startswith ";
    }

    @Override
    public String visitBoolStartsWithAt(SudokuParser.BoolStartsWithAtContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + visit(ctx.iexpr()) + "startswith ";
    }

    @Override
    public String visitBoolMatches(SudokuParser.BoolMatchesContext ctx) {
        return visit(ctx.strexpr(1)) + visit(ctx.strexpr(0)) + "regexmatch ";
    }

    // Date comparisons
    @Override
    public String visitBoolDateEq(SudokuParser.BoolDateEqContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d== ";
    }

    @Override
    public String visitBoolDateLt(SudokuParser.BoolDateLtContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d< ";
    }

    @Override
    public String visitBoolDateGt(SudokuParser.BoolDateGtContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d> ";
    }

    @Override
    public String visitBoolDateGte(SudokuParser.BoolDateGteContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d< not ";
    }

    @Override
    public String visitBoolDateLte(SudokuParser.BoolDateLteContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d> not ";
    }

    @Override
    public String visitBoolDateBefore(SudokuParser.BoolDateBeforeContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d< ";
    }

    @Override
    public String visitBoolDateAfter(SudokuParser.BoolDateAfterContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d> ";
    }

    @Override
    public String visitBoolDateBetween(SudokuParser.BoolDateBetweenContext ctx) {
        String d1 = visit(ctx.dexpr(0));
        String d2 = visit(ctx.dexpr(1));
        String d3 = visit(ctx.dexpr(2));
        return d2 + d1 + "d> not " + d1 + d3 + " d> not and ";
    }

    // Entity comparisons
    @Override
    public String visitBoolEntityEq(SudokuParser.BoolEntityEqContext ctx) {
        return visit(ctx.eexpr(0)) + visit(ctx.eexpr(1)) + "req ";
    }

    @Override
    public String visitBoolEntityNeq(SudokuParser.BoolEntityNeqContext ctx) {
        return visit(ctx.eexpr(0)) + visit(ctx.eexpr(1)) + "req not ";
    }

    // Null tests
    @Override
    public String visitBoolBexprIsNull(SudokuParser.BoolBexprIsNullContext ctx) {
        return visit(ctx.bexpr()) + " isnull ";
    }

    @Override
    public String visitBoolBexprIsNotNull(SudokuParser.BoolBexprIsNotNullContext ctx) {
        return visit(ctx.bexpr()) + " isnull not ";
    }

    @Override
    public String visitBoolNumIsNull(SudokuParser.BoolNumIsNullContext ctx) {
        return visit(ctx.number()) + " isnull ";
    }

    @Override
    public String visitBoolNumIsNotNull(SudokuParser.BoolNumIsNotNullContext ctx) {
        return visit(ctx.number()) + " isnull not ";
    }

    @Override
    public String visitBoolDateIsNull(SudokuParser.BoolDateIsNullContext ctx) {
        return visit(ctx.dexpr()) + " isnull ";
    }

    @Override
    public String visitBoolDateIsNotNull(SudokuParser.BoolDateIsNotNullContext ctx) {
        return visit(ctx.dexpr()) + " isnull not ";
    }

    @Override
    public String visitBoolArrayIsNull(SudokuParser.BoolArrayIsNullContext ctx) {
        return visit(ctx.arrayExpr()) + " isnull ";
    }

    @Override
    public String visitBoolArrayIsNotNull(SudokuParser.BoolArrayIsNotNullContext ctx) {
        return visit(ctx.arrayExpr()) + " isnull not ";
    }

    @Override
    public String visitBoolStrIsNull(SudokuParser.BoolStrIsNullContext ctx) {
        return visit(ctx.strexpr()) + " isnull ";
    }

    @Override
    public String visitBoolStrIsNotNull(SudokuParser.BoolStrIsNotNullContext ctx) {
        return visit(ctx.strexpr()) + " isnull not ";
    }

    @Override
    public String visitBoolEntityIsNull(SudokuParser.BoolEntityIsNullContext ctx) {
        return visit(ctx.eexpr()) + " isnull ";
    }

    @Override
    public String visitBoolEntityIsNotNull(SudokuParser.BoolEntityIsNotNullContext ctx) {
        return visit(ctx.eexpr()) + " isnull not ";
    }

    // ========================================================================
    // Entity expressions
    // ========================================================================

    @Override
    public String visitEntityTyped(SudokuParser.EntityTypedContext ctx) {
        return resolveEntity(ctx.typedEntity()) + " ";
    }

    @Override
    public String visitEntityParen(SudokuParser.EntityParenContext ctx) {
        return visit(ctx.eexpr());
    }

    @Override
    public String visitEntityIndex(SudokuParser.EntityIndexContext ctx) {
        return visit(ctx.indxExpr());
    }

    @Override
    public String visitEntityNewName(SudokuParser.EntityNewNameContext ctx) {
        return visit(ctx.nexpr()) + " createentity ";
    }

    @Override
    public String visitEntityNewTyped(SudokuParser.EntityNewTypedContext ctx) {
        return "/" + resolveEntity(ctx.typedEntity()) + " createentity ";
    }

    @Override
    public String visitEntityClone(SudokuParser.EntityCloneContext ctx) {
        return visit(ctx.eexpr()) + "clone ";
    }

    @Override
    public String visitEntityColonRef(SudokuParser.EntityColonRefContext ctx) {
        String ref = visit(ctx.colonRef());
        String e = resolveEntity(ctx.typedEntity());
        return ref + "entitypush " + e + " entitypop ";
    }

    // ========================================================================
    // Date expressions
    // ========================================================================

    @Override
    public String visitDateTyped(SudokuParser.DateTypedContext ctx) {
        return resolveDate(ctx.typedDate()) + " ";
    }

    @Override
    public String visitDateParen(SudokuParser.DateParenContext ctx) {
        return visit(ctx.dexpr());
    }

    @Override
    public String visitDateColonRef(SudokuParser.DateColonRefContext ctx) {
        String ref = visit(ctx.colonRef());
        String d = resolveDate(ctx.typedDate());
        return ref + "entitypush " + d + " entitypop ";
    }

    @Override
    public String visitDateFromStrCast(SudokuParser.DateFromStrCastContext ctx) {
        return visit(ctx.strexpr()) + "cvd ";
    }

    @Override
    public String visitDateFromStrFunc(SudokuParser.DateFromStrFuncContext ctx) {
        return visit(ctx.strexpr()) + "cvd ";
    }

    @Override
    public String visitDateAdd(SudokuParser.DateAddContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d+ ";
    }

    @Override
    public String visitDateSub(SudokuParser.DateSubContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d- ";
    }

    @Override
    public String visitDateDays(SudokuParser.DateDaysContext ctx) {
        return visit(ctx.number()) + "days ";
    }

    @Override
    public String visitDateFirstOfYear(SudokuParser.DateFirstOfYearContext ctx) {
        return visit(ctx.dexpr()) + "firstofyear ";
    }

    @Override
    public String visitDateFirstOfMonth(SudokuParser.DateFirstOfMonthContext ctx) {
        return visit(ctx.dexpr()) + "firstofmonth ";
    }

    @Override
    public String visitDateEndOfMonth(SudokuParser.DateEndOfMonthContext ctx) {
        return visit(ctx.dexpr()) + "endofmonth ";
    }

    @Override
    public String visitDateUsing(SudokuParser.DateUsingContext ctx) {
        String ee = visit(ctx.eexpr());
        String e = visit(ctx.dexpr());
        return ee + "entitypush " + e + "entitypop ";
    }

    @Override
    public String visitDateEarliestAfter(SudokuParser.DateEarliestAfterContext ctx) {
        String a = visit(ctx.arrayExpr());
        String d = visit(ctx.dexpr());
        return d + "dup >r { { pop pop i i } over i d> if pop } " + a + "dup false sortarray for r> pop ";
    }

    // ========================================================================
    // Array expressions
    // ========================================================================

    @Override
    public String visitArrayPolicyStatements(SudokuParser.ArrayPolicyStatementsContext ctx) {
        return "policystatements ";
    }

    @Override
    public String visitArrayColonRef(SudokuParser.ArrayColonRefContext ctx) {
        String ref = visit(ctx.colonRef());
        String a = resolveArray(ctx.typedArray());
        return ref + "entitypush " + a + " entitypop ";
    }

    @Override
    public String visitArrayBase(SudokuParser.ArrayBaseContext ctx) {
        return visit(ctx.arrayExpr2());
    }

    @Override
    public String visitArrayTyped(SudokuParser.ArrayTypedContext ctx) {
        return resolveArray(ctx.typedArray()) + " ";
    }

    @Override
    public String visitArrayParen(SudokuParser.ArrayParenContext ctx) {
        return visit(ctx.arrayExpr());
    }

    @Override
    public String visitArrayCopy(SudokuParser.ArrayCopyContext ctx) {
        return visit(ctx.arrayExpr()) + "copyelements ";
    }

    @Override
    public String visitArrayCopySimple(SudokuParser.ArrayCopySimpleContext ctx) {
        return visit(ctx.arrayExpr()) + "copyelements ";
    }

    @Override
    public String visitArrayDeepCopy(SudokuParser.ArrayDeepCopyContext ctx) {
        return visit(ctx.arrayExpr()) + "deepcopy ";
    }

    @Override
    public String visitArrayDeepCopySimple(SudokuParser.ArrayDeepCopySimpleContext ctx) {
        return visit(ctx.arrayExpr()) + "deepcopy ";
    }

    @Override
    public String visitArrayOfValues(SudokuParser.ArrayOfValuesContext ctx) {
        return "mark " + visit(ctx.arrayList()) + " arraytomark ";
    }

    @Override
    public String visitArrayTokenize(SudokuParser.ArrayTokenizeContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "tokenize ";
    }

    @Override
    public String visitArrayLiteral(SudokuParser.ArrayLiteralContext ctx) {
        return visit(ctx.arrayLit());
    }

    @Override
    public String visitArrayLit(SudokuParser.ArrayLitContext ctx) {
        return "{ " + visit(ctx.arrayList()) + "} ";
    }

    @Override
    public String visitIndxExpr(SudokuParser.IndxExprContext ctx) {
        return visit(ctx.arrayExpr()) + visit(ctx.iexpr()) + "getat ";
    }

    // ========================================================================
    // Name expressions
    // ========================================================================

    @Override
    public String visitNameTyped(SudokuParser.NameTypedContext ctx) {
        return resolveName(ctx.typedName()) + " ";
    }

    @Override
    public String visitNameOf(SudokuParser.NameOfContext ctx) {
        return visit(ctx.eexpr()) + "getname ";
    }

    @Override
    public String visitNameTheName(SudokuParser.NameTheNameContext ctx) {
        return visit(ctx.strexpr()) + "cvn ";
    }

    @Override
    public String visitNameLiteral(SudokuParser.NameLiteralContext ctx) {
        String name = ctx.NAME().getText();
        if (name.startsWith("$")) {
            name = name.substring(1);
        }
        return "/" + name + " ";
    }

    @Override
    public String visitNameUsing(SudokuParser.NameUsingContext ctx) {
        String ee = visit(ctx.eexpr());
        String e = visit(ctx.nexpr());
        return ee + "entitypush " + e + "entitypop ";
    }

    @Override
    public String visitNameColonRef(SudokuParser.NameColonRefContext ctx) {
        String ref = visit(ctx.colonRef());
        String n = resolveName(ctx.typedName());
        return ref + "entitypush " + n + " entitypop ";
    }

    @Override
    public String visitNameFromStr(SudokuParser.NameFromStrContext ctx) {
        return visit(ctx.strexpr()) + "cvn ";
    }

    // ========================================================================
    // Table expressions
    // ========================================================================

    @Override
    public String visitTableTyped(SudokuParser.TableTypedContext ctx) {
        return resolveTable(ctx.typedTable()) + " ";
    }

    // ========================================================================
    // Debug statements
    // ========================================================================

    @Override
    public String visitDebugStr(SudokuParser.DebugStrContext ctx) {
        return visit(ctx.strexpr()) + "debug ";
    }

    @Override
    public String visitDebugBool(SudokuParser.DebugBoolContext ctx) {
        return visit(ctx.bexpr()) + "debug ";
    }

    @Override
    public String visitDebugInt(SudokuParser.DebugIntContext ctx) {
        return visit(ctx.iexpr()) + "debug ";
    }

    @Override
    public String visitDebugFloat(SudokuParser.DebugFloatContext ctx) {
        return visit(ctx.fexpr()) + "debug ";
    }

    @Override
    public String visitDebugEntity(SudokuParser.DebugEntityContext ctx) {
        return visit(ctx.eexpr()) + "debug ";
    }

    @Override
    public String visitDebugDate(SudokuParser.DebugDateContext ctx) {
        return visit(ctx.dexpr()) + "debug ";
    }

    @Override
    public String visitDebugArray(SudokuParser.DebugArrayContext ctx) {
        return visit(ctx.arrayExpr()) + "debug ";
    }

    @Override
    public String visitPrintStr(SudokuParser.PrintStrContext ctx) {
        return visit(ctx.strexpr()) + "print ";
    }

    @Override
    public String visitPrintBool(SudokuParser.PrintBoolContext ctx) {
        return visit(ctx.bexpr()) + "print ";
    }

    @Override
    public String visitPrintInt(SudokuParser.PrintIntContext ctx) {
        return visit(ctx.iexpr()) + "print ";
    }

    @Override
    public String visitPrintFloat(SudokuParser.PrintFloatContext ctx) {
        return visit(ctx.fexpr()) + "print ";
    }

    @Override
    public String visitPrintEntity(SudokuParser.PrintEntityContext ctx) {
        return visit(ctx.eexpr()) + "print ";
    }

    @Override
    public String visitPrintDate(SudokuParser.PrintDateContext ctx) {
        return visit(ctx.dexpr()) + "print ";
    }

    @Override
    public String visitPrintArray(SudokuParser.PrintArrayContext ctx) {
        return visit(ctx.arrayExpr()) + "print ";
    }

    // ========================================================================
    // Perform statements
    // ========================================================================

    @Override
    public String visitPerformDT(SudokuParser.PerformDTContext ctx) {
        return resolveDecisionTable(ctx.typedDecisionTable());
    }

    @Override
    public String visitPerformDTExplicit(SudokuParser.PerformDTExplicitContext ctx) {
        return resolveDecisionTable(ctx.typedDecisionTable());
    }

    @Override
    public String visitPerformName(SudokuParser.PerformNameContext ctx) {
        String name = ctx.NAME().getText();
        if (name.startsWith("$")) {
            name = name.substring(1);
        }
        return name + " ";
    }

    // ========================================================================
    // Set statements
    // ========================================================================

    @Override
    public String visitSetInt(SudokuParser.SetIntContext ctx) {
        String v = resolveIdentLeft(ctx.leftIexpr().getText());
        String e = visit(ctx.number());
        return e + "cvi " + v;
    }

    @Override
    public String visitSetFloat(SudokuParser.SetFloatContext ctx) {
        String v = resolveIdentLeft(ctx.leftFexpr().getText());
        String e = visit(ctx.number());
        return e + "cvr " + v;
    }

    @Override
    public String visitSetBool(SudokuParser.SetBoolContext ctx) {
        String v = resolveIdentLeft(ctx.leftBexpr().getText());
        String e = visit(ctx.bexpr());
        return e + "cvb " + v;
    }

    @Override
    public String visitSetEntity(SudokuParser.SetEntityContext ctx) {
        String v = resolveIdentLeft(ctx.leftEexpr().getText());
        String e = visit(ctx.eexpr());
        return e + "cve " + v;
    }

    @Override
    public String visitSetString(SudokuParser.SetStringContext ctx) {
        String v = resolveIdentLeft(ctx.leftStrexpr().getText());
        String e = visit(ctx.strexpr());
        return e + "cvs " + v;
    }

    @Override
    public String visitSetDate(SudokuParser.SetDateContext ctx) {
        String v = resolveIdentLeft(ctx.leftDexpr().getText());
        String e = visit(ctx.dexpr());
        return e + "cvd " + v;
    }

    @Override
    public String visitIncrementLong(SudokuParser.IncrementLongContext ctx) {
        String v = resolveLong(ctx.typedLong());
        String left = resolveIdentLeft(ctx.typedLong().getText());
        return v + " 1 ladd " + left;
    }

    @Override
    public String visitIncrementDouble(SudokuParser.IncrementDoubleContext ctx) {
        String v = resolveDouble(ctx.typedDouble());
        String left = resolveIdentLeft(ctx.typedDouble().getText());
        return v + " 1 dadd " + left;
    }

    @Override
    public String visitDecrementLong(SudokuParser.DecrementLongContext ctx) {
        String v = resolveLong(ctx.typedLong());
        String left = resolveIdentLeft(ctx.typedLong().getText());
        return v + " 1 lsub " + left;
    }

    @Override
    public String visitDecrementDouble(SudokuParser.DecrementDoubleContext ctx) {
        String v = resolveDouble(ctx.typedDouble());
        String left = resolveIdentLeft(ctx.typedDouble().getText());
        return v + " 1 dsub " + left;
    }

    // ========================================================================
    // If statements
    // ========================================================================

    @Override
    public String visitIfThen(SudokuParser.IfThenContext ctx) {
        String b = visit(ctx.bexpr());
        String blk = visit(ctx.block());
        return "{ " + blk + "} " + b + "if ";
    }

    @Override
    public String visitIfThenElse(SudokuParser.IfThenElseContext ctx) {
        String b = visit(ctx.bexpr());
        String blk1 = visit(ctx.block(0));
        String blk2 = visit(ctx.block(1));
        return "{ " + blk1 + "} { " + blk2 + "} " + b + "ifelse ";
    }

    @Override
    public String visitIfblock(SudokuParser.IfblockContext ctx) {
        String b = visit(ctx.bexpr());
        String e1 = visit(ctx.statementList());
        String e2 = visit(ctx.ifcontinue());
        if (e2.trim().length() > 0) {
            return "{ " + e1 + "} {" + e2 + "} " + b + "ifelse ";
        } else {
            return "{ " + e1 + "} " + b + "if ";
        }
    }

    @Override
    public String visitIfEnd(SudokuParser.IfEndContext ctx) {
        return "";
    }

    @Override
    public String visitIfElse(SudokuParser.IfElseContext ctx) {
        return visit(ctx.statementList());
    }

    @Override
    public String visitIfElseIf(SudokuParser.IfElseIfContext ctx) {
        return visit(ctx.ifblock());
    }

    // ========================================================================
    // Possessive and colon references
    // ========================================================================

    @Override
    public String visitPossessiveChain(SudokuParser.PossessiveChainContext ctx) {
        try {
            SudokuTypeResolver.ResolvedIdentifier resolved = typeResolver.resolvePossessive(ctx.POSSESSIVE().getText());
            String e2 = visit(ctx.possessiveRef());
            return resolved.value + " entitypush " + e2 + "entitypop ";
        } catch (RulesException e) {
            throw new RuntimeException(e);
        }
    }

    @Override
    public String visitPossessiveSingle(SudokuParser.PossessiveSingleContext ctx) {
        try {
            SudokuTypeResolver.ResolvedIdentifier resolved = typeResolver.resolvePossessive(ctx.POSSESSIVE().getText());
            return resolved.value + " ";
        } catch (RulesException e) {
            throw new RuntimeException(e);
        }
    }

    @Override
    public String visitColonPossessiveChain(SudokuParser.ColonPossessiveChainContext ctx) {
        String e1 = resolveEntity(ctx.typedEntity());
        String e2 = visit(ctx.possessiveRef());
        return e1 + " entitypush " + e2 + "entitypop ";
    }

    @Override
    public String visitColonPossessiveSingle(SudokuParser.ColonPossessiveSingleContext ctx) {
        return resolveEntity(ctx.typedEntity()) + " ";
    }
}
