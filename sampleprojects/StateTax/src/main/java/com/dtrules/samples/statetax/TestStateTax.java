/*
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
package com.dtrules.samples.statetax;

import java.io.PrintStream;

import com.dtrules.entity.IREntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RArray;
import com.dtrules.session.IRSession;
import com.dtrules.testsupport.ATestHarness;
import com.dtrules.testsupport.ITestHarness;

public class TestStateTax extends ATestHarness {

    public boolean  Trace()                 { return true;                      }
    public boolean  Console()               { return true;                      }
    public String   getPath()               { return CompileStateTax.path;      }
    public String   getRulesDirectoryPath()  { return getPath() + "xml/";        }
    public String   getRuleSetName()         { return "StateTax";                }
    public String   getDecisionTableName()   { return "Compute_Tax";             }
    public String   getRulesDirectoryFile()  { return "DTRules.xml";             }

    public static void main(String[] args) throws Exception {
        ITestHarness t = new TestStateTax();
        t.runTests();
    }

    @Override
    public void printReport(int runNumber, IRSession session, PrintStream out) throws RulesException {
        out.println("\n=== State Income Tax Results (Test Case " + runNumber + ") ===\n");

        IRObject obj = session.getState().find("job");
        IREntity job = obj.rEntityValue();
        RArray results = job.get("results").rArrayValue();

        for (int i = 0; i < results.size(); i++) {
            IREntity result = results.get(i).rEntityValue();

            String taxpayerId    = result.get("taxpayer_ID").stringValue();
            String filingStatus  = result.get("filing_status").stringValue();
            int    grossIncome   = result.get("grossIncome").intValue();
            int    agi           = result.get("agi").intValue();
            int    deduction     = result.get("deduction").intValue();
            int    exemptions    = result.get("exemptions").intValue();
            int    taxableIncome = result.get("taxableIncome").intValue();
            int    taxOwed       = result.get("taxOwed").intValue();

            out.println("Taxpayer:        " + taxpayerId);
            out.println("Filing Status:   " + filingStatus);
            out.println("Gross Income:    $" + String.format("%,d", grossIncome));
            out.println("AGI:             $" + String.format("%,d", agi));
            out.println("Deduction:       $" + String.format("%,d", deduction));
            out.println("Exemptions:      " + exemptions + " x $4,350 = $" + String.format("%,d", exemptions * 4350));
            out.println("Taxable Income:  $" + String.format("%,d", taxableIncome));
            out.println("Tax Owed:        $" + String.format("%,d", taxOwed));

            if (grossIncome > 0) {
                double effectiveRate = (100.0 * taxOwed) / grossIncome;
                out.println("Effective Rate:  " + String.format("%.2f%%", effectiveRate));
            }

            RArray notes = result.get("notes").rArrayValue();
            if (notes.size() > 0) {
                out.println("\nNotes:");
                for (int j = 0; j < notes.size(); j++) {
                    out.println("  - " + notes.get(j).stringValue());
                }
            }
            out.println();
        }
    }
}
