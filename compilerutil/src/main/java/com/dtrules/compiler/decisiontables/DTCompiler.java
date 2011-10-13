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

import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.io.PrintStream;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.Iterator;

import com.dtrules.session.ICompiler;
import com.dtrules.xmlparser.GenericXMLParser;
import com.dtrules.xmlparser.IGenericXMLParser;

public class DTCompiler implements IGenericXMLParser{

    private ArrayList<CompileError> errors  = new ArrayList<CompileError>();
    private ArrayList<Changed>      changes = new ArrayList<Changed>();
    

    ICompiler compiler = null;
    int       contextsCompiled         = 0;
    int       initialActionsCompiled   = 0;
    int       conditionsCompiled       = 0;
    int       actionsCompiled          = 0;
    int       policystatementsCompiled = 0;
    String    message                  = null;
    
    boolean   newstatement = true;        //Set to true at start of condition or action
    
    String    tablename;                  //Keep the Decision Table name too.
    String    filename;                   //Filename (if source was Excel)
    String    source;                     //Keep the source after we parse it.
    String    oldpostfix ="None in XML";  //Keep the old postfix after we parse it.
                                          //(I set the oldpostfix to an impossible value to 
                                          // pick up cases were we have no description to compile)
    String    newpostfix;                 //Keep the new postfix after we generate it.
    
    PrintStream out;
    // Indicates that the "printer" is on a newline or not
    boolean newline = true;
    
    public ArrayList<CompileError> getErrors()  { return errors; }
    public ArrayList<Changed>      getChanges() { return changes; }
    
    void printStartTag(String tag, HashMap<String,String> attribs){
        if(!newline)out.println();
        out.print("<"); out.print(tag);
        if(attribs!=null){
            Object [] keys = attribs.keySet().toArray();
            for(int i=0;i<keys.length;i++){
                out.print(" ");
                out.print(keys[i].toString());
                out.print("=\"");
                out.print(GenericXMLParser.encode(attribs.get(keys[i]).toString()));
                out.print('"');
            }
        }  
        out.print(">");
        newline = false;
    }
    
    void printEndTag(String tag, String body){
        if(body!=null) out.print(GenericXMLParser.encode(body));
        out.print("</"); out.print(tag); out.print(">");   
    }
    
    public void beginTag(String[] tagstk, int tagstkptr, String tag, HashMap<String,String> attribs) throws IOException, Exception {
        if(!tag.equals("condition_postfix") && 
           !tag.equals("action_postfix")){
              printStartTag(tag, attribs);
        }
    }

    public void logError(String decisionTableName, String filename,String source, String message, int line, String info){
        errors.add(new CompileError(decisionTableName,filename,source,message,line,info,new ArrayList<String>()));
    }
    /**
     * Used to compile a Context, Initial_Action, Condition, Action, or Policy Statement.
     * Note that the TokenFilter class is going to tack on a semicolon to EVERY statement.  You
     * have to parse for this, or you will get a hard to find error!
     * @param prefix
     * @param body
     */
    private void compileone(String prefix, String body){
        source = GenericXMLParser.unencode(body).trim();
        try{
            if(prefix.equals("action")){
               newpostfix = compiler.compileAction(source);
            }else if(prefix.equals("condition")){
               newpostfix = compiler.compileCondition(source);
            }else if(prefix.equals("context")){
               newpostfix = compiler.compileContext(source);
            }else if(prefix.equals("initial_action")){
               newpostfix = compiler.compileInitialAction(source);
            }else if(prefix.equals("policy_statement")){
               newpostfix = compiler.compilePolicyStatement(source); 
            }
        }catch(Exception e){
            message    = e.toString();
            if(message==null)message = "unknown";
            newpostfix = "";
        }
        printStartTag(prefix+"_postfix", null);
        out.print(GenericXMLParser.encode(newpostfix));
        printEndTag(prefix+"_postfix",null);
    }
    
    /**
     * Log the result of the compile, after comparing if the new result
     * matches the old result.  Note, we can't do this until we reach
     * the end of the description.
     * @param prefix
     */
    private void logResult(String prefix){
        if (message!=null){
            errors.add(new CompileError(tablename,filename,source,message,-1,null,compiler.getParsedTokens()));
            printStartTag("compile_error",null);
            printEndTag("compile_error",message);
            message    = null;
            oldpostfix = null;
        }else if (!oldpostfix.equals(newpostfix)){
            if(prefix.equals("action")){
                actionsCompiled++;
             }else if(prefix.equals("condition")){
                conditionsCompiled++;
             }else if(prefix.equals("context")){
                contextsCompiled++;
             }else if(prefix.equals("initial_action")){
                initialActionsCompiled++;
             }else if(prefix.equals("policy_statement")){
                policystatementsCompiled++;
             }
            changes.add(new Changed(tablename,source,GenericXMLParser.unencode(newpostfix),oldpostfix));
        }
        oldpostfix="None in XML";
    }
    public void endTag(String[] tagstk, int tagstkptr, String tag, String body, HashMap<String,String> attribs) throws Exception, IOException {
        if(!tag.equals("condition_postfix") && 
                !tag.equals("action_postfix")){
                   printEndTag(tag, body);
        }else{
            oldpostfix = GenericXMLParser.unencode(body);
        }
        if(tag.equals("action_description")){
            compileone("action",body);
        }else if(tag.equals("condition_description")){
            compileone("condition",body);
        }else if(tag.equals("context_description")){
            compileone("context",body);
        }else if(tag.equals("initial_action_description")){
        	compileone("initial_action",body);
        }else if(tag.equals("policy_description")){
            compileone("policy_statement",body);
       
        // NOTE!!! You have to LOG your errors if you want them to show up
        // against the right section!
            
        }else if(tag.equals("condition_details" )){ 
            logResult("condition");
        }else if (tag.equals("action_details")){
           logResult("action");
        }else if (tag.equals("initial_action_details")){
           logResult("initial_action");
        }else if (tag.equals("context_details")){
           logResult("context");
        }else if (tag.equals("policy_statement")){
           logResult("policy_statement");
        } else if(tag.equals("table_name")){
            tablename = body;
        } else if(tag.equals("xls_file")){
            filename = body;
        } else if(tag.equals("decision_table")){
            tablename = "";
            filename  = "";
            compiler.newDecisionTable();
        }

    }

