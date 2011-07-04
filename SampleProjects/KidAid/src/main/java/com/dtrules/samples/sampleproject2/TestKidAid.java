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
package com.dtrules.samples.sampleproject2;


import java.io.PrintStream;
import com.dtrules.entity.IREntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RArray;
import com.dtrules.session.IRSession;
import com.dtrules.testsupport.ATestHarness;
import com.dtrules.testsupport.ITestHarness;
import com.dtrules.xmlparser.XMLPrinter;

public class TestKidAid extends ATestHarness {
    
	    public boolean  Trace()                   { return true;                            }
	    public boolean  Console()                 { return true;                            }
	    public String   getPath()                 { return CompileKidAid.path;              }
	    public String   getRulesDirectoryPath()   { return getPath()+"xml/";                }
	    public String   getRuleSetName()          { return "KidAid";                        }
	    public String   getDecisionTableName()    { return "Compute_Eligibility";           }
	    public String   getRulesDirectoryFile()   { return "DTRules.xml";                   }             
	   
	    public static void main(String[] args) {
	        ITestHarness t = new TestKidAid();
	        t.runTests();
	    }
	    
	    public void printReport(int runNumber, IRSession session, PrintStream _out) throws RulesException {
	        XMLPrinter xout = new XMLPrinter(_out);
	        xout.opentag("results","runNumber",runNumber);
	        RArray results = session.getState().find("job.results").rArrayValue();
	        for(IRObject r :results){
	            IREntity result = r.rEntityValue();

	            xout.opentag("Client","id",result.get("client_id").stringValue());
	            prt(xout,result,"totalGroupIncome");
	            prt(xout,result,"client_fpl");
	            if(result.get("eligible").booleanValue()){
	                xout.opentag("Approved");
		                prt(xout,result,"program");
		                prt(xout,result,"programLevel");
		                RArray notes = result.get("notes").rArrayValue();
		                xout.opentag("Notes");
		                    for(IRObject n : notes){
		                       xout.printdata("note",n.stringValue());
		                    }
	                    xout.closetag();
	                xout.closetag();
	            }else{
	                xout.opentag("NotApproved");
	                    prt(xout,result,"program");
	                    RArray notes = result.get("notes").rArrayValue();
	                    xout.opentag("Notes");
	                        for(IRObject n : notes){
	                           xout.printdata("note",n.stringValue());
	                        }
	                    xout.closetag();
	                xout.closetag();
	            }
	            xout.closetag();
	        }
	        xout.close();
	    }
	 
	    private void prt(XMLPrinter xout, IREntity entity, String attrib){
	        IRObject value = entity.get(attrib);
	        xout.printdata(attrib,value);
	    }
	    
	}    
	    
