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
    private String                  tag;
    private HashMap<String, Object> _attribs = null;
    private ArrayList<XMLNode>      _tags    = null;
    private Object                  body     = null;
    private XMLNode                 parent;
    
    public XMLTag(String tag, XMLNode parent){
        if(tag==null)tag = "root";
    	this.tag    = tag;
        this.parent = parent;
    }
    
    public String toString(){
        String r = "<"+tag;
        if(_attribs != null) for(String key : _attribs.keySet()){
            r +=" "+key +"='"+_attribs.get(key).toString()+"'";
        }
        if(body != null){
            String b = body.toString();
            if(b.length()>20){
                b = b.substring(0,18)+"...";
            }
            r +=">"+b+"</"+tag+">";
        }else if( _tags == null || _tags.size()==0 ){
           r += "/>";
        }else{
           r += ">";
        }
        return r;
    }
    
    
    @Override
	public int childCount() {
		// TODO Auto-generated method stub
		if(_tags == null) return 0;
		return _tags.size();
	}

	/* (non-Javadoc)
     * @see com.dtrules.mapping.XMLNode#addChild(com.dtrules.mapping.XMLNode)
     */
    @Override
    public void addChild(XMLNode node) {
        if(_tags == null){
        	_tags = new ArrayList<XMLNode>(5); 
        }
    	_tags.add(node);
    }
    
    /* (non-Javadoc)
     * @see com.dtrules.mapping.XMLNode#remove(com.dtrules.mapping.XMLNode)
     */
    @Override
    public void remove(XMLNode node){
        if(_tags!= null){
        	_tags.remove(node);
        }
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
        return _tags;
    }

    /**
     * @param tags the tags to set
     */
    public void setTags(ArrayList<XMLNode> tags) {
        this._tags = tags;
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
    
    public Type type(){ return Type.TAG; }

	@Override
	public Object getAttrib(String key) {
		if (_attribs == null) return null;
		return _attribs.get(key);
	}

	@Override
	public void setAttrib(String key, Object value) {
		if(_attribs == null){
			_attribs = new HashMap<String,Object>(2,1.0f);
		}
		_attribs.put(key,value);
		
	}

    public HashMap<String,Object> getAttribs(){
    	if(_attribs == null){
    		_attribs = new HashMap<String,Object>(2,1.0f);
    	}
    	return _attribs;
    }

	@Override
	public void clearRef() {
		if(_attribs != null && _attribs.size()==0){
			_attribs = null;
		}
		if(_tags !=null && _tags.size() == 0){
			_tags = null;
		}
	}
}
    