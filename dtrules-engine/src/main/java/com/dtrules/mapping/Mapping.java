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

import java.io.FileNotFoundException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.io.Reader;
import java.io.StringReader;
import java.util.ArrayList;
import java.util.HashMap;

import com.dtrules.entity.IREntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.RName;
import com.dtrules.session.DTState;
import com.dtrules.session.IStreamSource.FileType;
import com.dtrules.session.IRSession;
import com.dtrules.session.RSession;
import com.dtrules.session.RuleSet;
import com.dtrules.session.RulesDirectory;
import com.dtrules.xmlparser.GenericXMLParser;

/**
 * The Mapping class is used to load XML formated data into an instance of
 * an EntityFactory.  
 * @author Paul Snow
 *
 */
public class Mapping {
	
    private final DTState state;
	 	 
    private final IRSession session;
     	
    HashMap<RName,EntityInfo>               entities      = new HashMap<RName,EntityInfo>();
    HashMap<String,String>                  multiple      = new HashMap<String,String>();   // Specifies a tag that uses an attribute to specify what entity to create.
	HashMap<String,EntityInfo>              requests      = new HashMap<String,EntityInfo>(); 
	HashMap<String,String>                  entityinfo	  = new HashMap<String,String>();
	HashMap<String,ArrayList<DataObjectMap>>dataObjects   = new HashMap<String,ArrayList<DataObjectMap>>();
    /**
     * List of attribute/list pairs.  Every entity that defines the given attribute has every instance saved in the given list.
     * Each entry is an array of two strings, the attribute and the list name.
     */
    ArrayList<String[]> attribute2listPairs = new ArrayList<String[]>();
    
	/**
	 * A List of entities that should be created at initialization and put on the Entity Stack.
	 */
	ArrayList<String> entitystack   = new ArrayList<String>(); // Specifies what Entities should be on the Entity Stack to begin with.
   
	
	/**
	 * This is the set of Attribute objects used to map XML tags to Attributes on Entities.
	 */ 
	HashMap<String,AttributeInfo>   setattributes = new HashMap<String,AttributeInfo>();   // Sets an attribute.

    
    private Mapping(IRSession session){
        this.session = session;
        this.state   = session.getState();
    }
    
    private void initialize(){
        for(String entity : entitystack){
            try {
                IREntity newEntity = ((RSession)session).createEntity(null,entity);
                session.getState().entitypush(newEntity);
            } catch (RulesException e) {
                throw new RuntimeException("Failed to initialize the Entity Stack: "+e);
            }
        }
    }
    
    /**
     * For internal use only.  If you need a Mapping object, get it from the appropriate
     * RuleSet object.
     * @param rd
     * @param session
     * @param filename
     * @return
     */
    public static Mapping newMapping(RulesDirectory rd, IRSession session, String filename){
        Mapping mapping = new Mapping(session);
    	try {
			InputStream s = session.getRulesDirectory().getFileSource().openStreamSearch(
			        FileType.MAP, session.getRuleSet(), filename);
            session.getState().traceTagBegin("loadMapping", "file",filename);
			mapping.loadMap(s);
			session.getState().traceTagEnd(); 
		} catch (FileNotFoundException e) {
 			throw new RuntimeException(e);
		} catch (Exception e) {
 			throw new RuntimeException("Error accessing '"+filename+"'\r\n"+e);
		}
		return mapping;
    }
    
    
    /**
     * Clones the mapping structures for use by a given session
     * @param session
     * @param amapping
     * @return
     */
    public Mapping clone(IRSession session){
    	Mapping mapping = new Mapping(session);
    	mapping.entities = entities;
    	mapping.multiple = multiple;
    	mapping.requests = requests;
    	mapping.entityinfo = entityinfo;
    	mapping.entitystack = entitystack;
    	mapping.setattributes = setattributes;
    	mapping.attribute2listPairs = attribute2listPairs;
    	mapping.dataObjects = dataObjects;
    	mapping.initialize();
    	return mapping;
    }
    
    
    /**
     * Constructor for creating a Mapping.  This has been depreciated because
     * building a map this way reloads the mapping file over and over.  Instead
     * you should use:
     * 
     *    Mapping map = session.getMapping();
     * 
     * @param rd
     * @param session
     * @param rs
     * @deprecated
     */
     public Mapping(RulesDirectory rd,IRSession session, RuleSet rs){
       this.session = session;
       this.state   = session.getState();
       String filename = rs.getMapPath().get(0);
       try {
           InputStream s = session.getRulesDirectory().getFileSource().openStreamSearch(
                   FileType.MAP, session.getRuleSet(), filename);
           session.getState().traceTagBegin("loadMapping", "file",filename);
           this.loadMap(s);
           session.getState().traceTagEnd(); 
       } catch (FileNotFoundException e) {
           throw new RuntimeException(e);
       } catch (Exception e) {
           throw new RuntimeException(e);
       }
    }
    
