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

package com.dtrules.mapping;

import com.dtrules.entity.IREntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RArray;
import com.dtrules.interpreter.RName;
import com.dtrules.session.DTState;
import com.dtrules.session.IRSession;

/**
 * Holds information about each entity expected in the XML input data.
 * @author paul snow
 *
 */
class EntityInfo {
	String  id;
    String  name;
    IREntity entity;
    String  number;
    String  list;
    /**
     * Get a new instance of an Entity upon a reference in the data file.
     * If only one instance of an Entity is to be created, then return
     * that one.  If the Entity is a constant Entity, return the constant
     * Entity.  Otherwise create a new clone and return that.
     * @return
     */
    IREntity getInstance(IRSession s)throws RulesException{
        if(number.equals("1"))return entity;
        DTState state = s.getState();
        IRObject rarray = state.find(RName.getRName(name+"s"));
        
        IREntity newentity = (IREntity) entity.clone(s);
        if(rarray!=null && rarray.type().getId() ==IRObject.iArray){
           ((RArray)rarray).add(newentity); 
           if (state.testState(DTState.TRACE)) {
               state.traceInfo("addto", "arrayId", ((RArray)rarray).getID() + "", newentity.postFix());
           }
        }
        return newentity;
    }
}