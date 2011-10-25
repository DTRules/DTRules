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

package com.dtrules.automapping.access;

import java.util.ArrayList;
import java.util.Date;
import java.util.HashMap;
import java.util.List;
import java.util.Set;

import com.dtrules.automapping.AutoDataMap;
import com.dtrules.automapping.AutoDataMapDef;
import com.dtrules.automapping.Label;
import com.dtrules.automapping.LabelMap;
import com.dtrules.automapping.nodes.IMapNode;
import com.dtrules.automapping.nodes.MapNodeAttribute;
import com.dtrules.automapping.nodes.MapNodeList;
import com.dtrules.automapping.nodes.MapNodeMap;
import com.dtrules.automapping.nodes.MapNodeObject;
import com.dtrules.automapping.nodes.MapNodeRef;
import com.dtrules.entity.IREntity;
import com.dtrules.entity.REntityEntry;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RArray;
import com.dtrules.interpreter.RBoolean;
import com.dtrules.interpreter.RDouble;
import com.dtrules.interpreter.RInteger;
import com.dtrules.interpreter.RName;
import com.dtrules.interpreter.RNull;
import com.dtrules.interpreter.RString;
import com.dtrules.interpreter.RTable;
import com.dtrules.interpreter.RDate;
import com.dtrules.interpreter.RType;
import com.dtrules.session.DTState;

/**
 * @author Paul Snow
 *
 */
public class DTRulesTarget implements IDataTarget {

    private final AutoDataMapDef autoDataMapDef;

    /**
     * Every IDataTarget is associated with an autoDataMapDef, and has a
     * name.  No state should be stored in the iDataTarget specific to a
     * particular session.  General state can be stored, that applies to
     * all sessions.
     * @param autoDataMapDef
     * @param name
     */
    public DTRulesTarget(AutoDataMapDef autoDataMapDef){
        this.autoDataMapDef = autoDataMapDef;
    }

    static final RType strType = RType.getType("string");

    /**
     * Looks to make sure that we have not yet created an Entity 
     * of this name with the given code.  If we have, we return the
     * earlier created Entity.  Otherwise, we create a new instance.
     * @param entity -- We assume this is a valid Entity name (no dot syntax)
     * @param code
     * @return Returns null if no entity was found.
     * @throws RulesException
     */
    public IREntity findEntity(AutoDataMap autoDataMap, String entity, Label label, Object key) {
       IREntity e    = null;       
       IRObject iKey = iconvert(key);
       try {
           if(label.isSingular()){                          // If singular, look for an instance
               e = autoDataMap.getEntities().get(entity); //   on the entity stack.
               if(e==null){                               // None found? create one.
                   e = autoDataMap.getSession().getState().findEntity(entity+"."+entity);
                   if(e==null){
                      e = autoDataMap.getSession().createEntity(null,entity);
                      autoDataMap.getEntities().put(entity,e);   // Remember the one we created, so we 
                   }   
               }                                              //   don't create another one.
           }else {
               String skey = entity+"$"+iKey.stringValue(); 
               if(key!=null && 
            		   !(key instanceof String 
            	         && ((String)key).length()==0)) {                // NOTE: We are NOT allowing "" as a key here!
                                                                         // If so, construct a key for that entity
                   e = (IREntity)autoDataMap.getEntities().get(skey);    //    and look for it.
               }   
               if(e==null) {                                              // Haven't created an entity with that key?
                   e = autoDataMap.getSession().createEntity(iKey,entity);// do so.
                   if(key!=null){
                       autoDataMap.getEntities().put(skey,e);   // If we have a key, remember this entity!
                   }
               }
           }

       if(e==null)throw new RuntimeException("Failed to create the entity "+entity);
       if(key != null){
           e.addAttribute(
                   IREntity.mappingKey, 
                   key.toString(), 
                   iKey, 
                   false, 
                   true, 
                   strType, 
                   "", "", "","");
           e.put(null, IREntity.mappingKey, iKey);
       }
       
       return e;
       } catch (RulesException e1) {
            return null;
       }
    }   

