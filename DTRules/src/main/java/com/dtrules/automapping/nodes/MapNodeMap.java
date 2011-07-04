/**
 * 
 */
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
