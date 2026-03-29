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

import com.dtrules.automapping.AutoDataMap;
import com.dtrules.automapping.AutoDataMapDef;
import com.dtrules.automapping.LabelMap;
import com.dtrules.automapping.access.AAttribute;
import com.dtrules.automapping.access.IAttribute;
import com.dtrules.automapping.access.IDataTarget;
import com.dtrules.xmlparser.XMLPrinter;


/**
 * @author Paul Snow
 * Defines an Attribute on an object.
 */
public class MapNodeAttribute extends AMapNode {
    
    private Object                  data        = null;
    private boolean                 altered     = true;
    
    public String toString(){
    	if(data!=null){
            return getAttribute().getName()+" = "+data.toString();
    	}else{
    	    return getAttribute().getName()+" = null";
    	}
    }
    
    public String getName(){
    	return getAttribute().getName();
    }
    
    public MapNodeAttribute(IAttribute attribute, IMapNode parent){
        super(attribute,parent);
    }
    
    /**
     * This constructor is used to create a MapNodeAttribute that isn't really associated
     * with the data source, but is supplied by the data loader for the application.  It
     * constructs an AAttribute which isn't part of the mapping of the source to the 
     * AutoDataMap.
     * 
     * @param parent
     * @param labelname
     * @param type
     * @param attributeName
     * @param value
     */
    public MapNodeAttribute(
            IMapNode parent, 
            String labelname, 
            String type, 
            String attributeName, 
            Object value){
        super(null,parent);
        setAttribute(new AAttribute(attributeName, type, this));
        this.data      = value;
    }
    
    public void addChild(IMapNode node) {
    }
    
    public void remove(IMapNode node){
    }
    
    public String getLabel() {
        return getAttribute().getName();
    }

    /**
     * Returns true if the data has changed, false if this set
     * didn't change the attribute.
     * @param data
     * @return
     */
    public boolean setData(Object data){
        if(this.data == null || !this.data.equals(data)){
            this.data = data;
            return true;
        }
        return false;
    }
 
    public Object getData() {
        return data;
    }

    /**
     * @return the altered
     */
    public boolean isAltered() {
        return altered;
    }

    /**
     * @param altered the altered to set
     */
    public void setAltered(boolean altered) {
        this.altered = altered;
    }

    /**
     * @return the attribs
     */
    public HashMap<String, Object> getAttribs() {
        return null;
    }
    
    @Override
    public Object mapNode(AutoDataMap autoDataMap, LabelMap labelMap) {
        IDataTarget dataTarget = autoDataMap.getCurrentGroup().getDataTarget();
        return dataTarget.mapAttribute(autoDataMap, labelMap, this);
        
    }

    public void printDataLoadXML(AutoDataMap autoDataMap, XMLPrinter xout) {
        xout.printdata(getAttribute().getName(),
                "node", "attribute",
                "type", getAttribute().getTypeText(), 
                AutoDataMapDef.convert(data)); 
    }

	@Override
	public void update(AutoDataMap autoDataMap) {
		if( getParent() instanceof MapNodeObject && data != null ){
			autoDataMap.getCurrentGroup().getDataTarget().update(autoDataMap, this);
			((MapNodeObject)getParent()).getGroup().getDataSource().update(autoDataMap, this);
		}
	}

}