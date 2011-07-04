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

import com.dtrules.automapping.AutoDataMap;
import com.dtrules.automapping.LabelMap;
import com.dtrules.automapping.nodes.MapNodeAttribute;
import com.dtrules.automapping.nodes.MapNodeList;
import com.dtrules.automapping.nodes.MapNodeMap;
import com.dtrules.automapping.nodes.MapNodeObject;
import com.dtrules.automapping.nodes.MapNodeRef;
import com.dtrules.interpreter.IRObject;


/**
 * A Target Interface for writing data to a given target type,
 * such as Java Objects or DTRules Entities.
 * 
 * @author Paul Snow
 *
 */
public interface IDataTarget {    
  
    /**
     * Give the target a chance to initialize itself, if necessary.
     * @param autoDataMap
     */
    void init(AutoDataMap autoDataMap);
    /**
     * Maps the data held in the given IMapNode to the target.  The target
     * object updated is returned, or a null if no update occurs.
     * @param autoDataMap
     * @param labelMap
     * @param node
     */
    Object mapObject(AutoDataMap autoDataMap, LabelMap labelMap, MapNodeObject node);
    /**
     * Maps the attribute in the given IMapNode to the target.
     * @param autoDataMap
     * @param labelMap
     * @param node
     */
    Object mapAttribute(AutoDataMap autoDataMap, LabelMap labelMap, MapNodeAttribute node);
    
    /**
     * Maps a list over to the Target Data space
     * @param autoDataMap
     * @param labelMap
     * @param node
     * @return
     */
    Object mapList(AutoDataMap autoDataMap, LabelMap labelMap, MapNodeList node);

    /**
     * Map a hashmap from the datamap to an RTable in the rules engine
     * @param autoDataMap
     * @param node
     * @return
     */
    public Object mapMap(AutoDataMap autoDataMap, LabelMap labelMap, MapNodeMap node);

    
    /**
     * A Reference attribute is a direct reference to another object.  This call maps
     * that other reference, then updates the Reference Attribute.
     * @param autoDataMap
     * @param labelMap
     * @param node
     * @return
     */
    Object mapRef(AutoDataMap autoDataMap, LabelMap labelMap, MapNodeRef node);
    
    /**
     * Looks up the LabelMap needed to map this node.  If one doesn't exist, it is 
     * created.
     * @param autoDataMap
     * @param node
     * @return
     */
    public LabelMap getLabelMap(AutoDataMap autoDataMap, MapNodeObject node);
    /**
     * Convert a DataMap sort of primitive object into an object acceptable by the target
     * @param object
     * @return
     */
    public IRObject iconvert(Object object);
    
    /**
     * Return the default Label name for an Object from a given source.
     * @param obj
     * @return
     */
    public String getName(Object obj);
    
    /**
     * Updates Changes made in the target to Attribute values in the DataMap
     * @param autoDataMap
     * @param node
     */
    public void update(AutoDataMap autoDataMap, MapNodeAttribute    node);
    
    /**
     * Updates Changes made in the target to Maps in the DataMap
     * @param autoDataMap
     * @param node
     */    
    public void update(AutoDataMap autoDataMap, MapNodeMap          node);
    
    /**
     * Updates Changes made in the target to Objects in the DataMap
     * @param autoDataMap
     * @param node
     */
    public void update(AutoDataMap autoDataMap, MapNodeObject       node);
    
    /**
     * Updates Changes made in the target to Lists in the DataMap
     * @param autoDataMap
     * @param node
     */
    public void update(AutoDataMap autoDataMap, MapNodeList         node);
    
    /**
     * Updates Changes made in the target to Object References in the DataMap
     * @param autoDataMap
     * @param node
     */
    public void update(AutoDataMap autoDataMap, MapNodeRef          node);
       
}