    /**
	 * Load parses XML to EDD Map file into a set of actions
	 * which can be used to build an EDD file.
	 * <br><br>
	 * 
	 * @param file
	 */
	 protected void loadMap(InputStream xmlStream) throws Exception {
		LoadMap map = new LoadMap(state,this);
		GenericXMLParser.load(xmlStream,map);
		if(!map.isLoadSuccessful()){
		    
			throw new Exception("Map failed to load due to errors!");
		}
	} 
	/**
	 * Load Data into the XML from a String.  
	 * 
	 * @param session - The session that the data needs to be loaded into
	 * @param Source - This is for error reporting.  What is the source of the data?
	 * @param data - The XML data as a String.
	 * 
	 * @throws RulesException
	 */
	public void loadStringData(IRSession session, String Source, String data)throws RulesException {
	    Reader strReader = new StringReader(data);
	    loadData(session,strReader, Source);
	}
	 
	public void loadData(IRSession session, String dataSource )throws RulesException {
	    loadData(session, dataSource, dataSource);
	}	      
	
	public void loadData(IRSession session, String dataSource, String source )throws RulesException {
        InputStream input = session.getRulesDirectory().getFileSource().openStreamSearch(
                FileType.MAP, session.getRuleSet(), dataSource);
        if(input == null){
            throw new RulesException("File Not Found","Mapping.loadData()","Could not open "+dataSource);
        }
        loadData(session,input,source);
    }
    
    /**
     * This is the state for dataloading... So that data can be split over multiple files, we
     * cache our LoadXMLData object between calls to loadData().
     */
    LoadXMLData dataloader = null;
    
    /**
     * Loads data using this mapping from an XML data Source into the given session.
     * @param session
     * @param dataSource
     * @throws RulesException
     */
    public void loadData (IRSession session, InputStream dataSource, String source) throws RulesException {
       Reader dataSrc = new InputStreamReader(dataSource);
       loadData(session, dataSrc, source);
    }
    public void loadData (IRSession session, Reader dataSource, String source) throws RulesException {
            if(dataloader ==null) {
            dataloader = new LoadXMLData(this,session,session.getRuleSet().getName());
        }
        if(source==null){
            source = "XML file";
        }
        try {
            if(session.getState().testState(DTState.TRACE)){
                session.getState().traceTagBegin("dataLoad","source",source);   
            }
            
            GenericXMLParser.load(dataSource, dataloader);

            if(session.getState().testState(DTState.TRACE)){
                session.getState().traceTagEnd();
            }
        } catch (Exception e) {
            if(session.getState().testState(DTState.TRACE)){
                session.getState().traceTagEnd();
            }
            throw new RulesException("Parse Error","LoadMap.loadData()",e.getMessage());
        }
    }
    
    /**
     * Load an in Memory data structure into the given session.
     * @param session
     * @param datasrc
     * @throws RulesException
     */
    public void loadData(IRSession session, DataMap datasrc)throws RulesException {
        if(dataloader ==null) {
            dataloader = new LoadDatamapData(this,session,session.getRuleSet().getName());
        }
        try {
            if(session.getState().testState(DTState.TRACE)){
                session.getState().traceTagBegin("loadData");
            }
            XMLNode tag = datasrc.getRootTag();
            processTag(tag);
            if(session.getState().testState(DTState.TRACE)){
                session.getState().traceTagEnd();
            }
        }catch(Exception e){
            if(session.getState().testState(DTState.TRACE)){
                session.getState().traceTagEnd();
            }
            throw new RulesException("Parse Error","LoadMap.loadData()",e.getMessage());
        }
        
    }
    String tagstk[]  = new String[1000];
    int    tagstkptr = 0;
    /**
     * Load a tag into a session.
     */
    void processTag(XMLNode tag) throws Exception {
        if(tag.type()== XMLNode.Type.TAG){
            state.traceTagBegin(tag.getTag()==null?"process":tag.getTag(),null);
            tagstk[tagstkptr++] = tag.getTag();
            
            dataloader.beginTag(tagstk, tagstkptr, tag.getTag(), tag.getAttribs());
            if(tag.getTags()!= null) for(XMLNode nextTag : tag.getTags()){
                processTag(nextTag);
            }
            ((LoadDatamapData)dataloader).endTag(tagstk, tagstkptr, tag, tag.getBody(), tag.getAttribs());
            
            tag.clearRef();
 
            tagstkptr--;
            tagstk[tagstkptr]= null;
            state.traceTagEnd();
        }
    }


    /**
     * @return the state
     */
    public DTState getState() {
        return state;
    }


    /**
     * @return the session
     */
    public IRSession getSession() {
        return session;
    }
    
    
}
