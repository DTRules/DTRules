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
import java.lang.reflect.Field;
import java.util.ArrayList;
import java.util.HashMap;

import com.dtrules.compiler.el.cup.parser.RLocalType;
import com.dtrules.compiler.el.cup.parser.RSymbol;
import com.dtrules.compiler.el.cup.parser.sym;
import com.dtrules.entity.REntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RName;
import com.dtrules.session.DTState;
import com.dtrules.session.IRSession;

import java_cup.runtime.*;

public class TokenFilter implements Scanner{
    
    Scanner                     scanner;
    HashMap<RName, RType>       types;
    IRSession                   session;
    DTState                     state;
    boolean                     EOF         = false;
    RSymbol                     lastToken   = null;
    RSymbol                     unget       = null;
    ArrayList<String>           tokens      = new ArrayList<String>();
    HashMap<String,RLocalType>  localtypes  = null;
    String						possessive  = null;
    
    public ArrayList<String> getParsedTokens(){
        return tokens;
    }
    
    TokenFilter(IRSession session, Scanner scanner, HashMap<RName, RType> types, HashMap<String,RLocalType> localtypes){
        this.types      = types;
        this.scanner    = scanner;
        this.session    = session;
        this.state      = session.getState();
        this.localtypes = localtypes;
    }
    /**
     * Look up an Identifier in the symbol table and return its type.
     * @param ident
     * @return
     */
    int identType(String ident,String entity){
        
           RType type = (RType) types.get(RName.getRName(ident,true));
           boolean defined = true;
           if(entity != null && entity.length()!=0 && type!=null){
              defined = false;
              for(REntity rEntity : type.getEntities()){
                  if(entity.equalsIgnoreCase(rEntity.getName().stringValue())){
                      if(rEntity.containsAttribute(RName.getRName(ident))){
                          defined = true;
                      }
                      break;
                  }
              }
           }
           if(type==null || defined==false){
               RName rname = RName.getRName(
                  (entity==null||entity.length()==0?"":entity+".")+ident);
               try {
                  IRObject o = session.getState().find(rname);
                  if(o==null)return -1;
                  return o.type();
                } catch (RulesException e) {
                  return -1;
                }
           }
           
           type.addRef(entity);
           
           return type.getType();
         
    }
    /**
     * Looks at the sym.class with reflection and looks up the type
     * associated with the given code.
     * 
     * @param type integer code associated with a symbol.
     */
    @SuppressWarnings("unchecked")
    private String type2str(int type){
        try{
            Class   tokentypes  = sym.class;
            Field[] types       = tokentypes.getDeclaredFields();
            for(int i=0;i<types.length;i++){
                String typeName = types[i].getName();
                String fieldType = types[i].getType().getName();
                if(fieldType.equals("int")&&types[i].getInt(types[i])==type){
                    return typeName; 
                }
            }
        }catch(Exception e){
            //Who Cares?  Ignore!
        }
        return "-not found-";
    }
    
    
    public Symbol next_token() throws Exception {
        RSymbol next=null;
        
        if(unget!=null){
            next  = unget;
            unget = null;
        }else{
            try {
              next = new RSymbol(false, scanner.next_token());
              if(next.sym == sym.IDENT){
                  next.leftvalue = "/"+next.value.toString()+" xdef ";
              }
            } catch (Error e) {
              throw new Exception(e);
            }
        }
        
        if(next.sym == sym.RCURLY && lastToken.sym != sym.SEMI){
            lastToken = new RSymbol(false, new Symbol(sym.SEMI));
            unget     = next;
            next      = lastToken;
        }
        
        if(next.sym == sym.NAME &&
              next.value.toString().charAt(0)=='$'){
            next.value = next.value.toString().substring(1);
        }
        
        if(next.value==null){
            if(EOF==false) {
                next.value=";";
                next.sym = sym.SEMI;
                EOF = true;
            }else{    
                next.value="EOF";
                next.sym=sym.EOF;
            }
        }else{
            EOF = false;
        }
        
        // Look for Possessives.
        // Remove the 's from them.  They are in a form:  client's plan's contract
        // What we feed the parser is:  POSSESSIVE client COMMA , POSSESSIVE plan COMMA , ENTITY contract
        if(next.sym==sym.POSSESSIVE){
            String entity = "";
            String ident = next.value.toString();
            ident = ident.replace("'s", "");
            if(ident.indexOf('.')>=0){
                entity=ident.substring(0,ident.indexOf('.'));
                ident =ident.substring(ident.indexOf('.')+1);
            }
            int    theType = identType(ident,entity);
            
            if(theType!= IRObject.iEntity && theType != -1 ){
                throw new RuntimeException("Invalid Possessive. "+ident+" is not an Entity");
            }
            
            if(theType == -1){
                if(localtypes.containsKey(ident.toLowerCase())){
                    RLocalType t = localtypes.get(ident.toLowerCase());
                    theType    = t.type;
                    next.local = true;
                    next.value = t.index +" local@ ";
                }else{
                    throw new RuntimeException("Undefined attribute '"+next.value+"'");
                }
            }else{
                next.value = (entity.length()>0?entity+".":"")+ident;
            }        
            unget = new RSymbol(false, new Symbol(sym.COMMA)); // Inject a comma after every possessive
            unget.value = ",";
            
        }
        
        if(next.sym==sym.IDENT){
            String entity  = "";
            String ident   = next.value.toString();
            if(ident.indexOf('.')>=0){
                entity=ident.substring(0,ident.indexOf('.'));
                ident =ident.substring(ident.indexOf('.')+1);
            }
            int    theType = identType(ident,entity);
            
            if(theType == -1 ){
                if(localtypes.containsKey(next.value.toString().toLowerCase())){
                    RLocalType t = localtypes.get(next.value.toString().toLowerCase());
                    theType = t.type;
                    next = new RSymbol(true, next);
                    next.value = t.index +" local@ ";
                    ((RSymbol)next).leftvalue = t.index +" local! ";
                }
            }
            
            switch(theType){
                case IRObject.iEntity           : next.sym=(sym.RENTITY);        break;
                case IRObject.iName             : next.sym=(sym.RNAME);          break;
                case IRObject.iInteger          : next.sym=(sym.RLONG);          break;
                case IRObject.iDouble           : next.sym=(sym.RDOUBLE);        break;
                case IRObject.iString           : next.sym=(sym.RSTRING);        break;
                case IRObject.iBoolean          : next.sym=(sym.RBOOLEAN);       break;
                case IRObject.iDecisiontable    : next.sym=(sym.RDECISIONTABLE); break;
                case IRObject.iArray            : next.sym=(sym.RARRAY);         break;
                case IRObject.iTime             : next.sym=(sym.RDATE);          break;
                case IRObject.iTable            : next.sym=(sym.RTABLE);         break;
                case IRObject.iOperator         : next.sym=(sym.ROPERATOR);      break;
                case IRObject.iXmlValue         : next.sym=(sym.RXMLVALUE);      break;
                case -1                         : next.sym=(sym.UNDEFINED);      break;                        
                default: 
                    System.out.println("Unhandled Type");
                    tokens.add(next.sym+" "+type2str(next.sym)+" "+next.value.toString());
                    throw new RuntimeException("Unhandled Type: "+next.value);
            }
        }
        
        tokens.add(next.sym+" "+type2str(next.sym)+" "+next.value.toString());
        lastToken = next;
        return next;
    
    }

}