    public boolean error(String v) throws Exception {
        return true;
    }
    
    InputStream  dtinput;
    OutputStream dtoutput;
    
    /**
     * Compiles the XML stream, writing the "compiled" XML to result.
     * Note that it is the caller's responsiblity to ask the compiler
     * for a list of errors and to report these errors....
     * 
     * @param source InputStream
     * @param result OutputStream
     * 
     */
    public DTCompiler(ICompiler compiler){
       this.compiler = compiler;        
    }
    
    
    
    public void compile( String decisionTable,String resultDecisionTable){
        try {
            
            InputStream      in  = new FileInputStream(decisionTable);
            FileOutputStream out = new FileOutputStream(resultDecisionTable);
            
            compile(in,out);
        }catch(Exception e){
            System.out.println("Error openning files: \n"+e);
        }
    }
    
    public void compile(InputStream in, OutputStream out) {
        this.out = new PrintStream(out);
            try {
                GenericXMLParser.load(in, this);
            } catch (Exception e) {
                errors.add(new CompileError(
                		
                        tablename,filename,"Error occurred in the XML parsing", e.getMessage(), -1,null
                        ,compiler.getParsedTokens())
                );
            }
    }
    
    public void printErrors(PrintStream eOut){
        printErrors(eOut,0x7fffffff);
    }
    
    public void printTypes(PrintStream eOut){
        try {
            compiler.printTypes(eOut);
        } catch (Exception e) {
            eOut.println(e);
        }   
    }
    
    public void printErrors(PrintStream eOut, int count){
            Iterator<CompileError> err = errors.iterator();
            while (err.hasNext()&&count>0) {
                count--;    
                CompileError error = err.next();
                eOut.println("\nError:");
                eOut.println("Decision Table: " + error.tablename);
                eOut.println("Filename:       " + error.filename);
                eOut.println();
                eOut.println("Source:         " + error.source);
                eOut.println("Error:          " + error.message);
                eOut.println("Info:           " + error.info);
                eOut.println("\nThe following are the tokens that the parser recognized:");
                Iterator<String> tokens = error.tokens.iterator();
                while(tokens.hasNext())eOut.println(tokens.next());
                if(error.lineNumber>=0){
                eOut.println("Line:           " + error.lineNumber);
                }
                eOut.println();
            }
    }
    
    public void printUnreferenced(PrintStream eOut){
            ArrayList<String> unreferenced = compiler.getUnReferenced();
            for(int i=0;i<unreferenced.size()-1;i++){
                for(int j=0;j<unreferenced.size()-1-i;j++){
                    if(unreferenced.get(j).compareTo(unreferenced.get(j+1))>0){
                        String t = unreferenced.get(j);
                        unreferenced.set(j, unreferenced.get(j+1));
                        unreferenced.set(j+1, t);
                    }
                }
            }
            
            eOut.println("Found "+ unreferenced.size()+" Unreferenced attributes: ");
            for (String unref : unreferenced){
               eOut.println("    "+unref);            
            }
            ArrayList<String> attriblist = compiler.getPossibleReferenced();
            for(int i=0;i<attriblist.size()-1;i++){
                for(int j=0;j<attriblist.size()-1-i;j++){
                    if(attriblist.get(j).compareTo(attriblist.get(j+1))>0){
                        String t = attriblist.get(j);
                        attriblist.set(j, attriblist.get(j+1));
                        attriblist.set(j+1, t);
                    }
                }
            }
            
            eOut.println("Found "+ attriblist.size()+" Ambiguous attributes: ");
            for (String attrib : attriblist){
               eOut.println("    "+attrib);            
            }

    }
    
    public void printChanges(PrintStream eOut){
            Iterator<Changed> chg = changes.iterator();
            while (chg.hasNext()) {
                Changed change = chg.next();
                eOut.println("Change:");
                eOut.println("Decision Table: " + change.tablename);
                eOut.println("Source:         " + change.source);
                eOut.println("Old:            " + change.oldPostfix);
                eOut.println("New:            " + change.newPostfix);

            }

            eOut.println("Conditions compiled: " + conditionsCompiled);
            eOut.println("Actions compiled:    " + actionsCompiled);
            eOut.println("Changes:             " + changes.size());
            eOut.println("Errors:              " + errors.size());
            eOut.println();
    }
}
