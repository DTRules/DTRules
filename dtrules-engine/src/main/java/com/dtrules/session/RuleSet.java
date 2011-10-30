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

package com.dtrules.session;

import java.io.InputStream;
import java.util.ArrayList;
import java.util.Collection;
import java.util.HashMap;
import java.util.Iterator;
import java.util.List;
import java.util.Map;

import com.dtrules.automapping.AutoDataMap;
import com.dtrules.automapping.AutoDataMapDef;
import com.dtrules.entity.IREntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RName;
import com.dtrules.mapping.Mapping;
import com.dtrules.session.IStreamSource.FileType;

/**
 * Defines the set of artifacts which make up a logical set of
 * rules.  These include the schema for the rules (The Entity
 * Description Dictionary or EDD) as well as the entityfactory 
 * created from the EDD.  A Rule Set also includes the decision
 * tables, and a set of mappings to map XML data into the 
 * Schema defined by the EDD.
 * <br><br>
 * This implementation is really just a place holder.  We need
 * to explore how to define sets of Rules, their schemas, their
 * decision tables and the entry points into these tables, the
 * connections between rules and databases, and rules and 
 * perhaps several UI implementations.  This is a much more 
 * complex problem than it first appears.
 * 
 * @author paul snow
 * Jan 17, 2007
 *
 */
public class RuleSet {
                                             // We need a reference to the IRObject here to 
    public int t = IRObject.iInteger;        //   force the class loader to build our types.
    
	protected Class<ICompiler>            defaultCompiler  = null;
	protected RName        	              name;
	protected boolean   	              resource         = false;
	protected ArrayList<String>           edd_names        = new ArrayList<String>();
	protected ArrayList<String>           dt_names         = new ArrayList<String>();
    protected ArrayList<String>           map_paths        = new ArrayList<String>();
    protected ArrayList<String>           includedRuleSets = new ArrayList<String>();
    protected String                      excel_edd        = null;
    protected String                      excel_dtfolder   = null;
    
    @Deprecated
    protected HashMap<String,Mapping>  mappings            = new HashMap<String,Mapping>();

    protected EntityFactory               ef 			   = null;
	protected RulesDirectory              rd;
	protected String                      firstDecisionTableName;
	
    protected String                      resourcepath     = null;
    protected String                      filepath         = null;
    protected String                      workingdirectory = null;
    
    // Support for tracking various attributes useful in various systems and
    //   configurations
    Map<String, Object>                   attributes       = new HashMap<String,Object>();
    
    // Support for the new AutoMap data interface
    protected Map<String,String>          mapFiles         = new HashMap<String,String>();
    protected Map<String,AutoDataMapDef>  mapDefinitions   = new HashMap<String,AutoDataMapDef>();
    protected Map<String, String>         entrypoints      = new HashMap<String,String>();
    protected Map<String, List<String>>   contexts         = new HashMap<String,List<String>>();
    
    /**
     * Set an attribute on this Rule Set
     * @param key
     * @param value
     */
    public void setAttribute(String key, Object value){
        attributes.put(key, value);
    }
    /**
     * Return an attribute that has been previously set on this Rule Set.
     * Returns null if no attribute of that name has been previously set.
     * @param key
     * @return
     */
    public Object getAttribute(String key) {
        return attributes.get(key);
    }
    
    public AutoDataMap getAutoDataMap(IRSession session, String name) throws RulesException{
        if(!mapDefinitions.containsKey(name)){
            String mapfilename = mapFiles.get(name);
            InputStream mapStream = null;
            
            if(mapfilename != null){
                mapStream = rd.streamSource.openStreamSearch(FileType.AUTO_MAP, this, mapfilename);
            }
            if(mapStream==null){
                throw new RulesException("undefined", "getAutoDataMapDef()", 
                        "The mapping '"+name+"' is undefined");
            }
            
            AutoDataMapDef admd = new AutoDataMapDef();
            admd.configure(mapStream);
            mapDefinitions.put(name, admd);
        }
        AutoDataMap autoDataMap = mapDefinitions.get(name).newAutoDataMap(session);
        autoDataMap.setSession(session);
        return autoDataMap;
    }
    
