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
  
package com.dtrules.interpreter;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.operators.ROperator;
import com.dtrules.session.DTState;
import com.dtrules.session.IRSession;
import com.dtrules.session.RuleSet;

/**
 * @author Paul Snow
 *
 */
public class RString extends ARObject {

	public static RType type = RType.newType("string");

	RString  pair;
    String   value;
    boolean  executable = false;
    
    public double doubleValue() throws RulesException {
        double d;
        try {
           d = Double.parseDouble(value);
        } catch (NumberFormatException e) {
           return super.doubleValue();
        }
        return d;
    }

    @Override
    public int intValue() throws RulesException {
        int i;
        try {
           i = Integer.parseInt(value);
        } catch (NumberFormatException e) {
           return super.intValue();
        }
        return i;
    }

    @Override
    public long longValue() throws RulesException {
        long l;
        try {
           l = Long.parseLong(value);
        } catch (NumberFormatException e) {
           return super.longValue();
        }
        return l;
    }

    @Override
    public RDouble rDoubleValue() throws RulesException {
        return RDouble.getRDoubleValue(doubleValue());
    }

    @Override
    public RInteger rIntegerValue() throws RulesException {
        return RInteger.getRIntegerValue(longValue());
    }

    @Override
    public RName rNameValue() throws RulesException {
        return RName.getRName(value);
    }

    @Override
    public RString rStringValue() {
        return this;
    }
    
	private RString(String v,boolean executable,RString pair){
		value = v;
		this.pair       = pair;
		this.executable = executable;
	}
    /**
     * Return a non Executable string
     * @param v
     * @return
     */
	static public RString newRString(String v){
		return newRString(v,false);
	}
	/**
	 * Return an RString of the given executable nature.
	 * @param v
	 * @param executable
	 * @return
	 */
	static public RString newRString(String v, boolean executable){
		RString s   = new RString(v,executable,null);
		s.pair      = new RString(v,!executable,s);
		return s;
	}
	
	@Override
	public IRObject getExecutable() {
		if(isExecutable()){
			return this;
		}else{
			return pair;
		}
	}

	@Override
	public IRObject getNonExecutable() {
		if(isExecutable()){
			return pair;
		}else{
			return this;
		}
	}
	
    /**
     * Returns a boolean value if the String can be reasonably 
     * interpreted as a boolean.
     * 
     * @Override
     */
    public boolean booleanValue() throws RulesException {
       return RBoolean.booleanValue(value);
    }

	/**
     * 
	 */
    public RBoolean rBooleanValue() throws RulesException {
        return RBoolean.getRBoolean(booleanValue());
    }

	/**
	 * Returns the type for this object.
	 */
	public RType type() {
		return type;
	}

	static RType iArray = RType.newType("array");
	
	/**
	 * Here we look to see if we can do a compile time lookup of
	 * an object.  If we can't, we just return the object unchanged.
	 * But if we can, then we return the value we looked up.  That
	 * saves many, many runtime lookups.
	 */
	static IRObject lookup(IRSession session, RName name){
		IRObject v = ROperator.getPrimitives().get(name); // First check if it is an operator
		if(v==null){                                      // No? Then
           try {                                          //   look for a decision table.  
              v = session.getEntityFactory().getDecisionTable(name);
           } catch (RulesException e) {  }                // Any error just means the name isn't                       
        }                                                 //   a decision table.  Not a problem.
		if(v==null || v.type()== iArray) return name;
		return v;
	}
    
	/**
	 * Compiles the String and returns the executable Array Object
	 * that results.  Unless the compilation fails, at which time 
	 * we throw an exception.
	 * 
	 * @return
	 * @throws RulesException
	 */
	static public IRObject compile(IRSession session, String v, boolean executable) throws RulesException{
		   
	       if(v==null)v=""; // Allow the compiling of null strings (we just don't do anything).
           
	       SimpleTokenizer tokenizer = new SimpleTokenizer(session, v);
            
           IRObject result = compile(session.getRuleSet(), tokenizer, v, executable, 0);
           return result;
    }       
    
