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
package com.dtrules.samples.chipeligibility;


import java.io.File;
import java.io.FileOutputStream;
import java.io.OutputStream;
import java.io.PrintStream;
import java.util.Date;

import com.dtrules.entity.IREntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RArray;
import com.dtrules.interpreter.RName;
import com.dtrules.mapping.DataMap;
import com.dtrules.session.DTState;
import com.dtrules.session.IRSession;
import com.dtrules.session.RuleSet;
import com.dtrules.session.RulesDirectory;
import com.dtrules.testsupport.ATestHarness;
import com.dtrules.testsupport.Coverage;
import com.dtrules.testsupport.ITestHarness;
import com.dtrules.xmlparser.XMLPrinter;

public class TestChip extends ATestHarness {
    	
		static int threads = 320;
		
	    public static String path    = System.getProperty("user.dir")+"/";
		
		static  Date start = new Date();
	    @Override
    	public boolean  Verbose()                 { return false;	                        }
		public boolean  Trace()                   { return true;                           }
	    public boolean  Console()                 { return false;                           }
	    public boolean  coverageReport()          { return true;                       		}
		public String   getPath()                 { return path;                            }
	    public String   getRulesDirectoryPath()   { return getPath()+"xml/";                }
	    public String   getRuleSetName()          { return "CHIP";                          }
	    public String   getDecisionTableName()    { return "Compute_Eligibility";           }
	    public String   getRulesDirectoryFile()   { return "DTRules.xml";                   }             
	    
	    public static void main(String[] args) {
	    	
	    	ITestHarness t = new TestChip();
	        t.runTests();
	        String fields[] = { "table number" };
	        t.writeDecisionTables("tables",fields,true,10);
	    }
	      
	}    
	    