    /**
     * Return the default compiler for this Rule Set.
     * @return Class<ICompiler>
     * @throws RulesException
     */
    public Class<ICompiler> getDefaultCompiler() throws RulesException {
    	if(defaultCompiler == null){
    		defaultCompiler = rd.getDefaultCompiler();
    	}
		return defaultCompiler;
	}

    /**
     * When we deploy the Rules Engine, we don't have to have the compiler.  So If we can't
     * find the compiler, we just ignore the issue.  We wait until some code actually tries
     * to *get* the compiler before we throw an error.
     * 
     * @param qualifiedCompilerClassName
     */
	@SuppressWarnings("unchecked")
	public void setDefaultCompiler(String qualifiedCompilerClassName) {
		try{
		   this.defaultCompiler = (Class<ICompiler>) Class.forName(qualifiedCompilerClassName);
		}catch(ClassNotFoundException e){}
	}
	
	/**
	 * Set the default compiler to the given compiler.
	 * @param compiler
	 */
	public void setDefaultCompiler(Class<ICompiler> compiler){
		this.defaultCompiler = compiler;
	}

	/**
     * Get the default mapping (the first mapping file)
     * @param session
     * @return
     */
    public Mapping getMapping(IRSession session){
        String filename = map_paths.get(0);
        return getMapping(filename,session);
    }
    
    /**
     * get Mapping.
     * We create an instance that has a reference to this
     * session, but is otherwise identical to the reference
     * mapping.
     */
    public synchronized Mapping getMapping(String filename,IRSession session){
        Mapping map = mappings.get(filename);
        if(map != null)return map.clone(session);
        if(map_paths.indexOf(filename)<0){
            throw new RuntimeException("Bad Mapping File: "+filename+" For the rule set: "+name.stringValue());
        }
        map = Mapping.newMapping(rd, session, filename);
        mappings.put(filename, map);
        return map.clone(session);
    }
    
    /**
     * @return the excel_dtfolder
     */
    public String getExcel_dtfolder() {
        return excel_dtfolder;
    }

    /**
     * @param excel_dtfolder the excel_dtfolder to set
     */
    public void setExcel_dtfolder(String excel_dtfolder) {
        if(excel_dtfolder.startsWith("/") || excel_dtfolder.startsWith("\\")){
            excel_dtfolder = excel_dtfolder.substring(1);
        }
        this.excel_dtfolder = excel_dtfolder;
    }

    /**
     * @return the excel_edd
     */
    public String getExcel_edd() {
        return excel_edd;
    }

    /**
     * @param excel_edd the excel_edd to set
     */
    public void setExcel_edd(String excel_edd) {
        if(excel_edd.startsWith("/") || excel_edd.startsWith("\\")){
            excel_edd = excel_edd.substring(1);
        }
        this.excel_edd = excel_edd;
    }

    /**
     * Appends the directory specified in the RulesDirectory (if present) to the
     * filepath for the Rule Set.  This is the directory where all the XML for 
     * the Rule Set is kept.  It is where the XML produced from the Excel files
     * is written. 
     * @return the filepath
     */
    public String getFilepath() {
        return rd.getFilepath()+"/"+filepath;
    }
 
    /**
     * @param filepath the filepath to set
     */
    public void setFilepath(String filepath) {
        filepath = filepath.trim();
        if( !filepath.endsWith("/") && !filepath.endsWith("\\")){
            filepath = filepath + "/";
        }
        //Remove any leading slash.
        if(filepath.startsWith("/")||filepath.startsWith("\\")){
            filepath = filepath.substring(1);
        }
        this.filepath = filepath;
    }

    /**
     * @return the resourcepath
     */
    public String getResourcepath() {
        return resourcepath;
    }

