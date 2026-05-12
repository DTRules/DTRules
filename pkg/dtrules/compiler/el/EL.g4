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

grammar EL;

options {
    caseInsensitive = true;
}

// ============================================================================
// Parser Rules
// ============================================================================

// Helper rule for optional terminal semicolon (TokenFilter adds SEMI at EOF)
optSemi
    : SEMI?
    ;

done
    : ACTION SEMI                                           # emptyAction
    | CONDITION SEMI                                        # emptyCondition
    | CONTEXT SEMI                                          # emptyContext
    | POLICYSTATEMENT SEMI                                  # emptyPolicyStatement
    | ACTION statementList optSemi                          # actionStatement
    | CONDITION bexpr optSemi                               # conditionExpr
    | CONDITION debugstatement SEMI bexpr optSemi           # conditionDebugBefore
    | CONDITION bexpr SEMI debugstatement optSemi           # conditionDebugAfter
    | CONTEXT contextForTable optSemi                       # contextStatement
    | CONTEXT debugstatement SEMI contextForTable optSemi   # contextDebugBefore
    | POLICYSTATEMENT strexpr optSemi                       # policyStrExpr
    | POLICYSTATEMENT nexpr optSemi                         # policyNExpr
    | POLICYSTATEMENT iexpr optSemi                         # policyIExpr
    | POLICYSTATEMENT fexpr optSemi                         # policyFExpr
    | POLICYSTATEMENT bexpr optSemi                         # policyBExpr
    | POLICYSTATEMENT dexpr optSemi                         # policyDExpr
    ;

statementList
    : block+
    ;

separator
    : SEMI
    | COMMA
    ;

statement
    : setstatement separator
    | performstatement separator
    | debugstatement separator
    | ifstatement separator
    | addtostatement separator
    | clearstatement separator
    | usingstatement separator
    | randomstatements separator
    | operatorstatements separator
    | contextstatement separator
    | datestatement separator
    | xmlvaluestatements separator
    | errorstatement separator
    | warnstatement separator
    | createstatement separator
    ;

createstatement
    : CREATE typedEntity AS undefinedIdent                  # createEntityAs
    ;

usingblock
    : typedEntity usingblock                                # usingBlockEntity
    | typedEntity COMMA usingblock                          # usingBlockEntityComma
    | block                                                 # usingBlockBase
    ;

// possessiveRef handles entity chains like "client's plan's" or ":Client:plan's"
// Note: Original Cup grammar used COMMA injection after each POSSESSIVE, but ANTLR doesn't.
// This grammar matches consecutive possessives directly.
possessiveRef
    : POSSESSIVE+ possessiveRef?                            # possessiveChain
    | COLON typedEntity COLON possessiveRef?                # colonChain
    ;

colonRef
    : possessiveRef
    ;

// Note: gforallblock removed - inlined into block rule to avoid left recursion

contextForTable
    : debugstatement                                        # contextDebug
    | forctl                                                # contextFor
    | forallctl                                             # contextForallCtl
    | forfirstctl                                           # contextForfirst
    | contextstatement                                      # contextCtx
    | localvariables                                        # contextLocal
    ;

localvariables
    : LOCAL ENTITY undefinedIdent                           # localEntityUndef
    | LOCAL ENTITY undefinedIdent ASSIGN eexpr              # localEntityInit
    | LOCAL ENTITY typedEntity                              # localEntityDefined
    | LOCAL LONG undefinedIdent                             # localLongUndef
    | LOCAL LONG undefinedIdent ASSIGN number               # localLongInit
    | LOCAL LONG typedLong                                  # localLongDefined
    | LOCAL DOUBLE undefinedIdent                           # localDoubleUndef
    | LOCAL DOUBLE undefinedIdent ASSIGN number             # localDoubleInit
    | LOCAL DOUBLE typedDouble                              # localDoubleDefined
    | LOCAL BOOLEAN undefinedIdent                          # localBoolUndef
    | LOCAL BOOLEAN undefinedIdent ASSIGN bexpr             # localBoolInit
    | LOCAL BOOLEAN typedBoolean                            # localBoolDefined
    | LOCAL DATE undefinedIdent                             # localDateUndef
    | LOCAL DATE undefinedIdent ASSIGN dexpr                # localDateInit
    | LOCAL DATE typedDate                                  # localDateDefined
    | LOCAL ARRAY undefinedIdent                            # localArrayUndef
    | LOCAL ARRAY undefinedIdent ASSIGN arrayExpr           # localArrayInit
    | LOCAL ARRAY typedArray                                # localArrayDefined
    | LOCAL STRING undefinedIdent                           # localStringUndef
    | LOCAL STRING undefinedIdent ASSIGN strexpr            # localStringInit
    | LOCAL STRING typedString                              # localStringDefined
    | LOCAL BIGINT undefinedIdent                           # localBigIntUndef
    | LOCAL BIGINT undefinedIdent ASSIGN bigexpr            # localBigIntInit
    | LOCAL BIGINT typedBigInt                              # localBigIntDefined
    | LOCAL FIXED undefinedIdent                            # localFixedUndef
    | LOCAL FIXED undefinedIdent ASSIGN iexpr               # localFixedInit
    | LOCAL FIXED typedLong                                 # localFixedDefined
    | LOCAL BYTES undefinedIdent                            # localBytesUndef
    | LOCAL BYTES undefinedIdent ASSIGN bytesexpr           # localBytesInit
    | LOCAL BYTES typedBytes                                # localBytesDefined
    ;

ifstatement
    : IF bexpr THEN block ENDIF                             # ifThen
    | IF bexpr THEN block ELSE block ENDIF                  # ifThenElse
    ;

forallctl
    : FORALL arrayExpr                                      # forallSimple
    | FORALL arrayExpr ALLOWING ARRAY TOBEREMOVED           # forallAllowRemove
    | FORALL arrayExpr IN eexpr                             # forallInEntity
    | FORALL arrayExpr IN eexpr ALLOWING ARRAY TOBEREMOVED  # forallInEntityAllowRemove
    | FORALL arrayExpr IN eexpr WHERE bexpr                 # forallInEntityWhere
    | FORALL arrayExpr WHERE bexpr                          # forallWhere
    | FORALL arrayExpr WHERE bexpr ALLOWING ARRAY TOBEREMOVED # forallWhereAllowRemove
    | FORALL typedEntity ENTITIES                           # forallTypeEntities
    | FORALL typedEntity ENTITIES WHERE bexpr               # forallTypeEntitiesWhere
    | FORALL arrayExpr AS undefinedIdent                    # forallAs
    | FORALL arrayExpr AS undefinedIdent WHERE bexpr        # forallAsWhere
    ;

forallblock
    : arrayExpr block                                       # forallBlockSimple
    | arrayExpr WHERE bexpr block                           # forallBlockWhere
    ;

foreachblock
    : eexpr IN arrayExpr block                              # foreachSimple
    | eexpr IN arrayExpr WHERE bexpr block                  # foreachWhere
    | eexpr AND ITS eexpr IN arrayExpr block                # foreachIts
    | eexpr AND ITS eexpr IN arrayExpr WHERE bexpr block    # foreachItsWhere
    ;

forfirstctl
    : FOR FIRST OF arrayExpr WHERE bexpr                    # forfirstOf
    | FOR FIRST OF arrayExpr AND ITS eexpr WHERE bexpr      # forfirstOfIts
    | FOR FIRST IN arrayExpr WHERE bexpr                    # forfirstIn
    ;

firstblock
    : FOR FIRST OF arrayExpr WHERE bexpr THEN block ELSEIFNONEAREFOUND block ENDFF  # firstBlockElse
    | FOR FIRST OF arrayExpr WHERE bexpr THEN block ENDFF                            # firstBlockSimple
    | FOR FIRST OF arrayExpr AND ITS eexpr WHERE bexpr THEN block ELSEIFNONEAREFOUND block ENDFF # firstBlockItsElse
    ;

