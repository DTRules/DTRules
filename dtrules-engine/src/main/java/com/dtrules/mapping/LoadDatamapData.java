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
import java.math.BigDecimal;
import java.text.ParseException;
import java.util.Date;
import java.util.HashMap;
import java.util.Iterator;

import com.dtrules.entity.IREntity;
import com.dtrules.entity.REntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RArray;
import com.dtrules.interpreter.RBoolean;
import com.dtrules.interpreter.RDouble;
import com.dtrules.interpreter.RInteger;
import com.dtrules.interpreter.RName;
import com.dtrules.interpreter.RString;
import com.dtrules.interpreter.RDate;
import com.dtrules.interpreter.RXmlValue;
import com.dtrules.mapping.XMLNode;
import com.dtrules.session.DTState;
import com.dtrules.session.IRSession;
import com.dtrules.session.RSession;

@SuppressWarnings({"unchecked"})
public class LoadDatamapData extends LoadXMLData {

    XMLNode xmltag = null;

    
    public LoadDatamapData(Mapping _map){
        super(_map);
    }
    
    public LoadDatamapData(Mapping _map, IRSession session, String _ruleSetName){
        super(_map, session, _ruleSetName);
    }
    
    public void endTag(String[] tagstk, int tagstkptr, XMLNode tag, Object body,
            HashMap attribs) throws Exception, IOException {
       xmltag = tag;
       if(attribs.containsKey("create entity")){
           state.entityfetch(0).setRXmlValue(new RXmlValue(state,xmltag));
       }
       endTag(tagstk,tagstkptr,tag.getTag(),body,attribs);
       xmltag = null;
    }

    public void endTag(String[] tagstk, int tagstkptr, String tag, Object body,
            HashMap attribs) throws Exception, IOException 
    {
    	String  attr;
		REntity entity = null;
    
		if(attribs.containsKey("create entity")){		  					// For create Entity Tags, we pop the Entity 
																			//   from the Entity Stack on the End Tag.
	        attribs.remove("create entity");
		    entity = (REntity) state.entitypop();
			Iterator pairs  = map.attribute2listPairs.iterator();
			body = "";                                                      // Don't care about the Body if creating an 
																			//    Entity, but it can't be null either.
			while(pairs.hasNext()){
				Object [] pair =  (Object []) pairs.next();
			    if(entity.containsAttribute(RName.getRName((String)pair[0]))){
			    	RName.getRName((String) pair[1],true).execute(state);
			    	state.entitypush(entity);
			    	RName.getRName("addto",true).execute(state);
			    	
			    }
			}
			if(state.testState(DTState.TRACE)){
                state.traceTagEnd();
            }  
		}    		

	    //  If this is a Date format, we are going to reformat it, and let it feed into
        //  the regular set attribute code.
        if ((attr = (String) attribs.get("set attribute date"))!=null){     // Look and see if we have an attribute 
        																	//   name defined.
            attribs.remove("set attribute date");
            if(body instanceof String && body != null ){
                String sbody = body.toString();
                if (sbody.trim().length() > 0)
                {
                    Date       date = cvd(body);
                    if(date == null){
                        throw new RuntimeException("Bad Date encountered: ("+tag+")="+body);
                    }   
                    body = date;
                    attribs.put("set attribute",attr);
                }
            }
        }else {           
            attr = (String) attribs.get("set attribute");
        }    
		if (attr != null){      	
		    attribs.remove("set attribute");
			// Look and see if we have an attribute name defined.
            if(body!=null){
					RName a = RName.getRName(attr);								 
					IRObject value;
                     
                    IREntity enclosingEntity = session.getState().findEntity(a);
                    if(enclosingEntity==null){
                        throw new Exception ("No Entity is in the context that defines "+ a.stringValue());
                    }
					int type = enclosingEntity.getEntry(a).type.getId();
					
					if(type == IRObject.iInteger){
						value = RInteger.getRIntegerValue(getLong(body));
					} else if (type == IRObject.iDouble) {
						value = RDouble.getRDoubleValue(getDouble(body)); 
					} else if (type == IRObject.iBoolean){
                        value = RBoolean.getRBoolean(body.toString());
                    } else if (type == IRObject.iDate){
                        Date date = cvd(body);
                        if(date != null ){
                            value = RDate.getRTime( cvd(body) );
                        }else{
                            throw new RuntimeException("Bad Date encountered: ("+tag+")="+body);
                        }
                    } else if (type == IRObject.iEntity) {
                        if(entity!=null){
                            value = entity;
                        }else{
                            throw new RulesException("MappingError",
                            		"LoadDatamapData","Entity Tags have to create some Entity Reference");
                        }
                    } else if (type == IRObject.iString) {    
						value = RString.newRString(body.toString());
                    } else if (type == IRObject.iXmlValue){	
                        if(xmltag != null){
                            value = new RXmlValue(state,xmltag);
                        }else{
                            throw new RulesException("MappingError",
                            		"LoadDatamapData","Somehow we are missing the XML Tag for the attribute: "+a);
                        }
                    } else {
					    throw new RulesException("MappingError","LoadDatamapData","Unsupported type encountered: "+
					    		enclosingEntity.getEntry(a).type);
					}
					//   conversion in the Rules Engine to do the proper thing.
					state.def(a,value,false);
                    state.traceInfo("message","   /"+a+" \""+body+"\" def");
				}				
		}
		
	    return;	
	}
    