    /**
     * Convert a Java object into a Rules Engine object (generally allocate a wrapper
     * around said java object).
     * @param object
     * @return
     */
    public IRObject iconvert(Object object){
        if(object instanceof String){
            return RString.newRString((String) object);
        }else if (object instanceof Integer){
            return RInteger.getRIntegerValue((Integer)object);
        }else if (object instanceof Date){
            return RDate.getRTime((Date)object);
        }else if (object instanceof Double){
            return RDouble.getRDoubleValue((Double) object);
        }else if (object instanceof Long){
            return RInteger.getRIntegerValue((Long) object);
        }else if (object instanceof Boolean){
        	return RBoolean.getRBoolean((Boolean)object);
        }
        
        return RNull.getRNull();
    }
    
    /**
     * Convert a Rules Engine object into a Java object 
     * @param object
     * @return
     */
    public Object convert(IRObject object){
        try {
            int otype = object.type().getId();
            
            if(otype == IRObject.iString)  return object.stringValue();
            if(otype == IRObject.iInteger) return object.longValue();
            if(otype == IRObject.iDouble)  return object.doubleValue();
            if(otype == IRObject.iName)    return object.stringValue();
            if(otype == IRObject.iDate)    return object.timeValue();
            if(otype == IRObject.iBoolean) return object.booleanValue();
            if(otype == IRObject.iNull)    return null;
            
        } catch (RulesException e) { }
        return null;
    }        
     
    
    /**
     * Maps a list to the target.  There are two sorts of lists supported.  Lists of
     * primitive objects (like strings, integers, dates, numbers, etc.) and lists of
     * objects (like clients, addresses, providers, etc.).
     */
    @SuppressWarnings({ "unchecked", "deprecation" })
    @Override
    public Object mapList(AutoDataMap autoDataMap, LabelMap labelMap, MapNodeList node) {
        IAttribute a = node.getAttribute();
        try {
            RArray list = null;
            RName rname = null;
            Object object = autoDataMap.getCurrentObject();
            if(object instanceof IREntity){
                IREntity entity = (IREntity) object;
                rname = mapName(autoDataMap, labelMap, node.getLabel());
                IRObject olist = entity.get(rname);
                if(olist != null && olist.type().getId() == IRObject.iArray){
                    list = olist.rArrayValue();
                }
            }
            
            node.setTargetList(list);
            
            // Handle Arrays of primitives ... We just keep a link to the list 
            // from our data source in this case.
            if(a.getSubType().isPrimitive()){
                if(list == null) return null;
       
                if(node.getData()!=null) for (Object d : (List<Object>) node.getData()){
                    IRObject dobj = iconvert(d);
                    list.add(dobj);
                    if (autoDataMap.getSession().getState().testState(DTState.TRACE)) {
                        autoDataMap.getSession().getState().traceInfo(
                                "addto", "arrayId", ((RArray)list).getID() + "", dobj.postFix());
                    }

                }
                return list;
            }else{
                List<IMapNode> children = node.getChildren();
                for(IMapNode c : children){
                    autoDataMap.pushMark(); // Make sure no children try and update our list's parent
                    Object o = c.mapNode(autoDataMap, labelMap); 
                    if(o instanceof List && ((List<Object>)o).size()==1){ // Mostly we are going to get an array of length  
                        o = ((List<Object>)o).get(0);                     //   one of the object we want.  If that's the 
                    }                                                     //   case, get the object we want from the List.
                    autoDataMap.pop();                                    // Remove the mark.
                    if(list!=null                                         // If we have a list, and
                            && o != null                                  //   a rules engine object
                            && o instanceof IRObject){                    //   then add it to our list.
                        list.add((IRObject)o);
                        if (autoDataMap.getSession().getState().testState(DTState.TRACE)) {
                            autoDataMap.getSession().getState().traceInfo(
                                    "addto", "arrayId", ((RArray)list).getID() + "", ((IRObject)o).postFix());
                        }                          
                    }
                }
            }
        } catch (RulesException e) {}
        
        return null;
    }

