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

package com.dtrules.automapping.nodes;
import java.util.HashMap;

import com.dtrules.automapping.AutoDataMap;
import com.dtrules.automapping.LabelMap;
import com.dtrules.automapping.access.IAttribute;
import com.dtrules.xmlparser.XMLPrinter;


/**
 * @author ps24876
 *
 */
public class MapNodeRef extends AMapNode {
    private IMapNode                child;
        
    public String toString(){
        return getAttribute().getName()+"->"+child.toString();
    }
 
    public String getName(){
    	return getAttribute().getName();
    }
 
    public MapNodeRef(IAttribute attribute, IMapNode parent){
        super(attribute, parent);
    }
    
    public void addChild(IMapNode node) {
        child = node;
    }
    
    public void remove(IMapNode node){
        if (child == node) child = null;
    }
    
    public String getLabel() {
        return getAttribute().getName();
    }

    public Object getData() {
        return null;
    }

    public void setData(Object data) {
    }
  
    /**
     * @return the child
     */
    public IMapNode getChild() {
        return child;
    }

    /**
     * @return the attribs
     */
    public HashMap<String, Object> getAttribs() {
        return null;
    }
   
       @Override
    public Object mapNode(AutoDataMap autoDataMap, LabelMap labelMap) {
        Object ref = autoDataMap.getCurrentGroup().getDataTarget()
                         .mapRef(autoDataMap, labelMap, this);
        return ref;
    }
    
    public void printDataLoadXML(AutoDataMap autoDataMap, XMLPrinter xout) {
        if(child!=null){
            xout.opentag(getAttribute().getName(),
                "type",getAttribute().getType(),
                "subType",getAttribute().getSubType(),
                "node","reference");
                child.printDataLoadXML(autoDataMap, xout);
            xout.closetag();
        }
    }

}