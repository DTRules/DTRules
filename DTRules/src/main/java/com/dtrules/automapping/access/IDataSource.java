/**
 * 
 */
package com.dtrules.automapping.access;

import java.util.List;

import com.dtrules.automapping.AutoDataMap;
import com.dtrules.automapping.Group;
import com.dtrules.automapping.Label;
import com.dtrules.automapping.nodes.MapNodeAttribute;
import com.dtrules.automapping.nodes.MapNodeList;
import com.dtrules.automapping.nodes.MapNodeMap;
import com.dtrules.automapping.nodes.MapNodeObject;
import com.dtrules.automapping.nodes.MapNodeRef;

/**
 * A Data Source for reading data from some particular Data
 * source type, such as Java Objects or DTRules Entities
 * @author Paul Snow
 *
 */
public interface IDataSource {
        
    
    /**
     * Return a spec given an Object from a given source.
     * @param obj
     * @return
     */
    public String getSpec(Object obj);
    
    /**
     * Return the default Label name for an Object from a given source.  How
     * this name is determined from an Object is source dependent.
     * @param obj
     * @return
     */
    public String getName(Object obj);
    
    /**
     * Return what we assume to be the key for the Object from a given source.
     * @param obj
     * @return
     */
    public String getKey(Object obj);
    
    /**
     * Find the label associated with this object.  If it isn't in the
     * specifications yet, then it will be created.  If the object hasn't
     * been cached yet, it will be cached.
     * 
     * @param labelName     If null, the simple name of the obj is used as the label Name.
     * @param key           If null, no key is used.
     * @param singular      
     * @param object
     * @return
     */
    public Label createLabel(
            AutoDataMap autoDataMap, 
            Group  group,
            String labelName, 
            String key, 
            boolean singular, 
            Object object);
    
    /**
     * Returns a list of children for this object.  This call is made when an object
     * is encountered in the source that doesn't have any obvious mapping into the
     * autoDataMap.  We don't just want to stop (say when processing an XML stream
     * for which a tag doesn't map), but instead we want to (possibly) process any 
     * children of the source object (such as the tags underneath the tag in an XML
     * source that doesn't map).
     *  
     * @param object
     * @return
     */
    public List<Object> getChildren(Object object);
    
    /**
     * If possible, returns the key value for this node.  If this is not possible for
     * the source type, then the current value for the key for the node is returned.
     * @param node
     * @return
     */
    public Object getKeyValue(MapNodeObject node, Object object);
    
    /**
     * Update the source for this change in an attribute.
     * @param autoDataMap
     * @param node
     */
    public void update(AutoDataMap autoDataMap, MapNodeAttribute node);
    /**
     * Update the source for this change in a List
     * @param autoDataMap
     * @param node
     */
    public void update(AutoDataMap autoDataMap, MapNodeList      node);
    /**
     * Update the source for this change in a Map
     * @param autoDataMap
     * @param node
     */
    public void update(AutoDataMap autoDataMap, MapNodeMap       node);
    /**
     * Update the source for this change in an Object
     * @param autoDataMap
     * @param node
     */
    public void update(AutoDataMap autoDataMap, MapNodeObject    node);
    /**
     * Update the source for this change in an Object Reference
     * @param autoDataMap
     * @param node
     */
    public void update(AutoDataMap autoDataMap, MapNodeRef       node);
    
}
