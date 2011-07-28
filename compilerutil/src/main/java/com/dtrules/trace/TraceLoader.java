package com.dtrules.trace;

import java.io.IOException;
import java.util.HashMap;
import java.util.Stack;

import com.dtrules.xmlparser.AGenericXMLParser;


public class TraceLoader extends AGenericXMLParser {
		
	int number = 1;
	
	Stack<TraceNode> tagStack = new Stack<TraceNode>();

	@Override
	public void beginTag(String[] tagstk, int tagstkptr, String tag,
			HashMap<String, String> attribs) throws IOException, Exception {
			TraceNode thisNode = new TraceNode(number, tag,attribs);
			number++;
			if(tagStack.size()>0){
			   tagStack.lastElement().addChild(thisNode);
			   thisNode.setParent(tagStack.lastElement());
			}
			tagStack.push(thisNode);
	}

	@Override
	public void endTag(String[] tagstk, int tagstkptr, String tag, String body,
			HashMap<String, String> attribs) throws Exception, IOException {
	
		tagStack.pop().setBody(body);

	}
	
	
}
