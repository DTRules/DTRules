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
package com.dtrules.compiler.cup;

import java.io.ByteArrayInputStream;
import java.io.DataInputStream;
import java.io.InputStream;
import java.io.PrintStream;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.Iterator;
import com.dtrules.compiler.RType;
import com.dtrules.entity.REntity;
import com.dtrules.entity.REntityEntry;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RName;
import com.dtrules.session.EntityFactory;
import com.dtrules.session.ICompiler;
import com.dtrules.session.IRSession;

public class Compiler implements ICompiler {
    
    private       HashMap<RName,RType>     types = null;
    private final EntityFactory            ef;
    private final IRSession                session;
       
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
    private void addType( REntity entity, RName name, int itype) throws Exception {
        RType type =  types.get(name);
        if(type==null){
            type      = new RType(name,itype,entity);
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
    public HashMap<RName,RType> getTypes(EntityFactory ef) throws Exception {
        
        if(types!=null)return types;
        
        types = new HashMap<RName, RType>();
        Iterator<RName> entities = ef.getEntityRNameIterator();
        while(entities.hasNext()){
            RName    name    = entities.next();
            REntity  entity  = ef.findRefEntity(name);
            Iterator<RName> attribs = entity.getAttributeIterator();
            addType(entity,entity.getName(),IRObject.iEntity);
            while(attribs.hasNext()){
                RName        attribname = attribs.next();
                REntityEntry entry      = entity.getEntry(attribname);
                addType(entity,attribname,entry.type);
            }
        }
        
        Iterator<RName> tables = ef.getDecisionTableRNameIterator();
        while(tables.hasNext()){
            RName tablename = tables.next();
            RType type = new RType(tablename,IRObject.iDecisiontable,(REntity) ef.getDecisiontables());
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
                if(types.get(one).getType()> types.get(two).getType()){
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
    private String compile (boolean action, String s)throws Exception {

        
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
           throw new Exception( "Error found at Line:Char ="+lexer.linenumber()+":"+lexer.charnumber()+" "+
                   e.toString()); 
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
     * @param entityfactory Needed to generate the symbol table used by the compiler.
     * @throws Exception    Throws an exception of a compiler error is encountered.
     */
    public Compiler(IRSession session) throws Exception {
        this.session = session;
        ef = session.getEntityFactory();
        getTypes(ef);
    }

    
    /* (non-Javadoc)
     * @see com.dtrules.compiler.ICompiler#compileContext(java.lang.String)
     */
    public String compileContext(String context) throws Exception {
        return compile(true,"context "+context);
    }
    
    /* (non-Javadoc)
     * @see com.dtrules.compiler.ICompiler#compileAction(java.lang.String)
     */
    public String compileAction(String action) throws Exception {
        return compile(true,"action "+action);
    }
    /* (non-Javadoc)
     * @see com.dtrules.compiler.ICompiler#compileCondition(java.lang.String)
     */
    public String compileCondition(String condition) throws Exception {
        return compile(false,"condition "+ condition);
    }
    /* (non-Javadoc)
     * @see com.dtrules.compiler.ICompiler#getTypes()
     */
    public HashMap<RName,RType> getTypes() {
        return types;
    }
    /**
     * Return the list of Possibly (but we can't tell for sure) referenced attributes
     * so far by this compiler.
     */
    public ArrayList<String> getPossibleReferenced() {
        ArrayList<String> v = new ArrayList<String>();
        for(RType type :types.values()){
            v.addAll(type.getPossibleReferenced());
        }
        return v;
    }
    /**
     * Return the list of UnReferenced attributes so far by this compiler
     */
    public ArrayList<String> getUnReferenced() {
        ArrayList<String> v = new ArrayList<String>();
        for(RType type :types.values()){
            v.addAll(type.getUnReferenced());
        }
        return v;
    }   
}