    /**
     * @param resourcepath the resourcepath to set
     */
    public void setResourcepath(String resourcepath) {
        this.resourcepath = resourcepath;
    }

    /**
     * @return the workingdirectory
     */
    public String getWorkingdirectory() {
        return rd.getFilepath()+"/"+workingdirectory;
    }

    /**
     * @param workingdirectory the workingdirectory to set
     */
    public void setWorkingdirectory(String workingdirectory) {
        workingdirectory = workingdirectory.trim();
        if( !workingdirectory.endsWith("/") && !workingdirectory.endsWith("\\")){
            workingdirectory = workingdirectory + "/";
        }
        //Remove any leading slash.
        if(workingdirectory.startsWith("/")||workingdirectory.startsWith("\\")){
            workingdirectory = workingdirectory.substring(1);
        }

        this.workingdirectory = workingdirectory;
    }

    /**
     * Accessor for getting the Rules Directory used to create this Rule Set
     * @return
     */
    public RulesDirectory getRulesDirectory(){
        return rd;
    }
    
	RuleSet(RulesDirectory _rd){
		rd = _rd;
	}
	/**
     * Returns an interator for the paths used to define the
     * decision tables. These paths may point to XML files on
     * the file system, or to resources within a jar.
     * @return
	 */
    public Iterator<String> DTPathIterator(){
        return dt_names.iterator();
    }
    
    /**
     * Returns an iterator for the paths used to define the 
     * EDD for this rule set.  These paths may point to XML files
     * on the file system, or to resources within a jar.
     * @return
     */
    public Iterator<String> eDDPathIterator(){
        return edd_names.iterator();
    }
    
    public void clearCache() throws RulesException {
    	getEntityFactory(true, newSession());
    }
    
    /**
     * Creates a new Session set up to execute rules within this
     * Rule Set.  Note that a RuleSet is stateless, so a Session
     * can point to a RuleSet, but a RuleSet can belong to many
     * sessions.
     * @return
     * @throws RulesException
     */
    synchronized public IRSession newSession () throws RulesException{
        IRSession s = new RSession(this);
        getEntityFactory(s);
        return s;
    }    
    
    /**
     * Get the EntityFactory associated with this ruleset. 
     * An EntityFactory is stateless, so many sessions can use
     * the reference to a single EntityFactory. 
     * @return
     * @throws RulesException
     */
    protected EntityFactory getEntityFactory(IRSession session) throws RulesException{
    	return getEntityFactory(ef==null, session);
    }
    
	protected EntityFactory getEntityFactory(boolean load, IRSession session) throws RulesException{
			
		if(load){
			synchronized (this) {
				ef                     = new EntityFactory(this);
				Iterator<String> iedds = edd_names.iterator();
				session.setEntityFactory(ef);
				while(iedds.hasNext()){
					String filename = iedds.next();
					InputStream s= rd.streamSource.openStreamSearch(FileType.EDD, this, filename);
					if(s==null){
						if(s==null){
							System.out.println("No EDD XML found.  " +
	                    		"\r\n   Looking for:      "+filename+
	                    		"\r\n   WorkingDirectory: "+session.getRuleSet().getWorkingdirectory()+
	                            "\r\n   ResourcePath:     "+session.getRuleSet().getResourcepath()+
	                    		"\r\n   SystemDirecotry:  "+session.getRuleSet().getSystemPath());
						}                 
	                }     
	                if(s!=null) ef.loadedd(session, filename,s);
			   }
			   Iterator<String> idts = dt_names.iterator();
			   while(idts.hasNext()){	   
	               String filename = idts.next();
	          	   InputStream s = rd.streamSource.openStreamSearch(FileType.DECISIONTABLES, this, filename);
				   if(s==null){
	                      System.out.println("No Decision Table XML found" +
	                              "\r\n   Looking for:      "+filename+
	                              "\r\n   WorkingDirectory: "+session.getRuleSet().getWorkingdirectory()+
	                              "\r\n   ResourcePath:     "+session.getRuleSet().getResourcepath()+
	                              "\r\n   SystemDirecotry:  "+session.getRuleSet().getSystemPath());
					}
					if(s!=null) ef.loaddt(session, s);
			   }
			}
		 }
		 return ef;
	 }

