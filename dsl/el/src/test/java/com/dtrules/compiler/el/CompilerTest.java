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

import static org.junit.Assert.*;

import java.io.File;
import java.util.Arrays;
import java.util.Collection;

import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.junit.runners.Parameterized;
import org.junit.runners.Parameterized.Parameters;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.RName;
import com.dtrules.session.ICompiler;
import com.dtrules.session.IRSession;
import com.dtrules.session.RuleSet;
import com.dtrules.session.RulesDirectory;

/**
 * Comprehensive test suite for DTRules compilers.
 *
 * This parameterized test runs the same tests against both:
 * - EL (JFlex/CUP implementation)
 * - ELAntlr (ANTLR 4 implementation)
 *
 * Tests cover:
 * - Conditions (boolean expressions)
 * - Actions (statements with entity attributes)
 * - Policy statements (string templates)
 * - Error handling
 */
@RunWith(Parameterized.class)
public class CompilerTest {

    private final String compilerName;
    private final Class<? extends ICompiler> compilerClass;
    private ICompiler compiler;
    private IRSession session;

    private static final String TEST_PROJECT_PATH = findTestProjectPath();

    @Parameters
    public static Collection<Object[]> compilers() {
        return Arrays.asList(new Object[][] {
            { "EL (JFlex/CUP)", EL.class },
            { "ELAntlr (ANTLR 4)", ELAntlr.class }
        });
    }

    public CompilerTest(String name, Class<? extends ICompiler> compilerClass) {
        this.compilerName = name;
        this.compilerClass = compilerClass;
    }

    private static String findTestProjectPath() {
        String[] paths = {
            System.getProperty("user.dir") + "/../../sampleprojects/SyntaxTests/",
            System.getProperty("user.dir") + "/sampleprojects/SyntaxTests/",
            System.getProperty("user.home") + "/DTRules/sampleprojects/SyntaxTests/",
            "/home/paul/DTRules/sampleprojects/SyntaxTests/"
        };
        for (String path : paths) {
            File f = new File(path + "DTRules.xml");
            if (f.exists()) {
                return path;
            }
        }
        return paths[paths.length - 1];
    }

    @Before
    public void setUp() throws Exception {
        RulesDirectory rd = new RulesDirectory(TEST_PROJECT_PATH, "DTRules.xml");
        RuleSet rs = rd.getRuleSet(RName.getRName("SyntaxExamples"));
        if (rs == null) {
            throw new RuntimeException("Could not load SyntaxExamples from " + TEST_PROJECT_PATH);
        }
        session = rs.newSession();
        compiler = compilerClass.getDeclaredConstructor().newInstance();
        compiler.setSession(session);
        compiler.newDecisionTable();
    }

    // ========================================================================
    // CONDITION TESTS - Numeric Comparisons
    // ========================================================================

    @Test
    public void testCondition_IntLessThan() throws RulesException {
        String result = compiler.compileCondition("1 < 2");
        assertNotNull("Should compile", result);
        assertFalse("Should not be empty", result.trim().isEmpty());
    }

    @Test
    public void testCondition_IntGreaterThan() throws RulesException {
        String result = compiler.compileCondition("2 > 1");
        assertNotNull(result);
    }

    @Test
    public void testCondition_IntLessOrEqual() throws RulesException {
        String result = compiler.compileCondition("1 <= 2");
        assertNotNull(result);
    }

    @Test
    public void testCondition_IntGreaterOrEqual() throws RulesException {
        String result = compiler.compileCondition("2 >= 1");
        assertNotNull(result);
    }

    @Test
    public void testCondition_IntEqual() throws RulesException {
        String result = compiler.compileCondition("1 == 1");
        assertNotNull(result);
    }

    @Test
    public void testCondition_IntNotEqual() throws RulesException {
        String result = compiler.compileCondition("1 != 2");
        assertNotNull(result);
    }

    // ========================================================================
    // CONDITION TESTS - Natural Language Comparisons
    // ========================================================================

