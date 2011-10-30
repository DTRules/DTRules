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

import java.io.IOException;
import java.io.InputStream;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.RName;
import com.dtrules.xmlparser.AGenericXMLParser;
import com.dtrules.xmlparser.GenericXMLParser;

@SuppressWarnings({"unchecked"})
public class RulesDirectory {
    
    private String                 name;        // Name of this Rules Directory.
    private Map<String,Object>     attributes = new HashMap<String,Object>();
    
	boolean                loaded;
	HashMap<RName,RuleSet> rulesets;
    String                 systemPath=""; // Path to the directory on the File System
                                        // where files can be read and written.  If
                                        // present, it is appended to paths used by
                                        // rule sets.
    Class<ICompiler>       defaultCompiler;
    
    									// We use the streamSource object to read files.  
    									// We have a default implementation, but will take
    									// an implementation upon creating a RulesDirectory.
    IStreamSource		   streamSource = new StreamSource();
    
    
    
    /**
     * Return the name of this Rules Directory.  Useful in tools to 
     * identify the grouping of a set of Rule Sets
     * @return
     */
    public String getName() {
        return name;
    }

    /**
     * Set the name of this Rules Directory.
     * @param name
     */
    public void setName(String name) {
        this.name = name;
    }

    /**
     * Return an object associated with this Rules Directory.  This is
     * useful for marking or managing meta information about the Rules
     * Directory.
     * @param key
     * @return
     */
    public Object getAttribute(String key){
        return attributes.get(key);
    }
    
    /**
     * Put a value into the attributes of this Rules Directory.  Useful
     * for managing meta informatin about the Rules Directory.
     * @param key
     * @param value
     */
    public void setAttribute(String key, Object value){
        attributes.put(key, value);
    }
    
