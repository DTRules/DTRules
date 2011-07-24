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
import java.util.Arrays;
import java.util.Collection;
import java.util.HashMap;
import java.util.Iterator;

import com.dtrules.decisiontables.DTLoader;
import com.dtrules.decisiontables.RDecisionTable;
import com.dtrules.entity.IREntity;
import com.dtrules.entity.REntity;
import com.dtrules.entity.REntityEntry;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RName;
import com.dtrules.xmlparser.GenericXMLParser;
import com.dtrules.xmlparser.XMLPrinter;

public class EntityFactory {
	
	String   create_stamp;
    int      uniqueID       = 10;  // Leave room for some fixed IDs. Primitives have an ID = 1, decisiontables have an ID=2;
    boolean  frozen         = false;
    RuleSet  ruleset;
    HashMap<Object,IREntity> javaObjectEntityMap = new HashMap<Object,IREntity>();
    HashMap<IREntity,Object> entityJavaObjectMap = new HashMap<IREntity,Object>();
    HashMap<RName,IREntity>  referenceEntities   = new HashMap<RName,IREntity>();  
    
    ArrayList<RName>         decisiontablelist   = new ArrayList<RName>();
    
    IREntity decisiontables = new REntity(2,true,RName.getRName("decisiontables"));
    
    IComputeDefaultValue     computeDefaultValue;
    /**
     * Provides a HashMap that provides a mapping of Java Objects (as a key)
     * to Rules Entity objects.
     * @return
     */
    public HashMap<Object,IREntity> getJavaObjectEntityMap(){
        return javaObjectEntityMap;
    }

    /**
     * Provides a HashMap that provides a mapping of Rules Entities (as a key)
     * to Java Objects.
     * @return
     */
    public HashMap<IREntity,Object> getEntityJavaObjectMap(){
        return entityJavaObjectMap;
    }
    
    /**
     * Map the given Java Object and Entity to each other
     */
    public void map(Object obj, IREntity entity){
        javaObjectEntityMap.put(obj, entity);
        entityJavaObjectMap.put(entity,obj);
    }
	
    /**
     * Creates and returns a new Decision Table object.
     * @param name -- The name of the new decision table.
     * @return -- The new decision table
     */
    public RDecisionTable newDecisionTable(String name, IRSession session)throws RulesException{
        RName    rname = RName.getRName(name);
        return newDecisionTable(rname, session);
    }
    
    /**
     * Creates and returns a new Decision Table object.
     * @param name -- The name of the new decision table.
     * @return -- The new decision table
     */
    public RDecisionTable newDecisionTable(RName name, IRSession session)throws RulesException{
        IRObject table = decisiontables.get(name);
        if(table !=null && table.type().getId() != IRObject.iDecisiontable){
            throw new RulesException("ParsingError","New Decision Table","For some reason, "+name.stringValue()+" isn't a decision table");
        }
        if(table != null){
            session.getState().debug("Overwritting the Decision Table: "+name.stringValue());
        }
        RDecisionTable dtTable = new RDecisionTable(session,name.stringValue());
        decisiontablelist.add(name);
        decisiontables.addAttribute(name, "", dtTable, false, true, RDecisionTable.dttype ,null,"","","");
        decisiontables.put(null, name, dtTable);
        return dtTable;
    }
    /**
     * Delete Decision Table doesn't really delete the decision table, but removes it
     * from the structures in the entityfactory.
     */
    public void deleteDecisionTable(RName name)throws RulesException {
        IRObject table = getDecisionTable(name);
        if(table==null)return;
        decisiontables.removeAttribute(name);
        decisiontablelist.remove(name);
    }
    /**
     * Returns an Iterator which provides each Entity name.
     * @return
     */
	public Iterator<RName> getEntityRNameIterator (){
		return referenceEntities.keySet().iterator();
	}
	
	/**
	 * Returns the referenceEntities
	 */
	public Collection<IREntity> getRefEntities() {
		return referenceEntities.values();
	}
	
	/**
     * Provides an Iterator that provides each decision table name. 
     * @return
	 */
	public Iterator<RName> getDecisionTableRNameIterator () {
		return decisiontablelist.iterator();
	}
	
    /**
     * Look up a Decision Table in the decision tables held in this Rule Set.
     * @param name
     * @return
     * @throws RulesException
     */
    public RDecisionTable getDecisionTable(RName name)throws RulesException{
        IRObject dt = decisiontables.get(name);
        if(dt==null || dt.type().getId()!=IRObject.iDecisiontable){
            return null;
        }
        return (RDecisionTable) dt;
    }
    
    public EntityFactory(RuleSet rs) {
        ruleset = rs;
        ((REntity)decisiontables).removeAttribute(RName.getRName("decisiontables"));
    }
    /**
     * Return the RDecisionTable for the given name
     * @param tablename
     * @return
     */
    public RDecisionTable findTable(String tablename){
        return findTable(RName.getRName(tablename));
    }
    
    /**
     * Return the RDecisionTable for the given name
     * @param tablename
     * @return
     */
	public RDecisionTable findTable(RName tablename){
		return (RDecisionTable) decisiontables.get(tablename);
	}
	
