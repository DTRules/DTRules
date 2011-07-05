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

import java.lang.reflect.InvocationTargetException;
import java.lang.reflect.Method;
import java.util.HashMap;

import com.dtrules.infrastructure.RulesException;
import com.dtrules.mapping.AttributeInfo.Attrib;

/**
 * Defines the DO accesser to Entity Object Attribute Mapping.  Each
 * instance of a DataObjectMap maps a DO to an Entity.  If you need
 * to be able to map the DO to several Entities, then you will need
 * several DataObjectMaps, one for each Entity.
 * @author Paul Snow
 * May 8, 2008
 *
 */
@SuppressWarnings({"unchecked"})
class DataObjectMap {
    String                tag;                   // Entity Tag to look for
    boolean               loaded = false;        // Have I initialized this object yet?
    String                dataObjName;           // Name for this DO
    @SuppressWarnings("rawtypes")
	Class                 dataObj;               // This is the Data Object Mapped by this Map
                                                 // Getter to Attribute Map for this DO
    String                entityName    = null;  // Names of Entities which receive attributes 
                                                 //   from this Data Object 
    String                key           = null;  // The Key identifying this DO
    String                keyAccessor   = null;  // The name of the Accessor to get the Key
    String                key_attribute = null;  // The Attribute identifying the Entity in the XML

    // We figure out what accessors in the DO provide values we can
    // use populate attributes on the Entity, and we store the accessor to tag
    // mapping here. 
    HashMap<String,String> tagMap= new HashMap<String,String>();  
    
    
    DataObjectMap(String dataObjName,String entity, String tag, String key,String key_attribute) throws Exception {
        this.dataObjName   = dataObjName;
        this.entityName    = entity;
        this.tag           = tag;
        this.dataObj       = Class.forName(dataObjName);
        this.key           = key;
        this.key_attribute = key_attribute==null?key:key_attribute;
        if(key!=null){
           keyAccessor = "get"+key.substring(0,1).toUpperCase();
            if(key.length()>1){
                keyAccessor = keyAccessor+key.substring(1);
            }
        }
    }    
 
    /**
     * Returns true if an Entity tag has been openned.
     * @param datamap
     * @param idataObj
     * @return
     * @throws Exception
     */
    public boolean OpenEntityTag(DataMap datamap, Object idataObj){
        try{
            @SuppressWarnings("rawtypes")
			Class params[]     = {};
            Object paramsObj[] = {};          
            Object keyValue = null;
            if(keyAccessor!=null){ 
                keyValue = dataObj.getMethod(keyAccessor,params).invoke(idataObj,paramsObj);
            }    
            if(!datamap.isInContext(tag, key_attribute, keyValue)){ 
               if(key_attribute!=null){ 
                  datamap.opentag(tag,key_attribute,keyValue);
               }else{
                  datamap.opentag(tag);
               }
               return true;
            }
            return false;
         }catch(Exception e){
            return false;
         }   
    }
    
    public void mapDO(DataMap datamap, Object idataObj)throws RulesException {
        if(loaded==false){
            String err = init(datamap);
            if(err!=null)throw new RulesException("Undefined","DataObjectMap",err);
        }
        @SuppressWarnings("rawtypes")
		Class params[]     = {};
        Object paramsObj[] = {};
        
        try {
            boolean needToClose = OpenEntityTag(datamap, idataObj);
            for(String getter : tagMap.keySet()){
                String tag = tagMap.get(getter);
                Method getV = dataObj.getMethod(getter, params);
                Object v    = getV.invoke(idataObj, paramsObj);
                datamap.printdata(tag, v);
            }
            if(needToClose){
                datamap.closetag();
            }
        } catch (SecurityException e) {
           throw new RulesException("invalidAccess","DataObjectMap",e.toString());
        } catch (IllegalArgumentException e) {
            throw new RulesException("invalidAccess","DataObjectMap",e.toString());
        } catch (NoSuchMethodException e) {
            throw new RulesException("undefined","DataObjectMap",e.toString());
        } catch (IllegalAccessException e) {
            throw new RulesException("invalidAccess","DataObjectMap",e.toString());
        } catch (InvocationTargetException e) {
            throw new RulesException("unknown","DataObjectMap",e.toString());
        }
    }
    /**
     * Returns null if the prefix isn't found.
     * @param prefix
     * @param n
     * @return
     */
    private String removePrefix(String prefix, Method method){
        if(method.getParameterTypes().length!=0) return null;
        String mName = method.getName();
        int l = prefix.length();
        if(mName.length() < l) return null;
        if(!mName.startsWith(prefix))return null;
        String first = mName.substring(l,l+1);
        if(mName.length()>l+1){
          mName = first.toLowerCase()+mName.substring(l+1);
        }
        return mName;
    }
    /**
     * Returns an array of error messages if the initialization fails.  This
     * routine looks through all the getters in the DO and looks for matches 
     * with the tags defined in the mapping file.  
     * @return
     */
    public synchronized String init(DataMap datamap)  {
        if(loaded == true) return null;
        try {
        	@SuppressWarnings("rawtypes")
    		Class c = Class.forName(dataObjName);
            Method [] methods = c.getMethods();
            for(Method method : methods){
                String tag = removePrefix("is", method);
                if(tag==null) tag = removePrefix("get",method);
                if(tag!=null){
                    AttributeInfo info = datamap.map.setattributes.get(tag);
                    if(info!= null) for(Attrib attrib : info.getTag_instances() ){
                        if( entityName.equalsIgnoreCase(attrib.enclosure)){
                            tagMap.put(method.getName(), tag);
                        }  
                    }
                }
            }
        } catch (ClassNotFoundException e) {
            loaded = true;
            return "Could not find the DO: "+dataObjName;
        }
        loaded = true;
        return null;
    }
 
}
