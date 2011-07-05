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
 * @author Paul Snow
 *
 */
public class RDouble extends ARObject {
    
	static RType type = RType.newType("double");

	final double value;
    
    static RDouble mone = new RDouble(-1.0);
    static RDouble zero = new RDouble(0.0);
    static RDouble one  = new RDouble(1.0);
    
    private RDouble(double v){
        value = v;
    }
    
    static public double getDoubleValue(String s) throws RulesException {
        if(s==null || s.trim().length()==0){
            return 0.0D;
        }
        try {
            double v = Double.parseDouble(s);
            return v;
        } catch (NumberFormatException e) {
            throw new RulesException("Conversion Error","RDouble.getDoubleValue()","Could not covert the string '"+s+"' to a double: "+e.getMessage());
        }
    }
    /**
     * Parses the given string, and returns an RDouble Object
     * @param s String to parse
     * @return RDouble value for the given string
     * @throws RulesException
     */
    static public RDouble getRDoubleValue(String s) throws RulesException {
        return getRDoubleValue(getDoubleValue(s));
    }

    
    
    static public RDouble getRDoubleValue(double v){
        if(v == -1.0) return mone;
        if(v ==  0.0) return zero;
        if(v ==  1.0) return one;
        
        return new RDouble(v);
    }
    
    static public RDouble getRDoubleValue(int v){
        return getRDoubleValue((double) v);
    }
    
    static public RDouble getRDoubleValue(long v){
        return getRDoubleValue((double) v);
    }

    public RDouble rDoubleValue() {
        return this;
    }
    
	public double doubleValue() {
		return value;
	}

	public int intValue() {
		return (int) value;
	}


	public long longValue() {
		return (long) value;
	}

	public RInteger rIntegerValue() {
		return RInteger.getRIntegerValue((long)value);
	}

	/**
	 * Returns the type for this object.
	 */
	public RType type() {
		return type;
	}

    public String stringValue() {
        return Double.toString(value);
    }

    /**
     * returns 0 if both are equal. -1 if this object is less than the argument. 
     * 1 if this object is greater than the argument
     */	
	public int compare(IRObject irObject) throws RulesException {
		return (this.value==irObject.doubleValue())?0:((this.value<irObject.doubleValue())?-1:0);	
	}

	@Override
	public boolean equals(IRObject o) throws RulesException {
		return value == o.doubleValue();
	}

    public String toString() {
        return Double.toString(value);
    }

	
}
