package com.dtrules.trace;

import java.io.FileInputStream;
import java.io.InputStream;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.Iterator;
import java.util.List;
import java.util.Map;

import com.dtrules.entity.IREntity;
import com.dtrules.infrastructure.RulesException;
import com.dtrules.interpreter.RArray;
import com.dtrules.interpreter.RName;
import com.dtrules.session.IRSession;
import com.dtrules.session.RuleSet;
import com.dtrules.session.RulesDirectory;
import com.dtrules.xmlparser.GenericXMLParser;
import com.dtrules.xmlparser.XMLPrinter;

public class Trace {

    TraceNode             root        = null;
    Map<String, IREntity> entitytable = new HashMap<String, IREntity>();
    Map<Integer, RArray>  arraytable  = new HashMap<Integer, RArray>();
    
    // This is the execute_table under execution.  It is null if no execute_table has
    // yet been executed.
    TraceNode             execute_table     = null;

    // Keep a list of Change objects to provide an easy "back pointer" to where
    // an attribute was changed.
    Map<Change, Change>   changes     = new HashMap<Change, Change>();

    IRSession             session;
    TraceNode             position;

    /**
     * Create entity. If we already know the entity, we return it, otherwise we get the entity from the session.
     */
    public IREntity createEntity(String id, String name) throws RulesException {
        IREntity e = entitytable.get(id);
        if (e == null) {
            e = session.createEntity(id, name);
            entitytable.put(id, e);
        }
        return e;
    }

    /**
     * Create a TraceNode tree from the given tracefile filepath.
     * 
     * @param tracefile
     * @return
     * @throws Exception
     */
    public TraceNode load(String tracefile) throws Exception {
        InputStream tracefilestream = new FileInputStream(tracefile);
        return load(tracefilestream);
    }

    /**
     * Create a TraceNode tree from the given tracefilestream.
     * 
     * @param tracefilestream
     * @return
     * @throws Exception
     */
    public TraceNode load(InputStream tracefilestream) throws Exception {
        TraceLoader loader = new TraceLoader();
        GenericXMLParser.load(tracefilestream, loader);
        root = loader.tagStack.pop().children.get(0);
        return root;
    }

    public TraceNode root() {
        return root;
    }

    /**
     * Print the TraceNode tree for this object.
     */
    public void print() {
        XMLPrinter out = new XMLPrinter(System.out);
        out.setSpaceCnt(2);
        if (root != null) {
            root.print(out);
        } else {
            System.out.println("No tree has been loaded");
        }
    }

    /**
     * Returns a session that represents a Rules Engine in the state at the given node. On any error encountered by the
     * Rules Engine, we return the state of the Rules Engine at that point. If we could not build a session at all, we
     * return a null.
     * 
     * @param rs
     * @param position
     * @return IRSession
     */
    public IRSession setState(RuleSet rs, TraceNode position) {
        this.position = position;
        session = null;
        this.execute_table = null;
        changes.clear();
        entitytable.clear();
        try {
            session = rs.newSession();
            root.setState(this, position);
        } catch (RulesException e) {
        }
        return session;
    }

    /**
     * Find the i'th node in the tree.
     * 
     * @param i
     * @return
     */
    public TraceNode find(int i) {
        return root.find(i);
    }

    /**
     * Find all the instances of a particular entity.  If no position has been
     * set, then this will return an empty list.
     * @param entityName
     * @return
     * @throws RulesException
     */
    public List<IREntity> instancesOf(String entityName) throws RulesException {

        List<IREntity> entityList = new ArrayList<IREntity>();
        if(session == null) return entityList;
        root.searchTree(this, entityName, entityList);
        return entityList;
    }

    /**
     * Return the actions of the current position. If not a column, then it looks for the enclosing column. If there is
     * no enclosing column, a null is returned.
     * 
     * @return List<Integer>
     */
    public List<Integer> getActions() {
        return position.getActions();
    }

    /**
     * Return the actions of the given position. If not a column, then it looks for the enclosing column. If there is no
     * enclosing column, a null is returned.
     * 
     * @return List<Integer>
     */
    public List<Integer> getActions(TraceNode position) {
        return position.getActions();
    }

    /**
     * Returns a position in the trace where the given entity's attribute has been changed by a 
     * Decision Table.  A null is returned if the attribute hasn't been changed by a Decision Table.
     * That does NOT mean that the attribute wasn't loaded with data.  Use isDefaultValue() to check
     * that.
     * @param e
     * @param attribute
     * @return
     */
    public TraceNode isChanged(IREntity e, RName attribute) {
        Change c = new Change(e, attribute, null);                 // Make a look up key.
        c = changes.get(c);                                 // Replace the key with the value
        if(c == null) return null;                          // Attribute still has default value.
        return c.execute_table;                             // Return the changed flag.
    }

    /**
     * If an attribute has not been loaded with a value from the input data or
     * Data Map, this routine returns true.  If the value of this attribute is 
     * the default value for the attribute, or the attribute has been changed by
     * a decision table, this routine returns false.
     */
    public boolean isDefaultValue (IREntity e, RName attribute ){
        Change c = new Change(e, attribute, null);                 // Make a look up key.
        c = changes.get(c);                                 // Replace the key with the value
        return (c == null);                                 // Return true if still the default value.
    }
    
    /**
     * An executable for testing functions in Trace
     * 
     * @param args
     * @throws Exception
     */
    static public void main(String[] args) throws Exception {

        String tracefile = System.getProperty("user.dir")
                + "/../sampleprojects/CHIP/testfiles/output/test_01_trace.xml";
        Trace trace = new Trace();

        trace.load(tracefile);
        trace.print();

        TraceNode t = trace.find(428);

        RulesDirectory rd = new RulesDirectory(
                System.getProperty("user.dir") + "/../sampleprojects/CHIP/",
                "DTRules.xml");

        RuleSet   rs = rd.getRuleSet("CHIP");
        IRSession s  = trace.setState(rs, t);
        
        if (s == null) {
            System.out.println("Could not build a session.");
            return;
        }

        for(IREntity e : trace.entitytable.values()){
            System.out.println(e.getName() + " " + e.getID());
            Iterator<RName> ai = e.getAttributeIterator();
            while(ai.hasNext()){
                RName attrib = ai.next();
                TraceNode n = trace.isChanged(e, attrib);
                boolean   d = trace.isDefaultValue(e, attrib);
                System.out.printf("    %1s %8d %20s\n",d?"*":" ",n!=null?n.number:-1,attrib);
            }
        }

    }

}
