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
  
package com.dtrules.interpreter.operators;

import com.dtrules.entity.IREntity;
import com.dtrules.entity.REntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.ARObject;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RName;

/**
 * This Class creates the primities entity in a rather cute way.  The actual operators
 * are for the most part extend ROperator and only supply a new execute() method.  Each
 * of these classes are defined as subclasses of the classes listed in the static 
 * section.  They in turn have a static section that lists all of classes that implement
 * the operators in their section.  
 * <br><br>
 * Each operator calls the super constructor with their name, and the super constructor
 * (defined here) creates the entry in the primitives entity for the operator.
 * 
 * @author paul snow
 *
 */
public class ROperator extends ARObject {
    
	static final REntity primitives = new REntity(1,true,RName.getRName("primities",false));
    
	static {
		new RMath();
		new RArrayOps();
        new RControl();
        new RBooleanOps();
        new RMiscOps();
        new RDateTimeOps();
        new RTableOps();
        new RXmlValueOps();
        new RStringOps();
	}
	
	static public IREntity getPrimitives() { 
		return primitives; 
	}
    
	final RName name;
    public int type() { return iOperator; }
    
    /**
     * Puts another entry into the primitives entity under a different name.  This is
     * useful for operators that we would like to define under two names (such as "pop" and
     * "drop".)
     * 
     * This function also allows us to define constants such as "true" and "false" as objects
     * rather than defining operators that return "true" and "false".
     * 
     * @param o
     * @param n
     */
    protected static void alias (IRObject o ,String n) {
        try {
            RName rn = RName.getRName(n);
            if(primitives.containsAttribute(rn)){
            	throw new RuntimeException("Duplicate definitions for "+rn.stringValue());
            }
            primitives.addAttribute(rn, "", o, false, true, o.type(),null,"operator","");
            primitives.put(rn,o);
        } catch (RulesException e) {
            throw new RuntimeException("An Error occured in alias building the primitives Entity: "+n);
        }
    }
    
    /**
     * A method that makes it a bit easier to call the other alias function when I am
     * creating an alias for an operator in its own constructor.
     * 
     * @param n
     */
    protected void alias (String n){
        alias(this,n);
    }
    
    /**
     * All of the operators extend ROperator.  They call the super constructor with their 
     * name as a string.  Here we convert the name to an RName, set the name as a final
     * field in the Operator, and create an entry in the primities entity for the 
     * operator.
     * @param _name
     */
    public ROperator(String _name){
      name = RName.getRName(_name,true);
      try {
         primitives.addAttribute(name, "", this,false, true,iOperator,null,"operator","");
	     primitives.put(name,this);
	  } catch (RulesException e) {
		 throw new RuntimeException("An Error occured building the primitives Entity: "+name);
	  }
    }
	
    public boolean isExecutable() { return true; }
	
    public String stringValue() {
		return name.stringValue();
	}
	
    public String toString(){
		return name.stringValue();
	}
	
    public String postFix(){
		return name.stringValue();
	}
}
