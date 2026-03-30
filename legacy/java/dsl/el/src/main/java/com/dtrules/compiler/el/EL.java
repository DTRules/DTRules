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

import java.io.ByteArrayInputStream;
import java.io.DataInputStream;
import java.io.InputStream;
import java.io.PrintStream;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.Iterator;

import javax.management.RuntimeErrorException;

import com.dtrules.compiler.el.ELType;
import com.dtrules.compiler.el.cup.parser.DTRulesParser;
import com.dtrules.compiler.el.cup.parser.RLocalType;
import com.dtrules.compiler.el.flex.scanner.DTRulesscanner;
import com.dtrules.entity.IREntity;
import com.dtrules.entity.REntity;
import com.dtrules.entity.REntityEntry;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RName;
import com.dtrules.session.EntityFactory;
import com.dtrules.session.ICompiler;
import com.dtrules.session.IRSession;
import com.dtrules.session.IRType;

public class EL implements ICompiler {
    
    private       HashMap<RName,IRType>    types = null;
    private       EntityFactory            ef;
    private       IRSession                session;
       
    private       int                      localcnt = 0;  // Count of local variables 
                  HashMap<String,RLocalType> localtypes = new HashMap<String,RLocalType>();
       
    public void setTableName(String tablename) {
        
    }
    
    
    /**
     * Add a identifier and type to the types HashMap.
     * @param entity
     * @param name
     * @param itype
     * @throws Exception
     */
    private void addType( IREntity entity, RName name, int itype) throws Exception {
        ELType type =  (ELType) types.get(name);
        if(type==null){
            type      = new ELType(name,itype,entity);
            types.put(name,type);
        }else{
            if(type.getType()!=itype){
                String entitylist = entity.getName().stringValue();
                for(int i=0; i<type.getEntities().size(); i++){
                    entitylist += " "+type.getEntities().get(i).getName().stringValue();
                }
                throw new Exception("Conflicting types for attribute "+name+" on "+entitylist);
            }
            type.addEntityAttribute(entity);
        }

    }
    /**
     * Get all the types out of the Entity Factory and make them
     * available to the compiler.
     * @param ef
     * @return
     * @throws Exception
     */
    public HashMap<RName,IRType> getTypes(EntityFactory ef) throws Exception {
        
        if(types!=null)return types;
        
        types = new HashMap<RName, IRType>();
        Iterator<RName> entities = ef.getEntityRNameIterator();
        while(entities.hasNext()){
            RName     name    = entities.next();
            IREntity  entity  = ef.findRefEntity(name);
            Iterator<RName> attribs = entity.getAttributeIterator();
            addType(entity,entity.getName(),IRObject.iEntity);
            while(attribs.hasNext()){
                RName        attribname = attribs.next();
                REntityEntry entry      = entity.getEntry(attribname);
                addType(entity,attribname,entry.type.getId());
            }
        }
        
        Iterator<RName> tables = ef.getDecisionTableRNameIterator();
        while(tables.hasNext()){
            RName tablename = tables.next();
            ELType type = new ELType(tablename,IRObject.iDecisiontable,(REntity) ef.getDecisiontables());
            if(types.containsKey(tablename)){
                System.out.println("Multiple Decision Tables found with the name '"+types.get(tablename)+"'");
            }
            types.put(tablename, type);
        }
        
        return types;
    }
    
    /**
     * Prints all the types known to the compiler
     */
    public void printTypes(PrintStream out) throws RulesException {
        Object typenames[] = types.keySet().toArray();
        for(int i=0;i<typenames.length-1;i++){
            for(int j=0;j<typenames.length-1;j++){
                RName one = (RName)typenames[j], two = (RName)typenames[j+1];
                if(one.compare(two)>0){
                    Object hold = typenames[j];
                    typenames[j]=typenames[j+1];
                    typenames[j+1]=hold;
                }
            }
        }
        for(int i=0;i<typenames.length-1;i++){
            for(int j=0;j<typenames.length-1;j++){
                RName one = (RName)typenames[j], two = (RName)typenames[j+1];
                if(((ELType)types.get(one)).getType()> ((ELType)types.get(two)).getType()){
                    Object hold = typenames[j];
                    typenames[j]=typenames[j+1];
                    typenames[j+1]=hold;
                }
            }
        }
        for(int i=0;i<typenames.length; i++){
            out.println(types.get(typenames[i]));
        }
            
    }
    
    
    private TokenFilter   tfilter;
    
    public ArrayList<String> getParsedTokens(){
        return tfilter.getParsedTokens();
    }
    
    public void newDecisionTable () {
        localcnt = 0;
        localtypes.clear();
    }
    
