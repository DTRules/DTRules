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

import java.io.IOException;
import java.lang.reflect.InvocationTargetException;
import java.util.ArrayList;
import java.util.HashMap;
import com.dtrules.entity.IREntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.RName;
import com.dtrules.session.DTState;
import com.dtrules.session.EntityFactory;
import com.dtrules.xmlparser.IGenericXMLParser;


/**
 * LoadMap class is used by the GenerixXMLParser to load the Map defining 
 * the XML to EDD transformation.
 * 
 * @author Paul Snow
 *
 */
class LoadMap implements IGenericXMLParser {
    /**
     * We track if we encountered any errors, but we print as many messages as we can.
     * Then we report if we failed to load, assuming anyone cares.
     */
    private boolean loadSuccessful = true;
    private HashMap<String, String > undefinedEntities;
    private HashMap<String, String > definedAttributes;
    
	private final DTState           state;
    private final EntityFactory     ef;
    private final Mapping           map;
    
	private HashMap<String,String>  _attribs;
	
	/**
     * @param map
     */
    LoadMap(DTState state, Mapping map) {
        this.map   = map;
        this.state = state;
        this.ef    = state.getSession().getEntityFactory();
    }
    
	/**
     * This boolean is set at the end of loading a file.
     * @return the loadSuccessful
     */
    public boolean isLoadSuccessful() {
        return loadSuccessful;
    }
    /**
     * 
     */

    public void begin_dataobjects(){}
    
    public void begin_do2entitymap(){
        
        String doClass       = _attribs.get("class");
        String key           = _attribs.get("key");
        String key_attribute = _attribs.get("key_attribute");
        String entity        = _attribs.get("entity");
        String entity_tag    = _attribs.get("entity_tag");
        ArrayList<DataObjectMap> dataobjmaps = map.dataObjects.get(doClass);
        
        if(dataobjmaps==null){
            dataobjmaps = new ArrayList<DataObjectMap>();
            map.dataObjects.put(doClass, dataobjmaps);
        }
        
        for(DataObjectMap dataobjmap : dataobjmaps){
           if(dataobjmap.entityName.equals(entity) && 
              dataobjmap.tag.equals(entity_tag)){
               throw new RuntimeException("Duplicate DO to Entity mappings for: "+entity);
           }
        }
        
        try {
          DataObjectMap dataobjmap = new DataObjectMap(doClass,entity,entity_tag,key,key_attribute);
          dataobjmaps.add(dataobjmap);
        } catch (Exception e) {
          // System.out.println("Undefined Data Object: '"+doClass+"'\n");
        }
    }
    
	/**
	 * Mapping tag. <mapping>
	 * used simply to organize the mapping xml under a single tag.
     * This does any initialization of the mapping state
	 *
	 */
	public void begin_mapping () {
        undefinedEntities = new HashMap<String,String>();
        definedAttributes = new HashMap<String,String>();
        loadSuccessful = true;   
    }
	
	/**
	 * Contains the tags that define the XML to EDD mapping.  Right
	 * now that is all we implement.  In the future we will add an
	 * EDD to XML mapping set of tags, maybe.
	 *
	 */
	public void begin_XMLtoEDD () {}
	
	/**
	 * Tag groups all the entity tags.
	 */
	public void begin_entities () {}
	   
	
	/**
	 * Process an Entity tag.  Example:
	 *    <entity name="individual" number="+"/>
	 * Valid number specifications are:
	 *    "*" (0 or more)
	 *    "+" (1 or more)
	 *    "1" (1)   
	 */
	public void begin_entity () {
		String entity     = ((String) _attribs.get("name")).toLowerCase();
		String number     = (String) _attribs.get("number");
		
		IREntity rEntity = map.getSession().getEntityFactory().findRefEntity(RName.getRName(entity));
		if(rEntity==null){
		    System.out.println("The Entity specified, '"+entity+"' is not defined");
		    loadSuccessful = false;
		}
		if(number.equals("1") || number.equals("+") || number.equals("*")){
			this.map.entityinfo.put(entity,number);
		}else{
            state.traceInfo("error","Number value must be '1', '*', or '+'.  Encounterd: "+number);
			throw new RuntimeException("Number value must be '1', '*', or '+'.  Encounterd: "+number);
		}
	}

	/**
	 * Every entity that defines the given attribute gets its instances logged
	 * in to the given list.
	 */
	public void addalltolist () {
		String withAttribute = (String) _attribs.get("withAttribute");
		String toList        = (String) _attribs.get("toList");
		String pair[] = {withAttribute.toLowerCase(),toList.toLowerCase()};
		map.attribute2listPairs.add(pair);
		
	}
	
	/**
	 * Groups initalentity tags.
	 *
	 */
	public void begin_initialization(){}
	
	/**
	 * Defines an entity to be placed on the Entity Stack at initialization.
	 *
	 */
	public void begin_initialentity(){
		String entity = (String) _attribs.get("entity");
		this.map.entitystack.add(entity.toLowerCase());
	}	
	