block
    : LCURLY statementList RCURLY                           # blockCurly
    | USING usingblock                                      # blockUsing
    | LCURLY statementList RCURLY forallctl                 # blockGforall
    | FORALL forallblock                                    # blockForall
    | FOREACH foreachblock                                  # blockForeach
    | firstblock                                            # blockFirst
    | IF ifblock                                            # blockIf
    | statement                                             # blockStatement
    ;

usingstatement
    : USING usingblock separator
    ;

leftIexpr
    : typedLong                                             # leftIexprSimple
    | colonRef leftIexpr                                    # leftIexprColon
    ;

leftFexpr
    : typedDouble                                           # leftFexprSimple
    | colonRef leftFexpr                                    # leftFexprColon
    ;

leftBexpr
    : typedBoolean                                          # leftBexprSimple
    | colonRef leftBexpr                                    # leftBexprColon
    ;

leftEexpr
    : typedEntity                                           # leftEexprSimple
    | colonRef leftEexpr                                    # leftEexprColon
    ;

leftStrexpr
    : typedString                                           # leftStrexprSimple
    | colonRef leftStrexpr                                  # leftStrexprColon
    ;

leftDexpr
    : typedDate                                             # leftDexprSimple
    | colonRef leftDexpr                                    # leftDexprColon
    ;

leftTexpr
    : typedTable                                            # leftTexprSimple
    | colonRef leftTexpr                                    # leftTexprColon
    ;

leftBigexpr
    : typedBigInt                                           # leftBigexprSimple
    | colonRef leftBigexpr                                  # leftBigexprColon
    ;

leftArrayRef
    : typedArray                                            # leftArraySimple
    | colonRef leftArrayRef                                 # leftArrayColon
    ;

setstatement
    : SET leftIexpr ASSIGN number                           # setInt
    | SET leftFexpr ASSIGN number                           # setFloat
    | SET leftBexpr ASSIGN bexpr                            # setBool
    | SET leftEexpr ASSIGN eexpr                            # setEntity
    | SET leftStrexpr ASSIGN strexpr                        # setString
    | SET leftStrexpr ASSIGN number                         # setStringFromNumber
    | SET leftStrexpr ASSIGN dexpr                          # setStringFromDate
    | SET leftStrexpr ASSIGN nexpr                          # setStringFromName
    | SET leftStrexpr ASSIGN texpr                          # setStringFromTable
    | SET leftBexpr ASSIGN nexpr                            # setBoolFromName
    | SET leftDexpr ASSIGN dexpr                            # setDate
    | SET leftTexpr ASSIGN texpr                            # setTable
    | SET leftArrayRef ASSIGN eexpr                         # setArrayEntity
    | SET leftArrayRef ASSIGN strexpr                       # setArrayString
    | SET leftArrayRef ASSIGN fexpr                         # setArrayFloat
    | SET leftArrayRef ASSIGN iexpr                         # setArrayInt
    | SET leftArrayRef ASSIGN dexpr                         # setArrayDate
    | SET leftArrayRef ASSIGN arrayExpr                     # setArrayArray
    | SET leftBigexpr ASSIGN bigexpr                        # setBigInt
    | INCREMENT typedLong                                   # incrementLong
    | INCREMENT typedDouble                                 # incrementDouble
    | DECREMENT typedLong                                   # decrementLong
    | DECREMENT typedDouble                                 # decrementDouble
    ;

forctl
    : FOR leftIexpr ASSIGN number SEMI bexpr SEMI statement
    ;

performstatement
    : PERFORM typedDecisionTable AND ONERROR ADD eexpr TO CONTEXT AND PERFORM typedDecisionTable  # performCatchError
    | PERFORM TABLE NAMED LPAREN strexpr RPAREN             # performDynamicTable
    | typedDecisionTable                                    # performDT
    | PERFORM typedDecisionTable                            # performDTExplicit
    | PERFORM NAME                                          # performName
    ;

errorstatement
    : ERROR strexpr                                         # errorStmt
    ;

warnstatement
    : WARN strexpr                                          # warnStmt
    ;

debugstatement
    : DEBUG strexpr                                         # debugStr
    | DEBUG bexpr                                           # debugBool
    | DEBUG iexpr                                           # debugInt
    | DEBUG fexpr                                           # debugFloat
    | DEBUG eexpr                                           # debugEntity
    | DEBUG dexpr                                           # debugDate
    | DEBUG arrayExpr                                       # debugArray
    | PRINT strexpr                                         # printStr
    | PRINT bexpr                                           # printBool
    | PRINT iexpr                                           # printInt
    | PRINT fexpr                                           # printFloat
    | PRINT eexpr                                           # printEntity
    | PRINT dexpr                                           # printDate
    | PRINT arrayExpr                                       # printArray
    ;

ifblock
    : bexpr THEN statementList ifcontinue
    ;

ifcontinue
    : ENDIF                                                 # ifEnd
    | ELSE statementList ENDIF                              # ifElse
    | ELSEIF ifblock                                        # ifElseIf
    ;

number
    : iexpr
    | fexpr
    ;

addtodest2
    : arrayExpr2                                            # addDestArray2
    | typedLong                                             # addDestLong2
    | typedDouble                                           # addDestDouble2
    ;

addtodest
    : arrayExpr2                                            # addDestArray
    | typedLong                                             # addDestLong
    | typedDouble                                           # addDestDouble
    | colonRef addtodest2                                   # addDestColon
    | POSSESSIVE typedLong                                  # addDestPossessiveLong
    | POSSESSIVE typedDouble                                # addDestPossessiveDouble
    ;

subtodest
    : typedLong                                             # subDestLong
    | typedDouble                                           # subDestDouble
    | colonRef addtodest2                                   # subDestColon
    | POSSESSIVE typedLong                                  # subDestPossessiveLong
    | POSSESSIVE typedDouble                                # subDestPossessiveDouble
    ;

addtostatement
    : ADD arrayExpr TO arrayExpr IF NOT MEMBER              # addArrayNoMember
    | ADD arrayExpr TO arrayExpr                            # addArrayToArray
    | ADD eexpr TO addtodest                                # addEntityToDest
    | ADD eexpr TO addtodest AND TO addtodest               # addEntityToDestDup
    | ADD strexpr TO addtodest                              # addStrToDest
    | ADD strexpr TO addtodest AND TO addtodest             # addStrToDestDup
    | ADD dexpr TO addtodest                                # addDateToDest
    | ADD dexpr TO addtodest AND TO addtodest               # addDateToDestDup
    | ADD number TO addtodest                               # addNumToDest
    | ADD number TO addtodest AND TO addtodest              # addNumToDestDup
    | SUBTRACT number FROM subtodest                        # subtractNum
    | ADD eexpr IF NOT MEMBER TO arrayExpr                  # addEntityNoDups
    | ADD eexpr IF NOT MEMBER TO arrayExpr AND TO arrayExpr # addEntityNoDupsDup
    | ADD strexpr IF NOT MEMBER TO arrayExpr                # addStrNoDups
    | ADD strexpr IF NOT MEMBER TO arrayExpr AND TO arrayExpr # addStrNoDupsDup
    ;

contextstatement
    : ADD eexpr TO CONTEXT OF THIS TABLE                    # addToContextOf
    | ADD eexpr TO CONTEXT FOR THIS TABLE                   # addToContextFor
    ;

clearstatement
    : CLEAR arrayExpr
    ;

randomstatements
    : REMOVE iexpr ELEMENT FROM arrayExpr ARRAY             # removeAtIndex
    | REMOVE EACH eexpr FROM arrayExpr WHERE bexpr          # removeEachWhere
    | REMOVE nexpr FROM arrayExpr ARRAY                     # removeName
    | REMOVE strexpr FROM arrayExpr ARRAY                   # removeString
    | REMOVE eexpr FROM arrayExpr ARRAY                     # removeEntity
    | RANDOMIZE arrayExpr                                   # randomizeArray
    | CLEAR arrayExpr                                       # clearArray
    | SORT arrayExpr IN ASCENDINGORDER BY nexpr             # sortAscending
    | SORT arrayExpr IN DESCENDINGORDER BY nexpr            # sortDescending
    ;

