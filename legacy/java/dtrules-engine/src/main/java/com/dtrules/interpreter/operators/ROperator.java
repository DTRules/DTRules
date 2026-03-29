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
  
package com.dtrules.interpreter.operators;

import com.dtrules.entity.IREntity;
import com.dtrules.entity.REntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.ARObject;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RName;
import com.dtrules.interpreter.RType;
import com.dtrules.session.DTState;

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
public abstract class ROperator extends ARObject {
    
	public static RType type = RType.newType("operator");
	
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
    
	public RType type(){
		return type;
	}
	
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
                System.err.println("Duplicate Operators defined for " + rn.stringValue());
            	throw new RuntimeException("Duplicate definitions for " + rn.stringValue());
            }
            primitives.addAttribute(rn, "", o, false, true, o.type(),null,"operator","","");
            primitives.put(null, rn,o);
        } catch (RulesException e) {
            System.err.println("An Error occured in alias building the primitives Entity: "+n);
            throw new RuntimeException("An Error occured in alias building the primitives Entity: "+n);
        }
    }

    /**
     * Defines an Operator with this name. No checking for existing operators is done, so you 
     * can use this operator to override the definition of an existing operator.  You should 
     * not call this operator from within the definition of an operator as you might using
     * alias().  If you do, you might have compiler ordering issues which may or may not work
     * out as you might like.  No, you should instead use the override call after you have 
     * allocated at least one session within a JVM, and before you have executed any of the 
     * Rule Sets that depend on your override.
     * 
     * Priorities *are* respected, and all built in operators are given a priority of 0.  Any
     * operator with a higher priority will overwrite a built in operator, and a built in 
     * operator will not overwrite a higher priority operator.
     * 
     * @param o
     * @param n
     */
    public static void override (IRObject o ,String n) {
        try {
            RName rn = RName.getRName(n);
            primitives.addAttribute(rn, "", o, false, true, o.type(),null,"operator","","");
            IRObject op = primitives.get(rn);
            if(op instanceof ROperator && o instanceof ROperator ){ // Operator overwriting operator.
                ROperator rop = (ROperator) op;                     // Don't overwrite if the existing operator is
                ROperator ro  = (ROperator) o;                      // a higher priority
                if(ro.priority() > rop.priority() || rop.priority()==0){  // Overwrite anyway if priority is zero
                    primitives.put(null, rn,o);
                }
            }else{                                  // We can't worry about priorities if this isn't
                primitives.put(null, rn,o);         // an operator overwriting an operator.
            }
        } catch (RulesException e) {
            System.err.println("An Error occured in alias building the primitives Entity: "+n);
            throw new RuntimeException("An Error occured in alias building the primitives Entity: "+n);
        }
    }
    /**
     * When overriding operators, the priority allows some help configuring how these overrides are 
     * applied.  At the easiest level, the default operators all have priorities of 0.  So any user
     * defined operators can be 1, and you know that no matter what order the operators are added, the
     * right result will occur, i.e. operators with priority 1 will win.
     * <br><br>
     * It gets more complicated if you want to override not only the default operators but some user
     * defined operators as well.  This simply requires a bit of documentation on the part of those
     * defining user operators to be used with DTRules.
     * <br><br>
     * To define a priority other than 0, override this method.
     * @return
     */
    public int priority() {
        return 0;
    }
    
    /**
     * Get an instance of this operator.
     * @return
     */
    public static ROperator getInstance(String name ){
        return (ROperator) primitives.get(name); 
    }
    
    /**
     * Get an instance of this operator.
     * @return
     */
    public static ROperator getInstance(RName name ){ 
        return (ROperator) primitives.get(name); 
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
        name = RName.getRName(_name, true);
        alias((IRObject) this,_name);
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
    
    abstract public void execute(DTState state) throws RulesException ;
    
    public void arrayExecute(DTState state) throws RulesException {
    	execute(state);
    }
}
