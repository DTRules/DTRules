/**
 * 
 */
package com.dtrules.automapping.nodes;

import java.util.List;

import com.dtrules.automapping.AutoDataMap;
import com.dtrules.automapping.LabelMap;
import com.dtrules.xmlparser.XMLPrinter;

/**
 * The MapNode is used to track where data came from, primarily.  Just because
 * data ends up in the AutoDataMap doesn't mean it gets mapped to the target.
 * The target takes what data it can accept from the data provided from the source.
 * <br><br>
 * If you wish to restrict the data that goes into the autoDataMap, this is 
 * done by specifying in the mapping configuration exactly what attributes to
 * pull from a source object (be it from a Java, Entity, or XML source).
 * <br><br>
 * Furthermore, we only map list of objects when they are specifically specified
 * as a source with a sub-type defined in the object definitions.
 * <br><br>
 * <b>Right now we are doing some things that would interfere with loading the 
 * data from a source, and driving that data to different targets.  The issue is
 * the invalid flag used to avoid nodes that are "dead" to the target, i.e. 
 * attributes from the source that do not map to an attribute in the target.</b>
 * 
 * @author Paul Snow
 *
 */
public interface IMapNode {
        
	String getName();
	
	void addChild(IMapNode node);
     
    void setInvalid(boolean invalid);
     
    boolean isInvalid();
     
    List<IMapNode> getChildren();
          
    IMapNode getParent();

    void setParent(IMapNode parent);
          
    void printDataLoadXML(AutoDataMap autoDataMap, XMLPrinter xout);
    
    /**
      * Maps this node (and its children nodes) to the target.  The target
      * object updated is returned (or a null if no update occurs). 
      * @param autoDataMap
      * @param labelMap
      * @param target
      * @return
      */
    Object mapNode(AutoDataMap autoDataMap, LabelMap labelMap);

    void update(AutoDataMap autoDataMap);
     
}
