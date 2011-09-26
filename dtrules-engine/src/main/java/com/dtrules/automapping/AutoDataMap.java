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

import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Stack;

import com.dtrules.automapping.access.IAttribute;
import com.dtrules.automapping.access.IDataSource;
import com.dtrules.automapping.access.JavaAttribute;
import com.dtrules.automapping.nodes.IMapNode;
import com.dtrules.automapping.nodes.MapNodeAttribute;
import com.dtrules.automapping.nodes.MapNodeList;
import com.dtrules.automapping.nodes.MapNodeMap;
import com.dtrules.automapping.nodes.MapNodeObject;
import com.dtrules.automapping.nodes.MapNodeRef;
import com.dtrules.entity.IREntity;
import com.dtrules.mapping.Mapping;
import com.dtrules.session.IRSession;
import com.dtrules.xmlparser.XMLPrinter;
import com.dtrules.xmlparser.XMLTree;
import com.dtrules.xmlparser.XMLTree.Node;

/**
 * The AutoDataMap interface is best used when mapping Java objects into DTRules.  There
 * are a few assumptions made about these Java objects that you need to keep in mind.
 * <br><br>
 *        !!!NOTE IF YOU ARE LOSING DATA!!!! ... Data you know to be in your 
 *        Java objects, but doesn't get written to your dataload files, or 
 *        doesn't show up in your Rule Set.
 * <br><br>
 * First of all, we look at the accessors to find attributes of a Java object.  If your
 * Java object doesn't have accessors defined, then this code isn't going to work for 
 * you.
 * <br><br>
 * Second of all, all the accessors and even your class needs to be public.  If the 
 * class isn't public (NOT just the accessors, but the class itself!!!) we are not going 
 * to throw any errors, but we are not going to access any of your class' attributes 
 * either.  
 * <br><br>
 * On the other hand, if your Java Objects don't really line up with the Entities and the
 * attributes in your Rule Sets, or if you primarily will be consuming XML that is 
 * generated to a spec that is independent of your Rule Set, the older DataMap object
 * is a better way to go.
 * <br><br>
 * The AutoDataMap also handles the writing of data back out to your Java Objects, and
 * is intended to also handle the transport of data between Java objects.  This latter
 * functionality needs more development at the time of this writing, but the idea is to
 * provide the same sort of abstraction of functionality to Java code that the Rule
 * Sets enjoy, which might even go so far as to include the support for automated testing,
 * management of expected test results, etc. that can be done with Rule Sets.
 *    
 * @author Paul Snow
 *
 */
public class AutoDataMap {
    
    private final AutoDataMapDef      autoDataMapDef;           // Holds the state shared between sessions
                                                                //   for a mapping definition.
    
    // This is the root of the objects being mapped.  The root is an array, as
    // we may map many objects into the DataMap.  The order in which they are
    // applied matters, as we do allow data to be duplicated, and thus when
    // mapped to the target, overwritten.
    List<IMapNode>                    dataMap      = new ArrayList<IMapNode>();
       
    List<String>                      groupTypes   = new ArrayList<String>();
    
    private IRSession                 session;                  // If mapping to a Rule Set, we have
                                                                //   to have the Rules session.
    private Map<String,IREntity>      entities     = new HashMap<String,IREntity>(); // List of entities created.

    private Map<Object,MapNodeObject> written      = new HashMap<Object,MapNodeObject>();
    
    /**
     * used when mapping data from the Data Map to a target data source.  While
     * you might have many sources for data, you will only have one Group (the 
     * source group) while loading data into the AutoDataMap, or one Group 
     * (the target group) while mapping data out of the AutoDataMap. 
     */
    private Group                     currentGroup  = null;               
    
    private Stack<Object>             mapStack      = new Stack<Object>();
        
    final private static String       mark          = "mark";
    
    private Stack<IMapNode>           mapNodeStack  = new Stack<IMapNode>();
    
    
    
    public IMapNode getParent(){
        if(mapNodeStack.size()==0)return null;
        return mapNodeStack.peek();
    }
    
    public IMapNode popMapNode(){
        return mapNodeStack.pop();
    }
    
    public void pushMapNode(IMapNode mapNode){
        mapNodeStack.push(mapNode);
    }
    
    
    public AutoDataMap(IRSession session, AutoDataMapDef admd){
        this.autoDataMapDef = admd;
    }
    
    public AutoDataMap(IRSession session, AutoDataMapDef admd, Mapping mapping, String tag, OutputStream stream){
        this.autoDataMapDef = admd;
    }
        
