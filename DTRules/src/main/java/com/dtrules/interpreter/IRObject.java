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
  
package com.dtrules.interpreter;


import java.util.ArrayList;
import java.util.Date;
import java.util.HashMap;

import com.dtrules.entity.IREntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.mapping.XMLNode;
import com.dtrules.session.DTState;
import com.dtrules.session.IRSession;
/**
 * Every Rules Engine object implements the IRObject interface.  This interface
 * is used by the Interpreter to execute objects, and by the implementation of
 * operators to manipulate objects.
 * <br><br>
 * This interface implements the necessary methods for doing the data conversions
 * necessary when extracting data from the Rules Engine state after having 
 * executed a set of rules.
 * @author Paul Snow
 *
 */
@SuppressWarnings({"unchecked"})
public interface IRObject {

    public IRObject clone(IRSession s) throws RulesException;
    
	//  *************** NOTE !!!!!!
	//  You can't put static methods on an interface. So the String to integer conversion
	//  for types is a static method on the RSession class.
 	/**
 	 * Every Object in the Rules Engine has a Type, as listed here.  Each type
 	 * has an integer value, which we actually use in the code (specified in the
 	 * types[] array.  This code predates Enumerations in Java, and perhaps we
 	 * should fix that sometime in the future.
 	 * @author Paul Snow
 	 */
	final String rBoolean       = "boolean",
	             rString        = "string",
	             rInteger       = "integer",
	             rFloat         = "float",
	             rEntity        = "entity",
	             rName          = "name",
	             rArray         = "array",
	             rDecisiontable = "decisiontable",
	             rNull          = "null",
	             rMark          = "mark",
	             rOperator      = "operator",
                 rTime          = "time",
                 rTable         = "table",
                 rXmlValue      = "xmlvalue";
	
	/**
	 * The String types in this array are associated with the integer values
	 * of the types.
	 */
	final String types[] = { rBoolean, rString, rInteger, rFloat,
			                 rEntity,  rName,   rArray,   rDecisiontable,  
			                 rNull,    rMark,   rOperator, rTime,
                             rTable, rXmlValue};
	
    /**
     * These integer values are used to check object types by the Rules Interpreter.
     * They can be converted to a name using static methods typeInt2Str() and 
     * typeStr2Int() found on the RSession class.
     */
	final int    iBoolean       = 0,
	             iString        = 1,
	             iInteger       = 2,
	             iDouble        = 3,
	             iEntity        = 4,
	             iName          = 5,
	             iArray         = 6,
	             iDecisiontable = 7,
	             iNull          = 8,
	             iMark          = 9,
	             iOperator      = 10,
                 iTime          = 11,
                 iTable         = 12,
                 iXmlValue      = 13;
 	
	/**
	 * This method defines the executable behavior of a object within the 
	 * Rules Interpreter.
	 * @param state The Rules Engine State
	 * @throws RulesException
	 */
	void execute(DTState state) throws RulesException;
	
	/**
	 * Returns an executable version of this object.  The non-executable behavior
	 * of all objects is to push themselves onto the data stack.  If the executable
	 * behavior is the same as the non-executable behavior, then this method may
	 * return a non-executable object.
	 * @return
	 */
	public IRObject getExecutable();
    
	/**
     * Returns the non-executable representation of this object.
     * @return
     */
    public IRObject getNonExecutable();
    
    /**
     * We implement special rules for comparing some Rules Engine objects.  This 
     * method is used to distribute such rules to the objects themselves.
     * @param o
     * @return
     * @throws RulesException
     */
    public boolean equals(IRObject o) throws RulesException;
    
    /**
     * Returns true if the object is executable.  When an array is executing 
     * each of its elements, executable objects have their execute() method
     * called.  Non-executable objects are simply pushed to the data stack.
     * @return
     */
    public boolean isExecutable();
    
    /**
     * By default, the toString() method for most Objects should provide
     * a representation of the object that can be used as the postFix value.
     * The stringValue should provide simply the data for the object.  Thus
     * If I want to append the a String and a Date to create a new string,
     * I should append the results from their stringValue() implementation.
     * <br><br>
     * If I needed the postfix (and thus the quotes and maybe other ) then
     * call postFix().  But if the postFix is going to match either stringValue()
     * or toString, it is going to match toString().
     * <br><br>
     * @return
     */
    public String   stringValue();
    /**
     * Return the String Reperesntation of an object.  For some objects, this
     * method will perform a conversion (such as a number to a string).
     * @return
     */
    public RString  rStringValue();
    /**
     * Returns the postFix representation of an object.  This method is used
     * to build a trace file which allows a reconstruction of the Rules Engine
     * state.  Also used in some debugging situations.
     * @return
     */
    public String postFix();
    
