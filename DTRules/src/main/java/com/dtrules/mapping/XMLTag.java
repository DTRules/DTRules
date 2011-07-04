/** 
 * Copyright 2004-2009 DTRules.com, Inc.
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
 * This object represents an XML tag, but I avoid serializaiton under
 * the assumption that any conversion to a string would have been 
 * undone by a conversion from a string back to the object anyway. I
 * also use the same structure for both tagged data and a tag holding
 * tags.
 * 
 * @author Paul Snow
 * Sep 24, 2007
 *
 */
public class XMLTag implements XMLNode {
    String                  tag;
    HashMap<String, Object> attribs = new HashMap<String,Object>();
    ArrayList<XMLNode>      tags    = new ArrayList<XMLNode>();
    Object                  body    = null;
    XMLNode                 parent;
    
    public XMLTag(String tag, XMLNode parent){
        this.tag    = tag;
        this.parent = parent;
    }
    
    public String toString(){
        String r = "<"+tag;
        for(String key : attribs.keySet()){
            r +=" "+key +"='"+attribs.get(key).toString()+"'";
        }
        if(body != null){
            String b = body.toString();
            if(b.length()>20){
                b = b.substring(0,18)+"...";
            }
            r +=">"+b+"</"+tag+">";
        }else if( tags.size()==0 ){
           r += "/>";
        }else{
           r += ">";
        }
        return r;
    }
    
    /* (non-Javadoc)
     * @see com.dtrules.mapping.XMLNode#addChild(com.dtrules.mapping.XMLNode)
     */
    @Override
    public void addChild(XMLNode node) {
       tags.add(node);
    }
    
    /* (non-Javadoc)
     * @see com.dtrules.mapping.XMLNode#remove(com.dtrules.mapping.XMLNode)
     */
    @Override
    public void remove(XMLNode node){
        tags.remove(node);
    }
    
    /**
     * @return the tag
     */
    public String getTag() {
        return tag;
    }

    /**
     * @param tag the tag to set
     */
    public void setTag(String tag) {
        this.tag = tag;
    }

    /**
     * @return the tags
     */
    public ArrayList<XMLNode> getTags() {
        return tags;
    }

    /**
     * @param tags the tags to set
     */
    public void setTags(ArrayList<XMLNode> tags) {
        this.tags = tags;
    }

    /**
     * @return the body
     */
    public Object getBody() {
        return body;
    }

    /**
     * @param body the body to set
     */
    public void setBody(Object body) {
        this.body = body;
    }

    /**
     * @return the parent
     */
    public XMLNode getParent() {
        return parent;
    }

    /**
     * @param parent the parent to set
     */
    public void setParent(XMLTag parent) {
        if(this.parent != null){
            this.parent.remove(this);
        }
        this.parent = parent;
    }

    /**
     * @return the attribs
     */
    public HashMap<String, Object> getAttribs() {
        return attribs;
    }
    
    public Type type(){ return Type.TAG; }
}