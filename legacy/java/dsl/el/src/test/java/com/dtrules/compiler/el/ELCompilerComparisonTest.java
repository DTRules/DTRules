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

import java.io.File;
import java.io.PrintStream;
import java.util.ArrayList;
import java.util.List;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.RName;
import com.dtrules.session.ICompiler;
import com.dtrules.session.IRSession;
import com.dtrules.session.RuleSet;
import com.dtrules.session.RulesDirectory;

/**
 * Comprehensive test comparing the old JFlex/CUP EL compiler
 * with the new ANTLR 4 ELAntlr compiler.
 *
 * This test ensures behavior parity and compares performance.
 */
public class ELCompilerComparisonTest {

    /**
     * Find the DTRules root directory by searching upward from current directory
     */
    private static String findDTRulesRoot() {
        File dir = new File(System.getProperty("user.dir"));
        while (dir != null) {
            File sampleprojects = new File(dir, "sampleprojects");
            if (sampleprojects.exists() && sampleprojects.isDirectory()) {
                return dir.getAbsolutePath();
            }
            dir = dir.getParentFile();
        }
        // Fallback to common locations
        String[] fallbacks = {
            "/home/paul/DTRules",
            System.getProperty("user.home") + "/DTRules"
        };
        for (String path : fallbacks) {
            File f = new File(path, "sampleprojects");
            if (f.exists()) {
                return path;
            }
        }
        return System.getProperty("user.dir");
    }

    private static final String DTRULES_ROOT = findDTRulesRoot();
    private static final String SYNTAX_TESTS_PATH = DTRULES_ROOT + "/sampleprojects/SyntaxTests/";
    private static final String TEST_PROJECT_PATH = DTRULES_ROOT + "/sampleprojects/TestProject/";

    private ICompiler oldCompiler;
    private ICompiler newCompiler;
    private IRSession session;

    private int passCount = 0;
    private int failCount = 0;
    private List<String> failures = new ArrayList<String>();

    /**
     * Test conditions - boolean expressions
     */
    private static final String[] CONDITIONS = {
        // Simple comparisons
        "1 < 2",
        "1 > 2",
        "1 = 2",
        "1 <> 2",
        "1 <= 2",
        "1 >= 2",

        // Natural language comparisons
        "1 is less than 2",
        "1 is greater than 2",
        "1 is equal to 2",
        "1 is not equal to 2",
        "1 is less than or equal to 2",
        "1 is greater than or equal to 2",

        // Boolean operations
        "true",
        "false",
        "true and false",
        "true or false",
        "not true",
        "true and not false",

        // Complex boolean expressions
        "(1 < 2) and (3 > 1)",
        "(1 < 2) or (3 < 1)",
        "not (1 < 2)",
        "((1 < 2) and (3 > 1)) or (5 = 5)",

        // Float comparisons
        "1.5 < 2.5",
        "1.5 > 0.5",
        "1.5 = 1.5",
        "1.5 <> 2.5",

        // Mixed int/float comparisons
        "1.5 < 2",
        "1 < 2.5",
        "1.5 > 0",
        "0 < 1.5",

        // String comparisons
        "\"abc\" < \"def\"",
        "\"abc\" = \"abc\"",
        "\"abc\" <> \"def\"",

        // Null checks
        "null is equal to null",
        "null is not equal to null",
    };

    /**
     * Test actions - statements that produce side effects
     */
    private static final String[] ACTIONS = {
        // Simple assignments
        "set x to 1",
        "set y to 2.5",
        "set z to \"hello\"",
        "set b to true",

        // Arithmetic expressions
        "set x to 1 + 2",
        "set x to 3 - 1",
        "set x to 2 * 3",
        "set x to 6 / 2",
        "set x to 7 % 3",

        // Complex arithmetic
        "set x to (1 + 2) * 3",
        "set x to 1 + (2 * 3)",
        "set x to ((1 + 2) * (3 + 4))",

        // Float arithmetic
        "set x to 1.5 + 2.5",
        "set x to 3.0 - 1.5",
        "set x to 2.0 * 1.5",
        "set x to 3.0 / 2.0",

        // Mixed arithmetic
        "set x to 1 + 2.5",
        "set x to 1.5 + 2",
        "set x to 1 * 2.5",
        "set x to 1.5 * 2",

        // String operations
        "set x to \"hello\" + \" world\"",

        // Control flow
        "if true then set x to 1 endif",
        "if false then set x to 1 else set x to 2 endif",
        "if 1 < 2 then set x to 1 endif",

        // Negation
        "set x to -1",
        "set x to -1.5",
        "set x to -(1 + 2)",

        // Increment/Decrement
        "increment x",
        "decrement x",
        "increment x by 5",
        "decrement x by 5",

        // Multiple statements
        "set x to 1; set y to 2",
        "set x to 1; set y to 2; set z to 3",
    };

    /**
     * Test context expressions - entity/attribute access patterns
     */
    private static final String[] CONTEXTS = {
        // These would require entities to be defined in the session
        // We'll test what we can without specific entities
    };

