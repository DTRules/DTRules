/** 
 * Copyright 2004-2011 DTRules.com, Inc.
 * 
 * See http://DTRules.com for updates and documentation for the DTRules Rules Engine  
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
package com.dtrules.compiler.decisiontables;

import java.util.ArrayList;

/**
 * A simple class, really just a structure, for collecting 
 * information about reported errors as we compile a set of 
 * decision tables.
 * 
 * @author Paul Snow
 *
 */
public class CompileError {

    public String            tablename;       // Name of the decision table with the error
    public String            filename;        // Name of a Excel file if that was the source
    public String            source;          // The text we were trying to compile
    public String            message;         // Error message generated
    public int               lineNumber;      // Line number in the XML file
    public String            info;            // Any other information we might have.
    public ArrayList<String> tokens;          // List of parsed Tokens
    /**
     * Store all the info when we create an instance of a CompileError
     * @param dtname String decision table name
     * @param src    String the code being compiled
     * @param msg    String the error message generated
     * @param line   int    the line in the XML where the error was detected.
     * @param info   String any other information we might have.
     */
    CompileError(String dtname, String filename,String src, String msg, int line, String info, ArrayList<String> _tokens){
        tablename       = dtname==null?"":dtname;
        this.filename   = filename;
        source          = src==null?"":src;
        message         = msg==null?"":msg;
        lineNumber      = line;
        this.info       = info==null?"":info;
        tokens          = _tokens;
        
        if(tokens.size()>0){
        	String last = tokens.get(tokens.size()-1);
        	String text = last.substring(last.indexOf(" : ")+3);
        	if(last.startsWith("UNDEFINED")){
        		this.info = "'"+text+"' is undefined"; 
        	}else{
        		if(message.indexOf("*=> "+text)<0){
        			this.info = "Unbalanced quotes or unexpected end of statement occured (All the tokens parsed are good)";
        		}else{
        			this.info = "The parser didn't expect the token '"+text+"'";
        		}
        	}
        }
        
    }
}