	/**
	 * Groups all of the mapping tags.
	 *
	 */
	public void begin_map(){}
	
	
	/**
	 * Saves away the information required to create an entity.
	 *
	 */
	public void begin_createentity(){
		final String entity    = (String) _attribs.get("entity");
		final String tag       = (String) _attribs.get("tag");
		final String attribute = (String) _attribs.get("attribute");
		final String value     = (String) _attribs.get("value");
		final String id        = (String) _attribs.get("id");
              String list      = (String) _attribs.get("list");
        list = list==null?"":list;
        
        try {
            IREntity theentity = ef.findRefEntity(RName.getRName(entity));
            if(theentity==null)throw new Exception();
        } catch (Exception e) {
            System.out.println("\nThe Entity found in the Map File => "+entity+" <= isn't defined by the EDD.");
            loadSuccessful = false;
        }
        
		EntityInfo info = new EntityInfo();
		info.id = id;
		info.name = entity.toLowerCase();
		info.list = list.toLowerCase().trim();
		if(attribute== null || value==null ){
           this.map.requests.put(tag,info);
        }else{
		   this.map.multiple.put(tag,attribute);	
		   this.map.requests.put(value,info);
		}
	}
	
	
	
	/**
	 * The Rules Engine expects everything to be lowercase!!
	 * Be careful how you use attributes!
	 */
	public void begin_setattribute(){
		String        tag        = (String) _attribs.get("tag");				
		String        type       = (String) _attribs.get("type");						
		String        entity  = (String) _attribs.get("enclosure");		// This is an optional entity enclosure....		
		if(entity == null){
		              entity  = (String) _attribs.get("entity");         // This is an optional entity enclosure....
		}		
		String        rattribute = (String) _attribs.get("RAttribute");     // This is the Entity Attribute name 

		if(rattribute == null)rattribute = tag;                             // If no rattribute name is specified, default to the tag.
	   
		if(entity!=null)entity = entity.toLowerCase();
		
 	    AttributeInfo info = (AttributeInfo) this.map.setattributes.get(tag);
	    if(info==null){
	   		info = new AttributeInfo();
	    }	    
	    try {
            IREntity e = ef.findRefEntity(RName.getRName(entity));
            if(e==null){
                if(!undefinedEntities.containsKey(entity)){
                   System.out.println("The entity "+entity+" isn't defined in the EDD");
                   loadSuccessful = false;
                   undefinedEntities.put(entity, entity);
                }   
            }else{
                if(definedAttributes.containsKey(entity+"*"+rattribute)){
                    System.out.println("The Entity "+entity+" and Attribute "+rattribute +" have multiple definitions");
                    loadSuccessful = false;
                }
                if(e.getEntry(RName.getRName(rattribute))==null){
                    System.out.println("The Attribute "+rattribute+" isn't defined by "+entity);
                    loadSuccessful = false;
                }
                info.add(state,tag, entity,rattribute.toLowerCase(),type);
            }    
        } catch (RulesException e) {}
           
	    this.map.setattributes.put(tag,info);
	}			

	
	/**
	 * Because all of the tags possible within a Mapping XML are defined, and because we only have to load the Mapping
	 * File once at initialization, all we do here is take our parameters and store them away within our class, then
	 * by intraspection load the proper method for this tag, and execute it.
	 */
	@SuppressWarnings({"unchecked"})
	public void beginTag(String[] tagstk, int tagstkptr, String tag, HashMap attribs) throws IOException, Exception {
		
		_attribs   = (HashMap<String,String>)attribs;

        if(state.testState(DTState.VERBOSE)){
            state.traceTagBegin(tag, attribs);
        }
        
        try {
			this.getClass().getMethod("begin_"+tag, (Class[])null).invoke(this,(Object [])null);
		} catch (IllegalArgumentException e) {
			e.printStackTrace();
		} catch (SecurityException e) {
			e.printStackTrace();
		} catch (IllegalAccessException e) {
			e.printStackTrace();
		} catch (InvocationTargetException e) {
			System.out.println( e.getCause().getMessage());
			loadSuccessful = false;
		} catch (NoSuchMethodException e) {
            System.out.println("No implmentation for "+tag+"()");
			throw new RuntimeException("Undefined tag found in mapping file: "+tag);
		}
	}

	/**
	 * We are not doing anything on an end tag right now.
	 */
	@SuppressWarnings({"unchecked"})
	public void endTag(String[] tagstk, int tagstkptr, String tag, String body, HashMap attribs) throws Exception, IOException {
	    if(state.testState(DTState.VERBOSE)){
            state.traceTagEnd();
        }
		
	}
    /**
     *  All errors throw an Exception. If we run into problems, there is no
     *  recovery, no fix.  
     */
	public boolean error(String v) throws Exception {
		return true;
	}
    
}