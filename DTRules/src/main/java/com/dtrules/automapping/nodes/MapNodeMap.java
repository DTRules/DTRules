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

package com.dtrules.automapping.nodes;

import java.util.Map;
import java.util.Set;

import com.dtrules.automapping.AutoDataMap;
import com.dtrules.automapping.Group;
import com.dtrules.automapping.LabelMap;
import com.dtrules.automapping.access.IAttribute;
import com.dtrules.xmlparser.XMLPrinter;

/**
 * @author ps24876
 *
 */
public class MapNodeMap extends AMapNode {

	private Map<Object, Object>     map;
    private Object                  targetMap;
    
    public MapNodeMap(IAttribute attribute, IMapNode parent){
        super(attribute,parent);
    }
    
    public String getName(){
    	return getAttribute().getName();
    }
    
    public Map<Object,Object> getMap(){
        return map;
    }
    
    @Override
    public void addChild(IMapNode node) {
    }

    @Override
    public Object mapNode(AutoDataMap autoDataMap, LabelMap labelMap) {
        
        Group target = autoDataMap.getCurrentGroup();
        
        return target.getDataTarget().mapMap(autoDataMap,labelMap, this);

    }

    @Override
    public void printDataLoadXML(AutoDataMap autoDataMap, XMLPrinter xout) {
        xout.opentag(getAttribute().getName(),"type","map","node","map");
            if(map != null){
	            Set<Object> keys = map.keySet();
	            for(Object key : keys){
	                xout.opentag("pair");
	                String type = key.getClass().getSimpleName();
	                xout.printdata("key","type",type,key.toString());
	                Object value = map.get(key);
	                type = value.getClass().getSimpleName();
	                xout.printdata("value","type",type,value.toString());
	                xout.closetag();
	            }
            }
        xout.closetag();
    }

    /**
     * @param map the map to set
     */
    public void setMap(Map<Object, Object> map) {
        this.map = map;
    }

    /**
     * @return the targetMap
     */
    public Object getTargetMap() {
        return targetMap;
    }

    /**
     * @param targetMap the targetMap to set
     */
    public void setTargetMap(Object targetMap) {
        this.targetMap = targetMap;
    }

	@Override
	public void update(AutoDataMap autoDataMap) {
		super.update(autoDataMap);
		if(getParent() instanceof MapNodeObject){
			autoDataMap.getCurrentGroup().getDataTarget().update(autoDataMap, this);
			((MapNodeObject)getParent()).getGroup().getDataSource().update(autoDataMap, this);
		}
	}
    
    
    
}
