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

import java.util.ArrayList;
import java.util.HashMap;

/**
 * @author ps24876
 *
 */
public interface XMLNode {

    public enum Type { TAG, COMMENT, HEADER } 
        
    public Type type ();
    
    public abstract String toString();

    /**
     * Adds the given XMLNode to the children of this tag
     */
    public abstract void addChild(XMLNode node);
    
    /**
     * Returns number of children
     */
    public int childCount();
    
    /**
     * Removes any reference to the given node from
     * this XML Node 
     */
    public abstract void remove(XMLNode node);
    /**
     * @return the tag
     */
    public abstract String getTag();

    /**
     * @param tag the tag to set
     */
    public abstract void setTag(String tag);

    /**
     * @return the tags
     */
    public abstract ArrayList<XMLNode> getTags();
    
    /**
     * @return the body
     */
    public abstract Object getBody();

    /**
     * @param body the body to set
     */
    public abstract void setBody(Object body);

    /**
     * @return the parent
     */
    public abstract XMLNode getParent();

    /**
     * @param parent the parent to set
     */
    public abstract void setParent(XMLTag parent);

    /**
     * @return the attribute
     */
    public abstract Object getAttrib(String key);

    /**
     * Set the attribute
     * @param key
     * @param value
     */
    public abstract void setAttrib(String key, Object value);
    
    public abstract HashMap<String,Object> getAttribs();
    
    /**
     * Clear any unused structures
     */
    public void clearRef();
}