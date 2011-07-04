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

import java.util.HashMap;
import java.util.Map;
import java.util.Set;

import com.dtrules.automapping.access.IDataSource;
import com.dtrules.automapping.access.IDataTarget;
import com.dtrules.xmlparser.XMLPrinter;

/**
 * The Group class provides a mechanism for mapping data through the same 
 * mapping for multiple purposes.  A group must be one of the supported 
 * types (java, dtrules, xml), which defines the mechanism by which the
 * mapping configuration is constructed, data is accessed, and/or data is
 * asserted.
 * 
 * Labels are generally global for a type.  However, specific object mappings
 * (that do not follow the general conventions, or object pruning) can be
 * done for groups only.  Thus when loading or asserting data, Label searches 
 * or Pruning searches are done first at the group level, then at the type 
 * level, with the first match taken.
 *
 * @author Paul Snow
 *
 */
public class Group {
    final private String              name;
    final private String              type;
    final private Map<String,String>  attribs;
    final private Map<String,Label>   labels          = new HashMap<String,Label>();
    final private Map<String,Label>   allLabels       = new HashMap<String,Label>();
    final private Map<String,Label>   bySpecs         = new HashMap<String,Label>();
    final private IDataSource         dataSource;
    final private IDataTarget         dataTarget;
    // This map holds the specifications for objects that should be ignored.  When
    // an object is pruned, it is pruned from every reference.  We can scope pruning
    // later if we need it.  For now, the complexity of managing scoped pruning is
    // slowing development.
    private Map<String,Object>        prune        = new HashMap<String, Object>();
    

    public boolean isPruned(String spec){
        return prune.containsKey(spec);
    }

    public void Prune(String spec){
        prune.put(spec, spec);
    }
    
    public String toString(){
        return name;
    }
    
    Group (AutoDataMapDef autoDataMapDef, String name, String type, Map<String,String> attribs){
        this.name           = name;
        this.type           = type;
        this.attribs        = attribs;
        this.dataSource     = autoDataMapDef.getDataSource(type);
        this.dataTarget     = autoDataMapDef.getDataTarget(type);
     }
    
    /**
     * @return the name
     */
    public String getName() {
        return name;
    }

    /**
     * @return the type
     */
    public String getType() {
        return type;
    }

    /**
     * @return the dataSource
     */
    public IDataSource getDataSource() {
        return dataSource;
    }

    /**
     * @return the dataTarget
     */
    public IDataTarget getDataTarget() {
        return dataTarget;
    }
    

    protected Map<String, Label> getBySpecs() {
        return bySpecs;
    }


    protected Map<String, Label> getLabels() {
        return labels;
    }


    /**
     * @return the allLabels
     */
    public Map<String, Label> getAllLabels() {
        return allLabels;
    }

    /**
     * @return the attribs
     */
    public Map<String, String> getAttribs() {
        return attribs;
    }

    /**
     * Find the first Label with this name.  You can use the function Label.next() to
     * get the next Label with the same name (as they can differ by object sources (i.e.
     * by spec).
     * 
     * @param label
     * @return
     */
    public Label findLabel(String label){
        Label r = getLabels().get(label);
        return r;
    }
    
    /**
     * Find the first Label with the given spec.  You can use the function Label.nextSpec()
     * to get the next Label with the same object source (i.e. the same spec) as the same
     * object can be mapped under different Label names.
     * 
     * @param spec
     * @return
     */
    public Label findLabelBySpec(String spec){
        Label r = getBySpecs().get(spec);
        return r;
    }
    
    /**
     * Find an exact match for this label.
     * @param name
     * @param type
     * @param spec
     * @return
     */
    public Label findLabel(String name, String group, String spec){
        Label label = findLabel(name);
        while(label != null 
                && !label.getGroup().equals(group)
                && !label.getSpec().equals(spec)){
            label = label.getNext();
        }
        return label;
    }
    
    
    public void printXML(AutoDataMapDef autoDataMapDef,XMLPrinter xout){
        xout.opentag("group","name",name,"type",type);
            Set<String> keys = prune.keySet();
            for(String key : keys){
                xout.printdata("prune","spec",key,null);
            }
        
            keys = labels.keySet();
            for(String key : keys){
                Label label = labels.get(key);
                label.printXML(autoDataMapDef, xout);
            }
        xout.closetag();
    }
    
}