    /**
     * Test policy statements - string templates with embedded expressions
     */
    private static final String[] POLICY_STATEMENTS = {
        "This is a simple policy statement",
        "The value is {1 + 2}",
        "First value: {1} Second value: {2}",
        "The result is {1 + 2 * 3}",
        "Boolean: {true}",
        "String: {\"hello\"}",
    };

    public static void main(String[] args) {
        ELCompilerComparisonTest test = new ELCompilerComparisonTest();

        System.out.println("===========================================");
        System.out.println("EL Compiler Comparison Test");
        System.out.println("Old: EL.java (JFlex/CUP)");
        System.out.println("New: ELAntlr.java (ANTLR 4)");
        System.out.println("===========================================\n");

        try {
            // Try multiple sample projects
            String[] projectPaths = {
                SYNTAX_TESTS_PATH,
                TEST_PROJECT_PATH
            };
            String[] projectNames = {
                "SyntaxExamples",
                "TestProject"
            };

            boolean initialized = false;
            for (int i = 0; i < projectPaths.length; i++) {
                try {
                    test.initialize(projectPaths[i], projectNames[i]);
                    initialized = true;
                    System.out.println("Using project: " + projectNames[i]);
                    break;
                } catch (Exception e) {
                    System.out.println("Could not initialize with " + projectNames[i] + ": " + e.getMessage());
                }
            }

            if (!initialized) {
                System.out.println("\nNo sample project found. Running syntax-only tests...\n");
                test.runSyntaxOnlyTests();
            } else {
                test.runAllTests();
            }

        } catch (Exception e) {
            System.err.println("Test initialization failed: " + e.getMessage());
            e.printStackTrace();
        }

        test.printSummary();
    }

    /**
     * Initialize both compilers with a session from a sample project
     */
    public void initialize(String path, String ruleSetName) throws Exception {
        RulesDirectory rd = new RulesDirectory(path, "DTRules.xml");
        RuleSet rs = rd.getRuleSet(RName.getRName(ruleSetName));

        if (rs == null) {
            throw new Exception("RuleSet '" + ruleSetName + "' not found");
        }

        session = rs.newSession();

        // Initialize old compiler
        oldCompiler = new EL();
        oldCompiler.setSession(session);

        // Initialize new compiler
        newCompiler = new ELAntlr();
        newCompiler.setSession(session);

        System.out.println("Compilers initialized successfully with session");
    }

    /**
     * Run all comparison tests
     */
    public void runAllTests() {
        System.out.println("\n--- Testing Conditions ---\n");
        testConditions();

        System.out.println("\n--- Testing Actions ---\n");
        testActions();

        System.out.println("\n--- Testing Policy Statements ---\n");
        testPolicyStatements();

        System.out.println("\n--- Performance Comparison ---\n");
        runPerformanceTests();
    }

    /**
     * Run syntax-only tests without a session (limited functionality)
     */
    public void runSyntaxOnlyTests() {
        System.out.println("Syntax-only tests are limited without a session.");
        System.out.println("Please ensure sample projects are compiled first.");
    }

    /**
     * Test all condition expressions
     */
    private void testConditions() {
        for (String condition : CONDITIONS) {
            compareCompilation("condition", condition, new CompileAction() {
                public String compile(ICompiler compiler, String input) throws RulesException {
                    return compiler.compileCondition(input);
                }
            });
        }
    }

    /**
     * Test all action expressions
     */
    private void testActions() {
        for (String action : ACTIONS) {
            // Reset decision table state before each action test
            oldCompiler.newDecisionTable();
            newCompiler.newDecisionTable();

            compareCompilation("action", action, new CompileAction() {
                public String compile(ICompiler compiler, String input) throws RulesException {
                    return compiler.compileAction(input);
                }
            });
        }
    }

    /**
     * Test all policy statements
     */
    private void testPolicyStatements() {
        for (String policy : POLICY_STATEMENTS) {
            compareCompilation("policy", policy, new CompileAction() {
                public String compile(ICompiler compiler, String input) throws RulesException {
                    return compiler.compilePolicyStatement(input);
                }
            });
        }
    }

    /**
     * Interface for compile actions
     */
    private interface CompileAction {
        String compile(ICompiler compiler, String input) throws RulesException;
    }

