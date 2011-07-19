package com.dtrules.trace;

import java.io.IOException;
import java.util.HashMap;
import java.util.Stack;

import com.dtrules.xmlparser.AGenericXMLParser;


public class TraceLoader extends AGenericXMLParser {
		
	Stack<TraceNode> tagStack = new Stack<TraceNode>();

	@Override
	public void beginTag(String[] tagstk, int tagstkptr, String tag,
			HashMap<String, String> attribs) throws IOException, Exception {
			TraceNode thisNode = new TraceNode(tag,attribs);
			if(tagStack.size()>0){
			   tagStack.lastElement().addChild(thisNode);
			}
			tagStack.push(thisNode);
	}

	@Override
	public void endTag(String[] tagstk, int tagstkptr, String tag, String body,
			HashMap<String, String> attribs) throws Exception, IOException {
	
		tagStack.pop().setBody(body);

	}
	
	
}