operatorlist
    : strexpr COMMA operatorlist                            # opListStr
    | iexpr COMMA operatorlist                              # opListInt
    | fexpr COMMA operatorlist                              # opListFloat
    | eexpr COMMA operatorlist                              # opListEntity
    | strexpr                                               # opListStrSingle
    | iexpr                                                 # opListIntSingle
    | fexpr                                                 # opListFloatSingle
    | eexpr                                                 # opListEntitySingle
    ;

operatorstatements
    : typedOperator LPAREN operatorlist RPAREN
    ;

xmlvalues
    : strexpr
    | iexpr
    | fexpr
    | dexpr
    | nexpr
    ;

xmlvaluestatements
    : typedXmlValue COLON SET ATTRIBUTE strexpr ASSIGN xmlvalues    # xmlSetAttr
    | eexpr COLON SET ATTRIBUTE strexpr ASSIGN xmlvalues            # xmlSetAttrEntity
    | typedXmlValue COLON ADD ATTRIBUTE strexpr ASSIGN xmlvalues    # xmlAddAttr
    | eexpr COLON ADD ATTRIBUTE strexpr ASSIGN xmlvalues            # xmlAddAttrEntity
    ;

arrayExpr
    : POLICYSTATEMENTS                                      # arrayPolicyStatements
    | colonRef typedArray                                   # arrayColonRef
    | arrayExpr2                                            # arrayBase
    ;

arrayExpr2
    : MAP arrayExpr THROUGH texpr                           # arrayMap
    | LPAREN arrayExpr RPAREN                               # arrayParen
    | typedArray                                            # arrayTyped
    | LPAREN ARRAY RPAREN NAME                              # arrayName
    | GET COPY OF arrayExpr                                 # arrayCopy
    | COPY OF arrayExpr                                     # arrayCopySimple
    | GET DEEPCOPY OF arrayExpr                             # arrayDeepCopy
    | DEEPCOPY OF arrayExpr                                 # arrayDeepCopySimple
    | arrayLit                                              # arrayLiteral
    | ARRAY_OF_VALUES LBRACE arrayList RBRACE               # arrayOfValues
    | TOKENIZE strexpr BY strexpr                           # arrayTokenize
    ;

arrayLit
    : LBRACE arrayList RBRACE
    ;

arrayList
    : arrayList COMMA strexpr                               # arrayListStr
    | arrayList COMMA iexpr                                 # arrayListInt
    | arrayList COMMA eexpr                                 # arrayListEntity
    | arrayList COMMA fexpr                                 # arrayListFloat
    | arrayList COMMA nexpr                                 # arrayListName
    | arrayList COMMA arrayExpr                             # arrayListArray
    | arrayList COMMA bexpr                                 # arrayListBool
    | bexpr                                                 # arrayListBoolSingle
    | arrayExpr                                             # arrayListArraySingle
    | nexpr                                                 # arrayListNameSingle
    | fexpr                                                 # arrayListFloatSingle
    | eexpr                                                 # arrayListEntitySingle
    | iexpr                                                 # arrayListIntSingle
    | strexpr                                               # arrayListStrSingle
    ;

indxExpr
    : arrayExpr LBRACE iexpr RBRACE
    ;

eexpr
    : typedEntity                                           # entityTyped
    | LPAREN eexpr RPAREN                                   # entityParen
    | indxExpr                                              # entityIndex
    | NEW nexpr ENTITY                                      # entityNewName
    | NEW typedEntity ENTITY                                # entityNewTyped
    | CLONE OF eexpr                                        # entityClone
    | colonRef typedEntity                                  # entityColonRef
    | LPAREN ENTITY RPAREN typedTable LPAREN tablelist RPAREN # entityTableLookup
    | FIRST eexpr IN arrayExpr WHERE bexpr                  # entityFirstIn
    | FIRST eexpr WHERE bexpr                               # entityFirst
    | strexpr OF eexpr                                      # entityRelationship
    ;

datestatement
    : SUBTRACT number YEARS FROM typedDate                  # dateSubYears
    | SUBTRACT number MONTHS FROM typedDate                 # dateSubMonths
    | SUBTRACT number DAYS FROM typedDate                   # dateSubDays
    | ADD number YEARS TO typedDate                         # dateAddYears
    | ADD number MONTHS TO typedDate                        # dateAddMonths
    | ADD number DAYS TO typedDate                          # dateAddDays
    ;

dexpr
    : LPAREN dexpr RPAREN                                   # dateParen
    | typedDate                                             # dateTyped
    | LPAREN DATE RPAREN strexpr                            # dateFromStrCast
    | DATE LPAREN strexpr RPAREN                            # dateFromStrFunc
    | LPAREN DATE RPAREN indxExpr                           # dateFromIndex
    | LPAREN DATE RPAREN typedArray LBRACE iexpr RBRACE     # dateFromArrayAt
    | USING eexpr LPAREN dexpr RPAREN                       # dateUsing
    | colonRef typedDate                                    # dateColonRef
    | LPAREN number DAYS RPAREN                             # dateDays
    | dexpr PLUS dexpr                                      # dateAdd
    | dexpr MINUS dexpr                                     # dateSub
    | LPAREN DATE RPAREN typedTable LPAREN tablelist RPAREN # dateTableLookup
    // Phase 2 of #743: in-zone alts must precede their plain counterparts so
    // ANTLR matches the longer form first.
    | CURRENT_DATE INZONE strexpr                           # dateCurrentDateInZone
    | CURRENT_DATE                                          # dateCurrentDate
    | SUBTRACT number YEARS FROM dexpr                      # dateExprSubYears
    | SUBTRACT number MONTHS FROM dexpr                     # dateExprSubMonths
    | SUBTRACT number DAYS FROM dexpr                       # dateExprSubDays
    | ADD number YEARS TO dexpr                             # dateExprAddYears
    | ADD number MONTHS TO dexpr                            # dateExprAddMonths
    | ADD number DAYS TO dexpr                              # dateExprAddDays
    | dexpr MINUS number YEARS                              # dateMinusYears
    | dexpr MINUS number MONTHS                             # dateMinusMonths
    | dexpr MINUS number DAYS                               # dateMinusDays
    | dexpr PLUS number YEARS                               # datePlusYears
    | dexpr PLUS number MONTHS                              # datePlusMonths
    | dexpr PLUS number DAYS                                # datePlusDays
    | FIRST OF YEARS OF dexpr INZONE strexpr                # dateFirstOfYearInZone
    | FIRST OF YEARS OF dexpr                               # dateFirstOfYear
    | FIRST OF MONTHS OF dexpr INZONE strexpr               # dateFirstOfMonthInZone
    | FIRST OF MONTHS OF dexpr                              # dateFirstOfMonth
    | END OF MONTHS OF dexpr INZONE strexpr                 # dateEndOfMonthInZone
    | END OF MONTHS OF dexpr                                # dateEndOfMonth
    // Phase 3 of #743: week/quarter/year buckets. Long forms (with optional
    // STARTING / INZONE clauses) listed before plain so ANTLR matches them
    // first.
    | FIRST OF WEEKS OF dexpr STARTING strexpr INZONE strexpr   # dateFirstOfWeekStartingInZone
    | FIRST OF WEEKS OF dexpr STARTING strexpr                  # dateFirstOfWeekStarting
    | FIRST OF WEEKS OF dexpr INZONE strexpr                    # dateFirstOfWeekInZone
    | FIRST OF WEEKS OF dexpr                                   # dateFirstOfWeek
    | END OF WEEKS OF dexpr STARTING strexpr INZONE strexpr     # dateEndOfWeekStartingInZone
    | END OF WEEKS OF dexpr STARTING strexpr                    # dateEndOfWeekStarting
    | END OF WEEKS OF dexpr INZONE strexpr                      # dateEndOfWeekInZone
    | END OF WEEKS OF dexpr                                     # dateEndOfWeek
    | FIRST OF QUARTERS OF dexpr INZONE strexpr                 # dateFirstOfQuarterInZone
    | FIRST OF QUARTERS OF dexpr                                # dateFirstOfQuarter
    | END OF QUARTERS OF dexpr INZONE strexpr                   # dateEndOfQuarterInZone
    | END OF QUARTERS OF dexpr                                  # dateEndOfQuarter
    | END OF YEARS OF dexpr INZONE strexpr                      # dateEndOfYearInZone
    | END OF YEARS OF dexpr                                     # dateEndOfYear
    | EARLIEST OF arrayExpr AFTER dexpr                     # dateEarliestAfter
    // Phase 4 of #743: explicit `new date Y, M, D[, h, m, s] in zone <s>
    // [with dst_rule <s>]` constructor. The longer with-dst-rule forms must
    // precede the plain forms so ANTLR matches the larger production first.
    | NEW DATE iexpr COMMA iexpr COMMA iexpr COMMA iexpr COMMA iexpr COMMA iexpr INZONE strexpr WITH_DST_RULE strexpr   # dateNewYMDhmsInZoneWithDST
    | NEW DATE iexpr COMMA iexpr COMMA iexpr COMMA iexpr COMMA iexpr COMMA iexpr INZONE strexpr                         # dateNewYMDhmsInZone
    | NEW DATE iexpr COMMA iexpr COMMA iexpr INZONE strexpr WITH_DST_RULE strexpr                                       # dateNewYMDInZoneWithDST
    | NEW DATE iexpr COMMA iexpr COMMA iexpr INZONE strexpr                                                             # dateNewYMDInZone
    // Phase 2 of #743: rewrap any date in a zone for downstream extraction.
    // Listed last so the more specific alternatives above are matched first.
    | dexpr INZONE strexpr                                  # dateInZone
    ;

