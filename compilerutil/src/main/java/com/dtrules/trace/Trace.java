package com.dtrules.trace;

import java.io.FileInputStream;
import java.io.InputStream;
import java.util.HashMap;

import com.dtrules.xmlparser.GenericXMLParser;
import com.dtrules.xmlparser.XMLPrinter;

public class Trace {
	TraceNode root = new TraceNode("root", new HashMap<String,String>());
	
	public TraceNode load (String tracefile) throws Exception {
		InputStream tracefilestream = new FileInputStream(tracefile);
		return load(tracefilestream);
	}

	public TraceNode load (InputStream tracefilestream) throws Exception {
		TraceLoader loader = new TraceLoader();
		GenericXMLParser.load(tracefilestream, loader);
		root = loader.tagStack.pop().children.get(0);
		return root;
	}

	public void print(){
		XMLPrinter out = new XMLPrinter(System.out);
		out.setSpaceCnt(2);
		root.print( out);
	}
	
	static public void main(String [] args) throws Exception {
		
		String tracefile    = System.getProperty("user.dir")+"/testdata/job_trace.xml";
		Trace trace 		= new Trace();
		
		trace.load(tracefile);
		trace.print();
	}
	
	
}
