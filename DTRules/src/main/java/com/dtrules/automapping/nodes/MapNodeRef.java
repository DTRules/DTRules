/**
 * 
 */
package com.dtrules.automapping.nodes;
import java.util.HashMap;

import com.dtrules.automapping.AutoDataMap;
import com.dtrules.automapping.LabelMap;
import com.dtrules.automapping.access.IAttribute;
import com.dtrules.xmlparser.XMLPrinter;


/**
 * @author ps24876
 *
 */
public class MapNodeRef extends AMapNode {
    private IMapNode                child;
        
    public String toString(){
        return getAttribute().getName()+"->"+child.toString();
    }
 
    public String getName(){
    	return getAttribute().getName();
    }
 
    public MapNodeRef(IAttribute attribute, IMapNode parent){
        super(attribute, parent);
    }
    
    public void addChild(IMapNode node) {
        child = node;
    }
    
    public void remove(IMapNode node){
        if (child == node) child = null;
    }
    
    public String getLabel() {
        return getAttribute().getName();
    }

    public Object getData() {
        return null;
    }

    public void setData(Object data) {
    }
  
    /**
     * @return the child
     */
    public IMapNode getChild() {
        return child;
    }

    /**
     * @return the attribs
     */
    public HashMap<String, Object> getAttribs() {
        return null;
    }
   
       @Override
    public Object mapNode(AutoDataMap autoDataMap, LabelMap labelMap) {
        Object ref = autoDataMap.getCurrentGroup().getDataTarget()
                         .mapRef(autoDataMap, labelMap, this);
        return ref;
    }
    
    public void printDataLoadXML(AutoDataMap autoDataMap, XMLPrinter xout) {
        if(child!=null){
            xout.opentag(getAttribute().getName(),
                "type",getAttribute().getType(),
                "subType",getAttribute().getSubType(),
                "node","reference");
                child.printDataLoadXML(autoDataMap, xout);
            xout.closetag();
        }
    }

}