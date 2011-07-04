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

import java.io.File;
import java.io.FileOutputStream;
import java.io.FileReader;
import java.io.IOException;
import java.io.InputStream;
import java.io.Reader;
import java.util.Date;
import java.util.HashMap;
import java.util.Map;
import java.util.Set;

import com.dtrules.automapping.access.DTRulesTarget;
import com.dtrules.automapping.access.IDataSource;
import com.dtrules.automapping.access.IDataTarget;
import com.dtrules.automapping.access.JavaSource;
import com.dtrules.automapping.access.XMLSource;
import com.dtrules.session.DateParser;
import com.dtrules.session.IDateParser;
import com.dtrules.session.IRSession;
import com.dtrules.xmlparser.AGenericXMLParser;
import com.dtrules.xmlparser.GenericXMLParser;
import com.dtrules.xmlparser.XMLPrinter;

/**
 * @author paul snow
 *
 */
public class AutoDataMapDef {

    // These are our supported Object types.  The basic two are Java and DTRules.
    // However, we want to simulate both of these using XML
    private Map<String,Label>       labels          = new HashMap<String,Label>();
    
    private Map<String,Group>       groups          = new HashMap<String,Group>();
   
    private Map<String,IDataSource> dataSources     = new HashMap<String,IDataSource>();
    
    private Map<String,IDataTarget> dataTargets     = new HashMap<String,IDataTarget>();
    
    private Map<String,LabelMap>    labelMaps       = new HashMap<String,LabelMap>();
     
    private static IDateParser      dateParser      = new DateParser();
    
    /**
     * No type exists at the AutoDataMapDef level.  Return a null.
     */
    public String getType() { return null; }
    
    public AutoDataMapDef(){
        dataSources.put("java",    new JavaSource(this));
        dataSources.put("xml",     new XMLSource(this));
        dataTargets.put("dtrules", new DTRulesTarget(this));
    }

    public IDataSource getDataSource(String dataSource){
        return dataSources.get(dataSource);
    }
    
    public IDataTarget getDataTarget(String dataTarget){
        return dataTargets.get(dataTarget);
    }
        
    public Group findGroup (String group){
        Group g = groups.get(group);
        return g;
    }
    
    /**
     * Create a new group (or return the existing one)
     * @return the new group
     */
    public Group newGroup(String group, String type, Map<String,String> attribs) {
        Group g = findGroup(group);
        if(g!=null){
            if(!g.getType().equals(type)){
                throw new RuntimeException("Group '"+group+"' is defined with multiple types: "+
                        "'"+type+"' '"+g.getType()+"'");
            }
            return g;
        }
        g = new Group(this, group, type, attribs);
        groups.put(group, g);
        return g;
    }

    public static Boolean parseBoolean(String s){
        if(s == null) return false;
        if(s.equalsIgnoreCase("true"))return true;
        if(s.equalsIgnoreCase("false"))return false;
        return null;
    }

    public LabelMap findLabelMap(String source, String target){
        LabelMap labelMap = labelMaps.get(source);
        while(labelMap != null && !labelMap.getTarget().equals(target)){
            labelMap = labelMap.getNext();
        }
        return labelMap;
    }    
    
    public LabelMap findLabelMap(String source){
        LabelMap labelMap = labelMaps.get(source);
        return labelMap;
    }
    
    public void addLabelMap(LabelMap labelMap){
        LabelMap existing = labelMaps.get(labelMap.getSource()); 
        labelMap.setNext(existing);
        labelMaps.put(labelMap.getSource(), labelMap);
    }
    /**
     * @return the groups
     */
    public Map<String, Group> getGroups() {
        return groups;
    }

    /**
     * Build an AutoDataMap.  In the future, we may want to do some intialization
     * here.
     * 
     */
    public AutoDataMap newAutoDataMap(IRSession session) {
        AutoDataMap adm         = new AutoDataMap(session, this);         
        return adm;
    }

   
    /**************************************************************
     *   In the following section of code, we load the initial 
     *   AutoDataMapDef State.
     **************************************************************/
    Group    currentGroup;
    Label    currentLabel;
    LabelMap currentLabelMap;
    
    class SpecLoader extends AGenericXMLParser {
        /**
         * Parse tag beginnings.  The actual work is done in methods in the
         * AutoDataMapDef class, but the parser does maintain the current state
         * for AutoDataMapDef.
         */
        @Override
        public void beginTag(String[] tagstk, int tagstkptr, String tag,
                HashMap<String, String> attribs) throws IOException, Exception {
            if(tag.equals("prune")){
                prune(attribs);
            }else if (tag.equals("group")){
                currentGroup = createGroup(attribs);
            }else if (tag.equals("object")){
                currentLabel = createLabel(attribs);
            }else if (tag.equals("mapobject")){
                currentLabelMap = createLabelMap(attribs);
            }else if (tag.equals("mapattribute")){
                mapAttribute(attribs);
            }
        }
        
        /**
         * We don't allow nesting in the specification, mostly because it isn't
         * any use to us.  So on the end tag of various nested sorts of tags,
         * we clear the state that this tag set.
         */
        @Override
        public void endTag(String[] tagstk, int tagstkptr, String tag,
                String body, HashMap<String, String> attribs) throws Exception,
                IOException {
            if(tag.equals("group")){
                currentGroup = null;
            }else if (tag.equals("object")){
                currentLabel = null; 
            }else if (tag.equals("mapobject")){
                currentLabelMap = null;
            }
        }
    }
    /**
     * Prune the given object at the level of the currentContext
     * @param attribs
     */
    private void     prune(Map<String,String> attribs){
        String spec = attribs.get("spec").trim();
        
        if(currentGroup == null){
            throw new RuntimeException("Can only prune within groups");
        }
        
        if(spec == null){
            throw new RuntimeException("Must specify an object to prune");
        }
        
        currentGroup.Prune(spec);
    }
    
