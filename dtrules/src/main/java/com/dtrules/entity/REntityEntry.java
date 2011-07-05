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

  
package com.dtrules.entity;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RName;
import com.dtrules.interpreter.RType;
import com.dtrules.session.RSession;

/**
 * 
 *  A Entry holds the attribute type and attribute value as held in an
 *  Entity.  Past implementations of the Rules Engine used a separate
 *  Attribute and Value hashmaps.  However, too many operations require
 *  both an inspection of the attribute as well as the value.  So now
 *  I have one hashmap for the attribute, which provides an index into
 *  an ArrayList to get the value.
 *   
 * @author Paul Snow
 *
 */
public class REntityEntry {
	public  REntity   entity;               // The entity involved.
    public  RName     attribute;            // The name of this attribute.
    public  String    defaulttxt;           // Text for the default value.
    public  IRObject  defaultvalue;	        // Every Entry has default value, which may be null.
    public  boolean   writable;             // We allow Entries to be locked.
    public  boolean   readable;             // We allow some Entries to be write only.
    public  RType     type;                 // The type value, i.e. integer, float, etc. as defined
    										// by IRObject
    public  String    subtype;              // The Subtype (for some kinds of attributes)...
    public  int       index;			    // Index into the values array to get the current
                                            //   value for this attribute.
    public String     comment;              // A comment associated with this attribute
    public String     input;                // These are the mapping sources which populate this attribute
    public String     output;               // Entries to be auto updated in the source objects
    /**
     * Allows the insertion of the REntityEntry into an Entity after the
     * fact.
     * @param newIndex The index into the values array where the value of this attribute can be found.
     */
    void updateIndex(int newIndex){         
        index = newIndex;
    }
    
    REntityEntry(
            REntity  _entity,
            RName    _attribute,
            String   _defaulttxt,
            IRObject _defaultvalue, 
    		boolean  _writable,
    		boolean  _readable,
    		RType    _type,
    		String   _subtype,
    		int      _index,
    		String   comment,
    		String   input,
    		String   output){
        
        attribute    = _attribute;
        defaulttxt   = _defaulttxt;
    	defaultvalue = _defaultvalue;
    	writable     = _writable;
    	readable     = _readable;
    	type         = _type;
    	subtype      = _subtype;
    	index        = _index;
    	setComment(comment);
    	setInput(input);
    	this.output  = output;
    }
   
    @Override
    public String toString() {
    	String thetype="";
        try{
        	thetype = "("+type+")";
        }catch(Exception e){}
        return thetype +" default: "+defaulttxt;
    }

	/**
	 * @return the attribute
	 */
	public RName getAttribute() {
		return attribute;
	}

	/**
	 * @param attribute the attribute to set
	 */
	public void setAttribute(RName attribute) {
		this.attribute = attribute;
	}

	/**
	 * @return the defaulttxt
	 */
	public String getDefaulttxt() {
		return defaulttxt;
	}

	/**
	 * @param defaulttxt the defaulttxt to set
	 */
	public void setDefaulttxt(String defaulttxt) {
		this.defaulttxt = defaulttxt;
	}

	/**
	 * @return the defaultvalue
	 */
	public IRObject getDefaultvalue() {
		return defaultvalue;
	}

	/**
	 * @param defaultvalue the defaultvalue to set
	 */
	public void setDefaultvalue(IRObject defaultvalue) {
		this.defaultvalue = defaultvalue;
	}

	/**
	 * @return the type
	 */
	public RType getType() {
		return type;
	}

	/**
	 * @param type the type to set
	 */
	public void setType(RType type) throws RulesException {
		this.type = type;
	}

	/**
	 * @return the writable
	 */
	public boolean isWritable() {
		return writable;
	}

	/**
	 * @param writable the writable to set
	 */
	public void setWritable(boolean writable) {
		this.writable = writable;
	}
	
	public String getAttributeStrValue() {
		return attribute.stringValue();
	}

	public void setAttributeStrValue(String attributeStrValue) {
		setAttribute(RName.getRName(attributeStrValue));
	}

    public final int getIndex() {
        return index;
    }

    public final void setIndex(int index) {
        this.index = index;
    }

    /**
     * @return the subtype
     */
    public String getSubtype() {
        return subtype;
    }

    /**
     * @return the readable
     */
    public boolean isReadable() {
        return readable;
    }

    /**
     * @param readable the readable to set
     */
    public void setReadable(boolean readable) {
        this.readable = readable;
    }

    /**
     * @return the comment
     */
    public String getComment() {
        return comment;
    }

    /**
     * @param comment the comment to set
     */
    public void setComment(String comment) {
        this.comment = comment;
    }

    /**
     * @param subtype the subtype to set
     */
    public void setSubtype(String subtype) {
        this.subtype = subtype;
    }

    /**
     * @return the input
     */
    public String getInput() {
        return input;
    }

    /**
     * @param input the input to set
     */
    public void setInput(String input) {
        this.input = input;
    }
    
    
}
