// Code generated from EL.g4 by ANTLR 4.13.1. DO NOT EDIT.

package el // EL
import "github.com/antlr4-go/antlr/v4"

type BaseELVisitor struct {
	*antlr.BaseParseTreeVisitor
}

func (v *BaseELVisitor) VisitOptSemi(ctx *OptSemiContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitEmptyAction(ctx *EmptyActionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitEmptyCondition(ctx *EmptyConditionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitEmptyContext(ctx *EmptyContextContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitEmptyPolicyStatement(ctx *EmptyPolicyStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitActionStatement(ctx *ActionStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitConditionExpr(ctx *ConditionExprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitConditionDebugBefore(ctx *ConditionDebugBeforeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitConditionDebugAfter(ctx *ConditionDebugAfterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitContextStatement(ctx *ContextStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitContextDebugBefore(ctx *ContextDebugBeforeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitPolicyStrExpr(ctx *PolicyStrExprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitPolicyNExpr(ctx *PolicyNExprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitPolicyIExpr(ctx *PolicyIExprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitPolicyFExpr(ctx *PolicyFExprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitPolicyBExpr(ctx *PolicyBExprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitPolicyDExpr(ctx *PolicyDExprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStatementList(ctx *StatementListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSeparator(ctx *SeparatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStatement(ctx *StatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitUsingBlockEntity(ctx *UsingBlockEntityContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitUsingBlockEntityComma(ctx *UsingBlockEntityCommaContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitUsingBlockBase(ctx *UsingBlockBaseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitPossessiveChain(ctx *PossessiveChainContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitColonChain(ctx *ColonChainContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitColonRef(ctx *ColonRefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitContextDebug(ctx *ContextDebugContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitContextFor(ctx *ContextForContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitContextForallCtl(ctx *ContextForallCtlContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitContextForfirst(ctx *ContextForfirstContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitContextCtx(ctx *ContextCtxContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitContextLocal(ctx *ContextLocalContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalEntityUndef(ctx *LocalEntityUndefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalEntityInit(ctx *LocalEntityInitContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalEntityDefined(ctx *LocalEntityDefinedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalLongUndef(ctx *LocalLongUndefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalLongInit(ctx *LocalLongInitContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalLongDefined(ctx *LocalLongDefinedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalDoubleUndef(ctx *LocalDoubleUndefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalDoubleInit(ctx *LocalDoubleInitContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalDoubleDefined(ctx *LocalDoubleDefinedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalBoolUndef(ctx *LocalBoolUndefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalBoolInit(ctx *LocalBoolInitContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalBoolDefined(ctx *LocalBoolDefinedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalDateUndef(ctx *LocalDateUndefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalDateInit(ctx *LocalDateInitContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalDateDefined(ctx *LocalDateDefinedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalArrayUndef(ctx *LocalArrayUndefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalArrayInit(ctx *LocalArrayInitContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalArrayDefined(ctx *LocalArrayDefinedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalStringUndef(ctx *LocalStringUndefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalStringInit(ctx *LocalStringInitContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalStringDefined(ctx *LocalStringDefinedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalBigIntUndef(ctx *LocalBigIntUndefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalBigIntInit(ctx *LocalBigIntInitContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalBigIntDefined(ctx *LocalBigIntDefinedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalBytesUndef(ctx *LocalBytesUndefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalBytesInit(ctx *LocalBytesInitContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLocalBytesDefined(ctx *LocalBytesDefinedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIfThen(ctx *IfThenContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIfThenElse(ctx *IfThenElseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitForallSimple(ctx *ForallSimpleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitForallAllowRemove(ctx *ForallAllowRemoveContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitForallInEntity(ctx *ForallInEntityContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitForallInEntityAllowRemove(ctx *ForallInEntityAllowRemoveContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitForallInEntityWhere(ctx *ForallInEntityWhereContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitForallWhere(ctx *ForallWhereContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitForallWhereAllowRemove(ctx *ForallWhereAllowRemoveContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitForallBlockSimple(ctx *ForallBlockSimpleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitForallBlockWhere(ctx *ForallBlockWhereContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitForeachSimple(ctx *ForeachSimpleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitForeachWhere(ctx *ForeachWhereContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitForeachIts(ctx *ForeachItsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitForeachItsWhere(ctx *ForeachItsWhereContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitForfirstOf(ctx *ForfirstOfContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitForfirstOfIts(ctx *ForfirstOfItsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitForfirstIn(ctx *ForfirstInContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFirstBlockElse(ctx *FirstBlockElseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFirstBlockSimple(ctx *FirstBlockSimpleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFirstBlockItsElse(ctx *FirstBlockItsElseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBlockCurly(ctx *BlockCurlyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBlockUsing(ctx *BlockUsingContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBlockGforall(ctx *BlockGforallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBlockForall(ctx *BlockForallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBlockForeach(ctx *BlockForeachContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBlockFirst(ctx *BlockFirstContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBlockIf(ctx *BlockIfContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBlockStatement(ctx *BlockStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitUsingstatement(ctx *UsingstatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLeftIexprSimple(ctx *LeftIexprSimpleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLeftIexprColon(ctx *LeftIexprColonContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLeftFexprSimple(ctx *LeftFexprSimpleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLeftFexprColon(ctx *LeftFexprColonContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLeftBexprSimple(ctx *LeftBexprSimpleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLeftBexprColon(ctx *LeftBexprColonContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLeftEexprSimple(ctx *LeftEexprSimpleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLeftEexprColon(ctx *LeftEexprColonContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLeftStrexprSimple(ctx *LeftStrexprSimpleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLeftStrexprColon(ctx *LeftStrexprColonContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLeftDexprSimple(ctx *LeftDexprSimpleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLeftDexprColon(ctx *LeftDexprColonContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLeftTexprSimple(ctx *LeftTexprSimpleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLeftTexprColon(ctx *LeftTexprColonContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLeftBigexprSimple(ctx *LeftBigexprSimpleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLeftBigexprColon(ctx *LeftBigexprColonContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLeftArraySimple(ctx *LeftArraySimpleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitLeftArrayColon(ctx *LeftArrayColonContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSetInt(ctx *SetIntContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSetFloat(ctx *SetFloatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSetBool(ctx *SetBoolContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSetEntity(ctx *SetEntityContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSetString(ctx *SetStringContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSetStringFromNumber(ctx *SetStringFromNumberContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSetStringFromDate(ctx *SetStringFromDateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSetStringFromName(ctx *SetStringFromNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSetStringFromTable(ctx *SetStringFromTableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSetBoolFromName(ctx *SetBoolFromNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSetDate(ctx *SetDateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSetTable(ctx *SetTableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSetArrayEntity(ctx *SetArrayEntityContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSetArrayString(ctx *SetArrayStringContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSetArrayFloat(ctx *SetArrayFloatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSetArrayInt(ctx *SetArrayIntContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSetArrayDate(ctx *SetArrayDateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSetArrayArray(ctx *SetArrayArrayContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSetBigInt(ctx *SetBigIntContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIncrementLong(ctx *IncrementLongContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIncrementDouble(ctx *IncrementDoubleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDecrementLong(ctx *DecrementLongContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDecrementDouble(ctx *DecrementDoubleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitForctl(ctx *ForctlContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitPerformCatchError(ctx *PerformCatchErrorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitPerformDT(ctx *PerformDTContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitPerformDTExplicit(ctx *PerformDTExplicitContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitPerformName(ctx *PerformNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitErrorStmt(ctx *ErrorStmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitWarnStmt(ctx *WarnStmtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDebugStr(ctx *DebugStrContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDebugBool(ctx *DebugBoolContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDebugInt(ctx *DebugIntContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDebugFloat(ctx *DebugFloatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDebugEntity(ctx *DebugEntityContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDebugDate(ctx *DebugDateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDebugArray(ctx *DebugArrayContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitPrintStr(ctx *PrintStrContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitPrintBool(ctx *PrintBoolContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitPrintInt(ctx *PrintIntContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitPrintFloat(ctx *PrintFloatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitPrintEntity(ctx *PrintEntityContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitPrintDate(ctx *PrintDateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitPrintArray(ctx *PrintArrayContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIfblock(ctx *IfblockContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIfEnd(ctx *IfEndContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIfElse(ctx *IfElseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIfElseIf(ctx *IfElseIfContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitNumber(ctx *NumberContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitAddDestArray2(ctx *AddDestArray2Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitAddDestLong2(ctx *AddDestLong2Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitAddDestDouble2(ctx *AddDestDouble2Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitAddDestArray(ctx *AddDestArrayContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitAddDestLong(ctx *AddDestLongContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitAddDestDouble(ctx *AddDestDoubleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitAddDestColon(ctx *AddDestColonContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitAddDestPossessiveLong(ctx *AddDestPossessiveLongContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitAddDestPossessiveDouble(ctx *AddDestPossessiveDoubleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSubDestLong(ctx *SubDestLongContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSubDestDouble(ctx *SubDestDoubleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSubDestColon(ctx *SubDestColonContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSubDestPossessiveLong(ctx *SubDestPossessiveLongContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSubDestPossessiveDouble(ctx *SubDestPossessiveDoubleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitAddArrayNoMember(ctx *AddArrayNoMemberContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitAddArrayToArray(ctx *AddArrayToArrayContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitAddEntityToDest(ctx *AddEntityToDestContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitAddEntityToDestDup(ctx *AddEntityToDestDupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitAddStrToDest(ctx *AddStrToDestContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitAddStrToDestDup(ctx *AddStrToDestDupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitAddDateToDest(ctx *AddDateToDestContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitAddDateToDestDup(ctx *AddDateToDestDupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitAddNumToDest(ctx *AddNumToDestContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitAddNumToDestDup(ctx *AddNumToDestDupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSubtractNum(ctx *SubtractNumContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitAddEntityNoDups(ctx *AddEntityNoDupsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitAddEntityNoDupsDup(ctx *AddEntityNoDupsDupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitAddStrNoDups(ctx *AddStrNoDupsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitAddStrNoDupsDup(ctx *AddStrNoDupsDupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitAddToContextOf(ctx *AddToContextOfContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitAddToContextFor(ctx *AddToContextForContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitClearstatement(ctx *ClearstatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitRemoveAtIndex(ctx *RemoveAtIndexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitRemoveEachWhere(ctx *RemoveEachWhereContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitRemoveName(ctx *RemoveNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitRemoveString(ctx *RemoveStringContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitRemoveEntity(ctx *RemoveEntityContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitRandomizeArray(ctx *RandomizeArrayContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitClearArray(ctx *ClearArrayContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSortAscending(ctx *SortAscendingContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitSortDescending(ctx *SortDescendingContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitOpListStr(ctx *OpListStrContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitOpListInt(ctx *OpListIntContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitOpListFloat(ctx *OpListFloatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitOpListEntity(ctx *OpListEntityContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitOpListStrSingle(ctx *OpListStrSingleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitOpListIntSingle(ctx *OpListIntSingleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitOpListFloatSingle(ctx *OpListFloatSingleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitOpListEntitySingle(ctx *OpListEntitySingleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitOperatorstatements(ctx *OperatorstatementsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitXmlvalues(ctx *XmlvaluesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitXmlSetAttr(ctx *XmlSetAttrContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitXmlSetAttrEntity(ctx *XmlSetAttrEntityContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitXmlAddAttr(ctx *XmlAddAttrContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitXmlAddAttrEntity(ctx *XmlAddAttrEntityContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayPolicyStatements(ctx *ArrayPolicyStatementsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayColonRef(ctx *ArrayColonRefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayBase(ctx *ArrayBaseContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayMap(ctx *ArrayMapContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayParen(ctx *ArrayParenContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayTyped(ctx *ArrayTypedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayName(ctx *ArrayNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayCopy(ctx *ArrayCopyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayCopySimple(ctx *ArrayCopySimpleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayDeepCopy(ctx *ArrayDeepCopyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayDeepCopySimple(ctx *ArrayDeepCopySimpleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayLiteral(ctx *ArrayLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayOfValues(ctx *ArrayOfValuesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayTokenize(ctx *ArrayTokenizeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayLit(ctx *ArrayLitContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayListNameSingle(ctx *ArrayListNameSingleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayListArraySingle(ctx *ArrayListArraySingleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayListBoolSingle(ctx *ArrayListBoolSingleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayListFloatSingle(ctx *ArrayListFloatSingleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayListBool(ctx *ArrayListBoolContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayListInt(ctx *ArrayListIntContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayListFloat(ctx *ArrayListFloatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayListStr(ctx *ArrayListStrContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayListArray(ctx *ArrayListArrayContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayListIntSingle(ctx *ArrayListIntSingleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayListName(ctx *ArrayListNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayListEntitySingle(ctx *ArrayListEntitySingleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayListStrSingle(ctx *ArrayListStrSingleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitArrayListEntity(ctx *ArrayListEntityContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIndxExpr(ctx *IndxExprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitEntityTyped(ctx *EntityTypedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitEntityParen(ctx *EntityParenContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitEntityIndex(ctx *EntityIndexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitEntityNewName(ctx *EntityNewNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitEntityNewTyped(ctx *EntityNewTypedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitEntityClone(ctx *EntityCloneContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitEntityColonRef(ctx *EntityColonRefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitEntityTableLookup(ctx *EntityTableLookupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitEntityFirstIn(ctx *EntityFirstInContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitEntityFirst(ctx *EntityFirstContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitEntityRelationship(ctx *EntityRelationshipContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateSubYears(ctx *DateSubYearsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateSubMonths(ctx *DateSubMonthsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateSubDays(ctx *DateSubDaysContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateAddYears(ctx *DateAddYearsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateAddMonths(ctx *DateAddMonthsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateAddDays(ctx *DateAddDaysContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateFromStrFunc(ctx *DateFromStrFuncContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateFromStrCast(ctx *DateFromStrCastContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateTableLookup(ctx *DateTableLookupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateExprSubMonths(ctx *DateExprSubMonthsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDatePlusYears(ctx *DatePlusYearsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateMinusDays(ctx *DateMinusDaysContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateExprAddYears(ctx *DateExprAddYearsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateTyped(ctx *DateTypedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateFirstOfYear(ctx *DateFirstOfYearContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateFirstOfMonth(ctx *DateFirstOfMonthContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateExprSubYears(ctx *DateExprSubYearsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateCurrentDate(ctx *DateCurrentDateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateAdd(ctx *DateAddContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateFromIndex(ctx *DateFromIndexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateExprAddDays(ctx *DateExprAddDaysContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateMinusYears(ctx *DateMinusYearsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDatePlusMonths(ctx *DatePlusMonthsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateMinusMonths(ctx *DateMinusMonthsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateEndOfMonth(ctx *DateEndOfMonthContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateUsing(ctx *DateUsingContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateExprAddMonths(ctx *DateExprAddMonthsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateEarliestAfter(ctx *DateEarliestAfterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDatePlusDays(ctx *DatePlusDaysContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateParen(ctx *DateParenContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateColonRef(ctx *DateColonRefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateSub(ctx *DateSubContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateDays(ctx *DateDaysContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateExprSubDays(ctx *DateExprSubDaysContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitDateFromArrayAt(ctx *DateFromArrayAtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitNameTyped(ctx *NameTypedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitNameOf(ctx *NameOfContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitNameTheName(ctx *NameTheNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitNameArrayAt(ctx *NameArrayAtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitNameLiteral(ctx *NameLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitNameUsing(ctx *NameUsingContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitNameColonRef(ctx *NameColonRefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitNameFromStr(ctx *NameFromStrContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitTableListMulti(ctx *TableListMultiContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitTableListSingle(ctx *TableListSingleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitTableTyped(ctx *TableTypedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitTableNew(ctx *TableNewContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrXmlValue(ctx *StrXmlValueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrToLower(ctx *StrToLowerContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrXmlAttr(ctx *StrXmlAttrContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrParen(ctx *StrParenContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrRelationship(ctx *StrRelationshipContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrConcatInt(ctx *StrConcatIntContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrSubstring(ctx *StrSubstringContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrConcat(ctx *StrConcatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrConcatEntity(ctx *StrConcatEntityContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrValueOfOp(ctx *StrValueOfOpContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrHexOfBytes(ctx *StrHexOfBytesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrConcatDate(ctx *StrConcatDateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrValueOfFloat(ctx *StrValueOfFloatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrValueOfInt(ctx *StrValueOfIntContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrColonRef(ctx *StrColonRefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrLiteral(ctx *StrLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrConcatInvalid(ctx *StrConcatInvalidContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrMappingKey(ctx *StrMappingKeyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrTableInfo(ctx *StrTableInfoContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrTyped(ctx *StrTypedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrConcatNull(ctx *StrConcatNullContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrAttrOf(ctx *StrAttrOfContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrValueOfDate(ctx *StrValueOfDateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrToUpper(ctx *StrToUpperContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrBase58CheckOfBytes(ctx *StrBase58CheckOfBytesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrValueOfBool(ctx *StrValueOfBoolContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrBech32OfBytes(ctx *StrBech32OfBytesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrConcatFloat(ctx *StrConcatFloatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrTableLookup(ctx *StrTableLookupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrUsing(ctx *StrUsingContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrConcatArray(ctx *StrConcatArrayContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrTimestamp(ctx *StrTimestampContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrFromIndex(ctx *StrFromIndexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrTrim(ctx *StrTrimContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitStrConcatName(ctx *StrConcatNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatAddFloat(ctx *FloatAddFloatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatParen(ctx *FloatParenContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatMulFloat(ctx *FloatMulFloatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatDivFloat(ctx *FloatDivFloatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatValueOfOp(ctx *FloatValueOfOpContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatRoundedTo(ctx *FloatRoundedToContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatAddInt(ctx *FloatAddIntContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatTableLookup(ctx *FloatTableLookupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatSubFloat(ctx *FloatSubFloatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatLiteral(ctx *FloatLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatMulBy(ctx *FloatMulByContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatUsing(ctx *FloatUsingContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntDivFloat(ctx *IntDivFloatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntAddFloat(ctx *IntAddFloatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatTyped(ctx *FloatTypedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatSubInt(ctx *FloatSubIntContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatDivInt(ctx *FloatDivIntContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatSubFrom(ctx *FloatSubFromContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntSubFloat(ctx *IntSubFloatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntMulFloat(ctx *IntMulFloatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatDivBy(ctx *FloatDivByContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatFromIndex(ctx *FloatFromIndexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatRounded(ctx *FloatRoundedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatRoundedBoundry(ctx *FloatRoundedBoundryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatColonRef(ctx *FloatColonRefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatMulInt(ctx *FloatMulIntContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatFromInt(ctx *FloatFromIntContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatAddTo(ctx *FloatAddToContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatFromStr(ctx *FloatFromStrContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatAbs(ctx *FloatAbsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatNegate(ctx *FloatNegateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitFloatSumOf(ctx *FloatSumOfContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntDaysInYear(ctx *IntDaysInYearContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntYearOf(ctx *IntYearOfContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntAdd(ctx *IntAddContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntIndexOf(ctx *IntIndexOfContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntNumberOf(ctx *IntNumberOfContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntNumberOfWhere(ctx *IntNumberOfWhereContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntParen(ctx *IntParenContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntFromNumber(ctx *IntFromNumberContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntUsing(ctx *IntUsingContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntYearsBetween(ctx *IntYearsBetweenContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntValueOfOp(ctx *IntValueOfOpContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntTableLookup(ctx *IntTableLookupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntSumOf(ctx *IntSumOfContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntDiv(ctx *IntDivContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntLengthArray(ctx *IntLengthArrayContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntDayOfMonth(ctx *IntDayOfMonthContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntSub(ctx *IntSubContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntMulBy(ctx *IntMulByContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntFromStr(ctx *IntFromStrContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntMul(ctx *IntMulContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntDaysBetween(ctx *IntDaysBetweenContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntLiteral(ctx *IntLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntTyped(ctx *IntTypedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntFromIndex(ctx *IntFromIndexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntLengthStr(ctx *IntLengthStrContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntUsingArray(ctx *IntUsingArrayContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntNegate(ctx *IntNegateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntAddTo(ctx *IntAddToContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntDivBy(ctx *IntDivByContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntAbs(ctx *IntAbsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntDaysInMonth(ctx *IntDaysInMonthContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntColonRef(ctx *IntColonRefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntBytesIndex(ctx *IntBytesIndexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntSubFrom(ctx *IntSubFromContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntMonthsBetween(ctx *IntMonthsBetweenContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIntLengthBytes(ctx *IntLengthBytesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBigAbs(ctx *BigAbsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBigDiv(ctx *BigDivContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBigColonRef(ctx *BigColonRefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBigFromBytes(ctx *BigFromBytesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBigFromFloat(ctx *BigFromFloatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBigNegate(ctx *BigNegateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBigUsing(ctx *BigUsingContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBigSub(ctx *BigSubContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBigParen(ctx *BigParenContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBigAdd(ctx *BigAddContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBigFromStr(ctx *BigFromStrContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBigFromInt(ctx *BigFromIntContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBigMul(ctx *BigMulContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBigTyped(ctx *BigTypedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBytesSha256(ctx *BytesSha256Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBytesLiteral(ctx *BytesLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBytesCvBase58Check(ctx *BytesCvBase58CheckContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBytesCvBech32(ctx *BytesCvBech32Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBytesRipemd160(ctx *BytesRipemd160Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBytesColonRef(ctx *BytesColonRefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBytesCvHex(ctx *BytesCvHexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBytesCvBigInt(ctx *BytesCvBigIntContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBytesSlice(ctx *BytesSliceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBytesConcat(ctx *BytesConcatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBytesKeccak256(ctx *BytesKeccak256Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBytesTyped(ctx *BytesTypedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBytesParen(ctx *BytesParenContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBytesSha3(ctx *BytesSha3Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIncludeNumber(ctx *IncludeNumberContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIncludeDate(ctx *IncludeDateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIncludeEntity(ctx *IncludeEntityContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitIncludeString(ctx *IncludeStringContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitInthe(ctx *IntheContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitThereis(ctx *ThereisContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBlistMulti(ctx *BlistMultiContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBlistOr(ctx *BlistOrContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBlistIcMulti(ctx *BlistIcMultiContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBlistIcOr(ctx *BlistIcOrContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolIntLteFloat(ctx *BoolIntLteFloatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolFloatLteInt(ctx *BoolFloatLteIntContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolFromStr(ctx *BoolFromStrContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolNumIsNull(ctx *BoolNumIsNullContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolEntityIsOf(ctx *BoolEntityIsOfContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolTypedIsLiteral(ctx *BoolTypedIsLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolDateGte(ctx *BoolDateGteContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolNameEq(ctx *BoolNameEqContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolStartsWith(ctx *BoolStartsWithContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolThereIsNoInEntityWhere(ctx *BoolThereIsNoInEntityWhereContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolBigLt(ctx *BoolBigLtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolArrayIsNull(ctx *BoolArrayIsNullContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolIntEq(ctx *BoolIntEqContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolEntityHasaWhere(ctx *BoolEntityHasaWhereContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolStrEqList(ctx *BoolStrEqListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolBigNeq(ctx *BoolBigNeqContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolIntLt(ctx *BoolIntLtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolIntEqFloat(ctx *BoolIntEqFloatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolFromIndex(ctx *BoolFromIndexContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolFloatNeq(ctx *BoolFloatNeqContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolFloatLt(ctx *BoolFloatLtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolThereIsWhere(ctx *BoolThereIsWhereContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolBoolNeq(ctx *BoolBoolNeqContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolOneOfHasa(ctx *BoolOneOfHasaContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolStrIsOneOf(ctx *BoolStrIsOneOfContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolNameNeq(ctx *BoolNameNeqContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolColonIsNotLiteral(ctx *BoolColonIsNotLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolThereIsNoInArrayWhere(ctx *BoolThereIsNoInArrayWhereContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolStrIsNotOneOf(ctx *BoolStrIsNotOneOfContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolWasQuestion(ctx *BoolWasQuestionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolIntGteFloat(ctx *BoolIntGteFloatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolNameNeqStr(ctx *BoolNameNeqStrContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolDateIsNull(ctx *BoolDateIsNullContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolEntityInContext(ctx *BoolEntityInContextContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolDateAfter(ctx *BoolDateAfterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolBytesNeq(ctx *BoolBytesNeqContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolDateLt(ctx *BoolDateLtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolStrEntityInContext(ctx *BoolStrEntityInContextContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolFloatEq(ctx *BoolFloatEqContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolDateLte(ctx *BoolDateLteContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolFloatGtInt(ctx *BoolFloatGtIntContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolLiteral(ctx *BoolLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolEntityIsNull(ctx *BoolEntityIsNullContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolStrEq(ctx *BoolStrEqContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolEntityNeq(ctx *BoolEntityNeqContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolIntGte(ctx *BoolIntGteContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolDoesQuestion(ctx *BoolDoesQuestionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolNot(ctx *BoolNotContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolStrIsNotNull(ctx *BoolStrIsNotNullContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolAnd(ctx *BoolAndContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolBytesEq(ctx *BoolBytesEqContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolStrIsNot(ctx *BoolStrIsNotContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolIntGt(ctx *BoolIntGtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolFloatLte(ctx *BoolFloatLteContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolBigLte(ctx *BoolBigLteContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolStrEqIc(ctx *BoolStrEqIcContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolTyped(ctx *BoolTypedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolUsing(ctx *BoolUsingContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolEntityNotInContext(ctx *BoolEntityNotInContextContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolStrLt(ctx *BoolStrLtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolStrGte(ctx *BoolStrGteContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolStrEntityNotInContext(ctx *BoolStrEntityNotInContextContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolArrayDoesInclude(ctx *BoolArrayDoesIncludeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolIntGtFloat(ctx *BoolIntGtFloatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolValueOfOp(ctx *BoolValueOfOpContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolColonRef(ctx *BoolColonRefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolBigGt(ctx *BoolBigGtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolFloatGt(ctx *BoolFloatGtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolStrIsNull(ctx *BoolStrIsNullContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolStrGt(ctx *BoolStrGtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolColonIsLiteral(ctx *BoolColonIsLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolEntityEq(ctx *BoolEntityEqContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolNumIsNotNull(ctx *BoolNumIsNotNullContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolStartsWithAt(ctx *BoolStartsWithAtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolMatches(ctx *BoolMatchesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolFloatGteInt(ctx *BoolFloatGteIntContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolStrNeqIc(ctx *BoolStrNeqIcContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolArrayIsNotNull(ctx *BoolArrayIsNotNullContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolDateBetween(ctx *BoolDateBetweenContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolBexprIsNotNull(ctx *BoolBexprIsNotNullContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolIntLte(ctx *BoolIntLteContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolIntNeqFloat(ctx *BoolIntNeqFloatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolArrayAt(ctx *BoolArrayAtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolEntityNotHas(ctx *BoolEntityNotHasContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolBigGte(ctx *BoolBigGteContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolDateEq(ctx *BoolDateEqContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolFloatGte(ctx *BoolFloatGteContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolStrLte(ctx *BoolStrLteContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolNameEqStr(ctx *BoolNameEqStrContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolDateBefore(ctx *BoolDateBeforeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolEntityHasa(ctx *BoolEntityHasaContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolThereIsInEntityWhere(ctx *BoolThereIsInEntityWhereContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolFloatLtInt(ctx *BoolFloatLtIntContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolArrayNotInclude(ctx *BoolArrayNotIncludeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolFloatNeqInt(ctx *BoolFloatNeqIntContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolBexprIsNull(ctx *BoolBexprIsNullContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolAllHave(ctx *BoolAllHaveContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolIntLtFloat(ctx *BoolIntLtFloatContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolBoolEq(ctx *BoolBoolEqContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolBigEq(ctx *BoolBigEqContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolPlusOrMinus(ctx *BoolPlusOrMinusContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolParen(ctx *BoolParenContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolStrIs(ctx *BoolStrIsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolThereIsNoWhere(ctx *BoolThereIsNoWhereContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolFloatEqInt(ctx *BoolFloatEqIntContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolIntNeq(ctx *BoolIntNeqContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolArrayIncludes(ctx *BoolArrayIncludesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolStrEqIcList(ctx *BoolStrEqIcListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolDateGt(ctx *BoolDateGtContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolOr(ctx *BoolOrContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolFunction(ctx *BoolFunctionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolDateIsNotNull(ctx *BoolDateIsNotNullContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolThereIsInArrayWhere(ctx *BoolThereIsInArrayWhereContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolTypedIsNotLiteral(ctx *BoolTypedIsNotLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolEntityIsNotNull(ctx *BoolEntityIsNotNullContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolStrNeq(ctx *BoolStrNeqContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolMatchForall(ctx *BoolMatchForallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolWithinPercent(ctx *BoolWithinPercentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitBoolIsQuestion(ctx *BoolIsQuestionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitCommonerror(ctx *CommonerrorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitTypedEntity(ctx *TypedEntityContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitTypedLong(ctx *TypedLongContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitTypedDouble(ctx *TypedDoubleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitTypedString(ctx *TypedStringContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitTypedBoolean(ctx *TypedBooleanContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitTypedDate(ctx *TypedDateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitTypedArray(ctx *TypedArrayContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitTypedTable(ctx *TypedTableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitTypedName(ctx *TypedNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitTypedDecisionTable(ctx *TypedDecisionTableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitTypedOperator(ctx *TypedOperatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitTypedXmlValue(ctx *TypedXmlValueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitTypedNull(ctx *TypedNullContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitTypedInvalid(ctx *TypedInvalidContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitTypedBoolFunction(ctx *TypedBoolFunctionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitTypedBigInt(ctx *TypedBigIntContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitTypedBytes(ctx *TypedBytesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseELVisitor) VisitUndefinedIdent(ctx *UndefinedIdentContext) interface{} {
	return v.VisitChildren(ctx)
}
