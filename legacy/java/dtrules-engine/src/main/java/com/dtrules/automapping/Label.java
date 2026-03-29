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

package com.dtrules.automapping;

import java.util.ArrayList;
import java.util.Map;
import com.dtrules.automapping.access.IAttribute;
import com.dtrules.xmlparser.XMLPrinter;

/**
 * @author Paul Snow
 *
 */
public class Label {
    
    final private String name;           // Name used by the Mapping code (Doesn't have to match
                                         //   the name of the JavaObject; In fact it often will not!)
    Group                group;          // The group to which this label belongs.
    private String       spec;           // Has to be a valid class name, if a CLASS.  If an entity,
    private String       key;            // The object attribute to be used as a key
    private boolean      singular;       // True of only one of these labels should be created.

    private Label        next;           // labels map to or from a class, entity, or XML section;  Each type
                                         //    of object needs their own entry.  We keep a list of them.
    private Label        nextSpec;       // The same object can be mapped via many labels.  This is a list
                                         //    organized by source specification
    private ArrayList<IAttribute> attributes = new ArrayList<IAttribute>();
    
    private boolean      cached     = false; // Indicates that we might have properties to 
                                             //    learn about
    
    Label(String name){
        this.name = name;
    }
   
    public String toString(){
        return name;
    }
    
    /**
     * Tests if two Labels are functionally equal to each other.
     * @param label
     * @return
     */
    public boolean equals(Label label){
        if(!name.equals(label.name))        return false;
        if(!spec.equals(label.spec))        return false;
        if(!key.equals(label.key))          return false;
        if(!singular == (label.singular))   return false;        
        return true;
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
     * Add a label to the given group.
     * @param label
     */
    public void addLabel(Group group){
        next = group.getLabels().get(name);           // Add to head of list
        group.getLabels().put(name, this);            //   organized by label
        nextSpec = group.getBySpecs().get(spec);      // Add to head of list
        group.getBySpecs().put(spec, this);           //   organized by spec
    }

    
    
    public static Label newLabel(Group group, Map<String, String> attribs){
        
        
        String  name          = attribs.get("label");
        String  spec          = attribs.get("spec");
        String  key           = attribs.get("key");
        Boolean singular      = AutoDataMapDef.parseBoolean(attribs.get("singular"));
        return newLabel(group, name,spec,key,singular);
    }
    
    public static Label newLabel(Group group, String name, String spec, String key, Boolean singular){    
        if(name == null)throw new RuntimeException("Name cannot be null");
        Label label = group.findLabel(name, group.getType(), spec);
        if(label !=null){
            if(singular != null) label.setSingular(singular);
            return label;  // Ignore the new label, and return the existing one.
        }else{
            label = new Label(name);
            label.group    = group;
            label.spec     = spec;
            label.key      = key;
            if(singular != null) label.singular = singular;
            
            label.addLabel(group);
        }
        return label;
    }
            
    /**
     * Returns the next label of the same type.
     * @param label
     * @return
     */
    Label nextLabel(Label label){
        Label nextLabel = label.getNext();
        while(nextLabel!=null && !nextLabel.group.equals(label.group)){
            nextLabel = nextLabel.getNext();
        }
        return nextLabel;
    }

    
    /**
     * @return the name
     */
    public String getName() {
        return name;
    }
    
    /**
     * @return the spec
     */
    public String getSpec() {
        return spec;
    }

    /**
     * @param spec the spec to set
     */
    public void setSpec(String spec) {
        this.spec = spec;
    }
    
    /**
     * @return the group
     */
    public Group getGroup() {
        return group;
    }

    /**
     * @return the key
     */
    public String getKey() {
        return key;
    }
    /**
     * @param key the key to set
     */
    public void setKey(String key) {
        this.key = key;
    }
    /**
     * @return the singular
     */
    public boolean isSingular() {
        return singular;
    }
    /**
     * @param singular the singular to set
     */
    public void setSingular(boolean singular) {
        this.singular = singular;
    }
    /**
     * @return the next
     */
    public Label getNext() {
        return next;
    }
    /**
     * @param next the next to set
     */
    public void setNext(Label next) {
        this.next = next;
    }
    /**
     * @return the nextSpec
     */
    public Label getNextSpec() {
        return nextSpec;
    }

    /**
     * @param nextSpec the nextSpec to set
     */
    public void setNextSpec(Label nextSpec) {
        this.nextSpec = nextSpec;
    }

    /**
     * @return the attributes
     */
    public ArrayList<IAttribute> getAttributes() {
        return attributes;
    }

    /**
     * Gets the IAttribute object on this Label by the same name.
     * If no IAttribute of that name exists, a null is returned.
     * @param name
     * @return
     */
    public IAttribute getAttribute(String name){
    	for(IAttribute a : attributes){
    		if(a.getName().equals(name)){
    			return a;
    		}
    	}
    	return null;
    }
    
    
    public void printXML(AutoDataMapDef autoDataMapDef, XMLPrinter xout){
        
        String _singular = singular ? "true" : "false";
        
        xout.opentag("object", 
                "label",     name, 
                "spec",      spec, 
                "key",       key,
                "singular",  _singular);
        Object as[] = this.attributes.toArray();
        AutoDataMap.sort(as);
        
        for(Object a : as){
            ((IAttribute) a).printXML(xout);
        }    
        xout.closetag();        
        if(next!=null){
            next.printXML(autoDataMapDef, xout);
        }
    }
    
}