    public Class<ICompiler> getDefaultCompiler() throws RulesException {
    	if(defaultCompiler == null){
    		try{
    			defaultCompiler = (Class<ICompiler>)Class.forName("com.dtrules.compiler.el.EL");
    		}catch(ClassNotFoundException e){
    			return null;			// No default compiler found.
    		}
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
	public void setDefaultCompiler(String qualifiedCompilerClassName) {
		try{
		   this.defaultCompiler = (Class<ICompiler>) Class.forName(qualifiedCompilerClassName);
		}catch(ClassNotFoundException e){
		    System.err.println("WARNING:  Cannot find the specfied compiler: "+qualifiedCompilerClassName);
		}
	}
    
	/**
     * Returns true if the RulesDirectory has been successfully loaded.
     * @return boolean
     */
    public boolean isLoaded(){return loaded;}
    
    /**
     * Returns the ruleset associated with the given name.  The conversion
     * of the String to an RName is done by this routine.  
     * All hash tables in DTRules should use RNames as keys.
     * 
     * @param setname the name of the Rule Set
     * @return RuleSet
     */
    public RuleSet getRuleSet(String setname){
        return getRuleSet(RName.getRName(setname));
    }
    
    /**
     * Returns the rule set associated with the given name.  Note that the
     * key is an RName.  All hash tables in DTRules should use RNames as keys.
     * 
     * @param setname
     * @return
     */
    public RuleSet getRuleSet(RName setname){
    	return (RuleSet) rulesets.get(setname);
    }	
    
    
 
    
 
    String propertyfile;
    
    /**
     * The RulesDirectory manages the various RuleSets and the versions of 
     * RuleSets.  We need to do a good bit of work to make all of this 
     * manageable. For right now, I am loading the property list from the 
     * path provided this class.  It first attempts to use this path as a
     * jar resource, then an URL, then a file.
     * 
     * The systemPath is assumed to be the name of a directory, either with
     * or without a ending '/' or '\'.
     * 
     * @param systemPath A Path to the property file, and a point in the file
     *                   system from which all paths in the property file are
     *                   relative to.
     * @param propertyfile The name of the property file.
     * 
     * @param opt True if the decision tables should be optimized, and false
     *            otherwise (needed for trace for accurate path analysis)
     */
    public RulesDirectory(
    		IStreamSource 	filesource, 
    		String 			systemPath, 
    		String 			propertyfile, 
    		boolean 		opt) {
    	
    	this.streamSource = filesource;
    	
    	load(systemPath,propertyfile);
    }
    
    /**
     * The RulesDirectory manages the various RuleSets and the versions of 
     * RuleSets.  We need to do a good bit of work to make all of this 
     * manageable. For right now, I am loading the property list from the 
     * path provided this class.  It first attempts to use this path as a
     * jar resource, then an URL, then a file.
     * 
     * The systemPath is assumed to be the name of a directory, either with
     * or without a ending '/' or '\'.
     * 
     * @param systemPath A Path to the property file, and a point in the file
     *                   system from which all paths in the property file are
     *                   relative to.
     * @param propertyfile The name of the property file.
     * 
     * @param opt True if the decision tables should be optimized, and false
     *            otherwise (needed for trace for accurate path analysis)
     */
    public void load(String systemPath, String propertyfile){	
    	if(systemPath.endsWith("/")||systemPath.endsWith("\\")){
            // If it has an ending slash, chop it off.
            systemPath = systemPath.substring(0,systemPath.length()-1);
        }
        if(propertyfile.startsWith("/")||propertyfile.startsWith("\\")){
            // If the property file has a leading slash, chop that off.
            propertyfile = propertyfile.substring(1);
        }
        this.propertyfile = propertyfile;  
        this.systemPath   = systemPath.trim();
        String f = systemPath + "/" + propertyfile;
        InputStream s = streamSource.openstream(IStreamSource.FileType.DTRULES_CONFIG,f);
    	loadRulesDirectory(s);
    }

    /**
     * The RulesDirectory manages the various RuleSets and the versions of 
     * RuleSets.  We need to do a good bit of work to make all of this 
     * manageable. For right now, I am loading the property list from the 
     * path provided this class.  It first attempts to use this path as a
     * jar resource, then an URL, then a file.
     * 
     * The systemPath is assumed to be the name of a directory, either with
     * or without a ending '/' or '\'.
     * 
     * @param systemPath A Path to the property file, and a point in the file
     *                   system from which all paths in the property file are
     *                   relative to.
     * @param propertyfile The name of the property file.
     * 
     */
    public RulesDirectory(String systemPath, String propertyfile){
    	load(systemPath, propertyfile);
    }

    
    public RulesDirectory(String systemPath, InputStream s) {
        if(systemPath.endsWith("/")||systemPath.endsWith("\\")){
            systemPath = systemPath+"/";
        }
        this.systemPath     = systemPath.trim();
        propertyfile = s.toString();
        
       loadRulesDirectory(s);
    }
    
    public void loadRulesDirectory(InputStream s){
    	LoadDirectory parser = new LoadDirectory(this);
    	
    	if(s==null){  
    		throw new RuntimeException("Could not find the file/inputstream: "+systemPath+"/"+propertyfile);
    	}
    	try {
			GenericXMLParser.load(s,parser);
		} catch (Exception e) {
			throw new RuntimeException("Error parsing property file/inputstream: "+systemPath+"/"+propertyfile+"\n"+e);
		}
    	loaded = true;
    }
    
    static class LoadDirectory extends AGenericXMLParser {
	
    	final RulesDirectory rd;
    	LoadDirectory(RulesDirectory _rd){
    		rd=_rd;
    		rd.rulesets = new HashMap<RName,RuleSet>();
    	}
    	
    	RuleSet    currentset  = null;
    	String     entryPoint  = null;
    	
    	HashMap<String,String> compileralias = new HashMap <String,String>();
    	
    	public void beginTag(
    	        String[] tagstk, 
    	        int tagstkptr, 
    	        String tag, 
    	        HashMap<String,String> attribs) throws IOException, Exception {
    	    
		    if (tag.equals("RuleSet")){
			    currentset = new RuleSet(rd);
				currentset.setName((String) attribs.get("name"));
				if(currentset.name==null){
				    throw new RuntimeException("Missing name in RuleSet");
				}
				rd.rulesets.put(currentset.name, currentset);
			}else if (tag.equals("entryPoint")){
                entryPoint = (String) attribs.get("name");
                if(entryPoint == null || entryPoint.trim().length()==0){
                    throw new RuntimeException("Entrypoints must have a name");
                }
                entryPoint = entryPoint.trim();
			}
		}
		
		public void endTag(String[] tagstk, 
		        int tagstkptr, 
		        String tag, 
		        String body, 
		        HashMap<String,String> attribs) throws Exception, IOException {
	
			// Parse Rule Set tags.... If not inside a Rule Set, then we are going to skip processing these tags
			if(currentset==null) {
			    if (tag.equals("DTRules")){
			        // Nothing to do here; Just ignore this tag.
			    }else if (tag.equals("name")){
			        rd.setName(body);
			    }else if (tag.equals("compileralias")){
                    String key = attribs.get("name");
                    compileralias.put(key,body);
                }else if(tag.equals("compiler")){
				    if(compileralias.containsKey(body)){
				        body = compileralias.get(body);
				    }
					rd.setDefaultCompiler(body.trim());
				}else if (tag.equals("value")){
                    String name  = attribs.get("name");
                    String value = attribs.get("value");
                    rd.setAttribute(name, value);
                }else{
                    throw new RulesException("undefined", "RulesDirectory","Unexpected tag encountered: "+tag);
                }
			}else{
				if (tag.equals("include")){
				    currentset.getIncludedRuleSets().add(body);
				}else if (tag.equals("compiler")){
			        if(compileralias.containsKey(body)){
			            body = compileralias.get(body);
			        }
					currentset.setDefaultCompiler(body.trim());
				}else if (tag.equals("RuleSetResourcePath")){
					currentset.setResourcepath(body.trim());
				}else if (tag.equals("RuleSetFilePath")){
					currentset.setFilepath(body.trim());
	            }else if (tag.equals("WorkingDirectory")){
	                currentset.setWorkingdirectory(body.trim());
	            }else if (tag.equals("Entities") ||
	                      tag.equals("EDD") ){
					currentset.edd_names.add(attribs.get("name"));
				}else if (tag.equals("Decisiontables")){
					currentset.dt_names.add(attribs.get("name"));
				}else if (tag.equals("Map")){
					currentset.map_paths.add(attribs.get("name"));
				}else if (tag.equals("DTExcelFolder")){
				    currentset.setExcel_dtfolder(body.trim());
	            }else if (tag.equals("EDDExcelFile")||
	                      tag.equals("EDDExcelFolder")){
	                currentset.setExcel_edd(body.trim());
	            }else if (tag.equals("RuleSet")){
	            	currentset = null;
	            }else if (tag.equals("map")){
	                String name      = attribs.get("name");
	                String filename  = attribs.get("filename");
	                currentset.mapFiles.put(name, filename);
	            }else if (tag.equals("entryPoint")){
	                entryPoint = null;
	            }else if (tag.equals("decisionTable")){
	                String table = body.trim();
	                if(table==null){
	                    throw new RulesException("undefined", "LoadDirectory", 
	                            "The Decisiontable '"+body+"' is not defined in the" +
	                            " ruleset '"+currentset.getName());
	                }
	                currentset.entrypoints.put(entryPoint, table);
	            }else if (tag.equals("entity")){
	                String entity = body.trim();
	                if(entity == null){
	                    throw new RulesException("undefined", "LoadDirectory", 
                                "The Entity '"+body+"' is not defined in the" +
                                " ruleset '"+currentset.getName());
	                }
	                List<String> entities = currentset.contexts.get(entryPoint);
	                if(entities == null){
	                    entities = new ArrayList<String>();
	                    currentset.contexts.put(entryPoint, entities);
	                }
	                entities.add(body.trim());
	            }else if (tag.equals("value")){
	                String name  = attribs.get("name");
	                String value = attribs.get("value");
	                currentset.setAttribute(name, value);
	            }else{
	                throw new RulesException("undefined", "RulesDirectory","Unexpected tag encountered: "+tag);
	            }
			}
		}    	
    }
	
	/**
	 * @return the rulesets
	 */
	public HashMap<RName,RuleSet>  getRulesets() {
		return rulesets;
	}
    /**
     * @return the filepath
     */
    public String getFilepath() {
        return systemPath;
    }
    /**
     * @param filepath the filepath to set
     */
    public void setFilepath(String filepath) {
        this.systemPath = filepath;
    }
    /**
     * @return the systemPath
     */
    public String getSystemPath() {
        return systemPath;
    }
    /**
     * @param systemPath the systemPath to set
     */
    public void setSystemPath(String systemPath) {
        this.systemPath = systemPath;
    }

    public IStreamSource getFileSource() {
        return streamSource;
    }

    public void setFileSource(IStreamSource streamSource) {
        this.streamSource = streamSource;
    }
    
}
