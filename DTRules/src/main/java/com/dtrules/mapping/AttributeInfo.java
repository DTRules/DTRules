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
import java.util.Iterator;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.session.DTState;

/**
 * A structure for holding information about Attributes.  Instances of AttributeInfo are stored in the setAttributes HashMap.
 * Note that Attributes are a bit more complex than Entities.  We allow not only the tag to be specfied, but the enclosing tag
 * for the identification of an Attribute.  To further complicate things, we can specify different target attributes...
 * @author ps24876
 *
 */
@SuppressWarnings({"unchecked"})
class AttributeInfo {
	
	public static final String DATE_STRING_TYPE     = "DateString";  //Strings
	public static final String STRING_TYPE          = "String";
	public static final String INTEGER_TYPE         = "Integer";
	public static final String NONE_TYPE			= "None";
	public static final String FLOAT_TYPE			= "Float";
    public static final String BOOLEAN_TYPE         = "Boolean";
    public static final String ARRAY_TYPE           = "list";
	public static final String ENTITY_TYPE          = "entity";
	public static final String XMLVALUE_TYPE        = "XmlValue";
    
	public static final int DATE_STRING_CODE    = 0;             //Codes
	public static final int STRING_CODE         = 1;
	public static final int NONE_CODE			= 2;
	public static final int INTEGER_CODE        = 3;
	public static final int FLOAT_CODE          = 4;
    public static final int BOOLEAN_CODE        = 5;
    public static final int ARRAY_CODE          = 6;
    public static final int ENTITY_CODE         = 7;
    public static final int XMLVALUE_CODE       = 8;
    
	public static final String int2str[] = {DATE_STRING_TYPE,
		                                    STRING_TYPE,
		                                    NONE_TYPE,
		                                    INTEGER_TYPE,
		                                    FLOAT_TYPE,
                                            BOOLEAN_TYPE,
                                            ARRAY_TYPE,
                                            ENTITY_TYPE,
                                            XMLVALUE_TYPE,
		                                    };
	
	/**
	 * This is the information we collect for an attribute
	 */
	public class Attrib {
		String enclosure;          // This is the XML Tag that must enclose this attribute.          
		String rAttribute;         // This is the Rules Engine attribute to assign the value to
		String rEntity;            // If an Entity type, this is the Entity to create.
		int    type;               // This is the type for this attribute.		
	}
	/**
	 * A list of Attrib values for a given attribute mapping.
	 */
	ArrayList<Attrib> tag_instances = new ArrayList<Attrib>();
	
	public ArrayList<Attrib> getTag_instances() {
        return tag_instances;
	}

    /**
	 * We add an enclosure attribute pair to these arraylists.  An
	 * exception is thrown if we get two of the same enclosures.
	 * @param _entity
	 * @param _attribute
	 * @param _type
	 */
	public void add(DTState state, 
            final String tag, 
            final String _entity, 
            final String _attribute,
                  String _type)throws RulesException {
                
		final Iterator iattribs = tag_instances.iterator();
		int attribType = -1;
        if(_type.equalsIgnoreCase("date"))_type = "datestring";
        if(_type.equalsIgnoreCase("time"))_type = "datestring";
        for(int i=0;i<int2str.length;i++){
            if(_type.equalsIgnoreCase(int2str[i])){
                attribType = i;
                break;
            }
        }
       
        if(attribType == -1){
			throw new RuntimeException("Invalid mapping type encountered in mapping file: "+_type);  //NOPMD
		}
		
		while(iattribs.hasNext()){
			final Attrib attrib = (Attrib)iattribs.next();			
			/* Duplicate attributes may be encountered in mapping xml. 
			   So we won't throw this error. */
			if(attrib.enclosure.equalsIgnoreCase(_entity))	{
				if (" ".equals(attrib.rAttribute)) {
					attrib.rAttribute = _attribute;  
				}
				
				if (attrib.type == NONE_CODE) {
					attrib.type = attribType;
				}
						
				boolean thisisanerror = true;
				
				if(attrib.rAttribute.equalsIgnoreCase(_attribute)&& 
                   int2str[attrib.type].equalsIgnoreCase(_type)){
					  thisisanerror = false;
				}
				if (thisisanerror){
                    state.traceInfo("error", "Duplicate:" + _entity + "." + tag);
                }
                state.traceInfo("error", 
				         (thisisanerror?"ERROR: ":"WARNING: ") + "\n"+
				          "The tag <"+tag+"> and enclosure <"+_entity+"> "+
						  "have been encountered more than once in this mapping file\n"+
				          "For "+ (_entity==""?"":"<"+_entity+"> ") +tag+"> \n"+
						           "  Existing: RAttribute '"+attrib.rAttribute+"'\n"+
						           "            type      '"+int2str[attrib.type]+"'\n"+
						           "  New:      RAttribute '"+_attribute+"'\n"+
						           "            type      '"+_type+"'\n");                        
				if(thisisanerror) throw new RuntimeException("Duplicate Enclosures encountered"); 
			}
		}	
		final Attrib attrib = new Attrib();
		attrib.enclosure  = _entity==null?"":_entity;
		attrib.rAttribute = _attribute;		
		attrib.type       =  attribType;
		tag_instances.add(attrib);
	}
	
	/**
	 * Looks up the given enclosure, and returns the attribute
	 * associated with that enclosure, or a null if not found.
	 * @param _enclosure
	 * @return
	 */
	public Attrib lookup(String _enclosure){
		if(_enclosure==null) _enclosure="";
		Iterator iattribs = tag_instances.iterator();
		while(iattribs.hasNext()){
			Attrib attrib = (Attrib) iattribs.next();
			if(attrib.enclosure.equalsIgnoreCase(_enclosure)){
			   return attrib;
			}
		}
		return null;
	}

	public boolean elementExists(String tag, String _enclosure)
	{
		if(_enclosure==null) _enclosure="";
		Iterator iattribs = tag_instances.iterator();
		while(iattribs.hasNext()){
			Attrib attrib = (Attrib)iattribs.next();
			if(attrib.enclosure.equalsIgnoreCase(_enclosure))
			{
				return true;
			}
		}
		return false;
	}

}