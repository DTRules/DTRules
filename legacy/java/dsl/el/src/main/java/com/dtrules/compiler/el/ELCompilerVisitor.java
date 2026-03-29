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
package com.dtrules.compiler.el;

import org.antlr.v4.runtime.tree.ParseTree;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;

/**
 * ANTLR 4 Visitor that generates postfix code from the EL parse tree.
 * This replaces the semantic actions from the CUP parser.
 */
public class ELCompilerVisitor extends ELBaseVisitor<String> {

    private final ELTypeResolver typeResolver;
    private int localCnt = 0;

    public ELCompilerVisitor(ELTypeResolver typeResolver) {
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
    public String visitEmptyAction(ELParser.EmptyActionContext ctx) {
        return "";
    }

    @Override
    public String visitEmptyCondition(ELParser.EmptyConditionContext ctx) {
        return "";
    }

    @Override
    public String visitEmptyContext(ELParser.EmptyContextContext ctx) {
        return "execute ";
    }

    @Override
    public String visitEmptyPolicyStatement(ELParser.EmptyPolicyStatementContext ctx) {
        return "";
    }

    @Override
    public String visitActionStatement(ELParser.ActionStatementContext ctx) {
        return "\n" + visit(ctx.statementList()) + "\n";
    }

    @Override
    public String visitConditionExpr(ELParser.ConditionExprContext ctx) {
        return "\n" + visit(ctx.bexpr()) + "\n";
    }

    @Override
    public String visitConditionDebugBefore(ELParser.ConditionDebugBeforeContext ctx) {
        return "\n" + visit(ctx.debugstatement()) + visit(ctx.bexpr()) + "\n";
    }

    @Override
    public String visitConditionDebugAfter(ELParser.ConditionDebugAfterContext ctx) {
        return "\n" + visit(ctx.bexpr()) + visit(ctx.debugstatement()) + "\n";
    }

    @Override
    public String visitContextStatement(ELParser.ContextStatementContext ctx) {
        return "\n" + visit(ctx.contextForTable()) + "\n";
    }

    @Override
    public String visitContextDebugBefore(ELParser.ContextDebugBeforeContext ctx) {
        return "\n" + visit(ctx.debugstatement()) + visit(ctx.contextForTable()) + "\n";
    }

    @Override
    public String visitPolicyStrExpr(ELParser.PolicyStrExprContext ctx) {
        return visit(ctx.strexpr());
    }

    @Override
    public String visitPolicyNExpr(ELParser.PolicyNExprContext ctx) {
        return visit(ctx.nexpr());
    }

    @Override
    public String visitPolicyIExpr(ELParser.PolicyIExprContext ctx) {
        return visit(ctx.iexpr());
    }

    @Override
    public String visitPolicyFExpr(ELParser.PolicyFExprContext ctx) {
        return visit(ctx.fexpr());
    }

    @Override
    public String visitPolicyBExpr(ELParser.PolicyBExprContext ctx) {
        return visit(ctx.bexpr());
    }

    @Override
    public String visitPolicyDExpr(ELParser.PolicyDExprContext ctx) {
        return visit(ctx.dexpr());
    }

    // ========================================================================
    // Statement list and blocks
    // ========================================================================

    @Override
    public String visitStatementList(ELParser.StatementListContext ctx) {
        StringBuilder sb = new StringBuilder();
        for (ELParser.BlockContext block : ctx.block()) {
            sb.append(visit(block));
        }
        return sb.toString();
    }

    @Override
    public String visitBlockCurly(ELParser.BlockCurlyContext ctx) {
        return visit(ctx.statementList()) + "\n";
    }

    @Override
    public String visitBlockUsing(ELParser.BlockUsingContext ctx) {
        return visit(ctx.usingblock());
    }

    @Override
    public String visitBlockGforall(ELParser.BlockGforallContext ctx) {
        String blk = visit(ctx.statementList());
        String ctl = visit(ctx.forallctl());
        return "{ " + blk + "} " + ctl + "pop ";
    }

    @Override
    public String visitBlockForall(ELParser.BlockForallContext ctx) {
        return visit(ctx.forallblock());
    }

    @Override
    public String visitBlockForeach(ELParser.BlockForeachContext ctx) {
        return visit(ctx.foreachblock());
    }

    @Override
    public String visitBlockFirst(ELParser.BlockFirstContext ctx) {
        return visit(ctx.firstblock());
    }

    @Override
    public String visitBlockIf(ELParser.BlockIfContext ctx) {
        return visit(ctx.ifblock());
    }

    @Override
    public String visitBlockStatement(ELParser.BlockStatementContext ctx) {
        return visit(ctx.statement());
    }

    // ========================================================================
    // Using blocks
    // ========================================================================

    @Override
    public String visitUsingBlockEntity(ELParser.UsingBlockEntityContext ctx) {
        String ee = resolveEntity(ctx.typedEntity());
        String e = visit(ctx.usingblock());
        return ee + " entitypush " + e + "entitypop ";
    }

    @Override
    public String visitUsingBlockEntityComma(ELParser.UsingBlockEntityCommaContext ctx) {
        String ee = resolveEntity(ctx.typedEntity());
        String e = visit(ctx.usingblock());
        return ee + " entitypush " + e + "entitypop ";
    }

    @Override
    public String visitUsingBlockBase(ELParser.UsingBlockBaseContext ctx) {
        return visit(ctx.block());
    }

    // ========================================================================
    // Context for table
    // ========================================================================

    @Override
    public String visitContextDebug(ELParser.ContextDebugContext ctx) {
        return visit(ctx.debugstatement()) + "execute ";
    }

    @Override
    public String visitContextFor(ELParser.ContextForContext ctx) {
        return visit(ctx.forctl()) + "pop ";
    }

    @Override
    public String visitContextForallCtl(ELParser.ContextForallCtlContext ctx) {
        return visit(ctx.forallctl()) + "pop ";
    }

    @Override
    public String visitContextForfirst(ELParser.ContextForfirstContext ctx) {
        return visit(ctx.forfirstctl()) + "pop ";
    }

    @Override
    public String visitContextCtx(ELParser.ContextCtxContext ctx) {
        return visit(ctx.contextstatement()) + "execute entitypop ";
    }

    @Override
    public String visitContextLocal(ELParser.ContextLocalContext ctx) {
        return visit(ctx.localvariables());
    }

    // ========================================================================
    // Helper methods for type resolution
    // ========================================================================

    private String resolveEntity(ELParser.TypedEntityContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveLong(ELParser.TypedLongContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveDouble(ELParser.TypedDoubleContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveString(ELParser.TypedStringContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveBoolean(ELParser.TypedBooleanContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveDate(ELParser.TypedDateContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveArray(ELParser.TypedArrayContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveTable(ELParser.TypedTableContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveName(ELParser.TypedNameContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveDecisionTable(ELParser.TypedDecisionTableContext ctx) {
        return "/" + ctx.IDENT().getText() + " execute ";
    }

    private String resolveIdent(String ident) {
        try {
            ELTypeResolver.ResolvedIdentifier resolved = typeResolver.resolve(ident);
            return resolved.value;
        } catch (RulesException e) {
            throw new RuntimeException("Failed to resolve identifier: " + ident, e);
        }
    }

    private String resolveIdentLeft(String ident) {
        try {
            ELTypeResolver.ResolvedIdentifier resolved = typeResolver.resolve(ident);
            return resolved.leftValue;
        } catch (RulesException e) {
            throw new RuntimeException("Failed to resolve identifier: " + ident, e);
        }
    }

    // ========================================================================
    // Integer expressions
    // ========================================================================

    @Override
    public String visitIntAdd(ELParser.IntAddContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "+ ";
    }

    @Override
    public String visitIntSub(ELParser.IntSubContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "- ";
    }

    @Override
    public String visitIntMul(ELParser.IntMulContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "* ";
    }

    @Override
    public String visitIntDiv(ELParser.IntDivContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "div ";
    }

    @Override
    public String visitIntLiteral(ELParser.IntLiteralContext ctx) {
        return ctx.INT_LITERAL().getText() + " ";
    }

    @Override
    public String visitIntNegate(ELParser.IntNegateContext ctx) {
        return visit(ctx.iexpr()) + "negate ";
    }

    @Override
    public String visitIntParen(ELParser.IntParenContext ctx) {
        return visit(ctx.iexpr()) + " ";
    }

    @Override
    public String visitIntTyped(ELParser.IntTypedContext ctx) {
        return resolveLong(ctx.typedLong()) + " ";
    }

    @Override
    public String visitIntColonRef(ELParser.IntColonRefContext ctx) {
        String ref = visit(ctx.colonRef());
        String n = resolveLong(ctx.typedLong());
        return ref + "entitypush " + n + " entitypop ";
    }

    @Override
    public String visitIntNumberOf(ELParser.IntNumberOfContext ctx) {
        return visit(ctx.arrayExpr()) + "length ";
    }

    @Override
    public String visitIntNumberOfWhere(ELParser.IntNumberOfWhereContext ctx) {
        String a = visit(ctx.arrayExpr());
        String b = visit(ctx.bexpr());
        return "0 { { 1 ladd } " + b + "if }" + a + "forall ";
    }

    @Override
    public String visitIntLengthArray(ELParser.IntLengthArrayContext ctx) {
        return visit(ctx.arrayExpr()) + "length ";
    }

    @Override
    public String visitIntLengthStr(ELParser.IntLengthStrContext ctx) {
        return visit(ctx.strexpr()) + "strlength ";
    }

    @Override
    public String visitIntIndexOf(ELParser.IntIndexOfContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "indexof ";
    }

    @Override
    public String visitIntDaysBetween(ELParser.IntDaysBetweenContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "daysbetween ";
    }

    @Override
    public String visitIntMonthsBetween(ELParser.IntMonthsBetweenContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "monthsbetween ";
    }

    @Override
    public String visitIntYearsBetween(ELParser.IntYearsBetweenContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "yearsbetween ";
    }

    @Override
    public String visitIntYearOf(ELParser.IntYearOfContext ctx) {
        return visit(ctx.dexpr()) + "yearof ";
    }

    @Override
    public String visitIntAbs(ELParser.IntAbsContext ctx) {
        return visit(ctx.iexpr()) + "labs ";
    }

    @Override
    public String visitIntUsing(ELParser.IntUsingContext ctx) {
        String ee = visit(ctx.eexpr());
        String e = visit(ctx.iexpr());
        return ee + "entitypush " + e + "entitypop ";
    }

    // ========================================================================
    // Float expressions
    // ========================================================================

    @Override
    public String visitFloatLiteral(ELParser.FloatLiteralContext ctx) {
        return ctx.FLOAT_LITERAL().getText() + " ";
    }

    @Override
    public String visitFloatTyped(ELParser.FloatTypedContext ctx) {
        return resolveDouble(ctx.typedDouble()) + " ";
    }

    @Override
    public String visitFloatColonRef(ELParser.FloatColonRefContext ctx) {
        String ref = visit(ctx.colonRef());
        String f = resolveDouble(ctx.typedDouble());
        return ref + "entitypush " + f + " entitypop ";
    }

    @Override
    public String visitFloatAddInt(ELParser.FloatAddIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "fadd ";
    }

    @Override
    public String visitFloatAddFloat(ELParser.FloatAddFloatContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "fadd ";
    }

    @Override
    public String visitIntAddFloat(ELParser.IntAddFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "fadd ";
    }

    @Override
    public String visitFloatSubInt(ELParser.FloatSubIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "fsub ";
    }

    @Override
    public String visitIntSubFloat(ELParser.IntSubFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "fsub ";
    }

    @Override
    public String visitFloatSubFloat(ELParser.FloatSubFloatContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "fsub ";
    }

    @Override
    public String visitFloatMulInt(ELParser.FloatMulIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "fmul ";
    }

    @Override
    public String visitIntMulFloat(ELParser.IntMulFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "fmul ";
    }

    @Override
    public String visitFloatMulFloat(ELParser.FloatMulFloatContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "fmul ";
    }

    @Override
    public String visitFloatDivInt(ELParser.FloatDivIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "fdiv ";
    }

    @Override
    public String visitIntDivFloat(ELParser.IntDivFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "fdiv ";
    }

    @Override
    public String visitFloatDivFloat(ELParser.FloatDivFloatContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "fdiv ";
    }

    @Override
    public String visitFloatNegate(ELParser.FloatNegateContext ctx) {
        return visit(ctx.fexpr()) + "fnegate ";
    }

    @Override
    public String visitFloatParen(ELParser.FloatParenContext ctx) {
        return visit(ctx.fexpr());
    }

    @Override
    public String visitFloatAbs(ELParser.FloatAbsContext ctx) {
        return visit(ctx.fexpr()) + "fabs ";
    }

    @Override
    public String visitFloatUsing(ELParser.FloatUsingContext ctx) {
        String ee = visit(ctx.eexpr());
        String e = visit(ctx.fexpr());
        return ee + "entitypush " + e + "entitypop ";
    }

    @Override
    public String visitFloatRounded(ELParser.FloatRoundedContext ctx) {
        return visit(ctx.fexpr()) + "0 0.5 roundto ";
    }

    @Override
    public String visitFloatRoundedTo(ELParser.FloatRoundedToContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "0.5 roundto ";
    }

    @Override
    public String visitFloatRoundedBoundry(ELParser.FloatRoundedBoundryContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.iexpr()) + visit(ctx.fexpr(1)) + " roundto ";
    }

    // ========================================================================
    // String expressions
    // ========================================================================

    @Override
    public String visitStrLiteral(ELParser.StrLiteralContext ctx) {
        return ctx.STRING_LITERAL().getText() + " ";
    }

    @Override
    public String visitStrTyped(ELParser.StrTypedContext ctx) {
        return resolveString(ctx.typedString()) + " ";
    }

    @Override
    public String visitStrColonRef(ELParser.StrColonRefContext ctx) {
        String ref = visit(ctx.colonRef());
        String s = visit(ctx.strexpr());
        return ref + "entitypush " + s + "entitypop ";
    }

    @Override
    public String visitStrConcat(ELParser.StrConcatContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "strconcat ";
    }

    @Override
    public String visitStrConcatInt(ELParser.StrConcatIntContext ctx) {
        return visit(ctx.strexpr()) + visit(ctx.iexpr()) + "strconcat ";
    }

    @Override
    public String visitStrConcatFloat(ELParser.StrConcatFloatContext ctx) {
        return visit(ctx.strexpr()) + visit(ctx.fexpr()) + "strconcat ";
    }

    @Override
    public String visitStrConcatName(ELParser.StrConcatNameContext ctx) {
        return visit(ctx.strexpr()) + visit(ctx.nexpr()) + "strconcat ";
    }

    @Override
    public String visitStrConcatEntity(ELParser.StrConcatEntityContext ctx) {
        return visit(ctx.strexpr()) + visit(ctx.eexpr()) + "strconcat ";
    }

    @Override
    public String visitStrConcatDate(ELParser.StrConcatDateContext ctx) {
        return visit(ctx.strexpr()) + visit(ctx.dexpr()) + "strconcat ";
    }

    @Override
    public String visitStrConcatArray(ELParser.StrConcatArrayContext ctx) {
        return visit(ctx.strexpr()) + visit(ctx.arrayExpr()) + "strconcat ";
    }

    @Override
    public String visitStrTrim(ELParser.StrTrimContext ctx) {
        return visit(ctx.strexpr()) + "strtrim ";
    }

    @Override
    public String visitStrToLower(ELParser.StrToLowerContext ctx) {
        return visit(ctx.strexpr()) + "tolowercase ";
    }

    @Override
    public String visitStrToUpper(ELParser.StrToUpperContext ctx) {
        return visit(ctx.strexpr()) + "touppercase ";
    }

    @Override
    public String visitStrTimestamp(ELParser.StrTimestampContext ctx) {
        return "getdate gettimestamp ";
    }

    @Override
    public String visitStrUsing(ELParser.StrUsingContext ctx) {
        String ee = visit(ctx.eexpr());
        String e = visit(ctx.strexpr());
        return ee + "entitypush " + e + "entitypop ";
    }

    @Override
    public String visitStrParen(ELParser.StrParenContext ctx) {
        return visit(ctx.strexpr());
    }

    @Override
    public String visitStrSubstring(ELParser.StrSubstringContext ctx) {
        String s = visit(ctx.strexpr());
        String start = visit(ctx.iexpr(0));
        String end = visit(ctx.iexpr(1));
        return end + start + s + "substring ";
    }

    @Override
    public String visitStrValueOfFloat(ELParser.StrValueOfFloatContext ctx) {
        return visit(ctx.fexpr()) + "tostring ";
    }

    @Override
    public String visitStrValueOfInt(ELParser.StrValueOfIntContext ctx) {
        return visit(ctx.iexpr()) + "tostring ";
    }

    @Override
    public String visitStrValueOfDate(ELParser.StrValueOfDateContext ctx) {
        return visit(ctx.dexpr()) + "tostring ";
    }

    @Override
    public String visitStrValueOfBool(ELParser.StrValueOfBoolContext ctx) {
        return visit(ctx.bexpr()) + "tostring ";
    }

    @Override
    public String visitStrTableInfo(ELParser.StrTableInfoContext ctx) {
        return "actionstring ";
    }

    @Override
    public String visitStrMappingKey(ELParser.StrMappingKeyContext ctx) {
        return "\"mapping*key\" cvn execute ";
    }

    // ========================================================================
    // Boolean expressions
    // ========================================================================

    @Override
    public String visitBoolTyped(ELParser.BoolTypedContext ctx) {
        return resolveBoolean(ctx.typedBoolean()) + " ";
    }

    @Override
    public String visitBoolLiteral(ELParser.BoolLiteralContext ctx) {
        // Boolean literals: true, false, default, otherwise, always, perform when called
        return ctx.RBOOLEAN().getText().toLowerCase() + " ";
    }

    @Override
    public String visitBoolColonRef(ELParser.BoolColonRefContext ctx) {
        String ref = visit(ctx.colonRef());
        String b = resolveBoolean(ctx.typedBoolean());
        return ref + "entitypush " + b + " entitypop ";
    }

    @Override
    public String visitBoolAnd(ELParser.BoolAndContext ctx) {
        String e1 = visit(ctx.bexpr(0));
        String e2 = visit(ctx.bexpr(1));
        return e1 + "{ pop " + e2 + "} over if\n";
    }

    @Override
    public String visitBoolOr(ELParser.BoolOrContext ctx) {
        String e1 = visit(ctx.bexpr(0));
        String e2 = visit(ctx.bexpr(1));
        return e1 + "{ pop " + e2 + "} over not if\n";
    }

    @Override
    public String visitBoolNot(ELParser.BoolNotContext ctx) {
        return visit(ctx.bexpr()) + "not ";
    }

    @Override
    public String visitBoolParen(ELParser.BoolParenContext ctx) {
        return visit(ctx.bexpr());
    }

    @Override
    public String visitBoolUsing(ELParser.BoolUsingContext ctx) {
        String ee = visit(ctx.eexpr());
        String e = visit(ctx.bexpr());
        return ee + "entitypush " + e + "entitypop ";
    }

    // Integer comparisons
    @Override
    public String visitBoolIntEq(ELParser.BoolIntEqContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "== ";
    }

    @Override
    public String visitBoolIntNeq(ELParser.BoolIntNeqContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "== not ";
    }

    @Override
    public String visitBoolIntGt(ELParser.BoolIntGtContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "> ";
    }

    @Override
    public String visitBoolIntGte(ELParser.BoolIntGteContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + ">= ";
    }

    @Override
    public String visitBoolIntLt(ELParser.BoolIntLtContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "< ";
    }

    @Override
    public String visitBoolIntLte(ELParser.BoolIntLteContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "<= ";
    }

    // Float comparisons
    @Override
    public String visitBoolFloatEq(ELParser.BoolFloatEqContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "f== ";
    }

    @Override
    public String visitBoolFloatNeq(ELParser.BoolFloatNeqContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "f== not ";
    }

    @Override
    public String visitBoolFloatGt(ELParser.BoolFloatGtContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "f> ";
    }

    @Override
    public String visitBoolFloatGte(ELParser.BoolFloatGteContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "f>= ";
    }

    @Override
    public String visitBoolFloatLt(ELParser.BoolFloatLtContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "f< ";
    }

    @Override
    public String visitBoolFloatLte(ELParser.BoolFloatLteContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "f<= ";
    }

    // Mixed comparisons
    @Override
    public String visitBoolFloatEqInt(ELParser.BoolFloatEqIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "f== ";
    }

    @Override
    public String visitBoolIntEqFloat(ELParser.BoolIntEqFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "f== ";
    }

    @Override
    public String visitBoolFloatNeqInt(ELParser.BoolFloatNeqIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "f== not ";
    }

    @Override
    public String visitBoolIntNeqFloat(ELParser.BoolIntNeqFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "f== not ";
    }

    @Override
    public String visitBoolFloatGtInt(ELParser.BoolFloatGtIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "f> ";
    }

    @Override
    public String visitBoolIntGtFloat(ELParser.BoolIntGtFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "f> ";
    }

    @Override
    public String visitBoolFloatGteInt(ELParser.BoolFloatGteIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "f>= ";
    }

    @Override
    public String visitBoolIntGteFloat(ELParser.BoolIntGteFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "f>= ";
    }

    @Override
    public String visitBoolFloatLtInt(ELParser.BoolFloatLtIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "f< ";
    }

    @Override
    public String visitBoolIntLtFloat(ELParser.BoolIntLtFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "f< ";
    }

    @Override
    public String visitBoolFloatLteInt(ELParser.BoolFloatLteIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "f<= ";
    }

    @Override
    public String visitBoolIntLteFloat(ELParser.BoolIntLteFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "f<= ";
    }

    // String comparisons
    @Override
    public String visitBoolStrEq(ELParser.BoolStrEqContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "streq ";
    }

    @Override
    public String visitBoolStrNeq(ELParser.BoolStrNeqContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "streq not ";
    }

    @Override
    public String visitBoolStrEqIc(ELParser.BoolStrEqIcContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "sic== ";
    }

    @Override
    public String visitBoolStrNeqIc(ELParser.BoolStrNeqIcContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "sic== not ";
    }

    @Override
    public String visitBoolStrGt(ELParser.BoolStrGtContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "s> ";
    }

    @Override
    public String visitBoolStrLt(ELParser.BoolStrLtContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "s< ";
    }

    @Override
    public String visitBoolStrGte(ELParser.BoolStrGteContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "s>= ";
    }

    @Override
    public String visitBoolStrLte(ELParser.BoolStrLteContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "s<= ";
    }

    @Override
    public String visitBoolStartsWith(ELParser.BoolStartsWithContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "0 startswith ";
    }

    @Override
    public String visitBoolStartsWithAt(ELParser.BoolStartsWithAtContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + visit(ctx.iexpr()) + "startswith ";
    }

    @Override
    public String visitBoolMatches(ELParser.BoolMatchesContext ctx) {
        return visit(ctx.strexpr(1)) + visit(ctx.strexpr(0)) + "regexmatch ";
    }

    // Date comparisons
    @Override
    public String visitBoolDateEq(ELParser.BoolDateEqContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d== ";
    }

    @Override
    public String visitBoolDateLt(ELParser.BoolDateLtContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d< ";
    }

    @Override
    public String visitBoolDateGt(ELParser.BoolDateGtContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d> ";
    }

    @Override
    public String visitBoolDateGte(ELParser.BoolDateGteContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d< not ";
    }

    @Override
    public String visitBoolDateLte(ELParser.BoolDateLteContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d> not ";
    }

    @Override
    public String visitBoolDateBefore(ELParser.BoolDateBeforeContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d< ";
    }

    @Override
    public String visitBoolDateAfter(ELParser.BoolDateAfterContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d> ";
    }

    @Override
    public String visitBoolDateBetween(ELParser.BoolDateBetweenContext ctx) {
        String d1 = visit(ctx.dexpr(0));
        String d2 = visit(ctx.dexpr(1));
        String d3 = visit(ctx.dexpr(2));
        return d2 + d1 + "d> not " + d1 + d3 + " d> not and ";
    }

    // Entity comparisons
    @Override
    public String visitBoolEntityEq(ELParser.BoolEntityEqContext ctx) {
        return visit(ctx.eexpr(0)) + visit(ctx.eexpr(1)) + "req ";
    }

    @Override
    public String visitBoolEntityNeq(ELParser.BoolEntityNeqContext ctx) {
        return visit(ctx.eexpr(0)) + visit(ctx.eexpr(1)) + "req not ";
    }

    // Null tests
    @Override
    public String visitBoolBexprIsNull(ELParser.BoolBexprIsNullContext ctx) {
        return visit(ctx.bexpr()) + " isnull ";
    }

    @Override
    public String visitBoolBexprIsNotNull(ELParser.BoolBexprIsNotNullContext ctx) {
        return visit(ctx.bexpr()) + " isnull not ";
    }

    @Override
    public String visitBoolNumIsNull(ELParser.BoolNumIsNullContext ctx) {
        return visit(ctx.number()) + " isnull ";
    }

    @Override
    public String visitBoolNumIsNotNull(ELParser.BoolNumIsNotNullContext ctx) {
        return visit(ctx.number()) + " isnull not ";
    }

    @Override
    public String visitBoolDateIsNull(ELParser.BoolDateIsNullContext ctx) {
        return visit(ctx.dexpr()) + " isnull ";
    }

    @Override
    public String visitBoolDateIsNotNull(ELParser.BoolDateIsNotNullContext ctx) {
        return visit(ctx.dexpr()) + " isnull not ";
    }

    @Override
    public String visitBoolArrayIsNull(ELParser.BoolArrayIsNullContext ctx) {
        return visit(ctx.arrayExpr()) + " isnull ";
    }

    @Override
    public String visitBoolArrayIsNotNull(ELParser.BoolArrayIsNotNullContext ctx) {
        return visit(ctx.arrayExpr()) + " isnull not ";
    }

    @Override
    public String visitBoolStrIsNull(ELParser.BoolStrIsNullContext ctx) {
        return visit(ctx.strexpr()) + " isnull ";
    }

    @Override
    public String visitBoolStrIsNotNull(ELParser.BoolStrIsNotNullContext ctx) {
        return visit(ctx.strexpr()) + " isnull not ";
    }

    @Override
    public String visitBoolEntityIsNull(ELParser.BoolEntityIsNullContext ctx) {
        return visit(ctx.eexpr()) + " isnull ";
    }

    @Override
    public String visitBoolEntityIsNotNull(ELParser.BoolEntityIsNotNullContext ctx) {
        return visit(ctx.eexpr()) + " isnull not ";
    }

    // ========================================================================
    // Entity expressions
    // ========================================================================

    @Override
    public String visitEntityTyped(ELParser.EntityTypedContext ctx) {
        return resolveEntity(ctx.typedEntity()) + " ";
    }

    @Override
    public String visitEntityParen(ELParser.EntityParenContext ctx) {
        return visit(ctx.eexpr());
    }

    @Override
    public String visitEntityIndex(ELParser.EntityIndexContext ctx) {
        return visit(ctx.indxExpr());
    }

    @Override
    public String visitEntityNewName(ELParser.EntityNewNameContext ctx) {
        return visit(ctx.nexpr()) + " createentity ";
    }

    @Override
    public String visitEntityNewTyped(ELParser.EntityNewTypedContext ctx) {
        return "/" + resolveEntity(ctx.typedEntity()) + " createentity ";
    }

    @Override
    public String visitEntityClone(ELParser.EntityCloneContext ctx) {
        return visit(ctx.eexpr()) + "clone ";
    }

    @Override
    public String visitEntityColonRef(ELParser.EntityColonRefContext ctx) {
        String ref = visit(ctx.colonRef());
        String e = resolveEntity(ctx.typedEntity());
        return ref + "entitypush " + e + " entitypop ";
    }

    // ========================================================================
    // Date expressions
    // ========================================================================

    @Override
    public String visitDateTyped(ELParser.DateTypedContext ctx) {
        return resolveDate(ctx.typedDate()) + " ";
    }

    @Override
    public String visitDateParen(ELParser.DateParenContext ctx) {
        return visit(ctx.dexpr());
    }

    @Override
    public String visitDateColonRef(ELParser.DateColonRefContext ctx) {
        String ref = visit(ctx.colonRef());
        String d = resolveDate(ctx.typedDate());
        return ref + "entitypush " + d + " entitypop ";
    }

    @Override
    public String visitDateFromStrCast(ELParser.DateFromStrCastContext ctx) {
        return visit(ctx.strexpr()) + "cvd ";
    }

    @Override
    public String visitDateFromStrFunc(ELParser.DateFromStrFuncContext ctx) {
        return visit(ctx.strexpr()) + "cvd ";
    }

    @Override
    public String visitDateCurrentDate(ELParser.DateCurrentDateContext ctx) {
        return "getdate ";
    }

    @Override
    public String visitDateAdd(ELParser.DateAddContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d+ ";
    }

    @Override
    public String visitDateSub(ELParser.DateSubContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d- ";
    }

    @Override
    public String visitDateDays(ELParser.DateDaysContext ctx) {
        return visit(ctx.number()) + "days ";
    }

    @Override
    public String visitDateFirstOfYear(ELParser.DateFirstOfYearContext ctx) {
        return visit(ctx.dexpr()) + "firstofyear ";
    }

    @Override
    public String visitDateFirstOfMonth(ELParser.DateFirstOfMonthContext ctx) {
        return visit(ctx.dexpr()) + "firstofmonth ";
    }

    @Override
    public String visitDateEndOfMonth(ELParser.DateEndOfMonthContext ctx) {
        return visit(ctx.dexpr()) + "endofmonth ";
    }

    @Override
    public String visitDateUsing(ELParser.DateUsingContext ctx) {
        String ee = visit(ctx.eexpr());
        String e = visit(ctx.dexpr());
        return ee + "entitypush " + e + "entitypop ";
    }

    // ========================================================================
    // Array expressions
    // ========================================================================

    @Override
    public String visitArrayPolicyStatements(ELParser.ArrayPolicyStatementsContext ctx) {
        return "policystatements ";
    }

    @Override
    public String visitArrayColonRef(ELParser.ArrayColonRefContext ctx) {
        String ref = visit(ctx.colonRef());
        String a = resolveArray(ctx.typedArray());
        return ref + "entitypush " + a + " entitypop ";
    }

    @Override
    public String visitArrayBase(ELParser.ArrayBaseContext ctx) {
        return visit(ctx.arrayExpr2());
    }

    @Override
    public String visitArrayTyped(ELParser.ArrayTypedContext ctx) {
        return resolveArray(ctx.typedArray()) + " ";
    }

    @Override
    public String visitArrayParen(ELParser.ArrayParenContext ctx) {
        return visit(ctx.arrayExpr());
    }

    @Override
    public String visitArrayCopy(ELParser.ArrayCopyContext ctx) {
        return visit(ctx.arrayExpr()) + "copyelements ";
    }

    @Override
    public String visitArrayCopySimple(ELParser.ArrayCopySimpleContext ctx) {
        return visit(ctx.arrayExpr()) + "copyelements ";
    }

    @Override
    public String visitArrayDeepCopy(ELParser.ArrayDeepCopyContext ctx) {
        return visit(ctx.arrayExpr()) + "deepcopy ";
    }

    @Override
    public String visitArrayDeepCopySimple(ELParser.ArrayDeepCopySimpleContext ctx) {
        return visit(ctx.arrayExpr()) + "deepcopy ";
    }

    @Override
    public String visitArrayOfValues(ELParser.ArrayOfValuesContext ctx) {
        return "mark " + visit(ctx.arrayList()) + " arraytomark ";
    }

    @Override
    public String visitArrayTokenize(ELParser.ArrayTokenizeContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "tokenize ";
    }

    @Override
    public String visitArrayLiteral(ELParser.ArrayLiteralContext ctx) {
        return visit(ctx.arrayLit());
    }

    @Override
    public String visitArrayLit(ELParser.ArrayLitContext ctx) {
        return "{ " + visit(ctx.arrayList()) + "} ";
    }

    @Override
    public String visitIndxExpr(ELParser.IndxExprContext ctx) {
        return visit(ctx.arrayExpr()) + visit(ctx.iexpr()) + "getat ";
    }

    // ========================================================================
    // Name expressions
    // ========================================================================

    @Override
    public String visitNameTyped(ELParser.NameTypedContext ctx) {
        return resolveName(ctx.typedName()) + " ";
    }

    @Override
    public String visitNameOf(ELParser.NameOfContext ctx) {
        return visit(ctx.eexpr()) + "getname ";
    }

    @Override
    public String visitNameTheName(ELParser.NameTheNameContext ctx) {
        return visit(ctx.strexpr()) + "cvn ";
    }

    @Override
    public String visitNameLiteral(ELParser.NameLiteralContext ctx) {
        String name = ctx.NAME().getText();
        if (name.startsWith("$")) {
            name = name.substring(1);
        }
        return "/" + name + " ";
    }

    @Override
    public String visitNameUsing(ELParser.NameUsingContext ctx) {
        String ee = visit(ctx.eexpr());
        String e = visit(ctx.nexpr());
        return ee + "entitypush " + e + "entitypop ";
    }

    @Override
    public String visitNameColonRef(ELParser.NameColonRefContext ctx) {
        String ref = visit(ctx.colonRef());
        String n = resolveName(ctx.typedName());
        return ref + "entitypush " + n + " entitypop ";
    }

    @Override
    public String visitNameFromStr(ELParser.NameFromStrContext ctx) {
        return visit(ctx.strexpr()) + "cvn ";
    }

    // ========================================================================
    // Table expressions
    // ========================================================================

    @Override
    public String visitTableTyped(ELParser.TableTypedContext ctx) {
        return resolveTable(ctx.typedTable()) + " ";
    }

    // ========================================================================
    // Debug statements
    // ========================================================================

    @Override
    public String visitDebugStr(ELParser.DebugStrContext ctx) {
        return visit(ctx.strexpr()) + "debug ";
    }

    @Override
    public String visitDebugBool(ELParser.DebugBoolContext ctx) {
        return visit(ctx.bexpr()) + "debug ";
    }

    @Override
    public String visitDebugInt(ELParser.DebugIntContext ctx) {
        return visit(ctx.iexpr()) + "debug ";
    }

    @Override
    public String visitDebugFloat(ELParser.DebugFloatContext ctx) {
        return visit(ctx.fexpr()) + "debug ";
    }

    @Override
    public String visitDebugEntity(ELParser.DebugEntityContext ctx) {
        return visit(ctx.eexpr()) + "debug ";
    }

    @Override
    public String visitDebugDate(ELParser.DebugDateContext ctx) {
        return visit(ctx.dexpr()) + "debug ";
    }

    @Override
    public String visitDebugArray(ELParser.DebugArrayContext ctx) {
        return visit(ctx.arrayExpr()) + "debug ";
    }

    @Override
    public String visitPrintStr(ELParser.PrintStrContext ctx) {
        return visit(ctx.strexpr()) + "debug ";
    }

    @Override
    public String visitPrintBool(ELParser.PrintBoolContext ctx) {
        return visit(ctx.bexpr()) + "debug ";
    }

    @Override
    public String visitPrintInt(ELParser.PrintIntContext ctx) {
        return visit(ctx.iexpr()) + "debug ";
    }

    @Override
    public String visitPrintFloat(ELParser.PrintFloatContext ctx) {
        return visit(ctx.fexpr()) + "debug ";
    }

    @Override
    public String visitPrintEntity(ELParser.PrintEntityContext ctx) {
        return visit(ctx.eexpr()) + "debug ";
    }

    @Override
    public String visitPrintDate(ELParser.PrintDateContext ctx) {
        return visit(ctx.dexpr()) + "debug ";
    }

    @Override
    public String visitPrintArray(ELParser.PrintArrayContext ctx) {
        return visit(ctx.arrayExpr()) + "debug ";
    }

    // ========================================================================
    // Perform statements
    // ========================================================================

    @Override
    public String visitPerformDT(ELParser.PerformDTContext ctx) {
        return resolveDecisionTable(ctx.typedDecisionTable());
    }

    @Override
    public String visitPerformDTExplicit(ELParser.PerformDTExplicitContext ctx) {
        return resolveDecisionTable(ctx.typedDecisionTable());
    }

    @Override
    public String visitPerformName(ELParser.PerformNameContext ctx) {
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
    public String visitSetInt(ELParser.SetIntContext ctx) {
        String v = resolveIdentLeft(ctx.leftIexpr().getText());
        String e = visit(ctx.number());
        return e + "cvi " + v;
    }

    @Override
    public String visitSetFloat(ELParser.SetFloatContext ctx) {
        String v = resolveIdentLeft(ctx.leftFexpr().getText());
        String e = visit(ctx.number());
        return e + "cvr " + v;
    }

    @Override
    public String visitSetBool(ELParser.SetBoolContext ctx) {
        String v = resolveIdentLeft(ctx.leftBexpr().getText());
        String e = visit(ctx.bexpr());
        return e + "cvb " + v;
    }

    @Override
    public String visitSetEntity(ELParser.SetEntityContext ctx) {
        String v = resolveIdentLeft(ctx.leftEexpr().getText());
        String e = visit(ctx.eexpr());
        return e + "cve " + v;
    }

    @Override
    public String visitSetString(ELParser.SetStringContext ctx) {
        String v = resolveIdentLeft(ctx.leftStrexpr().getText());
        String e = visit(ctx.strexpr());
        return e + "cvs " + v;
    }

    @Override
    public String visitSetDate(ELParser.SetDateContext ctx) {
        String v = resolveIdentLeft(ctx.leftDexpr().getText());
        String e = visit(ctx.dexpr());
        return e + "cvd " + v;
    }

    @Override
    public String visitIncrementLong(ELParser.IncrementLongContext ctx) {
        String v = resolveLong(ctx.typedLong());
        String left = resolveIdentLeft(ctx.typedLong().getText());
        return v + " 1 ladd " + left;
    }

    @Override
    public String visitIncrementDouble(ELParser.IncrementDoubleContext ctx) {
        String v = resolveDouble(ctx.typedDouble());
        String left = resolveIdentLeft(ctx.typedDouble().getText());
        return v + " 1 dadd " + left;
    }

    @Override
    public String visitDecrementLong(ELParser.DecrementLongContext ctx) {
        String v = resolveLong(ctx.typedLong());
        String left = resolveIdentLeft(ctx.typedLong().getText());
        return v + " 1 lsub " + left;
    }

    @Override
    public String visitDecrementDouble(ELParser.DecrementDoubleContext ctx) {
        String v = resolveDouble(ctx.typedDouble());
        String left = resolveIdentLeft(ctx.typedDouble().getText());
        return v + " 1 dsub " + left;
    }

    // ========================================================================
    // If statements
    // ========================================================================

    @Override
    public String visitIfThen(ELParser.IfThenContext ctx) {
        String b = visit(ctx.bexpr());
        String blk = visit(ctx.block());
        return "{ " + blk + "} " + b + "if ";
    }

    @Override
    public String visitIfThenElse(ELParser.IfThenElseContext ctx) {
        String b = visit(ctx.bexpr());
        String blk1 = visit(ctx.block(0));
        String blk2 = visit(ctx.block(1));
        return "{ " + blk1 + "} { " + blk2 + "} " + b + "ifelse ";
    }

    @Override
    public String visitIfblock(ELParser.IfblockContext ctx) {
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
    public String visitIfEnd(ELParser.IfEndContext ctx) {
        return "";
    }

    @Override
    public String visitIfElse(ELParser.IfElseContext ctx) {
        return visit(ctx.statementList());
    }

    @Override
    public String visitIfElseIf(ELParser.IfElseIfContext ctx) {
        return visit(ctx.ifblock());
    }

    // ========================================================================
    // Possessive and colon references
    // ========================================================================

    @Override
    public String visitPossessiveChain(ELParser.PossessiveChainContext ctx) {
        try {
            ELTypeResolver.ResolvedIdentifier resolved = typeResolver.resolvePossessive(ctx.POSSESSIVE().getText());
            String e2 = visit(ctx.possessiveRef());
            return resolved.value + " entitypush " + e2 + "entitypop ";
        } catch (RulesException e) {
            throw new RuntimeException(e);
        }
    }

    @Override
    public String visitPossessiveSingle(ELParser.PossessiveSingleContext ctx) {
        try {
            ELTypeResolver.ResolvedIdentifier resolved = typeResolver.resolvePossessive(ctx.POSSESSIVE().getText());
            return resolved.value + " ";
        } catch (RulesException e) {
            throw new RuntimeException(e);
        }
    }

    @Override
    public String visitColonPossessiveChain(ELParser.ColonPossessiveChainContext ctx) {
        String e1 = resolveEntity(ctx.typedEntity());
        String e2 = visit(ctx.possessiveRef());
        return e1 + " entitypush " + e2 + "entitypop ";
    }

    @Override
    public String visitColonPossessiveSingle(ELParser.ColonPossessiveSingleContext ctx) {
        return resolveEntity(ctx.typedEntity()) + " ";
    }
}
