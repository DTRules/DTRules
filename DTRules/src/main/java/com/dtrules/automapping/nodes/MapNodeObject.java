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
import java.util.ArrayList;
import java.util.List;
import com.dtrules.automapping.AutoDataMap;
import com.dtrules.automapping.Group;
import com.dtrules.automapping.Label;
import com.dtrules.automapping.LabelMap;
import com.dtrules.automapping.access.IAttribute;
import com.dtrules.xmlparser.XMLPrinter;


/**
 * @author ps24876
 *
 */
public class MapNodeObject extends AMapNode {

    final private Label             sourceLabel;
    private Object                  key          = "";
    private Object                  source       = null;
    private Object                  targetObject = null;
    private String                  property     = null;
    private Group                   group        = null;
   
    
    public String toString(){
        return sourceLabel.getName();
    }
    
    public String getName(){
    	return sourceLabel.getName();
    }
 
    
    public Group getGroup (){
        return group;
    }
    
    
    public MapNodeObject(Group group, Label sourceLabel, IMapNode parent){
        super(null, parent);
        this.sourceLabel = sourceLabel;
        this.group       = group;
        if(sourceLabel == null){
            throw new RuntimeException("Map Objects must map a source");
        }
    }
        
    public String getLabel() {
        return sourceLabel.getName();
    }

    /**
     * @return the sourceLabel
     */
    public Label getSourceLabel() {
        return sourceLabel;
    }

    /**
     * @return the key
     */
    public Object getKey() {
        return key;
    }

    /**
     * @param key the key to set
     */
    public void setKey(Object key) {
        if(key == null)key = "";
        this.key = key;
    }

    /**
     * @return the source
     */
    public Object getSource() {
        return source;
    }

    /**
     * @param source the source to set
     */
    public void setSource(Object source) {
        this.source = source;
    }

    /**
     * @return the targetObject
     */
    public Object getTargetObject() {
        return targetObject;
    }

    /**
     * @param targetObject the targetObject to set
     */
    public void setTargetObject(Object targetObject) {
        this.targetObject = targetObject;
    }

    /**
     * @return the property
     */
    public String getProperty() {
        return property;
    }

    /**
     * @param property the property to set
     */
    public void setProperty(String property) {
        this.property = property;
    }

    @Override
    public Object mapNode(AutoDataMap autoDataMap, LabelMap labelMap) {
        
        Group target = autoDataMap.getCurrentGroup();
        
        labelMap = autoDataMap.getAutoDataMapDef().findLabelMap(sourceLabel.getName());
        
        if(labelMap == null ){
            labelMap = autoDataMap.getCurrentGroup().getDataTarget().getLabelMap(autoDataMap, this);
        }
        
        List<Object> objects = new ArrayList<Object>();
        while(labelMap != null){
            Object object = target.getDataTarget().mapObject(autoDataMap,  labelMap, this);
            
            // If we created an object, put it on our MapStack (for use by our children)
            if(object != null ){
                autoDataMap.push(object);   // Push an Entity
                objects.add(object);
            }else{
                autoDataMap.pushMark();     // Push a place holder; If we don't have an entity,
            }                               //   we still want to map the children.
            // Map the children.
            for(IMapNode child : getChildren()){
                child.mapNode(autoDataMap, labelMap);
            }
            // Pop the entity/mark off the object stack.
            autoDataMap.pop();
                        
            labelMap = labelMap.getNext();
        }

        return objects;
    }

    public void printDataLoadXML(AutoDataMap autoDataMap, XMLPrinter xout) {
        xout.opentag(sourceLabel.getName(),
                "node","object",
                "type","object",
                "group",group,
                "key", key);
            for(IMapNode a : getChildren()){
                a.printDataLoadXML(autoDataMap, xout);
            }
        xout.closetag();
    }    
  
    /**
     * Find an Attribute by name on the given MapNodeObject
     * @param attributeName
     * @return
     */
    public IAttribute findAttribute(String attributeName){
        this.sourceLabel.getAttributes();
        for(IMapNode node : getChildren()){
            if(node instanceof MapNodeAttribute){
                IAttribute ret = ((MapNodeAttribute)node).getAttribute();
                if(ret.getName().equals(attributeName)){
                    return ret; 
                }
            }
        }
        return null;
    }

    
}