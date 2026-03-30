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

package com.dtrules.automapping.access;

import com.dtrules.automapping.MapType;
import com.dtrules.xmlparser.XMLPrinter;

/**
 * @author ps24876
 *
 */
public interface IAttribute {

    /**
     * @return the name
     */
    String getName();

    /**
     * @return the type
     */
    MapType getType();

    /**
     * @return the typeText
     */
    String getTypeText();

    /**
     * @return the subType
     */
    MapType getSubType();

    /**
     * @return the subTypeText
     */
    String getSubTypeText();

    /**
     * Get the attribute value from the given Object
     * @param obj
     * @return
     */
    Object get(Object obj);

    /**
     * Set the attribute value on the given Object
     * @param obj
     * @param label
     */
    void set(Object obj, Object value);

    /**
     * Returns true if this is the key for the object to which the attribute
     * belongs.
     * @return
     */
    boolean isKey();
    
    void printXML(XMLPrinter xout);

}