    /**
     * The pushMark() pushes an invalid object that can be placed on the mapStack
     * when a non-mapping object in the autoDataMap to the target is encountered.
     */
    public void pushMark() { mapStack.push(mark); }
    /**
     * get the current Object (leaves the Object stack alone).  Returns null
     * if the stack is empty or the top of stack is a mark() value.
     * @return
     */
    public Object getCurrentObject() {
        if(mapStack.size()==0)return null;
        Object object = mapStack.peek();
        if(object == mark)return null;
        return object;
    }
    /**
     * Push an object onto the stack.
     * @param object
     */
    public void push(Object object){
        mapStack.push(object);
    }
    /**
     * Pop an object off the stack and return it.  If the object is
     * a mark, the mark isn't returned, but a null is returned instead.
     * @return
     */
    public Object pop(){
        if(mapStack.size()==0) return null;
        Object object = mapStack.pop();
        if(object == mark)return null;
        return object;
    }
    
    /**
     * Simple parsing of text for a boolean; returns false if passed in a null,
     * and returns a null if passed text that isn't understood.
     * 
     * @param s
     * @return
     */
    static public Boolean parseBoolean(String s) {
        if (s == null)
            return false;
        if (s.equalsIgnoreCase("true"))
            return true;
        if (s.equalsIgnoreCase("false"))
            return false;
        return null;
    }

    /**
     * Trims whitespace from inputs, and converts null length strings to nulls.
     * 
     * @param a
     * @return
     */
    static public String fix(String a) {
        if (a == null)
            return null;
        a = a.trim();
        if (a.length() == 0)
            return null;
        return a;
    }


    /**
     * Sorts an Object array that points to an array of strings. Needed to parse
     * key sets from Array Lists. Bubble sort, so should only be used for short
     * arrays of data, or none runtime purposes. We use it to print.
     * 
     * @param keys
     */
    public static void sort(Object[] keys) {
        for (int i = 0; i < keys.length - 1; i++) {
            for (int j = i; j < keys.length - 1; j++) {
                if (keys[j].toString().compareTo(keys[j + 1].toString()) > 0) {
                    Object h = keys[j];
                    keys[j] = keys[j + 1];
                    keys[j + 1] = h;
                }
            }
        }
    }
        
    /**
     * Searches for the first component (Attribute or object)
     * with the given name.  Starts the search at the given node.  This
     * lets you find multiple objects with the same name.
     * 
     * Returns a null if none is found.
     * 
     * @param start
     * @param name
     * @return
     */
    public IMapNode find (IMapNode start, String name){
    	boolean searching = false;		// The way this works is we run through the
    	if(start == null) {				// DataMap until we find the starting point.
    		searching = true;			// (a null node means start at the start).
    	}								// Then we return the first match after that.
    
    	for (IMapNode node : dataMap){
    		IMapNode result = find(node, start, searching, name);
    		if(result != null ){
    			if(result == start){
    				searching = true;
    			}else{
        			return result;
    			}
    		}
    	}
    	return null;
    }
    
    private IMapNode find(IMapNode node, IMapNode start, boolean searching, String name){
    	if(node.getName().equals(name)){
    		if(searching){
    			return node;
    		}else{
    			if (node == start){
    				searching = true;
    			}
    		}
    	}
    	for (IMapNode child : node.getChildren()){
    		IMapNode result = find(child,start,searching,name);
    		if(result != null ){
    			if(result == start){
    				searching = true;
    			}else{
        			return result;
    			}
    		}
    	}
    	return searching ? start : null;
    }
    
    
    /**
     * @return the dataMap
     */
    public List<IMapNode> getDataMap() {
        return dataMap;
    }
    /**
     * @param dataMap the dataMap to set
     */
    public void setDataMap(List<IMapNode> dataMap) {
        this.dataMap = dataMap;
    }
    /**
     * @return the autoDataMapDef
     */
    public AutoDataMapDef getAutoDataMapDef() {
        return autoDataMapDef;
    }
    
    /**
     * @return the session
     */
    public IRSession getSession() {
        return session;
    }
    /**
     * @return the entities
     */
    public Map<String, IREntity> getEntities() {
        return entities;
    }

        
    public void setSession(IRSession session ){
        this.session = session;
    }

    /**
     * @return the currentGroup
     */
    public Group getCurrentGroup() {
        return currentGroup;
    }

    /**
     * @param currentGroup the currentGroup to set
     */
    public void setCurrentGroup(Group currentGroup) {
        this.currentGroup = currentGroup;
    }

    /**
     * Set the current group by looking up the name in the autoDataMapDef
     * @param currentGroup
     */
    public void setCurrentGroup(String currentGroup){
        this.currentGroup = autoDataMapDef.findGroup(currentGroup);
    }
    
