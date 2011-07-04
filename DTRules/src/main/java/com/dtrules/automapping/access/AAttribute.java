/**
 * 
 */
package com.dtrules.automapping.access;

import com.dtrules.automapping.Label;
import com.dtrules.automapping.MapType;
import com.dtrules.automapping.nodes.MapNodeAttribute;
import com.dtrules.xmlparser.XMLPrinter;

/**
 * Allows the insertion of an attribute value anonymously into a DataMap;
 * In other words, you can place a value into the DataMap that isn't 
 * associated with an Object from the Data Source.  For example, the 
 * currentDate can be added to a main Object, even if the main Object
 * doesn't have a currentDate attribute value.  Any attempt to modify this
 * attribute when mapping data back to the source will be ignored.
 * 
 * The AAttribute cannot be part of the general configuration of a RuleSet
 * as it holds references to actual nodes in the DataMap.  
 * 
 * @author Paul Snow
 *
 */
public class AAttribute implements IAttribute {
    
    String            name;
    String            type;
    String            subType;
    MapNodeAttribute  mapNodeAttribute;

    public AAttribute (String name,String type, MapNodeAttribute mapNodeAttribute){
        this.name               = name;
        this.type               = type;
        this.mapNodeAttribute   = mapNodeAttribute;
    }
    
    /**
     * You cannot read the object source;  The value of an AAttribute is
     * only defined by Java code in the loader.
     */
    @Override
    public Object get(Object obj) {
        return mapNodeAttribute.getData();
    }

    /**
     * Return this attribute name.
     */
    @Override
    public String getName() {
        return name;
    }

    /**
     * Return null for now
     */
    @Override
    public MapType getSubType() {
        return MapType.get(subType);
    }

    /**
     * Return null for now
     */
    @Override
    public String getSubTypeText() {
        return subType;
    }

    /**
     * Return null for now
     */
    @Override
    public MapType getType() {
        return MapType.get(type);
    }

    /**
     * Return null for now
     */
    @Override
    public String getTypeText() {
        return type;
    }

    /**
     * Return null for now
     */
    @Override
    public boolean isKey() {
        return false;
    }

    /**
     * Do Nothing.
     */
    @Override
    public void printXML(XMLPrinter xout) {
        
    }

    public void set(Object obj, Object value) {

    }

}