    @Test
    public void testCondition_NL_IsLessThan() throws RulesException {
        String result = compiler.compileCondition("1 is less than 2");
        assertNotNull(result);
    }

    @Test
    public void testCondition_NL_IsGreaterThan() throws RulesException {
        String result = compiler.compileCondition("2 is greater than 1");
        assertNotNull(result);
    }

    @Test
    public void testCondition_NL_IsEqualTo() throws RulesException {
        String result = compiler.compileCondition("1 is equal to 1");
        assertNotNull(result);
    }

    @Test
    public void testCondition_NL_IsNotEqualTo() throws RulesException {
        String result = compiler.compileCondition("1 is not equal to 2");
        assertNotNull(result);
    }

    @Test
    public void testCondition_NL_IsLessThanOrEqualTo() throws RulesException {
        String result = compiler.compileCondition("1 is less than or equal to 2");
        assertNotNull(result);
    }

    @Test
    public void testCondition_NL_IsGreaterThanOrEqualTo() throws RulesException {
        String result = compiler.compileCondition("2 is greater than or equal to 1");
        assertNotNull(result);
    }

    // ========================================================================
    // CONDITION TESTS - Boolean Literals and Operations
    // ========================================================================

    @Test
    public void testCondition_BoolTrue() throws RulesException {
        String result = compiler.compileCondition("true");
        assertNotNull(result);
    }

    @Test
    public void testCondition_BoolFalse() throws RulesException {
        String result = compiler.compileCondition("false");
        assertNotNull(result);
    }

    @Test
    public void testCondition_BoolAnd() throws RulesException {
        String result = compiler.compileCondition("true and false");
        assertNotNull(result);
    }

    @Test
    public void testCondition_BoolOr() throws RulesException {
        String result = compiler.compileCondition("true or false");
        assertNotNull(result);
    }

    @Test
    public void testCondition_BoolNot() throws RulesException {
        String result = compiler.compileCondition("not true");
        assertNotNull(result);
    }

    @Test
    public void testCondition_BoolComplex() throws RulesException {
        String result = compiler.compileCondition("(1 < 2) and (3 > 1)");
        assertNotNull(result);
    }

    @Test
    public void testCondition_BoolNested() throws RulesException {
        String result = compiler.compileCondition("((1 < 2) and (3 > 1)) or (5 > 4)");
        assertNotNull(result);
    }

    @Test
    public void testCondition_NotComplex() throws RulesException {
        String result = compiler.compileCondition("not (1 < 2)");
        assertNotNull(result);
    }

    // ========================================================================
    // CONDITION TESTS - Float Comparisons
    // ========================================================================

    @Test
    public void testCondition_FloatLessThan() throws RulesException {
        String result = compiler.compileCondition("1.5 < 2.5");
        assertNotNull(result);
    }

    @Test
    public void testCondition_FloatGreaterThan() throws RulesException {
        String result = compiler.compileCondition("2.5 > 1.5");
        assertNotNull(result);
    }

    @Test
    public void testCondition_MixedIntFloat() throws RulesException {
        String result = compiler.compileCondition("1.5 < 2");
        assertNotNull(result);
    }

    @Test
    public void testCondition_MixedFloatInt() throws RulesException {
        String result = compiler.compileCondition("1 < 2.5");
        assertNotNull(result);
    }

    // ========================================================================
    // CONDITION TESTS - String Comparisons
    // ========================================================================

    @Test
    public void testCondition_StringLessThan() throws RulesException {
        String result = compiler.compileCondition("\"abc\" < \"def\"");
        assertNotNull(result);
    }

    @Test
    public void testCondition_StringEqual() throws RulesException {
        String result = compiler.compileCondition("\"abc\" is equal to \"abc\"");
        assertNotNull(result);
    }

    @Test
    public void testCondition_StringNotEqual() throws RulesException {
        String result = compiler.compileCondition("\"abc\" is not equal to \"def\"");
        assertNotNull(result);
    }

