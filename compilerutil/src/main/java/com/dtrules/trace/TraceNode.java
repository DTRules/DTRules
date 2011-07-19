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
}