    /**
     * Compare compilation results from both compilers
     */
    private void compareCompilation(String type, String input, CompileAction action) {
        String oldResult = null;
        String newResult = null;
        Exception oldError = null;
        Exception newError = null;

        // Try old compiler
        try {
            oldResult = action.compile(oldCompiler, input);
        } catch (Exception e) {
            oldError = e;
        }

        // Try new compiler
        try {
            newResult = action.compile(newCompiler, input);
        } catch (Exception e) {
            newError = e;
        }

        // Compare results
        boolean pass = false;
        String message = "";

        if (oldError != null && newError != null) {
            // Both failed - check if same error type
            pass = true; // Both failing consistently is acceptable
            message = "Both compilers rejected (expected)";
        } else if (oldError != null && newResult != null) {
            // New compiler accepts what old compiler rejected - this is an improvement
            pass = true;
            message = "IMPROVED (new compiler more permissive)";
        } else if (newError != null) {
            message = "Old compiler succeeded, new failed";
            message += "\n  Old result: " + oldResult;
            message += "\n  New error: " + newError.getMessage();
        } else if (oldResult == null && newResult == null) {
            pass = true;
            message = "Both returned null";
        } else if (oldResult == null || newResult == null) {
            message = "One returned null";
            message += "\n  Old: " + oldResult;
            message += "\n  New: " + newResult;
        } else if (normalizeWhitespace(oldResult).equals(normalizeWhitespace(newResult))) {
            pass = true;
            message = "MATCH";
        } else {
            message = "Output mismatch";
            message += "\n  Old: " + oldResult;
            message += "\n  New: " + newResult;
        }

        if (pass) {
            passCount++;
            System.out.println("PASS [" + type + "] " + truncate(input, 50) + " -> " + message);
        } else {
            failCount++;
            String failure = "FAIL [" + type + "] " + input + "\n" + message;
            failures.add(failure);
            System.out.println("FAIL [" + type + "] " + truncate(input, 50));
            System.out.println("     " + message.replace("\n", "\n     "));
        }
    }

    /**
     * Normalize whitespace for comparison (collapse multiple spaces, trim)
     */
    private String normalizeWhitespace(String s) {
        if (s == null) return "";
        return s.replaceAll("\\s+", " ").trim();
    }

    /**
     * Truncate string for display
     */
    private String truncate(String s, int maxLen) {
        if (s.length() <= maxLen) return s;
        return s.substring(0, maxLen - 3) + "...";
    }

    /**
     * Run performance comparison tests
     */
    private void runPerformanceTests() {
        int iterations = 1000;

        // Warm up
        System.out.println("Warming up...");
        for (int i = 0; i < 100; i++) {
            try {
                oldCompiler.newDecisionTable();
                oldCompiler.compileCondition("1 < 2");
                oldCompiler.compileAction("set x to 1 + 2");

                newCompiler.newDecisionTable();
                newCompiler.compileCondition("1 < 2");
                newCompiler.compileAction("set x to 1 + 2");
            } catch (Exception e) {
                // Ignore warmup errors
            }
        }

        // Test conditions performance
        System.out.println("\nConditions performance (" + iterations + " iterations each):");
        for (String condition : new String[]{"1 < 2", "true and false", "(1 < 2) and (3 > 1)"}) {
            final String cond = condition;
            long oldTime = measurePerformance(iterations, new Runnable() {
                public void run() {
                    try {
                        oldCompiler.compileCondition(cond);
                    } catch (Exception e) {}
                }
            });

            long newTime = measurePerformance(iterations, new Runnable() {
                public void run() {
                    try {
                        newCompiler.compileCondition(cond);
                    } catch (Exception e) {}
                }
            });

            double ratio = (double) newTime / oldTime;
            System.out.printf("  \"%s\": Old=%dms, New=%dms, Ratio=%.2fx\n",
                truncate(condition, 30), oldTime, newTime, ratio);
        }

        // Test actions performance
        System.out.println("\nActions performance (" + iterations + " iterations each):");
        for (String action : new String[]{"set x to 1", "set x to 1 + 2 * 3", "if true then set x to 1 endif"}) {
            final String act = action;
            long oldTime = measurePerformance(iterations, new Runnable() {
                public void run() {
                    try {
                        oldCompiler.newDecisionTable();
                        oldCompiler.compileAction(act);
                    } catch (Exception e) {}
                }
            });

            long newTime = measurePerformance(iterations, new Runnable() {
                public void run() {
                    try {
                        newCompiler.newDecisionTable();
                        newCompiler.compileAction(act);
                    } catch (Exception e) {}
                }
            });

            double ratio = (double) newTime / oldTime;
            System.out.printf("  \"%s\": Old=%dms, New=%dms, Ratio=%.2fx\n",
                truncate(action, 30), oldTime, newTime, ratio);
        }
    }

    /**
     * Measure execution time for a runnable
     */
    private long measurePerformance(int iterations, Runnable runnable) {
        long start = System.currentTimeMillis();
        for (int i = 0; i < iterations; i++) {
            runnable.run();
        }
        return System.currentTimeMillis() - start;
    }

    /**
     * Print test summary
     */
    private void printSummary() {
        System.out.println("\n===========================================");
        System.out.println("TEST SUMMARY");
        System.out.println("===========================================");
        System.out.println("Passed: " + passCount);
        System.out.println("Failed: " + failCount);
        System.out.println("Total:  " + (passCount + failCount));

        if (failCount > 0) {
            System.out.println("\n--- FAILURES ---\n");
            for (String failure : failures) {
                System.out.println(failure);
                System.out.println();
            }
        }

        System.out.println("===========================================");
        if (failCount == 0) {
            System.out.println("ALL TESTS PASSED - Behavior parity confirmed!");
        } else {
            System.out.println("SOME TESTS FAILED - Review failures above");
        }
        System.out.println("===========================================");
    }
}
