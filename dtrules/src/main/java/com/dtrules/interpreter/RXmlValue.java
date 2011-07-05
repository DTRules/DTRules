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

package com.dtrules.interpreter;

import java.util.ArrayList;
import java.util.Date;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.mapping.XMLNode;
import com.dtrules.session.DTState;

/**
 * @author paul snow
 * 
 * An XmlValue object represents a node in the XML input stream that
 * supplied data to the Rules Engine.  It provides a way to update
 * and modify that XML based on rules defined in Decision Tables.
 *
 */
public class RXmlValue extends ARObject {
	
	static RType type = RType.newType("xmlvalue");

    XMLNode tag;
    DTState state;
    int     id;
    
    public RXmlValue(DTState state, XMLNode tag){
        this.tag = tag;
        id = state.getSession().getUniqueID();
    }
    
    /**
     * Sets the value of an Attribute on the tag for this RXmlValue.
     *  
     * @param attribute
     * @param value
     */
    public void setAttribute(String attribute, String value){
       tag.setAttrib(attribute, value);
    }
    
    /**
     * Gets the value of an Attribute on the tag for this RXmlValue.
     * Returns a null if the Attribute isn't defined on this tag. 
     * @param attribute
     * @param value
     */
    public String getAttribute(String attribute){
       return tag.getAttrib(attribute).toString();
    }
    
    /**
     * The following are all the accessors that are suppored
     * for working with RXmlValue objects
     */
    
    public String toString(){
        if(tag.getBody()!= null){
            return tag.getBody().toString();
        }else{
            return null;
        }
    }
    /**
     * The string value of an XMLTag is its body value
     */
    public String stringValue() {
        return toString();
    }
    
	/**
	 * Returns the type for this object.
	 */
	public RType type() {
		return type;
	}


    public ArrayList<IRObject> arrayValue() throws RulesException {
        ArrayList<IRObject> a = new ArrayList<IRObject>();
        if(tag.getTags().size()>0){
           for(XMLNode t : tag.getTags()){
               a.add(new RXmlValue(state,t));
           }
        }
        return a;
    }

    @Override
    public boolean booleanValue() throws RulesException {
        if(tag.getBody()== null ) return super.booleanValue();
        return RBoolean.booleanValue(tag.getBody().toString());
    }

    @Override
    public double doubleValue() throws RulesException {
        if(tag.getBody()== null ) return super.doubleValue();
        return RDouble.getDoubleValue(tag.getBody().toString());
    }

    @Override
    public boolean equals(IRObject o) throws RulesException {
        return rStringValue().equals(o);
    }

    @Override
    public int intValue() throws RulesException {
        if(tag.getBody()== null ) return super.intValue();
        return (int)RInteger.getIntegerValue(tag.getBody().toString());
    }

   
    @Override
    public long longValue() throws RulesException {
        if(tag.getBody()== null ) return super.longValue();
        return RInteger.getIntegerValue(toString());
    }

    @Override
    public RBoolean rBooleanValue() throws RulesException {
        return RBoolean.getRBoolean(booleanValue());
    }

    @Override
    public RDouble rDoubleValue() throws RulesException {
        return RDouble.getRDoubleValue(doubleValue());
    }

    @Override
    public RInteger rIntegerValue() throws RulesException {
        return RInteger.getRIntegerValue(longValue());
    }

    @Override
    public RName rNameValue() throws RulesException {
        return RName.getRName(stringValue(),false);
    }

    public RString rStringValue() {
        if(tag.getBody()==null)return RString.newRString("");
        return RString.newRString(stringValue());
    }

    public RDate rTimeValue() throws RulesException {
        return RDate.getRTime(timeValue());
    }

    public Date timeValue() throws RulesException {
        return state.getSession().getDateParser().getDate(tag.getBody().toString());
    }

    /* (non-Javadoc)
     * @see com.dtrules.interpreter.ARObject#rXmlValue()
     */
    @Override
    public RXmlValue rXmlValue() throws RulesException {
        return this;
    }

    /* (non-Javadoc)
     * @see com.dtrules.interpreter.ARObject#xmlTagValue()
     */
    @Override
    public XMLNode xmlTagValue() throws RulesException {
        return tag;
    }

    
    
}