    /**
     * Return the integer type of this object
     * @return
     */
    public int type();
	
    /**
     * Clone the Rules Engine object.  For many types (boolean, integer, Strings), 
     * this method will simply return a pointer to the original object.  This 
     * makes sense in cases where the state of the object cannot be modified by
     * the Rules Engine.
     * @return
     */
    public IRObject rclone();
    /**
     * Return an XMLNode representation of this object.  
     * A Rules Exception will be thrown if no such representation exists.
     * @return
     * @throws RulesException
     */
    public XMLNode              xmlTagValue()   throws RulesException;
    /**
     * Return a long representation of this object.  
     * A Rules Exception will be thrown if no such representation exists.
     * @return
     * @throws RulesException
     */
    public long                 longValue ()    throws RulesException;
    /**
     * Return an int representation of this object.  
     * A Rules Exception will be thrown if no such representation exists.
     * @return
     * @throws RulesException
     */
    public int                  intValue ()     throws RulesException;
    /**
     * Return a RInteger representation of this object.  
     * A Rules Exception will be thrown if no such representation exists.
     * @return
     * @throws RulesException
     */
    public RInteger             rIntegerValue() throws RulesException;
    /**
     * Return a double representation of this object.  
     * A Rules Exception will be thrown if no such representation exists.
     * @return
     * @throws RulesException
     */
    public double               doubleValue ()  throws RulesException;
    /**
     * Return a RDouble representation of this object.  
     * A Rules Exception will be thrown if no such representation exists.
     * @return
     * @throws RulesException
     */
    public RDouble              rDoubleValue()  throws RulesException;
    /**
     * Return an ArrayList representation of this object.  
     * A Rules Exception will be thrown if no such representation exists.
     * @return
     * @throws RulesException
     */
    public ArrayList<IRObject>  arrayValue ()   throws RulesException;
    /**
     * Return a RArray representation of this object.  
     * A Rules Exception will be thrown if no such representation exists.
     * @return
     * @throws RulesException
     */
    public RArray               rArrayValue()   throws RulesException;
    /**
     * Return a boolean representation of this object.  
     * A Rules Exception will be thrown if no such representation exists.
     * @return
     * @throws RulesException
     */
    public boolean              booleanValue () throws RulesException;
    /**
     * Return a RBoolean representation of this object.  
     * A Rules Exception will be thrown if no such representation exists.
     * @return
     * @throws RulesException
     */
    public RBoolean             rBooleanValue() throws RulesException;
    /**
     * Return a HashMap representation of this object.  
     * A Rules Exception will be thrown if no such representation exists.
     * @return
     * @throws RulesException
     */
    public HashMap              hashMapValue () throws RulesException;
    /**
     * Return a RXmlValue representation of this object.  This method
     * can return an RNull as well, so it returns an IRObject.
     * A Rules Exception will be thrown if no such representation exists.
     * @return
     * @throws RulesException
     */
    public IRObject             rXmlValue ()    throws RulesException; // Because it can return an RNull
    /**
     * Return a IREntity representation of this object.  
     * A Rules Exception will be thrown if no such representation exists.
     * @return
     * @throws RulesException
     */
    public IREntity             rEntityValue () throws RulesException;
    /**
     * Return a RName representation of this object.  
     * A Rules Exception will be thrown if no such representation exists.
     * @return
     * @throws RulesException
     */
    public RName                rNameValue ()   throws RulesException;
    /**
     * Return a RTime representation of this object.  
     * A Rules Exception will be thrown if no such representation exists.
     * @return
     * @throws RulesException
     */
    public RTime                rTimeValue ()   throws RulesException;
    /**
     * Return a Date representation of this object.  
     * A Rules Exception will be thrown if no such representation exists.
     * @return
     * @throws RulesException
     */
    public Date                 timeValue ()    throws RulesException;
    /**
     * Return a RTable representation of this object.  
     * A Rules Exception will be thrown if no such representation exists.
     * @return
     * @throws RulesException
     */
    public RTable               rTableValue()   throws RulesException;
    /**
     * Return a HashMap representation of this object.  
     * A Rules Exception will be thrown if no such representation exists.
     * @return
     * @throws RulesException
     */
    public HashMap              tableValue()    throws RulesException;
    /**
     * Compares this Rules Engine Object with the given Rules Engine object.
     *  
     * @param irObject
     * @return -1 if less than, 0 if equal, 1 if greater than
     * @throws RulesException
     */
    public int                  compare(IRObject irObject) throws RulesException;
}