    /**
     * Looks up the reference Entity given by name.  Returns
     * a null if it isn't defined.
     * @param name
     * @return
     */
    public IREntity findRefEntity(String name){
        return findRefEntity(RName.getRName(name));
    }
	/**
	 * Looks up the reference Entity given by name.  Returns
	 * a null if it isn't defined.
	 * @param name
	 * @return
	 */
	public IREntity findRefEntity(RName name){
		return (REntity) referenceEntities.get(name);
	}
	/**
	 * Looks up the reference Entity given by name.  Creates a new
	 * reference entity if it doesn't yet exist.  Otherwise it returns
	 * the existing reference entity.
	 * 
	 * @param name
	 */
	public IREntity findcreateRefEntity(boolean readonly, RName name)throws RulesException {
		if(!referenceEntities.containsKey(name)){
			IREntity entity = new REntity(getUniqueID(), readonly, name); 
			referenceEntities.put(name,entity);
		}
		return (REntity) referenceEntities.get(name);
	}
	
    public void loadedd(IRSession session, String filename, InputStream edd) throws RulesException {
    	EDDLoader loader = new EDDLoader(filename, session, this);
        try {
    	   GenericXMLParser.load(edd,loader);
    	   if(loader.succeeded==false){
    	       throw new RulesException("Parsing Error(s)","EDD Loader",loader.errorMsgs);
    	   }
        } catch(Exception e){
            throw new RulesException("Parsing Error","EDD Loader",e.toString());
        }
    }
    
    /**
     * Looks up the given Decision Table in the list of tables and returns 
     * it.
     * @param dt
     * @return
     */
    public RDecisionTable findDecisionTable(RName dt){
        IRObject dttable = decisiontables.get(dt);
        if(dttable==null || dttable.type().getId()!=IRObject.iDecisiontable){
            return null;
        }
        return (RDecisionTable) dttable;
    }
    /**
     * Loads a set of decision tables from the given input stream.
     * @param dt Source of Decision Tables in XML.
     * @throws RulesException
     */
    public void loaddt(IRSession session, InputStream dt) throws RulesException {
    	DTLoader loader = new DTLoader(session, this);
    	try {
    		GenericXMLParser.load(dt, loader);
    	}catch (Exception e) {
            throw new RulesException("Parsing Error","Decision Table Loader",e.toString());
		}
    }
    
    @Override
    public String toString() {
		Iterator<RName> ikeys = referenceEntities.keySet().iterator();
		StringBuffer buff = new StringBuffer();
		while(ikeys.hasNext()){
			IREntity e = referenceEntities.get(ikeys.next());
			buff.append(e.getName().toString());
			buff.append("\r\n");
			Iterator<RName> iattribs = e.getAttributeIterator();
			while(iattribs.hasNext()){
				RName        entryname  = (RName)iattribs.next();
				REntityEntry entry      = e.getEntry(entryname);
				buff.append("   ");
				buff.append(entryname);
				if(entry.defaultvalue!=null){
					buff.append("  --  default value: ");
					buff.append(entry.defaultvalue.toString());
				}
				buff.append("\r\n");
			}
		}
		return buff.toString();
	}
    
    
    
    public void writeAttributes(XMLPrinter xout) throws RulesException {
       RName entities[]= new RName[0];
       entities = referenceEntities.keySet().toArray(entities); 
       Arrays.sort(entities);
       for ( RName key : entities ){
           IREntity entity = referenceEntities.get(key);
           {
               String access  = entity.isReadOnly()?"r":"rw";
               String comment = entity.getComment();
               String name    = key.stringValue();
               xout.opentag("entity","name",name,"access",access,"comment",comment);
           }
           RName attributes [] = new RName[0];
           attributes = entity.getAttributeSet().toArray(attributes);
           Arrays.sort(attributes);
           for (RName attribute : attributes){
               REntityEntry entry = entity.getEntry(attribute);
               
               String name          = attribute.stringValue();
               String type          = entry.getType().toString();
               String subtype       = entry.getSubtype();
               String access        = (entry.readable ? "r":"") + (entry.writable ? "w":"");
               String input         = entry.getInput();
               String default_value = entry.getDefaulttxt();
               String comment       = entry.getComment();
               
               xout.opentag("field",
                       "name",name,
                       "type",type,
                       "subtype",subtype,
                       "access",access,
                       "input",input,
                       "default_value",default_value,
                       "comment",comment);
               xout.closetag();
           }
           xout.closetag();
       }
    }
    
    /**
     * EntityFactories create things that need IDs.  The EntityFactory
     * has to be created prior to any sessions that are going to
     * execute against the entityfactory.
     * @return uniqueID (a long)
     */
    public int getUniqueID() throws RulesException{ 
        if(frozen)throw new RulesException("No UniqueID","EntityFactory.getUniqueID()","Once a session has been created, you can't modify the EntityFactory");
        return uniqueID++;
        
    }

    /**
     * @return the decisiontables
     */
    public IREntity getDecisiontables() {
        return decisiontables;
    }

    /**
     * If you need a sorted list of entities or a sorted list of attributes,
     * you can use this quick and dirty bubble sort to convert one of the 
     * Interators we provide into a list of sorted RNames.
     * 
     * @param list
     * @return
     */
    public static ArrayList<RName> sorted(Iterator<RName> list, boolean ascending){
        ArrayList<RName> slist = new ArrayList<RName>();
        while(list.hasNext()){
            slist.add(list.next());
        }
                
        for(int i = 0; i < slist.size()-1; i++){
            for(int j=0; j <slist.size()-1-i; j++){
                if (slist.get(j).compareTo(slist.get(j+1))<0 ^ ascending){
                    RName hld = slist.get(j);
                    slist.set(j, slist.get(j+1));
                    slist.set(j+1, hld);
                }
            }
        }
        return slist;
    }

    
}