nexpr
    : typedName                                             # nameTyped
    | NAMEOF eexpr                                          # nameOf
    | THENAME strexpr                                       # nameTheName
    | NAME typedArray LBRACE iexpr RBRACE                   # nameArrayAt
    | NAME                                                  # nameLiteral
    | USING eexpr LPAREN nexpr RPAREN                       # nameUsing
    | colonRef typedName                                    # nameColonRef
    | LPAREN NAME RPAREN strexpr                            # nameFromStr
    ;

tablelist
    : strexpr COMMA tablelist                               # tableListMulti
    | strexpr                                               # tableListSingle
    ;

texpr
    : typedTable                                            # tableTyped
    | NEW strexpr TABLE OF strexpr                          # tableNew
    ;

strexpr
    : ATTRIBUTE strexpr OF eexpr                            # strAttrOf
    | MAPPINGKEY                                            # strMappingKey
    | typedXmlValue                                         # strXmlValue
    | typedXmlValue COLON GET ATTRIBUTE strexpr             # strXmlAttr
    | SUBSTRING OF strexpr FROM iexpr TO iexpr              # strSubstring
    | TABLEINFORMATION                                      # strTableInfo
    | STRING VALUE OF operatorstatements                    # strValueOfOp
    | LPAREN STRING RPAREN texpr LPAREN tablelist RPAREN    # strTableLookup
    | typedString                                           # strTyped
    | colonRef strexpr                                      # strColonRef
    | STRING_LITERAL                                        # strLiteral
    | strexpr PLUS strexpr                                  # strConcat
    | STRING VALUE OF fexpr                                 # strValueOfFloat
    | STRING VALUE OF iexpr                                 # strValueOfInt
    | STRING VALUE OF dexpr                                 # strValueOfDate
    | STRING VALUE OF BOOLEAN bexpr                         # strValueOfBool
    | LPAREN strexpr RPAREN                                 # strParen
    | strexpr PLUS iexpr                                    # strConcatInt
    | strexpr PLUS fexpr                                    # strConcatFloat
    | strexpr PLUS nexpr                                    # strConcatName
    | strexpr PLUS eexpr                                    # strConcatEntity
    | strexpr PLUS dexpr                                    # strConcatDate
    | strexpr PLUS arrayExpr                                # strConcatArray
    | strexpr PLUS typedNull                                # strConcatNull
    | strexpr PLUS typedInvalid                             # strConcatInvalid
    | TRIM LPAREN strexpr RPAREN                            # strTrim
    | LPAREN STRING RPAREN indxExpr                         # strFromIndex
    | CHANGE strexpr TO LOWER_CASE                          # strToLower
    | CHANGE strexpr TO UPPER_CASE                          # strToUpper
    | GET CURRENT_TIMESTAMP                                 # strTimestamp
    | USING eexpr LPAREN strexpr RPAREN                     # strUsing
    | RELATIONSHIP_BETWEEN eexpr AND eexpr                  # strRelationship
    | HEX OF bytesexpr                                      # strHexOfBytes
    | BASE58CHECK OF bytesexpr VERSION iexpr                # strBase58CheckOfBytes
    | BECH32 OF bytesexpr HRP strexpr                       # strBech32OfBytes
    // Phase 4 of #743: explicit `format(<dexpr>, <strexpr>) [in zone <s>]`
    // for audit-trail rendering. Layout is a Go time.Format reference
    // string. In-zone alt listed first so the longer match wins.
    | FORMAT LPAREN dexpr COMMA strexpr RPAREN INZONE strexpr   # strFormatDateInZone
    | FORMAT LPAREN dexpr COMMA strexpr RPAREN                  # strFormatDate
    ;

fexpr
    : FLOAT_LITERAL                                         # floatLiteral
    | colonRef typedDouble                                  # floatColonRef
    | typedDouble                                           # floatTyped
    | LPAREN DOUBLE RPAREN strexpr                          # floatFromStr
    | LPAREN DOUBLE RPAREN iexpr                            # floatFromInt
    | LPAREN DOUBLE RPAREN typedTable LPAREN tablelist RPAREN # floatTableLookup
    // Multiplication/division have higher precedence (listed first)
    | fexpr TIMES iexpr                                     # floatMulInt
    | iexpr TIMES fexpr                                     # intMulFloat
    | fexpr TIMES fexpr                                     # floatMulFloat
    | fexpr DIVIDE iexpr                                    # floatDivInt
    | iexpr DIVIDE fexpr                                    # intDivFloat
    | fexpr DIVIDE fexpr                                    # floatDivFloat
    // Addition/subtraction have lower precedence (listed after)
    | fexpr PLUS iexpr                                      # floatAddInt
    | fexpr PLUS fexpr                                      # floatAddFloat
    | iexpr PLUS fexpr                                      # intAddFloat
    | fexpr MINUS iexpr                                     # floatSubInt
    | iexpr MINUS fexpr                                     # intSubFloat
    | fexpr MINUS fexpr                                     # floatSubFloat
    | MINUS fexpr                                           # floatNegate
    | LPAREN fexpr RPAREN                                   # floatParen
    | LPAREN DOUBLE RPAREN indxExpr                         # floatFromIndex
    | ADD TO typedDouble number                             # floatAddTo
    | SUBTRACT FROM typedDouble number                      # floatSubFrom
    | MULTIPLY typedDouble BY number                        # floatMulBy
    | DIVIDE typedDouble BY number                          # floatDivBy
    | ABSOLUTEVALUE OF fexpr                                # floatAbs
    | USING eexpr LPAREN fexpr RPAREN                       # floatUsing
    | DOUBLE VALUE OF operatorstatements                    # floatValueOfOp
    | fexpr ROUNDED                                         # floatRounded
    | fexpr ROUNDED TO iexpr DECIMAL_PLACES                 # floatRoundedTo
    | fexpr ROUNDED TO iexpr DECIMAL_PLACES WITH_BOUNDRY fexpr # floatRoundedBoundry
    | SUM_OF typedDouble IN arrayExpr                       # floatSumOf
    | MINIMUM fexpr AND fexpr                               # floatMinOfFloat
    | MINIMUM fexpr AND iexpr                               # floatMinOfInt
    | MINIMUM iexpr AND fexpr                               # floatMinIntOf
    | MINIMUM fexpr COMMA fexpr                             # floatMinOfFloatComma
    | MINIMUM fexpr COMMA iexpr                             # floatMinOfIntComma
    | MINIMUM iexpr COMMA fexpr                             # floatMinIntOfComma
    | MAXIMUM fexpr AND fexpr                               # floatMaxOfFloat
    | MAXIMUM fexpr AND iexpr                               # floatMaxOfInt
    | MAXIMUM iexpr AND fexpr                               # floatMaxIntOf
    | MAXIMUM fexpr COMMA fexpr                             # floatMaxOfFloatComma
    | MAXIMUM fexpr COMMA iexpr                             # floatMaxOfIntComma
    | MAXIMUM iexpr COMMA fexpr                             # floatMaxIntOfComma
    ;

