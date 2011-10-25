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
/**
 * Loads the mapping file (the description of how data should be moved
 * from XML to the EDD)
 * 
 */
import java.io.IOException;
import java.text.DateFormat;
import java.text.ParseException;
import java.text.SimpleDateFormat;
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
import com.dtrules.interpreter.RNull;
import com.dtrules.interpreter.RString;
import com.dtrules.interpreter.RDate;
import com.dtrules.session.DTState;
import com.dtrules.session.IRSession;
import com.dtrules.session.RSession;
import com.dtrules.xmlparser.IGenericXMLParser;

@SuppressWarnings({"unchecked"})
public class LoadMapping implements IGenericXMLParser {
    Mapping       map;
    int           codeCnt = 0;
    IRSession     session;
    IRObject      def;
    DTState       state;
    String        ruleSetName;
    DateFormat df_in   = new SimpleDateFormat ("yyyy-MM-dd");
	
	DateFormat df_out  = new SimpleDateFormat ("MM/dd/yyyy");
	
    
    public LoadMapping(Mapping _map){
        map = _map;
    }
    
    public LoadMapping(Mapping _map, IRSession session, String _ruleSetName){
        map = _map;
        this.session = session;
        this.state   = session.getState();
        this.ruleSetName = _ruleSetName;
                
    	try{	// Cache the def operator.  
    	   def = session.getState().find(RName.getRName("def"));
    	}catch(Exception e){
            state.traceInfo("error","General Rules Engine Failure");
            throw new RuntimeException(e);
    	}
    	
    	Iterator<RName> es = this.map.entities.keySet().iterator();
    	while(es.hasNext()){
    		RName  ename = (RName) es.next();         		
    	   try {
			  IREntity e     = findEntity(ename.stringValue().toLowerCase(),null,null);
			  state.entitypush(e);
		   } catch (RulesException e) {
              state.traceInfo("error", "Failed to initialize the Entity Stack (Failed on "+ename+")\n"+e);
    		  throw new RuntimeException(e);  
		   }
    	}
    }
    
    /**
	 * Looks to make sure that we have not yet created an Entity 
	 * of this name with the given code.  If we have, we return the
	 * earlier created Entity.  Otherwise, we create a new instance.
	 * @param entity -- We assume this is a valid Entity name (no dot syntax)
	 * @param code
	 * @return
	 * @throws RulesException
	 */
	IREntity findEntity( String entity, String code, EntityInfo info) throws RulesException{
	   String number = (String) this.map.entityinfo.get(entity);
	   IREntity e;
	   if(number==null){
		   number = "*";
	   }
	   if(number.equals("1")){
		  e = (IREntity)entities.get(entity);
		  if(e==null){
		      e = session.getState().findEntity(RName.getRName(entity+"."+entity));
		      if(e==null){
                 e = ((RSession)session).createEntity(null,entity);
		      }   
              entities.put(entity,e);
          }
	   }else { // We assume number.equals("*") || number.equals("+")
		  e = null;
          String key = "";
		  if(code!=null && code.length()!=0) {
              key = entity+"$"+code;
			  e = (IREntity)entities.get(key);
		  }	  
		  if(e==null) {
              e = ((RSession)session).createEntity(null,entity);
          }
		  if(code!=null) entities.put(key,e);
	   }
       if(e==null)throw new RulesException("undefined","LoadMapping.findEntity()","Failed to create the entity "+entity);
       UpdateReferences(e,info);
       return e;
	}	
	
    public void beginTag(String[] tagstk, int tagstkptr, String tag,
            HashMap attribs) throws IOException, Exception {
        String name = tag;                                 // We assume the tag might create an entity
		@SuppressWarnings("unused")
        boolean       traceopen = false;
		EntityInfo    info  = (EntityInfo)    this.map.requests.get(name);
		AttributeInfo aInfo = (AttributeInfo) this.map.setattributes.get(tag);

			
		//		 If I get info, then create an entity.
		//		 Get the code from this tag.
		//		 If a fixed entity name is specified,
		//		 or the tag name, use it.  Otherwise use the multiple name
		if(info!=null){										
			Object objCode = attribs.get(info.id);
            String code = objCode==null?"":objCode.toString();
			String eName = info.name;
			if (eName == null || eName.length() <= 0)
			{
				eName =  (String) attribs.get("name");
			}
			IREntity e = findEntity(eName, code, info);     // Look up the entity I should create.
		    if(e!=null){									// I hope to goodness I can find it!
		      attribs.put("create entity","true");	
		      if(code.length()!=0) {
		    	  e.put(null, IREntity.mappingKey,RString.newRString(code));
		      }else{
		    	  e.put(null, IREntity.mappingKey,RString.newRString("v"+ (++codeCnt)));
		      }
		    
		      state.entitypush(e);
			  if(state.testState(DTState.TRACE)){
		          state.traceTagBegin("mapCreateEntity", "name",info.name,"data_id",code);
		          traceopen = true;
			  }    
		    }else{
                
		      state.traceInfo("error","The Mapping defines '"+info.entity+"', but this entity isn't defined in the EDD");
		      throw new Exception("The Mapping defines '"+info.entity+"', but this entity isn't defined in the EDD");
		    }
		} 

		if(aInfo!=null){ // If we are supposed to set an attribute, then we set ourselves up
                         // to define the entity attribute on the end tag.  We may be setting this
		                 // Attribute to the value of the Entity we just created/looked up. 
			/** 
			 * First check enclosures, then check the blank case. This allows
			 * the user to specify a default mapping, yet still direct some attributes to
			 * specific destinations based on the enclosure.
			 */
			AttributeInfo.Attrib attrib;
			{                                                   // Not only do you have to match the attribute name,
			    int i=tagstkptr-2;				                //   But you must match the immediately enclosing tag
				attrib = aInfo.lookup(tagstk[i]);    
				if(attrib!=null){
			       queueSetAttribute(attrib, attribs);
				}
			}	                       
			attrib = aInfo.lookup("");                           // If I don't find the enclosure defined, look to see
		    queueSetAttribute(attrib,attribs);                   //   if a general default is defined.
		}
    }

