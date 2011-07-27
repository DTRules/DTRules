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
import com.dtrules.interpreter.RName;
import com.dtrules.session.DTState;
import com.dtrules.session.IRSession;
import com.dtrules.xmlparser.XMLPrinter;
/**
 * Holds the information in a trace file.
 * 
 * @author paul
 *
 */
public class TraceNode {
	protected String 				name;
	protected Map<String, String> 	attributes;
	protected List<TraceNode> 		children = new ArrayList<TraceNode>();
	protected String 				body;
	protected TraceNode             parent = null;
	
	TraceNode(String name, Map<String,String> attributes){
		this.name 		= name;
		this.attributes = attributes;
	}
	
	public void addChild(TraceNode child){
		children.add(child);
	}
	
	public List<TraceNode> getChildren(){
		return children;
	}

	public void setAttributes(Map<String,String> attributes){
		this.attributes = attributes;
	}
	
	public Map<String,String> getAttributes(){
		return attributes;
	}

	public String getBody() {
		return body;
	}
	
	public String getName() {
		return name;
	}

	public void setBody(String body) {
		this.body = body;
	}
	
	public void print(XMLPrinter out) { 
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
	IREntity getEntity(Trace trace, IRSession session) throws RulesException {
		String   id = attributes.get("id");
		String   n  = attributes.get("entity");
		IREntity e  = trace.entitytable.get(id);
		if(e==null){
			e = session.createEntity(id, n);
			trace.entitytable.put(id, e);
		}
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
			IRSession 	session, 
			TraceNode 	position) throws RulesException {

		DTState ds = session.getState();
		
		// Found our position.  We are Done!
		if(position == this) return true;
		
		// We are an entitypush.  Find that entity and push it.
		if(name.equals("entitypush")){
			ds.entitypush(getEntity(trace,session));
		}
		
		// We are an entitypop.  Pop an entity!
		if(name.equals("entitypop")){
			ds.entitypop();
		}
		
		// If we are setting an attribute
		if(name.equals("def")){
			session.execute(body);
			String 		name = attributes.get("name");
			IRObject 	v	 = ds.datapop();
			IREntity    e    = getEntity(trace, session);
			e.put(session, RName.getRName(name), v);
		}
	
		
		
		
		for(TraceNode child : children){
			if(child.setState(trace, session, position)){
				return true;
			}
		}
				
		return false;
	}

	
	public TraceNode getParent() {
		return parent;
	}

	public void setParent(TraceNode parent) {
		this.parent = parent;
	}
	
	
}