iexpr
    // Multiplication/division have higher precedence (listed first in ANTLR 4)
    : iexpr TIMES iexpr                                     # intMul
    | iexpr DIVIDE iexpr                                    # intDiv
    // Addition/subtraction have lower precedence (listed after)
    | iexpr PLUS iexpr                                      # intAdd
    | iexpr MINUS iexpr                                     # intSub
    | FP_LITERAL                                            # fixedLiteral
    | INT_LITERAL                                           # intLiteral
    | MINUS iexpr                                           # intNegate
    | LPAREN iexpr RPAREN                                   # intParen
    | typedLong                                             # intTyped
    // Phase 2 of #743: in-zone variants matched first.
    | GET DAYS IN YEAROF dexpr INZONE strexpr               # intDaysInYearInZone
    | GET DAYS IN YEAROF dexpr                              # intDaysInYear
    | GET DAYS IN MONTHS FOR dexpr INZONE strexpr           # intDaysInMonthInZone
    | GET DAYS IN MONTHS FOR dexpr                          # intDaysInMonth
    | GET DAYS OF MONTHS FOR dexpr INZONE strexpr           # intDayOfMonthInZone
    | GET DAYS OF MONTHS FOR dexpr                          # intDayOfMonth
    | colonRef typedLong                                    # intColonRef
    | LPAREN LONG RPAREN indxExpr                           # intFromIndex
    | LPAREN LONG RPAREN strexpr                            # intFromStr
    | LPAREN LONG RPAREN number                             # intFromNumber
    | LPAREN LONG RPAREN typedTable LPAREN tablelist RPAREN # intTableLookup
    | LPAREN FIXED RPAREN strexpr                           # fixedFromStr
    | LPAREN FIXED RPAREN iexpr                             # fixedFromNumber
    | LPAREN FIXED RPAREN fexpr                             # fixedFromFloat
    | LPAREN FIXED RPAREN indxExpr                          # fixedFromIndex
    | NUMBEROF arrayExpr                                    # intNumberOf
    | NUMBEROF arrayExpr WHERE bexpr                        # intNumberOfWhere
    | LENGTH OF arrayExpr                                   # intLengthArray
    | LENGTH OF strexpr                                     # intLengthStr
    | LENGTH OF bytesexpr                                   # intLengthBytes
    | bytesexpr LBRACE iexpr RBRACE                         # intBytesIndex
    | INDEX_OF strexpr IN strexpr                           # intIndexOf
    | USING arrayExpr number                                # intUsingArray
    | ADD TO typedLong number                               # intAddTo
    | SUBTRACT FROM typedLong number                        # intSubFrom
    | MULTIPLY typedLong BY number                          # intMulBy
    | DIVIDE typedLong BY number                            # intDivBy
    | ABSOLUTEVALUE OF iexpr                                # intAbs
    | USING eexpr LPAREN iexpr RPAREN                       # intUsing
    | DAYS FROM dexpr TO dexpr                              # intDaysBetween
    | MONTHS FROM dexpr TO dexpr                            # intMonthsBetween
    | YEARS FROM dexpr TO dexpr                             # intYearsBetween
    | GET YEAROF dexpr INZONE strexpr                       # intYearOfInZone
    | GET YEAROF dexpr                                      # intYearOf
    // Phase 3 of #743: time-component extractors. In-zone alts listed first.
    | GET HOUROF dexpr INZONE strexpr                       # intHourOfInZone
    | GET HOUROF dexpr                                      # intHourOf
    | GET MINUTEOF dexpr INZONE strexpr                     # intMinuteOfInZone
    | GET MINUTEOF dexpr                                    # intMinuteOf
    | GET SECONDOF dexpr INZONE strexpr                     # intSecondOfInZone
    | GET SECONDOF dexpr                                    # intSecondOf
    | GET DAYOFWEEK OF dexpr INZONE strexpr                 # intDayOfWeekInZone
    | GET DAYOFWEEK OF dexpr                                # intDayOfWeek
    | GET WEEKOFYEAR OF dexpr INZONE strexpr                # intWeekOfYearInZone
    | GET WEEKOFYEAR OF dexpr                               # intWeekOfYear
    | LONG VALUE OF operatorstatements                      # intValueOfOp
    | SUM_OF iexpr IN arrayExpr                             # intSumOf
    | MINIMUM iexpr AND iexpr                               # intMinOf
    | MINIMUM iexpr COMMA iexpr                             # intMinOfComma
    | MAXIMUM iexpr AND iexpr                               # intMaxOf
    | MAXIMUM iexpr COMMA iexpr                             # intMaxOfComma
    ;

bigexpr
    : bigexpr TIMES bigexpr                                 # bigMul
    | bigexpr DIVIDE bigexpr                                # bigDiv
    | bigexpr PLUS bigexpr                                  # bigAdd
    | bigexpr MINUS bigexpr                                 # bigSub
    | MINUS bigexpr                                         # bigNegate
    | LPAREN bigexpr RPAREN                                 # bigParen
    | typedBigInt                                           # bigTyped
    | colonRef typedBigInt                                  # bigColonRef
    | LPAREN BIGINT RPAREN strexpr                          # bigFromStr
    | LPAREN BIGINT RPAREN iexpr                            # bigFromInt
    | LPAREN BIGINT RPAREN fexpr                            # bigFromFloat
    | USING eexpr LPAREN bigexpr RPAREN                     # bigUsing
    | ABSOLUTEVALUE OF bigexpr                              # bigAbs
    | BIGINT OF BYTES bytesexpr                             # bigFromBytes
    ;

bytesexpr
    : HEX_BYTES_LITERAL                                     # bytesLiteral
    | typedBytes                                            # bytesTyped
    | colonRef typedBytes                                   # bytesColonRef
    | LPAREN bytesexpr RPAREN                               # bytesParen
    | bytesexpr PLUS bytesexpr                              # bytesConcat
    | bytesexpr FROM iexpr TO iexpr                         # bytesSlice
    | SHA256 OF bytesexpr                                   # bytesSha256
    | KECCAK256 OF bytesexpr                                # bytesKeccak256
    | RIPEMD160 OF bytesexpr                                # bytesRipemd160
    | SHA3 OF bytesexpr                                     # bytesSha3
    | BYTES OF HEX strexpr                                  # bytesCvHex
    | BYTES OF BASE58CHECK strexpr                          # bytesCvBase58Check
    | BYTES OF BECH32 strexpr                               # bytesCvBech32
    | BYTES OF BIGINT bigexpr SIZE iexpr                    # bytesCvBigInt
    ;

includeSearch
    : VALUE number                                          # includeNumber
    | DATE dexpr                                            # includeDate
    | eexpr                                                 # includeEntity
    | STRING strexpr                                        # includeString
    ;

inthe
    : IN
    | FOR
    | ON
    ;