    /**
     * Do any necessary mapping, and return the RName for the attribute/property.
     * @param autoDataMap
     * @param labelMap
     * @param name
     * @return
     */
    private RName mapName(AutoDataMap autoDataMap, LabelMap labelMap, String name){
        String tname = labelMap.getMapAttributes().get(name);
        if(tname == null) tname = name;
        RName rname = RName.getRName(tname);
        return rname;
    }

    
    @Override
    public void update(AutoDataMap autoDataMap, MapNodeAttribute node) {
        if(node.getParent() instanceof MapNodeObject){
            MapNodeObject parentNode = (MapNodeObject)node.getParent();
            Object        tgtObj     = parentNode.getTargetObject();
            if(tgtObj == null) return;
            IREntity      entity     = (IREntity) tgtObj;
            RName         attribute  = RName.getRName(node.getAttribute().getName());
            REntityEntry  entry      = entity.getEntry(attribute);
            if(entry != null && entry.writable){
                IRObject  value  = entity.get(node.getAttribute().getName());
                Object    v      = convert(value);    
                node.setData(v);
            }
        }
        
    }


    @Override
    public void update(AutoDataMap autoDataMap, MapNodeList node) {
    	if(node.getList() == null){
    		node.setList(new ArrayList<Object>());
    	}else{
    	    node.getList().clear();
    	}
    	
        RArray result = (RArray) node.getTargetList();
        for(IRObject obj : result){
            Object value   = convert(obj);
            node.getList().add(value);
        }
        
    }

    @Override
    public void update(AutoDataMap autoDataMap, MapNodeMap node) {
        
    	if(node.getMap() == null){
    		node.setMap(new HashMap<Object,Object>());
    	}else{
    	    node.getMap().clear();
    	}
    	
        RTable result = (RTable) node.getTargetMap();
        Set<IRObject> keys = result.getTable().keySet();
        for(IRObject key : keys){
            Object thekey   = convert(key);
            Object thevalue = null;
            try {
                thevalue = convert(result.getValue(key));
            } catch (RulesException e) {} 
            node.getMap().put(thekey, thevalue);
        }
    }

    @Override
    public void update(AutoDataMap autoDataMap, MapNodeObject node) {
        
    }

    @Override
    public void update(AutoDataMap autoDataMap, MapNodeRef node) {
        
    }


    /* (non-Javadoc)
     * @see com.dtrules.automapping.access.IDataTarget#mapRef(com.dtrules.automapping.AutoDataMap, com.dtrules.automapping.LabelMap, com.dtrules.automapping.nodes.IMapNode)
     */
    @SuppressWarnings("unchecked")
    @Override
    public Object mapRef(AutoDataMap autoDataMap, LabelMap labelMap, MapNodeRef node) {
        
        // Ignore references where no data is stored.
        if(node.getChild()==null)return null;
        // We ignore whatever labelMap has been passed in.  We need the one that maps
        //   the object we are referencing.
        String sourceLabel = ((MapNodeObject) (node.getChild())).getLabel();
        labelMap = autoDataMap.getAutoDataMapDef().findLabelMap(sourceLabel);

        // Now we need the target label that matches this source label.  There is a 
        // problem here, that we might need to map this label to multiple source 
        // objects in the target environment.  In this case, it becomes unclear how
        // to resolve such a reference.
        while(labelMap != null && labelMap.getTargetGroup()!=autoDataMap.getCurrentGroup()){
            labelMap = labelMap.getNext();
        }        
        
        // If we don't have a Label Map, it needs to be created by our target source.
        if(labelMap == null){
            labelMap = autoDataMap.getCurrentGroup().getDataTarget()
                            .getLabelMap(autoDataMap, (MapNodeObject) node.getChild());
        }

        // We get the object first, because if the LabelMap is going to get cached, it
        // is going to get cached as a result of this call.  
        autoDataMap.pushMark();
        List<Object> results = (List<Object>) node.getChild().mapNode(autoDataMap, labelMap);
        IRObject ref = null;
        if(results != null && results.size()==1){
            ref = (IRObject) results.get(0);
        }
        autoDataMap.pop();
        
        if(labelMap == null){
            labelMap = autoDataMap.getAutoDataMapDef().findLabelMap(sourceLabel);
            while(labelMap != null && labelMap.getTargetGroup()!=autoDataMap.getCurrentGroup()){
                labelMap = labelMap.getNext();
            }
        }

        RName rname = mapName(autoDataMap, labelMap, node.getAttribute().getName());
        
        IREntity entity = (IREntity) autoDataMap.getCurrentObject();
        
        try{                        // We are going to ignore assertion errors.
           entity.put(autoDataMap.getSession(), rname, ref);
           return ref;
        }catch(Exception e){}

        return ref;                
    }
    
