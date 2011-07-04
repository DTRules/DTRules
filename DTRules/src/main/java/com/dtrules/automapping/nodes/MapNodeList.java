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
import java.util.HashMap;
import java.util.List;

import com.dtrules.automapping.AutoDataMap;
import com.dtrules.automapping.AutoDataMapDef;
import com.dtrules.automapping.LabelMap;
import com.dtrules.automapping.MapType;
import com.dtrules.automapping.access.IAttribute;
import com.dtrules.automapping.access.IDataTarget;
import com.dtrules.xmlparser.XMLPrinter;


/**
 * @author Paul Snow
 * Defines an Attribute on an object.
 */
public class MapNodeList extends AMapNode {
    private List<Object>            list        = null;
    private Object                  targetList;
    private String                  subTypeText;
    private MapType                 subType;
    private boolean                 cached      = false;
    
    public MapNodeList(IAttribute attribute, IMapNode parent){
        super(attribute,parent);
    }
    
    public String toString(){
        return getAttribute().getName()+ " list of "+subTypeText;
    }
    
    public String getName(){
    	return getAttribute().getName();
    }
 
    /**
     * @return the targetList
     */
    public Object getTargetList() {
        return targetList;
    }

    /**
     * @param targetList the targetList to set
     */
    public void setTargetList(Object targetList) {
        this.targetList = targetList;
    }

    /**
     * @return the subType
     */
    public String getSubType() {
        return subTypeText;
    }

    /**
     * @param subType the subType to set
     */
    public void setSubType(String subType) {
        this.subTypeText = subType;
        this.subType     = MapType.get(subType);
    }
  
    
    public String getLabel() {
        return getAttribute().getName();
    }

    @Deprecated
    public Object getData() {
        return list;
    }

    @Deprecated
    @SuppressWarnings("unchecked")
    public void setData(Object data) {
        this.list = (List<Object>) data;
    }

    public List<Object> getList() {
		return list;
	}

	public void setList(List<Object> list) {
		this.list = list;
	}

	/**
     * @return the attribs
     */
    public HashMap<String, Object> getAttribs() {
        return null;
    }

    /**
     * @return the subTypeText
     */
    public String getSubTypeText() {
        return subTypeText;
    }

    /**
     * @param subTypeText the subTypeText to set
     */
    public void setSubTypeText(String subTypeText) {
        this.subTypeText = subTypeText;
    }

    /**
     * @return the cached
     */
    public boolean isCached() {
        return cached;
    }

    /**
     * @param cached the cached to set
     */
    public void setCached(boolean cached) {
        this.cached = cached;
    }

    /**
     * @param subType the subType to set
     */
    public void setSubType(MapType subType) {
        this.subType = subType;
    }

    /* (non-Javadoc)
     * @see com.dtrules.automapping.nodes.IMapNode#mapNode(com.dtrules.automapping.AutoDataMap, com.dtrules.automapping.LabelMap, com.dtrules.automapping.Group)
     */
    @Override
    public Object mapNode(AutoDataMap autoDataMap, LabelMap labelMap) {
        IDataTarget dtarget = autoDataMap.getCurrentGroup().getDataTarget();
        dtarget.mapList(autoDataMap, labelMap, this);
        return null;
    }

    public void printDataLoadXML(AutoDataMap autoDataMap, XMLPrinter xout) {
        if(subType == MapType.OBJECT){
            xout.opentag(getAttribute().getName(),
                    "node", "list",
                    "type","list",
                    "subType",subTypeText);
                List<IMapNode> cs = getChildren();
                for(IMapNode c : cs){
                    c.printDataLoadXML(autoDataMap, xout);
                }
            xout.closetag();
        }else{
            xout.opentag(getAttribute().getName(),
                    "node", "list",
                    "type","list",
                    "subType",subTypeText);
                if(list != null) for(Object o : (List<Object>)list){
                    xout.printdata(subTypeText,
                            "node","primitive",
                            AutoDataMapDef.convert(o));
                }
            xout.closetag();
        }
    }

	@Override
	public void update(AutoDataMap autoDataMap) {
		super.update(autoDataMap);
		if(subType.isPrimitive() && getParent() instanceof MapNodeObject && targetList != null ){
			autoDataMap.getCurrentGroup().getDataTarget().update(autoDataMap, this);
			((MapNodeObject)getParent()).getGroup().getDataSource().update(autoDataMap, this);
		}
	}

}