thereis
    : THERE IS
    | IS THERE
    ;

blist
    : strexpr COMMA blist                                   # blistMulti
    | OR strexpr                                            # blistOr
    ;

blistIc
    : strexpr COMMA blist                                   # blistIcMulti
    | OR strexpr                                            # blistIcOr
    ;

bexpr
    // Array inclusion tests
    : arrayExpr DOES NOT INCLUDE includeSearch              # boolArrayNotInclude
    | arrayExpr DOES INCLUDE includeSearch                  # boolArrayDoesInclude
    | arrayExpr INCLUDES includeSearch                      # boolArrayIncludes

    // Complex tests
    | thereis MATCH FORALL arrayExpr TO nexpr IN arrayExpr  # boolMatchForall
    | thereis eexpr WHERE bexpr                             # boolThereIsWhere
    | thereis eexpr inthe eexpr WHERE bexpr                 # boolThereIsInEntityWhere
    | thereis eexpr inthe arrayExpr WHERE bexpr             # boolThereIsInArrayWhere
    | THERE IS NO eexpr WHERE bexpr                         # boolThereIsNoWhere
    | THERE IS NO eexpr inthe eexpr WHERE bexpr             # boolThereIsNoInEntityWhere
    | THERE IS NO eexpr inthe arrayExpr WHERE bexpr         # boolThereIsNoInArrayWhere
    | ALL arrayExpr HAVE bexpr                              # boolAllHave
    | ONE OF arrayExpr HASA bexpr                           # boolOneOfHasa

    // Relationship tests
    | eexpr DOES NOT HAVE strexpr                           # boolEntityNotHas
    | eexpr HASA strexpr                                    # boolEntityHasa
    | eexpr HASA strexpr WHERE bexpr                        # boolEntityHasaWhere
    | eexpr IS strexpr OF eexpr                             # boolEntityIsOf

    // Numeric within/percent tests
    | fexpr IS WITHIN number PERCENTOF fexpr                # boolWithinPercent
    | fexpr IS PLUSORMINUS number OF fexpr                  # boolPlusOrMinus

    // Name comparisons (must come BEFORE integer comparisons so IDENT vs IDENT
    // is parsed as name comparison, not integer comparison)
    | nexpr EQ nexpr                                        # boolNameEq
    | nexpr EQ strexpr                                      # boolNameEqStr
    | nexpr NEQ nexpr                                       # boolNameNeq
    | nexpr NEQ strexpr                                     # boolNameNeqStr

    // Integer comparisons (IDENT vs INT_LITERAL still works since INT_LITERAL
    // cannot be parsed as nexpr)
    | iexpr EQ iexpr                                        # boolIntEq
    | fexpr EQ iexpr                                        # boolFloatEqInt
    | iexpr EQ fexpr                                        # boolIntEqFloat
    | fexpr EQ fexpr                                        # boolFloatEq
    | iexpr NEQ iexpr                                       # boolIntNeq
    | fexpr NEQ iexpr                                       # boolFloatNeqInt
    | iexpr NEQ fexpr                                       # boolIntNeqFloat
    | fexpr NEQ fexpr                                       # boolFloatNeq
    | iexpr GT iexpr                                        # boolIntGt
    | fexpr GT iexpr                                        # boolFloatGtInt
    | iexpr GT fexpr                                        # boolIntGtFloat
    | fexpr GT fexpr                                        # boolFloatGt
    | iexpr GTE iexpr                                       # boolIntGte
    | fexpr GTE iexpr                                       # boolFloatGteInt
    | iexpr GTE fexpr                                       # boolIntGteFloat
    | fexpr GTE fexpr                                       # boolFloatGte
    | iexpr LT iexpr                                        # boolIntLt
    | fexpr LT iexpr                                        # boolFloatLtInt
    | iexpr LT fexpr                                        # boolIntLtFloat
    | fexpr LT fexpr                                        # boolFloatLt
    | iexpr LTE iexpr                                       # boolIntLte
    | fexpr LTE iexpr                                       # boolFloatLteInt
    | iexpr LTE fexpr                                       # boolIntLteFloat
    | fexpr LTE fexpr                                       # boolFloatLte

    // BigInt comparisons
    | bigexpr EQ bigexpr                                    # boolBigEq
    | bigexpr NEQ bigexpr                                   # boolBigNeq
    | bigexpr GT bigexpr                                    # boolBigGt
    | bigexpr GTE bigexpr                                   # boolBigGte
    | bigexpr LT bigexpr                                    # boolBigLt
    | bigexpr LTE bigexpr                                   # boolBigLte

    // Bytes comparisons (constant-time equality)
    | bytesexpr EQ bytesexpr                                # boolBytesEq
    | bytesexpr NEQ bytesexpr                               # boolBytesNeq

    // Boolean reference
    | typedBoolean                                          # boolTyped
    | colonRef typedBoolean                                 # boolColonRef

    // Boolean literals (true, false, default, otherwise, always)
    | RBOOLEAN                                              # boolLiteral

    // String comparisons
    | strexpr EQ_IGNORE_CASE blistIc                        # boolStrEqIcList
    | strexpr EQ blist                                      # boolStrEqList
    | strexpr EQ strexpr                                    # boolStrEq
    | strexpr NEQ strexpr                                   # boolStrNeq
    | strexpr IS strexpr                                    # boolStrIs
    | strexpr IS NOT strexpr                                # boolStrIsNot
    | strexpr EQ_IGNORE_CASE strexpr                        # boolStrEqIc
    | strexpr NEQ_IGNORE_CASE strexpr                       # boolStrNeqIc
    | strexpr AT iexpr STARTS_WITH strexpr                  # boolStartsWithAt
    | strexpr STARTS_WITH strexpr                           # boolStartsWith
    | strexpr IS ONE OF arrayExpr                           # boolStrIsOneOf
    | strexpr IS NOT ONE OF arrayExpr                       # boolStrIsNotOneOf

    // Question mark tests
    | DOES bexpr QUESTIONMARK                               # boolDoesQuestion
    | IS bexpr QUESTIONMARK                                 # boolIsQuestion
    | WAS bexpr QUESTIONMARK                                # boolWasQuestion

    // String compares
    | strexpr GT strexpr                                    # boolStrGt
    | strexpr LT strexpr                                    # boolStrLt
    | strexpr GTE strexpr                                   # boolStrGte
    | strexpr LTE strexpr                                   # boolStrLte

    // Regex
    | strexpr MATCHES strexpr                               # boolMatches

    // Boolean "is" tests (flag.a is true, flag.b is false)
    | typedBoolean IS RBOOLEAN                              # boolTypedIsLiteral
    | typedBoolean IS NOT RBOOLEAN                          # boolTypedIsNotLiteral
    | colonRef typedBoolean IS RBOOLEAN                     # boolColonIsLiteral
    | colonRef typedBoolean IS NOT RBOOLEAN                 # boolColonIsNotLiteral

    // Boolean operations
    | bexpr EQ bexpr                                        # boolBoolEq
    | bexpr NEQ bexpr                                       # boolBoolNeq
    | bexpr AND bexpr                                       # boolAnd
    | bexpr OR bexpr                                        # boolOr
    | NOT bexpr                                             # boolNot

    // Null tests
    | bexpr ISNULL                                          # boolBexprIsNull
    | bexpr ISNOTNULL                                       # boolBexprIsNotNull
    | number ISNULL                                         # boolNumIsNull
    | number ISNOTNULL                                      # boolNumIsNotNull
    | dexpr ISNULL                                          # boolDateIsNull
    | arrayExpr ISNULL                                      # boolArrayIsNull
    | strexpr ISNULL                                        # boolStrIsNull
    | eexpr ISNULL                                          # boolEntityIsNull
    | dexpr ISNOTNULL                                       # boolDateIsNotNull
    | arrayExpr ISNOTNULL                                   # boolArrayIsNotNull
    | strexpr ISNOTNULL                                     # boolStrIsNotNull
    | eexpr ISNOTNULL                                       # boolEntityIsNotNull

    // Using
    | USING eexpr LPAREN bexpr RPAREN                       # boolUsing
    | LPAREN bexpr RPAREN                                   # boolParen
    | LPAREN BOOLEAN RPAREN indxExpr                        # boolFromIndex
    | LPAREN BOOLEAN RPAREN strexpr                         # boolFromStr
    | BOOLEAN typedArray LBRACE iexpr RBRACE                # boolArrayAt

    // Date comparisons
    | dexpr EQ dexpr                                        # boolDateEq
    | dexpr LT dexpr                                        # boolDateLt
    | dexpr IS BEFORE dexpr                                 # boolDateBefore
    | dexpr GT dexpr                                        # boolDateGt
    | dexpr IS AFTER dexpr                                  # boolDateAfter
    | dexpr GTE dexpr                                       # boolDateGte
    | dexpr LTE dexpr                                       # boolDateLte
    | dexpr IS BETWEEN dexpr AND dexpr                      # boolDateBetween

    // Phase 3 of #743: calendar comparisons. `the` is skipped by ARTICLE so
    // the surface phrase `is the same calendar day as ...` lexes as
    // `IS SAME CALENDAR DAYS AS ...`. INZONE strexpr is mandatory at the
    // grammar level — there is no plain alt.
    | dexpr IS SAME CALENDAR DAYS AS dexpr INZONE strexpr                       # boolSameCalendarDay
    | dexpr IS SAME CALENDAR WEEKS AS dexpr STARTING strexpr INZONE strexpr     # boolSameCalendarWeekStarting
    | dexpr IS SAME CALENDAR WEEKS AS dexpr INZONE strexpr                      # boolSameCalendarWeek
    | dexpr IS SAME CALENDAR MONTHS AS dexpr INZONE strexpr                     # boolSameCalendarMonth
    | dexpr IS SAME CALENDAR QUARTERS AS dexpr INZONE strexpr                   # boolSameCalendarQuarter
    | dexpr IS SAME CALENDAR YEARS AS dexpr INZONE strexpr                      # boolSameCalendarYear

    // Entity comparisons
    | eexpr EQ eexpr                                        # boolEntityEq
    | eexpr NEQ eexpr                                       # boolEntityNeq
    | typedEntity ENTITY IS NOT IN CONTEXT                  # boolEntityNotInContext
    | typedEntity ENTITY IS IN CONTEXT                      # boolEntityInContext
    | strexpr ENTITY IS IN CONTEXT                          # boolStrEntityInContext
    | strexpr ENTITY IS NOT IN CONTEXT                      # boolStrEntityNotInContext

    // Operator and function
    | BOOLEAN VALUE OF operatorstatements                   # boolValueOfOp
    | typedBoolFunction                                     # boolFunction
    ;

