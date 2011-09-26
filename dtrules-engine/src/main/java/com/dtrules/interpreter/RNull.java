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

public class RNull extends ARObject {
	
	public static RType type = RType.newType("null");

    /**
     * returns 0 if both are equal. -1 Otherwise.  A Null is considered
     * less than anything else.  We do this just so sorting arrays were
     * some values are null doesn't blow up. 
     */
    public int compare(IRObject irObject) throws RulesException {
        if(irObject.equals(this))return 0;   // A Null is equal to other nulls.
        return -1;                           // A Null is less than anything else.
    }

    
    
	static RNull theNullObject = new RNull();
	/** 
	 * Nobody needs the constructor.
	 *
	 */
    private RNull(){}
    
    /**
     * Returns the RNull object.  There is no need for more than
     * one of these objects.  
     * @return
     */
    public static RNull getRNull(){ return theNullObject; }
    
	/**
     * A Null is only equal to the null object
     */
	public boolean equals(IRObject o) {
		return o.type()==type;
	}

	public String stringValue() {
		return "";
	}

	/**
	 * A Null is only equal to the null object, i.e. any 
	 * Null object.  They are all equal.
	 */
	public boolean equals(Object arg0) {
		return arg0.getClass().equals(this.getClass());
	}

	/**
	 * We override toString() to simply return "null".
	 * Because we do this, we override hashcode() as well.
	 */
	public String toString() {
		return "";
	}
	
	public int hashCode() {
		return 0;
	}

	/**
	 * Returns the type for this object.
	 */
	public RType type() {
		return type;
	}

}