    /**
     * The actual routine to compile either an action or condition.  The code
     * is all the same, only a flag is needed to decide to compile an action vs
     * condition.
     * @param action    Flag
     * @param s         String to compile
     * @return          Postfix
     * @throws Exception    Throws an Exception on any error.
     */
    private String compile (String s)throws RulesException {

        
        InputStream      stream  = new ByteArrayInputStream(s.getBytes());
        DataInputStream  input   = new DataInputStream(stream);
        DTRulesscanner   lexer   = new DTRulesscanner (input);
                         tfilter = new TokenFilter(session, lexer,types, localtypes);
        DTRulesParser    parser  = new DTRulesParser(tfilter);
        Object           result  = null;
        
        parser.localCnt = localcnt;
        try {
           result = parser.parse().value;
        }catch(Exception e){
           int line = 0;
           for (int i = 0; i < lexer.linenumber(); i++){
        	   int tryhere = s.indexOf("\n", line);
        	   if(tryhere >0) line = tryhere;
           }
           s = s.replaceAll("[\n]", " ");
           s = s.replaceAll("[\r]", " ");
           
           String before = s.substring(0, line+lexer.charnumber());
           String after  = s.substring(line+lexer.charnumber());
           
           String location = "Parsing a "+before.substring(0,before.indexOf(" "));
           before = before.substring(before.indexOf(" ")+1);
           
           throw new RulesException("compileError", location, before+" *ERROR*=> "+after);
        }
        localcnt = parser.localCnt;
        localtypes.putAll(parser.localtypes);
        return result.toString();
    }
    
    /* (non-Javadoc)
     * @see com.dtrules.compiler.ICompiler#getLastPreFixExp()
     */
    public String getLastPreFixExp(){
        return "";
    }

    /**
     * Build a compiler instance.  This compiler compiles either a condition or
     * an action.  Use the compiler access methods to compile each.
     * 
     */
    public EL(){};

    /**
     * @param session Needed to generate the symbol table used by the compiler.  
     * @throws If we cannot set up the session and entityfactory, an exception might be thrown.
     */
    public void setSession(IRSession session) throws Exception {
        this.session = session;
        ef = session.getEntityFactory();
        getTypes(ef);
    }

    
    /**
     * @see com.dtrules.compiler.ICompiler#compileContext(java.lang.String)
     **/
    @Override
    public String compileContext(String context) throws RulesException {
        return compile("context "+context);
    }
    
    /**
     * @see com.dtrules.compiler.ICompiler#compileAction(java.lang.String)
     **/
    @Override
    public String compileAction(String action) throws RulesException {
        return compile("action "+action);
    }
    
    /**
     * We allow all the same actions in the initial action section as we do
     * the Action Section.  However, just because this compiler allows this doesn't
     * mean others have to.
     **/
    @Override
    public String compileInitialAction(String action) throws RulesException {
        return compile("action "+action);
    }
    /**
     * @see com.dtrules.compiler.ICompiler#compileCondition(java.lang.String)
     **/
    @Override
    public String compileCondition(String condition) throws RulesException {
        return compile("condition "+ condition);
    }
    
    /**
     * Returns the compiled version of the policy statement.  Double quotes are
     * replaced forcefully by single quotes.
     */
    @Override
    public String compilePolicyStatement(String policyStatement) throws RulesException {
        if(policyStatement==null)return "";
        policyStatement = policyStatement.replaceAll("\"", "'");
        StringBuffer  sbuff = new StringBuffer();
        int s = 0;
        int e = policyStatement.indexOf("{",s);
        boolean first = true;
        while(e>0){
            sbuff.append("\"");
            sbuff.append(policyStatement.substring(s, e));
            if(first){
            	first = false;
            	sbuff.append("\" ");
            }else{
            	sbuff.append("\" s+ ");
            }
            s = e;
            e = policyStatement.indexOf("}",s);
            if(e<0){
                throw new RuntimeException("Unbalanced braces: "+policyStatement);
            }
            
            String source = "policystatement " + policyStatement.substring(s+1,e);

            try{
                String value = compile(source);
                sbuff.append(value);
                sbuff.append("cvs strconcat ");
            }catch(Exception ex){
                throw new RulesException("ParseError","PolicyStatements", ex.toString()+ "\n Source: >>"+ source +"<<");
            }
            
            s = e+1;
            e = policyStatement.indexOf("{",s);
        }
        
        sbuff.append("\"");
        sbuff.append(policyStatement.substring(s));
        if(s==0){
            sbuff.append("\" ");
        }else{
            sbuff.append("\" strconcat");
        }
        
        return sbuff.toString();
    }


    /**
     * @see com.dtrules.compiler.ICompiler#getTypes()
     **/
    public HashMap<RName,IRType> getTypes() {
        return types;
    }
    /**
     * Return the list of Possibly (but we can't tell for sure) referenced attributes
     * so far by this compiler.
     */
    public ArrayList<String> getPossibleReferenced() {
        ArrayList<String> v = new ArrayList<String>();
        for(IRType type :types.values()){
            v.addAll(((ELType)type).getPossibleReferenced());
        }
        return v;
    }
    /**
     * Return the list of UnReferenced attributes so far by this compiler
     */
    public ArrayList<String> getUnReferenced() {
        ArrayList<String> v = new ArrayList<String>();
        for(IRType type :types.values()){
            v.addAll(((ELType)type).getUnReferenced());
        }
        return v;
    }   
}