    /**
     * Loads the given object into the label (identifying where in the parent
     * object the reference to this object should be attached).  You can pass
     * in a null for the label, in which case your object will be loaded as 
     * a root object.  (This works just fine for objects which have keys, or 
     * are singular and are referenced elsewhere in the structure built in the
     * Rules Engine.)
     * @param label
     * @param object
     */
    public void loadObjects( String label, Object object){
        loadObjects(getParent(), currentGroup, label, object);
    }

    /**
     * Push a Label and allow value to be added to the Label
     * @param label
     * @param key
     */
    public void pushLabel(String label, Object key){
        Label labelObj = Label.newLabel(currentGroup,label,label,label+"Id",null);
        MapNodeObject mno = new MapNodeObject(currentGroup, labelObj, getParent());
        mno.setKey(key);
        IMapNode parent = getParent();
        if(parent == null){
            dataMap.add(mno);
        }
        mapNodeStack.push(mno);
    }
    
    public void pushLabel(IMapNode mapNode){
    	mapNodeStack.push(mapNode);
    }
    
    /**
     * Pop the top Label off the Label Stack
     * @return
     */
    public IMapNode popLabel(){
        return mapNodeStack.pop();
    }
    
    /**
     * Sets a value in the DataMap on an attribute that doesn't necessarily exist on the actual
     * object.  If the value DOES exist on the actual object, then the last value set is the one
     * that will get mapped to the target (i.e. if an object has a current date value, and you 
     * set it with a setValue() call, then load the object, then the value on the object will 
     * overwrite (in a sense) the value you gave in the setValue() call.  If you made the setValue()
     * call last (i.e. after you load the object), then the setValue() call will overwrite the
     * value provided by the object.
     * 
     * @param labelname     The labelname for the object where the attribute should be written
     * @param key           If the label provides a key, then this is the value of that key
     * @param type          The attribute type;  Needed for producing a valid dataload file.
     * @param attributeName The attribute name.
     * @param value         The value for the attribute.
     */
    public void setValue(String attributeName, String type, Object value){
        IMapNode          parent    = getParent();
        if(parent == null || !(parent instanceof MapNodeObject)){
            throw new RuntimeException("Attributes must have Objects to hold them");
        }
        String            labelname = ((MapNodeObject)parent).getLabel();
        new MapNodeAttribute(parent,labelname,type,attributeName,value);
    }
    
    /**
     * Sets a list in the DataMap on an attribute that doesn't necessarily exist on the actual
     * object.  If a list DOES exist, then whatever you do last will be what sticks, from the
     * Rules Engine point of view.
     * 
     * Make sure the object you wish to insert the list upon is the "current node" in the 
     * AutoDataMap state.
     * 
     * @param listname
     * @param list
     * @param subtype
     */
	public void setList(String listname, List<Object> list, String subtype){
    	try {
			MapType       mapType   = MapType.get(subtype);
			MapNodeObject parent    = (MapNodeObject) getParent();
			Label         labelObj  = parent.getSourceLabel();
			
			IAttribute a = labelObj.getAttribute(listname);
    
			if(a == null){
			    a = JavaAttribute.newAttribute(
			    		labelObj, 
			    		listname, 
			    		null, 
			    		null, 
			    		Class.forName(subtype), 
			    		MapType.get("list"),
			    		"list",
			    		mapType,
			    		subtype
			    		);
			    
			    labelObj.getAttributes().add(a);
			}
			
			MapNodeList mnl = new MapNodeList(a, parent);
			
			if(mapType.isPrimitive()){
			    mnl.setSubType(a.getSubTypeText());
			    mnl.setList(list);
			}else {
			    mnl.setSubType(a.getSubTypeText());
			    mnl.setList(list);
			    if( list!=null ) for(Object obj2 : list){
			        loadObjects(mnl, currentGroup,(String) null, obj2);
			    }
			}
		} catch (ClassNotFoundException e) {
			throw new RuntimeException("subtype class not found: "+subtype);
		}
    }
    
    @SuppressWarnings("unchecked")
    private void loadObjects(IMapNode parent,  Group groupObj, String label, Object object){
        
        IDataSource dataSrc   = groupObj.getDataSource();
        
        String spec = dataSrc.getSpec(object);
        
        boolean pruned = groupObj.isPruned(spec); 
        if(pruned) return;
        
        if(object instanceof List){
            for(Object child : ((List<Object>)object)){
                loadObjects(parent,groupObj,dataSrc.getName(child),child);
            }
            return;
        }
        
        // Look up that label.
        Label labelObj = groupObj.findLabel(
                label,
                groupObj.getType(),
                spec);
        
        // If we haven't got a label for this sort of object yet, create one.
        if (labelObj == null  || !labelObj.isCached()){
        
            labelObj = dataSrc.createLabel(
                this,
                groupObj,
                dataSrc.getName(object),           // Get our default label name
                dataSrc.getKey(object),            // See if we can find a key for this obj                  
                false,
                object);
        }
        
        // If we still don't have a LabelObj, then just process its children.
        if(labelObj == null){
            for(Object child : dataSrc.getChildren(object) ){
                loadObjects(parent, groupObj, dataSrc.getName(child), child);
            }
            return;
        }
        loadObjects(parent, groupObj, labelObj, object);
    }
    