// Common error handling
commonerror
    : /* empty - error productions handled by ANTLR error handling */
    ;

// ============================================================================
// Typed identifier placeholders (to be resolved by listener/visitor)
// These are replaced with actual type tokens by the type resolution phase
// ============================================================================

typedEntity         : IDENT ;
typedLong           : IDENT ;
typedDouble         : IDENT ;
typedString         : IDENT ;
typedBoolean        : IDENT ;
typedDate           : IDENT ;
typedArray          : IDENT ;
typedTable          : IDENT ;
typedName           : IDENT ;
typedDecisionTable  : IDENT ;
typedOperator       : IDENT ;
typedXmlValue       : IDENT ;
typedNull           : IDENT ;
typedInvalid        : IDENT ;
typedBoolFunction   : IDENT ;
typedBigInt         : IDENT ;
typedBytes          : IDENT ;
undefinedIdent      : IDENT ;

// ============================================================================
// Lexer Rules
// ============================================================================

// Keywords
ACTION              : 'action' ;
CONDITION           : 'condition' ;
POLICYSTATEMENT     : 'policystatement' ;
POLICYSTATEMENTS    : 'policy' WS+ 'statements' ;

// Boolean literals
RBOOLEAN            : 'true' | 'false' | 'default' | 'otherwise' | 'always'
                    | 'perform' WS+ 'when' WS+ 'called' ;

// Type keywords
DATE                : 'date' | 'time' ;
BOOLEAN             : 'boolean' ;
DOUBLE              : 'double' ;
LONG                : 'int' | 'long' ;
STRING              : 'string' ;
ENTITY              : 'entity' ;
ENTITIES            : 'entities' ;
ARRAY               : 'array' ;
TABLE               : 'table' ;
NAMED               : 'named' ;
CREATE              : 'create' ;
BIGINT              : 'bigint' | 'biginteger' ;
FIXED               : 'fixed' ;
BYTES               : 'bytes' ;
SHA256              : 'sha256' ;
KECCAK256           : 'keccak256' ;
RIPEMD160           : 'ripemd160' ;
SHA3                : 'sha3' ;
HEX                 : 'hex' ;
BASE58CHECK         : 'base58check' ;
VERSION             : 'version' ;
BECH32              : 'bech32' ;
HRP                 : 'hrp' ;
SIZE                : 'size' ;

// Current date/time
CURRENT_TIMESTAMP   : 'current' WS+ 'timestamp' ;
CURRENT_DATE        : 'current' WS+ 'date' ;

// Phase 2 of #743: explicit timezone DSL — `<dexpr> in zone <strexpr>`.
// Single token so the natural-language phrase can't be split across other
// rules that consume IN.
INZONE              : 'in' WS+ 'zone' ;

// Phase 4 of #743: explicit DST disambiguation clause for component
// constructors and an explicit format() builtin. WITH_DST_RULE is a single
// token so the surface phrase can't be split across other rules that
// consume WITH or WITHIN.
WITH_DST_RULE       : 'with' WS+ 'dst_rule' ;
FORMAT              : 'format' ;

// Punctuation
SEMI                : ';' ;
COLON               : ':' ;
COMMA               : ',' ;
QUESTIONMARK        : '?' ;
LPAREN              : '(' ;
RPAREN              : ')' ;
LBRACE              : '[' ;
RBRACE              : ']' ;
LCURLY              : '{' ;
RCURLY              : '}' ;

// Operators
PLUS                : '+' ;
MINUS               : '-' ;
DIVIDE              : '/' | 'div' ;
TIMES               : '*' ;
ASSIGN              : '=' ;

// Comparison operators - natural language (longer patterns first to ensure correct matching)
GTE                 : '>=' | '&gt=' | '\u2265'
                    | 'is' WS+ 'greater' WS+ 'than' WS+ 'or' WS+ 'equal' WS+ 'to'
                    | 'greater' WS+ 'than' WS+ 'or' WS+ 'equal' WS+ 'to'
                    | 'at' WS+ 'or' WS+ 'above' ;
LTE                 : '<=' | '&lt=' | '\u2264'
                    | 'is' WS+ 'less' WS+ 'than' WS+ 'or' WS+ 'equal' WS+ 'to'
                    | 'less' WS+ 'than' WS+ 'or' WS+ 'equal' WS+ 'to'
                    | 'at' WS+ 'or' WS+ 'below' ;
GT                  : '>' | '&gt'
                    | 'is' WS+ 'greater' WS+ 'than'
                    | 'greater' WS+ 'than' ;
LT                  : '<' | '&lt'
                    | 'is' WS+ 'less' WS+ 'than'
                    | 'less' WS+ 'than' ;
NEQ_IGNORE_CASE     : 'is' WS+ 'not' WS+ 'equal' WS+ ('to' WS+)? 'ignore' WS+ 'case' ;
EQ_IGNORE_CASE      : 'is' WS+ 'equal' WS+ ('to' WS+)? 'ignore' WS+ 'case' ;
NEQ                 : '!='
                    | 'is' WS+ 'not' WS+ 'equal' (WS+ 'to')?
                    | 'not' WS+ 'equal' (WS+ 'to')? ;
