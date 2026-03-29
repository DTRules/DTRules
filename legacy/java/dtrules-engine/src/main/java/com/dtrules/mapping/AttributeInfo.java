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
import java.util.Iterator;
import java.util.Map;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.RName;
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
	
    static Map<RName, Integer> string2Type = new HashMap<RName, Integer>();
    static Map<Integer, RName> type2String = new HashMap<Integer, RName>();
    
    static {
        string2Type.put(RName.getRName("DateString"),  0 );
        string2Type.put(RName.getRName("Date"),        0 );
        string2Type.put(RName.getRName("String"),      1 );
        string2Type.put(RName.getRName("Integer"),     2 );
        string2Type.put(RName.getRName("None"),        3 );
        string2Type.put(RName.getRName("Double"),      4 );
        string2Type.put(RName.getRName("Float"),       4 );
        string2Type.put(RName.getRName("Boolean"),     5 );
        string2Type.put(RName.getRName("List"),        6 );
        string2Type.put(RName.getRName("Array"),       6 );
        string2Type.put(RName.getRName("Entity"),      7 );
        string2Type.put(RName.getRName("XmlValue"),    8 );
        
        type2String.put(0, RName.getRName("Date"));
        type2String.put(1, RName.getRName("String"));
        type2String.put(2, RName.getRName("Integer"));
        type2String.put(3, RName.getRName("None"));
        type2String.put(4, RName.getRName("Double"));
        type2String.put(5, RName.getRName("Boolean"));
        type2String.put(6, RName.getRName("Array"));
        type2String.put(7, RName.getRName("Entity"));
        type2String.put(9, RName.getRName("XmlValue"));
    }
    
	public static final int DATE_STRING_CODE    = 0;             //Codes
	public static final int STRING_CODE         = 1;
	public static final int NONE_CODE			= 2;
	public static final int INTEGER_CODE        = 3;
	public static final int FLOAT_CODE          = 4;
    public static final int BOOLEAN_CODE        = 5;
    public static final int ARRAY_CODE          = 6;
    public static final int ENTITY_CODE         = 7;
    public static final int XMLVALUE_CODE       = 8;
    	
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
		
		Integer attribType = string2Type.get(RName.getRName(_type));
        
         if(attribType == null){
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
                   attrib.type == string2Type.get(RName.getRName(_type))){
					  thisisanerror = false;
				}
				if (thisisanerror){
                    state.traceInfo("error", "Duplicate:" + _entity + "." + tag);
                }
				
                String errorString =
                    (thisisanerror?"ERROR: ":"WARNING: ") + "\n"+
                    "The tag <"+tag+"> and enclosure <"+_entity+"> "+
                    "have been encountered more than once in this mapping file\n"+
                    "For "+ (_entity==""?"":"<"+_entity+"> ") +tag+"> \n"+
                             "  Existing: RAttribute '"+attrib.rAttribute+"'\n"+
                             "            type      '"+type2String.get(attrib.type).stringValue()+"'\n"+
                             "  New:      RAttribute '"+_attribute+"'\n"+
                             "            type      '"+_type+"'\n";                        
				state.traceInfo("error", errorString);
				if(thisisanerror) throw new RuntimeException("Duplicate Enclosures encountered:\r\n"+errorString); 
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
		int end =  tag_instances.size();
		Attrib attrib;
		for(int i=0; i < end; i++){
			attrib = tag_instances.get(i);
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