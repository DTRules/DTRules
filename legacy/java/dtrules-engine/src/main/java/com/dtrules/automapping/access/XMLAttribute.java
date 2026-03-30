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
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import com.dtrules.automapping.Label;
import com.dtrules.automapping.MapType;
import com.dtrules.xmlparser.XMLPrinter;
import com.dtrules.xmlparser.XMLTree.Node;

/**
 * @author ps24876
 *
 */
public class XMLAttribute implements IAttribute {
    
    String                name;
    Label                 label;
    final MapType         type;
    final String          typeText;
    MapType               subType;
    String                subTypeText;
    Map<String, String>   attribs;
    
    XMLAttribute (String name, Label label, String type, String subType, Map<String,String> attribs){
        this.name           = name;
        this.label          = label;
        this.type           = MapType.get(type);
        this.typeText       = type;
        this.subType        = MapType.get(subType);
        this.subTypeText    = subType;
        this.attribs        = attribs;
    }
    
    public String toString(){
        return name +" type: "+typeText+" subtype: "+subTypeText;
    }
    
    @Override
    public boolean isKey() {
        return name.equals(label.getKey());
    }

    /**
     * Get the value for this attribute. So what we have to do is look through
     * the children of this object for a match for this attribute.
     */
    public Object get(Object obj) {
        Node xmlnode = (Node)obj;
        for (Node xmlchild : xmlnode.getTags()){
            String childtype = xmlchild.getAttributes().get("type");
            if(xmlchild.getName().equals(this.name)){
                if(childtype.equalsIgnoreCase("object") ){
                    if(xmlchild.getTags().size() > 0){
                        return xmlchild.getTags().get(0);
                    }
                }else if (type == MapType.LIST){
                    if(subType.isPrimitive()){
                        List<Object> ret = new ArrayList<Object>();
                        for(Node listItem : xmlchild.getTags()){
                            ret.add(listItem.getBody());
                        }
                        return ret;
                    }else{
                        return xmlchild.getTags();
                    }
                }else if (childtype.equalsIgnoreCase("int")
                	   || childtype.equalsIgnoreCase("short")
                       || childtype.equalsIgnoreCase("integer")
                       || childtype.equalsIgnoreCase("long")
                       || childtype.equalsIgnoreCase("string")
                       || childtype.equalsIgnoreCase("boolean")
                       || childtype.equalsIgnoreCase("double")
                       || childtype.equalsIgnoreCase("date")){
                    String v = xmlchild.getBody();
                    return v;
                }else if (type == MapType.MAP){
                    Map<Object, Object> map = new HashMap<Object,Object>();
                    for(Node pair : xmlchild.getTags()){
                        Node keyNode   = pair.getTags().get(0);
                        Node valueNode = pair.getTags().get(1);
                        Object key = convert(keyNode);
                        Object value = convert(valueNode);
                        map.put(key, value);
                    }
                    return map;
                }
                return xmlchild;
            }
        }
        return null;
    }
    
    private Object convert(Node node){
        String type = node.getAttributes().get("type");
        if(type.equalsIgnoreCase("string")){
            return node.getBody();
        }else if (type.equalsIgnoreCase("int") || type.equalsIgnoreCase("integer")){
            if (node.getBody()==null)return null;
            return Integer.parseInt(node.getBody());
        }
        return node.getBody();
    }

    public String getName() {
        return name;
    }

    public MapType getSubType() {
        return subType;
    }

    public String getSubTypeText() {
        return subTypeText;
    }

    public MapType getType() {
        return type;
    }

    public String getTypeText() {
        return typeText;
    }

    @Override
    public void printXML(XMLPrinter xout) {

    }

    @Override
    public void set(Object obj, Object value) {

    }

}
