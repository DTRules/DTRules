/** 
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
 **/ 
package com.dtrules.samples.test;


import java.io.PrintStream;
import com.dtrules.entity.IREntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RArray;
import com.dtrules.session.IRSession;
import com.dtrules.testsupport.ATestHarness;
import com.dtrules.testsupport.ITestHarness;
import com.dtrules.xmlparser.XMLPrinter;

public class Test_Test extends ATestHarness {
    
	    public boolean  Trace()                   { return true;                            }
	    public boolean  Console()                 { return true;                            }
	    public String   getPath()                 { return Compile_Test.path;               }
	    public String   getRulesDirectoryPath()   { return getPath()+"xml/";                }
	    public String   getRuleSetName()          { return "TestProject";                   }
	    public String   getDecisionTableName()    { return "Test_Entry_Point";              }
	    public String   getRulesDirectoryFile()   { return "DTRules.xml";                   }             
	   
	    public static void main(String[] args) {
	        ITestHarness t = new Test_Test();
	        t.runTests();
	    }
	    
	}    
	    