    @SuppressWarnings("unchecked")
	private void loadObjects(IMapNode parent,  Group groupObj, Label labelObj, Object object){
        
    	IDataSource dataSrc   = groupObj.getDataSource();
        
    	MapNodeObject node = new MapNodeObject(groupObj, labelObj, parent);
        node.setSource(object);
        
        node.setKey(dataSrc.getKeyValue(node,object));
        
        if(parent==null){
            getDataMap().add(node);
        }
        
        // If we have written this object already, then we don't need to repeat its
        // attributes.  But we do need the key from the previous write!
        if(written.get(object)!=null){
            node.setKey(written.get(object).getKey()); 
        }else{
            written.put(object,node);
            for(IAttribute a : labelObj.getAttributes()){
                Object r = a.get(object);
                if(a.getType().isPrimitive()){
                    MapNodeAttribute mna = new MapNodeAttribute(a, node);
                    mna.setData(r);
                    if(a.isKey()){
                        node.setKey(r);
                    }
                }else{
                    if(a.getType() == MapType.LIST 
                            && a.getSubType().isPrimitive()){
                        MapNodeList mnl = new MapNodeList(a, node);
                        mnl.setSubType(a.getSubTypeText());
                        mnl.setList((List<Object>)r);
                    }else if(a.getType()== MapType.LIST){
                        MapNodeList mnl = new MapNodeList(a,node);
                        mnl.setSubType(a.getSubTypeText());
                        mnl.setList((List<Object>)r);
                        if( r!=null ) for(Object obj2 : (List<Object>)r){
                            loadObjects(mnl, groupObj,(String) null, obj2);
                        }
                    }else if (a.getType() == MapType.MAP){
                        MapNodeMap mnm = new MapNodeMap(a,node);
                        mnm.setMap((Map<Object,Object>)r);
                    }else if (a.getType() == MapType.OBJECT){
                        if(r!=null) if(groupObj.isPruned(dataSrc.getSpec(r))) continue;
                        MapNodeRef mnr = new MapNodeRef (a, node);
                        if(r!=null) loadObjects(mnr, groupObj, (String) null, r);
                    }
                }
            }
        }
    }

    /**
     * In order to map the data to a target, we need to know the group
     * @param groupName
     * @param entryPoint
     */
    public void mapDataToTarget(String groupName){
        currentGroup = autoDataMapDef.findGroup(groupName);
        mapDataToTarget();
    }
        
    public void mapDataToTarget(){
        currentGroup.getDataTarget().init(this);
        for(IMapNode node : dataMap){
            node.mapNode(this, null);
        }
    }
    
    /**
     * Pushes back changes made in the rules engine back into objects
     * from the source.  If you want the changes made in the RulesEngine
     * to be propagated back to the source objects, you need to call 
     * this method after having executed your RuleSet.
     */
    public void update(){
        for(IMapNode node : dataMap){
            node.update(this);
        }
    }
    
    public void printDataLoadXML(String dataloadFilename) {
        try{
            OutputStream fout = new FileOutputStream(dataloadFilename);
            printDataLoadXML(fout);
            fout.close();
        }catch(IOException e){
            throw new RuntimeException(e.toString());
        }
    }
    
    /**
     * Prints the current state of the AutoDataMap as a single configuration file.
     * @param fstream
     */
    public void printDataLoadXML(OutputStream fstream) {
        XMLPrinter xout = new XMLPrinter("dataload",fstream);
        
        for(IMapNode mno : dataMap){
            mno.printDataLoadXML(this, xout);
        }
        
        xout.close();
    }
    
    public void print(OutputStream out) {
        printDataLoadXML(out);
    }

    public void LoadXML(String dataloadFilename){
        try{
            InputStream fin = new FileInputStream(dataloadFilename);
            LoadXML(fin);
            fin.close();
        }catch(IOException e){
            throw new RuntimeException(e.toString());
        }
    }
    
    public void LoadXML(InputStream in){
        try {
            Node root = XMLTree.BuildTree(in, false, false);
            loadObjects(null,root);            
        } catch (Exception e) {
            e.printStackTrace();
        }
    }
}

