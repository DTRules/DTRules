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
package com.dtrules.compiler.ebl;

import org.antlr.v4.runtime.tree.ParseTree;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;

/**
 * ANTLR 4 Visitor that generates postfix code from the EL parse tree.
 * This replaces the semantic actions from the CUP parser.
 */
public class EBLCompilerVisitor extends EBLBaseVisitor<String> {

    private final EBLTypeResolver typeResolver;
    private int localCnt = 0;

    public EBLCompilerVisitor(EBLTypeResolver typeResolver) {
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
    public String visitEmptyAction(EBLParser.EmptyActionContext ctx) {
        return "";
    }

    @Override
    public String visitEmptyCondition(EBLParser.EmptyConditionContext ctx) {
        return "";
    }

    @Override
    public String visitEmptyContextEbl(EBLParser.EmptyContextEblContext ctx) {
        // EBL returns "" for empty context, unlike EL which returns "execute "
        return "";
    }

    @Override
    public String visitEmptyPolicyStatement(EBLParser.EmptyPolicyStatementContext ctx) {
        return "";
    }

    @Override
    public String visitActionStatement(EBLParser.ActionStatementContext ctx) {
        return "\n" + visit(ctx.statementList()) + "\n";
    }

    @Override
    public String visitConditionExpr(EBLParser.ConditionExprContext ctx) {
        return "\n" + visit(ctx.bexpr()) + "\n";
    }

    @Override
    public String visitConditionDebugBefore(EBLParser.ConditionDebugBeforeContext ctx) {
        return "\n" + visit(ctx.debugstatement()) + visit(ctx.bexpr()) + "\n";
    }

    @Override
    public String visitConditionDebugAfter(EBLParser.ConditionDebugAfterContext ctx) {
        return "\n" + visit(ctx.bexpr()) + visit(ctx.debugstatement()) + "\n";
    }

    @Override
    public String visitContextStatement(EBLParser.ContextStatementContext ctx) {
        return "\n" + visit(ctx.contextForTable()) + "\n";
    }

    @Override
    public String visitContextDebugBefore(EBLParser.ContextDebugBeforeContext ctx) {
        return "\n" + visit(ctx.debugstatement()) + visit(ctx.contextForTable()) + "\n";
    }

    @Override
    public String visitContextFindClause(EBLParser.ContextFindClauseContext ctx) {
        return "\n" + visit(ctx.findclause()) + "\n";
    }

    @Override
    public String visitFindclause(EBLParser.FindclauseContext ctx) {
        String entity = ctx.typedEntity().getText();
        String array = visit(ctx.arrayExpr());
        String context = visit(ctx.contextForTable());
        return "{ " + context + "} { 0 entityfetch /" + entity + " get " + entity + " req } " + array + "forfirst ";
    }

    // EBL-specific: iexpr ISWITHIN arrayExpr
    @Override
    public String visitBoolIsWithin(EBLParser.BoolIsWithinContext ctx) {
        String i = visit(ctx.iexpr());
        String array = visit(ctx.arrayExpr());
        return " { true } { false } { " + i + "dup begin_page >= swap end_page <= && } " + array + "forfirstelse ";
    }

    @Override
    public String visitPolicyStrExpr(EBLParser.PolicyStrExprContext ctx) {
        return visit(ctx.strexpr());
    }

    @Override
    public String visitPolicyNExpr(EBLParser.PolicyNExprContext ctx) {
        return visit(ctx.nexpr());
    }

    @Override
    public String visitPolicyIExpr(EBLParser.PolicyIExprContext ctx) {
        return visit(ctx.iexpr());
    }

    @Override
    public String visitPolicyFExpr(EBLParser.PolicyFExprContext ctx) {
        return visit(ctx.fexpr());
    }

    @Override
    public String visitPolicyBExpr(EBLParser.PolicyBExprContext ctx) {
        return visit(ctx.bexpr());
    }

    @Override
    public String visitPolicyDExpr(EBLParser.PolicyDExprContext ctx) {
        return visit(ctx.dexpr());
    }

    // ========================================================================
    // Statement list and blocks
    // ========================================================================

    @Override
    public String visitStatementList(EBLParser.StatementListContext ctx) {
        StringBuilder sb = new StringBuilder();
        for (EBLParser.BlockContext block : ctx.block()) {
            sb.append(visit(block));
        }
        return sb.toString();
    }

    @Override
    public String visitBlockCurly(EBLParser.BlockCurlyContext ctx) {
        return visit(ctx.statementList()) + "\n";
    }

    @Override
    public String visitBlockUsing(EBLParser.BlockUsingContext ctx) {
        return visit(ctx.usingblock());
    }

    @Override
    public String visitBlockGforall(EBLParser.BlockGforallContext ctx) {
        String blk = visit(ctx.statementList());
        String ctl = visit(ctx.forallctl());
        return "{ " + blk + "} " + ctl + "pop ";
    }

    @Override
    public String visitBlockForall(EBLParser.BlockForallContext ctx) {
        return visit(ctx.forallblock());
    }

    @Override
    public String visitBlockForeach(EBLParser.BlockForeachContext ctx) {
        return visit(ctx.foreachblock());
    }

    @Override
    public String visitBlockFirst(EBLParser.BlockFirstContext ctx) {
        return visit(ctx.firstblock());
    }

    @Override
    public String visitBlockIf(EBLParser.BlockIfContext ctx) {
        return visit(ctx.ifblock());
    }

    @Override
    public String visitBlockStatement(EBLParser.BlockStatementContext ctx) {
        return visit(ctx.statement());
    }

    // ========================================================================
    // Using blocks
    // ========================================================================

    @Override
    public String visitUsingBlockEntity(EBLParser.UsingBlockEntityContext ctx) {
        String ee = resolveEntity(ctx.typedEntity());
        String e = visit(ctx.usingblock());
        return ee + " entitypush " + e + "entitypop ";
    }

    @Override
    public String visitUsingBlockEntityComma(EBLParser.UsingBlockEntityCommaContext ctx) {
        String ee = resolveEntity(ctx.typedEntity());
        String e = visit(ctx.usingblock());
        return ee + " entitypush " + e + "entitypop ";
    }

    @Override
    public String visitUsingBlockBase(EBLParser.UsingBlockBaseContext ctx) {
        return visit(ctx.block());
    }

    // ========================================================================
    // Context for table
    // ========================================================================

    @Override
    public String visitContextDebug(EBLParser.ContextDebugContext ctx) {
        return visit(ctx.debugstatement()) + "execute ";
    }

    @Override
    public String visitContextFor(EBLParser.ContextForContext ctx) {
        return visit(ctx.forctl()) + "pop ";
    }

    @Override
    public String visitContextForallCtl(EBLParser.ContextForallCtlContext ctx) {
        return visit(ctx.forallctl()) + "pop ";
    }

    @Override
    public String visitContextForfirst(EBLParser.ContextForfirstContext ctx) {
        return visit(ctx.forfirstctl()) + "pop ";
    }

    @Override
    public String visitContextCtx(EBLParser.ContextCtxContext ctx) {
        return visit(ctx.contextstatement()) + "execute entitypop ";
    }

    @Override
    public String visitContextLocal(EBLParser.ContextLocalContext ctx) {
        return visit(ctx.localvariables());
    }

    // ========================================================================
    // Helper methods for type resolution
    // ========================================================================

    private String resolveEntity(EBLParser.TypedEntityContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveLong(EBLParser.TypedLongContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveDouble(EBLParser.TypedDoubleContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveString(EBLParser.TypedStringContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveBoolean(EBLParser.TypedBooleanContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveDate(EBLParser.TypedDateContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveArray(EBLParser.TypedArrayContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveTable(EBLParser.TypedTableContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveName(EBLParser.TypedNameContext ctx) {
        return resolveIdent(ctx.IDENT().getText());
    }

    private String resolveDecisionTable(EBLParser.TypedDecisionTableContext ctx) {
        return "/" + ctx.IDENT().getText() + " execute ";
    }

    private String resolveIdent(String ident) {
        try {
            EBLTypeResolver.ResolvedIdentifier resolved = typeResolver.resolve(ident);
            return resolved.value;
        } catch (RulesException e) {
            throw new RuntimeException("Failed to resolve identifier: " + ident, e);
        }
    }

    private String resolveIdentLeft(String ident) {
        try {
            EBLTypeResolver.ResolvedIdentifier resolved = typeResolver.resolve(ident);
            return resolved.leftValue;
        } catch (RulesException e) {
            throw new RuntimeException("Failed to resolve identifier: " + ident, e);
        }
    }

    // ========================================================================
    // Integer expressions
    // ========================================================================

    @Override
    public String visitIntAdd(EBLParser.IntAddContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "+ ";
    }

    @Override
    public String visitIntSub(EBLParser.IntSubContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "- ";
    }

    @Override
    public String visitIntMul(EBLParser.IntMulContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "* ";
    }

    @Override
    public String visitIntDiv(EBLParser.IntDivContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "div ";
    }

    @Override
    public String visitIntLiteral(EBLParser.IntLiteralContext ctx) {
        return ctx.INT_LITERAL().getText() + " ";
    }

    @Override
    public String visitIntNegate(EBLParser.IntNegateContext ctx) {
        return visit(ctx.iexpr()) + "negate ";
    }

    @Override
    public String visitIntParen(EBLParser.IntParenContext ctx) {
        return visit(ctx.iexpr()) + " ";
    }

    @Override
    public String visitIntTyped(EBLParser.IntTypedContext ctx) {
        return resolveLong(ctx.typedLong()) + " ";
    }

    @Override
    public String visitIntColonRef(EBLParser.IntColonRefContext ctx) {
        String ref = visit(ctx.colonRef());
        String n = resolveLong(ctx.typedLong());
        return ref + "entitypush " + n + " entitypop ";
    }

    @Override
    public String visitIntNumberOf(EBLParser.IntNumberOfContext ctx) {
        return visit(ctx.arrayExpr()) + "length ";
    }

    @Override
    public String visitIntNumberOfWhere(EBLParser.IntNumberOfWhereContext ctx) {
        String a = visit(ctx.arrayExpr());
        String b = visit(ctx.bexpr());
        return "0 { { 1 ladd } " + b + "if }" + a + "forall ";
    }

    @Override
    public String visitIntLengthArray(EBLParser.IntLengthArrayContext ctx) {
        return visit(ctx.arrayExpr()) + "length ";
    }

    @Override
    public String visitIntLengthStr(EBLParser.IntLengthStrContext ctx) {
        return visit(ctx.strexpr()) + "strlength ";
    }

    @Override
    public String visitIntIndexOf(EBLParser.IntIndexOfContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "indexof ";
    }

    @Override
    public String visitIntDaysBetween(EBLParser.IntDaysBetweenContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "daysbetween ";
    }

    @Override
    public String visitIntMonthsBetween(EBLParser.IntMonthsBetweenContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "monthsbetween ";
    }

    @Override
    public String visitIntYearsBetween(EBLParser.IntYearsBetweenContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "yearsbetween ";
    }

    @Override
    public String visitIntYearOf(EBLParser.IntYearOfContext ctx) {
        return visit(ctx.dexpr()) + "yearof ";
    }

    @Override
    public String visitIntAbs(EBLParser.IntAbsContext ctx) {
        return visit(ctx.iexpr()) + "labs ";
    }

    @Override
    public String visitIntUsing(EBLParser.IntUsingContext ctx) {
        String ee = visit(ctx.eexpr());
        String e = visit(ctx.iexpr());
        return ee + "entitypush " + e + "entitypop ";
    }

    // ========================================================================
    // Float expressions
    // ========================================================================

    @Override
    public String visitFloatLiteral(EBLParser.FloatLiteralContext ctx) {
        return ctx.FLOAT_LITERAL().getText() + " ";
    }

    @Override
    public String visitFloatTyped(EBLParser.FloatTypedContext ctx) {
        return resolveDouble(ctx.typedDouble()) + " ";
    }

    @Override
    public String visitFloatColonRef(EBLParser.FloatColonRefContext ctx) {
        String ref = visit(ctx.colonRef());
        String f = resolveDouble(ctx.typedDouble());
        return ref + "entitypush " + f + " entitypop ";
    }

    @Override
    public String visitFloatAddInt(EBLParser.FloatAddIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "fadd ";
    }

    @Override
    public String visitFloatAddFloat(EBLParser.FloatAddFloatContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "fadd ";
    }

    @Override
    public String visitIntAddFloat(EBLParser.IntAddFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "fadd ";
    }

    @Override
    public String visitFloatSubInt(EBLParser.FloatSubIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "fsub ";
    }

    @Override
    public String visitIntSubFloat(EBLParser.IntSubFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "fsub ";
    }

    @Override
    public String visitFloatSubFloat(EBLParser.FloatSubFloatContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "fsub ";
    }

    @Override
    public String visitFloatMulInt(EBLParser.FloatMulIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "fmul ";
    }

    @Override
    public String visitIntMulFloat(EBLParser.IntMulFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "fmul ";
    }

    @Override
    public String visitFloatMulFloat(EBLParser.FloatMulFloatContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "fmul ";
    }

    @Override
    public String visitFloatDivInt(EBLParser.FloatDivIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "fdiv ";
    }

    @Override
    public String visitIntDivFloat(EBLParser.IntDivFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "fdiv ";
    }

    @Override
    public String visitFloatDivFloat(EBLParser.FloatDivFloatContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "fdiv ";
    }

    @Override
    public String visitFloatNegate(EBLParser.FloatNegateContext ctx) {
        return visit(ctx.fexpr()) + "fnegate ";
    }

    @Override
    public String visitFloatParen(EBLParser.FloatParenContext ctx) {
        return visit(ctx.fexpr());
    }

    @Override
    public String visitFloatAbs(EBLParser.FloatAbsContext ctx) {
        return visit(ctx.fexpr()) + "fabs ";
    }

    @Override
    public String visitFloatUsing(EBLParser.FloatUsingContext ctx) {
        String ee = visit(ctx.eexpr());
        String e = visit(ctx.fexpr());
        return ee + "entitypush " + e + "entitypop ";
    }

    @Override
    public String visitFloatRounded(EBLParser.FloatRoundedContext ctx) {
        return visit(ctx.fexpr()) + "0 0.5 roundto ";
    }

    @Override
    public String visitFloatRoundedTo(EBLParser.FloatRoundedToContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "0.5 roundto ";
    }

    @Override
    public String visitFloatRoundedBoundry(EBLParser.FloatRoundedBoundryContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.iexpr()) + visit(ctx.fexpr(1)) + " roundto ";
    }

    // ========================================================================
    // String expressions
    // ========================================================================

    @Override
    public String visitStrLiteral(EBLParser.StrLiteralContext ctx) {
        return ctx.STRING_LITERAL().getText() + " ";
    }

    @Override
    public String visitStrTyped(EBLParser.StrTypedContext ctx) {
        return resolveString(ctx.typedString()) + " ";
    }

    @Override
    public String visitStrColonRef(EBLParser.StrColonRefContext ctx) {
        String ref = visit(ctx.colonRef());
        String s = visit(ctx.strexpr());
        return ref + "entitypush " + s + "entitypop ";
    }

    @Override
    public String visitStrConcat(EBLParser.StrConcatContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "strconcat ";
    }

    @Override
    public String visitStrConcatInt(EBLParser.StrConcatIntContext ctx) {
        return visit(ctx.strexpr()) + visit(ctx.iexpr()) + "strconcat ";
    }

    @Override
    public String visitStrConcatFloat(EBLParser.StrConcatFloatContext ctx) {
        return visit(ctx.strexpr()) + visit(ctx.fexpr()) + "strconcat ";
    }

    @Override
    public String visitStrConcatName(EBLParser.StrConcatNameContext ctx) {
        return visit(ctx.strexpr()) + visit(ctx.nexpr()) + "strconcat ";
    }

    @Override
    public String visitStrConcatEntity(EBLParser.StrConcatEntityContext ctx) {
        return visit(ctx.strexpr()) + visit(ctx.eexpr()) + "strconcat ";
    }

    @Override
    public String visitStrConcatDate(EBLParser.StrConcatDateContext ctx) {
        return visit(ctx.strexpr()) + visit(ctx.dexpr()) + "strconcat ";
    }

    @Override
    public String visitStrConcatArray(EBLParser.StrConcatArrayContext ctx) {
        return visit(ctx.strexpr()) + visit(ctx.arrayExpr()) + "strconcat ";
    }

    @Override
    public String visitStrTrim(EBLParser.StrTrimContext ctx) {
        return visit(ctx.strexpr()) + "strtrim ";
    }

    @Override
    public String visitStrToLower(EBLParser.StrToLowerContext ctx) {
        return visit(ctx.strexpr()) + "tolowercase ";
    }

    @Override
    public String visitStrToUpper(EBLParser.StrToUpperContext ctx) {
        return visit(ctx.strexpr()) + "touppercase ";
    }

    @Override
    public String visitStrTimestamp(EBLParser.StrTimestampContext ctx) {
        return "getdate gettimestamp ";
    }

    @Override
    public String visitStrUsing(EBLParser.StrUsingContext ctx) {
        String ee = visit(ctx.eexpr());
        String e = visit(ctx.strexpr());
        return ee + "entitypush " + e + "entitypop ";
    }

    @Override
    public String visitStrParen(EBLParser.StrParenContext ctx) {
        return visit(ctx.strexpr());
    }

    @Override
    public String visitStrSubstring(EBLParser.StrSubstringContext ctx) {
        String s = visit(ctx.strexpr());
        String start = visit(ctx.iexpr(0));
        String end = visit(ctx.iexpr(1));
        return end + start + s + "substring ";
    }

    @Override
    public String visitStrValueOfFloat(EBLParser.StrValueOfFloatContext ctx) {
        return visit(ctx.fexpr()) + "tostring ";
    }

    @Override
    public String visitStrValueOfInt(EBLParser.StrValueOfIntContext ctx) {
        return visit(ctx.iexpr()) + "tostring ";
    }

    @Override
    public String visitStrValueOfDate(EBLParser.StrValueOfDateContext ctx) {
        return visit(ctx.dexpr()) + "tostring ";
    }

    @Override
    public String visitStrValueOfBool(EBLParser.StrValueOfBoolContext ctx) {
        return visit(ctx.bexpr()) + "tostring ";
    }

    @Override
    public String visitStrTableInfo(EBLParser.StrTableInfoContext ctx) {
        return "actionstring ";
    }

    @Override
    public String visitStrMappingKey(EBLParser.StrMappingKeyContext ctx) {
        return "\"mapping*key\" cvn execute ";
    }

    // ========================================================================
    // Boolean expressions
    // ========================================================================

    @Override
    public String visitBoolTyped(EBLParser.BoolTypedContext ctx) {
        return resolveBoolean(ctx.typedBoolean()) + " ";
    }

    @Override
    public String visitBoolLiteral(EBLParser.BoolLiteralContext ctx) {
        // Boolean literals: true, false, default, otherwise, always, perform when called
        return ctx.RBOOLEAN().getText().toLowerCase() + " ";
    }

    @Override
    public String visitBoolColonRef(EBLParser.BoolColonRefContext ctx) {
        String ref = visit(ctx.colonRef());
        String b = resolveBoolean(ctx.typedBoolean());
        return ref + "entitypush " + b + " entitypop ";
    }

    @Override
    public String visitBoolAnd(EBLParser.BoolAndContext ctx) {
        String e1 = visit(ctx.bexpr(0));
        String e2 = visit(ctx.bexpr(1));
        return e1 + "{ pop " + e2 + "} over if\n";
    }

    @Override
    public String visitBoolOr(EBLParser.BoolOrContext ctx) {
        String e1 = visit(ctx.bexpr(0));
        String e2 = visit(ctx.bexpr(1));
        return e1 + "{ pop " + e2 + "} over not if\n";
    }

    @Override
    public String visitBoolNot(EBLParser.BoolNotContext ctx) {
        return visit(ctx.bexpr()) + "not ";
    }

    @Override
    public String visitBoolParen(EBLParser.BoolParenContext ctx) {
        return visit(ctx.bexpr());
    }

    @Override
    public String visitBoolUsing(EBLParser.BoolUsingContext ctx) {
        String ee = visit(ctx.eexpr());
        String e = visit(ctx.bexpr());
        return ee + "entitypush " + e + "entitypop ";
    }

    // Integer comparisons
    @Override
    public String visitBoolIntEq(EBLParser.BoolIntEqContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "== ";
    }

    @Override
    public String visitBoolIntNeq(EBLParser.BoolIntNeqContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "== not ";
    }

    @Override
    public String visitBoolIntGt(EBLParser.BoolIntGtContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "> ";
    }

    @Override
    public String visitBoolIntGte(EBLParser.BoolIntGteContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + ">= ";
    }

    @Override
    public String visitBoolIntLt(EBLParser.BoolIntLtContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "< ";
    }

    @Override
    public String visitBoolIntLte(EBLParser.BoolIntLteContext ctx) {
        return visit(ctx.iexpr(0)) + visit(ctx.iexpr(1)) + "<= ";
    }

    // Float comparisons
    @Override
    public String visitBoolFloatEq(EBLParser.BoolFloatEqContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "f== ";
    }

    @Override
    public String visitBoolFloatNeq(EBLParser.BoolFloatNeqContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "f== not ";
    }

    @Override
    public String visitBoolFloatGt(EBLParser.BoolFloatGtContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "f> ";
    }

    @Override
    public String visitBoolFloatGte(EBLParser.BoolFloatGteContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "f>= ";
    }

    @Override
    public String visitBoolFloatLt(EBLParser.BoolFloatLtContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "f< ";
    }

    @Override
    public String visitBoolFloatLte(EBLParser.BoolFloatLteContext ctx) {
        return visit(ctx.fexpr(0)) + visit(ctx.fexpr(1)) + "f<= ";
    }

    // Mixed comparisons
    @Override
    public String visitBoolFloatEqInt(EBLParser.BoolFloatEqIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "f== ";
    }

    @Override
    public String visitBoolIntEqFloat(EBLParser.BoolIntEqFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "f== ";
    }

    @Override
    public String visitBoolFloatNeqInt(EBLParser.BoolFloatNeqIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "f== not ";
    }

    @Override
    public String visitBoolIntNeqFloat(EBLParser.BoolIntNeqFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "f== not ";
    }

    @Override
    public String visitBoolFloatGtInt(EBLParser.BoolFloatGtIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "f> ";
    }

    @Override
    public String visitBoolIntGtFloat(EBLParser.BoolIntGtFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "f> ";
    }

    @Override
    public String visitBoolFloatGteInt(EBLParser.BoolFloatGteIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "f>= ";
    }

    @Override
    public String visitBoolIntGteFloat(EBLParser.BoolIntGteFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "f>= ";
    }

    @Override
    public String visitBoolFloatLtInt(EBLParser.BoolFloatLtIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "f< ";
    }

    @Override
    public String visitBoolIntLtFloat(EBLParser.BoolIntLtFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "f< ";
    }

    @Override
    public String visitBoolFloatLteInt(EBLParser.BoolFloatLteIntContext ctx) {
        return visit(ctx.fexpr()) + visit(ctx.iexpr()) + "f<= ";
    }

    @Override
    public String visitBoolIntLteFloat(EBLParser.BoolIntLteFloatContext ctx) {
        return visit(ctx.iexpr()) + visit(ctx.fexpr()) + "f<= ";
    }

    // String comparisons
    @Override
    public String visitBoolStrEq(EBLParser.BoolStrEqContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "streq ";
    }

    @Override
    public String visitBoolStrNeq(EBLParser.BoolStrNeqContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "streq not ";
    }

    @Override
    public String visitBoolStrEqIc(EBLParser.BoolStrEqIcContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "sic== ";
    }

    @Override
    public String visitBoolStrNeqIc(EBLParser.BoolStrNeqIcContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "sic== not ";
    }

    @Override
    public String visitBoolStrGt(EBLParser.BoolStrGtContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "s> ";
    }

    @Override
    public String visitBoolStrLt(EBLParser.BoolStrLtContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "s< ";
    }

    @Override
    public String visitBoolStrGte(EBLParser.BoolStrGteContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "s>= ";
    }

    @Override
    public String visitBoolStrLte(EBLParser.BoolStrLteContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "s<= ";
    }

    @Override
    public String visitBoolStartsWith(EBLParser.BoolStartsWithContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "0 startswith ";
    }

    @Override
    public String visitBoolStartsWithAt(EBLParser.BoolStartsWithAtContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + visit(ctx.iexpr()) + "startswith ";
    }

    @Override
    public String visitBoolMatches(EBLParser.BoolMatchesContext ctx) {
        return visit(ctx.strexpr(1)) + visit(ctx.strexpr(0)) + "regexmatch ";
    }

    // Date comparisons
    @Override
    public String visitBoolDateEq(EBLParser.BoolDateEqContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d== ";
    }

    @Override
    public String visitBoolDateLt(EBLParser.BoolDateLtContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d< ";
    }

    @Override
    public String visitBoolDateGt(EBLParser.BoolDateGtContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d> ";
    }

    @Override
    public String visitBoolDateGte(EBLParser.BoolDateGteContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d< not ";
    }

    @Override
    public String visitBoolDateLte(EBLParser.BoolDateLteContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d> not ";
    }

    @Override
    public String visitBoolDateBefore(EBLParser.BoolDateBeforeContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d< ";
    }

    @Override
    public String visitBoolDateAfter(EBLParser.BoolDateAfterContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d> ";
    }

    @Override
    public String visitBoolDateBetween(EBLParser.BoolDateBetweenContext ctx) {
        String d1 = visit(ctx.dexpr(0));
        String d2 = visit(ctx.dexpr(1));
        String d3 = visit(ctx.dexpr(2));
        return d2 + d1 + "d> not " + d1 + d3 + " d> not and ";
    }

    // Entity comparisons
    @Override
    public String visitBoolEntityEq(EBLParser.BoolEntityEqContext ctx) {
        return visit(ctx.eexpr(0)) + visit(ctx.eexpr(1)) + "req ";
    }

    @Override
    public String visitBoolEntityNeq(EBLParser.BoolEntityNeqContext ctx) {
        return visit(ctx.eexpr(0)) + visit(ctx.eexpr(1)) + "req not ";
    }

    // Null tests
    @Override
    public String visitBoolBexprIsNull(EBLParser.BoolBexprIsNullContext ctx) {
        return visit(ctx.bexpr()) + " isnull ";
    }

    @Override
    public String visitBoolBexprIsNotNull(EBLParser.BoolBexprIsNotNullContext ctx) {
        return visit(ctx.bexpr()) + " isnull not ";
    }

    @Override
    public String visitBoolNumIsNull(EBLParser.BoolNumIsNullContext ctx) {
        return visit(ctx.number()) + " isnull ";
    }

    @Override
    public String visitBoolNumIsNotNull(EBLParser.BoolNumIsNotNullContext ctx) {
        return visit(ctx.number()) + " isnull not ";
    }

    @Override
    public String visitBoolDateIsNull(EBLParser.BoolDateIsNullContext ctx) {
        return visit(ctx.dexpr()) + " isnull ";
    }

    @Override
    public String visitBoolDateIsNotNull(EBLParser.BoolDateIsNotNullContext ctx) {
        return visit(ctx.dexpr()) + " isnull not ";
    }

    @Override
    public String visitBoolArrayIsNull(EBLParser.BoolArrayIsNullContext ctx) {
        return visit(ctx.arrayExpr()) + " isnull ";
    }

    @Override
    public String visitBoolArrayIsNotNull(EBLParser.BoolArrayIsNotNullContext ctx) {
        return visit(ctx.arrayExpr()) + " isnull not ";
    }

    @Override
    public String visitBoolStrIsNull(EBLParser.BoolStrIsNullContext ctx) {
        return visit(ctx.strexpr()) + " isnull ";
    }

    @Override
    public String visitBoolStrIsNotNull(EBLParser.BoolStrIsNotNullContext ctx) {
        return visit(ctx.strexpr()) + " isnull not ";
    }

    @Override
    public String visitBoolEntityIsNull(EBLParser.BoolEntityIsNullContext ctx) {
        return visit(ctx.eexpr()) + " isnull ";
    }

    @Override
    public String visitBoolEntityIsNotNull(EBLParser.BoolEntityIsNotNullContext ctx) {
        return visit(ctx.eexpr()) + " isnull not ";
    }

    // ========================================================================
    // Entity expressions
    // ========================================================================

    @Override
    public String visitEntityTyped(EBLParser.EntityTypedContext ctx) {
        return resolveEntity(ctx.typedEntity()) + " ";
    }

    @Override
    public String visitEntityParen(EBLParser.EntityParenContext ctx) {
        return visit(ctx.eexpr());
    }

    @Override
    public String visitEntityIndex(EBLParser.EntityIndexContext ctx) {
        return visit(ctx.indxExpr());
    }

    @Override
    public String visitEntityNewName(EBLParser.EntityNewNameContext ctx) {
        return visit(ctx.nexpr()) + " createentity ";
    }

    @Override
    public String visitEntityNewTyped(EBLParser.EntityNewTypedContext ctx) {
        return "/" + resolveEntity(ctx.typedEntity()) + " createentity ";
    }

    @Override
    public String visitEntityClone(EBLParser.EntityCloneContext ctx) {
        return visit(ctx.eexpr()) + "clone ";
    }

    @Override
    public String visitEntityColonRef(EBLParser.EntityColonRefContext ctx) {
        String ref = visit(ctx.colonRef());
        String e = resolveEntity(ctx.typedEntity());
        return ref + "entitypush " + e + " entitypop ";
    }

    // ========================================================================
    // Date expressions
    // ========================================================================

    @Override
    public String visitDateTyped(EBLParser.DateTypedContext ctx) {
        return resolveDate(ctx.typedDate()) + " ";
    }

    @Override
    public String visitDateParen(EBLParser.DateParenContext ctx) {
        return visit(ctx.dexpr());
    }

    @Override
    public String visitDateColonRef(EBLParser.DateColonRefContext ctx) {
        String ref = visit(ctx.colonRef());
        String d = resolveDate(ctx.typedDate());
        return ref + "entitypush " + d + " entitypop ";
    }

    @Override
    public String visitDateFromStrCast(EBLParser.DateFromStrCastContext ctx) {
        return visit(ctx.strexpr()) + "cvd ";
    }

    @Override
    public String visitDateFromStrFunc(EBLParser.DateFromStrFuncContext ctx) {
        return visit(ctx.strexpr()) + "cvd ";
    }

    @Override
    public String visitDateCurrentDate(EBLParser.DateCurrentDateContext ctx) {
        return "getdate ";
    }

    @Override
    public String visitDateAdd(EBLParser.DateAddContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d+ ";
    }

    @Override
    public String visitDateSub(EBLParser.DateSubContext ctx) {
        return visit(ctx.dexpr(0)) + visit(ctx.dexpr(1)) + "d- ";
    }

    @Override
    public String visitDateDays(EBLParser.DateDaysContext ctx) {
        return visit(ctx.number()) + "days ";
    }

    @Override
    public String visitDateFirstOfYear(EBLParser.DateFirstOfYearContext ctx) {
        return visit(ctx.dexpr()) + "firstofyear ";
    }

    @Override
    public String visitDateFirstOfMonth(EBLParser.DateFirstOfMonthContext ctx) {
        return visit(ctx.dexpr()) + "firstofmonth ";
    }

    @Override
    public String visitDateEndOfMonth(EBLParser.DateEndOfMonthContext ctx) {
        return visit(ctx.dexpr()) + "endofmonth ";
    }

    @Override
    public String visitDateUsing(EBLParser.DateUsingContext ctx) {
        String ee = visit(ctx.eexpr());
        String e = visit(ctx.dexpr());
        return ee + "entitypush " + e + "entitypop ";
    }

    // ========================================================================
    // Array expressions
    // ========================================================================

    @Override
    public String visitArrayPolicyStatements(EBLParser.ArrayPolicyStatementsContext ctx) {
        return "policystatements ";
    }

    @Override
    public String visitArrayColonRef(EBLParser.ArrayColonRefContext ctx) {
        String ref = visit(ctx.colonRef());
        String a = resolveArray(ctx.typedArray());
        return ref + "entitypush " + a + " entitypop ";
    }

    @Override
    public String visitArrayBase(EBLParser.ArrayBaseContext ctx) {
        return visit(ctx.arrayExpr2());
    }

    @Override
    public String visitArrayTyped(EBLParser.ArrayTypedContext ctx) {
        return resolveArray(ctx.typedArray()) + " ";
    }

    @Override
    public String visitArrayParen(EBLParser.ArrayParenContext ctx) {
        return visit(ctx.arrayExpr());
    }

    @Override
    public String visitArrayCopy(EBLParser.ArrayCopyContext ctx) {
        return visit(ctx.arrayExpr()) + "copyelements ";
    }

    @Override
    public String visitArrayCopySimple(EBLParser.ArrayCopySimpleContext ctx) {
        return visit(ctx.arrayExpr()) + "copyelements ";
    }

    @Override
    public String visitArrayDeepCopy(EBLParser.ArrayDeepCopyContext ctx) {
        return visit(ctx.arrayExpr()) + "deepcopy ";
    }

    @Override
    public String visitArrayDeepCopySimple(EBLParser.ArrayDeepCopySimpleContext ctx) {
        return visit(ctx.arrayExpr()) + "deepcopy ";
    }

    @Override
    public String visitArrayOfValues(EBLParser.ArrayOfValuesContext ctx) {
        return "mark " + visit(ctx.arrayList()) + " arraytomark ";
    }

    @Override
    public String visitArrayTokenize(EBLParser.ArrayTokenizeContext ctx) {
        return visit(ctx.strexpr(0)) + visit(ctx.strexpr(1)) + "tokenize ";
    }

    @Override
    public String visitArrayLiteral(EBLParser.ArrayLiteralContext ctx) {
        return visit(ctx.arrayLit());
    }

    @Override
    public String visitArrayLit(EBLParser.ArrayLitContext ctx) {
        return "{ " + visit(ctx.arrayList()) + "} ";
    }

    @Override
    public String visitIndxExpr(EBLParser.IndxExprContext ctx) {
        return visit(ctx.arrayExpr()) + visit(ctx.iexpr()) + "getat ";
    }

    // ========================================================================
    // Name expressions
    // ========================================================================

    @Override
    public String visitNameTyped(EBLParser.NameTypedContext ctx) {
        return resolveName(ctx.typedName()) + " ";
    }

    @Override
    public String visitNameOf(EBLParser.NameOfContext ctx) {
        return visit(ctx.eexpr()) + "getname ";
    }

    @Override
    public String visitNameTheName(EBLParser.NameTheNameContext ctx) {
        return visit(ctx.strexpr()) + "cvn ";
    }

    @Override
    public String visitNameLiteral(EBLParser.NameLiteralContext ctx) {
        String name = ctx.NAME().getText();
        if (name.startsWith("$")) {
            name = name.substring(1);
        }
        return "/" + name + " ";
    }

    @Override
    public String visitNameUsing(EBLParser.NameUsingContext ctx) {
        String ee = visit(ctx.eexpr());
        String e = visit(ctx.nexpr());
        return ee + "entitypush " + e + "entitypop ";
    }

    @Override
    public String visitNameColonRef(EBLParser.NameColonRefContext ctx) {
        String ref = visit(ctx.colonRef());
        String n = resolveName(ctx.typedName());
        return ref + "entitypush " + n + " entitypop ";
    }

    @Override
    public String visitNameFromStr(EBLParser.NameFromStrContext ctx) {
        return visit(ctx.strexpr()) + "cvn ";
    }

    // ========================================================================
    // Table expressions
    // ========================================================================

    @Override
    public String visitTableTyped(EBLParser.TableTypedContext ctx) {
        return resolveTable(ctx.typedTable()) + " ";
    }

    // ========================================================================
    // Debug statements
    // ========================================================================

    @Override
    public String visitDebugStr(EBLParser.DebugStrContext ctx) {
        return visit(ctx.strexpr()) + "debug ";
    }

    @Override
    public String visitDebugBool(EBLParser.DebugBoolContext ctx) {
        return visit(ctx.bexpr()) + "debug ";
    }

    @Override
    public String visitDebugInt(EBLParser.DebugIntContext ctx) {
        return visit(ctx.iexpr()) + "debug ";
    }

    @Override
    public String visitDebugFloat(EBLParser.DebugFloatContext ctx) {
        return visit(ctx.fexpr()) + "debug ";
    }

    @Override
    public String visitDebugEntity(EBLParser.DebugEntityContext ctx) {
        return visit(ctx.eexpr()) + "debug ";
    }

    @Override
    public String visitDebugDate(EBLParser.DebugDateContext ctx) {
        return visit(ctx.dexpr()) + "debug ";
    }

    @Override
    public String visitDebugArray(EBLParser.DebugArrayContext ctx) {
        return visit(ctx.arrayExpr()) + "debug ";
    }

    @Override
    public String visitPrintStr(EBLParser.PrintStrContext ctx) {
        return visit(ctx.strexpr()) + "debug ";
    }

    @Override
    public String visitPrintBool(EBLParser.PrintBoolContext ctx) {
        return visit(ctx.bexpr()) + "debug ";
    }

    @Override
    public String visitPrintInt(EBLParser.PrintIntContext ctx) {
        return visit(ctx.iexpr()) + "debug ";
    }

    @Override
    public String visitPrintFloat(EBLParser.PrintFloatContext ctx) {
        return visit(ctx.fexpr()) + "debug ";
    }

    @Override
    public String visitPrintEntity(EBLParser.PrintEntityContext ctx) {
        return visit(ctx.eexpr()) + "debug ";
    }

    @Override
    public String visitPrintDate(EBLParser.PrintDateContext ctx) {
        return visit(ctx.dexpr()) + "debug ";
    }

    @Override
    public String visitPrintArray(EBLParser.PrintArrayContext ctx) {
        return visit(ctx.arrayExpr()) + "debug ";
    }

    // ========================================================================
    // Perform statements
    // ========================================================================

    @Override
    public String visitPerformDT(EBLParser.PerformDTContext ctx) {
        return resolveDecisionTable(ctx.typedDecisionTable());
    }

    @Override
    public String visitPerformDTExplicit(EBLParser.PerformDTExplicitContext ctx) {
        return resolveDecisionTable(ctx.typedDecisionTable());
    }

    @Override
    public String visitPerformName(EBLParser.PerformNameContext ctx) {
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
    public String visitSetInt(EBLParser.SetIntContext ctx) {
        String v = resolveIdentLeft(ctx.leftIexpr().getText());
        String e = visit(ctx.number());
        return e + "cvi " + v;
    }

    @Override
    public String visitSetFloat(EBLParser.SetFloatContext ctx) {
        String v = resolveIdentLeft(ctx.leftFexpr().getText());
        String e = visit(ctx.number());
        return e + "cvr " + v;
    }

    @Override
    public String visitSetBool(EBLParser.SetBoolContext ctx) {
        String v = resolveIdentLeft(ctx.leftBexpr().getText());
        String e = visit(ctx.bexpr());
        return e + "cvb " + v;
    }

    @Override
    public String visitSetEntity(EBLParser.SetEntityContext ctx) {
        String v = resolveIdentLeft(ctx.leftEexpr().getText());
        String e = visit(ctx.eexpr());
        return e + "cve " + v;
    }

    @Override
    public String visitSetString(EBLParser.SetStringContext ctx) {
        String v = resolveIdentLeft(ctx.leftStrexpr().getText());
        String e = visit(ctx.strexpr());
        return e + "cvs " + v;
    }

    @Override
    public String visitSetDate(EBLParser.SetDateContext ctx) {
        String v = resolveIdentLeft(ctx.leftDexpr().getText());
        String e = visit(ctx.dexpr());
        return e + "cvd " + v;
    }

    @Override
    public String visitIncrementLong(EBLParser.IncrementLongContext ctx) {
        String v = resolveLong(ctx.typedLong());
        String left = resolveIdentLeft(ctx.typedLong().getText());
        return v + " 1 ladd " + left;
    }

    @Override
    public String visitIncrementDouble(EBLParser.IncrementDoubleContext ctx) {
        String v = resolveDouble(ctx.typedDouble());
        String left = resolveIdentLeft(ctx.typedDouble().getText());
        return v + " 1 dadd " + left;
    }

    @Override
    public String visitDecrementLong(EBLParser.DecrementLongContext ctx) {
        String v = resolveLong(ctx.typedLong());
        String left = resolveIdentLeft(ctx.typedLong().getText());
        return v + " 1 lsub " + left;
    }

    @Override
    public String visitDecrementDouble(EBLParser.DecrementDoubleContext ctx) {
        String v = resolveDouble(ctx.typedDouble());
        String left = resolveIdentLeft(ctx.typedDouble().getText());
        return v + " 1 dsub " + left;
    }

    // ========================================================================
    // If statements
    // ========================================================================

    @Override
    public String visitIfThen(EBLParser.IfThenContext ctx) {
        String b = visit(ctx.bexpr());
        String blk = visit(ctx.block());
        return "{ " + blk + "} " + b + "if ";
    }

    @Override
    public String visitIfThenElse(EBLParser.IfThenElseContext ctx) {
        String b = visit(ctx.bexpr());
        String blk1 = visit(ctx.block(0));
        String blk2 = visit(ctx.block(1));
        return "{ " + blk1 + "} { " + blk2 + "} " + b + "ifelse ";
    }

    @Override
    public String visitIfblock(EBLParser.IfblockContext ctx) {
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
    public String visitIfEnd(EBLParser.IfEndContext ctx) {
        return "";
    }

    @Override
    public String visitIfElse(EBLParser.IfElseContext ctx) {
        return visit(ctx.statementList());
    }

    @Override
    public String visitIfElseIf(EBLParser.IfElseIfContext ctx) {
        return visit(ctx.ifblock());
    }

    // ========================================================================
    // Possessive and colon references
    // ========================================================================

    @Override
    public String visitPossessiveChain(EBLParser.PossessiveChainContext ctx) {
        try {
            EBLTypeResolver.ResolvedIdentifier resolved = typeResolver.resolvePossessive(ctx.POSSESSIVE().getText());
            String e2 = visit(ctx.possessiveRef());
            return resolved.value + " entitypush " + e2 + "entitypop ";
        } catch (RulesException e) {
            throw new RuntimeException(e);
        }
    }

    @Override
    public String visitPossessiveSingle(EBLParser.PossessiveSingleContext ctx) {
        try {
            EBLTypeResolver.ResolvedIdentifier resolved = typeResolver.resolvePossessive(ctx.POSSESSIVE().getText());
            return resolved.value + " ";
        } catch (RulesException e) {
            throw new RuntimeException(e);
        }
    }

    @Override
    public String visitColonPossessiveChain(EBLParser.ColonPossessiveChainContext ctx) {
        String e1 = resolveEntity(ctx.typedEntity());
        String e2 = visit(ctx.possessiveRef());
        return e1 + " entitypush " + e2 + "entitypop ";
    }

    @Override
    public String visitColonPossessiveSingle(EBLParser.ColonPossessiveSingleContext ctx) {
        return resolveEntity(ctx.typedEntity()) + " ";
    }
}
