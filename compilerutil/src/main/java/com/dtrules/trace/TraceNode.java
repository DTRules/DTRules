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

package com.dtrules.trace;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

import com.dtrules.entity.IREntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.IRObject;
import com.dtrules.interpreter.RArray;
import com.dtrules.interpreter.RName;
import com.dtrules.interpreter.RNull;
import com.dtrules.session.DTState;
import com.dtrules.xmlparser.XMLPrinter;
/**
 * Holds the information in a trace file.  Provides accessors that 
 * return interesting information about the trace data.
 * 
 * @author paul
 *
 */
public class TraceNode {
	protected int 					number;
	protected String 				name;
	protected Map<String, String> 	attributes;
	protected List<TraceNode> 		children = new ArrayList<TraceNode>();
	protected String 				body;
	protected TraceNode             parent = null;
	
	TraceNode(int number, String name, Map<String,String> attributes){
		this.number     = number;
		this.name 		= name;
		this.attributes = attributes;
	}
	
	/**
	 * As I load the trace in, I give each node a number.  If you know
	 * very little about the trace, you can find a particular node by
	 * examinging the XML, and picking a number.
	 * 
	 * @param n
	 * @return TraceNode
	 */
	public TraceNode find(int n){
		if(number == n)return this;
		for(TraceNode child :children){
			TraceNode tn = child.find(n);
			if(tn!= null){
				return tn;
			}
		}
		return null;
	}
	
	/**
	 * Prints the data loaded from a Trace.  Really just a debugging feature.
	 * The given XMLPrinter is used for the output.
	 * @param out
	 */
	public void print(XMLPrinter out) { 
		attributes.put("t_num", ""+number);
		if(children.size()>0){
			out.opentag(name, attributes);
			for(TraceNode node : children){
				node.print(out);
			}
			out.closetag();
		}else{
			out.printdata(name, attributes, body);
		}
	}

	/**
	 * Parses an entity out of the trace info.
	 * @param trace
	 * @param session
	 * @return
	 * @throws RulesException
	 */
	IREntity getEntity(Trace trace) throws RulesException {
		String   id = attributes.get("id");
		String   n  = attributes.get("entity");
		IREntity e  = trace.createEntity(id, n);
		return e;
	}
	
	/**
	 * Returns true when it finds this node in the state tree, and has
	 * set the session to reflect that state.
	 * @param trace
	 * @param session
	 * @param position
	 * @return
	 * @throws RulesException
	 */
	public boolean setState(
			Trace 		trace, 
			TraceNode 	position) throws RulesException {

		DTState ds = trace.session.getState();
		
		// Found our position.  We are Done!
		if(position == this) return true;
		
		// Keep track of where we actually execute a table.  This will happen even
		// if we don't actually end up executing a column.
		if(name.equals("execute_table")){
		    trace.execute_table = this;
		}
		// We are an entitypush.  Find that entity and push it.
		if(name.equals("entitypush")){
			ds.entitypush(getEntity(trace));
		}
		
		// We are an entitypop.  Pop an entity!
		if(name.equals("entitypop")){
			ds.entitypop();
		}
		
		// If we are setting an attribute
		if(name.equals("def")){
            IRObject v;
			String   name = attributes.get("name");
			
			if(body.length()==0){
			    v = RNull.getRNull();
			}else{
	            trace.session.execute(body);
		        v = ds.datapop();
			}

			IREntity    e    = getEntity(trace);
			RName       rn   = RName.getRName(name);
			e.put(trace.session, rn, v);
			
			Change c = new Change(e, rn, trace.execute_table);
			
			trace.changes.put(c, c);     // Keep a hash lookup of my change object.
			
		}
	
		// Creating an array
		if(name.equals("newarray")){
		    int id = Integer.parseInt(attributes.get("arrayId"));
		    if(!trace.arraytable.containsKey(id)){
		        trace.arraytable.put(id, RArray.newArrayTraceInterface(id, true, false));
		    }
		}

		// Adding to an array
		if(name.equals("addto")){
		    int    id = Integer.parseInt(attributes.get("arrayId"));
            RArray ar = trace.arraytable.get(id);
            if(ar==null){   // Now this shouldn't happen, but if it does, create the array
                ar = RArray.newArrayTraceInterface(id, true, false);
                trace.arraytable.put(id, ar);
            }
            IRObject v;
            
            if(body.length()==0){
                v = RNull.getRNull();
            }else{
                trace.session.execute(body);
                v = ds.datapop();
            }

            ar.add(v);
		}
		
		// Remove a value from an array
		if(name.equals("remove")){
		    int    id = Integer.parseInt(attributes.get("arrayId"));
            RArray ar = trace.arraytable.get(id);
            if(ar==null){   // Now this shouldn't happen, but if it does, create the array
                ar = RArray.newArrayTraceInterface(id, true, false);
                trace.arraytable.put(id, ar);
            }
            IRObject v;
            
            if(body.length()==0){
                v = RNull.getRNull();
            }else{
                trace.session.execute(body);
                v = ds.datapop();
            }
            ds.datapush(ar);
            ds.datapush(v);
            ds.getSession().execute("remove");
            ds.datapop();
		}
		
	      // Remove a value from an array at a location
        if(name.equals("remove")){
            int    id = Integer.parseInt(attributes.get("arrayId"));
            RArray ar = trace.arraytable.get(id);
            if(ar==null){   // Now this shouldn't happen, but if it does, create the array
                ar = RArray.newArrayTraceInterface(id, true, false);
                trace.arraytable.put(id, ar);
            }
            trace.session.execute(body);
            IRObject i = ds.datapop();
            
            ds.datapush(ar);
            ds.datapush(i);
            ds.getSession().execute("removeat");
            ds.datapop();
        }
		
		
		for(TraceNode child : children){
			if(child.setState(trace, position)){
				return true;
			}
		}
				
		return false;
	}	
	
