// Code generated from EL.g4 by ANTLR 4.13.1. DO NOT EDIT.

package el

import "github.com/antlr4-go/antlr/v4"

// A complete Visitor for a parse tree produced by ELParser.
type ELVisitor interface {
	antlr.ParseTreeVisitor

	// Visit a parse tree produced by ELParser#optSemi.
	VisitOptSemi(ctx *OptSemiContext) interface{}

	// Visit a parse tree produced by ELParser#emptyAction.
	VisitEmptyAction(ctx *EmptyActionContext) interface{}

	// Visit a parse tree produced by ELParser#emptyCondition.
	VisitEmptyCondition(ctx *EmptyConditionContext) interface{}

	// Visit a parse tree produced by ELParser#emptyContext.
	VisitEmptyContext(ctx *EmptyContextContext) interface{}

	// Visit a parse tree produced by ELParser#emptyPolicyStatement.
	VisitEmptyPolicyStatement(ctx *EmptyPolicyStatementContext) interface{}

	// Visit a parse tree produced by ELParser#actionStatement.
	VisitActionStatement(ctx *ActionStatementContext) interface{}

	// Visit a parse tree produced by ELParser#conditionExpr.
	VisitConditionExpr(ctx *ConditionExprContext) interface{}

	// Visit a parse tree produced by ELParser#conditionDebugBefore.
	VisitConditionDebugBefore(ctx *ConditionDebugBeforeContext) interface{}

	// Visit a parse tree produced by ELParser#conditionDebugAfter.
	VisitConditionDebugAfter(ctx *ConditionDebugAfterContext) interface{}

	// Visit a parse tree produced by ELParser#contextStatement.
	VisitContextStatement(ctx *ContextStatementContext) interface{}

	// Visit a parse tree produced by ELParser#contextDebugBefore.
	VisitContextDebugBefore(ctx *ContextDebugBeforeContext) interface{}

	// Visit a parse tree produced by ELParser#policyStrExpr.
	VisitPolicyStrExpr(ctx *PolicyStrExprContext) interface{}

	// Visit a parse tree produced by ELParser#policyNExpr.
	VisitPolicyNExpr(ctx *PolicyNExprContext) interface{}

	// Visit a parse tree produced by ELParser#policyIExpr.
	VisitPolicyIExpr(ctx *PolicyIExprContext) interface{}

	// Visit a parse tree produced by ELParser#policyFExpr.
	VisitPolicyFExpr(ctx *PolicyFExprContext) interface{}

	// Visit a parse tree produced by ELParser#policyBExpr.
	VisitPolicyBExpr(ctx *PolicyBExprContext) interface{}

	// Visit a parse tree produced by ELParser#policyDExpr.
	VisitPolicyDExpr(ctx *PolicyDExprContext) interface{}

	// Visit a parse tree produced by ELParser#statementList.
	VisitStatementList(ctx *StatementListContext) interface{}

	// Visit a parse tree produced by ELParser#separator.
	VisitSeparator(ctx *SeparatorContext) interface{}

	// Visit a parse tree produced by ELParser#statement.
	VisitStatement(ctx *StatementContext) interface{}

	// Visit a parse tree produced by ELParser#usingBlockEntity.
	VisitUsingBlockEntity(ctx *UsingBlockEntityContext) interface{}

	// Visit a parse tree produced by ELParser#usingBlockEntityComma.
	VisitUsingBlockEntityComma(ctx *UsingBlockEntityCommaContext) interface{}

	// Visit a parse tree produced by ELParser#usingBlockBase.
	VisitUsingBlockBase(ctx *UsingBlockBaseContext) interface{}

	// Visit a parse tree produced by ELParser#possessiveChain.
	VisitPossessiveChain(ctx *PossessiveChainContext) interface{}

	// Visit a parse tree produced by ELParser#colonChain.
	VisitColonChain(ctx *ColonChainContext) interface{}

	// Visit a parse tree produced by ELParser#colonRef.
	VisitColonRef(ctx *ColonRefContext) interface{}

	// Visit a parse tree produced by ELParser#contextDebug.
	VisitContextDebug(ctx *ContextDebugContext) interface{}

	// Visit a parse tree produced by ELParser#contextFor.
	VisitContextFor(ctx *ContextForContext) interface{}

	// Visit a parse tree produced by ELParser#contextForallCtl.
	VisitContextForallCtl(ctx *ContextForallCtlContext) interface{}

	// Visit a parse tree produced by ELParser#contextForfirst.
	VisitContextForfirst(ctx *ContextForfirstContext) interface{}

	// Visit a parse tree produced by ELParser#contextCtx.
	VisitContextCtx(ctx *ContextCtxContext) interface{}

	// Visit a parse tree produced by ELParser#contextLocal.
	VisitContextLocal(ctx *ContextLocalContext) interface{}

	// Visit a parse tree produced by ELParser#localEntityUndef.
	VisitLocalEntityUndef(ctx *LocalEntityUndefContext) interface{}

	// Visit a parse tree produced by ELParser#localEntityInit.
	VisitLocalEntityInit(ctx *LocalEntityInitContext) interface{}

	// Visit a parse tree produced by ELParser#localEntityDefined.
	VisitLocalEntityDefined(ctx *LocalEntityDefinedContext) interface{}

	// Visit a parse tree produced by ELParser#localLongUndef.
	VisitLocalLongUndef(ctx *LocalLongUndefContext) interface{}

	// Visit a parse tree produced by ELParser#localLongInit.
	VisitLocalLongInit(ctx *LocalLongInitContext) interface{}

	// Visit a parse tree produced by ELParser#localLongDefined.
	VisitLocalLongDefined(ctx *LocalLongDefinedContext) interface{}

	// Visit a parse tree produced by ELParser#localDoubleUndef.
	VisitLocalDoubleUndef(ctx *LocalDoubleUndefContext) interface{}

	// Visit a parse tree produced by ELParser#localDoubleInit.
	VisitLocalDoubleInit(ctx *LocalDoubleInitContext) interface{}

	// Visit a parse tree produced by ELParser#localDoubleDefined.
	VisitLocalDoubleDefined(ctx *LocalDoubleDefinedContext) interface{}

	// Visit a parse tree produced by ELParser#localBoolUndef.
	VisitLocalBoolUndef(ctx *LocalBoolUndefContext) interface{}

	// Visit a parse tree produced by ELParser#localBoolInit.
	VisitLocalBoolInit(ctx *LocalBoolInitContext) interface{}

	// Visit a parse tree produced by ELParser#localBoolDefined.
	VisitLocalBoolDefined(ctx *LocalBoolDefinedContext) interface{}

	// Visit a parse tree produced by ELParser#localDateUndef.
	VisitLocalDateUndef(ctx *LocalDateUndefContext) interface{}

	// Visit a parse tree produced by ELParser#localDateInit.
	VisitLocalDateInit(ctx *LocalDateInitContext) interface{}

	// Visit a parse tree produced by ELParser#localDateDefined.
	VisitLocalDateDefined(ctx *LocalDateDefinedContext) interface{}

	// Visit a parse tree produced by ELParser#localArrayUndef.
	VisitLocalArrayUndef(ctx *LocalArrayUndefContext) interface{}

	// Visit a parse tree produced by ELParser#localArrayInit.
	VisitLocalArrayInit(ctx *LocalArrayInitContext) interface{}

	// Visit a parse tree produced by ELParser#localArrayDefined.
	VisitLocalArrayDefined(ctx *LocalArrayDefinedContext) interface{}

	// Visit a parse tree produced by ELParser#localStringUndef.
	VisitLocalStringUndef(ctx *LocalStringUndefContext) interface{}

	// Visit a parse tree produced by ELParser#localStringInit.
	VisitLocalStringInit(ctx *LocalStringInitContext) interface{}

	// Visit a parse tree produced by ELParser#localStringDefined.
	VisitLocalStringDefined(ctx *LocalStringDefinedContext) interface{}

	// Visit a parse tree produced by ELParser#localBigIntUndef.
	VisitLocalBigIntUndef(ctx *LocalBigIntUndefContext) interface{}

	// Visit a parse tree produced by ELParser#localBigIntInit.
	VisitLocalBigIntInit(ctx *LocalBigIntInitContext) interface{}

	// Visit a parse tree produced by ELParser#localBigIntDefined.
	VisitLocalBigIntDefined(ctx *LocalBigIntDefinedContext) interface{}

	// Visit a parse tree produced by ELParser#localFixedUndef.
	VisitLocalFixedUndef(ctx *LocalFixedUndefContext) interface{}

	// Visit a parse tree produced by ELParser#localFixedInit.
	VisitLocalFixedInit(ctx *LocalFixedInitContext) interface{}

	// Visit a parse tree produced by ELParser#localFixedDefined.
	VisitLocalFixedDefined(ctx *LocalFixedDefinedContext) interface{}

	// Visit a parse tree produced by ELParser#localBytesUndef.
	VisitLocalBytesUndef(ctx *LocalBytesUndefContext) interface{}

	// Visit a parse tree produced by ELParser#localBytesInit.
	VisitLocalBytesInit(ctx *LocalBytesInitContext) interface{}

	// Visit a parse tree produced by ELParser#localBytesDefined.
	VisitLocalBytesDefined(ctx *LocalBytesDefinedContext) interface{}

	// Visit a parse tree produced by ELParser#ifThen.
	VisitIfThen(ctx *IfThenContext) interface{}

	// Visit a parse tree produced by ELParser#ifThenElse.
	VisitIfThenElse(ctx *IfThenElseContext) interface{}

	// Visit a parse tree produced by ELParser#forallSimple.
	VisitForallSimple(ctx *ForallSimpleContext) interface{}

	// Visit a parse tree produced by ELParser#forallAllowRemove.
	VisitForallAllowRemove(ctx *ForallAllowRemoveContext) interface{}

	// Visit a parse tree produced by ELParser#forallInEntity.
	VisitForallInEntity(ctx *ForallInEntityContext) interface{}

	// Visit a parse tree produced by ELParser#forallInEntityAllowRemove.
	VisitForallInEntityAllowRemove(ctx *ForallInEntityAllowRemoveContext) interface{}

	// Visit a parse tree produced by ELParser#forallInEntityWhere.
	VisitForallInEntityWhere(ctx *ForallInEntityWhereContext) interface{}

	// Visit a parse tree produced by ELParser#forallWhere.
	VisitForallWhere(ctx *ForallWhereContext) interface{}

	// Visit a parse tree produced by ELParser#forallWhereAllowRemove.
	VisitForallWhereAllowRemove(ctx *ForallWhereAllowRemoveContext) interface{}

	// Visit a parse tree produced by ELParser#forallTypeEntities.
	VisitForallTypeEntities(ctx *ForallTypeEntitiesContext) interface{}

	// Visit a parse tree produced by ELParser#forallTypeEntitiesWhere.
	VisitForallTypeEntitiesWhere(ctx *ForallTypeEntitiesWhereContext) interface{}

	// Visit a parse tree produced by ELParser#forallAs.
	VisitForallAs(ctx *ForallAsContext) interface{}

	// Visit a parse tree produced by ELParser#forallAsWhere.
	VisitForallAsWhere(ctx *ForallAsWhereContext) interface{}

	// Visit a parse tree produced by ELParser#forallBlockSimple.
	VisitForallBlockSimple(ctx *ForallBlockSimpleContext) interface{}

	// Visit a parse tree produced by ELParser#forallBlockWhere.
	VisitForallBlockWhere(ctx *ForallBlockWhereContext) interface{}

	// Visit a parse tree produced by ELParser#foreachSimple.
	VisitForeachSimple(ctx *ForeachSimpleContext) interface{}

	// Visit a parse tree produced by ELParser#foreachWhere.
	VisitForeachWhere(ctx *ForeachWhereContext) interface{}

	// Visit a parse tree produced by ELParser#foreachIts.
	VisitForeachIts(ctx *ForeachItsContext) interface{}

	// Visit a parse tree produced by ELParser#foreachItsWhere.
	VisitForeachItsWhere(ctx *ForeachItsWhereContext) interface{}

	// Visit a parse tree produced by ELParser#forfirstOf.
	VisitForfirstOf(ctx *ForfirstOfContext) interface{}

	// Visit a parse tree produced by ELParser#forfirstOfIts.
	VisitForfirstOfIts(ctx *ForfirstOfItsContext) interface{}

	// Visit a parse tree produced by ELParser#forfirstIn.
	VisitForfirstIn(ctx *ForfirstInContext) interface{}

	// Visit a parse tree produced by ELParser#firstBlockElse.
	VisitFirstBlockElse(ctx *FirstBlockElseContext) interface{}

	// Visit a parse tree produced by ELParser#firstBlockSimple.
	VisitFirstBlockSimple(ctx *FirstBlockSimpleContext) interface{}

	// Visit a parse tree produced by ELParser#firstBlockItsElse.
	VisitFirstBlockItsElse(ctx *FirstBlockItsElseContext) interface{}

	// Visit a parse tree produced by ELParser#blockCurly.
	VisitBlockCurly(ctx *BlockCurlyContext) interface{}

	// Visit a parse tree produced by ELParser#blockUsing.
	VisitBlockUsing(ctx *BlockUsingContext) interface{}

	// Visit a parse tree produced by ELParser#blockGforall.
	VisitBlockGforall(ctx *BlockGforallContext) interface{}

	// Visit a parse tree produced by ELParser#blockForall.
	VisitBlockForall(ctx *BlockForallContext) interface{}

	// Visit a parse tree produced by ELParser#blockForeach.
	VisitBlockForeach(ctx *BlockForeachContext) interface{}

	// Visit a parse tree produced by ELParser#blockFirst.
	VisitBlockFirst(ctx *BlockFirstContext) interface{}

	// Visit a parse tree produced by ELParser#blockIf.
	VisitBlockIf(ctx *BlockIfContext) interface{}

	// Visit a parse tree produced by ELParser#blockStatement.
	VisitBlockStatement(ctx *BlockStatementContext) interface{}

	// Visit a parse tree produced by ELParser#usingstatement.
	VisitUsingstatement(ctx *UsingstatementContext) interface{}

	// Visit a parse tree produced by ELParser#leftIexprSimple.
	VisitLeftIexprSimple(ctx *LeftIexprSimpleContext) interface{}

	// Visit a parse tree produced by ELParser#leftIexprColon.
	VisitLeftIexprColon(ctx *LeftIexprColonContext) interface{}

	// Visit a parse tree produced by ELParser#leftFexprSimple.
	VisitLeftFexprSimple(ctx *LeftFexprSimpleContext) interface{}

	// Visit a parse tree produced by ELParser#leftFexprColon.
	VisitLeftFexprColon(ctx *LeftFexprColonContext) interface{}

	// Visit a parse tree produced by ELParser#leftBexprSimple.
	VisitLeftBexprSimple(ctx *LeftBexprSimpleContext) interface{}

	// Visit a parse tree produced by ELParser#leftBexprColon.
	VisitLeftBexprColon(ctx *LeftBexprColonContext) interface{}

	// Visit a parse tree produced by ELParser#leftEexprSimple.
	VisitLeftEexprSimple(ctx *LeftEexprSimpleContext) interface{}

	// Visit a parse tree produced by ELParser#leftEexprColon.
	VisitLeftEexprColon(ctx *LeftEexprColonContext) interface{}

	// Visit a parse tree produced by ELParser#leftStrexprSimple.
	VisitLeftStrexprSimple(ctx *LeftStrexprSimpleContext) interface{}

	// Visit a parse tree produced by ELParser#leftStrexprColon.
	VisitLeftStrexprColon(ctx *LeftStrexprColonContext) interface{}

	// Visit a parse tree produced by ELParser#leftDexprSimple.
	VisitLeftDexprSimple(ctx *LeftDexprSimpleContext) interface{}

	// Visit a parse tree produced by ELParser#leftDexprColon.
	VisitLeftDexprColon(ctx *LeftDexprColonContext) interface{}

	// Visit a parse tree produced by ELParser#leftTexprSimple.
	VisitLeftTexprSimple(ctx *LeftTexprSimpleContext) interface{}

	// Visit a parse tree produced by ELParser#leftTexprColon.
	VisitLeftTexprColon(ctx *LeftTexprColonContext) interface{}

	// Visit a parse tree produced by ELParser#leftBigexprSimple.
	VisitLeftBigexprSimple(ctx *LeftBigexprSimpleContext) interface{}

	// Visit a parse tree produced by ELParser#leftBigexprColon.
	VisitLeftBigexprColon(ctx *LeftBigexprColonContext) interface{}

	// Visit a parse tree produced by ELParser#leftArraySimple.
	VisitLeftArraySimple(ctx *LeftArraySimpleContext) interface{}

	// Visit a parse tree produced by ELParser#leftArrayColon.
	VisitLeftArrayColon(ctx *LeftArrayColonContext) interface{}

	// Visit a parse tree produced by ELParser#setInt.
	VisitSetInt(ctx *SetIntContext) interface{}

	// Visit a parse tree produced by ELParser#setFloat.
	VisitSetFloat(ctx *SetFloatContext) interface{}

	// Visit a parse tree produced by ELParser#setBool.
	VisitSetBool(ctx *SetBoolContext) interface{}

	// Visit a parse tree produced by ELParser#setEntity.
	VisitSetEntity(ctx *SetEntityContext) interface{}

	// Visit a parse tree produced by ELParser#setString.
	VisitSetString(ctx *SetStringContext) interface{}

	// Visit a parse tree produced by ELParser#setStringFromNumber.
	VisitSetStringFromNumber(ctx *SetStringFromNumberContext) interface{}

	// Visit a parse tree produced by ELParser#setStringFromDate.
	VisitSetStringFromDate(ctx *SetStringFromDateContext) interface{}

	// Visit a parse tree produced by ELParser#setStringFromName.
	VisitSetStringFromName(ctx *SetStringFromNameContext) interface{}

	// Visit a parse tree produced by ELParser#setStringFromTable.
	VisitSetStringFromTable(ctx *SetStringFromTableContext) interface{}

	// Visit a parse tree produced by ELParser#setBoolFromName.
	VisitSetBoolFromName(ctx *SetBoolFromNameContext) interface{}

	// Visit a parse tree produced by ELParser#setDate.
	VisitSetDate(ctx *SetDateContext) interface{}

	// Visit a parse tree produced by ELParser#setTable.
	VisitSetTable(ctx *SetTableContext) interface{}

	// Visit a parse tree produced by ELParser#setArrayEntity.
	VisitSetArrayEntity(ctx *SetArrayEntityContext) interface{}

	// Visit a parse tree produced by ELParser#setArrayString.
	VisitSetArrayString(ctx *SetArrayStringContext) interface{}

	// Visit a parse tree produced by ELParser#setArrayFloat.
	VisitSetArrayFloat(ctx *SetArrayFloatContext) interface{}

	// Visit a parse tree produced by ELParser#setArrayInt.
	VisitSetArrayInt(ctx *SetArrayIntContext) interface{}

	// Visit a parse tree produced by ELParser#setArrayDate.
	VisitSetArrayDate(ctx *SetArrayDateContext) interface{}

	// Visit a parse tree produced by ELParser#setArrayArray.
	VisitSetArrayArray(ctx *SetArrayArrayContext) interface{}

	// Visit a parse tree produced by ELParser#setBigInt.
	VisitSetBigInt(ctx *SetBigIntContext) interface{}

	// Visit a parse tree produced by ELParser#incrementLong.
	VisitIncrementLong(ctx *IncrementLongContext) interface{}

	// Visit a parse tree produced by ELParser#incrementDouble.
	VisitIncrementDouble(ctx *IncrementDoubleContext) interface{}

	// Visit a parse tree produced by ELParser#decrementLong.
	VisitDecrementLong(ctx *DecrementLongContext) interface{}

	// Visit a parse tree produced by ELParser#decrementDouble.
	VisitDecrementDouble(ctx *DecrementDoubleContext) interface{}

	// Visit a parse tree produced by ELParser#forctl.
	VisitForctl(ctx *ForctlContext) interface{}

	// Visit a parse tree produced by ELParser#performCatchError.
	VisitPerformCatchError(ctx *PerformCatchErrorContext) interface{}

	// Visit a parse tree produced by ELParser#performDT.
	VisitPerformDT(ctx *PerformDTContext) interface{}

	// Visit a parse tree produced by ELParser#performDTExplicit.
	VisitPerformDTExplicit(ctx *PerformDTExplicitContext) interface{}

	// Visit a parse tree produced by ELParser#performName.
	VisitPerformName(ctx *PerformNameContext) interface{}

	// Visit a parse tree produced by ELParser#errorStmt.
	VisitErrorStmt(ctx *ErrorStmtContext) interface{}

	// Visit a parse tree produced by ELParser#warnStmt.
	VisitWarnStmt(ctx *WarnStmtContext) interface{}

	// Visit a parse tree produced by ELParser#debugStr.
	VisitDebugStr(ctx *DebugStrContext) interface{}

	// Visit a parse tree produced by ELParser#debugBool.
	VisitDebugBool(ctx *DebugBoolContext) interface{}

	// Visit a parse tree produced by ELParser#debugInt.
	VisitDebugInt(ctx *DebugIntContext) interface{}

	// Visit a parse tree produced by ELParser#debugFloat.
	VisitDebugFloat(ctx *DebugFloatContext) interface{}

	// Visit a parse tree produced by ELParser#debugEntity.
	VisitDebugEntity(ctx *DebugEntityContext) interface{}

	// Visit a parse tree produced by ELParser#debugDate.
	VisitDebugDate(ctx *DebugDateContext) interface{}

	// Visit a parse tree produced by ELParser#debugArray.
	VisitDebugArray(ctx *DebugArrayContext) interface{}

	// Visit a parse tree produced by ELParser#printStr.
	VisitPrintStr(ctx *PrintStrContext) interface{}

	// Visit a parse tree produced by ELParser#printBool.
	VisitPrintBool(ctx *PrintBoolContext) interface{}

	// Visit a parse tree produced by ELParser#printInt.
	VisitPrintInt(ctx *PrintIntContext) interface{}

	// Visit a parse tree produced by ELParser#printFloat.
	VisitPrintFloat(ctx *PrintFloatContext) interface{}

	// Visit a parse tree produced by ELParser#printEntity.
	VisitPrintEntity(ctx *PrintEntityContext) interface{}

	// Visit a parse tree produced by ELParser#printDate.
	VisitPrintDate(ctx *PrintDateContext) interface{}

	// Visit a parse tree produced by ELParser#printArray.
	VisitPrintArray(ctx *PrintArrayContext) interface{}

	// Visit a parse tree produced by ELParser#ifblock.
	VisitIfblock(ctx *IfblockContext) interface{}

	// Visit a parse tree produced by ELParser#ifEnd.
	VisitIfEnd(ctx *IfEndContext) interface{}

	// Visit a parse tree produced by ELParser#ifElse.
	VisitIfElse(ctx *IfElseContext) interface{}

	// Visit a parse tree produced by ELParser#ifElseIf.
	VisitIfElseIf(ctx *IfElseIfContext) interface{}

	// Visit a parse tree produced by ELParser#number.
	VisitNumber(ctx *NumberContext) interface{}

	// Visit a parse tree produced by ELParser#addDestArray2.
	VisitAddDestArray2(ctx *AddDestArray2Context) interface{}

	// Visit a parse tree produced by ELParser#addDestLong2.
	VisitAddDestLong2(ctx *AddDestLong2Context) interface{}

	// Visit a parse tree produced by ELParser#addDestDouble2.
	VisitAddDestDouble2(ctx *AddDestDouble2Context) interface{}

	// Visit a parse tree produced by ELParser#addDestArray.
	VisitAddDestArray(ctx *AddDestArrayContext) interface{}

	// Visit a parse tree produced by ELParser#addDestLong.
	VisitAddDestLong(ctx *AddDestLongContext) interface{}

	// Visit a parse tree produced by ELParser#addDestDouble.
	VisitAddDestDouble(ctx *AddDestDoubleContext) interface{}

	// Visit a parse tree produced by ELParser#addDestColon.
	VisitAddDestColon(ctx *AddDestColonContext) interface{}

	// Visit a parse tree produced by ELParser#addDestPossessiveLong.
	VisitAddDestPossessiveLong(ctx *AddDestPossessiveLongContext) interface{}

	// Visit a parse tree produced by ELParser#addDestPossessiveDouble.
	VisitAddDestPossessiveDouble(ctx *AddDestPossessiveDoubleContext) interface{}

	// Visit a parse tree produced by ELParser#subDestLong.
	VisitSubDestLong(ctx *SubDestLongContext) interface{}

	// Visit a parse tree produced by ELParser#subDestDouble.
	VisitSubDestDouble(ctx *SubDestDoubleContext) interface{}

	// Visit a parse tree produced by ELParser#subDestColon.
	VisitSubDestColon(ctx *SubDestColonContext) interface{}

	// Visit a parse tree produced by ELParser#subDestPossessiveLong.
	VisitSubDestPossessiveLong(ctx *SubDestPossessiveLongContext) interface{}

	// Visit a parse tree produced by ELParser#subDestPossessiveDouble.
	VisitSubDestPossessiveDouble(ctx *SubDestPossessiveDoubleContext) interface{}

	// Visit a parse tree produced by ELParser#addArrayNoMember.
	VisitAddArrayNoMember(ctx *AddArrayNoMemberContext) interface{}

	// Visit a parse tree produced by ELParser#addArrayToArray.
	VisitAddArrayToArray(ctx *AddArrayToArrayContext) interface{}

	// Visit a parse tree produced by ELParser#addEntityToDest.
	VisitAddEntityToDest(ctx *AddEntityToDestContext) interface{}

	// Visit a parse tree produced by ELParser#addEntityToDestDup.
	VisitAddEntityToDestDup(ctx *AddEntityToDestDupContext) interface{}

	// Visit a parse tree produced by ELParser#addStrToDest.
	VisitAddStrToDest(ctx *AddStrToDestContext) interface{}

	// Visit a parse tree produced by ELParser#addStrToDestDup.
	VisitAddStrToDestDup(ctx *AddStrToDestDupContext) interface{}

	// Visit a parse tree produced by ELParser#addDateToDest.
	VisitAddDateToDest(ctx *AddDateToDestContext) interface{}

	// Visit a parse tree produced by ELParser#addDateToDestDup.
	VisitAddDateToDestDup(ctx *AddDateToDestDupContext) interface{}

	// Visit a parse tree produced by ELParser#addNumToDest.
	VisitAddNumToDest(ctx *AddNumToDestContext) interface{}

	// Visit a parse tree produced by ELParser#addNumToDestDup.
	VisitAddNumToDestDup(ctx *AddNumToDestDupContext) interface{}

	// Visit a parse tree produced by ELParser#subtractNum.
	VisitSubtractNum(ctx *SubtractNumContext) interface{}

	// Visit a parse tree produced by ELParser#addEntityNoDups.
	VisitAddEntityNoDups(ctx *AddEntityNoDupsContext) interface{}

	// Visit a parse tree produced by ELParser#addEntityNoDupsDup.
	VisitAddEntityNoDupsDup(ctx *AddEntityNoDupsDupContext) interface{}

	// Visit a parse tree produced by ELParser#addStrNoDups.
	VisitAddStrNoDups(ctx *AddStrNoDupsContext) interface{}

	// Visit a parse tree produced by ELParser#addStrNoDupsDup.
	VisitAddStrNoDupsDup(ctx *AddStrNoDupsDupContext) interface{}

	// Visit a parse tree produced by ELParser#addToContextOf.
	VisitAddToContextOf(ctx *AddToContextOfContext) interface{}

	// Visit a parse tree produced by ELParser#addToContextFor.
	VisitAddToContextFor(ctx *AddToContextForContext) interface{}

	// Visit a parse tree produced by ELParser#clearstatement.
	VisitClearstatement(ctx *ClearstatementContext) interface{}

	// Visit a parse tree produced by ELParser#removeAtIndex.
	VisitRemoveAtIndex(ctx *RemoveAtIndexContext) interface{}

	// Visit a parse tree produced by ELParser#removeEachWhere.
	VisitRemoveEachWhere(ctx *RemoveEachWhereContext) interface{}

	// Visit a parse tree produced by ELParser#removeName.
	VisitRemoveName(ctx *RemoveNameContext) interface{}

	// Visit a parse tree produced by ELParser#removeString.
	VisitRemoveString(ctx *RemoveStringContext) interface{}

	// Visit a parse tree produced by ELParser#removeEntity.
	VisitRemoveEntity(ctx *RemoveEntityContext) interface{}

	// Visit a parse tree produced by ELParser#randomizeArray.
	VisitRandomizeArray(ctx *RandomizeArrayContext) interface{}

	// Visit a parse tree produced by ELParser#clearArray.
	VisitClearArray(ctx *ClearArrayContext) interface{}

	// Visit a parse tree produced by ELParser#sortAscending.
	VisitSortAscending(ctx *SortAscendingContext) interface{}

	// Visit a parse tree produced by ELParser#sortDescending.
	VisitSortDescending(ctx *SortDescendingContext) interface{}

	// Visit a parse tree produced by ELParser#opListStr.
	VisitOpListStr(ctx *OpListStrContext) interface{}

	// Visit a parse tree produced by ELParser#opListInt.
	VisitOpListInt(ctx *OpListIntContext) interface{}

	// Visit a parse tree produced by ELParser#opListFloat.
	VisitOpListFloat(ctx *OpListFloatContext) interface{}

	// Visit a parse tree produced by ELParser#opListEntity.
	VisitOpListEntity(ctx *OpListEntityContext) interface{}

	// Visit a parse tree produced by ELParser#opListStrSingle.
	VisitOpListStrSingle(ctx *OpListStrSingleContext) interface{}

	// Visit a parse tree produced by ELParser#opListIntSingle.
	VisitOpListIntSingle(ctx *OpListIntSingleContext) interface{}

	// Visit a parse tree produced by ELParser#opListFloatSingle.
	VisitOpListFloatSingle(ctx *OpListFloatSingleContext) interface{}

	// Visit a parse tree produced by ELParser#opListEntitySingle.
	VisitOpListEntitySingle(ctx *OpListEntitySingleContext) interface{}

	// Visit a parse tree produced by ELParser#operatorstatements.
	VisitOperatorstatements(ctx *OperatorstatementsContext) interface{}

	// Visit a parse tree produced by ELParser#xmlvalues.
	VisitXmlvalues(ctx *XmlvaluesContext) interface{}

	// Visit a parse tree produced by ELParser#xmlSetAttr.
	VisitXmlSetAttr(ctx *XmlSetAttrContext) interface{}

	// Visit a parse tree produced by ELParser#xmlSetAttrEntity.
	VisitXmlSetAttrEntity(ctx *XmlSetAttrEntityContext) interface{}

	// Visit a parse tree produced by ELParser#xmlAddAttr.
	VisitXmlAddAttr(ctx *XmlAddAttrContext) interface{}

	// Visit a parse tree produced by ELParser#xmlAddAttrEntity.
	VisitXmlAddAttrEntity(ctx *XmlAddAttrEntityContext) interface{}

	// Visit a parse tree produced by ELParser#arrayPolicyStatements.
	VisitArrayPolicyStatements(ctx *ArrayPolicyStatementsContext) interface{}

	// Visit a parse tree produced by ELParser#arrayColonRef.
	VisitArrayColonRef(ctx *ArrayColonRefContext) interface{}

	// Visit a parse tree produced by ELParser#arrayBase.
	VisitArrayBase(ctx *ArrayBaseContext) interface{}

	// Visit a parse tree produced by ELParser#arrayMap.
	VisitArrayMap(ctx *ArrayMapContext) interface{}

	// Visit a parse tree produced by ELParser#arrayParen.
	VisitArrayParen(ctx *ArrayParenContext) interface{}

	// Visit a parse tree produced by ELParser#arrayTyped.
	VisitArrayTyped(ctx *ArrayTypedContext) interface{}

	// Visit a parse tree produced by ELParser#arrayName.
	VisitArrayName(ctx *ArrayNameContext) interface{}

	// Visit a parse tree produced by ELParser#arrayCopy.
	VisitArrayCopy(ctx *ArrayCopyContext) interface{}

	// Visit a parse tree produced by ELParser#arrayCopySimple.
	VisitArrayCopySimple(ctx *ArrayCopySimpleContext) interface{}

	// Visit a parse tree produced by ELParser#arrayDeepCopy.
	VisitArrayDeepCopy(ctx *ArrayDeepCopyContext) interface{}

	// Visit a parse tree produced by ELParser#arrayDeepCopySimple.
	VisitArrayDeepCopySimple(ctx *ArrayDeepCopySimpleContext) interface{}

	// Visit a parse tree produced by ELParser#arrayLiteral.
	VisitArrayLiteral(ctx *ArrayLiteralContext) interface{}

	// Visit a parse tree produced by ELParser#arrayOfValues.
	VisitArrayOfValues(ctx *ArrayOfValuesContext) interface{}

	// Visit a parse tree produced by ELParser#arrayTokenize.
	VisitArrayTokenize(ctx *ArrayTokenizeContext) interface{}

	// Visit a parse tree produced by ELParser#arrayLit.
	VisitArrayLit(ctx *ArrayLitContext) interface{}

	// Visit a parse tree produced by ELParser#arrayListNameSingle.
	VisitArrayListNameSingle(ctx *ArrayListNameSingleContext) interface{}

	// Visit a parse tree produced by ELParser#arrayListArraySingle.
	VisitArrayListArraySingle(ctx *ArrayListArraySingleContext) interface{}

	// Visit a parse tree produced by ELParser#arrayListBoolSingle.
	VisitArrayListBoolSingle(ctx *ArrayListBoolSingleContext) interface{}

	// Visit a parse tree produced by ELParser#arrayListFloatSingle.
	VisitArrayListFloatSingle(ctx *ArrayListFloatSingleContext) interface{}

	// Visit a parse tree produced by ELParser#arrayListBool.
	VisitArrayListBool(ctx *ArrayListBoolContext) interface{}

	// Visit a parse tree produced by ELParser#arrayListInt.
	VisitArrayListInt(ctx *ArrayListIntContext) interface{}

	// Visit a parse tree produced by ELParser#arrayListFloat.
	VisitArrayListFloat(ctx *ArrayListFloatContext) interface{}

	// Visit a parse tree produced by ELParser#arrayListStr.
	VisitArrayListStr(ctx *ArrayListStrContext) interface{}

	// Visit a parse tree produced by ELParser#arrayListArray.
	VisitArrayListArray(ctx *ArrayListArrayContext) interface{}

	// Visit a parse tree produced by ELParser#arrayListIntSingle.
	VisitArrayListIntSingle(ctx *ArrayListIntSingleContext) interface{}

	// Visit a parse tree produced by ELParser#arrayListName.
	VisitArrayListName(ctx *ArrayListNameContext) interface{}

	// Visit a parse tree produced by ELParser#arrayListEntitySingle.
	VisitArrayListEntitySingle(ctx *ArrayListEntitySingleContext) interface{}

	// Visit a parse tree produced by ELParser#arrayListStrSingle.
	VisitArrayListStrSingle(ctx *ArrayListStrSingleContext) interface{}

	// Visit a parse tree produced by ELParser#arrayListEntity.
	VisitArrayListEntity(ctx *ArrayListEntityContext) interface{}

	// Visit a parse tree produced by ELParser#indxExpr.
	VisitIndxExpr(ctx *IndxExprContext) interface{}

	// Visit a parse tree produced by ELParser#entityTyped.
	VisitEntityTyped(ctx *EntityTypedContext) interface{}

	// Visit a parse tree produced by ELParser#entityParen.
	VisitEntityParen(ctx *EntityParenContext) interface{}

	// Visit a parse tree produced by ELParser#entityIndex.
	VisitEntityIndex(ctx *EntityIndexContext) interface{}

	// Visit a parse tree produced by ELParser#entityNewName.
	VisitEntityNewName(ctx *EntityNewNameContext) interface{}

	// Visit a parse tree produced by ELParser#entityNewTyped.
	VisitEntityNewTyped(ctx *EntityNewTypedContext) interface{}

	// Visit a parse tree produced by ELParser#entityClone.
	VisitEntityClone(ctx *EntityCloneContext) interface{}

	// Visit a parse tree produced by ELParser#entityColonRef.
	VisitEntityColonRef(ctx *EntityColonRefContext) interface{}

	// Visit a parse tree produced by ELParser#entityTableLookup.
	VisitEntityTableLookup(ctx *EntityTableLookupContext) interface{}

	// Visit a parse tree produced by ELParser#entityFirstIn.
	VisitEntityFirstIn(ctx *EntityFirstInContext) interface{}

	// Visit a parse tree produced by ELParser#entityFirst.
	VisitEntityFirst(ctx *EntityFirstContext) interface{}

	// Visit a parse tree produced by ELParser#entityRelationship.
	VisitEntityRelationship(ctx *EntityRelationshipContext) interface{}

	// Visit a parse tree produced by ELParser#dateSubYears.
	VisitDateSubYears(ctx *DateSubYearsContext) interface{}

	// Visit a parse tree produced by ELParser#dateSubMonths.
	VisitDateSubMonths(ctx *DateSubMonthsContext) interface{}

	// Visit a parse tree produced by ELParser#dateSubDays.
	VisitDateSubDays(ctx *DateSubDaysContext) interface{}

	// Visit a parse tree produced by ELParser#dateAddYears.
	VisitDateAddYears(ctx *DateAddYearsContext) interface{}

	// Visit a parse tree produced by ELParser#dateAddMonths.
	VisitDateAddMonths(ctx *DateAddMonthsContext) interface{}

	// Visit a parse tree produced by ELParser#dateAddDays.
	VisitDateAddDays(ctx *DateAddDaysContext) interface{}

	// Visit a parse tree produced by ELParser#dateFromStrFunc.
	VisitDateFromStrFunc(ctx *DateFromStrFuncContext) interface{}

	// Visit a parse tree produced by ELParser#dateFromStrCast.
	VisitDateFromStrCast(ctx *DateFromStrCastContext) interface{}

	// Visit a parse tree produced by ELParser#dateTableLookup.
	VisitDateTableLookup(ctx *DateTableLookupContext) interface{}

	// Visit a parse tree produced by ELParser#dateExprSubMonths.
	VisitDateExprSubMonths(ctx *DateExprSubMonthsContext) interface{}

	// Visit a parse tree produced by ELParser#datePlusYears.
	VisitDatePlusYears(ctx *DatePlusYearsContext) interface{}

	// Visit a parse tree produced by ELParser#dateMinusDays.
	VisitDateMinusDays(ctx *DateMinusDaysContext) interface{}

	// Visit a parse tree produced by ELParser#dateExprAddYears.
	VisitDateExprAddYears(ctx *DateExprAddYearsContext) interface{}

	// Visit a parse tree produced by ELParser#dateTyped.
	VisitDateTyped(ctx *DateTypedContext) interface{}

	// Visit a parse tree produced by ELParser#dateFirstOfYear.
	VisitDateFirstOfYear(ctx *DateFirstOfYearContext) interface{}

	// Visit a parse tree produced by ELParser#dateFirstOfMonth.
	VisitDateFirstOfMonth(ctx *DateFirstOfMonthContext) interface{}

	// Visit a parse tree produced by ELParser#dateExprSubYears.
	VisitDateExprSubYears(ctx *DateExprSubYearsContext) interface{}

	// Visit a parse tree produced by ELParser#dateCurrentDate.
	VisitDateCurrentDate(ctx *DateCurrentDateContext) interface{}

	// Visit a parse tree produced by ELParser#dateAdd.
	VisitDateAdd(ctx *DateAddContext) interface{}

	// Visit a parse tree produced by ELParser#dateFromIndex.
	VisitDateFromIndex(ctx *DateFromIndexContext) interface{}

	// Visit a parse tree produced by ELParser#dateExprAddDays.
	VisitDateExprAddDays(ctx *DateExprAddDaysContext) interface{}

	// Visit a parse tree produced by ELParser#dateMinusYears.
	VisitDateMinusYears(ctx *DateMinusYearsContext) interface{}

	// Visit a parse tree produced by ELParser#datePlusMonths.
	VisitDatePlusMonths(ctx *DatePlusMonthsContext) interface{}

	// Visit a parse tree produced by ELParser#dateMinusMonths.
	VisitDateMinusMonths(ctx *DateMinusMonthsContext) interface{}

	// Visit a parse tree produced by ELParser#dateEndOfMonth.
	VisitDateEndOfMonth(ctx *DateEndOfMonthContext) interface{}

	// Visit a parse tree produced by ELParser#dateUsing.
	VisitDateUsing(ctx *DateUsingContext) interface{}

	// Visit a parse tree produced by ELParser#dateExprAddMonths.
	VisitDateExprAddMonths(ctx *DateExprAddMonthsContext) interface{}

	// Visit a parse tree produced by ELParser#dateEarliestAfter.
	VisitDateEarliestAfter(ctx *DateEarliestAfterContext) interface{}

	// Visit a parse tree produced by ELParser#datePlusDays.
	VisitDatePlusDays(ctx *DatePlusDaysContext) interface{}

	// Visit a parse tree produced by ELParser#dateParen.
	VisitDateParen(ctx *DateParenContext) interface{}

	// Visit a parse tree produced by ELParser#dateColonRef.
	VisitDateColonRef(ctx *DateColonRefContext) interface{}

	// Visit a parse tree produced by ELParser#dateSub.
	VisitDateSub(ctx *DateSubContext) interface{}

	// Visit a parse tree produced by ELParser#dateDays.
	VisitDateDays(ctx *DateDaysContext) interface{}

	// Visit a parse tree produced by ELParser#dateExprSubDays.
	VisitDateExprSubDays(ctx *DateExprSubDaysContext) interface{}

	// Visit a parse tree produced by ELParser#dateFromArrayAt.
	VisitDateFromArrayAt(ctx *DateFromArrayAtContext) interface{}

	// Visit a parse tree produced by ELParser#nameTyped.
	VisitNameTyped(ctx *NameTypedContext) interface{}

	// Visit a parse tree produced by ELParser#nameOf.
	VisitNameOf(ctx *NameOfContext) interface{}

	// Visit a parse tree produced by ELParser#nameTheName.
	VisitNameTheName(ctx *NameTheNameContext) interface{}

	// Visit a parse tree produced by ELParser#nameArrayAt.
	VisitNameArrayAt(ctx *NameArrayAtContext) interface{}

	// Visit a parse tree produced by ELParser#nameLiteral.
	VisitNameLiteral(ctx *NameLiteralContext) interface{}

	// Visit a parse tree produced by ELParser#nameUsing.
	VisitNameUsing(ctx *NameUsingContext) interface{}

	// Visit a parse tree produced by ELParser#nameColonRef.
	VisitNameColonRef(ctx *NameColonRefContext) interface{}

	// Visit a parse tree produced by ELParser#nameFromStr.
	VisitNameFromStr(ctx *NameFromStrContext) interface{}

	// Visit a parse tree produced by ELParser#tableListMulti.
	VisitTableListMulti(ctx *TableListMultiContext) interface{}

	// Visit a parse tree produced by ELParser#tableListSingle.
	VisitTableListSingle(ctx *TableListSingleContext) interface{}

	// Visit a parse tree produced by ELParser#tableTyped.
	VisitTableTyped(ctx *TableTypedContext) interface{}

	// Visit a parse tree produced by ELParser#tableNew.
	VisitTableNew(ctx *TableNewContext) interface{}

	// Visit a parse tree produced by ELParser#strXmlValue.
	VisitStrXmlValue(ctx *StrXmlValueContext) interface{}

	// Visit a parse tree produced by ELParser#strToLower.
	VisitStrToLower(ctx *StrToLowerContext) interface{}

	// Visit a parse tree produced by ELParser#strXmlAttr.
	VisitStrXmlAttr(ctx *StrXmlAttrContext) interface{}

	// Visit a parse tree produced by ELParser#strParen.
	VisitStrParen(ctx *StrParenContext) interface{}

	// Visit a parse tree produced by ELParser#strRelationship.
	VisitStrRelationship(ctx *StrRelationshipContext) interface{}

	// Visit a parse tree produced by ELParser#strConcatInt.
	VisitStrConcatInt(ctx *StrConcatIntContext) interface{}

	// Visit a parse tree produced by ELParser#strSubstring.
	VisitStrSubstring(ctx *StrSubstringContext) interface{}

	// Visit a parse tree produced by ELParser#strConcat.
	VisitStrConcat(ctx *StrConcatContext) interface{}

	// Visit a parse tree produced by ELParser#strConcatEntity.
	VisitStrConcatEntity(ctx *StrConcatEntityContext) interface{}

	// Visit a parse tree produced by ELParser#strValueOfOp.
	VisitStrValueOfOp(ctx *StrValueOfOpContext) interface{}

	// Visit a parse tree produced by ELParser#strHexOfBytes.
	VisitStrHexOfBytes(ctx *StrHexOfBytesContext) interface{}

	// Visit a parse tree produced by ELParser#strConcatDate.
	VisitStrConcatDate(ctx *StrConcatDateContext) interface{}

	// Visit a parse tree produced by ELParser#strValueOfFloat.
	VisitStrValueOfFloat(ctx *StrValueOfFloatContext) interface{}

	// Visit a parse tree produced by ELParser#strValueOfInt.
	VisitStrValueOfInt(ctx *StrValueOfIntContext) interface{}

	// Visit a parse tree produced by ELParser#strColonRef.
	VisitStrColonRef(ctx *StrColonRefContext) interface{}

	// Visit a parse tree produced by ELParser#strLiteral.
	VisitStrLiteral(ctx *StrLiteralContext) interface{}

	// Visit a parse tree produced by ELParser#strConcatInvalid.
	VisitStrConcatInvalid(ctx *StrConcatInvalidContext) interface{}

	// Visit a parse tree produced by ELParser#strMappingKey.
	VisitStrMappingKey(ctx *StrMappingKeyContext) interface{}

	// Visit a parse tree produced by ELParser#strTableInfo.
	VisitStrTableInfo(ctx *StrTableInfoContext) interface{}

	// Visit a parse tree produced by ELParser#strTyped.
	VisitStrTyped(ctx *StrTypedContext) interface{}

	// Visit a parse tree produced by ELParser#strConcatNull.
	VisitStrConcatNull(ctx *StrConcatNullContext) interface{}

	// Visit a parse tree produced by ELParser#strAttrOf.
	VisitStrAttrOf(ctx *StrAttrOfContext) interface{}

	// Visit a parse tree produced by ELParser#strValueOfDate.
	VisitStrValueOfDate(ctx *StrValueOfDateContext) interface{}

	// Visit a parse tree produced by ELParser#strToUpper.
	VisitStrToUpper(ctx *StrToUpperContext) interface{}

	// Visit a parse tree produced by ELParser#strBase58CheckOfBytes.
	VisitStrBase58CheckOfBytes(ctx *StrBase58CheckOfBytesContext) interface{}

	// Visit a parse tree produced by ELParser#strValueOfBool.
	VisitStrValueOfBool(ctx *StrValueOfBoolContext) interface{}

	// Visit a parse tree produced by ELParser#strBech32OfBytes.
	VisitStrBech32OfBytes(ctx *StrBech32OfBytesContext) interface{}

	// Visit a parse tree produced by ELParser#strConcatFloat.
	VisitStrConcatFloat(ctx *StrConcatFloatContext) interface{}

	// Visit a parse tree produced by ELParser#strTableLookup.
	VisitStrTableLookup(ctx *StrTableLookupContext) interface{}

	// Visit a parse tree produced by ELParser#strUsing.
	VisitStrUsing(ctx *StrUsingContext) interface{}

	// Visit a parse tree produced by ELParser#strConcatArray.
	VisitStrConcatArray(ctx *StrConcatArrayContext) interface{}

	// Visit a parse tree produced by ELParser#strTimestamp.
	VisitStrTimestamp(ctx *StrTimestampContext) interface{}

	// Visit a parse tree produced by ELParser#strFromIndex.
	VisitStrFromIndex(ctx *StrFromIndexContext) interface{}

	// Visit a parse tree produced by ELParser#strTrim.
	VisitStrTrim(ctx *StrTrimContext) interface{}

	// Visit a parse tree produced by ELParser#strConcatName.
	VisitStrConcatName(ctx *StrConcatNameContext) interface{}

	// Visit a parse tree produced by ELParser#floatMinOfFloat.
	VisitFloatMinOfFloat(ctx *FloatMinOfFloatContext) interface{}

	// Visit a parse tree produced by ELParser#floatMaxIntOf.
	VisitFloatMaxIntOf(ctx *FloatMaxIntOfContext) interface{}

	// Visit a parse tree produced by ELParser#floatAddFloat.
	VisitFloatAddFloat(ctx *FloatAddFloatContext) interface{}

	// Visit a parse tree produced by ELParser#floatParen.
	VisitFloatParen(ctx *FloatParenContext) interface{}

	// Visit a parse tree produced by ELParser#floatMulFloat.
	VisitFloatMulFloat(ctx *FloatMulFloatContext) interface{}

	// Visit a parse tree produced by ELParser#floatMaxOfFloat.
	VisitFloatMaxOfFloat(ctx *FloatMaxOfFloatContext) interface{}

	// Visit a parse tree produced by ELParser#floatDivFloat.
	VisitFloatDivFloat(ctx *FloatDivFloatContext) interface{}

	// Visit a parse tree produced by ELParser#floatValueOfOp.
	VisitFloatValueOfOp(ctx *FloatValueOfOpContext) interface{}

	// Visit a parse tree produced by ELParser#floatRoundedTo.
	VisitFloatRoundedTo(ctx *FloatRoundedToContext) interface{}

	// Visit a parse tree produced by ELParser#floatMinIntOf.
	VisitFloatMinIntOf(ctx *FloatMinIntOfContext) interface{}

	// Visit a parse tree produced by ELParser#floatAddInt.
	VisitFloatAddInt(ctx *FloatAddIntContext) interface{}

	// Visit a parse tree produced by ELParser#floatMinOfIntComma.
	VisitFloatMinOfIntComma(ctx *FloatMinOfIntCommaContext) interface{}

	// Visit a parse tree produced by ELParser#floatTableLookup.
	VisitFloatTableLookup(ctx *FloatTableLookupContext) interface{}

	// Visit a parse tree produced by ELParser#floatSubFloat.
	VisitFloatSubFloat(ctx *FloatSubFloatContext) interface{}

	// Visit a parse tree produced by ELParser#floatMinIntOfComma.
	VisitFloatMinIntOfComma(ctx *FloatMinIntOfCommaContext) interface{}

	// Visit a parse tree produced by ELParser#floatLiteral.
	VisitFloatLiteral(ctx *FloatLiteralContext) interface{}

	// Visit a parse tree produced by ELParser#floatMulBy.
	VisitFloatMulBy(ctx *FloatMulByContext) interface{}

	// Visit a parse tree produced by ELParser#floatMaxOfIntComma.
	VisitFloatMaxOfIntComma(ctx *FloatMaxOfIntCommaContext) interface{}

	// Visit a parse tree produced by ELParser#floatUsing.
	VisitFloatUsing(ctx *FloatUsingContext) interface{}

	// Visit a parse tree produced by ELParser#intDivFloat.
	VisitIntDivFloat(ctx *IntDivFloatContext) interface{}

	// Visit a parse tree produced by ELParser#intAddFloat.
	VisitIntAddFloat(ctx *IntAddFloatContext) interface{}

	// Visit a parse tree produced by ELParser#floatTyped.
	VisitFloatTyped(ctx *FloatTypedContext) interface{}

	// Visit a parse tree produced by ELParser#floatSubInt.
	VisitFloatSubInt(ctx *FloatSubIntContext) interface{}

	// Visit a parse tree produced by ELParser#floatDivInt.
	VisitFloatDivInt(ctx *FloatDivIntContext) interface{}

	// Visit a parse tree produced by ELParser#floatSubFrom.
	VisitFloatSubFrom(ctx *FloatSubFromContext) interface{}

	// Visit a parse tree produced by ELParser#intSubFloat.
	VisitIntSubFloat(ctx *IntSubFloatContext) interface{}

	// Visit a parse tree produced by ELParser#intMulFloat.
	VisitIntMulFloat(ctx *IntMulFloatContext) interface{}

	// Visit a parse tree produced by ELParser#floatDivBy.
	VisitFloatDivBy(ctx *FloatDivByContext) interface{}

	// Visit a parse tree produced by ELParser#floatMinOfInt.
	VisitFloatMinOfInt(ctx *FloatMinOfIntContext) interface{}

	// Visit a parse tree produced by ELParser#floatFromIndex.
	VisitFloatFromIndex(ctx *FloatFromIndexContext) interface{}

	// Visit a parse tree produced by ELParser#floatMaxIntOfComma.
	VisitFloatMaxIntOfComma(ctx *FloatMaxIntOfCommaContext) interface{}

	// Visit a parse tree produced by ELParser#floatRounded.
	VisitFloatRounded(ctx *FloatRoundedContext) interface{}

	// Visit a parse tree produced by ELParser#floatRoundedBoundry.
	VisitFloatRoundedBoundry(ctx *FloatRoundedBoundryContext) interface{}

	// Visit a parse tree produced by ELParser#floatColonRef.
	VisitFloatColonRef(ctx *FloatColonRefContext) interface{}

	// Visit a parse tree produced by ELParser#floatMulInt.
	VisitFloatMulInt(ctx *FloatMulIntContext) interface{}

	// Visit a parse tree produced by ELParser#floatFromInt.
	VisitFloatFromInt(ctx *FloatFromIntContext) interface{}

	// Visit a parse tree produced by ELParser#floatAddTo.
	VisitFloatAddTo(ctx *FloatAddToContext) interface{}

	// Visit a parse tree produced by ELParser#floatFromStr.
	VisitFloatFromStr(ctx *FloatFromStrContext) interface{}

	// Visit a parse tree produced by ELParser#floatMinOfFloatComma.
	VisitFloatMinOfFloatComma(ctx *FloatMinOfFloatCommaContext) interface{}

	// Visit a parse tree produced by ELParser#floatAbs.
	VisitFloatAbs(ctx *FloatAbsContext) interface{}

	// Visit a parse tree produced by ELParser#floatMaxOfFloatComma.
	VisitFloatMaxOfFloatComma(ctx *FloatMaxOfFloatCommaContext) interface{}

	// Visit a parse tree produced by ELParser#floatNegate.
	VisitFloatNegate(ctx *FloatNegateContext) interface{}

	// Visit a parse tree produced by ELParser#floatMaxOfInt.
	VisitFloatMaxOfInt(ctx *FloatMaxOfIntContext) interface{}

	// Visit a parse tree produced by ELParser#floatSumOf.
	VisitFloatSumOf(ctx *FloatSumOfContext) interface{}

	// Visit a parse tree produced by ELParser#intDaysInYear.
	VisitIntDaysInYear(ctx *IntDaysInYearContext) interface{}

	// Visit a parse tree produced by ELParser#intYearOf.
	VisitIntYearOf(ctx *IntYearOfContext) interface{}

	// Visit a parse tree produced by ELParser#intAdd.
	VisitIntAdd(ctx *IntAddContext) interface{}

	// Visit a parse tree produced by ELParser#intIndexOf.
	VisitIntIndexOf(ctx *IntIndexOfContext) interface{}

	// Visit a parse tree produced by ELParser#intMinOfComma.
	VisitIntMinOfComma(ctx *IntMinOfCommaContext) interface{}

	// Visit a parse tree produced by ELParser#intNumberOf.
	VisitIntNumberOf(ctx *IntNumberOfContext) interface{}

	// Visit a parse tree produced by ELParser#intNumberOfWhere.
	VisitIntNumberOfWhere(ctx *IntNumberOfWhereContext) interface{}

	// Visit a parse tree produced by ELParser#intMaxOf.
	VisitIntMaxOf(ctx *IntMaxOfContext) interface{}

	// Visit a parse tree produced by ELParser#fixedLiteral.
	VisitFixedLiteral(ctx *FixedLiteralContext) interface{}

	// Visit a parse tree produced by ELParser#intParen.
	VisitIntParen(ctx *IntParenContext) interface{}

	// Visit a parse tree produced by ELParser#intFromNumber.
	VisitIntFromNumber(ctx *IntFromNumberContext) interface{}

	// Visit a parse tree produced by ELParser#intUsing.
	VisitIntUsing(ctx *IntUsingContext) interface{}

	// Visit a parse tree produced by ELParser#intYearsBetween.
	VisitIntYearsBetween(ctx *IntYearsBetweenContext) interface{}

	// Visit a parse tree produced by ELParser#intMaxOfComma.
	VisitIntMaxOfComma(ctx *IntMaxOfCommaContext) interface{}

	// Visit a parse tree produced by ELParser#intValueOfOp.
	VisitIntValueOfOp(ctx *IntValueOfOpContext) interface{}

	// Visit a parse tree produced by ELParser#intTableLookup.
	VisitIntTableLookup(ctx *IntTableLookupContext) interface{}

	// Visit a parse tree produced by ELParser#fixedFromStr.
	VisitFixedFromStr(ctx *FixedFromStrContext) interface{}

	// Visit a parse tree produced by ELParser#intSumOf.
	VisitIntSumOf(ctx *IntSumOfContext) interface{}

	// Visit a parse tree produced by ELParser#intDiv.
	VisitIntDiv(ctx *IntDivContext) interface{}

	// Visit a parse tree produced by ELParser#intLengthArray.
	VisitIntLengthArray(ctx *IntLengthArrayContext) interface{}

	// Visit a parse tree produced by ELParser#fixedFromNumber.
	VisitFixedFromNumber(ctx *FixedFromNumberContext) interface{}

	// Visit a parse tree produced by ELParser#fixedFromFloat.
	VisitFixedFromFloat(ctx *FixedFromFloatContext) interface{}

	// Visit a parse tree produced by ELParser#fixedFromIndex.
	VisitFixedFromIndex(ctx *FixedFromIndexContext) interface{}

	// Visit a parse tree produced by ELParser#intDayOfMonth.
	VisitIntDayOfMonth(ctx *IntDayOfMonthContext) interface{}

	// Visit a parse tree produced by ELParser#intSub.
	VisitIntSub(ctx *IntSubContext) interface{}

	// Visit a parse tree produced by ELParser#intMulBy.
	VisitIntMulBy(ctx *IntMulByContext) interface{}

	// Visit a parse tree produced by ELParser#intFromStr.
	VisitIntFromStr(ctx *IntFromStrContext) interface{}

	// Visit a parse tree produced by ELParser#intMul.
	VisitIntMul(ctx *IntMulContext) interface{}

	// Visit a parse tree produced by ELParser#intDaysBetween.
	VisitIntDaysBetween(ctx *IntDaysBetweenContext) interface{}

	// Visit a parse tree produced by ELParser#intLiteral.
	VisitIntLiteral(ctx *IntLiteralContext) interface{}

	// Visit a parse tree produced by ELParser#intTyped.
	VisitIntTyped(ctx *IntTypedContext) interface{}

	// Visit a parse tree produced by ELParser#intFromIndex.
	VisitIntFromIndex(ctx *IntFromIndexContext) interface{}

	// Visit a parse tree produced by ELParser#intLengthStr.
	VisitIntLengthStr(ctx *IntLengthStrContext) interface{}

	// Visit a parse tree produced by ELParser#intUsingArray.
	VisitIntUsingArray(ctx *IntUsingArrayContext) interface{}

	// Visit a parse tree produced by ELParser#intNegate.
	VisitIntNegate(ctx *IntNegateContext) interface{}

	// Visit a parse tree produced by ELParser#intAddTo.
	VisitIntAddTo(ctx *IntAddToContext) interface{}

	// Visit a parse tree produced by ELParser#intDivBy.
	VisitIntDivBy(ctx *IntDivByContext) interface{}

	// Visit a parse tree produced by ELParser#intAbs.
	VisitIntAbs(ctx *IntAbsContext) interface{}

	// Visit a parse tree produced by ELParser#intMinOf.
	VisitIntMinOf(ctx *IntMinOfContext) interface{}

	// Visit a parse tree produced by ELParser#intDaysInMonth.
	VisitIntDaysInMonth(ctx *IntDaysInMonthContext) interface{}

	// Visit a parse tree produced by ELParser#intColonRef.
	VisitIntColonRef(ctx *IntColonRefContext) interface{}

	// Visit a parse tree produced by ELParser#intBytesIndex.
	VisitIntBytesIndex(ctx *IntBytesIndexContext) interface{}

	// Visit a parse tree produced by ELParser#intSubFrom.
	VisitIntSubFrom(ctx *IntSubFromContext) interface{}

	// Visit a parse tree produced by ELParser#intMonthsBetween.
	VisitIntMonthsBetween(ctx *IntMonthsBetweenContext) interface{}

	// Visit a parse tree produced by ELParser#intLengthBytes.
	VisitIntLengthBytes(ctx *IntLengthBytesContext) interface{}

	// Visit a parse tree produced by ELParser#bigAbs.
	VisitBigAbs(ctx *BigAbsContext) interface{}

	// Visit a parse tree produced by ELParser#bigDiv.
	VisitBigDiv(ctx *BigDivContext) interface{}

	// Visit a parse tree produced by ELParser#bigColonRef.
	VisitBigColonRef(ctx *BigColonRefContext) interface{}

	// Visit a parse tree produced by ELParser#bigFromBytes.
	VisitBigFromBytes(ctx *BigFromBytesContext) interface{}

	// Visit a parse tree produced by ELParser#bigFromFloat.
	VisitBigFromFloat(ctx *BigFromFloatContext) interface{}

	// Visit a parse tree produced by ELParser#bigNegate.
	VisitBigNegate(ctx *BigNegateContext) interface{}

	// Visit a parse tree produced by ELParser#bigUsing.
	VisitBigUsing(ctx *BigUsingContext) interface{}

	// Visit a parse tree produced by ELParser#bigSub.
	VisitBigSub(ctx *BigSubContext) interface{}

	// Visit a parse tree produced by ELParser#bigParen.
	VisitBigParen(ctx *BigParenContext) interface{}

	// Visit a parse tree produced by ELParser#bigAdd.
	VisitBigAdd(ctx *BigAddContext) interface{}

	// Visit a parse tree produced by ELParser#bigFromStr.
	VisitBigFromStr(ctx *BigFromStrContext) interface{}

	// Visit a parse tree produced by ELParser#bigFromInt.
	VisitBigFromInt(ctx *BigFromIntContext) interface{}

	// Visit a parse tree produced by ELParser#bigMul.
	VisitBigMul(ctx *BigMulContext) interface{}

	// Visit a parse tree produced by ELParser#bigTyped.
	VisitBigTyped(ctx *BigTypedContext) interface{}

	// Visit a parse tree produced by ELParser#bytesSha256.
	VisitBytesSha256(ctx *BytesSha256Context) interface{}

	// Visit a parse tree produced by ELParser#bytesLiteral.
	VisitBytesLiteral(ctx *BytesLiteralContext) interface{}

	// Visit a parse tree produced by ELParser#bytesCvBase58Check.
	VisitBytesCvBase58Check(ctx *BytesCvBase58CheckContext) interface{}

	// Visit a parse tree produced by ELParser#bytesCvBech32.
	VisitBytesCvBech32(ctx *BytesCvBech32Context) interface{}

	// Visit a parse tree produced by ELParser#bytesRipemd160.
	VisitBytesRipemd160(ctx *BytesRipemd160Context) interface{}

	// Visit a parse tree produced by ELParser#bytesColonRef.
	VisitBytesColonRef(ctx *BytesColonRefContext) interface{}

	// Visit a parse tree produced by ELParser#bytesCvHex.
	VisitBytesCvHex(ctx *BytesCvHexContext) interface{}

	// Visit a parse tree produced by ELParser#bytesCvBigInt.
	VisitBytesCvBigInt(ctx *BytesCvBigIntContext) interface{}

	// Visit a parse tree produced by ELParser#bytesSlice.
	VisitBytesSlice(ctx *BytesSliceContext) interface{}

	// Visit a parse tree produced by ELParser#bytesConcat.
	VisitBytesConcat(ctx *BytesConcatContext) interface{}

	// Visit a parse tree produced by ELParser#bytesKeccak256.
	VisitBytesKeccak256(ctx *BytesKeccak256Context) interface{}

	// Visit a parse tree produced by ELParser#bytesTyped.
	VisitBytesTyped(ctx *BytesTypedContext) interface{}

	// Visit a parse tree produced by ELParser#bytesParen.
	VisitBytesParen(ctx *BytesParenContext) interface{}

	// Visit a parse tree produced by ELParser#bytesSha3.
	VisitBytesSha3(ctx *BytesSha3Context) interface{}

	// Visit a parse tree produced by ELParser#includeNumber.
	VisitIncludeNumber(ctx *IncludeNumberContext) interface{}

	// Visit a parse tree produced by ELParser#includeDate.
	VisitIncludeDate(ctx *IncludeDateContext) interface{}

	// Visit a parse tree produced by ELParser#includeEntity.
	VisitIncludeEntity(ctx *IncludeEntityContext) interface{}

	// Visit a parse tree produced by ELParser#includeString.
	VisitIncludeString(ctx *IncludeStringContext) interface{}

	// Visit a parse tree produced by ELParser#inthe.
	VisitInthe(ctx *IntheContext) interface{}

	// Visit a parse tree produced by ELParser#thereis.
	VisitThereis(ctx *ThereisContext) interface{}

	// Visit a parse tree produced by ELParser#blistMulti.
	VisitBlistMulti(ctx *BlistMultiContext) interface{}

	// Visit a parse tree produced by ELParser#blistOr.
	VisitBlistOr(ctx *BlistOrContext) interface{}

	// Visit a parse tree produced by ELParser#blistIcMulti.
	VisitBlistIcMulti(ctx *BlistIcMultiContext) interface{}

	// Visit a parse tree produced by ELParser#blistIcOr.
	VisitBlistIcOr(ctx *BlistIcOrContext) interface{}

	// Visit a parse tree produced by ELParser#boolIntLteFloat.
	VisitBoolIntLteFloat(ctx *BoolIntLteFloatContext) interface{}

	// Visit a parse tree produced by ELParser#boolFloatLteInt.
	VisitBoolFloatLteInt(ctx *BoolFloatLteIntContext) interface{}

	// Visit a parse tree produced by ELParser#boolFromStr.
	VisitBoolFromStr(ctx *BoolFromStrContext) interface{}

	// Visit a parse tree produced by ELParser#boolNumIsNull.
	VisitBoolNumIsNull(ctx *BoolNumIsNullContext) interface{}

	// Visit a parse tree produced by ELParser#boolEntityIsOf.
	VisitBoolEntityIsOf(ctx *BoolEntityIsOfContext) interface{}

	// Visit a parse tree produced by ELParser#boolTypedIsLiteral.
	VisitBoolTypedIsLiteral(ctx *BoolTypedIsLiteralContext) interface{}

	// Visit a parse tree produced by ELParser#boolDateGte.
	VisitBoolDateGte(ctx *BoolDateGteContext) interface{}

	// Visit a parse tree produced by ELParser#boolNameEq.
	VisitBoolNameEq(ctx *BoolNameEqContext) interface{}

	// Visit a parse tree produced by ELParser#boolStartsWith.
	VisitBoolStartsWith(ctx *BoolStartsWithContext) interface{}

	// Visit a parse tree produced by ELParser#boolThereIsNoInEntityWhere.
	VisitBoolThereIsNoInEntityWhere(ctx *BoolThereIsNoInEntityWhereContext) interface{}

	// Visit a parse tree produced by ELParser#boolBigLt.
	VisitBoolBigLt(ctx *BoolBigLtContext) interface{}

	// Visit a parse tree produced by ELParser#boolArrayIsNull.
	VisitBoolArrayIsNull(ctx *BoolArrayIsNullContext) interface{}

	// Visit a parse tree produced by ELParser#boolIntEq.
	VisitBoolIntEq(ctx *BoolIntEqContext) interface{}

	// Visit a parse tree produced by ELParser#boolEntityHasaWhere.
	VisitBoolEntityHasaWhere(ctx *BoolEntityHasaWhereContext) interface{}

	// Visit a parse tree produced by ELParser#boolStrEqList.
	VisitBoolStrEqList(ctx *BoolStrEqListContext) interface{}

	// Visit a parse tree produced by ELParser#boolBigNeq.
	VisitBoolBigNeq(ctx *BoolBigNeqContext) interface{}

	// Visit a parse tree produced by ELParser#boolIntLt.
	VisitBoolIntLt(ctx *BoolIntLtContext) interface{}

	// Visit a parse tree produced by ELParser#boolIntEqFloat.
	VisitBoolIntEqFloat(ctx *BoolIntEqFloatContext) interface{}

	// Visit a parse tree produced by ELParser#boolFromIndex.
	VisitBoolFromIndex(ctx *BoolFromIndexContext) interface{}

	// Visit a parse tree produced by ELParser#boolFloatNeq.
	VisitBoolFloatNeq(ctx *BoolFloatNeqContext) interface{}

	// Visit a parse tree produced by ELParser#boolFloatLt.
	VisitBoolFloatLt(ctx *BoolFloatLtContext) interface{}

	// Visit a parse tree produced by ELParser#boolThereIsWhere.
	VisitBoolThereIsWhere(ctx *BoolThereIsWhereContext) interface{}

	// Visit a parse tree produced by ELParser#boolBoolNeq.
	VisitBoolBoolNeq(ctx *BoolBoolNeqContext) interface{}

	// Visit a parse tree produced by ELParser#boolOneOfHasa.
	VisitBoolOneOfHasa(ctx *BoolOneOfHasaContext) interface{}

	// Visit a parse tree produced by ELParser#boolStrIsOneOf.
	VisitBoolStrIsOneOf(ctx *BoolStrIsOneOfContext) interface{}

	// Visit a parse tree produced by ELParser#boolNameNeq.
	VisitBoolNameNeq(ctx *BoolNameNeqContext) interface{}

	// Visit a parse tree produced by ELParser#boolColonIsNotLiteral.
	VisitBoolColonIsNotLiteral(ctx *BoolColonIsNotLiteralContext) interface{}

	// Visit a parse tree produced by ELParser#boolThereIsNoInArrayWhere.
	VisitBoolThereIsNoInArrayWhere(ctx *BoolThereIsNoInArrayWhereContext) interface{}

	// Visit a parse tree produced by ELParser#boolStrIsNotOneOf.
	VisitBoolStrIsNotOneOf(ctx *BoolStrIsNotOneOfContext) interface{}

	// Visit a parse tree produced by ELParser#boolWasQuestion.
	VisitBoolWasQuestion(ctx *BoolWasQuestionContext) interface{}

	// Visit a parse tree produced by ELParser#boolIntGteFloat.
	VisitBoolIntGteFloat(ctx *BoolIntGteFloatContext) interface{}

	// Visit a parse tree produced by ELParser#boolNameNeqStr.
	VisitBoolNameNeqStr(ctx *BoolNameNeqStrContext) interface{}

	// Visit a parse tree produced by ELParser#boolDateIsNull.
	VisitBoolDateIsNull(ctx *BoolDateIsNullContext) interface{}

	// Visit a parse tree produced by ELParser#boolEntityInContext.
	VisitBoolEntityInContext(ctx *BoolEntityInContextContext) interface{}

	// Visit a parse tree produced by ELParser#boolDateAfter.
	VisitBoolDateAfter(ctx *BoolDateAfterContext) interface{}

	// Visit a parse tree produced by ELParser#boolBytesNeq.
	VisitBoolBytesNeq(ctx *BoolBytesNeqContext) interface{}

	// Visit a parse tree produced by ELParser#boolDateLt.
	VisitBoolDateLt(ctx *BoolDateLtContext) interface{}

	// Visit a parse tree produced by ELParser#boolStrEntityInContext.
	VisitBoolStrEntityInContext(ctx *BoolStrEntityInContextContext) interface{}

	// Visit a parse tree produced by ELParser#boolFloatEq.
	VisitBoolFloatEq(ctx *BoolFloatEqContext) interface{}

	// Visit a parse tree produced by ELParser#boolDateLte.
	VisitBoolDateLte(ctx *BoolDateLteContext) interface{}

	// Visit a parse tree produced by ELParser#boolFloatGtInt.
	VisitBoolFloatGtInt(ctx *BoolFloatGtIntContext) interface{}

	// Visit a parse tree produced by ELParser#boolLiteral.
	VisitBoolLiteral(ctx *BoolLiteralContext) interface{}

	// Visit a parse tree produced by ELParser#boolEntityIsNull.
	VisitBoolEntityIsNull(ctx *BoolEntityIsNullContext) interface{}

	// Visit a parse tree produced by ELParser#boolStrEq.
	VisitBoolStrEq(ctx *BoolStrEqContext) interface{}

	// Visit a parse tree produced by ELParser#boolEntityNeq.
	VisitBoolEntityNeq(ctx *BoolEntityNeqContext) interface{}

	// Visit a parse tree produced by ELParser#boolIntGte.
	VisitBoolIntGte(ctx *BoolIntGteContext) interface{}

	// Visit a parse tree produced by ELParser#boolDoesQuestion.
	VisitBoolDoesQuestion(ctx *BoolDoesQuestionContext) interface{}

	// Visit a parse tree produced by ELParser#boolNot.
	VisitBoolNot(ctx *BoolNotContext) interface{}

	// Visit a parse tree produced by ELParser#boolStrIsNotNull.
	VisitBoolStrIsNotNull(ctx *BoolStrIsNotNullContext) interface{}

	// Visit a parse tree produced by ELParser#boolAnd.
	VisitBoolAnd(ctx *BoolAndContext) interface{}

	// Visit a parse tree produced by ELParser#boolBytesEq.
	VisitBoolBytesEq(ctx *BoolBytesEqContext) interface{}

	// Visit a parse tree produced by ELParser#boolStrIsNot.
	VisitBoolStrIsNot(ctx *BoolStrIsNotContext) interface{}

	// Visit a parse tree produced by ELParser#boolIntGt.
	VisitBoolIntGt(ctx *BoolIntGtContext) interface{}

	// Visit a parse tree produced by ELParser#boolFloatLte.
	VisitBoolFloatLte(ctx *BoolFloatLteContext) interface{}

	// Visit a parse tree produced by ELParser#boolBigLte.
	VisitBoolBigLte(ctx *BoolBigLteContext) interface{}

	// Visit a parse tree produced by ELParser#boolStrEqIc.
	VisitBoolStrEqIc(ctx *BoolStrEqIcContext) interface{}

	// Visit a parse tree produced by ELParser#boolTyped.
	VisitBoolTyped(ctx *BoolTypedContext) interface{}

	// Visit a parse tree produced by ELParser#boolUsing.
	VisitBoolUsing(ctx *BoolUsingContext) interface{}

	// Visit a parse tree produced by ELParser#boolEntityNotInContext.
	VisitBoolEntityNotInContext(ctx *BoolEntityNotInContextContext) interface{}

	// Visit a parse tree produced by ELParser#boolStrLt.
	VisitBoolStrLt(ctx *BoolStrLtContext) interface{}

	// Visit a parse tree produced by ELParser#boolStrGte.
	VisitBoolStrGte(ctx *BoolStrGteContext) interface{}

	// Visit a parse tree produced by ELParser#boolStrEntityNotInContext.
	VisitBoolStrEntityNotInContext(ctx *BoolStrEntityNotInContextContext) interface{}

	// Visit a parse tree produced by ELParser#boolArrayDoesInclude.
	VisitBoolArrayDoesInclude(ctx *BoolArrayDoesIncludeContext) interface{}

	// Visit a parse tree produced by ELParser#boolIntGtFloat.
	VisitBoolIntGtFloat(ctx *BoolIntGtFloatContext) interface{}

	// Visit a parse tree produced by ELParser#boolValueOfOp.
	VisitBoolValueOfOp(ctx *BoolValueOfOpContext) interface{}

	// Visit a parse tree produced by ELParser#boolColonRef.
	VisitBoolColonRef(ctx *BoolColonRefContext) interface{}

	// Visit a parse tree produced by ELParser#boolBigGt.
	VisitBoolBigGt(ctx *BoolBigGtContext) interface{}

	// Visit a parse tree produced by ELParser#boolFloatGt.
	VisitBoolFloatGt(ctx *BoolFloatGtContext) interface{}

	// Visit a parse tree produced by ELParser#boolStrIsNull.
	VisitBoolStrIsNull(ctx *BoolStrIsNullContext) interface{}

	// Visit a parse tree produced by ELParser#boolStrGt.
	VisitBoolStrGt(ctx *BoolStrGtContext) interface{}

	// Visit a parse tree produced by ELParser#boolColonIsLiteral.
	VisitBoolColonIsLiteral(ctx *BoolColonIsLiteralContext) interface{}

	// Visit a parse tree produced by ELParser#boolEntityEq.
	VisitBoolEntityEq(ctx *BoolEntityEqContext) interface{}

	// Visit a parse tree produced by ELParser#boolNumIsNotNull.
	VisitBoolNumIsNotNull(ctx *BoolNumIsNotNullContext) interface{}

	// Visit a parse tree produced by ELParser#boolStartsWithAt.
	VisitBoolStartsWithAt(ctx *BoolStartsWithAtContext) interface{}

	// Visit a parse tree produced by ELParser#boolMatches.
	VisitBoolMatches(ctx *BoolMatchesContext) interface{}

	// Visit a parse tree produced by ELParser#boolFloatGteInt.
	VisitBoolFloatGteInt(ctx *BoolFloatGteIntContext) interface{}

	// Visit a parse tree produced by ELParser#boolStrNeqIc.
	VisitBoolStrNeqIc(ctx *BoolStrNeqIcContext) interface{}

	// Visit a parse tree produced by ELParser#boolArrayIsNotNull.
	VisitBoolArrayIsNotNull(ctx *BoolArrayIsNotNullContext) interface{}

	// Visit a parse tree produced by ELParser#boolDateBetween.
	VisitBoolDateBetween(ctx *BoolDateBetweenContext) interface{}

	// Visit a parse tree produced by ELParser#boolBexprIsNotNull.
	VisitBoolBexprIsNotNull(ctx *BoolBexprIsNotNullContext) interface{}

	// Visit a parse tree produced by ELParser#boolIntLte.
	VisitBoolIntLte(ctx *BoolIntLteContext) interface{}

	// Visit a parse tree produced by ELParser#boolIntNeqFloat.
	VisitBoolIntNeqFloat(ctx *BoolIntNeqFloatContext) interface{}

	// Visit a parse tree produced by ELParser#boolArrayAt.
	VisitBoolArrayAt(ctx *BoolArrayAtContext) interface{}

	// Visit a parse tree produced by ELParser#boolEntityNotHas.
	VisitBoolEntityNotHas(ctx *BoolEntityNotHasContext) interface{}

	// Visit a parse tree produced by ELParser#boolBigGte.
	VisitBoolBigGte(ctx *BoolBigGteContext) interface{}

	// Visit a parse tree produced by ELParser#boolDateEq.
	VisitBoolDateEq(ctx *BoolDateEqContext) interface{}

	// Visit a parse tree produced by ELParser#boolFloatGte.
	VisitBoolFloatGte(ctx *BoolFloatGteContext) interface{}

	// Visit a parse tree produced by ELParser#boolStrLte.
	VisitBoolStrLte(ctx *BoolStrLteContext) interface{}

	// Visit a parse tree produced by ELParser#boolNameEqStr.
	VisitBoolNameEqStr(ctx *BoolNameEqStrContext) interface{}

	// Visit a parse tree produced by ELParser#boolDateBefore.
	VisitBoolDateBefore(ctx *BoolDateBeforeContext) interface{}

	// Visit a parse tree produced by ELParser#boolEntityHasa.
	VisitBoolEntityHasa(ctx *BoolEntityHasaContext) interface{}

	// Visit a parse tree produced by ELParser#boolThereIsInEntityWhere.
	VisitBoolThereIsInEntityWhere(ctx *BoolThereIsInEntityWhereContext) interface{}

	// Visit a parse tree produced by ELParser#boolFloatLtInt.
	VisitBoolFloatLtInt(ctx *BoolFloatLtIntContext) interface{}

	// Visit a parse tree produced by ELParser#boolArrayNotInclude.
	VisitBoolArrayNotInclude(ctx *BoolArrayNotIncludeContext) interface{}

	// Visit a parse tree produced by ELParser#boolFloatNeqInt.
	VisitBoolFloatNeqInt(ctx *BoolFloatNeqIntContext) interface{}

	// Visit a parse tree produced by ELParser#boolBexprIsNull.
	VisitBoolBexprIsNull(ctx *BoolBexprIsNullContext) interface{}

	// Visit a parse tree produced by ELParser#boolAllHave.
	VisitBoolAllHave(ctx *BoolAllHaveContext) interface{}

	// Visit a parse tree produced by ELParser#boolIntLtFloat.
	VisitBoolIntLtFloat(ctx *BoolIntLtFloatContext) interface{}

	// Visit a parse tree produced by ELParser#boolBoolEq.
	VisitBoolBoolEq(ctx *BoolBoolEqContext) interface{}

	// Visit a parse tree produced by ELParser#boolBigEq.
	VisitBoolBigEq(ctx *BoolBigEqContext) interface{}

	// Visit a parse tree produced by ELParser#boolPlusOrMinus.
	VisitBoolPlusOrMinus(ctx *BoolPlusOrMinusContext) interface{}

	// Visit a parse tree produced by ELParser#boolParen.
	VisitBoolParen(ctx *BoolParenContext) interface{}

	// Visit a parse tree produced by ELParser#boolStrIs.
	VisitBoolStrIs(ctx *BoolStrIsContext) interface{}

	// Visit a parse tree produced by ELParser#boolThereIsNoWhere.
	VisitBoolThereIsNoWhere(ctx *BoolThereIsNoWhereContext) interface{}

	// Visit a parse tree produced by ELParser#boolFloatEqInt.
	VisitBoolFloatEqInt(ctx *BoolFloatEqIntContext) interface{}

	// Visit a parse tree produced by ELParser#boolIntNeq.
	VisitBoolIntNeq(ctx *BoolIntNeqContext) interface{}

	// Visit a parse tree produced by ELParser#boolArrayIncludes.
	VisitBoolArrayIncludes(ctx *BoolArrayIncludesContext) interface{}

	// Visit a parse tree produced by ELParser#boolStrEqIcList.
	VisitBoolStrEqIcList(ctx *BoolStrEqIcListContext) interface{}

	// Visit a parse tree produced by ELParser#boolDateGt.
	VisitBoolDateGt(ctx *BoolDateGtContext) interface{}

	// Visit a parse tree produced by ELParser#boolOr.
	VisitBoolOr(ctx *BoolOrContext) interface{}

	// Visit a parse tree produced by ELParser#boolFunction.
	VisitBoolFunction(ctx *BoolFunctionContext) interface{}

	// Visit a parse tree produced by ELParser#boolDateIsNotNull.
	VisitBoolDateIsNotNull(ctx *BoolDateIsNotNullContext) interface{}

	// Visit a parse tree produced by ELParser#boolThereIsInArrayWhere.
	VisitBoolThereIsInArrayWhere(ctx *BoolThereIsInArrayWhereContext) interface{}

	// Visit a parse tree produced by ELParser#boolTypedIsNotLiteral.
	VisitBoolTypedIsNotLiteral(ctx *BoolTypedIsNotLiteralContext) interface{}

	// Visit a parse tree produced by ELParser#boolEntityIsNotNull.
	VisitBoolEntityIsNotNull(ctx *BoolEntityIsNotNullContext) interface{}

	// Visit a parse tree produced by ELParser#boolStrNeq.
	VisitBoolStrNeq(ctx *BoolStrNeqContext) interface{}

	// Visit a parse tree produced by ELParser#boolMatchForall.
	VisitBoolMatchForall(ctx *BoolMatchForallContext) interface{}

	// Visit a parse tree produced by ELParser#boolWithinPercent.
	VisitBoolWithinPercent(ctx *BoolWithinPercentContext) interface{}

	// Visit a parse tree produced by ELParser#boolIsQuestion.
	VisitBoolIsQuestion(ctx *BoolIsQuestionContext) interface{}

	// Visit a parse tree produced by ELParser#commonerror.
	VisitCommonerror(ctx *CommonerrorContext) interface{}

	// Visit a parse tree produced by ELParser#typedEntity.
	VisitTypedEntity(ctx *TypedEntityContext) interface{}

	// Visit a parse tree produced by ELParser#typedLong.
	VisitTypedLong(ctx *TypedLongContext) interface{}

	// Visit a parse tree produced by ELParser#typedDouble.
	VisitTypedDouble(ctx *TypedDoubleContext) interface{}

	// Visit a parse tree produced by ELParser#typedString.
	VisitTypedString(ctx *TypedStringContext) interface{}

	// Visit a parse tree produced by ELParser#typedBoolean.
	VisitTypedBoolean(ctx *TypedBooleanContext) interface{}

	// Visit a parse tree produced by ELParser#typedDate.
	VisitTypedDate(ctx *TypedDateContext) interface{}

	// Visit a parse tree produced by ELParser#typedArray.
	VisitTypedArray(ctx *TypedArrayContext) interface{}

	// Visit a parse tree produced by ELParser#typedTable.
	VisitTypedTable(ctx *TypedTableContext) interface{}

	// Visit a parse tree produced by ELParser#typedName.
	VisitTypedName(ctx *TypedNameContext) interface{}

	// Visit a parse tree produced by ELParser#typedDecisionTable.
	VisitTypedDecisionTable(ctx *TypedDecisionTableContext) interface{}

	// Visit a parse tree produced by ELParser#typedOperator.
	VisitTypedOperator(ctx *TypedOperatorContext) interface{}

	// Visit a parse tree produced by ELParser#typedXmlValue.
	VisitTypedXmlValue(ctx *TypedXmlValueContext) interface{}

	// Visit a parse tree produced by ELParser#typedNull.
	VisitTypedNull(ctx *TypedNullContext) interface{}

	// Visit a parse tree produced by ELParser#typedInvalid.
	VisitTypedInvalid(ctx *TypedInvalidContext) interface{}

	// Visit a parse tree produced by ELParser#typedBoolFunction.
	VisitTypedBoolFunction(ctx *TypedBoolFunctionContext) interface{}

	// Visit a parse tree produced by ELParser#typedBigInt.
	VisitTypedBigInt(ctx *TypedBigIntContext) interface{}

	// Visit a parse tree produced by ELParser#typedBytes.
	VisitTypedBytes(ctx *TypedBytesContext) interface{}

	// Visit a parse tree produced by ELParser#undefinedIdent.
	VisitUndefinedIdent(ctx *UndefinedIdentContext) interface{}
}