	/**
     * Returns the path to load the mapping file.  Ultimately, we
     * should support multiple mappings into the same EDD.
	 * @return the map_path
	 */
	public ArrayList<String> getMapPath() {
		return map_paths;
	}
    
	/**
     * Sets the path to the mapping file.
	 * @param map_path the map_path to set
	 */
	public void setMapPath(ArrayList<String> map_path) {
		this.map_paths = map_path;
	}
	/**
	 * @return the name
	 */
	public String getName() {
		return name.stringValue();
	}	
    
    /**
     * Get the name as a RName
     */
    public RName getRName(){
        return name;
    }
    
	/**
	 * @param name the name to set
	 */
	public void setName(String name) {
		this.name = RName.getRName(name);
	}
	/**
	 * @return the dt_paths
	 */
	public ArrayList<String> getDt_paths() {
		return dt_names;
	}
	/**
	 * @param dt_paths the dt_paths to set
	 */
	public void setDt_paths(ArrayList<String> dt_paths) {
		this.dt_names = dt_paths;
	}

    /**
     * Returns the single DecisionTable XML.  If there are more, or none, an
     * error is thrown.
     * @return filename 
     * @throws RulesException
     */
	public String getDT_XMLName() throws RulesException{
         if(dt_names.size()!=1){
             throw new RulesException("UnsupportedConfiguration","RuleSet","We can only have one DecisionTable XML file");
         }
         return dt_names.get(0);
    }
    /**
     * Returns the single EDD XML.  If there are more, or none, an error is
     * thrown.
     * @return filename
     * @throws RulesException
     */
    public String getEDD_XMLName() throws RulesException{
        if(edd_names.size()!=1){
            throw new RulesException("UnsupportedConfiguration","RuleSet","We can only have one EDD XML file");
        }
        return edd_names.get(0);
   }
	/**
	 * @return the edd_paths
	 */
	public ArrayList<String> getEdd_paths() {
		return edd_names;
	}
	/**
	 * @param edd_paths the edd_paths to set
	 */
	public void setEdd_paths(ArrayList<String> edd_paths) {
		this.edd_names = edd_paths;
	}
	/**
	 * @return the firstDecisionTableName
	 */
	public String getFirstDecisionTableName() {
		return firstDecisionTableName;
	}
	/**
	 * @param firstDecisionTableName the firstDecisionTableName to set
	 */
	public void setFirstDecisionTableName(String firstDecisionTableName) {
		this.firstDecisionTableName = firstDecisionTableName;
	}
    
    public String getSystemPath () {
        return rd.getSystemPath();
    }

    /**
     * @return the includedRuleSets
     */
    public ArrayList<String> getIncludedRuleSets() {
        return includedRuleSets;
    }

    /**
     * @param includedRuleSets the includedRuleSets to set
     */
    public void setIncludedRuleSets(ArrayList<String> includedRuleSets) {
        this.includedRuleSets = includedRuleSets;
    }
    
    /**
     * An accessor for the decision tables. Using newSession() because... seemed like a good idea
     * @return the decision tables
     * @throws RulesException 
     */
    public List<IRObject> getDecisionTables() throws RulesException {
    	List<IRObject> tables = getEntityFactory(newSession()).getDecisiontables().getValues();
    	return tables;
    }
    
    /**
     * An accessor for the reference entities. Like the above function
     * @return the reference entities
     * @throws RulesException 
     */
    public Collection<IREntity> getRefEntities() throws RulesException {
    	return getEntityFactory(newSession()).getRefEntities();
    }
}