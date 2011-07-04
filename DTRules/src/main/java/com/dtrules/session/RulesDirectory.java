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

package com.dtrules.session;

import java.io.FileInputStream;
import java.io.FileNotFoundException;
import java.io.IOException;
import java.io.InputStream;
import java.net.MalformedURLException;
import java.net.URL;
import java.net.URLConnection;
import java.util.HashMap;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.RName;
import com.dtrules.xmlparser.GenericXMLParser;
import com.dtrules.xmlparser.IGenericXMLParser;

@SuppressWarnings({"unchecked"})
public class RulesDirectory {
    
	boolean                loaded;
	HashMap<RName,RuleSet> rulesets;
    String                 systemPath=""; // Path to the directory on the File System
                                        // where files can be read and written.  If
                                        // present, it is appended to paths used by
                                        // rule sets.
    
    public void addRuleSet(RuleSet ruleset) throws RulesException {
        
    }
    
    /**
     * Returns the ruleset associated with the given name.  The conversion
     * of the String to an RName is done by this routine.  
     * All hash tables in DTRules should use RNames as keys.
     *
     * Returns true if the RulesDirectory has been successfully loaded.
     * @return
     */
    public boolean isLoaded(){return loaded;}
    
    public RuleSet getRuleSet(String setname){
        return getRuleSet(RName.getRName(setname));
    }
    
    /**
     * Returns the ruleset associated with the given name.  Note that the
     * key is an RName.  All hash tables in DTRules should use RNames as keys.
     * 
     * @param setname
     * @return
     */
    public RuleSet getRuleSet(RName setname){
    	return (RuleSet) rulesets.get(setname);
    }	
    
    
    /**
     * We attempt to open the streamname as a resource in our jar.
     * Then failing that, we attempt to open it as a URL.  
     * Then failing that, we attempt to open it as a file.
     * 
     * @param streamname
     * @return
     */
    public static InputStream openstream(Object object, String streamname){
    	// First try and open the stream as a resource 
        //	InputStream s = System.class.getResourceAsStream(streamname);
    	
    	InputStream s = object.getClass().getResourceAsStream(streamname);
    	
    	if(s!=null)return s;    
    	
    	// If that fails, try and open it as a URL
    	try {
			URL url = new URL(streamname);
			URLConnection urlc = url.openConnection();
			s = urlc.getInputStream();
			if(s!=null)return s;
		} catch (MalformedURLException e) {
        } catch (Exception e){} 
		
		// If that fails, try and open it as a file.
		try {
			s = new FileInputStream(streamname);
			return s;
		} catch (FileNotFoundException e) {}
		
		// If all these fail, return a null.
    	return null;
    	
    }
    
 
    String propertyfile;
    
    /**
     * The RulesDirectory manages the various RuleSets and the versions of 
     * RuleSets.  We need to do a good bit of work to make all of this 
     * managable. For right now, I am loading the property list from the 
     * path provided this class.  It first attempts to use this path as a
     * jar resource, then an URL, then a file.
     * 
     * The systemPath is assumed to be the name of a directory, either with
     * or without a ending '/' or '\'.
     * 
     * @param propertyfile
     */
    public RulesDirectory(String systemPath, String propertyfile) {
        if(systemPath.endsWith("/")||systemPath.endsWith("\\")){
            // If it has an ending slash, chop it off.
            systemPath = systemPath.substring(0,systemPath.length()-1);
        }
        if(propertyfile.startsWith("/")||propertyfile.startsWith("\\")){
            // If the propertyfile has a leading slash, chop that off.
            propertyfile = propertyfile.substring(1);
        }
        this.propertyfile = propertyfile;  
        this.systemPath   = systemPath.trim();
        String f = systemPath + "/" + propertyfile;
        InputStream s = openstream(this,f);
    	loadRulesDirectory(s);
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
    		throw new RuntimeException("Could not find the file/inputstream :"+propertyfile);
    	}
    	try {
			GenericXMLParser.load(s,parser);
		} catch (Exception e) {
			throw new RuntimeException("Error parsing property file/inputstream: "+propertyfile+"\n"+e);
		}
    	loaded = true;
    }
    
    static class LoadDirectory implements IGenericXMLParser {
	
    	final RulesDirectory rd;
    	LoadDirectory(RulesDirectory _rd){
    		rd=_rd;
    		rd.rulesets = new HashMap<RName,RuleSet>();
    	}
    	RuleSet currentset=null;
    	
		public void beginTag(String[] tagstk, int tagstkptr, String tag, HashMap attribs) throws IOException, Exception {
			if (tag.equals("RuleSet")){
				currentset = new RuleSet(rd);
				currentset.setName((String) attribs.get("name"));
				if(currentset.name==null){
					throw new RuntimeException("Missing name in RuleSet");
				}
				rd.rulesets.put(currentset.name, currentset);
			}	
		}
		
		public void endTag(String[] tagstk, int tagstkptr, String tag, String body, HashMap attribs) throws Exception, IOException {
			if(tag.equals("RuleSetResourcePath")){
				currentset.setResourcepath(body.trim());
			}else if (tag.equals("RuleSetFilePath")){
				currentset.setFilepath(body.trim());
            }else if (tag.equals("WorkingDirectory")){
                currentset.setWorkingdirectory(body.trim());
            }else if (tag.equals("Entities") ||
                      tag.equals("EDD") ){
				currentset.edd_names.add((String) attribs.get("name"));
			}else if (tag.equals("Decisiontables")){
				currentset.dt_names.add((String) attribs.get("name"));
			}else if (tag.equals("Map")){
				currentset.map_paths.add((String) attribs.get("name"));
			}else if (tag.equals("DTExcelFolder")){
			    currentset.setExcel_dtfolder(body.trim());
            }else if (tag.equals("EDDExcelFile")||
                      tag.equals("EDDExcelFolder")){
                currentset.setExcel_edd(body.trim());
            }
		}
		public boolean error(String v) throws Exception {
			return true;
		}
    	
    }
	
	/**
	 * @return the rulesets
	 */
	public HashMap getRulesets() {
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
    
}