    private Date cvd(Object o){
        if(o instanceof Date )return (Date) o;
        String     sbody = o.toString();
        Date       date;
        try {
                date = df_in.parse(sbody);
        } catch (ParseException e) {
            try{
               date = df_out.parse(sbody);
            }catch (ParseException e2){
               return null;
            }   
        }
        return date;
    }    
    
    
    long getLong(Object num){
        if(num.getClass()==BigDecimal.class)return ((BigDecimal)num).longValue();
        if(num.getClass()==Long.class)return ((Long)num).longValue();
        if(num.getClass()==Double.class)return ((Double)num).longValue();
        if(num.getClass()==Integer.class)return((Integer)num).longValue();
        if(num.getClass()==String.class)return Long.parseLong(num.toString());
        throw new RuntimeException("Can't figure out the value "+num.toString()+" "+num.getClass().getName());
    }
    
    double getDouble(Object num){
        if(num.getClass()==BigDecimal.class)return ((BigDecimal)num).doubleValue();
        if(num.getClass()==Long.class)return ((Long)num).doubleValue();
        if(num.getClass()==Double.class)return ((Double)num).doubleValue();
        if(num.getClass()==Integer.class)return((Integer)num).doubleValue();
        if(num.getClass()==String.class)return Double.parseDouble(num.toString());
        throw new RuntimeException("Can't figure out the value "+num.toString()+" "+num.getClass().getName());
    }

    
    @Override
    public boolean error(String v) throws Exception {
        return true;
    }
    
   
	/**
	 * We collect all the Entities we create as we go.  These
	 * are stored as a value, and their id as the key.  Then if
	 * we encounter the same key in the XML, we return the same
	 * Entity.
	 */
	HashMap entities = new HashMap();
	
	IREntity UpdateReferences(IREntity e)throws RulesException {
		RName listname   = RName.getRName(e.getName().stringValue()+"s");
        
        // First add this entity to any list found on the entity stack.
        for(int i=0; i< state.edepth(); i++){
            // Look for all Array Lists on the Entity Stack that look like lists of this Entity
            IRObject elist = state.getes(i).get(listname);
            if(elist!=null && elist.type().getId()==IRObject.iArray){
                // If not a member of this list, then add it.
                if(!((RArray)elist).contains(e)){
                   ((RArray)elist).add(e);
                   if (state.testState(DTState.TRACE)) {
                       state.traceInfo("addto", "arrayId", ((RArray)elist).getID() + "", e.postFix());
                   }

                }
            }
        }
        // Then update any reference to this entity that might be on the Entity Stack.
        // DON'T mess with any entity's self reference though!  That is BAD.
        for(int i=0;i< state.edepth(); i++){
        	if((state.getes(i)).get(e.getName())!=null){
                IREntity refto = state.getes(i);
                
                if(! refto.getName().equals(e.getName()))           // Update a reference to an Entity of the same name,
        		   (state.getes(i)).put(null,e.getName(), e);  //  but only if it isn't a self reference.
        		
        	}
        }
     
	    return e;
	}

	
	}
