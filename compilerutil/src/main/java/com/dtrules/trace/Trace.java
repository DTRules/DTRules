package com.dtrules.trace;

import java.io.FileInputStream;
import java.io.InputStream;
import java.util.HashMap;
import java.util.Map;

import com.dtrules.entity.IREntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.session.IRSession;
import com.dtrules.session.RuleSet;
import com.dtrules.session.RulesDirectory;
import com.dtrules.xmlparser.GenericXMLParser;
import com.dtrules.xmlparser.XMLPrinter;

public class Trace {
	
	TraceNode root   = null;
	
	Map<String,IREntity> entitytable = new HashMap<String,IREntity>();
	
	/**
	 * Create a TraceNode tree from the given tracefile filepath. 
	 * @param tracefile
	 * @return
	 * @throws Exception
	 */
	public TraceNode load (String tracefile) throws Exception {
		InputStream tracefilestream = new FileInputStream(tracefile);
		return load(tracefilestream);
	}

	/**
	 * Create a TraceNode tree from the given tracefilestream.
	 * @param tracefilestream
	 * @return
	 * @throws Exception
	 */
	public TraceNode load (InputStream tracefilestream) throws Exception {
		TraceLoader loader = new TraceLoader();
		GenericXMLParser.load(tracefilestream, loader);
		root = loader.tagStack.pop().children.get(0);
		return root;
	}

	/**
	 * Print the TraceNode tree for this object.
	 */
	public void print(){
		XMLPrinter out = new XMLPrinter(System.out);
		out.setSpaceCnt(2);
		if(root != null ){
			root.print( out);
		}else{
			System.out.println("No tree has been loaded");
		}
	}
	/**
	 * Returns a session that represents a Rules Engine in the state at 
	 * the given node.
	 * 
	 * @param rs
	 * @param position
	 * @return IRSession
	 */
	public IRSession setState(RuleSet rs, TraceNode position) {
		try{
			IRSession session = rs.newSession();
			root.setState(this, session,position);
			return session;
		}catch(RulesException e){
			return null;
		}
	}
	/**
	 * Find the i'th node in the tree.
	 * @param i
	 * @return
	 */
	public TraceNode find(int i){
		return root.find(i);
	}
	
	/**
	 * A simple test file.
	 * @param args
	 * @throws Exception
	 */
	static public void main(String [] args) throws Exception {
		
		String tracefile    = System.getProperty("user.dir")+"/testdata/job_trace.xml";
		Trace trace 		= new Trace();
		
		trace.load(tracefile);
		//trace.print();
		
		TraceNode t = trace.find(11074);
		
		RulesDirectory rd = new RulesDirectory(
				System.getProperty("user.dir")+"/../sampleprojects/CHIP/", "DTRules.xml");
		
		RuleSet rs = rd.getRuleSet("CHIP");
		
		IRSession s = trace.setState(rs, t);
		
		int edepth = s.getState().edepth();
		for(int i = 0; i < edepth; i++){
			IREntity e = s.getState().getes(i);
			System.out.println(e.getName()+" "+e.getID());
		}
		
	}
	
	
}