    // ========================================================================
    // CONDITION TESTS - Arithmetic in Conditions
    // ========================================================================

    @Test
    public void testCondition_ArithmeticAdd() throws RulesException {
        String result = compiler.compileCondition("1 + 2 < 5");
        assertNotNull(result);
    }

    @Test
    public void testCondition_ArithmeticMul() throws RulesException {
        String result = compiler.compileCondition("2 * 3 > 5");
        assertNotNull(result);
    }

    @Test
    public void testCondition_ArithmeticComplex() throws RulesException {
        String result = compiler.compileCondition("(1 + 2) * 3 > 5");
        assertNotNull(result);
    }

    @Test
    public void testCondition_NegativeNumber() throws RulesException {
        String result = compiler.compileCondition("-1 < 0");
        assertNotNull(result);
    }

    // ========================================================================
    // POLICY STATEMENT TESTS
    // ========================================================================

    @Test
    public void testPolicy_Simple() throws RulesException {
        String result = compiler.compilePolicyStatement("This is a test");
        assertNotNull(result);
        assertTrue(result.contains("This is a test"));
    }

    @Test
    public void testPolicy_WithExpression() throws RulesException {
        String result = compiler.compilePolicyStatement("Value is {1 + 2}");
        assertNotNull(result);
    }

    @Test
    public void testPolicy_MultipleExpressions() throws RulesException {
        String result = compiler.compilePolicyStatement("A={1} B={2}");
        assertNotNull(result);
    }

    @Test
    public void testPolicy_ComplexExpression() throws RulesException {
        String result = compiler.compilePolicyStatement("Result: {1 + 2 * 3}");
        assertNotNull(result);
    }

    @Test
    public void testPolicy_BooleanExpression() throws RulesException {
        String result = compiler.compilePolicyStatement("Bool: {true}");
        assertNotNull(result);
    }

    @Test
    public void testPolicy_Empty() throws RulesException {
        String result = compiler.compilePolicyStatement("");
        assertNotNull(result);
    }

    @Test
    public void testPolicy_Null() throws RulesException {
        String result = compiler.compilePolicyStatement(null);
        assertEquals("", result);
    }

    // ========================================================================
    // CONTEXT TESTS
    // ========================================================================

    @Test
    public void testContext_ForAllClients() throws RulesException {
        // Standard iteration context
        String result = compiler.compileContext("for all clients");
        assertNotNull(result);
    }

    @Test
    public void testContext_ForAllWithCondition() throws RulesException {
        // Iteration with where clause
        String result = compiler.compileContext("for all clients whose age == 18");
        assertNotNull(result);
    }

    @Test
    public void testContext_LocalEntity() throws RulesException {
        // Local variable declaration
        String result = compiler.compileContext("local entity FirstClient");
        assertNotNull(result);
    }

    @Test
    public void testContext_LocalLong() throws RulesException {
        // Local integer variable
        String result = compiler.compileContext("local long Anumber = 1234");
        assertNotNull(result);
    }

    @Test
    public void testContext_LocalBoolean() throws RulesException {
        // Local boolean variable
        String result = compiler.compileContext("local boolean Test = true");
        assertNotNull(result);
    }

    // ========================================================================
    // ACTION TESTS - Using Real Attributes from SyntaxTests EDD
    // ========================================================================

    @Test
    public void testAction_SetBooleanAttribute() throws RulesException {
        compiler.newDecisionTable();
        // Set a boolean attribute (eligible is rw on client)
        String result = compiler.compileAction("set eligible = true");
        assertNotNull(result);
    }

    @Test
    public void testAction_SetIntegerAttribute() throws RulesException {
        compiler.newDecisionTable();
        // Set an integer attribute (totalgroupincome is rw on client)
        String result = compiler.compileAction("set totalgroupincome = 30400");
        assertNotNull(result);
    }

    @Test
    public void testAction_SetDoubleAttribute() throws RulesException {
        compiler.newDecisionTable();
        // Set a double attribute (totalincome is rw on client)
        String result = compiler.compileAction("set totalincome = 1500.55");
        assertNotNull(result);
    }