    /**
     * The compiles of Strings are recursive.  When we see a [ or {,
     * we recurse.  Then on a close bracket ] or } we return the 
     * non-executable or executable array.  The RString checks to make
     * sure it is the right type, and throws an error if it isn't.
     * The recursive depth is checked by compile();
     * @param tokenizer
     * @return
     * @throws RulesException
     */
    static private IRObject compile(RuleSet ruleset, SimpleTokenizer tokenizer, String v, boolean executable, int depth) throws RulesException {        
       try{
    	   IRSession session  = ruleset.newSession();
           RArray   result    = RArray.newArray(session,true,executable);
           Token    token;    
		   while((token=tokenizer.nextToken())!=null){
			   if(token.getType()== Token.Type.STRING) {
                          IRObject rs = RString.newRString(token.strValue);
                          result.add(rs);
               }else if(token.getType()== Token.Type.LSQUARE) {
                          IRObject o = compile(ruleset,tokenizer, v, false, depth+1);
                          result.add(o);
               }else if(token.getType()== Token.Type.RSQUARE) {
                          if(depth==0 || executable){
                              throw new RulesException("Parsing Error", 
                                      "String Compile", 
                                      "\nError parsing <<"+v+">> \nThe token ']' was unexpected.");
                                     
                          }
                          return result;
               }else if(token.getType()== Token.Type.LCURLY) {
                          IRObject o = compile(ruleset,tokenizer,v,true,  depth+1);
                          result.add(o);
               }else if(token.getType()== Token.Type.RCURLY) {
                          if(depth==0 || !executable){
                              throw new RulesException("Parsing Error",
                                      "String Compile", 
                                      "\nError parsing <<"+v+">> \nThe token '}' was unexpected.");
                                      
                          }
                          return result;
               }else if(token.getType()== Token.Type.NAME) {
                      if(token.nameValue.isExecutable()){       // All executable names are checked for compile time lookup.
                          IRObject prim = lookup(session,token.nameValue);
                          if(prim == null){                     // If this name is not a primitive, then
                              result.add(token.nameValue);      // then add the name as it is to the array.
                          }else{    
                              result.add(prim);                 // Otherwise, compile the reference to the operator.
                          }
                      }else{
			    	      result.add(token.nameValue);
                      }
                      
               }else if (token.getType() == Token.Type.DATE){
                      result.add(token.datevalue);
                      
               }else if (token.getType()== Token.Type.INT) {	  
			    	  RInteger i = RInteger.getRIntegerValue(token.longValue);
			    	  result.add(i);
			   }else if (token.getType()== Token.Type.REAL) {
			    	  RDouble d = RDouble.getRDoubleValue(token.doubleValue);
			    	  result.add(d);
			   }
		   }
           if(depth!=0){
               throw new RulesException("Parsing Error", 
                       "String Compile", 
                       "\nError parsing << " + v + " >>\n missing a " + (executable ? "}" : "]"));
                       
           }
		   return (IRObject) result;
	    } catch (RuntimeException e) {
		   throw new RulesException("Undefined","String Compile","Error compiling string: '"+v+"'\n"+e);
	    }
       
	}
	/**
     * Compiles this String and returns the object. 
     * @param executable
     * @return
     * @throws RulesException
	 */
    public IRObject compile(IRSession session, boolean executable) throws RulesException{
        return compile(session, value,executable);
    }
    
    @Override
    public boolean isExecutable() {
    	return executable;
    }
    
	public void execute(DTState state) throws RulesException {
		if(isExecutable()){
		   IRObject o = compile(state.getSession(),value,true);
		   o.execute(state);
		}else{
		   state.datapush(this);	
		}
	}
	
	public String stringValue() {
        return value;
    }
    public String toString(){
		return "\""+value+"\"";
	}

    
    /**
     * returns 0 if both are equal. -1 if this object is less than the argument. 
     * 1 if this object is greater than the argument
     */
    public int compare(IRObject irObject) throws RulesException {
    	int f = this.value.compareTo(irObject.stringValue());
        if(f<0)return -1;
        if(f>0)return 1;
        return f;
    }
	
    @Override
    public int hashCode() {
        return this.value.hashCode();
    }

    public boolean equals(IRObject o) {
		return value.equals(o.stringValue());
	}

	@Override
	public boolean equals(Object object){
        if(object instanceof IRObject){
            return value.equals(((IRObject)object).stringValue());
        }
	    return super.equals(object);
	}
	
    @Override
    public RDate rTimeValue(IRSession session) throws RulesException {
        return RDate.getRTime(session.getDateParser().getDate(value));   
    }
	
	
}