    public void endTag(String[] tagstk, int tagstkptr, String tag, String body,
            HashMap attribs) throws Exception, IOException 
    {
    	String  attr;
		REntity entity = null;
    
		if(attribs.containsKey("create entity")){		  					// For create Entity Tags, we pop the Entity from the Entity Stack on the End Tag.
			entity = (REntity) state.entitypop();
			Iterator pairs  = map.attribute2listPairs.iterator();
			body = "";                                                      // Don't care about the Body if we created an Entity.
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
		if ((attr = (String) attribs.get("set attribute date"))!=null){      	// Look and see if we have an attribute name defined.
			if (body.trim().length() > 0)
			{
				Date       date;
				try {
                    date = df_in.parse(body);
            	} catch (ParseException e) {
                    try{
                       date = df_out.parse(body);
                    }catch (ParseException e2){
                       date = df_out.parse("01/05/2008");
                       //throw new RuntimeException("Bad Date encountered: ("+tag+")="+body);
                    }   
				}
				body = df_out.format(date);
				attribs.put("set attribute",attr);
			}
		}			
		
		if ((attr = (String) attribs.get("set attribute"))!=null){      	
			// Look and see if we have an attribute name defined.
				{	
					RName a = RName.getRName(attr);								 
					IRObject value;
                     
                    IREntity enclosingEntity = session.getState().findEntity(a);
                    if(enclosingEntity!=null){
                        
                    
    					int type = enclosingEntity.getEntry(a).type.getId();
    					
    					if(type == IRObject.iInteger){
    						value = RInteger.getRIntegerValue(body.length()==0? "0" : body);
    					} else if (type == IRObject.iDouble) {
    						value = RDouble.getRDoubleValue(body.length()==0? "0" : body); 
    					} else if (type == IRObject.iBoolean){
                            value = RBoolean.getRBoolean(body.length()==0? "false" : body);
                        } else if (type == IRObject.iDate){
                            if(body.trim().length()>0){
                              value = RDate.getRDate(session,body);
                              if(value == null){
                                throw new RulesException("MappingError","LoadMapping","Bad Date... Could not parse '"+body+"'");
                              }
                            }else{
                              value = RNull.getRNull();
                            }
                        } else if (type == IRObject.iEntity){
                            if(entity!=null){
                               value = entity;
                            }else{
                                throw new RulesException("MappingError","LoadMapping","Entity Tags have to create some Entity Reference");
                            }    
                        }else {
    						value = RString.newRString(body);
    					}
    					//   conversion in the Rules Engine to do the proper thing.
    					state.def(a,value,false);
                    }
				}				
		}
		
	    return;	
	}

    public boolean error(String v) throws Exception {
        return true;
    }
    
    /** 
	 * Does nothing if the info is null... Just means we are not mapping this attribute.
	 * Otherwise it updates the attribs hashmap.
	 * 
	 * @param info
	 */
	private void queueSetAttribute( AttributeInfo.Attrib attrib, HashMap attribs){
		if(attrib==null)return;
		switch (attrib.type ){
		    case AttributeInfo.ARRAY_CODE    :    // We just ignore arrays.
                break;
            case AttributeInfo.DATE_STRING_CODE :
				attribs.put("set attribute date",attrib.rAttribute); 
				break;
			case AttributeInfo.STRING_CODE   :  
			case AttributeInfo.NONE_CODE     :
			case AttributeInfo.INTEGER_CODE  :
            case AttributeInfo.BOOLEAN_CODE  :
            case AttributeInfo.FLOAT_CODE    :
            case AttributeInfo.ENTITY_CODE   :
            case AttributeInfo.XMLVALUE_CODE :
                attribs.put("set attribute",attrib.rAttribute);
                break;
			default:
			    String type = "(Unknown Code: "+attrib.type+")";
				throw new RuntimeException("Bad Type Code "+type+
						" in com.dtrules.mapping.AttributeInfo: "+attrib.rAttribute);
		}
	}
	
	/**
	 * We collect all the Entities we create as we go.  These
	 * are stored as a value, and their id as the key.  Then if
	 * we encounter the same key in the XML, we return the same
	 * Entity.
	 */
	HashMap entities = new HashMap();
	
	IREntity UpdateReferences(IREntity e, EntityInfo info)throws RulesException {
		RName listname;
		if(info!=null && info.list.length()==0){
		    listname   = RName.getRName(e.getName().stringValue()+"s");
		}else{
			listname   = RName.getRName(info.list);
		}
        // First add this entity to any list found on the entity stack.
        for(int i=0; i< state.edepth(); i++){
            // Look for all Array Lists on the Entity Stack that look like lists of this Entity
            IREntity entity = state.getes(i);
            IRObject elist = entity.get(listname);
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
        // Then update the reference to this entity that might be on the Entity Stack.
        // DON'T go wild.  Only look at the top entity (or you may overwrite a reference
        //     you'd rather leave alone.
        // DON'T mess with any entity's self reference though!  That is BAD.
        int i=state.edepth()-1;
    	if(((IREntity)state.getes(i)).get(e.getName())!=null){
            IREntity refto = state.getes(i);
            if(! refto.getName().equals(e.getName()))           // Update a reference to an Entity of the same name,
    		   ((IREntity)state.getes(i)).put(null, e.getName(), e);  //  but only if it isn't a self reference.
    		
    	}
     
	    return e;
	}

	
	}
