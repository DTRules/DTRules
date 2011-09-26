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
/**
 * A Mark object is used as a place holder on the data stack.
 * arraytomark then pops all elements upto but not including the
 * mark, and puts them into an array.
 * 
 * @author paul snow
 *
 */
public class RMark extends ARObject {

	static RType type = RType.newType("mark");

	static RMark themark = new RMark();
    
    private RMark(){}
    
    static public RMark getMark() { return themark; }
    
    public String stringValue() {
        return "Mark";
    }

	/**
	 * Returns the type for this object.
	 */
	public RType type() {
		return type;
	}


}