    @Override
    public LabelMap getLabelMap(AutoDataMap autoDataMap, MapNodeObject node){
        LabelMap labelMap = autoDataMapDef.findLabelMap(node.getLabel());
        while(labelMap != null && labelMap.getTargetGroup() != autoDataMap.getCurrentGroup()){
            labelMap = labelMap.getNext();
        }
        if(labelMap == null){
            Label sourceLabel = node.getSourceLabel();
            String targetLabelname = node.getLabel();
            Label.newLabel(
                    autoDataMap.getCurrentGroup(), 
                    targetLabelname,
                    targetLabelname,
                    sourceLabel.getKey(),
                    false);
            labelMap = new LabelMap(autoDataMapDef, node.getLabel(), targetLabelname);
            labelMap.setSourceGroup(node.getGroup());
            labelMap.setTargetGroup(autoDataMap.getCurrentGroup());
            
            autoDataMapDef.addLabelMap(labelMap);
        }
        return labelMap;
    }
    
    
    /**
     * We map the given object to the target context.  If no labelMap yet exists
     * for this mapping, we create one.
     */
    @Override
    public Object mapObject(AutoDataMap autoDataMap, LabelMap labelMap, MapNodeObject node) {        
       Label target = autoDataMap.getCurrentGroup().findLabel(labelMap.getTarget());
       IREntity entity = findEntity(autoDataMap, target.getSpec(),target, node.getKey());   
       node.setTargetObject(entity);
       return entity;
    }

    
    
    @Override
    public Object mapAttribute(AutoDataMap autoDataMap, LabelMap labelMap, MapNodeAttribute node) {
        RName rname = mapName(autoDataMap, labelMap, node.getAttribute().getName());
        IREntity entity   = (IREntity) autoDataMap.getCurrentObject();
        if(entity == null){
            return null;
        }
        Object data = node.getData();
        IRObject value = iconvert(data);
        try {
            entity.put(autoDataMap.getSession(), rname, value);
            value = entity.get(rname);
        } catch (RulesException e) {
            
        }        
        return value;
    }

    /**
     * Return the default Label name for an Object from a given source.
     * @param obj
     * @return
     */
    public String getName(Object obj){
        return obj.getClass().getSimpleName();
    }

    @Override
    public void init(AutoDataMap autoDataMap) {
        String entryPoint = autoDataMap.getCurrentGroup().getAttribs().get("entryPoint");
        try{
            autoDataMap.getSession().initialize(entryPoint);
        }catch(Exception e){
            throw new RuntimeException("Error initializing the session for accepting data. "+e);
        }
    }
    
    @Override
    public Object mapMap(AutoDataMap autoDataMap, LabelMap labelMap, MapNodeMap node){
        RTable rTable = null;
        try {
            RName rname = null;
            Object object = autoDataMap.getCurrentObject();
            if(object instanceof IREntity){
                IREntity entity = (IREntity) object;
                rname = mapName(autoDataMap, labelMap, node.getAttribute().getName() );
                IRObject olist = entity.get(rname);
                if(olist != null && olist.type().getId() == IRObject.iTable){
                    rTable = olist.rTableValue();
                }
            }
            if(rTable == null){
                rTable = RTable.newRTable(autoDataMap.getSession().getEntityFactory(),
                        null , null);
            }
            
            node.setTargetMap(rTable);
            if(node.getMap() != null){
	            Set<Object> keys = node.getMap().keySet();
	            for(Object key : keys ){
	                rTable.setValue(iconvert(key), 
	                        iconvert(node.getMap().get(key).toString()));
	            }
            }
        } catch (RulesException e) { }
        
        return rTable;
    }
 
    
    
}
