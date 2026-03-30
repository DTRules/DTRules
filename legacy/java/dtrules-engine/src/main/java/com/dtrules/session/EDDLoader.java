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
import java.util.HashMap;

import com.dtrules.entity.IREntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RName;
import com.dtrules.interpreter.RNull;
import com.dtrules.interpreter.RType;
import com.dtrules.xmlparser.IGenericXMLParser;

public class EDDLoader implements IGenericXMLParser {

	final IRSession        session;
	final EntityFactory    ef;
	final String           filename;
    boolean                succeeded =  true;
	String                 errorMsgs =  "";
	int                    version   =  1;
	
	EDDLoader(String _filename, IRSession session, EntityFactory ef){
		this.session = session;
		this.ef = ef;
        filename = _filename;
	}
	
	/**
	 * If this string has a non-zero length, the EDD did not load
	 * properly.  The caller is responsible for checking this.  Otherwise
	 * the loader can only report a single error.
	 * 
	 * @return
	 */
	
	
	public String getErrorMsgs() {
        return errorMsgs;
    }




    public void beginTag(String[] tagstk, int tagstkptr, String tag,
		HashMap<String,String> attribs) throws IOException, Exception {
        
        if(tag.equals("entity_data_dictionary") ){
            
            try{
                version = Integer.parseInt((String) attribs.get("version"));
            }catch(NullPointerException e){}   // Ignore any errors
            catch(Exception e){} 
        
        }else if(version == 2){
            beginTag2(tagstk,tagstkptr,tag,attribs);
        }    
	}

	public void endTag(String[] tagstk, 
			           int      tagstkptr, 
			           String   tag, 
			           String   body, 
			           HashMap<String,String>  attribs) throws Exception, IOException {
	                 // 
	    
	    if(version==2){
	        
	        endTag2(tagstk,tagstkptr,tag,body,attribs);
	    
	    }else if(tag.equals("entity")){
		    
		  String entityname = (String) attribs.get("entityname");
		  String attribute  = (String) attribs.get("attribute");
		  String type       = (String) attribs.get("type");
		  String subtype    = (String) attribs.get("subtype");
		  String access     = (String) attribs.get("access");
		  String defaultv   = (String) attribs.get("default");
		  String comment    = (String) attribs.get("comment");
		  String input      = (String) attribs.get("input");
		  String output     = (String) attribs.get("output");
		  
		  if(comment == null)comment = "";
		  if(input   == null)input   = "";
		  if(output  == null)output  = "";
		  
		  boolean  writeable = true; 	// We need to convert access to a boolean
		  boolean  readable  = true;    // Make an assumption of r/w
		  RType    rtype     = null;    // We need to convert the type to an int.
		  IRObject defaultO  = null;    // We need to convert the default into a Rules Engine Object.
		  
		  writeable = access.toLowerCase().indexOf("w")>=0;
		  readable  = access.toLowerCase().indexOf("r")>=0;
		  if(!writeable && !readable){
		      errorMsgs +="\nThe attribute "+attribute+" has to be either readable or writable\r\n";
		      succeeded=false;
		      rtype = RNull.type;
		  }
		  
		  // Now the type.  An easy thing.
          if(!RType.isType(type)){
        	  errorMsgs+= "The type specified: '"+type+"' is not a valid type.";
  			  succeeded = false;
          }else{
        	  rtype = RType.getType(type);
		  } 
		  
          try{		  
              defaultO = session.getComputeDefault().computeDefaultValue(session, ef, defaultv, rtype) ;
          } catch (RulesException e) { 
              errorMsgs += "Bad Default Value '"+defaultv+"' encountered on entity: '"+entityname+"' attribute: '"+attribute+"' \n";
              succeeded = false;
          }
		  RName  entityRName = RName.getRName(entityname.trim(),false);
		  RName  attributeRName = RName.getRName(attribute.trim(),false);
		  IREntity entity = ef.findcreateRefEntity(false,entityRName);
          RType   rtype2 = null;
          if(!RType.isType(type)){
        	  errorMsgs += "Bad Type: '"+type+"' encountered on entity: '"+entityname+"' attribute: '"+attribute+"' \n";
        	  succeeded = false;
        	  rtype2 = RNull.type;
          }else{
        	  rtype2 = RType.getType(type);
		  }  
			
		  String errstr  = entity.addAttribute(attributeRName,
		                                       defaultv, 
		                                       defaultO,
		                                       writeable,
		                                       readable,
		                                       rtype2,
		                                       subtype,
		                                       comment,
		                                       input,
		                                       output);
		  if(errstr!=null){
		      succeeded = false;
		      errorMsgs += errstr;
		  }
        }
	}

	public boolean error(String v) throws Exception {
		return true;
	}

	
	/** Support for the New EDD format **/
	
	String entityname;
	String entitycomment;
	String entityaccess;
    public void beginTag2(String[] tagstk, int tagstkptr, String tag,
            HashMap<String,String> attribs) throws IOException, Exception {
        if(tag.equals("entity")){
            entityname      = (String) attribs.get("name");
            entitycomment   = (String) attribs.get("comment");
            entityaccess    = (String) attribs.get("access");
        }
    }

    public void endTag2(String[] tagstk, 
                           int      tagstkptr, 
                           String   tag, 
                           String   body, 
                           HashMap<String,String>  attribs) throws Exception, IOException {
        if(!tag.equals("field")) return;
             
        String default_value  = (String) attribs.get("default_value");
        String attrib_name    = (String) attribs.get("name");
        String access         = (String) attribs.get("access");
        String subtype        = (String) attribs.get("subtype");
	    String type           = (String) attribs.get("type");
	    String comment        = (String) attribs.get("comment");
        String input          = (String) attribs.get("input");
        String output         = (String) attribs.get("output");
        
        if(comment == null)comment = "";
        if(input   == null)input   = "";
        if(output  == null)output  = "";
    	    
	    boolean writeable = access.toLowerCase().indexOf("w")>=0;
        boolean readable  = access.toLowerCase().indexOf("r")>=0;
        if(!writeable && !readable){
            errorMsgs +="\nThe attribute "+attrib_name+" has to be either readable or writable\r\n";
            succeeded=false;
        }
        
        RType rtype = null;

        // Now the type.  An easy thing.
        if(!RType.isType(type)){
        	errorMsgs+= "The type: '"+type+"' is not a valid type";
            succeeded = false;
        }else{
        	rtype = RType.getType(type);
        }
                
        IRObject defaultO = session.getComputeDefault().computeDefaultValue(session, ef, default_value, rtype) ;
        
        RName  entityRName = RName.getRName(entityname.trim(),false);
        RName  attributeRName = RName.getRName(attrib_name.trim(),false);
        IREntity entity = ef.findcreateRefEntity(false,entityRName);
        
        RType rtype2 = null;

        if(!RType.isType(type)){
        	errorMsgs += "Bad Type: '"+type+"' encountered on entity: '"+entityname+"' attribute: '"+attrib_name+"' \n";
            succeeded = false;
        }else{
        	rtype2 = RType.getType(type);
        }

        String errstr  = entity.addAttribute(attributeRName,
                                             default_value, 
                                             defaultO,
                                             writeable,
                                             readable,
                                             rtype2,
                                             subtype,
                                             comment,
                                             input,
                                             output);
        if(errstr!=null){
            succeeded = false;
            errorMsgs += errstr;
    }
    }
	
	
	
	
	
	
	
	
	
	
}