EQ                  : '=='
                    | 'is' WS+ 'equal' (WS+ 'to')?
                    | 'equal' (WS+ 'to')? ;

// Keywords - statements
SET                 : 'set' ;
END                 : 'end' ;
ADD                 : 'add' ;
SUM_OF              : 'sum' WS+ 'of' ;
INCREMENT           : 'increment' ;
DECREMENT           : 'decrement' ;
SUBTRACT            : 'subtract' ;
MULTIPLY            : 'multiply' ;
ROUNDED             : 'rounded' ;
DECIMAL_PLACES      : 'decimal' WS+ 'places' ;
WITH_BOUNDRY        : 'with' WS+ 'boundry' ;
REMOVE              : 'remove' ;
FROM                : 'from' ;
ARRAY_OF_VALUES     : 'array' WS+ 'of' WS+ 'values' ;
ELEMENT             : 'element' ;
INCLUDE             : 'include' ;
INCLUDES            : 'includes' ;
ATTRIBUTE           : 'attribute' ;
VALUE               : 'value' ;
NAME                : '$' IDENT_CHAR+ | 'name' ;
LOCAL               : 'local' ;
SUBSTRING           : 'substring' ;
TRIM                : 'trim' ;
INDEX_OF            : 'index' WS+ 'of' ;
MEMBER              : 'member' 's'? ;
THIS                : 'this' ;
CONTEXT             : 'context' ;
FORALL              : 'for' WS* 'all' ;
FOREACH             : 'for' WS* 'each' ;
EACH                : 'each' ;
ALL                 : 'all' ;
PERFORM             : 'perform' ;
IN                  : 'in' ;
TO                  : 'to' ;
IS                  : 'is' | 'are' ;
ITS                 : 'its' ;

// Note: Natural language comparison operators are merged into GT, LT, GTE, LTE, EQ, NEQ above

// Boolean operators
AND                 : 'and' | '&&' ;
OR                  : 'or' | '||' ;
NOT                 : 'not' ;
NO                  : 'no' ;

// Control flow
IF                  : 'if' ;
THEN                : 'then' ;
ENDFF               : 'endff' ;
ENDIF               : 'endif' ;
ELSE                : 'else' ;
ELSEIF              : 'else' WS+ 'if' ;
ELSEIFNONEAREFOUND  : 'else' WS+ 'if' WS+ 'none' WS+ 'are' WS+ 'found' ;

// Other keywords
FIRST               : 'first' ;
OF                  : 'of' ;
ON                  : 'on' ;
USING               : 'using' ;
COPY                : 'copy' ;
DEEPCOPY            : 'deep' WS+ 'copy' ;
GET                 : 'get' ;
SORT                : 'sort' ;
BY                  : 'by' ;
NEW                 : 'new' ;
EARLIEST            : 'earliest' ;
DEBUG               : 'debug' ;
PRINT               : 'print' ;
ERROR               : 'error' ;
WARN                : 'warn' ;
CLEAR               : 'clear' ;
CLONE               : 'clone' ;
FOR                 : 'for' ;
RANDOMIZE           : 'randomize' ;
WAS                 : 'was' ;
ONE                 : 'one' ;
DOES                : 'does' ;
DAYS                : 'day' 's'? ;
ISNULL              : 'is' WS+ 'null' ;
ISNOTNULL           : 'is' WS+ 'not' WS+ 'null' ;
CHANGE              : 'change' ;
UPPER_CASE          : 'upper' WS+ 'case' ;
LOWER_CASE          : 'lower' WS+ 'case' ;
BETWEEN             : 'between' ;
BEFORE              : 'before' ;
AFTER               : 'after' ;
LENGTH              : 'length' ;
THERE               : 'there' ;
NUMBEROF            : 'number' WS+ 'of' ;
THENAME             : 'the' WS+ 'name' ;
RELATIONSHIP_BETWEEN : 'relationship' WS+ 'between' ;
STARTS_WITH         : 'starts' WS+ 'with' ;
ALLOWING            : 'allowing' ;
HAVE                : 'have' ;
YEARS               : 'year' 's'? ;
MONTHS              : 'month' 's'? ;
// Phase 3 of #743: week/quarter unit tokens for bucket and calendar-comparison
// ops. Must precede IDENT so the natural-language form lexes correctly.
WEEKS               : 'week' 's'? ;
QUARTERS            : 'quarter' 's'? ;
TOKENIZE            : 'tokenize' ;
TOBEREMOVED         : 'to' WS+ 'be' WS+ 'removed' ;
TABLEINFORMATION    : 'table' WS+ 'information' ;
WITHIN              : 'with' WS* 'in' ;
PERCENTOF           : 'percent' WS+ 'of' ;
PLUSORMINUS         : 'plus' WS+ 'or' WS+ 'minus' ;
MATCH               : 'match' ;
MATCHES             : 'matches' ;
ONERROR             : 'on' WS+ 'error' ;
ABSOLUTEVALUE       : 'absolute' WS+ 'value' ;
MINIMUM             : 'minimum' WS+ 'of' | 'smaller' WS+ 'of' ;
MAXIMUM             : 'maximum' WS+ 'of' | 'larger' WS+ 'of' ;
HASA                : 'has' WS+ ('a' | 'an') ;
DESCENDINGORDER     : 'descending' WS+ 'order'? ;
ASCENDINGORDER      : 'ascending' WS+ 'order'? ;
WHERE               : 'where' | 'whose' | 'which' | 'while' ;
MAP                 : 'map' ;
MAPPINGKEY          : 'mapping' WS+ 'key' ;
THROUGH             : 'through' ;
YEAROF              : 'yearof' ;
// Phase 3 of #743: time-component / calendar tokens. Single tokens (mirroring
// YEAROF) so the lexer doesn't have to disambiguate against IDENT.
HOUROF              : 'hourof' ;
MINUTEOF            : 'minuteof' ;
SECONDOF            : 'secondof' ;
DAYOFWEEK           : 'day' WS+ 'of' WS+ 'week' | 'dayofweek' ;
WEEKOFYEAR          : 'week' WS+ 'of' WS+ 'year' | 'weekofyear' ;
STARTING            : 'starting' ;
SAME                : 'same' ;
CALENDAR            : 'calendar' ;
NAMEOF              : 'nameof' ;
AT                  : 'at' ;
AS                  : 'as' ;

// Literals
// FP_LITERAL must precede FLOAT_LITERAL / INT_LITERAL so the fp/FP suffix is
// consumed here rather than split into FLOAT_LITERAL + IDENT.
FP_LITERAL          : DIGIT+ '.' DIGIT* 'fp'
                    | DIGIT* '.' DIGIT+ 'fp'
                    | DIGIT+ 'fp'
                    ;
INT_LITERAL         : DIGIT+ ;
FLOAT_LITERAL       : DIGIT+ '.' DIGIT* | DIGIT* '.' DIGIT+ ;
STRING_LITERAL      : '"' ~["]* '"' | '\'' ~[']* '\'' ;
HEX_BYTES_LITERAL   : '0x' HEX_DIGIT* ;

// Articles - ignored (must come before IDENT to take precedence)
ARTICLE             : ('a' | 'an' | 'the') -> skip ;

// Possessive
POSSESSIVE          : IDENT_CHAR+ ('.' IDENT_CHAR+)? '\'' 's' ;

// Identifier - must come after all keywords
IDENT               : IDENT_CHAR+ ('.' IDENT_CHAR+)? ;

// Comments
LINE_COMMENT        : '//' ~[\r\n]* -> skip ;
BLOCK_COMMENT       : '/*' .*? '*/' -> skip ;

// Whitespace
WS                  : [ \t\r\n\f]+ -> skip ;

// Fragments
fragment IDENT_CHAR : [a-z] | [A-Z] | [0-9] | '_' ;
fragment DIGIT      : [0-9] ;
fragment LETTER     : [a-zA-Z] ;
fragment HEX_DIGIT  : [0-9a-fA-F] ;
