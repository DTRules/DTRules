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

public class TestSudoku extends ATestHarness {
    	
		static int threads = 1;
		
	    public static String path    = System.getProperty("user.dir")+"/";
		
		static  Date start = new Date();
	    @Override
    	public boolean  Verbose()                 { return false;	                        }
		public boolean  Trace()                   { return false;                           }
	    public boolean  Console()                 { return true;                            }
	    public boolean  coverageReport()          { return true;                       		}
		public String   getPath()                 { return path;                            }
	    public String   getRulesDirectoryPath()   { return getPath()+"xml/";                }
	    public String   getRuleSetName()          { return "Sudoku";                        }
	    public String   getDecisionTableName()    { return "Solve";                         }
	    public String   getRulesDirectoryFile()   { return "DTRules.xml";                   }             
	   
	    
	    public static void main(String[] args) throws Exception {
	    	CompileSudoku.main(null);
	    	ITestHarness t = new TestSudoku();
	    	t.runTests();
	        String fields[] = { "table number" };
	        t.writeDecisionTables("tables",fields,true,10);
	    }
	    
	    
	    
	    
	    @Override
		public void executeDecisionTables(IRSession session)
				throws RulesException {
			session.getState().setState(DTState.DEBUG);
			super.executeDecisionTables(session);
		}
		IREntity findcell(RArray cells, int row, int col) throws Exception{
	    	for (IRObject cell : cells){
	    		int r = cell.rEntityValue().get("row").intValue();
	    		int c = cell.rEntityValue().get("column").intValue();
	    		if(r==row && c==col)return cell.rEntityValue();
	    	}
	    	return null;
	    }
	    
	    IREntity findv(RArray vals, int row, int col) throws Exception{
	    	for (IRObject val : vals){
	    		int r = val.rEntityValue().get("row").intValue();
	    		int c = val.rEntityValue().get("column").intValue();
	    		if(r==row && c==col)return val.rEntityValue();
	    	}
	    	return null;
	    }
	    
	    @Override
	    public void printReport(int runNumber, IRSession session, PrintStream out)
	    	throws Exception {
	    	RArray cells = session.getState().find("cells").rArrayValue();
	    	String solution= session.getState().find("done")+"\n";
	    	for(int cr = 0; cr < 3; cr++){
	    		solution += "\n";
	    		for(int r = 0; r < 3; r++){
	    			solution += "\n";
	    	    	for(int cc=0; cc < 3; cc++){
	    	    		solution += " ";
	    	    		for(int c=0; c < 3; c++){
	    		    	    IREntity cell = findcell(cells, cr, cc);
	    		    	    if(cell != null){
		    		    	    RArray   vals = cell.get("positions").rArrayValue();
		    		    	    IREntity val  = findcell(vals,r,c);
		    		    	    solution += val.get("value").stringValue()+" ";
	    		    	    }
	    	    		}
	    	    	}
	    		}
	    	}
	    	RArray notes = session.getState().find("notes").rArrayValue();
	    	for(IRObject note :notes){
	    		solution += "\n"+note.stringValue();
	    	}
	    	String verticalLine =
	    		" \n+-------------------------------+-------------------------------+-------------------------------+";
	    	for(int cr = 0; cr < 3; cr++){
    			solution += (cr==0?"":" |")+verticalLine;
	    		for(int r = 0; r < 3; r++){
	    			solution += (r==0?"":" |")+"\n";
	    	    	for(int cc=0; cc < 3; cc++){
	    	    		solution += (cc==0?"":" ")+"|";
	    	    		for(int c=0; c < 3; c++){
	    		    	    IREntity cell = findcell(cells, cr, cc);
	    		    	    if(cell != null){
		    		    	    RArray   vals = cell.get("positions").rArrayValue();
		    		    	    IREntity val  = findcell(vals,r,c);
		    		    	    String p = "";
		    		    	    RArray pvs = val.get("possiblevalues").rArrayValue();
		    		    	    for(IRObject pv : pvs){
		    		    	    	p+=pv.rEntityValue().get("value").stringValue();
		    		    	    }
		    		    	    int v = val.get("value").intValue();
		    		    	    if(p.length()==0 || val.get("value").intValue()>0){
		    		    	    	p="["+v+"]";
		    		    	    }
		    		    	    while(p.length() < 10){
		    		    	    	if((p.length()&1)==1){
		    		    	    		p=" "+p;
		    		    	    	}else{
		    		    	    		p=p+" ";
		    		    	    	}
		    		    	    }
		    		    	    solution += p;
	    		    	    }
	    	    		}
	    	    	}
	    		}
	    	}
			solution += " |"+verticalLine;
	    	
	    	out.println("<solution>\n"+solution+"\n</solution>");
	    }
	    
	}    
	    
