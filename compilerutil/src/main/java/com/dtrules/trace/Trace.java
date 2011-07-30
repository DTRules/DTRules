package com.dtrules.trace;

import java.io.FileInputStream;
import java.io.InputStream;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
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
	
	IRSession session;
	
	TraceNode position;
	
	/**
	 * Create entity.  If we already know the entity, we return it, otherwise
	 * we get the entity from the session.
	 */
	public IREntity createEntity(String id, String name) throws RulesException {
		IREntity e = entitytable.get(id);
		if(e == null){
			e = session.createEntity(id, name);
			entitytable.put(id, e);
		}
		return e;
	}
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
	
	public TraceNode root () {
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
	 * the given node.  On any error encountered by the Rules Engine, we
	 * return the state of the Rules Engine at that point.  If we could
	 * not build a session at all, we return a null.
	 * 
	 * @param rs
	 * @param position
	 * @return IRSession
	 */
	public IRSession setState(RuleSet rs, TraceNode position) {
		this.position = position;
		session = null;
		try{
			session = rs.newSession();
			root.setState(this, position);
		}catch(RulesException e){ }
		return session;
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
	 * Get the instances of a given entity, given the current position
	 * in the trace.	
	 * @param entityName
	 * @param position
	 * @return
	 * @throws RulesException
	 */
	public List<IREntity> instancesOf( String entityName ) throws RulesException { 
		
		List<IREntity> entityList = new ArrayList<IREntity>();
		root.searchTree(this, entityName, entityList);
		return entityList;
	}
	
	/**
	 * An executable for testing functions in Trace
	 * @param args
	 * @throws Exception
	 */
	static public void main(String [] args) throws Exception {
		
		String tracefile    = System.getProperty("user.dir")+
				"/../sampleprojects/CHIP/testfiles/output/test_1_trace.xml";
		Trace trace 		= new Trace();
		
		trace.load(tracefile);
		trace.print();
		
		TraceNode t = trace.find(250);
		
		RulesDirectory rd = new RulesDirectory(
				System.getProperty("user.dir")+"/../sampleprojects/CHIP/", "DTRules.xml");
		
		RuleSet rs = rd.getRuleSet("CHIP");
		
		IRSession s = trace.setState(rs, t);
	
		List<IREntity> entities = trace.instancesOf("client");
		
		for(IREntity e : entities){
			System.out.println(e.getName().stringValue()+" "+e.getID());
		}
		
		if(s==null) {
			System.out.println("Could not build a session.");
			return;
		}
		
		int edepth = s.getState().edepth();
		for(int i = 0; i < edepth; i++){
			IREntity e = s.getState().getes(i);
			System.out.println(e.getName()+" "+e.getID());
		}
		
	}
	
	
}