    /**
     * Find the given Group, (possibly creating a new instance of the Group) and
     * return the reference.
     * 
     * @param attribs
     * @return
     */
    private Group    createGroup(Map<String,String> attribs){
        String name = attribs.get("name");
        String type = attribs.get("type");
        return createGroup(name,type,attribs);
    }
    
    public Group createGroup(String name, String type, Map<String,String> attribs){
        if(name==null || type == null ){
            throw new RuntimeException("Groups must specify a name and a type");
        }
        name = name.trim();
        type = type.trim();
        if(name.length()==0 || type.length()==0){
            throw new RuntimeException("Groups must specify a name and a type");            
        }
        Group g = newGroup(name, type, attribs);
        return g;
    }
    
    public Group findGroupByLabel(String labelname){
        Set<String> keys = groups.keySet();
        for(String key : keys){
            Group group = groups.get(key);
            if(!(group.findLabel(labelname)==null)){
                return group;
            }
        }
        return null;
    }
    /**
     * Create the given label (or return the existing one) which matches the
     * Label specifications.  To qualify as an existing label, the label has
     * to exist at the given context level.  The same Label can have different
     * specs (different source objects can define its values).
     * @param attribs
     * @return
     */
    private Label    createLabel(Map<String,String> attribs){
        Label  r = Label.newLabel(currentGroup, attribs);
        return r;
    }
    
    private void     mapAttribute(Map<String,String> attribs){
        if(currentLabelMap == null){
            throw new RuntimeException("<mapAttribute> can only be used within a <mapobject>");
        }
        String source = attribs.get("source");
        String target = attribs.get("target");
        currentLabelMap.getMapAttributes().put(source,target);
    }
    
    private LabelMap createLabelMap(Map<String,String> attribs){
        String source = attribs.get("source");
        String target = attribs.get("target");
        return createLabelMap(source, target);
    }
    
    public LabelMap createLabelMap(String source, String target){
        LabelMap root = labelMaps.get(source);
        
        LabelMap map;
        for(map = root; 
            map != null && !map.getTarget().equals(target);
            map = map.getNext());

        if(map != null){                        // If there is an existing map, just
            return map;                         //   return that one.
        }   
            
        map = new LabelMap(this,source,target); // Otherwise allocate one.
        map.setNext(root);                      // Link the new map into the head of list
        labelMaps.put(source,map);              //   and log it as the new root.
        
        return map;                             // return the new map.
    }
    
    
    /*******************************************************************************
     *      The methods above support loading the Initial AutoDataMapDef state.
     *******************************************************************************/
    
    /**
     * We do a conversion of primitive objects (int, double, long, etc.) in a fashion 
     * that allows us to serialize the primitive objects to XML
     * @param object
     * @return
     */
    public static String convert(Object object){
        if(object == null)return "";
        if(object instanceof Date){
        	return dateParser.getDateString((Date)object);
        }
        return object.toString();
    }           

    /**
     * Configure an AutoDataMap by parsing the given XML file.  The provided
     * session should be for the rule set mapped by the AutoDataMap, and that
     * rule set should be fully configured (i.e. all entity definitions loaded).
     * @param file
     */
    public void configure(InputStream stream) {
        try {
            SpecLoader loader = new SpecLoader();
            GenericXMLParser.load(stream, loader);
        } catch (Exception e) {
            throw new RuntimeException("Cannot configure AutoDataMap file");
        }
    }
    
    /**
     * Configure an AutoDataMap by parsing the given XML file.  The provided
     * session should be for the rule set mapped by the AutoDataMap, and that
     * rule set should be fully configured (i.e. all entity definitions loaded).
     * @param file
     */
    public void configure(File file) {
            try {
                Reader xmlReader = new FileReader(file);
                SpecLoader loader = new SpecLoader();
                GenericXMLParser.load(xmlReader, loader);
            } catch (Exception e) {
                throw new RuntimeException("Cannot configure AutoDataMap file");
            }
    }
    
    public void printLabelMapXML(AutoDataMapDef autoDataMapDef, XMLPrinter xout){
        String [] keys = labelMaps.keySet().toArray(new String[0]);
        AutoDataMap.sort(keys);
        xout.opentag("maps");
            for(String key : keys){
                labelMaps.get(key).printXML(xout);
            }
        xout.closetag();
    }

    
    /**
     * Prints the current state of the AutoDataMap as a single configuration file.
     * @param fstream
     */
    public void printMappingConfigurationXML(FileOutputStream fstream) {
        XMLPrinter xout = new XMLPrinter("config",fstream);
        printMappingConfigurationXML(xout);
    }

    /**
     * Prints the current state of the AutoDataMap as a single configuration file.
     * @param xout
     */
    public void printMappingConfigurationXML(XMLPrinter xout){    
        {
            Set<String> keys = labels.keySet();
            for(String key : keys){
                Label label = labels.get(key);
                label.printXML(this, xout);
            }
        }
        {
            Set<String> keys = groups.keySet();
            for(String key : keys){
                Group group = groups.get(key);
                group.printXML(this,xout);
            }
        }
        printLabelMapXML(this,xout);
    }
}
