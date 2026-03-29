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

import com.dtrules.xmlparser.XMLPrinter;

/**
 * Defines how objects from a given source should be mapped to a given target.
 * 
 * @author Paul Snow
 *
 */
public class LabelMap {
    final private String                  source;                 // The source label
    final private String                  target;                 // The target label
          private Group                   sourceGroup;
          private Group                   targetGroup;
    final private Map<String,String>      mapAttributes = new HashMap<String,String>();

    private LabelMap                      next = null;            // We keep our source labels linked!
   
    public String toString(){
        return source +"->"+target;
    }
    
    public LabelMap (AutoDataMapDef autoDataMapDef, 
            String source,      String target){
        this.source      = source;
        this.target      = target;
        this.sourceGroup = autoDataMapDef.findGroupByLabel(source);
        this.targetGroup = autoDataMapDef.findGroupByLabel(target);
        
        if(sourceGroup == null || targetGroup == null){
            String badones = (sourceGroup == null ? " '" + source + "' ":" ") + 
                             (targetGroup == null ? "'"  + target + "' ":" ");
            throw new RuntimeException("Undefined Labels(s):"+badones);
        }
    }
    
    public static LabelMap newLabelMap(AutoDataMapDef autoDataMapDef, Map<String,String> attribs){
        String             source = attribs.get("source");
        String             target = attribs.get("target");
        
        
        if(source == null || source.trim().length()==0){ 
            throw new RuntimeException("<map> missing source");
        }
        
        if(target == null || source.trim().length()==0){
            throw new RuntimeException("The map missing target");
        }
                 
        LabelMap labelMap = autoDataMapDef.findLabelMap(source,target);
        
        if(labelMap == null){
            labelMap = new LabelMap(autoDataMapDef,source,target);
            autoDataMapDef.addLabelMap(labelMap);
        }
        
        return labelMap;
    }

    public void attributeChanged (AutoDataMapDef autoMapDataDef, Map<String,String> attribs){
        String source = attribs.get("source");
        String target = attribs.get("target");
        mapAttributes.put(source, target);
    }
    

    /**
     * @return the next
     */
    public LabelMap getNext() {
        return next;
    }

    /**
     * @param next the next to set
     */
    public void setNext(LabelMap next) {
        this.next = next;
    }

    /**
     * @return the mapAttributes
     */
    public Map<String, String> getMapAttributes() {
        return mapAttributes;
    }

    /**
     * @return the source
     */
    public String getSource() {
        return source;
    }


    /**
     * @return the target
     */
    public String getTarget() {
        return target;
    }


    /**
     * @return the sourceGroup
     */
    public Group getSourceGroup() {
        return sourceGroup;
    }

    /**
     * @return the targetGroup
     */
    public Group getTargetGroup() {
        return targetGroup;
    }

    /**
     * @param sourceGroup the sourceGroup to set
     */
    public void setSourceGroup(Group sourceGroup) {
        this.sourceGroup = sourceGroup;
    }

    /**
     * @param targetGroup the targetGroup to set
     */
    public void setTargetGroup(Group targetGroup) {
        this.targetGroup = targetGroup;
    }

    public void printXML(XMLPrinter xout){
        xout.opentag("mapobject", 
                "source", source,
                "target", target);
            Object [] keys =  mapAttributes.keySet().toArray();
            AutoDataMap.sort(keys);
            for(Object key : keys){
                xout.printdata("mapname","source",key,"target",mapAttributes.get(key),null);
            }
        xout.closetag();
    }
    
}