    @Test
    public void testAction_SetStringAttribute() throws RulesException {
        compiler.newDecisionTable();
        // Set a string attribute (state is rw on address)
        String result = compiler.compileAction("set state = 'Wyoming'");
        assertNotNull(result);
    }

    @Test
    public void testAction_ForAllWithSet() throws RulesException {
        compiler.newDecisionTable();
        // Loop with set action
        String result = compiler.compileAction("for all clients set eligible = true");
        assertNotNull(result);
    }

    @Test
    public void testAction_ForAllWithCondition() throws RulesException {
        compiler.newDecisionTable();
        // Loop with where clause and set action
        String result = compiler.compileAction("for all clients where age < 18 set eligible = false");
        assertNotNull(result);
    }

    @Test
    public void testAction_AddToNumeric() throws RulesException {
        compiler.newDecisionTable();
        // Add value to numeric attribute
        String result = compiler.compileAction("Add 42 to totalincome");
        assertNotNull(result);
    }

    @Test
    public void testAction_SubtractFromNumeric() throws RulesException {
        compiler.newDecisionTable();
        // Subtract value from numeric attribute
        String result = compiler.compileAction("Subtract 42 from totalincome");
        assertNotNull(result);
    }

    @Test
    public void testAction_IfThenEndif() throws RulesException {
        compiler.newDecisionTable();
        // Conditional action - note: statements inside if/then require semicolons
        String result = compiler.compileAction("if true then set eligible = true ; endif");
        assertNotNull(result);
    }

    @Test
    public void testAction_IfThenElseEndif() throws RulesException {
        compiler.newDecisionTable();
        // Conditional action with else - semicolons required after each statement
        String result = compiler.compileAction("if false then set eligible = true ; else set eligible = false ; endif");
        assertNotNull(result);
    }

    // ========================================================================
    // ERROR HANDLING TESTS
    // ========================================================================

    @Test(expected = RulesException.class)
    public void testError_UnmatchedParen() throws RulesException {
        compiler.compileCondition("(1 < 2");
    }

    @Test(expected = RulesException.class)
    public void testError_UnmatchedQuote() throws RulesException {
        compiler.compileCondition("\"hello");
    }

    @Test(expected = RuntimeException.class)
    public void testError_UnbalancedBraces() throws RulesException {
        compiler.compilePolicyStatement("Value: {1 + 2");
    }

    // ========================================================================
    // COMPILER STATE TESTS
    // ========================================================================

    @Test
    public void testState_GetTypes() {
        assertNotNull(compiler.getTypes());
        assertFalse(compiler.getTypes().isEmpty());
    }

    @Test
    public void testState_NewDecisionTable() throws RulesException {
        compiler.newDecisionTable();
        String result1 = compiler.compileCondition("true");
        compiler.newDecisionTable();
        String result2 = compiler.compileCondition("true");
        assertNotNull(result1);
        assertNotNull(result2);
    }

    @Test
    public void testState_GetUnReferenced() {
        assertNotNull(compiler.getUnReferenced());
    }

    @Test
    public void testState_GetPossibleReferenced() {
        assertNotNull(compiler.getPossibleReferenced());
    }

    // ========================================================================
    // NUMERIC EDGE CASES
    // ========================================================================

    @Test
    public void testEdge_Zero() throws RulesException {
        String result = compiler.compileCondition("0 < 1");
        assertNotNull(result);
    }

    @Test
    public void testEdge_LargeNumber() throws RulesException {
        String result = compiler.compileCondition("999999 > 0");
        assertNotNull(result);
    }

    @Test
    public void testEdge_SmallFloat() throws RulesException {
        String result = compiler.compileCondition("0.001 < 0.01");
        assertNotNull(result);
    }

    @Test
    public void testEdge_NegativeFloat() throws RulesException {
        String result = compiler.compileCondition("-1.5 < 0");
        assertNotNull(result);
    }
}
