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

/**
 * Integers are longs.
 * @author ps24876
 *
 */
public class RInteger extends ARObject {

	static RType type = RType.newType("integer");
	
	final long value;

    static RInteger mone = new RInteger(-1);
    static RInteger zero = new RInteger( 0);
    static RInteger one  = new RInteger( 1);
    
    private RInteger(long i){
        value = i;
    }

    /**
     * Parses the given string, and returns an RInteger Object
     * @param s String to parse
     * @return RInteger value for the given string
     * @throws RulesException
     */
    static public long getIntegerValue(String s) throws RulesException {
        if(s==null || (s = s.trim()).length()==0 || s.equalsIgnoreCase("null")){
            return 0L;
        }
        try {
            long v = Long.parseLong(s);
            return v;
        } catch (NumberFormatException e) {
            throw new RulesException("Conversion Error","RInteger.getIntegerValue()","Could not covert the string '"+s+"' to an integer: "+e.getMessage());
        }
    }
    
    static public RInteger getRIntegerValue(String s) throws RulesException {
        return getRIntegerValue(getIntegerValue(s));
    }
    
    static public RInteger getRIntegerValue(long i){
        
        if(i == -1) return mone;
        if(i ==  0) return zero;
        if(i ==  1) return one;
        return new RInteger(i);
    }

    static public RInteger getRIntegerValue(double i){
        return getRIntegerValue((long) i);
    }    
    
    static public RInteger getRIntegerValue(int i){
        return getRIntegerValue((long) i);
    }    
        
	/**
	 * Returns the type for this object.
	 */
	public RType type() {
		return type;
	}
  
	public double doubleValue()  {
		return (double)value;
	}

	public int intValue()  {
		return (int)value;
	}
    
	public long longValue() throws RulesException {
		return (long) value;
	}

	public boolean equals(IRObject o) throws RulesException {
		return value == o.intValue();
	}

	public String postFix() {
		return stringValue();
	}

	public RInteger rIntegerValue() throws RulesException {
		return this;
	}

	public String stringValue() {
		String str = Long.toString(value);
		return str;
	}
	
	public String toString(){
		return stringValue();
	}

    /**
     * returns 0 if both are equal. -1 if this object is less than the argument. 
     * 1 if this object is greater than the argument
     */	
	public int compare(IRObject irObject) throws RulesException {
		return (this.value==irObject.intValue())?0:((this.value<irObject.intValue())?-1:0);	
	}
}
