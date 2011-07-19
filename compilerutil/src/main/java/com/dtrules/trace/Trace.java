package com.dtrules.trace;

import java.io.FileInputStream;
import java.io.InputStream;
import java.util.HashMap;

import com.dtrules.xmlparser.GenericXMLParser;
import com.dtrules.xmlparser.XMLPrinter;

public class Trace {
	TraceNode root = new TraceNode("root", new HashMap<String,String>());
	
	public void load (String tracefile) throws Exception {
		InputStream tracefilestream = new FileInputStream(tracefile);
		load(tracefilestream);
	}

	public void load (InputStream tracefilestream) throws Exception {
		TraceLoader loader = new TraceLoader();
		GenericXMLParser.load(tracefilestream, loader);
		root = loader.tagStack.pop().children.get(0);
	}

	public void print(){
		XMLPrinter out = new XMLPrinter(System.out);
		out.setSpaceCnt(2);
		root.print( out);
	}
	
	static public void main(String [] args) throws Exception {
		Trace trace = new Trace();
		trace.load("/home/paul/java/merge/new/DTRules/sampleprojects/CHIP/testfiles/output/test_1_trace.xml");
		trace.print();
	}
	
	
}