	/**
	 * Recursive search for all entities up to and including the given position.  All
	 * entities found are added to the entityList.
	 * @param position
	 * @param entityName
	 * @param entityList
	 * @return boolean true if done with search.
	 */
	public boolean searchTree( Trace trace, String entityName, List<IREntity> entityList) throws RulesException {
		
		if(name.equals("createentity")){
			if(attributes.get("name").equalsIgnoreCase(entityName)){
				String id = attributes.get("id");
				IREntity e = trace.createEntity(id, entityName);
				
				if(!entityList.contains(e)){
				   entityList.add(e);
				}
			}
		}
		
		if(this.equals(trace.position)) return true;
		
		for(TraceNode child : children){
			if(child.searchTree(trace, entityName, entityList)){
				return true;
			}
		}
		return false;
	}

	/**
	 * Make a node print something useful.
	 * @return String
	 */
	@Override
	public String toString() {
		return name + " (" + body + ") " + attributes;
	}

	/**
	 * Finds the actions executed from the given position in the trace.
	 * First I look at the current node, and its parents, until I find
	 * a column tag.  then I return the actions fired under that column.
	 * 
	 * The alternatives are to return an empty list if the position 
	 * handed to me is not a column tag, or look down the list of tags 
	 * until I find a column.  
	 * 
	 * @param trace
	 * @return
	 */
	public List<Integer> getActions(){
		List<Integer> actions = null;
		
		// Ah! we have our column!  Return its action children!
		if(name.equals("column")){
			actions = new ArrayList<Integer>();
			for(TraceNode child : children){
				if(child.name.equals("action")){
					String n = child.getAttributes().get("n");
					try { // Ignore bad columns
						actions.add(Integer.parseInt(n));
					} catch (NumberFormatException e) {}
				}
			}
			
		// Okay, this isn't the column.  Look to our parents?	
		}else if ( parent!= null ){
			actions = getParent().getActions();
		}
		
		// We only return a null of there is no column position to be found.
		return actions;
	}
	
	/** 
	 * ================================================================
	 * The following methods are just accessor methods.  If Java was a
	 * better language, these would be available for all properties of 
	 * a class, by default.  But it isn't.
	 * ================================================================
	 * 
	 */
	
	
	/**
	 * 
	 * @return TraceNode
	 */
	public TraceNode getParent() {
		return parent;
	}

	
	/**
	 * 
	 * @param parent
	 */
	public void setParent(TraceNode parent) {
		this.parent = parent;
	}
	
	/**
	 * 
	 * @param child
	 */
	public void addChild(TraceNode child){
		children.add(child);
	}
	
	/**
	 * 
	 * @return List<TraceNode> 
	 */
	public List<TraceNode> getChildren(){
		return children;
	}

	/**
	 * 
	 * @param attributes
	 */
	public void setAttributes(Map<String,String> attributes){
		this.attributes = attributes;
	}
	
	/**
	 * 
	 * @return  Map<String,String>
	 */
	public Map<String,String> getAttributes(){
		return attributes;
	}

	/**
	 * 
	 * @return String
	 */
	public String getBody() {
		return body;
	}
	
	/**
	 * 
	 * @return String
	 */
	public String getName() {
		return name;
	}

	/**
	 * 
	 * @param body
	 */
	public void setBody(String body) {
		this.body = body;
	}
}
