/** 
My * Copyright 2004-2009 DTRules.com, Inc.
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

import java.lang.reflect.Method;
import java.util.ArrayList;
import java.util.HashMap;

import com.dtrules.entity.IREntity;
import com.dtrules.entity.REntityEntry;
import com.dtrules.interpreter.RName;


/**
 * @author Paul Snow
 * This object provides functionality to copy the data within an Entity
 * to a Java Object.  It also handles mappings of lists of Entities to 
 * lists of Objects.  
 * <br><br>
 *
 */
public class EntityToObject {

    class e2oMapping {
        Method setValue = null;
    }
    
    HashMap<String,ArrayList<e2oMapping>> mappings = new HashMap<String,ArrayList<e2oMapping>>();
    /**
     * Creates a method name by adding the given prefix, and uppercasing the first
     * letter of the given base name.  So if given "is" as a prefix and "cached" as
     * a base, the resulting name would be "isCached".
     * <br><br>
     * Following Java naming conventions are very important for making this work.
     * @param prefix
     * @param base
     * @return
     */
    private String makeName(String prefix,String base){
        
        if(base.length()==0){
            return prefix;
        }
        
        if(base.length()==1){
            return prefix + base.substring(0,1).toUpperCase();
        }
        
        return prefix +
               base.substring(0, 1).toUpperCase()+
               base.substring(1);
    }
    
    @SuppressWarnings("unchecked")
    public void mapEntityToObject(IREntity entity, Object obj){
        
        ArrayList<Method> methods = new ArrayList<Method>();
        Class theClass = obj.getClass();
        
        for(Method method : theClass.getDeclaredMethods()){
            methods.add(method);
        }
        
        for (RName key : entity.getAttributeSet()){
            REntityEntry entry = entity.getEntry(key);
            String setName = makeName("set",key.stringValue());
            
            setName.substring(0, 1).toUpperCase();
        }
    }
    
}
