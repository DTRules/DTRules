/*  
 * Copyright 2004-2008 MTBJ, Inc.  
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
 */
package com.dtrules.compiler.el;

import java.util.ArrayList;

import com.dtrules.entity.IREntity;
import com.dtrules.interpreter.RName;
import com.dtrules.session.IRType;

public class ELType implements IRType {
	 RName     name;
     int       type;
     ArrayList<IREntity> entities = new ArrayList<IREntity>();
     ArrayList<String>   refs     = new ArrayList<String>();
     
    public RName      getRName(){return name;}
    public int        getType() {return type;} 
    public ArrayList<IREntity> getEntities(){return entities;}
       
    public static final String REF          = "ref";
    public static final String NOT_REF      = "not ref";
    public static final String POSSIBLE_REF = "possible ref";
    
    public String toString() {
    	try{
	    	String s = name 
            +"\n              type:     "+com.dtrules.interpreter.RType.getType(type).toString()
            +"\n              Entities: ";
	    	for(int i=0;i<entities.size();i++){
	    		s+=entities.get(i)+" ";
	    	}
	    	return s;
    	}catch(Exception e){
    		return name +" failed to convert to a string.";
    	}
    }
    public void addEntityAttribute(IREntity _entity){
        if(_entity != null ){
            if(!entities.contains(_entity)){
                entities.add(_entity);
                refs.add(NOT_REF);
            }    
        }else{
            throw new RuntimeException("What the heck?");
        }
    }
    /**
     * Mark an attribute as referenced.  If the entity isn't specified,
     * we can't tell.
     * @param entity
     * @return
     */
    public void addRef(String entity){
         
        if(entity.length()==0)entity = null; // Make my tests easier.
        /**
         * If we have a reference without the entity specifier,
         * then this is a possible reference across all entities
         * with that type.
         */
        for(int i=0;i < refs.size(); i++){
            // If no entity is specified, then this is a *possible* ref 
            if(entity==null  && refs.get(i).equals(NOT_REF)){
                refs.set(i, POSSIBLE_REF);
            }
            // If the entity is specified, this is a definite ref
            if(entity !=null && entities.get(i).getName().stringValue().equals(entity)){
                refs.set(i, REF);
            }
            // However, if there is only one entity defining the attribute,
            // that is also a definite reference (Don't care if I set it twice)
            if(refs.size()==1){
                refs.set(i,REF);
            }
        }
        
    }
    /**
     * Return a list of unreferenced Attributes
     * @return
     */
    public ArrayList<String> getUnReferenced(){
        
        ArrayList<String> unreferenced = new ArrayList<String>();
        
        //Ignore the mapping "key" attribute.
        if(name.equals(IREntity.mappingKey))return unreferenced;
        
        for(int i=0; i< refs.size(); i++){
            if(refs.get(i).equals(NOT_REF)){
                RName entityName = entities.get(i).getName();
                // Ignore self references.
                if(!entityName.equals(name)){
                  unreferenced.add(entityName.stringValue()+"."+name.stringValue());
                }  
            }
        }
        return unreferenced;
    }
    
    /**
     * Possible Referenced attributes
     * @return
     */
    public ArrayList<String> getPossibleReferenced(){
        ArrayList<String> attribs = new ArrayList<String>();
        
//      Ignore the mapping "key" attribute.
        if(name.equals(IREntity.mappingKey))return attribs;
        
        for(int i=0; i< refs.size(); i++){
            if(refs.get(i).equals(POSSIBLE_REF)){
                RName entityName = entities.get(i).getName();
                // Ignore self references.
                if(!entityName.equals(name)){
                    attribs.add(entityName.stringValue()+"."+name.stringValue());
                }  
            }
        }
        return attribs;
    }

    
    public ELType(RName _name,int _type, IREntity _entity){
        name = _name;
        type = _type;
        addEntityAttribute(_entity);
    }
}
