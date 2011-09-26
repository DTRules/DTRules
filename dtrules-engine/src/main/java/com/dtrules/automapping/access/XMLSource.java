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

import java.util.ArrayList;
import java.util.List;

import com.dtrules.automapping.AutoDataMap;
import com.dtrules.automapping.AutoDataMapDef;
import com.dtrules.automapping.Group;
import com.dtrules.automapping.Label;
import com.dtrules.automapping.nodes.MapNodeAttribute;
import com.dtrules.automapping.nodes.MapNodeList;
import com.dtrules.automapping.nodes.MapNodeMap;
import com.dtrules.automapping.nodes.MapNodeObject;
import com.dtrules.automapping.nodes.MapNodeRef;
import com.dtrules.xmlparser.XMLTree.Node;

/**
 * @author ps24876
 *
 */
public class XMLSource implements IDataSource {

    final AutoDataMapDef  autoDataMapDef;
    final String          type = "xml";
    
    public XMLSource(AutoDataMapDef autoDataMapDef){
        this.autoDataMapDef = autoDataMapDef;
    }
    
    @Override
    public Label createLabel(AutoDataMap autoDataMap, Group group,
            String labelName, String key, boolean singular, Object object) {
        
        Node xmlnode = (Node)object;
        
        if(!xmlnode.getAttributes().containsKey("node"))return null;
        
        Label labelObj = group.findLabel(labelName,"xml",labelName);
        
        if(labelObj == null ){
            labelObj = Label.newLabel(group,labelName,labelName,key,singular);
        }
        if(labelObj.isCached())return labelObj;
        labelObj.setCached(true);
        
        for(Node xmltag : xmlnode.getTags()){
            String node    = xmltag.getAttributes().get("node");
            String type    = xmltag.getAttributes().get("type");
            String subType = xmltag.getAttributes().get("subType");
            // First check if this is just a tag wrapping the dataload file.  If so, 
            // just claim it is a List.
            if(type==null){
                node    = "object";
                type    = "list";
                subType = "object";
            }
            if(node.equals("primitive")){
                type    = xmltag.getName();
                subType = "";
                
            }
            
            IAttribute a = new XMLAttribute(
                    xmltag.getName(), 
                    labelObj, 
                    type, subType, 
                    xmltag.getAttributes());
            
            labelObj.getAttributes().add(a);
        }
        return labelObj;
    }
    
    @Override
    public String getKey(Object obj) {
        if(obj instanceof Node){
            String key = ((Node)obj).getName()+"Id";
            return key;
        }
        return "";
    }

    @Override
    public String getName(Object obj) {
        return ((Node)obj).getName();
    }

    /* (non-Javadoc)
     * @see com.dtrules.automapping.access.IDataSource#getSpec(java.lang.Object)
     */
    @Override
    public String getSpec(Object obj) {
        return ((Node) obj).getName();
    }

    
    @Override
    public List<?> getChildren(Object obj){
        if(obj instanceof Node){
            return ((Node)obj).getTags();
        }
        return new ArrayList<Object>();    
    }
    /**
     * We don't need this mechanism for Java Objects as it is pretty easy for us
     * to just go grab the key.
     */
    public Object getKeyValue(MapNodeObject node, Object object){
        if(object instanceof Node){
            Object key = ((Node)object).getAttributes().get("key");
            if(key != null  && key.toString().length()!=0)return key;
        }
        return node.getKey();
    }

    @Override
    public void update(AutoDataMap autoDataMap, MapNodeAttribute node) {
        
    }

    @Override
    public void update(AutoDataMap autoDataMap, MapNodeList node) {
        
    }

    @Override
    public void update(AutoDataMap autoDataMap, MapNodeMap node) {

    }

    @Override
    public void update(AutoDataMap autoDataMap, MapNodeObject node) {
        
    }

    @Override
    public void update(AutoDataMap autoDataMap, MapNodeRef node) {
        
    }

}
