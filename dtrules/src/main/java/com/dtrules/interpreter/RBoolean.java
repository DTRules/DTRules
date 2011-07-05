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

public class RBoolean extends ARObject {

	static RType type = RType.newType("boolean");
	
    /**
     * Provides all the supported conversions of a String to a boolean value.
     * @param value
     * @return
     * @throws RulesException
     */
    public static boolean booleanValue(String value)throws RulesException {
        String v = value.trim();
        if(v.equalsIgnoreCase("true") ||
           v.equalsIgnoreCase("y")    ||
           v.equalsIgnoreCase("t")    ||
           v.equalsIgnoreCase("yes")){
            return true;
        }else if(v.equalsIgnoreCase("false") ||
                 v.equalsIgnoreCase("n")     ||
                 v.equalsIgnoreCase("f")     ||
                 v.equalsIgnoreCase("no")){
            return false;
        }
        throw new RulesException("typecheck","String Conversion to Boolean","No boolean value for this string: "+value);
    }
    /**
     * Really, there is no reason for more that two instances of Boolean objects
     * in the Rules Engine world.
     */
	private static RBoolean trueValue  = new RBoolean(true);
	private static RBoolean falseValue = new RBoolean(false);
	/**
     * The value of this boolean object. 
	 */
	public final boolean value; 
	/**
     * A private constructor to avoid any creation of boolean values besides
     * the two (true and false) 
     * @param _value
	 */
	private RBoolean(boolean _value){
		value = _value;
	}
	
	public RType type() {
		return type;
	}
    /**
     * Return the proper boolean object for the given boolean value.
     * @param value
     * @return
     */
	public static RBoolean getRBoolean(boolean value){
		if(value)return trueValue;
		return falseValue;
	}

    /**
     * Attempt to convert the string, and return the proper boolean object.
     * @param value
     * @return
     * @throws RulesException
     */
    public static RBoolean getRBoolean(String value) throws RulesException {
        return getRBoolean(booleanValue(value));
    }
    /**
     * Return my value.
     */
	public boolean booleanValue() throws RulesException {
		return value;
	}
	/**
     * We *COULD* simply do an object equality test... 
	 */
	public boolean equals(IRObject o) throws RulesException {
		return value == o.booleanValue();
	}
	/**
     * Return the string value for this boolean. 
	 */
	public String stringValue() {
		if(value)return "true";
		return "false";
	}
	/**
     * Return the string value for this boolean
	 */
	public String toString() {
		return stringValue();
	}
	/**
     * The postfix is nothing more than the string value for this boolean. 
	 */
	public String postFix() {
		return stringValue();
	}

}
