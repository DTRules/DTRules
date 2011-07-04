/**
 * 
 */
package com.dtrules.automapping.access;

import com.dtrules.automapping.Label;
import com.dtrules.automapping.MapType;
import com.dtrules.xmlparser.XMLPrinter;

/**
 * @author ps24876
 *
 */
public interface IAttribute {

    /**
     * @return the name
     */
    String getName();

    /**
     * @return the type
     */
    MapType getType();

    /**
     * @return the typeText
     */
    String getTypeText();

    /**
     * @return the subType
     */
    MapType getSubType();

    /**
     * @return the subTypeText
     */
    String getSubTypeText();

    /**
     * Get the attribute value from the given Object
     * @param obj
     * @return
     */
    Object get(Object obj);

    /**
     * Set the attribute value on the given Object
     * @param obj
     * @param label
     */
    void set(Object obj, Object value);

    /**
     * Returns true if this is the key for the object to which the attribute
     * belongs.
     * @return
     */
    boolean isKey();
    
    void printXML(XMLPrinter xout);

}